// The Bubble Tea model for the full-screen chat surface. It owns the transcript, the input
// textarea, the scrollback viewport, and the streaming state machine that drives one
// agent.Session turn at a time. ALL model mutation happens inside Update on Bubble Tea's single
// goroutine: engine events arrive only as messages (msgs.go, pumped by pump.go), so the model
// never touches its fields off that goroutine and the whole design is race-free by construction.
//
// The session boundary is an interface (streamer) rather than *agent.Session, so Update is unit
// testable with a fake that needs no real Session, DB, or LLM (model_test.go). The turn context
// is derived from the model's base context and its cancel func is held on the model; esc calls
// it to cancel an in-flight turn. Because eino does not propagate ctx-cancel into a stuck
// provider stream, cancel only flips the footer to "cancelling…" — the pump keeps draining until
// the engine's terminal event lands, then input re-enables as usual.
package tui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2"
	"charm.land/lipgloss/v2"
	"github.com/atotto/clipboard"
	"github.com/pocketbase/pocketbase/core"

	"github.com/hsb3/deskkit/internal/core/config"
	"github.com/hsb3/deskkit/internal/core/tuiview"
	"github.com/hsb3/deskkit/internal/modules/librarian/agent"
)

// streamer is the model's view of an agent.Session: driving a turn, plus the run identity the
// picker needs to exclude the live session from the resume offers. *agent.Session satisfies it;
// a fake satisfies it in tests, so the model needs no real Session, DB, or LLM to unit-test
// Update.
type streamer interface {
	StreamTurn(ctx context.Context, userInput string) <-chan agent.Event
	RunID() string
}

// sessionProvider is the model's view of the agent package's conversation lifecycle: listing
// resumable runs, resuming one, opening a fresh one, closing a session, and the sessions-manager
// operations (rename, delete, preview). Keeping it an interface means Update never imports the
// agent package's functions directly, so the picker + session-swap paths are unit-testable with a
// fake (picker_test.go).
type sessionProvider interface {
	list(limit int, excludeRunID string, includeArchived bool) ([]agent.ConversationInfo, error)
	resume(ctx context.Context, runID string) (streamer, []agent.TranscriptEntry, error)
	fresh(ctx context.Context) (streamer, error)
	closeSession(ctx context.Context, s streamer) error
	rename(runID, title string) error
	delete(runID string) error
	setArchived(runID string, archived bool) error
	preview(runID string) ([]agent.TranscriptEntry, error)
}

// previewMaxRows bounds how many recent transcript rows the picker's preview pane loads per run —
// a glance at the tail of the conversation, refreshed as the cursor moves.
const previewMaxRows = 20

// agentProvider is the production sessionProvider: a thin adapter over the agent package bound to
// the app and resolved config. It is separate from the model's cfg (which the header still needs).
type agentProvider struct {
	app core.App
	cfg *config.Config
}

// list returns the most recent resumable conversations for the picker, excluding the caller's
// own live run. includeArchived reveals soft-archived conversations (default view hides them).
func (p *agentProvider) list(limit int, excludeRunID string, includeArchived bool) ([]agent.ConversationInfo, error) {
	return agent.ListConversations(p.app, limit, excludeRunID, includeArchived)
}

// resume rebuilds a live Session over an existing run and returns it (as a streamer) with the
// human-facing transcript. *agent.Session satisfies streamer.
func (p *agentProvider) resume(ctx context.Context, runID string) (streamer, []agent.TranscriptEntry, error) {
	s, ts, err := agent.ResumeSession(ctx, p.app, p.cfg, runID)
	if err != nil {
		return nil, nil, err
	}
	return s, ts, nil
}

// fresh opens a brand-new Session and returns it as a streamer.
func (p *agentProvider) fresh(ctx context.Context) (streamer, error) {
	s, err := agent.NewSession(ctx, p.app, p.cfg)
	if err != nil {
		return nil, err
	}
	return s, nil
}

// closeSession finalizes a session's agent_runs row. A non-*agent.Session streamer (a test fake)
// or a nil session is a no-op.
func (p *agentProvider) closeSession(ctx context.Context, s streamer) error {
	if as, ok := s.(*agent.Session); ok && as != nil {
		return as.Close(ctx)
	}
	return nil
}

// rename sets a conversation's title (its input_summary). A blank title is rejected in the agent
// layer; see agent.RenameConversation.
func (p *agentProvider) rename(runID, title string) error {
	return agent.RenameConversation(p.app, runID, title)
}

// delete hard-deletes a conversation's run row; its messages cascade away with it. This is not a
// record-original-first boundary violation — a chat conversation is the user's own history to
// discard (see agent/sessions.go).
func (p *agentProvider) delete(runID string) error {
	return agent.DeleteConversation(p.app, runID)
}

// setArchived toggles a conversation's soft-archive flag. Archiving is reversible and leaves the
// conversation's messages intact — distinct from delete's hard cascade (see agent/sessions.go).
func (p *agentProvider) setArchived(runID string, archived bool) error {
	return agent.SetConversationArchived(p.app, runID, archived)
}

// preview loads the highlighted conversation's recent transcript for the picker's preview pane.
func (p *agentProvider) preview(runID string) ([]agent.TranscriptEntry, error) {
	return agent.PreviewConversation(p.app, runID, previewMaxRows)
}

// Layout constants: the header and footer are one full-width line each; the input textarea is a
// fixed inner height wrapped in a rounded border (inputBorderHeight rows: one top, one bottom).
// The viewport takes whatever height remains.
const (
	headerHeight      = 1
	footerHeight      = 1
	inputHeight       = 3
	inputBorderHeight = 2 // rounded input box: top + bottom border rows added around the textarea
	tabStripHeight    = 1 // view-switcher strip: one full-width row, present ONLY when views are mounted
)

// maxMeasure caps the transcript's readable text width. Wide terminals keep a comfortable measure
// instead of stretching a line the full width of the screen; full-width chrome (the header/footer
// bars, the input border) still spans the real terminal width — only the text measure is capped.
const maxMeasure = 120

// measureWidth is the capped text measure for the transcript: min(terminalWidth - chrome, 120),
// floored at minWrap so a very narrow terminal still wraps sanely. chrome reserves the two columns
// the left gutter/block border + its trailing space consume, so a gutter-prefixed line at the full
// measure still fits the viewport width. Kept pure so the clamp is unit-tested without a terminal.
func measureWidth(termWidth int) int {
	const chrome = 2 // left gutter glyph + its trailing space (or the block border + padding)
	w := termWidth - chrome
	if w > maxMeasure {
		w = maxMeasure
	}
	if w < minWrap {
		w = minWrap
	}
	return w
}

// role distinguishes a transcript entry's author.
type role int

const (
	roleUser role = iota
	roleAssistant
	roleInfo // ambient host guidance (the one-time launch nudge), not a conversation turn
)

// entry is one transcript item. For an assistant turn, text is the streaming/answer bubble
// (the authoritative final Content once finalized); steps carries any tool invocations;
// commentary the model streamed before a tool call is retagged onto the step (see steps.go),
// not left in text. interrupted marks a canceled turn (dim "(interrupted)" badge, UI-only),
// isError a real terminal error (red). finalized gates markdown rendering: while false the
// bubble renders as plain text (smooth streaming), and glamour runs once when it flips true.
type entry struct {
	role        role
	text        string
	steps       []step
	rawLines    []string // unexpected event shapes, rendered dim rather than crashing
	interrupted bool
	isError     bool
	errText     string
	finalized   bool
	duration    time.Duration // wall time of the assistant turn, shown as a per-turn footer once finalized
	tokens      int           // completion tokens this turn generated, shown in the per-turn footer when >0
}

// matchStep returns the index of the step a tool_end belongs to: the last not-yet-done step
// with the matching CallID, falling back to the last not-done step (a provider that omits the
// call id). -1 when none is open.
func (e *entry) matchStep(callID string) int {
	for i := len(e.steps) - 1; i >= 0; i-- {
		if !e.steps[i].done && (callID == "" || e.steps[i].callID == callID) {
			return i
		}
	}
	for i := len(e.steps) - 1; i >= 0; i-- {
		if !e.steps[i].done {
			return i
		}
	}
	return -1
}

// keymap is the surface's binding table — the single place keys are declared, each carrying its own
// help text so the bubbles/help footer stays self-maintaining. openPicker (ctrl+o) opens the
// conversation-resume overlay and newConversation (ctrl+n) starts a fresh conversation; both are
// no-ops while a turn is streaming (like send), so no in-flight turn is ever disturbed. historyPrev
// (up) / historyNext (down) walk prior prompts only at the textarea's edge rows; copyLast (ctrl+y)
// copies the last answer's raw markdown; editExternal (ctrl+e) composes the draft in $EDITOR; help
// (ctrl+g, or "?") toggles the expanded help. "?" is bound to help only when the chat draft is empty
// or a module view is active, so a "?" typed mid-message still inserts literally (the empty-draft
// guard lives in handleKey); ctrl+g always toggles help regardless of the draft.
type keymap struct {
	send            key.Binding
	newline         key.Binding
	cancel          key.Binding
	editExternal    key.Binding
	toggleSteps     key.Binding
	scrollUp        key.Binding
	scrollDown      key.Binding
	historyPrev     key.Binding
	historyNext     key.Binding
	copyLast        key.Binding
	help            key.Binding
	quit            key.Binding
	openPicker      key.Binding
	newConversation key.Binding
	cycleViews      key.Binding
}

func defaultKeymap() keymap {
	return keymap{
		send:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "send")),
		newline: key.NewBinding(key.WithKeys("alt+enter"), key.WithHelp("alt+enter", "newline")),
		cancel:  key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
		// ctrl+e composes the draft in $EDITOR (the aider/gptme convention). It is intercepted before
		// the textarea, so it shadows the textarea's own emacs-style end-of-line motion — an accepted
		// trade for the far higher-value external-compose affordance.
		editExternal:    key.NewBinding(key.WithKeys("ctrl+e"), key.WithHelp("ctrl+e", "editor")),
		toggleSteps:     key.NewBinding(key.WithKeys("ctrl+t"), key.WithHelp("ctrl+t", "steps")),
		scrollUp:        key.NewBinding(key.WithKeys("pgup"), key.WithHelp("pgup", "scroll up")),
		scrollDown:      key.NewBinding(key.WithKeys("pgdown"), key.WithHelp("pgdn", "scroll down")),
		historyPrev:     key.NewBinding(key.WithKeys("up"), key.WithHelp("↑", "prev prompt")),
		historyNext:     key.NewBinding(key.WithKeys("down"), key.WithHelp("↓", "next prompt")),
		copyLast:        key.NewBinding(key.WithKeys("ctrl+y"), key.WithHelp("ctrl+y", "copy answer")),
		help:            key.NewBinding(key.WithKeys("ctrl+g", "?"), key.WithHelp("?", "help")),
		quit:            key.NewBinding(key.WithKeys("ctrl+c"), key.WithHelp("ctrl+c", "quit")),
		openPicker:      key.NewBinding(key.WithKeys("ctrl+o"), key.WithHelp("ctrl+o", "resume")),
		newConversation: key.NewBinding(key.WithKeys("ctrl+n"), key.WithHelp("ctrl+n", "new")),
		// Module views (spec §5.3): disabled until attachViews mounts a non-empty set, so a
		// librarian-only desk's surface (keys + help) is unchanged.
		cycleViews: key.NewBinding(key.WithKeys("ctrl+p"), key.WithHelp("ctrl+p", "views"), key.WithDisabled()),
	}
}

// ShortHelp is the one-line help footer's binding order (bubbles/help KeyMap). It surfaces the
// everyday keys; the rest live under the ctrl+g full-help expansion.
func (k keymap) ShortHelp() []key.Binding {
	return []key.Binding{k.send, k.editExternal, k.openPicker, k.newConversation, k.cycleViews, k.copyLast, k.toggleSteps, k.help, k.quit}
}

// FullHelp is the grouped expansion shown when ctrl+g toggles ShowAll (bubbles/help KeyMap),
// columns roughly by function: composing, conversation, transcript, meta.
func (k keymap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.send, k.newline, k.editExternal, k.cancel},
		{k.openPicker, k.newConversation, k.cycleViews, k.copyLast},
		{k.toggleSteps, k.scrollUp, k.scrollDown, k.historyPrev, k.historyNext},
		{k.help, k.quit},
	}
}

// pickerKeymap is the sessions-surface's contextual bindings — the keys that only mean something
// while the picker overlay is open, so they live OFF the main surface keymap (they never appear in
// the chat footer's help). Navigation, filter, resume, and close are handled via the list's own
// keys and the model's send/cancel bindings; these three are the lifecycle verbs the picker adds.
type pickerKeymap struct {
	rename       key.Binding // enter inline rename over the selected row
	delete       key.Binding // open the delete-confirm gate for the selected row
	confirmYes   key.Binding // confirm a pending delete
	archive      key.Binding // toggle the selected row's soft-archive flag (archive / unarchive)
	showArchived key.Binding // reveal / hide archived conversations in the list
}

func defaultPickerKeymap() pickerKeymap {
	return pickerKeymap{
		rename:       key.NewBinding(key.WithKeys("r")),
		delete:       key.NewBinding(key.WithKeys("d", "delete")),
		confirmYes:   key.NewBinding(key.WithKeys("y")),
		archive:      key.NewBinding(key.WithKeys("a")),
		showArchived: key.NewBinding(key.WithKeys("A")),
	}
}

// model is the chat surface. picker is the sessions overlay: nil when closed, and it only ever
// lives while NOT streaming (ctrl+o opens it only when no turn is in flight). provider is the
// session lifecycle boundary (list/resume/fresh/close/rename/delete/preview), an interface so
// Update stays testable without a real Session.
type model struct {
	baseCtx  context.Context
	sess     streamer
	provider sessionProvider

	deskName    string
	llmProvider string
	llmModel    string

	// Token accounting (spec §6 usage surface). ctxWindow is the model's context budget, resolved
	// ONCE in newModel from the config + the per-model table (usage.go). ctxTokens is the latest
	// finished turn's prompt-token count — the current context size, which drives the header's ctx%
	// gauge. sessionTokens is the cumulative total across the session (kept for the deferred
	// SSE/webapp surface; not shown in the one-line header today).
	ctxWindow     int
	ctxTokens     int
	sessionTokens int

	theme      string // concrete resolved palette ("light"/"dark"), never "auto"
	styles     styleSet
	keymap     keymap
	pickerKeys pickerKeymap // sessions-overlay contextual bindings (rename/delete/confirm)
	hlp        help.Model
	ta         textarea.Model
	vp         viewport.Model
	sp         spinner.Model
	renderer   *glamour.TermRenderer

	history history // prior-prompt recall (up/down at the textarea edges)
	reduced bool    // reduced motion (NO_COLOR set): static "working…" instead of the spinner

	entries     []entry
	inflightIdx int // index of the assistant entry being streamed (valid while streaming)

	streaming  bool
	cancelling bool
	quitting   bool
	showSteps  bool

	turnStart time.Time // monotonic start of the in-flight turn, for its per-turn footer
	toast     string    // transient footer toast (copy confirmation / error); "" when none
	toastSeq  int       // stamps the live toast so a stale expiry never clears a newer one

	cancelTurn context.CancelFunc
	events     <-chan agent.Event

	width    int
	height   int
	contentW int // capped transcript text measure (measureWidth); drives glamour wrap + user blocks
	ready    bool

	picker *pickerModel // conversation-resume overlay (ctrl+o); nil when closed.

	// resumeFirst requests the sessions overlay be opened once at launch when prior resumable
	// conversations exist (resume-first launch). Set by Run via enableResumeFirst; consumed
	// (cleared) on the first sizing WindowSizeMsg so a later terminal resize never reopens it. It
	// defaults false, so the pure-Update tests — which never set it — keep their launch behavior.
	resumeFirst bool

	// launchHint arms a one-time transcript nudge (seedLaunchHint) naming the mounted module views
	// and the ?/ctrl+p keys that reach them. Set by Run via enableLaunchHint, consumed on the first
	// WindowSizeMsg (like resumeFirst) and only when views are mounted. Defaults false, so the
	// pure-Update tests — which never arm it — keep their launch behavior and entry counts.
	launchHint bool

	// Module-contributed views (spec §5.3; host_views.go): views is the mounted set (empty on
	// a librarian-only desk), activeView the index of the one occupying the body region, or
	// -1 when the chat transcript is showing.
	views      []tuiview.View
	activeView int
}

// newModel builds the chat model against an injected streamer (the real *agent.Session in
// production, a fake in tests), the session provider (list/resume/fresh/close), the resolved
// config for the header, and the theme resolved once at startup (a concrete "light"/"dark"; see
// theme.go). The theme drives both the lipgloss palette and the glamour markdown style.
func newModel(baseCtx context.Context, sess streamer, provider sessionProvider, cfg *config.Config, theme string) model {
	ta := textarea.New()
	ta.Placeholder = "Ask the librarian… (enter to send, alt+enter for a newline)"
	// No textarea prompt glyph: the rounded input box border is the composing-line cue now.
	ta.Prompt = ""
	ta.ShowLineNumbers = false
	ta.CharLimit = 0
	// Plain enter is the send key (handled in Update); rebind the textarea's own newline to
	// alt+enter so it never swallows the send key.
	ta.KeyMap.InsertNewline = key.NewBinding(key.WithKeys("alt+enter"))
	// The textarea keeps its inline virtual cursor — the only cursor the surface has ever had —
	// rather than wiring the real terminal cursor through tea.View.Cursor (ADR 0007). Explicit,
	// though it is also bubbles' default.
	ta.SetVirtualCursor(true)
	// The AdaptiveColor-based defaults that v1 resolved via the lipgloss.SetHasDarkBackground pin
	// (tui.go) are gone with the pin itself; textarea.DefaultStyles(isDark) is v2's own per-theme
	// default table, selected by the same resolved theme, so the chrome (cursor line, line
	// numbers, placeholder) still varies by theme without any runtime query.
	ta.SetStyles(textarea.DefaultStyles(theme == themeDark))
	ta.Focus()

	sp := spinner.New(spinner.WithSpinner(spinner.Dot))

	styles := newStyles(theme)
	hlp := help.New()
	hlp.Styles = styles.help // concrete per-theme styles, never help.New()'s AdaptiveColor defaults

	return model{
		baseCtx:     baseCtx,
		sess:        sess,
		provider:    provider,
		deskName:    cfg.DeskName,
		llmProvider: cfg.LLMProvider,
		llmModel:    cfg.LLMModel,
		theme:       theme,
		styles:      styles,
		keymap:      defaultKeymap(),
		pickerKeys:  defaultPickerKeymap(),
		hlp:         hlp,
		ta:          ta,
		vp:          viewport.New(),
		sp:          sp,
		history:     newHistory(nil),
		reduced:     reducedMotion(os.Getenv("NO_COLOR")),
		inflightIdx: -1,
		activeView:  -1,
		// Resolve the context-window budget once: the profile/env override (Config.LLMContextWindow)
		// wins, else the per-model table default (usage.go). Fixed for the session's provider/model.
		ctxWindow: contextWindow(cfg.LLMProvider, cfg.LLMModel, cfg.LLMContextWindow),
	}
}

// reducedMotion reports whether the reduced-motion path is active: any non-empty NO_COLOR value.
// Read once at model construction (never per-frame) so the whole session is consistent. lipgloss
// and termenv already strip color under NO_COLOR; this flag governs only the animated spinner,
// which is swapped for a static "working…" state.
func reducedMotion(noColor string) bool {
	return noColor != ""
}

// Init starts the textarea cursor blink; nothing streams until the first turn.
func (m model) Init() tea.Cmd {
	return textarea.Blink
}

// Update is the whole state machine. It never mutates model fields off this goroutine; engine
// events reach it only as eventMsg/turnDoneMsg via the pump.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.resize(msg.Width, msg.Height)
		if m.launchHint {
			// One-shot, BEFORE the resume overlay: the nudge is seeded into the transcript so it is
			// still there once the picker (if any) is dismissed. Clear the flag first so a later
			// terminal resize never reseeds it; seed only when views are actually mounted.
			m.launchHint = false
			if len(m.views) > 0 {
				m.seedLaunchHint()
				m.refreshViewport()
			}
		}
		if m.resumeFirst {
			// One-shot: the overlay needs a sized viewport (only known after this first
			// WindowSizeMsg), so resume-first fires here rather than in Init/newModel. Clear the
			// flag first so a later resize can never reopen it.
			m.resumeFirst = false
			m.openLaunchPicker()
		}
		return m, nil

	case tea.KeyPressMsg:
		return m.handleKey(msg)

	case tuiview.SwitchMsg:
		// A view asked the host to activate a named sibling (e.g. the pm board's enter opening
		// the item detail). Unknown names are ignored.
		for i, v := range m.views {
			if v.Name() == msg.Name {
				return m.activateView(i)
			}
		}
		return m, nil

	case eventMsg:
		m.handleEvent(msg.ev)
		m.refreshViewport()
		if m.events != nil {
			return m, waitForEvent(m.events)
		}
		return m, nil

	case turnDoneMsg:
		m.streaming = false
		m.cancelling = false
		m.cancelTurn = nil
		m.events = nil
		if m.inflightIdx >= 0 && m.inflightIdx < len(m.entries) {
			m.entries[m.inflightIdx].finalized = true
			m.entries[m.inflightIdx].duration = time.Since(m.turnStart)
		}
		m.refreshViewport()
		if m.quitting {
			return m, tea.Quit
		}
		return m, nil

	case toastExpireMsg:
		// Only clear if this expiry belongs to the currently-shown toast; a newer toast bumped the
		// sequence and scheduled its own expiry.
		if msg.seq == m.toastSeq {
			m.toast = ""
		}
		return m, nil

	case editorFinishedMsg:
		// The external $EDITOR compose returned: read the draft back into the textarea (or toast a
		// failure), always removing the temp file.
		return m.handleEditorFinished(msg)

	case spinner.TickMsg:
		// Under reduced motion the spinner never animates (a static "working…" shows instead), so
		// drop stray ticks rather than re-scheduling them.
		if !m.streaming || m.reduced {
			return m, nil
		}
		var cmd tea.Cmd
		m.sp, cmd = m.sp.Update(msg)
		return m, cmd
	}
	return m, nil
}

// handleKey routes a keypress. Send/cancel/quit/steps are intercepted; scrolling goes to the
// viewport; everything else (typing, alt+enter newline) goes to the textarea. ctrl+c quits from
// anywhere (even with the overlay open); otherwise, when the picker overlay is open every key
// routes to it first (handlePickerKey).
func (m model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// ctrl+c is honored before the overlay so the picker can never trap the user. The picker only
	// lives while NOT streaming, so this path is only ever the immediate-quit branch when open.
	if key.Matches(msg, m.keymap.quit) {
		// ctrl+c: if a turn is in flight, cancel it and mark quitting; the existing pump keeps
		// draining and turnDoneMsg then quits (never Close a channel mid-flight). Otherwise quit
		// immediately. No new waitForEvent is issued here — exactly one pump is ever in flight.
		if m.streaming {
			if m.cancelTurn != nil {
				m.cancelTurn()
			}
			m.cancelling = true
			m.quitting = true
			return m, nil
		}
		return m, tea.Quit
	}

	if m.picker != nil {
		return m.handlePickerKey(msg)
	}

	// A mounted module view is active: it owns the keys (esc back to chat, ctrl+p next view,
	// everything else routed to the view) — see host_views.go.
	if m.activeView >= 0 {
		return m.handleViewKey(msg)
	}

	switch {
	case key.Matches(msg, m.keymap.cancel):
		// esc cancels an in-flight turn (footer → "cancelling…"); the pump drains to the terminal
		// event, which may take a moment because eino cannot abort a stuck provider stream.
		if m.streaming && m.cancelTurn != nil {
			m.cancelTurn()
			m.cancelling = true
		}
		return m, nil

	case key.Matches(msg, m.keymap.toggleSteps):
		m.showSteps = !m.showSteps
		m.refreshViewport()
		return m, nil

	case key.Matches(msg, m.keymap.cycleViews):
		// Enter the mounted module views (first view). The binding is enabled only once views are
		// mounted (attachViews), so this branch is only reachable on a desk that has them.
		return m.activateView(0)

	case msg.String() == "ctrl+p":
		// ctrl+p reached here only because the switcher binding is DISABLED — no module views are
		// mounted. Explain rather than no-op so the key never reads as dead; the chat stays active
		// (activeView unchanged at -1).
		return m.showToast("no module views on this desk — PM is off")

	case key.Matches(msg, m.keymap.openPicker):
		return m.openPicker()

	case key.Matches(msg, m.keymap.newConversation):
		return m.newConversation()

	case key.Matches(msg, m.keymap.copyLast):
		return m.copyLastAnswer()

	case key.Matches(msg, m.keymap.editExternal):
		return m.editInExternalEditor()

	case key.Matches(msg, m.keymap.help):
		// ctrl+g always toggles the full-help overlay. "?" is also bound to help, but only opens it
		// when the draft is empty — a "?" typed into a message-in-progress inserts literally instead
		// of hijacking the key.
		if msg.String() == "?" && m.ta.Value() != "" {
			var cmd tea.Cmd
			m.ta, cmd = m.ta.Update(msg)
			return m, cmd
		}
		m.hlp.ShowAll = !m.hlp.ShowAll
		return m, nil

	case key.Matches(msg, m.keymap.send):
		if m.streaming {
			return m, nil // no-op while a turn is in flight (never start an overlapping turn)
		}
		input := strings.TrimSpace(m.ta.Value())
		if input == "" {
			return m, nil
		}
		return m.startTurn(input)

	case key.Matches(msg, m.keymap.historyPrev):
		// Up recalls an older prompt only from the textarea's first row; elsewhere it is normal
		// cursor movement. A recall that does not move (no history / already oldest) is swallowed
		// (up at the first row would be a no-op in the textarea anyway).
		if m.ta.Line() == 0 {
			if v, moved := m.history.prev(m.ta.Value()); moved {
				m.ta.SetValue(v)
				m.ta.CursorEnd()
			}
			return m, nil
		}
		var cmd tea.Cmd
		m.ta, cmd = m.ta.Update(msg)
		return m, cmd

	case key.Matches(msg, m.keymap.historyNext):
		// Down walks toward newer prompts (and restores the stashed draft past the newest) only from
		// the textarea's last row; elsewhere it is normal cursor movement.
		if m.ta.Line() == m.ta.LineCount()-1 {
			if v, moved := m.history.next(m.ta.Value()); moved {
				m.ta.SetValue(v)
				m.ta.CursorEnd()
			}
			return m, nil
		}
		var cmd tea.Cmd
		m.ta, cmd = m.ta.Update(msg)
		return m, cmd

	case key.Matches(msg, m.keymap.scrollUp), key.Matches(msg, m.keymap.scrollDown):
		var cmd tea.Cmd
		m.vp, cmd = m.vp.Update(msg)
		return m, cmd

	default:
		var cmd tea.Cmd
		m.ta, cmd = m.ta.Update(msg)
		return m, cmd
	}
}

// copyLastAnswer copies the last finalized assistant answer's RAW markdown to the system clipboard
// (not the glamour-rendered output: terminal-wrapped rendering is mangled, the source is the point)
// and shows a transient footer toast confirming the copy, its failure, or that there is nothing to
// copy. The toast self-expires via a tea.Tick stamped with the toast sequence.
func (m model) copyLastAnswer() (tea.Model, tea.Cmd) {
	md, ok := lastAssistantMarkdown(m.entries)
	if !ok {
		return m.showToast("nothing to copy")
	}
	if err := clipboard.WriteAll(md); err != nil {
		return m.showToast("copy failed: " + err.Error())
	}
	return m.showToast("copied")
}

// editInExternalEditor composes the current draft in the user's external editor, resolved from
// $EDITOR (falling back to $VISUAL). It is a no-op while a turn is streaming — the same guard as
// send/openPicker/newConversation, so no in-flight turn is ever disturbed. With no editor
// configured it surfaces a footer toast rather than failing silently or crashing.
//
// The draft is written to a temp .md file, then handed to tea.ExecProcess — the sanctioned way to
// shell out under Bubble Tea: it releases the terminal to the child editor and restores the program
// on return, so there is no stray terminal query racing the input reader (the no-runtime-query
// invariant, ADR 0004). Any other shell-out would leave Bubble Tea holding the tty. When the editor
// exits, editorFinishedMsg carries the temp path back to Update, which reads the composed text into
// the textarea and removes the file. The draft is NOT auto-sent — the user reviews it and presses
// enter.
func (m model) editInExternalEditor() (tea.Model, tea.Cmd) {
	if m.streaming {
		return m, nil // no-op while a turn is in flight (mirror send/openPicker)
	}
	// $VISUAL is preferred over $EDITOR (the git/less/crontab convention: VISUAL names the
	// full-screen-capable editor, which is exactly what a real terminal hand-off wants; $EDITOR is
	// the fallback). editorCommand takes them highest-precedence-first.
	cmdline := editorCommand(os.Getenv("VISUAL"), os.Getenv("EDITOR"))
	if len(cmdline) == 0 {
		return m.showToast("set $EDITOR to compose externally")
	}
	path, err := writeDraft(m.ta.Value())
	if err != nil {
		return m.showToast("could not open editor: " + err.Error())
	}
	// cmdline[0] is the editor; cmdline[1:] its configured flags; the draft path is the final arg.
	args := append(cmdline[1:], path)
	c := exec.Command(cmdline[0], args...)
	return m, tea.ExecProcess(c, func(runErr error) tea.Msg {
		return editorFinishedMsg{path: path, err: runErr}
	})
}

// handleEditorFinished folds the result of an external-editor compose back into the surface, on
// Bubble Tea's goroutine. On success the composed text replaces the textarea value (readDraft has
// already trimmed the editor's trailing newline); on failure a toast reports it. The temp file is
// ALWAYS removed, whatever the outcome. The draft is left in the input for review — never auto-sent.
func (m model) handleEditorFinished(msg editorFinishedMsg) (tea.Model, tea.Cmd) {
	defer os.Remove(msg.path)
	if msg.err != nil {
		return m.showToast("editor exited with an error")
	}
	contents, err := readDraft(msg.path)
	if err != nil {
		return m.showToast("could not read the composed draft")
	}
	m.ta.SetValue(contents)
	return m, nil
}

// showToast sets the transient footer toast and returns the command that clears it after a short
// window. Each toast bumps toastSeq so a stale expiry for a superseded toast is ignored.
func (m model) showToast(text string) (tea.Model, tea.Cmd) {
	m.toastSeq++
	m.toast = text
	seq := m.toastSeq
	return m, tea.Tick(3*time.Second, func(time.Time) tea.Msg { return toastExpireMsg{seq: seq} })
}

// lastAssistantMarkdown returns the raw markdown of the most recent finalized assistant turn that
// carries answer text, and whether one exists. Steps-only or empty assistant turns are skipped
// (there is no answer body to copy); a still-streaming turn is skipped (not yet finalized).
func lastAssistantMarkdown(entries []entry) (string, bool) {
	for i := len(entries) - 1; i >= 0; i-- {
		e := entries[i]
		if e.role == roleAssistant && e.finalized && e.text != "" {
			return e.text, true
		}
	}
	return "", false
}

// openPicker opens the sessions overlay. It is a no-op while a turn is streaming (so no in-flight
// turn is ever disturbed) or when the picker is already open. It lists the recent conversations
// synchronously — a fast local DB read — sizes the overlay to the viewport area, and loads the
// preview for the initially-highlighted row. A list error leaves the picker closed rather than
// crashing the surface.
func (m model) openPicker() (tea.Model, tea.Cmd) {
	if m.streaming || m.picker != nil {
		return m, nil
	}
	convos, err := m.provider.list(pickerLimit, m.sess.RunID(), false)
	if err != nil {
		// Degraded: keep the surface open without a picker, but say why — a silent ctrl+o that
		// does nothing reads as a dead keybinding, not a failed lookup.
		m.appendError("could not list conversations: " + err.Error())
		m.refreshViewport()
		return m, nil
	}
	m.picker = newPicker(convos, m.styles, m.vp.Width(), m.vp.Height())
	m.refreshPreview()
	return m, nil
}

// enableResumeFirst arms the resume-first launch behavior: when prior resumable conversations
// exist, the surface opens on the sessions list at startup instead of dropping straight into a
// fresh conversation. Called once by Run before the program starts.
func (m *model) enableResumeFirst() { m.resumeFirst = true }

// enableLaunchHint arms the one-time launch nudge (seedLaunchHint), fired on the first WindowSizeMsg
// like resume-first. Called once by Run before the program starts; a no-op on a librarian-only desk
// (the consume guard only seeds when views are mounted).
func (m *model) enableLaunchHint() { m.launchHint = true }

// seedLaunchHint appends the one-time launch nudge to the transcript: a faint line naming the mounted
// module views and the keys that reach them (ctrl+p) and the help overlay (?). Called once, only when
// views are mounted, so a librarian-only desk's transcript is unchanged. The view names come from the
// modules (view.Name()), never hardcoded here, so this carries no deployment identity.
func (m *model) seedLaunchHint() {
	names := make([]string, 0, len(m.views))
	for _, v := range m.views {
		names = append(names, v.Name())
	}
	text := "module views mounted: " + strings.Join(names, ", ") +
		"  ·  ctrl+p switches views  ·  ? opens help"
	m.entries = append(m.entries, entry{role: roleInfo, text: text, finalized: true})
}

// openLaunchPicker opens the sessions overlay at startup when prior resumable conversations exist
// (resume-first launch): the reader lands on the list to pick one, or esc / ctrl+n to start
// fresh in the session Run already created. With NO prior conversations — or a list error — it is a
// no-op and the surface drops straight into the fresh conversation. This deliberately differs from
// openPicker, which opens even on an empty list (an explicit ctrl+o earns visible feedback): an
// empty overlay the user never asked for would just be dead chrome to esc past on first run.
func (m *model) openLaunchPicker() {
	if m.streaming || m.picker != nil {
		return
	}
	convos, err := m.provider.list(pickerLimit, m.sess.RunID(), false)
	if err != nil || len(convos) == 0 {
		return
	}
	m.picker = newPicker(convos, m.styles, m.vp.Width(), m.vp.Height())
	m.refreshPreview()
}

// appendError appends an inline red assistant entry — the visible-feedback path for a degraded
// provider call (a silent no-op reads as a dead keybinding). Shared by openPicker and the
// resume/rename/delete degraded branches.
func (m *model) appendError(text string) {
	m.entries = append(m.entries, entry{role: roleAssistant, isError: true, finalized: true, errText: text})
}

// newConversation abandons the current conversation and starts a fresh one. It is a no-op while a
// turn is streaming. Open-before-close: the fresh session is built FIRST, and the current one is
// closed only once a replacement exists — so a failed open leaves the old session genuinely live
// (not finalized out from under the user). No drain is needed at swap time: ctrl+n is a no-op
// while streaming, so there is never an in-flight turn.
func (m model) newConversation() (tea.Model, tea.Cmd) {
	if m.streaming {
		return m, nil
	}
	fresh, err := m.provider.fresh(m.baseCtx)
	if err != nil {
		// Degraded: the fresh session failed to build. The old session was not touched, so the
		// surface stays fully usable on it — but say why (same visible-error path as openPicker),
		// since a silent ctrl+n that does nothing reads as a dead keybinding, not a failed build.
		m.entries = append(m.entries, entry{
			role: roleAssistant, isError: true, finalized: true,
			errText: "could not start a new conversation: " + err.Error(),
		})
		m.refreshViewport()
		return m, nil
	}
	_ = m.provider.closeSession(m.baseCtx, m.sess)
	m.sess = fresh
	m.entries = nil
	m.inflightIdx = -1
	m.ctxTokens = 0             // usage is live-only; don't show the previous session's gauge
	m.history = newHistory(nil) // fresh conversation: no prior prompts to recall
	m.picker = nil
	m.refreshViewport()
	return m, nil
}

// handlePickerKey routes a keypress while the sessions overlay is open, by the picker's mode:
//   - rename mode routes to the inline title editor (handleRenameKey);
//   - confirm-delete mode routes to the y/n gate (handleConfirmDeleteKey);
//   - browse mode: while the list is actively capturing a filter query, EVERY key goes to the list
//     so its own esc cancels and enter applies the filter (the outer esc/enter must not fire).
//     Otherwise esc closes the overlay (or clears an applied filter first), enter resumes the
//     highlighted run, r starts an inline rename, d opens the delete-confirm gate, and any other
//     key drives the list (cursor movement / starting a filter with "/"), refreshing the preview
//     as the selection moves.
//
// Because the picker only lives while NOT streaming (ctrl+o/ctrl+n are no-ops mid-turn), there is
// never an in-flight turn to drain at swap time, so closing the current session before opening the
// resumed one is safe immediately — this is how the picker path respects the engine's drain contract.
func (m model) handlePickerKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch m.picker.mode {
	case pickerRename:
		return m.handleRenameKey(msg)
	case pickerConfirmDelete:
		return m.handleConfirmDeleteKey(msg)
	}

	// browse mode
	if m.picker.settingFilter() {
		// The list is editing its filter query: route the key to it and let its own esc/enter
		// cancel/apply the filter. Refresh the preview since a filter edit can move the selection.
		cmd := m.picker.Update(msg)
		m.refreshPreview()
		return m, cmd
	}

	switch {
	case key.Matches(msg, m.keymap.cancel):
		// esc: clear an applied filter first (a natural two-stage esc), otherwise dismiss the
		// overlay. There is no in-flight turn to cancel here (the picker only opens while not
		// streaming), so esc is purely filter-clear / overlay-dismiss.
		if m.picker.filterApplied() {
			cmd := m.picker.Update(msg) // the list's own esc clears the applied filter
			m.refreshPreview()
			return m, cmd
		}
		m.picker = nil
		m.refreshViewport()
		return m, nil

	case key.Matches(msg, m.keymap.send):
		return m.resumeSelected()

	case key.Matches(msg, m.pickerKeys.rename):
		if m.picker.selectedRunID() == "" {
			return m, nil
		}
		return m, m.picker.startRename()

	case key.Matches(msg, m.pickerKeys.delete):
		if m.picker.selectedRunID() == "" {
			return m, nil
		}
		m.picker.startConfirmDelete()
		return m, nil

	case key.Matches(msg, m.pickerKeys.archive):
		return m.toggleArchiveSelected()

	case key.Matches(msg, m.pickerKeys.showArchived):
		return m.toggleShowArchived()

	default:
		cmd := m.picker.Update(msg)
		m.refreshPreview()
		return m, cmd
	}
}

// toggleArchiveSelected soft-archives the highlighted conversation, or unarchives it when it is
// already archived — a reversible hide from the default resume list that never touches the
// conversation's messages (distinct from delete's hard cascade). It then reloads the list: when
// archiving out of the default view the row vanishes, so the list settles on the nearest remaining
// row; when unarchiving in the reveal view the row stays, so it keeps the selection. An empty
// selection is a no-op; a store error dismisses the overlay with a visible reason.
func (m model) toggleArchiveSelected() (tea.Model, tea.Cmd) {
	runID := m.picker.selectedRunID()
	if runID == "" {
		return m, nil
	}
	archived := !m.picker.selectedArchived()
	if err := m.provider.setArchived(runID, archived); err != nil {
		m.picker = nil
		m.appendError("could not archive conversation: " + err.Error())
		m.refreshViewport()
		return m, nil
	}
	keep := ""
	if m.picker.showArchived {
		keep = runID // still visible in the reveal view — keep it selected
	}
	return m.reloadPicker(keep)
}

// toggleShowArchived flips whether archived conversations are revealed in the list, then reloads it
// (keeping the current selection when it survives the new filter). This is the only path that opens
// the archived rows for unarchiving, since the default view hides them.
func (m model) toggleShowArchived() (tea.Model, tea.Cmd) {
	m.picker.showArchived = !m.picker.showArchived
	return m.reloadPicker(m.picker.selectedRunID())
}

// resumeSelected resumes the highlighted conversation, swapping the live session for it. An empty
// selection (empty list) is a no-op that leaves the overlay open. Open-before-close (same reasoning
// as newConversation): resume FIRST, and close the current session only once the replacement
// exists, so a failed resume leaves the old session genuinely live rather than finalized out from
// under the user; a visible inline error replaces a silent dismissal.
func (m model) resumeSelected() (tea.Model, tea.Cmd) {
	runID := m.picker.selectedRunID()
	if runID == "" {
		return m, nil
	}
	newSess, transcript, err := m.provider.resume(m.baseCtx, runID)
	if err != nil {
		m.picker = nil
		m.appendError("could not resume conversation: " + err.Error())
		m.refreshViewport()
		return m, nil
	}
	_ = m.provider.closeSession(m.baseCtx, m.sess)
	m.sess = newSess
	m.entries = entriesFromTranscript(transcript)
	m.inflightIdx = -1
	m.ctxTokens = 0                                           // usage is live-only; reads 0 until the first resumed turn reports
	m.history = newHistory(userPromptsNewestFirst(m.entries)) // resume seeds prompt recall
	m.picker = nil
	m.refreshViewport()
	return m, nil
}

// handleRenameKey drives the inline title editor: esc cancels (row untouched), enter commits the
// trimmed title via provider.rename then reloads the list, and every other key edits the input. An
// empty/whitespace title on enter is treated as a cancel (the agent layer would reject it anyway,
// and an empty input_summary would hide the run from the list).
func (m model) handleRenameKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keymap.cancel):
		m.picker.cancelRename()
		return m, nil

	case key.Matches(msg, m.keymap.send):
		title := m.picker.renameValue() // already trimmed
		runID := m.picker.selectedRunID()
		if title == "" || runID == "" {
			m.picker.cancelRename()
			return m, nil
		}
		if err := m.provider.rename(runID, title); err != nil {
			m.picker.cancelRename()
			m.picker = nil
			m.appendError("could not rename conversation: " + err.Error())
			m.refreshViewport()
			return m, nil
		}
		return m.reloadPicker(runID)

	default:
		cmd := m.picker.Update(msg) // routes to the rename input (picker mode == rename)
		return m, cmd
	}
}

// handleConfirmDeleteKey is the delete gate: y hard-deletes the highlighted run (cascading its
// messages) and reloads the list; n / esc / any other key backs out untouched. A hard delete is
// correct here — a chat conversation is the user's own history, not a desk-file fix under the
// record-original-first boundary (see agent/sessions.go).
func (m model) handleConfirmDeleteKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if key.Matches(msg, m.pickerKeys.confirmYes) {
		runID := m.picker.selectedRunID()
		if runID == "" {
			m.picker.cancelConfirmDelete()
			return m, nil
		}
		if err := m.provider.delete(runID); err != nil {
			m.picker.cancelConfirmDelete()
			m.picker = nil
			m.appendError("could not delete conversation: " + err.Error())
			m.refreshViewport()
			return m, nil
		}
		// The deleted row is gone; pass "" so the rebuilt list settles on the nearest remaining row.
		return m.reloadPicker("")
	}
	// n, esc, or anything else: back out of the gate without deleting.
	m.picker.cancelConfirmDelete()
	return m, nil
}

// reloadPicker re-lists the conversations after a rename or delete and rebuilds the overlay in
// place, re-selecting selectRunID when it is still present (a delete passes "" so the list settles
// on the nearest remaining row), then reloads the preview for the new selection. A list error
// dismisses the overlay with a visible reason rather than leaving a stale list.
func (m model) reloadPicker(selectRunID string) (tea.Model, tea.Cmd) {
	convos, err := m.provider.list(pickerLimit, m.sess.RunID(), m.picker.showArchived)
	if err != nil {
		m.picker = nil
		m.appendError("could not list conversations: " + err.Error())
		m.refreshViewport()
		return m, nil
	}
	m.picker.reload(convos, selectRunID)
	m.refreshPreview()
	return m, nil
}

// refreshPreview loads the highlighted run's recent transcript into the picker's preview pane,
// skipping the DB read when the pane already shows that run (a guard against re-querying on every
// keystroke). An empty selection clears the pane; a preview-load error clears the rows so the pane
// degrades to its neutral hint rather than showing a stale transcript.
func (m *model) refreshPreview() {
	if m.picker == nil {
		return
	}
	runID := m.picker.selectedRunID()
	if runID == "" {
		m.picker.setPreview(nil, "")
		return
	}
	if m.picker.previewID == runID {
		return
	}
	ts, err := m.provider.preview(runID)
	if err != nil {
		// Mark this run as previewed (empty) so the pane shows its neutral hint and we do not
		// re-hit the failing query on the next keystroke.
		m.picker.setPreview(nil, runID)
		return
	}
	m.picker.setPreview(ts, runID)
}

// pickerLimit bounds how many recent conversations the resume overlay lists.
const pickerLimit = 50

// entriesFromTranscript maps a resumed conversation's transcript to display entries, one entry per
// TranscriptEntry (the rows are already ordered and capped by the agent package). All entries are
// finalized (they are history, not streaming). A user row is a user bubble; a plain assistant row
// is an assistant bubble; a tool-calling assistant row and a tool row each become an assistant
// entry carrying a single done step (the assistant's row keeps its commentary text; the tool's row
// keeps its result text). An unrecognized role is skipped.
func entriesFromTranscript(ts []agent.TranscriptEntry) []entry {
	out := make([]entry, 0, len(ts))
	for _, t := range ts {
		switch t.Role {
		case "user":
			out = append(out, entry{role: roleUser, text: t.Text, finalized: true})
		case "assistant":
			if t.ToolName == "" {
				out = append(out, entry{role: roleAssistant, text: t.Text, finalized: true})
			} else {
				out = append(out, entry{
					role:      roleAssistant,
					steps:     []step{{tool: t.ToolName, commentary: t.Text, done: true}},
					finalized: true,
				})
			}
		case "tool":
			out = append(out, entry{
				role:      roleAssistant,
				steps:     []step{{tool: t.ToolName, result: t.Text, done: true}},
				finalized: true,
			})
		default:
			// Unknown role: skip rather than render a misleading bubble.
		}
	}
	return out
}

// userPromptsNewestFirst extracts the user prompts from a transcript in newest-first order, seeding
// the prompt-recall history when a conversation is resumed so up/down walks the loaded messages.
func userPromptsNewestFirst(entries []entry) []string {
	var out []string
	for i := len(entries) - 1; i >= 0; i-- {
		if entries[i].role == roleUser && entries[i].text != "" {
			out = append(out, entries[i].text)
		}
	}
	return out
}

// startTurn opens one streaming turn: derive a cancelable turn context, hand the input to the
// engine, append the user entry and an empty in-flight assistant entry, and kick off the pump
// plus the spinner. Returns the model and its initial commands.
func (m model) startTurn(input string) (tea.Model, tea.Cmd) {
	turnCtx, cancel := context.WithCancel(m.baseCtx)
	ch := m.sess.StreamTurn(turnCtx, input)

	m.entries = append(m.entries, entry{role: roleUser, text: input, finalized: true})
	m.entries = append(m.entries, entry{role: roleAssistant})
	m.inflightIdx = len(m.entries) - 1
	m.history.add(input) // recallable next turn; also resets any in-progress recall

	m.events = ch
	m.cancelTurn = cancel
	m.streaming = true
	m.cancelling = false
	m.turnStart = time.Now()
	m.ta.Reset()
	m.refreshViewport()
	// Under reduced motion the spinner is replaced by static text, so its tick is never started.
	if m.reduced {
		return m, waitForEvent(ch)
	}
	return m, tea.Batch(waitForEvent(ch), m.sp.Tick)
}

// handleEvent folds one engine event into the in-flight assistant entry. Token text
// accumulates into the answer bubble; a tool_start retags the current bubble as that step's
// commentary and resets the bubble (post-tool tokens become the real answer); a tool_end
// resolves the step (Err → failed ✗, else Result); final replaces the bubble with the
// authoritative content; error records the interrupted/error terminal state. An unrecognized
// Kind is stored as a dim raw line, never a panic.
func (m *model) handleEvent(ev agent.Event) {
	if m.inflightIdx < 0 || m.inflightIdx >= len(m.entries) {
		return
	}
	e := &m.entries[m.inflightIdx]
	switch ev.Kind {
	case agent.EventToken:
		e.text += ev.Token
	case agent.EventToolStart:
		e.steps = append(e.steps, step{
			tool:       ev.Tool,
			callID:     ev.CallID,
			args:       ev.Args,
			commentary: e.text,
		})
		e.text = "" // the answer bubble restarts after the tool; pre-tool text is now commentary
	case agent.EventToolEnd:
		if si := e.matchStep(ev.CallID); si >= 0 {
			e.steps[si].done = true
			if ev.Err != "" {
				e.steps[si].failed = true
				e.steps[si].errText = ev.Err
			} else {
				e.steps[si].result = ev.Result
			}
		}
	case agent.EventFinal:
		e.text = ev.Content
		e.finalized = true
		m.recordUsage(e, ev)
	case agent.EventError:
		if e.text == "" && ev.Partial != "" {
			e.text = ev.Partial
		}
		e.errText = ev.Err
		if ev.Canceled {
			e.interrupted = true
		} else {
			e.isError = true
		}
		e.finalized = true
		m.recordUsage(e, ev)
	default:
		e.rawLines = append(e.rawLines, fmt.Sprintf("unexpected event: %+v", ev))
	}
}

// recordUsage folds a terminal event's token accounting into the model and the finishing entry:
// the turn's completion tokens land on the entry (its per-turn footer), the prompt-token count
// becomes the live context size (the header ctx% gauge) when the provider reported one, and the
// total adds to the running session tally. A turn whose provider reported no usage leaves all
// three at zero, so the header/footer segments simply stay hidden.
func (m *model) recordUsage(e *entry, ev agent.Event) {
	e.tokens = ev.CompletionTokens
	if ev.PromptTokens > 0 {
		m.ctxTokens = ev.PromptTokens
	}
	m.sessionTokens += ev.TotalTokens
}

// resize recomputes the layout and rebuilds the markdown renderer to the new viewport width
// (a renderer keeps its word-wrap, so it must be rebuilt on every size change).
func (m *model) resize(w, h int) {
	m.width = w
	m.height = h
	m.contentW = measureWidth(w)
	m.ready = true

	// The view-switcher strip claims one row below the header, but only on a desk that mounted
	// module views — a librarian-only desk keeps the original body height (and no strip).
	tabH := 0
	if len(m.views) > 0 {
		tabH = tabStripHeight
	}
	vpHeight := h - headerHeight - tabH - footerHeight - inputHeight - inputBorderHeight
	if vpHeight < 1 {
		vpHeight = 1
	}
	m.vp.SetWidth(w)
	m.vp.SetHeight(vpHeight)
	// The textarea sits inside the rounded input box; reserve the two columns its left/right
	// border consume so the box spans the full terminal width without wrapping.
	m.ta.SetWidth(w - 2)
	m.ta.SetHeight(inputHeight)
	m.hlp.SetWidth(w) // the help footer truncates its short view to the terminal width
	// Glamour wraps at the CAPPED measure, not the raw width — the transcript keeps a readable
	// line length on a wide terminal even though the bars span the whole screen.
	m.renderer = newRenderer(m.contentW, m.theme)
	if m.picker != nil {
		m.picker.SetSize(w, vpHeight) // the overlay occupies the viewport area
	}
	for _, v := range m.views {
		v.SetSize(w, vpHeight) // module views occupy the same body region as the transcript
	}
	m.refreshViewport()
}

// refreshViewport re-renders the transcript into the viewport, preserving the user's scroll
// position unless they were already at the bottom (in which case new output auto-scrolls). The
// follow decision is captured BEFORE the content update — from the pre-update scroll position — so
// a reader who scrolled up is never yanked back down by newly streamed content.
func (m *model) refreshViewport() {
	if !m.ready {
		return
	}
	follow := shouldAutoFollow(m.vp.ScrollPercent())
	m.vp.SetContent(m.renderTranscript())
	if follow {
		m.vp.GotoBottom()
	}
}

// renderTranscript renders every entry, separated by a blank line.
func (m model) renderTranscript() string {
	parts := make([]string, 0, len(m.entries))
	for i := range m.entries {
		parts = append(parts, m.renderEntry(m.entries[i]))
	}
	return strings.Join(parts, "\n\n")
}

// renderEntry renders one transcript item. A USER turn is a block: a thick ▌ accent left border,
// left padding, and a subtle per-theme fill (renderEntry above). An ASSISTANT turn keeps the role
// label, then steps, the body bubble (glamour once finalized, plain while streaming), any
// interrupted/error badge, and — for a finished turn — a faint per-turn footer, all prefixed with
// a faint thin │ gutter and NO background fill (answers stay on the terminal background for maximum
// readability). Both role treatments survive NO_COLOR: lipgloss strips the color but the ▌ / │
// glyphs remain, so turns stay visually separated colorless.
func (m model) renderEntry(e entry) string {
	switch e.role {
	case roleUser:
		// A user turn is a BLOCK: a thick ▌ accent left border + left padding + a subtle block
		// fill spanning the capped measure, so it reads as a raised surface distinct from the
		// header/footer bars. The label and text carry the block fill too, so the tint is
		// continuous behind the text rather than only in the trailing pad. Under NO_COLOR the
		// fill and border color drop via the color profile, but the ▌ glyph remains as structure.
		inner := m.styles.userLabel.Render("you") + "\n" + m.styles.user.Render(e.text)
		// Width sets only a MINIMUM; MaxWidth caps it so an unbreakable long token (a URL, a
		// pasted path) wraps inside the measure instead of stretching the block past it.
		return m.styles.userBlock.Width(m.contentW).MaxWidth(m.contentW).Render(inner)

	case roleInfo:
		// Ambient host guidance (the one-time launch nudge): a faint italic single line with no role
		// label or gutter, so it reads as chrome rather than a conversation turn.
		return m.styles.infoLine.Render(e.text)

	default: // roleAssistant
		var b strings.Builder
		b.WriteString(m.styles.roleLabel.Render("librarian") + "\n")
		if steps := renderSteps(e.steps, m.styles, m.showSteps); steps != "" {
			b.WriteString(steps)
		}
		for _, raw := range e.rawLines {
			b.WriteString(m.styles.raw.Render(raw) + "\n")
		}
		if e.text != "" {
			var body string
			if e.finalized {
				// Finalized: glamour (its "dark" style carries its own high-contrast colors).
				body = strings.TrimRight(renderMarkdown(m.renderer, e.text), "\n")
			} else {
				// Streaming: plain text at an explicit bright foreground, so the answer never
				// relies on the terminal's default (which can be a low-contrast gray) and stays
				// clearly brighter than the faint step lines.
				body = m.styles.assistant.Render(e.text)
			}
			b.WriteString(body)
		}
		if e.interrupted {
			if e.text != "" {
				b.WriteString("\n")
			}
			b.WriteString(m.styles.interrupted.Render("(interrupted)"))
		} else if e.isError {
			if e.text != "" {
				b.WriteString("\n")
			}
			b.WriteString(m.styles.errText.Render("error: " + e.errText))
		}
		// Per-turn footer: only on a finished turn with a recorded wall time. A still-streaming turn
		// (duration zero) and a resumed history entry (no timing) show none. Per-turn tokens join the
		// existing `model · latency` line (no new line) when the provider reported a completion count.
		if e.finalized && e.duration > 0 {
			footer := m.llmModel + " · " + fmtDuration(e.duration)
			if e.tokens > 0 {
				footer += " · " + fmtTokens(e.tokens) + " tok"
			}
			b.WriteString("\n" + m.styles.turnFooter.Render(footer))
		}
		return applyGutter(b.String(), m.styles.assistantGutter, "│")
	}
}

// applyGutter prefixes every line of an entry's rendered content with a colored left-gutter glyph,
// separating turns visually. The glyph flows through a lipgloss style, so NO_COLOR degrades it to
// the bare glyph — which still separates the turn — rather than losing the border entirely.
func applyGutter(content string, style lipgloss.Style, glyph string) string {
	prefix := style.Render(glyph) + " "
	// TrimRight the trailing renderer newlines first, so they don't split into empty lines that
	// would each render as an orphan line carrying only the gutter glyph. Embedded newlines
	// (blank lines WITHIN the content) are preserved — only a trailing run is dropped.
	lines := strings.Split(strings.TrimRight(content, "\n"), "\n")
	for i, ln := range lines {
		lines[i] = prefix + ln
	}
	return strings.Join(lines, "\n")
}

// fmtDuration renders a turn's wall time compactly: sub-minute as seconds with one decimal
// (e.g. "4.2s"), a minute or more as "1m03s".
func fmtDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	m := int(d / time.Minute)
	s := int((d % time.Minute) / time.Second)
	return fmt.Sprintf("%dm%02ds", m, s)
}

// View composes the surface: header, transcript viewport, input textarea, footer. Before the
// first WindowSizeMsg it shows a minimal placeholder (bubbletea sends the size immediately).
//
// v2's Model.View returns a tea.View rather than a bare string; AltScreen is now a per-frame View
// field (not a tea.WithAltScreen ProgramOption), and every return path below sets it — a path that
// forgot it would silently drop the surface out of the alternate screen mid-run. BackgroundColor is
// deliberately left nil in both paths: the TUI renders on the terminal's own background (ADR 0004 /
// ADR 0007), never overriding it.
func (m model) View() tea.View {
	if !m.ready {
		v := tea.NewView("initializing…")
		v.AltScreen = true
		return v
	}
	// When the resume overlay is open it renders in place of the transcript viewport; the header,
	// input, and footer stay put so the surface never loses its frame. An active module view
	// (spec §5.3) occupies the same body region, padded to the viewport height so the input box
	// and footer never jump.
	body := m.vp.View()
	if m.picker != nil {
		body = m.picker.View()
	} else if m.activeView >= 0 && m.activeView < len(m.views) {
		body = padToHeight(m.views[m.activeView].Render(), m.vp.Height())
	}
	// The view-switcher strip sits directly under the header, but only on a desk that mounted module
	// views — so a librarian-only surface is byte-identical to before. resize reserves its row in the
	// body-height math, so the input box and footer never get pushed off-screen.
	parts := make([]string, 0, 5)
	parts = append(parts, m.renderHeader())
	if len(m.views) > 0 {
		parts = append(parts, m.renderTabs())
	}
	parts = append(parts, body, m.renderInput(), m.renderFooter())
	content := lipgloss.JoinVertical(lipgloss.Left, parts...)
	v := tea.NewView(content)
	v.AltScreen = true
	return v
}

// renderHeader renders `desk · provider/model` as a full-width bar: a subtle per-theme background
// fill spanning the whole terminal width, bold accent desk name, muted provider/model. The bar
// segments each carry the fill so it is continuous behind the text; the outer bar pads the rest
// and truncates to width so the one-line bar never wraps.
func (m model) renderHeader() string {
	left := m.styles.headerAccent.Render(m.deskName)
	right := m.styles.header.Render(" · " + m.llmProvider + "/" + m.llmModel)
	seg := left + right
	// Trailing usage segment: `NN% ctx · 12.3K tok`, once a turn has reported a prompt count and
	// the window is known. Appended, not width-computed — the outer bar's MaxWidth still truncates
	// the one-line bar, so a narrow terminal drops the tail rather than wrapping.
	if m.ctxWindow > 0 && m.ctxTokens > 0 {
		pct := m.ctxTokens * 100 / m.ctxWindow
		if pct < 0 {
			pct = 0
		}
		if pct > 100 {
			pct = 100
		}
		seg += m.styles.headerToken.Render(fmt.Sprintf("  %d%% ctx · %s tok", pct, fmtTokens(m.ctxTokens)))
	}
	return m.styles.headerBar.Width(m.width).MaxWidth(m.width).Render(seg)
}

// renderTabs renders the persistent view-switcher strip — `chat · <view1> · <view2> · …` — as a
// full-width bar directly under the header, with the ACTIVE segment highlighted (chat is active when
// activeView == -1; otherwise views[activeView]). It is the always-visible affordance that makes
// every mounted module view discoverable on sight, without pressing ctrl+p. Only ever rendered when
// views are mounted (View gates it), so a librarian-only desk never shows it. The segments each carry
// the bar fill so the tint is continuous behind the text; the outer bar truncates to width so the
// one-line strip never wraps.
func (m model) renderTabs() string {
	seg := func(label string, active bool) string {
		if active {
			return m.styles.tabActive.Render(label)
		}
		return m.styles.tabInactive.Render(label)
	}
	parts := make([]string, 0, len(m.views)+1)
	parts = append(parts, seg("chat", m.activeView == -1))
	for i, v := range m.views {
		parts = append(parts, seg(v.Name(), i == m.activeView))
	}
	strip := strings.Join(parts, m.styles.tabSep.Render(" · "))
	return m.styles.tabBar.Width(m.width).MaxWidth(m.width).Render(strip)
}

// renderInput renders the textarea wrapped in a rounded, full-width border box. The border color
// is a state cue, not a lock: accent when ready for input, faint while a turn is streaming (typing
// still works throughout). The textarea was sized to width-2 in resize so the box spans the full
// terminal width.
func (m model) renderInput() string {
	border := m.styles.inputBorder
	if m.streaming {
		border = m.styles.inputBorderBusy
	}
	return border.Render(m.ta.View())
}

// renderFooter renders the self-documenting keybind help (bubbles/help) plus the live state
// segment. When ctrl+g has toggled ShowAll, the grouped full help renders as an un-barred overlay
// on the terminal background (a multi-line list, kept legible). Otherwise it is a full-width status
// bar: a subtle per-theme fill matching the header bar, with the help hints on the left and the
// live state on the right. The state shows a transient toast when set (copy confirmation / error);
// otherwise the streaming/cancelling/ready indicator, with a "▼ new output" hint appended while the
// reader has scrolled up mid-stream. Under reduced motion the animated spinner is replaced by
// static "working…" text. The bar segments carry the fill so it is continuous behind the text.
func (m model) renderFooter() string {
	// ctrl+g / ? expansion: a grouped overlay list on the terminal background, not a bar. Checked
	// first so the overlay surfaces in BOTH chat and view mode (? opens the same help everywhere).
	if m.hlp.ShowAll {
		hlp := m.hlp
		hlp.Styles = m.styles.help
		return hlp.View(m.keymap)
	}
	// An active module view gets its own footer (name + switcher hints; host_views.go).
	if m.activeView >= 0 {
		return m.viewFooter()
	}

	var state string
	switch {
	case m.toast != "":
		state = m.styles.toast.Render("  " + m.toast)
	case m.cancelling:
		state = m.styles.footerState.Render("  cancelling…")
	case m.streaming:
		work := "working…"
		if !m.reduced {
			work = m.sp.View() + " streaming"
		}
		if !m.vp.AtBottom() {
			work += "  ▼ new output"
		}
		state = m.styles.footerState.Render("  " + work)
	default:
		state = m.styles.footerState.Render("  ready")
	}

	// Reserve room for the always-visible state segment; the help hints fill whatever remains and
	// truncate themselves to fit (bubbles/help), so the live state is never clipped off the end.
	// The help uses the barred palette so its segments sit on the status-bar fill.
	hlp := m.hlp
	hlp.Styles = m.styles.helpBar
	if w := m.width - lipgloss.Width(state); w > 0 {
		hlp.SetWidth(w)
	}
	hints := hlp.View(m.keymap)
	return m.styles.footerBar.Width(m.width).MaxWidth(m.width).Render(hints + state)
}

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
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/pocketbase/pocketbase/core"

	"github.com/example/pocket-librarian/internal/agent"
	"github.com/example/pocket-librarian/internal/config"
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
// resumable runs, resuming one, opening a fresh one, and closing a session. Keeping it an
// interface means Update never imports agent.ListConversations / ResumeSession / NewSession
// directly, so the picker + session-swap paths are unit-testable with a fake (picker_test.go).
type sessionProvider interface {
	list(limit int, excludeRunID string) ([]agent.ConversationInfo, error)
	resume(ctx context.Context, runID string) (streamer, []agent.TranscriptEntry, error)
	fresh(ctx context.Context) (streamer, error)
	closeSession(ctx context.Context, s streamer) error
}

// agentProvider is the production sessionProvider: a thin adapter over the agent package bound to
// the app and resolved config. It is separate from the model's cfg (which the header still needs).
type agentProvider struct {
	app core.App
	cfg *config.Config
}

// list returns the most recent resumable conversations for the picker, excluding the caller's
// own live run.
func (p *agentProvider) list(limit int, excludeRunID string) ([]agent.ConversationInfo, error) {
	return agent.ListConversations(p.app, limit, excludeRunID)
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

// Layout constants: the header and footer are one full-width line each; the input textarea is a
// fixed inner height wrapped in a rounded border (inputBorderHeight rows: one top, one bottom).
// The viewport takes whatever height remains.
const (
	headerHeight      = 1
	footerHeight      = 1
	inputHeight       = 3
	inputBorderHeight = 2 // rounded input box: top + bottom border rows added around the textarea
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
// copies the last answer's raw markdown; help (ctrl+g) toggles the expanded help. Bare "?" is
// deliberately NOT bound — the textarea is always focused and must type it.
type keymap struct {
	send            key.Binding
	newline         key.Binding
	cancel          key.Binding
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
}

func defaultKeymap() keymap {
	return keymap{
		send:            key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "send")),
		newline:         key.NewBinding(key.WithKeys("alt+enter"), key.WithHelp("alt+enter", "newline")),
		cancel:          key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
		toggleSteps:     key.NewBinding(key.WithKeys("ctrl+t"), key.WithHelp("ctrl+t", "steps")),
		scrollUp:        key.NewBinding(key.WithKeys("pgup"), key.WithHelp("pgup", "scroll up")),
		scrollDown:      key.NewBinding(key.WithKeys("pgdown"), key.WithHelp("pgdn", "scroll down")),
		historyPrev:     key.NewBinding(key.WithKeys("up"), key.WithHelp("↑", "prev prompt")),
		historyNext:     key.NewBinding(key.WithKeys("down"), key.WithHelp("↓", "next prompt")),
		copyLast:        key.NewBinding(key.WithKeys("ctrl+y"), key.WithHelp("ctrl+y", "copy answer")),
		help:            key.NewBinding(key.WithKeys("ctrl+g"), key.WithHelp("ctrl+g", "help")),
		quit:            key.NewBinding(key.WithKeys("ctrl+c"), key.WithHelp("ctrl+c", "quit")),
		openPicker:      key.NewBinding(key.WithKeys("ctrl+o"), key.WithHelp("ctrl+o", "resume")),
		newConversation: key.NewBinding(key.WithKeys("ctrl+n"), key.WithHelp("ctrl+n", "new")),
	}
}

// ShortHelp is the one-line help footer's binding order (bubbles/help KeyMap). It surfaces the
// everyday keys; the rest live under the ctrl+g full-help expansion.
func (k keymap) ShortHelp() []key.Binding {
	return []key.Binding{k.send, k.openPicker, k.newConversation, k.copyLast, k.toggleSteps, k.help, k.quit}
}

// FullHelp is the grouped expansion shown when ctrl+g toggles ShowAll (bubbles/help KeyMap),
// columns roughly by function: composing, conversation, transcript, meta.
func (k keymap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.send, k.newline, k.cancel},
		{k.openPicker, k.newConversation, k.copyLast},
		{k.toggleSteps, k.scrollUp, k.scrollDown, k.historyPrev, k.historyNext},
		{k.help, k.quit},
	}
}

// model is the chat surface. picker is the conversation-resume overlay: nil when closed, and it
// only ever lives while NOT streaming (ctrl+o opens it only when no turn is in flight). provider
// is the session lifecycle boundary (list/resume/fresh/close), an interface so Update stays
// testable without a real Session.
type model struct {
	baseCtx  context.Context
	sess     streamer
	provider sessionProvider

	deskName    string
	llmProvider string
	llmModel    string

	theme    string // concrete resolved palette ("light"/"dark"), never "auto"
	styles   styleSet
	keymap   keymap
	hlp      help.Model
	ta       textarea.Model
	vp       viewport.Model
	sp       spinner.Model
	renderer *glamour.TermRenderer

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
		hlp:         hlp,
		ta:          ta,
		vp:          viewport.New(0, 0),
		sp:          sp,
		history:     newHistory(nil),
		reduced:     reducedMotion(os.Getenv("NO_COLOR")),
		inflightIdx: -1,
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
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

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
func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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

	case key.Matches(msg, m.keymap.openPicker):
		return m.openPicker()

	case key.Matches(msg, m.keymap.newConversation):
		return m.newConversation()

	case key.Matches(msg, m.keymap.copyLast):
		return m.copyLastAnswer()

	case key.Matches(msg, m.keymap.help):
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

// openPicker opens the conversation-resume overlay. It is a no-op while a turn is streaming (so no
// in-flight turn is ever disturbed) or when the picker is already open. It lists the recent
// conversations synchronously — a fast local DB read — and sizes the overlay to the viewport area.
// A list error leaves the picker closed rather than crashing the surface.
func (m model) openPicker() (tea.Model, tea.Cmd) {
	if m.streaming || m.picker != nil {
		return m, nil
	}
	convos, err := m.provider.list(pickerLimit, m.sess.RunID())
	if err != nil {
		// Degraded: keep the surface open without a picker, but say why — a silent ctrl+o that
		// does nothing reads as a dead keybinding, not a failed lookup.
		m.entries = append(m.entries, entry{
			role: roleAssistant, isError: true, finalized: true,
			errText: "could not list conversations: " + err.Error(),
		})
		m.refreshViewport()
		return m, nil
	}
	m.picker = newPicker(convos, m.vp.Width, m.vp.Height)
	return m, nil
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
	m.history = newHistory(nil) // fresh conversation: no prior prompts to recall
	m.picker = nil
	m.refreshViewport()
	return m, nil
}

// handlePickerKey routes a keypress while the overlay is open: esc closes it, enter resumes the
// selected conversation, and any other key drives the inner list (cursor movement). Because the
// picker only lives while NOT streaming (ctrl+o/ctrl+n are no-ops mid-turn), there is never an
// in-flight turn to drain at swap time, so closing the current session before opening the resumed
// one is safe immediately — this is how the picker path respects the engine's drain contract.
func (m model) handlePickerKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keymap.cancel):
		// esc closes the overlay. There is no in-flight turn to cancel here (the picker only opens
		// while not streaming), so esc is purely "dismiss the overlay".
		m.picker = nil
		m.refreshViewport()
		return m, nil

	case key.Matches(msg, m.keymap.send):
		runID := m.picker.selectedRunID()
		if runID == "" {
			return m, nil
		}
		// Open-before-close (same reasoning as newConversation): resume FIRST, and close the
		// current session only once the replacement exists, so a failed resume leaves the old
		// session genuinely live rather than finalized out from under the user.
		newSess, transcript, err := m.provider.resume(m.baseCtx, runID)
		if err != nil {
			// Degraded: resume failed. The old session was not touched; dismiss the overlay and
			// say why instead of silently doing nothing.
			m.picker = nil
			m.entries = append(m.entries, entry{
				role: roleAssistant, isError: true, finalized: true,
				errText: "could not resume conversation: " + err.Error(),
			})
			m.refreshViewport()
			return m, nil
		}
		_ = m.provider.closeSession(m.baseCtx, m.sess)
		m.sess = newSess
		m.entries = entriesFromTranscript(transcript)
		m.inflightIdx = -1
		m.history = newHistory(userPromptsNewestFirst(m.entries)) // resume seeds prompt recall
		m.picker = nil
		m.refreshViewport()
		return m, nil

	default:
		cmd := m.picker.Update(msg)
		return m, cmd
	}
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
	default:
		e.rawLines = append(e.rawLines, fmt.Sprintf("unexpected event: %+v", ev))
	}
}

// resize recomputes the layout and rebuilds the markdown renderer to the new viewport width
// (a renderer keeps its word-wrap, so it must be rebuilt on every size change).
func (m *model) resize(w, h int) {
	m.width = w
	m.height = h
	m.contentW = measureWidth(w)
	m.ready = true

	vpHeight := h - headerHeight - footerHeight - inputHeight - inputBorderHeight
	if vpHeight < 1 {
		vpHeight = 1
	}
	m.vp.Width = w
	m.vp.Height = vpHeight
	// The textarea sits inside the rounded input box; reserve the two columns its left/right
	// border consume so the box spans the full terminal width without wrapping.
	m.ta.SetWidth(w - 2)
	m.ta.SetHeight(inputHeight)
	m.hlp.Width = w // the help footer truncates its short view to the terminal width
	// Glamour wraps at the CAPPED measure, not the raw width — the transcript keeps a readable
	// line length on a wide terminal even though the bars span the whole screen.
	m.renderer = newRenderer(m.contentW, m.theme)
	if m.picker != nil {
		m.picker.SetSize(w, vpHeight) // the overlay occupies the viewport area
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
		return m.styles.userBlock.Width(m.contentW).Render(inner)

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
		// (duration zero) and a resumed history entry (no timing) show none.
		if e.finalized && e.duration > 0 {
			b.WriteString("\n" + m.styles.turnFooter.Render(m.llmModel+" · "+fmtDuration(e.duration)))
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
func (m model) View() string {
	if !m.ready {
		return "initializing…"
	}
	// When the resume overlay is open it renders in place of the transcript viewport; the header,
	// input, and footer stay put so the surface never loses its frame.
	body := m.vp.View()
	if m.picker != nil {
		body = m.picker.View()
	}
	return lipgloss.JoinVertical(lipgloss.Left,
		m.renderHeader(),
		body,
		m.renderInput(),
		m.renderFooter(),
	)
}

// renderHeader renders `desk · provider/model` as a full-width bar: a subtle per-theme background
// fill spanning the whole terminal width, bold accent desk name, muted provider/model. The bar
// segments each carry the fill so it is continuous behind the text; the outer bar pads the rest
// and truncates to width so the one-line bar never wraps.
func (m model) renderHeader() string {
	left := m.styles.headerAccent.Render(m.deskName)
	right := m.styles.header.Render(" · " + m.llmProvider + "/" + m.llmModel)
	return m.styles.headerBar.Width(m.width).MaxWidth(m.width).Render(left + right)
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
	// ctrl+g expansion: a grouped overlay list on the terminal background, not a bar.
	if m.hlp.ShowAll {
		hlp := m.hlp
		hlp.Styles = m.styles.help
		return hlp.View(m.keymap)
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
		hlp.Width = w
	}
	hints := hlp.View(m.keymap)
	return m.styles.footerBar.Width(m.width).MaxWidth(m.width).Render(hints + state)
}

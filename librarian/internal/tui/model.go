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
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"

	"github.com/example/pocket-librarian/internal/agent"
	"github.com/example/pocket-librarian/internal/config"
)

// streamer is the model's view of an agent.Session: the single method it needs to drive a turn.
// *agent.Session satisfies it; a fake satisfies it in tests, so the model needs no real Session,
// DB, or LLM to unit-test Update.
type streamer interface {
	StreamTurn(ctx context.Context, userInput string) <-chan agent.Event
}

// Layout constants: the header and footer are one line each; the input textarea is a fixed
// height. The viewport takes whatever height remains.
const (
	headerHeight = 1
	footerHeight = 1
	inputHeight  = 3
)

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

// keymap is the surface's binding table — the single place keys are declared, which is also the
// Phase-4 mount seam: the conversation picker (ctrl+o) and new-conversation (ctrl+n) bindings
// belong here when that phase lands. They are intentionally NOT declared or handled yet.
type keymap struct {
	send        key.Binding
	newline     key.Binding
	cancel      key.Binding
	toggleSteps key.Binding
	scrollUp    key.Binding
	scrollDown  key.Binding
	quit        key.Binding
	// Phase 4 (do not build here): openPicker key.Binding // ctrl+o
	//                              newConversation key.Binding // ctrl+n
}

func defaultKeymap() keymap {
	return keymap{
		send:        key.NewBinding(key.WithKeys("enter")),
		newline:     key.NewBinding(key.WithKeys("alt+enter")),
		cancel:      key.NewBinding(key.WithKeys("esc")),
		toggleSteps: key.NewBinding(key.WithKeys("ctrl+t")),
		scrollUp:    key.NewBinding(key.WithKeys("pgup")),
		scrollDown:  key.NewBinding(key.WithKeys("pgdown")),
		quit:        key.NewBinding(key.WithKeys("ctrl+c")),
	}
}

// model is the chat surface. picker is the Phase-4 overlay mount point — always nil in Phases
// 2-3; View reserves the slot where it will render.
type model struct {
	baseCtx context.Context
	sess    streamer

	deskName string
	provider string
	llmModel string

	styles   styleSet
	keymap   keymap
	ta       textarea.Model
	vp       viewport.Model
	sp       spinner.Model
	renderer *glamour.TermRenderer

	entries     []entry
	inflightIdx int // index of the assistant entry being streamed (valid while streaming)

	streaming  bool
	cancelling bool
	quitting   bool
	showSteps  bool

	cancelTurn context.CancelFunc
	events     <-chan agent.Event

	width  int
	height int
	ready  bool

	// picker *pickerModel // Phase 4 — conversation picker overlay (ctrl+o); nil until then.
}

// newModel builds the chat model against an injected streamer (the real *agent.Session in
// production, a fake in tests) and the resolved config for the header.
func newModel(baseCtx context.Context, sess streamer, cfg *config.Config) model {
	ta := textarea.New()
	ta.Placeholder = "Ask the librarian… (enter to send, alt+enter for a newline)"
	ta.Prompt = "▏ "
	ta.ShowLineNumbers = false
	ta.CharLimit = 0
	// Plain enter is the send key (handled in Update); rebind the textarea's own newline to
	// alt+enter so it never swallows the send key.
	ta.KeyMap.InsertNewline = key.NewBinding(key.WithKeys("alt+enter"))
	ta.Focus()

	sp := spinner.New(spinner.WithSpinner(spinner.Dot))

	return model{
		baseCtx:  baseCtx,
		sess:     sess,
		deskName: cfg.DeskName,
		provider: cfg.LLMProvider,
		llmModel: cfg.LLMModel,
		styles:   newStyles(),
		keymap:   defaultKeymap(),
		ta:       ta,
		vp:       viewport.New(0, 0),
		sp:       sp,
	}
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
		}
		m.refreshViewport()
		if m.quitting {
			return m, tea.Quit
		}
		return m, nil

	case spinner.TickMsg:
		if !m.streaming {
			return m, nil
		}
		var cmd tea.Cmd
		m.sp, cmd = m.sp.Update(msg)
		return m, cmd
	}
	return m, nil
}

// handleKey routes a keypress. Send/cancel/quit/steps are intercepted; scrolling goes to the
// viewport; everything else (typing, alt+enter newline) goes to the textarea.
func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keymap.quit):
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

	case key.Matches(msg, m.keymap.cancel):
		// esc cancels an in-flight turn (footer → "cancelling…"); the pump drains to the terminal
		// event, which may take a moment because eino cannot abort a stuck provider stream.
		// Phase-4 seam: when a picker overlay is open, esc closes it instead.
		if m.streaming && m.cancelTurn != nil {
			m.cancelTurn()
			m.cancelling = true
		}
		return m, nil

	case key.Matches(msg, m.keymap.toggleSteps):
		m.showSteps = !m.showSteps
		m.refreshViewport()
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

// startTurn opens one streaming turn: derive a cancelable turn context, hand the input to the
// engine, append the user entry and an empty in-flight assistant entry, and kick off the pump
// plus the spinner. Returns the model and its initial commands.
func (m model) startTurn(input string) (tea.Model, tea.Cmd) {
	turnCtx, cancel := context.WithCancel(m.baseCtx)
	ch := m.sess.StreamTurn(turnCtx, input)

	m.entries = append(m.entries, entry{role: roleUser, text: input, finalized: true})
	m.entries = append(m.entries, entry{role: roleAssistant})
	m.inflightIdx = len(m.entries) - 1

	m.events = ch
	m.cancelTurn = cancel
	m.streaming = true
	m.cancelling = false
	m.ta.Reset()
	m.refreshViewport()
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
	m.ready = true

	vpHeight := h - headerHeight - footerHeight - inputHeight
	if vpHeight < 1 {
		vpHeight = 1
	}
	m.vp.Width = w
	m.vp.Height = vpHeight
	m.ta.SetWidth(w)
	m.ta.SetHeight(inputHeight)
	m.renderer = newRenderer(w)
	m.refreshViewport()
}

// refreshViewport re-renders the transcript into the viewport, preserving the user's scroll
// position unless they were already at the bottom (in which case new output auto-scrolls).
func (m *model) refreshViewport() {
	if !m.ready {
		return
	}
	atBottom := m.vp.AtBottom()
	m.vp.SetContent(m.renderTranscript())
	if atBottom {
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

// renderEntry renders one transcript item: the role label, then steps (assistant only), the
// body bubble (glamour once finalized, plain while streaming), and any interrupted/error badge.
func (m model) renderEntry(e entry) string {
	switch e.role {
	case roleUser:
		return m.styles.roleLabel.Render("you") + "\n" + m.styles.user.Render(e.text)

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
		return b.String()
	}
}

// View composes the surface: header, transcript viewport, input textarea, footer. Before the
// first WindowSizeMsg it shows a minimal placeholder (bubbletea sends the size immediately).
func (m model) View() string {
	if !m.ready {
		return "initializing…"
	}
	// Phase-4 seam: when a picker overlay is active it will render in place of the viewport here.
	return lipgloss.JoinVertical(lipgloss.Left,
		m.renderHeader(),
		m.vp.View(),
		m.ta.View(),
		m.renderFooter(),
	)
}

// renderHeader renders `desk · provider/model`, clipped to the terminal width.
func (m model) renderHeader() string {
	left := m.styles.headerAccent.Render(m.deskName)
	right := m.styles.header.Render(" · " + m.provider + "/" + m.llmModel)
	return clip(left+right, m.width)
}

// renderFooter renders the keybind hints and the live state (streaming spinner / cancelling…
// / ready), clipped to the terminal width.
func (m model) renderFooter() string {
	hints := "enter send · alt+enter newline · ctrl+t steps · pgup/pgdn scroll · esc cancel · ctrl+c quit"
	var state string
	switch {
	case m.cancelling:
		state = "cancelling…"
	case m.streaming:
		state = m.sp.View() + " streaming"
	default:
		state = "ready"
	}
	line := m.styles.footer.Render(hints) + "  " + m.styles.footerState.Render(state)
	return clip(line, m.width)
}

// clip hard-limits a rendered line to width display cells, so header/footer never wrap.
func clip(s string, width int) string {
	if width <= 0 {
		return s
	}
	return lipgloss.NewStyle().MaxWidth(width).Render(s)
}

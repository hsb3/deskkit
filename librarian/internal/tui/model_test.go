package tui

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/example/pocket-librarian/internal/agent"
	"github.com/example/pocket-librarian/internal/config"
)

// fakeStreamer is the injected session boundary: it captures the turn context (so a test can
// prove esc cancels it) and hands back a channel the test does NOT need to feed — pure-Update
// tests deliver engine events as eventMsg/turnDoneMsg directly, without a real Session or LLM.
type fakeStreamer struct {
	ch    chan agent.Event
	ctx   context.Context
	input string
	calls int
}

func (f *fakeStreamer) StreamTurn(ctx context.Context, userInput string) <-chan agent.Event {
	f.ctx = ctx
	f.input = userInput
	f.calls++
	if f.ch == nil {
		f.ch = make(chan agent.Event, 64)
	}
	return f.ch
}

// newTestModel builds a model sized to an 80x24 terminal and ready to drive.
func newTestModel(t *testing.T) (model, *fakeStreamer) {
	t.Helper()
	cfg := &config.Config{DeskName: "test-desk", LLMProvider: "openai", LLMModel: "test-model"}
	fs := &fakeStreamer{}
	m := newModel(context.Background(), fs, cfg)
	next, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	return next.(model), fs
}

// send routes one message through Update and returns the resulting model (dropping the command:
// these are pure-Update tests, the pump/spinner commands are never executed).
func send(m model, msg tea.Msg) model {
	next, _ := m.Update(msg)
	return next.(model)
}

// startStreaming enters one turn with the given input and returns the streaming model.
func startStreaming(t *testing.T, m model, input string) model {
	t.Helper()
	m.ta.SetValue(input)
	return send(m, tea.KeyMsg{Type: tea.KeyEnter})
}

func TestWindowSizeMsg_Sizing(t *testing.T) {
	m, _ := newTestModel(t)
	if !m.ready {
		t.Fatal("model not marked ready after WindowSizeMsg")
	}
	if m.vp.Width != 80 {
		t.Errorf("viewport width = %d, want 80", m.vp.Width)
	}
	// height 24 - header(1) - footer(1) - input(3) = 19
	if m.vp.Height != 19 {
		t.Errorf("viewport height = %d, want 19", m.vp.Height)
	}
	if m.renderer == nil {
		t.Error("markdown renderer not built on WindowSizeMsg")
	}
}

func TestEnter_StartsTurnAndCapturesInput(t *testing.T) {
	m, fs := newTestModel(t)
	m = startStreaming(t, m, "hello there")
	if !m.streaming {
		t.Fatal("streaming flag not set after enter")
	}
	if fs.calls != 1 {
		t.Fatalf("StreamTurn calls = %d, want 1", fs.calls)
	}
	if fs.input != "hello there" {
		t.Errorf("streamed input = %q, want %q", fs.input, "hello there")
	}
	// user entry + in-flight assistant entry
	if len(m.entries) != 2 || m.entries[0].role != roleUser || m.entries[1].role != roleAssistant {
		t.Fatalf("entries = %+v, want [user, assistant]", m.entries)
	}
	if m.entries[0].text != "hello there" {
		t.Errorf("user entry text = %q", m.entries[0].text)
	}
	if got := m.ta.Value(); got != "" {
		t.Errorf("textarea not reset after send, value = %q", got)
	}
}

func TestTokenAccumulation(t *testing.T) {
	m, _ := newTestModel(t)
	m = startStreaming(t, m, "q")
	m = send(m, eventMsg{ev: agent.Event{Kind: agent.EventToken, Step: 1, Token: "Hel"}})
	m = send(m, eventMsg{ev: agent.Event{Kind: agent.EventToken, Step: 1, Token: "lo"}})
	if got := m.entries[m.inflightIdx].text; got != "Hello" {
		t.Errorf("accumulated answer = %q, want %q", got, "Hello")
	}
}

func TestToolStep_SuccessAttachment(t *testing.T) {
	m, _ := newTestModel(t)
	m = startStreaming(t, m, "q")
	m = send(m, eventMsg{ev: agent.Event{Kind: agent.EventToken, Step: 1, Token: "let me check"}})
	m = send(m, eventMsg{ev: agent.Event{Kind: agent.EventToolStart, Tool: "query", CallID: "c1", Args: `{"kind":"summary"}`}})
	m = send(m, eventMsg{ev: agent.Event{Kind: agent.EventToolEnd, Tool: "query", CallID: "c1", Result: `{"ok":true}`}})

	e := m.entries[m.inflightIdx]
	if len(e.steps) != 1 {
		t.Fatalf("steps = %d, want 1", len(e.steps))
	}
	s := e.steps[0]
	if !s.done || s.failed {
		t.Errorf("success step: done=%v failed=%v, want done=true failed=false", s.done, s.failed)
	}
	if s.result != `{"ok":true}` {
		t.Errorf("step result = %q", s.result)
	}
	// pre-tool tokens retagged as commentary, cleared from the answer bubble
	if s.commentary != "let me check" {
		t.Errorf("step commentary = %q, want %q", s.commentary, "let me check")
	}
	if e.text != "" {
		t.Errorf("answer bubble not reset after tool_start, text = %q", e.text)
	}
}

func TestToolStep_ErrorBadgeState(t *testing.T) {
	m, _ := newTestModel(t)
	m = startStreaming(t, m, "q")
	m = send(m, eventMsg{ev: agent.Event{Kind: agent.EventToolStart, Tool: "apply_fix", CallID: "c9", Args: "{}"}})
	m = send(m, eventMsg{ev: agent.Event{Kind: agent.EventToolEnd, Tool: "apply_fix", CallID: "c9", Err: "store is read-only"}})

	s := m.entries[m.inflightIdx].steps[0]
	if !s.done || !s.failed {
		t.Errorf("failed step: done=%v failed=%v, want both true", s.done, s.failed)
	}
	if s.errText != "store is read-only" {
		t.Errorf("step errText = %q", s.errText)
	}
	// the ✗ badge must appear in the rendered transcript
	if !strings.Contains(m.renderTranscript(), "✗") {
		t.Error("failed-tool ✗ badge not rendered")
	}
}

func TestTurnDone_Transition(t *testing.T) {
	m, _ := newTestModel(t)
	m = startStreaming(t, m, "q")
	m = send(m, eventMsg{ev: agent.Event{Kind: agent.EventToken, Step: 1, Token: "answer"}})
	m = send(m, eventMsg{ev: agent.Event{Kind: agent.EventFinal, Content: "answer"}})
	idx := m.inflightIdx
	m = send(m, turnDoneMsg{})

	if m.streaming {
		t.Error("streaming still set after turnDoneMsg")
	}
	if m.cancelTurn != nil {
		t.Error("cancelTurn not cleared after turnDoneMsg")
	}
	if m.events != nil {
		t.Error("events channel not cleared after turnDoneMsg")
	}
	if !m.entries[idx].finalized {
		t.Error("assistant entry not finalized after turnDoneMsg")
	}
}

func TestCtrlT_TogglesStepsInOutput(t *testing.T) {
	m, _ := newTestModel(t)
	m = startStreaming(t, m, "q")
	m = send(m, eventMsg{ev: agent.Event{Kind: agent.EventToolStart, Tool: "query", CallID: "c1", Args: `{"kind":"summary"}`}})
	m = send(m, eventMsg{ev: agent.Event{Kind: agent.EventToolEnd, Tool: "query", CallID: "c1", Result: "ok"}})
	m = send(m, eventMsg{ev: agent.Event{Kind: agent.EventFinal, Content: "done"}})
	m = send(m, turnDoneMsg{})

	before := m.View()
	if strings.Contains(m.renderTranscript(), "args:") {
		t.Fatal("expanded step detail shown before ctrl+t")
	}
	m = send(m, tea.KeyMsg{Type: tea.KeyCtrlT})
	after := m.View()

	if !m.showSteps {
		t.Error("showSteps not toggled on")
	}
	if !strings.Contains(m.renderTranscript(), "args:") {
		t.Error("expanded step detail not shown after ctrl+t")
	}
	if before == after {
		t.Error("View() output unchanged by ctrl+t")
	}
}

func TestEsc_CancelsInFlightTurn(t *testing.T) {
	m, fs := newTestModel(t)
	m = startStreaming(t, m, "q")
	if fs.ctx.Err() != nil {
		t.Fatal("turn context already canceled before esc")
	}
	m = send(m, tea.KeyMsg{Type: tea.KeyEsc})
	if fs.ctx.Err() != context.Canceled {
		t.Errorf("turn context err after esc = %v, want context.Canceled", fs.ctx.Err())
	}
	if !m.cancelling {
		t.Error("cancelling flag not set after esc")
	}
	if !m.streaming {
		t.Error("streaming should remain true until the terminal event drains")
	}
}

func TestEnterWhileStreaming_NoOp(t *testing.T) {
	m, fs := newTestModel(t)
	m = startStreaming(t, m, "first")
	entriesBefore := len(m.entries)

	m.ta.SetValue("second")
	m = send(m, tea.KeyMsg{Type: tea.KeyEnter})

	if fs.calls != 1 {
		t.Errorf("StreamTurn called %d times, want 1 (enter must be a no-op while streaming)", fs.calls)
	}
	if len(m.entries) != entriesBefore {
		t.Errorf("entries changed while streaming: %d -> %d", entriesBefore, len(m.entries))
	}
}

func TestErrorEvent_InterruptedBadge(t *testing.T) {
	m, _ := newTestModel(t)
	m = startStreaming(t, m, "q")
	m = send(m, eventMsg{ev: agent.Event{Kind: agent.EventError, Canceled: true, Partial: "half an answer"}})
	m = send(m, turnDoneMsg{})

	e := m.entries[m.inflightIdx]
	if !e.interrupted {
		t.Error("interrupted flag not set on a canceled terminal event")
	}
	if e.isError {
		t.Error("canceled event must not set isError (badge, not red error)")
	}
	if e.text != "half an answer" {
		t.Errorf("partial not carried into the bubble, text = %q", e.text)
	}
	if !strings.Contains(m.renderTranscript(), "(interrupted)") {
		t.Error("(interrupted) badge not rendered")
	}
}

func TestErrorEvent_RealError(t *testing.T) {
	m, _ := newTestModel(t)
	m = startStreaming(t, m, "q")
	m = send(m, eventMsg{ev: agent.Event{Kind: agent.EventError, Err: "provider exploded"}})
	m = send(m, turnDoneMsg{})

	e := m.entries[m.inflightIdx]
	if !e.isError {
		t.Error("isError not set on a non-canceled terminal error")
	}
	if e.interrupted {
		t.Error("non-canceled error must not set interrupted")
	}
	if !strings.Contains(m.renderTranscript(), "provider exploded") {
		t.Error("error text not rendered")
	}
}

func TestUnexpectedEvent_RendersRawNoPanic(t *testing.T) {
	m, _ := newTestModel(t)
	m = startStreaming(t, m, "q")
	m = send(m, eventMsg{ev: agent.Event{Kind: agent.EventKind("bogus"), Content: "weird"}})
	if len(m.entries[m.inflightIdx].rawLines) != 1 {
		t.Fatal("unexpected event not captured as a raw line")
	}
	if !strings.Contains(m.renderTranscript(), "unexpected event") {
		t.Error("raw fallback line not rendered")
	}
}

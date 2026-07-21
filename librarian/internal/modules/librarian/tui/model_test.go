package tui

import (
	"context"
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/hsb3/desk-standard/librarian/internal/core/config"
	"github.com/hsb3/desk-standard/librarian/internal/modules/librarian/agent"
)

// fakeStreamer is the injected session boundary: it captures the turn context (so a test can
// prove esc cancels it) and hands back a channel the test does NOT need to feed — pure-Update
// tests deliver engine events as eventMsg/turnDoneMsg directly, without a real Session or LLM.
type fakeStreamer struct {
	ch    chan agent.Event
	ctx   context.Context
	input string
	calls int
	runID string
}

func (f *fakeStreamer) RunID() string { return f.runID }

func (f *fakeStreamer) StreamTurn(ctx context.Context, userInput string) <-chan agent.Event {
	f.ctx = ctx
	f.input = userInput
	f.calls++
	if f.ch == nil {
		f.ch = make(chan agent.Event, 64)
	}
	return f.ch
}

// newTestModel builds a model sized to an 80x24 terminal and ready to drive, against an empty
// fake session provider (no resumable conversations). Tests that exercise the picker/resume paths
// use newTestModelWithProvider to inject a populated fake.
func newTestModel(t *testing.T) (model, *fakeStreamer) {
	t.Helper()
	m, fs, _ := newTestModelWithProvider(t, &fakeProvider{})
	return m, fs
}

// newTestModelWithProvider builds a ready 80x24 model against the given session provider, so the
// picker/new-conversation/resume paths can be driven with a fake that needs no real Session.
func newTestModelWithProvider(t *testing.T, provider *fakeProvider) (model, *fakeStreamer, *fakeProvider) {
	t.Helper()
	cfg := &config.Config{DeskName: "test-desk", LLMProvider: "openai", LLMModel: "test-model"}
	fs := &fakeStreamer{runID: "live-run"}
	m := newModel(context.Background(), fs, provider, cfg, themeDark)
	next, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	return next.(model), fs, provider
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
	return send(m, tea.KeyPressMsg{Code: tea.KeyEnter})
}

func TestWindowSizeMsg_Sizing(t *testing.T) {
	m, _ := newTestModel(t)
	if !m.ready {
		t.Fatal("model not marked ready after WindowSizeMsg")
	}
	if m.vp.Width() != 80 {
		t.Errorf("viewport width = %d, want 80", m.vp.Width())
	}
	// height 24 - header(1) - footer(1) - input(3) - input border(2) = 17
	if m.vp.Height() != 17 {
		t.Errorf("viewport height = %d, want 17", m.vp.Height())
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

	// tea.View (v2's Model.View return type) carries a func field (OnMouse), so the struct is not
	// comparable with == at all — compare the rendered Content string instead, which is what this
	// assertion actually cares about.
	before := m.View().Content
	if strings.Contains(m.renderTranscript(), "args:") {
		t.Fatal("expanded step detail shown before ctrl+t")
	}
	m = send(m, tea.KeyPressMsg{Mod: tea.ModCtrl, Code: 't'})
	after := m.View().Content

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
	m = send(m, tea.KeyPressMsg{Code: tea.KeyEscape})
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
	m = send(m, tea.KeyPressMsg{Code: tea.KeyEnter})

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

// TestFmtTokens covers the compact K/M formatter's boundaries.
func TestFmtTokens(t *testing.T) {
	cases := []struct {
		in   int
		want string
	}{
		{0, "0"},
		{42, "42"},
		{999, "999"},
		{1000, "1.0K"},
		{12300, "12.3K"},
		{999_999, "1000.0K"},
		{1_000_000, "1.0M"},
		{1_200_000, "1.2M"},
	}
	for _, c := range cases {
		if got := fmtTokens(c.in); got != c.want {
			t.Errorf("fmtTokens(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestContextWindow covers override precedence, model-substring matching, provider fallback, and
// the conservative default for a fully-unknown provider/model.
func TestContextWindow(t *testing.T) {
	cases := []struct {
		provider, model string
		override, want  int
	}{
		{"anthropic", "claude-opus-4-8", 12345, 12345},        // override always wins
		{"openai", "claude-opus-4-8", 0, windowAnthropic},     // model substring beats provider
		{"anthropic", "gpt-5", 0, windowOpenAI},               // gpt substring
		{"x", "gemini-3-pro", 0, windowGemini},                // gemini substring
		{"openai", "some-custom-deployment", 0, windowOpenAI}, // provider fallback
		{"anthropic", "mystery", 0, windowAnthropic},          // provider fallback
		{"local-llm", "mystery", 0, windowDefault},            // unknown/unknown → default
	}
	for _, c := range cases {
		if got := contextWindow(c.provider, c.model, c.override); got != c.want {
			t.Errorf("contextWindow(%q, %q, %d) = %d, want %d", c.provider, c.model, c.override, got, c.want)
		}
	}
}

// TestUsageAccounting_TerminalEventRecords proves a terminal EventFinal folds its token accounting
// into the model + the finishing entry, and that the header's ctx% segment and the per-turn footer
// then render the usage. The provider/model here is unknown ("test-model") so ctxWindow falls back
// to the openai provider family — the percent is computed against whatever that resolves to.
func TestUsageAccounting_TerminalEventRecords(t *testing.T) {
	m, _ := newTestModel(t) // provider "openai", model "test-model"
	m = startStreaming(t, m, "q")
	m = send(m, eventMsg{ev: agent.Event{Kind: agent.EventToken, Step: 1, Token: "answer"}})

	// A token event alone must NOT move the context gauge (only a terminal event carries usage).
	if m.ctxTokens != 0 {
		t.Fatalf("ctxTokens = %d before terminal event, want 0", m.ctxTokens)
	}

	m = send(m, eventMsg{ev: agent.Event{
		Kind: agent.EventFinal, Content: "answer",
		PromptTokens: 100_000, CompletionTokens: 500, TotalTokens: 100_500,
	}})
	m = send(m, turnDoneMsg{})

	if m.ctxTokens != 100_000 {
		t.Errorf("ctxTokens = %d, want 100000", m.ctxTokens)
	}
	if m.sessionTokens != 100_500 {
		t.Errorf("sessionTokens = %d, want 100500", m.sessionTokens)
	}
	if got := m.entries[m.inflightIdx].tokens; got != 500 {
		t.Errorf("entry.tokens = %d, want 500", got)
	}

	// Header shows `NN% ctx · <tok> tok`, with the percent computed against the resolved window.
	wantPct := 100_000 * 100 / m.ctxWindow
	hdr := m.renderHeader()
	if !strings.Contains(hdr, fmt.Sprintf("%d%% ctx", wantPct)) {
		t.Errorf("header missing ctx%% segment (want %d%%): %q", wantPct, hdr)
	}
	if !strings.Contains(hdr, "tok") {
		t.Errorf("header missing token segment: %q", hdr)
	}

	// Per-turn footer joins tokens onto the `model · latency` line without a new line.
	transcript := m.renderTranscript()
	if !strings.Contains(transcript, "500 tok") {
		t.Errorf("per-turn footer missing token count: %q", transcript)
	}
}

// TestUsageAccounting_HeaderHiddenWithoutUsage: before any terminal usage the header carries only
// the desk/provider/model chrome — no ctx% segment (the gauge stays hidden until a turn reports).
func TestUsageAccounting_HeaderHiddenWithoutUsage(t *testing.T) {
	m, _ := newTestModel(t)
	if strings.Contains(m.renderHeader(), "ctx") {
		t.Errorf("header shows a ctx segment before any usage was reported: %q", m.renderHeader())
	}
}

// TestUsageAccounting_SessionTotalAccumulates: sessionTokens sums TotalTokens across turns, and
// ctxTokens tracks the LATEST turn's prompt count (the current context size).
func TestUsageAccounting_SessionTotalAccumulates(t *testing.T) {
	m, _ := newTestModel(t)

	m = startStreaming(t, m, "q1")
	m = send(m, eventMsg{ev: agent.Event{Kind: agent.EventFinal, Content: "a1", PromptTokens: 1000, CompletionTokens: 100, TotalTokens: 1100}})
	m = send(m, turnDoneMsg{})
	if m.sessionTokens != 1100 || m.ctxTokens != 1000 {
		t.Fatalf("after turn 1: sessionTokens=%d ctxTokens=%d, want 1100/1000", m.sessionTokens, m.ctxTokens)
	}

	m = startStreaming(t, m, "q2")
	m = send(m, eventMsg{ev: agent.Event{Kind: agent.EventFinal, Content: "a2", PromptTokens: 2000, CompletionTokens: 200, TotalTokens: 2200}})
	m = send(m, turnDoneMsg{})
	if m.sessionTokens != 3300 {
		t.Errorf("sessionTokens = %d, want 3300 (1100+2200)", m.sessionTokens)
	}
	if m.ctxTokens != 2000 {
		t.Errorf("ctxTokens = %d, want 2000 (latest turn's prompt)", m.ctxTokens)
	}
}

// TestUsageAccounting_CanceledTurnRecords: a canceled terminal event still records whatever usage
// was counted (partial turns report what they had), so the gauge reflects the interrupted turn.
func TestUsageAccounting_CanceledTurnRecords(t *testing.T) {
	m, _ := newTestModel(t)
	m = startStreaming(t, m, "q")
	m = send(m, eventMsg{ev: agent.Event{
		Kind: agent.EventError, Canceled: true, Partial: "half",
		PromptTokens: 4000, CompletionTokens: 15, TotalTokens: 4015,
	}})
	m = send(m, turnDoneMsg{})

	if m.ctxTokens != 4000 {
		t.Errorf("ctxTokens = %d, want 4000 (counted on a canceled turn)", m.ctxTokens)
	}
	if got := m.entries[m.inflightIdx].tokens; got != 15 {
		t.Errorf("entry.tokens = %d, want 15", got)
	}
	if m.sessionTokens != 4015 {
		t.Errorf("sessionTokens = %d, want 4015", m.sessionTokens)
	}
}

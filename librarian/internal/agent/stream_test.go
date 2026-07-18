package agent

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"github.com/example/pocket-librarian/internal/config"
)

// scriptedModel is a model.ToolCallingChatModel whose Stream returns a canned reply per call,
// popped from an ordered script. It is the streaming successor to session_test.go's
// fakeChatModel: each script entry is a multi-chunk stream (schema.StreamReaderFromArray, or a
// pipe for the blocking/cancel cases), so a test can drive the REAL registered tool loop —
// tool-call step then final answer — through the same eino ReAct engine production uses.
type scriptedModel struct {
	mu      sync.Mutex
	steps   []streamStep
	call    int
	genLens []int
}

// streamStep produces the model output for one Stream call (ctx is the call's context, so a
// step can block on cancellation).
type streamStep func(ctx context.Context) (*schema.StreamReader[*schema.Message], error)

func (m *scriptedModel) Stream(ctx context.Context, input []*schema.Message, _ ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	m.mu.Lock()
	i := m.call
	m.call++
	m.genLens = append(m.genLens, len(input))
	var step streamStep
	if i < len(m.steps) {
		step = m.steps[i]
	}
	m.mu.Unlock()
	if step != nil {
		return step(ctx)
	}
	return schema.StreamReaderFromArray([]*schema.Message{schema.AssistantMessage("(default)", nil)}), nil
}

func (m *scriptedModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	sr, err := m.Stream(ctx, input, opts...)
	if err != nil {
		return nil, err
	}
	return schema.ConcatMessageStream(sr)
}

func (m *scriptedModel) WithTools(_ []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	return m, nil
}

// contentStep streams the given content pieces as separate assistant chunks (a multi-chunk
// reply whose concatenation is the joined content — the token source for a final answer).
func contentStep(pieces ...string) streamStep {
	return func(_ context.Context) (*schema.StreamReader[*schema.Message], error) {
		chunks := make([]*schema.Message, 0, len(pieces))
		for _, p := range pieces {
			chunks = append(chunks, &schema.Message{Role: schema.Assistant, Content: p})
		}
		return schema.StreamReaderFromArray(chunks), nil
	}
}

// toolCallStep streams optional leading content then a complete tool call, exercising the real
// tool loop (the named tool must be registered — query always is).
func toolCallStep(content, name, callID, args string) streamStep {
	return func(_ context.Context) (*schema.StreamReader[*schema.Message], error) {
		var chunks []*schema.Message
		if content != "" {
			chunks = append(chunks, &schema.Message{Role: schema.Assistant, Content: content})
		}
		chunks = append(chunks, &schema.Message{
			Role: schema.Assistant,
			ToolCalls: []schema.ToolCall{{
				ID:       callID,
				Type:     "function",
				Function: schema.FunctionCall{Name: name, Arguments: args},
			}},
		})
		return schema.StreamReaderFromArray(chunks), nil
	}
}

// blockingStep sends preChunks then blocks until `release` fires, then ends the stream with a
// context.Canceled error — modeling a provider whose stream is interrupted mid-flight (with
// preChunks) or before the first token (without). It keys off `release` (the test's own
// cancelable context), NOT the context eino hands the model node: eino v0.9.12's react
// streaming does not propagate caller-ctx cancellation down to abort a stuck model stream (the
// node ctx has a nil Done channel); a real provider ends its own stream when it observes
// cancellation, which is exactly what firing `release` here emulates. The turn is still
// classified canceled because StreamTurn checks the caller ctx (ctx.Err()).
func blockingStep(release context.Context, started chan struct{}, preChunks ...string) streamStep {
	return func(_ context.Context) (*schema.StreamReader[*schema.Message], error) {
		if started != nil {
			close(started) // let the test cancel only AFTER this step is really in flight
		}
		sr, sw := schema.Pipe[*schema.Message](len(preChunks) + 1)
		go func() {
			for _, p := range preChunks {
				sw.Send(&schema.Message{Role: schema.Assistant, Content: p}, nil)
			}
			<-release.Done()
			sw.Send(nil, context.Canceled)
			sw.Close()
		}()
		return sr, nil
	}
}

// errorStep fails the model call outright (a mid-loop provider error).
func errorStep(err error) streamStep {
	return func(_ context.Context) (*schema.StreamReader[*schema.Message], error) {
		return nil, err
	}
}

// installModel points chatModelFactory at m for the duration of the test.
func installModel(t *testing.T, m model.ToolCallingChatModel) {
	t.Helper()
	orig := chatModelFactory
	chatModelFactory = func(_ context.Context, _ *config.Config) (model.ToolCallingChatModel, error) {
		return m, nil
	}
	t.Cleanup(func() { chatModelFactory = orig })
}

// anthropicEnv is newSessionTestEnv with the provider flipped to anthropic, which activates the
// whole-stream StreamToolCallChecker (agent.go) — required so a content-then-tool-call reply is
// recognized as a tool call rather than being cut short by the first-chunk default checker.
func anthropicEnv(t *testing.T) (core.App, *config.Config) {
	t.Helper()
	app, cfg := newSessionTestEnv(t)
	cfg.LLMProvider = "anthropic"
	return app, cfg
}

// drainAll collects every event until the channel closes.
func drainAll(ch <-chan Event) []Event {
	var evs []Event
	for ev := range ch {
		evs = append(evs, ev)
	}
	return evs
}

// loadRows returns the messages rows for a run, ordered by seq ascending.
func loadRows(t *testing.T, app core.App, runID string) []*core.Record {
	t.Helper()
	recs, err := app.FindRecordsByFilter("messages", "run = {:r}", "seq", 500, 0, dbx.Params{"r": runID})
	if err != nil {
		t.Fatalf("load messages: %v", err)
	}
	return recs
}

// assertDenseSeq asserts the rows carry seq 1..N with no gaps and no duplicates.
func assertDenseSeq(t *testing.T, recs []*core.Record) {
	t.Helper()
	for i, r := range recs {
		if got := r.GetInt("seq"); got != i+1 {
			t.Fatalf("row %d: seq = %d, want %d (gap or duplicate)", i, got, i+1)
		}
	}
}

// rolesOf extracts the role of each row in seq order.
func rolesOf(recs []*core.Record) []string {
	out := make([]string, len(recs))
	for i, r := range recs {
		out[i] = r.GetString("role")
	}
	return out
}

func countKind(evs []Event, k EventKind) int {
	n := 0
	for _, ev := range evs {
		if ev.Kind == k {
			n++
		}
	}
	return n
}

// TestStreamTurn_TokenFinalOrdering: tokens precede the single terminal event, the final
// Content equals the concatenation of the last step's tokens, and the channel closes after
// exactly one terminal event.
func TestStreamTurn_TokenFinalOrdering(t *testing.T) {
	m := &scriptedModel{steps: []streamStep{contentStep("Hello, ", "desk.")}}
	installModel(t, m)
	app, cfg := newSessionTestEnv(t)
	ctx := context.Background()

	sess, err := NewSession(ctx, app, cfg)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	evs := drainAll(sess.StreamTurn(ctx, "hi"))

	// Exactly one terminal event, and it is EventFinal at the very end.
	if got := countKind(evs, EventFinal) + countKind(evs, EventError); got != 1 {
		t.Fatalf("expected exactly one terminal event, got %d (%v)", got, evs)
	}
	if evs[len(evs)-1].Kind != EventFinal {
		t.Fatalf("last event kind = %q, want final", evs[len(evs)-1].Kind)
	}

	// All tokens precede the terminal; concatenated tokens equal the final content.
	var tokens strings.Builder
	sawFinal := false
	for _, ev := range evs {
		switch ev.Kind {
		case EventToken:
			if sawFinal {
				t.Fatalf("token emitted after final event")
			}
			tokens.WriteString(ev.Token)
		case EventFinal:
			sawFinal = true
			if ev.Content != "Hello, desk." {
				t.Fatalf("final content = %q, want %q", ev.Content, "Hello, desk.")
			}
		}
	}
	if tokens.String() != "Hello, desk." {
		t.Fatalf("token concatenation = %q, want %q", tokens.String(), "Hello, desk.")
	}
}

// TestStreamTurn_ToolEventsBetweenSteps: tool_start/tool_end land strictly between step-1 and
// step-2 tokens, with name/args/result/callID populated from the real tool loop.
func TestStreamTurn_ToolEventsBetweenSteps(t *testing.T) {
	m := &scriptedModel{steps: []streamStep{
		toolCallStep("Let me check. ", "query", "call-1", `{"kind":"live_files"}`),
		contentStep("Found ", "nothing."),
	}}
	installModel(t, m)
	app, cfg := anthropicEnv(t)
	ctx := context.Background()

	sess, err := NewSession(ctx, app, cfg)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	evs := drainAll(sess.StreamTurn(ctx, "what is live"))

	var lastStep1Token, toolStart, toolEnd, firstStep2Token = -1, -1, -1, -1
	var start, end Event
	for i, ev := range evs {
		switch ev.Kind {
		case EventToken:
			if ev.Step == 1 {
				lastStep1Token = i
			}
			if ev.Step == 2 && firstStep2Token == -1 {
				firstStep2Token = i
			}
		case EventToolStart:
			toolStart, start = i, ev
		case EventToolEnd:
			toolEnd, end = i, ev
		}
	}
	if lastStep1Token == -1 || toolStart == -1 || toolEnd == -1 || firstStep2Token == -1 {
		t.Fatalf("missing an expected event; sequence = %v", evs)
	}
	if !(lastStep1Token < toolStart && toolStart < toolEnd && toolEnd < firstStep2Token) {
		t.Fatalf("bad ordering: step1=%d toolStart=%d toolEnd=%d step2=%d",
			lastStep1Token, toolStart, toolEnd, firstStep2Token)
	}
	if start.Tool != "query" || start.Args == "" || start.CallID != "call-1" {
		t.Fatalf("tool_start not populated: %+v", start)
	}
	if end.Tool != "query" || end.Result == "" || end.CallID != "call-1" {
		t.Fatalf("tool_end not populated: %+v", end)
	}
}

// TestStreamTurn_ExactlyOnceTranscript is THE regression guard for the per-turn hwm baseline
// bug. It runs two no-tool turns AND two tool-using turns, through BOTH the StreamTurn and Turn
// code paths, and asserts the messages rows are strictly the exactly-once alternating shape with
// dense seq. Against the old per-run-hwm Generate-based Turn this FAILS: turn 2 re-persists the
// prior assistant row (a duplicate), so the role sequence and dense-seq checks both break.
func TestStreamTurn_ExactlyOnceTranscript(t *testing.T) {
	noToolSteps := func() []streamStep {
		return []streamStep{contentStep("first answer"), contentStep("second answer")}
	}
	toolSteps := func() []streamStep {
		return []streamStep{
			toolCallStep("", "query", "c1", `{"kind":"live_files"}`),
			contentStep("answer one"),
			toolCallStep("", "query", "c2", `{"kind":"live_files"}`),
			contentStep("answer two"),
		}
	}

	// runViaStream drains two StreamTurn turns; runViaTurn uses the blocking Turn wrapper.
	runViaStream := func(t *testing.T, sess *Session, ctx context.Context) {
		drainAll(sess.StreamTurn(ctx, "q1"))
		drainAll(sess.StreamTurn(ctx, "q2"))
	}
	runViaTurn := func(t *testing.T, sess *Session, ctx context.Context) {
		if _, err := sess.Turn(ctx, "q1"); err != nil {
			t.Fatalf("Turn 1: %v", err)
		}
		if _, err := sess.Turn(ctx, "q2"); err != nil {
			t.Fatalf("Turn 2: %v", err)
		}
	}

	cases := []struct {
		name      string
		anthropic bool
		steps     func() []streamStep
		wantRoles []string
	}{
		{"noTool", false, noToolSteps, []string{"system", "user", "assistant", "user", "assistant"}},
		{"tool", true, toolSteps, []string{
			"system", "user", "assistant", "tool", "assistant",
			"user", "assistant", "tool", "assistant",
		}},
	}
	drivers := []struct {
		name string
		run  func(*testing.T, *Session, context.Context)
	}{
		{"viaStream", runViaStream},
		{"viaTurn", runViaTurn},
	}

	for _, tc := range cases {
		for _, d := range drivers {
			t.Run(tc.name+"/"+d.name, func(t *testing.T) {
				m := &scriptedModel{steps: tc.steps()}
				installModel(t, m)
				var app core.App
				var cfg *config.Config
				if tc.anthropic {
					app, cfg = anthropicEnv(t)
				} else {
					app, cfg = newSessionTestEnv(t)
				}
				ctx := context.Background()
				sess, err := NewSession(ctx, app, cfg)
				if err != nil {
					t.Fatalf("NewSession: %v", err)
				}
				d.run(t, sess, ctx)

				recs := loadRows(t, app, sess.run.Id)
				if got := rolesOf(recs); !equalStrings(got, tc.wantRoles) {
					t.Fatalf("roles = %v, want %v", got, tc.wantRoles)
				}
				assertDenseSeq(t, recs)
			})
		}
	}
}

// TestStreamTurn_CancelWithPartial: a stream that blocks after two chunks until cancel yields
// EventError{Canceled, Partial}, persists a plain assistant row carrying the partial, and the
// next turn succeeds with a consistent, dense transcript.
func TestStreamTurn_CancelWithPartial(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	m := &scriptedModel{steps: []streamStep{
		blockingStep(ctx, nil, "par", "tial"),
		contentStep("second turn done"),
	}}
	installModel(t, m)
	app, cfg := newSessionTestEnv(t)

	sess, err := NewSession(context.Background(), app, cfg)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}

	ch := sess.StreamTurn(ctx, "start")
	var evs []Event
	tokens := 0
	for ev := range ch {
		evs = append(evs, ev)
		if ev.Kind == EventToken {
			tokens++
			if tokens == 2 {
				cancel() // interrupt after the second live token
			}
		}
	}
	cancel()

	term := evs[len(evs)-1]
	if term.Kind != EventError || !term.Canceled {
		t.Fatalf("terminal event = %+v, want EventError canceled", term)
	}
	if term.Partial != "partial" {
		t.Fatalf("partial = %q, want %q", term.Partial, "partial")
	}

	// A plain assistant row carrying the partial must be persisted (no synthetic marker).
	recs := loadRows(t, app, sess.run.Id)
	foundPartial := false
	for _, r := range recs {
		if r.GetString("role") == "assistant" && r.GetString("content") == "partial" {
			foundPartial = true
		}
	}
	if !foundPartial {
		t.Fatalf("no persisted assistant row carrying the partial; rows = %v", rolesOf(recs))
	}

	// The next turn succeeds on a fresh context with a consistent, dense transcript.
	out, err := sess.Turn(context.Background(), "again")
	if err != nil {
		t.Fatalf("next turn after cancel: %v", err)
	}
	if out != "second turn done" {
		t.Fatalf("next turn content = %q, want %q", out, "second turn done")
	}
	assertDenseSeq(t, loadRows(t, app, sess.run.Id))
}

// TestStreamTurn_CancelBeforeFirstToken: cancel before any token rolls the history back (no
// dangling user message) and leaves the session usable for the next turn.
func TestStreamTurn_CancelBeforeFirstToken(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	started := make(chan struct{})
	m := &scriptedModel{steps: []streamStep{
		blockingStep(ctx, started), // blocks immediately, no tokens
		contentStep("recovered"),
	}}
	installModel(t, m)
	app, cfg := newSessionTestEnv(t)

	sess, err := NewSession(context.Background(), app, cfg)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}

	ch := sess.StreamTurn(ctx, "start")
	<-started // ensure the first model step is in flight before canceling
	cancel()
	evs := drainAll(ch)

	term := evs[len(evs)-1]
	if term.Kind != EventError || !term.Canceled {
		t.Fatalf("terminal event = %+v, want EventError canceled", term)
	}
	if term.Partial != "" {
		t.Fatalf("partial = %q, want empty (no tokens streamed)", term.Partial)
	}
	if len(sess.history) != 0 {
		t.Fatalf("history not rolled back: len = %d, want 0", len(sess.history))
	}

	out, err := sess.Turn(context.Background(), "again")
	if err != nil {
		t.Fatalf("next turn after cancel-before-first-token: %v", err)
	}
	if out != "recovered" {
		t.Fatalf("next turn content = %q, want %q", out, "recovered")
	}
}

// TestStreamTurn_MidLoopError: a model error on the second step (after a tool call) surfaces as
// a non-canceled error, leaves no dangling user message in history, and the run continues.
func TestStreamTurn_MidLoopError(t *testing.T) {
	m := &scriptedModel{steps: []streamStep{
		toolCallStep("", "query", "c1", `{"kind":"live_files"}`),
		errorStep(errors.New("provider boom")),
		contentStep("after error"),
	}}
	installModel(t, m)
	app, cfg := anthropicEnv(t)

	sess, err := NewSession(context.Background(), app, cfg)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}

	_, turnErr := sess.Turn(context.Background(), "do it")
	if turnErr == nil {
		t.Fatalf("expected a mid-loop error, got nil")
	}
	if errors.Is(turnErr, context.Canceled) {
		t.Fatalf("mid-loop error must not be classified as canceled: %v", turnErr)
	}
	if !strings.Contains(turnErr.Error(), "boom") {
		t.Fatalf("error = %v, want it to carry the provider failure", turnErr)
	}
	if len(sess.history) != 0 {
		t.Fatalf("dangling history after error: len = %d, want 0", len(sess.history))
	}

	out, err := sess.Turn(context.Background(), "again")
	if err != nil {
		t.Fatalf("run did not continue after mid-loop error: %v", err)
	}
	if out != "after error" {
		t.Fatalf("recovery turn content = %q, want %q", out, "after error")
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

package agent

import (
	"context"
	"errors"
	"testing"

	"github.com/cloudwego/eino/compose"
	"github.com/pocketbase/pocketbase/core"

	"github.com/hsb3/desk-standard/librarian/internal/core/config"
	"github.com/hsb3/desk-standard/librarian/internal/core/toolcore"
	"github.com/hsb3/desk-standard/librarian/internal/modules/librarian/tools"
)

// registerToolsWithCanceler registers the librarian tool set PLUS a zero-argument tool named
// "canceltoolrun" whose invocation cancels the provided context and returns a real (non-empty)
// result. It lets a test drive the real eino loop to a NON-MaxStep abort that lands exactly at the
// gap's point of interest: the tool runs to completion (its OnEnd fires, buffering the round), and
// the graph runner then observes the canceled context at the top of the next super-step — before it
// submits the next model task, so that model's OnStart never fires and the round is never carried
// into the transcript by the input-side delta. Cleanup restores the librarian-only registry.
func registerToolsWithCanceler(t *testing.T, cancel context.CancelFunc) {
	t.Helper()
	toolcore.Reset()
	specs := append(tools.Specs(), cancelToolSpec(cancel))
	toolcore.Register(specs...)
	t.Cleanup(func() {
		toolcore.Reset()
		toolcore.Register(tools.Specs()...)
	})
}

// cancelToolSpec builds the librarian-module "canceltoolrun" tool. It is AgentDefault so it lands
// in the agent's gated slice, WritesFiles is false, and it returns a non-empty JSON payload so the
// persisted tool row proves the call really executed (not a placeholder).
func cancelToolSpec(cancel context.CancelFunc) toolcore.ToolSpec {
	return toolcore.New[struct{}]("librarian", "canceltoolrun",
		"test-only tool that cancels the run context after executing",
		false, true, false,
		func(_ context.Context, _ core.App, _ *config.Config, _ *struct{}) (any, error) {
			cancel()
			return map[string]any{"canceled": true}, nil
		})
}

// TestRun_CancelAfterToolFlushesToolCall guards the audit-trail gap for a NON-MaxStep abort in the
// one-shot Run() path: a context cancellation that lands right after a state-mutating tool executes,
// between the tool's OnEnd and the next model call's OnStart. The tool's effect is real; the round
// must still be recorded.
//
// PRE-FIX this FAILS: Run() flushed rc.pending only on errors.Is(genErr, ErrExceedMaxSteps), so a
// context.Canceled abort left messages = [system, user] with the assistant tool-call + tool-result
// rows lost even though the tool ran. POST-FIX Run() flushes on ANY genErr != nil with a non-empty
// buffer, so the executed tool call is recorded on the cancellation path too.
func TestRun_CancelAfterToolFlushesToolCall(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	registerToolsWithCanceler(t, cancel)

	m := &scriptedModel{steps: []streamStep{
		// The first reply calls the cancel tool (empty content so the default first-chunk tool-call
		// checker recognizes it under the openai provider). The tool cancels ctx; the loop aborts at
		// the next super-step check, before any second model call.
		toolCallStep("", "canceltoolrun", "c1", `{}`),
		contentStep("never reached"),
	}}
	installModel(t, m)

	app, cfg := newSessionTestEnv(t) // openai provider, high AgentMaxStep — the abort is the cancel, not MaxStep

	final, err := Run(ctx, app, cfg, "manual", "audit the desk")

	// The loop aborted on a context cancellation — NOT MaxStep — and produced no final text.
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Run error = %v, want it to wrap context.Canceled", err)
	}
	if errors.Is(err, compose.ErrExceedMaxSteps) {
		t.Fatalf("Run error = %v, must NOT be a MaxStep abort (this is the non-MaxStep gap)", err)
	}
	if final != "" {
		t.Fatalf("final text = %q, want empty on an aborted run", final)
	}

	run, err := app.FindRecordById("agent_runs", runIDOfLatest(t, app))
	if err != nil {
		t.Fatalf("reload run: %v", err)
	}
	if got := run.GetString("status"); got != "failed" {
		t.Fatalf("run status = %q, want failed", got)
	}

	// THE FIX: the transcript records the executed tool call and its result even though the abort was
	// a cancellation, not MaxStep. Pre-fix this was just [system, user].
	recs := loadRows(t, app, run.Id)
	if got := rolesOf(recs); !equalStrings(got, []string{"system", "user", "assistant", "tool"}) {
		t.Fatalf("roles = %v, want [system user assistant tool] (the tool call must be flushed on a cancel abort)", got)
	}
	assertDenseSeq(t, recs)

	asst := recs[2]
	if calls := toolCallsOf(asst); len(calls) != 1 || calls[0].ID != "c1" || calls[0].Function.Name != "canceltoolrun" {
		t.Fatalf("assistant row tool_calls = %+v, want one call c1/canceltoolrun", toolCallsOf(asst))
	}
	toolRow := recs[3]
	if got := toolRow.GetString("tool_call_id"); got != "c1" {
		t.Fatalf("tool row tool_call_id = %q, want c1", got)
	}
	if got := toolRow.GetString("tool_name"); got != "canceltoolrun" {
		t.Fatalf("tool row tool_name = %q, want canceltoolrun", got)
	}
	if toolRow.GetString("content") == "" {
		t.Fatalf("tool row content is empty, want the tool's result")
	}
}

// TestStreamTurn_MaxStepFlushesToolCall guards the audit-trail gap for the STREAMING Session path
// when the loop aborts on MaxStep right after a tool executes. It is the streaming twin of the
// one-shot Run() MaxStep guard: same eino ReAct engine, same gap, different entry point.
//
// PRE-FIX this FAILS: StreamTurn registered only the input-side persistHandler (no round buffer), so
// a MaxStep abort mid-session dropped the last round's tool call from messages even though the tool
// ran. POST-FIX turnEvents reconstructs the round into rc.pending and StreamTurn flushes it on the
// abort, so the transcript records the tool call.
func TestStreamTurn_MaxStepFlushesToolCall(t *testing.T) {
	registerLibrarianTools(t)

	m := &scriptedModel{steps: []streamStep{
		toolCallStep("", "query", "c1", `{"kind":"live_files"}`),
		contentStep("never reached"), // a second step exists but MaxStep aborts before it
	}}
	installModel(t, m)

	app, cfg := anthropicEnv(t)
	cfg.AgentMaxStep = 2 // ChatModel(step 0) + ToolsNode(step 1), then step 2 >= 2 aborts

	ctx := context.Background()
	sess, err := NewSession(ctx, app, cfg)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	evs := drainAll(sess.StreamTurn(ctx, "audit the desk"))

	// Exactly one terminal event, and it is an error that is NOT a cancellation (MaxStep aborts).
	if got := countKind(evs, EventFinal) + countKind(evs, EventError); got != 1 {
		t.Fatalf("expected exactly one terminal event, got %d (%v)", got, evs)
	}
	term := evs[len(evs)-1]
	if term.Kind != EventError {
		t.Fatalf("terminal event = %+v, want EventError on a MaxStep abort", term)
	}
	if term.Canceled {
		t.Fatalf("MaxStep abort must not be classified as canceled: %+v", term)
	}

	// THE FIX: the transcript records the tool round the aborted turn executed.
	recs := loadRows(t, app, sess.run.Id)
	if got := rolesOf(recs); !equalStrings(got, []string{"system", "user", "assistant", "tool"}) {
		t.Fatalf("roles = %v, want [system user assistant tool] (the streaming tool call must be flushed)", got)
	}
	assertDenseSeq(t, recs)

	asst := recs[2]
	if calls := toolCallsOf(asst); len(calls) != 1 || calls[0].ID != "c1" || calls[0].Function.Name != "query" {
		t.Fatalf("assistant row tool_calls = %+v, want one call c1/query", toolCallsOf(asst))
	}
	toolRow := recs[3]
	if got := toolRow.GetString("tool_call_id"); got != "c1" {
		t.Fatalf("tool row tool_call_id = %q, want c1", got)
	}
	if got := toolRow.GetString("tool_name"); got != "query" {
		t.Fatalf("tool row tool_name = %q, want query", got)
	}
	if toolRow.GetString("content") == "" {
		t.Fatalf("tool row content is empty, want the query tool's result")
	}
}

// TestStreamTurn_CancelAfterToolFlushesToolCall guards the streaming path against the NON-MaxStep
// abort of the same family: a cancellation that lands right after a tool executes. It combines both
// tracked gaps for the streaming surface — the flush is registered on the streaming path (gap 1) AND
// it fires on a non-MaxStep abort (gap 2).
//
// PRE-FIX this FAILS: with no round buffer on the streaming path, the tool call was dropped from
// messages; the turn's empty-content tool step also left no partial, so the whole round vanished.
// POST-FIX the buffered round is flushed on the cancellation.
func TestStreamTurn_CancelAfterToolFlushesToolCall(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	registerToolsWithCanceler(t, cancel)

	m := &scriptedModel{steps: []streamStep{
		toolCallStep("", "canceltoolrun", "c1", `{}`), // tool cancels ctx; loop aborts before the next model call
		contentStep("never reached"),
	}}
	installModel(t, m)

	app, cfg := anthropicEnv(t)

	sess, err := NewSession(context.Background(), app, cfg)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	evs := drainAll(sess.StreamTurn(ctx, "audit the desk"))

	term := evs[len(evs)-1]
	if term.Kind != EventError || !term.Canceled {
		t.Fatalf("terminal event = %+v, want EventError canceled", term)
	}

	// THE FIX: the executed tool round is recorded even on the cancellation abort.
	recs := loadRows(t, app, sess.run.Id)
	if got := rolesOf(recs); !equalStrings(got, []string{"system", "user", "assistant", "tool"}) {
		t.Fatalf("roles = %v, want [system user assistant tool] (tool call flushed on a streaming cancel abort)", got)
	}
	assertDenseSeq(t, recs)

	asst := recs[2]
	if calls := toolCallsOf(asst); len(calls) != 1 || calls[0].ID != "c1" || calls[0].Function.Name != "canceltoolrun" {
		t.Fatalf("assistant row tool_calls = %+v, want one call c1/canceltoolrun", toolCallsOf(asst))
	}
	if got := recs[3].GetString("tool_name"); got != "canceltoolrun" {
		t.Fatalf("tool row tool_name = %q, want canceltoolrun", got)
	}
	if recs[3].GetString("content") == "" {
		t.Fatalf("tool row content is empty, want the tool's result")
	}

	// The session stays usable: a fresh-context turn succeeds with a consistent, dense transcript.
	out, err := sess.Turn(context.Background(), "again")
	if err != nil {
		t.Fatalf("next turn after cancel abort: %v", err)
	}
	if out != "never reached" { // the scripted second step is the next turn's first (and only) reply
		t.Fatalf("next turn content = %q, want %q", out, "never reached")
	}
	assertDenseSeq(t, loadRows(t, app, sess.run.Id))
}

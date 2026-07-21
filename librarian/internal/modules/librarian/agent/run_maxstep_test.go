package agent

import (
	"context"
	"errors"
	"testing"

	"github.com/cloudwego/eino/compose"
	"github.com/pocketbase/pocketbase/core"

	"github.com/hsb3/desk-standard/librarian/internal/core/toolcore"
	"github.com/hsb3/desk-standard/librarian/internal/modules/librarian/tools"
)

// registerLibrarianTools populates the shared toolcore registry with the librarian specs so a
// test that drives the REAL tool loop in isolation (not relying on another test's registration
// side effect) finds tools like "query". Cleanup restores the librarian-only registry that the
// rest of the package assumes.
func registerLibrarianTools(t *testing.T) {
	t.Helper()
	toolcore.Reset()
	toolcore.Register(tools.Specs()...)
	t.Cleanup(func() {
		toolcore.Reset()
		toolcore.Register(tools.Specs()...)
	})
}

// TestRun_MaxStepFlushesToolCall is THE regression guard for the audit-trail gap (issue: a
// MaxStep abort right after a state-mutating tool call left that call out of the messages
// transcript). It drives the one-shot Run() through the SAME eino ReAct loop production uses,
// with AgentMaxStep=2 so the loop aborts immediately after the first tool executes and before
// the next model call. The tool result therefore only ever appears in the model input that never
// happens — so the input-side persistence callback alone would drop the round.
//
// PRE-FIX this FAILS: rc.pending did not exist and Run() persisted the final message only on
// genErr == nil, so a MaxStep run left messages = [system, user] and the assistant tool-call +
// tool-result rows were lost even though the tool ran. POST-FIX Run() flushes the buffered round
// on the MaxStep error, so the transcript records the tool call.
//
// query stands in for any real tool: the persistence gap is independent of whether the tool
// mutates state (the live repro used propose_fix's revisions write); the invariant restored here
// is simply "a tool call that executed appears in the transcript, even on a failed/blocked run".
func TestRun_MaxStepFlushesToolCall(t *testing.T) {
	registerLibrarianTools(t)

	m := &scriptedModel{steps: []streamStep{
		// The model's first reply is a real tool call (empty content so the default, first-chunk
		// tool-call checker recognizes it under the openai provider).
		toolCallStep("", "query", "c1", `{"kind":"live_files"}`),
		// A second step exists but is never reached: MaxStep aborts before the next model call.
		contentStep("never reached"),
	}}
	installModel(t, m)

	app, cfg := newSessionTestEnv(t)
	cfg.AgentMaxStep = 2 // ChatModel(step 0) + ToolsNode(step 1), then step 2 >= 2 aborts

	final, err := Run(context.Background(), app, cfg, "manual", "audit the desk")

	// The loop aborted on MaxStep: the error surfaces to the caller and there is no final text.
	if !errors.Is(err, compose.ErrExceedMaxSteps) {
		t.Fatalf("Run error = %v, want it to wrap compose.ErrExceedMaxSteps", err)
	}
	if final != "" {
		t.Fatalf("final text = %q, want empty on a MaxStep abort", final)
	}

	// The run row is the terminal failed state the issue describes.
	run, err := app.FindRecordById("agent_runs", runIDOfLatest(t, app))
	if err != nil {
		t.Fatalf("reload run: %v", err)
	}
	if got := run.GetString("status"); got != "failed" {
		t.Fatalf("run status = %q, want failed", got)
	}
	if got := run.GetInt("step_count"); got != 1 {
		t.Fatalf("step_count = %d, want 1 (one model call before the abort)", got)
	}

	// THE FIX: the transcript now records the tool call and its result. Pre-fix this was just
	// [system, user] and both assertions below failed.
	recs := loadRows(t, app, run.Id)
	if got := rolesOf(recs); !equalStrings(got, []string{"system", "user", "assistant", "tool"}) {
		t.Fatalf("roles = %v, want [system user assistant tool] (the tool call must be flushed)", got)
	}
	assertDenseSeq(t, recs)

	// The assistant row carries the tool call; the tool row carries the matching id/name and a
	// real (non-empty) result — proving the executed call, not a placeholder, was persisted.
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

// TestRun_SuccessTranscriptUnchanged pins the invariant the fix must NOT disturb: a Run() that
// completes normally (no MaxStep) still produces exactly the [system, user, assistant, tool,
// assistant] transcript with dense seq and no duplication. The capture handler that backs the
// MaxStep flush is registered on every Run(), so this proves it only observes messages in memory
// and never perturbs the successful-completion persistence (which still comes from the input-side
// callback plus the caller's single final persist).
func TestRun_SuccessTranscriptUnchanged(t *testing.T) {
	registerLibrarianTools(t)

	m := &scriptedModel{steps: []streamStep{
		toolCallStep("", "query", "c1", `{"kind":"live_files"}`), // one tool round
		contentStep("all indexed"),                               // then a final answer -> END
	}}
	installModel(t, m)

	app, cfg := newSessionTestEnv(t) // AgentMaxStep default 12 — the loop runs to completion

	final, err := Run(context.Background(), app, cfg, "manual", "audit the desk")
	if err != nil {
		t.Fatalf("Run returned an error on a completing run: %v", err)
	}
	if final != "all indexed" {
		t.Fatalf("final text = %q, want %q", final, "all indexed")
	}

	run, err := app.FindRecordById("agent_runs", runIDOfLatest(t, app))
	if err != nil {
		t.Fatalf("reload run: %v", err)
	}
	if got := run.GetString("status"); got != "succeeded" {
		t.Fatalf("run status = %q, want succeeded", got)
	}

	recs := loadRows(t, app, run.Id)
	if got := rolesOf(recs); !equalStrings(got, []string{"system", "user", "assistant", "tool", "assistant"}) {
		t.Fatalf("roles = %v, want [system user assistant tool assistant] (exactly-once, no dup)", got)
	}
	assertDenseSeq(t, recs)
}

// runIDOfLatest returns the id of the most recently started agent_runs row — the run just opened
// by Run() (which returns only the final text and error, not its run id).
func runIDOfLatest(t *testing.T, app core.App) string {
	t.Helper()
	runs, err := app.FindRecordsByFilter("agent_runs", "id != ''", "-started", 1, 0)
	if err != nil {
		t.Fatalf("find latest run: %v", err)
	}
	if len(runs) == 0 {
		t.Fatalf("no agent_runs row was created")
	}
	return runs[0].Id
}

package trigger

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"

	"github.com/hsb3/desk-standard/librarian/internal/core/config"

	// Blank-import registers this project's Go migrations (files, tasks, ...) into the
	// same global registry PocketBase's built-in migrations use, so tests.NewTestApp's
	// RunAllMigrations() creates our collections too.
	_ "github.com/hsb3/desk-standard/librarian/internal/modules/librarian/collections"
)

// newTestEnv boots a fresh PocketBase test app (own temp SQLite data dir, all migrations
// applied) plus a Config pointed at a fresh temp DESK_ROOT with an empty (non-blocking)
// ignore file. No provider/LLM key is set anywhere in this package's tests: deterministic
// task kinds never touch a model, and the agentic kinds are exercised with a nil or fake
// AgentAction, never the real provider loop.
func newTestEnv(t *testing.T) (*tests.TestApp, *config.Config) {
	t.Helper()
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("tests.NewTestApp: %v", err)
	}
	t.Cleanup(app.Cleanup)

	deskRoot := t.TempDir()
	cfg := &config.Config{
		DeskRoot:     deskRoot,
		DeskName:     "test-desk",
		DecisionsDir: "_structure/decisions",
		TasksDir:     "tasks",
		AnalysesDir:  "analyses",
		JournalDir:   "journal",
		SecretsDir:   "_meta/secrets",
		IgnoreConfig: filepath.Join(deskRoot, ".librarian-ignore"),
		HandoffPath:  "_meta/HANDOFF.md",
	}
	if err := os.WriteFile(cfg.IgnoreConfig, []byte("# empty — nothing ignored\n"), 0o644); err != nil {
		t.Fatalf("write ignore file: %v", err)
	}
	return app, cfg
}

func reloadTask(t *testing.T, app core.App, id string) *core.Record {
	t.Helper()
	rec, err := app.FindRecordById("tasks", id)
	if err != nil {
		t.Fatalf("reload task %s: %v", id, err)
	}
	return rec
}

// TestClaimOnce_RunsPatrolTask proves the claimer runs a deterministic kind straight through
// (claim -> tool call -> done) with NO LLM key and NO injected AgentAction at all.
func TestClaimOnce_RunsPatrolTask(t *testing.T) {
	app, cfg := newTestEnv(t)

	task, err := enqueue(app, "patrol", "test", nil)
	if err != nil {
		t.Fatalf("enqueue patrol task: %v", err)
	}

	claimed, err := ClaimOnce(context.Background(), app, cfg, nil)
	if err != nil {
		t.Fatalf("ClaimOnce: %v", err)
	}
	if !claimed {
		t.Fatalf("expected claimed=true")
	}

	after := reloadTask(t, app, task.Id)
	if got := after.GetString("state"); got != "done" {
		t.Fatalf("task state = %q, want done", got)
	}
	if after.GetDateTime("finished_at").IsZero() {
		t.Fatalf("expected finished_at to be set")
	}
}

// TestClaimOnce_NoTasks: an empty queue claims nothing and errors nothing.
func TestClaimOnce_NoTasks(t *testing.T) {
	app, cfg := newTestEnv(t)

	claimed, err := ClaimOnce(context.Background(), app, cfg, nil)
	if err != nil {
		t.Fatalf("ClaimOnce: %v", err)
	}
	if claimed {
		t.Fatalf("expected claimed=false on an empty queue")
	}
}

// TestClaimOnce_TransactionalClaim proves a task cannot be double-run: once ClaimOnce has
// carried the single queued task through to a terminal state, a second poll finds nothing
// left to claim.
func TestClaimOnce_TransactionalClaim(t *testing.T) {
	app, cfg := newTestEnv(t)

	if _, err := enqueue(app, "sweep", "test", nil); err != nil {
		t.Fatalf("enqueue sweep task: %v", err)
	}

	first, err := ClaimOnce(context.Background(), app, cfg, nil)
	if err != nil {
		t.Fatalf("first ClaimOnce: %v", err)
	}
	if !first {
		t.Fatalf("expected the first ClaimOnce to claim the only queued task")
	}

	second, err := ClaimOnce(context.Background(), app, cfg, nil)
	if err != nil {
		t.Fatalf("second ClaimOnce: %v", err)
	}
	if second {
		t.Fatalf("expected the second ClaimOnce to find nothing left to claim (no double-run)")
	}
}

// TestApplyFixTaskDeferredWhenGateOff: an enqueued apply_fix task with the autonomous-write
// gate off becomes deferred (terminal), never actually applied.
func TestApplyFixTaskDeferredWhenGateOff(t *testing.T) {
	app, cfg := newTestEnv(t)
	cfg.AutonomousWrites = false

	task, err := enqueue(app, "apply_fix", "test", nil)
	if err != nil {
		t.Fatalf("enqueue apply_fix task: %v", err)
	}

	claimed, err := ClaimOnce(context.Background(), app, cfg, nil)
	if err != nil {
		t.Fatalf("ClaimOnce: %v", err)
	}
	if !claimed {
		t.Fatalf("expected claimed=true (the task was claimed and driven to a terminal state)")
	}

	after := reloadTask(t, app, task.Id)
	if got := after.GetString("state"); got != "deferred" {
		t.Fatalf("task state = %q, want deferred", got)
	}
}

// TestRegisterHooks_EnqueuesPatrol proves the files-created hook enqueues a scoped patrol
// task rather than running the agent inline.
func TestRegisterHooks_EnqueuesPatrol(t *testing.T) {
	app, cfg := newTestEnv(t)

	if err := RegisterHooks(app, cfg); err != nil {
		t.Fatalf("RegisterHooks: %v", err)
	}

	filesColl, err := app.FindCollectionByNameOrId("files")
	if err != nil {
		t.Fatalf("find files collection: %v", err)
	}
	rec := core.NewRecord(filesColl)
	rec.Set("path", "tasks/example.md")
	rec.Set("desk", cfg.DeskName)
	if err := app.Save(rec); err != nil {
		t.Fatalf("save files record: %v", err)
	}

	tasks, err := app.FindRecordsByFilter("tasks", "state = 'queued' && kind = 'patrol'", "", 0, 0)
	if err != nil {
		t.Fatalf("list tasks: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected exactly one queued patrol task, got %d", len(tasks))
	}
	if got := tasks[0].GetString("source"); got != "OnRecordAfterCreateSuccess:files" {
		t.Fatalf("task source = %q, want OnRecordAfterCreateSuccess:files", got)
	}
}

// TestRegisterCron_Registers proves the hourly cron job is registered under a stable id.
func TestRegisterCron_Registers(t *testing.T) {
	app, cfg := newTestEnv(t)

	if err := RegisterCron(app, cfg); err != nil {
		t.Fatalf("RegisterCron: %v", err)
	}

	var found bool
	for _, j := range app.Cron().Jobs() {
		if j.Id() == "desk-patrol" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected a %q cron job to be registered", "desk-patrol")
	}
}

// TestClaimOnce_AgenticDispatchesToInjectedAction proves query/custom kinds call the
// injected AgentAction (never the real provider loop) and mark the task done on success.
func TestClaimOnce_AgenticDispatchesToInjectedAction(t *testing.T) {
	app, cfg := newTestEnv(t)

	task, err := enqueue(app, "query", "test", map[string]any{"query": "what changed today"})
	if err != nil {
		t.Fatalf("enqueue query task: %v", err)
	}

	var gotTrigger, gotInput string
	action := func(_ context.Context, _ core.App, _ *config.Config, trigger, input string) error {
		gotTrigger, gotInput = trigger, input
		return nil
	}

	claimed, err := ClaimOnce(context.Background(), app, cfg, action)
	if err != nil {
		t.Fatalf("ClaimOnce: %v", err)
	}
	if !claimed {
		t.Fatalf("expected claimed=true")
	}
	if gotTrigger != "task" {
		t.Fatalf("action trigger = %q, want task", gotTrigger)
	}
	if gotInput != "what changed today" {
		t.Fatalf("action input = %q, want the payload's query text", gotInput)
	}

	after := reloadTask(t, app, task.Id)
	if got := after.GetString("state"); got != "done" {
		t.Fatalf("task state = %q, want done", got)
	}
}

// TestClaimOnce_PanicInDispatchMarksFailed proves a panicking task (here an injected action
// that panics, standing in for a nil-deref in a tool function) is contained: ClaimOnce returns
// normally (it does NOT re-panic, so the claimer goroutine and the serve process survive) and
// the task is driven to the terminal failed state.
func TestClaimOnce_PanicInDispatchMarksFailed(t *testing.T) {
	app, cfg := newTestEnv(t)

	task, err := enqueue(app, "custom", "test", nil)
	if err != nil {
		t.Fatalf("enqueue custom task: %v", err)
	}

	panicAction := func(_ context.Context, _ core.App, _ *config.Config, _, _ string) error {
		panic("simulated tool panic")
	}

	// If the recover boundary were missing this call would itself panic and fail the test
	// process — reaching the assertions below is itself part of the proof that the loop
	// survives.
	claimed, err := ClaimOnce(context.Background(), app, cfg, panicAction)
	if err == nil {
		t.Fatalf("expected a non-nil error surfacing the recovered panic")
	}
	if !claimed {
		t.Fatalf("expected claimed=true (the task was claimed and driven to a terminal state)")
	}

	after := reloadTask(t, app, task.Id)
	if got := after.GetString("state"); got != "failed" {
		t.Fatalf("task state = %q, want failed", got)
	}
	if after.GetDateTime("finished_at").IsZero() {
		t.Fatalf("expected finished_at to be set on the failed task")
	}

	// The claimer can keep working: a second poll on the now-drained queue is a clean no-op.
	claimed2, err := ClaimOnce(context.Background(), app, cfg, panicAction)
	if err != nil {
		t.Fatalf("second ClaimOnce after a contained panic: %v", err)
	}
	if claimed2 {
		t.Fatalf("expected claimed=false: the failed task is terminal and nothing else is queued")
	}
}

// TestClaimOnce_AgenticWithoutActionFails: a query/custom task with no injected AgentAction
// fails loudly rather than silently no-opping or crashing.
func TestClaimOnce_AgenticWithoutActionFails(t *testing.T) {
	app, cfg := newTestEnv(t)

	task, err := enqueue(app, "custom", "test", nil)
	if err != nil {
		t.Fatalf("enqueue custom task: %v", err)
	}

	claimed, err := ClaimOnce(context.Background(), app, cfg, nil)
	if err == nil {
		t.Fatalf("expected an error when no AgentAction is injected for an agentic kind")
	}
	if !claimed {
		t.Fatalf("expected claimed=true (the task was claimed and driven to a terminal failed state)")
	}

	after := reloadTask(t, app, task.Id)
	if got := after.GetString("state"); got != "failed" {
		t.Fatalf("task state = %q, want failed", got)
	}
}

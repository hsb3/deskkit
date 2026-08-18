// Package trigger implements the wake layer (spec section on how PocketBase hooks wake the
// agent): event hooks and cron enqueue `tasks` rows rather than running the agent inline, and
// a single claimer goroutine started in OnServe polls the queue, claims the highest-priority
// queued row transactionally, runs the mapped work, and marks it done/failed.
//
// Deterministic kinds (sweep, patrol, propose_fix, apply_fix, restore) call the matching tool
// function directly — no model, no loop. Agentic kinds (query, custom) need the LLM ReAct
// loop; to keep this package testable with no provider key and free of an import cycle risk,
// that dispatch is injected via AgentAction rather than importing the agent package.
package trigger

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"

	"github.com/hsb3/deskkit/internal/core/config"
	"github.com/hsb3/deskkit/internal/modules/librarian/tools"
)

// AgentAction runs the LLM loop for agentic task kinds (query/custom). Injected so the
// claimer needs no LLM key in tests and does not depend on the agent package. trigger names
// the wake source (e.g. "task"); input is the free-form request text derived from the task's
// payload.
type AgentAction func(ctx context.Context, app core.App, cfg *config.Config, trigger, input string) error

// RegisterHooks binds the record-event side of the wake layer: a newly indexed file enqueues
// a scoped patrol task so the agent inspects just that file, rather than running inline in
// the hook (kept fast and non-blocking; the wake is auditable via the tasks row).
func RegisterHooks(app core.App, cfg *config.Config) error {
	app.OnRecordAfterCreateSuccess("files").BindFunc(func(e *core.RecordEvent) error {
		payload := map[string]any{
			"file_id": e.Record.Id,
			"path":    e.Record.GetString("path"),
		}
		if _, err := enqueue(e.App, "patrol", "OnRecordAfterCreateSuccess:files", payload); err != nil {
			return err
		}
		return e.Next()
	})
	return nil
}

// RegisterCron binds the hourly cron side of the wake layer: it enqueues a sweep task and a
// patrol task each tick. Enqueue failures inside the job are swallowed (a cron job has no
// caller to report to); the next tick will simply enqueue again.
func RegisterCron(app core.App, cfg *config.Config) error {
	return app.Cron().Add("desk-patrol", "0 * * * *", func() {
		if _, err := enqueue(app, "sweep", "cron:desk-patrol", nil); err != nil {
			return
		}
		if _, err := enqueue(app, "patrol", "cron:desk-patrol", nil); err != nil {
			return
		}
	})
}

// StartClaimer starts the single background claimer goroutine (spec: "one claimer goroutine
// started in OnServe"). It sleeps cfg.ClaimerPollInterval between polls and stops when ctx is
// cancelled. Each tick's claim/dispatch/finish cycle is ClaimOnce, kept as a separate
// exported, deterministic function so it can be unit tested without a running loop.
func StartClaimer(ctx context.Context, app core.App, cfg *config.Config, action AgentAction) {
	interval := cfg.ClaimerPollInterval
	if interval <= 0 {
		interval = 5 * time.Second
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Defense in depth: ClaimOnce already converts a panicking task into a
				// failed-marked task (safeDispatch), so it should not panic. This recover
				// guards the poll loop against a panic anywhere ELSE in the cycle (query,
				// transaction, save) so one bad iteration can never take down serve — the
				// loop logs and keeps polling.
				func() {
					defer func() {
						if r := recover(); r != nil {
							app.Logger().Error("claimer: recovered panic in poll cycle", "err", r)
						}
					}()
					_, _ = ClaimOnce(ctx, app, cfg, action)
				}()
			}
		}
	}()
}

// ClaimOnce finds the highest-priority queued task, claims it transactionally (re-checking
// state == "queued" inside the transaction so a second poll can never double-run the same
// row), runs the mapped work outside the claim transaction, and marks the task done/failed
// (or, for a gate-blocked apply_fix, deferred). claimed reports whether a task was claimed;
// when claimed is true and err is non-nil, the task was claimed AND run, but the run failed
// (the task row is already marked failed — err is returned for caller-side logging/testing).
func ClaimOnce(ctx context.Context, app core.App, cfg *config.Config, action AgentAction) (claimed bool, err error) {
	candidates, ferr := app.FindRecordsByFilter("tasks", "state = 'queued'", "-priority,created", 1, 0, dbx.Params{})
	if ferr != nil {
		return false, ferr
	}
	if len(candidates) == 0 {
		return false, nil
	}
	taskID := candidates[0].Id

	var claimedRec *core.Record
	txErr := app.RunInTransaction(func(txApp core.App) error {
		rec, gerr := txApp.FindRecordById("tasks", taskID)
		if gerr != nil {
			return gerr
		}
		if rec.GetString("state") != "queued" {
			// Lost the race to a concurrent poll (or the row was otherwise already moved
			// on) — nothing to claim.
			return nil
		}
		rec.Set("state", "claimed")
		rec.Set("claimed_at", time.Now().UTC())
		if serr := txApp.Save(rec); serr != nil {
			return serr
		}
		claimedRec = rec
		return nil
	})
	if txErr != nil {
		return false, txErr
	}
	if claimedRec == nil {
		return false, nil
	}

	deferred, runErr := safeDispatch(ctx, app, cfg, claimedRec, action)
	if deferred {
		// dispatch already set the terminal "deferred" state and saved it.
		return true, runErr
	}

	finished := time.Now().UTC()
	if runErr != nil {
		claimedRec.Set("state", "failed")
		claimedRec.Set("finished_at", finished)
		claimedRec.Set("payload", withError(payloadBytes(claimedRec), runErr))
		if serr := app.Save(claimedRec); serr != nil {
			return true, serr
		}
		return true, runErr
	}
	claimedRec.Set("state", "done")
	claimedRec.Set("finished_at", finished)
	if serr := app.Save(claimedRec); serr != nil {
		return true, serr
	}
	return true, nil
}

// safeDispatch runs dispatch with a recover so a panicking tool function or injected action
// (e.g. a nil-deref inside a claimed task) becomes a normal error — ClaimOnce then marks the
// task failed — rather than unwinding the claimer goroutine and taking the serve process down
// with it. This is the per-task failure boundary; StartClaimer's loop recover is the backstop
// for panics outside dispatch.
func safeDispatch(ctx context.Context, app core.App, cfg *config.Config, rec *core.Record, action AgentAction) (deferred bool, err error) {
	defer func() {
		if r := recover(); r != nil {
			deferred = false
			err = fmt.Errorf("trigger: task %s panicked during dispatch: %v", rec.Id, r)
		}
	}()
	return dispatch(ctx, app, cfg, rec, action)
}

// dispatch runs the claimed task's mapped work by kind. deferred reports whether dispatch
// itself decided a terminal state and already persisted it (currently only the gated
// apply_fix path); ClaimOnce owns the done/failed transition for every other outcome.
func dispatch(ctx context.Context, app core.App, cfg *config.Config, rec *core.Record, action AgentAction) (deferred bool, err error) {
	kind := rec.GetString("kind")
	payload := payloadBytes(rec)

	switch kind {
	case "sweep":
		var in tools.SweepInput
		_, err = tools.Sweep(ctx, app, cfg, &in)
		return false, err
	case "patrol":
		var in tools.PatrolInput
		_ = json.Unmarshal(payload, &in)
		_, err = tools.Patrol(ctx, app, cfg, &in)
		return false, err
	case "propose_fix":
		var in tools.ProposeFixInput
		_ = json.Unmarshal(payload, &in)
		_, err = tools.ProposeFix(ctx, app, cfg, &in)
		return false, err
	case "apply_fix":
		// §5.4 autonomous-write gate: an enqueued apply_fix only runs when
		// LIBRARIAN_AUTONOMOUS_WRITES is set; otherwise it is left deferred (terminal) for
		// a supervised CLI run rather than applied on the wake path.
		if !cfg.AutonomousWrites {
			rec.Set("state", "deferred")
			rec.Set("finished_at", time.Now().UTC())
			return true, app.Save(rec)
		}
		var in tools.ApplyFixInput
		_ = json.Unmarshal(payload, &in)
		_, err = tools.ApplyFix(ctx, app, cfg, &in)
		return false, err
	case "restore":
		var in tools.RestoreInput
		_ = json.Unmarshal(payload, &in)
		_, err = tools.Restore(ctx, app, cfg, &in)
		return false, err
	case "query", "custom":
		if action == nil {
			return false, fmt.Errorf("trigger: task %s kind %q needs an agentic action but none was injected", rec.Id, kind)
		}
		return false, action(ctx, app, cfg, "task", agenticInput(payload))
	default:
		return false, fmt.Errorf("trigger: task %s has unknown kind %q", rec.Id, kind)
	}
}

// enqueue creates one queued tasks row. Exported-in-spirit shape kept unexported since only
// this package's hook/cron/test callers need it; a payload of nil is stored as an empty
// object.
func enqueue(app core.App, kind, source string, payload map[string]any) (*core.Record, error) {
	coll, err := app.FindCollectionByNameOrId("tasks")
	if err != nil {
		return nil, err
	}
	rec := core.NewRecord(coll)
	rec.Set("kind", kind)
	if payload == nil {
		payload = map[string]any{}
	}
	rec.Set("payload", payload)
	rec.Set("state", "queued")
	rec.Set("priority", 0)
	rec.Set("source", source)
	if err := app.Save(rec); err != nil {
		return nil, err
	}
	return rec, nil
}

// payloadBytes normalizes the tasks.payload JSON field (stored/loaded as types.JSONRaw) to a
// plain []byte for json.Unmarshal, tolerating the handful of other shapes core.Record.Get
// could plausibly hand back for a JSON column.
func payloadBytes(rec *core.Record) []byte {
	switch v := rec.Get("payload").(type) {
	case types.JSONRaw:
		return []byte(v)
	case []byte:
		return v
	case string:
		return []byte(v)
	default:
		if v == nil {
			return nil
		}
		b, _ := json.Marshal(v)
		return b
	}
}

// agenticPayload is the optional shape an enqueued query/custom task's payload may carry: a
// "query" or "input" string field holding the free-form request text for the agent loop.
type agenticPayload struct {
	Query string `json:"query"`
	Input string `json:"input"`
}

// agenticInput derives the AgentAction input string from a task's raw payload: prefers a
// "query" field, then "input", and otherwise falls back to the raw payload JSON so nothing is
// silently dropped.
func agenticInput(payload []byte) string {
	var f agenticPayload
	if err := json.Unmarshal(payload, &f); err == nil {
		if f.Query != "" {
			return f.Query
		}
		if f.Input != "" {
			return f.Input
		}
	}
	if len(payload) == 0 {
		return ""
	}
	return string(payload)
}

// withError returns payload with an "error" key set to msg's text, for a failed task's
// audit trail. Falls back to a fresh object if the original payload was not a JSON object.
func withError(payload []byte, taskErr error) map[string]any {
	out := map[string]any{}
	if len(payload) > 0 {
		_ = json.Unmarshal(payload, &out)
	}
	out["error"] = taskErr.Error()
	return out
}

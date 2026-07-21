// Conversation + run persistence (spec §6.5). One agent_runs row per run; the ENTIRE
// transcript persisted to messages through ONE mechanism — an eino callback — so no message
// (tool result or otherwise) is written twice.
//
// Single-mechanism design (how double-persist is avoided): the callback fires on each
// ChatModel node INPUT (model inputs are always concrete messages, never streams, so this
// fires reliably regardless of whether the provider streams). Each model input is the
// CUMULATIVE conversation so far, so a high-water mark persists only the messages appended
// since the previous model call — the initial system+user, and every intermediate
// assistant(tool_calls)+tool turn. The FINAL assistant message (the loop's output) never
// appears in a later model input and is persisted once by the caller after the loop returns.
// The tool functions themselves never persist messages.
//
// hwm scope — per RUN for the one-shot Run(), but RE-BASELINED PER TURN for a multi-turn
// Session. A Session's model input restarts every turn at [system]+history+[userN], so a
// single per-run hwm would re-persist the prior turn's rows (duplicate) or skip the new user
// row (drop). StreamTurn (stream.go) therefore sets rc.hwm = 1 + len(history) before each
// turn's first model call (0 on the very first turn so the system row persists exactly once).
// With that baseline the first delta of every turn is exactly [userN], and the exactly-once
// [system, u1, a1, u2, a2, …] transcript shape holds across turns.
//
// messages.run always targets the agent_runs record's 15-char system id (rc.runID), never
// run_label (§4.7/§4.8/§6.5).
package agent

import (
	"context"
	"strings"
	"sync"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	cbutils "github.com/cloudwego/eino/utils/callbacks"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"

	"github.com/hsb3/desk-standard/librarian/internal/core/config"
)

// runCtx carries the runID and a mutex-guarded monotonic seq counter for one run. seq is
// strictly increasing per run, backed by the unique (run, seq) index (§4.7). hwm is the count
// of leading model-input messages already persisted (re-baselined per turn by StreamTurn for a
// multi-turn Session; see the file header); steps counts model generations (= assistant turns)
// for agent_runs.step_count.
type runCtx struct {
	app   core.App
	cfg   *config.Config
	runID string // == agent_runs record id (15-char), the messages.run relation target
	mu    sync.Mutex
	seq   int
	hwm   int
	steps int
	// pending buffers the CURRENT round's messages (the model's assistant output and its tool
	// result[s]) between model calls, for the one-shot Run() only. The input-side callback
	// (persistHandler) flushes a round only on the NEXT model call — where those messages first
	// appear in a model input — so a MaxStep abort right after a tool executes would lose the
	// round from the transcript even though the tool really ran. Run()'s MaxStep path flushes
	// this buffer (flushPending) to close that gap. Reset at each model OnStart (the prior round
	// is persisted there by the delta), so it only ever holds the not-yet-persisted current round.
	pending []*schema.Message
}

// persistLocked assigns the next seq and writes the message. Caller must hold rc.mu.
func (rc *runCtx) persistLocked(m *schema.Message) error {
	rc.seq++
	return persistMessage(rc.app, rc.runID, rc.seq, m)
}

// persist assigns the next seq and writes the message under the lock.
func (rc *runCtx) persist(m *schema.Message) error {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	return rc.persistLocked(m)
}

// persistMessage writes one message row (spec §6.5). Role is lowercased to match the
// messages.role select values (system/user/assistant/tool).
func persistMessage(app core.App, runID string, seq int, m *schema.Message) error {
	coll, err := app.FindCollectionByNameOrId("messages")
	if err != nil {
		return err
	}
	rec := core.NewRecord(coll)
	rec.Set("run", runID)
	rec.Set("seq", seq)
	rec.Set("role", strings.ToLower(string(m.Role))) // System/User/Assistant/Tool -> lowercase
	rec.Set("content", m.Content)
	if len(m.ToolCalls) > 0 {
		rec.Set("tool_calls", m.ToolCalls)
	}
	if m.ToolCallID != "" {
		rec.Set("tool_call_id", m.ToolCallID)
	}
	if m.ToolName != "" {
		rec.Set("tool_name", m.ToolName)
	}
	return app.Save(rec)
}

// persistHandler builds the ONE eino callback that persists the transcript. It hooks only the
// ChatModel node's INPUT (OnStart): inputs are concrete message slices even when the model
// streams its output, so this fires on every model call. Persisting the input high-water-mark
// delta captures system+user and every intermediate assistant+tool turn exactly once.
func (rc *runCtx) persistHandler() callbacks.Handler {
	modelHandler := &cbutils.ModelCallbackHandler{
		OnStart: func(ctx context.Context, info *callbacks.RunInfo, in *model.CallbackInput) context.Context {
			if in == nil {
				return ctx
			}
			rc.mu.Lock()
			defer rc.mu.Unlock()
			rc.steps++ // one model generation == one assistant turn
			delta, newHwm := deltaMessages(in.Messages, rc.hwm)
			for _, m := range delta {
				if err := rc.persistLocked(m); err != nil {
					rc.app.Logger().Error("persist message", "run", rc.runID, "seq", rc.seq, "err", err)
				}
			}
			rc.hwm = newHwm
			return ctx
		},
	}
	return cbutils.NewHandlerHelper().ChatModel(modelHandler).Handler()
}

// deltaMessages returns the input messages not yet persisted (those at index >= hwm, i.e.
// appended since the previous model call) and the new high-water mark. This is the invariant
// that prevents double-persistence: each cumulative model input is persisted only for its
// tail delta, so every message is written exactly once across the run's model calls.
func deltaMessages(msgs []*schema.Message, hwm int) ([]*schema.Message, int) {
	if hwm >= len(msgs) {
		return nil, hwm
	}
	return msgs[hwm:], len(msgs)
}

// captureHandler builds a SECONDARY eino callback that mirrors the current round's messages into
// rc.pending, in memory, WITHOUT persisting anything. It exists solely so the one-shot Run() can
// recover the last round when the loop aborts on MaxStep: the input-side persistHandler only
// flushes a round on the NEXT model call (that is when the round's assistant+tool messages first
// appear in a model input), so if MaxStep cuts the loop right after a tool executes, that round
// never reaches the transcript — even though the tool ran (e.g. propose_fix wrote a revisions
// row). Run()'s MaxStep path flushes rc.pending to close that audit-trail gap.
//
// Model OnEnd captures the exact assistant message eino produced (content + all tool_calls);
// Tool OnEnd reconstructs each tool result from the callback (call id, tool name, response). The
// model OnStart resets the buffer because the delta on the same event persists the prior round,
// so after a model call rc.pending only holds the not-yet-persisted current round. This handler
// is registered ONLY by Run(); the streaming Session path does not use it and is unchanged (it
// has the same MaxStep transcript gap, tracked separately, not fixed here).
//
// Ordering invariant this relies on: every tool OnEnd for round N must complete before round
// N+1's model OnStart resets rc.pending — true for eino's ReAct loop today (rc.mu only
// serializes individual appends, it doesn't order them). If a future change dispatches tool
// calls in a way that lets one straggle past the next model call's OnStart, that tool's result
// would be silently dropped from the buffer — the same class of risk persistHandler's own
// delta-based capture already has, not a new one introduced here.
func (rc *runCtx) captureHandler() callbacks.Handler {
	modelHandler := &cbutils.ModelCallbackHandler{
		OnStart: func(ctx context.Context, _ *callbacks.RunInfo, _ *model.CallbackInput) context.Context {
			rc.mu.Lock()
			rc.pending = nil // the prior round is being persisted by the input-side delta now
			rc.mu.Unlock()
			return ctx
		},
		OnEnd: func(ctx context.Context, _ *callbacks.RunInfo, out *model.CallbackOutput) context.Context {
			if out == nil || out.Message == nil {
				return ctx
			}
			rc.mu.Lock()
			rc.pending = append(rc.pending, out.Message)
			rc.mu.Unlock()
			return ctx
		},
	}
	toolHandler := &cbutils.ToolCallbackHandler{
		OnEnd: func(ctx context.Context, info *callbacks.RunInfo, out *tool.CallbackOutput) context.Context {
			if out == nil {
				return ctx
			}
			rc.mu.Lock()
			rc.pending = append(rc.pending, &schema.Message{
				Role:       schema.Tool,
				Content:    out.Response,
				ToolCallID: compose.GetToolCallID(ctx),
				ToolName:   toolName(info),
			})
			rc.mu.Unlock()
			return ctx
		},
	}
	return cbutils.NewHandlerHelper().ChatModel(modelHandler).Tool(toolHandler).Handler()
}

// flushPending writes the buffered current-round messages (the assistant tool-call message and
// its tool result[s]) that the input-side callback never flushed, assigning each the next run
// seq. Run() calls this ONLY when the loop aborts on MaxStep — the one outcome where a round's
// tool call really executed but no later model input carried it into the transcript. On every
// other outcome the buffer is either empty (a fresh model OnStart reset it before returning) or
// deliberately left unflushed (a successful final message is persisted by the caller instead),
// so this is never a double-persist.
func (rc *runCtx) flushPending() {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	for _, m := range rc.pending {
		if err := rc.persistLocked(m); err != nil {
			rc.app.Logger().Error("flush pending message", "run", rc.runID, "seq", rc.seq, "err", err)
		}
	}
	rc.pending = nil
}

// newRunLabel is a HUMAN-READABLE display label only, never the relation target.
func newRunLabel() string {
	return "run-" + types.NowDateTime().Time().UTC().Format("20060102T150405Z")
}

// createAgentRun opens the run row (status running); PocketBase assigns its 15-char id here,
// and that id is the runID the messages.run relation points at (spec §6.5).
func createAgentRun(app core.App, trigger, input string, cfg *config.Config) (*core.Record, error) {
	coll, err := app.FindCollectionByNameOrId("agent_runs")
	if err != nil {
		return nil, err
	}
	run := core.NewRecord(coll)
	run.Set("trigger", trigger) // hook | cron | manual | task
	run.Set("status", "running")
	run.Set("provider", cfg.LLMProvider)
	run.Set("model", cfg.LLMModel)
	run.Set("run_label", newRunLabel()) // display label, NOT the id
	run.Set("input_summary", summarize(input))
	run.Set("started", types.NowDateTime())
	if err := app.Save(run); err != nil { // assigns run.Id
		return nil, err
	}
	return run, nil
}

// finishRun patches the run to its terminal state (spec §6.5). A returned final message that
// still carries tool calls means MaxStep cut the loop short -> blocked (not a crash). A loop
// error -> failed (and is returned to the caller). Otherwise succeeded.
func finishRun(app core.App, run *core.Record, rc *runCtx, out *schema.Message, genErr error) error {
	run.Set("step_count", rc.steps)
	run.Set("finished", types.NowDateTime())
	switch {
	case genErr != nil:
		run.Set("status", "failed")
		run.Set("error", genErr.Error())
	case out != nil && len(out.ToolCalls) > 0:
		run.Set("status", "blocked") // MaxStep reached without a final answer
	default:
		run.Set("status", "succeeded")
		if out != nil {
			run.Set("output_summary", summarize(out.Content))
		}
	}
	if err := app.Save(run); err != nil {
		return err
	}
	return genErr
}

// failRun records a pre-loop failure (provider/tool/agent construction) and returns the error.
func failRun(app core.App, run *core.Record, rc *runCtx, cause error) error {
	run.Set("status", "failed")
	run.Set("error", cause.Error())
	run.Set("step_count", rc.steps)
	run.Set("finished", types.NowDateTime())
	if err := app.Save(run); err != nil {
		return err
	}
	return cause
}

// summarize trims a summary field to a bounded length (input_summary/output_summary are
// summaries, not full transcripts — the full text lives in messages).
func summarize(s string) string {
	const max = 500
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "…"
}

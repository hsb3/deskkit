// Conversation resume for the interactive chat surface (chat-TUI plan, Phase 4). A prior
// manual run persisted its whole transcript to messages (persist.go); resume rebuilds a live
// Session over that SAME agent_runs row so the model continues the conversation, and returns a
// human-facing transcript of what was said so far.
//
// Two distinct views come out of the stored rows:
//
//   - HISTORY (replayed to the model): only the clean alternating user + final-assistant
//     sequence NewSession/StreamTurn maintain — system rows, tool rows, and tool-calling
//     assistant rows are dropped, because the model rebuilds the system prompt itself
//     (newAgent's MessageModifier) and re-derives tool steps as needed. This mirrors what a
//     live Session keeps in Session.history.
//   - TRANSCRIPT (rendered to the human): every non-system row in order, including tool steps,
//     so the reader sees the full exchange.
//
// Orphan collapse (why HISTORY is not just "every user + final-assistant row"): a canceled or
// errored turn persists its user row at model-input time but never lands a final-assistant
// answer, leaving an orphaned user row. Replaying two consecutive user messages to the model is
// invalid, so a retained user message is kept IFF the next retained message is an assistant.
// That one rule covers every position: an interior orphan (u2 before u3), a leading orphan (u1
// before u2), and a trailing orphan (a final u4 with nothing after it — dropped so the next
// turn's model input is not two users back to back). Assistant messages are always kept.
package agent

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/cloudwego/eino/schema"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"

	"github.com/hsb3/deskkit/internal/core/config"
)

// transcriptCap bounds the human-facing transcript to its most recent rows, so resuming a very
// long conversation does not build an unbounded slice. The model-facing history has its own,
// separate bound (capHistory / maxHistoryMessages).
const transcriptCap = 200

// ConversationInfo is one row of the resume picker: a manual run and the fields needed to list
// and identify it. Title prefers the input summary, falling back to the display label. MsgCount is
// the count of persisted messages rows for the run; LastActivity is the most-recent message's
// created timestamp (falling back to Started when the run has no messages) — the two the sessions
// surface renders alongside the title so a reader can size and age a conversation before resuming.
type ConversationInfo struct {
	RunID        string
	Title        string
	Started      types.DateTime
	Status       string
	MsgCount     int
	LastActivity types.DateTime
	Archived     bool // soft-archived (hidden from the default resume list; see SetConversationArchived)
}

// TranscriptEntry is one rendered line of a resumed conversation. ToolName is set for tool rows
// and for tool-calling assistant rows (the invoked tool), and is empty otherwise.
type TranscriptEntry struct {
	Role     string // "user" | "assistant" | "tool"
	Text     string
	ToolName string
}

// ListConversations returns the most recent manual (chat) runs, newest started first, capped to
// limit. Kinds of runs excluded because they are not resumable conversations to offer:
//   - non-manual runs (hook/cron/task);
//   - the caller's own live run (excludeRunID) — a chat session creates its run row at launch,
//     so without this the picker's newest (default-selected) row is always the current session
//     itself, and "resuming" it replays an empty history;
//   - runs with an empty input_summary — the summary is backfilled on a run's first turn, so an
//     empty one means the run never had a turn and there is nothing to resume;
//   - archived runs, UNLESS includeArchived is set — a soft-archive hides a conversation from the
//     default resume list; passing includeArchived reveals them (so the sessions manager can offer
//     an unarchive). Archived runs come back with Archived set so the caller can mark them.
func ListConversations(app core.App, limit int, excludeRunID string, includeArchived bool) ([]ConversationInfo, error) {
	filter := "trigger = {:t} && id != {:x} && input_summary != ''"
	if !includeArchived {
		filter += " && archived != true"
	}
	runs, err := app.FindRecordsByFilter("agent_runs",
		filter, "-started", limit, 0, dbx.Params{"t": "manual", "x": excludeRunID})
	if err != nil {
		return nil, err
	}
	out := make([]ConversationInfo, 0, len(runs))
	for _, r := range runs {
		title := r.GetString("input_summary")
		if title == "" {
			title = r.GetString("run_label")
		}
		// Per-run message count + last activity. pickerLimit is small (50) and both are fast local
		// reads on the (run) index, so a per-run query is fine here rather than a grouped join.
		count, err := app.CountRecords("messages", dbx.HashExp{"run": r.Id})
		if err != nil {
			return nil, err
		}
		last := r.GetDateTime("started") // fall back to the run's start when it has no messages
		latest, err := app.FindRecordsByFilter("messages", "run = {:r}", "-created", 1, 0, dbx.Params{"r": r.Id})
		if err != nil {
			return nil, err
		}
		if len(latest) > 0 {
			last = latest[0].GetDateTime("created")
		}
		out = append(out, ConversationInfo{
			RunID:        r.Id,
			Title:        title,
			Started:      r.GetDateTime("started"),
			Status:       r.GetString("status"),
			MsgCount:     int(count),
			LastActivity: last,
			Archived:     r.GetBool("archived"),
		})
	}
	return out, nil
}

// ResumeSession rebuilds a live Session over an existing manual run so the conversation can
// continue. It reconstructs the agent exactly as NewSession does, rehydrates the model-facing
// history from the stored transcript (filtered and orphan-collapsed), restores the seq/step
// counters so the next turn appends without violating the unique (run, seq) index, reopens the
// run row (status back to running, finished cleared), and returns the human-facing transcript.
//
// The returned Session leaves mu/busy/termErr zero — it is ready for its first resumed Turn.
func ResumeSession(ctx context.Context, app core.App, cfg *config.Config, runID string) (*Session, []TranscriptEntry, error) {
	run, err := app.FindRecordById("agent_runs", runID)
	if err != nil {
		return nil, nil, err
	}

	// Rebuild the agent in the SAME constructor order as NewSession. On any build failure we
	// return the error unwrapped; unlike NewSession we do not fail the run row, because resume
	// must not corrupt an already-finalized prior run just because reconstruction hiccuped.
	rc := &runCtx{app: app, cfg: cfg, runID: run.Id}
	chatModel, err := chatModelFactory(ctx, cfg)
	if err != nil {
		return nil, nil, err
	}
	toolset, err := buildTools(app, cfg) // §5.4 gating: restore absent, apply_fix gated
	if err != nil {
		return nil, nil, err
	}
	ag, err := newAgent(ctx, app, chatModel, toolset, cfg) // MessageModifier prepends systemPrompt
	if err != nil {
		return nil, nil, err
	}

	// Load the whole transcript in seq order (limit far exceeds any realistic run length).
	rows, err := app.FindRecordsByFilter("messages", "run = {:r}", "seq", 100000, 0, dbx.Params{"r": run.Id})
	if err != nil {
		return nil, nil, err
	}

	history := rehydrateHistory(rows)
	last := lastAssistant(history)
	transcript := buildTranscript(rows)

	// Counter continuity: seq must resume above the highest persisted row so the next turn's
	// rows do not collide with the unique (run, seq) index. Rows are seq-ascending, so the last
	// row holds the max, but loop defensively in case ordering ever changes.
	maxSeq := 0
	for _, r := range rows {
		if s := r.GetInt("seq"); s > maxSeq {
			maxSeq = s
		}
	}
	rc.seq = maxSeq
	rc.steps = run.GetInt("step_count")
	// rc.hwm stays 0 here. stream.go's runTurn keys firstTurn on rc.seq == 0; a resumed session
	// has seq > 0, so firstTurn is false and StreamTurn re-baselines rc.hwm = 1 + len(history)
	// on the next turn's first model call automatically. The first-turn input_summary backfill
	// is likewise skipped, preserving the original conversation's title.

	// Reopen the run row: it was finalized (succeeded/failed/blocked) with a finished timestamp;
	// resuming makes it live again. A zero types.DateTime clears the DateField (stored empty).
	run.Set("status", "running")
	run.Set("finished", types.DateTime{})
	if err := app.Save(run); err != nil {
		return nil, nil, err
	}

	return &Session{
		app:     app,
		cfg:     cfg,
		agent:   ag,
		run:     run,
		rc:      rc,
		history: history,
		last:    last,
	}, transcript, nil
}

// rehydrateHistory converts the stored rows into the model-facing history: the clean user +
// final-assistant alternation, with system/tool/tool-calling-assistant rows dropped and orphaned
// user rows collapsed (see the file header). capHistory bounds the result.
func rehydrateHistory(rows []*core.Record) []*schema.Message {
	type candidate struct {
		msg    *schema.Message
		isUser bool
	}
	var cands []candidate
	for _, r := range rows {
		switch r.GetString("role") {
		case "user":
			cands = append(cands, candidate{msg: schema.UserMessage(r.GetString("content")), isUser: true})
		case "assistant":
			if !hasToolCalls(r) { // final answers only; tool-calling rows are skipped
				cands = append(cands, candidate{msg: schema.AssistantMessage(r.GetString("content"), nil), isUser: false})
			}
		default:
			// system and tool rows are never replayed to the model.
		}
	}

	var history []*schema.Message
	for i, c := range cands {
		if !c.isUser {
			history = append(history, c.msg) // assistants are always retained
			continue
		}
		// Keep a user message IFF the next retained message is an assistant. Assistants are
		// always retained, so this is exactly "the next candidate is an assistant" — which drops
		// interior, leading, and trailing orphaned user rows alike.
		if i+1 < len(cands) && !cands[i+1].isUser {
			history = append(history, c.msg)
		}
	}
	return capHistory(history)
}

// lastAssistant returns the most recent assistant message in the rehydrated history (what
// Close -> finishRun re-finalizes), or nil if the history has no assistant turn.
func lastAssistant(history []*schema.Message) *schema.Message {
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].Role == schema.Assistant {
			return history[i]
		}
	}
	return nil
}

// buildTranscript renders every non-system row (in seq order) as a TranscriptEntry, capped to
// the most recent transcriptCap rows. Tool steps are included so the human sees the full
// exchange; ToolName is set on tool rows and tool-calling assistant rows.
func buildTranscript(rows []*core.Record) []TranscriptEntry {
	return renderTranscriptRows(nonSystemRows(rows), transcriptCap)
}

// nonSystemRows drops the system rows (never shown to the human), preserving order. System rows
// are the model's own prompt scaffolding, not part of the readable exchange.
func nonSystemRows(rows []*core.Record) []*core.Record {
	visible := make([]*core.Record, 0, len(rows))
	for _, r := range rows {
		if r.GetString("role") == "system" {
			continue
		}
		visible = append(visible, r)
	}
	return visible
}

// renderTranscriptRows renders the given rows as TranscriptEntry values, keeping only the most
// recent maxRows when the slice is longer (maxRows <= 0 keeps all). Shared by buildTranscript
// (resume) and PreviewConversation (the picker's preview pane) so the two render identically.
func renderTranscriptRows(rows []*core.Record, maxRows int) []TranscriptEntry {
	if maxRows > 0 && len(rows) > maxRows {
		rows = rows[len(rows)-maxRows:]
	}
	out := make([]TranscriptEntry, 0, len(rows))
	for _, r := range rows {
		out = append(out, renderTranscriptRow(r))
	}
	return out
}

// renderTranscriptRow renders one message row as a TranscriptEntry. ToolName is set on a
// tool-calling assistant row (the invoked tool) and on a tool row (its tool_name), empty otherwise.
func renderTranscriptRow(r *core.Record) TranscriptEntry {
	role := r.GetString("role")
	e := TranscriptEntry{Role: role, Text: r.GetString("content")}
	switch role {
	case "assistant":
		if calls := toolCallsOf(r); len(calls) > 0 {
			e.ToolName = calls[0].Function.Name
		}
	case "tool":
		e.ToolName = r.GetString("tool_name")
	}
	return e
}

// hasToolCalls reports whether a message row carries tool calls, judged by the raw field alone.
// persistMessage only sets tool_calls when a message actually carries them, so ANY non-empty
// value marks a tool-calling row — including one whose JSON no longer parses (e.g. truncated by
// a crash). History rehydration keys on this, not on toolCallsOf, so a corrupt row fails closed
// (excluded from the model-facing history) instead of being misread as a final answer.
func hasToolCalls(r *core.Record) bool {
	raw := strings.TrimSpace(r.GetString("tool_calls"))
	return raw != "" && raw != "null" && raw != "[]"
}

// toolCallsOf parses a message row's tool_calls JSON, for display purposes (the invoked tool's
// name in the transcript). An empty, "null", "[]", or unparseable payload yields nil — display
// gracefully degrades to no tool name, while hasToolCalls still classifies the row correctly.
func toolCallsOf(r *core.Record) []schema.ToolCall {
	raw := strings.TrimSpace(r.GetString("tool_calls"))
	if raw == "" || raw == "null" || raw == "[]" {
		return nil
	}
	var calls []schema.ToolCall
	if err := json.Unmarshal([]byte(raw), &calls); err != nil {
		return nil
	}
	return calls
}

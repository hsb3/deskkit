package agent

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/schema"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

// seedMessage writes one messages row at an explicit seq, mirroring persistMessage's field
// layout but under the test's control (so orphan/filtering patterns can be seeded verbatim).
func seedMessage(t *testing.T, app core.App, runID string, seq int, role, content string, toolCalls []schema.ToolCall, toolName string) {
	t.Helper()
	coll, err := app.FindCollectionByNameOrId("messages")
	if err != nil {
		t.Fatalf("find messages collection: %v", err)
	}
	rec := core.NewRecord(coll)
	rec.Set("run", runID)
	rec.Set("seq", seq)
	rec.Set("role", role)
	rec.Set("content", content)
	if len(toolCalls) > 0 {
		rec.Set("tool_calls", toolCalls)
	}
	if toolName != "" {
		rec.Set("tool_name", toolName)
	}
	if err := app.Save(rec); err != nil {
		t.Fatalf("seed message seq=%d: %v", seq, err)
	}
}

// oneToolCall builds a single-entry tool_calls payload naming the given tool.
func oneToolCall(name string) []schema.ToolCall {
	return []schema.ToolCall{{
		ID:       "call-" + name,
		Type:     "function",
		Function: schema.FunctionCall{Name: name, Arguments: "{}"},
	}}
}

// contentsOf extracts the content string of each history message.
func contentsOf(h []*schema.Message) []string {
	out := make([]string, len(h))
	for i, m := range h {
		out[i] = m.Content
	}
	return out
}

// historyRolesOf extracts the schema role of each history message as a lowercase string.
func historyRolesOf(h []*schema.Message) []string {
	out := make([]string, len(h))
	for i, m := range h {
		switch m.Role {
		case schema.User:
			out[i] = "user"
		case schema.Assistant:
			out[i] = "assistant"
		default:
			out[i] = string(m.Role)
		}
	}
	return out
}

// TestResume_RehydrationFiltering: system, tool, and tool-calling-assistant rows are excluded
// from the model-facing history; only user + final-assistant rows survive.
func TestResume_RehydrationFiltering(t *testing.T) {
	installModel(t, &scriptedModel{})
	app, cfg := newSessionTestEnv(t)
	ctx := context.Background()

	run, err := createAgentRun(app, "manual", "", cfg)
	if err != nil {
		t.Fatalf("createAgentRun: %v", err)
	}
	seedMessage(t, app, run.Id, 1, "system", "you are the librarian", nil, "")
	seedMessage(t, app, run.Id, 2, "user", "u1", nil, "")
	seedMessage(t, app, run.Id, 3, "assistant", "a1", nil, "")                       // final answer
	seedMessage(t, app, run.Id, 4, "assistant", "", oneToolCall("query"), "")        // tool-calling
	seedMessage(t, app, run.Id, 5, "tool", "tool result", nil, "query")              // tool row
	seedMessage(t, app, run.Id, 6, "user", "u2", nil, "")
	seedMessage(t, app, run.Id, 7, "assistant", "a2", nil, "")

	sess, transcript, err := ResumeSession(ctx, app, cfg, run.Id)
	if err != nil {
		t.Fatalf("ResumeSession: %v", err)
	}

	wantRoles := []string{"user", "assistant", "user", "assistant"}
	if got := historyRolesOf(sess.history); !equalStrings(got, wantRoles) {
		t.Fatalf("history roles = %v, want %v", got, wantRoles)
	}
	wantContent := []string{"u1", "a1", "u2", "a2"}
	if got := contentsOf(sess.history); !equalStrings(got, wantContent) {
		t.Fatalf("history contents = %v, want %v", got, wantContent)
	}

	// The transcript keeps every non-system row in order, including the tool step.
	wantTranscript := []struct {
		role, text, tool string
	}{
		{"user", "u1", ""},
		{"assistant", "a1", ""},
		{"assistant", "", "query"}, // tool-calling assistant carries the invoked tool name
		{"tool", "tool result", "query"},
		{"user", "u2", ""},
		{"assistant", "a2", ""},
	}
	if len(transcript) != len(wantTranscript) {
		t.Fatalf("transcript len = %d, want %d (%+v)", len(transcript), len(wantTranscript), transcript)
	}
	for i, w := range wantTranscript {
		e := transcript[i]
		if e.Role != w.role || e.Text != w.text || e.ToolName != w.tool {
			t.Fatalf("transcript[%d] = %+v, want role=%q text=%q tool=%q", i, e, w.role, w.text, w.tool)
		}
	}
}

// TestRehydrateHistory_CorruptToolCallsFailClosed: an assistant row whose tool_calls payload no
// longer parses (e.g. truncated by a crash) is still classified as tool-calling and EXCLUDED from
// the model-facing history — it must not be misread as a final answer.
func TestRehydrateHistory_CorruptToolCallsFailClosed(t *testing.T) {
	app, _ := newSessionTestEnv(t)
	coll, err := app.FindCollectionByNameOrId("messages")
	if err != nil {
		t.Fatalf("find messages collection: %v", err)
	}
	mkRow := func(role, content, rawToolCalls string) *core.Record {
		rec := core.NewRecord(coll)
		rec.Set("role", role)
		rec.Set("content", content)
		if rawToolCalls != "" {
			rec.Set("tool_calls", types.JSONRaw(rawToolCalls))
		}
		return rec
	}

	rows := []*core.Record{
		mkRow("user", "u1", ""),
		mkRow("assistant", "step commentary", `[{"id":"call-1","type":"function","fu`), // truncated JSON
		mkRow("assistant", "a1", ""),
	}
	got := contentsOf(rehydrateHistory(rows))
	want := []string{"u1", "a1"}
	if !equalStrings(got, want) {
		t.Fatalf("history contents = %v, want %v (corrupt tool-calling row must be excluded)", got, want)
	}
}

// TestResume_OrphanCollapse proves the orphan-collapse rule for interior, leading, and trailing
// orphaned user rows (canceled/errored turns leave a user row with no assistant answer).
func TestResume_OrphanCollapse(t *testing.T) {
	type seed struct {
		role, content string
	}
	cases := []struct {
		name  string
		rows  []seed
		want  []string // expected history contents
	}{
		{
			name: "interior",
			rows: []seed{{"user", "u1"}, {"assistant", "a1"}, {"user", "u2"}, {"user", "u3"}, {"assistant", "a3"}},
			want: []string{"u1", "a1", "u3", "a3"},
		},
		{
			name: "leading",
			rows: []seed{{"user", "u1"}, {"user", "u2"}, {"assistant", "a2"}},
			want: []string{"u2", "a2"},
		},
		{
			name: "trailing",
			rows: []seed{{"user", "u1"}, {"assistant", "a1"}, {"user", "u2"}},
			want: []string{"u1", "a1"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			installModel(t, &scriptedModel{})
			app, cfg := newSessionTestEnv(t)
			run, err := createAgentRun(app, "manual", "", cfg)
			if err != nil {
				t.Fatalf("createAgentRun: %v", err)
			}
			for i, s := range tc.rows {
				seedMessage(t, app, run.Id, i+1, s.role, s.content, nil, "")
			}
			sess, _, err := ResumeSession(context.Background(), app, cfg, run.Id)
			if err != nil {
				t.Fatalf("ResumeSession: %v", err)
			}
			if got := contentsOf(sess.history); !equalStrings(got, tc.want) {
				t.Fatalf("history = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestResume_CapHistory: a run with more retained rows than maxHistoryMessages rehydrates to a
// history bounded by the cap.
func TestResume_CapHistory(t *testing.T) {
	installModel(t, &scriptedModel{})
	app, cfg := newSessionTestEnv(t)
	run, err := createAgentRun(app, "manual", "", cfg)
	if err != nil {
		t.Fatalf("createAgentRun: %v", err)
	}
	// Seed many complete user+assistant pairs; every candidate is retained (each user is
	// followed by an assistant), so the pre-cap history far exceeds maxHistoryMessages.
	pairs := maxHistoryMessages // 40 pairs -> 80 retained messages pre-cap
	seq := 0
	for i := 0; i < pairs; i++ {
		seq++
		seedMessage(t, app, run.Id, seq, "user", "u", nil, "")
		seq++
		seedMessage(t, app, run.Id, seq, "assistant", "a", nil, "")
	}
	sess, _, err := ResumeSession(context.Background(), app, cfg, run.Id)
	if err != nil {
		t.Fatalf("ResumeSession: %v", err)
	}
	if len(sess.history) > maxHistoryMessages {
		t.Fatalf("history len = %d, want <= %d", len(sess.history), maxHistoryMessages)
	}
	if len(sess.history) != maxHistoryMessages {
		t.Fatalf("history len = %d, want exactly %d (cap keeps a full window)", len(sess.history), maxHistoryMessages)
	}
}

// TestResume_SeqContinuation is the unique-(run,seq)-index regression: after resuming a run with
// existing rows (max seq N), the next turn appends rows at seq > N with a dense, gap-free
// transcript — no index collision.
func TestResume_SeqContinuation(t *testing.T) {
	m := &scriptedModel{steps: []streamStep{contentStep("resumed answer")}}
	installModel(t, m)
	app, cfg := newSessionTestEnv(t)
	run, err := createAgentRun(app, "manual", "", cfg)
	if err != nil {
		t.Fatalf("createAgentRun: %v", err)
	}
	run.Set("step_count", 1)
	if err := app.Save(run); err != nil {
		t.Fatalf("save run: %v", err)
	}
	// A prior turn: system, user, final-assistant at seq 1..3.
	seedMessage(t, app, run.Id, 1, "system", "you are the librarian", nil, "")
	seedMessage(t, app, run.Id, 2, "user", "earlier question", nil, "")
	seedMessage(t, app, run.Id, 3, "assistant", "earlier answer", nil, "")

	sess, _, err := ResumeSession(context.Background(), app, cfg, run.Id)
	if err != nil {
		t.Fatalf("ResumeSession: %v", err)
	}
	if sess.rc.seq != 3 {
		t.Fatalf("resumed rc.seq = %d, want 3 (max seed seq)", sess.rc.seq)
	}

	out, err := sess.Turn(context.Background(), "next question")
	if err != nil {
		t.Fatalf("resumed turn: %v", err)
	}
	if out != "resumed answer" {
		t.Fatalf("turn content = %q, want %q", out, "resumed answer")
	}

	recs := loadRows(t, app, run.Id)
	assertDenseSeq(t, recs) // dense 1..M across the whole run — no unique-index violation
	if len(recs) <= 3 {
		t.Fatalf("expected the resumed turn to append rows, still %d rows", len(recs))
	}
	if last := recs[len(recs)-1]; last.GetInt("seq") <= 3 {
		t.Fatalf("last seq = %d, want > 3 (turn appended above the seed)", last.GetInt("seq"))
	}
	wantRoles := []string{"system", "user", "assistant", "user", "assistant"}
	if got := rolesOf(recs); !equalStrings(got, wantRoles) {
		t.Fatalf("roles = %v, want %v", got, wantRoles)
	}
}

// TestListConversations: newest-first ordering by started; non-manual runs, the caller's own
// live run (excludeRunID), and never-turned runs (empty input_summary — nothing to resume) are
// all excluded.
func TestListConversations(t *testing.T) {
	app, cfg := newSessionTestEnv(t)

	mkRun := func(trigger, input string, started types.DateTime) *core.Record {
		r, err := createAgentRun(app, trigger, input, cfg)
		if err != nil {
			t.Fatalf("createAgentRun: %v", err)
		}
		r.Set("started", started)
		if err := app.Save(r); err != nil {
			t.Fatalf("save run: %v", err)
		}
		return r
	}

	t0 := types.NowDateTime()
	older, _ := types.ParseDateTime(t0.Time().Add(-2 * 60 * 1e9)) // t0 - 2m
	newer, _ := types.ParseDateTime(t0.Time().Add(-1 * 60 * 1e9)) // t0 - 1m

	oldRun := mkRun("manual", "first chat", older)  // resumable
	newRun := mkRun("manual", "second chat", newer) // resumable, newer
	emptyRun := mkRun("manual", "", t0)             // never-turned (empty input_summary): excluded
	liveRun := mkRun("manual", "live chat", t0)     // the caller's own run: excluded by ID
	mkRun("hook", "a hook run", t0)                 // non-manual: excluded

	convos, err := ListConversations(app, 10, liveRun.Id)
	if err != nil {
		t.Fatalf("ListConversations: %v", err)
	}
	if len(convos) != 2 {
		t.Fatalf("got %d conversations, want 2 (non-manual, live, and empty runs excluded); %+v", len(convos), convos)
	}
	for _, c := range convos {
		if c.RunID == liveRun.Id {
			t.Fatalf("the caller's own live run %s was offered for resume", liveRun.Id)
		}
		if c.RunID == emptyRun.Id {
			t.Fatalf("the never-turned run %s was offered for resume", emptyRun.Id)
		}
	}
	// Newest started first.
	if convos[0].RunID != newRun.Id || convos[1].RunID != oldRun.Id {
		t.Fatalf("ordering = [%s, %s], want newest first [%s, %s]",
			convos[0].RunID, convos[1].RunID, newRun.Id, oldRun.Id)
	}
	// Titles come from input_summary.
	if convos[0].Title != "second chat" || convos[1].Title != "first chat" {
		t.Fatalf("titles = [%q, %q], want input summaries", convos[0].Title, convos[1].Title)
	}
	if convos[0].Status == "" {
		t.Fatalf("Status not populated: %+v", convos[0])
	}
}

// TestResume_ReopenAndClose: ResumeSession reopens a finalized run (status running, finished
// cleared), and Close re-finalizes it to a terminal status with finished set.
func TestResume_ReopenAndClose(t *testing.T) {
	installModel(t, &scriptedModel{})
	app, cfg := newSessionTestEnv(t)
	run, err := createAgentRun(app, "manual", "prior chat", cfg)
	if err != nil {
		t.Fatalf("createAgentRun: %v", err)
	}
	// Finalize it as a prior succeeded run would be.
	run.Set("status", "succeeded")
	run.Set("finished", types.NowDateTime())
	run.Set("step_count", 1)
	if err := app.Save(run); err != nil {
		t.Fatalf("finalize seed run: %v", err)
	}
	seedMessage(t, app, run.Id, 1, "system", "you are the librarian", nil, "")
	seedMessage(t, app, run.Id, 2, "user", "prior question", nil, "")
	seedMessage(t, app, run.Id, 3, "assistant", "prior answer", nil, "")

	sess, _, err := ResumeSession(context.Background(), app, cfg, run.Id)
	if err != nil {
		t.Fatalf("ResumeSession: %v", err)
	}

	reopened, err := app.FindRecordById("agent_runs", run.Id)
	if err != nil {
		t.Fatalf("reload run: %v", err)
	}
	if got := reopened.GetString("status"); got != "running" {
		t.Fatalf("reopened status = %q, want running", got)
	}
	if !reopened.GetDateTime("finished").IsZero() {
		t.Fatalf("finished not cleared on resume: %v", reopened.GetDateTime("finished"))
	}

	// Close re-finalizes using the last rehydrated assistant message.
	if err := sess.Close(context.Background()); err != nil {
		t.Fatalf("Close: %v", err)
	}
	final, err := app.FindRecordById("agent_runs", run.Id)
	if err != nil {
		t.Fatalf("reload run after close: %v", err)
	}
	if got := final.GetString("status"); got != "succeeded" {
		t.Fatalf("status after Close = %q, want succeeded", got)
	}
	if final.GetDateTime("finished").IsZero() {
		t.Fatalf("finished not set after Close")
	}
}

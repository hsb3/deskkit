package agent

import (
	"testing"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/tools/types"
)

// TestRenameConversation_SetsTitle: a rename replaces the run's input_summary (the title the
// picker lists) with the trimmed new title, verbatim.
func TestRenameConversation_SetsTitle(t *testing.T) {
	app, cfg := newSessionTestEnv(t)
	run, err := createAgentRun(app, "manual", "original title", cfg)
	if err != nil {
		t.Fatalf("createAgentRun: %v", err)
	}

	if err := RenameConversation(app, run.Id, "  a renamed conversation  "); err != nil {
		t.Fatalf("RenameConversation: %v", err)
	}

	reloaded, err := app.FindRecordById("agent_runs", run.Id)
	if err != nil {
		t.Fatalf("reload run: %v", err)
	}
	if got := reloaded.GetString("input_summary"); got != "a renamed conversation" {
		t.Fatalf("title after rename = %q, want %q (trimmed, verbatim)", got, "a renamed conversation")
	}
}

// TestRenameConversation_RejectsEmpty: a blank or whitespace-only title is rejected and the
// stored title is left untouched — an empty input_summary would hide the run from the list.
func TestRenameConversation_RejectsEmpty(t *testing.T) {
	app, cfg := newSessionTestEnv(t)
	run, err := createAgentRun(app, "manual", "keep me", cfg)
	if err != nil {
		t.Fatalf("createAgentRun: %v", err)
	}

	for _, bad := range []string{"", "   ", "\t\n"} {
		if err := RenameConversation(app, run.Id, bad); err == nil {
			t.Fatalf("RenameConversation(%q) returned nil, want a rejection error", bad)
		}
	}

	reloaded, err := app.FindRecordById("agent_runs", run.Id)
	if err != nil {
		t.Fatalf("reload run: %v", err)
	}
	if got := reloaded.GetString("input_summary"); got != "keep me" {
		t.Fatalf("title after rejected rename = %q, want the original %q untouched", got, "keep me")
	}
}

// TestRenameConversation_MissingRun: renaming a nonexistent run surfaces the store's not-found
// error rather than silently succeeding.
func TestRenameConversation_MissingRun(t *testing.T) {
	app, _ := newSessionTestEnv(t)
	if err := RenameConversation(app, "nope-not-a-real-id", "x"); err == nil {
		t.Fatal("RenameConversation on a missing run returned nil, want a not-found error")
	}
}

// TestDeleteConversation_CascadesMessages: a hard delete removes the run row AND its messages
// (the messages.run relation carries CascadeDelete, collections 0007), leaving nothing behind.
func TestDeleteConversation_CascadesMessages(t *testing.T) {
	app, cfg := newSessionTestEnv(t)
	run, err := createAgentRun(app, "manual", "doomed chat", cfg)
	if err != nil {
		t.Fatalf("createAgentRun: %v", err)
	}
	seedMessage(t, app, run.Id, 1, "system", "you are the librarian", nil, "")
	seedMessage(t, app, run.Id, 2, "user", "a question", nil, "")
	seedMessage(t, app, run.Id, 3, "assistant", "an answer", nil, "")

	// Sanity: the messages exist before the delete.
	before, err := app.CountRecords("messages", dbxRun(run.Id))
	if err != nil {
		t.Fatalf("count messages before delete: %v", err)
	}
	if before != 3 {
		t.Fatalf("messages before delete = %d, want 3", before)
	}

	if err := DeleteConversation(app, run.Id); err != nil {
		t.Fatalf("DeleteConversation: %v", err)
	}

	// The run row is gone.
	if _, err := app.FindRecordById("agent_runs", run.Id); err == nil {
		t.Fatal("run row still present after DeleteConversation")
	}
	// The messages cascaded away with it.
	after, err := app.CountRecords("messages", dbxRun(run.Id))
	if err != nil {
		t.Fatalf("count messages after delete: %v", err)
	}
	if after != 0 {
		t.Fatalf("messages after delete = %d, want 0 (CascadeDelete should remove them)", after)
	}
}

// TestDeleteConversation_MissingRun: deleting a nonexistent run surfaces a not-found error.
func TestDeleteConversation_MissingRun(t *testing.T) {
	app, _ := newSessionTestEnv(t)
	if err := DeleteConversation(app, "nope-not-a-real-id"); err == nil {
		t.Fatal("DeleteConversation on a missing run returned nil, want a not-found error")
	}
}

// TestPreviewConversation_TailNonSystem: preview returns the last maxRows non-system rows in seq
// order, rendered as TranscriptEntry (system rows dropped, tool names carried) — identical to the
// resume transcript's rendering.
func TestPreviewConversation_TailNonSystem(t *testing.T) {
	app, cfg := newSessionTestEnv(t)
	run, err := createAgentRun(app, "manual", "chat", cfg)
	if err != nil {
		t.Fatalf("createAgentRun: %v", err)
	}
	seedMessage(t, app, run.Id, 1, "system", "you are the librarian", nil, "")
	seedMessage(t, app, run.Id, 2, "user", "u1", nil, "")
	seedMessage(t, app, run.Id, 3, "assistant", "", oneToolCall("query"), "") // tool-calling assistant
	seedMessage(t, app, run.Id, 4, "tool", "tool result", nil, "query")       // tool row
	seedMessage(t, app, run.Id, 5, "assistant", "a1", nil, "")                // final answer

	// maxRows large enough to keep all non-system rows.
	got, err := PreviewConversation(app, run.Id, 20)
	if err != nil {
		t.Fatalf("PreviewConversation: %v", err)
	}
	want := []struct{ role, text, tool string }{
		{"user", "u1", ""},
		{"assistant", "", "query"},
		{"tool", "tool result", "query"},
		{"assistant", "a1", ""},
	}
	if len(got) != len(want) {
		t.Fatalf("preview len = %d, want %d (system row dropped); %+v", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i].Role != w.role || got[i].Text != w.text || got[i].ToolName != w.tool {
			t.Fatalf("preview[%d] = %+v, want role=%q text=%q tool=%q", i, got[i], w.role, w.text, w.tool)
		}
	}
}

// TestPreviewConversation_CapsToMaxRows: with more non-system rows than maxRows, preview keeps
// only the most recent maxRows (the tail), so the pane shows the latest exchange.
func TestPreviewConversation_CapsToMaxRows(t *testing.T) {
	app, cfg := newSessionTestEnv(t)
	run, err := createAgentRun(app, "manual", "long chat", cfg)
	if err != nil {
		t.Fatalf("createAgentRun: %v", err)
	}
	seedMessage(t, app, run.Id, 1, "system", "sys", nil, "")
	// 10 user rows u0..u9 at seq 2..11.
	for i := 0; i < 10; i++ {
		seedMessage(t, app, run.Id, i+2, "user", "u"+string(rune('0'+i)), nil, "")
	}

	got, err := PreviewConversation(app, run.Id, 3)
	if err != nil {
		t.Fatalf("PreviewConversation: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("preview len = %d, want 3 (capped to maxRows)", len(got))
	}
	// The tail: u7, u8, u9.
	wantTexts := []string{"u7", "u8", "u9"}
	for i, w := range wantTexts {
		if got[i].Text != w {
			t.Fatalf("preview[%d].Text = %q, want %q (most-recent tail)", i, got[i].Text, w)
		}
	}
}

// TestPreviewConversation_DefaultMaxRows: a non-positive maxRows defaults to 20 rather than
// returning everything or nothing.
func TestPreviewConversation_DefaultMaxRows(t *testing.T) {
	app, cfg := newSessionTestEnv(t)
	run, err := createAgentRun(app, "manual", "chat", cfg)
	if err != nil {
		t.Fatalf("createAgentRun: %v", err)
	}
	seedMessage(t, app, run.Id, 1, "system", "sys", nil, "")
	for i := 0; i < 25; i++ { // 25 non-system rows
		seedMessage(t, app, run.Id, i+2, "user", "u", nil, "")
	}
	got, err := PreviewConversation(app, run.Id, 0)
	if err != nil {
		t.Fatalf("PreviewConversation: %v", err)
	}
	if len(got) != 20 {
		t.Fatalf("preview len = %d, want 20 (the default cap for a non-positive maxRows)", len(got))
	}
}

// TestListConversations_EnrichesCountAndActivity: ListConversations populates MsgCount and
// LastActivity — the count of persisted messages and the most-recent message's created time,
// falling back to Started for a run that has no messages.
func TestListConversations_EnrichesCountAndActivity(t *testing.T) {
	app, cfg := newSessionTestEnv(t)

	// A run with messages: count and last-activity come from the messages rows.
	withMsgs, err := createAgentRun(app, "manual", "busy chat", cfg)
	if err != nil {
		t.Fatalf("createAgentRun: %v", err)
	}
	seedMessage(t, app, withMsgs.Id, 1, "system", "sys", nil, "")
	seedMessage(t, app, withMsgs.Id, 2, "user", "q", nil, "")
	seedMessage(t, app, withMsgs.Id, 3, "assistant", "a", nil, "")

	// A run with NO messages: LastActivity must fall back to its started time, MsgCount 0.
	started, _ := types.ParseDateTime(types.NowDateTime().Time().Add(-5 * 60 * 1e9)) // now - 5m
	empty, err := createAgentRun(app, "manual", "quiet chat", cfg)
	if err != nil {
		t.Fatalf("createAgentRun: %v", err)
	}
	empty.Set("started", started)
	if err := app.Save(empty); err != nil {
		t.Fatalf("save empty run: %v", err)
	}

	convos, err := ListConversations(app, 10, "no-such-live-run", false)
	if err != nil {
		t.Fatalf("ListConversations: %v", err)
	}

	byID := map[string]ConversationInfo{}
	for _, c := range convos {
		byID[c.RunID] = c
	}

	busy, ok := byID[withMsgs.Id]
	if !ok {
		t.Fatalf("run with messages missing from the list; got %+v", convos)
	}
	if busy.MsgCount != 3 {
		t.Fatalf("MsgCount = %d, want 3", busy.MsgCount)
	}
	// LastActivity is the newest message's created — it must be at/after the run's started time,
	// and must not be zero.
	if busy.LastActivity.IsZero() {
		t.Fatal("LastActivity is zero for a run with messages")
	}
	if busy.LastActivity.Time().Before(busy.Started.Time()) {
		t.Fatalf("LastActivity %v is before Started %v", busy.LastActivity, busy.Started)
	}

	quiet, ok := byID[empty.Id]
	if !ok {
		t.Fatalf("run without messages missing from the list; got %+v", convos)
	}
	if quiet.MsgCount != 0 {
		t.Fatalf("MsgCount for an empty run = %d, want 0", quiet.MsgCount)
	}
	if !quiet.LastActivity.Equal(quiet.Started) {
		t.Fatalf("LastActivity %v != Started %v for a run with no messages (must fall back)", quiet.LastActivity, quiet.Started)
	}
}

// TestSetConversationArchived_SoftHideAndReveal proves archive is a soft, reversible hide: an
// archived run drops out of the default ListConversations offers but is REVEALED with
// includeArchived (carrying Archived=true), its messages are left intact (no cascade like delete),
// and unarchiving returns it to the default list. A conversation is the user's own history — this
// is not a record-original-first concern.
func TestSetConversationArchived_SoftHideAndReveal(t *testing.T) {
	app, cfg := newSessionTestEnv(t)

	keep, err := createAgentRun(app, "manual", "keep visible", cfg)
	if err != nil {
		t.Fatalf("createAgentRun keep: %v", err)
	}
	arch, err := createAgentRun(app, "manual", "to archive", cfg)
	if err != nil {
		t.Fatalf("createAgentRun arch: %v", err)
	}
	seedMessage(t, app, arch.Id, 1, "user", "a question", nil, "")
	seedMessage(t, app, arch.Id, 2, "assistant", "an answer", nil, "")

	// Archive it.
	if err := SetConversationArchived(app, arch.Id, true); err != nil {
		t.Fatalf("SetConversationArchived(true): %v", err)
	}

	// Default view excludes the archived run.
	def, err := ListConversations(app, 10, "no-live", false)
	if err != nil {
		t.Fatalf("ListConversations default: %v", err)
	}
	if idsContain(def, arch.Id) {
		t.Error("archived run still offered in the default resume list")
	}
	if !idsContain(def, keep.Id) {
		t.Error("non-archived run missing from the default list")
	}

	// Reveal includes it, marked Archived.
	all, err := ListConversations(app, 10, "no-live", true)
	if err != nil {
		t.Fatalf("ListConversations reveal: %v", err)
	}
	var revealed *ConversationInfo
	for i := range all {
		if all[i].RunID == arch.Id {
			revealed = &all[i]
		}
	}
	if revealed == nil {
		t.Fatal("archived run not revealed with includeArchived")
	}
	if !revealed.Archived {
		t.Error("revealed run's Archived flag not set")
	}

	// Messages are untouched (soft, not a delete cascade).
	if n, err := app.CountRecords("messages", dbxRun(arch.Id)); err != nil || n != 2 {
		t.Fatalf("archived run's messages = %d (err %v), want 2 intact (archive must not cascade)", n, err)
	}

	// Unarchive returns it to the default list.
	if err := SetConversationArchived(app, arch.Id, false); err != nil {
		t.Fatalf("SetConversationArchived(false): %v", err)
	}
	back, err := ListConversations(app, 10, "no-live", false)
	if err != nil {
		t.Fatalf("ListConversations after unarchive: %v", err)
	}
	if !idsContain(back, arch.Id) {
		t.Error("unarchived run not back in the default resume list")
	}
}

// TestSetConversationArchived_MissingRun: archiving a nonexistent run surfaces a not-found error.
func TestSetConversationArchived_MissingRun(t *testing.T) {
	app, _ := newSessionTestEnv(t)
	if err := SetConversationArchived(app, "nope-not-a-real-id", true); err == nil {
		t.Fatal("SetConversationArchived on a missing run returned nil, want a not-found error")
	}
}

// idsContain reports whether any listed conversation has the given run id.
func idsContain(convos []ConversationInfo, runID string) bool {
	for _, c := range convos {
		if c.RunID == runID {
			return true
		}
	}
	return false
}

// dbxRun is a small test helper: the messages.run filter expression for CountRecords.
func dbxRun(runID string) dbx.Expression { return dbx.HashExp{"run": runID} }

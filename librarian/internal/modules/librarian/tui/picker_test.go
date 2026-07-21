package tui

import (
	"context"
	"image/color"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/hsb3/desk-standard/librarian/internal/modules/librarian/agent"
)

// fakeProvider is the injected session lifecycle for pure-Update tests: it records what was asked
// (resumedID, closed count) and hands back fakeStreamer sessions whose channels are never fed —
// these tests drive the model synchronously through Update, without a real Session, DB, or LLM.
type fakeProvider struct {
	convos     []agent.ConversationInfo
	transcript []agent.TranscriptEntry
	resumeCh   chan agent.Event
	freshCh    chan agent.Event
	closed     int
	resumedID  string
	listErr    error
	resumeErr  error
	freshErr   error

	resumeSess streamer // the session handed back by the last resume (for identity assertions)
	freshSess  streamer // the session handed back by the last fresh

	listExcluded        string // the excludeRunID the model passed on the last list call
	listIncludeArchived bool   // the includeArchived flag the model passed on the last list call
	listCalls           int    // how many times list was called (a reload after rename/delete calls it)

	// Sessions-manager recording (rename/delete/preview), mirroring the resumeErr/freshErr pattern.
	renamedID    string
	renamedTitle string
	renameErr    error
	deletedID    string
	deleteErr    error
	archivedID   string // last run passed to setArchived
	archivedFlag bool   // last archived value passed to setArchived
	archiveErr   error
	previewedID  string
	previewData  []agent.TranscriptEntry
	previewErr   error
}

// list mimics the store: with includeArchived false it filters out convos whose Archived flag is
// set (the default-hide behavior), else it returns them all — so the picker's archive/reveal round
// trip is exercised against realistic list results.
func (f *fakeProvider) list(limit int, excludeRunID string, includeArchived bool) ([]agent.ConversationInfo, error) {
	f.listExcluded = excludeRunID
	f.listIncludeArchived = includeArchived
	f.listCalls++
	if f.listErr != nil {
		return nil, f.listErr
	}
	if includeArchived {
		return f.convos, nil
	}
	out := make([]agent.ConversationInfo, 0, len(f.convos))
	for _, c := range f.convos {
		if !c.Archived {
			out = append(out, c)
		}
	}
	return out, nil
}

// setArchived records the toggle and mutates the fake's convo set so a subsequent list reflects it,
// modelling the store's soft, reversible archive.
func (f *fakeProvider) setArchived(runID string, archived bool) error {
	if f.archiveErr != nil {
		return f.archiveErr
	}
	f.archivedID = runID
	f.archivedFlag = archived
	for i := range f.convos {
		if f.convos[i].RunID == runID {
			f.convos[i].Archived = archived
		}
	}
	return nil
}

func (f *fakeProvider) rename(runID, title string) error {
	if f.renameErr != nil {
		return f.renameErr
	}
	f.renamedID = runID
	f.renamedTitle = title
	return nil
}

func (f *fakeProvider) delete(runID string) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
	f.deletedID = runID
	return nil
}

func (f *fakeProvider) preview(runID string) ([]agent.TranscriptEntry, error) {
	f.previewedID = runID
	if f.previewErr != nil {
		return nil, f.previewErr
	}
	return f.previewData, nil
}

func (f *fakeProvider) resume(ctx context.Context, runID string) (streamer, []agent.TranscriptEntry, error) {
	if f.resumeErr != nil {
		return nil, nil, f.resumeErr
	}
	f.resumedID = runID
	f.resumeSess = &fakeStreamer{ch: f.resumeCh}
	return f.resumeSess, f.transcript, nil
}

func (f *fakeProvider) fresh(ctx context.Context) (streamer, error) {
	if f.freshErr != nil {
		return nil, f.freshErr
	}
	f.freshSess = &fakeStreamer{ch: f.freshCh}
	return f.freshSess, nil
}

func (f *fakeProvider) closeSession(ctx context.Context, s streamer) error {
	f.closed++
	return nil
}

// openPickerKey drives ctrl+o through Update.
func openPickerKey(m model) model { return send(m, tea.KeyPressMsg{Mod: tea.ModCtrl, Code: 'o'}) }

func TestCtrlO_OpensPicker(t *testing.T) {
	m, _, fp := newTestModelWithProvider(t, &fakeProvider{
		convos: []agent.ConversationInfo{{RunID: "run-1", Title: "a chat", Status: "succeeded"}},
	})
	m = openPickerKey(m)
	if m.picker == nil {
		t.Fatal("ctrl+o did not open the picker while idle")
	}
	// The model must exclude its own live run from the offers — otherwise the picker's newest
	// (default-selected) row is always the current session itself.
	if fp.listExcluded != "live-run" {
		t.Fatalf("list excludeRunID = %q, want the live session's run ID %q", fp.listExcluded, "live-run")
	}
}

func TestCtrlO_NoOpWhileStreaming(t *testing.T) {
	m, _, _ := newTestModelWithProvider(t, &fakeProvider{
		convos: []agent.ConversationInfo{{RunID: "run-1", Title: "a chat"}},
	})
	m = startStreaming(t, m, "q")
	m = openPickerKey(m)
	if m.picker != nil {
		t.Error("ctrl+o opened the picker while a turn was streaming (must be a no-op)")
	}
}

func TestEsc_ClosesPicker_NoTurnCancel(t *testing.T) {
	m, _, _ := newTestModelWithProvider(t, &fakeProvider{
		convos: []agent.ConversationInfo{{RunID: "run-1", Title: "a chat"}},
	})
	m = openPickerKey(m)
	if m.picker == nil {
		t.Fatal("picker not open before esc")
	}
	m = send(m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.picker != nil {
		t.Error("esc did not close the picker")
	}
	if m.cancelling {
		t.Error("esc set cancelling with no turn in flight (picker esc must only dismiss)")
	}
	if m.streaming {
		t.Error("esc while picker open must not start/keep a stream")
	}
}

func TestPicker_SelectResumes(t *testing.T) {
	provider := &fakeProvider{
		convos: []agent.ConversationInfo{{RunID: "run-1", Title: "prior chat", Status: "succeeded"}},
		transcript: []agent.TranscriptEntry{
			{Role: "user", Text: "hello"},
			{Role: "assistant", Text: "hi there"},
		},
	}
	m, fs, _ := newTestModelWithProvider(t, provider)
	origSess := m.sess

	m = openPickerKey(m)
	if m.picker == nil {
		t.Fatal("picker not open")
	}
	m = send(m, tea.KeyPressMsg{Code: tea.KeyEnter})

	if provider.resumedID != "run-1" {
		t.Errorf("resumed run id = %q, want %q", provider.resumedID, "run-1")
	}
	if provider.closed < 1 {
		t.Error("old session was not closed on resume")
	}
	if m.sess == origSess || m.sess != provider.resumeSess {
		t.Error("session was not swapped to the resumed session")
	}
	if m.sess == fs {
		t.Error("session still points at the original fake streamer after resume")
	}
	if m.picker != nil {
		t.Error("picker not cleared after resume")
	}
	if len(m.entries) != 2 {
		t.Fatalf("entries after resume = %d, want 2 (rebuilt from transcript)", len(m.entries))
	}
	if m.entries[0].role != roleUser || m.entries[0].text != "hello" {
		t.Errorf("first rebuilt entry = %+v, want user 'hello'", m.entries[0])
	}
	if m.entries[1].role != roleAssistant || m.entries[1].text != "hi there" {
		t.Errorf("second rebuilt entry = %+v, want assistant 'hi there'", m.entries[1])
	}
	if m.inflightIdx != -1 {
		t.Errorf("inflightIdx after resume = %d, want -1", m.inflightIdx)
	}
}

func TestPicker_SelectEmpty_NoResume(t *testing.T) {
	// An empty conversation list: enter must not resume (selectedRunID == "").
	provider := &fakeProvider{}
	m, _, _ := newTestModelWithProvider(t, provider)
	m = openPickerKey(m)
	if m.picker == nil {
		t.Fatal("picker not open")
	}
	m = send(m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if provider.resumedID != "" {
		t.Errorf("resume attempted on an empty list (id = %q)", provider.resumedID)
	}
	// The overlay stays open (nothing selected); the user can esc out.
	if m.picker == nil {
		t.Error("picker closed on enter with an empty list; expected it to stay open")
	}
}

func TestCtrlN_NewConversation(t *testing.T) {
	provider := &fakeProvider{}
	m, fs, _ := newTestModelWithProvider(t, provider)
	// Seed some transcript so we can prove it is cleared.
	m = startStreaming(t, m, "q")
	m = send(m, eventMsg{ev: agent.Event{Kind: agent.EventFinal, Content: "answer"}})
	m = send(m, turnDoneMsg{})
	if len(m.entries) == 0 {
		t.Fatal("expected entries before ctrl+n")
	}
	closedBefore := provider.closed

	m = send(m, tea.KeyPressMsg{Mod: tea.ModCtrl, Code: 'n'})

	if provider.closed != closedBefore+1 {
		t.Errorf("closeSession calls = %d, want %d (old session closed on new)", provider.closed, closedBefore+1)
	}
	if m.sess == fs || m.sess != provider.freshSess {
		t.Error("session not swapped to the fresh session on ctrl+n")
	}
	if len(m.entries) != 0 {
		t.Errorf("entries not cleared on ctrl+n: %d", len(m.entries))
	}
	if m.inflightIdx != -1 {
		t.Errorf("inflightIdx after ctrl+n = %d, want -1", m.inflightIdx)
	}
}

// TestCtrlN_FreshFails_OldSessionUntouched: open-before-close — when the fresh session cannot be
// built, the old session must NOT have been closed (it stays genuinely live, not finalized out
// from under the user).
func TestCtrlN_FreshFails_OldSessionUntouched(t *testing.T) {
	provider := &fakeProvider{freshErr: context.Canceled}
	m, fs, _ := newTestModelWithProvider(t, provider)

	m = send(m, tea.KeyPressMsg{Mod: tea.ModCtrl, Code: 'n'})

	if provider.closed != 0 {
		t.Errorf("closeSession calls = %d, want 0 (old session must not be closed when fresh fails)", provider.closed)
	}
	if m.sess != fs {
		t.Error("session swapped away from the old (still-live) session despite the fresh failure")
	}
	// Visible feedback: a failed fresh must append an inline error entry, not silently do nothing
	// (a dead-looking ctrl+n). Mirrors the openPicker/resume degraded paths.
	if len(m.entries) == 0 || !m.entries[len(m.entries)-1].isError {
		t.Error("no inline error entry after a failed fresh (silent failure reads as a dead keybinding)")
	}
}

// TestPicker_ResumeFails_OldSessionUntouched: same open-before-close rule on the resume path,
// plus visible feedback (an inline error entry) instead of a silent dismissal.
func TestPicker_ResumeFails_OldSessionUntouched(t *testing.T) {
	provider := &fakeProvider{
		convos:    []agent.ConversationInfo{{RunID: "run-1", Title: "prior chat"}},
		resumeErr: context.Canceled,
	}
	m, fs, _ := newTestModelWithProvider(t, provider)
	m = openPickerKey(m)
	if m.picker == nil {
		t.Fatal("picker not open")
	}

	m = send(m, tea.KeyPressMsg{Code: tea.KeyEnter})

	if provider.closed != 0 {
		t.Errorf("closeSession calls = %d, want 0 (old session must not be closed when resume fails)", provider.closed)
	}
	if m.sess != fs {
		t.Error("session swapped away from the old (still-live) session despite the resume failure")
	}
	if m.picker != nil {
		t.Error("picker still open after a failed resume; expected it dismissed")
	}
	if len(m.entries) == 0 || !m.entries[len(m.entries)-1].isError {
		t.Error("no inline error entry after a failed resume (silent failure)")
	}
}

func TestCtrlN_NoOpWhileStreaming(t *testing.T) {
	provider := &fakeProvider{}
	m, _, _ := newTestModelWithProvider(t, provider)
	m = startStreaming(t, m, "q")
	closedBefore := provider.closed

	m = send(m, tea.KeyPressMsg{Mod: tea.ModCtrl, Code: 'n'})

	if provider.closed != closedBefore {
		t.Error("ctrl+n closed a session while streaming (must be a no-op)")
	}
	if !m.streaming {
		t.Error("ctrl+n disturbed the in-flight turn (streaming flipped off)")
	}
}

func TestEntriesFromTranscript_AllRoles(t *testing.T) {
	ts := []agent.TranscriptEntry{
		{Role: "user", Text: "ask"},
		{Role: "assistant", Text: "reply"},
		{Role: "assistant", Text: "let me look", ToolName: "query"},
		{Role: "tool", Text: `{"ok":true}`, ToolName: "query"},
		{Role: "system", Text: "ignored"},
	}
	got := entriesFromTranscript(ts)

	if len(got) != 4 {
		t.Fatalf("entries = %d, want 4 (system row skipped)", len(got))
	}

	// user
	if got[0].role != roleUser || got[0].text != "ask" || !got[0].finalized {
		t.Errorf("user entry = %+v", got[0])
	}
	// plain assistant
	if got[1].role != roleAssistant || got[1].text != "reply" || len(got[1].steps) != 0 {
		t.Errorf("assistant entry = %+v", got[1])
	}
	// tool-calling assistant: text becomes a step's commentary
	if got[2].role != roleAssistant || len(got[2].steps) != 1 {
		t.Fatalf("tool-calling assistant entry = %+v", got[2])
	}
	if s := got[2].steps[0]; s.tool != "query" || s.commentary != "let me look" || !s.done || s.result != "" {
		t.Errorf("tool-calling step = %+v", s)
	}
	if got[2].text != "" {
		t.Errorf("tool-calling assistant text = %q, want empty (moved to commentary)", got[2].text)
	}
	// tool row: text becomes a step's result
	if got[3].role != roleAssistant || len(got[3].steps) != 1 {
		t.Fatalf("tool entry = %+v", got[3])
	}
	if s := got[3].steps[0]; s.tool != "query" || s.result != `{"ok":true}` || !s.done || s.commentary != "" {
		t.Errorf("tool step = %+v", s)
	}
}

func TestFooter_ResumeHints(t *testing.T) {
	m, _, _ := newTestModelWithProvider(t, &fakeProvider{})
	foot := m.renderFooter()
	if !strings.Contains(foot, "ctrl+o") {
		t.Error("footer missing ctrl+o resume hint")
	}
	if !strings.Contains(foot, "ctrl+n") {
		t.Error("footer missing ctrl+n new hint")
	}
	if !strings.Contains(m.View().Content, "ctrl+o") {
		t.Error("View output missing ctrl+o hint")
	}
}

// twoConvoModel opens the picker over two conversations (run-1 newest, run-2 older) with a preview
// payload, and returns the opened model plus the shared fake provider.
func twoConvoModel(t *testing.T) (model, *fakeProvider) {
	t.Helper()
	fp := &fakeProvider{
		convos: []agent.ConversationInfo{
			{RunID: "run-1", Title: "first chat", Status: "succeeded", MsgCount: 2},
			{RunID: "run-2", Title: "second chat", Status: "succeeded", MsgCount: 4},
		},
		previewData: []agent.TranscriptEntry{{Role: "user", Text: "hi"}, {Role: "assistant", Text: "hello"}},
	}
	m, _, _ := newTestModelWithProvider(t, fp)
	m = openPickerKey(m)
	if m.picker == nil {
		t.Fatal("picker not open")
	}
	return m, fp
}

// key builders for the sessions-surface contextual keys.
func renameKey(m model) model { return send(m, tea.KeyPressMsg{Code: 'r'}) }
func deleteKey(m model) model { return send(m, tea.KeyPressMsg{Code: 'd'}) }
func yesKey(m model) model    { return send(m, tea.KeyPressMsg{Code: 'y'}) }
func noKey(m model) model     { return send(m, tea.KeyPressMsg{Code: 'n'}) }
func filterKey(m model) model { return send(m, tea.KeyPressMsg{Code: '/'}) }
func escKey(m model) model    { return send(m, tea.KeyPressMsg{Code: tea.KeyEscape}) }
func enterKey(m model) model  { return send(m, tea.KeyPressMsg{Code: tea.KeyEnter}) }
func downKey(m model) model   { return send(m, tea.KeyPressMsg{Code: tea.KeyDown}) }
func archiveKey(m model) model {
	return send(m, tea.KeyPressMsg{Code: 'a'})
}
func showArchivedKey(m model) model {
	// A real shift+a arrives as Code 'a' + ModShift with Text "A"; String() (what key.Matches reads)
	// falls back to the "A" text, matching the "A" binding.
	return send(m, tea.KeyPressMsg{Code: 'a', Mod: tea.ModShift, Text: "A"})
}

// TestPicker_PreviewLoadsOnOpenAndMove: the preview pane loads the highlighted run on open and
// re-loads it as the cursor moves — a fast local query per selection.
func TestPicker_PreviewLoadsOnOpenAndMove(t *testing.T) {
	m, fp := twoConvoModel(t)

	// On open, the newest (default-selected) row's preview is loaded.
	if fp.previewedID != "run-1" {
		t.Fatalf("preview loaded for %q on open, want run-1 (the highlighted row)", fp.previewedID)
	}
	if m.picker.previewID != "run-1" || len(m.picker.previewEntries) != 2 {
		t.Fatalf("picker preview state = id %q, %d entries; want run-1 with 2 entries", m.picker.previewID, len(m.picker.previewEntries))
	}

	// Moving the cursor down re-loads the preview for the newly highlighted row.
	m = downKey(m)
	if m.picker.selectedRunID() != "run-2" {
		t.Fatalf("cursor did not move to run-2 (selected %q)", m.picker.selectedRunID())
	}
	if fp.previewedID != "run-2" {
		t.Fatalf("preview not reloaded on move; previewedID = %q, want run-2", fp.previewedID)
	}
}

// TestPicker_ViewRendersSurface: with the overlay open, the composed View renders the sessions
// list title, a row label, and the browse hint line — a smoke guard that the list + preview + hint
// layout composes without panicking and that the surface is on screen.
func TestPicker_ViewRendersSurface(t *testing.T) {
	m, _ := twoConvoModel(t)
	content := m.View().Content
	for _, want := range []string{"sessions", "first chat", "enter resume"} {
		if !strings.Contains(content, want) {
			t.Errorf("picker View missing %q; got:\n%s", want, content)
		}
	}
}

// TestPicker_RenameCommits: r enters rename mode, enter commits the edited title via
// provider.rename and reloads the list back to browse mode.
func TestPicker_RenameCommits(t *testing.T) {
	m, fp := twoConvoModel(t)

	m = renameKey(m)
	if m.picker.mode != pickerRename {
		t.Fatalf("r did not enter rename mode (mode = %d)", m.picker.mode)
	}
	// The input seeds with the current title.
	if got := m.picker.rename.Value(); got != "first chat" {
		t.Fatalf("rename input seeded with %q, want the current title %q", got, "first chat")
	}
	m.picker.rename.SetValue("renamed chat")

	listBefore := fp.listCalls
	m = enterKey(m)

	if fp.renamedID != "run-1" || fp.renamedTitle != "renamed chat" {
		t.Fatalf("rename called with (%q, %q), want (run-1, renamed chat)", fp.renamedID, fp.renamedTitle)
	}
	if fp.listCalls != listBefore+1 {
		t.Errorf("list not re-called after rename (calls %d -> %d)", listBefore, fp.listCalls)
	}
	if m.picker == nil || m.picker.mode != pickerBrowse {
		t.Error("picker did not return to browse mode after a committed rename")
	}
}

// TestPicker_RenameEmptyCancels: committing a blank/whitespace title does not call provider.rename
// (an empty input_summary would hide the run) and returns to browse without touching the row.
func TestPicker_RenameEmptyCancels(t *testing.T) {
	m, fp := twoConvoModel(t)

	m = renameKey(m)
	m.picker.rename.SetValue("   ")
	m = enterKey(m)

	if fp.renamedID != "" {
		t.Errorf("rename called with a blank title (id = %q); it must be rejected", fp.renamedID)
	}
	if m.picker == nil || m.picker.mode != pickerBrowse {
		t.Error("picker did not return to browse mode after a blank-title rename")
	}
}

// TestPicker_RenameEscCancels: esc abandons the rename without calling provider.rename.
func TestPicker_RenameEscCancels(t *testing.T) {
	m, fp := twoConvoModel(t)

	m = renameKey(m)
	m.picker.rename.SetValue("throwaway")
	m = escKey(m)

	if fp.renamedID != "" {
		t.Errorf("esc during rename still committed (id = %q)", fp.renamedID)
	}
	if m.picker == nil {
		t.Fatal("esc during rename closed the whole picker; it must only cancel the rename")
	}
	if m.picker.mode != pickerBrowse {
		t.Errorf("esc did not return to browse mode (mode = %d)", m.picker.mode)
	}
}

// TestPicker_DeleteConfirmDeletes: d opens the confirm gate, y hard-deletes via provider.delete and
// reloads the list.
func TestPicker_DeleteConfirmDeletes(t *testing.T) {
	m, fp := twoConvoModel(t)

	m = deleteKey(m)
	if m.picker.mode != pickerConfirmDelete {
		t.Fatalf("d did not open the delete-confirm gate (mode = %d)", m.picker.mode)
	}
	listBefore := fp.listCalls
	m = yesKey(m)

	if fp.deletedID != "run-1" {
		t.Fatalf("delete called with %q, want run-1", fp.deletedID)
	}
	if fp.listCalls != listBefore+1 {
		t.Errorf("list not re-called after delete (calls %d -> %d)", listBefore, fp.listCalls)
	}
	if m.picker == nil || m.picker.mode != pickerBrowse {
		t.Error("picker did not return to browse mode after a confirmed delete")
	}
}

// TestPicker_DeleteCancelWithN: n backs out of the confirm gate without deleting.
func TestPicker_DeleteCancelWithN(t *testing.T) {
	m, fp := twoConvoModel(t)

	m = deleteKey(m)
	m = noKey(m)

	if fp.deletedID != "" {
		t.Errorf("n still deleted (id = %q); the gate must require y", fp.deletedID)
	}
	if m.picker == nil || m.picker.mode != pickerBrowse {
		t.Error("n did not return to browse mode without deleting")
	}
}

// TestPicker_DeleteCancelWithEsc: esc backs out of the confirm gate without deleting AND without
// closing the whole picker.
func TestPicker_DeleteCancelWithEsc(t *testing.T) {
	m, fp := twoConvoModel(t)

	m = deleteKey(m)
	m = escKey(m)

	if fp.deletedID != "" {
		t.Errorf("esc still deleted (id = %q)", fp.deletedID)
	}
	if m.picker == nil {
		t.Fatal("esc during the delete gate closed the whole picker; it must only cancel the gate")
	}
	if m.picker.mode != pickerBrowse {
		t.Errorf("esc did not return to browse mode (mode = %d)", m.picker.mode)
	}
}

// TestPickerDelegate_ThemedColors guards the sessions-list delegate theming: the
// list rows must follow the resolved theme and the surface's cyan accent, not the bubbles
// DefaultDelegate's hardcoded-dark defaults (a light-gray title invisible on a light terminal and a
// magenta selected row that ignores the app palette). Concrete per-theme colors only — the
// no-runtime-query invariant (ADR 0004) forbids AdaptiveColor here as everywhere else.
func TestPickerDelegate_ThemedColors(t *testing.T) {
	accent := lipgloss.Color("6") // cyan — the surface accent
	cases := []struct {
		theme    string
		wantBody color.Color
	}{
		{themeDark, lipgloss.Color("15")}, // bright white — legible on a dark terminal
		{themeLight, lipgloss.Color("0")}, // black — legible on a light terminal
	}
	for _, tc := range cases {
		d := newStyles(tc.theme).pickerDelegate
		if got := d.NormalTitle.GetForeground(); got != tc.wantBody {
			t.Errorf("%s: normal row foreground = %v, want %v (rows must follow the theme body tone, not the delegate's hardcoded dark)", tc.theme, got, tc.wantBody)
		}
		if got := d.SelectedTitle.GetForeground(); got != accent {
			t.Errorf("%s: selected row foreground = %v, want the cyan accent %v", tc.theme, got, accent)
		}
		if got := d.SelectedTitle.GetBorderLeftForeground(); got != accent {
			t.Errorf("%s: selected row left border = %v, want the cyan accent %v", tc.theme, got, accent)
		}
		if d.NormalTitle.GetForeground() == d.SelectedTitle.GetForeground() {
			t.Errorf("%s: normal and selected rows share a foreground; the selected row must stand out", tc.theme)
		}
	}
}

// TestResumeFirst_OpensPickerWhenConvosExist guards resume-first launch: when
// prior resumable conversations exist, arming resume-first (as Run does) and delivering the first
// sizing WindowSizeMsg opens the sessions overlay at launch. It is one-shot — a later terminal
// resize must never reopen it once dismissed.
func TestResumeFirst_OpensPickerWhenConvosExist(t *testing.T) {
	m, _, fp := newTestModelWithProvider(t, &fakeProvider{
		convos: []agent.ConversationInfo{{RunID: "run-1", Title: "prior chat", Status: "succeeded"}},
	})
	// The pure-Update constructor does NOT arm resume-first, so the picker is closed after the
	// initial WindowSizeMsg even with prior conversations — the default the existing tests rely on.
	if m.picker != nil {
		t.Fatal("picker unexpectedly open before resume-first was armed (default launch must stay fresh)")
	}

	m.enableResumeFirst()
	m = send(m, tea.WindowSizeMsg{Width: 80, Height: 24})
	if m.picker == nil {
		t.Fatal("resume-first did not open the sessions overlay at launch with prior conversations")
	}
	if fp.listExcluded != "live-run" {
		t.Errorf("launch list excludeRunID = %q, want the live run %q (the just-created session must not offer itself)", fp.listExcluded, "live-run")
	}

	// Dismiss, then resize: resume-first is one-shot and must not reopen.
	m = escKey(m)
	if m.picker != nil {
		t.Fatal("esc did not dismiss the launch picker")
	}
	m = send(m, tea.WindowSizeMsg{Width: 100, Height: 30})
	if m.picker != nil {
		t.Error("a later resize reopened the launch picker; resume-first must fire exactly once")
	}
}

// TestResumeFirst_NoConvos_StartsFresh: with no prior resumable conversations, resume-first is a
// no-op — the surface drops straight into the fresh session rather than showing an empty overlay.
func TestResumeFirst_NoConvos_StartsFresh(t *testing.T) {
	m, _, _ := newTestModelWithProvider(t, &fakeProvider{}) // empty conversation list
	m.enableResumeFirst()
	m = send(m, tea.WindowSizeMsg{Width: 80, Height: 24})
	if m.picker != nil {
		t.Error("resume-first opened an empty overlay; with no prior conversations it must start fresh")
	}
}

// TestResumeFirst_ListError_StartsFresh: a failed launch listing must degrade to a fresh session
// (no overlay), not crash or leave dead chrome — the user can retry via ctrl+o, which surfaces the
// error on its own path.
func TestResumeFirst_ListError_StartsFresh(t *testing.T) {
	m, _, _ := newTestModelWithProvider(t, &fakeProvider{listErr: context.DeadlineExceeded})
	m.enableResumeFirst()
	m = send(m, tea.WindowSizeMsg{Width: 80, Height: 24})
	if m.picker != nil {
		t.Error("resume-first opened an overlay despite a list error; it must degrade to a fresh session")
	}
}

// TestPicker_FilterRoutesKeys: "/" starts the built-in fuzzy filter; while filtering, esc cancels
// the FILTER (not the overlay) and outer lifecycle keys are inert (typed into the filter, not acted
// on). This is the SetFilteringEnabled re-enablement the surface asked for.
func TestPicker_FilterRoutesKeys(t *testing.T) {
	m, fp := twoConvoModel(t)

	m = filterKey(m)
	if !m.picker.settingFilter() {
		t.Fatal("\"/\" did not put the list into its filtering state")
	}

	// While filtering, a bare "d" is filter input, not a delete.
	m = deleteKey(m)
	if fp.deletedID != "" {
		t.Errorf("a key while filtering triggered a delete (id = %q); it must route to the filter", fp.deletedID)
	}
	if m.picker == nil {
		t.Fatal("the picker closed while filtering")
	}

	// esc while filtering cancels the FILTER, leaving the overlay open.
	m = escKey(m)
	if m.picker == nil {
		t.Fatal("esc while filtering closed the overlay; it must only cancel the filter")
	}
	if m.picker.settingFilter() {
		t.Error("esc did not cancel the filtering state")
	}
}

// TestPicker_OpensDefaultView: a fresh open lists with archived excluded and the reveal off.
func TestPicker_OpensDefaultView(t *testing.T) {
	m, fp := twoConvoModel(t)
	if fp.listIncludeArchived {
		t.Error("initial open listed with archived included; the default view must hide archived")
	}
	if m.picker.showArchived {
		t.Error("a fresh picker must open with the archived reveal off")
	}
}

// TestPicker_ArchiveHidesFromDefault: `a` soft-archives the highlighted conversation, which then
// drops out of the default (archived-hidden) list on reload; the selection settles on the remaining
// row. Archive is a store toggle, never a delete.
func TestPicker_ArchiveHidesFromDefault(t *testing.T) {
	m, fp := twoConvoModel(t) // run-1 (selected), run-2 — both active

	m = archiveKey(m)

	if fp.archivedID != "run-1" || fp.archivedFlag != true {
		t.Fatalf("archive called with (%q, %v), want (run-1, true)", fp.archivedID, fp.archivedFlag)
	}
	if fp.deletedID != "" {
		t.Errorf("archive triggered a delete (id = %q); archive must be soft, never a delete", fp.deletedID)
	}
	if got := len(m.picker.list.Items()); got != 1 {
		t.Fatalf("after archiving run-1, default list = %d rows, want 1 (archived hidden)", got)
	}
	if m.picker.selectedRunID() != "run-2" {
		t.Errorf("selection did not settle on the remaining run-2 (got %q)", m.picker.selectedRunID())
	}
	if m.picker == nil || m.picker.mode != pickerBrowse {
		t.Error("picker did not return to browse mode after archiving")
	}
}

// TestPicker_ShowArchivedAndUnarchive: the reveal key (`A`) exposes archived conversations (marked
// in the view), and `a` on a revealed archived row unarchives it — the archive/unarchive round-trip.
func TestPicker_ShowArchivedAndUnarchive(t *testing.T) {
	m, fp := twoConvoModel(t)

	// Archive run-1 out of the default view.
	m = archiveKey(m)
	if len(m.picker.list.Items()) != 1 {
		t.Fatalf("run-1 not hidden after archive; list = %d rows", len(m.picker.list.Items()))
	}

	// Reveal archived conversations.
	m = showArchivedKey(m)
	if !m.picker.showArchived {
		t.Fatal("A did not enable the archived reveal")
	}
	if !fp.listIncludeArchived {
		t.Fatal("the reveal must re-list with archived included")
	}
	if got := len(m.picker.list.Items()); got != 2 {
		t.Fatalf("revealed list = %d rows, want 2 (archived shown alongside active)", got)
	}
	if !strings.Contains(m.View().Content, "archived") {
		t.Error("the revealed archived row is not marked 'archived' in the composed view")
	}

	// Select the archived run-1 (newest, index 0) and unarchive it.
	m.picker.list.Select(0)
	if m.picker.selectedRunID() != "run-1" {
		t.Fatalf("expected run-1 selected at index 0, got %q", m.picker.selectedRunID())
	}
	if !m.picker.selectedArchived() {
		t.Fatal("run-1 should read as archived before the unarchive toggle")
	}

	m = archiveKey(m) // toggle: run-1 is archived → unarchive

	if fp.archivedID != "run-1" || fp.archivedFlag != false {
		t.Fatalf("unarchive called with (%q, %v), want (run-1, false)", fp.archivedID, fp.archivedFlag)
	}
	// Still revealed, both rows present, run-1 no longer archived in the store.
	if got := len(m.picker.list.Items()); got != 2 {
		t.Errorf("list = %d rows after unarchive in the reveal view, want 2", got)
	}
	for _, c := range fp.convos {
		if c.RunID == "run-1" && c.Archived {
			t.Error("run-1 still archived in the store after the unarchive toggle")
		}
	}
}

// TestPicker_ArchiveEmpty_NoOp: with no selection (empty list), `a` does not call setArchived and
// leaves the overlay open.
func TestPicker_ArchiveEmpty_NoOp(t *testing.T) {
	m, _, fp := newTestModelWithProvider(t, &fakeProvider{})
	m = openPickerKey(m)
	if m.picker == nil {
		t.Fatal("picker not open")
	}
	m = archiveKey(m)
	if fp.archivedID != "" {
		t.Errorf("archive called on an empty selection (id = %q)", fp.archivedID)
	}
	if m.picker == nil {
		t.Error("archive on an empty list closed the overlay; it must be a no-op")
	}
}

// TestPicker_ArchiveErrorDismisses: a failed setArchived dismisses the overlay with a visible inline
// error, mirroring the rename/delete degraded paths.
func TestPicker_ArchiveErrorDismisses(t *testing.T) {
	fp := &fakeProvider{
		convos:     []agent.ConversationInfo{{RunID: "run-1", Title: "a chat", Status: "succeeded"}},
		archiveErr: context.DeadlineExceeded,
	}
	m, _, _ := newTestModelWithProvider(t, fp)
	m = openPickerKey(m)
	m = archiveKey(m)
	if m.picker != nil {
		t.Error("a failed archive did not dismiss the overlay")
	}
	if len(m.entries) == 0 || !m.entries[len(m.entries)-1].isError {
		t.Error("a failed archive did not append a visible inline error")
	}
}

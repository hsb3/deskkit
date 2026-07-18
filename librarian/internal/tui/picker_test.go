package tui

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/example/pocket-librarian/internal/agent"
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

	listExcluded string // the excludeRunID the model passed on the last list call
}

func (f *fakeProvider) list(limit int, excludeRunID string) ([]agent.ConversationInfo, error) {
	f.listExcluded = excludeRunID
	return f.convos, f.listErr
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
func openPickerKey(m model) model { return send(m, tea.KeyMsg{Type: tea.KeyCtrlO}) }

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
	m = send(m, tea.KeyMsg{Type: tea.KeyEsc})
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
	m = send(m, tea.KeyMsg{Type: tea.KeyEnter})

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
	m = send(m, tea.KeyMsg{Type: tea.KeyEnter})
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

	m = send(m, tea.KeyMsg{Type: tea.KeyCtrlN})

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

func TestCtrlN_NoOpWhileStreaming(t *testing.T) {
	provider := &fakeProvider{}
	m, _, _ := newTestModelWithProvider(t, provider)
	m = startStreaming(t, m, "q")
	closedBefore := provider.closed

	m = send(m, tea.KeyMsg{Type: tea.KeyCtrlN})

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
	if !strings.Contains(m.View(), "ctrl+o") {
		t.Error("View output missing ctrl+o hint")
	}
}

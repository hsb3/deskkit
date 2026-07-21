package tui

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// TestEditorCommand_Resolution covers the env resolution of the compose-in-editor hatch, in the
// production argument order (model.go passes $VISUAL first): $VISUAL is preferred, $EDITOR is the
// fallback, neither yields nil (no editor configured), a value with flags splits into command +
// args, and a whitespace-only value is treated as unset so the fallback wins.
func TestEditorCommand_Resolution(t *testing.T) {
	cases := []struct {
		name           string
		visual, editor string
		want           []string
	}{
		{"visual set", "vim", "", []string{"vim"}},
		{"editor fallback when visual empty", "", "nano", []string{"nano"}},
		{"visual wins over editor", "vim", "nano", []string{"vim"}},
		{"neither set is nil", "", "", nil},
		{"visual with flags splits", "code --wait", "", []string{"code", "--wait"}},
		{"whitespace-only visual falls back", "   ", "emacsclient -nw", []string{"emacsclient", "-nw"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := editorCommand(c.visual, c.editor); !reflect.DeepEqual(got, c.want) {
				t.Errorf("editorCommand(%q, %q) = %v, want %v", c.visual, c.editor, got, c.want)
			}
		})
	}
}

// TestWriteReadDraft_RoundTrip proves a draft written out for the editor reads back byte-for-byte
// (embedded newlines preserved), that the temp file carries the .md suffix (editor highlighting),
// and that removing it — as the finish handler always does — actually deletes it.
func TestWriteReadDraft_RoundTrip(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir()) // contain the temp file in an auto-cleaned dir

	const draft = "first line\n\nthird line after a blank"
	path, err := writeDraft(draft)
	if err != nil {
		t.Fatalf("writeDraft: %v", err)
	}
	if filepath.Ext(path) != ".md" {
		t.Errorf("temp file %q lacks the .md suffix", path)
	}

	got, err := readDraft(path)
	if err != nil {
		t.Fatalf("readDraft: %v", err)
	}
	if got != draft {
		t.Errorf("round-trip = %q, want %q", got, draft)
	}

	if err := os.Remove(path); err != nil {
		t.Fatalf("removing the draft file: %v", err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("draft file still present after removal (stat err = %v)", err)
	}
}

// TestReadDraft_TrimsTrailingNewline pins the trailing-newline trim: the newline(s) an editor
// appends on save must not linger in the textarea, while newlines WITHIN the draft are preserved.
func TestReadDraft_TrimsTrailingNewline(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())

	path, err := writeDraft("body text\ninner line\n\n")
	if err != nil {
		t.Fatalf("writeDraft: %v", err)
	}
	defer os.Remove(path)

	got, err := readDraft(path)
	if err != nil {
		t.Fatalf("readDraft: %v", err)
	}
	if want := "body text\ninner line"; got != want {
		t.Errorf("readDraft trimmed = %q, want %q", got, want)
	}
}

// TestReadDraft_MissingFile: reading a path that does not exist surfaces the error rather than
// panicking (the finish handler turns it into a toast).
func TestReadDraft_MissingFile(t *testing.T) {
	if _, err := readDraft(filepath.Join(t.TempDir(), "does-not-exist.md")); err == nil {
		t.Error("readDraft of a missing file returned no error")
	}
}

// TestEditExternal_NoOpWhileStreaming: ctrl+e must not launch an editor mid-turn. $EDITOR is set so
// that, absent the streaming guard, the keypress WOULD return an ExecProcess command — the guard is
// proven by the command being nil and the streaming state untouched.
func TestEditExternal_NoOpWhileStreaming(t *testing.T) {
	t.Setenv("EDITOR", "vim")
	m, _ := newTestModel(t)
	m = startStreaming(t, m, "a question")

	next, cmd := m.Update(tea.KeyPressMsg{Mod: tea.ModCtrl, Code: 'e'})
	nm := next.(model)
	if cmd != nil {
		t.Error("ctrl+e returned a command while streaming; it must be a no-op mid-turn")
	}
	if !nm.streaming {
		t.Error("ctrl+e disturbed the streaming state")
	}
	if nm.toast != "" {
		t.Errorf("ctrl+e set a toast while streaming: %q", nm.toast)
	}
}

// TestEditExternal_MissingEditorShowsToast: with neither $EDITOR nor $VISUAL set, ctrl+e reports a
// footer toast (and schedules its expiry) instead of crashing or doing nothing visible.
func TestEditExternal_MissingEditorShowsToast(t *testing.T) {
	t.Setenv("EDITOR", "")
	t.Setenv("VISUAL", "")
	m, _ := newTestModel(t)

	next, cmd := m.Update(tea.KeyPressMsg{Mod: tea.ModCtrl, Code: 'e'})
	nm := next.(model)
	if nm.toast != "set $EDITOR to compose externally" {
		t.Errorf("toast = %q, want the missing-editor hint", nm.toast)
	}
	if cmd == nil {
		t.Error("no expiry command scheduled for the missing-editor toast")
	}
}

// TestEditExternal_ConfiguredEditorLaunches: with $EDITOR set and no turn in flight, ctrl+e returns
// a non-nil command (the tea.ExecProcess that hands the terminal to the editor). TMPDIR is pinned
// to an auto-cleaned dir so the draft written for the — never actually executed — command does not
// leak.
func TestEditExternal_ConfiguredEditorLaunches(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	t.Setenv("EDITOR", "true") // a real, harmless command; the returned Cmd is never run here
	m, _ := newTestModel(t)
	m.ta.SetValue("draft to hand to the editor")

	_, cmd := m.Update(tea.KeyPressMsg{Mod: tea.ModCtrl, Code: 'e'})
	if cmd == nil {
		t.Error("ctrl+e with $EDITOR set returned no command; it must launch the editor")
	}
}

// TestEditorFinished_LoadsDraftAndCleansUp: the finish msg reads the composed text back into the
// textarea (trailing newline trimmed) and removes the temp file, never auto-sending.
func TestEditorFinished_LoadsDraftAndCleansUp(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	m, fs := newTestModel(t)

	path, err := writeDraft("composed answer\n")
	if err != nil {
		t.Fatalf("writeDraft: %v", err)
	}

	next, _ := m.Update(editorFinishedMsg{path: path})
	nm := next.(model)
	if got := nm.ta.Value(); got != "composed answer" {
		t.Errorf("textarea after editor = %q, want %q (trailing newline trimmed)", got, "composed answer")
	}
	if _, statErr := os.Stat(path); !errors.Is(statErr, os.ErrNotExist) {
		t.Errorf("temp draft not removed after the finish msg (stat err = %v)", statErr)
	}
	if fs.calls != 0 {
		t.Errorf("editor finish auto-sent a turn (StreamTurn calls = %d, want 0)", fs.calls)
	}
}

// TestEditorFinished_ErrorTostsAndCleansUp: an editor that exited with an error surfaces a toast and
// still removes the temp file, leaving the textarea untouched.
func TestEditorFinished_ErrorTostsAndCleansUp(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	m, _ := newTestModel(t)
	m.ta.SetValue("original draft")

	path, err := writeDraft("ignored on error")
	if err != nil {
		t.Fatalf("writeDraft: %v", err)
	}

	next, cmd := m.Update(editorFinishedMsg{path: path, err: errors.New("boom")})
	nm := next.(model)
	if nm.toast == "" {
		t.Error("an editor error set no toast")
	}
	if cmd == nil {
		t.Error("no expiry command scheduled for the editor-error toast")
	}
	if got := nm.ta.Value(); got != "original draft" {
		t.Errorf("textarea changed on editor error, value = %q", got)
	}
	if _, statErr := os.Stat(path); !errors.Is(statErr, os.ErrNotExist) {
		t.Errorf("temp draft not removed after an editor error (stat err = %v)", statErr)
	}
}

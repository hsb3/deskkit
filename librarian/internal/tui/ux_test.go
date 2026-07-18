package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/example/pocket-librarian/internal/agent"
)

// TestLastAssistantMarkdown_Selection: copy targets the most recent FINALIZED assistant answer's
// raw text, skipping user turns, an in-flight (unfinalized) turn, and steps-only assistant turns.
func TestLastAssistantMarkdown_Selection(t *testing.T) {
	t.Run("none when empty", func(t *testing.T) {
		if got, ok := lastAssistantMarkdown(nil); ok || got != "" {
			t.Errorf("empty = (%q, %v), want (\"\", false)", got, ok)
		}
	})
	t.Run("none when only a user turn", func(t *testing.T) {
		entries := []entry{{role: roleUser, text: "hi", finalized: true}}
		if _, ok := lastAssistantMarkdown(entries); ok {
			t.Error("user-only transcript reported a copy target")
		}
	})
	t.Run("picks the last finalized assistant answer", func(t *testing.T) {
		entries := []entry{
			{role: roleUser, text: "q1", finalized: true},
			{role: roleAssistant, text: "# first answer", finalized: true},
			{role: roleUser, text: "q2", finalized: true},
			{role: roleAssistant, text: "**second answer**", finalized: true},
		}
		got, ok := lastAssistantMarkdown(entries)
		if !ok || got != "**second answer**" {
			t.Errorf("= (%q, %v), want (**second answer**, true)", got, ok)
		}
	})
	t.Run("skips an in-flight and a steps-only assistant turn", func(t *testing.T) {
		entries := []entry{
			{role: roleAssistant, text: "done answer", finalized: true},
			{role: roleAssistant, steps: []step{{tool: "query", done: true}}, finalized: true}, // no text
			{role: roleAssistant, text: "streaming so far", finalized: false},                  // in-flight
		}
		got, ok := lastAssistantMarkdown(entries)
		if !ok || got != "done answer" {
			t.Errorf("= (%q, %v), want (done answer, true)", got, ok)
		}
	})
}

// TestReducedMotion_FromEnv: any non-empty NO_COLOR enables reduced motion; empty does not.
func TestReducedMotion_FromEnv(t *testing.T) {
	cases := []struct {
		env  string
		want bool
	}{
		{"", false},
		{"1", true},
		{"true", true},
		{"0", true}, // NO_COLOR semantics: presence (any value) is what matters, not truthiness
	}
	for _, tc := range cases {
		if got := reducedMotion(tc.env); got != tc.want {
			t.Errorf("reducedMotion(%q) = %v, want %v", tc.env, got, tc.want)
		}
	}
}

// TestReducedMotion_FooterText: with NO_COLOR set the streaming footer shows static "working…"
// text instead of the animated spinner's "streaming" segment.
func TestReducedMotion_FooterText(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	m, _ := newTestModel(t)
	if !m.reduced {
		t.Fatal("reduced-motion flag not set from NO_COLOR")
	}
	m = startStreaming(t, m, "q")
	if got := m.renderFooter(); !strings.Contains(got, "working…") {
		t.Errorf("reduced-motion footer = %q, want it to contain \"working…\"", got)
	}
}

// TestGutterStyles_DifferByRole: each theme defines distinct left-gutter styles for the two roles
// (accent for the user, faint for the librarian) so turns are visually separated.
func TestGutterStyles_DifferByRole(t *testing.T) {
	for _, theme := range []string{themeDark, themeLight} {
		st := newStyles(theme)
		u := st.userGutter.GetForeground()
		a := st.assistantGutter.GetForeground()
		if u == a {
			t.Errorf("%s: user and assistant gutter share foreground %v; roles must be distinguishable", theme, u)
		}
	}
}

// TestGutterRendered_InTranscript: both gutter glyphs appear in the rendered transcript so the
// visual turn separation is actually emitted.
func TestGutterRendered_InTranscript(t *testing.T) {
	m, _ := newTestModel(t)
	m = startStreaming(t, m, "hello")
	m = send(m, eventMsg{ev: agent.Event{Kind: agent.EventFinal, Content: "hi there"}})
	m = send(m, turnDoneMsg{})
	out := m.renderTranscript()
	if !strings.Contains(out, "▌") {
		t.Error("user gutter glyph ▌ not in transcript")
	}
	if !strings.Contains(out, "│") {
		t.Error("assistant gutter glyph │ not in transcript")
	}
}

// TestPerTurnFooter_AfterFinalize: a finished assistant turn shows a per-turn footer carrying the
// model id; a still-streaming turn shows none.
func TestPerTurnFooter_AfterFinalize(t *testing.T) {
	m, _ := newTestModel(t) // model id "test-model" (newTestModelWithProvider)
	m = startStreaming(t, m, "q")
	m = send(m, eventMsg{ev: agent.Event{Kind: agent.EventToken, Token: "answer"}})
	if strings.Contains(m.renderTranscript(), "test-model · ") {
		t.Error("per-turn footer shown while the turn is still streaming")
	}
	m = send(m, eventMsg{ev: agent.Event{Kind: agent.EventFinal, Content: "answer"}})
	m = send(m, turnDoneMsg{})
	if !strings.Contains(m.renderTranscript(), "test-model · ") {
		t.Error("per-turn footer with model id not shown after finalize")
	}
}

// TestHelpToggle_CtrlG: ctrl+g flips the help component into its full (grouped) view and back,
// and never quits. Bare "?" must NOT toggle help (the textarea types it).
func TestHelpToggle_CtrlG(t *testing.T) {
	m, _ := newTestModel(t)
	if m.hlp.ShowAll {
		t.Fatal("help started expanded")
	}
	m = send(m, tea.KeyMsg{Type: tea.KeyCtrlG})
	if !m.hlp.ShowAll {
		t.Error("ctrl+g did not expand help")
	}
	m = send(m, tea.KeyMsg{Type: tea.KeyCtrlG})
	if m.hlp.ShowAll {
		t.Error("second ctrl+g did not collapse help")
	}
	// "?" is a literal keypress: it must reach the textarea, not toggle help.
	m2, _ := newTestModel(t)
	m2 = send(m2, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	if m2.hlp.ShowAll {
		t.Error("bare ? toggled help; it must be typed into the textarea")
	}
	if m2.ta.Value() != "?" {
		t.Errorf("bare ? not typed into textarea, value = %q", m2.ta.Value())
	}
}

// TestEsc_KeepsPartialStreamedText pins the "cancel keeps partial work" contract end-to-end from
// the esc keypress: streamed tokens already in the bubble survive a canceled terminal event, and
// the turn carries the (interrupted) badge rather than being dropped or shown as a red error.
func TestEsc_KeepsPartialStreamedText(t *testing.T) {
	m, fs := newTestModel(t)
	m = startStreaming(t, m, "q")
	m = send(m, eventMsg{ev: agent.Event{Kind: agent.EventToken, Token: "partial so "}})
	m = send(m, eventMsg{ev: agent.Event{Kind: agent.EventToken, Token: "far"}})

	// esc cancels the in-flight turn.
	m = send(m, tea.KeyMsg{Type: tea.KeyEsc})
	if fs.ctx.Err() == nil {
		t.Fatal("esc did not cancel the turn context")
	}
	// The engine drains to a canceled terminal event (no Partial: the tokens are already in the
	// bubble), then the channel closes.
	m = send(m, eventMsg{ev: agent.Event{Kind: agent.EventError, Canceled: true}})
	m = send(m, turnDoneMsg{})

	e := m.entries[m.inflightIdx]
	if e.text != "partial so far" {
		t.Errorf("partial streamed text = %q, want %q", e.text, "partial so far")
	}
	if !e.interrupted || e.isError {
		t.Errorf("canceled turn: interrupted=%v isError=%v, want interrupted=true isError=false", e.interrupted, e.isError)
	}
	if !strings.Contains(m.renderTranscript(), "(interrupted)") {
		t.Error("(interrupted) badge not rendered after esc-cancel")
	}
	// The partial body renders (glamour styles words separately, so assert a word, not the phrase;
	// the contiguous partial is pinned on e.text above).
	if !strings.Contains(m.renderTranscript(), "partial") {
		t.Error("partial body not rendered in the transcript after cancel")
	}
}

// TestCopyToast_NothingToCopy: ctrl+y with no assistant answer yet sets the "nothing to copy"
// toast and schedules its expiry.
func TestCopyToast_NothingToCopy(t *testing.T) {
	m, _ := newTestModel(t)
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlY})
	nm := next.(model)
	if nm.toast != "nothing to copy" {
		t.Errorf("toast = %q, want %q", nm.toast, "nothing to copy")
	}
	if cmd == nil {
		t.Error("no expiry command scheduled for the toast")
	}
	// A stale expiry for a superseded toast must not clear a newer one.
	nm.toast = "copied"
	nm.toastSeq = 5
	cleared := send(nm, toastExpireMsg{seq: 1})
	if cleared.toast != "copied" {
		t.Error("stale toast expiry cleared a newer toast")
	}
	matched := send(nm, toastExpireMsg{seq: 5})
	if matched.toast != "" {
		t.Error("matching toast expiry did not clear the toast")
	}
}

// TestHistoryRecall_UpAtFirstLine: Up at the textarea's first line recalls the previous prompt;
// Down past it restores the stashed draft.
func TestHistoryRecall_UpAtFirstLine(t *testing.T) {
	m, _ := newTestModel(t)
	m = startStreaming(t, m, "first prompt")
	m = send(m, eventMsg{ev: agent.Event{Kind: agent.EventFinal, Content: "ok"}})
	m = send(m, turnDoneMsg{})

	// Type a fresh draft, then Up to recall the sent prompt.
	m.ta.SetValue("half typed")
	m = send(m, tea.KeyMsg{Type: tea.KeyUp})
	if got := m.ta.Value(); got != "first prompt" {
		t.Errorf("after Up, textarea = %q, want %q", got, "first prompt")
	}
	// Down past the newest restores the stashed draft.
	m = send(m, tea.KeyMsg{Type: tea.KeyDown})
	if got := m.ta.Value(); got != "half typed" {
		t.Errorf("after Down, textarea = %q, want restored draft %q", got, "half typed")
	}
}

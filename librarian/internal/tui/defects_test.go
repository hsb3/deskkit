package tui

import (
	"testing"

	"github.com/charmbracelet/glamour/styles"
	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"
)

// TestGlamourStyle_NoRuntimeQuery guards the Defect-1 fix: the markdown style must resolve
// statically, never via glamour's WithAutoStyle (which queries the terminal for its background —
// an OSC 11 + DSR probe that, fired after the Bubble Tea program starts, races the input reader
// and leaks the terminal's escape-sequence response into the textarea as typed input). A default
// of the fixed "dark" style, a GLAMOUR_STYLE override, and an explicit rejection of "auto" (which
// would reintroduce the query) together assert the query can never happen.
func TestGlamourStyle_NoRuntimeQuery(t *testing.T) {
	t.Run("default is static dark, not auto", func(t *testing.T) {
		t.Setenv("GLAMOUR_STYLE", "")
		if got := glamourStyle(); got != styles.DarkStyle {
			t.Errorf("default style = %q, want %q (never the querying auto style)", got, styles.DarkStyle)
		}
		if glamourStyle() == styles.AutoStyle {
			t.Fatal("default resolved to the auto style, which triggers a terminal query")
		}
	})
	t.Run("GLAMOUR_STYLE overrides", func(t *testing.T) {
		t.Setenv("GLAMOUR_STYLE", "light")
		if got := glamourStyle(); got != "light" {
			t.Errorf("style = %q, want light (GLAMOUR_STYLE must win)", got)
		}
	})
	t.Run("auto is rejected back to dark", func(t *testing.T) {
		t.Setenv("GLAMOUR_STYLE", styles.AutoStyle)
		if got := glamourStyle(); got != styles.DarkStyle {
			t.Errorf("style = %q, want %q (an explicit auto must not reintroduce the query)", got, styles.DarkStyle)
		}
	})
}

// TestCtrlT_TogglesNeverQuits guards the Defect-3 fix. In the field ctrl+t exited the app; the
// root cause was Defect-1 input corruption (a real ctrl+t is byte 0x14 → KeyCtrlT → "ctrl+t",
// which this exact KeyMsg reproduces — verified against bubbletea's key table). This asserts the
// binding does exactly one thing — toggle the steps view — and NEVER quits: it emits no command
// (a non-nil command here would be the regression toward tea.Quit) and sets no quit flag.
func TestCtrlT_TogglesNeverQuits(t *testing.T) {
	m, _ := newTestModel(t)

	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlT})
	nm := next.(model)
	if !nm.showSteps {
		t.Error("ctrl+t did not toggle the steps view on")
	}
	if cmd != nil {
		t.Error("ctrl+t emitted a command; it must be a no-op (regression guard against quitting)")
	}
	if nm.quitting {
		t.Error("ctrl+t set the quitting flag — it must never quit the app")
	}

	next2, _ := nm.Update(tea.KeyMsg{Type: tea.KeyCtrlT})
	if next2.(model).showSteps {
		t.Error("a second ctrl+t did not toggle the steps view back off")
	}
}

// TestAssistantStyle_HighContrast guards the Defect-2 fix: the streaming answer body must carry
// an explicit bright foreground so it never renders at the terminal's (possibly low-contrast)
// default, and it must be visibly distinct from the faint step-line style.
func TestAssistantStyle_HighContrast(t *testing.T) {
	st := newStyles()
	if got := st.assistant.GetForeground(); got != lipgloss.Color("15") {
		t.Errorf("assistant foreground = %v, want bright white (15)", got)
	}
	if st.assistant.GetForeground() == st.step.GetForeground() {
		t.Error("assistant body and faint step lines share a foreground; answer must be brighter")
	}
}

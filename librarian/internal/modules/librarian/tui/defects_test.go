package tui

import (
	"context"
	"image/color"
	"testing"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2/styles"
	"charm.land/lipgloss/v2"

	"github.com/example/pocket-librarian/internal/core/config"
)

// TestGlamourStyle_NoRuntimeQuery guards the Defect-1 fix: the markdown style must resolve
// statically, never via glamour's WithAutoStyle (which queries the terminal for its background —
// an OSC 11 + DSR probe that, fired after the Bubble Tea program starts, races the input reader
// and leaks the terminal's escape-sequence response into the textarea as typed input). A default
// of the fixed "dark" style, a GLAMOUR_STYLE override, and an explicit rejection of "auto" (which
// would reintroduce the query) together assert the query can never happen.
func TestGlamourStyle_NoRuntimeQuery(t *testing.T) {
	t.Run("resolved theme picks a static style, never auto", func(t *testing.T) {
		t.Setenv("GLAMOUR_STYLE", "")
		if got := glamourStyle(themeDark); got != styles.DarkStyle {
			t.Errorf("dark theme style = %q, want %q (never the querying auto style)", got, styles.DarkStyle)
		}
		if got := glamourStyle(themeLight); got != styles.LightStyle {
			t.Errorf("light theme style = %q, want %q (glamour's static light config)", got, styles.LightStyle)
		}
		if glamourStyle(themeDark) == themeAuto || glamourStyle(themeLight) == themeAuto {
			t.Fatal("a resolved theme produced the auto style, which triggers a terminal query")
		}
	})
	t.Run("GLAMOUR_STYLE overrides the theme", func(t *testing.T) {
		t.Setenv("GLAMOUR_STYLE", "light")
		if got := glamourStyle(themeDark); got != "light" {
			t.Errorf("style = %q, want light (GLAMOUR_STYLE must win over the theme)", got)
		}
	})
	// glamour v2 removed both WithAutoStyle and the "auto" style key itself (there is no such
	// entry in styles.DefaultStyles any more), so the guard this pins is now entirely OUR OWN
	// invariant (themeAuto, shared with theme.go) rather than a glamour-exported sentinel.
	t.Run("auto is rejected back to the theme default", func(t *testing.T) {
		t.Setenv("GLAMOUR_STYLE", themeAuto)
		if got := glamourStyle(themeDark); got != styles.DarkStyle {
			t.Errorf("style = %q, want %q (an explicit auto must not reintroduce the query)", got, styles.DarkStyle)
		}
	})
}

// TestCtrlT_TogglesNeverQuits guards the Defect-3 fix. In the field ctrl+t exited the app; the
// root cause was Defect-1 input corruption (a real ctrl+t is byte 0x14, which bubbletea v2 decodes
// as Mod: ModCtrl, Code: 't' → Keystroke "ctrl+t" — this synthesized tea.KeyPressMsg reproduces
// that decode, verified against ultraviolet's key table). This asserts the binding does exactly one
// thing — toggle the steps view — and NEVER quits: it emits no command (a non-nil command here
// would be the regression toward tea.Quit) and sets no quit flag.
func TestCtrlT_TogglesNeverQuits(t *testing.T) {
	m, _ := newTestModel(t)

	next, cmd := m.Update(tea.KeyPressMsg{Mod: tea.ModCtrl, Code: 't'})
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

	next2, _ := nm.Update(tea.KeyPressMsg{Mod: tea.ModCtrl, Code: 't'})
	if next2.(model).showSteps {
		t.Error("a second ctrl+t did not toggle the steps view back off")
	}
}

// TestAssistantStyle_HighContrast guards the Defect-2 fix across both static themes: the
// streaming answer body must carry an explicit high-contrast foreground for the resolved
// background — bright white on dark, black on light — so it never renders at the terminal's
// (possibly invisible) default, and it must stay visibly distinct from the faint step-line style.
func TestAssistantStyle_HighContrast(t *testing.T) {
	cases := []struct {
		theme string
		want  color.Color
	}{
		{themeDark, lipgloss.Color("15")}, // bright white on a dark background
		{themeLight, lipgloss.Color("0")}, // black on a light background
	}
	for _, tc := range cases {
		st := newStyles(tc.theme)
		if got := st.assistant.GetForeground(); got != tc.want {
			t.Errorf("%s: assistant foreground = %v, want %v", tc.theme, got, tc.want)
		}
		if st.assistant.GetForeground() == st.step.GetForeground() {
			t.Errorf("%s: assistant body and faint step lines share a foreground; answer must stand out", tc.theme)
		}
	}
}

// TestView_SetsAltScreen guards the v2 migration invariant: bubbletea v2 dropped
// tea.WithAltScreen (a ProgramOption) in favor of a per-frame tea.View.AltScreen field, so every
// return path of Model.View must set it explicitly — a path that forgets it silently drops the
// surface out of the alternate screen mid-run. Both the pre-ready placeholder path (before the
// first WindowSizeMsg) and the fully composed surface are asserted. The picker-open branch needs
// no separate case: it only swaps the body string, and every composed frame passes through the
// same unconditional AltScreen set.
func TestView_SetsAltScreen(t *testing.T) {
	notReady := newModel(context.Background(), &fakeStreamer{}, &fakeProvider{}, &config.Config{}, themeDark)
	if v := notReady.View(); !v.AltScreen {
		t.Error("the not-ready placeholder View() does not set AltScreen")
	}

	m, _, _ := newTestModelWithProvider(t, &fakeProvider{})
	if v := m.View(); !v.AltScreen {
		t.Error("the composed (ready) View() does not set AltScreen")
	}
}

// TestAltEnterBinding_MatchesV2KeyPressMsg guards the key-string-delta migration class the brief
// called out: v2's Key.Keystroke() must still render Alt+Enter as "alt+enter" (ctrl/alt/shift/…
// are always prefixed in that fixed order, then the special-key name from the keyTypeString
// table) for the newline binding — and the textarea's own rebound InsertNewline key — to keep
// matching post-migration. A synthesized tea.KeyPressMsg with Mod: ModAlt, Code: KeyEnter stands
// in for a real Alt+Enter terminal keypress (Text is empty for non-printable combos, so String()
// falls back to Keystroke(), exactly as it does for a live keypress).
func TestAltEnterBinding_MatchesV2KeyPressMsg(t *testing.T) {
	msg := tea.KeyPressMsg{Mod: tea.ModAlt, Code: tea.KeyEnter}
	if got := msg.String(); got != "alt+enter" {
		t.Fatalf("alt+enter keypress String() = %q, want %q", got, "alt+enter")
	}
	km := defaultKeymap()
	if !key.Matches(msg, km.newline) {
		t.Error("synthesized alt+enter KeyPressMsg does not match the newline binding")
	}
}

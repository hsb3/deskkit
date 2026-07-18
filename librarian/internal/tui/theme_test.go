package tui

import "testing"

// TestResolveTheme_Precedence pins the resolution order flag > env > auto, and the auto path's
// fall-through to the injected detector. It uses a fake detector so it exercises pure logic with
// no terminal query — the real detectBackground is the only place that touches the terminal, and
// it only runs in the pre-program safe window.
func TestResolveTheme_Precedence(t *testing.T) {
	// A detector that would fail the test if the real terminal probe were reached instead of the
	// injected one; it stands in for the auto-detected result.
	detectLight := func() string { return themeLight }
	detectDark := func() string { return themeDark }

	cases := []struct {
		name   string
		flag   string
		env    string
		detect func() string
		want   string
	}{
		{"flag light wins over dark env", themeLight, themeDark, detectDark, themeLight},
		{"flag dark wins over light env", themeDark, themeLight, detectLight, themeDark},
		{"explicit flag skips detection", themeLight, "", detectDark, themeLight},
		{"env used when flag unset", "", themeLight, detectDark, themeLight},
		{"env wins over auto default", "", themeDark, detectLight, themeDark},
		{"auto flag defers to detection", themeAuto, "", detectLight, themeLight},
		{"auto flag over env still detects", themeAuto, themeDark, detectLight, themeLight},
		{"no flag no env falls to detection", "", "", detectLight, themeLight},
		{"unrecognized choice falls to detection", "solarized", "", detectDark, themeDark},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := resolveTheme(tc.flag, tc.env, tc.detect); got != tc.want {
				t.Errorf("resolveTheme(%q, %q) = %q, want %q", tc.flag, tc.env, got, tc.want)
			}
		})
	}
}

// TestResolveTheme_AutoFallsBackToDark documents that the auto path resolves to a concrete theme
// and, when the detector reports no dark background is determinable, the historical dark default
// is preserved. detectBackground returns themeDark whenever termenv cannot determine a light
// background, so an indeterminate terminal keeps today's behavior.
func TestResolveTheme_AutoFallsBackToDark(t *testing.T) {
	indeterminate := func() string { return themeDark } // termenv's fallback when it cannot tell
	if got := resolveTheme(themeAuto, "", indeterminate); got != themeDark {
		t.Errorf("auto with indeterminate background = %q, want %q (historical dark default)", got, themeDark)
	}
	// The resolver must never surface "auto" to the model — callers rely on a concrete value.
	if got := resolveTheme(themeAuto, "", indeterminate); got == themeAuto {
		t.Error("resolveTheme returned \"auto\"; it must always resolve to a concrete theme")
	}
}

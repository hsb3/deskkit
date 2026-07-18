// Theme resolution for the chat surface. The chat TUI uses STATIC per-theme palettes, chosen
// once at startup — there is no adaptive/runtime color resolution. This is deliberate: any
// terminal-background query (glamour's WithAutoStyle, lipgloss.AdaptiveColor) fired after the
// Bubble Tea program starts races the program's stdin reader and leaks the terminal's
// escape-sequence response into the textarea as typed input. Resolving the theme to a concrete
// "light"/"dark" value BEFORE tea.NewProgram runs (the only safe window for terminal I/O) keeps
// the whole render path query-free.
package tui

import (
	"os"

	"github.com/muesli/termenv"
)

// Theme values are concrete: after resolution the model never carries "auto".
const (
	themeLight = "light"
	themeDark  = "dark"
	themeAuto  = "auto"
)

// envTheme is the environment override consulted when the --theme flag is unset.
const envTheme = "LIBRARIAN_THEME"

// ResolveTheme picks the concrete palette to use, applying precedence
// flag > LIBRARIAN_THEME env > auto-detect, and resolving "auto" via a single pre-program
// background probe. It always returns themeLight or themeDark, never "auto". Call it in the
// safe window BEFORE tea.NewProgram — never at render time.
func ResolveTheme(flag string) string {
	return resolveTheme(flag, os.Getenv(envTheme), detectBackground)
}

// resolveTheme is the pure precedence logic, with background detection injected so it can be
// unit-tested without a terminal. An empty flag defers to env; an empty env defers to auto; an
// unrecognized choice is treated as auto (detect rather than guess wrong).
func resolveTheme(flag, env string, detect func() string) string {
	choice := flag
	if choice == "" {
		choice = env
	}
	if choice == "" {
		choice = themeAuto
	}
	switch choice {
	case themeLight:
		return themeLight
	case themeDark:
		return themeDark
	default: // themeAuto or anything unrecognized
		return detect()
	}
}

// detectBackground reads the terminal background ONCE via termenv (a pre-program query, safe
// before the Bubble Tea stdin reader starts). termenv falls back to a dark background when it
// cannot determine one (dumb terminal, non-TTY, probe error), so this returns themeDark in the
// indeterminate case — preserving the historical dark default.
func detectBackground() string {
	if termenv.HasDarkBackground() {
		return themeDark
	}
	return themeLight
}

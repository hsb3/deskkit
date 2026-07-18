// Lipgloss styles for the chat surface: the header (desk · provider/model), the footer
// (keybind hints + live state), and the transcript roles (user, assistant, steps, errors,
// interrupted badge). Kept in one place so the palette is auditable and the View code reads as
// composition, not styling.
//
// The palette is STATIC per theme, not adaptive. newStyles takes the theme resolved once at
// startup (see theme.go) and returns concrete lipgloss.Color values for that theme — never
// lipgloss.AdaptiveColor, which would query the terminal background at render time and corrupt
// the input (the no-runtime-query invariant, ADR 0004). Semantic colors (cyan/red/green) are
// shared across themes; the answer body and muted/faint chrome carry per-theme tones so they
// keep real contrast on both light and dark backgrounds.
package tui

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/lipgloss"
)

// styleSet bundles the styles the View composes. It is built once (newStyles) and carried on
// the model; nothing here is desk- or identity-specific.
type styleSet struct {
	header          lipgloss.Style
	headerAccent    lipgloss.Style
	footer          lipgloss.Style
	footerState     lipgloss.Style
	toast           lipgloss.Style // transient copy/status toast in the footer
	user            lipgloss.Style
	assistant       lipgloss.Style // answer body while streaming (before glamour finalizes it)
	roleLabel       lipgloss.Style
	userGutter      lipgloss.Style // thick accent left border ▌ for a user turn
	assistantGutter lipgloss.Style // faint thin left border │ for an assistant turn
	turnFooter      lipgloss.Style // faint per-turn "model · 4.2s" line
	step            lipgloss.Style // faint one-liners
	stepErr         lipgloss.Style // failed tool badge (red ✗)
	stepOK          lipgloss.Style // succeeded tool badge (green ✓)
	errText         lipgloss.Style // terminal error body (red)
	interrupted     lipgloss.Style // dim "(interrupted)" badge
	raw             lipgloss.Style // dim raw fallback for an unexpected event shape

	help help.Styles // bubbles/help styles, concrete per-theme (never AdaptiveColor)
}

// themeColors resolves the per-theme grayscale tones. body is the answer-body foreground, muted
// the header/footer chrome, faint the dim step/commentary lines. These are the tones that broke on
// light terminals — they need opposite ends of the contrast range by background — so they are the
// only per-theme split; the semantic accent/red/green are shared. Kept in one place so newStyles
// and the help palette stay in lockstep and the "no runtime query" invariant is auditable here.
func themeColors(theme string) (body, muted, faint lipgloss.Color) {
	switch theme {
	case themeLight:
		// On a light terminal, 15 (bright white) is invisible and 7 (light gray) near-invisible.
		body = lipgloss.Color("0")  // black — max contrast for the answer body on a light background
		muted = lipgloss.Color("0") // black — readable header/footer chrome (bold weight distinguishes it)
		faint = lipgloss.Color("8") // dark gray — dim step lines, still legible on white
	default: // themeDark — the historical palette
		body = lipgloss.Color("15") // bright white — high contrast on a dark background
		muted = lipgloss.Color("7")  // light gray — header/footer chrome
		faint = lipgloss.Color("8")  // bright black — steps, dim commentary
	}
	return body, muted, faint
}

// newStyles builds the palette for the resolved theme (themeLight or themeDark). It must be a
// concrete theme, never "auto": the colors are fixed lipgloss.Color values so the invariant "no
// terminal query at render time" is trivially auditable from this one function.
func newStyles(theme string) styleSet {
	// Shared semantic colors. Cyan/red/green carry meaning (accent, failure, success) and read
	// acceptably on both light and dark backgrounds, so they are not per-theme.
	const (
		accent = lipgloss.Color("6")  // cyan — user / header accent
		red    = lipgloss.Color("9")  // errors, failed tool
		green  = lipgloss.Color("10") // succeeded tool
	)
	body, muted, faint := themeColors(theme)
	return styleSet{
		header:          lipgloss.NewStyle().Foreground(muted).Bold(true),
		headerAccent:    lipgloss.NewStyle().Foreground(accent).Bold(true),
		footer:          lipgloss.NewStyle().Foreground(faint),
		footerState:     lipgloss.NewStyle().Foreground(accent),
		toast:           lipgloss.NewStyle().Foreground(accent).Bold(true),
		user:            lipgloss.NewStyle().Foreground(accent).Bold(true),
		assistant:       lipgloss.NewStyle().Foreground(body),
		roleLabel:       lipgloss.NewStyle().Foreground(muted).Bold(true),
		userGutter:      lipgloss.NewStyle().Foreground(accent).Bold(true),
		assistantGutter: lipgloss.NewStyle().Foreground(faint),
		turnFooter:      lipgloss.NewStyle().Foreground(faint),
		step:            lipgloss.NewStyle().Foreground(faint),
		stepErr:         lipgloss.NewStyle().Foreground(red),
		stepOK:          lipgloss.NewStyle().Foreground(green),
		errText:         lipgloss.NewStyle().Foreground(red).Bold(true),
		interrupted:     lipgloss.NewStyle().Foreground(faint).Italic(true),
		raw:             lipgloss.NewStyle().Foreground(faint).Italic(true),
		help:            helpStyles(muted, faint),
	}
}

// helpStyles builds the bubbles/help palette from concrete per-theme colors. help.New()'s defaults
// use lipgloss.AdaptiveColor, which queries the terminal background at render time — forbidden
// after the program starts (ADR 0004). Overriding every field with fixed colors keeps the help
// footer inside the no-runtime-query invariant: keys are muted, descriptions and separators faint.
func helpStyles(muted, faint lipgloss.Color) help.Styles {
	key := lipgloss.NewStyle().Foreground(muted)
	desc := lipgloss.NewStyle().Foreground(faint)
	sep := lipgloss.NewStyle().Foreground(faint)
	return help.Styles{
		Ellipsis:       sep,
		ShortKey:       key,
		ShortDesc:      desc,
		ShortSeparator: sep,
		FullKey:        key,
		FullDesc:       desc,
		FullSeparator:  sep,
	}
}

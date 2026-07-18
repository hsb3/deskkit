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

// userBorder is the thick left border glyph for a user block (▌, a left half-block). It is a
// bespoke one-sided lipgloss.Border so the accent edge reads as app chrome, not a quoted line —
// the design language borrowed (only the look) from a colored-left-border chat treatment.
var userBorder = lipgloss.Border{Left: "▌"}

// styleSet bundles the styles the View composes. It is built once (newStyles) and carried on
// the model; nothing here is desk- or identity-specific.
//
// Surfaces read as three distinct planes: the header/footer BARS carry a subtle per-theme
// background fill (barBG); user turns are BLOCKS with a slightly different fill (blockBG) plus a
// thick accent ▌ border so they read as a raised surface above the bars; assistant answers stay
// on the terminal's own background (no fill) for maximum readability. The bar/block segment
// styles all set their fill so the fill is continuous behind text, not just in the trailing pad.
type styleSet struct {
	header          lipgloss.Style // "· provider/model" segment, on the header bar fill
	headerAccent    lipgloss.Style // bold desk-name segment, on the header bar fill
	headerBar       lipgloss.Style // header bar fill (Width applied at render)
	footer          lipgloss.Style
	footerBar       lipgloss.Style // footer/status bar fill (Width applied at render)
	footerState     lipgloss.Style // live state segment, on the footer bar fill
	toast           lipgloss.Style // transient copy/status toast, on the footer bar fill
	user            lipgloss.Style // user text, on the user block fill
	userLabel       lipgloss.Style // "you" label, on the user block fill
	userBlock       lipgloss.Style // thick ▌ accent border + padding + block fill (Width at render)
	assistant       lipgloss.Style // answer body while streaming (before glamour finalizes it)
	roleLabel       lipgloss.Style
	assistantGutter lipgloss.Style // faint thin left border │ for an assistant turn
	inputBorder     lipgloss.Style // rounded input box border, accent (ready for input)
	inputBorderBusy lipgloss.Style // rounded input box border, faint (a turn is streaming)
	turnFooter      lipgloss.Style // faint per-turn "model · 4.2s" line
	step            lipgloss.Style // faint one-liners
	stepErr         lipgloss.Style // failed tool badge (red ✗)
	stepOK          lipgloss.Style // succeeded tool badge (green ✓)
	errText         lipgloss.Style // terminal error body (red)
	interrupted     lipgloss.Style // dim "(interrupted)" badge
	raw             lipgloss.Style // dim raw fallback for an unexpected event shape

	help    help.Styles // bubbles/help styles for the ctrl+g overlay (no bar fill)
	helpBar help.Styles // bubbles/help styles for the collapsed status bar (with bar fill)
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

// themeSurfaces resolves the two per-theme surface fills the app chrome uses. barBG is the
// header/footer status-bar fill; blockBG is the user-turn block fill, one step off barBG so bars
// and blocks read as different planes. On a DARK terminal both are near-black grays and the block
// is a touch LIGHTER than the bar (a raised panel); on a LIGHT terminal both are near-white and
// the block is a touch DARKER than the bar (a recessed panel). Assistant answers use neither —
// they stay on the terminal's own background for maximum readability. Concrete ANSI-256 tones,
// like every other color here, so the no-runtime-query invariant (ADR 0004) stays auditable.
func themeSurfaces(theme string) (barBG, blockBG lipgloss.Color) {
	switch theme {
	case themeLight:
		barBG = lipgloss.Color("254")   // near-white bar on a light terminal
		blockBG = lipgloss.Color("252") // one step darker — user block reads as a recessed panel
	default: // themeDark
		barBG = lipgloss.Color("236")   // near-black bar on a dark terminal
		blockBG = lipgloss.Color("238") // one step lighter — user block reads as a raised panel
	}
	return barBG, blockBG
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
	barBG, blockBG := themeSurfaces(theme)
	return styleSet{
		header:          lipgloss.NewStyle().Foreground(muted).Background(barBG).Bold(true),
		headerAccent:    lipgloss.NewStyle().Foreground(accent).Background(barBG).Bold(true),
		headerBar:       lipgloss.NewStyle().Background(barBG),
		footer:          lipgloss.NewStyle().Foreground(faint),
		footerBar:       lipgloss.NewStyle().Background(barBG),
		footerState:     lipgloss.NewStyle().Foreground(accent).Background(barBG),
		toast:           lipgloss.NewStyle().Foreground(accent).Background(barBG).Bold(true),
		user:            lipgloss.NewStyle().Foreground(accent).Background(blockBG).Bold(true),
		userLabel:       lipgloss.NewStyle().Foreground(muted).Background(blockBG).Bold(true),
		userBlock: lipgloss.NewStyle().
			Border(userBorder, false, false, false, true).
			BorderForeground(accent).
			BorderBackground(blockBG).
			PaddingLeft(1).
			Background(blockBG),
		assistant:       lipgloss.NewStyle().Foreground(body),
		roleLabel:       lipgloss.NewStyle().Foreground(muted).Bold(true),
		assistantGutter: lipgloss.NewStyle().Foreground(faint),
		inputBorder:     lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(accent),
		inputBorderBusy: lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(faint),
		turnFooter:      lipgloss.NewStyle().Foreground(faint),
		step:            lipgloss.NewStyle().Foreground(faint),
		stepErr:         lipgloss.NewStyle().Foreground(red),
		stepOK:          lipgloss.NewStyle().Foreground(green),
		errText:         lipgloss.NewStyle().Foreground(red).Bold(true),
		interrupted:     lipgloss.NewStyle().Foreground(faint).Italic(true),
		raw:             lipgloss.NewStyle().Foreground(faint).Italic(true),
		help:            helpStyles(muted, faint, nil),
		helpBar:         helpStyles(muted, faint, barBG),
	}
}

// helpStyles builds the bubbles/help palette from concrete per-theme colors. help.New()'s defaults
// use lipgloss.AdaptiveColor, which queries the terminal background at render time — forbidden
// after the program starts (ADR 0004). Overriding every field with fixed colors keeps the help
// footer inside the no-runtime-query invariant: keys are muted, descriptions and separators faint.
// A non-nil bg fills every segment so the help text sits on the status-bar surface with no holes
// between segments; pass nil for the ctrl+g overlay, which renders on the terminal background.
func helpStyles(muted, faint lipgloss.Color, bg lipgloss.TerminalColor) help.Styles {
	key := lipgloss.NewStyle().Foreground(muted)
	desc := lipgloss.NewStyle().Foreground(faint)
	sep := lipgloss.NewStyle().Foreground(faint)
	if bg != nil {
		key = key.Background(bg)
		desc = desc.Background(bg)
		sep = sep.Background(bg)
	}
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

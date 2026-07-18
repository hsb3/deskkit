// Lipgloss styles for the chat surface: the header (desk · provider/model), the footer
// (keybind hints + live state), and the transcript roles (user, assistant, steps, errors,
// interrupted badge). Kept in one place so the palette is auditable and the View code reads as
// composition, not styling. Colors are adaptive terminal-palette references (lipgloss resolves
// them against the terminal background), so nothing here assumes a light or dark theme.
package tui

import "github.com/charmbracelet/lipgloss"

// styleSet bundles the styles the View composes. It is built once (newStyles) and carried on
// the model; nothing here is desk- or identity-specific.
type styleSet struct {
	header       lipgloss.Style
	headerAccent lipgloss.Style
	footer       lipgloss.Style
	footerState  lipgloss.Style
	user         lipgloss.Style
	assistant    lipgloss.Style // answer body while streaming (before glamour finalizes it)
	roleLabel    lipgloss.Style
	step         lipgloss.Style // faint one-liners
	stepErr      lipgloss.Style // failed tool badge (red ✗)
	stepOK       lipgloss.Style // succeeded tool badge (green ✓)
	errText      lipgloss.Style // terminal error body (red)
	interrupted  lipgloss.Style // dim "(interrupted)" badge
	raw          lipgloss.Style // dim raw fallback for an unexpected event shape
}

func newStyles() styleSet {
	const (
		accent = lipgloss.Color("6")  // cyan — user / header accent
		faint  = lipgloss.Color("8")  // bright black — steps, dim commentary
		red    = lipgloss.Color("9")  // errors, failed tool
		green  = lipgloss.Color("10") // succeeded tool
		muted  = lipgloss.Color("7")  // header/footer chrome
		bright = lipgloss.Color("15") // bright white — streaming answer body, high contrast
	)
	return styleSet{
		header:       lipgloss.NewStyle().Foreground(muted).Bold(true),
		headerAccent: lipgloss.NewStyle().Foreground(accent).Bold(true),
		footer:       lipgloss.NewStyle().Foreground(faint),
		footerState:  lipgloss.NewStyle().Foreground(accent),
		user:         lipgloss.NewStyle().Foreground(accent).Bold(true),
		assistant:    lipgloss.NewStyle().Foreground(bright),
		roleLabel:    lipgloss.NewStyle().Foreground(muted).Bold(true),
		step:         lipgloss.NewStyle().Foreground(faint),
		stepErr:      lipgloss.NewStyle().Foreground(red),
		stepOK:       lipgloss.NewStyle().Foreground(green),
		errText:      lipgloss.NewStyle().Foreground(red).Bold(true),
		interrupted:  lipgloss.NewStyle().Foreground(faint).Italic(true),
		raw:          lipgloss.NewStyle().Foreground(faint).Italic(true),
	}
}

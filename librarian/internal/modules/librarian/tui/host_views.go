// Module-view hosting (spec §5.3): the chat TUI mounts the views the enabled modules
// contribute through Module.TUIViews — the core tuiview plug-point D2 deferred — without
// knowing any module exists. The chat transcript stays the home surface; ctrl+p cycles
// chat → view 1 → … → view N → chat; esc returns to chat from any view; ?/ctrl+g toggle the
// full-help overlay; every other key while a view is active routes to the view. On a librarian-only
// desk the mounted set is empty, the ctrl+p binding stays disabled (hidden from help), and the
// surface is unchanged apart from an explanatory toast if ctrl+p is pressed anyway.
package tui

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/hsb3/desk-standard/librarian/internal/core/tuiview"
)

// attachViews mounts the module-contributed view set and enables the switcher binding when
// the set is non-empty. Called once by Run before the program starts.
func (m *model) attachViews(views []tuiview.View) {
	m.views = views
	m.activeView = -1
	if len(views) > 0 {
		m.keymap.cycleViews.SetEnabled(true)
	}
}

// activateView makes views[i] the active body view: size it to the current body region and
// run its Init (the load/refresh hook).
func (m model) activateView(i int) (tea.Model, tea.Cmd) {
	if i < 0 || i >= len(m.views) {
		return m, nil
	}
	m.activeView = i
	m.views[i].SetSize(m.vp.Width(), m.vp.Height())
	return m, m.views[i].Init()
}

// handleViewKey routes a keypress while a module view is active: esc returns to the chat,
// ctrl+p advances to the next view (wrapping back to the chat after the last), and any other
// key drives the active view's own Update. ctrl+c quit is already honored before this in
// handleKey, so a view can never trap the user.
func (m model) handleViewKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keymap.cancel):
		m.activeView = -1
		m.refreshViewport()
		return m, nil

	case key.Matches(msg, m.keymap.cycleViews):
		next := m.activeView + 1
		if next >= len(m.views) {
			m.activeView = -1
			m.refreshViewport()
			return m, nil
		}
		return m.activateView(next)

	case key.Matches(msg, m.keymap.help):
		// ? / ctrl+g toggle the same full-help overlay here as in chat. While a view is active every
		// key routes here, so ? always means help (no empty-draft guard) — the textarea is not in play.
		m.hlp.ShowAll = !m.hlp.ShowAll
		return m, nil

	default:
		v, cmd := m.views[m.activeView].Update(msg)
		m.views[m.activeView] = v
		return m, cmd
	}
}

// viewFooter renders the footer bar while a module view is active: the view's name plus the
// switcher hints, on the same status-bar fill as the chat footer.
func (m model) viewFooter() string {
	name := ""
	if m.activeView >= 0 && m.activeView < len(m.views) {
		name = m.views[m.activeView].Name()
	}
	hints := m.styles.footerState.Render("esc chat · ctrl+p next view · ? help · ctrl+c quit")
	state := m.styles.footerState.Render("  " + name)
	return m.styles.footerBar.Width(m.width).MaxWidth(m.width).Render(hints + state)
}

// padToHeight pads content with trailing newlines to exactly h lines (truncating overflow),
// so a module view occupies the same body region the transcript viewport does and the input
// box + footer never jump.
func padToHeight(content string, h int) string {
	if h <= 0 {
		return content
	}
	lines := strings.Split(strings.TrimRight(content, "\n"), "\n")
	if len(lines) > h {
		lines = lines[:h]
	}
	for len(lines) < h {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

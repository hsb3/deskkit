// Package tuiview is the core TUI plug-point (spec §5.3, §2.4 TUIViews): the minimal view
// contract a module implements to mount full-screen views into the shared chat TUI without
// the TUI knowing the module exists. D2 deferred this interface ("D4 adds TUIViews when PM
// TUI views land" — module.go note); this is that addition. It lives in core (not the
// librarian's tui package) so modules/pm can implement it without importing modules/librarian
// (§10.5 import discipline), and the librarian TUI can host it without importing modules/pm.
package tuiview

import tea "charm.land/bubbletea/v2"

// View is one module-contributed TUI view. The host (the shared chat TUI) owns the frame
// (header/footer, sizing, the switch keys); the view owns its body region. Views follow the
// Bubble Tea idiom: all mutation happens through Init/Update on the program goroutine.
type View interface {
	// Name is the short switcher label (e.g. "pm context"), unique within the mounted set.
	Name() string
	// Init runs when the view becomes active (load/refresh); it may return a command.
	Init() tea.Cmd
	// Update handles a message routed to the active view and returns the (possibly replaced)
	// view plus a command.
	Update(msg tea.Msg) (View, tea.Cmd)
	// SetSize tells the view its body region in cells (called on activation and resize).
	SetSize(width, height int)
	// Render returns the view's body content, at most the sized height.
	Render() string
}

// SwitchMsg is emitted by a view's Update command to ask the host to activate the named
// sibling view (e.g. a board's enter opening the item detail view). An unknown name is
// ignored by the host.
type SwitchMsg struct{ Name string }

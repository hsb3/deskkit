package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/example/pocket-librarian/internal/core/tuiview"
)

// fakeView is a minimal tuiview.View: it records lifecycle calls and can ask the host to
// switch to a named sibling on "enter".
type fakeView struct {
	name       string
	inits      int
	w, h       int
	lastKey    string
	switchTo   string
	renderText string
}

func (f *fakeView) Name() string  { return f.name }
func (f *fakeView) Init() tea.Cmd { f.inits++; return nil }
func (f *fakeView) Update(msg tea.Msg) (tuiview.View, tea.Cmd) {
	if k, ok := msg.(tea.KeyPressMsg); ok {
		f.lastKey = k.String()
		if f.lastKey == "enter" && f.switchTo != "" {
			target := f.switchTo
			return f, func() tea.Msg { return tuiview.SwitchMsg{Name: target} }
		}
	}
	return f, nil
}
func (f *fakeView) SetSize(w, h int) { f.w, f.h = w, h }
func (f *fakeView) Render() string   { return f.renderText }

func ctrlP() tea.KeyPressMsg { return tea.KeyPressMsg{Mod: tea.ModCtrl, Code: 'p'} }

// newTestModelWithViews builds the standard ready 80x24 test model with two fake views
// mounted (as Run's attachViews does on a pm-enabled desk).
func newTestModelWithViews(t *testing.T) (model, *fakeView, *fakeView) {
	t.Helper()
	m, _ := newTestModel(t)
	v1 := &fakeView{name: "pm context", renderText: "CONTEXT BODY"}
	v2 := &fakeView{name: "pm board", renderText: "BOARD BODY", switchTo: "pm context"}
	m.attachViews([]tuiview.View{v1, v2})
	return m, v1, v2
}

// TestViews_DisabledWithoutMount: on a librarian-only desk (no mounted views) ctrl+p is a
// no-op, the binding stays disabled (hidden from help), and the surface is unchanged.
func TestViews_DisabledWithoutMount(t *testing.T) {
	m, _ := newTestModel(t)
	if m.keymap.cycleViews.Enabled() {
		t.Error("cycleViews binding must start disabled with no mounted views")
	}
	next := send(m, ctrlP())
	if next.activeView != -1 {
		t.Errorf("ctrl+p with no views activated view %d", next.activeView)
	}
}

// TestViews_CycleAndReturn drives the switcher: ctrl+p enters view 1 (Init + sized), ctrl+p
// again advances to view 2, a third wraps back to the chat; esc from any view returns to chat.
func TestViews_CycleAndReturn(t *testing.T) {
	m, v1, v2 := newTestModelWithViews(t)
	if !m.keymap.cycleViews.Enabled() {
		t.Fatal("cycleViews binding must be enabled once views are mounted")
	}

	m = send(m, ctrlP())
	if m.activeView != 0 || v1.inits != 1 {
		t.Fatalf("ctrl+p should activate view 0 and Init it (active=%d inits=%d)", m.activeView, v1.inits)
	}
	if v1.w != 80 || v1.h != 17 {
		t.Errorf("view sized to %dx%d, want 80x17 (the body region)", v1.w, v1.h)
	}
	if body := m.View().Content; !strings.Contains(body, "CONTEXT BODY") {
		t.Error("active view's Render not shown in the body")
	}
	// The view footer replaces the chat footer.
	if footer := m.View().Content; !strings.Contains(footer, "esc chat") {
		t.Error("view-mode footer hints missing")
	}

	m = send(m, ctrlP())
	if m.activeView != 1 || v2.inits != 1 {
		t.Fatalf("second ctrl+p should activate view 1 (active=%d)", m.activeView)
	}
	m = send(m, ctrlP())
	if m.activeView != -1 {
		t.Fatalf("ctrl+p past the last view should return to chat (active=%d)", m.activeView)
	}

	// esc returns to chat from a view.
	m = send(m, ctrlP())
	m = send(m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.activeView != -1 {
		t.Fatalf("esc should return to chat (active=%d)", m.activeView)
	}
}

// TestViews_KeyRoutingAndSwitchMsg: other keys route to the active view, and a view-emitted
// SwitchMsg activates the named sibling (the board's enter opening the detail view pattern).
func TestViews_KeyRoutingAndSwitchMsg(t *testing.T) {
	m, v1, v2 := newTestModelWithViews(t)
	m = send(m, ctrlP()) // view 0
	m = send(m, ctrlP()) // view 1 (the board fake)
	m = send(m, tea.KeyPressMsg{Code: 'r'})
	if v2.lastKey != "r" {
		t.Fatalf("key not routed to the active view (last=%q)", v2.lastKey)
	}

	// enter on the fake board emits SwitchMsg{"pm context"}; deliver it as Update would.
	next, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = next.(model)
	if cmd == nil {
		t.Fatal("enter should produce the switch command")
	}
	next, _ = m.Update(cmd())
	m = next.(model)
	if m.activeView != 0 {
		t.Fatalf("SwitchMsg should activate the named view (active=%d)", m.activeView)
	}
	if v1.inits != 2 { // once on first activation, once on the switch
		t.Errorf("switch should re-Init the target view (inits=%d)", v1.inits)
	}

	// While a view is active, typing keys do NOT reach the textarea.
	m = send(m, tea.KeyPressMsg{Code: 'x', Text: "x"})
	if m.ta.Value() != "" {
		t.Errorf("textarea received input while a view was active: %q", m.ta.Value())
	}
}

// TestViews_ChatUnchangedWhileInactive: with views mounted but the chat active, the
// transcript body renders as usual and typing reaches the textarea (no behavior change).
func TestViews_ChatUnchangedWhileInactive(t *testing.T) {
	m, _, _ := newTestModelWithViews(t)
	m = send(m, tea.KeyPressMsg{Code: 'h', Text: "h"})
	if m.ta.Value() != "h" {
		t.Errorf("typing must still reach the textarea when the chat is active (got %q)", m.ta.Value())
	}
	if body := m.View().Content; strings.Contains(body, "CONTEXT BODY") {
		t.Error("inactive view leaked into the chat body")
	}
}

// TestPadToHeight pins the body-padding helper: exact height, truncation, padding.
func TestPadToHeight(t *testing.T) {
	if got := padToHeight("a\nb", 4); got != "a\nb\n\n" {
		t.Errorf("pad: %q", got)
	}
	if got := padToHeight("a\nb\nc", 2); got != "a\nb" {
		t.Errorf("truncate: %q", got)
	}
	if got := padToHeight("x", 0); got != "x" {
		t.Errorf("h<=0 passthrough: %q", got)
	}
}

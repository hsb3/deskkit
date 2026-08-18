package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/hsb3/deskkit/internal/core/tuiview"
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
// mounted (as Run's attachViews does on a pm-enabled desk). It re-sends the sizing message AFTER
// attaching so the layout reflects the mounted-views body height (the persistent tab strip claims
// one row) — mirroring production, where attachViews runs before the first WindowSizeMsg.
func newTestModelWithViews(t *testing.T) (model, *fakeView, *fakeView) {
	t.Helper()
	m, _ := newTestModel(t)
	v1 := &fakeView{name: "pm context", renderText: "CONTEXT BODY"}
	v2 := &fakeView{name: "pm board", renderText: "BOARD BODY", switchTo: "pm context"}
	m.attachViews([]tuiview.View{v1, v2})
	m = send(m, tea.WindowSizeMsg{Width: 80, Height: 24})
	return m, v1, v2
}

// TestViews_ExplainWithoutMount: on a librarian-only desk (no mounted views) the switcher binding
// stays disabled (hidden from help), and ctrl+p never fails silently — it surfaces an explanatory
// toast WITHOUT activating any view (activeView stays -1).
func TestViews_ExplainWithoutMount(t *testing.T) {
	m, _ := newTestModel(t)
	if m.keymap.cycleViews.Enabled() {
		t.Error("cycleViews binding must start disabled with no mounted views")
	}
	next := send(m, ctrlP())
	if next.activeView != -1 {
		t.Errorf("ctrl+p with no views activated view %d", next.activeView)
	}
	if next.toast == "" {
		t.Error("ctrl+p with no views must surface an explanatory toast, not fail silently")
	}
	if next.ta.Value() != "" {
		t.Errorf("ctrl+p leaked into the textarea: %q", next.ta.Value())
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
	if v1.w != 80 || v1.h != 16 {
		t.Errorf("view sized to %dx%d, want 80x16 (the body region: 24 - header - tab strip - footer - input - border)", v1.w, v1.h)
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

// TestViews_TabStripVisible: with views mounted, the persistent switcher strip is rendered in the
// surface WITHOUT pressing ctrl+p — chat plus every mounted view is discoverable on sight.
func TestViews_TabStripVisible(t *testing.T) {
	m, _, _ := newTestModelWithViews(t)
	body := m.View().Content
	for _, want := range []string{"chat", "pm context", "pm board"} {
		if !strings.Contains(body, want) {
			t.Errorf("tab strip missing %q segment; mounted views must be discoverable on sight", want)
		}
	}
}

// TestViews_TabStripAbsentWithoutMount: a librarian-only desk (no views) shows no switcher strip, so
// the "chat" tab label — present only in the strip — never appears. The surface stays unchanged.
func TestViews_TabStripAbsentWithoutMount(t *testing.T) {
	m, _ := newTestModel(t)
	if strings.Contains(m.View().Content, "chat") {
		t.Error("tab strip rendered on a desk with no mounted views (found the 'chat' segment)")
	}
}

// TestViews_HelpOverlayInViewMode: while a module view is active, ? opens the SAME full-help overlay
// ctrl+g surfaces in chat, and a second ? closes it. The overlay carries a full-help-only binding
// ("prev prompt"), absent from the view footer, so its presence is unambiguous.
func TestViews_HelpOverlayInViewMode(t *testing.T) {
	m, _, _ := newTestModelWithViews(t)
	m = send(m, ctrlP()) // enter view 0
	if m.activeView != 0 {
		t.Fatalf("expected view 0 active, got %d", m.activeView)
	}
	m = send(m, tea.KeyPressMsg{Text: "?", Code: '?'})
	if !m.hlp.ShowAll {
		t.Fatal("? in view mode did not open the help overlay")
	}
	if !strings.Contains(m.View().Content, "prev prompt") {
		t.Error("full-help overlay not rendered in view mode")
	}
	m = send(m, tea.KeyPressMsg{Text: "?", Code: '?'})
	if m.hlp.ShowAll {
		t.Error("second ? did not close the help overlay in view mode")
	}
}

// TestViews_LaunchHint: with views mounted and the launch nudge armed, the first WindowSizeMsg seeds
// exactly one roleInfo transcript line naming the views and the ? help key, and a later resize never
// reseeds it (one-time).
func TestViews_LaunchHint(t *testing.T) {
	m, _ := newTestModel(t)
	v1 := &fakeView{name: "pm context", renderText: "CONTEXT BODY"}
	v2 := &fakeView{name: "pm board", renderText: "BOARD BODY"}
	m.attachViews([]tuiview.View{v1, v2})
	m.enableLaunchHint()
	m = send(m, tea.WindowSizeMsg{Width: 80, Height: 24})
	if len(m.entries) != 1 || m.entries[0].role != roleInfo {
		t.Fatalf("launch hint not seeded as a single roleInfo entry: %+v", m.entries)
	}
	if !strings.Contains(m.entries[0].text, "pm board") || !strings.Contains(m.entries[0].text, "?") {
		t.Errorf("launch hint missing a view name or the ? help key: %q", m.entries[0].text)
	}
	m = send(m, tea.WindowSizeMsg{Width: 80, Height: 30})
	if len(m.entries) != 1 {
		t.Errorf("launch hint reseeded on a later resize: %d entries", len(m.entries))
	}
}

// TestViews_LaunchHintAbsentWithoutMount: arming the nudge on a librarian-only desk seeds nothing.
func TestViews_LaunchHintAbsentWithoutMount(t *testing.T) {
	m, _ := newTestModel(t)
	m.enableLaunchHint()
	m = send(m, tea.WindowSizeMsg{Width: 80, Height: 24})
	if len(m.entries) != 0 {
		t.Errorf("launch hint seeded with no views mounted: %+v", m.entries)
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

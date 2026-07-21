// Package tui is the PM module's three views for the shared chat TUI (spec §5.3, mounted via
// Module.TUIViews): the get_context landing view, the board/queue view, and the item detail
// view. Each view is a THIN read surface over the SAME engine core functions the MCP tools
// and the CLI call (§10.10 parity) — no transition/gate logic lives here. Data loads
// synchronously on activation/refresh (fast local store reads, the picker idiom); rendering
// is plain text so it sits on any terminal theme.
package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/pocketbase/pocketbase/core"

	"github.com/hsb3/desk-standard/librarian/internal/core/config"
	"github.com/hsb3/desk-standard/librarian/internal/core/tuiview"
	"github.com/hsb3/desk-standard/librarian/internal/modules/pm/engine"
)

// View names — the switcher labels and the SwitchMsg targets (frozen for the host + tests).
const (
	ContextViewName = "pm context"
	BoardViewName   = "pm board"
	DetailViewName  = "pm item"
)

// state is the shared cross-view state: the engine binding and the board's selection, which
// the detail view reads (enter on the board switches to the detail of the selected item).
type state struct {
	eng      *engine.Engine
	selected string
}

// Views builds the three PM views over one shared state (spec §5.3 order: landing first).
func Views(app core.App, cfg *config.Config) []tuiview.View {
	st := &state{eng: &engine.Engine{App: app, Cfg: cfg}}
	return []tuiview.View{
		&contextView{st: st},
		&boardView{st: st},
		&detailView{st: st},
	}
}

// --- the get_context landing view (§5.2/§5.3) ---

type contextView struct {
	st            *state
	width, height int
	res           *engine.ContextResult
	err           error
}

func (v *contextView) Name() string { return ContextViewName }

// Init loads the briefing synchronously — the SAME engine.GetContext call the get_context
// tool and `pm context` CLI make (§10.10).
func (v *contextView) Init() tea.Cmd {
	v.res, v.err = v.st.eng.GetContext(context.Background(), 0)
	return nil
}

// Result exposes the loaded briefing (the §10.10 parity test compares it across surfaces).
func (v *contextView) Result() *engine.ContextResult { return v.res }

func (v *contextView) Update(msg tea.Msg) (tuiview.View, tea.Cmd) {
	if k, ok := msg.(tea.KeyPressMsg); ok && k.String() == "r" {
		return v, v.Init()
	}
	return v, nil
}

func (v *contextView) SetSize(w, h int) { v.width, v.height = w, h }

func (v *contextView) Render() string {
	if v.err != nil {
		return "pm context: " + v.err.Error()
	}
	if v.res == nil {
		return "pm context: loading… (press r to refresh)"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "desk %s — generated %s\n", v.res.Desk, v.res.GeneratedAt)
	fmt.Fprintf(&b, "counts: %s | %s\n\n", countLine("phase", v.res.Counts.ByPhase), countLine("court", v.res.Counts.ByCourt))

	fmt.Fprintf(&b, "ACTIVE (%d)\n", len(v.res.Active))
	for _, it := range v.res.Active {
		fmt.Fprintf(&b, "  %s\n", summaryLine(it))
	}
	fmt.Fprintf(&b, "\nBLOCKED (%d)\n", len(v.res.Blocked))
	for _, it := range v.res.Blocked {
		fmt.Fprintf(&b, "  %s", summaryLine(it.ItemSummary))
		if len(it.BlockingItems) > 0 {
			fmt.Fprintf(&b, "  <- blocked by %s", strings.Join(it.BlockingItems, ", "))
		}
		if it.BlockedReason != "" {
			fmt.Fprintf(&b, "  (%s)", it.BlockedReason)
		}
		b.WriteString("\n")
	}
	fmt.Fprintf(&b, "\nSTALLED (%d)\n", len(v.res.Stalled))
	for _, it := range v.res.Stalled {
		fmt.Fprintf(&b, "  %s  — %dd since last transition\n", summaryLine(it.ItemSummary), it.DaysSinceLastTransition)
	}
	fmt.Fprintf(&b, "\nRECENT TRANSITIONS (%d)\n", len(v.res.RecentTransitions))
	for _, tr := range v.res.RecentTransitions {
		fmt.Fprintf(&b, "  %s  %s %s", tr.At, tr.Event, tr.Item)
		if tr.From != "" || tr.To != "" {
			fmt.Fprintf(&b, "  %s->%s", tr.From, tr.To)
		}
		if tr.Actor != "" {
			fmt.Fprintf(&b, "  by %s", tr.Actor)
		}
		b.WriteString("\n")
	}
	b.WriteString("\n[r refresh]")
	return clampHeight(b.String(), v.height)
}

// --- the board/queue view (§5.3 graph/queue) ---

type boardView struct {
	st            *state
	width, height int
	items         []engine.ItemSummary
	cursor        int
	err           error
}

func (v *boardView) Name() string { return BoardViewName }

// Init loads the full graph via the SAME engine.ListItems the list_items tool calls.
func (v *boardView) Init() tea.Cmd {
	v.items, v.err = v.st.eng.ListItems(context.Background(), engine.ListFilter{})
	if v.cursor >= len(v.items) {
		v.cursor = 0
	}
	return nil
}

func (v *boardView) Update(msg tea.Msg) (tuiview.View, tea.Cmd) {
	k, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return v, nil
	}
	switch k.String() {
	case "r":
		return v, v.Init()
	case "up", "k":
		if v.cursor > 0 {
			v.cursor--
		}
	case "down", "j":
		if v.cursor < len(v.items)-1 {
			v.cursor++
		}
	case "enter":
		if v.cursor >= 0 && v.cursor < len(v.items) {
			v.st.selected = v.items[v.cursor].ID
			return v, func() tea.Msg { return tuiview.SwitchMsg{Name: DetailViewName} }
		}
	}
	return v, nil
}

func (v *boardView) SetSize(w, h int) { v.width, v.height = w, h }

func (v *boardView) Render() string {
	if v.err != nil {
		return "pm board: " + v.err.Error()
	}
	if len(v.items) == 0 {
		return "pm board: no items on this desk yet (create one with the create_item tool or `pm create`)\n\n[r refresh]"
	}
	var b strings.Builder
	phase := ""
	for i, it := range v.items {
		if it.Phase != phase {
			phase = it.Phase
			fmt.Fprintf(&b, "%s\n", strings.ToUpper(phase))
		}
		marker := "  "
		if i == v.cursor {
			marker = "> "
		}
		fmt.Fprintf(&b, "%s%s\n", marker, summaryLine(it))
	}
	b.WriteString("\n[up/down select · enter detail · r refresh]")
	return clampHeight(b.String(), v.height)
}

// --- the item detail view (§5.3) ---

type detailView struct {
	st            *state
	width, height int
	loadedFor     string
	detail        *engine.ItemDetail
	err           error
}

func (v *detailView) Name() string { return DetailViewName }

// Init loads the selected item via the SAME engine.GetItem the get_item tool calls.
func (v *detailView) Init() tea.Cmd {
	v.loadedFor = v.st.selected
	if v.loadedFor == "" {
		v.detail, v.err = nil, nil
		return nil
	}
	v.detail, v.err = v.st.eng.GetItem(context.Background(), v.loadedFor)
	return nil
}

func (v *detailView) Update(msg tea.Msg) (tuiview.View, tea.Cmd) {
	// Reload when the board moved the selection under us (review finding: state changes
	// belong in Init/Update, never in Render — Render stays a pure read).
	if v.loadedFor != v.st.selected {
		return v, v.Init()
	}
	if k, ok := msg.(tea.KeyPressMsg); ok && k.String() == "r" {
		return v, v.Init()
	}
	return v, nil
}

func (v *detailView) SetSize(w, h int) { v.width, v.height = w, h }

func (v *detailView) Render() string {
	if v.st.selected == "" {
		return "pm item: nothing selected — pick an item on the board (enter) first"
	}
	if v.err != nil {
		return "pm item: " + v.err.Error()
	}
	if v.detail == nil {
		return "pm item: loading… (press r to refresh)"
	}
	d := v.detail
	var b strings.Builder
	fmt.Fprintf(&b, "%s  [%s]\n", d.Title, d.ID)
	fmt.Fprintf(&b, "phase %s (%s)  type %s  court %s  severity %s  priority %d  version %d\n",
		d.Phase, d.StatusLabel, dash(d.Type), dash(d.Court), dash(d.Severity), d.Priority, d.Version)
	if d.Blocked {
		b.WriteString("BLOCKED\n")
	}
	if d.Pointer != "" {
		fmt.Fprintf(&b, "pointer: %s\n", d.Pointer)
	}
	if d.ClaimedBy != "" {
		fmt.Fprintf(&b, "claimed by: %s\n", d.ClaimedBy)
	}
	if len(d.Ancestors) > 0 {
		fmt.Fprintf(&b, "ancestors: %s\n", strings.Join(d.Ancestors, " > "))
	}
	fmt.Fprintf(&b, "\nDEPENDENCIES (%d)\n", len(d.Dependencies))
	for _, e := range d.Dependencies {
		fmt.Fprintf(&b, "  %s -%s-> %s", e.From, e.Kind, e.To)
		if e.UnblockAt != "" {
			fmt.Fprintf(&b, " (unblock at %s, cascade %s)", e.UnblockAt, e.Cascade)
		}
		b.WriteString("\n")
	}
	fmt.Fprintf(&b, "\nNOTES (%d)\n", len(d.Notes))
	for _, n := range d.Notes {
		fmt.Fprintf(&b, "  [%s/%s] %s\n", n.Phase, n.Key, n.Body)
	}
	fmt.Fprintf(&b, "\nTRANSITIONS (%d)\n", len(d.RecentTransitions))
	for _, tr := range d.RecentTransitions {
		fmt.Fprintf(&b, "  %s  %s %s->%s", tr.At, tr.Event, tr.From, tr.To)
		if tr.Actor != "" {
			fmt.Fprintf(&b, "  by %s", tr.Actor)
		}
		if tr.Detail != "" {
			fmt.Fprintf(&b, "  — %s", tr.Detail)
		}
		b.WriteString("\n")
	}
	b.WriteString("\n[r refresh]")
	return clampHeight(b.String(), v.height)
}

// --- helpers ---

// summaryLine renders one item summary as a single line.
func summaryLine(it engine.ItemSummary) string {
	parts := []string{it.ID, it.Title}
	meta := []string{it.Phase}
	if it.StatusLabel != "" && it.StatusLabel != it.Phase {
		meta = append(meta, it.StatusLabel)
	}
	if it.Court != "" {
		meta = append(meta, "court:"+it.Court)
	}
	if it.Type != "" {
		meta = append(meta, "type:"+it.Type)
	}
	if it.Blocked {
		meta = append(meta, "BLOCKED")
	}
	if it.ClaimedBy != "" {
		meta = append(meta, "claimed:"+it.ClaimedBy)
	}
	return fmt.Sprintf("%s  (%s)", strings.Join(parts, "  "), strings.Join(meta, " · "))
}

// countLine renders a count map deterministically (sorted keys).
func countLine(label string, m map[string]int) string {
	if len(m) == 0 {
		return label + ": none"
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = fmt.Sprintf("%s=%d", k, m[k])
	}
	return label + " " + strings.Join(parts, " ")
}

// dash renders an empty field as "-" so the detail line stays scannable.
func dash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// clampHeight truncates content to the view's body height (the host viewport does not
// scroll module views in v1; the top of the content is the useful part).
func clampHeight(s string, h int) string {
	if h <= 0 {
		return s
	}
	lines := strings.Split(s, "\n")
	if len(lines) <= h {
		return s
	}
	return strings.Join(lines[:h], "\n")
}

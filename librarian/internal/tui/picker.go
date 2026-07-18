// The conversation-resume picker overlay for the chat surface (chat-TUI plan, Phase 4). It wraps
// a bubbles/list of prior manual runs (agent.ListConversations) and renders in place of the
// transcript viewport while open. The model routes keys to it first (picker.go stays a pure view
// component: no Session, DB, or LLM here); the model's Update owns the session swap on enter.
//
// The list only ever lives while NOT streaming — the model opens it on ctrl+o only when a turn is
// not in flight — so nothing in this file has to reason about an in-flight turn.
package tui

import (
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"

	"github.com/example/pocket-librarian/internal/agent"
)

// pickerItem adapts one agent.ConversationInfo to the bubbles/list Item (and DefaultItem)
// interfaces. The whole label lives on Title so the list renders as single-line rows (the
// delegate's description line is disabled); Description returns "" accordingly.
type pickerItem struct {
	runID string
	label string
}

// Title is the single visible line: `<summary> — <date> · <status>`.
func (i pickerItem) Title() string { return i.label }

// Description is empty: the picker uses single-line rows.
func (i pickerItem) Description() string { return "" }

// FilterValue is what the list's built-in filter matches against — the visible label.
func (i pickerItem) FilterValue() string { return i.label }

// pickerModel is the overlay: a bubbles/list of resumable conversations. It is a view component
// only — selecting a row hands its RunID back to the model, which performs the session swap.
type pickerModel struct {
	list list.Model
}

// newPicker builds the overlay from the listed conversations, sized to the given viewport area.
// Each row reads `<title> — <date> · <status>`, where the date is the run's start time. The
// filter/status chrome is trimmed so the overlay stays a compact list of choices.
func newPicker(convos []agent.ConversationInfo, width, height int) *pickerModel {
	items := make([]list.Item, 0, len(convos))
	for _, c := range convos {
		items = append(items, pickerItem{runID: c.RunID, label: convoLabel(c)})
	}

	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false // single-line rows: the whole label lives on Title

	l := list.New(items, delegate, width, height)
	l.Title = "resume a conversation"
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.DisableQuitKeybindings() // the model owns esc/enter; the list must not intercept them

	return &pickerModel{list: l}
}

// convoLabel renders one conversation's single-line label. A blank title falls back to a neutral
// placeholder so an untitled run is still selectable.
func convoLabel(c agent.ConversationInfo) string {
	title := c.Title
	if title == "" {
		title = "(untitled conversation)"
	}
	return title + " — " + c.Started.Time().Format("2006-01-02 15:04") + " · " + c.Status
}

// Update forwards a message to the inner list (cursor movement, resize handled by the list).
func (p *pickerModel) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	p.list, cmd = p.list.Update(msg)
	return cmd
}

// View renders the list overlay.
func (p *pickerModel) View() string {
	return p.list.View()
}

// SetSize resizes the inner list on a WindowSizeMsg.
func (p *pickerModel) SetSize(width, height int) {
	p.list.SetSize(width, height)
}

// selectedRunID returns the RunID of the highlighted conversation, or "" when the list is empty.
func (p *pickerModel) selectedRunID() string {
	if it, ok := p.list.SelectedItem().(pickerItem); ok {
		return it.runID
	}
	return ""
}

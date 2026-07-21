// The sessions surface for the chat overlay (chat-TUI plan, Phase 4): a bubbles/list of prior
// manual runs (agent.ListConversations) with a live transcript preview, built-in fuzzy filter,
// and inline rename / delete-with-confirm. It renders in place of the transcript viewport while
// open. picker.go stays a PURE VIEW COMPONENT — no Session, DB, or LLM here — so it only holds
// view state (the list, the current mode, the rename input, the loaded preview) and reports the
// selected run back to the model; the model's Update owns every provider call (list, resume,
// rename, delete, preview). The overlay only ever lives while NOT streaming (ctrl+o opens it only
// when no turn is in flight), so nothing here reasons about an in-flight turn.
package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/hsb3/desk-standard/librarian/internal/modules/librarian/agent"
)

// pickerMode is the overlay's current interaction mode. In browse the cursor walks the list and a
// preview follows it; rename is an inline text-input over the selected row's title; confirmDelete
// is the y/n gate that guards a hard delete.
type pickerMode int

const (
	pickerBrowse pickerMode = iota
	pickerRename
	pickerConfirmDelete
)

// renamePromptLabel prefixes the inline rename input on the action line.
const renamePromptLabel = "rename: "

// pickerItem adapts one agent.ConversationInfo to the bubbles/list Item (and DefaultItem)
// interfaces. label is the rendered single-line row (title · when · count · status); title is the
// raw title, kept so entering rename can seed the input with it. The whole label lives on Title so
// rows are single-line (the delegate's description line is disabled); Description returns "".
type pickerItem struct {
	runID string
	title string
	label string
}

// Title is the single visible line.
func (i pickerItem) Title() string { return i.label }

// Description is empty: the picker uses single-line rows.
func (i pickerItem) Description() string { return "" }

// FilterValue is what the list's built-in fuzzy filter matches against — the conversation title,
// so a search reads as searching by what the run was named.
func (i pickerItem) FilterValue() string {
	if i.title == "" {
		return i.label
	}
	return i.title
}

// pickerModel is the overlay. It is a view component only: selecting a row hands its RunID back to
// the model (selectedRunID), which performs the session swap / rename / delete.
type pickerModel struct {
	list   list.Model
	styles styleSet
	mode   pickerMode
	rename textinput.Model

	previewID      string                  // runID whose preview is currently loaded ("" = none)
	previewEntries []agent.TranscriptEntry // recent transcript of the highlighted conversation

	width  int
	height int
}

// newPicker builds the overlay from the listed conversations, sized to the given viewport area.
// Filtering is re-enabled (the built-in fuzzy search); quit keybindings stay disabled so the list
// never eats a bare enter/esc while browsing — the model owns those.
func newPicker(convos []agent.ConversationInfo, styles styleSet, width, height int) *pickerModel {
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false        // single-line rows: the whole label lives on Title
	delegate.Styles = styles.pickerDelegate // theme the rows to the app palette (styles.go), not the
	// delegate's hardcoded-dark defaults — so the title is legible on a light terminal and the
	// selected row carries the surface's cyan accent instead of bubbles' magenta.

	l := list.New(pickerItemsOf(convos), delegate, width, height)
	l.Title = "sessions"
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true) // built-in fuzzy filter/search over the titles
	l.DisableQuitKeybindings()  // the model owns bare enter/esc when NOT filtering

	ti := textinput.New()
	ti.Prompt = ""
	// Inline virtual cursor, consistent with the main textarea (ADR 0007): no terminal cursor is
	// wired through tea.View.Cursor, so nothing queries the terminal after the program starts.
	ti.SetVirtualCursor(true)

	p := &pickerModel{list: l, styles: styles, rename: ti}
	p.SetSize(width, height)
	return p
}

// pickerItemsOf renders each conversation into a list item.
func pickerItemsOf(convos []agent.ConversationInfo) []list.Item {
	items := make([]list.Item, 0, len(convos))
	for _, c := range convos {
		items = append(items, pickerItem{runID: c.RunID, title: c.Title, label: convoLabel(c)})
	}
	return items
}

// convoLabel renders one conversation's single-line row: `<title> — <last activity> · <N msgs> ·
// <status>`. A blank title falls back to a neutral placeholder so an untitled run is still
// selectable. The timestamp is the last activity (not the start), so the row reads as "when did
// this conversation last move".
func convoLabel(c agent.ConversationInfo) string {
	title := c.Title
	if title == "" {
		title = "(untitled conversation)"
	}
	when := c.LastActivity.Time().Format("2006-01-02 15:04")
	return title + " — " + when + " · " + msgCountLabel(c.MsgCount) + " · " + c.Status
}

// msgCountLabel renders a message count with singular/plural agreement.
func msgCountLabel(n int) string {
	if n == 1 {
		return "1 msg"
	}
	return fmt.Sprintf("%d msgs", n)
}

// layout splits the overlay's height into the list region, the preview region, and the one-line
// divider + one-line action row between and below them. The preview takes roughly two-fifths.
func (p *pickerModel) layout() (listH, previewH int) {
	const dividerRows, actionRows = 1, 1
	previewH = p.height * 2 / 5
	if previewH < 3 {
		previewH = 3
	}
	listH = p.height - previewH - dividerRows - actionRows
	if listH < 1 {
		listH = 1
		previewH = p.height - listH - dividerRows - actionRows
		if previewH < 1 {
			previewH = 1
		}
	}
	return listH, previewH
}

// Update forwards a message to the active sub-component: the rename input while renaming, else the
// inner list (cursor movement, filter editing). Confirm-delete consumes no keys through here — the
// model handles y/n directly.
func (p *pickerModel) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	if p.mode == pickerRename {
		p.rename, cmd = p.rename.Update(msg)
		return cmd
	}
	p.list, cmd = p.list.Update(msg)
	return cmd
}

// View renders the overlay: the list, a divider, the preview pane, and a mode-dependent action
// row, each padded to its budgeted height so the input box and footer below never jump.
func (p *pickerModel) View() string {
	listH, previewH := p.layout()
	parts := []string{
		padToHeight(p.list.View(), listH),
		p.styles.pickerDivider.Render(dividerLine(p.width)),
		padToHeight(p.renderPreview(), previewH),
		p.renderAction(),
	}
	return strings.Join(parts, "\n")
}

// renderPreview renders the highlighted conversation's recent transcript as compact one-line
// entries. An empty preview (no selection, or a run with no readable rows) shows a neutral hint.
func (p *pickerModel) renderPreview() string {
	if len(p.previewEntries) == 0 {
		return p.styles.pickerHint.Render("  (no preview)")
	}
	lines := make([]string, 0, len(p.previewEntries))
	for _, e := range p.previewEntries {
		lines = append(lines, p.renderPreviewLine(e))
	}
	return strings.Join(lines, "\n")
}

// renderPreviewLine renders one transcript entry as a compact, single line: a role label, then
// the text with internal whitespace collapsed and the whole line truncated to the pane width.
func (p *pickerModel) renderPreviewLine(e agent.TranscriptEntry) string {
	text := strings.Join(strings.Fields(e.Text), " ")
	var label string
	switch e.Role {
	case "user":
		label = "you"
	case "assistant":
		if e.ToolName != "" {
			label = "librarian → " + e.ToolName
		} else {
			label = "librarian"
		}
	case "tool":
		label = "  " + e.ToolName
	default:
		label = e.Role
	}
	line := label
	if text != "" {
		line += ": " + text
	}
	return p.styles.pickerPreview.Render(truncateLine(line, maxInt(p.width-1, 8)))
}

// renderAction renders the bottom action row for the current mode: the inline rename input, the
// delete confirm prompt, or the browse hint (which degrades to an empty-list message).
func (p *pickerModel) renderAction() string {
	switch p.mode {
	case pickerRename:
		return p.styles.pickerRenamePrompt.Render(renamePromptLabel) + p.rename.View()
	case pickerConfirmDelete:
		return p.styles.pickerDeleteConfirm.Render("delete this conversation? (y)es / (n)o")
	default:
		if p.selectedRunID() == "" {
			return p.styles.pickerHint.Render("no conversations · esc close")
		}
		return p.styles.pickerHint.Render("↑↓ navigate · enter resume · r rename · d delete · / filter · esc close")
	}
}

// SetSize resizes the inner list and rename input on a WindowSizeMsg.
func (p *pickerModel) SetSize(width, height int) {
	p.width = width
	p.height = height
	listH, _ := p.layout()
	p.list.SetSize(width, listH)
	p.rename.SetWidth(maxInt(width-len(renamePromptLabel)-1, 8))
}

// selectedRunID returns the RunID of the highlighted conversation, or "" when the list is empty.
func (p *pickerModel) selectedRunID() string {
	if it, ok := p.list.SelectedItem().(pickerItem); ok {
		return it.runID
	}
	return ""
}

// selectedTitle returns the raw title of the highlighted conversation (seeds the rename input).
func (p *pickerModel) selectedTitle() string {
	if it, ok := p.list.SelectedItem().(pickerItem); ok {
		return it.title
	}
	return ""
}

// settingFilter reports whether the list is actively editing a filter query (the user is typing
// after pressing "/"). While true, the model routes every key to the list.
func (p *pickerModel) settingFilter() bool { return p.list.SettingFilter() }

// filterApplied reports whether a filter is applied and the user is no longer editing it — the
// state where esc should clear the filter rather than close the overlay.
func (p *pickerModel) filterApplied() bool { return p.list.FilterState() == list.FilterApplied }

// startRename enters rename mode, seeding the input with the selected row's current title and
// focusing it. The returned command starts the input's cursor blink.
func (p *pickerModel) startRename() tea.Cmd {
	p.mode = pickerRename
	p.rename.SetValue(p.selectedTitle())
	p.rename.CursorEnd()
	return p.rename.Focus()
}

// cancelRename returns to browse mode, discarding the in-progress rename.
func (p *pickerModel) cancelRename() {
	p.mode = pickerBrowse
	p.rename.Blur()
	p.rename.Reset()
}

// renameValue is the trimmed current rename input.
func (p *pickerModel) renameValue() string { return strings.TrimSpace(p.rename.Value()) }

// startConfirmDelete enters the delete-confirm gate.
func (p *pickerModel) startConfirmDelete() { p.mode = pickerConfirmDelete }

// cancelConfirmDelete backs out of the delete-confirm gate to browse mode.
func (p *pickerModel) cancelConfirmDelete() { p.mode = pickerBrowse }

// setPreview stores the loaded preview for the highlighted run (runID guards a redundant reload).
func (p *pickerModel) setPreview(entries []agent.TranscriptEntry, runID string) {
	p.previewEntries = entries
	p.previewID = runID
}

// reload rebuilds the list from a fresh conversation set after a rename or delete, returns to a
// clean unfiltered browse, and re-selects selectRunID when it is still present (a delete passes ""
// so the list settles on the nearest remaining row). The preview is invalidated so it reloads for
// the new selection.
func (p *pickerModel) reload(convos []agent.ConversationInfo, selectRunID string) {
	p.list.ResetFilter()
	p.list.SetItems(pickerItemsOf(convos))
	p.mode = pickerBrowse
	p.rename.Blur()
	p.rename.Reset()
	if selectRunID != "" {
		for i, it := range p.list.Items() {
			if pit, ok := it.(pickerItem); ok && pit.runID == selectRunID {
				p.list.Select(i)
				break
			}
		}
	}
	p.previewID = ""
	p.previewEntries = nil
}

// dividerLine renders a horizontal rule of the given width (floored at 1).
func dividerLine(width int) string {
	return strings.Repeat("─", maxInt(width, 1))
}

// truncateLine clips a plain (ANSI-free) line to width display columns, appending an ellipsis when
// it overflows. Preview text is raw message content, so a rune/width walk is sufficient.
func truncateLine(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= width {
		return s
	}
	if width == 1 {
		return "…"
	}
	var b strings.Builder
	w := 0
	for _, c := range s {
		cw := lipgloss.Width(string(c))
		if w+cw > width-1 {
			break
		}
		b.WriteRune(c)
		w += cw
	}
	b.WriteRune('…')
	return b.String()
}

// maxInt returns the larger of two ints.
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

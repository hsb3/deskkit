// Conversation lifecycle for the chat surface's sessions manager: rename, delete, and preview a
// prior manual run. These are the store-side helpers behind the picker's rename/delete/preview
// intents (the TUI's model owns the calls; the picker stays a pure view component).
//
// Delete is a HARD delete of the agent_runs row. The record-original-first boundary that governs
// desk-FILE fixes (byte-reversible via `restore`) does NOT extend to the user's own chat history:
// a conversation is the user's to discard, so deleting it outright is correct and is not a
// boundary violation. The messages.run relation carries CascadeDelete (collections 0007), so the
// run's messages are removed with it — no separate cleanup and no new migration.
package agent

import (
	"errors"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// errEmptyTitle rejects a blank rename. An empty input_summary would drop the run from
// ListConversations' offers entirely (that filter is how never-turned runs are excluded), so the
// rename must never clear it.
var errEmptyTitle = errors.New("conversation title must not be empty")

// RenameConversation sets a run's title (its input_summary, which the picker lists). A blank or
// whitespace-only title is rejected — see errEmptyTitle. The title is stored verbatim (trimmed),
// not re-summarized, because it is the user's chosen label, not a derived summary of an input.
func RenameConversation(app core.App, runID, title string) error {
	trimmed := strings.TrimSpace(title)
	if trimmed == "" {
		return errEmptyTitle
	}
	run, err := app.FindRecordById("agent_runs", runID)
	if err != nil {
		return err
	}
	run.Set("input_summary", trimmed)
	return app.Save(run)
}

// DeleteConversation hard-deletes a run row. The messages.run relation's CascadeDelete removes the
// run's messages automatically (see the file header for why a hard delete is the right semantics
// here). Deleting an already-absent run surfaces the not-found error from FindRecordById.
func DeleteConversation(app core.App, runID string) error {
	run, err := app.FindRecordById("agent_runs", runID)
	if err != nil {
		return err
	}
	return app.Delete(run)
}

// SetConversationArchived sets a run's archived flag. Archiving is a SOFT, reversible hide from the
// default resume list (ListConversations excludes archived runs unless asked to include them); it
// never removes the run or its messages — the opposite of DeleteConversation's hard cascade. A chat
// conversation is the user's own history to organize, so toggling this flag is the user's to make
// and is not a record-original-first boundary concern (see the file header). Toggling an
// already-absent run surfaces the not-found error from FindRecordById.
func SetConversationArchived(app core.App, runID string, archived bool) error {
	run, err := app.FindRecordById("agent_runs", runID)
	if err != nil {
		return err
	}
	run.Set("archived", archived)
	return app.Save(run)
}

// PreviewConversation returns the last maxRows non-system message rows of a run, in seq order,
// rendered as TranscriptEntry — the recent-transcript preview the picker shows before a reader
// commits to resuming. It reuses the same row-rendering path as the resume transcript, so the
// preview reads identically to what a resume would restore. A non-positive maxRows defaults to 20.
func PreviewConversation(app core.App, runID string, maxRows int) ([]TranscriptEntry, error) {
	if maxRows <= 0 {
		maxRows = 20
	}
	// Load in seq order (the limit far exceeds any realistic run length), then keep the most
	// recent maxRows non-system rows — mirroring ResumeSession's load so the two never diverge.
	rows, err := app.FindRecordsByFilter("messages", "run = {:r}", "seq", 100000, 0, dbx.Params{"r": runID})
	if err != nil {
		return nil, err
	}
	return renderTranscriptRows(nonSystemRows(rows), maxRows), nil
}

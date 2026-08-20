package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pocketbase/pocketbase/core"

	"github.com/hsb3/deskkit/internal/core/config"
	"github.com/hsb3/deskkit/internal/modules/librarian/desklib"
)

// DeleteDocInput removes ONE existing desk document. It is the write path's sibling and
// deliberately carries the same two fields: a desk-relative path and the checksum the caller's
// copy was loaded with. The checksum is not optional — a delete is the least reversible thing a
// surface can ask for, so it gets the strictest version of the compare-and-swap, not a looser
// one. An outside edit between load and delete means the operator is deciding about a document
// that no longer exists in the form they read.
type DeleteDocInput struct {
	Path         string `json:"path"`
	BaseChecksum string `json:"base_checksum"`
}

// DeleteDocResult mirrors WriteDocResult, including the conflict payload, so a surface can
// handle both verbs' 409 with one branch.
type DeleteDocResult struct {
	Path       string `json:"path"`
	Outcome    string `json:"outcome"` // deleted | conflict
	RevisionID string `json:"revision_id,omitempty"`
	// Conflict payload: the disk's current state, so the surface shows what changed.
	CurrentChecksum string `json:"current_checksum,omitempty"`
	CurrentContent  string `json:"current_content,omitempty"`
}

// DeleteDoc removes a desk document reversibly. It is the same shape as WriteDoc — same path
// cleaning, same ignore boundary, same Lstat-not-Stat symlink refusal, same compare-and-swap,
// same record-original-first boundary — because the browser reaches both, and a delete that
// enforced anything less than a write would be the softer door into the same desk.
//
// Reversibility is the whole point: the original bytes and their checksum land in the
// `revisions` ledger with action "delete" and applied=true BEFORE os.Remove runs, so
// `deskkit restore --by-path <path>` resolves it (restore's by-path query matches applied,
// unrestored revisions) and writes the exact bytes back. Restore's own action handling needs no
// delete branch: it removes a moved file only for action "move", then writes the original at
// `path` — which for a deleted file is precisely re-creating it.
//
// Ordering under a crash: ledger, then disk, then the row. A crash after the ledger write but
// before os.Remove leaves an applied revision for a file that is still present — restore then
// rewrites identical bytes, which is a no-op rather than damage. The reverse order could remove
// a file with no record of its contents, which is unrecoverable. Fail toward the recoverable
// side.
func DeleteDoc(ctx context.Context, app core.App, cfg *config.Config, in *DeleteDocInput) (*DeleteDocResult, error) {
	rel, err := cleanDeskRel(in.Path)
	if err != nil {
		return nil, fmt.Errorf("delete_doc: %w", err)
	}
	if in.BaseChecksum == "" {
		return nil, fmt.Errorf("delete_doc: base_checksum is required")
	}

	// Ignore boundary, fail closed exactly like write_doc/apply_fix: an unreadable list or a
	// protected path refuses the delete outright.
	ignoreList, ignoreErr := desklib.LoadIgnoreList(cfg.IgnoreConfig)
	if ignoreErr != nil {
		return nil, fmt.Errorf("delete_doc: ignore list unreadable, refusing to delete: %w", ignoreErr)
	}
	if desklib.IsIgnored(rel, ignoreList) {
		return nil, fmt.Errorf("delete_doc: %s is write-protected (.librarian-ignore)", rel)
	}

	// Existing regular file only — Lstat so a symlink pointing out of the desk is refused
	// rather than followed, and a directory is never removed by a document verb.
	abs := filepath.Join(cfg.DeskRoot, filepath.FromSlash(rel))
	fi, statErr := os.Lstat(abs)
	if statErr != nil {
		return nil, fmt.Errorf("delete_doc: %s: %w", rel, statErr)
	}
	if !fi.Mode().IsRegular() {
		return nil, fmt.Errorf("delete_doc: %s is not a regular file", rel)
	}

	current, readErr := os.ReadFile(abs)
	if readErr != nil {
		return nil, fmt.Errorf("delete_doc: read %s: %w", rel, readErr)
	}
	currentSum := desklib.Checksum(current)
	if currentSum != in.BaseChecksum {
		return &DeleteDocResult{
			Path: rel, Outcome: "conflict",
			CurrentChecksum: currentSum, CurrentContent: string(current),
		}, nil
	}

	// RECORD ORIGINAL FIRST (Boundary 1): no filesystem removal may follow a failed
	// original-record. Unlike write_doc, `applied` is set true up front — the ledger row is
	// the ONLY copy of the file's bytes once os.Remove runs, and restore --by-path resolves
	// only applied revisions, so a row left applied=false after a successful removal would be
	// an unrecoverable file with its contents sitting one boolean away from reach. write_doc
	// can afford the two-phase flag because a failed flip there still leaves the file readable.
	revCol, err := app.FindCollectionByNameOrId("revisions")
	if err != nil {
		return nil, fmt.Errorf("delete_doc: revisions collection: %w", err)
	}
	rev := core.NewRecord(revCol)
	rev.Set("path", rel)
	rev.Set("action", "delete")
	rev.Set("original_content", string(current))
	rev.Set("original_checksum", currentSum)
	rev.Set("applied", true)
	rev.Set("restored", false)
	rev.Set("run_id", "delete_doc")
	if err := app.Save(rev); err != nil {
		return nil, fmt.Errorf("delete_doc: record original for %s: %w", rel, err)
	}

	if err := os.Remove(abs); err != nil {
		return nil, fmt.Errorf("delete_doc: remove %s: %w", rel, err)
	}

	// Post-removal bookkeeping: soft-delete the row. The filesystem is already committed and
	// is not transactional, so a failure here leaves the disk right and the row lagging — log
	// loudly, the same shape write_doc uses, and the next sweep converges it (sweep
	// soft-deletes rows whose file it no longer finds).
	if err := deindexFile(app, rel); err != nil {
		app.Logger().Warn("delete_doc: file removed but row soft-delete failed; next sweep converges",
			"path", rel, "revision", rev.Id, "err", err)
		return nil, fmt.Errorf("delete_doc: post-delete bookkeeping for %s: %w", rel, err)
	}

	return &DeleteDocResult{Path: rel, Outcome: "deleted", RevisionID: rev.Id}, nil
}

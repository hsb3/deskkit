package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"github.com/example/pocket-librarian/internal/core/config"
	"github.com/example/pocket-librarian/internal/modules/librarian/desklib"
)

// Restore — §5.5: reverse a change to the exact recorded original. Supports --by-path
// resolution (latest applied, unrestored revision whose path/new_path matches, with a
// filesystem-confirmed fallback for the §5.4 half-applied-move crash window), verifies
// sha256(original_content) == original_checksum before writing, restores byte-exact, and
// reopens the finding to flagged. CLI/supervised only (never in the autonomous agent set).
func Restore(ctx context.Context, app core.App, cfg *config.Config, in *RestoreInput) (*RestoreResult, error) {
	revID := in.RevisionID

	// 0. By-path resolution (only when RevisionID is empty and Path is set).
	if revID == "" {
		if in.Path == "" {
			return nil, fmt.Errorf("restore: either revision_id or path must be set")
		}
		resolved, err := resolveRevisionByPath(app, cfg, in.Path)
		if err != nil {
			return nil, err
		}
		revID = resolved
	}

	rev, err := app.FindRecordById("revisions", revID)
	if err != nil {
		return nil, fmt.Errorf("restore: revision %s not found: %w", revID, err)
	}

	// 1. Guards. The file is untouched on any error here.
	//   - already restored → always a hard error.
	//   - never applied → normally a hard error, EXCEPT the half-applied-move crash window
	//     (§5.4): apply_fix's os.Rename committed but the applied=true DB patch never landed.
	//     When the filesystem confirms exactly that state (move only — a half-applied edit
	//     leaves the file in place, indistinguishable from a concurrent user edit), catch the
	//     applied flag up (below, inside the restore transaction) and reverse the move. Any
	//     other unconfirmed not-applied state errors loudly — never guess.
	if rev.GetBool("restored") {
		return nil, fmt.Errorf("restore: revision %s already restored", revID)
	}
	halfApplied := false
	if !rev.GetBool("applied") {
		if !confirmHalfApplied(cfg, rev) {
			return nil, fmt.Errorf("restore: revision %s was never applied and the filesystem does not confirm a half-applied move (expected the file absent at %q and present at %q) — refusing to guess", revID, rev.GetString("path"), rev.GetString("new_path"))
		}
		halfApplied = true
	}

	// 2. Verify the stored original against its checksum BEFORE writing anything.
	original := rev.GetString("original_content")
	originalChecksum := rev.GetString("original_checksum")
	if desklib.Checksum([]byte(original)) != originalChecksum {
		return nil, fmt.Errorf("restore: refusing to restore — stored original for revision %s does not match its checksum", revID)
	}

	path := rev.GetString("path")
	action := rev.GetString("action")
	newPath := rev.GetString("new_path")
	absOriginal := filepath.Join(cfg.DeskRoot, path)

	// 3. If the action was a move and the moved file exists at new_path, remove it.
	if action == "move" && newPath != "" {
		movedAbs := filepath.Join(cfg.DeskRoot, newPath)
		if fi, statErr := os.Stat(movedAbs); statErr == nil && fi.Mode().IsRegular() {
			if rmErr := os.Remove(movedAbs); rmErr != nil {
				return nil, fmt.Errorf("restore: remove moved file %s: %w", newPath, rmErr)
			}
		}
	}

	// 4. Write the exact original bytes back at the original path.
	if err := desklib.WriteExact(absOriginal, []byte(original)); err != nil {
		return nil, fmt.Errorf("restore: write original for %s: %w", path, err)
	}

	result := &RestoreResult{RevisionID: revID, Path: path, Restored: true}

	// 5. Patch revisions.restored + reopen the finding, atomically.
	findingID := rev.GetString("finding")
	txErr := app.RunInTransaction(func(txApp core.App) error {
		if halfApplied {
			// DB catch-up the crashed apply_fix transaction never did: the move committed to
			// the filesystem, so the row must read applied before it can read restored.
			rev.Set("applied", true)
		}
		rev.Set("restored", true)
		if err := txApp.Save(rev); err != nil {
			return err
		}
		if findingID != "" {
			finding, ferr := txApp.FindRecordById("patrol_findings", findingID)
			if ferr != nil {
				return ferr
			}
			finding.Set("state", "flagged")
			if err := txApp.Save(finding); err != nil {
				return err
			}
			result.Reopened = true
		}
		return nil
	})
	if txErr != nil {
		return nil, fmt.Errorf("restore: patch after write for revision %s: %w", revID, txErr)
	}

	return result, nil
}

// resolveRevisionByPath resolves --by-path to the latest applied, not-yet-restored
// revision whose path or new_path matches (spec §5.5 step 0). If no applied row matches, it
// falls back to the §5.4 half-applied-move recovery: the newest applied=false, unrestored
// move whose crash window the filesystem confirms (rename done, applied patch never landed).
// Restore's step 1 re-runs confirmHalfApplied and catches the applied flag up before reversing.
func resolveRevisionByPath(app core.App, cfg *config.Config, path string) (string, error) {
	filter := "applied = true && restored = false && (path = {:path} || new_path = {:path})"
	recs, err := app.FindRecordsByFilter("revisions", filter, "-created", 1, 0, dbx.Params{"path": path})
	if err != nil {
		return "", fmt.Errorf("restore: resolve by-path %s: %w", path, err)
	}
	if len(recs) > 0 {
		return recs[0].Id, nil
	}

	// Fallback (§5.4): scan applied=false, unrestored candidates newest-first for one whose
	// filesystem state confirms a half-applied move.
	fbFilter := "applied = false && restored = false && (path = {:path} || new_path = {:path})"
	fbRecs, err := app.FindRecordsByFilter("revisions", fbFilter, "-created", 0, 0, dbx.Params{"path": path})
	if err != nil {
		return "", fmt.Errorf("restore: resolve by-path (half-applied) %s: %w", path, err)
	}
	for _, r := range fbRecs {
		if confirmHalfApplied(cfg, r) {
			return r.Id, nil
		}
	}
	if len(fbRecs) > 0 {
		return "", fmt.Errorf("restore: no applied, unrestored revision for path %s; found %d unapplied revision(s) but none match a filesystem-confirmed half-applied move", path, len(fbRecs))
	}
	return "", fmt.Errorf("restore: no applied, unrestored revision for path %s", path)
}

// confirmHalfApplied reports whether an applied=false revision is in the §5.4
// half-applied-move crash window: apply_fix's os.Rename committed (the file is gone from
// `path` and present at `new_path`) but the `applied=true` DB patch never landed. Only a
// "move" produces an FS-confirmable window — a half-applied "edit" leaves the file in place
// at `path` and is indistinguishable from a concurrent user edit, so it is never confirmed
// here. Any non-move action, missing `new_path`, a file still present at `path`, an absent
// `new_path`, or an unexpected stat error yields false (the caller then errors loudly rather
// than guessing). It reads the filesystem only — it never mutates DB or disk.
func confirmHalfApplied(cfg *config.Config, rev *core.Record) bool {
	if rev.GetString("action") != "move" {
		return false
	}
	newPath := rev.GetString("new_path")
	if newPath == "" {
		return false
	}
	// The original path must be absent (the rename removed it and no stub was written).
	absPath := filepath.Join(cfg.DeskRoot, rev.GetString("path"))
	if _, err := os.Stat(absPath); !os.IsNotExist(err) {
		return false
	}
	// The moved file must be present at new_path as a regular file.
	fi, err := os.Stat(filepath.Join(cfg.DeskRoot, newPath))
	return err == nil && fi.Mode().IsRegular()
}

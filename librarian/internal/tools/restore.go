package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"github.com/example/pocket-librarian/internal/config"
	"github.com/example/pocket-librarian/internal/desklib"
)

// Restore — §5.5: reverse a change to the exact recorded original. Supports --by-path
// resolution (latest applied, unrestored revision whose path/new_path matches), verifies
// sha256(original_content) == original_checksum before writing, restores byte-exact, and
// reopens the finding to flagged. CLI/supervised only (never in the autonomous agent set).
func Restore(ctx context.Context, app core.App, cfg *config.Config, in *RestoreInput) (*RestoreResult, error) {
	revID := in.RevisionID

	// 0. By-path resolution (only when RevisionID is empty and Path is set).
	if revID == "" {
		if in.Path == "" {
			return nil, fmt.Errorf("restore: either revision_id or path must be set")
		}
		resolved, err := resolveRevisionByPath(app, in.Path)
		if err != nil {
			return nil, err
		}
		revID = resolved
	}

	rev, err := app.FindRecordById("revisions", revID)
	if err != nil {
		return nil, fmt.Errorf("restore: revision %s not found: %w", revID, err)
	}

	// 1. Hard errors: not applied, or already restored. File is untouched on any of these.
	if !rev.GetBool("applied") {
		return nil, fmt.Errorf("restore: revision %s was never applied", revID)
	}
	if rev.GetBool("restored") {
		return nil, fmt.Errorf("restore: revision %s already restored", revID)
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
// revision whose path or new_path matches (spec §5.5 step 0).
func resolveRevisionByPath(app core.App, path string) (string, error) {
	filter := "applied = true && restored = false && (path = {:path} || new_path = {:path})"
	recs, err := app.FindRecordsByFilter("revisions", filter, "-created", 1, 0, dbx.Params{"path": path})
	if err != nil {
		return "", fmt.Errorf("restore: resolve by-path %s: %w", path, err)
	}
	if len(recs) == 0 {
		return "", fmt.Errorf("restore: no applied, unrestored revision for path %s", path)
	}
	return recs[0].Id, nil
}

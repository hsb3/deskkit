package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"github.com/hsb3/deskkit/internal/core/config"
	"github.com/hsb3/deskkit/internal/modules/librarian/desklib"
)

// WriteDoc is the write-through path (SPA overhaul phase 0, implementing the
// files-stay-authoritative ruling, decision 0009): ONE
// operation that writes a desk document to disk and updates its files row together,
// recording the original first so `restore --by-path` reverses it byte-exact.
//
// It is deliberately NOT a toolcore spec: the autonomous agent's write boundary stays
// "apply_fix only, gated" — this path exists for the human-driven surfaces (the CLI
// subcommand and the SPA's /desk/doc/write route), which are supervised by the person
// pressing save (owner-ruled: a human save IS the supervised path; the autonomous-writes
// flag gates unattended agents, not people).
//
// Concurrency contract: compare-and-swap on the file's on-disk checksum. The caller
// supplies the checksum its copy was loaded with; if the disk has moved on (an outside
// edit — the NORMAL case, the owner writes in Obsidian), the write is refused and the
// current disk state returned so the surface can show the difference. Never overwrite,
// never silently merge; overwriting is an explicit re-submit with the fresh checksum.
type WriteDocInput struct {
	// Path is the desk-relative path of an EXISTING document (creation is a later phase).
	Path string `json:"path"`
	// BaseChecksum is the sha256 the caller's copy was loaded with (files.checksum).
	BaseChecksum string `json:"base_checksum"`
	// Content replaces the whole document (full-content mode). Exactly one of
	// Content / Set must be given.
	Content string `json:"content,omitempty"`
	// Set edits top-level frontmatter fields server-side, preserving every other byte
	// (field mode — the browser never rewrites YAML).
	Set map[string]string `json:"set,omitempty"`
}

type WriteDocResult struct {
	Path       string `json:"path"`
	Outcome    string `json:"outcome"` // written | noop | conflict
	Checksum   string `json:"checksum,omitempty"`
	RevisionID string `json:"revision_id,omitempty"`
	// Conflict payload: the disk's current state, so the surface shows the difference.
	CurrentChecksum string `json:"current_checksum,omitempty"`
	CurrentContent  string `json:"current_content,omitempty"`
}

func WriteDoc(ctx context.Context, app core.App, cfg *config.Config, in *WriteDocInput) (*WriteDocResult, error) {
	rel, err := cleanDeskRel(in.Path)
	if err != nil {
		return nil, fmt.Errorf("write_doc: %w", err)
	}
	if (in.Content == "") == (len(in.Set) == 0) {
		return nil, fmt.Errorf("write_doc: exactly one of content / set must be given")
	}
	if in.BaseChecksum == "" {
		return nil, fmt.Errorf("write_doc: base_checksum is required")
	}

	// Ignore boundary, fail closed exactly like apply_fix (§10.1): unreadable list or a
	// protected path refuses the write outright.
	ignoreList, ignoreErr := desklib.LoadIgnoreList(cfg.IgnoreConfig)
	if ignoreErr != nil {
		return nil, fmt.Errorf("write_doc: ignore list unreadable, refusing to write: %w", ignoreErr)
	}
	if desklib.IsIgnored(rel, ignoreList) {
		return nil, fmt.Errorf("write_doc: %s is write-protected (.librarian-ignore)", rel)
	}

	// Existing regular file only — Lstat so a symlink pointing out of the desk is refused,
	// not followed (this path is reachable from the browser).
	abs := filepath.Join(cfg.DeskRoot, filepath.FromSlash(rel))
	fi, statErr := os.Lstat(abs)
	if statErr != nil {
		return nil, fmt.Errorf("write_doc: %s: %w", rel, statErr)
	}
	if !fi.Mode().IsRegular() {
		return nil, fmt.Errorf("write_doc: %s is not a regular file", rel)
	}

	current, readErr := os.ReadFile(abs)
	if readErr != nil {
		return nil, fmt.Errorf("write_doc: read %s: %w", rel, readErr)
	}
	currentSum := desklib.Checksum(current)
	if currentSum != in.BaseChecksum {
		return &WriteDocResult{
			Path: rel, Outcome: "conflict",
			CurrentChecksum: currentSum, CurrentContent: string(current),
		}, nil
	}

	next := []byte(in.Content)
	if len(in.Set) > 0 {
		next = current
		keys := make([]string, 0, len(in.Set))
		for k := range in.Set {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			next, err = desklib.SetFrontmatterField(next, k, in.Set[k])
			if err != nil {
				return nil, fmt.Errorf("write_doc: %s: %w", rel, err)
			}
		}
	}
	if string(next) == string(current) {
		return &WriteDocResult{Path: rel, Outcome: "noop", Checksum: currentSum}, nil
	}

	// RECORD ORIGINAL FIRST (Boundary 1): no filesystem write may follow a failed
	// original-record. applied flips true only after the write lands, mirroring
	// propose_fix/apply_fix, so restore's guards read this row exactly like theirs.
	revCol, err := app.FindCollectionByNameOrId("revisions")
	if err != nil {
		return nil, fmt.Errorf("write_doc: revisions collection: %w", err)
	}
	rev := core.NewRecord(revCol)
	rev.Set("path", rel)
	rev.Set("action", "edit")
	rev.Set("original_content", string(current))
	rev.Set("original_checksum", currentSum)
	rev.Set("applied", false)
	rev.Set("restored", false)
	rev.Set("run_id", "write_doc")
	if err := app.Save(rev); err != nil {
		return nil, fmt.Errorf("write_doc: record original for %s: %w", rel, err)
	}

	if err := desklib.WriteExact(abs, next); err != nil {
		return nil, fmt.Errorf("write_doc: write %s: %w", rel, err)
	}

	// Post-write bookkeeping: mark the revision applied and re-index the file's row in one
	// transaction. The FS write is already committed (the filesystem is not transactional);
	// on failure here the disk is right and the row/revision lag — log loudly, same shape
	// as apply_fix's half-applied warning, and the next sweep converges the row.
	txErr := app.RunInTransaction(func(txApp core.App) error {
		rev.Set("applied", true)
		if err := txApp.Save(rev); err != nil {
			return err
		}
		return indexOneFile(txApp, cfg, rel)
	})
	if txErr != nil {
		app.Logger().Warn("write_doc: disk written but row/revision patch failed; next sweep converges",
			"path", rel, "revision", rev.Id, "err", txErr)
		return nil, fmt.Errorf("write_doc: post-write bookkeeping for %s: %w", rel, txErr)
	}

	return &WriteDocResult{
		Path: rel, Outcome: "written",
		Checksum: desklib.Checksum(next), RevisionID: rev.Id,
	}, nil
}

// cleanDeskRel normalizes a caller-supplied desk-relative path and refuses anything that
// could escape the desk root (absolute, empty, or ../ after cleaning).
func cleanDeskRel(p string) (string, error) {
	if strings.TrimSpace(p) == "" {
		return "", fmt.Errorf("path is required")
	}
	rel := filepath.ToSlash(filepath.Clean(filepath.FromSlash(p)))
	if filepath.IsAbs(filepath.FromSlash(rel)) || rel == "." || rel == ".." || strings.HasPrefix(rel, "../") {
		return "", fmt.Errorf("path %q escapes the desk root", p)
	}
	return rel, nil
}

// indexOneFile upserts the files row for one desk file from its current on-disk state —
// the single-file equivalent of one Sweep loop iteration, shared by write_doc and the
// watcher. Identity matching mirrors Sweep: doc_id first (rename survival), path fallback.
func indexOneFile(app core.App, cfg *config.Config, rel string) error {
	row, err := scanFile(cfg.DeskRoot, rel, cfg.EntityDirMap(), cfg.SecretsDir, cfg.DeskName)
	if err != nil {
		return err
	}
	col, err := app.FindCollectionByNameOrId("files")
	if err != nil {
		return err
	}

	var old *core.Record
	if row.DocID != "" {
		byID, idErr := app.FindRecordsByFilter("files", "doc_id = {:id}", "+created", 1, 0, dbx.Params{"id": row.DocID})
		if idErr == nil && len(byID) > 0 {
			old = byID[0]
		}
	}
	if old == nil {
		byPath, pErr := app.FindRecordsByFilter("files", "path = {:path}", "", 1, 0, dbx.Params{"path": rel})
		if pErr == nil && len(byPath) > 0 {
			old = byPath[0]
		}
	}

	now := time.Now().UTC()
	if old == nil {
		rec := core.NewRecord(col)
		applyFileRow(rec, row)
		rec.Set("last_seen", now)
		return app.Save(rec)
	}
	if old.GetString("path") != row.Path || fileRowDiffers(old, row) {
		applyFileRow(old, row)
		old.Set("last_seen", now)
		return app.Save(old)
	}
	return nil
}

// deindexFile soft-deletes the files row at rel, if one exists — the watcher's remove
// path. A rename shows up as remove(old)+create(new); when the doc carries a frontmatter
// id the create leg has already MOVED the row to the new path (doc_id match), so the
// remove leg finds nothing at the old path and is a no-op, exactly like Sweep's
// record-identity soft-delete.
func deindexFile(app core.App, rel string) error {
	recs, err := app.FindRecordsByFilter("files", "path = {:path} && deleted = false", "", 1, 0, dbx.Params{"path": rel})
	if err != nil || len(recs) == 0 {
		return err
	}
	recs[0].Set("deleted", true)
	return app.Save(recs[0])
}

package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pocketbase/dbx"

	"github.com/hsb3/deskkit/internal/modules/librarian/desklib"
)

const ddDoc = `---
type: guide
status: draft
---

# A doc to remove

body with trailing newline
`

// TestDeleteDoc_RemovesAndRestoreReversesByteExact is the contract the whole delete verb rests
// on: the file leaves the desk, its row soft-deletes, and `restore --by-path` puts the exact
// original bytes back. A delete that restore cannot reverse is not a delete, it is data loss.
func TestDeleteDoc_RemovesAndRestoreReversesByteExact(t *testing.T) {
	app, cfg := newTestEnv(t)
	abs := mustWriteFile(t, cfg.DeskRoot, "notes/doomed.md", ddDoc)
	if _, err := Sweep(context.Background(), app, cfg, &SweepInput{}); err != nil {
		t.Fatalf("sweep: %v", err)
	}

	base := desklib.Checksum([]byte(ddDoc))
	res, err := DeleteDoc(context.Background(), app, cfg, &DeleteDocInput{
		Path: "notes/doomed.md", BaseChecksum: base,
	})
	if err != nil {
		t.Fatalf("DeleteDoc: %v", err)
	}
	if res.Outcome != "deleted" {
		t.Fatalf("outcome = %q, want deleted (%+v)", res.Outcome, res)
	}
	if res.RevisionID == "" {
		t.Fatal("delete recorded no revision — nothing to restore from")
	}
	if res.Path != "notes/doomed.md" {
		t.Errorf("result path = %q, want the desk-relative path", res.Path)
	}

	if _, statErr := os.Stat(abs); !os.IsNotExist(statErr) {
		t.Fatalf("file still on disk after delete (stat err = %v)", statErr)
	}

	// The row soft-deletes rather than vanishing: the store keeps the record of what was there,
	// which is what makes the removal an event with provenance instead of an absence.
	rows, err := app.FindRecordsByFilter("files", "path = {:p}", "", 1, 0, dbx.Params{"p": "notes/doomed.md"})
	if err != nil || len(rows) == 0 {
		t.Fatalf("files row gone entirely (err %v) — delete must SOFT-delete", err)
	}
	if !rows[0].GetBool("deleted") {
		t.Error("files row not marked deleted")
	}

	// The recorded original must verify against its own checksum, or restore refuses it.
	rev, err := app.FindRecordById("revisions", res.RevisionID)
	if err != nil {
		t.Fatalf("find revision %s: %v", res.RevisionID, err)
	}
	if rev.GetString("action") != "delete" {
		t.Errorf("revision action = %q, want delete", rev.GetString("action"))
	}
	if rev.GetString("original_content") != ddDoc {
		t.Errorf("recorded original is not the file's bytes: %q", rev.GetString("original_content"))
	}
	if rev.GetString("original_checksum") != base {
		t.Errorf("recorded checksum %q does not match the original bytes", rev.GetString("original_checksum"))
	}
	if !rev.GetBool("applied") {
		t.Error("revision not marked applied — restore --by-path resolves only applied revisions")
	}

	// The reversal, end to end, through the SAME command an operator types.
	rres, err := Restore(context.Background(), app, cfg, &RestoreInput{Path: "notes/doomed.md"})
	if err != nil {
		t.Fatalf("Restore by-path after delete: %v", err)
	}
	if !rres.Restored {
		t.Fatalf("restore did not restore: %+v", rres)
	}
	back, err := os.ReadFile(abs)
	if err != nil {
		t.Fatalf("file did not come back: %v", err)
	}
	if string(back) != ddDoc {
		t.Errorf("restore not byte-exact:\n got %q\nwant %q", back, ddDoc)
	}
	if desklib.Checksum(back) != base {
		t.Errorf("restored checksum %q != original %q", desklib.Checksum(back), base)
	}
}

// TestDeleteDoc_ConflictRefusesAndKeepsFile: the same compare-and-swap the write path uses. An
// outside edit between the browser loading the doc and pressing delete means the operator is
// looking at a different document than the one on disk — deleting it anyway would destroy an
// edit nobody reviewed.
func TestDeleteDoc_ConflictRefusesAndKeepsFile(t *testing.T) {
	app, cfg := newTestEnv(t)
	abs := mustWriteFile(t, cfg.DeskRoot, "notes/doomed.md", ddDoc)
	if _, err := Sweep(context.Background(), app, cfg, &SweepInput{}); err != nil {
		t.Fatalf("sweep: %v", err)
	}
	stale := desklib.Checksum([]byte(ddDoc))

	// Somebody edits the file in their editor after the browser loaded it.
	moved := strings.Replace(ddDoc, "status: draft", "status: approved", 1)
	if err := os.WriteFile(abs, []byte(moved), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := DeleteDoc(context.Background(), app, cfg, &DeleteDocInput{
		Path: "notes/doomed.md", BaseChecksum: stale,
	})
	if err != nil {
		t.Fatalf("DeleteDoc: %v", err)
	}
	if res.Outcome != "conflict" {
		t.Fatalf("outcome = %q, want conflict", res.Outcome)
	}
	if res.CurrentContent != moved || res.CurrentChecksum != desklib.Checksum([]byte(moved)) {
		t.Errorf("conflict payload does not carry the disk's current state: %+v", res)
	}
	got, err := os.ReadFile(abs)
	if err != nil {
		t.Fatalf("the refused delete removed the file anyway: %v", err)
	}
	if string(got) != moved {
		t.Errorf("file changed under a refused delete: %q", got)
	}
	// A refused delete must leave no revision behind: a ledger entry for an operation that
	// never happened would let restore resurrect a file over a newer one.
	revs, _ := app.FindRecordsByFilter("revisions", "path = {:p}", "", 0, 0, dbx.Params{"p": "notes/doomed.md"})
	if len(revs) != 0 {
		t.Errorf("refused delete recorded %d revision(s)", len(revs))
	}
}

// TestDeleteDoc_RefusesOutsideItsBoundary: the guards inherited from the write path. Each of
// these reaches the tool straight from the browser, so each must be refused BEFORE anything
// leaves the disk.
func TestDeleteDoc_RefusesOutsideItsBoundary(t *testing.T) {
	app, cfg := newTestEnv(t)
	mustWriteFile(t, cfg.DeskRoot, "notes/doomed.md", ddDoc)
	protectedAbs := mustWriteFile(t, cfg.DeskRoot, "notes/protected.md", ddDoc)
	if err := os.WriteFile(cfg.IgnoreConfig, []byte("notes/protected.md\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	base := desklib.Checksum([]byte(ddDoc))

	for name, in := range map[string]*DeleteDocInput{
		"escapes the desk root": {Path: "../outside.md", BaseChecksum: base},
		"empty path":            {Path: "", BaseChecksum: base},
		"no base checksum":      {Path: "notes/doomed.md"},
		"absent file":           {Path: "notes/never-existed.md", BaseChecksum: base},
		"write-protected path":  {Path: "notes/protected.md", BaseChecksum: base},
		"a directory":           {Path: "notes", BaseChecksum: base},
	} {
		if _, err := DeleteDoc(context.Background(), app, cfg, in); err == nil {
			t.Errorf("%s: DeleteDoc returned no error", name)
		}
	}

	// The protected file is still there — the ignore boundary held.
	if _, err := os.Stat(protectedAbs); err != nil {
		t.Errorf("a write-protected file was deleted: %v", err)
	}
	// And so is the sibling the escaping paths might have resolved onto.
	if _, err := os.Stat(filepath.Join(cfg.DeskRoot, "notes", "doomed.md")); err != nil {
		t.Errorf("a refused delete removed the wrong file: %v", err)
	}
}

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

const wdDoc = `---
type: guide
status: draft
---

# A doc

body
`

// TestWriteDoc_RoundTripAndRestore is the phase-0 core contract: one
// field edit lands on disk through the shared path, the row updates with it, and
// `restore --by-path` puts the file back byte-for-byte.
func TestWriteDoc_RoundTripAndRestore(t *testing.T) {
	app, cfg := newTestEnv(t)
	abs := mustWriteFile(t, cfg.DeskRoot, "notes/doc.md", wdDoc)
	if _, err := Sweep(context.Background(), app, cfg, &SweepInput{}); err != nil {
		t.Fatalf("sweep: %v", err)
	}

	base := desklib.Checksum([]byte(wdDoc))
	res, err := WriteDoc(context.Background(), app, cfg, &WriteDocInput{
		Path: "notes/doc.md", BaseChecksum: base, Set: map[string]string{"status": "active"},
	})
	if err != nil {
		t.Fatalf("WriteDoc: %v", err)
	}
	if res.Outcome != "written" || res.RevisionID == "" {
		t.Fatalf("want written+revision, got %+v", res)
	}

	// Disk carries the edit, only the edit.
	got, _ := os.ReadFile(abs)
	want := strings.Replace(wdDoc, "status: draft", "status: active", 1)
	if string(got) != want {
		t.Errorf("disk content:\n got %q\nwant %q", got, want)
	}

	// The files row moved with the disk in the same operation — no sweep in between.
	rec, err := app.FindRecordsByFilter("files", "path = {:p}", "", 1, 0, dbx.Params{"p": "notes/doc.md"})
	if err != nil || len(rec) == 0 {
		t.Fatalf("files row: %v", err)
	}
	if rec[0].GetString("status") != "active" || rec[0].GetString("checksum") != res.Checksum {
		t.Errorf("row not re-indexed: status=%q checksum=%q want status=active checksum=%s",
			rec[0].GetString("status"), rec[0].GetString("checksum"), res.Checksum)
	}

	// restore --by-path reverses byte-exact.
	rres, err := Restore(context.Background(), app, cfg, &RestoreInput{Path: "notes/doc.md"})
	if err != nil {
		t.Fatalf("Restore by-path: %v", err)
	}
	if !rres.Restored {
		t.Fatalf("restore did not restore: %+v", rres)
	}
	back, _ := os.ReadFile(abs)
	if string(back) != wdDoc {
		t.Errorf("restore not byte-exact:\n got %q\nwant %q", back, wdDoc)
	}
}

// TestWriteDoc_ConflictRefusesAndShowsCurrent: an outside edit between load and save stops
// the write, returns the disk's current state, and touches nothing (the conflict rule).
func TestWriteDoc_ConflictRefusesAndShowsCurrent(t *testing.T) {
	app, cfg := newTestEnv(t)
	mustWriteFile(t, cfg.DeskRoot, "doc.md", wdDoc)

	// The caller loaded wdDoc... then Obsidian saved something newer underneath.
	outside := strings.Replace(wdDoc, "# A doc", "# Edited outside", 1)
	abs := mustWriteFile(t, cfg.DeskRoot, "doc.md", outside)

	res, err := WriteDoc(context.Background(), app, cfg, &WriteDocInput{
		Path: "doc.md", BaseChecksum: desklib.Checksum([]byte(wdDoc)),
		Set: map[string]string{"status": "active"},
	})
	if err != nil {
		t.Fatalf("WriteDoc: %v", err)
	}
	if res.Outcome != "conflict" {
		t.Fatalf("want conflict, got %+v", res)
	}
	if res.CurrentChecksum != desklib.Checksum([]byte(outside)) || res.CurrentContent != outside {
		t.Errorf("conflict payload does not show the current disk state")
	}
	got, _ := os.ReadFile(abs)
	if string(got) != outside {
		t.Errorf("conflict clobbered the file: %q", got)
	}
	revs, _ := app.FindRecordsByFilter("revisions", "path = 'doc.md'", "", 0, 0)
	if len(revs) != 0 {
		t.Errorf("conflict recorded %d revision(s); want none", len(revs))
	}
}

func TestWriteDoc_NoopAndFullContentMode(t *testing.T) {
	app, cfg := newTestEnv(t)
	abs := mustWriteFile(t, cfg.DeskRoot, "doc.md", wdDoc)
	base := desklib.Checksum([]byte(wdDoc))

	// Identical bytes: no write, no revision.
	res, err := WriteDoc(context.Background(), app, cfg, &WriteDocInput{
		Path: "doc.md", BaseChecksum: base, Content: wdDoc,
	})
	if err != nil || res.Outcome != "noop" {
		t.Fatalf("want noop, got %+v err=%v", res, err)
	}

	// Full-content mode replaces the document.
	next := wdDoc + "\nmore\n"
	res, err = WriteDoc(context.Background(), app, cfg, &WriteDocInput{
		Path: "doc.md", BaseChecksum: base, Content: next,
	})
	if err != nil || res.Outcome != "written" {
		t.Fatalf("want written, got %+v err=%v", res, err)
	}
	got, _ := os.ReadFile(abs)
	if string(got) != next {
		t.Errorf("full-content write mismatch: %q", got)
	}
}

func TestWriteDoc_Refusals(t *testing.T) {
	app, cfg := newTestEnv(t)
	mustWriteFile(t, cfg.DeskRoot, "doc.md", wdDoc)
	mustWriteFile(t, cfg.DeskRoot, "_meta/HANDOFF.md", wdDoc)
	base := desklib.Checksum([]byte(wdDoc))

	// Write-protected path (.deskkitignore).
	if err := os.WriteFile(cfg.IgnoreConfig, []byte("_meta/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := WriteDoc(context.Background(), app, cfg, &WriteDocInput{
		Path: "_meta/HANDOFF.md", BaseChecksum: base, Set: map[string]string{"status": "x"},
	}); err == nil || !strings.Contains(err.Error(), "write-protected") {
		t.Errorf("ignored path: want write-protected error, got %v", err)
	}

	// Unreadable ignore list fails closed.
	if err := os.Remove(cfg.IgnoreConfig); err != nil {
		t.Fatal(err)
	}
	if _, err := WriteDoc(context.Background(), app, cfg, &WriteDocInput{
		Path: "doc.md", BaseChecksum: base, Set: map[string]string{"status": "x"},
	}); err == nil || !strings.Contains(err.Error(), "refusing") {
		t.Errorf("missing ignore list: want fail-closed error, got %v", err)
	}
	if err := os.WriteFile(cfg.IgnoreConfig, []byte("# empty\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Traversal.
	if _, err := WriteDoc(context.Background(), app, cfg, &WriteDocInput{
		Path: "../outside.md", BaseChecksum: base, Set: map[string]string{"status": "x"},
	}); err == nil || !strings.Contains(err.Error(), "escapes") {
		t.Errorf("traversal: want escape error, got %v", err)
	}

	// Symlink refused, not followed.
	target := mustWriteFile(t, t.TempDir(), "outside.md", wdDoc)
	link := filepath.Join(cfg.DeskRoot, "link.md")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if _, err := WriteDoc(context.Background(), app, cfg, &WriteDocInput{
		Path: "link.md", BaseChecksum: base, Set: map[string]string{"status": "x"},
	}); err == nil || !strings.Contains(err.Error(), "not a regular file") {
		t.Errorf("symlink: want not-a-regular-file error, got %v", err)
	}

	// Missing file (creation is a later phase, not this one).
	if _, err := WriteDoc(context.Background(), app, cfg, &WriteDocInput{
		Path: "absent.md", BaseChecksum: base, Set: map[string]string{"status": "x"},
	}); err == nil {
		t.Error("absent file: want error, got nil")
	}

	// Both modes / neither mode.
	if _, err := WriteDoc(context.Background(), app, cfg, &WriteDocInput{
		Path: "doc.md", BaseChecksum: base, Content: "x", Set: map[string]string{"a": "b"},
	}); err == nil || !strings.Contains(err.Error(), "exactly one") {
		t.Errorf("both modes: want exactly-one error, got %v", err)
	}
	if _, err := WriteDoc(context.Background(), app, cfg, &WriteDocInput{
		Path: "doc.md", BaseChecksum: base,
	}); err == nil || !strings.Contains(err.Error(), "exactly one") {
		t.Errorf("neither mode: want exactly-one error, got %v", err)
	}
}

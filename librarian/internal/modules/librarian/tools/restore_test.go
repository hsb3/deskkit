package tools

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/pocketbase/pocketbase/core"

	"github.com/example/pocket-librarian/internal/core/config"
	"github.com/example/pocket-librarian/internal/modules/librarian/desklib"
)

func TestRestore_RoundTripByteIdentical(t *testing.T) {
	app, cfg := newTestEnv(t)
	// Mixed line endings, no trailing newline: proves write_exact/restore never do any
	// newline translation (spec §5.4/§5.5 byte-exactness).
	original := []byte("no frontmatter here\r\nwith mixed endings\nno trailing newline")
	abs := mustWriteFile(t, cfg.DeskRoot, "tasks/example.md", string(original))
	checksum := desklib.Checksum(original)
	fileRec := mustCreateFileRecord(t, app, "tasks/example.md", "tasks", "task", checksum)
	finding := mustCreateFinding(t, app, fileRec, "R1", checksum, "run-1")

	proposed, err := ProposeFix(context.Background(), app, cfg, &ProposeFixInput{RunID: "run-1"})
	if err != nil || len(proposed.Proposed) != 1 || proposed.Proposed[0].Outcome != "recorded" {
		t.Fatalf("setup ProposeFix failed: err=%v res=%+v", err, proposed)
	}
	revID := proposed.Proposed[0].RevisionID

	applied, err := ApplyFix(context.Background(), app, cfg, &ApplyFixInput{RunID: "run-1"})
	if err != nil || len(applied.Outcomes) != 1 || applied.Outcomes[0].Outcome != "applied" {
		t.Fatalf("setup ApplyFix failed: err=%v res=%+v", err, applied)
	}

	mutated, rerr := os.ReadFile(abs)
	if rerr != nil {
		t.Fatalf("read mutated file: %v", rerr)
	}
	if bytes.Equal(mutated, original) {
		t.Fatalf("sanity check failed: apply_fix should have changed the file")
	}

	restored, err := Restore(context.Background(), app, cfg, &RestoreInput{RevisionID: revID})
	if err != nil {
		t.Fatalf("Restore: %v", err)
	}
	if !restored.Restored || !restored.Reopened {
		t.Fatalf("expected Restored && Reopened, got %+v", restored)
	}

	final, rerr := os.ReadFile(abs)
	if rerr != nil {
		t.Fatalf("read restored file: %v", rerr)
	}
	if !bytes.Equal(final, original) {
		t.Fatalf("restore is not byte-identical to the original:\n got  %q\n want %q", final, original)
	}

	rev := reloadRecord(t, app, "revisions", revID)
	if !rev.GetBool("restored") {
		t.Fatalf("revision should be marked restored")
	}
	findingAfter := reloadRecord(t, app, "patrol_findings", finding.Id)
	if findingAfter.GetString("state") != "flagged" {
		t.Fatalf("finding state = %q, want flagged (reopened)", findingAfter.GetString("state"))
	}
}

func TestRestore_ChecksumMismatchRefusal(t *testing.T) {
	app, cfg := newTestEnv(t)
	currentContent := "current file contents, untouched\n"
	abs := mustWriteFile(t, cfg.DeskRoot, "tasks/example.md", currentContent)

	// A revisions row built directly with a deliberately corrupted checksum, simulating a
	// tampered/corrupted ledger row (spec §5.5 step 2).
	rev := core.NewRecord(mustCollection(t, app, "revisions"))
	rev.Set("path", "tasks/example.md")
	rev.Set("action", "edit")
	rev.Set("original_content", "this is not what the checksum below claims")
	rev.Set("original_checksum", "0000000000000000000000000000000000000000000000000000000000000000")
	rev.Set("applied", true)
	rev.Set("restored", false)
	if err := app.Save(rev); err != nil {
		t.Fatalf("save revision: %v", err)
	}

	if _, err := Restore(context.Background(), app, cfg, &RestoreInput{RevisionID: rev.Id}); err == nil {
		t.Fatalf("expected an error refusing to restore on checksum mismatch")
	}

	onDisk, rerr := os.ReadFile(abs)
	if rerr != nil {
		t.Fatalf("read file: %v", rerr)
	}
	if string(onDisk) != currentContent {
		t.Fatalf("file must be untouched on a checksum-mismatch refusal, got %q", string(onDisk))
	}

	revAfter := reloadRecord(t, app, "revisions", rev.Id)
	if revAfter.GetBool("restored") {
		t.Fatalf("revision must not be marked restored when the refusal fires")
	}
}

func TestRestore_ByPathResolvesLatestAppliedUnrestored(t *testing.T) {
	app, cfg := newTestEnv(t)
	original := "no frontmatter here\n"
	abs := mustWriteFile(t, cfg.DeskRoot, "tasks/example.md", original)
	checksum := desklib.Checksum([]byte(original))
	fileRec := mustCreateFileRecord(t, app, "tasks/example.md", "tasks", "task", checksum)
	mustCreateFinding(t, app, fileRec, "R1", checksum, "run-1")

	proposed, err := ProposeFix(context.Background(), app, cfg, &ProposeFixInput{RunID: "run-1"})
	if err != nil || len(proposed.Proposed) != 1 || proposed.Proposed[0].Outcome != "recorded" {
		t.Fatalf("setup ProposeFix failed: err=%v res=%+v", err, proposed)
	}
	if _, err := ApplyFix(context.Background(), app, cfg, &ApplyFixInput{RunID: "run-1"}); err != nil {
		t.Fatalf("setup ApplyFix failed: %v", err)
	}

	restored, err := Restore(context.Background(), app, cfg, &RestoreInput{Path: "tasks/example.md"})
	if err != nil {
		t.Fatalf("Restore --by-path: %v", err)
	}
	if !restored.Restored {
		t.Fatalf("expected Restored = true, got %+v", restored)
	}

	onDisk, _ := os.ReadFile(abs)
	if string(onDisk) != original {
		t.Fatalf("restore --by-path did not reproduce the original bytes: %q", string(onDisk))
	}
}

// setupHalfAppliedMove builds the §5.4 crash state by hand: a files row + flagged finding, a
// revisions row recorded (applied=false) exactly as propose_fix would, then the os.Rename
// performed directly so the file is gone from oldRel and present at newRel while the row's
// applied=true patch never landed. Returns the revision, its finding, and both abs paths.
func setupHalfAppliedMove(t *testing.T, app core.App, cfg *config.Config, oldRel, newRel string, original []byte) (*core.Record, *core.Record, string, string) {
	t.Helper()
	oldAbs := mustWriteFile(t, cfg.DeskRoot, oldRel, string(original))
	checksum := desklib.Checksum(original)
	fileRec := mustCreateFileRecord(t, app, oldRel, "journal", "journal", checksum)
	finding := mustCreateFinding(t, app, fileRec, "R2", checksum, "run-1")

	rev := core.NewRecord(mustCollection(t, app, "revisions"))
	rev.Set("path", oldRel)
	rev.Set("action", "move")
	rev.Set("new_path", newRel)
	rev.Set("original_content", string(original))
	rev.Set("original_checksum", checksum)
	rev.Set("finding", finding.Id)
	rev.Set("applied", false)
	rev.Set("restored", false)
	rev.Set("run_id", "run-1")
	if err := app.Save(rev); err != nil {
		t.Fatalf("save revision: %v", err)
	}

	newAbs := filepath.Join(cfg.DeskRoot, newRel)
	if err := os.MkdirAll(filepath.Dir(newAbs), 0o755); err != nil {
		t.Fatalf("mkdir new_path: %v", err)
	}
	if err := os.Rename(oldAbs, newAbs); err != nil {
		t.Fatalf("simulate move (rename): %v", err)
	}
	return rev, finding, oldAbs, newAbs
}

// assertMoveReversed checks the post-restore invariants shared by the half-applied-move
// recovery paths: original bytes back at oldAbs, new_path gone, and the row caught up to
// applied=true + restored=true with its finding reopened to flagged.
func assertMoveReversed(t *testing.T, app core.App, rev, finding *core.Record, oldAbs, newAbs string, original []byte) {
	t.Helper()
	back, rerr := os.ReadFile(oldAbs)
	if rerr != nil {
		t.Fatalf("read restored file: %v", rerr)
	}
	if !bytes.Equal(back, original) {
		t.Fatalf("restore is not byte-identical:\n got  %q\n want %q", back, original)
	}
	if _, statErr := os.Stat(newAbs); !os.IsNotExist(statErr) {
		t.Fatalf("moved file at new_path should have been removed, stat err=%v", statErr)
	}
	revAfter := reloadRecord(t, app, "revisions", rev.Id)
	if !revAfter.GetBool("applied") {
		t.Fatalf("half-applied recovery must catch applied up to true")
	}
	if !revAfter.GetBool("restored") {
		t.Fatalf("revision should be marked restored")
	}
	findingAfter := reloadRecord(t, app, "patrol_findings", finding.Id)
	if findingAfter.GetString("state") != "flagged" {
		t.Fatalf("finding state = %q, want flagged (reopened)", findingAfter.GetString("state"))
	}
}

// (i) by-path recovers a filesystem-confirmed half-applied move (§5.4).
func TestRestore_ByPathRecoversHalfAppliedMove(t *testing.T) {
	app, cfg := newTestEnv(t)
	// Mixed endings, no trailing newline: recovery must be byte-exact.
	original := []byte("moved journal note\r\nmixed endings\nno trailing newline")
	oldRel := "journal/stray-note.md"
	newRel := "journal/2026-07-16-stray-note.md"
	rev, finding, oldAbs, newAbs := setupHalfAppliedMove(t, app, cfg, oldRel, newRel, original)

	restored, err := Restore(context.Background(), app, cfg, &RestoreInput{Path: oldRel})
	if err != nil {
		t.Fatalf("Restore --by-path (half-applied): %v", err)
	}
	if !restored.Restored || !restored.Reopened {
		t.Fatalf("expected Restored && Reopened, got %+v", restored)
	}
	assertMoveReversed(t, app, rev, finding, oldAbs, newAbs, original)
}

// (ii) by-RevisionID recovers the same window — an operator handed the id by the WARNING log
// can restore it directly, without the by-path lookup.
func TestRestore_ByRevisionIDRecoversHalfAppliedMove(t *testing.T) {
	app, cfg := newTestEnv(t)
	original := []byte("moved journal note\nsecond line")
	oldRel := "journal/stray-note.md"
	newRel := "journal/2026-07-16-stray-note.md"
	rev, finding, oldAbs, newAbs := setupHalfAppliedMove(t, app, cfg, oldRel, newRel, original)

	restored, err := Restore(context.Background(), app, cfg, &RestoreInput{RevisionID: rev.Id})
	if err != nil {
		t.Fatalf("Restore --by revision id (half-applied): %v", err)
	}
	if !restored.Restored || !restored.Reopened {
		t.Fatalf("expected Restored && Reopened, got %+v", restored)
	}
	assertMoveReversed(t, app, rev, finding, oldAbs, newAbs, original)
}

// (iii) an applied=false row whose filesystem state does NOT confirm the window (the file is
// still at its original path) must error and mutate nothing — never guess.
func TestRestore_HalfAppliedUnconfirmedRefuses(t *testing.T) {
	app, cfg := newTestEnv(t)
	original := []byte("still at the original path\n")
	oldRel := "journal/stray-note.md"
	newRel := "journal/2026-07-16-stray-note.md"
	oldAbs := mustWriteFile(t, cfg.DeskRoot, oldRel, string(original))
	checksum := desklib.Checksum(original)
	fileRec := mustCreateFileRecord(t, app, oldRel, "journal", "journal", checksum)
	finding := mustCreateFinding(t, app, fileRec, "R2", checksum, "run-1")

	rev := core.NewRecord(mustCollection(t, app, "revisions"))
	rev.Set("path", oldRel)
	rev.Set("action", "move")
	rev.Set("new_path", newRel)
	rev.Set("original_content", string(original))
	rev.Set("original_checksum", checksum)
	rev.Set("finding", finding.Id)
	rev.Set("applied", false)
	rev.Set("restored", false)
	rev.Set("run_id", "run-1")
	if err := app.Save(rev); err != nil {
		t.Fatalf("save revision: %v", err)
	}
	// NOTE: no rename performed — the file is still at oldRel, so the FS does not confirm.

	if _, err := Restore(context.Background(), app, cfg, &RestoreInput{RevisionID: rev.Id}); err == nil {
		t.Fatalf("expected an error: an unconfirmed not-applied revision must not restore")
	}
	if _, err := Restore(context.Background(), app, cfg, &RestoreInput{Path: oldRel}); err == nil {
		t.Fatalf("expected an error: by-path found no filesystem-confirmed revision")
	}

	onDisk, rerr := os.ReadFile(oldAbs)
	if rerr != nil {
		t.Fatalf("read file: %v", rerr)
	}
	if !bytes.Equal(onDisk, original) {
		t.Fatalf("file must be untouched on refusal, got %q", string(onDisk))
	}
	revAfter := reloadRecord(t, app, "revisions", rev.Id)
	if revAfter.GetBool("applied") || revAfter.GetBool("restored") {
		t.Fatalf("no flag may flip on an unconfirmed refusal: applied=%v restored=%v",
			revAfter.GetBool("applied"), revAfter.GetBool("restored"))
	}
}

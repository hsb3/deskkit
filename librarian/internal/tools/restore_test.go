package tools

import (
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/pocketbase/pocketbase/core"

	"github.com/example/pocket-librarian/internal/desklib"
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

package tools

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/hsb3/deskkit/internal/modules/librarian/desklib"
)

func TestApplyFix_AppliesWriteAndMarksFixed(t *testing.T) {
	app, cfg := newTestEnv(t)
	content := "no frontmatter here\n"
	abs := mustWriteFile(t, cfg.DeskRoot, "tasks/example.md", content)
	checksum := desklib.Checksum([]byte(content))
	fileRec := mustCreateFileRecord(t, app, "tasks/example.md", "tasks", "task", checksum)
	finding := mustCreateFinding(t, app, fileRec, "R1", checksum, "run-1")

	proposed, err := ProposeFix(context.Background(), app, cfg, &ProposeFixInput{RunID: "run-1"})
	if err != nil || len(proposed.Proposed) != 1 || proposed.Proposed[0].Outcome != "recorded" {
		t.Fatalf("setup ProposeFix failed: err=%v res=%+v", err, proposed)
	}
	revID := proposed.Proposed[0].RevisionID

	applied, err := ApplyFix(context.Background(), app, cfg, &ApplyFixInput{RunID: "run-1"})
	if err != nil {
		t.Fatalf("ApplyFix: %v", err)
	}
	if len(applied.Outcomes) != 1 || applied.Outcomes[0].Outcome != "applied" {
		t.Fatalf("expected a single applied outcome, got %+v", applied.Outcomes)
	}
	if applied.Outcomes[0].RevisionID != revID {
		t.Fatalf("outcome revision id = %q, want %q", applied.Outcomes[0].RevisionID, revID)
	}

	onDisk, rerr := os.ReadFile(abs)
	if rerr != nil {
		t.Fatalf("read applied file: %v", rerr)
	}
	if !strings.Contains(string(onDisk), "type: task") {
		t.Fatalf("expected the R1 fixer to insert frontmatter with type: task, got %q", string(onDisk))
	}
	if !strings.HasSuffix(string(onDisk), content) {
		t.Fatalf("expected the original body preserved verbatim at the end, got %q", string(onDisk))
	}

	rev := reloadRecord(t, app, "revisions", revID)
	if !rev.GetBool("applied") {
		t.Fatalf("revision should be marked applied")
	}
	findingAfter := reloadRecord(t, app, "patrol_findings", finding.Id)
	if findingAfter.GetString("state") != "fixed" {
		t.Fatalf("finding state = %q, want fixed", findingAfter.GetString("state"))
	}

	logs, lerr := app.FindRecordsByFilter("adoption_log", "", "", 0, 0)
	if lerr != nil {
		t.Fatalf("list adoption_log: %v", lerr)
	}
	if len(logs) != 1 {
		t.Fatalf("expected exactly one adoption_log row after the batch, got %d", len(logs))
	}
	if logs[0].GetString("event") != "fix" {
		t.Fatalf("adoption_log event = %q, want fix", logs[0].GetString("event"))
	}
}

func TestApplyFix_StalenessGuardReCheckedAtApplyTime(t *testing.T) {
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

	// The file changes again AFTER propose_fix recorded the original but BEFORE apply_fix
	// commits the write — the re-check must catch this even though propose_fix's own
	// staleness guard already passed once.
	changed := "someone edited this in between\n"
	if err := os.WriteFile(abs, []byte(changed), 0o644); err != nil {
		t.Fatalf("simulate a concurrent edit: %v", err)
	}

	applied, err := ApplyFix(context.Background(), app, cfg, &ApplyFixInput{RunID: "run-1"})
	if err != nil {
		t.Fatalf("ApplyFix: %v", err)
	}
	if len(applied.Outcomes) != 1 || applied.Outcomes[0].Outcome != "stale" {
		t.Fatalf("expected a stale outcome re-checked at apply time, got %+v", applied.Outcomes)
	}

	onDisk, _ := os.ReadFile(abs)
	if string(onDisk) != changed {
		t.Fatalf("apply_fix must never write over a file that went stale; got %q", string(onDisk))
	}
}

func TestApplyFix_IgnoreFailClosedAtApplyTime(t *testing.T) {
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
	revID := proposed.Proposed[0].RevisionID

	// Break the ignore boundary between propose and apply (defense in depth re-check).
	breakIgnoreFile(t, cfg)

	applied, err := ApplyFix(context.Background(), app, cfg, &ApplyFixInput{RunID: "run-1"})
	if err != nil {
		t.Fatalf("ApplyFix must fail CLOSED, not error out: %v", err)
	}
	if len(applied.Outcomes) != 1 || applied.Outcomes[0].Outcome != "ignored" {
		t.Fatalf("expected ignored on a fail-closed load at apply time, got %+v", applied.Outcomes)
	}

	onDisk, _ := os.ReadFile(abs)
	if string(onDisk) != original {
		t.Fatalf("fail-closed apply must never write; got %q", string(onDisk))
	}
	rev := reloadRecord(t, app, "revisions", revID)
	if rev.GetBool("applied") {
		t.Fatalf("revision must not be marked applied when the batch fails closed")
	}
}

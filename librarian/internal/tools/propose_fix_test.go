package tools

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"

	"github.com/example/pocket-librarian/internal/config"
	"github.com/example/pocket-librarian/internal/desklib"

	// Blank-import registers this project's Go migrations (files, patrol_findings,
	// revisions, adoption_log, ...) into the same global registry PocketBase's built-in
	// migrations use, so tests.NewTestApp's RunAllMigrations() creates our collections too.
	_ "github.com/example/pocket-librarian/migrations"
)

// --- shared test scaffolding (used by propose_fix_test.go, apply_fix_test.go, restore_test.go) ---

// newTestEnv boots a fresh PocketBase test app (own temp SQLite data dir, all migrations
// applied) plus a Config pointed at a fresh temp DESK_ROOT with an empty (non-blocking)
// ignore file.
func newTestEnv(t *testing.T) (*tests.TestApp, *config.Config) {
	t.Helper()
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("tests.NewTestApp: %v", err)
	}
	t.Cleanup(app.Cleanup)

	deskRoot := t.TempDir()
	cfg := &config.Config{
		DeskRoot:     deskRoot,
		DeskName:     "test-desk",
		DecisionsDir: "_structure/decisions",
		TasksDir:     "tasks",
		AnalysesDir:  "analyses",
		JournalDir:   "journal",
		SecretsDir:   "_meta/secrets",
		IgnoreConfig: filepath.Join(deskRoot, ".librarian-ignore"),
		HandoffPath:  "_meta/HANDOFF.md",
	}
	if err := os.WriteFile(cfg.IgnoreConfig, []byte("# empty — nothing ignored\n"), 0o644); err != nil {
		t.Fatalf("write ignore file: %v", err)
	}
	return app, cfg
}

// mustWriteFile writes content at DESK_ROOT-relative rel, creating parent dirs, and
// returns the absolute path.
func mustWriteFile(t *testing.T, root, rel, content string) string {
	t.Helper()
	abs := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatalf("mkdir for %s: %v", rel, err)
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
	return abs
}

func mustCollection(t *testing.T, app core.App, name string) *core.Collection {
	t.Helper()
	col, err := app.FindCollectionByNameOrId(name)
	if err != nil {
		t.Fatalf("collection %s: %v", name, err)
	}
	return col
}

// mustCreateFileRecord inserts a `files` row mirroring what sweep would have produced.
func mustCreateFileRecord(t *testing.T, app core.App, path, dirKind, entityType, checksum string) *core.Record {
	t.Helper()
	rec := core.NewRecord(mustCollection(t, app, "files"))
	rec.Set("path", path)
	rec.Set("dir_kind", dirKind)
	rec.Set("entity_type", entityType)
	rec.Set("checksum", checksum)
	rec.Set("deleted", false)
	if err := app.Save(rec); err != nil {
		t.Fatalf("save files record %s: %v", path, err)
	}
	return rec
}

// mustCreateFinding inserts a flagged, mechanical `patrol_findings` row for fileRec.
func mustCreateFinding(t *testing.T, app core.App, fileRec *core.Record, rule, checksum, runID string) *core.Record {
	t.Helper()
	rec := core.NewRecord(mustCollection(t, app, "patrol_findings"))
	rec.Set("file", fileRec.Id)
	rec.Set("rule", rule)
	rec.Set("severity", "mechanical")
	rec.Set("detail", "test finding")
	rec.Set("proposed_fix", "test fix")
	rec.Set("state", "flagged")
	rec.Set("patrol_run", runID)
	rec.Set("checksum", checksum)
	if err := app.Save(rec); err != nil {
		t.Fatalf("save finding record for %s: %v", fileRec.GetString("path"), err)
	}
	return rec
}

// reloadRecord re-fetches a record by id, failing the test on error.
func reloadRecord(t *testing.T, app core.App, collection, id string) *core.Record {
	t.Helper()
	rec, err := app.FindRecordById(collection, id)
	if err != nil {
		t.Fatalf("reload %s/%s: %v", collection, id, err)
	}
	return rec
}

// breakIgnoreFile replaces the ignore config path with a directory, so any attempt to
// read it as a file fails deterministically (portable "unreadable file" simulation).
func breakIgnoreFile(t *testing.T, cfg *config.Config) {
	t.Helper()
	if err := os.RemoveAll(cfg.IgnoreConfig); err != nil {
		t.Fatalf("remove ignore file: %v", err)
	}
	if err := os.Mkdir(cfg.IgnoreConfig, 0o755); err != nil {
		t.Fatalf("mkdir in place of ignore file: %v", err)
	}
}

// failingSaveApp wraps a *tests.TestApp and forces app.Save to fail for one named
// collection — used to simulate a "store unreachable" failure at the exact
// record-original-first insert (spec §5.3 boundary 1) without needing a real outage.
type failingSaveApp struct {
	*tests.TestApp
	failFor string
}

func (f *failingSaveApp) Save(model core.Model) error {
	if rec, ok := model.(*core.Record); ok && rec.Collection() != nil && rec.Collection().Name == f.failFor {
		return errors.New("forced store failure (test)")
	}
	return f.TestApp.Save(model)
}

// --- propose_fix tests ---

func TestProposeFix_RecordsOriginalBeforeAnyWrite(t *testing.T) {
	app, cfg := newTestEnv(t)
	content := "no frontmatter here\n"
	abs := mustWriteFile(t, cfg.DeskRoot, "tasks/example.md", content)
	checksum := desklib.Checksum([]byte(content))
	fileRec := mustCreateFileRecord(t, app, "tasks/example.md", "tasks", "task", checksum)
	mustCreateFinding(t, app, fileRec, "R1", checksum, "run-1")

	res, err := ProposeFix(context.Background(), app, cfg, &ProposeFixInput{RunID: "run-1"})
	if err != nil {
		t.Fatalf("ProposeFix: %v", err)
	}
	if len(res.Proposed) != 1 {
		t.Fatalf("expected 1 proposed fix, got %d: %+v", len(res.Proposed), res.Proposed)
	}
	got := res.Proposed[0]
	if got.Outcome != "recorded" {
		t.Fatalf("outcome = %q, want recorded (%+v)", got.Outcome, got)
	}
	if got.RevisionID == "" {
		t.Fatalf("expected a revision id on a recorded outcome")
	}

	// The revision row exists with the matching original checksum...
	rev := reloadRecord(t, app, "revisions", got.RevisionID)
	if rev.GetString("original_checksum") != checksum {
		t.Fatalf("revision original_checksum = %q, want %q", rev.GetString("original_checksum"), checksum)
	}
	if rev.GetString("original_content") != content {
		t.Fatalf("revision original_content = %q, want %q", rev.GetString("original_content"), content)
	}
	if rev.GetBool("applied") {
		t.Fatalf("revision must not be applied yet — propose_fix never writes the fs")
	}

	// ...BEFORE any filesystem mutation: propose_fix does no fs write at all.
	onDisk, err := os.ReadFile(abs)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if string(onDisk) != content {
		t.Fatalf("propose_fix must never touch the fs, got %q", string(onDisk))
	}
}

func TestProposeFix_ForcedStoreFailureAbortsAndPreventsTheWrite(t *testing.T) {
	app, cfg := newTestEnv(t)
	content := "no frontmatter here\n"
	abs := mustWriteFile(t, cfg.DeskRoot, "tasks/example.md", content)
	checksum := desklib.Checksum([]byte(content))
	fileRec := mustCreateFileRecord(t, app, "tasks/example.md", "tasks", "task", checksum)
	mustCreateFinding(t, app, fileRec, "R1", checksum, "run-1")

	broken := &failingSaveApp{TestApp: app, failFor: "revisions"}

	if _, err := ProposeFix(context.Background(), broken, cfg, &ProposeFixInput{RunID: "run-1"}); err == nil {
		t.Fatalf("expected an error when the revisions store is unreachable")
	}

	// No revision row was created for the finding...
	revs, ferr := app.FindRecordsByFilter("revisions", "", "", 0, 0)
	if ferr != nil {
		t.Fatalf("list revisions: %v", ferr)
	}
	if len(revs) != 0 {
		t.Fatalf("expected zero revisions rows after an aborted operation, got %d", len(revs))
	}

	// ...so apply_fix has nothing to apply: no filesystem write can ever follow a failed
	// original-record. Prove it by running apply_fix over the same run and checking the
	// file is untouched.
	applied, aerr := ApplyFix(context.Background(), app, cfg, &ApplyFixInput{RunID: "run-1"})
	if aerr != nil {
		t.Fatalf("ApplyFix: %v", aerr)
	}
	if len(applied.Outcomes) != 0 {
		t.Fatalf("expected nothing to apply, got %+v", applied.Outcomes)
	}
	onDisk, rerr := os.ReadFile(abs)
	if rerr != nil {
		t.Fatalf("read file: %v", rerr)
	}
	if string(onDisk) != content {
		t.Fatalf("file was mutated despite the aborted propose_fix operation: %q", string(onDisk))
	}
}

func TestProposeFix_StalenessGuard(t *testing.T) {
	app, cfg := newTestEnv(t)
	original := "no frontmatter here\n"
	flaggedChecksum := desklib.Checksum([]byte(original))
	fileRec := mustCreateFileRecord(t, app, "tasks/example.md", "tasks", "task", flaggedChecksum)
	mustCreateFinding(t, app, fileRec, "R1", flaggedChecksum, "run-1")

	// The file changes AFTER the finding was flagged, before propose_fix ever runs.
	changed := "different content now\n"
	abs := mustWriteFile(t, cfg.DeskRoot, "tasks/example.md", changed)

	res, err := ProposeFix(context.Background(), app, cfg, &ProposeFixInput{RunID: "run-1"})
	if err != nil {
		t.Fatalf("ProposeFix: %v", err)
	}
	if len(res.Proposed) != 1 || res.Proposed[0].Outcome != "stale" {
		t.Fatalf("expected a single stale outcome, got %+v", res.Proposed)
	}
	if res.Proposed[0].RevisionID != "" {
		t.Fatalf("a stale outcome must not record a revision")
	}

	revs, _ := app.FindRecordsByFilter("revisions", "", "", 0, 0)
	if len(revs) != 0 {
		t.Fatalf("expected no revisions rows for a stale finding, got %d", len(revs))
	}
	onDisk, _ := os.ReadFile(abs)
	if string(onDisk) != changed {
		t.Fatalf("the staleness-guard path must never write; file = %q", string(onDisk))
	}
}

func TestProposeFix_IgnoreFailClosed(t *testing.T) {
	app, cfg := newTestEnv(t)
	content := "no frontmatter here\n"
	abs := mustWriteFile(t, cfg.DeskRoot, "tasks/example.md", content)
	checksum := desklib.Checksum([]byte(content))
	fileRec := mustCreateFileRecord(t, app, "tasks/example.md", "tasks", "task", checksum)
	mustCreateFinding(t, app, fileRec, "R1", checksum, "run-1")

	breakIgnoreFile(t, cfg)

	res, err := ProposeFix(context.Background(), app, cfg, &ProposeFixInput{RunID: "run-1"})
	if err != nil {
		t.Fatalf("ProposeFix must fail CLOSED, not error out: %v", err)
	}
	if len(res.Proposed) != 1 || res.Proposed[0].Outcome != "ignored" {
		t.Fatalf("expected every candidate ignored on a fail-closed ignore-list load, got %+v", res.Proposed)
	}

	revs, _ := app.FindRecordsByFilter("revisions", "", "", 0, 0)
	if len(revs) != 0 {
		t.Fatalf("expected zero writes on fail-closed, got %d revisions", len(revs))
	}
	onDisk, _ := os.ReadFile(abs)
	if string(onDisk) != content {
		t.Fatalf("fail-closed path must never write; file = %q", string(onDisk))
	}
}

func TestProposeFix_NeverClobberNoop(t *testing.T) {
	app, cfg := newTestEnv(t)
	content := "---\ntype: decision\ncreated: 2026-01-01\nupdated: 2026-01-01\ntags: []\nsynopsis: \"x\"\n---\nbody\n"
	abs := mustWriteFile(t, cfg.DeskRoot, "tasks/misplaced.md", content)
	checksum := desklib.Checksum([]byte(content))
	// entity_type "decision" but living under "tasks" -> R3 mismatch; expected dir is
	// cfg.DecisionsDir ("_structure/decisions").
	fileRec := mustCreateFileRecord(t, app, "tasks/misplaced.md", "tasks", "decision", checksum)
	mustCreateFinding(t, app, fileRec, "R3", checksum, "run-1")

	// The destination already exists -> the planner must never clobber it.
	destContent := "already lives here\n"
	destAbs := mustWriteFile(t, cfg.DeskRoot, "_structure/decisions/misplaced.md", destContent)

	res, err := ProposeFix(context.Background(), app, cfg, &ProposeFixInput{RunID: "run-1"})
	if err != nil {
		t.Fatalf("ProposeFix: %v", err)
	}
	if len(res.Proposed) != 1 || res.Proposed[0].Outcome != "noop" {
		t.Fatalf("expected noop (destination exists), got %+v", res.Proposed)
	}

	if got, _ := os.ReadFile(abs); string(got) != content {
		t.Fatalf("source file mutated on a noop: %q", string(got))
	}
	if got, _ := os.ReadFile(destAbs); string(got) != destContent {
		t.Fatalf("destination file clobbered on a noop: %q", string(got))
	}
	revs, _ := app.FindRecordsByFilter("revisions", "", "", 0, 0)
	if len(revs) != 0 {
		t.Fatalf("expected no revisions row for a noop, got %d", len(revs))
	}
}

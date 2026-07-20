package tools

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/pocketbase/pocketbase/core"

	"github.com/example/pocket-librarian/internal/core/config"
	"github.com/example/pocket-librarian/internal/modules/librarian/desklib"
)

// findingsQueryCount runs the `findings` query and returns its Count, in either the default
// live-only mode (includeDisposed=false) or the include-disposed mode. Shared by the disposition
// tests here and the persistence/re-open tests in patrol_test.go.
func findingsQueryCount(t *testing.T, app core.App, cfg *config.Config, includeDisposed bool) int {
	t.Helper()
	raw, err := Query(context.Background(), app, cfg, &QueryInput{Kind: "findings", IncludeDisposed: includeDisposed})
	if err != nil {
		t.Fatalf("Query findings (includeDisposed=%v): %v", includeDisposed, err)
	}
	var res findingsResult
	if err := json.Unmarshal(raw, &res); err != nil {
		t.Fatalf("unmarshal findings result: %v", err)
	}
	return res.Count
}

// defaultFindingsCount is findingsQueryCount in the default (live-only) mode.
func defaultFindingsCount(t *testing.T, app core.App, cfg *config.Config) int {
	t.Helper()
	return findingsQueryCount(t, app, cfg, false)
}

// TestDisposeFinding_WontFixHiddenByDefaultShownWithIncludeDisposed is DoD case 93a: disposing a
// finding wont_fix removes it from the default `query findings` view but keeps it visible with
// include_disposed=true. `state` is untouched (stays flagged) — disposition is orthogonal.
func TestDisposeFinding_WontFixHiddenByDefaultShownWithIncludeDisposed(t *testing.T) {
	app, cfg := newTestEnv(t)

	content := missingFM // fires R1
	mustWriteFile(t, cfg.DeskRoot, "tasks/x.md", content)
	checksum := desklib.Checksum([]byte(content))
	fileRec := mustCreateFileRecord(t, app, "tasks/x.md", "tasks", "task", checksum)
	finding := mustCreateFinding(t, app, fileRec, "R1", checksum, "run-1")
	// mustCreateFinding leaves disposition ''; set it 'open' to mirror what patrol now files so
	// this open finding is visible in the default view before we dispose it.
	finding.Set("disposition", "open")
	if err := app.Save(finding); err != nil {
		t.Fatalf("set finding disposition open: %v", err)
	}

	if n := defaultFindingsCount(t, app, cfg); n != 1 {
		t.Fatalf("open finding: default findings = %d, want 1", n)
	}

	res, err := DisposeFinding(context.Background(), app, cfg, finding.Id, "wont_fix", "", "")
	if err != nil {
		t.Fatalf("DisposeFinding: %v", err)
	}
	if res.Disposition != "wont_fix" || res.ID != finding.Id {
		t.Fatalf("DisposeResult = %+v, want id=%s disposition=wont_fix", res, finding.Id)
	}

	// Gone from the default (live-only) view...
	if n := defaultFindingsCount(t, app, cfg); n != 0 {
		t.Fatalf("after wont_fix dispose, default findings = %d, want 0", n)
	}
	// ...but present with include_disposed.
	if n := findingsQueryCount(t, app, cfg, true); n != 1 {
		t.Fatalf("after wont_fix dispose, include_disposed findings = %d, want 1", n)
	}

	// state is orthogonal — the finding is still flagged, only its disposition changed.
	got := reloadRecord(t, app, "patrol_findings", finding.Id)
	if got.GetString("state") != "flagged" {
		t.Fatalf("disposed finding state = %q, want flagged (disposition is orthogonal to state)", got.GetString("state"))
	}
	if got.GetString("disposition") != "wont_fix" {
		t.Fatalf("disposed finding disposition = %q, want wont_fix", got.GetString("disposition"))
	}
}

// TestDisposeFinding_NormalizesHyphenatedWontFix: the CLI-friendly "wont-fix" is stored as the
// canonical "wont_fix".
func TestDisposeFinding_NormalizesHyphenatedWontFix(t *testing.T) {
	app, cfg := newTestEnv(t)
	fileRec := mustCreateFileRecord(t, app, "tasks/x.md", "tasks", "task", "csum")
	finding := mustCreateFinding(t, app, fileRec, "R1", "csum", "run-1")

	res, err := DisposeFinding(context.Background(), app, cfg, finding.Id, "wont-fix", "", "")
	if err != nil {
		t.Fatalf("DisposeFinding(wont-fix): %v", err)
	}
	if res.Disposition != "wont_fix" {
		t.Fatalf("normalized disposition = %q, want wont_fix", res.Disposition)
	}
	if got := reloadRecord(t, app, "patrol_findings", finding.Id); got.GetString("disposition") != "wont_fix" {
		t.Fatalf("stored disposition = %q, want wont_fix", got.GetString("disposition"))
	}
}

// TestDisposeFinding_AcceptsEveryValidDisposition: each enum value round-trips.
func TestDisposeFinding_AcceptsEveryValidDisposition(t *testing.T) {
	app, cfg := newTestEnv(t)
	for _, d := range []string{"open", "acknowledged", "triaged", "wont_fix"} {
		fileRec := mustCreateFileRecord(t, app, "tasks/"+d+".md", "tasks", "task", "csum-"+d)
		finding := mustCreateFinding(t, app, fileRec, "R1", "csum-"+d, "run-1")
		res, err := DisposeFinding(context.Background(), app, cfg, finding.Id, d, "", "")
		if err != nil {
			t.Fatalf("DisposeFinding(%q): %v", d, err)
		}
		if res.Disposition != d {
			t.Fatalf("disposition = %q, want %q", res.Disposition, d)
		}
	}
}

// TestDisposeFinding_InvalidDispositionErrors: an unknown disposition value is rejected and the
// finding is left untouched.
func TestDisposeFinding_InvalidDispositionErrors(t *testing.T) {
	app, cfg := newTestEnv(t)
	fileRec := mustCreateFileRecord(t, app, "tasks/x.md", "tasks", "task", "csum")
	finding := mustCreateFinding(t, app, fileRec, "R1", "csum", "run-1")

	if _, err := DisposeFinding(context.Background(), app, cfg, finding.Id, "bogus", "", ""); err == nil {
		t.Fatalf("expected an error for an invalid disposition")
	}
	if got := reloadRecord(t, app, "patrol_findings", finding.Id); got.GetString("disposition") != "" {
		t.Fatalf("invalid dispose must not mutate the finding; disposition = %q", got.GetString("disposition"))
	}
}

// TestDisposeFinding_UnknownIDErrors: a non-existent finding id is an error.
func TestDisposeFinding_UnknownIDErrors(t *testing.T) {
	app, cfg := newTestEnv(t)
	if _, err := DisposeFinding(context.Background(), app, cfg, "nonexistentid00", "wont_fix", "", ""); err == nil {
		t.Fatalf("expected an error for an unknown finding id")
	}
}

// TestDisposeFinding_PersistsProvenance is the DoD provenance round-trip: disposing a finding
// non-open with an actor and a reason persists both, plus a non-empty disposed_at timestamp.
func TestDisposeFinding_PersistsProvenance(t *testing.T) {
	app, cfg := newTestEnv(t)
	fileRec := mustCreateFileRecord(t, app, "tasks/x.md", "tasks", "task", "csum")
	finding := mustCreateFinding(t, app, fileRec, "R1", "csum", "run-1")

	if _, err := DisposeFinding(context.Background(), app, cfg, finding.Id, "wont_fix", "somebody", "some reason"); err != nil {
		t.Fatalf("DisposeFinding: %v", err)
	}

	got := reloadRecord(t, app, "patrol_findings", finding.Id)
	if got.GetString("actor") != "somebody" {
		t.Fatalf("actor = %q, want somebody", got.GetString("actor"))
	}
	if got.GetString("reason") != "some reason" {
		t.Fatalf("reason = %q, want %q", got.GetString("reason"), "some reason")
	}
	if got.GetString("disposed_at") == "" {
		t.Fatalf("disposed_at is empty, want a stamped timestamp")
	}
}

// TestDisposeFinding_ReopenClearsProvenance: re-disposing a finding back to 'open' clears any
// previously-recorded actor/reason/disposed_at — an open finding carries no disposition
// provenance.
func TestDisposeFinding_ReopenClearsProvenance(t *testing.T) {
	app, cfg := newTestEnv(t)
	fileRec := mustCreateFileRecord(t, app, "tasks/x.md", "tasks", "task", "csum")
	finding := mustCreateFinding(t, app, fileRec, "R1", "csum", "run-1")

	if _, err := DisposeFinding(context.Background(), app, cfg, finding.Id, "wont_fix", "somebody", "some reason"); err != nil {
		t.Fatalf("DisposeFinding(wont_fix): %v", err)
	}
	if _, err := DisposeFinding(context.Background(), app, cfg, finding.Id, "open", "", ""); err != nil {
		t.Fatalf("DisposeFinding(open): %v", err)
	}

	got := reloadRecord(t, app, "patrol_findings", finding.Id)
	if got.GetString("actor") != "" {
		t.Fatalf("actor after re-open = %q, want empty", got.GetString("actor"))
	}
	if got.GetString("reason") != "" {
		t.Fatalf("reason after re-open = %q, want empty", got.GetString("reason"))
	}
	if got.GetString("disposed_at") != "" {
		t.Fatalf("disposed_at after re-open = %q, want empty", got.GetString("disposed_at"))
	}
}

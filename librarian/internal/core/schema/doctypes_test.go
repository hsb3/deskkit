package schema

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestDoctypesEmbeddedCopy_MatchesRepoRoot is the drift guard for the embedded vocabulary:
// go:embed cannot reach outside the Go module (librarian/), so core/schema carries a copy of
// the repo-root schema/doctypes.yaml. The two files must stay byte-identical — edit the repo
// root copy, then re-copy it here.
func TestDoctypesEmbeddedCopy_MatchesRepoRoot(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// librarian/internal/core/schema -> repo root is four levels up.
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..")
	canonical, err := os.ReadFile(filepath.Join(repoRoot, "schema", "doctypes.yaml"))
	if err != nil {
		t.Fatalf("read repo-root schema/doctypes.yaml: %v", err)
	}
	if string(canonical) != string(doctypesYAML) {
		t.Fatalf("internal/core/schema/doctypes.yaml has drifted from schema/doctypes.yaml; " +
			"re-copy the repo-root file (it is the source of truth)")
	}
}

func TestVocab_ParsesEmbeddedDoctypes(t *testing.T) {
	v, err := Vocab()
	if err != nil {
		t.Fatalf("Vocab: %v", err)
	}
	if !v.KnownType("decision") {
		t.Error("decision should be a known type")
	}
	if v.KnownType("no-such-type") {
		t.Error("no-such-type should be unknown")
	}
	ts, ok := v.Types["task"]
	if !ok || !ts.Lightweight {
		t.Errorf("task should be a lightweight type, got %+v (ok=%v)", ts, ok)
	}
	if !v.StatusAllowed("decision", "accepted") {
		t.Error("decision/accepted should be allowed")
	}
	if v.StatusAllowed("decision", "shipped") {
		t.Error("decision/shipped should be refused (spec family, not decision family)")
	}
	// Lightweight types have no family: any status string is accepted (the spec's §4.2
	// example gates a task doc on status `active`).
	if !v.StatusAllowed("task", "active") {
		t.Error("task/active should be allowed (no status family on lightweight types)")
	}
}

func TestValidateFrontmatter(t *testing.T) {
	v, err := Vocab()
	if err != nil {
		t.Fatalf("Vocab: %v", err)
	}

	valid := map[string]any{
		"type": "decision", "status": "accepted", "created": "2026-07-18",
		"updated": "2026-07-18", "tags": []string{"pm"},
		"decided_by": "owner", "affects_workstreams": []string{"pm"},
	}
	if reasons := v.ValidateFrontmatter(valid, "decision"); len(reasons) != 0 {
		t.Errorf("valid decision frontmatter rejected: %v", reasons)
	}

	missing := map[string]any{"type": "decision", "status": "accepted"}
	reasons := v.ValidateFrontmatter(missing, "decision")
	if len(reasons) == 0 {
		t.Fatal("frontmatter missing universal + required fields should be rejected")
	}

	badStatus := map[string]any{
		"type": "decision", "status": "nonsense", "created": "x", "updated": "x",
		"tags": []string{}, "decided_by": "o", "affects_workstreams": []string{"pm"},
	}
	reasons = v.ValidateFrontmatter(badStatus, "decision")
	if len(reasons) != 1 {
		t.Fatalf("expected exactly the status-family reason, got %v", reasons)
	}

	// Lightweight: status omitted is fine; universal keys still required.
	lw := map[string]any{"type": "task", "created": "x", "updated": "x", "tags": []string{}}
	if reasons := v.ValidateFrontmatter(lw, "task"); len(reasons) != 0 {
		t.Errorf("lightweight task without status should validate, got %v", reasons)
	}

	if reasons := v.ValidateFrontmatter(map[string]any{}, "no-such-type"); len(reasons) != 1 {
		t.Errorf("unknown type should yield exactly one reason, got %v", reasons)
	}
}

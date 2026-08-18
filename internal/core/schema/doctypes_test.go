package schema

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestDoctypesEmbeddedCopy_MatchesRepoRoot is the drift guard for the embedded vocabulary:
// go:embed cannot reach outside the Go module, so core/schema carries a copy of
// the repo-root schema/doctypes.yaml. The two files must stay byte-identical — edit the repo
// root copy, then re-copy it here. The `contract_version` marker (ADR 0009's shared-contract
// versioning) is part of what "byte-identical" now pins: bump it in the repo-root source and
// re-copy, never edit only one side.
func TestDoctypesEmbeddedCopy_MatchesRepoRoot(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// internal/core/schema -> repo root is three levels up.
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..", "..")
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

// TestParseDoctypes_RejectsUnknownContractVersion proves the loader fails loud when the
// contract_version marker is not in the known set (ADR 0009 shared-contract versioning). The
// test constructs a minimal doctypes byte slice in-memory; it does not edit the shipped file.
func TestParseDoctypes_RejectsUnknownContractVersion(t *testing.T) {
	// Valid structural shape, but a version this build does not understand.
	unknown := []byte("contract_version: 999\n" +
		"universal: [type]\n" +
		"status:\n  meta: [final]\n" +
		"types:\n  readme: { status: meta }\n")
	_, err := parseDoctypes(unknown)
	if err == nil {
		t.Fatal("parseDoctypes should reject an unrecognized contract_version")
	}
	if !strings.Contains(err.Error(), "contract_version") || !strings.Contains(err.Error(), "999") {
		t.Fatalf("error should name the unrecognized version 999, got: %v", err)
	}

	// A missing marker (contract_version 0) is likewise refused — the shipped contract always
	// carries an explicit version.
	missing := []byte("universal: [type]\nstatus:\n  meta: [final]\ntypes:\n  readme: { status: meta }\n")
	if _, err := parseDoctypes(missing); err == nil {
		t.Fatal("parseDoctypes should reject a missing contract_version")
	}

	// The same shape at a KNOWN version parses cleanly — proving the check gates on the version
	// value, not the structure.
	known := []byte("contract_version: 1\nuniversal: [type]\nstatus:\n  meta: [final]\ntypes:\n  readme: { status: meta }\n")
	if _, err := parseDoctypes(known); err != nil {
		t.Fatalf("parseDoctypes rejected a known contract_version: %v", err)
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

package schema

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestReferencesEmbeddedCopy_MatchesRepoRoot is the drift guard for the embedded reference
// contract: go:embed cannot reach outside the Go module (librarian/), so core/schema carries a
// copy of the repo-root schema/references.yaml. The two files must stay byte-identical — edit
// the repo-root copy, then re-copy it here.
func TestReferencesEmbeddedCopy_MatchesRepoRoot(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// librarian/internal/core/schema -> repo root is four levels up.
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..")
	canonical, err := os.ReadFile(filepath.Join(repoRoot, "schema", "references.yaml"))
	if err != nil {
		t.Fatalf("read repo-root schema/references.yaml: %v", err)
	}
	if string(canonical) != string(referencesYAML) {
		t.Fatalf("internal/core/schema/references.yaml has drifted from schema/references.yaml; " +
			"re-copy the repo-root file (it is the source of truth)")
	}
}

func TestReferenceVocab_ParsesEmbedded(t *testing.T) {
	v, err := ReferenceVocab()
	if err != nil {
		t.Fatalf("ReferenceVocab: %v", err)
	}
	if !v.KnownKind("issue") {
		t.Error("issue should be a known reference kind")
	}
	if !v.KnownKind("url") {
		t.Error("url should be a known reference kind")
	}
	if v.KnownKind("doc-pointer") {
		t.Error("doc-pointer should be unknown (the kind enum is closed to issue/url)")
	}
}

func TestValidateReference(t *testing.T) {
	// A known kind with a non-empty target validates.
	if err := ValidateReference("issue", "wb#42"); err != nil {
		t.Errorf("ValidateReference(issue, wb#42) should be nil, got %v", err)
	}
	// An unknown kind is a non-nil error.
	if err := ValidateReference("not-a-kind", "wb#42"); err == nil {
		t.Error("ValidateReference(not-a-kind, ...) should be a non-nil error")
	}
	// An empty target is a non-nil error.
	if err := ValidateReference("issue", ""); err == nil {
		t.Error("ValidateReference(issue, empty target) should be a non-nil error")
	}
	// A whitespace-only target is empty and is a non-nil error.
	if err := ValidateReference("issue", "   "); err == nil {
		t.Error("ValidateReference(issue, whitespace target) should be a non-nil error")
	}
	// The url kind is accepted too (seeded alongside issue).
	if err := ValidateReference("url", "https://example.test/x"); err != nil {
		t.Errorf("ValidateReference(url, ...) should be nil, got %v", err)
	}
}

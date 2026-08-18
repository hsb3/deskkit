package schema

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestProfileSchemaEmbeddedCopy_MatchesRepoRoot is the drift guard for the embedded profile
// contract: go:embed cannot reach outside the Go module, so core/schema carries a
// copy of the repo-root schema/profile.schema.yaml. The two files must stay byte-identical —
// edit the repo-root copy, then re-copy it here.
func TestProfileSchemaEmbeddedCopy_MatchesRepoRoot(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// internal/core/schema -> repo root is three levels up.
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..", "..")
	canonical, err := os.ReadFile(filepath.Join(repoRoot, "schema", "profile.schema.yaml"))
	if err != nil {
		t.Fatalf("read repo-root schema/profile.schema.yaml: %v", err)
	}
	if string(canonical) != string(profileSchemaYAML) {
		t.Fatal("internal/core/schema/profile.schema.yaml has drifted from schema/profile.schema.yaml; " +
			"re-copy the repo-root file (it is the source of truth)")
	}
}

// TestProfileSchema_ShippedContractVersion proves the shipped embedded schema declares a known
// contract version and resolves without error (a build defect would surface here, not at runtime).
func TestProfileSchema_ShippedContractVersion(t *testing.T) {
	raw, err := parseProfileSchemaObject(profileSchemaYAML)
	if err != nil {
		t.Fatalf("parse embedded profile schema: %v", err)
	}
	if v, ok := raw["x-contract-version"]; !ok || v != 1 {
		t.Fatalf("embedded profile schema x-contract-version = %#v, want 1", raw["x-contract-version"])
	}
	if _, err := compileProfileSchema(raw); err != nil {
		t.Fatalf("compileProfileSchema on the shipped schema: %v", err)
	}
	// The gate must have stripped the schema-meta key before compilation.
	if _, still := raw["x-contract-version"]; !still {
		t.Error("compileProfileSchema must not mutate its caller's schema object")
	}
}

// TestCompileProfileSchema_ContractVersionGate pins the fail-loud gate: an unknown numeric
// version, a missing key, and a non-numeric value are each refused with a message naming both
// the offending value and the known set.
func TestCompileProfileSchema_ContractVersionGate(t *testing.T) {
	base := func() map[string]any {
		return map[string]any{"type": "object"}
	}
	cases := []struct {
		name    string
		version any
		present bool
	}{
		{"unknown numeric version", 2, true},
		{"absent version", nil, false},
		{"non-numeric version", "1", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			raw := base()
			if tc.present {
				raw["x-contract-version"] = tc.version
			}
			_, err := compileProfileSchema(raw)
			if err == nil {
				t.Fatal("compileProfileSchema should refuse an unrecognized contract version")
			}
			if !strings.Contains(err.Error(), "1") {
				t.Errorf("error should list the known versions, got %q", err.Error())
			}
		})
	}

	// A known version compiles.
	raw := base()
	raw["x-contract-version"] = 1
	if _, err := compileProfileSchema(raw); err != nil {
		t.Errorf("a known contract version should compile, got %v", err)
	}
}

func TestValidateProfile(t *testing.T) {
	cases := []struct {
		name      string
		profile   map[string]any
		wantValid bool
		wantIn    string // substring every returned error list must mention
	}{
		{
			name: "a valid profile with an arbitrary custom block",
			profile: map[string]any{
				"schema_version": 1,
				"repos":          map[string]any{"default": "owner/repo"},
				"custom":         map[string]any{"anything": "goes"},
			},
			wantValid: true,
		},
		{
			name: "a deeply nested custom block never violates the schema",
			profile: map[string]any{
				"schema_version": 1,
				"custom": map[string]any{
					"a": map[string]any{"b": map[string]any{"c": []any{1, "two", map[string]any{"d": true}}}},
				},
			},
			wantValid: true,
		},
		{
			name:      "an unrecognized top-level key is rejected and named",
			profile:   map[string]any{"schema_version": 1, "not_a_field": "x"},
			wantValid: false,
			wantIn:    "not_a_field",
		},
		{
			name:      "a missing schema_version is rejected and named",
			profile:   map[string]any{"repos": map[string]any{"default": "owner/repo"}},
			wantValid: false,
			wantIn:    "schema_version",
		},
		{
			name:      "a non-slug repos.default is rejected and located",
			profile:   map[string]any{"schema_version": 1, "repos": map[string]any{"default": "not a slug"}},
			wantValid: false,
			wantIn:    "repos.default",
		},
		{
			name:      "an unknown nested key is rejected and named",
			profile:   map[string]any{"schema_version": 1, "identity": map[string]any{"nope": "x"}},
			wantValid: false,
			wantIn:    "nope",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := ValidateProfile(tc.profile)
			if err != nil {
				t.Fatalf("ValidateProfile: %v", err)
			}
			if res.Valid != tc.wantValid {
				t.Fatalf("Valid = %v (errors %v), want %v", res.Valid, res.Errors, tc.wantValid)
			}
			if tc.wantValid {
				if len(res.Errors) != 0 {
					t.Errorf("a valid profile should carry no errors, got %v", res.Errors)
				}
				return
			}
			if len(res.Errors) == 0 {
				t.Fatal("an invalid profile must carry at least one error string")
			}
			if !strings.Contains(strings.Join(res.Errors, " | "), tc.wantIn) {
				t.Errorf("errors %v should name %q", res.Errors, tc.wantIn)
			}
		})
	}
}

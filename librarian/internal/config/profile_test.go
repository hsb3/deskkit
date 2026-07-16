package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSubstituteMissingRequiredKeyFailsLoud(t *testing.T) {
	// AC7 (build-brief §6.2 / M-05): a {{profile.x}} with no default and an absent key
	// is a HARD load error, never a silent empty substitution.
	profile := map[string]any{"identity": map[string]any{"name": "Ada"}}

	out, err := Substitute("hello {{profile.repos.default}}", profile)
	if err == nil {
		t.Fatalf("missing required key must error; got out=%q err=nil", out)
	}
}

func TestSubstituteResolvesNested(t *testing.T) {
	profile := map[string]any{
		"identity": map[string]any{"github": map[string]any{"personal": "octocat"}},
		"board":    map[string]any{"number": 11},
	}
	got, err := Substitute("u={{profile.identity.github.personal}} p={{profile.board.number}}", profile)
	if err != nil {
		t.Fatalf("Substitute: %v", err)
	}
	if got != "u=octocat p=11" {
		t.Fatalf("got %q", got)
	}
}

func TestSubstituteDefaultUsedWhenAbsent(t *testing.T) {
	got, err := Substitute(`x={{profile.missing || "fallback"}}`, map[string]any{})
	if err != nil {
		t.Fatalf("optional placeholder with default must not error: %v", err)
	}
	if got != "x=fallback" {
		t.Fatalf("got %q, want x=fallback", got)
	}
}

func TestSubstituteEnv(t *testing.T) {
	t.Setenv("PL_ENV_SUB", "zz")
	got, err := Substitute("v={{env.PL_ENV_SUB}}", nil)
	if err != nil {
		t.Fatalf("Substitute env: %v", err)
	}
	if got != "v=zz" {
		t.Fatalf("got %q", got)
	}
	// Missing env with no default -> loud error.
	os.Unsetenv("PL_ENV_ABSENT")
	if _, err := Substitute("{{env.PL_ENV_ABSENT}}", nil); err == nil {
		t.Fatalf("missing env with no default must error")
	}
}

func TestDiscoverAndLoadProfileYAML(t *testing.T) {
	root := t.TempDir()
	kdir := filepath.Join(root, "_knowledge")
	if err := os.MkdirAll(kdir, 0o755); err != nil {
		t.Fatal(err)
	}
	yaml := "schema_version: 1\n" +
		"desk:\n  name: \"example-desk\"\n  paths:\n    decisions: \"docs/decisions\"\n" +
		"models:\n  provider: \"openai\"\n  model: \"gpt-5.4\"\n"
	if err := os.WriteFile(filepath.Join(kdir, "profile.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}

	// Discovery walks up from a nested subdir.
	sub := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	path, ok := DiscoverProfile(sub)
	if !ok {
		t.Fatalf("DiscoverProfile did not find the profile walking up")
	}
	p, err := LoadProfile(path)
	if err != nil {
		t.Fatalf("LoadProfile: %v", err)
	}
	if got := profileScalar(p, "desk.name"); got != "example-desk" {
		t.Errorf("desk.name = %q", got)
	}
	if got := profileScalar(p, "desk.paths.decisions"); got != "docs/decisions" {
		t.Errorf("desk.paths.decisions = %q", got)
	}
	if got := profileScalar(p, "models.provider"); got != "openai" {
		t.Errorf("models.provider = %q", got)
	}
}

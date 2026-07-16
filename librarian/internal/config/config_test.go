package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDotEnvNeverOverrides(t *testing.T) {
	dir := t.TempDir()
	env := "PL_TEST_PRESET=fromfile\nPL_TEST_FILEONLY=fromfile\n# a comment\nexport PL_TEST_EXPORTED=exported\n"
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(env), 0o644); err != nil {
		t.Fatal(err)
	}

	// PRESET is already set in the process env -> must NOT be overridden.
	t.Setenv("PL_TEST_PRESET", "fromenv")
	// Ensure the others are unset going in (t.Setenv restores at cleanup).
	t.Setenv("PL_TEST_FILEONLY", "")
	os.Unsetenv("PL_TEST_FILEONLY")
	t.Setenv("PL_TEST_EXPORTED", "")
	os.Unsetenv("PL_TEST_EXPORTED")

	if err := LoadDotEnv(dir); err != nil {
		t.Fatalf("LoadDotEnv: %v", err)
	}

	if got := os.Getenv("PL_TEST_PRESET"); got != "fromenv" {
		t.Errorf("PL_TEST_PRESET = %q, want fromenv (must not override process env)", got)
	}
	if got := os.Getenv("PL_TEST_FILEONLY"); got != "fromfile" {
		t.Errorf("PL_TEST_FILEONLY = %q, want fromfile", got)
	}
	if got := os.Getenv("PL_TEST_EXPORTED"); got != "exported" {
		t.Errorf("PL_TEST_EXPORTED = %q, want exported (export prefix stripped)", got)
	}
}

func TestLoadRequiresDeskIdentity(t *testing.T) {
	// With no env, no profile discoverable, and a cwd with no _knowledge/, Load must
	// error on the required DESK_ROOT/DESK_NAME rather than invent a personal default.
	dir := t.TempDir()
	restore := chdir(t, dir)
	defer restore()

	os.Unsetenv("DESK_ROOT")
	os.Unsetenv("DESK_NAME")

	if _, err := Load(); err == nil {
		t.Fatalf("Load() must error when DESK_ROOT/DESK_NAME are unset and no profile exists")
	}
}

func TestLoadEnvProvidesIdentity(t *testing.T) {
	dir := t.TempDir()
	restore := chdir(t, dir)
	defer restore()

	t.Setenv("DESK_ROOT", dir)
	t.Setenv("DESK_NAME", "example-desk")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.DeskName != "example-desk" || cfg.DeskRoot != dir {
		t.Fatalf("identity not resolved from env: %+v", cfg)
	}
	// Defaults present; IgnoreConfig derived from DeskRoot.
	if cfg.DecisionsDir != "_structure/decisions" {
		t.Errorf("DecisionsDir default = %q", cfg.DecisionsDir)
	}
	if want := filepath.Join(dir, ".librarian-ignore"); cfg.IgnoreConfig != want {
		t.Errorf("IgnoreConfig = %q, want %q", cfg.IgnoreConfig, want)
	}
	if cfg.LLMProvider != "anthropic" || cfg.LLMModel != "claude-opus-4-8" {
		t.Errorf("model defaults wrong: %q/%q", cfg.LLMProvider, cfg.LLMModel)
	}
}

// chdir changes to dir and returns a restore func (helper — avoids leaking cwd between tests).
func chdir(t *testing.T, dir string) func() {
	t.Helper()
	prev, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	return func() { _ = os.Chdir(prev) }
}

package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
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

// TestLoadContextWindow covers LLMContextWindow: env LLM_CONTEXT_WINDOW > profile
// models.context_window > 0 (0 = unset; the TUI's per-model table default applies).
func TestLoadContextWindow(t *testing.T) {
	dir := t.TempDir()
	restore := chdir(t, dir)
	defer restore()
	t.Setenv("DESK_ROOT", dir)
	t.Setenv("DESK_NAME", "example-desk")

	// Default: 0 (unset).
	os.Unsetenv("LLM_CONTEXT_WINDOW")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.LLMContextWindow != 0 {
		t.Errorf("LLMContextWindow default = %d, want 0", cfg.LLMContextWindow)
	}

	// Profile supplies it when the env var is unset.
	if err := os.MkdirAll(filepath.Join(dir, "_knowledge"), 0o755); err != nil {
		t.Fatal(err)
	}
	profile := "desk:\n  name: example-desk\nmodels:\n  context_window: 128000\n"
	if err := os.WriteFile(filepath.Join(dir, "_knowledge", "profile.yaml"), []byte(profile), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err = Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.LLMContextWindow != 128000 {
		t.Errorf("profile models.context_window not honored: %d, want 128000", cfg.LLMContextWindow)
	}

	// Env wins over the profile.
	t.Setenv("LLM_CONTEXT_WINDOW", "500000")
	cfg, err = Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.LLMContextWindow != 500000 {
		t.Errorf("LLM_CONTEXT_WINDOW env not honored: %d, want 500000", cfg.LLMContextWindow)
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

// TestLoadPMGate covers the D3 additions: PM_ENABLED env > profile modules.pm.enabled > off
// (spec §2.9), and PM_CLAIM_TTL (spec §3.6, default 30m).
func TestLoadPMGate(t *testing.T) {
	dir := t.TempDir()
	restore := chdir(t, dir)
	defer restore()
	t.Setenv("DESK_ROOT", dir)
	t.Setenv("DESK_NAME", "example-desk")

	// Default: off, 30m.
	os.Unsetenv("PM_ENABLED")
	os.Unsetenv("PM_CLAIM_TTL")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.PMEnabled {
		t.Error("PMEnabled must default off")
	}
	if cfg.PMClaimTTL != 30*time.Minute {
		t.Errorf("PMClaimTTL default = %v, want 30m", cfg.PMClaimTTL)
	}

	// Profile turns it on when env is unset.
	if err := os.MkdirAll(filepath.Join(dir, "_knowledge"), 0o755); err != nil {
		t.Fatal(err)
	}
	profile := "desk:\n  name: example-desk\nmodules:\n  pm:\n    enabled: true\n"
	if err := os.WriteFile(filepath.Join(dir, "_knowledge", "profile.yaml"), []byte(profile), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err = Load()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.PMEnabled {
		t.Error("profile modules.pm.enabled: true must enable pm")
	}

	// Env wins over the profile.
	t.Setenv("PM_ENABLED", "false")
	cfg, err = Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.PMEnabled {
		t.Error("PM_ENABLED=false env must beat the profile's true")
	}

	t.Setenv("PM_CLAIM_TTL", "45m")
	cfg, err = Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.PMClaimTTL != 45*time.Minute {
		t.Errorf("PM_CLAIM_TTL=45m not honored: %v", cfg.PMClaimTTL)
	}
}

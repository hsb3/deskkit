package config

import (
	"os"
	"path/filepath"
	"testing"
)

// writeProfile plants a _knowledge/profile.yaml under dir — the same walk-up surface Load
// discovers.
func writeProfile(t *testing.T, dir, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, ProfileRootDir), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ProfileRootDir, "profile.yaml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestLoadEditorURL covers the editor hand-off leg: unset by default (a desk that declares no
// editor gets no hand-off), supplied by the profile's preferences block, and overridable by env
// like every other resolved field. The SOURCE is asserted alongside the value at each step —
// a surface that shows a value under the wrong leg is the exact value/source disagreement
// Config.Sources exists to prevent.
func TestLoadEditorURL(t *testing.T) {
	dir := t.TempDir()
	restore := chdir(t, dir)
	defer restore()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir()) // empty central leg
	t.Setenv("DESK_ROOT", dir)
	t.Setenv("DESK_NAME", "example-desk")
	os.Unsetenv("EDITOR_URL")

	// Default: unset. There is no neutral default editor to invent.
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.EditorURL != "" {
		t.Errorf("EditorURL default = %q, want empty", cfg.EditorURL)
	}
	if cfg.Sources["EDITOR_URL"] != SourceDefault {
		t.Errorf("unset EDITOR_URL source = %q, want %q", cfg.Sources["EDITOR_URL"], SourceDefault)
	}

	// The profile's preferences block supplies it.
	const tmpl = "x-editor://open?path={abs}"
	writeProfile(t, dir, "desk:\n  name: example-desk\npreferences:\n  editor_url: \""+tmpl+"\"\n")
	cfg, err = Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.EditorURL != tmpl {
		t.Errorf("profile preferences.editor_url not honored: %q, want %q", cfg.EditorURL, tmpl)
	}
	if cfg.Sources["EDITOR_URL"] != SourceProfile {
		t.Errorf("profile-supplied EDITOR_URL source = %q, want %q", cfg.Sources["EDITOR_URL"], SourceProfile)
	}

	// Env wins over the profile, the same precedence every other field follows.
	t.Setenv("EDITOR_URL", "x-other://{path}")
	cfg, err = Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.EditorURL != "x-other://{path}" {
		t.Errorf("EDITOR_URL env not honored: %q", cfg.EditorURL)
	}
	if cfg.Sources["EDITOR_URL"] != SourceEnv {
		t.Errorf("env-supplied EDITOR_URL source = %q, want %q", cfg.Sources["EDITOR_URL"], SourceEnv)
	}
}

package store

import (
	"os"
	"path/filepath"
	"testing"
)

// The D2b automatic path migration (spec §2.10): if the new deskkit/<desk> home is absent and
// the legacy pocket-librarian/<desk> directory exists, it moves; otherwise nothing happens.

func TestMigrateLegacyStoreDir_MovesLegacyHome(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_DATA_HOME", xdg)

	oldDir := filepath.Join(xdg, legacyAppDirName, "desk-alpha")
	if err := os.MkdirAll(oldDir, 0o700); err != nil {
		t.Fatalf("seed legacy store: %v", err)
	}
	marker := filepath.Join(oldDir, "data.db")
	if err := os.WriteFile(marker, []byte("marker-bytes"), 0o600); err != nil {
		t.Fatalf("seed marker file: %v", err)
	}

	if err := MigrateLegacyStoreDir("desk-alpha"); err != nil {
		t.Fatalf("MigrateLegacyStoreDir: %v", err)
	}

	newDir := filepath.Join(xdg, AppDirName, "desk-alpha")
	if _, err := os.Stat(oldDir); !os.IsNotExist(err) {
		t.Fatalf("legacy dir must be gone after migration; stat err = %v", err)
	}
	b, err := os.ReadFile(filepath.Join(newDir, "data.db"))
	if err != nil {
		t.Fatalf("marker file must survive the move: %v", err)
	}
	if string(b) != "marker-bytes" {
		t.Fatalf("marker content changed across the move: %q", b)
	}
}

func TestMigrateLegacyStoreDir_NoopWhenNewHomeExists(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_DATA_HOME", xdg)

	oldDir := filepath.Join(xdg, legacyAppDirName, "desk-beta")
	newDir := filepath.Join(xdg, AppDirName, "desk-beta")
	for _, d := range []string{oldDir, newDir} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatalf("seed %s: %v", d, err)
		}
	}
	if err := os.WriteFile(filepath.Join(oldDir, "old.txt"), []byte("old"), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}

	if err := MigrateLegacyStoreDir("desk-beta"); err != nil {
		t.Fatalf("MigrateLegacyStoreDir: %v", err)
	}
	// Both homes untouched: never overwrite an existing new home.
	if _, err := os.Stat(filepath.Join(oldDir, "old.txt")); err != nil {
		t.Fatalf("legacy dir must be left alone when the new home exists: %v", err)
	}
	if _, err := os.Stat(filepath.Join(newDir, "old.txt")); !os.IsNotExist(err) {
		t.Fatalf("new home must not gain the legacy content; stat err = %v", err)
	}
}

func TestMigrateLegacyStoreDir_ErrorWhenNewParentUncreatable(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_DATA_HOME", xdg)

	// Seed a legacy store so the migration would proceed.
	oldDir := filepath.Join(xdg, legacyAppDirName, "desk-err")
	if err := os.MkdirAll(oldDir, 0o700); err != nil {
		t.Fatalf("seed: %v", err)
	}
	// Block MkdirAll by planting a regular file where the deskkit/ parent dir must go.
	if err := os.WriteFile(filepath.Join(xdg, AppDirName), []byte("block"), 0o600); err != nil {
		t.Fatalf("block: %v", err)
	}

	if err := MigrateLegacyStoreDir("desk-err"); err == nil {
		t.Fatal("expected an error when the new home's parent cannot be created, got nil")
	}
	if _, err := os.Stat(oldDir); err != nil {
		t.Fatalf("legacy store must be untouched on a failed migration: %v", err)
	}
}

func TestMigrateLegacyStoreDir_NoopWhenNoLegacyHome(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_DATA_HOME", xdg)

	if err := MigrateLegacyStoreDir("desk-gamma"); err != nil {
		t.Fatalf("MigrateLegacyStoreDir on a fresh XDG home must be a silent no-op: %v", err)
	}
	if _, err := os.Stat(filepath.Join(xdg, AppDirName, "desk-gamma")); !os.IsNotExist(err) {
		t.Fatalf("no directory may be created by a no-op migration; stat err = %v", err)
	}
}

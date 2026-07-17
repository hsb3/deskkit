package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStoreDir_UsesXDGDataHomeWhenSet(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "/custom/xdg")

	got, err := StoreDir("desk-alpha")
	if err != nil {
		t.Fatalf("StoreDir: %v", err)
	}
	want := filepath.Join("/custom/xdg", AppDirName, "desk-alpha")
	if got != want {
		t.Fatalf("StoreDir with XDG_DATA_HOME set: got %q, want %q", got, want)
	}
}

func TestStoreDir_FallsBackToLocalShareWhenUnset(t *testing.T) {
	// t.Setenv registers restoration of the original value; os.Unsetenv within the test then
	// exercises the genuinely-unset path.
	t.Setenv("XDG_DATA_HOME", "placeholder")
	if err := os.Unsetenv("XDG_DATA_HOME"); err != nil {
		t.Fatalf("unset XDG_DATA_HOME: %v", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("no home dir available: %v", err)
	}

	got, err := StoreDir("desk-alpha")
	if err != nil {
		t.Fatalf("StoreDir: %v", err)
	}
	want := filepath.Join(home, ".local", "share", AppDirName, "desk-alpha")
	if got != want {
		t.Fatalf("StoreDir with XDG_DATA_HOME unset: got %q, want %q", got, want)
	}
}

func TestStoreDir_EmptyXDGTreatedAsUnset(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "")

	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("no home dir available: %v", err)
	}

	got, err := StoreDir("desk-alpha")
	if err != nil {
		t.Fatalf("StoreDir: %v", err)
	}
	want := filepath.Join(home, ".local", "share", AppDirName, "desk-alpha")
	if got != want {
		t.Fatalf("StoreDir with empty XDG_DATA_HOME: got %q, want %q", got, want)
	}
}

func TestStoreDir_EmbedsDeskName(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "/custom/xdg")

	got, err := StoreDir("desk-beta")
	if err != nil {
		t.Fatalf("StoreDir: %v", err)
	}
	if filepath.Base(got) != "desk-beta" {
		t.Fatalf("StoreDir must embed the desk name as the leaf dir: got %q", got)
	}
}

func TestStoreDir_ErrorsOnEmptyDeskName(t *testing.T) {
	if _, err := StoreDir(""); err == nil {
		t.Fatalf("StoreDir(\"\") must error: an empty DESK_NAME has no resolvable store dir")
	}
}

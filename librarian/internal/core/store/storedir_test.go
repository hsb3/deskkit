package store

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

func TestListDesks_ListsStoreDirsSorted(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_DATA_HOME", home)
	base := filepath.Join(home, AppDirName)
	for _, name := range []string{"beta", "alpha"} {
		if err := os.MkdirAll(filepath.Join(base, name), 0o700); err != nil {
			t.Fatalf("seed desk %s: %v", name, err)
		}
	}
	// A stray file in the store home is not a desk.
	if err := os.WriteFile(filepath.Join(base, "notes.txt"), []byte("x"), 0o600); err != nil {
		t.Fatalf("seed stray file: %v", err)
	}

	names, gotBase, err := ListDesks()
	if err != nil {
		t.Fatalf("ListDesks: %v", err)
	}
	if gotBase != base {
		t.Fatalf("ListDesks base: got %q, want %q", gotBase, base)
	}
	want := []string{"alpha", "beta"}
	if len(names) != len(want) {
		t.Fatalf("ListDesks names: got %v, want %v", names, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("ListDesks names: got %v, want %v (sorted, dirs only)", names, want)
		}
	}
}

func TestListDesks_MissingHomeIsNotAnError(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "never-created"))

	names, base, err := ListDesks()
	if err != nil {
		t.Fatalf("ListDesks on a missing store home must not error: %v", err)
	}
	if len(names) != 0 {
		t.Fatalf("ListDesks on a missing store home: got %v, want none", names)
	}
	if base == "" {
		t.Fatal("ListDesks must still report the base path it looked in")
	}
}

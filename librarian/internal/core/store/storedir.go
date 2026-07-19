package store

import (
	"fmt"
	"os"
	"path/filepath"
)

// AppDirName is the XDG application subdirectory that owns per-desk PocketBase stores
// (ADR 0002 §2, canonical store home). A desk's store lives at
// <data-home>/pocket-librarian/<DESK_NAME>/.
const AppDirName = "pocket-librarian"

// StoreDir resolves the canonical PocketBase data directory for a desk when no explicit
// --dir override is passed (ADR 0002 §2). It is $XDG_DATA_HOME/pocket-librarian/<deskName>,
// falling back to ~/.local/share/pocket-librarian/<deskName> when $XDG_DATA_HOME is unset or
// empty. deskName must be non-empty: the caller resolves DESK_NAME (env > profile) first and
// fails closed on an unresolvable location rather than inventing a directory name.
func StoreDir(deskName string) (string, error) {
	if deskName == "" {
		return "", fmt.Errorf("config: cannot resolve store dir: DESK_NAME is empty")
	}
	// An empty XDG_DATA_HOME is treated the same as unset (per the XDG base-dir convention),
	// falling back to the platform-neutral ~/.local/share.
	base := os.Getenv("XDG_DATA_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("config: resolve home directory for the default store location: %w", err)
		}
		base = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(base, AppDirName, deskName), nil
}

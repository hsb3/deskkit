package store

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// AppDirName is the XDG application subdirectory that owns per-desk PocketBase stores
// (ADR 0002 §2, canonical store home; renamed with the binary in D2b — spec §2.10). A desk's
// store lives at <data-home>/deskkit/<DESK_NAME>/.
const AppDirName = "deskkit"

// legacyAppDirName is the pre-rename store home: the binary shipped as `pocket-librarian`
// through v0.6.0, so existing desks may still hold their store under
// <data-home>/pocket-librarian/<DESK_NAME>/. MigrateLegacyStoreDir moves such a store to the
// AppDirName home on startup so no desk loses its store across the rename (spec §2.10).
const legacyAppDirName = "pocket-librarian"

// dataHome resolves the XDG data home: $XDG_DATA_HOME, falling back to ~/.local/share when
// unset or empty (per the XDG base-dir convention).
func dataHome() (string, error) {
	base := os.Getenv("XDG_DATA_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("config: resolve home directory for the default store location: %w", err)
		}
		base = filepath.Join(home, ".local", "share")
	}
	return base, nil
}

// StoreDir resolves the canonical PocketBase data directory for a desk when no explicit
// --dir override is passed (ADR 0002 §2). It is $XDG_DATA_HOME/deskkit/<deskName>,
// falling back to ~/.local/share/deskkit/<deskName> when $XDG_DATA_HOME is unset or
// empty. deskName must be non-empty: the caller resolves DESK_NAME (env > profile) first and
// fails closed on an unresolvable location rather than inventing a directory name.
func StoreDir(deskName string) (string, error) {
	if deskName == "" {
		return "", fmt.Errorf("config: cannot resolve store dir: DESK_NAME is empty")
	}
	base, err := dataHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, AppDirName, deskName), nil
}

// ListDesks enumerates the desk store directories under <data-home>/deskkit/ — the desks this
// machine has a store for. names holds the directory (= desk) names, sorted; base is the store
// home it looked in, returned even when nothing is there so a caller can name the path in an
// empty-state message. A MISSING home is not an error: it means no desk has a store yet, so
// names is empty and err is nil. Directories only — a stray file in the store home is not a
// desk. It creates nothing.
func ListDesks() (names []string, base string, err error) {
	home, err := dataHome()
	if err != nil {
		return nil, "", err
	}
	base = filepath.Join(home, AppDirName)
	// os.ReadDir returns entries sorted by filename, which is the sort order promised above.
	entries, rerr := os.ReadDir(base)
	if rerr != nil {
		if errors.Is(rerr, fs.ErrNotExist) {
			return nil, base, nil
		}
		return nil, base, fmt.Errorf("store: list desks in %s: %w", base, rerr)
	}
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names, base, nil
}

// MigrateLegacyStoreDir moves a desk's store from the pre-rename home
// (<data-home>/pocket-librarian/<deskName>/) to the canonical one
// (<data-home>/deskkit/<deskName>/) — the D2b automatic path migration (spec §2.10). It is a
// no-op unless the new home is absent AND the legacy directory exists. The move is os.Rename,
// falling back to copy+remove only if rename fails (e.g. across devices). Exactly one line is
// logged to stderr on a successful migration.
func MigrateLegacyStoreDir(deskName string) error {
	if deskName == "" {
		return nil // no resolvable store location; the caller's own guard reports that
	}
	base, err := dataHome()
	if err != nil {
		return nil // ditto: StoreDir surfaces this to the caller
	}
	newDir := filepath.Join(base, AppDirName, deskName)
	oldDir := filepath.Join(base, legacyAppDirName, deskName)
	if _, serr := os.Stat(newDir); serr == nil {
		return nil // new home already exists — never overwrite it
	}
	if fi, serr := os.Stat(oldDir); serr != nil || !fi.IsDir() {
		return nil // no legacy store — fresh desk, nothing to migrate
	}
	if mkErr := os.MkdirAll(filepath.Dir(newDir), 0o700); mkErr != nil {
		return fmt.Errorf("store: migrate legacy store %s -> %s: %w", oldDir, newDir, mkErr)
	}
	if rnErr := os.Rename(oldDir, newDir); rnErr != nil {
		if cpErr := copyTree(oldDir, newDir); cpErr != nil {
			_ = os.RemoveAll(newDir) // don't leave a half-copied new home behind
			return fmt.Errorf("store: migrate legacy store %s -> %s: %w", oldDir, newDir, cpErr)
		}
		// Best-effort cleanup; data is safely in the new home either way (the next startup
		// no-ops on "new home exists"), but tell the operator when the old copy lingers.
		if rmErr := os.RemoveAll(oldDir); rmErr != nil {
			fmt.Fprintf(os.Stderr, "deskkit: migrated store to %s but could not remove the old home %s: %v (safe to remove manually)\n", newDir, oldDir, rmErr)
		}
	}
	fmt.Fprintf(os.Stderr, "deskkit: migrated store %s -> %s\n", oldDir, newDir)
	return nil
}

// copyTree is the cross-device fallback for MigrateLegacyStoreDir: a plain recursive
// file-by-file copy preserving permissions.
func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, rerr := filepath.Rel(src, p)
		if rerr != nil {
			return rerr
		}
		target := filepath.Join(dst, rel)
		info, ierr := d.Info()
		if ierr != nil {
			return ierr
		}
		if d.IsDir() {
			return os.MkdirAll(target, info.Mode().Perm())
		}
		return copyFile(p, target, info.Mode().Perm())
	})
}

// copyFile streams src to dst (a SQLite store file can be large — never load it whole).
func copyFile(src, dst string, perm fs.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	if _, err = io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

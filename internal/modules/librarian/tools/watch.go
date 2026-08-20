package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/pocketbase/pocketbase/core"

	"github.com/hsb3/deskkit/internal/core/config"
)

// StartWatcher keeps the files index converged with the desk tree while serve runs:
// outside edits are the NORMAL case (the owner writes prose in Obsidian), so
// the index follows the disk without a manual sweep. One initial Sweep converges whatever
// changed while the server was down; after that, filesystem events re-index single files.
// Sweep remains the fallback and the full rebuild — a watcher failure only costs liveness,
// never correctness, so every error here logs and degrades rather than failing serve.
//
// Events are debounced (editors save in bursts: write + chmod + rename dances) into a
// pending set flushed on a short tick. write_doc's own writes come back through here and
// self-quiet: the re-scan produces an identical row, so nothing is saved.
func StartWatcher(ctx context.Context, app core.App, cfg *config.Config) (stop func(), err error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	root := cfg.DeskRoot
	if err := watchDirRecursive(w, root); err != nil {
		_ = w.Close()
		return nil, err
	}

	ctx, cancel := context.WithCancel(ctx)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer func() { _ = w.Close() }()

		// Converge the index with whatever happened while no server was watching.
		if _, serr := Sweep(ctx, app, cfg, &SweepInput{}); serr != nil {
			app.Logger().Warn("watcher: initial convergence sweep failed", "err", serr)
		}

		pending := map[string]bool{} // rel -> seen (existence re-checked at flush time)
		tick := time.NewTicker(500 * time.Millisecond)
		defer tick.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case werr, ok := <-w.Errors:
				if !ok {
					return
				}
				app.Logger().Warn("watcher: fs event error; `deskkit sweep` remains the fallback", "err", werr)
			case ev, ok := <-w.Events:
				if !ok {
					return
				}
				rel, keep := deskRelFor(root, ev.Name)
				if !keep {
					continue
				}
				// A created directory needs its own watch (fsnotify is not recursive), and
				// its contents (an editor's atomic-rename dance, `mv dir/`) need indexing.
				if ev.Op.Has(fsnotify.Create) {
					if fi, statErr := os.Lstat(ev.Name); statErr == nil && fi.IsDir() {
						if werr := watchDirRecursive(w, ev.Name); werr != nil {
							app.Logger().Warn("watcher: watch new dir", "dir", rel, "err", werr)
						}
						for _, sub := range filesUnder(root, ev.Name) {
							pending[sub] = true
						}
						continue
					}
				}
				pending[rel] = true
			case <-tick.C:
				if len(pending) == 0 {
					continue
				}
				for rel := range pending {
					flushOne(app, cfg, rel)
				}
				pending = map[string]bool{}
			}
		}
	}()

	return func() { cancel(); wg.Wait() }, nil
}

// flushOne re-indexes one desk-relative path from its current on-disk state: present
// regular file -> upsert its row; absent -> soft-delete. Errors log and never propagate —
// the next event or sweep converges.
func flushOne(app core.App, cfg *config.Config, rel string) {
	abs := filepath.Join(cfg.DeskRoot, filepath.FromSlash(rel))
	fi, statErr := os.Lstat(abs)
	switch {
	case statErr == nil && fi.Mode().IsRegular():
		if err := indexOneFile(app, cfg, rel); err != nil {
			app.Logger().Warn("watcher: re-index", "path", rel, "err", err)
		}
	case os.IsNotExist(statErr):
		if err := deindexFile(app, rel); err != nil {
			app.Logger().Warn("watcher: de-index", "path", rel, "err", err)
		}
	}
}

// deskRelFor maps an event's absolute path to the desk-relative slash form, applying the
// SAME pruning walkDeskFiles applies (.git, logs, pb_* — at any depth), so the watcher
// never indexes what a sweep would not.
func deskRelFor(root, abs string) (string, bool) {
	rel, err := filepath.Rel(root, abs)
	if err != nil || rel == "." || strings.HasPrefix(rel, "..") {
		return "", false
	}
	rel = filepath.ToSlash(rel)
	for _, seg := range strings.Split(rel, "/") {
		if seg == ".git" || seg == "logs" || strings.HasPrefix(seg, "pb_") {
			return "", false
		}
	}
	return rel, true
}

// watchDirRecursive adds watches for dir and every non-pruned subdirectory under it.
func watchDirRecursive(w *fsnotify.Watcher, dir string) error {
	return filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // unreadable subtree: skip, don't kill the watcher
		}
		if !d.IsDir() {
			return nil
		}
		name := d.Name()
		if p != dir && (name == ".git" || name == "logs" || strings.HasPrefix(name, "pb_")) {
			return filepath.SkipDir
		}
		if err := w.Add(p); err != nil {
			return err
		}
		return nil
	})
}

// filesUnder lists the desk-relative paths of regular files under dir (pruned like the
// walker), for indexing a directory that appeared whole.
func filesUnder(root, dir string) []string {
	var rels []string
	_ = filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if rel, keep := deskRelFor(root, p); keep {
			rels = append(rels, rel)
		}
		return nil
	})
	return rels
}

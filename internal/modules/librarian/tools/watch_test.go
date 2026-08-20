package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// pollRow polls for a files row matching filter until found or timeout; nil on timeout.
func pollRow(t *testing.T, app core.App, filter string, params dbx.Params, ok func(*core.Record) bool) *core.Record {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		recs, err := app.FindRecordsByFilter("files", filter, "", 1, 0, params)
		if err == nil && len(recs) > 0 && ok(recs[0]) {
			return recs[0]
		}
		time.Sleep(100 * time.Millisecond)
	}
	return nil
}

// TestWatcher_OutsideEditShowsUpWithoutSweep is the phase-0 done-when: an edit made by
// another editor reaches the index within seconds, with no manual sweep.
func TestWatcher_OutsideEditShowsUpWithoutSweep(t *testing.T) {
	app, cfg := newTestEnv(t)
	abs := mustWriteFile(t, cfg.DeskRoot, "notes/doc.md", wdDoc)

	stop, err := StartWatcher(context.Background(), app, cfg)
	if err != nil {
		t.Fatalf("StartWatcher: %v", err)
	}
	defer stop()

	// The watcher's initial convergence sweep indexes the pre-existing file.
	if pollRow(t, app, "path = {:p} && deleted = false", dbx.Params{"p": "notes/doc.md"},
		func(r *core.Record) bool { return r.GetString("status") == "draft" }) == nil {
		t.Fatal("initial sweep never indexed notes/doc.md")
	}

	// Outside edit: plain os.WriteFile, exactly what another editor does.
	edited := "---\ntype: guide\nstatus: shipped\n---\n\nedited outside\n"
	if err := os.WriteFile(abs, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}
	if pollRow(t, app, "path = {:p}", dbx.Params{"p": "notes/doc.md"},
		func(r *core.Record) bool { return r.GetString("status") == "shipped" }) == nil {
		t.Fatal("outside edit never reached the index")
	}

	// A new file in a new directory (fsnotify is not recursive; the watcher must follow).
	sub := filepath.Join(cfg.DeskRoot, "newdir")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "fresh.md"), []byte(wdDoc), 0o644); err != nil {
		t.Fatal(err)
	}
	if pollRow(t, app, "path = {:p} && deleted = false", dbx.Params{"p": "newdir/fresh.md"},
		func(r *core.Record) bool { return true }) == nil {
		t.Fatal("file in a newly created directory never reached the index")
	}

	// Deletion soft-deletes the row.
	if err := os.Remove(abs); err != nil {
		t.Fatal(err)
	}
	if pollRow(t, app, "path = {:p} && deleted = true", dbx.Params{"p": "notes/doc.md"},
		func(r *core.Record) bool { return true }) == nil {
		t.Fatal("deletion never soft-deleted the row")
	}
}

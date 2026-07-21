package main

// requireConfig is the one choke point every store-touching command (sweep, patrol,
// propose-fix, apply-fix, restore, agent, chat, mcp-serve, gui, and the pm group) routes
// through before touching the store (main.go). ADR 0003's auto-migration
// (app.RunAppMigrations() inside requireConfig) self-initializes a never-migrated store so a
// one-shot CLI command against a fresh desk does not leak sql.ErrNoRows. Until now that
// self-init was asserted only indirectly, via `query summary` in verify.sh. These tests assert
// it directly against a genuinely never-migrated app (not tests.NewTestApp, which pre-applies
// migrations and would make the assertion vacuous), and prove a second call is a no-op — the
// shared path the other nine commands depend on without their own direct coverage.

import (
	"io"
	"path/filepath"
	"testing"

	"github.com/pocketbase/pocketbase"

	"github.com/hsb3/desk-standard/librarian/internal/core/config"
	"github.com/hsb3/desk-standard/librarian/internal/core/module"
	"github.com/hsb3/desk-standard/librarian/internal/modules/librarian"
)

// newNeverMigratedApp builds a *pocketbase.PocketBase whose SQLite files exist (via
// app.Bootstrap()) but whose app migrations have never run — the exact state requireConfig's
// ADR-0003 self-init must handle. Unlike tests.NewTestApp, plain Bootstrap() does NOT run app
// migrations, so app.FindCollectionByNameOrId("files") fails until requireConfig (or `serve`,
// or `migrate up`) runs RunAppMigrations().
func newNeverMigratedApp(t *testing.T) *pocketbase.PocketBase {
	t.Helper()
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("app.Bootstrap: %v", err)
	}
	t.Cleanup(func() { _ = app.ResetBootstrapState() })
	return app
}

// setModuleReg registers a librarian-only module set (PM off) as the package-global moduleReg
// requireConfig reads for GuardDowngrade/StampModules, mirroring pm_test.go's idiom, and
// restores the prior global on cleanup so this test cannot leak state into siblings.
func setModuleReg(t *testing.T, cfg *config.Config) {
	t.Helper()
	prevReg := moduleReg
	t.Cleanup(func() { moduleReg = prevReg })
	reg, err := module.Register(cfg, librarian.New())
	if err != nil {
		t.Fatalf("module.Register: %v", err)
	}
	moduleReg = reg
}

// TestRequireConfig_SelfInitsAndIsIdempotent is the load-bearing ADR-0003 assertion for the
// SHARED requireConfig path all nine store-touching commands (sweep, patrol, propose-fix,
// apply-fix, restore, agent, chat, mcp-serve, gui) and the pm group reach before touching the
// store — previously exercised only indirectly via `query summary` in verify.sh.
func TestRequireConfig_SelfInitsAndIsIdempotent(t *testing.T) {
	deskRoot := t.TempDir()
	cfg := &config.Config{
		DeskRoot:     deskRoot,
		DeskName:     "selfinit-test",
		IgnoreConfig: filepath.Join(deskRoot, ".librarian-ignore"),
	}
	setModuleReg(t, cfg)

	app := newNeverMigratedApp(t)

	// BEFORE: a genuinely never-migrated store has no `files` collection yet.
	if _, err := app.FindCollectionByNameOrId("files"); err == nil {
		t.Fatal("store already has a files collection before requireConfig ran — precondition broken (not actually never-migrated)")
	}

	cfg2, err := requireConfig(app, cfg, nil)
	if err != nil {
		t.Fatalf("requireConfig (first call, self-init): %v", err)
	}
	if cfg2 == nil {
		t.Fatal("requireConfig (first call) returned a nil *config.Config alongside a nil error")
	}

	// AFTER: self-init (RunAppMigrations, ADR 0003) must have created the schema.
	if _, err := app.FindCollectionByNameOrId("files"); err != nil {
		t.Fatalf("files collection still missing after requireConfig's self-init: %v", err)
	}

	// A second call against the now-migrated store must be a no-op: no error, and the schema
	// stays intact. This is the "second call is a no-op" half of the issue's ask.
	if _, err := requireConfig(app, cfg, nil); err != nil {
		t.Fatalf("requireConfig (second call, expected no-op): %v", err)
	}
	if _, err := app.FindCollectionByNameOrId("files"); err != nil {
		t.Fatalf("files collection missing after the second (idempotent) requireConfig call: %v", err)
	}
}

// TestSweep_SelfInitsNeverMigratedStore is the issue's literal ask: run an actual non-query
// command (sweep, the simplest store-touching subcommand — no flags, no args) through the real
// RootCmd against a never-migrated store, and confirm both that it succeeds and that running it
// again is a no-op.
func TestSweep_SelfInitsNeverMigratedStore(t *testing.T) {
	deskRoot := t.TempDir()
	cfg := &config.Config{
		DeskRoot:     deskRoot,
		DeskName:     "selfinit-sweep-test",
		IgnoreConfig: filepath.Join(deskRoot, ".librarian-ignore"),
	}
	setModuleReg(t, cfg)

	app := newNeverMigratedApp(t)
	registerToolCommands(app, cfg, nil)

	// runSweep executes the real `sweep` RunE (which calls requireConfig, then tools.Sweep) via
	// cobra, discarding sweep's JSON output — only the exit status matters here. sweep's RunE
	// calls printJSON(cmd.OutOrStdout(), ...), so pointing RootCmd's own writer at io.Discard via
	// SetOut is enough; no process-global os.Stdout swap needed.
	runSweep := func() error {
		t.Helper()
		app.RootCmd.SetOut(io.Discard)
		app.RootCmd.SetArgs([]string{"sweep"})
		return app.RootCmd.Execute()
	}

	// First run: against a never-migrated store, requireConfig must self-init the schema before
	// tools.Sweep queries it, so an empty desk sweeps cleanly (zero files).
	if err := runSweep(); err != nil {
		t.Fatalf("sweep (first run, never-migrated store): %v", err)
	}

	// Second run: the store is already migrated, so requireConfig's self-init must be a no-op
	// and sweep must still succeed.
	if err := runSweep(); err != nil {
		t.Fatalf("sweep (second run, already-migrated store): %v", err)
	}
}

// Package gatedoff proves the feature gate's OFF side (spec §2.8/§2.9, test lane §10.6): on
// a desk that does not enable pm, the PM collections are NOT physically created — true
// physical omission, not inert tables.
//
// This is deliberately its own test-only package (its own test binary): registration into
// PocketBase's migration list is process-global, so the gated-ON proof (../gatedon) and this
// one must not share a process — an enabled registration in one test would leak PM
// collections into every later tests.NewTestApp here.
package gatedoff

import (
	"strings"
	"testing"

	pbcore "github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"

	"github.com/example/pocket-librarian/internal/core/config"
	"github.com/example/pocket-librarian/internal/core/migrate"
	"github.com/example/pocket-librarian/internal/core/module"
	"github.com/example/pocket-librarian/internal/core/toolcore"
	"github.com/example/pocket-librarian/internal/modules/pm"
	"github.com/example/pocket-librarian/internal/modules/pm/collections"

	// The librarian's migrations register via init()+blank-import exactly as in main — this
	// desk is a normal librarian desk that simply has not enabled pm.
	_ "github.com/example/pocket-librarian/internal/modules/librarian/collections"
)

// TestGatedOffDeskHasNoPMCollections registers the pm module DISABLED (PM_ENABLED unset ->
// cfg.PMEnabled false), boots a fresh store with all registered migrations applied, and
// asserts none of the five PM collections physically exists while the librarian's do.
func TestGatedOffDeskHasNoPMCollections(t *testing.T) {
	t.Cleanup(toolcore.Reset)
	cfg := &config.Config{DeskRoot: t.TempDir(), DeskName: "librarian-only-desk"} // PMEnabled false
	reg, err := module.Register(cfg, pm.New())
	if err != nil {
		t.Fatalf("module.Register: %v", err)
	}
	if len(reg.Enabled) != 0 {
		t.Fatalf("disabled pm must not be in the enabled set, got %d modules", len(reg.Enabled))
	}

	// The global migration list must not have grown any pm migration.
	for _, item := range pbcore.AppMigrations.Items() {
		if strings.Contains(item.File, "_pm_") {
			t.Fatalf("global migration list contains %q with pm DISABLED — the feature gate is broken", item.File)
		}
	}

	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("tests.NewTestApp: %v", err)
	}
	t.Cleanup(app.Cleanup)

	for _, name := range collections.Names() {
		if c, _ := app.FindCollectionByNameOrId(name); c != nil {
			t.Errorf("PM collection %q exists on a gated-off desk — must be physically absent (§10.6)", name)
		}
	}
	// The librarian's collections are untouched by the gate.
	for _, name := range []string{"files", "tasks", "feedback"} {
		if _, err := app.FindCollectionByNameOrId(name); err != nil {
			t.Errorf("librarian collection %q missing on a librarian-only desk: %v", name, err)
		}
	}
	// module_schema_versions (core-owned) exists, but carries no pm row after stamping.
	if err := migrate.StampModules(app, reg.MigrateModules()); err != nil {
		t.Fatalf("StampModules: %v", err)
	}
	if rec, _ := app.FindFirstRecordByFilter("module_schema_versions", "module = 'pm'", nil); rec != nil {
		t.Error("a gated-off desk must have no pm row in module_schema_versions")
	}
}

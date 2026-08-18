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

	"github.com/hsb3/deskkit/internal/core/config"
	"github.com/hsb3/deskkit/internal/core/mcp"
	"github.com/hsb3/deskkit/internal/core/migrate"
	"github.com/hsb3/deskkit/internal/core/module"
	"github.com/hsb3/deskkit/internal/core/toolcore"
	"github.com/hsb3/deskkit/internal/modules/librarian"
	"github.com/hsb3/deskkit/internal/modules/pm"
	"github.com/hsb3/deskkit/internal/modules/pm/collections"
	pmtools "github.com/hsb3/deskkit/internal/modules/pm/tools"

	// The librarian's migrations register via init()+blank-import exactly as in main — this
	// desk is a normal librarian desk that simply has not enabled pm.
	_ "github.com/hsb3/deskkit/internal/modules/librarian/collections"
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

// TestGatedOffDeskHasNoPMSurfaces is the D4 half of the OFF proof (spec §2.9): on a
// librarian-only desk, no PM tool reaches ANY surface — the shared tool registry holds none,
// the model-facing gate (agent/MCP: toolcore.ExposedTools) exposes none, the MCP server
// registers none, and no PM TUI view is mounted. The librarian's own surface set is exactly
// what it was on main.
func TestGatedOffDeskHasNoPMSurfaces(t *testing.T) {
	t.Cleanup(toolcore.Reset)
	toolcore.Reset()
	cfg := &config.Config{DeskRoot: t.TempDir(), DeskName: "librarian-only-desk"} // PMEnabled false
	reg, err := module.Register(cfg, librarian.New(), pm.New())
	if err != nil {
		t.Fatalf("module.Register: %v", err)
	}

	// The merged registry holds ONLY the librarian's seven tools.
	if got := len(toolcore.AllTools()); got != 7 {
		t.Fatalf("gated-off registry holds %d tools, want the librarian's 7", got)
	}
	for _, name := range pmtools.ToolNames() {
		if _, ok := toolcore.Spec(name); ok {
			t.Errorf("PM tool %q present in the registry on a gated-off desk", name)
		}
	}

	// The model-facing surfaces (eino agent + MCP share ExposedTools) see the unchanged
	// librarian set: 5 tools without autonomous writes, 6 with — never a PM tool.
	exposed := map[string]bool{}
	for _, n := range mcp.ExposedTools(cfg) {
		exposed[n] = true
	}
	if len(exposed) != 5 {
		t.Errorf("gated-off MCP exposes %d tools, want the librarian's 5 (%v)", len(exposed), exposed)
	}
	for _, name := range pmtools.ToolNames() {
		if exposed[name] {
			t.Errorf("PM tool %q exposed over MCP on a gated-off desk", name)
		}
	}
	if s, err := mcp.NewServer(nil, cfg); err != nil || s == nil {
		t.Fatalf("gated-off MCP server must still build for the librarian: %v", err)
	}

	// No PM TUI views are mounted (spec §5.3 gating).
	if views := reg.TUIViews(nil, cfg); len(views) != 0 {
		t.Errorf("gated-off desk mounts %d TUI views, want 0", len(views))
	}
}

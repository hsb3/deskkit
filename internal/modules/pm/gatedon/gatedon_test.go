// Package gatedon proves the feature gate's ON side + the module-scoped versioning
// discipline (spec §2.8, test lane §10.6): enabling pm registers its migrations
// programmatically, a fresh store gets the five collections, stamping records an independent
// pm row in module_schema_versions, and a store AHEAD of the binary refuses (GuardDowngrade).
//
// Own test-only package (own test binary): the programmatic registration below is
// process-global, so it must never share a process with the gated-off proof (../gatedoff).
package gatedon

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
	pmtui "github.com/hsb3/deskkit/internal/modules/pm/tui"

	// Librarian migrations register via init()+blank-import as in main: this desk runs BOTH
	// modules, proving the two version rows stay independent (R7.1).
	_ "github.com/hsb3/deskkit/internal/modules/librarian/collections"
)

func TestGatedOnDeskCreatesPMCollectionsAndStamps(t *testing.T) {
	t.Cleanup(toolcore.Reset)
	cfg := &config.Config{DeskRoot: t.TempDir(), DeskName: "pm-desk", PMEnabled: true}
	reg, err := module.Register(cfg, librarian.New(), pm.New())
	if err != nil {
		t.Fatalf("module.Register: %v", err)
	}
	if len(reg.Enabled) != 2 {
		t.Fatalf("expected librarian+pm enabled, got %d", len(reg.Enabled))
	}
	if reg.Validator == nil {
		t.Fatal("the librarian module must be captured as the DocumentValidator (§2.5)")
	}

	// RegisterProgrammatic (called inside module.Register) must have put the pm migrations on
	// PocketBase's global list.
	pmOnList := 0
	for _, item := range pbcore.AppMigrations.Items() {
		if strings.Contains(item.File, "_pm_") {
			pmOnList++
		}
	}
	if pmOnList != len(collections.Migrations()) {
		t.Fatalf("expected %d pm migrations on the runner, got %d", len(collections.Migrations()), pmOnList)
	}

	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("tests.NewTestApp: %v", err)
	}
	t.Cleanup(app.Cleanup)

	// The five PM collections physically exist alongside the librarian's.
	for _, name := range collections.Names() {
		if _, err := app.FindCollectionByNameOrId(name); err != nil {
			t.Errorf("PM collection %q missing on a gated-on desk: %v", name, err)
		}
	}
	if _, err := app.FindCollectionByNameOrId("files"); err != nil {
		t.Errorf("librarian collection files missing: %v", err)
	}

	// Stamping records INDEPENDENT per-module rows (§2.8b, R7.1).
	if err := migrate.StampModules(app, reg.MigrateModules()); err != nil {
		t.Fatalf("StampModules: %v", err)
	}
	assertVersion := func(mod string, want int) {
		rec, err := app.FindFirstRecordByFilter("module_schema_versions", "module = {:m}", map[string]any{"m": mod})
		if err != nil {
			t.Fatalf("no module_schema_versions row for %q: %v", mod, err)
		}
		if got := rec.GetInt("version"); got != want {
			t.Errorf("module %q stamped at %d, want %d", mod, got, want)
		}
	}
	// The invariant is that the store is stamped with each module's DECLARED SchemaVersion, not a
	// literal number. Derive BOTH expectations from the modules themselves so extending either
	// migration chain — whose sequence numbers are assigned at landing — does not break this test
	// on the version bump alone.
	assertVersion("pm", pm.New().SchemaVersion())
	assertVersion("librarian", librarian.New().SchemaVersion())

	// GuardDowngrade passes when store == binary...
	if err := migrate.GuardDowngrade(app, reg.MigrateModules()); err != nil {
		t.Fatalf("GuardDowngrade at matching versions: %v", err)
	}
	// ...and refuses to serve a store whose pm version is AHEAD of the binary (§2.8/§10.6),
	// while the librarian row is untouched — the desk lags/leads per module independently.
	rec, err := app.FindFirstRecordByFilter("module_schema_versions", "module = 'pm'", nil)
	if err != nil {
		t.Fatal(err)
	}
	rec.Set("version", 99)
	if err := app.Save(rec); err != nil {
		t.Fatal(err)
	}
	gerr := migrate.GuardDowngrade(app, reg.MigrateModules())
	if gerr == nil || !strings.Contains(gerr.Error(), "refusing to downgrade") {
		t.Fatalf("store ahead of binary must refuse, got %v", gerr)
	}
}

// TestGatedOnDeskHasPMSurfaces is the D4 half of the ON proof (spec §2.9/§5): with pm
// enabled, the merged registry carries librarian(7) + pm(12) tools, the model-facing gate
// exposes the PM family (all twelve with PM_AUTONOMOUS_WRITES on — the shipped default —
// and only the three reads with it off), the MCP server builds over the merged set, and the
// three PM TUI views mount.
func TestGatedOnDeskHasPMSurfaces(t *testing.T) {
	t.Cleanup(toolcore.Reset)
	toolcore.Reset()
	cfg := &config.Config{
		DeskRoot: t.TempDir(), DeskName: "pm-desk", PMEnabled: true, PMAutonomousWrites: true,
	}
	reg, err := module.Register(cfg, librarian.New(), pm.New())
	if err != nil {
		t.Fatalf("module.Register: %v", err)
	}

	if got := len(toolcore.AllTools()); got != 19 {
		t.Fatalf("gated-on registry holds %d tools, want 7+12=19", got)
	}
	exposed := map[string]bool{}
	for _, n := range mcp.ExposedTools(cfg) {
		exposed[n] = true
	}
	for _, name := range pmtools.ToolNames() {
		if !exposed[name] {
			t.Errorf("PM tool %q not exposed on the model-facing surface with writes on", name)
		}
	}
	// restore stays excluded (§5.5) and apply_fix stays behind ITS OWN librarian gate.
	if exposed["restore"] || exposed["apply_fix"] {
		t.Error("librarian gating must be unchanged by the pm module")
	}
	if s, serr := mcp.NewServer(nil, cfg); serr != nil || s == nil {
		t.Fatalf("gated-on MCP server build: %v", serr)
	}

	// PM_AUTONOMOUS_WRITES=false: agents are read-only over the graph (§13 item 9).
	toolcore.Reset()
	roCfg := &config.Config{
		DeskRoot: cfg.DeskRoot, DeskName: "pm-desk", PMEnabled: true, PMAutonomousWrites: false,
	}
	if _, err := module.Register(roCfg, librarian.New(), pm.New()); err != nil {
		t.Fatalf("module.Register (read-only): %v", err)
	}
	roExposed := map[string]bool{}
	for _, n := range mcp.ExposedTools(roCfg) {
		roExposed[n] = true
	}
	for _, name := range []string{"get_context", "list_items", "get_item"} {
		if !roExposed[name] {
			t.Errorf("read tool %q must stay exposed with writes off", name)
		}
	}
	for _, name := range []string{"create_item", "transition_item", "claim_item"} {
		if roExposed[name] {
			t.Errorf("write tool %q exposed despite PM_AUTONOMOUS_WRITES=false", name)
		}
	}

	// The three PM TUI views mount, landing view first (spec §5.3).
	views := reg.TUIViews(nil, cfg)
	if len(views) != 3 || views[0].Name() != pmtui.ContextViewName {
		names := make([]string, len(views))
		for i, v := range views {
			names[i] = v.Name()
		}
		t.Fatalf("gated-on TUI views = %v, want [%s %s %s]",
			names, pmtui.ContextViewName, pmtui.BoardViewName, pmtui.DetailViewName)
	}
}

// TestPMDownMigrationsReverse applies + reverses the pm migrations directly, proving the
// Down side is real (programmatic entries carry both directions).
func TestPMDownMigrationsReverse(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("tests.NewTestApp: %v", err)
	}
	t.Cleanup(app.Cleanup)

	// The gated-on registration in the sibling test may already have created these via
	// NewTestApp's migration run; drive Up idempotently only where absent, then Down all.
	migs := collections.Migrations()
	for _, mig := range migs {
		if c, _ := app.FindCollectionByNameOrId(pmCollectionFor(mig.Basename)); c == nil {
			if err := mig.Up(app); err != nil {
				t.Fatalf("up %s: %v", mig.Basename, err)
			}
		}
	}
	for i := len(migs) - 1; i >= 0; i-- {
		if err := migs[i].Down(app); err != nil {
			t.Fatalf("down %s: %v", migs[i].Basename, err)
		}
	}
	for _, name := range collections.Names() {
		if c, _ := app.FindCollectionByNameOrId(name); c != nil {
			t.Errorf("collection %q still exists after down migrations", name)
		}
	}
}

// pmCollectionFor maps a pm migration basename to the collection it creates.
func pmCollectionFor(basename string) string {
	switch {
	case strings.HasSuffix(basename, "_items"):
		return "items"
	case strings.HasSuffix(basename, "_dependencies"):
		return "dependencies"
	case strings.HasSuffix(basename, "_transitions"):
		return "transitions"
	case strings.HasSuffix(basename, "_notes"):
		return "notes"
	case strings.HasSuffix(basename, "_desk_config"):
		return "desk_config"
	}
	return basename
}

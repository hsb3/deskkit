package scenario

import (
	"context"
	"testing"
	"time"

	pbcore "github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"

	"github.com/hsb3/desk-standard/librarian/internal/core/config"
	"github.com/hsb3/desk-standard/librarian/internal/modules/pm/collections"
	"github.com/hsb3/desk-standard/librarian/internal/modules/pm/engine"
	"github.com/hsb3/desk-standard/librarian/internal/modules/pm/importer"
)

// newApp boots a fresh test app with the pm collections applied and a resolved config.
func newApp(t *testing.T) (pbcore.App, *config.Config) {
	t.Helper()
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("tests.NewTestApp: %v", err)
	}
	t.Cleanup(app.Cleanup)
	for _, mig := range collections.Migrations() {
		if err := mig.Up(app); err != nil {
			t.Fatalf("apply pm migration %q: %v", mig.Basename, err)
		}
	}
	cfg := &config.Config{
		DeskRoot: t.TempDir(), DeskName: "scenario-desk",
		PMEnabled: true, PMAutonomousWrites: true, PMClaimTTL: 30 * time.Minute,
	}
	return app, cfg
}

// TestFixtures_ThroughBothSurfaces runs every starter fixture through BOTH the engine core and
// the model/CLI tool bodies over independent stores. A fixture passing on both surfaces proves
// two things at once: the workflow behaves as designed (foreman amendment), and the surfaces
// agree step-for-step (thin-surface parity, §10.10) — the tool bodies add no divergent logic.
func TestFixtures_ThroughBothSurfaces(t *testing.T) {
	ctx := context.Background()
	for _, sc := range Fixtures() {
		sc := sc
		t.Run(sc.Name, func(t *testing.T) {
			for _, mk := range []struct {
				surface string
				make    func(pbcore.App, *config.Config) *Runner
			}{
				{"engine", NewEngineRunner},
				{"tools", NewToolsRunner},
			} {
				app, cfg := newApp(t)
				r := mk.make(app, cfg)
				if r.SurfaceName() != mk.surface {
					t.Fatalf("runner surface = %q, want %q", r.SurfaceName(), mk.surface)
				}
				if err := r.Run(ctx, sc); err != nil {
					t.Fatalf("[%s surface] %v", mk.surface, err)
				}
			}
		})
	}
}

// TestRunner_RedAbility guards that the harness genuinely fails when an expectation is wrong —
// a scenario runner that always passes proves nothing. It runs a scenario whose stated
// expectation contradicts the engine (an ungated analysis item that "should" be refused entry
// to work) and asserts the runner returns an error.
func TestRunner_RedAbility(t *testing.T) {
	ctx := context.Background()
	app, cfg := newApp(t)
	r := NewEngineRunner(app, cfg)
	bad := Scenario{
		Name: "deliberately-wrong",
		Steps: []Step{
			{Name: "create", Op: Create, Item: "x", Title: "x", Type: "analysis", Expect: Expect{Phase: "queue"}},
			// analysis has no gate on queue→work, so this advance SUCCEEDS; asserting a refusal
			// must make the runner fail.
			{Name: "advance wrongly expected to refuse", Op: Transition, Item: "x", To: "work",
				Expect: Expect{Refused: true, RefusalContains: "nope"}},
		},
	}
	if err := r.Run(ctx, bad); err == nil {
		t.Fatal("runner must fail when a step's stated expectation is false")
	}

	// And the opposite direction: a wrong resulting-phase expectation must also fail.
	app2, cfg2 := newApp(t)
	r2 := NewEngineRunner(app2, cfg2)
	badPhase := Scenario{
		Name: "wrong-phase",
		Steps: []Step{
			{Name: "create", Op: Create, Item: "y", Title: "y", Type: "analysis"},
			{Name: "advance but claim it landed in review", Op: Transition, Item: "y", To: "work",
				Expect: Expect{Phase: "review"}},
		},
	}
	if err := r2.Run(ctx, badPhase); err == nil {
		t.Fatal("runner must fail when the asserted resulting phase is wrong")
	}
}

// TestRunner_ReplaysImportedItems is the D8-reuse proof (spec §8.1, brief Part B/C bridge): the
// runner can drive a scenario built from IMPORTED items, not only ones a Create step makes.
// Import a small manifest, pre-bind the runner with importer.Result.IDs, then run a scenario
// that transitions an imported item and observes the imported dependency's auto-unblock — the
// exact shape D8 uses to replay this desk's real thread data through the harness.
func TestRunner_ReplaysImportedItems(t *testing.T) {
	ctx := context.Background()
	app, cfg := newApp(t)

	// Seed via the import seam (Part C), using the same desk the runner drives.
	eng := &engine.Engine{App: app, Cfg: cfg}
	m := importer.Manifest{
		Items: []importer.ManifestItem{
			{Key: "prereq", Title: "The prerequisite", Type: "analysis", Court: "desk"},
			{Key: "dependent", Title: "The dependent work", Type: "analysis", Court: "crew"},
		},
		Deps: []importer.ManifestDep{
			{From: "prereq", To: "dependent", Kind: "blocks", UnblockAt: "review", Cascade: "auto"},
		},
	}
	res, err := importer.Import(ctx, eng, m)
	if err != nil {
		t.Fatalf("import: %v", err)
	}

	r := NewEngineRunner(app, cfg)
	for key, id := range res.IDs { // pre-bind imported items (what D8 does with Result.IDs)
		r.Bind(key, id)
	}

	replay := Scenario{
		Name: "replay-imported",
		Steps: []Step{
			// No Create step: these keys resolve to the imported record ids.
			{Name: "prereq to work: dependent stays blocked", Op: Transition, Item: "prereq", To: "work",
				Expect: Expect{Phase: "work", StillBlocked: []string{"dependent"}}},
			{Name: "prereq to review: dependent auto-unblocks", Op: Transition, Item: "prereq", To: "review",
				Expect: Expect{Phase: "review", AutoUnblocked: []string{"dependent"}}},
		},
	}
	if err := r.Run(ctx, replay); err != nil {
		t.Fatalf("replay over imported items: %v", err)
	}
}

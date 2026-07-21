package importer

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/tests"

	"github.com/hsb3/desk-standard/librarian/internal/core/config"
	"github.com/hsb3/desk-standard/librarian/internal/modules/pm/collections"
	"github.com/hsb3/desk-standard/librarian/internal/modules/pm/engine"
)

// newEngine boots a fresh test app on the named desk with the pm collections applied directly
// (no global registration, cannot pollute other tests' migration lists), and returns an
// engine over it. No validator: the import writes only items + edges — it never advances a
// gated transition, so no DocumentValidator is needed.
func newEngine(t *testing.T, desk string) *engine.Engine {
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
	return &engine.Engine{
		App: app,
		Cfg: &config.Config{DeskRoot: t.TempDir(), DeskName: desk, PMClaimTTL: 30 * time.Minute},
	}
}

// sampleManifest is an identity-neutral fixture graph: a project with two children, one of which
// blocks the other (auto, unblock_at review), plus a standalone relates-to link. Generic names
// only (R5.3) — no real desk/person/repo.
func sampleManifest() Manifest {
	return Manifest{
		Items: []ManifestItem{
			{Key: "epic-alpha", Title: "Ship the widget", Type: "project", Court: "owner", Priority: 1},
			{Key: "task-spec", Title: "Write the spec", Type: "task", Court: "desk", Parent: "epic-alpha", Pointer: "tasks/spec.md", Priority: 1},
			{Key: "task-build", Title: "Build the widget", Type: "task", Court: "crew", Parent: "epic-alpha", Priority: 2},
			{Key: "note-item", Title: "A loosely related note", Type: "analysis", Court: "desk"},
		},
		Deps: []ManifestDep{
			{From: "task-spec", To: "task-build", Kind: "blocks", UnblockAt: "review", Cascade: "auto"},
			{From: "note-item", To: "epic-alpha", Kind: "relates-to"},
		},
	}
}

var idRe = regexp.MustCompile(`^[a-z0-9]{15}$`)

// TestItemID_ShapeAndDeterminism: the derived id fits PocketBase's record-id constraint and is
// a pure function of (desk, key) — the property §8.2 reproducibility rests on.
func TestItemID_ShapeAndDeterminism(t *testing.T) {
	a := ItemID("desk-one", "task-spec")
	b := ItemID("desk-one", "task-spec")
	if a != b {
		t.Fatalf("ItemID must be deterministic: %q != %q", a, b)
	}
	if !idRe.MatchString(a) {
		t.Fatalf("ItemID %q must be 15 lowercase-hex chars (PB record-id constraint)", a)
	}
	if ItemID("desk-two", "task-spec") == a {
		t.Fatal("ItemID must be desk-scoped: same key on two desks must differ")
	}
	if ItemID("desk-one", "task-build") == a {
		t.Fatal("different keys on one desk must differ")
	}
}

// TestImport_BuildsTheGraph: the manifest becomes the expected items + edges, with parent/root
// denormalization and the initial block effect the engine's Link applies.
func TestImport_BuildsTheGraph(t *testing.T) {
	eng := newEngine(t, "desk-one")
	ctx := context.Background()

	res, err := Import(ctx, eng, sampleManifest())
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if res.CreatedItems != 4 || res.CreatedDeps != 2 {
		t.Fatalf("expected 4 items + 2 deps, got %d items / %d deps", res.CreatedItems, res.CreatedDeps)
	}

	// The child's parent + root point at the epic's deterministic id (§3.1 denormalization).
	epicID := ItemID("desk-one", "epic-alpha")
	spec, err := eng.App.FindRecordById("items", ItemID("desk-one", "task-spec"))
	if err != nil {
		t.Fatal(err)
	}
	if spec.GetString("parent") != epicID || spec.GetString("root") != epicID {
		t.Fatalf("child parent/root wrong: parent=%q root=%q want %q",
			spec.GetString("parent"), spec.GetString("root"), epicID)
	}

	// task-spec blocks task-build (unblock_at review); at import time the blocker is in queue,
	// so the target starts blocked (the engine's Link applied the initial block effect).
	build, err := eng.App.FindRecordById("items", ItemID("desk-one", "task-build"))
	if err != nil {
		t.Fatal(err)
	}
	if !build.GetBool("blocked") {
		t.Fatal("task-build must be blocked by the unsatisfied blocks edge at import time")
	}
}

// TestImport_CarriesBody proves the long-form body survives the round trip the importer is the
// oracle for: manifest-with-body -> Import(fresh store) -> GraphSnapshot projection carries the
// exact bytes back. There is no store->Manifest exporter and item keys are not persisted (the
// record id is a one-way hash), so the faithful "export" is the ItemProjection/Snapshot the §10.8
// reproducibility oracle already compares. Before the e_createItem wiring passed Body, the imported
// body was empty and the projected body came back "" — this test caught that gap red-first.
func TestImport_CarriesBody(t *testing.T) {
	ctx := context.Background()

	// A non-trivial, multi-line body: exercises newline + indentation preservation, not just a
	// short token that a truncation bug could accidentally pass.
	body := "Acceptance criteria:\n" +
		"  - the widget renders\n" +
		"  - the round trip is byte-exact\n" +
		"\nNotes: body must survive export -> fresh-store import unchanged."
	m := Manifest{
		Items: []ManifestItem{
			{Key: "task-body", Title: "Item with a body", Type: "task", Court: "desk", Priority: 1, Body: body},
		},
	}

	engA := newEngine(t, "body-desk")
	if _, err := Import(ctx, engA, m); err != nil {
		t.Fatalf("import into store A: %v", err)
	}
	snapA, err := GraphSnapshot(ctx, engA)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapA.Items) != 1 {
		t.Fatalf("expected 1 projected item, got %d", len(snapA.Items))
	}
	if got := snapA.Items[0].Body; got != body {
		t.Fatalf("projected body did not survive the round trip:\n--- want ---\n%q\n--- got ---\n%q", body, got)
	}

	// The same manifest into a wholly independent fresh store is byte-identical (§8.2) — body
	// included — so the projection is deterministic and body is part of the graph's identity.
	engB := newEngine(t, "body-desk")
	if _, err := Import(ctx, engB, m); err != nil {
		t.Fatalf("import into store B: %v", err)
	}
	snapB, err := GraphSnapshot(ctx, engB)
	if err != nil {
		t.Fatal(err)
	}
	if snapA.Canonical() != snapB.Canonical() {
		t.Fatalf("two rebuilds must be byte-identical incl. body:\n--- A ---\n%s\n--- B ---\n%s",
			snapA.Canonical(), snapB.Canonical())
	}
}

// TestImport_Idempotent: a second import into the same store creates nothing new and leaves the
// graph identical (§8.1 "the import is idempotent and desk-scoped").
func TestImport_Idempotent(t *testing.T) {
	eng := newEngine(t, "desk-one")
	ctx := context.Background()
	if _, err := Import(ctx, eng, sampleManifest()); err != nil {
		t.Fatalf("first import: %v", err)
	}
	before, err := GraphSnapshot(ctx, eng)
	if err != nil {
		t.Fatal(err)
	}

	res, err := Import(ctx, eng, sampleManifest())
	if err != nil {
		t.Fatalf("second import: %v", err)
	}
	if res.CreatedItems != 0 || res.CreatedDeps != 0 {
		t.Fatalf("re-import must create nothing, got %d items / %d deps", res.CreatedItems, res.CreatedDeps)
	}
	if res.SkippedItems != 4 || res.SkippedDeps != 2 {
		t.Fatalf("re-import must skip all, got %d items / %d deps skipped", res.SkippedItems, res.SkippedDeps)
	}
	after, err := GraphSnapshot(ctx, eng)
	if err != nil {
		t.Fatal(err)
	}
	if before.Canonical() != after.Canonical() {
		t.Fatalf("re-import changed the graph:\n--- before ---\n%s\n--- after ---\n%s",
			before.Canonical(), after.Canonical())
	}
}

// TestImport_RejectsBadManifest: unknown parent / dependency references and duplicate keys fail
// loudly before any write (fail-loud discipline, R7). The "unknown item type" case is the
// importer's inherited half of ADR 0012: e_createItem calls engine.CreateItem directly, so a
// manifest item outside the schema-v1 vocabulary fails with the same propagated error and no
// importer-side code was needed to make it so.
func TestImport_RejectsBadManifest(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name string
		m    Manifest
	}{
		{"dup key", Manifest{Items: []ManifestItem{{Key: "a", Title: "A"}, {Key: "a", Title: "A2"}}}},
		{"unknown parent", Manifest{Items: []ManifestItem{{Key: "a", Title: "A", Parent: "ghost"}}}},
		{"unknown dep end", Manifest{
			Items: []ManifestItem{{Key: "a", Title: "A"}},
			Deps:  []ManifestDep{{From: "a", To: "ghost", Kind: "blocks", UnblockAt: "work", Cascade: "auto"}},
		}},
		{"empty key", Manifest{Items: []ManifestItem{{Title: "no key"}}}},
		{"parent cycle", Manifest{Items: []ManifestItem{
			{Key: "a", Title: "A", Parent: "b"},
			{Key: "b", Title: "B", Parent: "a"},
		}}},
		{"unknown item type", Manifest{Items: []ManifestItem{{Key: "a", Title: "A", Type: "epic"}}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			eng := newEngine(t, "desk-one")
			if _, err := Import(ctx, eng, tc.m); err == nil {
				t.Fatal("expected a loud manifest error, got nil")
			}
			// No partial write: the store holds no items.
			items, _ := eng.App.FindRecordsByFilter("items", "desk = {:d}", "", 0, 0, map[string]any{"d": "desk-one"})
			if len(items) != 0 {
				t.Fatalf("a rejected manifest must write nothing, found %d items", len(items))
			}
		})
	}
}

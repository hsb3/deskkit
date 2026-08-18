package collections

import (
	"testing"

	pbcore "github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"

	"github.com/hsb3/deskkit/internal/core/migrate"
)

// bodyField returns the items.body TextField and whether it is present.
func bodyField(t *testing.T, app pbcore.App) (*pbcore.TextField, bool) {
	t.Helper()
	c, err := app.FindCollectionByNameOrId("items")
	if err != nil {
		t.Fatalf("find items: %v", err)
	}
	tf, ok := c.Fields.GetByName("body").(*pbcore.TextField)
	return tf, ok
}

// findPMMigration locates a pm migration in the programmatic manifest by its exact Basename.
func findPMMigration(t *testing.T, basename string) migrate.Migration {
	t.Helper()
	for _, mig := range Migrations() {
		if mig.Basename == basename {
			return mig
		}
	}
	t.Fatalf("pm migration %q not found in Migrations()", basename)
	return migrate.Migration{}
}

// applyAll runs every pm migration Up on app (fresh-store replay).
func applyAll(t *testing.T, app pbcore.App) {
	t.Helper()
	for _, mig := range Migrations() {
		if err := mig.Up(app); err != nil {
			t.Fatalf("apply pm migration %q: %v", mig.Basename, err)
		}
	}
}

// TestMigration0006_AddsItemsBody_Fresh proves a fresh store (replay 0001..0006) has items.body
// with an explicit finite Max — the dedicated inline long-form surface (spec §3.1).
func TestMigration0006_AddsItemsBody_Fresh(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("tests.NewTestApp: %v", err)
	}
	t.Cleanup(app.Cleanup)
	applyAll(t, app)

	tf, ok := bodyField(t, app)
	if !ok {
		t.Fatal("fresh store: items.body missing")
	}
	if tf.Max <= 0 {
		t.Fatalf("items.body must carry an explicit finite Max, got %d", tf.Max)
	}
}

// TestMigration0006_AddsItemsBody_ExistingStore is the acceptance proof: the forward migration
// applies to an EXISTING store, not just a fresh replay. It stands up a store at schema v5 (only
// 0001..0005 applied), seeds an items row WITHOUT a body, then applies ONLY the 0006 migration —
// and confirms the field now exists and the pre-existing row can store and read a body value.
func TestMigration0006_AddsItemsBody_ExistingStore(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("tests.NewTestApp: %v", err)
	}
	t.Cleanup(app.Cleanup)

	// Stand up the pre-body store: apply every migration BEFORE 0006, located by basename so an
	// insertion earlier in the slice can never silently shift the boundary this test exercises.
	found := false
	for _, mig := range Migrations() {
		if mig.Basename == "0006_pm_items_body" {
			found = true
			break
		}
		if err := mig.Up(app); err != nil {
			t.Fatalf("apply pm migration %q: %v", mig.Basename, err)
		}
	}
	if !found {
		t.Fatal("0006_pm_items_body missing from Migrations()")
	}
	if _, ok := bodyField(t, app); ok {
		t.Fatal("v5 store must NOT yet have items.body")
	}

	// Seed a pre-existing item at v5 (title + phase are the required items fields).
	items, err := app.FindCollectionByNameOrId("items")
	if err != nil {
		t.Fatalf("find items: %v", err)
	}
	rec := pbcore.NewRecord(items)
	rec.Set("title", "pre-existing item")
	rec.Set("phase", "queue")
	if err := app.Save(rec); err != nil {
		t.Fatalf("seed v5 item: %v", err)
	}

	// Apply ONLY the 0006 migration (the forward step an existing store takes on next migrate).
	mig := findPMMigration(t, "0006_pm_items_body")
	if err := mig.Up(app); err != nil {
		t.Fatalf("apply 0006 on existing store: %v", err)
	}

	tf, ok := bodyField(t, app)
	if !ok {
		t.Fatal("after 0006 on existing store: items.body still missing")
	}
	if tf.Max <= 0 {
		t.Fatalf("items.body must carry an explicit finite Max, got %d", tf.Max)
	}

	// The pre-existing row can now store and read a body value.
	stored, err := app.FindRecordById("items", rec.Id)
	if err != nil {
		t.Fatalf("reload seeded item: %v", err)
	}
	const body = "narrative, acceptance criteria, and inline spec stored on the item"
	stored.Set("body", body)
	if err := app.Save(stored); err != nil {
		t.Fatalf("save body on pre-existing item: %v", err)
	}
	reread, err := app.FindRecordById("items", rec.Id)
	if err != nil {
		t.Fatalf("re-read seeded item: %v", err)
	}
	if got := reread.GetString("body"); got != body {
		t.Fatalf("pre-existing item body round-trip: got %q, want %q", got, body)
	}
}

// TestMigration0006_UpDownUp proves the 0006 migration is idempotent and reversible: down removes
// items.body, a second up re-adds it (data-safe schema rollback).
func TestMigration0006_UpDownUp(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("tests.NewTestApp: %v", err)
	}
	t.Cleanup(app.Cleanup)
	applyAll(t, app)

	mig := findPMMigration(t, "0006_pm_items_body")

	if err := mig.Down(app); err != nil {
		t.Fatalf("down: %v", err)
	}
	if _, ok := bodyField(t, app); ok {
		t.Fatal("after down, items.body must be gone")
	}
	if err := mig.Up(app); err != nil {
		t.Fatalf("up: %v", err)
	}
	if _, ok := bodyField(t, app); !ok {
		t.Fatal("after up->down->up, items.body must be present again")
	}
}

package collections

import (
	"testing"

	pbcore "github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

// filesHasField reports whether the files collection currently exposes a field by name.
func filesHasField(t *testing.T, app pbcore.App, name string) bool {
	t.Helper()
	c, err := app.FindCollectionByNameOrId("files")
	if err != nil {
		t.Fatalf("find files: %v", err)
	}
	return c.Fields.GetByName(name) != nil
}

// TestMigration0019_RenamesEntityTypeToDoctype proves the forward migration renames
// files.entity_type -> files.doctype IN PLACE (stable field id → SQLite column rename, not
// drop+add), so an up->down->up cycle preserves a seeded row's value in BOTH directions.
func TestMigration0019_RenamesEntityTypeToDoctype(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("tests.NewTestApp: %v", err)
	}
	t.Cleanup(app.Cleanup)

	// Fresh (all up): doctype present, entity_type renamed away.
	if !filesHasField(t, app, "doctype") {
		t.Fatal("fresh store: files.doctype missing")
	}
	if filesHasField(t, app, "entity_type") {
		t.Fatal("fresh store: files.entity_type must have been renamed to doctype")
	}

	// Seed a row carrying a doctype value + the required path.
	c, err := app.FindCollectionByNameOrId("files")
	if err != nil {
		t.Fatalf("find files: %v", err)
	}
	rec := pbcore.NewRecord(c)
	rec.Set("path", "_structure/decisions/0001-x.md")
	rec.Set("doctype", "decision")
	if err := app.Save(rec); err != nil {
		t.Fatalf("seed files row: %v", err)
	}
	id := rec.Id

	mig := findMigration(t, "0019_files_doctype_rename")

	// DOWN: doctype -> entity_type; the seeded value must survive under the old name (an in-place
	// rename preserves the underlying column data).
	if err := mig.Down(app); err != nil {
		t.Fatalf("down: %v", err)
	}
	if filesHasField(t, app, "doctype") || !filesHasField(t, app, "entity_type") {
		t.Fatal("after down, files must expose entity_type, not doctype")
	}
	back, err := app.FindRecordById("files", id)
	if err != nil {
		t.Fatalf("re-find seeded row after down: %v", err)
	}
	if got := back.GetString("entity_type"); got != "decision" {
		t.Fatalf("row data must survive the rename: entity_type = %q, want decision", got)
	}

	// UP: entity_type -> doctype; the value is still preserved through the round-trip.
	if err := mig.Up(app); err != nil {
		t.Fatalf("up: %v", err)
	}
	if !filesHasField(t, app, "doctype") || filesHasField(t, app, "entity_type") {
		t.Fatal("after up, files must expose doctype, not entity_type")
	}
	back, err = app.FindRecordById("files", id)
	if err != nil {
		t.Fatalf("re-find seeded row after up: %v", err)
	}
	if got := back.GetString("doctype"); got != "decision" {
		t.Fatalf("row data must survive the up->down->up round-trip: doctype = %q, want decision", got)
	}
}

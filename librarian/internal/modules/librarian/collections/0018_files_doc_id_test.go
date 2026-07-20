package collections

import (
	"testing"

	pbcore "github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

// docIDField returns the files.doc_id TextField and whether it is present.
func docIDField(t *testing.T, app pbcore.App) (*pbcore.TextField, bool) {
	t.Helper()
	c, err := app.FindCollectionByNameOrId("files")
	if err != nil {
		t.Fatalf("find files: %v", err)
	}
	tf, ok := c.Fields.GetByName("doc_id").(*pbcore.TextField)
	return tf, ok
}

// TestMigration0018_AddsDocIDField proves the forward migration adds files.doc_id with an explicit
// finite Max (the repo's explicit-Max convention), and that an up->down->up cycle removes then
// re-adds it (idempotent, data-safe).
func TestMigration0018_AddsDocIDField(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("tests.NewTestApp: %v", err)
	}
	t.Cleanup(app.Cleanup)

	// Fresh (all up): doc_id present with an explicit finite Max.
	tf, ok := docIDField(t, app)
	if !ok {
		t.Fatal("fresh store: files.doc_id missing")
	}
	if tf.Max <= 0 {
		t.Fatalf("files.doc_id must carry an explicit finite Max, got %d", tf.Max)
	}

	mig := findMigration(t, "0018_files_doc_id")

	// DOWN removes the field.
	if err := mig.Down(app); err != nil {
		t.Fatalf("down: %v", err)
	}
	if _, ok := docIDField(t, app); ok {
		t.Fatal("after down, files.doc_id must be gone")
	}

	// UP re-adds it.
	if err := mig.Up(app); err != nil {
		t.Fatalf("up: %v", err)
	}
	if _, ok := docIDField(t, app); !ok {
		t.Fatal("after up->down->up, files.doc_id must be present again")
	}
}

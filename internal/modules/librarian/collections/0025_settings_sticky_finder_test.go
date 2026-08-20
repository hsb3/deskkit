package collections

import (
	"net/http"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/tests"

	"github.com/hsb3/deskkit/internal/core/settings"
)

// TestMigration0025_SeedsDefaultOn: the ruled default for the sticky finder is ON, and a
// PocketBase bool is false when absent — so a migrated store must come up with the seeded row
// already carrying true, not with a zero value the browser has to guess about.
func TestMigration0025_SeedsDefaultOn(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("tests.NewTestApp: %v", err)
	}
	t.Cleanup(app.Cleanup)

	col, err := app.FindCollectionByNameOrId(settings.Collection)
	if err != nil {
		t.Fatalf("find settings: %v", err)
	}
	if col.Fields.GetByName(settings.FieldStickyFinder) == nil {
		t.Fatalf("settings has no %s field", settings.FieldStickyFinder)
	}
	rec, err := app.FindRecordById(settings.Collection, settings.RecordID)
	if err != nil {
		t.Fatalf("find seeded row: %v", err)
	}
	if !rec.GetBool(settings.FieldStickyFinder) {
		t.Fatalf("%s on the seeded row = false, want true (the ruled default is on)", settings.FieldStickyFinder)
	}
}

// TestMigration0025_BrowserCanFlipIt: the preference is only worth storing if the surface that
// owns it — a browser holding a superuser token — can write it over the record API and read the
// new value back.
func TestMigration0025_BrowserCanFlipIt(t *testing.T) {
	app, srv, token := settingsAPI(t)

	code, body := settingsRequest(t, http.MethodPatch, settingsRecordURL(srv),
		`{"`+settings.FieldStickyFinder+`":false}`, token)
	if code != http.StatusOK {
		t.Fatalf("PATCH status = %d, want 200; body: %s", code, body)
	}
	rec, err := app.FindRecordById(settings.Collection, settings.RecordID)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if rec.GetBool(settings.FieldStickyFinder) {
		t.Fatalf("%s stayed true after a PATCH setting it false", settings.FieldStickyFinder)
	}
	// The field is not hidden: the panel renders it, so it has to come back over the API.
	if !strings.Contains(body, settings.FieldStickyFinder) {
		t.Fatalf("PATCH response omits %s: %s", settings.FieldStickyFinder, body)
	}
}

// TestMigration0025_UpDownUp proves the field is reversible and re-appliable, and that the
// default rides back in with it.
func TestMigration0025_UpDownUp(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("tests.NewTestApp: %v", err)
	}
	t.Cleanup(app.Cleanup)

	mig := findMigration(t, "0025_settings_sticky_finder")

	if err := mig.Down(app); err != nil {
		t.Fatalf("down: %v", err)
	}
	col, err := app.FindCollectionByNameOrId(settings.Collection)
	if err != nil {
		t.Fatalf("find settings after down: %v", err)
	}
	if col.Fields.GetByName(settings.FieldStickyFinder) != nil {
		t.Fatalf("after down, %s must be gone", settings.FieldStickyFinder)
	}

	if err := mig.Up(app); err != nil {
		t.Fatalf("up: %v", err)
	}
	rec, err := app.FindRecordById(settings.Collection, settings.RecordID)
	if err != nil {
		t.Fatalf("find seeded row after up: %v", err)
	}
	if !rec.GetBool(settings.FieldStickyFinder) {
		t.Fatalf("after down->up, %s = false, want the default back", settings.FieldStickyFinder)
	}
}

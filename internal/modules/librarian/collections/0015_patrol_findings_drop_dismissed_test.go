package collections

import (
	"reflect"
	"strings"
	"testing"

	pbcore "github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

// stateValues returns the current Values slice of patrol_findings.state.
func stateValues(t *testing.T, app pbcore.App) []string {
	t.Helper()
	c, err := app.FindCollectionByNameOrId("patrol_findings")
	if err != nil {
		t.Fatalf("find patrol_findings: %v", err)
	}
	sf, ok := c.Fields.GetByName("state").(*pbcore.SelectField)
	if !ok {
		t.Fatal("patrol_findings.state is not a SelectField")
	}
	return sf.Values
}

// findMigration locates a self-registered migration by a substring of its .go basename.
func findMigration(t *testing.T, basenameSub string) *pbcore.Migration {
	t.Helper()
	for _, mig := range pbcore.AppMigrations.Items() {
		if strings.Contains(mig.File, basenameSub) {
			return mig
		}
	}
	t.Fatalf("migration containing %q not found on the app migrations list", basenameSub)
	return nil
}

// TestMigration0015_ShrinksStateAndRemapsDismissed proves the forward migration (1) narrows the
// state enum to exactly [flagged, fixed, resolved] and (2) data-first remaps a residual dismissed
// row to flagged before the shrink, and that an up->down->up cycle leaves no row outside the enum.
func TestMigration0015_ShrinksStateAndRemapsDismissed(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("tests.NewTestApp: %v", err)
	}
	t.Cleanup(app.Cleanup)

	// Fresh (all migrations up) store: dismissed is gone.
	if got, want := stateValues(t, app), []string{"flagged", "fixed", "resolved"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("fresh state enum = %v, want %v", got, want)
	}

	mig := findMigration(t, "0015_patrol_findings_drop_dismissed")

	// DOWN re-adds dismissed so a legacy row can once again be written.
	if err := mig.Down(app); err != nil {
		t.Fatalf("down: %v", err)
	}
	if !contains(stateValues(t, app), "dismissed") {
		t.Fatalf("after down, state enum = %v, want it to contain dismissed", stateValues(t, app))
	}

	// Seed a residual dismissed row (defensive: no setter writes this in the live tree).
	col, err := app.FindCollectionByNameOrId("patrol_findings")
	if err != nil {
		t.Fatalf("find collection: %v", err)
	}
	rec := pbcore.NewRecord(col)
	rec.Set("rule", "R1")
	rec.Set("severity", "mechanical")
	rec.Set("state", "dismissed")
	rec.Set("checksum", "deadbeef")
	if err := app.Save(rec); err != nil {
		t.Fatalf("seed dismissed row: %v", err)
	}
	seededID := rec.Id

	// UP remaps dismissed->flagged first, then shrinks the enum.
	if err := mig.Up(app); err != nil {
		t.Fatalf("up: %v", err)
	}

	if got, want := stateValues(t, app), []string{"flagged", "fixed", "resolved"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("after up->down->up, state enum = %v, want %v", got, want)
	}
	reloaded, err := app.FindRecordById("patrol_findings", seededID)
	if err != nil {
		t.Fatalf("reload seeded row: %v", err)
	}
	if reloaded.GetString("state") != "flagged" {
		t.Fatalf("seeded dismissed row state = %q, want flagged (data-first remap)", reloaded.GetString("state"))
	}
	leftover, err := app.FindRecordsByFilter("patrol_findings", "state = 'dismissed'", "", 0, 0)
	if err != nil {
		t.Fatalf("count dismissed rows: %v", err)
	}
	if len(leftover) != 0 {
		t.Fatalf("after up, %d rows still hold state=dismissed, want 0", len(leftover))
	}
}

func contains(vs []string, want string) bool {
	for _, v := range vs {
		if v == want {
			return true
		}
	}
	return false
}

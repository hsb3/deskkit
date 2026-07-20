package collections

import (
	"reflect"
	"testing"

	pbcore "github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

// eventValues returns the current Values slice of adoption_log.event.
func eventValues(t *testing.T, app pbcore.App) []string {
	t.Helper()
	c, err := app.FindCollectionByNameOrId("adoption_log")
	if err != nil {
		t.Fatalf("find adoption_log: %v", err)
	}
	sf, ok := c.Fields.GetByName("event").(*pbcore.SelectField)
	if !ok {
		t.Fatal("adoption_log.event is not a SelectField")
	}
	return sf.Values
}

// TestMigration0017_ShrinksEventAndDeletesAnomalous proves the forward migration (1) narrows the
// event enum to exactly [fix], (2) data-first deletes a row holding a dropped (writerless) value
// while preserving the sole real `fix` row, and that an up->down->up cycle leaves no orphan row.
func TestMigration0017_ShrinksEventAndDeletesAnomalous(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("tests.NewTestApp: %v", err)
	}
	t.Cleanup(app.Cleanup)

	// Fresh (all migrations up) store: only fix survives.
	if got, want := eventValues(t, app), []string{"fix"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("fresh event enum = %v, want %v", got, want)
	}

	mig := findMigration(t, "0017_adoption_log_shrink_event")

	// DOWN re-adds the five writerless values so legacy rows can once again be written.
	if err := mig.Down(app); err != nil {
		t.Fatalf("down: %v", err)
	}
	if !contains(eventValues(t, app), "note") {
		t.Fatalf("after down, event enum = %v, want it to contain note", eventValues(t, app))
	}

	col, err := app.FindCollectionByNameOrId("adoption_log")
	if err != nil {
		t.Fatalf("find collection: %v", err)
	}
	// A real fix row (must survive) and an anomalous note row (must be deleted).
	fixRec := pbcore.NewRecord(col)
	fixRec.Set("desk", "test-desk")
	fixRec.Set("event", "fix")
	fixRec.Set("detail", "run x: patched 1")
	if err := app.Save(fixRec); err != nil {
		t.Fatalf("seed fix row: %v", err)
	}
	fixID := fixRec.Id
	noteRec := pbcore.NewRecord(col)
	noteRec.Set("desk", "test-desk")
	noteRec.Set("event", "note")
	noteRec.Set("detail", "anomalous writerless row")
	if err := app.Save(noteRec); err != nil {
		t.Fatalf("seed note row: %v", err)
	}

	// UP deletes the anomalous row first, then shrinks the enum.
	if err := mig.Up(app); err != nil {
		t.Fatalf("up: %v", err)
	}

	if got, want := eventValues(t, app), []string{"fix"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("after up->down->up, event enum = %v, want %v", got, want)
	}
	if _, err := app.FindRecordById("adoption_log", fixID); err != nil {
		t.Fatalf("the fix row must survive the shrink: %v", err)
	}
	remaining, err := app.FindRecordsByFilter("adoption_log", "", "", 0, 0)
	if err != nil {
		t.Fatalf("list adoption_log: %v", err)
	}
	if len(remaining) != 1 || remaining[0].GetString("event") != "fix" {
		t.Fatalf("after shrink, adoption_log = %d rows (want 1 fix row); events=%v", len(remaining), remaining)
	}
}

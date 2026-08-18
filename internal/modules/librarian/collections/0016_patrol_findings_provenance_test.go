package collections

import (
	"testing"

	pbcore "github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

// TestMigration0016_AddsProvenanceFieldsReversibly proves the forward migration adds the three
// disposition-provenance fields to patrol_findings with the right types/ceilings — actor
// (TextField Max 200), reason (TextField Max 2000), disposed_at (a plain DateField) — and that an
// up->down->up cycle removes them on DOWN and restores them on the next UP. Sibling to the 0015
// and 0017 cycle tests; reuses findMigration from the 0015 test file (same package).
func TestMigration0016_AddsProvenanceFieldsReversibly(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("tests.NewTestApp: %v", err)
	}
	t.Cleanup(app.Cleanup)

	assertPresent := func(phase string) {
		c, err := app.FindCollectionByNameOrId("patrol_findings")
		if err != nil {
			t.Fatalf("%s: find patrol_findings: %v", phase, err)
		}
		actor, ok := c.Fields.GetByName("actor").(*pbcore.TextField)
		if !ok {
			t.Fatalf("%s: actor missing or not a TextField", phase)
		}
		if actor.Max != 200 {
			t.Fatalf("%s: actor.Max = %d, want 200", phase, actor.Max)
		}
		reason, ok := c.Fields.GetByName("reason").(*pbcore.TextField)
		if !ok {
			t.Fatalf("%s: reason missing or not a TextField", phase)
		}
		if reason.Max != 2000 {
			t.Fatalf("%s: reason.Max = %d, want 2000", phase, reason.Max)
		}
		if _, ok := c.Fields.GetByName("disposed_at").(*pbcore.DateField); !ok {
			t.Fatalf("%s: disposed_at missing or not a DateField", phase)
		}
	}
	assertAbsent := func(phase string) {
		c, err := app.FindCollectionByNameOrId("patrol_findings")
		if err != nil {
			t.Fatalf("%s: find patrol_findings: %v", phase, err)
		}
		for _, n := range []string{"actor", "reason", "disposed_at"} {
			if c.Fields.GetByName(n) != nil {
				t.Fatalf("%s: field %q must be absent after DOWN", phase, n)
			}
		}
	}

	// Fresh (all migrations up): the three provenance fields exist with their declared types.
	assertPresent("after up")

	mig := findMigration(t, "0016_patrol_findings_provenance")

	// DOWN drops all three columns.
	if err := mig.Down(app); err != nil {
		t.Fatalf("down: %v", err)
	}
	assertAbsent("after down")

	// UP re-adds them, restoring the same types/ceilings.
	if err := mig.Up(app); err != nil {
		t.Fatalf("up: %v", err)
	}
	assertPresent("after up->down->up")
}

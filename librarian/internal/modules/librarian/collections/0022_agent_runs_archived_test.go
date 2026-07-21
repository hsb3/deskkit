package collections

import (
	"testing"

	pbcore "github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

// archivedField returns the agent_runs.archived BoolField and whether it is present.
func archivedField(t *testing.T, app pbcore.App) (*pbcore.BoolField, bool) {
	t.Helper()
	c, err := app.FindCollectionByNameOrId("agent_runs")
	if err != nil {
		t.Fatalf("find agent_runs: %v", err)
	}
	bf, ok := c.Fields.GetByName("archived").(*pbcore.BoolField)
	return bf, ok
}

// TestMigration0022_AddsArchivedField proves the forward migration adds agent_runs.archived as a
// Bool, that a run created on a fresh store reads not-archived by default (no backfill needed), and
// that an up->down->up cycle removes then re-adds the field (idempotent, data-safe).
func TestMigration0022_AddsArchivedField(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("tests.NewTestApp: %v", err)
	}
	t.Cleanup(app.Cleanup)

	// Fresh (all migrations up): archived present as a Bool.
	if _, ok := archivedField(t, app); !ok {
		t.Fatal("fresh store: agent_runs.archived missing")
	}

	// A run created without setting archived reads false — the soft default the list relies on.
	c, err := app.FindCollectionByNameOrId("agent_runs")
	if err != nil {
		t.Fatalf("find agent_runs: %v", err)
	}
	run := pbcore.NewRecord(c)
	run.Set("trigger", "manual")
	run.Set("status", "succeeded")
	run.Set("input_summary", "a conversation")
	if err := app.Save(run); err != nil {
		t.Fatalf("save run: %v", err)
	}
	if run.GetBool("archived") {
		t.Error("a new run must default to not-archived")
	}

	mig := findMigration(t, "0022_agent_runs_archived")

	// DOWN removes the field.
	if err := mig.Down(app); err != nil {
		t.Fatalf("down: %v", err)
	}
	if _, ok := archivedField(t, app); ok {
		t.Fatal("after down, agent_runs.archived must be gone")
	}

	// UP re-adds it.
	if err := mig.Up(app); err != nil {
		t.Fatalf("up: %v", err)
	}
	if _, ok := archivedField(t, app); !ok {
		t.Fatal("after up->down->up, agent_runs.archived must be present again")
	}
}

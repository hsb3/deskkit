package collections

import (
	"strings"
	"testing"

	pbcore "github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

// textFieldMax returns the Max of a collection's text field (t.Fatal if it is not a TextField).
func textFieldMax(t *testing.T, app pbcore.App, coll, field string) int {
	t.Helper()
	c, err := app.FindCollectionByNameOrId(coll)
	if err != nil {
		t.Fatalf("find %s: %v", coll, err)
	}
	tf, ok := c.Fields.GetByName(field).(*pbcore.TextField)
	if !ok {
		t.Fatalf("%s.%s is not a TextField", coll, field)
	}
	return tf.Max
}

// capExpectations is the seven (collection, field) -> Max the migration must set.
var capExpectations = map[[2]string]int{
	{"patrol_findings", "detail"}:       50000,
	{"patrol_findings", "proposed_fix"}: 50000,
	{"patrol_log", "summary"}:           2000,
	{"adoption_log", "detail"}:          2000,
	{"agent_runs", "input_summary"}:     2000,
	{"agent_runs", "output_summary"}:    2000,
	{"agent_runs", "error"}:             2000,
}

// TestMigration0020_SetsExplicitCaps proves the forward migration sets the stated Max on all seven
// content fields, and an up->down->up cycle restores the implicit default then re-applies them.
func TestMigration0020_SetsExplicitCaps(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("tests.NewTestApp: %v", err)
	}
	t.Cleanup(app.Cleanup)

	for k, max := range capExpectations {
		if got := textFieldMax(t, app, k[0], k[1]); got != max {
			t.Fatalf("fresh store: %s.%s Max = %d, want %d", k[0], k[1], got, max)
		}
	}

	mig := findMigration(t, "0020_content_field_caps")

	// DOWN restores the implicit default (Max==0 → 5000 enforcement) on all seven.
	if err := mig.Down(app); err != nil {
		t.Fatalf("down: %v", err)
	}
	for k := range capExpectations {
		if got := textFieldMax(t, app, k[0], k[1]); got != 0 {
			t.Fatalf("after down: %s.%s Max = %d, want 0 (implicit default)", k[0], k[1], got)
		}
	}

	// UP re-applies the caps.
	if err := mig.Up(app); err != nil {
		t.Fatalf("up: %v", err)
	}
	for k, max := range capExpectations {
		if got := textFieldMax(t, app, k[0], k[1]); got != max {
			t.Fatalf("after up->down->up: %s.%s Max = %d, want %d", k[0], k[1], got, max)
		}
	}
}

// TestMigration0020_CapsAreEnforced_RedPreMigration is the red-able regression: it proves the caps
// are ENFORCED post-migration and would fail against the pre-migration schema (the implicit 5000
// default). For the WIDENED pair (detail @ 50000) a >5000 body saves only after the migration; for
// a TIGHTENED summary (@ 2000) a 2000<body<5000 payload is rejected only after the migration.
// Running 0020.Down() reverts to the implicit 5000 and flips every assertion — the in-process
// "must fail pre-migration" proof.
//
// (The issue body's "write a >5000 body to EACH" is only meaningful for the two widened fields:
// the five 2000-char caps are TIGHTER than the implicit 5000, so a >5000 body could never fit
// them. The tightened fields are proven by the mirror-image assertion — a between-cap payload
// rejected only post-migration — which carries the identical red-pre/green-post force.)
func TestMigration0020_CapsAreEnforced_RedPreMigration(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("tests.NewTestApp: %v", err)
	}
	t.Cleanup(app.Cleanup)

	body6k := strings.Repeat("x", 6000) // > 5000 implicit default, < 50000 widened cap
	body3k := strings.Repeat("y", 3000) // > 2000 tightened cap, < 5000 implicit default

	savePatrolFindingDetail := func(detail string) error {
		c, _ := app.FindCollectionByNameOrId("patrol_findings")
		r := pbcore.NewRecord(c)
		r.Set("rule", "R1")
		r.Set("detail", detail)
		r.Set("state", "flagged")
		r.Set("disposition", "open")
		return app.Save(r)
	}
	savePatrolLogSummary := func(summary string) error {
		c, _ := app.FindCollectionByNameOrId("patrol_log")
		r := pbcore.NewRecord(c)
		r.Set("run_id", "run-x")
		r.Set("summary", summary)
		return app.Save(r)
	}

	// POST-migration (fresh store, 0020 applied):
	//  - detail @ 50000 accepts a 6k body (would be rejected at the implicit 5000).
	if err := savePatrolFindingDetail(body6k); err != nil {
		t.Fatalf("post-migration: a 6000-char patrol_findings.detail must save (cap 50000): %v", err)
	}
	//  - summary @ 2000 REJECTS a 3k body (would save under the implicit 5000).
	if err := savePatrolLogSummary(body3k); err == nil {
		t.Fatal("post-migration: a 3000-char patrol_log.summary must be rejected (cap 2000), but it saved")
	}

	// Revert 0020 → the implicit 5000 default is back on all seven; every assertion flips (the
	// red-able proof that the migration, not pre-existing schema, is what enforces the caps).
	mig := findMigration(t, "0020_content_field_caps")
	if err := mig.Down(app); err != nil {
		t.Fatalf("down: %v", err)
	}
	//  - detail back at the implicit 5000: the same 6k body is now rejected.
	if err := savePatrolFindingDetail(body6k); err == nil {
		t.Fatal("pre-migration (post-down): a 6000-char detail must be rejected by the implicit 5000 cap, but it saved")
	}
	//  - summary back at the implicit 5000: the 3k body now saves.
	if err := savePatrolLogSummary(body3k); err != nil {
		t.Fatalf("pre-migration (post-down): a 3000-char summary must save under the implicit 5000 cap: %v", err)
	}
}

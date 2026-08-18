package store

import (
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"

	// Blank-import registers this project's Go migrations so tests.NewTestApp applies them.
	_ "github.com/hsb3/deskkit/internal/modules/librarian/collections"
)

const (
	deskAlpha = "desk-alpha"
	deskBeta  = "desk-beta"
)

func newGuardApp(t *testing.T) *tests.TestApp {
	t.Helper()
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("tests.NewTestApp: %v", err)
	}
	t.Cleanup(app.Cleanup)
	return app
}

// insertRow saves one record into collection with the given desk value (and a unique path when
// the collection requires it) so the guard has a row to consult.
func insertRow(t *testing.T, app core.App, collection, desk, path string) {
	t.Helper()
	col, err := app.FindCollectionByNameOrId(collection)
	if err != nil {
		t.Fatalf("find collection %q: %v", collection, err)
	}
	rec := core.NewRecord(col)
	rec.Set("desk", desk)
	if path != "" {
		rec.Set("path", path)
	}
	if err := app.Save(rec); err != nil {
		t.Fatalf("save %q row: %v", collection, err)
	}
}

func TestCheckDeskGuard_EmptyStorePasses(t *testing.T) {
	app := newGuardApp(t)
	if err := CheckDeskGuard(app, deskAlpha); err != nil {
		t.Fatalf("empty store must pass the guard: %v", err)
	}
}

func TestCheckDeskGuard_MatchingDeskPasses(t *testing.T) {
	app := newGuardApp(t)
	insertRow(t, app, "files", deskAlpha, "_meta/HANDOFF.md")
	if err := CheckDeskGuard(app, deskAlpha); err != nil {
		t.Fatalf("a store whose rows carry the configured desk must pass: %v", err)
	}
}

func TestCheckDeskGuard_MismatchOnFilesRowErrors(t *testing.T) {
	app := newGuardApp(t)
	insertRow(t, app, "files", deskAlpha, "_meta/HANDOFF.md")

	err := CheckDeskGuard(app, deskBeta)
	if err == nil {
		t.Fatalf("a store owned by %q must be refused when DESK_NAME is %q", deskAlpha, deskBeta)
	}
	msg := err.Error()
	if !strings.Contains(msg, deskAlpha) || !strings.Contains(msg, deskBeta) {
		t.Fatalf("error must name both the stored desk %q and the configured desk %q, got: %s",
			deskAlpha, deskBeta, msg)
	}
}

func TestCheckDeskGuard_MismatchDetectedViaPatrolLogWhenFilesEmpty(t *testing.T) {
	app := newGuardApp(t)
	// files is empty; the mismatch lives only in patrol_log — the guard must fall through to it.
	insertRow(t, app, "patrol_log", deskAlpha, "")

	err := CheckDeskGuard(app, deskBeta)
	if err == nil {
		t.Fatalf("mismatch recorded in patrol_log (files empty) must be refused")
	}
	if !strings.Contains(err.Error(), deskAlpha) || !strings.Contains(err.Error(), deskBeta) {
		t.Fatalf("error must name both desks, got: %s", err.Error())
	}
}

func TestCheckDeskGuard_MismatchDetectedViaAdoptionLogWhenFilesAndPatrolLogEmpty(t *testing.T) {
	app := newGuardApp(t)
	// files and patrol_log are both empty; the mismatch lives only in adoption_log — the guard
	// must fall through to the last collection in deskCarryingCollections, not just the first two.
	insertRow(t, app, "adoption_log", deskAlpha, "")

	err := CheckDeskGuard(app, deskBeta)
	if err == nil {
		t.Fatalf("mismatch recorded in adoption_log (files and patrol_log empty) must be refused")
	}
	if !strings.Contains(err.Error(), deskAlpha) || !strings.Contains(err.Error(), deskBeta) {
		t.Fatalf("error must name both desks, got: %s", err.Error())
	}
}

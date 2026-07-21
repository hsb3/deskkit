package prompt

import (
	"testing"

	"github.com/pocketbase/pocketbase/tests"

	// Blank-import registers this project's Go migrations so tests.NewTestApp's
	// RunAllMigrations() creates the prompts collection Seed writes into.
	_ "github.com/hsb3/desk-standard/librarian/internal/modules/librarian/collections"
)

// Seed materializes the editable system-prompt row. It runs from requireConfig (the shared
// one-shot entry path) as well as OnServe, so a CLI/MCP-only desk still gets the row instead
// of a silent no-op. This pins both first-run creation and idempotency on a fresh store.
func TestSeed_CreatesPromptRowThenIsIdempotent(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("tests.NewTestApp: %v", err)
	}
	t.Cleanup(app.Cleanup)

	// Fresh store: no seeded row yet.
	if _, err := app.FindFirstRecordByFilter("prompts", "key = 'librarian.system'"); err == nil {
		t.Fatalf("expected no seeded prompt row on a fresh store")
	}

	if err := Seed(app); err != nil {
		t.Fatalf("Seed: %v", err)
	}

	rec, err := app.FindFirstRecordByFilter("prompts", "key = 'librarian.system'")
	if err != nil {
		t.Fatalf("expected a seeded prompts row after Seed: %v", err)
	}
	if rec.GetString("content") != Embedded() {
		t.Fatalf("seeded content should equal the embedded default")
	}
	if !rec.GetBool("active") {
		t.Fatalf("seeded prompt should be active")
	}

	// Idempotent: a second Seed is a no-op — still exactly one row, unchanged id (GUI/REST
	// edits are never clobbered).
	if err := Seed(app); err != nil {
		t.Fatalf("second Seed: %v", err)
	}
	rows, err := app.FindRecordsByFilter("prompts", "key = 'librarian.system'", "", 0, 0)
	if err != nil {
		t.Fatalf("list prompts: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected exactly one prompts row after a repeat Seed, got %d", len(rows))
	}
	if rows[0].Id != rec.Id {
		t.Fatalf("second Seed must not replace the row (id changed %q → %q)", rec.Id, rows[0].Id)
	}
}

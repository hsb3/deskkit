package bootstrap

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/pocketbase/pocketbase/core"
)

// deskCarryingCollections are the collections whose rows carry a `desk` value (ADR 0002 §3),
// in the order the guard consults them: the first collection that holds any row decides.
var deskCarryingCollections = []string{"files", "patrol_log", "adoption_log"}

// CheckDeskGuard enforces the store-per-desk open-guard (ADR 0002 §3): on opening a store, if
// an existing row already carries a `desk` value different from the configured deskName, refuse
// to proceed. An empty store (no desk-carrying rows) passes — a first run has no rows by
// construction. The guard cannot prevent the data dir from being created (PocketBase bootstrap
// opens the DB before any RunE); it refuses to PROCEED, which is the enforceable boundary.
//
// `migrate` deliberately does NOT call this: it is schema-only and writes no desk rows, so a
// migrate against a store belonging to another desk is harmless (see cmd/pocket-librarian).
func CheckDeskGuard(app core.App, deskName string) error {
	for _, coll := range deskCarryingCollections {
		rec, err := app.FindFirstRecordByFilter(coll, "id != ''")
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				continue // empty collection — consult the next one
			}
			return fmt.Errorf("desk guard: query %q: %w", coll, err)
		}
		// First collection with any row decides (ADR 0002 §3). A row's desk is written from
		// cfg.DeskName by sweep/patrol/apply_fix, so a differing value means this store already
		// belongs to another desk.
		if stored := rec.GetString("desk"); stored != "" && stored != deskName {
			return fmt.Errorf("store at %s belongs to desk %q but DESK_NAME is %q",
				app.DataDir(), stored, deskName)
		}
		return nil
	}
	return nil
}

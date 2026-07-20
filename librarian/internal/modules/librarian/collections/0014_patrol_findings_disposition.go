package collections

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Extends patrol_findings with a "disposition" select — a supervisor's triage decision that is
// ORTHOGONAL to the finding's `state` (flagged/fixed/resolved) (dismissed retired in 0015). A finding stays
// `flagged` while it is dispositioned acknowledged/triaged/wont_fix, so the two axes do not
// collide: `state` tracks the finding's lifecycle, `disposition` tracks whether a human has
// chosen to defer/silence it. The default `query findings` view is live-only
// (disposition='open'); a disposed finding survives re-patrol because disposing sets ONLY
// disposition (state stays flagged), so the next patrol dedupes the existing row and its
// disposition rides along untouched.
//
// FORWARD: add the field (guard-before-add so a re-run is a no-op), then BACKFILL every existing
// row whose disposition is empty to 'open'. PocketBase leaves a new select column '' on existing
// rows; without the backfill those pre-existing findings would vanish from the disposition='open'
// default filter. The backfill is idempotent (only touches empty rows).
//
// DOWN: drop the disposition field entirely (guard-before-remove). No enum-value data remap is
// needed — unlike 0010 (which removed a single value from a KEPT field), here the whole column
// is dropped, so no row can be left holding a value outside a reverted enum.
func init() {
	m.Register(func(app core.App) error {
		c, err := app.FindCollectionByNameOrId("patrol_findings")
		if err != nil {
			return err
		}

		if c.Fields.GetByName("disposition") == nil {
			c.Fields.Add(&core.SelectField{
				Name:      "disposition",
				MaxSelect: 1,
				Values:    []string{"open", "acknowledged", "triaged", "wont_fix"},
			})
			if err := app.Save(c); err != nil {
				return err
			}
		}

		// Backfill existing rows: a new select column is left empty on pre-existing rows, but the
		// default findings filter is disposition='open', so an un-backfilled finding would silently
		// disappear. Set only the empty ones (idempotent — a re-run finds none). Fetch in id-sorted
		// pages rather than loading the whole collection at once; setting `disposition` changes
		// neither membership nor order of the unfiltered id-sorted set, so the offsets stay stable
		// while rows are mutated between pages.
		const pageSize = 500
		for offset := 0; ; offset += pageSize {
			recs, err := app.FindRecordsByFilter("patrol_findings", "", "id", pageSize, offset)
			if err != nil {
				return err
			}
			for _, r := range recs {
				if r.GetString("disposition") == "" {
					r.Set("disposition", "open")
					if err := app.Save(r); err != nil {
						return err
					}
				}
			}
			if len(recs) < pageSize {
				return nil
			}
		}
	}, func(app core.App) error {
		c, err := app.FindCollectionByNameOrId("patrol_findings")
		if err != nil {
			return nil
		}
		if c.Fields.GetByName("disposition") != nil {
			c.Fields.RemoveByName("disposition")
			return app.Save(c)
		}
		return nil
	})
}

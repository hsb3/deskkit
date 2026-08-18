package collections

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Retires the dead "dismissed" value from patrol_findings.state. The disposition sub-machine
// (0014) superseded the plain-dismiss path, and no setter anywhere writes state="dismissed"
// (verified repo-wide): the value only ever implied an unbuilt feature. The one human-judgment
// axis is now `disposition`; `state` is the finding lifecycle (flagged/fixed/resolved).
//
// Retiring an enum value INVERTS the 0010/0012 precedent: there the ADD is forward and the
// SHRINK (with its data-first remap) is the down step. Here the SHRINK is the FORWARD step, so
// the data-first remap lives in the FORWARD migration — remap any residual dismissed rows to
// flagged BEFORE narrowing the enum, so a shrink never leaves a row holding a value outside its
// (now-narrowed) enum. The DOWN merely re-adds "dismissed" (a grow needs no data remap; the
// value's slice position is cosmetic). Both directions are guarded so a re-run is a no-op.
func init() {
	m.Register(func(app core.App) error {
		c, err := app.FindCollectionByNameOrId("patrol_findings")
		if err != nil {
			return err
		}

		// Data migration FIRST: remap any residual dismissed rows to flagged before "dismissed" is
		// removed from the state enum below, so the shrink never leaves a row outside its enum. Zero
		// rows are expected (no setter writes dismissed), so this is defensive; it runs inside the
		// same migration transaction as the schema change (MigrationsRunner wraps Up() in one tx).
		dismissed, err := app.FindRecordsByFilter("patrol_findings", "state = 'dismissed'", "", 0, 0)
		if err != nil {
			return err
		}
		for _, r := range dismissed {
			r.Set("state", "flagged")
			if err := app.Save(r); err != nil {
				return err
			}
		}

		if state, ok := c.Fields.GetByName("state").(*core.SelectField); ok {
			kept := make([]string, 0, len(state.Values))
			for _, v := range state.Values {
				if v != "dismissed" {
					kept = append(kept, v)
				}
			}
			state.Values = kept
			return app.Save(c)
		}
		return nil
	}, func(app core.App) error {
		c, err := app.FindCollectionByNameOrId("patrol_findings")
		if err != nil {
			return nil
		}
		if state, ok := c.Fields.GetByName("state").(*core.SelectField); ok {
			has := false
			for _, v := range state.Values {
				if v == "dismissed" {
					has = true
					break
				}
			}
			if !has {
				state.Values = append(state.Values, "dismissed")
				return app.Save(c)
			}
		}
		return nil
	})
}

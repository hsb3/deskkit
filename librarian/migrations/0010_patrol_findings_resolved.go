package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Extends patrol_findings with a terminal "resolved" state and a resolved_run text field.
// This lets a FULL-desk patrol close findings whose (path, rule) stops firing (state ->
// resolved, resolved_run = the resolving run id), giving flagged findings a deterministic
// exit other than apply-fix. A scoped patrol resolves only findings within its scope. Both
// changes are applied idempotently so a re-run of the migration is a no-op.
func init() {
	m.Register(func(app core.App) error {
		c, err := app.FindCollectionByNameOrId("patrol_findings")
		if err != nil {
			return err
		}

		if state, ok := c.Fields.GetByName("state").(*core.SelectField); ok {
			has := false
			for _, v := range state.Values {
				if v == "resolved" {
					has = true
					break
				}
			}
			if !has {
				state.Values = append(state.Values, "resolved")
			}
		}

		if c.Fields.GetByName("resolved_run") == nil {
			c.Fields.Add(&core.TextField{Name: "resolved_run"})
		}

		return app.Save(c)
	}, func(app core.App) error {
		c, err := app.FindCollectionByNameOrId("patrol_findings")
		if err != nil {
			return nil
		}

		// Data migration FIRST: remap any resolved rows back to flagged before "resolved" is
		// removed from the state enum below, so a rollback never leaves a row holding a value
		// outside its (now-reverted) enum. Runs inside the same migration transaction as the
		// schema change (PocketBase's MigrationsRunner wraps Down() in one transaction).
		resolvedRecs, err := app.FindRecordsByFilter("patrol_findings", "state = 'resolved'", "", 0, 0)
		if err != nil {
			return err
		}
		for _, r := range resolvedRecs {
			r.Set("state", "flagged")
			r.Set("resolved_run", "")
			if err := app.Save(r); err != nil {
				return err
			}
		}

		c.Fields.RemoveByName("resolved_run")
		if state, ok := c.Fields.GetByName("state").(*core.SelectField); ok {
			kept := make([]string, 0, len(state.Values))
			for _, v := range state.Values {
				if v != "resolved" {
					kept = append(kept, v)
				}
			}
			state.Values = kept
		}
		return app.Save(c)
	})
}

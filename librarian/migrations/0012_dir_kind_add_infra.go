package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Adds an "infra" value to the files.dir_kind select enum. Non-entity infrastructure that
// lives in dotted dirs (.claude, .agents, .github, ...) was previously bucketed as "other" and
// so surfaced in the `orphans` view as if it were misfiled desk content; classifying it as its
// own dir_kind lets sweep label it and the orphans query exclude it (spec §5.1/§5.6).
// Applies to existing stores on the next migrate, and to fresh stores after 0001 (which now
// carries "infra" in its source decl too). Idempotent: re-running is a no-op.
func init() {
	m.Register(func(app core.App) error {
		c, err := app.FindCollectionByNameOrId("files")
		if err != nil {
			return err
		}
		if dk, ok := c.Fields.GetByName("dir_kind").(*core.SelectField); ok {
			has := false
			for _, v := range dk.Values {
				if v == "infra" {
					has = true
					break
				}
			}
			if !has {
				dk.Values = append(dk.Values, "infra")
			}
		}
		return app.Save(c)
	}, func(app core.App) error {
		c, err := app.FindCollectionByNameOrId("files")
		if err != nil {
			return nil
		}
		// Data migration FIRST: remap any rows currently labelled "infra" back to "other" before
		// "infra" is removed from the enum below, so a rollback never leaves a row holding a value
		// outside its (now-reverted) enum (mirrors 0010's resolved->flagged down migration; the
		// runner wraps Down() in one transaction). A subsequent sweep re-derives dir_kind anyway.
		infraRecs, err := app.FindRecordsByFilter("files", "dir_kind = 'infra'", "", 0, 0)
		if err != nil {
			return err
		}
		for _, r := range infraRecs {
			r.Set("dir_kind", "other")
			if err := app.Save(r); err != nil {
				return err
			}
		}
		if dk, ok := c.Fields.GetByName("dir_kind").(*core.SelectField); ok {
			kept := make([]string, 0, len(dk.Values))
			for _, v := range dk.Values {
				if v != "infra" {
					kept = append(kept, v)
				}
			}
			dk.Values = kept
		}
		return app.Save(c)
	})
}

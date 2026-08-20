package collections

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"

	"github.com/hsb3/deskkit/internal/core/settings"
)

// Adds settings.sticky_finder — the browser's "keep the finder minimised between items"
// preference. It rides the settings singleton because the preference belongs to the DESK, not to
// the browser that set it: a second machine opening the same desk gets the same behaviour.
//
// A forward migration rather than an edit to 0024: an applied migration is never re-run, so
// editing it would hand the field to fresh stores only.
//
// The ruled default is ON, and a PocketBase bool is false when absent — so the up also SETS the
// seeded row true. Both store ages land on the same value: a fresh store because 0024 seeds the
// row before this runs, an existing store because the row is already there. Idempotent:
// guard-before-add makes a re-run a no-op. DOWN removes the field.
func init() {
	m.Register(func(app core.App) error {
		c, err := app.FindCollectionByNameOrId(settings.Collection)
		if err != nil {
			return err
		}
		if c.Fields.GetByName(settings.FieldStickyFinder) == nil {
			c.Fields.Add(&core.BoolField{Name: settings.FieldStickyFinder})
			if err := app.Save(c); err != nil {
				return err
			}
		}
		rec, err := app.FindRecordById(settings.Collection, settings.RecordID)
		if err != nil {
			// No seeded row to carry the default. Nothing to do rather than an error: the
			// collection exists, and a reader that finds no row already means "nothing stored".
			return nil
		}
		rec.Set(settings.FieldStickyFinder, true)
		return app.Save(rec)
	}, func(app core.App) error {
		c, err := app.FindCollectionByNameOrId(settings.Collection)
		if err != nil {
			return nil
		}
		if c.Fields.GetByName(settings.FieldStickyFinder) != nil {
			c.Fields.RemoveByName(settings.FieldStickyFinder)
			return app.Save(c)
		}
		return nil
	})
}

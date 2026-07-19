package collections

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// patrol_log (spec §4.4). files_swept / findings_new are non-integer-constrained numbers
// (onlyInt false, the default).
func init() {
	m.Register(func(app core.App) error {
		c := core.NewBaseCollection("patrol_log")
		c.Id = "pbc_3052838673"
		c.Fields.Add(&core.TextField{Name: "run_id"})
		c.Fields.Add(&core.TextField{Name: "desk"})
		c.Fields.Add(&core.DateField{Name: "started"})
		c.Fields.Add(&core.DateField{Name: "finished"})
		c.Fields.Add(&core.NumberField{Name: "files_swept"})
		c.Fields.Add(&core.NumberField{Name: "findings_new"})
		c.Fields.Add(&core.TextField{Name: "summary"})
		return app.Save(c)
	}, func(app core.App) error {
		if c, _ := app.FindCollectionByNameOrId("patrol_log"); c != nil {
			return app.Delete(c)
		}
		return nil
	})
}

package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// adoption_log (spec §4.6). Event enum order per §4.6.
func init() {
	m.Register(func(app core.App) error {
		c := core.NewBaseCollection("adoption_log")
		c.Id = "pbc_2776405824"
		c.Fields.Add(&core.DateField{Name: "date"})
		c.Fields.Add(&core.TextField{Name: "desk"})
		c.Fields.Add(&core.SelectField{Name: "event", MaxSelect: 1,
			Values: []string{"patrol", "fix", "revert", "false_positive", "friction", "note"}})
		c.Fields.Add(&core.TextField{Name: "detail"})
		return app.Save(c)
	}, func(app core.App) error {
		if c, _ := app.FindCollectionByNameOrId("adoption_log"); c != nil {
			return app.Delete(c)
		}
		return nil
	})
}

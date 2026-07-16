package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

// prompts (spec §4.10) — the editable, versioned system prompt data surface. Stable id
// assigned so the seeded embedded default is reproducible across rebuilds. No relations;
// order-independent. The default ROW is seeded at first run (prompt.Seed), not here —
// mirroring the .librarian-ignore auto-create.
func init() {
	m.Register(func(app core.App) error {
		c := core.NewBaseCollection("prompts")
		c.Id = "pbc_1968329054"
		c.Fields.Add(&core.TextField{Name: "key", Required: true})
		c.Fields.Add(&core.TextField{Name: "name"})
		c.Fields.Add(&core.TextField{Name: "content"})
		c.Fields.Add(&core.NumberField{Name: "version", OnlyInt: true, Min: types.Pointer(0.0)})
		c.Fields.Add(&core.BoolField{Name: "active"})
		c.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
		c.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
		c.AddIndex("idx_prompts_key_active", false, "key, active", "")
		return app.Save(c)
	}, func(app core.App) error {
		if c, _ := app.FindCollectionByNameOrId("prompts"); c != nil {
			return app.Delete(c)
		}
		return nil
	})
}

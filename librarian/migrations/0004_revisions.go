package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// revisions — the record-original-first ledger (spec §4.5, decision 0014). Relation ->
// patrol_findings; patrol_findings (0002) migrates first.
func init() {
	m.Register(func(app core.App) error {
		findings, err := app.FindCollectionByNameOrId("patrol_findings")
		if err != nil {
			return err
		}
		c := core.NewBaseCollection("revisions")
		c.Id = "pbc_3986342941"
		c.Fields.Add(&core.TextField{Name: "path", Required: true})
		c.Fields.Add(&core.SelectField{Name: "action", MaxSelect: 1, Values: []string{"edit", "move", "delete"}})
		c.Fields.Add(&core.TextField{Name: "original_content"}) // Max widened off the 5000 default in 0011
		c.Fields.Add(&core.TextField{Name: "original_checksum"})
		c.Fields.Add(&core.TextField{Name: "new_path"})
		c.Fields.Add(&core.RelationField{Name: "finding", CollectionId: findings.Id, MaxSelect: 1, MinSelect: 0, CascadeDelete: false})
		c.Fields.Add(&core.BoolField{Name: "applied"})
		c.Fields.Add(&core.BoolField{Name: "restored"})
		c.Fields.Add(&core.TextField{Name: "run_id"})
		c.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
		c.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
		return app.Save(c)
	}, func(app core.App) error {
		if c, _ := app.FindCollectionByNameOrId("revisions"); c != nil {
			return app.Delete(c)
		}
		return nil
	})
}

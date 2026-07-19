package collections

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// patrol_findings (spec §4.3). Dedupe key (path, rule, checksum) is enforced in
// application code, NOT a DB index. Relation -> files; files (0001) migrates first.
func init() {
	m.Register(func(app core.App) error {
		files, err := app.FindCollectionByNameOrId("files")
		if err != nil {
			return err
		}
		c := core.NewBaseCollection("patrol_findings")
		c.Id = "pbc_134268848"
		c.Fields.Add(&core.RelationField{Name: "file", CollectionId: files.Id, MaxSelect: 1, MinSelect: 0, CascadeDelete: false})
		c.Fields.Add(&core.TextField{Name: "rule"})
		c.Fields.Add(&core.SelectField{Name: "severity", MaxSelect: 1, Values: []string{"mechanical", "judgment"}})
		c.Fields.Add(&core.TextField{Name: "detail"})
		c.Fields.Add(&core.TextField{Name: "proposed_fix"})
		c.Fields.Add(&core.SelectField{Name: "state", MaxSelect: 1, Values: []string{"flagged", "dismissed", "fixed"}})
		c.Fields.Add(&core.TextField{Name: "patrol_run"})
		c.Fields.Add(&core.TextField{Name: "checksum"})
		return app.Save(c)
	}, func(app core.App) error {
		if c, _ := app.FindCollectionByNameOrId("patrol_findings"); c != nil {
			return app.Delete(c)
		}
		return nil
	})
}

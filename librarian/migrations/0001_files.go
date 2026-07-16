package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// files — one row per file under DESK_ROOT (spec §4.2). Stable id preserved for rebuild
// reproducibility (§4.1). All api rules left nil => superuser-only.
func init() {
	m.Register(func(app core.App) error {
		files := core.NewBaseCollection("files")
		files.Id = "pbc_3446931122"
		files.Fields.Add(&core.TextField{Name: "path", Required: true})
		files.Fields.Add(&core.TextField{Name: "desk"})
		files.Fields.Add(&core.TextField{Name: "entity_type"})
		files.Fields.Add(&core.SelectField{Name: "dir_kind", MaxSelect: 1,
			Values: []string{"decisions", "tasks", "analyses", "journal", "meta", "memory", "root", "other"}})
		files.Fields.Add(&core.TextField{Name: "status"})
		files.Fields.Add(&core.TextField{Name: "synopsis"})
		files.Fields.Add(&core.TextField{Name: "origin"})
		files.Fields.Add(&core.TextField{Name: "graduated_to"})
		files.Fields.Add(&core.TextField{Name: "checksum"})
		files.Fields.Add(&core.TextField{Name: "git_last_commit"})
		files.Fields.Add(&core.TextField{Name: "fm_created"})
		files.Fields.Add(&core.TextField{Name: "fm_updated"})
		files.Fields.Add(&core.DateField{Name: "last_seen"})
		files.Fields.Add(&core.BoolField{Name: "deleted"})
		files.AddIndex("idx_files_path", true, "path", "")
		return app.Save(files)
	}, func(app core.App) error {
		if c, _ := app.FindCollectionByNameOrId("files"); c != nil {
			return app.Delete(c)
		}
		return nil
	})
}

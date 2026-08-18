package migrate

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// module_schema_versions — the core-owned meta collection StampModules writes to (§2.7/§2.8):
// one row per enabled module, recording the highest migration sequence PocketBase's own
// _migrations ledger shows that module has applied. Runs unconditionally (blank-imported by
// main alongside the librarian's collections), before any module-specific migration, so the
// collection always exists by the time StampModules runs. Stable id preserved like every other
// migration in this codebase (spec §4.1).
func init() {
	m.Register(func(app core.App) error {
		c := core.NewBaseCollection(moduleSchemaVersionsCollection)
		c.Id = "pbc_0000module0schema0versions"
		c.Fields.Add(&core.TextField{Name: "module", Required: true})
		c.Fields.Add(&core.NumberField{Name: "version", OnlyInt: true})
		c.Fields.Add(&core.DateField{Name: "applied_at"})
		c.AddIndex("idx_module_schema_versions_module", true, "module", "")
		return app.Save(c)
	}, func(app core.App) error {
		if c, _ := app.FindCollectionByNameOrId(moduleSchemaVersionsCollection); c != nil {
			return app.Delete(c)
		}
		return nil
	})
}

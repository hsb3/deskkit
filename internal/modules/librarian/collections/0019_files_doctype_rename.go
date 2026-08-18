package collections

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Renames files.entity_type -> files.doctype (ADR 0017). The column holds a doctype string (the
// frontmatter `type` value: decision, task, analysis, journal, ...), populated by sweep from the
// `type` key. It shared a NAME — but nothing else — with schema/doctypes.yaml's unrelated
// `entity_type` enum (a person/company classification bound to a different frontmatter field on
// `type: entity` docs); renaming the DB column clears that confusion trap. Nothing read the
// column expecting the schema's enum, so this is a hygiene rename, not a correctness fix.
//
// The rename mutates the EXISTING field object in place (stable field id), so PocketBase emits a
// SQLite column rename, not drop+add — row data is preserved in both directions. Applies to
// existing stores on the next migrate; on a fresh store 0001 creates `entity_type` and this
// migration renames it. Idempotent: guard-before-rename makes a re-run (or a run where the target
// already exists) a no-op. DOWN renames `doctype` back to `entity_type`.
func init() {
	m.Register(func(app core.App) error {
		return renameFilesField(app, "entity_type", "doctype")
	}, func(app core.App) error {
		return renameFilesField(app, "doctype", "entity_type")
	})
}

// renameFilesField renames a field on the files collection in place — preserving its field id and
// therefore its column data — guarding on both the target (already renamed) and the source
// (renamed/removed by another migration) so a re-run is a no-op rather than an error.
func renameFilesField(app core.App, from, to string) error {
	c, err := app.FindCollectionByNameOrId("files")
	if err != nil {
		return err
	}
	if c.Fields.GetByName(to) != nil {
		return nil // already renamed
	}
	f := c.Fields.GetByName(from)
	if f == nil {
		return nil // nothing to rename
	}
	f.SetName(to)
	return app.Save(c)
}

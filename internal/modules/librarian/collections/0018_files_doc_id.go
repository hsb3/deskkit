package collections

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Adds files.doc_id — the optional document-identity primitive (ADR 0017). A desk document may
// carry a frontmatter `id:` key; sweep stores it here and matches an existing row by doc_id
// BEFORE falling back to path, so a rename that carries the same id updates the SAME record at
// its new path instead of soft-deleting the old path and inserting a fresh row — rename stops
// discarding history. The identity is re-derivable by a fresh sweep from the desk tree alone
// (files-are-truth, decision 0009), so it survives a store rebuild.
//
// The DB column is `doc_id`, never `id`: PocketBase reserves the field name `id` for the
// collection's own system primary key, so a second field named `id` fails collection validation
// (validation_duplicated_field_name). The frontmatter KEY stays `id` (no collision at the YAML
// level). Max is explicit (an id is a short, bounded token), per the repo's explicit-Max
// convention (0011/0013). Applies to existing stores on the next migrate, and to fresh stores
// after 0001 creates `files`. Idempotent: guard-before-add makes a re-run a no-op. DOWN removes
// the field.
func init() {
	m.Register(func(app core.App) error {
		c, err := app.FindCollectionByNameOrId("files")
		if err != nil {
			return err
		}
		if c.Fields.GetByName("doc_id") == nil {
			c.Fields.Add(&core.TextField{Name: "doc_id", Max: 200})
			return app.Save(c)
		}
		return nil
	}, func(app core.App) error {
		c, err := app.FindCollectionByNameOrId("files")
		if err != nil {
			return nil
		}
		if c.Fields.GetByName("doc_id") != nil {
			c.Fields.RemoveByName("doc_id")
			return app.Save(c)
		}
		return nil
	})
}

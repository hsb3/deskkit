package collections

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Adds files.content — the swept file body, stored so a session can RETRIEVE and SEARCH indexed
// desk content through the query tool (the `content` and `search` kinds, §5.6). Before this, sweep
// used a file's raw bytes only to compute a checksum and parse frontmatter, never persisting the
// body, so nothing indexed was retrievable through any tool surface.
//
// The body is capped at 1,000,000 characters (an explicit Max, per the repo's explicit-Max
// convention 0011/0013 — a bare TextField silently caps at PocketBase's implicit 5,000 default and
// would reject longer bodies). Sweep truncates rune-safe to that cap, only indexes UTF-8 text, and
// never indexes a file living under the desk's configured SECRETS_DIR (the secret-home boundary is
// mirrored from the sweep exclusion logic). PocketBase's TextField Max is measured in runes, so a
// 1,000,000-rune body validates exactly at the cap.
//
// content is re-derivable by a fresh sweep from the desk tree alone — the store is disposable and
// desk files remain the source of truth (files-are-truth, decision 0009) — so it survives a store
// rebuild. Applies to existing stores on the next migrate, and to fresh stores after 0001 creates
// `files`. Idempotent: guard-before-add makes a re-run a no-op. DOWN removes the field.
func init() {
	m.Register(func(app core.App) error {
		c, err := app.FindCollectionByNameOrId("files")
		if err != nil {
			return err
		}
		if c.Fields.GetByName("content") == nil {
			c.Fields.Add(&core.TextField{Name: "content", Max: 1000000})
			return app.Save(c)
		}
		return nil
	}, func(app core.App) error {
		c, err := app.FindCollectionByNameOrId("files")
		if err != nil {
			return nil
		}
		if c.Fields.GetByName("content") != nil {
			c.Fields.RemoveByName("content")
			return app.Save(c)
		}
		return nil
	})
}

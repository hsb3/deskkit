// Package prompt owns the librarian system prompt as DATA, not compiled-in code
// (spec §4.10/§6.1). This spine slice implements first-run seeding: the //go:embed'd
// default (templates.SystemPrompt) is written into the prompts collection on first serve.
//
// The run-time RESOLVER (systemPrompt: load the active row, fall back to the embedded
// default, interpolate DESK_NAME/paths) is part of the agent-loop slice, not the spine —
// it ADDS to this package (a Resolve func) without changing Seed.
package prompt

import (
	"github.com/pocketbase/pocketbase/core"

	"github.com/example/pocket-librarian/templates"
)

// Embedded returns the verbatim embedded-default system prompt (spec §6.1), which is both
// the fallback and the first-run seed. It names no person, org, repo, or issue.
func Embedded() string { return templates.SystemPrompt }

// Seed inserts the embedded default into the prompts collection on first run, if and only
// if no row already exists for key "librarian.system" (spec §4.10 "Seeding & load",
// mirroring the .librarian-ignore auto-create). Idempotent: a second call is a no-op once
// a row exists. GUI/REST edits are never clobbered.
func Seed(app core.App) error {
	// Already seeded? (any row for the key)
	if _, err := app.FindFirstRecordByFilter("prompts", "key = 'librarian.system'"); err == nil {
		return nil
	}
	coll, err := app.FindCollectionByNameOrId("prompts")
	if err != nil {
		return err
	}
	rec := core.NewRecord(coll)
	rec.Set("key", "librarian.system")
	rec.Set("name", "Librarian system prompt")
	rec.Set("content", templates.SystemPrompt)
	rec.Set("version", 1)
	rec.Set("active", true)
	return app.Save(rec)
}

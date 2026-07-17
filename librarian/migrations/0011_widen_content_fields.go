package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// widenedContentMax lifts the content-bearing text fields off PocketBase's implicit 5,000-char
// default (a bare TextField with Max==0 validates at 5000; see core.TextField.Max). These three
// fields hold full document/transcript bodies, not summaries, so the default silently truncates
// — most damagingly on `revisions.original_content`, which IS the record-original-first safety
// boundary (decision 0014): no write may follow a failed original-record, so a desk file over
// ~5 KB could not be fixed at all. The value is a large finite ceiling — far beyond any realistic
// markdown desk file or agent-transcript message — chosen to remove the failure mode without
// leaving the field unbounded.
const widenedContentMax = 50_000_000

// contentTextFields are the (collection, field) pairs whose bodies are unbounded by intent:
//   - revisions.original_content — the byte-exact recorded original of a desk file (§5.4).
//   - messages.content           — full agent-transcript messages, incl. tool outputs that embed
//     file contents (input/output_summary on agent_runs stay the bounded summaries; §6.5).
//   - prompts.content            — the editable system prompt, which GUI/REST edits can grow (§4.10).
var contentTextFields = []struct{ coll, field string }{
	{"revisions", "original_content"},
	{"messages", "content"},
	{"prompts", "content"},
}

// Widens the content-bearing text fields to widenedContentMax. Idempotent (re-running sets the
// same Max); applies to existing stores on the next migrate, and to fresh stores after the
// collections above are created (0004/0007/0009).
func init() {
	m.Register(func(app core.App) error {
		return setContentMax(app, widenedContentMax)
	}, func(app core.App) error {
		// Down: restore the implicit default (Max==0 → 5000). Reverting the schema cap never
		// re-validates existing rows, so oversized recorded originals survive a rollback intact.
		return setContentMax(app, 0)
	})
}

func setContentMax(app core.App, max int) error {
	for _, f := range contentTextFields {
		c, err := app.FindCollectionByNameOrId(f.coll)
		if err != nil {
			return err
		}
		tf, ok := c.Fields.GetByName(f.field).(*core.TextField)
		if !ok {
			continue // field renamed/removed by a later migration — nothing to widen
		}
		tf.Max = max
		if err := app.Save(c); err != nil {
			return err
		}
	}
	return nil
}

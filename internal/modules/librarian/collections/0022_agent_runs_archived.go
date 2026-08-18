package collections

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Adds agent_runs.archived — a soft-archive flag for chat conversations. The sessions manager hides
// archived conversations from the resume list by default and can reveal them to unarchive; archiving
// is reversible and NEVER touches the conversation's messages. It is deliberately distinct from
// DeleteConversation's HARD delete (which cascades the run's message rows away): archive organizes,
// delete discards. A Bool defaults to false, so every existing run reads as not-archived after this
// migration with no data backfill. Applies to existing stores on the next migrate, and to fresh
// stores after 0006 creates agent_runs. Idempotent: guard-before-add makes a re-run a no-op. DOWN
// removes the field.
func init() {
	m.Register(func(app core.App) error {
		c, err := app.FindCollectionByNameOrId("agent_runs")
		if err != nil {
			return err
		}
		if c.Fields.GetByName("archived") == nil {
			c.Fields.Add(&core.BoolField{Name: "archived"})
			return app.Save(c)
		}
		return nil
	}, func(app core.App) error {
		c, err := app.FindCollectionByNameOrId("agent_runs")
		if err != nil {
			return nil
		}
		if c.Fields.GetByName("archived") != nil {
			c.Fields.RemoveByName("archived")
			return app.Save(c)
		}
		return nil
	})
}

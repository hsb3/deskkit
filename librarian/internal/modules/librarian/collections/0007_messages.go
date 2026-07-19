package collections

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

// messages (spec §4.7) — the ReAct transcript. The `run` relation targets the agent_runs
// record id (never run_label). Unique (run, seq) index guards a retried persist from
// duplicating a loop step. agent_runs (0006) migrates first.
func init() {
	m.Register(func(app core.App) error {
		runs, err := app.FindCollectionByNameOrId("agent_runs")
		if err != nil {
			return err
		}
		c := core.NewBaseCollection("messages")
		c.Fields.Add(&core.RelationField{Name: "run", CollectionId: runs.Id, MaxSelect: 1, MinSelect: 1, CascadeDelete: true})
		c.Fields.Add(&core.NumberField{Name: "seq", OnlyInt: true, Min: types.Pointer(0.0)})
		c.Fields.Add(&core.SelectField{Name: "role", MaxSelect: 1, Values: []string{"system", "user", "assistant", "tool"}})
		c.Fields.Add(&core.TextField{Name: "content"}) // Max widened off the 5000 default in 0011
		c.Fields.Add(&core.JSONField{Name: "tool_calls"})
		c.Fields.Add(&core.TextField{Name: "tool_call_id"})
		c.Fields.Add(&core.TextField{Name: "tool_name"})
		c.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
		c.AddIndex("idx_messages_run_seq", true, "run, seq", "")
		c.AddIndex("idx_messages_run", false, "run", "")
		return app.Save(c)
	}, func(app core.App) error {
		if c, _ := app.FindCollectionByNameOrId("messages"); c != nil {
			return app.Delete(c)
		}
		return nil
	})
}

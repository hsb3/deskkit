package collections

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

// agent_runs (spec §4.8) — MUST migrate before messages/tasks, which both relate to it.
// run_label is a display label only; the messages.run relation targets the record's
// system id, never run_label (§4.8/§6.5).
func init() {
	m.Register(func(app core.App) error {
		c := core.NewBaseCollection("agent_runs")
		c.Fields.Add(&core.SelectField{Name: "trigger", MaxSelect: 1, Values: []string{"hook", "cron", "manual", "task"}})
		c.Fields.Add(&core.SelectField{Name: "status", MaxSelect: 1, Values: []string{"running", "succeeded", "failed", "blocked"}})
		c.Fields.Add(&core.TextField{Name: "provider"})
		c.Fields.Add(&core.TextField{Name: "model"})
		c.Fields.Add(&core.TextField{Name: "run_label"})
		c.Fields.Add(&core.TextField{Name: "input_summary"})
		c.Fields.Add(&core.TextField{Name: "output_summary"})
		c.Fields.Add(&core.NumberField{Name: "step_count", OnlyInt: true, Min: types.Pointer(0.0)})
		c.Fields.Add(&core.TextField{Name: "error"})
		c.Fields.Add(&core.DateField{Name: "started"})
		c.Fields.Add(&core.DateField{Name: "finished"})
		c.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
		c.AddIndex("idx_agent_runs_status", false, "status", "")
		return app.Save(c)
	}, func(app core.App) error {
		if c, _ := app.FindCollectionByNameOrId("agent_runs"); c != nil {
			return app.Delete(c)
		}
		return nil
	})
}

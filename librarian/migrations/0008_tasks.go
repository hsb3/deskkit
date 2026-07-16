package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// tasks (spec §4.9) — the wake queue (Phase 2). result relation -> agent_runs; agent_runs
// (0006) migrates first. priority defaults to the zero value (0).
func init() {
	m.Register(func(app core.App) error {
		runs, err := app.FindCollectionByNameOrId("agent_runs")
		if err != nil {
			return err
		}
		c := core.NewBaseCollection("tasks")
		c.Fields.Add(&core.SelectField{Name: "kind", MaxSelect: 1,
			Values: []string{"sweep", "patrol", "propose_fix", "apply_fix", "restore", "query", "custom"}})
		c.Fields.Add(&core.JSONField{Name: "payload"})
		c.Fields.Add(&core.SelectField{Name: "state", MaxSelect: 1,
			Values: []string{"queued", "claimed", "done", "failed", "deferred"}})
		c.Fields.Add(&core.NumberField{Name: "priority", OnlyInt: true})
		c.Fields.Add(&core.TextField{Name: "source"})
		c.Fields.Add(&core.DateField{Name: "claimed_at"})
		c.Fields.Add(&core.DateField{Name: "finished_at"})
		c.Fields.Add(&core.RelationField{Name: "result", CollectionId: runs.Id, MaxSelect: 1, MinSelect: 0, CascadeDelete: false})
		c.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
		c.AddIndex("idx_tasks_state_priority", false, "state, priority", "")
		return app.Save(c)
	}, func(app core.App) error {
		if c, _ := app.FindCollectionByNameOrId("tasks"); c != nil {
			return app.Delete(c)
		}
		return nil
	})
}

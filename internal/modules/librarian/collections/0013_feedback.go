package collections

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// feedback — the librarian's store-native feedback log. One row per entry the agent records
// when it hits a problem mid-task (a tool failed, or a desk convention did not fit) or when the
// user explicitly asks it to record feedback. DB-only writes: unlike apply_fix/restore this
// touches no desk file, so it is NOT gated behind LIBRARIAN_AUTONOMOUS_WRITES (same write class
// as patrol filing findings). A fresh collection, so the down migration simply drops it.
//
// Every content-bearing text field carries an EXPLICIT Max: a bare TextField validates with
// Max==0, which PocketBase treats as the implicit 5,000-char cap (silently truncating longer
// bodies) — the repo convention (0011) is an explicit finite ceiling on every such field.
func init() {
	m.Register(func(app core.App) error {
		c := core.NewBaseCollection("feedback")
		c.Fields.Add(&core.SelectField{Name: "kind", Required: true, MaxSelect: 1, Values: []string{"problem", "feedback"}})
		c.Fields.Add(&core.TextField{Name: "summary", Required: true, Max: 2000})
		c.Fields.Add(&core.TextField{Name: "detail", Max: 50000})
		c.Fields.Add(&core.SelectField{Name: "source", Required: true, MaxSelect: 1, Values: []string{"agent", "user"}})
		c.Fields.Add(&core.TextField{Name: "context", Max: 2000})
		c.Fields.Add(&core.SelectField{Name: "status", Required: true, MaxSelect: 1, Values: []string{"open", "resolved"}})
		c.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
		c.AddIndex("idx_feedback_status", false, "status", "")
		return app.Save(c)
	}, func(app core.App) error {
		if c, _ := app.FindCollectionByNameOrId("feedback"); c != nil {
			return app.Delete(c)
		}
		return nil
	})
}

package collections

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Sets an explicit Max on seven content-bearing text fields that still rode PocketBase's implicit
// 5,000-char default, closing the gap 0011/0013 left and completing the repo's explicit-Max
// convention (a bare TextField validates at Max==0 → 5000, silently capping longer bodies; every
// content field should state its ceiling). Sizes are by ROLE (unruled by a dossier — a build-time
// call per the 0011/0013 precedent): the two finding-detail fields may embed a full replacement
// body (50,000); the summary/detail/error fields are deliberately-bounded summaries (2,000 — the
// same "bounded summary" role 0011's own comment names).
//
// Note the two directions of change: patrol_findings.detail/proposed_fix WIDEN off the 5,000
// default; the five 2,000 fields TIGHTEN below it. The tighten is intentional — it makes the
// intended bound explicit — and is non-destructive: a schema Max change never truncates existing
// rows, it only caps FUTURE saves (mirrors 0011's down note). Applies to existing stores on the
// next migrate, and to fresh stores after 0002/0003/0005/0006 create the collections. Idempotent
// (re-running sets the same Max). DOWN restores the implicit default (Max==0 → 5000) on all seven.
var cappedContentFields = []struct {
	coll, field string
	max         int
}{
	{"patrol_findings", "detail", 50000},
	{"patrol_findings", "proposed_fix", 50000},
	{"patrol_log", "summary", 2000},
	{"adoption_log", "detail", 2000},
	{"agent_runs", "input_summary", 2000},
	{"agent_runs", "output_summary", 2000},
	{"agent_runs", "error", 2000},
}

func init() {
	m.Register(func(app core.App) error {
		return applyCappedContentMax(app, false)
	}, func(app core.App) error {
		return applyCappedContentMax(app, true)
	})
}

// applyCappedContentMax sets each field's Max to its target (down=false) or back to the implicit
// default (down=true, Max=0 → 5000). Hard-errors if a collection is missing (a broken migration
// sequence), but soft-continues if a field was since renamed/removed — matching setContentMax
// (0011).
func applyCappedContentMax(app core.App, down bool) error {
	for _, f := range cappedContentFields {
		c, err := app.FindCollectionByNameOrId(f.coll)
		if err != nil {
			return err
		}
		tf, ok := c.Fields.GetByName(f.field).(*core.TextField)
		if !ok {
			continue // field renamed/removed by a later migration — nothing to cap
		}
		if down {
			tf.Max = 0
		} else {
			tf.Max = f.max
		}
		if err := app.Save(c); err != nil {
			return err
		}
	}
	return nil
}

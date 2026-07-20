package collections

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Shrinks adoption_log.event to writer-backed reality. The enum shipped six values
// [patrol, fix, revert, false_positive, friction, note] but the only writer (recordAdoptionLog)
// always writes "fix"; the other five have no writer anywhere (verified repo-wide). The reader
// (`query adoption`), the `fix` writer, and adoption_log's deskguard role all stay — the
// collection is alive; only the dead enum values retire. Wiring a new event happens when a
// concrete consumer pulls it.
//
// FORWARD is data-first (like the 0010/0012 shrink precedent): DELETE any row holding a dropped
// value BEFORE narrowing the enum, so the shrink never leaves a row outside its enum. The five
// dropped values are writerless, so no real adoption record can hold one, and there is no kept
// value to remap them to (remapping e.g. a `note` row to `fix` would fabricate a fix record).
// A dropped-value row is therefore anomalous, not a real record; deleting it loses nothing a
// writer created. Zero rows are expected, so the delete is defensive; the count is logged for
// observability. Then narrow the enum to [fix].
//
// DOWN re-adds the five values (a grow; no data remap possible or needed). Both directions are
// guarded so a re-run is a no-op.
func init() {
	m.Register(func(app core.App) error {
		anomalous, err := app.FindRecordsByFilter(
			"adoption_log",
			"event = 'patrol' || event = 'revert' || event = 'false_positive' || event = 'friction' || event = 'note'",
			"", 0, 0,
		)
		if err != nil {
			return err
		}
		for _, r := range anomalous {
			if err := app.Delete(r); err != nil {
				return err
			}
		}
		if len(anomalous) > 0 {
			app.Logger().Warn("adoption_log event shrink: deleted anomalous writerless rows", "count", len(anomalous))
		}

		c, err := app.FindCollectionByNameOrId("adoption_log")
		if err != nil {
			return err
		}
		if ev, ok := c.Fields.GetByName("event").(*core.SelectField); ok {
			kept := make([]string, 0, len(ev.Values))
			for _, v := range ev.Values {
				if v == "fix" {
					kept = append(kept, v)
				}
			}
			ev.Values = kept
			return app.Save(c)
		}
		return nil
	}, func(app core.App) error {
		c, err := app.FindCollectionByNameOrId("adoption_log")
		if err != nil {
			return nil
		}
		if ev, ok := c.Fields.GetByName("event").(*core.SelectField); ok {
			want := []string{"patrol", "fix", "revert", "false_positive", "friction", "note"}
			have := map[string]bool{}
			for _, v := range ev.Values {
				have[v] = true
			}
			complete := true
			for _, v := range want {
				if !have[v] {
					complete = false
					break
				}
			}
			if !complete {
				ev.Values = want
				return app.Save(c)
			}
		}
		return nil
	})
}

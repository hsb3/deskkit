package collections

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Adds disposition PROVENANCE to patrol_findings: who dispositioned it (actor), why (reason),
// and when (disposed_at). These are set by `findings dispose` when a finding moves to a non-open
// disposition and inherited on re-patrol alongside the disposition itself, so a resolve->re-fire
// cycle preserves who/why/when. All three are content/label fields, not relations — identity-
// neutral free text (no baked default actor ever ships).
//
//   - actor       TextField, Max 200  — short free-text identifier (like transitions.actor /
//     claimed_by); NOT a relation.
//   - reason      TextField, Max 2000 — content-bearing explanation; explicit Max because a bare
//     TextField silently caps at 5000. Optional for every disposition (a wont_fix may stay
//     anonymous); the spec SHOULD-recommends supplying one.
//   - disposed_at DateField (PLAIN, never AutodateField) — set at DISPOSE time, not record-create
//     time. An AutodateField OnCreate would stamp the finding's file time, which is wrong.
//
// FORWARD: add each field guard-before-add (a re-run is a no-op). No backfill loop: the columns
// are provenance for disposed findings only, so an empty value on every pre-existing row is the
// intended default (no filter depends on them, unlike 0014's disposition backfill).
//
// DOWN: drop all three fields guard-before-remove. No enum-value data remap is possible or needed
// — whole columns go, mirroring 0014's DOWN, so no row can be left holding an out-of-enum value.
func init() {
	m.Register(func(app core.App) error {
		c, err := app.FindCollectionByNameOrId("patrol_findings")
		if err != nil {
			return err
		}
		changed := false
		if c.Fields.GetByName("actor") == nil {
			c.Fields.Add(&core.TextField{Name: "actor", Max: 200})
			changed = true
		}
		if c.Fields.GetByName("reason") == nil {
			c.Fields.Add(&core.TextField{Name: "reason", Max: 2000})
			changed = true
		}
		if c.Fields.GetByName("disposed_at") == nil {
			c.Fields.Add(&core.DateField{Name: "disposed_at"})
			changed = true
		}
		if changed {
			return app.Save(c)
		}
		return nil
	}, func(app core.App) error {
		c, err := app.FindCollectionByNameOrId("patrol_findings")
		if err != nil {
			return nil
		}
		changed := false
		for _, name := range []string{"actor", "reason", "disposed_at"} {
			if c.Fields.GetByName(name) != nil {
				c.Fields.RemoveByName(name)
				changed = true
			}
		}
		if changed {
			return app.Save(c)
		}
		return nil
	})
}

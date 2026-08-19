package collections

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"

	"github.com/hsb3/deskkit/internal/core/settings"
)

// settings — the desk's store-backed LLM settings, so a browser can point a desk at a
// provider/model and install an API key with no shell access. It is a SINGLETON:
//
//   - the row lives at one fixed id (settings.RecordID), seeded right here by the migration, so
//     no surface ever has to decide whether to create it;
//   - every reader reads THAT id and no other (settings.Load / settings.LoadFromDir);
//   - a create hook (settings.BindHooks, bound under serve) refuses any record carrying a
//     different id, so a second row cannot become a silently-ignored place to put settings.
//
// All API rules are left nil, which in PocketBase means superuser-only. That IS the access
// control for this collection — no middleware of ours is involved, and the approval-gated `users`
// collection deliberately gets no access at all.
//
// llm_api_key is Hidden, so the field is stripped from every API response; the visible
// llm_api_key_hint (recomputed server-side from the stored key on every write) is what a browser
// renders to show which key is installed. Storing a secret in the store is a narrow, deliberate
// exception to the store-holds-no-secrets rule: a hosted desk's machine-wide config file lands
// outside the mounted data volume and is wiped on redeploy, while the store sits on it.
//
// Every TextField carries an explicit Max: a bare TextField silently caps at PocketBase's
// implicit 5,000-character default.
//
// DOWN deletes the collection (and with it the seeded row).
func init() {
	m.Register(func(app core.App) error {
		c := core.NewBaseCollection(settings.Collection)
		c.Id = "pbc_2874631905"
		c.Fields.Add(&core.TextField{Name: settings.FieldProvider, Max: 64})
		c.Fields.Add(&core.TextField{Name: settings.FieldModel, Max: 200})
		c.Fields.Add(&core.TextField{Name: settings.FieldAPIKey, Max: 500, Hidden: true})
		c.Fields.Add(&core.TextField{Name: settings.FieldAPIKeyHint, Max: 16})
		c.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
		c.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
		if err := app.Save(c); err != nil {
			return err
		}
		rec := core.NewRecord(c)
		rec.Id = settings.RecordID
		return app.Save(rec)
	}, func(app core.App) error {
		if c, _ := app.FindCollectionByNameOrId(settings.Collection); c != nil {
			return app.Delete(c)
		}
		return nil
	})
}

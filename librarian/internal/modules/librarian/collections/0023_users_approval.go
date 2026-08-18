package collections

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

// users approval gate — the auth collection a publicly-bound serve authenticates against.
//
// NOTE this is an ALTER, not a create: PocketBase ships a stock `users` auth collection in its
// own system migration (id "_pb_users_auth_"), so every store already has one. Its stock posture
// is open signup with immediate login, which is wrong for a store reachable from off-box: anyone
// who can reach the port could sign up and be authenticated. This migration adds an operator-held
// approval flag and makes it a precondition of authenticating.
//
// The rules, and why each is shaped this way:
//
//   - AuthRule: verified AND approved. Both are operator/system controlled; a self-created record
//     satisfies neither, so a fresh signup cannot authenticate at all until someone with superuser
//     rights flips them.
//   - CreateRule: signup stays public (empty-ish constraint) BUT the request body must not carry
//     `approved` at all — `:isset = false` rejects the field's mere PRESENCE, so a client cannot
//     self-approve even by sending `approved: false` and then patching it.
//   - UpdateRule: owner-only AND the same no-self-approval clause.
//   - List/View: owner-only (a user sees only their own record).
//   - DeleteRule: nil — superuser only. The stock collection lets a user delete their own record;
//     under an approval gate that would let a rejected account churn itself back to a fresh signup.
//
// The down migration restores the stock rule set verbatim and drops the added field.
const (
	usersOwnerRule = "id = @request.auth.id"
	// usersNoSelfApprove rejects a request whose body carries `approved` in any form.
	usersNoSelfApprove = "@request.body.approved:isset = false"
	usersAuthRule      = "verified = true && approved = true"
	// usersStockDeleteRule / usersStockCreateRule / usersStockAuthRule are the dependency's own
	// defaults, restored by the down migration.
	usersStockCreateRule = ""
	usersStockAuthRule   = ""
)

func init() {
	m.Register(func(app core.App) error {
		c, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}
		if c.Fields.GetByName("approved") == nil {
			c.Fields.Add(&core.BoolField{Name: "approved"})
		}
		c.AuthRule = types.Pointer(usersAuthRule)
		c.ListRule = types.Pointer(usersOwnerRule)
		c.ViewRule = types.Pointer(usersOwnerRule)
		c.CreateRule = types.Pointer(usersNoSelfApprove)
		c.UpdateRule = types.Pointer(usersOwnerRule + " && " + usersNoSelfApprove)
		c.DeleteRule = nil
		return app.Save(c)
	}, func(app core.App) error {
		c, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return nil // already gone; nothing to restore
		}
		c.Fields.RemoveByName("approved")
		c.AuthRule = types.Pointer(usersStockAuthRule)
		c.ListRule = types.Pointer(usersOwnerRule)
		c.ViewRule = types.Pointer(usersOwnerRule)
		c.CreateRule = types.Pointer(usersStockCreateRule)
		c.UpdateRule = types.Pointer(usersOwnerRule)
		c.DeleteRule = types.Pointer(usersOwnerRule)
		return app.Save(c)
	})
}

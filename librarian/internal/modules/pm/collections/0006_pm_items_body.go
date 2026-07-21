package collections

import (
	"github.com/pocketbase/pocketbase/core"
)

// bodyMax is the explicit ceiling for items.body (§3.1 long-form body). A bare TextField
// validates with Max==0, which PocketBase silently caps at 5,000 chars — far too small for an
// item's narrative, acceptance criteria, or inline spec. This mirrors the librarian's
// widenedContentMax rationale (collections/0011): a large finite ceiling that removes the
// truncation failure mode without leaving the field unbounded.
const bodyMax = 50_000_000

// upItemsBody adds items.body — the dedicated inline long-form surface for an item's narrative,
// acceptance criteria, or spec. Before this field, long-form content lived only behind the
// `pointer` external-doc reference or the separate `notes` collection; an adopter wanting to
// store an item's prose inline had nowhere to put it (`properties` is a free-form overflow bag,
// not a designated prose surface).
//
// This is a FORWARD migration, never a change to upItems (0001): existing stores at schema v5
// gain the field on their next migrate, and fresh stores gain it by replaying 0001..0006.
// Guard-before-add makes a re-run a no-op. Max is explicit (bodyMax) per the repo's
// explicit-Max convention.
func upItemsBody(app core.App) error {
	c, err := app.FindCollectionByNameOrId("items")
	if err != nil {
		return err
	}
	if c.Fields.GetByName("body") == nil {
		c.Fields.Add(&core.TextField{Name: "body", Max: bodyMax})
		return app.Save(c)
	}
	return nil
}

// downItemsBody removes items.body (guard-before-remove; schema rollback is non-destructive —
// existing rows are not truncated, only future saves lose the field until a fresh migrate
// re-applies 0006).
func downItemsBody(app core.App) error {
	c, err := app.FindCollectionByNameOrId("items")
	if err != nil {
		return nil
	}
	if c.Fields.GetByName("body") != nil {
		c.Fields.RemoveByName("body")
		return app.Save(c)
	}
	return nil
}

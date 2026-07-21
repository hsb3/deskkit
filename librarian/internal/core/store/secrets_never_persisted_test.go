package store

import (
	"regexp"
	"testing"

	"github.com/pocketbase/pocketbase/tests"

	librarianmod "github.com/hsb3/desk-standard/librarian/internal/modules/librarian"
	"github.com/hsb3/desk-standard/librarian/internal/modules/pm/collections"
)

// secretShapedFieldPattern flags collection field names that read as secret-bearing. It is
// conservative and case-insensitive: verified GREEN against every current librarian and pm field
// name (TestNoSecretShapedFieldsInStore), but tuned to trip the moment a future migration adds a
// field like "api_key", "token", or "password" to any collection.
var secretShapedFieldPattern = regexp.MustCompile(`(?i)(secret|token|passwd|password|api[_-]?key|apikey|access[_-]?key|private[_-]?key|credential|bearer)`)

// TestNoSecretShapedFieldsInStore is a schema-lint RECURRENCE GUARD for spec R6.3/§7: the store
// holds pointers and env-var NAMES, never secret values; Config carries nothing secret; keys live
// only in process env, read at provider-construction time. That invariant holds today by
// construction — no collection field in either the librarian or pm module is named/typed to carry
// a secret — so this test has nothing to catch right now. Its job is to fail loudly, forcing a
// review, the moment someone later adds a secret-shaped field to any collection migration.
func TestNoSecretShapedFieldsInStore(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("tests.NewTestApp: %v", err)
	}
	t.Cleanup(app.Cleanup)

	// The pm module's collections are deliberately NOT self-registered (no init()/m.Register —
	// see internal/modules/pm/collections/collections.go's binding-discipline comment) so the
	// per-desk feature gate can create them only when pm is enabled. tests.NewTestApp() alone
	// only applies the librarian's self-registered migrations (via deskguard_test.go's blank
	// import), so apply the pm module's Up migrations directly against this same app — the same
	// idiom internal/modules/pm/engine/engine_test.go uses — to bring items/dependencies/
	// transitions/notes/desk_config into scope for the lint below too.
	for _, mig := range collections.Migrations() {
		if mig.SelfRegistered || mig.Up == nil {
			t.Fatalf("pm migration %q must be programmatic with a real Up (spec §2.8a)", mig.Basename)
		}
		if err := mig.Up(app); err != nil {
			t.Fatalf("apply pm migration %q: %v", mig.Basename, err)
		}
	}

	// Only lint collections this project itself owns. tests.NewTestApp() clones PocketBase's own
	// fixture data dir (tests/data in the pocketbase module), which pre-seeds unrelated demo/test
	// collections ("users", "clients", PocketBase's own system collections like _superusers) that
	// carry PocketBase's own auth fields ("password", "tokenKey") — those are not desk-standard
	// schema and are out of scope for this project's R6.3 guard. Sourcing the allowlist from each
	// module's own OwnedCollections()/Names() (rather than hand-listing it here) means a future
	// collection is automatically covered without this test needing an update.
	ownedCollections := append([]string{}, librarianmod.New().OwnedCollections()...)
	ownedCollections = append(ownedCollections, collections.Names()...)

	checked := 0
	for _, name := range ownedCollections {
		col, err := app.FindCollectionByNameOrId(name)
		if err != nil {
			t.Fatalf("find owned collection %q: %v", name, err)
		}
		checked++
		for _, field := range col.Fields {
			fieldName := field.GetName()
			if secretShapedFieldPattern.MatchString(fieldName) {
				t.Errorf("collection %q field %q is secret-shaped — secrets must never be persisted in the store (spec R6.3/§7); store a pointer/env-var name instead",
					col.Name, fieldName)
			}
		}
	}

	if checked == 0 {
		t.Fatalf("no owned collections were checked — the lint would pass vacuously")
	}
}

package store

import (
	"regexp"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/tests"

	librarianmod "github.com/hsb3/deskkit/internal/modules/librarian"
	"github.com/hsb3/deskkit/internal/modules/pm/collections"
)

// secretShapedFieldPattern flags collection field names that read as secret-bearing. It is
// conservative and case-insensitive: verified GREEN against every current librarian and pm field
// name (TestNoSecretShapedFieldsInStore), but tuned to trip the moment a future migration adds a
// field like "api_key", "token", or "password" to any collection.
var secretShapedFieldPattern = regexp.MustCompile(`(?i)(secret|token|passwd|password|api[_-]?key|apikey|access[_-]?key|private[_-]?key|credential|bearer)`)

// sanctionedSecretShapedFields is the ONE narrowing of the R6.3/§7 invariant above, keyed by the
// QUALIFIED collection.field pair — never a bare field name, never a whole collection — so a
// future settings.oauth_token still trips the guard untouched.
//
//   - settings.llm_api_key holds a real secret, by an OWNER RULING (2026-08-18) that supersedes
//     R6.3 for this one field and no other. The reason is deployment-shaped: a hosted desk runs in
//     a container where the machine-wide config file (the sanctioned at-rest home for the key)
//     resolves under an XDG config home OUTSIDE the mounted data volume, so it is wiped on every
//     redeploy; the store lives ON that volume. Without this field there is no way to give a
//     hosted desk an API key from the browser, which is the capability the ruling bought.
//     The field is defended in depth: the settings collection leaves every API rule nil
//     (superuser-only), the field is declared Hidden, and a record-enrich hook re-hides it after
//     PocketBase's own superuser unhide — proven over real HTTP by
//     TestMigration0024_APIKeyNeverLeavesOverHTTP.
//   - settings.llm_api_key_hint is NOT a secret at all; it only matches the pattern by name. It
//     holds at most the key's last four characters, recomputed server-side on every write, so a
//     browser can show WHICH key is installed without ever receiving one.
//
// The pattern itself is deliberately untouched: widening it away would disarm the guard for every
// collection, where a qualified exemption disarms it for exactly two fields that were reviewed.
var sanctionedSecretShapedFields = map[string]bool{
	"settings.llm_api_key":      true,
	"settings.llm_api_key_hint": true,
}

// TestNoSecretShapedFieldsInStore is a schema-lint RECURRENCE GUARD for spec R6.3/§7: the store
// holds pointers and env-var NAMES, never secret values; Config carries nothing secret; keys live
// only in process env, read at provider-construction time. That invariant holds today by
// construction — with the single reviewed exception recorded in sanctionedSecretShapedFields — so
// this test has nothing to catch right now. Its job is to fail loudly, forcing a review, the
// moment someone later adds a secret-shaped field to any collection migration.
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
	// carry PocketBase's own auth fields ("password", "tokenKey") — those are not deskkit
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
			if sanctionedSecretShapedFields[col.Name+"."+fieldName] {
				continue
			}
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

// TestSecretShapedGuardStillHasTeeth proves the narrowing above did not disarm the guard: the
// pattern still matches the exempted names (so the exemption, not a weakened pattern, is what
// lets them through), it still matches a plausible FUTURE secret field on the very same
// collection, and the exemption map is qualified rather than bare-name — a "llm_api_key" landing
// on any other collection is still caught.
func TestSecretShapedGuardStillHasTeeth(t *testing.T) {
	for qualified := range sanctionedSecretShapedFields {
		coll, field, ok := strings.Cut(qualified, ".")
		if !ok || coll == "" || field == "" {
			t.Fatalf("exemption %q must be a qualified collection.field pair", qualified)
		}
		if !secretShapedFieldPattern.MatchString(field) {
			t.Errorf("exemption %q is dead weight: the pattern no longer matches %q, which means "+
				"the pattern itself was weakened instead of a field being exempted", qualified, field)
		}
	}

	// A future secret-shaped field, on the exempted collection and elsewhere, must still trip.
	for _, probe := range []string{"settings.oauth_token", "settings.client_secret", "files.llm_api_key"} {
		if sanctionedSecretShapedFields[probe] {
			t.Errorf("%q must not be exempt — the allowlist has been widened past its ruling", probe)
		}
		_, field, _ := strings.Cut(probe, ".")
		if !secretShapedFieldPattern.MatchString(field) {
			t.Errorf("secretShapedFieldPattern no longer matches %q — the guard has lost its teeth", field)
		}
	}
}

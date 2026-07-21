// Package collections declares the PM module's five collections (spec §3) as PROGRAMMATIC
// migrations: plain Migration values returned from Migrations(), registered into PocketBase's
// runner by core/migrate.RegisterProgrammatic ONLY when the pm module is enabled (spec §2.8a).
//
// BINDING DISCIPLINE — do not "fix" this to look like modules/librarian/collections: there is
// deliberately NO init() and NO m.Register call anywhere in this package. The librarian's
// init()+blank-import pattern registers unconditionally at compile time, which would create
// the PM collections on every desk and BREAK the per-desk feature gate (§2.9). A drift test
// (module_test.go TestNoSelfRegisteredMigrations) fails the build if that pattern creeps in.
//
// Stable pbc ids are preserved for rebuild reproducibility (§8.2), matching the librarian's
// discipline. API rules stay nil => superuser-only, matching every librarian collection.
package collections

import (
	"github.com/pocketbase/pocketbase/core"

	"github.com/hsb3/desk-standard/librarian/internal/core/migrate"
)

// Stable collection ids (§8.2). Authored once, never changed.
const (
	itemsID        = "pbc_pm_items0000001"
	dependenciesID = "pbc_pm_dependencie1"
	transitionsID  = "pbc_pm_transitions1"
	notesID        = "pbc_pm_notes0000001"
	deskConfigID   = "pbc_pm_desk0config1"
)

// Names lists the collections the pm module owns (module.OwnedCollections; §2.4 ownership
// guard). Unprefixed per the requirement's literal naming (spec §13 item 5).
func Names() []string {
	return []string{"items", "dependencies", "transitions", "notes", "desk_config"}
}

// Migrations returns the pm module's ordered migration manifest. SelfRegistered is false on
// every entry — that is the point (§2.8a).
func Migrations() []migrate.Migration {
	return []migrate.Migration{
		{Basename: "0001_pm_items", Up: upItems, Down: downCollection("items")},
		{Basename: "0002_pm_dependencies", Up: upDependencies, Down: downCollection("dependencies")},
		{Basename: "0003_pm_transitions", Up: upTransitions, Down: downCollection("transitions")},
		{Basename: "0004_pm_notes", Up: upNotes, Down: downCollection("notes")},
		{Basename: "0005_pm_desk_config", Up: upDeskConfig, Down: downCollection("desk_config")},
	}
}

// upItems creates `items` — the universal work item (§3.1). The self-referencing parent/root
// relations are added in a second save: PocketBase validates relation targets against stored
// collections, so the collection must exist before it can point at itself.
func upItems(app core.App) error {
	c := core.NewBaseCollection("items")
	c.Id = itemsID
	c.Fields.Add(&core.TextField{Name: "desk"})
	c.Fields.Add(&core.TextField{Name: "title", Required: true})
	c.Fields.Add(&core.TextField{Name: "type"})
	c.Fields.Add(&core.SelectField{Name: "phase", Required: true, MaxSelect: 1,
		Values: []string{"queue", "work", "review", "terminal"}})
	c.Fields.Add(&core.BoolField{Name: "blocked"})
	// restore_phase is the §3.2 side-state record: block() stores the phase to return to,
	// unblock() restores it. (Not in the §3.1 field table, required by the §3.2 machine text.)
	c.Fields.Add(&core.SelectField{Name: "restore_phase", MaxSelect: 1,
		Values: []string{"queue", "work", "review", "terminal"}})
	c.Fields.Add(&core.TextField{Name: "status_label"})
	c.Fields.Add(&core.SelectField{Name: "court", MaxSelect: 1,
		Values: []string{"owner", "desk", "crew", "vendor", "external-session"}})
	c.Fields.Add(&core.TextField{Name: "pointer"})
	c.Fields.Add(&core.SelectField{Name: "severity", MaxSelect: 1,
		Values: []string{"low", "medium", "high"}})
	c.Fields.Add(&core.NumberField{Name: "priority", OnlyInt: true})
	c.Fields.Add(&core.TextField{Name: "claimed_by"})
	c.Fields.Add(&core.DateField{Name: "claim_expires"})
	c.Fields.Add(&core.NumberField{Name: "version", OnlyInt: true})
	c.Fields.Add(&core.JSONField{Name: "properties"})
	c.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
	c.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
	c.AddIndex("idx_items_desk", false, "desk", "")
	if err := app.Save(c); err != nil {
		return err
	}
	saved, err := app.FindCollectionByNameOrId("items")
	if err != nil {
		return err
	}
	saved.Fields.Add(&core.RelationField{Name: "parent", CollectionId: itemsID, MaxSelect: 1})
	saved.Fields.Add(&core.RelationField{Name: "root", CollectionId: itemsID, MaxSelect: 1})
	return app.Save(saved)
}

// upDependencies creates `dependencies` — typed edges (§3.4). `kind` carries only the two
// CANONICAL values: `is-blocked-by` input is stored as the inverse `blocks` edge by the
// engine (one graph representation), so admitting it as a stored value would reopen the
// double-representation the spec closes.
func upDependencies(app core.App) error {
	c := core.NewBaseCollection("dependencies")
	c.Id = dependenciesID
	c.Fields.Add(&core.RelationField{Name: "from", CollectionId: itemsID, MaxSelect: 1, Required: true})
	c.Fields.Add(&core.RelationField{Name: "to", CollectionId: itemsID, MaxSelect: 1, Required: true})
	c.Fields.Add(&core.SelectField{Name: "kind", Required: true, MaxSelect: 1,
		Values: []string{"blocks", "relates-to"}})
	c.Fields.Add(&core.SelectField{Name: "unblock_at", MaxSelect: 1,
		Values: []string{"work", "review", "terminal"}})
	c.Fields.Add(&core.SelectField{Name: "cascade", MaxSelect: 1,
		Values: []string{"auto", "manual", "auto-reopen", "permanent"}})
	c.Fields.Add(&core.TextField{Name: "desk"})
	// Both directions are hot paths: the cascade scan queries outgoing edges (from), the
	// auto-unblock check queries incoming edges (to).
	c.AddIndex("idx_dependencies_from", false, "`from`", "")
	c.AddIndex("idx_dependencies_to", false, "`to`", "")
	return app.Save(c)
}

// upTransitions creates `transitions` — the append-only audit (§3.6). Append-only is enforced
// by the engine (it never updates/deletes a row) and hardened by the pm module's serve-time
// hooks; API rules nil keeps it superuser-only like every other collection.
func upTransitions(app core.App) error {
	c := core.NewBaseCollection("transitions")
	c.Id = transitionsID
	c.Fields.Add(&core.RelationField{Name: "item", CollectionId: itemsID, MaxSelect: 1, Required: true})
	c.Fields.Add(&core.TextField{Name: "from_phase"})
	c.Fields.Add(&core.TextField{Name: "to_phase"})
	c.Fields.Add(&core.SelectField{Name: "event", Required: true, MaxSelect: 1,
		Values: []string{"advance", "demote", "reopen", "block", "unblock", "claim", "release", "gate_refused"}})
	c.Fields.Add(&core.TextField{Name: "actor"})
	c.Fields.Add(&core.SelectField{Name: "actor_kind", MaxSelect: 1,
		Values: []string{"human", "agent"}})
	c.Fields.Add(&core.TextField{Name: "delegation_parent"})
	c.Fields.Add(&core.TextField{Name: "detail"})
	c.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
	c.AddIndex("idx_transitions_item", false, "item", "")
	return app.Save(c)
}

// upNotes creates `notes` — phase-scoped keyed notes, the lighter artifacts (§3.7).
func upNotes(app core.App) error {
	c := core.NewBaseCollection("notes")
	c.Id = notesID
	c.Fields.Add(&core.RelationField{Name: "item", CollectionId: itemsID, MaxSelect: 1, Required: true})
	c.Fields.Add(&core.TextField{Name: "phase"})
	c.Fields.Add(&core.TextField{Name: "key"})
	c.Fields.Add(&core.TextField{Name: "body"})
	c.Fields.Add(&core.TextField{Name: "actor"})
	c.Fields.Add(&core.SelectField{Name: "actor_kind", MaxSelect: 1,
		Values: []string{"human", "agent"}})
	c.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
	return app.Save(c)
}

// upDeskConfig creates `desk_config` — one row per desk (§3.8): the editable gate rules
// (YAML, validated at write time), the status_label vocabulary, the claim-TTL override, and
// the module-enabled flag.
func upDeskConfig(app core.App) error {
	c := core.NewBaseCollection("desk_config")
	c.Id = deskConfigID
	c.Fields.Add(&core.TextField{Name: "desk", Required: true})
	c.Fields.Add(&core.TextField{Name: "rules"}) // the §4.2 YAML, human-edited
	c.Fields.Add(&core.JSONField{Name: "status_labels"})
	c.Fields.Add(&core.NumberField{Name: "claim_ttl_minutes", OnlyInt: true})
	c.Fields.Add(&core.BoolField{Name: "pm_enabled"})
	c.AddIndex("idx_desk_config_desk", true, "desk", "")
	return app.Save(c)
}

// downCollection deletes a collection by name (the librarian down-migration idiom).
func downCollection(name string) func(app core.App) error {
	return func(app core.App) error {
		if c, _ := app.FindCollectionByNameOrId(name); c != nil {
			return app.Delete(c)
		}
		return nil
	}
}

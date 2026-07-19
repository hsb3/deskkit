// Package pm is the desk PM module (spec §3–§4, D3): the document-gated work graph. It owns
// five collections (items, dependencies, transitions, notes, desk_config), the rigid phase
// machine (statemachine/), the gate engine + editable per-desk rules (gates/), and the single
// transition path every surface routes through (engine/). Feature-gated OFF by default
// (spec §2.9): Enabled reads cfg.PMEnabled (env PM_ENABLED / profile modules.pm.enabled), and
// its migrations are PROGRAMMATIC — registered by core/migrate only when enabled — so a
// librarian-only desk gets NO PM collections, physically (§2.8a). Surfaces (tools, TUI,
// realtime) are D4; this module deliberately contributes none yet.
package pm

import (
	"fmt"

	"github.com/pocketbase/pocketbase/core"

	"github.com/example/pocket-librarian/internal/core/config"
	"github.com/example/pocket-librarian/internal/core/migrate"
	"github.com/example/pocket-librarian/internal/core/module"
	"github.com/example/pocket-librarian/internal/core/toolcore"
	"github.com/example/pocket-librarian/internal/modules/pm/collections"
	"github.com/example/pocket-librarian/internal/modules/pm/gates"
)

// New constructs the pm module (disabled unless cfg.PMEnabled; §2.9).
func New() module.Module { return &Mod{} }

// Mod implements module.Module.
type Mod struct{}

func (*Mod) Name() string { return "pm" }

// SchemaVersion is the highest migration sequence the pm module declares (0005).
func (*Mod) SchemaVersion() int { return 5 }

// Enabled is the per-desk feature gate (R5.5c): env PM_ENABLED > profile modules.pm.enabled >
// off. A nil cfg (config.Load failed) is off — fail closed.
func (*Mod) Enabled(cfg *config.Config) bool { return cfg != nil && cfg.PMEnabled }

// OwnedCollections lists the five PM collections (§3; ownership guard §2.4).
func (*Mod) OwnedCollections() []string { return collections.Names() }

// Migrations returns the programmatic manifest (§2.8a): real Up/Down values, SelfRegistered
// false on every entry, registered into the PocketBase runner by core/migrate ONLY when this
// module is enabled. NEVER convert these to the librarian's init()+blank-import pattern — that
// registers unconditionally and would defeat the feature gate (module_test.go guards this).
func (*Mod) Migrations() []migrate.Migration { return collections.Migrations() }

// Tools is empty in D3: the PM tool family is the D4 surfaces slice.
func (*Mod) Tools() []toolcore.ToolSpec { return nil }

// RegisterHooks binds the pm module's serve-time record hooks (only when enabled — core calls
// this per enabled module):
//   - desk_config write validation (§3.8/§4.2): a create/update carrying invalid gate rules or
//     status labels is REJECTED loud, never saved. The engine re-validates on read either way
//     (defense in depth for one-shot processes where serve hooks are not bound).
//   - transitions append-only hardening (§3.6): updates and deletes are refused.
func (*Mod) RegisterHooks(app core.App, cfg *config.Config) error {
	app.OnRecordCreate("desk_config").BindFunc(func(e *core.RecordEvent) error {
		if err := validateDeskConfigRecord(e.Record); err != nil {
			return err
		}
		return e.Next()
	})
	app.OnRecordUpdate("desk_config").BindFunc(func(e *core.RecordEvent) error {
		if err := validateDeskConfigRecord(e.Record); err != nil {
			return err
		}
		return e.Next()
	})
	app.OnRecordUpdate("transitions").BindFunc(func(e *core.RecordEvent) error {
		return fmt.Errorf("transitions is append-only (spec §3.6): rows are never updated")
	})
	app.OnRecordDelete("transitions").BindFunc(func(e *core.RecordEvent) error {
		return fmt.Errorf("transitions is append-only (spec §3.6): rows are never deleted")
	})
	return nil
}

// validateDeskConfigRecord fail-louds an invalid desk_config write (§4.2: an invalid config
// is rejected rather than silently disabling gates).
func validateDeskConfigRecord(rec *core.Record) error {
	if rulesYAML := rec.GetString("rules"); rulesYAML != "" {
		if _, err := gates.ParseRules(rulesYAML); err != nil {
			return err
		}
	}
	return nil
}

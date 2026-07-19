// Package pm is the desk PM module (spec §3–§5): the document-gated work graph. It owns
// five collections (items, dependencies, transitions, notes, desk_config), the rigid phase
// machine (statemachine/), the gate engine + editable per-desk rules (gates/), the single
// transition path every surface routes through (engine/), and — since D4 — the surfaces:
// the twelve-tool family (tools/) contributed to the shared tool core, the TUI views (tui/)
// mounted into the shared chat TUI, and the serve-only realtime emitter (§5.4).
// Feature-gated OFF by default (spec §2.9): Enabled reads cfg.PMEnabled (env PM_ENABLED /
// profile modules.pm.enabled), and its migrations are PROGRAMMATIC — registered by
// core/migrate only when enabled — so a librarian-only desk gets NO PM collections,
// physically (§2.8a), and (because only enabled modules register) no PM tools, views, hooks,
// or realtime on any surface.
package pm

import (
	"encoding/json"
	"fmt"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/subscriptions"

	"github.com/example/pocket-librarian/internal/core/config"
	"github.com/example/pocket-librarian/internal/core/migrate"
	"github.com/example/pocket-librarian/internal/core/module"
	"github.com/example/pocket-librarian/internal/core/schema"
	"github.com/example/pocket-librarian/internal/core/toolcore"
	"github.com/example/pocket-librarian/internal/core/tuiview"
	"github.com/example/pocket-librarian/internal/modules/pm/collections"
	"github.com/example/pocket-librarian/internal/modules/pm/gates"
	pmtools "github.com/example/pocket-librarian/internal/modules/pm/tools"
	pmtui "github.com/example/pocket-librarian/internal/modules/pm/tui"
)

// RealtimeTopic is the subscription topic PM realtime events are broadcast on (spec §5.4):
// one message per transitions write (advance/demote/reopen/block/unblock/claim/release/
// gate_refused — cascades included, since cascades append their own audit rows).
const RealtimeTopic = "pm/transitions"

// New constructs the pm module (disabled unless cfg.PMEnabled; §2.9).
func New() module.Module { return &Mod{} }

// Mod implements module.Module (+ module.Configurable, module.ValidatorConsumer, and
// module.RealtimeSource).
type Mod struct {
	cfg       *config.Config           // injected via Configure; nil until then
	validator schema.DocumentValidator // injected via SetValidator (the librarian's, via core)
}

// Configure implements module.Configurable: the resolved config drives the tool write gate
// (PM_AUTONOMOUS_WRITES) at Tools() time.
func (m *Mod) Configure(cfg *config.Config) { m.cfg = cfg }

// SetValidator implements module.ValidatorConsumer (spec §2.5): core injects the captured
// DocumentValidator after registration; the tool closures read it lazily at invoke time.
func (m *Mod) SetValidator(v schema.DocumentValidator) { m.validator = v }

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

// Tools returns the twelve-tool PM family (spec §5.1) for the shared tool core. Only enabled
// modules are asked, so a librarian-only desk gets no PM tool on any surface (§2.9). The
// write tools' agent exposure follows PM_AUTONOMOUS_WRITES (default ON; §13 item 9); the
// validator getter reads lazily because core injects it after Tools() is collected.
func (m *Mod) Tools() []toolcore.ToolSpec {
	writes := m.cfg == nil || m.cfg.PMAutonomousWrites
	return pmtools.Specs(func() schema.DocumentValidator { return m.validator }, writes)
}

// TUIViews returns the three PM views (spec §5.3) — the get_context landing view, the
// board/queue view, and the item detail view — mounted into the shared chat TUI. Only
// enabled modules are asked, so a librarian-only desk mounts none (§2.9).
func (m *Mod) TUIViews(app core.App, cfg *config.Config) []tuiview.View {
	return pmtui.Views(app, cfg)
}

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

// RegisterRealtime implements module.RealtimeSource (spec §5.4, R4.3 — adopted): every
// transitions write broadcasts one PB-native realtime message on RealtimeTopic to the clients
// subscribed to it. Wired by main under OnServe ONLY (realtime is serve-only, like the
// trigger layer; one-shot CLI calls emit no events). A marshal failure or an absent broker is
// swallowed — realtime is an observer channel, never allowed to fail a transition.
func (*Mod) RegisterRealtime(app core.App) error {
	app.OnRecordAfterCreateSuccess("transitions").BindFunc(func(e *core.RecordEvent) error {
		broadcastTransition(e.App, e.Record)
		return e.Next()
	})
	return nil
}

// broadcastTransition sends one transitions row to every client subscribed to RealtimeTopic.
func broadcastTransition(app core.App, rec *core.Record) {
	payload, err := json.Marshal(map[string]any{
		"item":       rec.GetString("item"),
		"event":      rec.GetString("event"),
		"from_phase": rec.GetString("from_phase"),
		"to_phase":   rec.GetString("to_phase"),
		"actor":      rec.GetString("actor"),
		"actor_kind": rec.GetString("actor_kind"),
		"detail":     rec.GetString("detail"),
		"created":    rec.GetDateTime("created").String(),
	})
	if err != nil {
		return
	}
	msg := subscriptions.Message{Name: RealtimeTopic, Data: payload}
	for _, client := range app.SubscriptionsBroker().Clients() {
		if client.IsDiscarded() || !client.HasSubscription(RealtimeTopic) {
			continue
		}
		// Async per client: DefaultClient.Send writes an UNBUFFERED channel that only the
		// client's live SSE writer drains, so a synchronous send from this record hook would
		// let one stuck consumer block the transition itself. Realtime is an observer channel
		// (§5.4); the transition never waits on it. Send recovers a closed-channel panic
		// internally, so a client discarded mid-flight is safe.
		go client.Send(msg)
	}
}

// validateDeskConfigRecord fail-louds an invalid desk_config write (§4.2: an invalid config
// is rejected rather than silently disabling gates). Both the gate rules YAML and the
// status_labels JSON are checked — the engine re-validates on read either way.
func validateDeskConfigRecord(rec *core.Record) error {
	if rulesYAML := rec.GetString("rules"); rulesYAML != "" {
		if _, err := gates.ParseRules(rulesYAML); err != nil {
			return err
		}
	}
	if _, err := gates.ParseLabels(rec.GetString("status_labels")); err != nil {
		return err
	}
	return nil
}

// Package module defines the Module interface every desk module (librarian, pm, ...)
// implements and the Register engine that wires the enabled subset into the running app: it
// merges each enabled module's tools into the shared toolcore registry, captures any
// schema.DocumentValidator implementation for later injection into gate consumers (D3), and
// wires migration registration. This is what lets a second module's tools appear on every
// surface (agent/mcp/CLI) without those surfaces knowing the module exists.
package module

import (
	"fmt"

	"github.com/pocketbase/pocketbase/core"

	"github.com/hsb3/deskkit/internal/core/config"
	"github.com/hsb3/deskkit/internal/core/migrate"
	"github.com/hsb3/deskkit/internal/core/schema"
	"github.com/hsb3/deskkit/internal/core/toolcore"
	"github.com/hsb3/deskkit/internal/core/tuiview"
)

// Module is the interface every desk module implements.
type Module interface {
	Name() string
	SchemaVersion() int
	Enabled(cfg *config.Config) bool
	Migrations() []migrate.Migration
	OwnedCollections() []string
	Tools() []toolcore.ToolSpec
	RegisterHooks(app core.App, cfg *config.Config) error
	// TUIViews returns the module's views for the shared chat TUI (spec §5.3), built lazily
	// against the LIVE app/config at TUI start (registration happens before the app exists).
	// nil = the module contributes no views (the librarian: the chat transcript IS its view).
	// This is the plug-point D2 deferred ("D4 adds TUIViews when PM TUI views land").
	TUIViews(app core.App, cfg *config.Config) []tuiview.View
}

// Registry captures the result of Register: the enabled module set, the merged tool registry
// (already populated into toolcore's package-global), and any captured
// schema.DocumentValidator (the librarian module implements it; D3's gate engine consumes it).
type Registry struct {
	Enabled   []Module
	Validator schema.DocumentValidator
}

// Configurable is the optional interface a module implements to receive the resolved config at
// registration time (D3): the librarian's DocumentValidator needs DeskRoot to resolve document
// pointers, and modules are constructed in main before config is threaded anywhere else. cfg
// may be nil (main registers modules even when config.Load failed); implementations must fail
// closed on a nil config, never panic.
type Configurable interface {
	Configure(cfg *config.Config)
}

// ValidatorConsumer is the optional interface a module implements to receive the captured
// schema.DocumentValidator after registration (spec §2.5): the pm module's tools/gate engine
// consume verdicts through it. Injection happens in a SECOND pass (after every module's tools
// are collected), so consumers must read the injected value lazily at invoke time.
type ValidatorConsumer interface {
	SetValidator(v schema.DocumentValidator)
}

// RealtimeSource is the optional capability a module implements to emit realtime events
// (spec §5.4, R4.3): core wires it to PocketBase's realtime subsystem under `serve` only —
// one-shot CLI commands emit no events (main.go's OnServe loop calls this).
type RealtimeSource interface {
	RegisterRealtime(app core.App) error
}

// Register wires the enabled subset of mods into the app: filters by Enabled(cfg), asserts no
// owned-collection collision across the enabled set, merges each enabled module's tools into
// toolcore, captures a schema.DocumentValidator if a module implements it, and registers each
// enabled module's non-self-registered migrations programmatically. RegisterHooks is NOT
// called here — it is serve-only and invoked separately under OnServe (see main.go wiring).
func Register(cfg *config.Config, mods ...Module) (*Registry, error) {
	var enabled []Module
	owned := map[string]string{} // collection name -> owning module, for the collision assert
	for _, mod := range mods {
		if !mod.Enabled(cfg) {
			continue
		}
		for _, c := range mod.OwnedCollections() {
			if prior, dup := owned[c]; dup {
				return nil, fmt.Errorf("module: collection %q claimed by both %q and %q", c, prior, mod.Name())
			}
			owned[c] = mod.Name()
		}
		enabled = append(enabled, mod)
	}

	reg := &Registry{Enabled: enabled}
	for _, mod := range enabled {
		if c, ok := mod.(Configurable); ok {
			c.Configure(cfg)
		}
		toolcore.Register(mod.Tools()...)
		if v, ok := mod.(schema.DocumentValidator); ok {
			// The librarian module is the base module and always enabled, so the first (and in
			// D2 the only) DocumentValidator wins; a future multi-validator composition scheme is
			// out of scope here.
			if reg.Validator == nil {
				reg.Validator = v
			}
		}
	}
	// Second pass (spec §2.5): inject the captured validator into consumers. Runs after the
	// tools pass because the validator-providing module may be registered in any position;
	// consumers read the value lazily at invoke time, so late injection is safe.
	for _, mod := range enabled {
		if c, ok := mod.(ValidatorConsumer); ok {
			c.SetValidator(reg.Validator)
		}
	}
	migrate.RegisterProgrammatic(reg.MigrateModules())

	return reg, nil
}

// TUIViews collects every enabled module's views for the shared chat TUI (spec §5.3), in
// module registration order. Called lazily at TUI start with the live app + config.
func (r *Registry) TUIViews(app core.App, cfg *config.Config) []tuiview.View {
	var out []tuiview.View
	for _, mod := range r.Enabled {
		out = append(out, mod.TUIViews(app, cfg)...)
	}
	return out
}

// MigrateModules adapts the enabled set to []migrate.Module (a narrower interface than
// module.Module, so each element converts implicitly) for RegisterProgrammatic/StampModules/
// GuardDowngrade. A slice of the wider Module interface cannot be passed directly where
// []migrate.Module is expected (Go slices are invariant), so this rebuilds the slice
// element-by-element.
func (r *Registry) MigrateModules() []migrate.Module {
	out := make([]migrate.Module, len(r.Enabled))
	for i, mod := range r.Enabled {
		out[i] = mod
	}
	return out
}

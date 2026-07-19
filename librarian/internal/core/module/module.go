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

	"github.com/example/pocket-librarian/internal/core/config"
	"github.com/example/pocket-librarian/internal/core/migrate"
	"github.com/example/pocket-librarian/internal/core/schema"
	"github.com/example/pocket-librarian/internal/core/toolcore"
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
	// TUIViews() is DEFERRED — see D2-DESIGN.md §core/module: the current TUI (internal/tui,
	// now modules/librarian/tui) exposes no View plug-point, so D2 does not invent one. The
	// librarian TUI keeps working as-is (main calls tui.Run directly, unchanged); D4 adds
	// TUIViews to this interface when PM TUI views land.
}

// Registry captures the result of Register: the enabled module set, the merged tool registry
// (already populated into toolcore's package-global), and any captured
// schema.DocumentValidator (the librarian module implements it; D3's gate engine consumes it).
type Registry struct {
	Enabled   []Module
	Validator schema.DocumentValidator
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
	migrate.RegisterProgrammatic(reg.MigrateModules())

	return reg, nil
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

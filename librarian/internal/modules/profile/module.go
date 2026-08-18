// Package profile is the desk personalization module: it contributes the four read-only tools
// that read a desk's `_knowledge/profile.*` profile and its `_knowledge/` background folder to
// every surface (agent, MCP, CLI) through the shared tool core.
//
// It is a PURE READ module and owns no persisted state: no collections, no migrations, no
// hooks, no TUI views. That is deliberate — the personalization surfaces are FILES on the desk,
// so their truth is the tree, never the store. Because it declares no migrations, its
// SchemaVersion is 0: stamp-by-observation never writes a row for it, and the downgrade guard
// has nothing to compare, which is the correct behaviour for a module that cannot make a store
// unreadable to an older binary.
//
// Always enabled (like the base librarian module): a desk always has a personalization surface,
// even when it is empty — the tools answer "this desk declares nothing" rather than vanishing,
// which is what makes an unpersonalized desk diagnosable instead of silently toolless.
package profile

import (
	"github.com/pocketbase/pocketbase/core"

	"github.com/hsb3/desk-standard/librarian/internal/core/config"
	"github.com/hsb3/desk-standard/librarian/internal/core/migrate"
	"github.com/hsb3/desk-standard/librarian/internal/core/module"
	"github.com/hsb3/desk-standard/librarian/internal/core/toolcore"
	"github.com/hsb3/desk-standard/librarian/internal/core/tuiview"
	"github.com/hsb3/desk-standard/librarian/internal/modules/profile/tools"
)

// New constructs the profile module.
func New() module.Module { return &Mod{} }

// Mod implements module.Module. It carries no state: every tool resolves the desk from the
// config it is handed at invoke time.
type Mod struct{}

func (*Mod) Name() string { return tools.Module }

// SchemaVersion is 0: this module declares no migrations and owns no collections.
func (*Mod) SchemaVersion() int { return 0 }

// Enabled is always true — a desk always has a personalization surface to read.
func (*Mod) Enabled(*config.Config) bool { return true }

// OwnedCollections is empty: the profile and the background folder are desk FILES, not store rows.
func (*Mod) OwnedCollections() []string { return nil }

// Migrations is empty (no collections to create or alter).
func (*Mod) Migrations() []migrate.Migration { return nil }

// Tools returns the module's four read-only specs.
func (*Mod) Tools() []toolcore.ToolSpec { return tools.Specs() }

// TUIViews is nil: this module contributes no mounted view.
func (*Mod) TUIViews(core.App, *config.Config) []tuiview.View { return nil }

// RegisterHooks is a no-op: nothing here reacts to store writes.
func (*Mod) RegisterHooks(core.App, *config.Config) error { return nil }

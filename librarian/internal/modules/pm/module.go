// Package pm is the disabled scaffold for the desk PM module (D3+). D2 registers it so the
// module.Register wiring exercises a real second module, but Enabled defaults false
// (PM_ENABLED unset), so the binary stays behaviorally identical to pre-refactor
// pocket-librarian: no PM tools, collections, or hooks are active.
package pm

import (
	"github.com/pocketbase/pocketbase/core"

	"github.com/example/pocket-librarian/internal/core/config"
	"github.com/example/pocket-librarian/internal/core/migrate"
	"github.com/example/pocket-librarian/internal/core/module"
	"github.com/example/pocket-librarian/internal/core/toolcore"
)

// New constructs the (disabled-by-default) pm module.
func New() module.Module { return &Mod{} }

// Mod implements module.Module as an empty, disabled-by-default scaffold.
type Mod struct{}

func (*Mod) Name() string                                 { return "pm" }
func (*Mod) SchemaVersion() int                           { return 0 }
func (*Mod) Enabled(cfg *config.Config) bool              { return cfg != nil && cfg.PMEnabled }
func (*Mod) OwnedCollections() []string                   { return nil }
func (*Mod) Tools() []toolcore.ToolSpec                   { return nil }
func (*Mod) Migrations() []migrate.Migration              { return nil }
func (*Mod) RegisterHooks(core.App, *config.Config) error { return nil }

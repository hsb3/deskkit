// Package librarian is the base desk module: it wraps the existing librarian collections,
// tools, and hooks behind the module.Module + schema.DocumentValidator interfaces (spec §2.7).
// It is always enabled (Enabled ignores cfg) so the binary stays behaviorally identical to
// pre-refactor pocket-librarian.
package librarian

import (
	"context"

	"github.com/pocketbase/pocketbase/core"

	"github.com/example/pocket-librarian/internal/core/config"
	"github.com/example/pocket-librarian/internal/core/migrate"
	"github.com/example/pocket-librarian/internal/core/module"
	"github.com/example/pocket-librarian/internal/core/schema"
	"github.com/example/pocket-librarian/internal/core/toolcore"
	"github.com/example/pocket-librarian/internal/modules/librarian/tools"
	"github.com/example/pocket-librarian/internal/modules/librarian/trigger"
)

// New constructs the librarian module.
func New() module.Module { return &Mod{} }

// Mod implements module.Module + schema.DocumentValidator.
type Mod struct{}

func (*Mod) Name() string { return "librarian" }

// SchemaVersion is the highest migration sequence the librarian module declares (0013).
func (*Mod) SchemaVersion() int { return 13 }

// Enabled is always true: librarian is the base module (spec §2.7).
func (*Mod) Enabled(*config.Config) bool { return true }

// OwnedCollections lists every collection created by the librarian's 0001..0013 migrations
// (enumerated from the migration bodies; see module_test.go's drift guard for the migrations
// side).
func (*Mod) OwnedCollections() []string {
	return []string{
		"files", "patrol_findings", "patrol_log", "revisions", "adoption_log",
		"agent_runs", "messages", "tasks", "prompts", "feedback",
	}
}

// Tools returns the librarian's seven tool specs.
func (*Mod) Tools() []toolcore.ToolSpec { return tools.Specs() }

// Migrations lists the librarian's 0001..0013 migrations. All are SelfRegistered: their
// bodies still call PocketBase's m.Register via their own init() (blank-imported by main via
// internal/modules/librarian/collections), so Up/Down are nil here — this manifest exists for
// stamp-by-observation (core/migrate.StampModules) and the drift test below, not to re-wire
// registration.
func (*Mod) Migrations() []migrate.Migration {
	basenames := []string{
		"0001_files", "0002_patrol_findings", "0003_patrol_log", "0004_revisions",
		"0005_adoption_log", "0006_agent_runs", "0007_messages", "0008_tasks",
		"0009_prompts", "0010_patrol_findings_resolved", "0011_widen_content_fields",
		"0012_dir_kind_add_infra", "0013_feedback",
	}
	out := make([]migrate.Migration, len(basenames))
	for i, b := range basenames {
		out[i] = migrate.Migration{Basename: b, SelfRegistered: true}
	}
	return out
}

// RegisterHooks wires the librarian's record hooks + cron (spec §2.4 wake layer). StartClaimer
// stays in main (it needs agentAction, which would otherwise pull internal/agent into
// internal/trigger) — see main.go wiring.
func (*Mod) RegisterHooks(app core.App, cfg *config.Config) error {
	if err := trigger.RegisterHooks(app, cfg); err != nil {
		return err
	}
	return trigger.RegisterCron(app, cfg)
}

// Verdict implements schema.DocumentValidator. D2 defines the seam only (§2.5) — no gate
// consumes it yet, and the schema.DocumentValidator interface takes no app/store handle (that
// wiring is D3's gate-engine concern: it will capture the real app at registration time). A
// real existence + frontmatter-validity check against the `files` collection requires that
// handle, so building one now would mean inventing untested plumbing nothing exercises. Kept
// deliberately minimal and honest per the design brief ("do NOT over-build"): it returns an
// explicit not-yet-wired verdict so a D3 caller sees a clear, unambiguous signal rather than a
// silently-wrong answer.
func (*Mod) Verdict(_ context.Context, pointer string, _ schema.ArtifactRequirement) (schema.Verdict, error) {
	return schema.Verdict{
		Exists:  false,
		Missing: []string{"librarian.Verdict is not yet wired to a store handle (D2 seam-only; D3 completes gate use) for pointer " + pointer},
	}, nil
}

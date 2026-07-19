package tools

import (
	"context"

	"github.com/pocketbase/pocketbase/core"

	cfgpkg "github.com/example/pocket-librarian/internal/core/config"
	"github.com/example/pocket-librarian/internal/core/toolcore"
)

// Specs returns the librarian module's seven tools as toolcore.ToolSpecs (spec §5, plus
// record_feedback). Descriptions and gate flags (WritesFiles/AgentDefault/AgentGated) are
// copied byte-for-byte from the pre-refactor internal/tools/registry.go Registry.
//
// Gate encoding (spec §5.4):
//   - the autonomous serve agent gets {query, sweep, patrol, propose_fix, record_feedback}
//     always, plus apply_fix ONLY when AutonomousWrites is set;
//   - restore is CLI/supervised-only (recovery is a human action) — it is in neither the
//     AgentDefault nor the AgentGated set, so an autonomous loop never receives it;
//   - the CLI always builds the full seven.
func Specs() []toolcore.ToolSpec {
	const mod = "librarian"
	return []toolcore.ToolSpec{
		toolcore.New[SweepInput](mod, "sweep",
			"Reindex the desk tree into the files collection. Idempotent.",
			false, true, false,
			func(ctx context.Context, app core.App, cfg *cfgpkg.Config, in *SweepInput) (any, error) {
				return Sweep(ctx, app, cfg, in)
			}),
		toolcore.New[PatrolInput](mod, "patrol",
			"Flag rule violations (R1–R6) as findings; never writes files.",
			false, true, false,
			func(ctx context.Context, app core.App, cfg *cfgpkg.Config, in *PatrolInput) (any, error) {
				return Patrol(ctx, app, cfg, in)
			}),
		toolcore.New[ProposeFixInput](mod, "propose_fix",
			"Compute a mechanical fix and record the file's original content before any write.",
			false, true, false,
			func(ctx context.Context, app core.App, cfg *cfgpkg.Config, in *ProposeFixInput) (any, error) {
				return ProposeFix(ctx, app, cfg, in)
			}),
		toolcore.New[ApplyFixInput](mod, "apply_fix",
			"Commit a previously proposed fix to disk, byte-exact.",
			true, false, true,
			func(ctx context.Context, app core.App, cfg *cfgpkg.Config, in *ApplyFixInput) (any, error) {
				return ApplyFix(ctx, app, cfg, in)
			}),
		toolcore.New[RestoreInput](mod, "restore",
			"Reverse a change to the exact recorded original.",
			true, false, false,
			func(ctx context.Context, app core.App, cfg *cfgpkg.Config, in *RestoreInput) (any, error) {
				return Restore(ctx, app, cfg, in)
			}),
		toolcore.New[QueryInput](mod, "query",
			"Read-only questions over the file index and findings.",
			false, true, false,
			func(ctx context.Context, app core.App, cfg *cfgpkg.Config, in *QueryInput) (any, error) {
				raw, err := Query(ctx, app, cfg, in)
				if err != nil {
					return nil, err
				}
				return raw, nil // raw = json.RawMessage; json.Marshal of it is the bytes verbatim
			}),
		toolcore.New[RecordFeedbackInput](mod, "record_feedback",
			"Record a problem or feedback entry to the store's feedback log.",
			false, true, false,
			func(ctx context.Context, app core.App, cfg *cfgpkg.Config, in *RecordFeedbackInput) (any, error) {
				return RecordFeedback(ctx, app, cfg, in)
			}),
	}
}

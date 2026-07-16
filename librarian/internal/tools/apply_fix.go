package tools

import (
	"context"

	"github.com/pocketbase/pocketbase/core"

	"github.com/example/pocket-librarian/internal/config"
)

// ApplyFix — §5.4: the write path. Reload the ignore list and re-run the ignore + staleness
// guards, re-derive the plan from (rec, original_content), write byte-exact
// (desklib.WriteExact / os.Rename), then patch revisions.applied + patrol_findings.state and
// append one adoption_log row. The ONLY tool that mutates the desk tree — its registration
// for the autonomous agent is gated by LIBRARIAN_AUTONOMOUS_WRITES (registry.AgentTools, §5.4).
//
// STUB — implement the §5.4 logic here (this file's owner).
func ApplyFix(ctx context.Context, app core.App, cfg *config.Config, in *ApplyFixInput) (*ApplyFixResult, error) {
	return nil, notImplemented("apply_fix")
}

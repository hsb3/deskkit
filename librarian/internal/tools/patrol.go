package tools

import (
	"context"

	"github.com/pocketbase/pocketbase/core"

	"github.com/example/pocket-librarian/internal/config"
)

// Patrol — §5.2: run rules R1–R6 over the non-deleted files (filtered by Path), file new
// patrol_findings (dedupe on (path, rule, checksum) in app code) + one patrol_log row.
// Never writes the filesystem. R6 (HANDOFF staleness) runs only when cfg.HandoffPath is in
// the filtered set. ISSUE_REF_RE lookbehind is reimplemented (RE2 has no lookbehind, §5.2).
//
// STUB — implement the §5.2 logic here (this file's owner).
func Patrol(ctx context.Context, app core.App, cfg *config.Config, in *PatrolInput) (*PatrolResult, error) {
	return nil, notImplemented("patrol")
}

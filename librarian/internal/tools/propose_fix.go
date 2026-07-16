package tools

import (
	"context"

	"github.com/pocketbase/pocketbase/core"

	"github.com/example/pocket-librarian/internal/config"
)

// ProposeFix — §5.3: load the ignore list FIRST (fail closed via desklib.Ignored), then for
// each flagged mechanical R1/R2/R3 finding run the guards in EXACT order
// (ignore → missing → read → staleness → plan → RECORD-ORIGINAL-FIRST) and create a
// revisions row. No filesystem write. A failed original-record aborts the operation.
//
// STUB — implement the §5.3 logic here (this file's owner).
func ProposeFix(ctx context.Context, app core.App, cfg *config.Config, in *ProposeFixInput) (*ProposeFixResult, error) {
	return nil, notImplemented("propose_fix")
}

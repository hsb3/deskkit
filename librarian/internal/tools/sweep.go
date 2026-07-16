package tools

import (
	"context"

	"github.com/pocketbase/pocketbase/core"

	"github.com/example/pocket-librarian/internal/config"
)

// Sweep — §5.1: walk DESK_ROOT, checksum + parse frontmatter (desklib.ParseFrontmatter),
// derive dir_kind (cfg.EntityDirMap prefix match), apply the pointer-stub heuristic, and
// upsert the files collection inside one app.RunInTransaction. Idempotent: COMPARE_FIELDS
// excludes path + last_seen, unchanged files are not patched.
//
// STUB — implement the §5.1 logic here (this file's owner).
func Sweep(ctx context.Context, app core.App, cfg *config.Config, in *SweepInput) (*SweepResult, error) {
	return nil, notImplemented("sweep")
}

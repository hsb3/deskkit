package tools

import (
	"context"

	"github.com/pocketbase/pocketbase/core"

	"github.com/example/pocket-librarian/internal/config"
)

// Restore — §5.5: reverse a change to the exact recorded original. Supports --by-path
// resolution (latest applied, unrestored revision whose path/new_path matches), verifies
// sha256(original_content) == original_checksum before writing, restores byte-exact, and
// reopens the finding to flagged. CLI/supervised only (never in the autonomous agent set).
//
// STUB — implement the §5.5 logic here (this file's owner).
func Restore(ctx context.Context, app core.App, cfg *config.Config, in *RestoreInput) (*RestoreResult, error) {
	return nil, notImplemented("restore")
}

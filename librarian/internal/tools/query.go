package tools

import (
	"context"
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"

	"github.com/example/pocket-librarian/internal/config"
)

// Query — §5.6: read-only queries over files/findings/adoption. Kind selects one of
// live_files | recent | orphans | uncollapsed | findings | summary | adoption; the returned
// JSON document echoes kind + count plus a kind-specific body (shapes in §5.6). Never writes.
//
// STUB — implement the §5.6 logic here (this file's owner).
func Query(ctx context.Context, app core.App, cfg *config.Config, in *QueryInput) (json.RawMessage, error) {
	return nil, notImplemented("query")
}

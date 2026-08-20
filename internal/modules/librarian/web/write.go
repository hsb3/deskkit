// The document write route (SPA overhaul phase 0): the SPA's one door from browser to disk.
// It calls the SAME tools.WriteDoc the CLI subcommand calls — desklib.WriteExact +
// record-original-first + row re-index — so the browser opens no second write path.
// Auth posture is inherited from this package's model: loopback stays unauthenticated
// (single-operator local UX), a public bind puts the route behind apis.RequireAuth and
// the strict same-origin guard, identical to the chat session routes.
package web

import (
	"encoding/json"
	"net/http"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"

	"github.com/hsb3/deskkit/internal/core/config"
	"github.com/hsb3/deskkit/internal/modules/librarian/tools"
)

// PathDocWrite is the POST endpoint the SPA's save action calls.
const PathDocWrite = "/desk/doc/write"

// maxWriteBody caps a write request. files.content caps at 1,000,000 runes, so 4 MiB
// leaves room for a full body of multi-byte runes plus the JSON envelope.
const maxWriteBody = 4 << 20

// RegisterDocWrite mounts the write-through route. Serve-only, and only when desk config
// resolved (the tool needs a desk root). public mirrors web.Register's argument.
func RegisterDocWrite(r *router.Router[*core.RequestEvent], app core.App, cfg *config.Config, public bool) {
	rt := r.POST(PathDocWrite, func(e *core.RequestEvent) error {
		if !originAllowed(e.Request, public) {
			return e.JSON(http.StatusForbidden, crossOriginRejected)
		}
		e.Request.Body = http.MaxBytesReader(e.Response, e.Request.Body, maxWriteBody)
		var in tools.WriteDocInput
		if err := json.NewDecoder(e.Request.Body).Decode(&in); err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body: " + err.Error()})
		}
		res, err := tools.WriteDoc(e.Request.Context(), app, cfg, &in)
		if err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		if res.Outcome == "conflict" {
			// 409 carries the full result: the save stops and the surface shows the
			// difference (the conflict rule; overwrite = explicit re-submit with
			// the fresh checksum).
			return e.JSON(http.StatusConflict, res)
		}
		return e.JSON(http.StatusOK, res)
	})
	if public {
		rt.Bind(apis.RequireAuth(authCollections...))
	}
}

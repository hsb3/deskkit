// The document delete route: the write route's sibling, and the second (and last) door from
// browser to disk. It calls tools.DeleteDoc — record-original-first, then remove, then
// soft-delete the row — so a deletion made here is reversed by the same
// `deskkit restore --by-path` an operator would type after a bad apply_fix.
//
// Auth posture is deliberately IDENTICAL to the write route rather than stricter: both are
// reversible operations on the same files, gated by the same compare-and-swap, and a delete
// with a byte-exact recorded original is no more destructive than an overwrite. Loopback stays
// unauthenticated (single-operator local UX); a public bind adds apis.RequireAuth plus the
// strict same-origin guard.
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

// PathDocDelete is the POST endpoint the SPA's delete action calls once its two-step in-page
// confirm commits.
const PathDocDelete = "/desk/doc/delete"

// RegisterDocDelete mounts the delete route. Serve-only, and only when desk config resolved
// (the tool needs a desk root). public mirrors web.Register's argument.
func RegisterDocDelete(r *router.Router[*core.RequestEvent], app core.App, cfg *config.Config, public bool) {
	rt := r.POST(PathDocDelete, func(e *core.RequestEvent) error {
		if !originAllowed(e.Request, public) {
			return e.JSON(http.StatusForbidden, crossOriginRejected)
		}
		// Same cap as the write route. A delete body is two short strings, but the reader is
		// what stops an unbounded upload before it is buffered, so it belongs here too.
		e.Request.Body = http.MaxBytesReader(e.Response, e.Request.Body, maxWriteBody)
		var in tools.DeleteDocInput
		if err := json.NewDecoder(e.Request.Body).Decode(&in); err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body: " + err.Error()})
		}
		res, err := tools.DeleteDoc(e.Request.Context(), app, cfg, &in)
		if err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		if res.Outcome == "conflict" {
			// 409 carries the disk's current state, the same body the write route returns, so
			// the surface handles both verbs' conflict with one branch.
			return e.JSON(http.StatusConflict, res)
		}
		return e.JSON(http.StatusOK, res)
	})
	if public {
		rt.Bind(apis.RequireAuth(authCollections...))
	}
}

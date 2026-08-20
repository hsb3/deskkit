package spa

import (
	"net/http"

	"github.com/pocketbase/pocketbase/core"

	coreschema "github.com/hsb3/deskkit/internal/core/schema"
)

// PathDoctypes is the doc-type vocabulary endpoint the SPA reads to render a status picker
// that offers the statuses legal for THAT record's type, instead of a free text box the
// operator can type an invalid status into.
const PathDoctypes = "/desk/doctypes"

// doctypeSpec is one type's entry: its status family ("" for lightweight types, which have
// none) plus its type-specific fields. Required/Optional are always emitted as lists, never
// null — the browser then needs no absent-field branch, and a `.map()` over them is safe.
type doctypeSpec struct {
	Family      string   `json:"family"`
	Lightweight bool     `json:"lightweight"`
	Required    []string `json:"required"`
	Optional    []string `json:"optional"`
}

// doctypesResponse is the wire shape PathDoctypes returns: the status families keyed by name,
// and every known type keyed by name.
type doctypesResponse struct {
	Status map[string][]string    `json:"status"`
	Types  map[string]doctypeSpec `json:"types"`
}

// doctypes serves the embedded schema-v1 vocabulary. It is derived from schema.Vocab per
// request rather than snapshotted at registration, so it cannot report a menu the validation
// engine disagrees with. Like the model catalog it carries no desk data — only the contract the
// binary ships — so it is registered unauthenticated in both bind modes.
//
// A vocabulary that fails to parse is a build defect, not a runtime condition, so it surfaces
// as a 500 rather than an empty menu: an empty picker would read to the operator as "this
// document has no legal statuses", which is a lie.
func doctypes(e *core.RequestEvent) error {
	v, err := coreschema.Vocab()
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	types := make(map[string]doctypeSpec, len(v.Types))
	for name, spec := range v.Types {
		types[name] = doctypeSpec{
			Family:      spec.StatusFamily,
			Lightweight: spec.Lightweight,
			Required:    orEmpty(spec.Required),
			Optional:    orEmpty(spec.Optional),
		}
	}
	return e.JSON(http.StatusOK, doctypesResponse{Status: v.StatusFamilies, Types: types})
}

// orEmpty replaces a nil slice with an empty one so it marshals to [] rather than null.
func orEmpty(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

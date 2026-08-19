package spa

import (
	"net/http"

	"github.com/pocketbase/pocketbase/core"

	"github.com/hsb3/deskkit/internal/core/config"
)

// PathModels is the model-catalog endpoint the SPA reads to render provider/model dropdowns.
const PathModels = "/desk/models"

// catalogResponse is the wire shape PathModels returns.
type catalogResponse struct {
	Providers []string              `json:"providers"`
	Models    []config.CatalogModel `json:"models"`
}

// models serves the generated LLM model catalog (config.ModelCatalog). It carries no secrets
// — provider/model names and ids only — so it is registered in both loopback and public mode,
// unauthenticated: the SPA needs it to populate model pickers before a login token exists.
func models(e *core.RequestEvent) error {
	return e.JSON(http.StatusOK, catalogResponse{
		Providers: config.CatalogProviders(),
		Models:    config.ModelCatalog,
	})
}

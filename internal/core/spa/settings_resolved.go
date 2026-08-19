package spa

import (
	"net/http"

	"github.com/pocketbase/pocketbase/core"

	"github.com/hsb3/deskkit/internal/core/config"
	"github.com/hsb3/deskkit/internal/core/settings"
)

// PathSettingsResolved is the resolved-configuration endpoint the settings panel reads to learn
// which leg currently wins per field. Without it the panel cannot tell an editable value from one
// an environment variable already controls, so it would accept edits that never take effect.
const PathSettingsResolved = "/desk/settings/resolved"

// resolvedField is one field's resolved value plus the leg that won it — the same leg vocabulary
// `config show` prints (env / profile / store / central / default).
type resolvedField struct {
	Value  string `json:"value"`
	Source string `json:"source"`
}

// resolvedSecret is a field whose leg is reportable but whose value is NOT. The API key has no
// value member at all — an absent field cannot be leaked by a logging proxy, a browser devtools
// tab, or a future refactor that decides to render whatever it is handed. An unresolved key
// reports an empty source, which the panel renders as "not set" rather than inventing a leg.
type resolvedSecret struct {
	Source string `json:"source"`
}

// resolvedResponse is the wire shape PathSettingsResolved returns.
type resolvedResponse struct {
	Provider resolvedField  `json:"provider"`
	Model    resolvedField  `json:"model"`
	APIKey   resolvedSecret `json:"api_key"`
}

// settingsResolved re-resolves the LLM configuration chain PER REQUEST rather than reading a
// snapshot taken at registration time: the panel saves a setting and immediately re-reads this
// endpoint to confirm which leg now wins, so a process-start snapshot would answer with the leg
// that won before the save and report the operator's own change as having no effect.
//
// The chain is config.Load's env > profile > central > default, with the desk's store spliced in
// between profile and central by ApplySettings — which promotes only fields no higher leg already
// claimed, so precedence lives in the resolver rather than being restated here.
//
// A store that cannot be read is not fatal: it degrades to "no store leg", the same tolerance
// every other config consumer applies to an older or unreadable store. An unresolvable desk IS
// fatal (500) because there is then no configuration to describe; the panel's fetch degrades to
// its "cannot confirm which source is winning" banner on any failure, which is the honest state.
func settingsResolved(e *core.RequestEvent) error {
	cfg, err := config.Load()
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	s, _ := settings.Load(e.App)
	config.ApplySettings(cfg, s)

	_, keySource := config.ResolveAPIKeySettings(s, config.APIKeyEnvName(cfg))

	return e.JSON(http.StatusOK, resolvedResponse{
		Provider: resolvedField{Value: cfg.LLMProvider, Source: cfg.Sources["LLM_PROVIDER"]},
		Model:    resolvedField{Value: cfg.LLMModel, Source: cfg.Sources["LLM_MODEL"]},
		APIKey:   resolvedSecret{Source: keySource},
	})
}

package config

import (
	"os"

	"github.com/pocketbase/pocketbase/core"

	"github.com/hsb3/deskkit/internal/core/settings"
)

// SourceStore is the per-desk store leg, sitting between the profile and the machine-wide central
// file: env > profile > store > central > default.
//
// It outranks central because the store belongs to ONE desk while the central file is
// machine-wide, and it loses to the profile because a profile is the desk's own declared,
// version-controlled intent — a value typed into a browser must not silently override it.
const SourceStore = "store"

// The fields the store leg can supply, named by the env var each is documented under — which is
// also the key Config.Sources is indexed by.
const (
	keyLLMProvider = "LLM_PROVIDER"
	keyLLMModel    = "LLM_MODEL"
)

// ApplyStore re-resolves the LLM legs of an already-loaded Config against the desk's store.
//
// It is a SECOND PASS rather than a leg inside Load, because Load runs before PocketBase
// bootstraps and no store handle exists there yet. Every store-touching entry point already holds
// an app by the time it needs these values — requireConfig for one-shot commands, the OnServe hook
// for the long-running surfaces — so the pass runs at those two choke points and storeless callers
// (`config`, `desks`, `init`) keep working untouched, having simply not consulted a leg that does
// not exist for them.
//
// A store that cannot be read is not fatal: config resolution runs on every command, and an older
// or unreadable store must degrade to the legs Load already resolved rather than break the command.
func ApplyStore(app core.App, cfg *Config) error {
	s, err := settings.Load(app)
	if err != nil {
		return err
	}
	ApplySettings(cfg, s)
	return nil
}

// ApplySettings applies an already-read settings row to a resolved Config, updating Sources so a
// display surface reports the winning leg honestly.
//
// The precedence test is Sources itself: a field whose recorded source is "env" or "profile" was
// won by a leg ABOVE the store and is left alone; anything else ("central", "default") is a leg
// the store outranks. Reading the recorded source, rather than re-deriving the chain here, is what
// keeps this pass from drifting away from the resolver that wrote it.
func ApplySettings(cfg *Config, s *settings.Settings) {
	if cfg == nil || s == nil {
		return
	}
	apply := func(key, stored string, dst *string) {
		if stored == "" || !storeMayWin(cfg, key) {
			return
		}
		*dst = stored
		if cfg.Sources == nil {
			cfg.Sources = map[string]string{}
		}
		cfg.Sources[key] = SourceStore
	}
	apply(keyLLMProvider, s.LLMProvider, &cfg.LLMProvider)
	apply(keyLLMModel, s.LLMModel, &cfg.LLMModel)
}

// storeMayWin reports whether the store leg outranks whichever leg currently holds key.
func storeMayWin(cfg *Config, key string) bool {
	switch cfg.Sources[key] {
	case SourceEnv, SourceProfile:
		return false
	default:
		return true
	}
}

// ResolveAPIKeySettings is ResolveAPIKey with the store leg spliced in: the env var named by
// envName, then the desk's store, then the machine-wide central config. source is SourceEnv,
// SourceStore, SourceCentral, or "" when unresolved.
//
// The key still never lands on Config (spec §6.3) — it is resolved at use time and handed straight
// to whatever needs it, and a display surface renders the result through MaskSecret. A nil s means
// "no store consulted", which is the honest state for the surfaces that open none.
func ResolveAPIKeySettings(s *settings.Settings, envName string) (key, source string) {
	if v := os.Getenv(envName); v != "" {
		return v, SourceEnv
	}
	if s != nil && s.LLMAPIKey != "" {
		return s.LLMAPIKey, SourceStore
	}
	// Falls through to the one resolver that owns the env + central legs, so those two can never
	// drift between here and ResolveAPIKey's other callers.
	return ResolveAPIKey(envName)
}

// WithStore returns a COPY of cfg with the store leg re-resolved against app — the per-use form
// of ApplyStore, for the long-lived surfaces.
//
// ApplyStore runs ONCE, at serve start, against the process-wide Config. That is enough for a
// one-shot command, but a browser that saves a provider/model into the store expects the very next
// chat session to use it, and a snapshot taken at process start cannot see a write that happened
// after it (on a hosted desk the operator has no restart short of a redeploy). Every long-lived
// consumer therefore re-resolves here, at the moment it builds something from the config.
//
// It returns a copy rather than mutating cfg because the caller is a request goroutine and cfg is
// shared with the rest of the process: mutating it would both race and let one browser session's
// choice leak into unrelated work. The copy is deep enough to cover Sources — a plain struct copy
// would still share that map, which ApplySettings writes to.
//
// A store that cannot be read degrades to the startup values (logged, never fatal): a chat request
// must not fail because a lower resolution leg is unreadable.
func WithStore(app core.App, cfg *Config) *Config {
	if cfg == nil {
		return nil
	}
	c := *cfg
	c.Sources = make(map[string]string, len(cfg.Sources))
	for k, v := range cfg.Sources {
		c.Sources[k] = v
	}
	if err := ApplyStore(app, &c); err != nil && app != nil {
		// ApplyStore applies nothing when the read fails, so c still holds the startup values.
		app.Logger().Error("read store settings; using startup configuration", "err", err)
	}
	return &c
}

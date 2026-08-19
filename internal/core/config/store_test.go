package config

import (
	"testing"

	"github.com/hsb3/deskkit/internal/core/settings"
)

func cfgWith(sources map[string]string, provider, model string) *Config {
	return &Config{LLMProvider: provider, LLMModel: model, Sources: sources}
}

// TestApplySettings_StoreBeatsCentralAndDefault: the store is PER-DESK, so it outranks the
// machine-wide central file and, obviously, the built-in default.
func TestApplySettings_StoreBeatsCentralAndDefault(t *testing.T) {
	for _, loser := range []string{SourceCentral, SourceDefault} {
		cfg := cfgWith(map[string]string{
			"LLM_PROVIDER": loser,
			"LLM_MODEL":    loser,
		}, "anthropic", "claude-from-"+loser)

		ApplySettings(cfg, &settings.Settings{LLMProvider: "openai", LLMModel: "gpt-from-store"})

		if cfg.LLMProvider != "openai" || cfg.Sources["LLM_PROVIDER"] != SourceStore {
			t.Errorf("provider over %s = %q (source %q), want openai from %q",
				loser, cfg.LLMProvider, cfg.Sources["LLM_PROVIDER"], SourceStore)
		}
		if cfg.LLMModel != "gpt-from-store" || cfg.Sources["LLM_MODEL"] != SourceStore {
			t.Errorf("model over %s = %q (source %q), want the store value", loser, cfg.LLMModel, cfg.Sources["LLM_MODEL"])
		}
	}
}

// TestApplySettings_EnvAndProfileStillBeatStore: a desk that DECLARES its model in profile.yaml,
// or an operator who exports it for one command, must not be silently overridden by a value a
// browser typed into the store once.
func TestApplySettings_EnvAndProfileStillBeatStore(t *testing.T) {
	for _, winner := range []string{SourceEnv, SourceProfile} {
		cfg := cfgWith(map[string]string{
			"LLM_PROVIDER": winner,
			"LLM_MODEL":    winner,
		}, "gemini", "gemini-from-"+winner)

		ApplySettings(cfg, &settings.Settings{LLMProvider: "openai", LLMModel: "gpt-from-store"})

		if cfg.LLMProvider != "gemini" || cfg.Sources["LLM_PROVIDER"] != winner {
			t.Errorf("the store overrode the %s leg: provider = %q (source %q)",
				winner, cfg.LLMProvider, cfg.Sources["LLM_PROVIDER"])
		}
		if cfg.LLMModel != "gemini-from-"+winner || cfg.Sources["LLM_MODEL"] != winner {
			t.Errorf("the store overrode the %s leg: model = %q (source %q)",
				winner, cfg.LLMModel, cfg.Sources["LLM_MODEL"])
		}
	}
}

// TestApplySettings_EmptyStoreChangesNothing: an unset store field is "no opinion", never an
// instruction to blank the value that a lower leg supplied.
func TestApplySettings_EmptyStoreChangesNothing(t *testing.T) {
	cfg := cfgWith(map[string]string{"LLM_PROVIDER": SourceDefault, "LLM_MODEL": SourceDefault},
		"anthropic", "claude-default")

	ApplySettings(cfg, &settings.Settings{})

	if cfg.LLMProvider != "anthropic" || cfg.Sources["LLM_PROVIDER"] != SourceDefault {
		t.Errorf("an empty store row rewrote the provider: %q (%q)", cfg.LLMProvider, cfg.Sources["LLM_PROVIDER"])
	}
	if cfg.LLMModel != "claude-default" || cfg.Sources["LLM_MODEL"] != SourceDefault {
		t.Errorf("an empty store row rewrote the model: %q (%q)", cfg.LLMModel, cfg.Sources["LLM_MODEL"])
	}
}

// TestApplySettings_NilInputsAreNoOps: config resolution runs on every command, so the pass must
// survive a store that could not be read and a Config built without a Sources map.
func TestApplySettings_NilInputsAreNoOps(t *testing.T) {
	ApplySettings(nil, &settings.Settings{LLMProvider: "openai"})

	cfg := &Config{LLMProvider: "anthropic"} // no Sources map at all
	ApplySettings(cfg, &settings.Settings{LLMProvider: "openai"})
	if cfg.LLMProvider != "openai" {
		t.Errorf("a Config with no Sources map must still take the store value, got %q", cfg.LLMProvider)
	}
	if cfg.Sources["LLM_PROVIDER"] != SourceStore {
		t.Errorf("the pass must create the Sources map it records into, got %q", cfg.Sources["LLM_PROVIDER"])
	}

	ApplySettings(cfg, nil)
	if cfg.LLMProvider != "openai" {
		t.Errorf("a nil settings pointer must change nothing, got %q", cfg.LLMProvider)
	}
}

// TestResolveAPIKeySettings_Precedence: env > store > central, and the store leg reports itself
// honestly so a display surface never mislabels where the key came from.
func TestResolveAPIKeySettings_Precedence(t *testing.T) {
	const envName = "DESKKIT_TEST_LLM_KEY"
	t.Setenv("XDG_CONFIG_HOME", t.TempDir()) // isolate from the developer's real central config

	if key, source := ResolveAPIKeySettings(nil, envName); key != "" || source != "" {
		t.Fatalf("unresolved key = (%q, %q), want empty", key, source)
	}

	central := &Central{}
	central.LLM.APIKey = "from-central"
	if err := SaveCentral(central); err != nil {
		t.Fatalf("SaveCentral: %v", err)
	}

	if key, source := ResolveAPIKeySettings(nil, envName); key != "from-central" || source != SourceCentral {
		t.Fatalf("central-only = (%q, %q), want (from-central, %s)", key, source, SourceCentral)
	}

	store := &settings.Settings{LLMAPIKey: "from-store"}
	if key, source := ResolveAPIKeySettings(store, envName); key != "from-store" || source != SourceStore {
		t.Fatalf("store over central = (%q, %q), want (from-store, %s)", key, source, SourceStore)
	}

	t.Setenv(envName, "from-env")
	if key, source := ResolveAPIKeySettings(store, envName); key != "from-env" || source != SourceEnv {
		t.Fatalf("env over store = (%q, %q), want (from-env, %s)", key, source, SourceEnv)
	}
}

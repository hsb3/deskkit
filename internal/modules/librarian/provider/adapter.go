// Package provider is the provider-adapter slice (spec §6.2/§6.3). It builds a single
// model.ToolCallingChatModel from config, selecting the concrete eino-ext component by
// LLM_PROVIDER. The rest of the system depends only on the model.ToolCallingChatModel
// interface — never on a concrete provider type (spec §6.2).
//
// Identity-neutral: API keys are read at construction time only, never stored in Config
// (spec §6.3, §3.4). A required key that is unset fails LOUD with a clear error (not a panic,
// not a deferred failure on first API call) so `agent` without a key exits cleanly with an
// actionable message (Phase-1 acceptance).
//
// Key resolution: env, then the desk's store-backed settings row, then the machine-wide central
// config's llm.api_key. On the env leg a profile may set secrets_ref.llm_api_key to the NAME of
// the env var holding the key (surfaced as cfg.LLMAPIKeyEnv); when set, that named var is read
// instead of the per-provider default (ANTHROPIC_API_KEY / OPENAI_API_KEY / GEMINI_API_KEY). The
// store leg exists because a hosted desk has no shell: its only writable, redeploy-surviving
// state is the store, so a key installed from the browser must resolve here. Either way the
// secret VALUE lives only in the environment, that store row, or the 0600 central file — never in
// Config, and never in a log line.
package provider

import (
	"context"
	"fmt"

	claude "github.com/cloudwego/eino-ext/components/model/claude"
	gemini "github.com/cloudwego/eino-ext/components/model/gemini"
	openai "github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/pocketbase/pocketbase/core"
	"google.golang.org/genai"

	"github.com/hsb3/deskkit/internal/core/config"
	"github.com/hsb3/deskkit/internal/core/settings"
)

// NewChatModel constructs the provider-selected chat model (spec §6.3). Selection is purely
// config-driven (LLM_PROVIDER + LLM_MODEL); LLM_MAX_TOKENS (default 4096) applies across all
// three providers. Note the OpenAI/Gemini configs take a *int MaxTokens, and Gemini takes a
// constructed *genai.Client rather than a raw API key.
//
// app is the desk's store handle, consulted for the store leg of key resolution; a nil app simply
// means that leg is unavailable (storeless callers keep working on env + central alone).
func NewChatModel(ctx context.Context, app core.App, cfg *config.Config) (model.ToolCallingChatModel, error) {
	switch cfg.LLMProvider {
	case "anthropic":
		key, envName, _ := resolveAPIKey(app, cfg)
		if key == "" {
			return nil, missingKeyErr("anthropic", envName)
		}
		return claude.NewChatModel(ctx, &claude.Config{
			APIKey:    key,
			Model:     cfg.LLMModel, // default claude-haiku-4-5-20251001
			MaxTokens: cfg.LLMMaxTokens,
		})
	case "openai":
		key, envName, _ := resolveAPIKey(app, cfg)
		if key == "" {
			return nil, missingKeyErr("openai", envName)
		}
		maxTokens := cfg.LLMMaxTokens
		return openai.NewChatModel(ctx, &openai.ChatModelConfig{
			APIKey:    key,
			Model:     cfg.LLMModel, // e.g. gpt-5.4
			MaxTokens: &maxTokens,   // *int on the OpenAI config
		})
	case "gemini":
		key, envName, _ := resolveAPIKey(app, cfg)
		if key == "" {
			return nil, missingKeyErr("gemini", envName)
		}
		client, err := genai.NewClient(ctx, &genai.ClientConfig{APIKey: key})
		if err != nil {
			return nil, fmt.Errorf("gemini: construct client: %w", err)
		}
		maxTokens := cfg.LLMMaxTokens
		return gemini.NewChatModel(ctx, &gemini.Config{
			Client:    client,
			Model:     cfg.LLMModel, // e.g. gemini-3-pro-preview
			MaxTokens: &maxTokens,   // *int on the Gemini config
		})
	default:
		return nil, fmt.Errorf("unknown LLM_PROVIDER %q (want anthropic|openai|gemini)", cfg.LLMProvider)
	}
}

// resolveAPIKey resolves the LLM API key: the env var named by cfg.LLMAPIKeyEnv (the profile's
// secrets_ref.llm_api_key indirection) or the per-provider default var, then the desk's store,
// then the central config's llm.api_key. envName is always the env var that WOULD hold the key,
// so the fail-loud message can name it. source is "env", "store", "central", or "" when
// unresolved.
//
// The key VALUE is never stored in Config (spec §6.3) — it is read here, at construction time,
// and handed straight to the concrete component. A store or central config that cannot be read is
// treated as absent rather than fatal, matching how config resolution degrades: an unreadable
// lower leg must not take down a command whose key may well come from a higher one.
func resolveAPIKey(app core.App, cfg *config.Config) (key, envName, source string) {
	envName = config.APIKeyEnvName(cfg)
	s, err := settings.Load(app)
	if err != nil {
		s = nil
	}
	key, source = config.ResolveAPIKeySettings(s, envName)
	return key, envName, source
}

// missingKeyErr builds the fail-loud message naming the exact env var that must be set — the
// resolved indirection target when configured, else the per-provider default — plus the two
// routes that need no env var: the browser Settings panel, which writes the key into this desk's
// store, and the machine-wide central config.
func missingKeyErr(provider, envName string) error {
	return fmt.Errorf(
		"LLM_PROVIDER=%s requires an API key: export %s (or add it to a discovered .env), "+
			"or save one in the browser Settings panel (stored per desk), "+
			"or store it in the central config with `deskkit config set llm.api_key <value>`",
		provider, envName)
}

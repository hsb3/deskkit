// Package provider is the provider-adapter slice (spec §6.2/§6.3). It builds a single
// model.ToolCallingChatModel from config, selecting the concrete eino-ext component by
// LLM_PROVIDER. The rest of the system depends only on the model.ToolCallingChatModel
// interface — never on a concrete provider type (spec §6.2).
//
// Identity-neutral: API keys are read from the environment at construction time only, never
// stored in Config (spec §6.3, §3.4). A required key that is unset fails LOUD with a clear
// error (not a panic, not a deferred failure on first API call) so `agent` without a key
// exits cleanly with an actionable message (Phase-1 acceptance).
//
// Key indirection: a profile may set secrets_ref.llm_api_key to the NAME of the env var that
// holds the API key (surfaced as cfg.LLMAPIKeyEnv). When set, that named var is read instead
// of the per-provider default (ANTHROPIC_API_KEY / OPENAI_API_KEY / GEMINI_API_KEY); when
// unset the default var is used. Either way the secret VALUE lives only in the environment.
package provider

import (
	"context"
	"fmt"
	"os"

	claude "github.com/cloudwego/eino-ext/components/model/claude"
	gemini "github.com/cloudwego/eino-ext/components/model/gemini"
	openai "github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"google.golang.org/genai"

	"github.com/example/pocket-librarian/internal/config"
)

// NewChatModel constructs the provider-selected chat model (spec §6.3). Selection is purely
// config-driven (LLM_PROVIDER + LLM_MODEL); LLM_MAX_TOKENS (default 4096) applies across all
// three providers. Note the OpenAI/Gemini configs take a *int MaxTokens, and Gemini takes a
// constructed *genai.Client rather than a raw API key.
func NewChatModel(ctx context.Context, cfg *config.Config) (model.ToolCallingChatModel, error) {
	switch cfg.LLMProvider {
	case "anthropic":
		key, envName := resolveAPIKey(cfg, "ANTHROPIC_API_KEY")
		if key == "" {
			return nil, missingKeyErr("anthropic", envName)
		}
		return claude.NewChatModel(ctx, &claude.Config{
			APIKey:    key,
			Model:     cfg.LLMModel, // default claude-opus-4-8
			MaxTokens: cfg.LLMMaxTokens,
		})
	case "openai":
		key, envName := resolveAPIKey(cfg, "OPENAI_API_KEY")
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
		key, envName := resolveAPIKey(cfg, "GEMINI_API_KEY")
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

// resolveAPIKey returns the API key value and the NAME of the env var it was read from.
// When the profile sets secrets_ref.llm_api_key (cfg.LLMAPIKeyEnv), that named var is used;
// otherwise the per-provider default var name is used. The env var VALUE is read here and
// never stored in Config (spec §6.3).
func resolveAPIKey(cfg *config.Config, defaultEnv string) (key, envName string) {
	envName = defaultEnv
	if cfg.LLMAPIKeyEnv != "" {
		envName = cfg.LLMAPIKeyEnv
	}
	return os.Getenv(envName), envName
}

// missingKeyErr builds the fail-loud message naming the exact env var that must be set —
// the resolved indirection target when configured, else the per-provider default.
func missingKeyErr(provider, envName string) error {
	return fmt.Errorf(
		"LLM_PROVIDER=%s requires %s, which is not set; export it or add it to a discovered .env",
		provider, envName)
}

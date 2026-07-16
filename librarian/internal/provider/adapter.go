// Package provider is the provider-adapter slice (spec §6.2/§6.3). It builds a single
// model.ToolCallingChatModel from config, selecting the concrete eino-ext component by
// LLM_PROVIDER. The rest of the system depends only on the model.ToolCallingChatModel
// interface — never on a concrete provider type (spec §6.2).
//
// Identity-neutral: API keys are read from the environment at construction time only, never
// stored in Config (spec §6.3, §3.4). A required key that is unset fails LOUD with a clear
// error (not a panic, not a deferred failure on first API call) so `agent` without a key
// exits cleanly with an actionable message (Phase-1 acceptance).
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
		key := os.Getenv("ANTHROPIC_API_KEY")
		if key == "" {
			return nil, fmt.Errorf(
				"LLM_PROVIDER=anthropic requires ANTHROPIC_API_KEY, which is not set; " +
					"export it or add it to a discovered .env")
		}
		return claude.NewChatModel(ctx, &claude.Config{
			APIKey:    key,
			Model:     cfg.LLMModel, // default claude-opus-4-8
			MaxTokens: cfg.LLMMaxTokens,
		})
	case "openai":
		key := os.Getenv("OPENAI_API_KEY")
		if key == "" {
			return nil, fmt.Errorf(
				"LLM_PROVIDER=openai requires OPENAI_API_KEY, which is not set; " +
					"export it or add it to a discovered .env")
		}
		maxTokens := cfg.LLMMaxTokens
		return openai.NewChatModel(ctx, &openai.ChatModelConfig{
			APIKey:    key,
			Model:     cfg.LLMModel, // e.g. gpt-5.4
			MaxTokens: &maxTokens,   // *int on the OpenAI config
		})
	case "gemini":
		key := os.Getenv("GEMINI_API_KEY")
		if key == "" {
			return nil, fmt.Errorf(
				"LLM_PROVIDER=gemini requires GEMINI_API_KEY, which is not set; " +
					"export it or add it to a discovered .env")
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

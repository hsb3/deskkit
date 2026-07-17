package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/example/pocket-librarian/internal/config"
)

// These tests exercise the provider-selection switch and the loud missing-key failure
// WITHOUT any network call: every case returns before constructing the concrete component.
// The positive construction path (a real key present) is covered by the foreman's live smoke,
// not here, so the unit suite never needs a credential.

func TestNewChatModel_UnknownProvider(t *testing.T) {
	cfg := &config.Config{LLMProvider: "grok", LLMModel: "x", LLMMaxTokens: 4096}
	m, err := NewChatModel(context.Background(), cfg)
	if err == nil {
		t.Fatalf("expected error for unknown provider, got model %v", m)
	}
	if !strings.Contains(err.Error(), "grok") {
		t.Fatalf("error should name the bad provider, got %q", err.Error())
	}
}

func TestResolveAPIKey_Fallback(t *testing.T) {
	// No profile indirection (LLMAPIKeyEnv empty) -> read the per-provider default var.
	t.Setenv("ANTHROPIC_API_KEY", "sk-default-value")
	cfg := &config.Config{LLMProvider: "anthropic"}
	key, envName := resolveAPIKey(cfg, "ANTHROPIC_API_KEY")
	if envName != "ANTHROPIC_API_KEY" {
		t.Fatalf("envName = %q, want ANTHROPIC_API_KEY", envName)
	}
	if key != "sk-default-value" {
		t.Fatalf("key = %q, want the default-var value", key)
	}
}

func TestResolveAPIKey_Indirection(t *testing.T) {
	// secrets_ref.llm_api_key names a custom env var; that var wins over the default one,
	// and the default var is NOT consulted even when it is also set.
	t.Setenv("ANTHROPIC_API_KEY", "sk-should-not-be-used")
	t.Setenv("DESK_LLM_KEY", "sk-indirect-value")
	cfg := &config.Config{LLMProvider: "anthropic", LLMAPIKeyEnv: "DESK_LLM_KEY"}
	key, envName := resolveAPIKey(cfg, "ANTHROPIC_API_KEY")
	if envName != "DESK_LLM_KEY" {
		t.Fatalf("envName = %q, want DESK_LLM_KEY", envName)
	}
	if key != "sk-indirect-value" {
		t.Fatalf("key = %q, want the indirection-target value (default var must be ignored)", key)
	}
}

func TestNewChatModel_MissingKeyNamesIndirectionTarget(t *testing.T) {
	// When indirection is configured but the named var is unset, the loud error must name the
	// resolved target var, not the per-provider default.
	t.Setenv("ANTHROPIC_API_KEY", "sk-present-but-not-the-target")
	t.Setenv("DESK_LLM_KEY", "") // empty == unset for our os.Getenv check
	cfg := &config.Config{LLMProvider: "anthropic", LLMModel: "x", LLMMaxTokens: 4096, LLMAPIKeyEnv: "DESK_LLM_KEY"}
	m, err := NewChatModel(context.Background(), cfg)
	if err == nil {
		t.Fatalf("expected a loud missing-key error, got model %v", m)
	}
	if !strings.Contains(err.Error(), "DESK_LLM_KEY") {
		t.Fatalf("error should name the indirection target DESK_LLM_KEY, got %q", err.Error())
	}
	if strings.Contains(err.Error(), "ANTHROPIC_API_KEY") {
		t.Fatalf("error must not name the default var when indirection is configured, got %q", err.Error())
	}
}

func TestNewChatModel_MissingKeyFailsLoud(t *testing.T) {
	cases := []struct {
		provider string
		envKey   string
	}{
		{"anthropic", "ANTHROPIC_API_KEY"},
		{"openai", "OPENAI_API_KEY"},
		{"gemini", "GEMINI_API_KEY"},
	}
	for _, tc := range cases {
		t.Run(tc.provider, func(t *testing.T) {
			t.Setenv(tc.envKey, "") // empty == unset for our os.Getenv check
			cfg := &config.Config{LLMProvider: tc.provider, LLMModel: "x", LLMMaxTokens: 4096}
			m, err := NewChatModel(context.Background(), cfg)
			if err == nil {
				t.Fatalf("%s: expected a loud missing-key error, got model %v", tc.provider, m)
			}
			if !strings.Contains(err.Error(), tc.envKey) {
				t.Fatalf("%s: error should name %s, got %q", tc.provider, tc.envKey, err.Error())
			}
		})
	}
}

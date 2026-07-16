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

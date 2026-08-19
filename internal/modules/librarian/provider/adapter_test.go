package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/hsb3/deskkit/internal/core/config"
)

// These tests exercise the provider-selection switch and the loud missing-key failure
// WITHOUT any network call: every case returns before constructing the concrete component.
// The positive construction path (a real key present) is covered by a live smoke, not here,
// so the unit suite never needs a credential.
//
// Key resolution now has a central-config leg, so every test pins XDG_CONFIG_HOME to a temp
// dir: without it a real ~/.config/deskkit/config.yaml holding an api_key would satisfy the
// missing-key cases and this suite would pass or fail per machine.
//
// These cases pass a nil app: no store, so they cover the env + central legs alone. The store
// leg has its own suite (settings_leg_test.go), which stands up a real migrated store.

func isolateCentral(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	return dir
}

// writeCentral puts a central config holding a FAKE key in the isolated config home.
func writeCentral(t *testing.T, c *config.Central) {
	t.Helper()
	if err := config.SaveCentral(c); err != nil {
		t.Fatal(err)
	}
}

func TestNewChatModel_UnknownProvider(t *testing.T) {
	isolateCentral(t)
	cfg := &config.Config{LLMProvider: "grok", LLMModel: "x", LLMMaxTokens: 4096}
	m, err := NewChatModel(context.Background(), nil, cfg)
	if err == nil {
		t.Fatalf("expected error for unknown provider, got model %v", m)
	}
	if !strings.Contains(err.Error(), "grok") {
		t.Fatalf("error should name the bad provider, got %q", err.Error())
	}
}

func TestResolveAPIKey_Fallback(t *testing.T) {
	isolateCentral(t)
	// No profile indirection (LLMAPIKeyEnv empty) -> read the per-provider default var.
	t.Setenv("ANTHROPIC_API_KEY", "sk-default-value")
	cfg := &config.Config{LLMProvider: "anthropic"}
	key, envName, source := resolveAPIKey(nil, cfg)
	if envName != "ANTHROPIC_API_KEY" {
		t.Fatalf("envName = %q, want ANTHROPIC_API_KEY", envName)
	}
	if key != "sk-default-value" {
		t.Fatalf("key = %q, want the default-var value", key)
	}
	if source != config.SourceEnv {
		t.Fatalf("source = %q, want env", source)
	}
}

// The central config supplies the key when NO relevant env var is set, and loses to the env
// var when one is. No network call: resolveAPIKey never constructs a component.
func TestResolveAPIKey_Central(t *testing.T) {
	isolateCentral(t)
	t.Setenv("ANTHROPIC_API_KEY", "") // empty == unset for the os.Getenv check
	central := &config.Central{}
	if err := central.Set("llm.api_key", "sk-central-fake-0000"); err != nil {
		t.Fatal(err)
	}
	writeCentral(t, central)

	cfg := &config.Config{LLMProvider: "anthropic"}
	key, envName, source := resolveAPIKey(nil, cfg)
	if key != "sk-central-fake-0000" || source != config.SourceCentral {
		t.Fatalf("key = %q (%s), want the central key with source central", key, source)
	}
	if envName != "ANTHROPIC_API_KEY" {
		t.Fatalf("envName = %q — the message must still name the env var that WOULD hold the key", envName)
	}

	// Env wins over central.
	t.Setenv("ANTHROPIC_API_KEY", "sk-env-fake-1111")
	key, _, source = resolveAPIKey(nil, cfg)
	if key != "sk-env-fake-1111" || source != config.SourceEnv {
		t.Fatalf("key = %q (%s), want the env key with source env", key, source)
	}
}

// A central key must satisfy the loud missing-key guard: with the key present in the central
// config, NewChatModel gets past it (it may still fail constructing the component offline —
// what must NOT happen is a missing-key error).
func TestNewChatModel_CentralKeyPassesTheGuard(t *testing.T) {
	isolateCentral(t)
	t.Setenv("ANTHROPIC_API_KEY", "")
	central := &config.Central{}
	if err := central.Set("llm.api_key", "sk-central-fake-0000"); err != nil {
		t.Fatal(err)
	}
	writeCentral(t, central)

	cfg := &config.Config{LLMProvider: "anthropic", LLMModel: "x", LLMMaxTokens: 4096}
	_, err := NewChatModel(context.Background(), nil, cfg)
	if err != nil && strings.Contains(err.Error(), "requires an API key") {
		t.Fatalf("central llm.api_key must satisfy the missing-key guard, got %q", err)
	}
}

// Nothing on the failure path may print a key: the message names env vars and the config
// command, never a value.
func TestMissingKeyErrNeverPrintsAKey(t *testing.T) {
	isolateCentral(t)
	t.Setenv("ANTHROPIC_API_KEY", "")
	central := &config.Central{}
	if err := central.Set("llm.api_key", ""); err != nil {
		t.Fatal(err)
	}
	writeCentral(t, central)

	cfg := &config.Config{LLMProvider: "anthropic", LLMModel: "x", LLMMaxTokens: 4096}
	_, err := NewChatModel(context.Background(), nil, cfg)
	if err == nil {
		t.Fatal("expected a loud missing-key error")
	}
	for _, leaked := range []string{"sk-", "api_key: "} {
		if strings.Contains(err.Error(), leaked) {
			t.Fatalf("missing-key error must not carry a key-shaped value, got %q", err.Error())
		}
	}
	if !strings.Contains(err.Error(), "deskkit config set llm.api_key") {
		t.Fatalf("missing-key error should point at the central-config route, got %q", err.Error())
	}
}

func TestResolveAPIKey_Indirection(t *testing.T) {
	isolateCentral(t)
	// secrets_ref.llm_api_key names a custom env var; that var wins over the default one,
	// and the default var is NOT consulted even when it is also set.
	t.Setenv("ANTHROPIC_API_KEY", "sk-should-not-be-used")
	t.Setenv("DESK_LLM_KEY", "sk-indirect-value")
	cfg := &config.Config{LLMProvider: "anthropic", LLMAPIKeyEnv: "DESK_LLM_KEY"}
	key, envName, _ := resolveAPIKey(nil, cfg)
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
	isolateCentral(t)
	t.Setenv("ANTHROPIC_API_KEY", "sk-present-but-not-the-target")
	t.Setenv("DESK_LLM_KEY", "") // empty == unset for our os.Getenv check
	cfg := &config.Config{LLMProvider: "anthropic", LLMModel: "x", LLMMaxTokens: 4096, LLMAPIKeyEnv: "DESK_LLM_KEY"}
	m, err := NewChatModel(context.Background(), nil, cfg)
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
	isolateCentral(t)
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
			m, err := NewChatModel(context.Background(), nil, cfg)
			if err == nil {
				t.Fatalf("%s: expected a loud missing-key error, got model %v", tc.provider, m)
			}
			if !strings.Contains(err.Error(), tc.envKey) {
				t.Fatalf("%s: error should name %s, got %q", tc.provider, tc.envKey, err.Error())
			}
		})
	}
}

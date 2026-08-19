package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/tests"

	"github.com/hsb3/deskkit/internal/core/config"
	"github.com/hsb3/deskkit/internal/core/settings"

	// Blank-import registers this project's Go migrations, so tests.NewTestApp's
	// RunAllMigrations() creates the real shipped settings collection — the test then exercises
	// the collection the product actually ships rather than a hand-built lookalike.
	_ "github.com/hsb3/deskkit/internal/modules/librarian/collections"
)

// storeApp opens a migrated throwaway store with the settings row's key set to storedKey (an
// empty storedKey leaves the seeded row's key unset). Env and central are isolated to empty in
// the same call, so whatever the adapter resolves can only have come from the store.
func storeApp(t *testing.T, storedKey string) *tests.TestApp {
	t.Helper()
	isolateCentral(t)
	t.Setenv("ANTHROPIC_API_KEY", "")

	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("tests.NewTestApp: %v", err)
	}
	t.Cleanup(app.Cleanup)

	rec, err := app.FindRecordById(settings.Collection, settings.RecordID)
	if err != nil {
		t.Fatalf("find seeded settings row: %v", err)
	}
	rec.Set(settings.FieldAPIKey, storedKey)
	if err := app.Save(rec); err != nil {
		t.Fatalf("save settings row: %v", err)
	}
	return app
}

// A key present ONLY in the store must reach the provider: this is the hosted case, where the
// operator has no shell and the env var is deliberately unset.
func TestResolveAPIKey_StoreOnly(t *testing.T) {
	app := storeApp(t, "sk-store-only-value")
	cfg := &config.Config{LLMProvider: "anthropic"}

	key, envName, source := resolveAPIKey(app, cfg)
	if envName != "ANTHROPIC_API_KEY" {
		t.Fatalf("envName = %q, want ANTHROPIC_API_KEY", envName)
	}
	if key != "sk-store-only-value" {
		t.Fatalf("key = %q, want the stored value", key)
	}
	if source != config.SourceStore {
		t.Fatalf("source = %q, want %q", source, config.SourceStore)
	}
}

// End-to-end through the exported constructor: with the key only in the store, NewChatModel must
// build a model rather than fail loud. Construction makes no network call, so this stays offline.
func TestNewChatModel_StoreOnlyKeyConstructs(t *testing.T) {
	app := storeApp(t, "sk-store-only-value")
	cfg := &config.Config{LLMProvider: "anthropic", LLMModel: "claude-haiku-4-5-20251001", LLMMaxTokens: 4096}

	m, err := NewChatModel(context.Background(), app, cfg)
	if err != nil {
		t.Fatalf("NewChatModel with a store-only key: %v", err)
	}
	if m == nil {
		t.Fatal("NewChatModel returned a nil model with no error")
	}
}

// The negative: no env, no central, and an EMPTY stored key must still fail loud.
func TestNewChatModel_EmptyStoreKeyStillFailsLoud(t *testing.T) {
	app := storeApp(t, "")
	cfg := &config.Config{LLMProvider: "anthropic", LLMModel: "claude-haiku-4-5-20251001", LLMMaxTokens: 4096}

	if _, err := NewChatModel(context.Background(), app, cfg); err == nil {
		t.Fatal("expected the missing-key error with every leg empty")
	} else if !strings.Contains(err.Error(), "requires an API key") ||
		!strings.Contains(err.Error(), "ANTHROPIC_API_KEY") {
		t.Fatalf("error should be the loud actionable one, got %q", err.Error())
	}
}

// A nil app is the storeless caller: no store leg, no panic, and the loud failure survives.
func TestNewChatModel_NilAppHasNoStoreLeg(t *testing.T) {
	isolateCentral(t)
	t.Setenv("ANTHROPIC_API_KEY", "")
	cfg := &config.Config{LLMProvider: "anthropic", LLMModel: "claude-haiku-4-5-20251001", LLMMaxTokens: 4096}

	if _, err := NewChatModel(context.Background(), nil, cfg); err == nil {
		t.Fatal("expected the missing-key error with a nil app and no other leg")
	}
}

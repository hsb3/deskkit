package agent

import (
	"context"
	"sync"
	"testing"

	"github.com/cloudwego/eino/components/model"
	"github.com/pocketbase/pocketbase/core"

	"github.com/hsb3/deskkit/internal/core/config"
	"github.com/hsb3/deskkit/internal/core/settings"
)

// storeProviderModel writes a provider/model pair into the desk's singleton settings row — the
// exact write the browser Settings panel performs.
func storeProviderModel(t *testing.T, app core.App, provider, model string) {
	t.Helper()
	rec, err := app.FindRecordById(settings.Collection, settings.RecordID)
	if err != nil {
		t.Fatalf("find seeded settings row: %v", err)
	}
	rec.Set(settings.FieldProvider, provider)
	rec.Set(settings.FieldModel, model)
	if err := app.Save(rec); err != nil {
		t.Fatalf("save settings row: %v", err)
	}
}

// THE regression this whole change exists for: the process starts with one provider/model, the
// operator saves a different pair from the browser, and the very next session must use the saved
// pair — WITHOUT a restart. Before the fix the session read the long-lived startup Config and the
// saved values were invisible until the process was restarted (on a hosted desk: a redeploy).
func TestNewSession_PicksUpStoreProviderModelWithoutRestart(t *testing.T) {
	app, cfg := newSessionTestEnv(t)
	cfg.Sources = map[string]string{"LLM_PROVIDER": config.SourceDefault, "LLM_MODEL": config.SourceDefault}

	storeProviderModel(t, app, "gemini", "stored-model")

	var seen *config.Config
	orig := chatModelFactory
	chatModelFactory = func(ctx context.Context, app core.App, c *config.Config) (model.ToolCallingChatModel, error) {
		seen = c
		return &fakeChatModel{}, nil
	}
	t.Cleanup(func() { chatModelFactory = orig })

	sess, err := NewSession(context.Background(), app, cfg)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	t.Cleanup(func() { _ = sess.Close(context.Background()) })

	if seen == nil {
		t.Fatal("chat model was never built")
	}
	if seen.LLMProvider != "gemini" || seen.LLMModel != "stored-model" {
		t.Fatalf("session built its model from provider/model %q/%q, want the stored gemini/stored-model",
			seen.LLMProvider, seen.LLMModel)
	}
	if seen.Sources["LLM_PROVIDER"] != config.SourceStore || seen.Sources["LLM_MODEL"] != config.SourceStore {
		t.Fatalf("Sources = %v, want both fields attributed to the store leg", seen.Sources)
	}

	// The run row is the audit record of what actually ran, so it must name the stored pair too.
	run, err := app.FindRecordById("agent_runs", sess.RunID())
	if err != nil {
		t.Fatalf("find run row: %v", err)
	}
	if run.GetString("provider") != "gemini" || run.GetString("model") != "stored-model" {
		t.Fatalf("run row recorded %q/%q, want gemini/stored-model",
			run.GetString("provider"), run.GetString("model"))
	}

	// The shared startup Config belongs to the whole process; a session resolves onto a copy.
	if cfg.LLMProvider != "openai" || cfg.LLMModel != "test-model" {
		t.Fatalf("startup cfg was mutated to %q/%q", cfg.LLMProvider, cfg.LLMModel)
	}
	if cfg.Sources["LLM_PROVIDER"] != config.SourceDefault {
		t.Fatalf("startup cfg.Sources was mutated: %v", cfg.Sources)
	}
}

// The per-session re-resolve runs on a request goroutine while other goroutines still read the
// shared startup Config, so the copy must be deep enough to cover Sources (a shallow struct copy
// shares that map). Run under -race, this fails if the map is shared.
func TestNewSession_ConcurrentReresolveLeavesSharedConfigRaceFree(t *testing.T) {
	app, cfg := newSessionTestEnv(t)
	cfg.Sources = map[string]string{"LLM_PROVIDER": config.SourceDefault, "LLM_MODEL": config.SourceDefault}
	storeProviderModel(t, app, "gemini", "stored-model")

	orig := chatModelFactory
	chatModelFactory = func(ctx context.Context, app core.App, c *config.Config) (model.ToolCallingChatModel, error) {
		return &fakeChatModel{}, nil
	}
	t.Cleanup(func() { chatModelFactory = orig })

	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sess, err := NewSession(context.Background(), app, cfg)
			if err != nil {
				t.Errorf("NewSession: %v", err)
				return
			}
			_ = sess.Close(context.Background())
		}()
	}
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				for k, v := range cfg.Sources {
					_, _ = k, v
				}
			}
		}()
	}
	wg.Wait()
}

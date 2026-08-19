package spa

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

// pinEnv fixes every leg the resolved-settings endpoint reads, so the assertions below describe
// THIS test and not the machine running it: config.Load consults a walk-up .env and the operator's
// XDG config home, and an already-set variable is the one thing it will never override.
func pinEnv(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir()) // empty central leg
	t.Setenv("DESK_ROOT", t.TempDir())       // config.Load errors without a resolvable desk
	t.Setenv("DESK_NAME", "resolved-probe")
	t.Setenv("LLM_PROVIDER", "openai")
	t.Setenv("LLM_MODEL", "gpt-4.1-mini")
	t.Setenv("OPENAI_API_KEY", "sk-not-a-real-key")
}

// authTokenFor mints a real auth token for a fresh record in the named collection, so the
// public-mode probes below present the header a live client would.
func authTokenFor(t *testing.T, app *tests.TestApp, collection, email string) string {
	t.Helper()
	col, err := app.FindCollectionByNameOrId(collection)
	if err != nil {
		t.Fatalf("find %s: %v", collection, err)
	}
	rec := core.NewRecord(col)
	rec.SetEmail(email)
	rec.SetPassword("a-sufficiently-long-password")
	rec.SetVerified(true)
	if err := app.Save(rec); err != nil {
		t.Fatalf("save %s record: %v", collection, err)
	}
	tok, err := rec.NewAuthToken()
	if err != nil {
		t.Fatalf("mint auth token: %v", err)
	}
	return tok
}

func getResolved(t *testing.T, url, token string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if token != "" {
		req.Header.Set("Authorization", token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	return resp
}

// TestSettingsResolved_Shape pins the wire contract the settings panel is already written
// against: a value+source per LLM field, a source-only api_key, and — the behavior the whole
// endpoint exists for — an env-supplied field reported as won by "env", which is what locks the
// field in the panel instead of accepting an edit that will never take effect.
func TestSettingsResolved_Shape(t *testing.T) {
	pinEnv(t)
	// The literal below is the path the browser client hardcodes; the constant must match it.
	if PathSettingsResolved != "/desk/settings/resolved" {
		t.Fatalf("PathSettingsResolved = %q, want the path the settings panel fetches", PathSettingsResolved)
	}
	srv, _ := newTestServer(t, false)

	resp := getResolved(t, srv.URL+"/desk/settings/resolved", "")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, want 200; body %s", resp.StatusCode, body)
	}
	var raw map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	for _, key := range []string{"provider", "model", "api_key"} {
		if _, ok := raw[key]; !ok {
			t.Fatalf("response has no %q field: %v", key, raw)
		}
	}
	provider, ok := raw["provider"].(map[string]any)
	if !ok {
		t.Fatalf("provider is not an object: %v", raw["provider"])
	}
	if provider["value"] != "openai" || provider["source"] != "env" {
		t.Fatalf("provider = %v, want value openai / source env", provider)
	}
	model, ok := raw["model"].(map[string]any)
	if !ok {
		t.Fatalf("model is not an object: %v", raw["model"])
	}
	if model["value"] != "gpt-4.1-mini" || model["source"] != "env" {
		t.Fatalf("model = %v, want value gpt-4.1-mini / source env", model)
	}

	apiKey, ok := raw["api_key"].(map[string]any)
	if !ok {
		t.Fatalf("api_key is not an object: %v", raw["api_key"])
	}
	// The key itself must never cross the wire — not even masked, not even to a superuser.
	if _, leaked := apiKey["value"]; leaked {
		t.Fatalf("api_key carries a value field: %v", apiKey)
	}
	if apiKey["source"] != "env" {
		t.Fatalf("api_key source = %v, want env", apiKey["source"])
	}
}

// TestSettingsResolved_PublicRequiresSuperuser: this endpoint reports how the desk is configured,
// so on a public bind it is superuser-only — an anonymous caller and a `users`-collection token
// are both refused, and only the operator's own credential reads it.
func TestSettingsResolved_PublicRequiresSuperuser(t *testing.T) {
	pinEnv(t)
	srv, app := newTestServer(t, true)

	anon := getResolved(t, srv.URL+"/desk/settings/resolved", "")
	anon.Body.Close()
	if anon.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthenticated public-mode status = %d, want 401", anon.StatusCode)
	}

	member := getResolved(t, srv.URL+"/desk/settings/resolved",
		authTokenFor(t, app, "users", "member@desk.test"))
	member.Body.Close()
	if member.StatusCode == http.StatusOK {
		t.Fatal("a non-superuser token read the resolved config in public mode")
	}
	if member.StatusCode != http.StatusForbidden {
		t.Fatalf("non-superuser public-mode status = %d, want 403", member.StatusCode)
	}

	op := getResolved(t, srv.URL+"/desk/settings/resolved",
		authTokenFor(t, app, core.CollectionNameSuperusers, "op@desk.test"))
	defer op.Body.Close()
	if op.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(op.Body)
		t.Fatalf("superuser public-mode status = %d, want 200; body %s", op.StatusCode, body)
	}
}

package spa

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/hsb3/deskkit/internal/core/config"
)

// resolvedBody fetches the resolved-settings endpoint and returns its decoded generic JSON, so
// the assertions inspect the RAW wire shape the browser receives.
func resolvedBody(t *testing.T, srvURL string) map[string]any {
	t.Helper()
	resp := getResolved(t, srvURL+PathSettingsResolved, "")
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200; body %s", resp.StatusCode, body)
	}
	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		t.Fatalf("decode body (%v): %s", err, body)
	}
	return raw
}

// field pulls one resolvedField object out of the response, failing if it is absent — an absent
// key and an empty one are different contracts, and only the latter is allowed.
func field(t *testing.T, raw map[string]any, key string) map[string]any {
	t.Helper()
	v, ok := raw[key]
	if !ok {
		t.Fatalf("response has no %q field: %v", key, raw)
	}
	obj, ok := v.(map[string]any)
	if !ok {
		t.Fatalf("%q is not an object: %v", key, v)
	}
	return obj
}

// TestSettingsResolved_EditorURLAndDeskRoot: the browser expands the editor URL template's
// {abs} placeholder itself, so it needs BOTH the template and the desk root from the same
// resolved report — fetching them from two endpoints could pair a template with a root from a
// different desk. Both ride the existing resolvedField shape and leg vocabulary.
func TestSettingsResolved_EditorURLAndDeskRoot(t *testing.T) {
	pinEnv(t)
	const tmpl = "x-editor://open?path={abs}"
	t.Setenv("EDITOR_URL", tmpl)
	srv, _ := newTestServer(t, false)

	raw := resolvedBody(t, srv.URL)

	editor := field(t, raw, "editor_url")
	if editor["value"] != tmpl {
		t.Errorf("editor_url value = %v, want %q", editor["value"], tmpl)
	}
	if editor["source"] != config.SourceEnv {
		t.Errorf("editor_url source = %v, want %q", editor["source"], config.SourceEnv)
	}

	root := field(t, raw, "desk_root")
	if root["value"] != os.Getenv("DESK_ROOT") {
		t.Errorf("desk_root value = %v, want the resolved desk root %q", root["value"], os.Getenv("DESK_ROOT"))
	}
	if root["source"] != config.SourceEnv {
		t.Errorf("desk_root source = %v, want %q", root["source"], config.SourceEnv)
	}

	// The pre-existing fields must survive the addition — this endpoint already has a client.
	for _, key := range []string{"provider", "model", "api_key"} {
		if _, ok := raw[key]; !ok {
			t.Errorf("adding editor_url dropped the %q field", key)
		}
	}
}

// TestSettingsResolved_EditorURLUnsetIsEmptySourceToo: an unset editor_url reports an empty
// value AND an empty source. Reporting the leg as "default" would be a lie — there is no
// default editor to have won — and the browser keys "render no hand-off button" off exactly
// this state, so the key must still be present rather than absent.
func TestSettingsResolved_EditorURLUnsetIsEmptySourceToo(t *testing.T) {
	pinEnv(t)
	os.Unsetenv("EDITOR_URL")
	srv, _ := newTestServer(t, false)

	editor := field(t, resolvedBody(t, srv.URL), "editor_url")
	if editor["value"] != "" {
		t.Errorf("unset editor_url value = %v, want empty", editor["value"])
	}
	if editor["source"] != "" {
		t.Errorf("unset editor_url source = %v, want empty (no leg won)", editor["source"])
	}
}

// TestSettingsResolved_EditorURLFromProfile: the leg the contract actually names. A desk
// declares its editor in _knowledge/profile.yaml, and the endpoint must report it as won by
// "profile" — that string is what tells the panel the value came from the desk's own declared
// config rather than the operator's shell.
func TestSettingsResolved_EditorURLFromProfile(t *testing.T) {
	pinEnv(t)
	os.Unsetenv("EDITOR_URL")

	// config.Load discovers the profile by walking up from the process cwd.
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, config.ProfileRootDir), 0o755); err != nil {
		t.Fatal(err)
	}
	const tmpl = "x-editor://file/{path}"
	body := "desk:\n  name: resolved-probe\npreferences:\n  editor_url: \"" + tmpl + "\"\n"
	if err := os.WriteFile(filepath.Join(dir, config.ProfileRootDir, "profile.yaml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	prev, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })

	srv, _ := newTestServer(t, false)
	editor := field(t, resolvedBody(t, srv.URL), "editor_url")
	if editor["value"] != tmpl {
		t.Errorf("profile editor_url value = %v, want %q", editor["value"], tmpl)
	}
	if editor["source"] != config.SourceProfile {
		t.Errorf("profile editor_url source = %v, want %q", editor["source"], config.SourceProfile)
	}
}

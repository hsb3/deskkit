package spa

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

// TestModelsCatalog pins the /desk/models shape the frontend depends on: 200 JSON, a non-empty
// model list, and every model's provider drawn from the deskkit provider vocabulary
// (anthropic | openai | gemini — internal/modules/librarian/provider/adapter.go).
func TestModelsCatalog(t *testing.T) {
	srv, _ := newTestServer(t, false)
	resp, err := http.Get(srv.URL + PathModels)
	if err != nil {
		t.Fatalf("GET %s: %v", PathModels, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("GET %s status = %d, body %s", PathModels, resp.StatusCode, body)
	}

	var payload catalogResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode %s body: %v", PathModels, err)
	}

	if len(payload.Models) == 0 {
		t.Fatal("catalog response has no models")
	}
	if len(payload.Providers) == 0 {
		t.Fatal("catalog response has no providers")
	}

	validProvider := map[string]bool{"anthropic": true, "openai": true, "gemini": true}
	for _, p := range payload.Providers {
		if !validProvider[p] {
			t.Fatalf("providers list contains %q, want one of anthropic|openai|gemini", p)
		}
	}
	for _, m := range payload.Models {
		if !validProvider[m.Provider] {
			t.Fatalf("model %q has provider %q, want one of anthropic|openai|gemini", m.ID, m.Provider)
		}
		if m.ID == "" || m.Name == "" {
			t.Fatalf("model missing id or name: %+v", m)
		}
	}
}

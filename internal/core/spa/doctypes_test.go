package spa

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	coreschema "github.com/hsb3/deskkit/internal/core/schema"
)

// getDoctypes fetches the vocabulary endpoint and returns the decoded generic JSON, so the
// assertions below can inspect the RAW shape (null vs []) the browser actually receives —
// decoding into the Go struct would silently normalize a null back into an empty slice and
// hide exactly the defect the never-null contract exists to prevent.
func getDoctypes(t *testing.T, url string) (int, map[string]any, []byte) {
	t.Helper()
	resp, err := http.Get(url + PathDoctypes)
	if err != nil {
		t.Fatalf("GET %s: %v", PathDoctypes, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	var raw map[string]any
	if resp.StatusCode == http.StatusOK {
		if err := json.Unmarshal(body, &raw); err != nil {
			t.Fatalf("GET %s body is not JSON (%v): %s", PathDoctypes, err, body)
		}
	}
	return resp.StatusCode, raw, body
}

// TestDoctypes_MatchesVocabulary pins the wire contract against the SOURCE of the vocabulary
// rather than a transcribed literal: every status family and every type is re-derived from the
// embedded schema, so adding a doc type to schema/doctypes.yaml can never leave the endpoint
// serving a stale menu that the browser then renders as the legal choices.
func TestDoctypes_MatchesVocabulary(t *testing.T) {
	// The literal below is the path the browser client fetches; the constant must match it.
	if PathDoctypes != "/desk/doctypes" {
		t.Fatalf("PathDoctypes = %q, want the path the SPA fetches", PathDoctypes)
	}
	srv, _ := newTestServer(t, false)

	status, raw, body := getDoctypes(t, srv.URL)
	if status != http.StatusOK {
		t.Fatalf("GET %s status = %d, want 200; body %s", PathDoctypes, status, body)
	}

	vocab, err := coreschema.Vocab()
	if err != nil {
		t.Fatalf("schema.Vocab: %v", err)
	}

	gotStatus, ok := raw["status"].(map[string]any)
	if !ok {
		t.Fatalf("response has no status object: %s", body)
	}
	if len(gotStatus) != len(vocab.StatusFamilies) {
		t.Fatalf("status families = %d, want %d (the embedded vocabulary's)", len(gotStatus), len(vocab.StatusFamilies))
	}
	for family, want := range vocab.StatusFamilies {
		got, ok := gotStatus[family].([]any)
		if !ok {
			t.Fatalf("status family %q missing or not a list: %v", family, gotStatus[family])
		}
		if len(got) != len(want) {
			t.Fatalf("status family %q has %d values, want %d", family, len(got), len(want))
		}
		for i, v := range want {
			if got[i] != v {
				t.Fatalf("status family %q value %d = %v, want %q", family, i, got[i], v)
			}
		}
	}

	gotTypes, ok := raw["types"].(map[string]any)
	if !ok {
		t.Fatalf("response has no types object: %s", body)
	}
	if len(gotTypes) != len(vocab.Types) {
		t.Fatalf("types = %d, want %d (the embedded vocabulary's)", len(gotTypes), len(vocab.Types))
	}
	for name, spec := range vocab.Types {
		entry, ok := gotTypes[name].(map[string]any)
		if !ok {
			t.Fatalf("type %q missing or not an object: %v", name, gotTypes[name])
		}
		if entry["family"] != spec.StatusFamily {
			t.Errorf("type %q family = %v, want %q", name, entry["family"], spec.StatusFamily)
		}
		if entry["lightweight"] != spec.Lightweight {
			t.Errorf("type %q lightweight = %v, want %v", name, entry["lightweight"], spec.Lightweight)
		}
		for key, want := range map[string][]string{"required": spec.Required, "optional": spec.Optional} {
			// Never null: the contract promises both keys are always present as a list so the
			// browser needs no absent-field branch. A nil Go slice marshals to null, which is
			// exactly the shape a `.map()` in the SPA would throw on.
			got, ok := entry[key].([]any)
			if !ok {
				t.Fatalf("type %q %s = %#v, want a JSON list (never null)", name, key, entry[key])
			}
			if len(got) != len(want) {
				t.Fatalf("type %q %s has %d entries, want %d", name, key, len(got), len(want))
			}
			for i, w := range want {
				if got[i] != w {
					t.Errorf("type %q %s[%d] = %v, want %q", name, key, i, got[i], w)
				}
			}
		}
	}
}

// TestDoctypes_LightweightHasEmptyFamilyAndLists pins the two shapes the contract calls out by
// name — a lightweight type reports family "" and both lists as [] — on a concrete type, so a
// regression is legible without cross-referencing the vocabulary.
func TestDoctypes_LightweightHasEmptyFamilyAndLists(t *testing.T) {
	srv, _ := newTestServer(t, false)
	_, raw, body := getDoctypes(t, srv.URL)

	types, _ := raw["types"].(map[string]any)
	journal, ok := types["journal"].(map[string]any)
	if !ok {
		t.Fatalf("no journal type in the response: %s", body)
	}
	if journal["family"] != "" {
		t.Errorf("journal family = %v, want \"\" (lightweight types have no status family)", journal["family"])
	}
	if journal["lightweight"] != true {
		t.Errorf("journal lightweight = %v, want true", journal["lightweight"])
	}
	for _, key := range []string{"required", "optional"} {
		list, ok := journal[key].([]any)
		if !ok || len(list) != 0 {
			t.Errorf("journal %s = %#v, want an empty JSON list", key, journal[key])
		}
	}

	decision, ok := types["decision"].(map[string]any)
	if !ok {
		t.Fatalf("no decision type in the response: %s", body)
	}
	if decision["family"] != "decision" {
		t.Errorf("decision family = %v, want \"decision\"", decision["family"])
	}
	if decision["lightweight"] != false {
		t.Errorf("decision lightweight = %v, want false", decision["lightweight"])
	}
}

// TestDoctypes_PublicModeUnauthenticated: the doc-type vocabulary is product-neutral — it
// describes the schema the binary ships, not this desk's data — so it stays open on a public
// bind, exactly like the model catalog. The SPA needs it to render a status picker before a
// login token exists; gating it would make the editor unusable on the login screen for no
// confidentiality gain.
func TestDoctypes_PublicModeUnauthenticated(t *testing.T) {
	srv, _ := newTestServer(t, true)
	status, raw, body := getDoctypes(t, srv.URL)
	if status != http.StatusOK {
		t.Fatalf("public-mode GET %s status = %d, want 200; body %s", PathDoctypes, status, body)
	}
	if _, ok := raw["types"].(map[string]any); !ok {
		t.Fatalf("public-mode response carries no types object: %s", body)
	}
}

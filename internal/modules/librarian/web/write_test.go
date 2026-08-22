package web

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/apis"
	pbcore "github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"

	"github.com/hsb3/deskkit/internal/core/config"
	"github.com/hsb3/deskkit/internal/modules/librarian/desklib"
	"github.com/hsb3/deskkit/internal/modules/librarian/tools"
)

// wdSeedDoc is the same minimal frontmatter document the tools package's write_doc tests
// use: a real desk doc whose `status` field is what a save edits.
const wdSeedDoc = `---
type: guide
status: draft
---

# A doc

body
`

const wdRel = "notes/doc.md"

// newWriteTestServer mirrors web_test.go's newTestServerMode for the write route: a real
// PocketBase test app (the blank-imported librarian migrations give it the files/revisions
// collections), a real router with ONLY the write route mounted, and a live HTTP test
// server. The returned config points at a throwaway desk root, so the handler's WriteDoc
// call runs against real files and no real desk.
func newWriteTestServer(t *testing.T, public bool) (*httptest.Server, *tests.TestApp, *config.Config) {
	t.Helper()
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("tests.NewTestApp: %v", err)
	}
	t.Cleanup(app.Cleanup)

	deskRoot := t.TempDir()
	cfg := &config.Config{
		DeskRoot:     deskRoot,
		DeskName:     "test-desk",
		DecisionsDir: "_structure/decisions",
		TasksDir:     "tasks",
		AnalysesDir:  "analyses",
		JournalDir:   "journal",
		SecretsDir:   "_meta/secrets",
		IgnoreConfig: filepath.Join(deskRoot, ".deskkitignore"),
		HandoffPath:  "_meta/HANDOFF.md",
	}
	if err := os.WriteFile(cfg.IgnoreConfig, []byte("# empty — nothing ignored\n"), 0o644); err != nil {
		t.Fatalf("write ignore file: %v", err)
	}

	r, err := apis.NewRouter(app)
	if err != nil {
		t.Fatalf("apis.NewRouter: %v", err)
	}
	RegisterDocWrite(r, app, cfg, public)

	mux, err := r.BuildMux()
	if err != nil {
		t.Fatalf("BuildMux: %v", err)
	}
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, app, cfg
}

// seedDoc plants content at a desk-relative path and returns its absolute path plus the
// checksum a caller would have loaded it with (files.checksum / write_doc's base_checksum).
func seedDoc(t *testing.T, cfg *config.Config, rel, content string) (string, string) {
	t.Helper()
	abs := filepath.Join(cfg.DeskRoot, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatalf("mkdir for %s: %v", rel, err)
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
	return abs, desklib.Checksum([]byte(content))
}

// decodeWriteResult reads the route's JSON body as the tool's own result type — the route
// returns tools.WriteDocResult verbatim, so the surface and the tool cannot drift.
func decodeWriteResult(t *testing.T, resp *http.Response) tools.WriteDocResult {
	t.Helper()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	var res tools.WriteDocResult
	if err := json.Unmarshal(b, &res); err != nil {
		t.Fatalf("body is not a WriteDocResult (%v): %s", err, b)
	}
	return res
}

// writeBody builds the save request the SPA POSTs: a path, the checksum the copy was loaded
// with, and one frontmatter field to set.
func writeBody(rel, base, field, value string) string {
	return `{"path":"` + rel + `","base_checksum":"` + base + `","set":{"` + field + `":"` + value + `"}}`
}

// newForeignAuthCollection creates an auth collection that is NOT in authCollections, so a
// token minted from it is a valid PocketBase token for the wrong audience.
func newForeignAuthCollection(t *testing.T, app *tests.TestApp, name string) {
	t.Helper()
	col := pbcore.NewAuthCollection(name)
	if err := app.Save(col); err != nil {
		t.Fatalf("create auth collection %s: %v", name, err)
	}
}

// TestDocWritePublicMode_UnauthenticatedIs401: on a non-loopback bind the write route —
// the one door from browser to disk — refuses an unauthenticated request, exactly like the
// chat session routes. Exposure alone must never open a write path.
func TestDocWritePublicMode_UnauthenticatedIs401(t *testing.T) {
	srv, _, cfg := newWriteTestServer(t, true)
	abs, base := seedDoc(t, cfg, wdRel, wdSeedDoc)

	resp := requestWithHeaders(t, http.MethodPost, srv.URL+PathDocWrite,
		writeBody(wdRel, base, "status", "active"), nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("POST %s (public, no token): status = %d, want 401", PathDocWrite, resp.StatusCode)
	}
	resp.Body.Close()

	// The rejection happened before the tool ran: the file is untouched.
	got, _ := os.ReadFile(abs)
	if string(got) != wdSeedDoc {
		t.Errorf("unauthenticated request reached the disk: %q", got)
	}
}

// TestDocWritePublicMode_WrongAuthCollectionIs403: a syntactically valid token from an auth
// collection outside authCollections is authenticated but not authorized — 403, not 401 and
// certainly not a write.
func TestDocWritePublicMode_WrongAuthCollectionIs403(t *testing.T) {
	srv, app, cfg := newWriteTestServer(t, true)
	abs, base := seedDoc(t, cfg, wdRel, wdSeedDoc)
	newForeignAuthCollection(t, app, "outsiders")
	tok := authToken(t, app, "outsiders", "outsider@desk.test")

	resp := requestWithHeaders(t, http.MethodPost, srv.URL+PathDocWrite,
		writeBody(wdRel, base, "status", "active"),
		map[string]string{"Authorization": tok, "Origin": srv.URL})
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("POST %s (public, foreign-collection token): status = %d, want 403", PathDocWrite, resp.StatusCode)
	}
	resp.Body.Close()

	got, _ := os.ReadFile(abs)
	if string(got) != wdSeedDoc {
		t.Errorf("a foreign-collection token reached the disk: %q", got)
	}
}

// TestDocWritePublicMode_CrossOriginRejected: with a VALID token, a cross-origin browser POST
// is still refused 403 by the strict same-origin guard — the write route carries the same
// CSRF posture as the session routes, so an authenticated operator's browser cannot be driven
// into writing by another site.
func TestDocWritePublicMode_CrossOriginRejected(t *testing.T) {
	srv, app, cfg := newWriteTestServer(t, true)
	abs, base := seedDoc(t, cfg, wdRel, wdSeedDoc)
	tok := authToken(t, app, pbcore.CollectionNameSuperusers, "op@desk.test")
	body := writeBody(wdRel, base, "status", "active")

	for _, origin := range []string{"https://evil.example", strings.Replace(srv.URL, "127.0.0.1", "localhost", 1)} {
		resp := requestWithHeaders(t, http.MethodPost, srv.URL+PathDocWrite, body,
			map[string]string{"Authorization": tok, "Origin": origin})
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("public-mode write from origin %q: status = %d, want 403", origin, resp.StatusCode)
		}
		resp.Body.Close()
	}

	got, _ := os.ReadFile(abs)
	if string(got) != wdSeedDoc {
		t.Errorf("a cross-origin request reached the disk: %q", got)
	}

	// Control: the SAME request from the page's own origin is accepted, so the 403s above
	// come from the origin guard and not from a route that rejects everything.
	resp := requestWithHeaders(t, http.MethodPost, srv.URL+PathDocWrite, body,
		map[string]string{"Authorization": tok, "Origin": srv.URL})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("public-mode same-origin write: status = %d, want 200", resp.StatusCode)
	}
	resp.Body.Close()
}

// TestDocWriteLoopback_WrittenThenConflict: the local posture end to end. A loopback POST
// with the checksum its copy was loaded with reaches WriteDoc, lands the field edit on disk,
// and answers 200 + outcome "written"; a second POST carrying the now-stale checksum answers
// 409 + outcome "conflict" with the disk's current state, and never clobbers the file.
func TestDocWriteLoopback_WrittenThenConflict(t *testing.T) {
	srv, _, cfg := newWriteTestServer(t, false)
	abs, base := seedDoc(t, cfg, wdRel, wdSeedDoc)

	resp := requestWithHeaders(t, http.MethodPost, srv.URL+PathDocWrite,
		writeBody(wdRel, base, "status", "active"), nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("loopback write: status = %d, want 200", resp.StatusCode)
	}
	res := decodeWriteResult(t, resp)
	resp.Body.Close()
	if res.Outcome != "written" || res.RevisionID == "" {
		t.Fatalf("want outcome written + a revision, got %+v", res)
	}

	want := strings.Replace(wdSeedDoc, "status: draft", "status: active", 1)
	got, _ := os.ReadFile(abs)
	if string(got) != want {
		t.Fatalf("disk after write:\n got %q\nwant %q", got, want)
	}
	if res.Checksum != desklib.Checksum([]byte(want)) {
		t.Errorf("result checksum %q does not match the bytes on disk", res.Checksum)
	}

	// The SAME base checksum is now stale — this is the outside-edit case a browser tab
	// sitting on an old copy hits.
	resp2 := requestWithHeaders(t, http.MethodPost, srv.URL+PathDocWrite,
		writeBody(wdRel, base, "status", "archived"), nil)
	if resp2.StatusCode != http.StatusConflict {
		t.Fatalf("stale-checksum write: status = %d, want 409", resp2.StatusCode)
	}
	conflict := decodeWriteResult(t, resp2)
	resp2.Body.Close()
	if conflict.Outcome != "conflict" {
		t.Fatalf("want outcome conflict, got %+v", conflict)
	}
	if conflict.CurrentChecksum != desklib.Checksum([]byte(want)) || conflict.CurrentContent != want {
		t.Errorf("409 payload does not carry the disk's current state: %+v", conflict)
	}
	after, _ := os.ReadFile(abs)
	if string(after) != want {
		t.Errorf("the refused write clobbered the file: %q", after)
	}
}

// TestDocWriteLoopback_BadRequestNotAWrite: a malformed body and a tool-level rejection both
// come back 400 without touching the disk (the route reports the tool's error rather than
// swallowing it).
func TestDocWriteLoopback_BadRequestNotAWrite(t *testing.T) {
	srv, _, cfg := newWriteTestServer(t, false)
	abs, base := seedDoc(t, cfg, wdRel, wdSeedDoc)

	bodies := []string{
		`{`,                        // malformed JSON
		`{"path":"` + wdRel + `"}`, // no base_checksum, no content/set
		writeBody("../escape.md", base, "status", "active"), // path escapes the desk root
	}
	for _, body := range bodies {
		resp := requestWithHeaders(t, http.MethodPost, srv.URL+PathDocWrite, body, nil)
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("body %q: status = %d, want 400", body, resp.StatusCode)
		}
		resp.Body.Close()
	}

	got, _ := os.ReadFile(abs)
	if string(got) != wdSeedDoc {
		t.Errorf("a rejected request reached the disk: %q", got)
	}
}

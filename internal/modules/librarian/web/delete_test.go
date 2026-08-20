package web

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/apis"
	pbcore "github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"

	"github.com/hsb3/deskkit/internal/core/config"
	"github.com/hsb3/deskkit/internal/modules/librarian/desklib"
	"github.com/hsb3/deskkit/internal/modules/librarian/tools"
)

// newDeleteTestServer mirrors newWriteTestServer for the delete route: a real PocketBase test
// app, a real router with ONLY the delete route mounted, and a throwaway desk root — so the
// handler's DeleteDoc call runs against real files and no real desk.
func newDeleteTestServer(t *testing.T, public bool) (*httptest.Server, *tests.TestApp, *config.Config) {
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
		IgnoreConfig: deskRoot + "/.librarian-ignore",
		HandoffPath:  "_meta/HANDOFF.md",
	}
	if err := os.WriteFile(cfg.IgnoreConfig, []byte("# empty — nothing ignored\n"), 0o644); err != nil {
		t.Fatalf("write ignore file: %v", err)
	}

	r, err := apis.NewRouter(app)
	if err != nil {
		t.Fatalf("apis.NewRouter: %v", err)
	}
	RegisterDocDelete(r, app, cfg, public)

	mux, err := r.BuildMux()
	if err != nil {
		t.Fatalf("BuildMux: %v", err)
	}
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, app, cfg
}

// deleteBody is the request the SPA POSTs when the two-step confirm commits.
func deleteBody(rel, base string) string {
	return `{"path":"` + rel + `","base_checksum":"` + base + `"}`
}

// decodeDeleteResult reads the route's JSON body as the TOOL's own result type — the route
// returns tools.DeleteDocResult verbatim, so the surface and the tool cannot drift.
func decodeDeleteResult(t *testing.T, resp *http.Response) tools.DeleteDocResult {
	t.Helper()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	var res tools.DeleteDocResult
	if err := json.Unmarshal(b, &res); err != nil {
		t.Fatalf("body is not a DeleteDocResult (%v): %s", err, b)
	}
	return res
}

// TestDocDeletePublicMode_UnauthenticatedIs401: exposure alone must never open a delete path.
// Same posture as the write route, applied to the more destructive verb.
func TestDocDeletePublicMode_UnauthenticatedIs401(t *testing.T) {
	srv, _, cfg := newDeleteTestServer(t, true)
	abs, base := seedDoc(t, cfg, wdRel, wdSeedDoc)

	resp := requestWithHeaders(t, http.MethodPost, srv.URL+PathDocDelete, deleteBody(wdRel, base), nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("POST %s (public, no token): status = %d, want 401", PathDocDelete, resp.StatusCode)
	}
	resp.Body.Close()

	if _, err := os.Stat(abs); err != nil {
		t.Errorf("an unauthenticated request deleted the file: %v", err)
	}
}

// TestDocDeletePublicMode_WrongAuthCollectionIs403: a valid token minted from an auth
// collection outside authCollections is authenticated but not authorized.
func TestDocDeletePublicMode_WrongAuthCollectionIs403(t *testing.T) {
	srv, app, cfg := newDeleteTestServer(t, true)
	abs, base := seedDoc(t, cfg, wdRel, wdSeedDoc)
	newForeignAuthCollection(t, app, "outsiders")
	tok := authToken(t, app, "outsiders", "outsider@desk.test")

	resp := requestWithHeaders(t, http.MethodPost, srv.URL+PathDocDelete, deleteBody(wdRel, base),
		map[string]string{"Authorization": tok, "Origin": srv.URL})
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("POST %s (public, foreign-collection token): status = %d, want 403", PathDocDelete, resp.StatusCode)
	}
	resp.Body.Close()

	if _, err := os.Stat(abs); err != nil {
		t.Errorf("a foreign-collection token deleted the file: %v", err)
	}
}

// TestDocDeletePublicMode_CrossOriginRejected: with a VALID token, a cross-origin browser POST
// is still refused — an authenticated operator's browser must not be drivable into deleting
// their documents by another site.
func TestDocDeletePublicMode_CrossOriginRejected(t *testing.T) {
	srv, app, cfg := newDeleteTestServer(t, true)
	abs, base := seedDoc(t, cfg, wdRel, wdSeedDoc)
	tok := authToken(t, app, pbcore.CollectionNameSuperusers, "op@desk.test")
	body := deleteBody(wdRel, base)

	for _, origin := range []string{"https://evil.example", strings.Replace(srv.URL, "127.0.0.1", "localhost", 1)} {
		resp := requestWithHeaders(t, http.MethodPost, srv.URL+PathDocDelete, body,
			map[string]string{"Authorization": tok, "Origin": origin})
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("public-mode delete from origin %q: status = %d, want 403", origin, resp.StatusCode)
		}
		resp.Body.Close()
	}
	if _, err := os.Stat(abs); err != nil {
		t.Errorf("a cross-origin request deleted the file: %v", err)
	}

	// Control: the SAME request from the page's own origin succeeds, so the 403s above come
	// from the origin guard and not from a route that refuses everything.
	resp := requestWithHeaders(t, http.MethodPost, srv.URL+PathDocDelete, body,
		map[string]string{"Authorization": tok, "Origin": srv.URL})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("public-mode same-origin delete: status = %d, want 200", resp.StatusCode)
	}
	resp.Body.Close()
	if _, err := os.Stat(abs); !os.IsNotExist(err) {
		t.Errorf("the accepted delete left the file in place (stat err = %v)", err)
	}
}

// TestDocDeleteLoopback_DeletedThenConflict: the local posture end to end. A loopback POST with
// the checksum its copy was loaded with removes the file and answers 200 + outcome "deleted"
// with a revision id; a second POST for a doc whose disk state has moved on answers 409 +
// the same conflict body the write route returns, and never removes the file.
func TestDocDeleteLoopback_DeletedThenConflict(t *testing.T) {
	srv, _, cfg := newDeleteTestServer(t, false)
	abs, base := seedDoc(t, cfg, wdRel, wdSeedDoc)

	resp := requestWithHeaders(t, http.MethodPost, srv.URL+PathDocDelete, deleteBody(wdRel, base), nil)
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("loopback delete: status = %d, want 200; body %s", resp.StatusCode, body)
	}
	res := decodeDeleteResult(t, resp)
	resp.Body.Close()
	if res.Outcome != "deleted" || res.RevisionID == "" {
		t.Fatalf("want outcome deleted + a revision, got %+v", res)
	}
	if res.Path != wdRel {
		t.Errorf("result path = %q, want %q", res.Path, wdRel)
	}
	if _, err := os.Stat(abs); !os.IsNotExist(err) {
		t.Fatalf("file still on disk after a 200 delete (stat err = %v)", err)
	}

	// A second doc whose disk state moved on after the browser loaded it: 409, file intact.
	const otherRel = "notes/other.md"
	otherAbs, staleBase := seedDoc(t, cfg, otherRel, wdSeedDoc)
	moved := strings.Replace(wdSeedDoc, "status: draft", "status: approved", 1)
	if err := os.WriteFile(otherAbs, []byte(moved), 0o644); err != nil {
		t.Fatal(err)
	}

	resp2 := requestWithHeaders(t, http.MethodPost, srv.URL+PathDocDelete, deleteBody(otherRel, staleBase), nil)
	if resp2.StatusCode != http.StatusConflict {
		t.Fatalf("stale-checksum delete: status = %d, want 409", resp2.StatusCode)
	}
	conflict := decodeDeleteResult(t, resp2)
	resp2.Body.Close()
	if conflict.Outcome != "conflict" {
		t.Fatalf("want outcome conflict, got %+v", conflict)
	}
	if conflict.CurrentChecksum != desklib.Checksum([]byte(moved)) || conflict.CurrentContent != moved {
		t.Errorf("409 payload does not carry the disk's current state: %+v", conflict)
	}
	if got, err := os.ReadFile(otherAbs); err != nil || string(got) != moved {
		t.Errorf("the refused delete touched the file: %q (err %v)", got, err)
	}
}

// TestDocDeleteLoopback_BadRequestNotADelete: a malformed body and a tool-level rejection both
// come back 400 with the tool's own error, and nothing leaves the disk.
func TestDocDeleteLoopback_BadRequestNotADelete(t *testing.T) {
	srv, _, cfg := newDeleteTestServer(t, false)
	abs, base := seedDoc(t, cfg, wdRel, wdSeedDoc)

	bodies := []string{
		`{`,                              // malformed JSON
		`{"path":"` + wdRel + `"}`,       // no base_checksum
		deleteBody("../escape.md", base), // path escapes the desk root
		deleteBody("notes/absent.md", base),
	}
	for _, body := range bodies {
		resp := requestWithHeaders(t, http.MethodPost, srv.URL+PathDocDelete, body, nil)
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("body %q: status = %d, want 400", body, resp.StatusCode)
		}
		resp.Body.Close()
	}

	if _, err := os.Stat(abs); err != nil {
		t.Errorf("a rejected request deleted the file: %v", err)
	}
}

package spa

import (
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

// newTestServer wires the SPA routes onto a real PocketBase router and returns a live HTTP
// test server plus the app handle (for seeding superuser records) — the same scripted-probe
// shape the web package's tests use.
func newTestServer(t *testing.T, public bool) (*httptest.Server, *tests.TestApp) {
	t.Helper()
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("tests.NewTestApp: %v", err)
	}
	t.Cleanup(app.Cleanup)

	r, err := apis.NewRouter(app)
	if err != nil {
		t.Fatalf("apis.NewRouter: %v", err)
	}
	Register(r, public)

	mux, err := r.BuildMux()
	if err != nil {
		t.Fatalf("BuildMux: %v", err)
	}
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, app
}

// resetSuperusers pins the store to exactly the given accounts: it creates them, then deletes
// every other superuser (the fixture ones tests.NewTestApp seeds). Creating first matters —
// the dependency refuses to delete the last remaining superuser, which also means a real
// store can never reach zero superusers, so that state is not tested here.
func resetSuperusers(t *testing.T, app core.App, emails ...string) map[string]*core.Record {
	t.Helper()
	keep := make(map[string]*core.Record, len(emails))
	for _, email := range emails {
		keep[email] = addSuperuser(t, app, email)
	}
	recs, err := app.FindAllRecords(core.CollectionNameSuperusers)
	if err != nil {
		t.Fatalf("list superusers: %v", err)
	}
	for _, rec := range recs {
		if _, ok := keep[rec.GetString("email")]; ok {
			continue
		}
		if err := app.Delete(rec); err != nil {
			t.Fatalf("delete superuser %s: %v", rec.GetString("email"), err)
		}
	}
	return keep
}

func addSuperuser(t *testing.T, app core.App, email string) *core.Record {
	t.Helper()
	col, err := app.FindCollectionByNameOrId(core.CollectionNameSuperusers)
	if err != nil {
		t.Fatalf("find superusers collection: %v", err)
	}
	rec := core.NewRecord(col)
	rec.SetEmail(email)
	rec.SetPassword("0123456789")
	if err := app.Save(rec); err != nil {
		t.Fatalf("save superuser %s: %v", email, err)
	}
	return rec
}

func getWithOrigin(t *testing.T, url, origin string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	return resp
}

func bootstrapToken(t *testing.T, resp *http.Response) string {
	t.Helper()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("bootstrap status = %d, body %s", resp.StatusCode, body)
	}
	var payload struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode bootstrap body: %v", err)
	}
	if payload.Token == "" {
		t.Fatal("bootstrap returned an empty token")
	}
	return payload.Token
}

// TestShellServes pins that the root wildcard always answers 200 HTML in both build flavors:
// the real SPA shell when a frontend build is embedded, the placeholder page when the binary
// was compiled from a bare checkout (dist holds only .gitkeep).
func TestShellServes(t *testing.T) {
	srv, _ := newTestServer(t, false)
	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET / status = %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Fatalf("GET / content-type = %q, want text/html", ct)
	}
	body, _ := io.ReadAll(resp.Body)
	sub, _ := fs.Sub(distFS, "dist")
	if _, err := fs.Stat(sub, "index.html"); err == nil {
		if !strings.Contains(string(body), `<div id="app">`) {
			t.Fatalf("built binary: / does not serve the SPA shell; body starts %q", string(body[:min(len(body), 120)]))
		}
		// Index fallback: a client-side route (and the pre-SPA chat URL) serves the shell too.
		for _, path := range []string{"/desk/chat", "/no/such/route"} {
			r2, err := http.Get(srv.URL + path)
			if err != nil {
				t.Fatalf("GET %s: %v", path, err)
			}
			b2, _ := io.ReadAll(r2.Body)
			r2.Body.Close()
			if r2.StatusCode != http.StatusOK || !strings.Contains(string(b2), `<div id="app">`) {
				t.Fatalf("GET %s = %d, want the SPA shell via index fallback", path, r2.StatusCode)
			}
		}
	} else {
		if !strings.Contains(string(body), "not built") {
			t.Fatalf("unbuilt binary: / does not serve the placeholder; body starts %q", string(body[:min(len(body), 120)]))
		}
	}
}

// TestBootstrap_PrefersAdministrableSuperuser pins the mint order: with both an administrable
// account and the installer placeholder present, the token belongs to the real account.
func TestBootstrap_PrefersAdministrableSuperuser(t *testing.T) {
	srv, app := newTestServer(t, false)
	admin := resetSuperusers(t, app, core.DefaultInstallerEmail, "op@example.com")["op@example.com"]

	token := bootstrapToken(t, getWithOrigin(t, srv.URL+PathBootstrap, ""))
	rec, err := app.FindAuthRecordByToken(token, core.TokenTypeAuth)
	if err != nil {
		t.Fatalf("minted token does not resolve: %v", err)
	}
	if rec.Id != admin.Id {
		t.Fatalf("token minted for %q, want the administrable account %q", rec.GetString("email"), admin.GetString("email"))
	}
}

// TestBootstrap_PlaceholderFallback: a store an ordinary local serve just initialized holds
// only the installer placeholder — the loopback operator still gets a token.
func TestBootstrap_PlaceholderFallback(t *testing.T) {
	srv, app := newTestServer(t, false)
	placeholder := resetSuperusers(t, app, core.DefaultInstallerEmail)[core.DefaultInstallerEmail]

	token := bootstrapToken(t, getWithOrigin(t, srv.URL+PathBootstrap, ""))
	rec, err := app.FindAuthRecordByToken(token, core.TokenTypeAuth)
	if err != nil {
		t.Fatalf("minted token does not resolve: %v", err)
	}
	if rec.Id != placeholder.Id {
		t.Fatalf("token minted for %q, want the placeholder fallback", rec.GetString("email"))
	}
}

// TestBootstrap_OriginGuard: a browser cross-site fetch (non-loopback Origin) must not be able
// to read a token; loopback origins and origin-less clients pass.
func TestBootstrap_OriginGuard(t *testing.T) {
	srv, app := newTestServer(t, false)
	resetSuperusers(t, app, "op@example.com")

	for _, tc := range []struct {
		origin string
		want   int
	}{
		{"https://example.com", http.StatusForbidden},
		{"chrome-extension://abc", http.StatusForbidden},
		{"http://localhost:5173", http.StatusOK},
		{"http://127.0.0.1:8090", http.StatusOK},
		{"", http.StatusOK},
	} {
		resp := getWithOrigin(t, srv.URL+PathBootstrap, tc.origin)
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if resp.StatusCode != tc.want {
			t.Fatalf("bootstrap with Origin %q = %d, want %d", tc.origin, resp.StatusCode, tc.want)
		}
	}
}

// TestBootstrap_AbsentInPublicMode pins fail-closed-by-absence: on a public bind the route is
// never registered, so the path falls through to the static wildcard and serves HTML — there
// is no token endpoint to guard, bypass, or misconfigure.
func TestBootstrap_AbsentInPublicMode(t *testing.T) {
	srv, app := newTestServer(t, true)
	resetSuperusers(t, app, "op@example.com")

	resp := getWithOrigin(t, srv.URL+PathBootstrap, "")
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if ct := resp.Header.Get("Content-Type"); strings.Contains(ct, "application/json") && strings.Contains(string(body), "token") {
		t.Fatalf("public mode served a token from %s: %s", PathBootstrap, body)
	}
}

// TestAdminRedirect pins that /admin — the path operators type — reaches the dependency's
// admin console instead of being swallowed by the shell wildcard. Registered in both bind
// modes: the console is the public-mode operator's only way in, so a public bind needs it
// most. Deeper /admin/* paths stay with the wildcard; only the bare path is claimed.
func TestAdminRedirect(t *testing.T) {
	noFollow := &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	for _, public := range []bool{false, true} {
		srv, _ := newTestServer(t, public)

		resp, err := noFollow.Get(srv.URL + PathAdmin)
		if err != nil {
			t.Fatalf("public=%v: GET %s: %v", public, PathAdmin, err)
		}
		resp.Body.Close()
		if resp.StatusCode < 300 || resp.StatusCode > 399 {
			t.Fatalf("public=%v: GET %s status = %d, want a 3xx redirect", public, PathAdmin, resp.StatusCode)
		}
		if loc := resp.Header.Get("Location"); loc != adminConsole {
			t.Fatalf("public=%v: GET %s Location = %q, want %q", public, PathAdmin, loc, adminConsole)
		}

		// The wildcard still owns everything else: a client-side route must not redirect.
		r2, err := noFollow.Get(srv.URL + "/documents")
		if err != nil {
			t.Fatalf("public=%v: GET /documents: %v", public, err)
		}
		r2.Body.Close()
		if r2.StatusCode != http.StatusOK {
			t.Fatalf("public=%v: GET /documents status = %d, want the shell's 200", public, r2.StatusCode)
		}
	}
}

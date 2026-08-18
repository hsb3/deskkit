package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/apis"
	pbcmd "github.com/pocketbase/pocketbase/cmd"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

// TestIsLoopbackAddr is the whole basis of "public mode": get this classification wrong in the
// permissive direction and an off-box listener silently keeps the unauthenticated local posture.
// So the table asserts both halves, and every ambiguous form (no host, no port, a bare hostname)
// resolves to PUBLIC — fail closed.
func TestIsLoopbackAddr(t *testing.T) {
	cases := []struct {
		addr     string
		loopback bool
		why      string
	}{
		// Loopback — today's local UX, which must stay byte-for-byte unchanged.
		{"", true, "empty means the dependency's own --http default, which is 127.0.0.1:8090"},
		{"   ", true, "whitespace-only is still absent"},
		{"127.0.0.1:8090", true, "the documented local default"},
		{"127.0.0.1", true, "host with no port"},
		{"127.0.0.53:8090", true, "the whole 127.0.0.0/8 range is loopback"},
		{"localhost:8090", true, "the other spelling an operator may browse to"},
		{"localhost", true, "hostname with no port"},
		{"[::1]:8090", true, "IPv6 loopback, bracketed with a port"},
		{"::1", true, "IPv6 loopback, bare"},
		{"[::1]", true, "IPv6 loopback, bracketed without a port"},

		// Public — anything reachable from another machine.
		{"0.0.0.0:8090", false, "all IPv4 interfaces"},
		{"0.0.0.0", false, "all IPv4 interfaces, no port"},
		{":8090", false, "an empty host binds every interface — the container form"},
		{":80", false, "same, on a privileged port"},
		{"[::]:8090", false, "all IPv6 interfaces"},
		{"::", false, "all IPv6 interfaces, bare"},
		{"192.168.1.10:8090", false, "a LAN address is still off-box"},
		{"10.0.0.4:8090", false, "a private routable address is still off-box"},
		{"203.0.113.7:8090", false, "a public address"},
		{"example.invalid:8090", false, "an unresolved hostname is treated as exposure, never as safety"},
		{"localhost.evil.invalid:8090", false, "a hostname that merely starts with localhost is not loopback"},
		{"not an address", false, "a malformed value falls through to the public default"},
	}

	for _, c := range cases {
		if got := isLoopbackAddr(c.addr); got != c.loopback {
			t.Errorf("isLoopbackAddr(%q) = %v, want %v — %s", c.addr, got, c.loopback, c.why)
		}
	}
}

// corsProbeServer wires a router the way the dependency's Serve() does — apis.NewRouter plus the
// SAME wildcard CORS bind Serve performs before it triggers OnServe — then applies
// hardenPublicCORS and builds the mux, exactly as the OnServe hook does at runtime.
//
// Reproducing that one bind is deliberate and load-bearing: apis.NewRouter ALONE carries no CORS
// middleware, which is precisely why an earlier header assertion against a NewRouter-only test
// server passed while the real binary still answered `Access-Control-Allow-Origin: *`. A header
// test is only meaningful against a router that actually has the header-emitting middleware on it.
func corsProbeServer(t *testing.T, public bool, origins []string) *httptest.Server {
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
	// Verbatim from the dependency's Serve(): an empty AllowedOrigins becomes ["*"], and the bind
	// carries this method set. If a dependency bump changes either, this probe stops matching
	// production — the live curl transcript in the handoff is the end-to-end backstop.
	bound := origins
	if len(bound) == 0 {
		bound = []string{"*"}
	}
	r.Bind(apis.CORS(apis.CORSConfig{
		AllowOrigins: bound,
		AllowMethods: []string{
			http.MethodGet, http.MethodHead, http.MethodPut,
			http.MethodPatch, http.MethodPost, http.MethodDelete,
		},
	}))
	r.POST("/probe", func(e *core.RequestEvent) error { return e.NoContent(http.StatusNoContent) })

	hardenPublicCORS(r, public, origins)

	mux, err := r.BuildMux()
	if err != nil {
		t.Fatalf("BuildMux: %v", err)
	}
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// headerFor issues one request with a foreign Origin and returns the named response header.
func headerFor(t *testing.T, srv *httptest.Server, method, path, header string, preflight bool) (int, string) {
	t.Helper()
	return headerForOrigin(t, srv, method, path, header, "http://evil.example", preflight)
}

// headerForOrigin issues one request with the given Origin and returns the named response header.
func headerForOrigin(t *testing.T, srv *httptest.Server, method, path, header, origin string, preflight bool) (int, string) {
	t.Helper()
	req, err := http.NewRequest(method, srv.URL+path, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Origin", origin)
	if preflight {
		req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	return resp.StatusCode, resp.Header.Get(header)
}

// TestHardenPublicCORS_PublicEmitsNoAllowOrigin: on a public bind NO response may carry an
// Access-Control-Allow-Origin header at all — not a wildcard, not an allowlist. The page is served
// same-origin so it needs none, and with none emitted a browser refuses a cross-origin read.
// Covers a normal request AND the OPTIONS preflight, which is the shape the CORS middleware
// answers specially.
func TestHardenPublicCORS_PublicEmitsNoAllowOrigin(t *testing.T) {
	srv := corsProbeServer(t, true, nil)

	for _, c := range []struct {
		name, method, path string
		preflight          bool
	}{
		{"health", http.MethodGet, "/api/health", false},
		{"collection", http.MethodGet, "/api/collections/files/records", false},
		{"custom route", http.MethodPost, "/probe", false},
		{"OPTIONS preflight", http.MethodOptions, "/probe", true},
	} {
		status, got := headerFor(t, srv, c.method, c.path, "Access-Control-Allow-Origin", c.preflight)
		if got != "" {
			t.Errorf("public mode, %s (%s %s -> %d): Access-Control-Allow-Origin = %q, want no header at all",
				c.name, c.method, c.path, status, got)
		}
		if _, methods := headerFor(t, srv, c.method, c.path, "Access-Control-Allow-Methods", c.preflight); methods != "" {
			t.Errorf("public mode, %s: Access-Control-Allow-Methods = %q, want no header at all", c.name, methods)
		}
	}
}

// TestHardenPublicCORS_LoopbackWildcardUnchanged pins the deliberate NON-change: on a loopback
// bind the dependency's wildcard stays exactly as it has always been. Local-dev tooling may rely
// on it, and "default local `deskkit serve` behavior unchanged" covers it. If this ever goes red,
// the local UX changed — that is a decision, not a cleanup.
func TestHardenPublicCORS_LoopbackWildcardUnchanged(t *testing.T) {
	srv := corsProbeServer(t, false, nil)

	if _, got := headerFor(t, srv, http.MethodGet, "/api/health", "Access-Control-Allow-Origin", false); got != "*" {
		t.Errorf("loopback mode: Access-Control-Allow-Origin = %q, want the dependency's unchanged %q", got, "*")
	}
	if _, got := headerFor(t, srv, http.MethodOptions, "/probe", "Access-Control-Allow-Origin", true); got != "*" {
		t.Errorf("loopback mode preflight: Access-Control-Allow-Origin = %q, want the dependency's unchanged %q", got, "*")
	}
}

// TestEffectiveServeAddrs pins the dependency's address defaulting that this binary mirrors. It is
// copied logic, so it is the first thing to re-check on a dependency bump: if the dependency ever
// changes a default, this table is where the divergence shows up rather than in a silent
// misclassification of a live listener.
func TestEffectiveServeAddrs(t *testing.T) {
	cases := []struct {
		name          string
		http, https   string
		hasDomainArgs bool
		want          []string
	}{
		{"bare serve", "", "", false, []string{"127.0.0.1:8090"}},
		{"explicit http only", "0.0.0.0:8090", "", false, []string{"0.0.0.0:8090"}},
		{"http + https", "0.0.0.0:8090", "127.0.0.1:8443", false, []string{"0.0.0.0:8090", "127.0.0.1:8443"}},
		{"https only", "", "127.0.0.1:8443", false, []string{"127.0.0.1:8090", "127.0.0.1:8443"}},
		{"domain args default both to all-interfaces", "", "", true, []string{"0.0.0.0:80", "0.0.0.0:443"}},
		{"domain args with explicit loopback https", "", "127.0.0.1:8443", true, []string{"0.0.0.0:80", "127.0.0.1:8443"}},
	}
	for _, c := range cases {
		got := effectiveServeAddrs(c.http, c.https, c.hasDomainArgs)
		if len(got) != len(c.want) {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("%s: got %v, want %v", c.name, got, c.want)
				break
			}
		}
	}
}

// TestIsPublicBind_ClassifiesEveryListener is the S-5 regression: the address the dependency
// reports on the serve event is the HTTPS one when --https is given, which HIDES a separate --http
// listener. `--https 127.0.0.1:8443 --http 0.0.0.0:8090` genuinely binds 0.0.0.0 while presenting a
// loopback address, so classifying the reported address alone calls that setup "local".
func TestIsPublicBind_ClassifiesEveryListener(t *testing.T) {
	// The exact reported-vs-real split. The reported address alone says loopback...
	reported := "127.0.0.1:8443"
	if !isLoopbackAddr(reported) {
		t.Fatal("precondition: the reported https address is loopback")
	}
	// ...but the full set must classify as public.
	addrs := effectiveServeAddrs("0.0.0.0:8090", reported, false)
	if !isPublicBind(append([]string{reported}, addrs...)...) {
		t.Error("a non-loopback --http listener must make the process public even when the " +
			"reported (https) address is loopback")
	}

	// The inverse split: loopback http, exposed https.
	if !isPublicBind("127.0.0.1:8090", "0.0.0.0:8443") {
		t.Error("a non-loopback --https listener must make the process public")
	}
	// All loopback stays loopback, and empty entries are skipped rather than counted.
	if isPublicBind("127.0.0.1:8090", "", "[::1]:8443") {
		t.Error("an all-loopback address set must not classify as public")
	}
	if isPublicBind() || isPublicBind("", "  ") {
		t.Error("an empty address set must not classify as public")
	}
}

// TestIsWildcardOrigins separates the dependency's default from an operator's real allowlist —
// the distinction hardenPublicCORS needs so it never discards an explicit --origins.
func TestIsWildcardOrigins(t *testing.T) {
	wildcard := [][]string{
		nil,
		{},
		{"*"},
		{" * "},
		// The mixed case stays fail-closed HERE as defense in depth, but on a public bind it is
		// refused outright by ValidatePublicOrigins — see TestValidatePublicOrigins_RefusesMixedWildcard.
		{"https://frontend.example", "*"},
	}
	for _, o := range wildcard {
		if !isWildcardOrigins(o) {
			t.Errorf("isWildcardOrigins(%q) = false, want true", o)
		}
	}
	allowlists := [][]string{
		{"https://frontend.example"},
		{"https://a.example", "https://b.example"},
		{"https://*.example.com"}, // a glob PATTERN is a real constraint, not a wildcard
	}
	for _, o := range allowlists {
		if isWildcardOrigins(o) {
			t.Errorf("isWildcardOrigins(%q) = true, want false — an operator allowlist must survive", o)
		}
	}
}

// TestHardenPublicCORS_PreservesExplicitOrigins is the S-1 regression: the dependency binds an
// operator's --origins value into the SAME middleware id this code unbinds, so an unconditional
// unbind silently threw away the only mechanism for legitimately serving a separate frontend
// origin. With an allowlist supplied, the middleware must survive and answer that origin.
func TestHardenPublicCORS_PreservesExplicitOrigins(t *testing.T) {
	srv := corsProbeServer(t, true, []string{"https://frontend.example"})

	// The allowed origin is echoed back — the allowlist is live, not discarded.
	if _, got := headerForOrigin(t, srv, http.MethodGet, "/api/health",
		"Access-Control-Allow-Origin", "https://frontend.example", false); got != "https://frontend.example" {
		t.Errorf("explicit --origins allowlist: Access-Control-Allow-Origin = %q, want the allowed origin "+
			"(an operator's allowlist must not be silently discarded)", got)
	}
	// A foreign origin is still refused — preserving the allowlist is not the same as reopening.
	if _, got := headerForOrigin(t, srv, http.MethodGet, "/api/health",
		"Access-Control-Allow-Origin", "http://evil.example", false); got == "*" {
		t.Errorf("explicit --origins allowlist must never answer a wildcard, got %q", got)
	}
}

// errStopBeforeListen aborts the dependency's serve before it opens a socket: returning an error
// from an OnServe hook WITHOUT calling e.Next() short-circuits the trigger, so the handler that
// builds the mux and listens never runs.
var errStopBeforeListen = errors.New("stop before listen")

// TestPinnedDependencyDefaultHTTPAddr pins the dependency default that effectiveServeAddrs
// REPRODUCES rather than reads (see that function's re-audit note: the dependency applies this
// default to a local variable inside its serve command, so there is nothing to read back at
// OnServe time). It runs the dependency's own serve command with no flags and captures the address
// the dependency actually resolved, aborting before any socket is opened — so a dependency bump
// that changes the default fails HERE, loudly, instead of silently misclassifying a live listener
// as loopback.
func TestPinnedDependencyDefaultHTTPAddr(t *testing.T) {
	const pinned = "127.0.0.1:8090"

	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("tests.NewTestApp: %v", err)
	}
	t.Cleanup(app.Cleanup)

	var got string
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		got = e.Server.Addr
		return errStopBeforeListen
	})

	cmd := pbcmd.NewServeCommand(app, false)
	cmd.SetArgs(nil)
	cmd.SilenceErrors, cmd.SilenceUsage = true, true
	if err := cmd.Execute(); !errors.Is(err, errStopBeforeListen) {
		t.Fatalf("serve did not reach the OnServe hook (err = %v); this test can no longer pin anything", err)
	}
	if got != pinned {
		t.Fatalf("the dependency's default serve address is now %q, but effectiveServeAddrs still "+
			"assumes %q — update effectiveServeAddrs (and re-audit its sibling 0.0.0.0 defaults) "+
			"before this misclassifies an exposed listener as loopback", got, pinned)
	}
	// And the reproduction agrees with the dependency, which is the property that actually matters.
	if repro := effectiveServeAddrs("", "", false); len(repro) != 1 || repro[0] != got {
		t.Fatalf("effectiveServeAddrs reproduces %v but the dependency resolved %q", repro, got)
	}
}

// TestValidatePublicOrigins_RefusesMixedWildcard is the S-1 ruling: on a public bind, a bare "*"
// mixed with explicit origins is a contradictory security instruction and must REFUSE TO START.
// Resolving it either way silently discards half of an operator's explicit input at a trust
// boundary — the same defect shape as a half-configured auth env. Two states only: it works, or it
// refuses and says why.
func TestValidatePublicOrigins_RefusesMixedWildcard(t *testing.T) {
	mixed := [][]string{
		{"*", "https://frontend.example"},
		{"https://frontend.example", "*"},
		{"https://a.example", "*", "https://b.example"},
		{" * ", "https://frontend.example"}, // whitespace does not launder it
	}
	for _, o := range mixed {
		err := ValidatePublicOrigins(true, o)
		if err == nil {
			t.Errorf("ValidatePublicOrigins(public, %q) = nil, want a refusal", o)
			continue
		}
		// The message must name the contradiction AND both sides, so the operator can see what they
		// actually typed rather than a generic complaint.
		for _, want := range []string{"contradict", "*", "https://frontend.example", "non-loopback"} {
			if want == "https://frontend.example" && !containsAny(o, want) {
				continue // this case does not use that host
			}
			if !strings.Contains(err.Error(), want) {
				t.Errorf("refusal for %q must mention %q, got: %v", o, want, err)
			}
		}
	}

	// Everything unambiguous still starts: pure wildcard, pure allowlist, empty, and blank padding.
	ok := [][]string{nil, {}, {"*"}, {"https://frontend.example"},
		{"https://a.example", "https://b.example"}, {"https://*.example.com"},
		{"https://frontend.example", ""}, // a trailing comma is a typo, not an instruction
	}
	for _, o := range ok {
		if err := ValidatePublicOrigins(true, o); err != nil {
			t.Errorf("ValidatePublicOrigins(public, %q) = %v, want nil", o, err)
		}
	}

	// Loopback is exempt: local dev must not start failing over a flag combination it has always
	// tolerated.
	for _, o := range mixed {
		if err := ValidatePublicOrigins(false, o); err != nil {
			t.Errorf("loopback must tolerate %q unchanged, got: %v", o, err)
		}
	}
}

func containsAny(list []string, want string) bool {
	for _, v := range list {
		if strings.TrimSpace(v) == want {
			return true
		}
	}
	return false
}

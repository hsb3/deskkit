package web

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/apis"
	pbcore "github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"

	"github.com/hsb3/desk-standard/librarian/internal/modules/librarian/agent"

	// Blank-import registers the librarian's Go migrations so tests.NewTestApp's
	// RunAllMigrations() produces a valid store for apis.NewRouter to build against.
	_ "github.com/hsb3/desk-standard/librarian/internal/modules/librarian/collections"
)

// fakeStreamer stands in for an *agent.Session: StreamTurn replays a scripted sequence of REAL
// agent.Event values (the same JSON-tagged type StreamTurn emits) so the SSE handler is exercised
// end-to-end with NO live LLM. It records the inputs it was turned on and whether it was Closed.
type fakeStreamer struct {
	mu     sync.Mutex
	events []agent.Event
	inputs []string
	closed bool
}

func (f *fakeStreamer) StreamTurn(_ context.Context, input string) <-chan agent.Event {
	f.mu.Lock()
	f.inputs = append(f.inputs, input)
	evs := f.events
	f.mu.Unlock()
	ch := make(chan agent.Event, len(evs)+1)
	for _, e := range evs {
		ch <- e
	}
	close(ch)
	return ch
}

func (f *fakeStreamer) Close(_ context.Context) error {
	f.mu.Lock()
	f.closed = true
	f.mu.Unlock()
	return nil
}

func (f *fakeStreamer) turnCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.inputs)
}

func (f *fakeStreamer) isClosed() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.closed
}

// scriptedTurn is a representative streamed turn: a token, a tool round, then the final answer.
func scriptedTurn() []agent.Event {
	return []agent.Event{
		{Kind: agent.EventToken, Step: 1, Token: "Look"},
		{Kind: agent.EventToken, Step: 1, Token: "ing"},
		{Kind: agent.EventToolStart, Tool: "query", CallID: "c1", Args: `{"kind":"summary"}`},
		{Kind: agent.EventToolEnd, Tool: "query", CallID: "c1", Result: `{"count":3}`},
		{Kind: agent.EventFinal, Content: "Looking done: 3 items.", PromptTokens: 100, CompletionTokens: 12, TotalTokens: 112},
	}
}

// newTestServer wires the surface's routes onto a real PocketBase router and returns a live HTTP
// test server — a genuine "scripted probe against a test server" for the route/registration DoD.
// The factory is called by the handler to build a (fake) session; factoryCalls counts creations.
func newTestServer(t *testing.T, factory NewSessionFunc) (*httptest.Server, func(context.Context)) {
	t.Helper()
	srv, cleanup, _ := newTestServerMode(t, factory, false)
	return srv, cleanup
}

// newTestServerMode is newTestServer with the exposure mode as an explicit argument, plus the
// app handle the public-mode tests need to mint a real auth token. public=false reproduces the
// historical local posture exactly.
func newTestServerMode(t *testing.T, factory NewSessionFunc, public bool) (*httptest.Server, func(context.Context), *tests.TestApp) {
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
	cleanup := Register(r, factory, public)

	mux, err := r.BuildMux()
	if err != nil {
		t.Fatalf("BuildMux: %v", err)
	}
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, cleanup, app
}

// parseSSE splits an event-stream body into its decoded agent.Event frames.
func parseSSE(t *testing.T, body string) []agent.Event {
	t.Helper()
	var out []agent.Event
	for _, frame := range strings.Split(body, "\n\n") {
		frame = strings.TrimSpace(frame)
		if frame == "" {
			continue
		}
		data, ok := strings.CutPrefix(frame, "data: ")
		if !ok {
			t.Fatalf("frame missing 'data: ' prefix: %q", frame)
		}
		var ev agent.Event
		if err := json.Unmarshal([]byte(data), &ev); err != nil {
			t.Fatalf("frame is not a valid agent.Event JSON (%v): %q", err, data)
		}
		out = append(out, ev)
	}
	return out
}

func postJSON(t *testing.T, url, body string) *http.Response {
	t.Helper()
	resp, err := http.Post(url, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST %s: %v", url, err)
	}
	return resp
}

// postWithOrigin POSTs with an explicit Origin header; an empty origin omits the header entirely
// (modeling a non-browser client like curl).
func postWithOrigin(t *testing.T, url, body, origin string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		t.Fatalf("new request %s: %v", url, err)
	}
	req.Header.Set("Content-Type", "application/json")
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST %s (origin %q): %v", url, origin, err)
	}
	return resp
}

// TestStreamRoute_SSEFrames: DoD 4 — a turn's events stream to the browser as SSE frames that
// are exactly StreamTurn's JSON-tagged agent.Event values, in order, ending on the terminal
// (final) event. The fake session is the provider stand-in (no live LLM).
func TestStreamRoute_SSEFrames(t *testing.T) {
	fake := &fakeStreamer{events: scriptedTurn()}
	srv, _ := newTestServer(t, func(context.Context) (Streamer, error) { return fake, nil })

	resp := postJSON(t, srv.URL+PathStream, `{"message":"summarize the desk"}`)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Fatalf("Content-Type = %q, want text/event-stream", ct)
	}
	b, _ := io.ReadAll(resp.Body)
	got := parseSSE(t, string(b))

	wantKinds := []agent.EventKind{
		agent.EventToken, agent.EventToken, agent.EventToolStart, agent.EventToolEnd, agent.EventFinal,
	}
	if len(got) != len(wantKinds) {
		t.Fatalf("got %d frames, want %d: %+v", len(got), len(wantKinds), got)
	}
	for i, k := range wantKinds {
		if got[i].Kind != k {
			t.Fatalf("frame %d kind = %q, want %q", i, got[i].Kind, k)
		}
	}
	final := got[len(got)-1]
	if final.Content != "Looking done: 3 items." {
		t.Fatalf("final content = %q, want the scripted answer", final.Content)
	}
	if final.TotalTokens != 112 {
		t.Fatalf("final total_tokens = %d, want 112 (token accounting rides the frame)", final.TotalTokens)
	}
	// The tool round is faithfully framed (tool name + result survive the wire).
	if got[2].Tool != "query" || got[3].Result != `{"count":3}` {
		t.Fatalf("tool round frames malformed: start=%+v end=%+v", got[2], got[3])
	}
	if n := fake.turnCount(); n != 1 {
		t.Fatalf("session turned %d times, want 1", n)
	}
	if fake.inputs[0] != "summarize the desk" {
		t.Fatalf("turn input = %q, want the posted message", fake.inputs[0])
	}
}

// TestStreamRoute_ErrorEventFrame: a terminal error event (e.g. a canceled/interrupted turn)
// rides the same SSE channel as a serializable error frame with its partial text.
func TestStreamRoute_ErrorEventFrame(t *testing.T) {
	fake := &fakeStreamer{events: []agent.Event{
		{Kind: agent.EventToken, Token: "partial ans"},
		{Kind: agent.EventError, Err: "context canceled", Canceled: true, Partial: "partial ans"},
	}}
	srv, _ := newTestServer(t, func(context.Context) (Streamer, error) { return fake, nil })

	resp := postJSON(t, srv.URL+PathStream, `{"message":"go"}`)
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	got := parseSSE(t, string(b))

	last := got[len(got)-1]
	if last.Kind != agent.EventError || !last.Canceled || last.Err == "" || last.Partial != "partial ans" {
		t.Fatalf("terminal error frame malformed: %+v", last)
	}
}

// TestStreamRoute_BadRequest: an empty message or malformed body is rejected with 400 BEFORE any
// SSE stream is opened — a plain error status, not a broken stream.
func TestStreamRoute_BadRequest(t *testing.T) {
	srv, _ := newTestServer(t, func(context.Context) (Streamer, error) {
		t.Fatalf("session must not be built for a bad request")
		return nil, nil
	})

	for _, body := range []string{`{"message":""}`, `{`, `not json`} {
		resp := postJSON(t, srv.URL+PathStream, body)
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("body %q: status = %d, want 400", body, resp.StatusCode)
		}
		resp.Body.Close()
	}
}

// TestSessionReuse_MultiTurn: DoD 3 — successive turns reuse ONE held session (the factory is
// called once), which is what makes the browser conversation multi-turn (the model sees prior
// turns) exactly like a single REPL invocation over one agent.Session.
func TestSessionReuse_MultiTurn(t *testing.T) {
	var calls int32
	fake := &fakeStreamer{events: scriptedTurn()}
	srv, _ := newTestServer(t, func(context.Context) (Streamer, error) {
		atomic.AddInt32(&calls, 1)
		return fake, nil
	})

	for _, msg := range []string{`{"message":"one"}`, `{"message":"two"}`} {
		resp := postJSON(t, srv.URL+PathStream, msg)
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}

	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("session factory called %d times across two turns, want 1 (session must be reused)", got)
	}
	if n := fake.turnCount(); n != 2 {
		t.Fatalf("held session turned %d times, want 2", n)
	}
}

// TestResetRoute_ClosesAndRebuilds: DoD 6/lifecycle — reset finalizes the held session (Close)
// and the NEXT turn builds a fresh one, so a "new conversation" starts clean.
func TestResetRoute_ClosesAndRebuilds(t *testing.T) {
	var calls int32
	var fakes []*fakeStreamer
	var mu sync.Mutex
	srv, _ := newTestServer(t, func(context.Context) (Streamer, error) {
		atomic.AddInt32(&calls, 1)
		f := &fakeStreamer{events: scriptedTurn()}
		mu.Lock()
		fakes = append(fakes, f)
		mu.Unlock()
		return f, nil
	})

	// The first turn builds the first session.
	resp := postJSON(t, srv.URL+PathStream, `{"message":"one"}`)
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	// Reset closes it.
	rr := postJSON(t, srv.URL+PathReset, ``)
	if rr.StatusCode != http.StatusNoContent {
		t.Fatalf("reset status = %d, want 204", rr.StatusCode)
	}
	rr.Body.Close()

	// The next turn builds a second, fresh session.
	resp2 := postJSON(t, srv.URL+PathStream, `{"message":"two"}`)
	io.Copy(io.Discard, resp2.Body)
	resp2.Body.Close()

	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Fatalf("factory called %d times, want 2 (reset must force a rebuild)", got)
	}
	mu.Lock()
	defer mu.Unlock()
	if !fakes[0].isClosed() {
		t.Fatalf("the first session was not Closed on reset")
	}
	if fakes[1].isClosed() {
		t.Fatalf("the second session must still be live")
	}
}

// TestOriginGuard: the state-changing routes (POST stream + reset) accept a request with NO Origin
// (non-browser tools) and one whose Origin is any loopback form (127.0.0.1 OR localhost, distinct
// origins), but reject a cross-origin browser request with 403 — and for the SSE route the
// rejection arrives as a JSON error status BEFORE any text/event-stream header, never a broken
// stream. This closes the browser cross-site vector without adding auth (the posture stays
// unauthenticated + loopback-bound). The GET page is a navigation and is intentionally unguarded.
func TestOriginGuard(t *testing.T) {
	srv, _ := newTestServer(t, func(context.Context) (Streamer, error) {
		return &fakeStreamer{events: scriptedTurn()}, nil
	})
	// srv.URL is http://127.0.0.1:<port>; the localhost form is a DISTINCT origin that must also pass.
	origin127 := srv.URL
	originLocalhost := strings.Replace(srv.URL, "127.0.0.1", "localhost", 1)

	for _, origin := range []string{"" /* absent */, origin127, originLocalhost} {
		resp := postWithOrigin(t, srv.URL+PathStream, `{"message":"hi"}`, origin)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("stream with allowed origin %q: status = %d, want 200", origin, resp.StatusCode)
		}
		if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
			t.Fatalf("stream with allowed origin %q: Content-Type = %q, want text/event-stream", origin, ct)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}

	// A cross-origin browser request is rejected 403 — as JSON, BEFORE any SSE header.
	resp := postWithOrigin(t, srv.URL+PathStream, `{"message":"hi"}`, "https://evil.example")
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("stream cross-origin: status = %d, want 403", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); strings.HasPrefix(ct, "text/event-stream") {
		t.Fatalf("cross-origin rejection must NOT open an SSE stream; Content-Type = %q", ct)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("cross-origin rejection Content-Type = %q, want application/json", ct)
	}
	resp.Body.Close()

	// The reset route is guarded the same way.
	rr := postWithOrigin(t, srv.URL+PathReset, ``, "https://evil.example")
	if rr.StatusCode != http.StatusForbidden {
		t.Fatalf("reset cross-origin: status = %d, want 403", rr.StatusCode)
	}
	rr.Body.Close()
}

// TestSessionHolder_ResetWaitsForInFlightTurn: reset (and shutdown close) must never Close the
// held session while a turn is still writing to it. An *agent.Session guards only busy/termErr
// with its mutex — Close reads s.last while runTurn writes s.last/s.history unguarded — so a
// concurrent reset would race those fields. The holder's turn-lock closes that gap: reset blocks
// until the in-flight turn ends. (Run the package under -race to catch a regression here.)
func TestSessionHolder_ResetWaitsForInFlightTurn(t *testing.T) {
	fake := &fakeStreamer{events: scriptedTurn()}
	h := &sessionHolder{newSess: func(context.Context) (Streamer, error) { return fake, nil }}

	// Materialize the held session and mark a turn in progress (as the stream handler does).
	h.beginTurn()
	if _, err := h.get(context.Background()); err != nil {
		t.Fatalf("get: %v", err)
	}

	resetReturned := make(chan struct{})
	go func() { h.reset(context.Background()); close(resetReturned) }()

	// While the turn is held, reset must block and the session must NOT be Closed.
	select {
	case <-resetReturned:
		t.Fatal("reset returned while a turn was in flight — Close could race the turn")
	case <-time.After(50 * time.Millisecond):
	}
	if fake.isClosed() {
		t.Fatal("session was Closed while a turn was in flight")
	}

	// End the turn; reset must now proceed and Close the session.
	h.endTurn()
	select {
	case <-resetReturned:
	case <-time.After(2 * time.Second):
		t.Fatal("reset did not return after the turn ended")
	}
	if !fake.isClosed() {
		t.Fatal("session was not Closed after the turn ended")
	}
}

// TestWriteBoundary_NoWritePaths: DoD 6 — the surface exposes ONLY the three session routes;
// there is no fix/apply/restore endpoint. The write boundary is inherited from the reused gated
// session (agent.buildTools -> toolcore.AgentTools), not re-opened here. Defense-in-depth: a
// would-be write path is a 404.
func TestWriteBoundary_NoWritePaths(t *testing.T) {
	srv, _ := newTestServer(t, func(context.Context) (Streamer, error) {
		return &fakeStreamer{events: scriptedTurn()}, nil
	})

	for _, p := range []string{
		"/desk/chat/apply-fix", "/desk/chat/apply_fix", "/desk/chat/restore",
		"/desk/apply-fix", "/desk/restore", "/desk/chat/fix",
	} {
		resp := postJSON(t, srv.URL+p, `{}`)
		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("write-ish path %q returned %d, want 404 (no write route may exist)", p, resp.StatusCode)
		}
		resp.Body.Close()
	}
}

// --- public mode (a non-loopback bind) ---

// authToken mints a real auth token for a record in the named collection, so the public-mode
// tests present the same Authorization header a live client would.
func authToken(t *testing.T, app *tests.TestApp, collection, email string) string {
	t.Helper()
	col, err := app.FindCollectionByNameOrId(collection)
	if err != nil {
		t.Fatalf("find %s: %v", collection, err)
	}
	rec := pbcore.NewRecord(col)
	rec.SetEmail(email)
	rec.SetPassword("a-sufficiently-long-password")
	rec.SetVerified(true)
	if col.Name == "users" {
		rec.Set("approved", true)
	}
	if err := app.Save(rec); err != nil {
		t.Fatalf("save %s record: %v", collection, err)
	}
	tok, err := rec.NewAuthToken()
	if err != nil {
		t.Fatalf("mint auth token: %v", err)
	}
	return tok
}

func requestWithHeaders(t *testing.T, method, url, body string, headers map[string]string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		t.Fatalf("new request %s: %v", url, err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	return resp
}

// TestPublicMode_UnauthenticatedIs401: on a non-loopback bind, both session routes
// refuse an unauthenticated request. This is the whole point of deriving the mode from the bind
// address: expose the port and the surface stops being open, with no flag to remember.
func TestPublicMode_UnauthenticatedIs401(t *testing.T) {
	srv, _, _ := newTestServerMode(t, func(context.Context) (Streamer, error) {
		t.Fatal("no session may be built for an unauthenticated public-mode request")
		return nil, nil
	}, true)

	cases := []struct {
		method, path, body string
	}{
		{http.MethodPost, PathStream, `{"message":"hi"}`},
		{http.MethodPost, PathReset, ``},
	}
	for _, c := range cases {
		resp := requestWithHeaders(t, c.method, srv.URL+c.path, c.body, nil)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("%s %s (public, no token): status = %d, want 401", c.method, c.path, resp.StatusCode)
		}
		resp.Body.Close()
	}
}

// TestPublicMode_AuthenticatedPasses: an approved `users` token and a superuser token both work —
// the operator's own admin credential needs no second account, and a hosted desk's members use
// the approval-gated collection.
func TestPublicMode_AuthenticatedPasses(t *testing.T) {
	for _, collection := range []string{"users", pbcore.CollectionNameSuperusers} {
		t.Run(collection, func(t *testing.T) {
			fake := &fakeStreamer{events: scriptedTurn()}
			srv, _, app := newTestServerMode(t, func(context.Context) (Streamer, error) { return fake, nil }, true)
			tok := authToken(t, app, collection, "member-"+collection+"@desk.test")

			// Same-origin POST (the chat screen's fetch) is accepted and streams.
			resp := requestWithHeaders(t, http.MethodPost, srv.URL+PathStream, `{"message":"hi"}`,
				map[string]string{"Authorization": tok, "Origin": srv.URL})
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("authenticated same-origin stream: status = %d, want 200", resp.StatusCode)
			}
			if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
				t.Fatalf("Content-Type = %q, want text/event-stream", ct)
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		})
	}
}

// TestPublicMode_OriginIsSameOriginNotWildcard: in public mode the loopback allowlist would 403
// every real request, so the rule becomes strict same-origin. It is NOT a wildcard: a foreign
// origin — and a loopback origin, which on a hosted bind can only be a cross-site forgery — are
// both still rejected, while an absent Origin (curl/API clients) is allowed as before.
func TestPublicMode_OriginIsSameOriginNotWildcard(t *testing.T) {
	fake := &fakeStreamer{events: scriptedTurn()}
	srv, _, app := newTestServerMode(t, func(context.Context) (Streamer, error) { return fake, nil }, true)
	tok := authToken(t, app, pbcore.CollectionNameSuperusers, "op@desk.test")

	allowed := []string{"" /* absent */, srv.URL /* same origin */}
	for _, origin := range allowed {
		h := map[string]string{"Authorization": tok}
		if origin != "" {
			h["Origin"] = origin
		}
		resp := requestWithHeaders(t, http.MethodPost, srv.URL+PathStream, `{"message":"hi"}`, h)
		if resp.StatusCode != http.StatusOK {
			t.Errorf("public-mode origin %q: status = %d, want 200", origin, resp.StatusCode)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}

	rejected := []string{
		"https://evil.example",
		"http://127.0.0.1:9999",                                  // loopback is NOT a free pass on a hosted bind
		strings.Replace(srv.URL, "127.0.0.1", "localhost", 1),    // a different host spelling is a different origin
		strings.Replace(srv.URL, "http://", "https://", 1) + ":", // malformed
	}
	for _, origin := range rejected {
		resp := requestWithHeaders(t, http.MethodPost, srv.URL+PathStream, `{"message":"hi"}`,
			map[string]string{"Authorization": tok, "Origin": origin})
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("public-mode origin %q: status = %d, want 403", origin, resp.StatusCode)
		}
		resp.Body.Close()
	}
}

// NOTE: the wildcard-CORS assertion that used to live here was a FALSE GREEN and has been moved
// to cmd/deskkit's TestHardenPublicCORS. The middleware that emits Access-Control-Allow-Origin is
// bound by the dependency's Serve() onto the serve router, NOT by apis.NewRouter — which is what
// newTestServerMode builds — so a header assertion against this server could never observe the
// wildcard and passed while the real binary emitted `Access-Control-Allow-Origin: *` on every
// response. Assert response headers only where the middleware under test is actually bound.

// TestLoopbackMode_UnchangedByPublicPlumbing: the local posture is byte-for-byte what it was —
// no token required on either route, and the loopback-origin allowlist still applies.
// This is the "default local deskkit serve behavior unchanged" assertion.
func TestLoopbackMode_UnchangedByPublicPlumbing(t *testing.T) {
	fake := &fakeStreamer{events: scriptedTurn()}
	srv, _, _ := newTestServerMode(t, func(context.Context) (Streamer, error) { return fake, nil }, false)

	resp := requestWithHeaders(t, http.MethodPost, srv.URL+PathStream, `{"message":"hi"}`,
		map[string]string{"Origin": strings.Replace(srv.URL, "127.0.0.1", "localhost", 1)})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("loopback stream from the localhost spelling: status = %d, want 200", resp.StatusCode)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
}

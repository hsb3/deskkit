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

	"github.com/pocketbase/pocketbase/apis"
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
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("tests.NewTestApp: %v", err)
	}
	t.Cleanup(app.Cleanup)

	r, err := apis.NewRouter(app)
	if err != nil {
		t.Fatalf("apis.NewRouter: %v", err)
	}
	cleanup := Register(r, factory)

	mux, err := r.BuildMux()
	if err != nil {
		t.Fatalf("BuildMux: %v", err)
	}
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, cleanup
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

// TestPageRoute_ServesHTML: DoD 1 — a running server serves the session page at the documented
// URL with 200 + text/html, via the custom Go route registered by web.Register.
func TestPageRoute_ServesHTML(t *testing.T) {
	srv, _ := newTestServer(t, func(context.Context) (Streamer, error) {
		return &fakeStreamer{}, nil // never called on the page route
	})

	resp, err := http.Get(srv.URL + PathChat)
	if err != nil {
		t.Fatalf("GET %s: %v", PathChat, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Fatalf("Content-Type = %q, want text/html…", ct)
	}
	b, _ := io.ReadAll(resp.Body)
	body := string(b)
	for _, marker := range []string{"<!doctype html", `id="messages"`, "deskkit"} {
		if !strings.Contains(strings.ToLower(body), strings.ToLower(marker)) {
			t.Fatalf("page body missing marker %q", marker)
		}
	}
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

// Package web is the local browser session surface for the librarian: a custom Go route
// mounted on the embedded PocketBase `serve` that serves a purpose-built, self-contained page
// driving the SAME multi-turn stewardship session the `chat` REPL exposes. It is the deferred
// follow-on recorded by ADR 0001 (interactive-surface: terminal first, PocketBase-served webapp
// later) — option (b): a small custom Go route serving an embedded page, kept in the one binary
// so the single-binary identity holds (no second toolchain, no separate frontend deploy).
//
// The write boundary and the stewardship-lane boundary are INHERITED, not re-declared: this
// surface opens no new tool or write path. Every turn runs over an *agent.Session, whose tool
// slice is the gated set (buildTools -> toolcore.SelectByModules(toolcore.AgentTools(cfg),
// "librarian")): `restore` is never exposed and `apply_fix` is present only when
// LIBRARIAN_AUTONOMOUS_WRITES is set. The compile-time assertion below pins that the surface
// streams the real session type, never a fork.
//
// Auth posture: the route is unauthenticated, exactly like the TUI/REPL. Safety comes from the
// server's loopback binding (the operator serves the DB on 127.0.0.1) — it is a local,
// on-demand, single-operator surface, not a hosted service. It is deliberately NOT wired to the
// superuser admin auth (which would disqualify it as a general surface, ADR 0001).
package web

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"

	_ "embed"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"

	"github.com/hsb3/desk-standard/librarian/internal/modules/librarian/agent"
)

// Route paths. The page is a human destination (served as HTML); stream and reset are the
// page's own fetch endpoints. Exported so tests and docs reference the same literals.
const (
	// PathChat is the documented URL a person visits in one browser request.
	PathChat = "/desk/chat"
	// PathStream is the SSE endpoint the page POSTs each turn to.
	PathStream = "/desk/chat/stream"
	// PathReset ends the current server-held conversation so the next turn starts fresh.
	PathReset = "/desk/chat/reset"
)

// maxRequestBody caps a turn's request body. The surface is loopback and single-operator, but a
// bound keeps a stray client from streaming an unbounded body into memory before the turn runs.
const maxRequestBody = 1 << 20 // 1 MiB

//go:embed assets/index.html
var indexHTML []byte

// Streamer is the slice of *agent.Session this surface depends on: drive one multi-turn
// conversation as a live event stream, and finalize it. Keeping the dependency an interface is
// what lets the SSE handler be exercised with a fake event source (no live LLM) while production
// injects the real session. The assertion pins that the production type satisfies it.
type Streamer interface {
	// StreamTurn drives one full ReAct turn over the running history and returns a channel of
	// live events; exactly one terminal (final|error) event is emitted, then the channel closes.
	StreamTurn(ctx context.Context, input string) <-chan agent.Event
	// Close finalizes the underlying run row (non-fatal at shutdown).
	Close(ctx context.Context) error
}

var _ Streamer = (*agent.Session)(nil)

// NewSessionFunc builds a fresh session on demand. Production wires it to agent.NewSession
// (which builds the gated tool slice + the data-backed system prompt); a test injects a fake.
type NewSessionFunc func(ctx context.Context) (Streamer, error)

// sessionHolder lazily creates and reuses ONE session across turns, so the browser conversation
// is multi-turn (the model sees prior turns) exactly like a single REPL invocation. The session
// carries its own history bound (maxHistoryMessages) and its own overlapping-turn guard, so a
// second concurrent turn is serialized by the session, not by a second session here.
type sessionHolder struct {
	mu      sync.Mutex
	sess    Streamer
	newSess NewSessionFunc
}

// get returns the held session, creating it on first use. A creation failure is returned to the
// caller (surfaced as an HTTP error BEFORE the SSE headers are written), never cached.
func (h *sessionHolder) get(ctx context.Context) (Streamer, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.sess != nil {
		return h.sess, nil
	}
	s, err := h.newSess(ctx)
	if err != nil {
		return nil, err
	}
	h.sess = s
	return s, nil
}

// reset closes and drops the held session so the next turn starts a fresh conversation. Close
// errors are non-fatal (the run row finalize is best-effort).
func (h *sessionHolder) reset(ctx context.Context) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.sess != nil {
		_ = h.sess.Close(ctx)
		h.sess = nil
	}
}

// close finalizes the held session at server shutdown (best-effort).
func (h *sessionHolder) close(ctx context.Context) {
	h.reset(ctx)
}

// handler binds the routes to one shared session holder.
type handler struct {
	holder *sessionHolder
}

// Register mounts the three session-surface routes on the serve router, all backed by one shared
// session holder. Call it once under OnServe (the routes are serve-only, like the wake layer).
// The returned cleanup finalizes the held session; wire it to app shutdown (best-effort).
func Register(r *router.Router[*core.RequestEvent], newSession NewSessionFunc) (cleanup func(context.Context)) {
	h := &handler{holder: &sessionHolder{newSess: newSession}}
	r.GET(PathChat, h.page)
	r.POST(PathStream, h.stream)
	r.POST(PathReset, h.reset)
	return h.holder.close
}

// page serves the self-contained session page. No session is created here — the page loads with
// no LLM key required; the session is built lazily on the first turn.
func (h *handler) page(e *core.RequestEvent) error {
	return e.Blob(http.StatusOK, "text/html; charset=utf-8", indexHTML)
}

// streamRequest is the turn request body the page POSTs.
type streamRequest struct {
	Message string `json:"message"`
}

// stream runs one turn and streams the session's events to the browser as Server-Sent Events.
// Body read + session build happen FIRST (they can still return a normal HTTP error status);
// once the SSE headers are written the response is committed to 200 and errors ride in-band as
// an agent.Event{Kind: error} frame — matching the streaming contract (exactly one terminal
// event per turn).
func (h *handler) stream(e *core.RequestEvent) error {
	e.Request.Body = http.MaxBytesReader(e.Response, e.Request.Body, maxRequestBody)
	var req streamRequest
	if err := json.NewDecoder(e.Request.Body).Decode(&req); err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body: " + err.Error()})
	}
	if req.Message == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": "message must not be empty"})
	}

	ctx := e.Request.Context()
	sess, err := h.holder.get(ctx)
	if err != nil {
		// Session build failed (e.g. no provider key): report as a normal error status BEFORE any
		// SSE header is written, so the page can show a plain error rather than a broken stream.
		return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	e.Response.Header().Set("Content-Type", "text/event-stream")
	e.Response.Header().Set("Cache-Control", "no-cache")
	e.Response.Header().Set("Connection", "keep-alive")
	e.Response.Header().Set("X-Accel-Buffering", "no") // defeat proxy buffering if one is ever in front
	e.Response.WriteHeader(http.StatusOK)
	_ = e.Flush()

	// Drain the whole turn to close. StreamTurn requires prompt draining — a stalled consumer
	// stalls the loop. On a client write error we keep ranging (never break) so the turn still
	// drains to its terminal event; only the on-wire delivery is lost.
	writeFailed := false
	for ev := range sess.StreamTurn(ctx, req.Message) {
		if writeFailed {
			continue
		}
		if err := writeSSE(e.Response, ev); err != nil {
			writeFailed = true
			continue
		}
		_ = e.Flush()
	}
	return nil
}

// reset ends the current conversation. The next stream builds a fresh session.
func (h *handler) reset(e *core.RequestEvent) error {
	h.holder.reset(e.Request.Context())
	return e.NoContent(http.StatusNoContent)
}

// writeSSE marshals one event and writes it as a single SSE `data:` frame. The event carries its
// own `kind`, so a single data line per frame is sufficient; the page switches on ev.kind.
func writeSSE(w io.Writer, ev agent.Event) error {
	b, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "data: %s\n\n", b); err != nil {
		return err
	}
	return nil
}

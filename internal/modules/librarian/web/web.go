// Package web is the browser session API for the librarian: custom Go routes on the embedded
// PocketBase `serve` that drive the SAME multi-turn stewardship session the `chat` REPL exposes.
// The chat page itself lives in the embedded SPA (the core spa package serves the shell at `/`,
// and /desk/chat lands there via the SPA's index fallback); what remains here is the session
// transport the SPA's chat screen POSTs to.
//
// The write boundary and the stewardship-lane boundary are INHERITED, not re-declared: this
// surface opens no new tool or write path. Every turn runs over an *agent.Session whose tool
// slice is the gated set, so `restore` is never exposed and `apply_fix` is present only under
// LIBRARIAN_AUTONOMOUS_WRITES. The compile-time assertion below pins that the surface streams the
// real session type, never a fork.
//
// Auth posture depends on the RESOLVED BIND ADDRESS, which the caller classifies and passes to
// Register as `public`.
//
//   - Loopback bind (the local default): the routes are unauthenticated, exactly like the
//     TUI/REPL. Safety comes from the loopback binding — a local, on-demand, single-operator
//     surface, not a hosted service. It is deliberately NOT wired to superuser admin auth, which
//     would disqualify it as a general surface.
//   - Non-loopback bind (public mode): both routes require a valid auth token, and the
//     cross-origin guard switches from the loopback allowlist to a strict same-origin check
//     against the request's own Host. No wildcard CORS is ever configured — a `*` policy would
//     hand the surface to any page on the internet.
//
// The mode is derived from the exposure rather than an opt-in flag, because a flag can be
// forgotten while still binding 0.0.0.0, which fails OPEN.
package web

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"

	"github.com/hsb3/deskkit/internal/modules/librarian/agent"
)

// Route paths — the chat screen's fetch endpoints. Exported so tests and docs reference the same
// literals. There is no GET page route here: /desk/chat serves the SPA shell through the spa
// package's index fallback.
const (
	// PathStream is the SSE endpoint the chat screen POSTs each turn to.
	PathStream = "/desk/chat/stream"
	// PathReset ends the current server-held conversation so the next turn starts fresh.
	PathReset = "/desk/chat/reset"
)

// maxRequestBody caps a turn's request body. The surface is loopback and single-operator, but a
// bound keeps a stray client from streaming an unbounded body into memory before the turn runs.
const maxRequestBody = 1 << 20 // 1 MiB

// Streamer is the slice of *agent.Session this surface depends on: drive one multi-turn
// conversation as a live event stream, and finalize it. An interface, so the SSE handler is
// exercisable with a fake event source and no live LLM; the assertion below pins that the
// production type satisfies it.
type Streamer interface {
	// StreamTurn drives one full ReAct turn over the running history and returns a channel of
	// live events; exactly one terminal (final|error) event is emitted, then the channel closes.
	StreamTurn(ctx context.Context, input string) <-chan agent.Event
	// Close finalizes the underlying run row (non-fatal at shutdown).
	Close(ctx context.Context) error
}

var _ Streamer = (*agent.Session)(nil)

// NewSessionFunc builds a fresh session on demand. Production wires it to agent.NewSession, which
// builds the gated tool slice + the data-backed system prompt; a test injects a fake.
type NewSessionFunc func(ctx context.Context) (Streamer, error)

// sessionHolder lazily creates and reuses ONE session across turns, so the browser conversation is
// multi-turn exactly like a single REPL invocation. The session carries its own history bound and
// its own overlapping-turn guard, so a second concurrent turn is serialized by the session, not by
// a second session here.
//
// Concurrency: an *agent.Session assumes a single sequential driver. Its Close() reads s.last
// while StreamTurn's runTurn writes s.last/s.history OUTSIDE the session's own mutex, which
// guards only busy/termErr. Concurrent HTTP callers make "reset during an in-flight turn"
// reachable, racing those fields. turnMu closes that gap without reaching into the session: a turn
// holds it for read for its whole duration, and reset/close take it for write, so a Close can
// never overlap a running turn.
type sessionHolder struct {
	turnMu  sync.RWMutex // a turn holds RLock; reset/close hold Lock — Close never overlaps a turn
	mu      sync.Mutex   // guards sess/newSess
	sess    Streamer
	newSess NewSessionFunc
}

// beginTurn marks a turn in progress; endTurn ends it. Between them a reset/close blocks rather
// than closing the session mid-turn. Held by the stream handler around get()+StreamTurn().
func (h *sessionHolder) beginTurn() { h.turnMu.RLock() }
func (h *sessionHolder) endTurn()   { h.turnMu.RUnlock() }

// get returns the held session, creating it on first use. A creation failure is returned to the
// caller (surfaced as an HTTP error BEFORE the SSE headers are written), never cached. Called
// while the caller holds the turn read-lock, so the session it returns cannot be closed under it.
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

// reset closes and drops the held session so the next turn starts a fresh conversation. It takes
// the turn write-lock first, so it waits for any in-flight turn to finish before Close reads the
// session's fields. Close errors are non-fatal: the run-row finalize is best-effort. Lock order is
// ALWAYS turnMu before mu — get/reset never take mu first — so there is no deadlock.
func (h *sessionHolder) reset(ctx context.Context) {
	h.turnMu.Lock()
	defer h.turnMu.Unlock()
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.sess != nil {
		_ = h.sess.Close(ctx)
		h.sess = nil
	}
}

// close finalizes the held session at server shutdown (best-effort); like reset it waits for any
// in-flight turn.
func (h *sessionHolder) close(ctx context.Context) {
	h.reset(ctx)
}

// authCollections are the auth collections whose tokens satisfy the public-mode requirement:
// `users`, the approval-gated collection where a member must be both verified and
// operator-approved before it can obtain a token, plus the stock superusers collection, so the
// operator's own admin token works without a second account.
var authCollections = []string{"users", core.CollectionNameSuperusers}

// handler binds the routes to one shared session holder. public mirrors Register's argument and
// is what the origin guard consults per request.
type handler struct {
	holder *sessionHolder
	public bool
}

// Register mounts the two session-API routes on the serve router, both backed by one shared
// session holder. Call it once under OnServe; the routes are serve-only. The returned cleanup
// finalizes the held session — wire it to app shutdown, best-effort.
//
// public says the server is bound to a non-loopback address (see the package doc). False for every
// local `deskkit serve`, where the routes stay unauthenticated and loopback-origin-guarded. When
// true, both routes bind behind apis.RequireAuth; the SPA shell fronting them is static and served
// without auth, matching the admin console's shell-loads-data-doesn't stance.
func Register(r *router.Router[*core.RequestEvent], newSession NewSessionFunc, public bool) (cleanup func(context.Context)) {
	h := &handler{holder: &sessionHolder{newSess: newSession}, public: public}
	routes := []*router.Route[*core.RequestEvent]{
		r.POST(PathStream, h.stream),
		r.POST(PathReset, h.reset),
	}
	if public {
		for _, rt := range routes {
			rt.Bind(apis.RequireAuth(authCollections...))
		}
	}
	return h.holder.close
}

// streamRequest is the turn request body the chat screen POSTs.
type streamRequest struct {
	Message string `json:"message"`
}

// stream runs one turn and streams the session's events to the browser as Server-Sent Events.
// Body read + session build happen FIRST, while a normal HTTP error status is still possible;
// once the SSE headers are written the response is committed to 200 and errors ride in-band as an
// agent.Event{Kind: error} frame, keeping the one-terminal-event-per-turn contract.
func (h *handler) stream(e *core.RequestEvent) error {
	if !originAllowed(e.Request, h.public) {
		return e.JSON(http.StatusForbidden, crossOriginRejected)
	}
	e.Request.Body = http.MaxBytesReader(e.Response, e.Request.Body, maxRequestBody)
	var req streamRequest
	if err := json.NewDecoder(e.Request.Body).Decode(&req); err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body: " + err.Error()})
	}
	if req.Message == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": "message must not be empty"})
	}

	// Hold the turn read-lock for the whole turn (session build + drain), so a concurrent reset or
	// a shutdown cannot Close the session while runTurn is still writing to it. Acquired only after
	// the request is validated, so a bad request takes no lock.
	h.holder.beginTurn()
	defer h.holder.endTurn()

	ctx := e.Request.Context()
	sess, err := h.holder.get(ctx)
	if err != nil {
		// Session build failed (e.g. no provider key): report a normal error status BEFORE any SSE
		// header is written, so the page shows a plain error rather than a broken stream.
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
	if !originAllowed(e.Request, h.public) {
		return e.JSON(http.StatusForbidden, crossOriginRejected)
	}
	h.holder.reset(e.Request.Context())
	return e.NoContent(http.StatusNoContent)
}

// crossOriginRejected is the 403 body for a request whose Origin fails the guard.
var crossOriginRejected = map[string]string{
	"error": "cross-origin request rejected: this surface accepts only same-origin browser requests",
}

// originAllowed is the cross-origin guard for the state-changing POST routes. On a loopback bind it
// adds NO authentication — the posture stays unauthenticated and loopback-bound — and closes only
// the browser cross-site vector: a page on another origin silently POSTing to this local surface.
// On a public bind it runs alongside the route-level RequireAuth as CSRF defense in depth.
//
// A request with NO Origin header (curl, other non-browser tools, a same-origin navigation) is
// allowed in both modes: absence means it is not a browser cross-site request. A present Origin
// must always carry an http/https scheme.
//
// Loopback mode: the Origin's host must be 127.0.0.1, localhost, or ::1, any port. Those are
// distinct origins, and the allowlist accepts both spellings regardless of which one the operator
// browsed to.
//
// Public mode: that allowlist would 403 every real request, so the rule becomes strict SAME-ORIGIN
// — the Origin's host:port must equal the request's own Host header. Unlike a permissive CORS
// policy it names no wildcard, so an unknown third-party origin is still rejected. The comparison
// is u.Host (port included when present) against r.Host, so a port mismatch is a rejection rather
// than a silent pass.
func originAllowed(r *http.Request, public bool) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true // not a browser cross-site request
	}
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}
	if public {
		return u.Host != "" && u.Host == r.Host
	}
	switch u.Hostname() {
	case "127.0.0.1", "localhost", "::1":
		return true
	default:
		return false
	}
}

// writeSSE marshals one event and writes it as a single SSE `data:` frame. The event carries its
// own `kind`, so one data line per frame is sufficient and the page switches on ev.kind.
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

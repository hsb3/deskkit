// Package mcp is the OUTBOUND MCP-server slice (build-brief §5, punch-list item 4;
// spec §7.2 dual-surface, added 2026-07-16). It exposes the enabled modules' frozen tool
// core (internal/core/toolcore) as an MCP stdio server — the librarian's "hands" — so a
// Claude Code or OpenCode session (or the dual-format plugin's plugin/mcp boundary) can call
// the tools directly. This is the one-binary "MCP server + CLI over a single tool core"
// pattern; the CLI (cmd/deskkit) and the eino agent loop
// (internal/modules/librarian/agent) remain the other two surfaces over the SAME tool core.
//
// Three invariants this slice enforces (all load-bearing):
//
//  1. Zero logic duplication (spec §2.6). Every MCP tool handler calls the SAME tool function
//     the CLI and the eino agent call (via toolcore.ToolSpec.RegisterMCP). Each tool's
//     parameter schema is derived by reflection over the SAME input struct (its tags) used
//     everywhere else — one source of truth, no re-implemented behavior.
//
//  2. The §5.4 write gate + §5.5 restore exclusion. The MCP surface is model-facing exactly
//     like the eino loop, so it uses the identical registration-time gate: the exposed set is
//     toolcore.ExposedTools(cfg) — every AgentDefault tool always, plus each AgentGated tool
//     ONLY when LIBRARIAN_AUTONOMOUS_WRITES=true. restore is NEVER exposed over MCP (recovery
//     is a supervised CLI action, §5.5); its exclusion is STRUCTURAL — ExposedTools filters it
//     defensively, and it is neither AgentDefault nor AgentGated so AgentTools never returns it.
//
//  3. Explicit input schemas (not the SDK's struct-tag inference). The frozen input structs in
//     modules/librarian/tools/types.go carry eino/invopop-style `jsonschema:"description=…"`
//     KEY-VALUE tags. The official MCP SDK's inference (google/jsonschema-go) treats a
//     `jsonschema` tag as PLAIN TEXT and rejects any tag beginning with `WORD=` — so it cannot
//     infer from these structs. eino (the live agent loop) depends on that same key-value
//     grammar, and types.go is frozen, so the structs cannot satisfy both parsers by tags
//     alone. toolcore therefore builds each tool's *jsonschema.Schema itself (moved verbatim
//     from this package) and sets mcp.Tool.InputSchema, which makes AddTool skip its own
//     inference.
package mcp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/pocketbase/pocketbase/core"

	"github.com/example/pocket-librarian/internal/core/config"
	"github.com/example/pocket-librarian/internal/core/toolcore"
)

// serverName / serverVersion identify the MCP server in the initialize handshake.
// Identity-neutral: no person/org/repo/issue (0013 item 9). The module path is already
// github.com/example/pocket-librarian; this keeps the wire identity neutral too.
const (
	serverName    = "deskkit"
	serverVersion = "v1"
)

// ExposedTools returns the tool names this server registers for cfg: toolcore.ExposedTools —
// the §5.4 gate applied with restore excluded (§5.5). This is the SINGLE source both the server
// registration loop and the tests consult, so the test verifies the real logic.
func ExposedTools(cfg *config.Config) []string {
	return toolcore.ExposedTools(cfg)
}

// NewServer builds the MCP server with exactly the gated tool set for cfg, looping over the
// merged toolcore registry (populated from the enabled module set at startup) instead of a
// hand-maintained name-keyed switch — so a second module's tools register here without editing
// this file. app is used only inside the tool handlers (invoked per request), not during
// registration, so it may be nil in tests that only assert the registered set.
func NewServer(app core.App, cfg *config.Config) (*mcp.Server, error) {
	s := mcp.NewServer(&mcp.Implementation{Name: serverName, Version: serverVersion}, nil)
	for _, name := range toolcore.ExposedTools(cfg) {
		spec, ok := toolcore.Spec(name)
		if !ok {
			return nil, fmt.Errorf("mcp: exposed tool %q not in registry (registrar drift)", name)
		}
		if err := spec.RegisterMCP(s, app, cfg); err != nil {
			return nil, fmt.Errorf("mcp: register tool %q: %w", name, err)
		}
	}
	return s, nil
}

// Serve builds the gated server and runs it over stdio until the client closes stdin (EOF)
// or ctx is cancelled. mcp-serve opens the DB like any one-shot tool and holds it for the
// session, so it must not run concurrently with `serve` (single-writer SQLite rule, §10.4).
//
// Clean shutdown is not a failure. A well-behaved stdio client closes stdin after its last
// request; the SDK's stdio session surfaces that as one of a small family of shutdown errors
// which isShutdownEOF classifies. Returning any of them would make cobra print a spurious
// "Error: server is closing: EOF" + a usage dump and exit non-zero — so we swallow them and
// exit 0/silent instead (field-eval finding).
func Serve(ctx context.Context, app core.App, cfg *config.Config) error {
	// Fail LOUD, never "silently absent". An MCP host (e.g. a plugin .mcp.json entry) launches
	// mcp-serve as a subprocess and surfaces its stderr in the server's own logs. If the desk did
	// not resolve, registering a degenerate/empty tool surface would look like a mysterious absence
	// rather than a fixable error — so name the missing identity and exit non-zero right here. A
	// direct os.Exit (not a returned error) keeps this to ONE clean line: PocketBase's RootCmd
	// silences neither errors nor usage, so a returned RunE error would print "Error: …" plus a
	// full usage dump. In normal operation the CLI's requireConfig gate already fails first, so
	// this is a defensive belt-and-suspenders for any caller (or refactor) that reaches Serve with
	// an unresolved cfg — the surface refuses to serve the wrong desk ON ITS OWN.
	if err := requireResolvedConfig(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "deskkit mcp-serve: %v\n", err)
		os.Exit(1)
	}
	s, err := NewServer(app, cfg)
	if err != nil {
		return err
	}
	// Mount signal: emit ONE concise line so a host operator can SEE the surface came up (and
	// which tools it carries — the PM tool family only appears when PM is enabled) instead of
	// guessing at a silent absence. It MUST go to stderr — stdout is the JSON-RPC channel and any
	// stray byte there corrupts the protocol.
	emitMountSignal(os.Stderr, ExposedTools(cfg))
	if rerr := s.Run(ctx, &mcp.StdioTransport{}); rerr != nil && !isShutdownEOF(rerr) {
		return rerr
	}
	return nil
}

// requireResolvedConfig reports whether cfg is a fully-resolved desk (a non-nil config with a
// DeskRoot and DeskName). An empty desk would otherwise register tools against a nil desk and
// answer requests wrongly, all while looking like a working-but-empty surface. The returned
// message names the missing identity and how to set it, so the failure is actionable straight
// from the host's server log.
func requireResolvedConfig(cfg *config.Config) error {
	if cfg == nil {
		return errors.New("desk not resolved: DESK_ROOT and DESK_NAME unset; set them via env or a discoverable _knowledge/profile.*")
	}
	var missing []string
	if strings.TrimSpace(cfg.DeskRoot) == "" {
		missing = append(missing, "DESK_ROOT")
	}
	if strings.TrimSpace(cfg.DeskName) == "" {
		missing = append(missing, "DESK_NAME")
	}
	if len(missing) > 0 {
		return fmt.Errorf("desk not resolved: %s unset; set via env or a discoverable _knowledge/profile.* (run deskkit inside the desk), then restart the MCP host session",
			strings.Join(missing, ", "))
	}
	return nil
}

// emitMountSignal writes ONE concise mount line to w: the server identity plus the exact tool set
// it exposed. Presence of this line in the host's server log is the observable "the surface
// mounted" signal (its absence, a diagnostic). Callers MUST pass a stderr writer — never stdout,
// the JSON-RPC channel.
func emitMountSignal(w io.Writer, tools []string) {
	fmt.Fprintf(w, "deskkit mcp-serve: mounted %q %s; %d tool(s) exposed: %s\n",
		serverName, serverVersion, len(tools), strings.Join(tools, ", "))
}

// isShutdownEOF reports whether err is the normal end of a stdio MCP session rather than a
// real failure: a plain io.EOF, a cancelled/expired context, or the SDK's jsonrpc2
// "server is closing: EOF" — which wraps its (internal, unimportable) ErrServerClosing
// sentinel via %w and appends io.EOF via %v, so errors.Is(err, io.EOF) does NOT match it and
// the message must be matched directly. The match is deliberately on the "server is closing"
// sentinel ALONE, not "…: EOF": a client that drops the read end mid-write terminates the
// session via a write error (broken pipe), yielding "server is closing: <writeErr>" — also a
// normal stdio disconnect that must exit clean. A genuine mid-session failure does not carry
// the "server is closing" sentinel, so it still surfaces.
func isShutdownEOF(err error) bool {
	if errors.Is(err, io.EOF) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	return strings.Contains(err.Error(), "server is closing")
}

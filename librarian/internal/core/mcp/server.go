// Package mcp is the OUTBOUND MCP-server slice (build-brief §5, punch-list item 4;
// spec §7.2 dual-surface, added 2026-07-16). It exposes the enabled modules' frozen tool
// core (internal/core/toolcore) as an MCP stdio server — the librarian's "hands" — so a
// Claude Code or OpenCode session (or the dual-format plugin's plugin/mcp boundary) can call
// the tools directly. This is the one-binary "MCP server + CLI over a single tool core"
// pattern; the CLI (cmd/pocket-librarian) and the eino agent loop
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
	serverName    = "pocket-librarian"
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
	s, err := NewServer(app, cfg)
	if err != nil {
		return err
	}
	if rerr := s.Run(ctx, &mcp.StdioTransport{}); rerr != nil && !isShutdownEOF(rerr) {
		return rerr
	}
	return nil
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

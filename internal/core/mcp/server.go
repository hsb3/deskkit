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

	"github.com/hsb3/deskkit/internal/core/config"
	"github.com/hsb3/deskkit/internal/core/toolcore"
)

// serverName / serverVersion identify the MCP server in the initialize handshake.
// Identity-neutral: no person/org/repo/issue (0013 item 9). The module path is already
// github.com/hsb3/deskkit; this keeps the wire identity neutral too.
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

// NewServer builds the MCP server with exactly the §5.4-gated tool set for cfg — the FULL exposed
// set (toolcore.ExposedTools(cfg)), NOT module-gated. Module gating (MCP_MODULES) is a Serve-path
// concern only, so NewServer stays the stable "all exposed tools" builder the module tests
// (gatedon/gatedoff) and TestNewServer_BuildsForBothGates rely on. It loops over the merged
// toolcore registry (populated from the enabled module set at startup) instead of a
// hand-maintained name-keyed switch — so a second module's tools register here without editing
// this file. app is used only inside the tool handlers (invoked per request), not during
// registration, so it may be nil in tests that only assert the registered set.
func NewServer(app core.App, cfg *config.Config) (*mcp.Server, error) {
	return newServerForTools(app, cfg, toolcore.ExposedTools(cfg))
}

// newServerForTools builds an MCP server registering exactly the tools in names (which the caller
// has already gated). Both NewServer (full exposed set) and Serve (optionally module-filtered set)
// call it, so the registration loop lives in ONE place regardless of which gate produced names.
func newServerForTools(app core.App, cfg *config.Config, names []string) (*mcp.Server, error) {
	s := mcp.NewServer(&mcp.Implementation{Name: serverName, Version: serverVersion}, nil)
	for _, name := range names {
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

	// Module gate (MCP_MODULES): a shared MCP mount narrows the exposed set to specific modules —
	// the pm-only mount declares MCP_MODULES=pm so it carries the twelve PM tools and none of the
	// librarian ride-alongs. resolveModuleGate encodes the three cases (unset = all, set-but-empty
	// = fail, unresolvable = fail); a failure is surfaced here as a direct os.Exit(1) — never a
	// returned error — mirroring the requireResolvedConfig precedent above, because PocketBase's
	// Start()/serve goroutine discards RunE errors, so a returned error would serve silently.
	raw, declared := os.LookupEnv("MCP_MODULES")
	names, modules, failReason, ok := resolveModuleGate(cfg, raw, declared)
	if !ok {
		fmt.Fprintf(os.Stderr, "deskkit mcp-serve: %s\n", failReason)
		os.Exit(1)
	}

	s, err := newServerForTools(app, cfg, names)
	if err != nil {
		return err
	}
	// Mount signal: emit ONE concise line so a host operator can SEE the surface came up (which
	// module set gated it and which tools it carries) instead of guessing at a silent absence. It
	// MUST go to stderr — stdout is the JSON-RPC channel and any stray byte there corrupts the
	// protocol.
	emitMountSignal(os.Stderr, modules, names)
	if rerr := s.Run(ctx, &mcp.StdioTransport{}); rerr != nil && !isShutdownEOF(rerr) {
		return rerr
	}
	return nil
}

// resolveModuleGate applies the MCP_MODULES gate to cfg's exposed tool set and returns the tool
// NAME list to register plus the module-set label for the mount signal. The three cases are kept
// DISTINCT — collapsing them is the design trap this gate exists to avoid:
//
//   - raw UNSET (declared == false): no module filter — every exposed tool (today's behavior,
//     unchanged); modules is nil (the signal renders "all"). This preserves the 5/6/17/18 counts.
//   - raw SET but resolving to zero declared module names (e.g. "" or " , " — after splitting on
//     comma and trimming, nothing remains): FAIL LOUD. ok == false, failReason names the empty
//     declaration. Do NOT fall back to "all" — an explicit-but-empty declaration is an operator
//     error, and silently exposing everything would defeat the mount's intent.
//   - raw SET, non-empty declared set: filter ExposedSpecs by module via SelectByModules. If the
//     FILTERED result is empty (unresolvable — a typo'd name, or a module not registered/enabled
//     on this desk, e.g. MCP_MODULES=pm without PM_ENABLED): FAIL LOUD. A partially-matching set
//     (e.g. "librarian,bogus" where librarian matches) yields a non-empty result and serves.
func resolveModuleGate(cfg *config.Config, raw string, declared bool) (names, modules []string, failReason string, ok bool) {
	if !declared {
		return toolcore.ExposedTools(cfg), nil, "", true
	}
	var mods []string
	for _, part := range strings.Split(raw, ",") {
		if m := strings.TrimSpace(part); m != "" {
			mods = append(mods, m)
		}
	}
	if len(mods) == 0 {
		return nil, nil, fmt.Sprintf(
			"MCP_MODULES is set to %q but names no module after splitting on comma and trimming; "+
				"set it to a comma-separated module list (e.g. \"pm\" or \"librarian,pm\") or unset it to expose every module",
			raw), false
	}
	names = toolcore.ToolNames(toolcore.SelectByModules(toolcore.ExposedSpecs(cfg), mods...))
	if len(names) == 0 {
		// mods is deliberately non-nil on this path (unlike the empty-declaration failure
		// above): the declaration DID name modules, and a caller inspecting a !ok result can
		// surface what was declared.
		return nil, mods, fmt.Sprintf(
			"MCP_MODULES=%q resolved to no exposed tools on this desk; none of those modules are registered/enabled here — "+
				"check the module name(s) and that the owning module is enabled (e.g. set PM_ENABLED=true for \"pm\")",
			raw), false
	}
	return names, mods, "", true
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

// emitMountSignal writes ONE concise mount line to w: the server identity, the gated MODULE SET,
// and the exact tool set it exposed. Presence of this line in the host's server log is the
// observable "the surface mounted" signal (its absence, a diagnostic); naming the module set makes
// a module-gated mount (e.g. desk-pm's MCP_MODULES=pm) legible at a glance. modules is the declared
// set (nil when MCP_MODULES is unset → rendered "all"). Callers MUST pass a stderr writer — never
// stdout, the JSON-RPC channel.
func emitMountSignal(w io.Writer, modules []string, tools []string) {
	moduleSet := "all"
	if len(modules) > 0 {
		moduleSet = strings.Join(modules, ",")
	}
	fmt.Fprintf(w, "deskkit mcp-serve: mounted %q %s; modules: %s; %d tool(s) exposed: %s\n",
		serverName, serverVersion, moduleSet, len(tools), strings.Join(tools, ", "))
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

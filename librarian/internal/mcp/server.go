// Package mcp is the OUTBOUND MCP-server slice (build-brief §5, punch-list item 4;
// spec §7.2 dual-surface, added 2026-07-16). It exposes the librarian's frozen six-tool
// core (internal/tools) as an MCP stdio server — the librarian's "hands" — so a Claude Code
// or OpenCode session (or the dual-format plugin's plugin/mcp boundary) can call the
// librarian's tools directly. This is the one-binary "MCP server + CLI over a single tool
// core" pattern; the CLI (cmd/pocket-librarian) and the eino agent loop (internal/agent)
// remain the other two surfaces over the SAME tool core.
//
// Three invariants this slice enforces (all load-bearing):
//
//  1. Zero logic duplication (spec §2.6). Every MCP tool handler calls the SAME tools.*
//     function the CLI and the eino agent call. Each tool's parameter schema is derived by
//     reflection over the SAME input struct (its tags) used everywhere else — one source of
//     truth, no re-implemented behavior.
//
//  2. The §5.4 write gate + §5.5 restore exclusion. The MCP surface is model-facing exactly
//     like the eino loop, so it uses the identical registration-time gate: the exposed set is
//     tools.AgentTools(cfg) — {sweep, patrol, propose_fix, query} always, plus apply_fix ONLY
//     when LIBRARIAN_AUTONOMOUS_WRITES=true. restore is NEVER exposed over MCP (recovery is a
//     supervised CLI action, §5.5); its exclusion is STRUCTURAL — there is no MCP registrar
//     for restore AND ExposedTools filters it defensively — not a runtime check inside a tool.
//
//  3. Explicit input schemas (not the SDK's struct-tag inference). The frozen input structs in
//     internal/tools/types.go carry eino/invopop-style `jsonschema:"description=…"` KEY-VALUE
//     tags. The official MCP SDK's inference (google/jsonschema-go) treats a `jsonschema` tag
//     as PLAIN TEXT and rejects any tag beginning with `WORD=` — so it cannot infer from these
//     structs. eino (the live agent loop) depends on that same key-value grammar, and types.go
//     is frozen, so the structs cannot satisfy both parsers by tags alone. We therefore build
//     each tool's *jsonschema.Schema ourselves (buildInputSchema) — reflecting over the same
//     struct, reading the `json` tag for names/optionality and the eino `description=` value
//     for docs — and set mcp.Tool.InputSchema, which makes AddTool skip its own inference.
package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/pocketbase/pocketbase/core"

	"github.com/example/pocket-librarian/internal/config"
	"github.com/example/pocket-librarian/internal/tools"
)

// serverName / serverVersion identify the MCP server in the initialize handshake.
// Identity-neutral: no person/org/repo/issue (0013 item 9). The module path is already
// github.com/example/pocket-librarian; this keeps the wire identity neutral too.
const (
	serverName    = "pocket-librarian"
	serverVersion = "v1"
)

// toolInputTypes is the canonical map from an exposable tool name to the input struct the
// parameter schema is reflected from. restore is DELIBERATELY ABSENT — it is never exposed
// over MCP (§5.5). NewServer requires every ExposedTools name to have an entry here (and a
// case in register), so a drift between this map and the registrar switch fails loud at
// construction instead of silently shipping an unschematized or missing tool.
var toolInputTypes = map[string]reflect.Type{
	"sweep":       reflect.TypeOf(tools.SweepInput{}),
	"patrol":      reflect.TypeOf(tools.PatrolInput{}),
	"propose_fix": reflect.TypeOf(tools.ProposeFixInput{}),
	"apply_fix":   reflect.TypeOf(tools.ApplyFixInput{}),
	"query":       reflect.TypeOf(tools.QueryInput{}),
}

// ExposedTools returns the tool names this server registers for cfg: the §5.4 gate applied
// (tools.AgentTools) with restore excluded (§5.5). This is the SINGLE source both the server
// registration loop and the tests consult, so the test verifies the real logic. Because
// tools.AgentTools already never returns restore, the explicit skip is defense-in-depth: a
// future change that (wrongly) added restore to the agent set would still not reach MCP.
func ExposedTools(cfg *config.Config) []string {
	var names []string
	for _, spec := range tools.AgentTools(cfg) {
		if spec.Name == "restore" {
			continue
		}
		names = append(names, spec.Name)
	}
	return names
}

// NewServer builds the MCP server with exactly the gated tool set for cfg. app is used only
// inside the tool handlers (invoked per request), not during registration, so it may be nil
// in tests that only assert the registered set.
func NewServer(app core.App, cfg *config.Config) (*mcp.Server, error) {
	s := mcp.NewServer(&mcp.Implementation{Name: serverName, Version: serverVersion}, nil)
	for _, name := range ExposedTools(cfg) {
		if _, ok := toolInputTypes[name]; !ok {
			return nil, fmt.Errorf(
				"mcp: exposed tool %q has no input-type entry (registrar/type-map drift)", name)
		}
		if err := register(s, name, app, cfg); err != nil {
			return nil, err
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
// the message must be matched directly. Scoped to the "server is closing" sentinel so a
// genuine mid-session transport failure still surfaces.
func isShutdownEOF(err error) bool {
	if errors.Is(err, io.EOF) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	return strings.Contains(err.Error(), "server is closing")
}

// register wires one tool onto the server. The tool-level Description comes from the frozen
// registry (tools.Spec), matching the eino loop; the parameter schema is built explicitly by
// buildInputSchema from the concrete input struct and passed via mcp.Tool.InputSchema (so the
// SDK does not attempt its own — incompatible — struct-tag inference). restore (and any
// non-exposable name) has no toolInputTypes entry and is refused here by design (§5.5).
func register(s *mcp.Server, name string, app core.App, cfg *config.Config) error {
	spec, ok := tools.Spec(name)
	if !ok {
		return fmt.Errorf("mcp: unknown tool %q (not in the frozen registry)", name)
	}
	in, ok := toolInputTypes[name]
	if !ok {
		return fmt.Errorf("mcp: no outbound registrar for tool %q (restore is CLI-only, §5.5)", name)
	}
	schema, err := buildInputSchema(in)
	if err != nil {
		return fmt.Errorf("mcp: build input schema for %q: %w", name, err)
	}
	t := &mcp.Tool{Name: name, Description: spec.Description, InputSchema: schema}
	switch name {
	case "sweep":
		mcp.AddTool(s, t, func(ctx context.Context, _ *mcp.CallToolRequest, in tools.SweepInput) (*mcp.CallToolResult, any, error) {
			return jsonContent(tools.Sweep(ctx, app, cfg, &in))
		})
	case "patrol":
		mcp.AddTool(s, t, func(ctx context.Context, _ *mcp.CallToolRequest, in tools.PatrolInput) (*mcp.CallToolResult, any, error) {
			return jsonContent(tools.Patrol(ctx, app, cfg, &in))
		})
	case "propose_fix":
		mcp.AddTool(s, t, func(ctx context.Context, _ *mcp.CallToolRequest, in tools.ProposeFixInput) (*mcp.CallToolResult, any, error) {
			return jsonContent(tools.ProposeFix(ctx, app, cfg, &in))
		})
	case "apply_fix":
		mcp.AddTool(s, t, func(ctx context.Context, _ *mcp.CallToolRequest, in tools.ApplyFixInput) (*mcp.CallToolResult, any, error) {
			return jsonContent(tools.ApplyFix(ctx, app, cfg, &in))
		})
	case "query":
		mcp.AddTool(s, t, func(ctx context.Context, _ *mcp.CallToolRequest, in tools.QueryInput) (*mcp.CallToolResult, any, error) {
			// tools.Query already returns a JSON document (json.RawMessage); json.Marshal of a
			// RawMessage is the bytes verbatim, so jsonContent forwards it unchanged.
			return jsonContent(tools.Query(ctx, app, cfg, &in))
		})
	default:
		return fmt.Errorf("mcp: no outbound registrar for tool %q (restore is CLI-only, §5.5)", name)
	}
	return nil
}

// jsonContent adapts a tools.* return (a typed result or json.RawMessage, plus an error) into
// an MCP CallToolResult carrying the JSON payload as text — the same JSON the eino loop hands
// the model (spec §2.6). A tool-execution error is surfaced as an MCP error RESULT
// (IsError=true) so the model sees the failure text and can react, rather than a protocol
// error; malformed-request errors are handled by the SDK before the handler runs.
func jsonContent(v any, err error) (*mcp.CallToolResult, any, error) {
	if err != nil {
		return errorResult(err.Error()), nil, nil
	}
	b, mErr := json.Marshal(v)
	if mErr != nil {
		return errorResult("marshal result: " + mErr.Error()), nil, nil
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(b)}},
	}, nil, nil
}

func errorResult(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: msg}},
	}
}

// buildInputSchema constructs a tool's parameter schema from its input struct and returns it
// as the SDK's *jsonschema.Schema. It builds a plain JSON-Schema map first (inputSchemaMap)
// and round-trips it through the SDK's own unmarshaler, so this code depends only on the
// standard JSON-Schema shape, not on the SDK's Go struct layout.
func buildInputSchema(t reflect.Type) (*jsonschema.Schema, error) {
	b, err := json.Marshal(inputSchemaMap(t))
	if err != nil {
		return nil, err
	}
	s := new(jsonschema.Schema)
	if err := json.Unmarshal(b, s); err != nil {
		return nil, err
	}
	return s, nil
}

// inputSchemaMap reflects a tool's input struct into a plain JSON-Schema object map. Property
// names + optionality come from the `json` tag (a non-omitempty field is `required`, mirroring
// the SDK's own rule and matching QueryInput.Kind's `;required` marker); property descriptions
// come from the eino `description=` tag value. This is the single-source-of-truth adapter that
// lets the frozen structs (which carry eino's key-value grammar) drive the MCP schema without
// the SDK's incompatible inference and without duplicating the field definitions.
func inputSchemaMap(t reflect.Type) map[string]any {
	props := map[string]any{}
	var required []string
	if t.Kind() == reflect.Struct {
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if f.PkgPath != "" { // unexported (incl. the blank `_` field on SweepInput)
				continue
			}
			name, omitempty := parseJSONTag(f.Tag.Get("json"))
			if name == "-" || name == "" {
				continue
			}
			p := goTypeSchema(f.Type)
			if p == nil {
				continue // unsupported kind; skip rather than emit a malformed property
			}
			if desc := parseTagDescription(f.Tag.Get("jsonschema")); desc != "" {
				p["description"] = desc
			}
			props[name] = p
			if !omitempty {
				required = append(required, name)
			}
		}
	}
	sort.Strings(required)
	schema := map[string]any{"type": "object", "properties": props}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

// goTypeSchema maps a Go field type to its JSON-Schema fragment. Covers the kinds the frozen
// input structs actually use (string, int, bool, []string); returns nil for anything else.
func goTypeSchema(ft reflect.Type) map[string]any {
	switch ft.Kind() {
	case reflect.String:
		return map[string]any{"type": "string"}
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return map[string]any{"type": "integer"}
	case reflect.Float32, reflect.Float64:
		return map[string]any{"type": "number"}
	case reflect.Slice, reflect.Array:
		if elem := goTypeSchema(ft.Elem()); elem != nil {
			return map[string]any{"type": "array", "items": elem}
		}
		return map[string]any{"type": "array"}
	default:
		return nil
	}
}

// parseJSONTag returns the property name and whether the field carries `omitempty`.
func parseJSONTag(tag string) (name string, omitempty bool) {
	if tag == "" {
		return "", false
	}
	parts := strings.Split(tag, ",")
	name = parts[0]
	for _, o := range parts[1:] {
		if o == "omitempty" {
			omitempty = true
		}
	}
	return name, omitempty
}

// parseTagDescription extracts the description text from an eino `jsonschema` struct tag.
//
// The frozen input structs (§5.1 types.go) use the eino/invopop key-value grammar, but their
// description VALUES contain commas and semicolons as natural text — e.g. "defaults to
// R1,R2,R3" and "Relative path to patrol; empty means whole desk". So we must NOT split on
// those separators (doing so would truncate the description). We take everything after
// `description=` and trim only the one structured trailing keyword actually used in these
// structs — a `required` marker (QueryInput.Kind: "…adoption;required"). Required-ness itself
// is derived from `omitempty`, not from this marker, so trimming it here is purely cosmetic.
func parseTagDescription(tag string) string {
	const key = "description="
	idx := strings.Index(tag, key)
	if idx < 0 {
		return ""
	}
	desc := tag[idx+len(key):]
	for _, kw := range []string{";required", ",required", "; required", ", required"} {
		desc = strings.TrimSuffix(desc, kw)
	}
	return strings.TrimSpace(desc)
}

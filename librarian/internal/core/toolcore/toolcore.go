// Package toolcore is the harness-agnostic tool registry + gate + schema-builder shared by
// every surface (the eino agent loop, the MCP stdio server, and — via each module's Specs —
// the CLI). It generalizes what were the librarian-only internal/tools/registry.go
// (ToolSpec/Registry/AgentTools/Spec/ToolNames) and internal/mcp/server.go's schema builder
// (buildInputSchema and friends) so a SECOND module's tools register on every surface without
// editing agent or mcp code: the surfaces loop over this package's merged registry, populated
// from the enabled module set at startup.
//
// The eino InferTool schema and the MCP buildInputSchema output are produced by the SAME code
// that produced them before the refactor (moved verbatim), so the model-facing schemas stay
// byte-identical.
package toolcore

import (
	"context"
	"encoding/json"
	"reflect"
	"sort"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	toolutils "github.com/cloudwego/eino/components/tool/utils"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/pocketbase/pocketbase/core"

	"github.com/example/pocket-librarian/internal/core/config"
)

// ToolSpec is the harness-agnostic descriptor for one tool. It carries the same fields the
// librarian's frozen registry did (Name/Description + the §5.4 gate flags), plus the Module
// that owns it and two closures that build the concrete eino InvokableTool / register the tool
// on an MCP server. The closures capture the tool's concrete input type, so the surfaces stay
// type-generic while each spec stays type-specific.
type ToolSpec struct {
	Module      string
	Name        string
	Description string
	InputType   reflect.Type
	// WritesFiles is true for the tools that mutate the desk tree (apply_fix, restore).
	WritesFiles bool
	// AgentDefault: included in the autonomous serve agent's tool set unconditionally.
	AgentDefault bool
	// AgentGated: included in the autonomous serve agent's set ONLY when AutonomousWrites is
	// set (the §5.4 registration-time gate). Applies to apply_fix.
	AgentGated bool
	// newEino builds the eino InvokableTool (InferTool over the concrete input type).
	newEino func(app core.App, cfg *config.Config) (tool.InvokableTool, error)
	// registerMCP registers the tool on an mcp.Server (typed handler + explicit InputSchema).
	registerMCP func(s *mcp.Server, app core.App, cfg *config.Config) error
}

// New builds a ToolSpec for input type I. invoke unmarshals into *I and returns the typed
// result (or a json.RawMessage) — the same value the eino path marshals and the mcp path wraps,
// so both surfaces stay byte-identical to the pre-refactor code.
func New[I any](module, name, description string, writesFiles, agentDefault, agentGated bool,
	invoke func(ctx context.Context, app core.App, cfg *config.Config, in *I) (any, error)) ToolSpec {
	var zero I
	return ToolSpec{
		Module: module, Name: name, Description: description,
		InputType:   reflect.TypeOf(zero),
		WritesFiles: writesFiles, AgentDefault: agentDefault, AgentGated: agentGated,
		newEino: func(app core.App, cfg *config.Config) (tool.InvokableTool, error) {
			return toolutils.InferTool(name, description, func(ctx context.Context, in I) (string, error) {
				r, err := invoke(ctx, app, cfg, &in)
				if err != nil {
					return "", err
				}
				b, merr := json.Marshal(r) // json.Marshal(json.RawMessage) == verbatim bytes
				if merr != nil {
					return "", merr
				}
				return string(b), nil
			})
		},
		registerMCP: func(s *mcp.Server, app core.App, cfg *config.Config) error {
			schema, err := buildInputSchema(reflect.TypeOf(zero))
			if err != nil {
				return err
			}
			t := &mcp.Tool{Name: name, Description: description, InputSchema: schema}
			mcp.AddTool(s, t, func(ctx context.Context, _ *mcp.CallToolRequest, in I) (*mcp.CallToolResult, any, error) {
				return jsonContent(invoke(ctx, app, cfg, &in))
			})
			return nil
		},
	}
}

// NewEinoTool builds the eino InvokableTool for this spec (used by the agent loop).
func (s ToolSpec) NewEinoTool(app core.App, cfg *config.Config) (tool.InvokableTool, error) {
	return s.newEino(app, cfg)
}

// RegisterMCP registers this spec's tool onto an MCP server (used by the mcp surface).
func (s ToolSpec) RegisterMCP(srv *mcp.Server, app core.App, cfg *config.Config) error {
	return s.registerMCP(srv, app, cfg)
}

// registered is the process-global merged tool set, populated from the enabled module set at
// startup (module.Register calls Register(mod.Tools()...) for each enabled module). It mirrors
// the old package-global tools.Registry style.
var registered []ToolSpec

// Register appends specs to the merged registry.
func Register(specs ...ToolSpec) { registered = append(registered, specs...) }

// Reset clears the registry (for tests).
func Reset() { registered = nil }

// AllTools returns a copy of the full merged tool set — the CLI/supervised surface builds from
// each module's Specs() directly, but this exposes the merged view.
func AllTools() []ToolSpec {
	out := make([]ToolSpec, len(registered))
	copy(out, registered)
	return out
}

// Spec returns the ToolSpec for name from the merged registry.
func Spec(name string) (ToolSpec, bool) {
	for _, t := range registered {
		if t.Name == name {
			return t, true
		}
	}
	return ToolSpec{}, false
}

// AgentTools applies the §5.4 registration-time gate (identical logic to the old
// tools.AgentTools): the autonomous serve agent gets every AgentDefault tool always, plus each
// AgentGated tool ONLY when cfg.AutonomousWrites is set. restore is neither, so it is never
// returned — the gate is enforced by EXCLUSION FROM THE SLICE.
func AgentTools(cfg *config.Config) []ToolSpec {
	autonomous := cfg != nil && cfg.AutonomousWrites
	var out []ToolSpec
	for _, t := range registered {
		if t.AgentDefault || (t.AgentGated && autonomous) {
			out = append(out, t)
		}
	}
	return out
}

// ExposedTools mirrors the old mcp.ExposedTools: AgentTools(cfg) with restore excluded
// defensively (§5.5). Because AgentTools already never returns restore, the explicit skip is
// defense-in-depth.
func ExposedTools(cfg *config.Config) []string {
	var names []string
	for _, s := range AgentTools(cfg) {
		if s.Name == "restore" {
			continue
		}
		names = append(names, s.Name)
	}
	return names
}

// ToolNames maps a []ToolSpec to its names (convenience for the loop/MCP slices).
func ToolNames(specs []ToolSpec) []string {
	names := make([]string, len(specs))
	for i, s := range specs {
		names[i] = s.Name
	}
	return names
}

// --- schema builder (moved VERBATIM from internal/mcp/server.go; §2.6 single source) ---
//
// The frozen input structs carry eino/invopop-style `jsonschema:"description=…"` key-value tags
// the official MCP SDK cannot infer from, so we build each tool's *jsonschema.Schema ourselves —
// reflecting over the same struct, reading the `json` tag for names/optionality and the eino
// `description=` value for docs — and set mcp.Tool.InputSchema, which makes AddTool skip its own
// inference. This is exactly the code that produced the pre-refactor schemas.

// jsonContent adapts a tool's return (a typed result or json.RawMessage, plus an error) into an
// MCP CallToolResult carrying the JSON payload as text — the same JSON the eino loop hands the
// model (spec §2.6). A tool-execution error is surfaced as an MCP error RESULT (IsError=true).
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

// buildInputSchema constructs a tool's parameter schema from its input struct and returns it as
// the SDK's *jsonschema.Schema. It builds a plain JSON-Schema map first (inputSchemaMap) and
// round-trips it through the SDK's own unmarshaler, so this code depends only on the standard
// JSON-Schema shape, not on the SDK's Go struct layout.
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

// BuildInputSchema is the exported wrapper the schema round-trip test consults.
func BuildInputSchema(t reflect.Type) (*jsonschema.Schema, error) { return buildInputSchema(t) }

// SchemaForType is the exported wrapper the schema-map test consults (it reflects a tool's input
// struct into a plain JSON-Schema object map).
func SchemaForType(t reflect.Type) map[string]any { return inputSchemaMap(t) }

// inputSchemaMap reflects a tool's input struct into a plain JSON-Schema object map. Property
// names + optionality come from the `json` tag (a non-omitempty field is `required`); property
// descriptions come from the eino `description=` tag value.
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

// parseTagDescription extracts the description text from an eino `jsonschema` struct tag. The
// frozen input structs use the eino/invopop key-value grammar, but their description VALUES
// contain commas and semicolons as natural text, so we must NOT split on those separators. We
// take everything after `description=` and trim only the one structured trailing keyword used in
// these structs — a `required` marker (QueryInput.Kind). Required-ness itself is derived from
// `omitempty`, not from this marker, so trimming it here is purely cosmetic.
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

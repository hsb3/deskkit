package mcp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"testing"

	"github.com/example/pocket-librarian/internal/config"
)

// TestIsShutdownEOF pins the clean-shutdown classifier: a stdio client closing stdin after its
// last request must exit 0/silent, not surface as a cobra "Error: ... EOF" + usage dump. The
// SDK's jsonrpc2 error wraps its ErrServerClosing sentinel via %w and appends io.EOF via %v,
// so errors.Is(err, io.EOF) does NOT match it — the "server is closing" message is matched
// directly. A genuine mid-session failure must still be reported (return false).
func TestIsShutdownEOF(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"plain io.EOF", io.EOF, true},
		{"wrapped io.EOF", fmt.Errorf("read: %w", io.EOF), true},
		{"context canceled", context.Canceled, true},
		{"context deadline", context.DeadlineExceeded, true},
		// The exact shape the SDK returns on stdin close: ErrServerClosing (%w) + io.EOF (%v).
		{"server is closing: EOF", fmt.Errorf("%w: %v", errors.New("server is closing"), io.EOF), true},
		{"real transport failure", errors.New("write: broken pipe"), false},
		{"real decode error", errors.New("invalid JSON-RPC frame"), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isShutdownEOF(tc.err); got != tc.want {
				t.Errorf("isShutdownEOF(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

// TestExposedTools_GateComposition is the load-bearing test: the MCP surface applies the same
// §5.4 registration-time write gate as the eino loop, and never exposes restore (§5.5).
//   - default (LIBRARIAN_AUTONOMOUS_WRITES unset): {sweep, patrol, propose_fix, query} — 4
//   - autonomous writes on: adds apply_fix only — 5
//   - restore never present in either state
func TestExposedTools_GateComposition(t *testing.T) {
	cases := []struct {
		name       string
		autonomous bool
		wantApply  bool
		wantCount  int
	}{
		{"default (no autonomous writes)", false, false, 4},
		{"autonomous writes on", true, true, 5},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &config.Config{AutonomousWrites: tc.autonomous}
			got := map[string]bool{}
			for _, n := range ExposedTools(cfg) {
				got[n] = true
			}
			for _, always := range []string{"sweep", "patrol", "propose_fix", "query"} {
				if !got[always] {
					t.Errorf("autonomous=%v: expected %q exposed over MCP", tc.autonomous, always)
				}
			}
			if got["restore"] {
				t.Errorf("autonomous=%v: restore must NEVER be exposed over MCP (§5.5)", tc.autonomous)
			}
			if got["apply_fix"] != tc.wantApply {
				t.Errorf("autonomous=%v: apply_fix exposed=%v, want %v (§5.4 gate)",
					tc.autonomous, got["apply_fix"], tc.wantApply)
			}
			if len(got) != tc.wantCount {
				t.Errorf("autonomous=%v: exposed %d tools, want %d", tc.autonomous, len(got), tc.wantCount)
			}
		})
	}
}

// TestNewServer_BuildsForBothGates exercises the real SDK registration path for each exposed
// tool in both gate states. NewServer calls mcp.AddTool with an EXPLICIT input schema for
// every tool (mcp.Tool.InputSchema), so the SDK skips its own struct-tag inference — the path
// that panicked on eino's key-value `jsonschema` tags. A registrar / type-map drift or a
// schema-build failure surfaces here as an error. app is nil because handlers are not invoked
// during registration.
func TestNewServer_BuildsForBothGates(t *testing.T) {
	for _, autonomous := range []bool{false, true} {
		cfg := &config.Config{AutonomousWrites: autonomous}
		s, err := NewServer(nil, cfg)
		if err != nil {
			t.Fatalf("autonomous=%v: NewServer error: %v", autonomous, err)
		}
		if s == nil {
			t.Fatalf("autonomous=%v: NewServer returned nil server", autonomous)
		}
	}
}

// TestInputSchemaMap_MatchesStructs asserts the explicit schema adapter (inputSchemaMap) — the
// single-source-of-truth reflection over the frozen input structs — produces, for each exposed
// tool: the right property set, a JSON type + description on every property, and the right
// required set (non-omitempty fields; QueryInput.Kind is the only required one). restore must
// have no entry (never exposed, §5.5). This replaces the earlier SDK-inference assertion, which
// is no longer used (the SDK cannot parse eino's key-value tags).
func TestInputSchemaMap_MatchesStructs(t *testing.T) {
	wantProps := map[string][]string{
		"sweep":       {}, // parameterless: SweepInput has only an ignored (`json:"-"`) field
		"patrol":      {"path"},
		"propose_fix": {"run_id", "rules"},
		"apply_fix":   {"run_id", "revision_ids"},
		"query":       {"kind", "days"},
	}
	wantRequired := map[string][]string{
		"query": {"kind"}, // only non-omitempty exposed field; all others are omitempty
	}
	for name, want := range wantProps {
		typ, ok := toolInputTypes[name]
		if !ok {
			t.Errorf("%s: missing input-type entry", name)
			continue
		}
		m := inputSchemaMap(typ)
		if m["type"] != "object" {
			t.Errorf("%s: schema type = %v, want object", name, m["type"])
		}
		props, _ := m["properties"].(map[string]any)
		if !equalSet(keysOf(props), want) {
			t.Errorf("%s: properties = %v, want %v", name, keysOf(props), want)
		}
		for pn, pv := range props {
			p, _ := pv.(map[string]any)
			if p["type"] == nil || p["type"] == "" {
				t.Errorf("%s.%s: expected a JSON type", name, pn)
			}
			if d, _ := p["description"].(string); d == "" {
				t.Errorf("%s.%s: expected a non-empty description from the eino tag", name, pn)
			}
		}
		var gotReq []string
		if r, ok := m["required"].([]string); ok {
			gotReq = r
		}
		if !equalSet(gotReq, wantRequired[name]) {
			t.Errorf("%s: required = %v, want %v", name, gotReq, wantRequired[name])
		}
	}
	if _, ok := toolInputTypes["restore"]; ok {
		t.Error("restore must not appear in toolInputTypes — it is never exposed over MCP (§5.5)")
	}
}

// TestParseTagDescription_NoTruncation guards the two subtle cases in the frozen tags: a
// description whose text contains a comma ("R1,R2,R3") must not be split, and the `;required`
// marker on QueryInput.Kind must be trimmed off the description.
func TestParseTagDescription_NoTruncation(t *testing.T) {
	rules := inputSchemaMap(toolInputTypes["propose_fix"])["properties"].(map[string]any)["rules"].(map[string]any)
	if d, _ := rules["description"].(string); !strings.Contains(d, "R1,R2,R3") {
		t.Errorf("propose_fix.rules description was truncated at a comma: %q", d)
	}
	kind := inputSchemaMap(toolInputTypes["query"])["properties"].(map[string]any)["kind"].(map[string]any)
	d, _ := kind["description"].(string)
	if strings.Contains(d, "required") {
		t.Errorf("query.kind description leaked the ';required' marker: %q", d)
	}
	if !strings.Contains(d, "live_files") {
		t.Errorf("query.kind description looks truncated: %q", d)
	}
}

// TestBuildInputSchema_RoundTrips confirms the SDK schema type accepts our explicit schema for
// every exposed tool (the map → *jsonschema.Schema conversion register() relies on).
func TestBuildInputSchema_RoundTrips(t *testing.T) {
	for name, typ := range toolInputTypes {
		if _, err := buildInputSchema(typ); err != nil {
			t.Errorf("%s: buildInputSchema error: %v", name, err)
		}
	}
}

func keysOf(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func equalSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	ac := append([]string(nil), a...)
	bc := append([]string(nil), b...)
	sort.Strings(ac)
	sort.Strings(bc)
	for i := range ac {
		if ac[i] != bc[i] {
			return false
		}
	}
	return true
}

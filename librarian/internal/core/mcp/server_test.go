package mcp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/example/pocket-librarian/internal/core/config"
	"github.com/example/pocket-librarian/internal/core/toolcore"
	"github.com/example/pocket-librarian/internal/modules/librarian/tools"
)

// TestMain populates the shared toolcore registry with the librarian's specs — exactly what
// module.Register does at startup — so the gate-composition, server-build, and schema tests
// below exercise the real merged registry. The schema builder + gate logic that used to live
// in this package now live in toolcore (post-refactor); these tests source input types and
// schemas from there while asserting the identical expected values as before.
func TestMain(m *testing.M) {
	toolcore.Reset()
	toolcore.Register(tools.Specs()...)
	os.Exit(m.Run())
}

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
//   - default (LIBRARIAN_AUTONOMOUS_WRITES unset): {sweep, patrol, propose_fix, query,
//     record_feedback} — 5 (record_feedback is a DB-only write, ungated like the read tools)
//   - autonomous writes on: adds apply_fix only — 6
//   - restore never present in either state
func TestExposedTools_GateComposition(t *testing.T) {
	cases := []struct {
		name       string
		autonomous bool
		wantApply  bool
		wantCount  int
	}{
		{"default (no autonomous writes)", false, false, 5},
		{"autonomous writes on", true, true, 6},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &config.Config{AutonomousWrites: tc.autonomous}
			got := map[string]bool{}
			for _, n := range toolcore.ExposedTools(cfg) {
				got[n] = true
			}
			for _, always := range []string{"sweep", "patrol", "propose_fix", "query", "record_feedback"} {
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
// tool in both gate states. NewServer builds each tool with an EXPLICIT input schema
// (mcp.Tool.InputSchema), so the SDK skips its own struct-tag inference — the path that panicked
// on eino's key-value `jsonschema` tags. A registry/schema-build failure surfaces here as an
// error. app is nil because handlers are not invoked during registration.
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

// TestInputSchemaMap_MatchesStructs asserts the explicit schema adapter (toolcore.SchemaForType,
// the moved-verbatim inputSchemaMap) — the single-source-of-truth reflection over the frozen
// input structs — produces, for each exposed tool: the right property set, a JSON type +
// description on every property, and the right required set (non-omitempty fields; QueryInput.Kind
// is the only required one). restore must be excluded from the exposed set (never exposed, §5.5).
func TestInputSchemaMap_MatchesStructs(t *testing.T) {
	wantProps := map[string][]string{
		"sweep":           {}, // parameterless: SweepInput has only an ignored (`json:"-"`) field
		"patrol":          {"path"},
		"propose_fix":     {"run_id", "rules"},
		"apply_fix":       {"run_id", "revision_ids"},
		"query":           {"kind", "days"},
		"record_feedback": {"kind", "summary", "detail", "context", "source"},
	}
	wantRequired := map[string][]string{
		"query":           {"kind"},            // only non-omitempty exposed field; all others are omitempty
		"record_feedback": {"kind", "summary"}, // the two non-omitempty fields; detail/context/source are omitempty
	}
	for name, want := range wantProps {
		spec, ok := toolcore.Spec(name)
		if !ok {
			t.Errorf("%s: missing registry entry", name)
			continue
		}
		m := toolcore.SchemaForType(spec.InputType)
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
	// restore IS a registered librarian spec (it carries an InputType), but must NEVER be exposed
	// over MCP (§5.5) — assert its exclusion from the exposed set (the type map that used to
	// enforce this by omission no longer exists post-refactor).
	exposed := map[string]bool{}
	for _, n := range toolcore.ExposedTools(&config.Config{AutonomousWrites: true}) {
		exposed[n] = true
	}
	if exposed["restore"] {
		t.Error("restore must never be exposed over MCP (§5.5)")
	}
}

// TestParseTagDescription_NoTruncation guards the two subtle cases in the frozen tags: a
// description whose text contains a comma ("R1,R2,R3") must not be split, and the `;required`
// marker on QueryInput.Kind must be trimmed off the description.
func TestParseTagDescription_NoTruncation(t *testing.T) {
	proposeSpec, _ := toolcore.Spec("propose_fix")
	rules := toolcore.SchemaForType(proposeSpec.InputType)["properties"].(map[string]any)["rules"].(map[string]any)
	if d, _ := rules["description"].(string); !strings.Contains(d, "R1,R2,R3") {
		t.Errorf("propose_fix.rules description was truncated at a comma: %q", d)
	}
	querySpec, _ := toolcore.Spec("query")
	kind := toolcore.SchemaForType(querySpec.InputType)["properties"].(map[string]any)["kind"].(map[string]any)
	d, _ := kind["description"].(string)
	if strings.Contains(d, "required") {
		t.Errorf("query.kind description leaked the ';required' marker: %q", d)
	}
	if !strings.Contains(d, "live_files") {
		t.Errorf("query.kind description looks truncated: %q", d)
	}
}

// TestBuildInputSchema_RoundTrips confirms the SDK schema type accepts our explicit schema for
// every exposed tool (the map → *jsonschema.Schema conversion NewServer relies on).
func TestBuildInputSchema_RoundTrips(t *testing.T) {
	for _, name := range toolcore.ExposedTools(&config.Config{AutonomousWrites: true}) {
		spec, ok := toolcore.Spec(name)
		if !ok {
			t.Fatalf("%s: not registered", name)
		}
		if _, err := toolcore.BuildInputSchema(spec.InputType); err != nil {
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

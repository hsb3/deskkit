package mcp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hsb3/desk-standard/librarian/internal/core/config"
	coreschema "github.com/hsb3/desk-standard/librarian/internal/core/schema"
	"github.com/hsb3/desk-standard/librarian/internal/core/toolcore"
	"github.com/hsb3/desk-standard/librarian/internal/modules/librarian/tools"
	pmtools "github.com/hsb3/desk-standard/librarian/internal/modules/pm/tools"
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
		"query":           {"kind", "days", "include_disposed", "term", "limit", "path", "show_index"},
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

// TestRequireResolvedConfig pins the fail-loud precondition: mcp-serve must refuse to serve an
// unresolved desk rather than register tools against a nil/empty desk and answer wrongly. A nil
// cfg and an empty DeskRoot/DeskName each yield an actionable error naming the missing identity;
// a fully-resolved cfg passes.
func TestRequireResolvedConfig(t *testing.T) {
	cases := []struct {
		name    string
		cfg     *config.Config
		wantErr bool
		mustSay []string
	}{
		{"nil config", nil, true, []string{"DESK_ROOT", "DESK_NAME"}},
		{"empty desk", &config.Config{}, true, []string{"DESK_ROOT", "DESK_NAME"}},
		{"missing DeskName", &config.Config{DeskRoot: "/tmp/desk"}, true, []string{"DESK_NAME"}},
		{"missing DeskRoot", &config.Config{DeskName: "example-desk"}, true, []string{"DESK_ROOT"}},
		{"resolved", &config.Config{DeskRoot: "/tmp/desk", DeskName: "example-desk"}, false, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := requireResolvedConfig(tc.cfg)
			if tc.wantErr && err == nil {
				t.Fatalf("expected an error for %s, got nil", tc.name)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("expected no error for %s, got %v", tc.name, err)
			}
			for _, want := range tc.mustSay {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("%s: error %q should name %q", tc.name, err.Error(), want)
				}
			}
		})
	}
}

// TestEmitMountSignal is the focused unit test on the mount-signal shape: ONE line naming the
// server identity, the gated MODULE SET, and the exact tool count/set, terminated by a newline.
// This is the "the surface mounted" observability signal — presence must be visible on stderr,
// never silently absent — and the module segment (DoD 2d) makes a module-gated mount legible.
func TestEmitMountSignal(t *testing.T) {
	tools := []string{"sweep", "patrol", "query"}

	// Module-gated form: modules names the declared set on the line.
	var buf bytes.Buffer
	emitMountSignal(&buf, []string{"pm"}, tools)
	got := buf.String()

	if strings.Count(got, "\n") != 1 || !strings.HasSuffix(got, "\n") {
		t.Errorf("mount signal must be exactly ONE newline-terminated line; got %q", got)
	}
	if !strings.Contains(got, serverName) {
		t.Errorf("mount signal should name the server %q; got %q", serverName, got)
	}
	if !strings.Contains(got, "modules: pm") {
		t.Errorf("mount signal should name the gated module set; got %q", got)
	}
	if !strings.Contains(got, "3 tool") {
		t.Errorf("mount signal should report the tool count; got %q", got)
	}
	for _, name := range tools {
		if !strings.Contains(got, name) {
			t.Errorf("mount signal should list tool %q; got %q", name, got)
		}
	}

	// Unset form (modules == nil): the segment renders "all".
	var allBuf bytes.Buffer
	emitMountSignal(&allBuf, nil, tools)
	if !strings.Contains(allBuf.String(), "modules: all") {
		t.Errorf("mount signal with no module filter should render \"modules: all\"; got %q", allBuf.String())
	}
}

// TestResolveModuleGate is the load-bearing unit test for the MCP_MODULES gate's three DISTINCT
// cases (the design trap this slice exists to get right). It runs against the librarian-only
// registry TestMain installs, so the "pm" and "nonexistent" cases are genuinely unresolvable here
// and prove the fail-loud branch WITHOUT re-exec. It never touches process env (raw/declared are
// passed in), so it is fully hermetic.
func TestResolveModuleGate(t *testing.T) {
	cfg := &config.Config{AutonomousWrites: false} // librarian default surface: 5 exposed tools
	librarian5 := []string{"sweep", "patrol", "propose_fix", "query", "record_feedback"}

	t.Run("unset exposes all (unchanged behavior)", func(t *testing.T) {
		names, modules, _, ok := resolveModuleGate(cfg, "", false)
		if !ok {
			t.Fatalf("unset MCP_MODULES must resolve, not fail loud")
		}
		if modules != nil {
			t.Errorf("unset must yield a nil module label (renders \"all\"); got %v", modules)
		}
		if !equalSet(names, librarian5) {
			t.Errorf("unset names = %v, want the full librarian set %v", names, librarian5)
		}
	})

	t.Run("librarian filter yields exactly the librarian tools", func(t *testing.T) {
		names, modules, _, ok := resolveModuleGate(cfg, "librarian", true)
		if !ok {
			t.Fatalf("MCP_MODULES=librarian must resolve on a librarian desk")
		}
		if !equalSet(modules, []string{"librarian"}) {
			t.Errorf("modules label = %v, want [librarian]", modules)
		}
		if !equalSet(names, librarian5) {
			t.Errorf("filtered names = %v, want %v", names, librarian5)
		}
	})

	t.Run("partial match still serves (do not overengineer per-name validation)", func(t *testing.T) {
		names, _, _, ok := resolveModuleGate(cfg, "librarian,bogus", true)
		if !ok {
			t.Fatalf("a partially-matching set must serve the matched subset, not fail loud")
		}
		if !equalSet(names, librarian5) {
			t.Errorf("filtered names = %v, want %v", names, librarian5)
		}
	})

	failCases := []struct {
		name    string
		raw     string
		mustSay string
	}{
		{"explicitly empty", "", "MCP_MODULES"},
		{"whitespace and commas only", " , ", "MCP_MODULES"},
		{"typo'd module name", "libarian", "no exposed tools"},
		{"module not registered on this desk", "pm", "no exposed tools"},
	}
	for _, fc := range failCases {
		t.Run("fail loud: "+fc.name, func(t *testing.T) {
			names, _, reason, ok := resolveModuleGate(cfg, fc.raw, true)
			if ok {
				t.Fatalf("MCP_MODULES=%q must fail loud (resolve to no tools), got names=%v", fc.raw, names)
			}
			if names != nil {
				t.Errorf("fail-loud path must return no names; got %v", names)
			}
			if !strings.Contains(reason, fc.mustSay) {
				t.Errorf("fail reason %q should mention %q", reason, fc.mustSay)
			}
		})
	}
}

// TestResolveModuleGate_PMFilter proves the desk-pm mount's gate: with BOTH librarian and pm
// registered (the PM_ENABLED case), MCP_MODULES=pm resolves to exactly the twelve PM tools and
// none of the librarian ride-alongs, while MCP_MODULES=librarian keeps exactly the librarian set.
// It registers pm specs like the agent test does and restores the librarian-only registry via
// t.Cleanup so the rest of the package's tests keep their TestMain-installed state — hermetic.
func TestResolveModuleGate_PMFilter(t *testing.T) {
	toolcore.Reset()
	toolcore.Register(tools.Specs()...)
	// writesEnabled=true so all twelve PM specs are AgentDefault and land in ExposedSpecs.
	toolcore.Register(pmtools.Specs(func() coreschema.DocumentValidator { return nil }, true)...)
	t.Cleanup(func() {
		toolcore.Reset()
		toolcore.Register(tools.Specs()...)
	})

	cfg := &config.Config{AutonomousWrites: true} // widest gate; ExposedSpecs still excludes restore

	pmNames, modules, _, ok := resolveModuleGate(cfg, "pm", true)
	if !ok {
		t.Fatalf("MCP_MODULES=pm must resolve when pm is registered")
	}
	if !equalSet(modules, []string{"pm"}) {
		t.Errorf("modules label = %v, want [pm]", modules)
	}
	if !equalSet(pmNames, pmtools.ToolNames()) {
		t.Errorf("MCP_MODULES=pm names = %v, want exactly the 12 PM tools %v", pmNames, pmtools.ToolNames())
	}
	if len(pmNames) != 12 {
		t.Errorf("MCP_MODULES=pm exposed %d tools, want exactly 12", len(pmNames))
	}
	// None of the five librarian ride-alongs may leak into the pm mount.
	pmSet := map[string]bool{}
	for _, n := range pmNames {
		pmSet[n] = true
	}
	for _, ride := range []string{"sweep", "patrol", "propose_fix", "query", "record_feedback", "apply_fix"} {
		if pmSet[ride] {
			t.Errorf("librarian tool %q leaked into the pm mount (MCP_MODULES=pm must drop the ride-alongs)", ride)
		}
	}

	// The complementary filter: MCP_MODULES=librarian keeps exactly the librarian tools (6 with
	// autonomous writes on: the 5 defaults + apply_fix), and no PM tool.
	libNames, _, _, ok := resolveModuleGate(cfg, "librarian", true)
	if !ok {
		t.Fatalf("MCP_MODULES=librarian must resolve")
	}
	wantLib := []string{"sweep", "patrol", "propose_fix", "query", "record_feedback", "apply_fix"}
	if !equalSet(libNames, wantLib) {
		t.Errorf("MCP_MODULES=librarian names = %v, want %v", libNames, wantLib)
	}

	// The combined declaration completes the matrix: both modules named exposes exactly the
	// union (18 = 6 librarian + 12 pm here) — pinning SelectByModules against accidental
	// deduplication or ordering bugs across a multi-module set.
	bothNames, bothModules, _, ok := resolveModuleGate(cfg, "librarian,pm", true)
	if !ok {
		t.Fatalf("MCP_MODULES=librarian,pm must resolve when both modules are registered")
	}
	if !equalSet(bothModules, []string{"librarian", "pm"}) {
		t.Errorf("modules label = %v, want [librarian pm]", bothModules)
	}
	if !equalSet(bothNames, toolcore.ToolNames(toolcore.ExposedSpecs(cfg))) {
		t.Errorf("MCP_MODULES=librarian,pm names = %v, want the full exposed set", bothNames)
	}
	if len(bothNames) != 18 {
		t.Errorf("MCP_MODULES=librarian,pm exposed %d tool(s), want 18 (6 librarian + 12 pm)", len(bothNames))
	}
}

// TestServe_MountSignalStderrNotStdout runs Serve end-to-end with a resolved desk and a stdin at
// immediate EOF (clean stdio shutdown). It asserts the mount signal lands on STDERR and NEVER on
// stdout — stdout is the JSON-RPC channel, so a stray diagnostic byte there corrupts the protocol.
// StdioTransport binds os.Stdin/os.Stdout at Connect time, so the globals are redirected for the
// duration and restored after.
func TestServe_MountSignalStderrNotStdout(t *testing.T) {
	cfg := &config.Config{DeskName: "example-desk", DeskRoot: t.TempDir()}

	origIn, origOut, origErr := os.Stdin, os.Stdout, os.Stderr
	restore := func() { os.Stdin, os.Stdout, os.Stderr = origIn, origOut, origErr }
	defer restore()

	inR, inW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	_ = inW.Close() // immediate EOF on stdin → clean stdio shutdown
	outR, outW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	errR, errW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdin, os.Stdout, os.Stderr = inR, outW, errW

	// Drain concurrently so a write by the SDK can never block Serve on a full pipe.
	var outBuf, errBuf bytes.Buffer
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); _, _ = io.Copy(&outBuf, outR) }()
	go func() { defer wg.Done(); _, _ = io.Copy(&errBuf, errR) }()

	serr := Serve(context.Background(), nil, cfg)

	_ = outW.Close()
	_ = errW.Close()
	wg.Wait()
	restore()

	if serr != nil {
		t.Fatalf("Serve returned an error on a clean EOF shutdown: %v", serr)
	}
	if !strings.Contains(errBuf.String(), serverName) || !strings.Contains(errBuf.String(), "mounted") {
		t.Errorf("mount signal missing from stderr; got %q", errBuf.String())
	}
	if strings.Contains(outBuf.String(), "mounted") {
		t.Errorf("mount signal leaked to stdout (JSON-RPC channel): %q", outBuf.String())
	}
}

// TestServe_FailsLoudOnUnresolvedDesk is the RED-able regression for the fail-loud contract:
// Serve on an unresolved desk must exit NON-ZERO with an actionable stderr message — never serve
// silently and exit 0. Serve calls os.Exit on that path, so the assertion runs in a re-exec'd
// child. RED (before the guard): the child registers tools, hits EOF stdin, and exits 0.
func TestServe_FailsLoudOnUnresolvedDesk(t *testing.T) {
	if os.Getenv("DESKKIT_MCP_FAILLOUD_CHILD") == "1" {
		// Child: an empty (unresolved) desk config must fail loud, not serve. Serve os.Exit(1)s;
		// if the guard were absent it would fall through to a clean EOF shutdown and exit 0.
		_ = Serve(context.Background(), nil, &config.Config{})
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=TestServe_FailsLoudOnUnresolvedDesk")
	cmd.Env = append(os.Environ(), "DESKKIT_MCP_FAILLOUD_CHILD=1")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()

	var ee *exec.ExitError
	if !errors.As(err, &ee) {
		t.Fatalf("expected a non-zero exit (fail loud), got err=%v; stderr=%q", err, stderr.String())
	}
	if code := ee.ExitCode(); code != 1 {
		t.Errorf("expected exit code 1, got %d; stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "DESK_ROOT") || !strings.Contains(stderr.String(), "DESK_NAME") {
		t.Errorf("stderr not actionable (should name DESK_ROOT and DESK_NAME); got %q", stderr.String())
	}
}

// TestServe_FailsLoudOnModuleGate is the RED-able regression for the MCP_MODULES fail-loud
// contract: a RESOLVED desk with MCP_MODULES set to an empty or unresolvable value must exit
// NON-ZERO with an actionable stderr message — never silently serve the full set or an empty set
// and exit 0. Serve os.Exit(1)s on that path (a returned error would be swallowed by PocketBase's
// serve goroutine), so the assertion runs in a re-exec'd child — the same pattern as
// TestServe_FailsLoudOnUnresolvedDesk. The child's TestMain registers librarian only, so
// MCP_MODULES=pm and MCP_MODULES=nonexistent are genuinely unresolvable there.
func TestServe_FailsLoudOnModuleGate(t *testing.T) {
	if os.Getenv("DESKKIT_MCP_MODGATE_CHILD") == "1" {
		// Child: a fully-resolved desk (so requireResolvedConfig passes and we reach the module
		// gate), with MCP_MODULES inherited from the parent. Empty/unresolvable → Serve os.Exit(1)s;
		// absent the gate it would build the server and fall through to a clean EOF shutdown, exit 0.
		_ = Serve(context.Background(), nil, &config.Config{DeskRoot: os.TempDir(), DeskName: "probe"})
		return
	}
	cases := []struct {
		name    string
		modules string
		mustSay []string
	}{
		{"explicitly empty", "", []string{"MCP_MODULES", "no module"}},
		{"whitespace and commas only", " , ", []string{"MCP_MODULES", "no module"}},
		{"unresolvable module name", "nonexistent", []string{"MCP_MODULES", "no exposed tools"}},
		{"pm requested but not registered on this desk", "pm", []string{"MCP_MODULES", "no exposed tools"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=TestServe_FailsLoudOnModuleGate")
			cmd.Env = append(os.Environ(), "DESKKIT_MCP_MODGATE_CHILD=1", "MCP_MODULES="+tc.modules)
			var stderr bytes.Buffer
			cmd.Stderr = &stderr
			err := cmd.Run()

			var ee *exec.ExitError
			if !errors.As(err, &ee) {
				t.Fatalf("expected a non-zero exit (fail loud), got err=%v; stderr=%q", err, stderr.String())
			}
			if code := ee.ExitCode(); code != 1 {
				t.Errorf("expected exit code 1, got %d; stderr=%q", code, stderr.String())
			}
			for _, want := range tc.mustSay {
				if !strings.Contains(stderr.String(), want) {
					t.Errorf("stderr not actionable (should mention %q); got %q", want, stderr.String())
				}
			}
		})
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

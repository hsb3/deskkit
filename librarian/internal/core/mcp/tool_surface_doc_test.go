package mcp

// Tool-surface drift guard — the only one. docs/development/specs/tool-surface.md is the
// authoritative, empirically derived map of every tool-bearing surface (ADR 0016: "tool-surface
// truth lives in the tool-surface spec, pinned by a drift guard so counts can't rot again"). This
// file is that guard whole: it READS every count the doc asserts and re-derives each from source —
// the gate-dependent MCP totals from the SAME toolcore gate the server registers, the CLI base
// count from the command registrations in cmd/deskkit/main.go — so the doc and the source cannot
// silently diverge. It rides the existing `go test ./...` lane (make test).
//
// The counts asserted here are the LIVE totals the binary exposes, i.e. every registered module:
// the always-on `profile` module (4 ungated read-only tools) plus the env-gated `librarian` and
// `pm` modules. The doc must state those live numbers — an "everything except the ungated module"
// figure would be a number no caller ever observes.
//
// It pins the counts, not the doc's bytes, so an unrelated prose/row edit (e.g. re-wording the
// `findings dispose` row) does NOT trip it — only a real count change (a tool added/removed
// without a matching doc edit, or a doc edit without the matching source change) turns it RED.

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/hsb3/desk-standard/librarian/internal/core/config"
	coreschema "github.com/hsb3/desk-standard/librarian/internal/core/schema"
	"github.com/hsb3/desk-standard/librarian/internal/core/toolcore"
	"github.com/hsb3/desk-standard/librarian/internal/modules/librarian/tools"
	pmtools "github.com/hsb3/desk-standard/librarian/internal/modules/pm/tools"
	profiletools "github.com/hsb3/desk-standard/librarian/internal/modules/profile/tools"
)

// docSurfaceCounts holds the MCP counts a parse of the tool-surface spec found, each as the list
// of documented occurrences (the doc states most of them twice — the §2 gate table AND the
// Summary table — so a mismatch BETWEEN the two doc tables is caught too, not just doc-vs-
// source). Every occurrence must equal the source-derived count.
type docSurfaceCounts struct {
	def         []int // default (neither flag)
	writes      []int // LIBRARIAN_AUTONOMOUS_WRITES=true
	pm          []int // PM_ENABLED=true
	both        []int // both flags
	deskPM      []int // pm-only mount (PM_ENABLED=true, MCP_MODULES=pm)
	deskProfile []int // profile-only mount (MCP_MODULES=profile)
}

// TestToolSurfaceDoc_MCPCounts is the drift guard: every MCP count the tool-surface spec states is
// re-derived from toolcore's real gate and asserted equal.
func TestToolSurfaceDoc_MCPCounts(t *testing.T) {
	root := repoRootFrom(t)
	docPath := filepath.Join(root, "docs", "development", "specs", "tool-surface.md")
	md, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatalf("read %s: %v", docPath, err)
	}
	doc := parseDocSurfaceCounts(string(md))

	// Every combo must be documented at least once; a zero-length list means the doc's structure
	// moved out from under the parser (which is itself drift worth failing on).
	for name, vals := range map[string][]int{
		"default (neither flag)":                   doc.def,
		"LIBRARIAN_AUTONOMOUS_WRITES=true":         doc.writes,
		"PM_ENABLED=true":                          doc.pm,
		"both flags":                               doc.both,
		"pm-only mount (MCP_MODULES=pm)":           doc.deskPM,
		"profile-only mount (MCP_MODULES=profile)": doc.deskProfile,
	} {
		if len(vals) == 0 {
			t.Fatalf("tool-surface spec: found no documented count for %q — the doc's table shape "+
				"changed; update the parser in tool_surface_doc_test.go alongside the doc", name)
		}
	}

	// --- Phase A: the always-registered modules → the live PM_ENABLED=false counts. ---
	// `profile` has no env gate and is registered unconditionally in cmd/deskkit/main.go, so a
	// PM-off desk still serves its 4 read-only tools on top of the librarian module's 5 (6 with
	// LIBRARIAN_AUTONOMOUS_WRITES). Restore TestMain's librarian-only registry afterwards so
	// sibling tests keep the state they were written against — hermetic, mirroring
	// TestResolveModuleGate_PMFilter.
	t.Cleanup(func() {
		toolcore.Reset()
		toolcore.Register(tools.Specs()...)
	})
	toolcore.Reset()
	toolcore.Register(profiletools.Specs()...)
	toolcore.Register(tools.Specs()...)

	srcDefault := len(toolcore.ExposedTools(&config.Config{AutonomousWrites: false}))
	srcWrites := len(toolcore.ExposedTools(&config.Config{AutonomousWrites: true}))
	assertCounts(t, "default (PM off, LIBRARIAN_AUTONOMOUS_WRITES unset)", doc.def, srcDefault)
	assertCounts(t, "LIBRARIAN_AUTONOMOUS_WRITES=true (PM off)", doc.writes, srcWrites)

	// --- Phase B: register the pm module too → the live PM-on counts + both gated mounts. ---
	// writesEnabled=true so all twelve PM specs are AgentDefault (PM_AUTONOMOUS_WRITES default ON),
	// matching the doc's "+ 12 PM tools".
	toolcore.Register(pmtools.Specs(func() coreschema.DocumentValidator { return nil }, true)...)

	srcPM := len(toolcore.ExposedTools(&config.Config{AutonomousWrites: false}))
	srcBoth := len(toolcore.ExposedTools(&config.Config{AutonomousWrites: true}))
	// Gated mounts: the exposed set filtered to one module — the same SelectByModules over
	// ExposedSpecs the Serve module gate uses. Neither count depends on the librarian write gate,
	// so AutonomousWrites here does not change them. The pm-only assertion is the regression this
	// axis exists for: the ungated `profile` module must NOT leak into a mount that declared pm.
	exposed := toolcore.ExposedSpecs(&config.Config{AutonomousWrites: true})
	srcPMOnly := len(toolcore.SelectByModules(exposed, "pm"))
	srcProfileOnly := len(toolcore.SelectByModules(exposed, "profile"))

	assertCounts(t, "PM_ENABLED=true", doc.pm, srcPM)
	assertCounts(t, "both flags (AUTONOMOUS_WRITES + PM_ENABLED)", doc.both, srcBoth)
	assertCounts(t, "pm-only mount (MCP_MODULES=pm)", doc.deskPM, srcPMOnly)
	assertCounts(t, "profile-only mount (MCP_MODULES=profile)", doc.deskProfile, srcProfileOnly)

	// The gated mounts must carry exactly their own module's tools — asserting only the totals
	// would pass if a tool moved between modules, which is the same drift by another name.
	assertModuleNames(t, "pm-only mount", toolcore.SelectByModules(exposed, "pm"), pmtools.ToolNames())
	assertModuleNames(t, "profile-only mount", toolcore.SelectByModules(exposed, "profile"), profiletools.ToolNames())
}

// TestToolSurfaceDoc_CLICount is the CLI leg of the same guard: the "N base" subcommand count the
// tool-surface spec's Summary row states, re-derived by reading the real registration sites out of
// cmd/deskkit/main.go — every app.RootCmd.AddCommand call (which includes the two framework system
// commands, serve + superuser) plus the migratecmd registration. The `pm` group (registered
// conditionally under PM_ENABLED) and cobra's auto-registered help/completion are excluded, exactly
// as the doc's "N base (+ `pm` group under `PM_ENABLED`)" excludes them.
func TestToolSurfaceDoc_CLICount(t *testing.T) {
	root := repoRootFrom(t)

	docPath := filepath.Join(root, "docs", "development", "specs", "tool-surface.md")
	md, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatalf("read %s: %v", docPath, err)
	}
	documented, ok := docCLICount(string(md))
	if !ok {
		t.Fatalf("tool-surface spec: found no parseable %q Summary row in %s — the doc's table shape "+
			"changed; update the parser in tool_surface_doc_test.go alongside the doc", cliSummaryLabel, docPath)
	}

	mainPath := filepath.Join(root, "librarian", "cmd", "deskkit", "main.go")
	src, err := os.ReadFile(mainPath)
	if err != nil {
		t.Fatalf("read %s: %v", mainPath, err)
	}
	adds := countLiveCalls(string(src), "app.RootCmd.AddCommand(")
	if adds == 0 {
		t.Fatalf("CLI surface: found 0 app.RootCmd.AddCommand( call sites in %s — the registration "+
			"pattern moved and this guard is deriving nothing; fix the derivation, do not pin a constant", mainPath)
	}
	migrate := 0
	if countLiveCalls(string(src), "migratecmd.MustRegister(") > 0 {
		migrate = 1
	}
	derived := adds + migrate

	if derived != documented {
		t.Errorf("tool-surface drift: the tool-surface spec says the CLI surface = %d base, but "+
			"cmd/deskkit/main.go derives %d (%d AddCommand + %d migratecmd) — a subcommand was "+
			"added/removed without updating the doc (or vice versa); reconcile the two",
			documented, derived, adds, migrate)
	}
}

// countLiveCalls counts occurrences of a call site on lines that are not commented out. A plain
// strings.Count over the whole file also matches a `// app.RootCmd.AddCommand(…)` line, so
// commenting a registration out would slip past the guard unseen.
// ponytail: line comments only, no /* */ handling — add it when a registration is ever block-commented.
func countLiveCalls(src, call string) int {
	n := 0
	for _, line := range strings.Split(src, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "//") {
			continue
		}
		n += strings.Count(line, call)
	}
	return n
}

// cliSummaryLabel keys the Summary-table row carrying the CLI base count.
const cliSummaryLabel = "Librarian CLI subcommands"

// docCLICount returns the base subcommand count the tool-surface spec's Summary row states. The
// count cell reads "N base (+ …)", which firstIntCell already handles.
func docCLICount(md string) (int, bool) {
	for _, line := range strings.Split(md, "\n") {
		cells := tableCells(line)
		if len(cells) < 2 || !strings.Contains(cells[0], cliSummaryLabel) {
			continue
		}
		if n, ok := firstIntCell(cells); ok {
			return n, true
		}
	}
	return 0, false
}

// assertModuleNames fails when a gated mount's exposed specs are not exactly the named set,
// ignoring order. Names, not just a count, so a tool swapping modules is caught.
func assertModuleNames(t *testing.T, mount string, specs []toolcore.ToolSpec, want []string) {
	t.Helper()
	got := make(map[string]bool, len(specs))
	for _, s := range specs {
		got[s.Name] = true
	}
	for _, w := range want {
		if !got[w] {
			t.Errorf("%s: expected tool %q to be exposed, but it is not", mount, w)
		}
		delete(got, w)
	}
	for extra := range got {
		t.Errorf("%s: unexpected tool %q leaked into the mount (it belongs to another module)", mount, extra)
	}
}

// assertCounts fails when any documented occurrence of a surface's count disagrees with the
// source-derived count, naming the surface and both numbers so the fix is obvious.
func assertCounts(t *testing.T, surface string, documented []int, source int) {
	t.Helper()
	for _, d := range documented {
		if d != source {
			t.Errorf("tool-surface drift: the tool-surface spec says %s = %d, but toolcore derives %d — "+
				"a tool was added/removed without updating the doc (or vice versa); reconcile the two",
				surface, d, source)
		}
	}
}

// parseDocSurfaceCounts scans the tool-surface spec's markdown TABLE ROWS for the MCP
// counts. It reads only structured table cells (never prose), keyed on stable labels, so the
// in-flight prose/row edits that never change a count leave it untouched. For each matched row it
// takes the first cell after the label whose leading token is an integer (the count column sits at
// column 1 in the §2/Summary tables but column 2 in the §2.1 mount table).
func parseDocSurfaceCounts(md string) docSurfaceCounts {
	var out docSurfaceCounts
	for _, line := range strings.Split(md, "\n") {
		cells := tableCells(line)
		if len(cells) < 2 {
			continue
		}
		label := cells[0]
		n, ok := firstIntCell(cells)
		if !ok {
			continue
		}
		lower := strings.ToLower(label)
		switch {
		// §2.1/§2.2 gated-mount rows + their Summary echoes. Matched FIRST: a mount row names its
		// module, and must not be swallowed by the broader "Librarian MCP …" cases below.
		case strings.Contains(lower, "profile-only mount"):
			out.deskProfile = append(out.deskProfile, n)
		case strings.Contains(lower, "pm-only mount"):
			out.deskPM = append(out.deskPM, n)
		// Summary + §2.1-mount rows: "Librarian MCP — default" / "…+LIBRARIAN_AUTONOMOUS_WRITES"
		// / "…+PM_ENABLED" / "…both". The §2.1 "Librarian MCP (default; PM_ENABLED implicit)" mount
		// row (MCP_MODULES unset) matches THIS PM_ENABLED case, not the plain "default" case below
		// — since 1.0 PM_ENABLED defaults on, so the live unset-mount count is the PM-on total.
		case strings.Contains(label, "Librarian MCP") && strings.Contains(label, "both"):
			out.both = append(out.both, n)
		case strings.Contains(label, "Librarian MCP") && strings.Contains(label, "PM_ENABLED"):
			out.pm = append(out.pm, n)
		case strings.Contains(label, "Librarian MCP") && strings.Contains(label, "AUTONOMOUS"):
			out.writes = append(out.writes, n)
		case strings.Contains(label, "Librarian MCP") && strings.Contains(label, "default"):
			out.def = append(out.def, n)
		// §2 gate table rows, keyed on the environment cell.
		case strings.Contains(label, "neither flag"):
			out.def = append(out.def, n)
		case label == "LIBRARIAN_AUTONOMOUS_WRITES=true":
			out.writes = append(out.writes, n)
		case label == "PM_ENABLED=true":
			out.pm = append(out.pm, n)
		case strings.Contains(label, "both flags"):
			out.both = append(out.both, n)
		}
	}
	return out
}

// tableCells splits a markdown table row into trimmed cells, stripping emphasis (**) and code
// (`) markers so a labeled/count cell compares cleanly. Returns nil for a non-table line.
func tableCells(line string) []string {
	s := strings.TrimSpace(line)
	if !strings.HasPrefix(s, "|") {
		return nil
	}
	s = strings.Trim(s, "|")
	parts := strings.Split(s, "|")
	cells := make([]string, len(parts))
	for i, p := range parts {
		c := strings.ReplaceAll(p, "**", "")
		c = strings.ReplaceAll(c, "`", "")
		cells[i] = strings.TrimSpace(c)
	}
	return cells
}

// firstIntCell returns the value of the first cell after the label (index >= 1) whose leading
// whitespace-delimited token parses as an integer (e.g. "5", or "18 base (+ …)" → 18).
func firstIntCell(cells []string) (int, bool) {
	for i := 1; i < len(cells); i++ {
		fields := strings.Fields(cells[i])
		if len(fields) == 0 {
			continue
		}
		if n, err := strconv.Atoi(fields[0]); err == nil {
			return n, true
		}
	}
	return 0, false
}

// repoRootFrom walks up from the test's working directory (the package dir under `go test`) to
// the repo root, identified by the presence of the tool-surface spec.
func repoRootFrom(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	dir := wd
	for i := 0; i < 16; i++ {
		if _, err := os.Stat(filepath.Join(dir, "docs", "development", "specs", "tool-surface.md")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatalf("could not locate docs/tool-surface.md walking up from %s", wd)
	return ""
}

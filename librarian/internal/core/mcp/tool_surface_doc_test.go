package mcp

// Tool-surface drift guard (Go half). docs/tool-surface.md is the authoritative, empirically
// derived map of every tool-bearing surface (ADR 0016: "tool-surface truth lives in
// docs/tool-surface.md, pinned by a drift guard so counts can't rot again"). The gate-dependent
// Librarian MCP counts are the part that is NOT statically obvious from source — they depend on
// two independent env gates (LIBRARIAN_AUTONOMOUS_WRITES x PM_ENABLED) plus the MCP_MODULES
// module axis — so the JS guard (scripts/check-tool-surface.mjs) deliberately does NOT
// re-implement Go's gate arithmetic. This test is the other half: it READS the counts the doc
// asserts and cross-checks each against the SAME toolcore gate the server registers, so the doc
// and the source cannot silently diverge. It rides the existing `go test ./...` lane (make test).
//
// It pins the counts, not the doc's bytes, so an unrelated prose/row edit (e.g. re-wording the
// `findings dispose` row) does NOT trip it — only a real count change (a tool added/removed
// without a matching doc edit, or a doc edit without the matching source change) turns it RED.
//
// Companion: scripts/check-tool-surface.mjs pins the TS server (4) and CLI (16 base) counts and
// asserts THIS file's presence, so removing the MCP-count guard fails `make check`.

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/example/pocket-librarian/internal/core/config"
	coreschema "github.com/example/pocket-librarian/internal/core/schema"
	"github.com/example/pocket-librarian/internal/core/toolcore"
	"github.com/example/pocket-librarian/internal/modules/librarian/tools"
	pmtools "github.com/example/pocket-librarian/internal/modules/pm/tools"
)

// docSurfaceCounts holds the Librarian-MCP counts a parse of docs/tool-surface.md found, each as
// the list of documented occurrences (the doc states most of them twice — the §2 gate table AND
// the Summary table — so a mismatch BETWEEN the two doc tables is caught too, not just doc-vs-
// source). Every occurrence must equal the source-derived count.
type docSurfaceCounts struct {
	def    []int // default (neither flag)
	writes []int // LIBRARIAN_AUTONOMOUS_WRITES=true
	pm     []int // PM_ENABLED=true
	both   []int // both flags
	deskPM []int // desk-pm mount (PM_ENABLED=true, MCP_MODULES=pm)
}

// TestToolSurfaceDoc_MCPCounts is the drift guard: every Librarian-MCP count docs/tool-surface.md
// states is re-derived from toolcore's real gate and asserted equal. Named so the JS guard can
// assert this file's presence by function name.
func TestToolSurfaceDoc_MCPCounts(t *testing.T) {
	root := repoRootFrom(t)
	docPath := filepath.Join(root, "docs", "tool-surface.md")
	md, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatalf("read %s: %v", docPath, err)
	}
	doc := parseDocSurfaceCounts(string(md))

	// Every combo must be documented at least once; a zero-length list means the doc's structure
	// moved out from under the parser (which is itself drift worth failing on).
	for name, vals := range map[string][]int{
		"default (neither flag)":           doc.def,
		"LIBRARIAN_AUTONOMOUS_WRITES=true": doc.writes,
		"PM_ENABLED=true":                  doc.pm,
		"both flags":                       doc.both,
		"desk-pm mount (MCP_MODULES=pm)":   doc.deskPM,
	} {
		if len(vals) == 0 {
			t.Fatalf("docs/tool-surface.md: found no documented count for %q — the doc's table shape "+
				"changed; update the parser in tool_surface_doc_test.go alongside the doc", name)
		}
	}

	// --- Phase A: librarian-only registry (what TestMain installed) → the 5 / 6 counts. ---
	src5 := len(toolcore.ExposedTools(&config.Config{AutonomousWrites: false}))
	src6 := len(toolcore.ExposedTools(&config.Config{AutonomousWrites: true}))
	assertCounts(t, "default (LIBRARIAN_AUTONOMOUS_WRITES unset)", doc.def, src5)
	assertCounts(t, "LIBRARIAN_AUTONOMOUS_WRITES=true", doc.writes, src6)

	// --- Phase B: register the pm module too → the 17 / 18 (+MCP_MODULES=pm → 12) counts. ---
	// writesEnabled=true so all twelve PM specs are AgentDefault (PM_AUTONOMOUS_WRITES default ON),
	// matching the doc's "+ 12 PM tools". Restore the librarian-only registry so sibling tests keep
	// their TestMain-installed state — hermetic, mirroring TestResolveModuleGate_PMFilter.
	toolcore.Reset()
	toolcore.Register(tools.Specs()...)
	toolcore.Register(pmtools.Specs(func() coreschema.DocumentValidator { return nil }, true)...)
	t.Cleanup(func() {
		toolcore.Reset()
		toolcore.Register(tools.Specs()...)
	})

	src17 := len(toolcore.ExposedTools(&config.Config{AutonomousWrites: false}))
	src18 := len(toolcore.ExposedTools(&config.Config{AutonomousWrites: true}))
	// desk-pm mount: the exposed set filtered to the pm module (MCP_MODULES=pm) — the same
	// SelectByModules over ExposedSpecs the Serve module gate uses. PM count is 12 regardless of
	// the librarian write gate, so AutonomousWrites here does not change it.
	src12 := len(toolcore.SelectByModules(toolcore.ExposedSpecs(&config.Config{AutonomousWrites: true}), "pm"))

	assertCounts(t, "PM_ENABLED=true", doc.pm, src17)
	assertCounts(t, "both flags (AUTONOMOUS_WRITES + PM_ENABLED)", doc.both, src18)
	assertCounts(t, "desk-pm mount (MCP_MODULES=pm)", doc.deskPM, src12)
}

// assertCounts fails when any documented occurrence of a surface's count disagrees with the
// source-derived count, naming the surface and both numbers so the fix is obvious.
func assertCounts(t *testing.T, surface string, documented []int, source int) {
	t.Helper()
	for _, d := range documented {
		if d != source {
			t.Errorf("tool-surface drift: docs/tool-surface.md says %s = %d, but toolcore derives %d — "+
				"a tool was added/removed without updating the doc (or vice versa); reconcile the two",
				surface, d, source)
		}
	}
}

// parseDocSurfaceCounts scans docs/tool-surface.md's markdown TABLE ROWS for the Librarian-MCP
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
		switch {
		// Summary + §2.1-mount rows: "Librarian MCP — default" / "…+LIBRARIAN_AUTONOMOUS_WRITES"
		// / "…+PM_ENABLED" / "…both", and the §2.1 "Librarian MCP (default)" mount row.
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
		// §2.1 module-gating table: the desk-pm mount row.
		case strings.Contains(label, "desk-pm mount"):
			out.deskPM = append(out.deskPM, n)
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
// whitespace-delimited token parses as an integer (e.g. "5", or "16 base (+ …)" → 16).
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
// the repo root, identified by the presence of docs/tool-surface.md.
func repoRootFrom(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	dir := wd
	for i := 0; i < 16; i++ {
		if _, err := os.Stat(filepath.Join(dir, "docs", "tool-surface.md")); err == nil {
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

package mcp

// Bundle-shape guard. The shipped marketplace bundle under plugins/desk-persona/ is authored by
// hand (only its librarian-operator agent is generated, by scripts/check-persona-drift.mjs), so
// nothing but this test holds its authored artifacts to the tool surface the binary actually
// serves. The failure mode it exists to catch: an artifact that names a mount or a tool which does
// not exist, shipped inside a bundle a marketplace install copies verbatim.
//
// The regression that motivated it: .mcp.json's MCP_MODULES is the only thing deciding which
// modules the mount registers. Dropping a module from that string silently shrinks the served tool
// surface while every other gate stays green — so the module list is cross-checked against the
// modules the Go registry really declares, not against a hardcoded copy of the same three strings.
//
// It reads the bundle from the repo tree at runtime (repoRootFrom, shared with the tool-surface
// guard) rather than embedding it: go:embed cannot reach outside the Go module, a test can.
// It touches none of toolcore's global registry — every set here comes from a pure Specs() call —
// so it is order-independent with respect to the sibling tests that do Reset/Register.
//
// NOT ported from the retired predecessor: the bare snake_case "invented tool name" heuristic. See
// the note above TestBundleNamespacedToolRefs_AreRealTools.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	coreschema "github.com/hsb3/desk-standard/librarian/internal/core/schema"
	"github.com/hsb3/desk-standard/librarian/internal/core/toolcore"
	libtools "github.com/hsb3/desk-standard/librarian/internal/modules/librarian/tools"
	pmtools "github.com/hsb3/desk-standard/librarian/internal/modules/pm/tools"
	profiletools "github.com/hsb3/desk-standard/librarian/internal/modules/profile/tools"

	"gopkg.in/yaml.v3"
)

// bundleRel is the shipped bundle's path relative to the repo root, and bundleServer the key its
// .mcp.json registers the mount under (which is also the mcp__<server>__ tool-name namespace).
const (
	bundleRel    = "plugins/desk-persona"
	bundleServer = "desk-persona"
)

// The bundle's authored inventory. Enumerated rather than discovered so that a skill silently
// deleted, renamed, or added is a failure, not a same-count coincidence.
var (
	bundleAgents = []string{"librarian-operator", "pm-operator"}
	bundleSkills = []string{
		"brownfield-adoption", "conventions-standard", "desk-setup", "harvest-loop",
		"pm-advance-item", "pm-session-open", "pm-triage",
	}
)

// allSpecs is the merged registry: every module's specs, gates wide open, so the name set covers
// real-but-gated tools too (apply_fix, restore). A gated tool named in prose is not a phantom.
func allSpecs() []toolcore.ToolSpec {
	specs := append([]toolcore.ToolSpec{}, profiletools.Specs()...)
	specs = append(specs, libtools.Specs()...)
	specs = append(specs, pmtools.Specs(func() coreschema.DocumentValidator { return nil }, true)...)
	return specs
}

// registeredModules returns the distinct module names the merged registry declares.
func registeredModules() []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range allSpecs() {
		if !seen[s.Module] {
			seen[s.Module] = true
			out = append(out, s.Module)
		}
	}
	return out
}

// registeredToolNames returns every tool name in the merged registry.
func registeredToolNames() map[string]bool {
	out := map[string]bool{}
	for _, s := range allSpecs() {
		out[s.Name] = true
	}
	return out
}

// TestBundleMCPJSON_ServesEveryRegisteredModule pins the mount's shape and — the load-bearing
// half — asserts MCP_MODULES names exactly the modules the binary registers. A module dropped
// from that string shrinks the shipped tool surface with no other signal.
func TestBundleMCPJSON_ServesEveryRegisteredModule(t *testing.T) {
	var mcpCfg map[string]struct {
		Command string            `json:"command"`
		Args    []string          `json:"args"`
		Env     map[string]string `json:"env"`
	}
	b := readBundleFile(t, ".mcp.json")
	if err := json.Unmarshal(b, &mcpCfg); err != nil {
		t.Fatalf("%s/.mcp.json: not valid JSON: %v", bundleRel, err)
	}
	srv, ok := mcpCfg[bundleServer]
	if !ok {
		t.Fatalf("%s/.mcp.json: no %q server entry (the mcp__%s__ tool namespace depends on this key)",
			bundleRel, bundleServer, bundleServer)
	}
	if srv.Command != "deskkit" {
		t.Errorf("%s/.mcp.json: command = %q, want %q — the bundle must launch the binary", bundleRel, srv.Command, "deskkit")
	}
	if len(srv.Args) != 1 || srv.Args[0] != "mcp-serve" {
		t.Errorf("%s/.mcp.json: args = %v, want exactly [mcp-serve]", bundleRel, srv.Args)
	}
	if srv.Env["PM_ENABLED"] != "true" {
		t.Errorf("%s/.mcp.json: env.PM_ENABLED = %q, want \"true\" — the bundle ships the PM surfaces",
			bundleRel, srv.Env["PM_ENABLED"])
	}

	var named []string
	for _, m := range strings.Split(srv.Env["MCP_MODULES"], ",") {
		if m = strings.TrimSpace(m); m != "" {
			named = append(named, m)
		}
	}
	assertSameSet(t, "env.MCP_MODULES", named, registeredModules(),
		"the composed bundle serves every module the binary registers; a name missing here silently "+
			"withholds that module's tools from every install of this bundle")
}

// TestBundlePMOperatorAgent_ScopedToRealPMTools holds the hand-authored pm-operator agent to the
// real PM registry: every deskkit tool it names must exist, correctly namespaced, and the twelve
// its own body promises must all be there.
//
// Set equality (not mere membership) is deliberate: the agent's body states it "work[s] entirely
// through the twelve PM tools", so a silently missing tool contradicts the shipped prose exactly
// as loudly as an invented one. Narrowing the agent on purpose therefore means editing this list
// too, which is the review moment worth forcing. The sibling librarian-operator agent is generated
// and already pinned by scripts/check-persona-drift.mjs — not re-asserted here.
func TestBundlePMOperatorAgent_ScopedToRealPMTools(t *testing.T) {
	fm := readFrontmatter(t, filepath.Join("agents", "pm-operator.md"))
	if fm.Name != "pm-operator" {
		t.Errorf("agents/pm-operator.md: frontmatter name = %q, want %q", fm.Name, "pm-operator")
	}
	want := []string{"Read"}
	for _, n := range pmtools.ToolNames() {
		want = append(want, "mcp__"+bundleServer+"__"+n)
	}
	assertSameSet(t, "agents/pm-operator.md frontmatter tools", fm.Tools, want,
		"every deskkit tool the agent names must be a real PM tool under the bundle's mcp namespace")
}

// TestBundleNamespacedToolRefs_AreRealTools scans every authored agent and skill for
// mcp__<server>__<name> references and fails on any <name> that is not a real registered tool.
// This is the phantom-tool guard: an artifact instructing an agent to call a tool the mount does
// not serve.
//
// Deliberately NOT ported from the retired predecessor: its companion heuristic flagging BARE
// snake_case identifiers as suspected invented tool names. That guard needed a hand-maintained
// allowlist of every non-tool two-word field a skill may mention in prose; on the merged bundle
// that allowlist starts at ~17 entries (schema frontmatter keys such as date_created,
// original_type, secrets_ref, affects_workstreams, store fields such as status_label, by_court)
// and grows with every doc-field mention. It cannot distinguish a schema key from a tool, so it
// would fail on ordinary prose far more often than on a real phantom, and a guard that cries wolf
// gets deleted. The actionable failure mode — an artifact telling an agent to CALL a tool — always
// uses the namespaced form, which is checked above.
func TestBundleNamespacedToolRefs_AreRealTools(t *testing.T) {
	real := registeredToolNames()
	ref := regexp.MustCompile(`mcp__` + regexp.QuoteMeta(bundleServer) + `__([a-z_]+)`)
	for _, rel := range bundleArtifactPaths() {
		text := string(readBundleFile(t, rel))
		for _, m := range ref.FindAllStringSubmatch(text, -1) {
			if !real[m[1]] {
				t.Errorf("%s/%s: references tool %q, which no registered module serves — "+
					"a phantom tool name shipped in the bundle", bundleRel, rel, m[1])
			}
		}
	}
}

// TestBundleInventory_AgentsAndSkills asserts the bundle carries exactly its authored inventory —
// no missing, no extra — and that each artifact's frontmatter name matches its file/directory name
// (the name the harness loads it under).
func TestBundleInventory_AgentsAndSkills(t *testing.T) {
	root := filepath.Join(repoRootFrom(t), bundleRel)

	var onDiskAgents []string
	entries, err := os.ReadDir(filepath.Join(root, "agents"))
	if err != nil {
		t.Fatalf("read %s/agents: %v", bundleRel, err)
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			onDiskAgents = append(onDiskAgents, strings.TrimSuffix(e.Name(), ".md"))
		}
	}
	assertSameSet(t, bundleRel+"/agents", onDiskAgents, bundleAgents,
		"the bundle's agent inventory changed; update this list in the same change")

	var onDiskSkills []string
	entries, err = os.ReadDir(filepath.Join(root, "skills"))
	if err != nil {
		t.Fatalf("read %s/skills: %v", bundleRel, err)
	}
	for _, e := range entries {
		if e.IsDir() {
			onDiskSkills = append(onDiskSkills, e.Name())
		}
	}
	assertSameSet(t, bundleRel+"/skills", onDiskSkills, bundleSkills,
		"the bundle's skill inventory changed; update this list in the same change")

	for _, name := range bundleAgents {
		rel := filepath.Join("agents", name+".md")
		if fm := readFrontmatter(t, rel); fm.Name != name {
			t.Errorf("%s/%s: frontmatter name = %q, want %q (it must match the file name)", bundleRel, rel, fm.Name, name)
		}
	}
	for _, name := range bundleSkills {
		rel := filepath.Join("skills", name, "SKILL.md")
		if fm := readFrontmatter(t, rel); fm.Name != name {
			t.Errorf("%s/%s: frontmatter name = %q, want %q (it must match the directory name)", bundleRel, rel, fm.Name, name)
		}
	}
}

// --- helpers -------------------------------------------------------------------------------

// bundleArtifactPaths lists the bundle-relative markdown the tool-reference scan covers: every
// authored agent and skill instruction.
func bundleArtifactPaths() []string {
	var out []string
	for _, n := range bundleAgents {
		out = append(out, filepath.Join("agents", n+".md"))
	}
	for _, n := range bundleSkills {
		out = append(out, filepath.Join("skills", n, "SKILL.md"))
	}
	return out
}

// readBundleFile reads a bundle-relative file from the repo tree.
func readBundleFile(t *testing.T, rel string) []byte {
	t.Helper()
	p := filepath.Join(repoRootFrom(t), bundleRel, rel)
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read %s: %v", p, err)
	}
	return b
}

// artifactFrontmatter is the slice of an agent/skill's YAML frontmatter this guard reads.
type artifactFrontmatter struct {
	Name  string   `yaml:"name"`
	Tools []string `yaml:"tools"`
}

// readFrontmatter parses the leading `---` YAML block of a bundle-relative markdown file, using a
// real YAML parser because that is how the harness loading these files reads them.
func readFrontmatter(t *testing.T, rel string) artifactFrontmatter {
	t.Helper()
	s := string(readBundleFile(t, rel))
	if !strings.HasPrefix(s, "---\n") {
		t.Fatalf("%s/%s: no opening frontmatter fence", bundleRel, rel)
	}
	end := strings.Index(s[4:], "\n---")
	if end < 0 {
		t.Fatalf("%s/%s: unterminated frontmatter fence", bundleRel, rel)
	}
	var fm artifactFrontmatter
	if err := yaml.Unmarshal([]byte(s[4:4+end]), &fm); err != nil {
		t.Fatalf("%s/%s: frontmatter is not valid YAML: %v", bundleRel, rel, err)
	}
	return fm
}

// assertSameSet fails, naming the missing and the unexpected entries plus why it matters, when got
// and want are not the same set (order and duplicates ignored).
func assertSameSet(t *testing.T, label string, got, want []string, why string) {
	t.Helper()
	inWant := map[string]bool{}
	for _, w := range want {
		inWant[w] = true
	}
	inGot := map[string]bool{}
	for _, g := range got {
		inGot[g] = true
	}
	var missing, extra []string
	for w := range inWant {
		if !inGot[w] {
			missing = append(missing, w)
		}
	}
	for g := range inGot {
		if !inWant[g] {
			extra = append(extra, g)
		}
	}
	if len(missing) == 0 && len(extra) == 0 {
		return
	}
	sort.Strings(missing)
	sort.Strings(extra)
	t.Errorf("%s: set mismatch — missing %v, unexpected %v; %s", label, missing, extra, why)
}

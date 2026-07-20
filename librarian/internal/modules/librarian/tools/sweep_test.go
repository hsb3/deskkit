package tools

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDirKindForRoot(t *testing.T) {
	dirMap := map[string]string{"decision": "_structure/decisions", "task": "tasks", "analysis": "analyses", "journal": "journal"}
	if got := dirKindFor("CLAUDE.md", dirMap, "_meta/secrets"); got != "root" {
		t.Fatalf("root file: got %q, want root", got)
	}
}

func TestDirKindForNestedEntityDirs(t *testing.T) {
	// The nested-prefix fix (spec §5.1): a bare-top-segment match would miss this.
	dirMap := map[string]string{"decision": "_structure/decisions", "task": "tasks", "analysis": "analyses", "journal": "journal"}
	cases := map[string]string{
		"_structure/decisions/0014-foo.md": "decisions",
		"_structure/decisions":             "decisions", // exact dir match (rel == D)
		"tasks/x.md":                       "tasks",
		"analyses/y.md":                    "analyses",
		"journal/2026-07-15-z.md":          "journal",
		"_structuredecisions/evil.md":      "other", // NOT a real prefix match (no "/" boundary)
	}
	for rel, want := range cases {
		if got := dirKindFor(rel, dirMap, "_meta/secrets"); got != want {
			t.Errorf("dirKindFor(%q) = %q, want %q", rel, got, want)
		}
	}
}

func TestDirKindForMetaAndSecrets(t *testing.T) {
	dirMap := map[string]string{"decision": "_structure/decisions", "task": "tasks", "analysis": "analyses", "journal": "journal"}
	cases := map[string]string{
		"_meta/HANDOFF.md":         "meta",
		"_meta/secrets/creds.yaml": "meta", // under _meta/ AND under SECRETS_DIR
		"custom-secrets/key.txt":   "meta", // SECRETS_DIR configured elsewhere, still "meta"
	}
	for rel, want := range cases {
		if got := dirKindFor(rel, dirMap, "custom-secrets"); got != want {
			t.Errorf("dirKindFor(%q) = %q, want %q", rel, got, want)
		}
	}
}

func TestDirKindForMemoryPrecedence(t *testing.T) {
	// Reproduces the PoC's exact operator precedence: memory OR (.claude AND /memory/).
	dirMap := map[string]string{}
	cases := map[string]string{
		"memory/foo.md":              "memory",
		".claude/memory/bar.md":      "memory",
		".claude/other/bar.md":       "infra", // .claude without /memory/ -> dotted infra dir
		"somewhere/memory/README.md": "other", // "memory" only matches as the TOP segment
	}
	for rel, want := range cases {
		if got := dirKindFor(rel, dirMap, ""); got != want {
			t.Errorf("dirKindFor(%q) = %q, want %q", rel, got, want)
		}
	}
}

// TestDirKindForInfra pins the "infra" bucket (issue #18): non-entity content under a dotted
// top-level dir is infrastructure, not misfiled desk content. The memory special-case still
// wins for .claude/memory/**, and a non-dotted "other" dir is unaffected.
func TestDirKindForInfra(t *testing.T) {
	dirMap := map[string]string{}
	cases := map[string]string{
		".claude/agents/reviewer.md": "infra",
		".agents/skills/x.md":        "infra",
		".github/workflows/ci.yml":   "infra",
		".claude/memory/note.md":     "memory", // memory precedence still wins
		"scratch/notes.md":           "other",  // non-dotted, non-entity -> still other
		".gitignore":                 "root",   // bare dotted FILE at root -> root (no "/"), not infra
		".env":                       "root",   // infra bucket catches dotted DIRS, not root dotfiles
	}
	for rel, want := range cases {
		if got := dirKindFor(rel, dirMap, ""); got != want {
			t.Errorf("dirKindFor(%q) = %q, want %q", rel, got, want)
		}
	}
}

func TestDirKindForLongestEntityDirWins(t *testing.T) {
	// A deliberately overlapping config: the more specific (longer) path wins.
	dirMap := map[string]string{"decision": "docs", "task": "docs/tasks"}
	if got := dirKindFor("docs/tasks/x.md", dirMap, ""); got != "tasks" {
		t.Fatalf("longest-prefix-wins: got %q, want tasks", got)
	}
}

func TestIssueRefFindWbPrefix(t *testing.T) {
	if got := issueRefFind("see wb#37 for details"); got != "wb#37" {
		t.Fatalf("got %q, want wb#37", got)
	}
}

func TestIssueRefFindBareHash(t *testing.T) {
	if got := issueRefFind("filed as #111 on the board"); got != "#111" {
		t.Fatalf("got %q, want #111", got)
	}
}

func TestIssueRefFindRejectsWordPrecededHash(t *testing.T) {
	// (?<![\w&]) lookbehind reimplementation: a #N glued to a preceding word char must NOT match.
	if got := issueRefFind("release20260716#42 shipped"); got != "" {
		t.Fatalf("word-preceded #N must not match, got %q", got)
	}
}

func TestIssueRefFindRejectsAmpersandPrecededHash(t *testing.T) {
	if got := issueRefFind("foo &#123 bar"); got != "" {
		t.Fatalf("&-preceded #N must not match, got %q", got)
	}
}

func TestIssueRefFindWbInsideDoesNotAlsoMatchBareHash(t *testing.T) {
	// "wb#123": the wb# alternative wins at the earlier start position; the bare-# reading
	// of the same substring must never be separately reported.
	if got := issueRefFind("track wb#123 here"); got != "wb#123" {
		t.Fatalf("got %q, want wb#123", got)
	}
}

func TestIssueRefFindLeftmostAcrossAlternatives(t *testing.T) {
	// Bare #5 appears before wb#9 in the text -> #5 (leftmost) wins over wb#9.
	if got := issueRefFind("ref #5 and also wb#9"); got != "#5" {
		t.Fatalf("got %q, want #5 (leftmost match)", got)
	}
}

func TestIssueRefFindNoMatch(t *testing.T) {
	if got := issueRefFind("nothing to see here"); got != "" {
		t.Fatalf("got %q, want empty", got)
	}
}

func TestGHURLRe(t *testing.T) {
	if got := ghURLRe.FindString("see https://github.com/acme/example-repo/issues/1 for context"); got != "https://github.com/acme/example-repo/issues/1" {
		t.Fatalf("got %q", got)
	}
	if got := ghURLRe.FindString("no url here"); got != "" {
		t.Fatalf("expected no match, got %q", got)
	}
}

func TestLineCount(t *testing.T) {
	cases := map[string]int{
		"":                          1,
		"a":                         1,
		"a\n":                       2,
		"a\nb\nc":                   3,
		"a\nb\nc\n":                 4,
		"---\ntype: x\n---\nbody\n": 5,
	}
	for text, want := range cases {
		if got := lineCount(text); got != want {
			t.Errorf("lineCount(%q) = %d, want %d", text, got, want)
		}
	}
}

func TestPathOrSubtree(t *testing.T) {
	cases := []struct {
		rel, target string
		want        bool
	}{
		{"_structure/decisions/0001-foo.md", "_structure/decisions", true},
		{"_structure/decisions", "_structure/decisions", true},
		{"_structure/decisionsXYZ.md", "_structure/decisions", false},
		{"tasks/x.md", "_structure/decisions", false},
	}
	for _, c := range cases {
		if got := pathOrSubtree(c.rel, c.target); got != c.want {
			t.Errorf("pathOrSubtree(%q, %q) = %v, want %v", c.rel, c.target, got, c.want)
		}
	}
}

func TestFindingDedupeKey(t *testing.T) {
	open := map[findingKey]bool{
		findingDedupeKey("tasks/x.md", "R1", "abc123"): true,
	}
	if !isDuplicateFinding(open, "tasks/x.md", "R1", "abc123") {
		t.Fatalf("expected duplicate for identical (path, rule, checksum)")
	}
	if isDuplicateFinding(open, "tasks/x.md", "R1", "def456") {
		t.Fatalf("a changed checksum must NOT dedupe (content changed since flagging)")
	}
	if isDuplicateFinding(open, "tasks/x.md", "R2", "abc123") {
		t.Fatalf("a different rule must NOT dedupe")
	}
	if isDuplicateFinding(open, "tasks/y.md", "R1", "abc123") {
		t.Fatalf("a different path must NOT dedupe")
	}
}

func TestBasename(t *testing.T) {
	if got := basename("journal/2026-07-15-foo.md"); got != "2026-07-15-foo.md" {
		t.Fatalf("got %q", got)
	}
	if got := basename("CLAUDE.md"); got != "CLAUDE.md" {
		t.Fatalf("got %q", got)
	}
}

func TestFmStr(t *testing.T) {
	fm := map[string]any{"type": "decision", "tags": []string{"a", "b"}}
	if got := fmStr(fm, "type"); got != "decision" {
		t.Fatalf("got %q", got)
	}
	if got := fmStr(fm, "tags"); got != "" {
		t.Fatalf("non-string value must reduce to empty string, got %q", got)
	}
	if got := fmStr(fm, "missing"); got != "" {
		t.Fatalf("missing key must reduce to empty string, got %q", got)
	}
}

// --- graduated_to / R5 explicit-marker gating ---
//
// NEUTRALITY: issue refs used as test DATA are assembled at runtime so no literal `#\d+`
// token appears in this source (scripts/check-neutrality.mjs family 2a). The scanner also
// whole-file-exempts this test, but building refs at runtime keeps the file clean regardless.
const (
	refHash = "#"    // bare-issue-ref sigil, kept split from its digits at the source level
	refN    = "111"  // the sample issue number
	refWB   = "wb#"  // the wb-prefixed alternative sigil
	refWBN  = "37"   // the wb sample number
)

// scanTempMD writes content to name under a throwaway desk root and returns the scanned
// fileRow. The temp root is not a git repo, so Origin / GitLastCommit resolve to "".
func scanTempMD(t *testing.T, name, content string) fileRow {
	t.Helper()
	root := t.TempDir()
	abs := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	dirMap := map[string]string{"decision": "_structure/decisions", "task": "tasks", "analysis": "analyses", "journal": "journal"}
	row, err := scanFile(root, name, dirMap, "_meta/secrets", "testdesk")
	if err != nil {
		t.Fatalf("scanFile(%q): %v", name, err)
	}
	return row
}

func padLines(n int) string {
	var b string
	for i := 0; i < n; i++ {
		b += "padding line of prose\n"
	}
	return b
}

// TestGraduationMarker is the pure gate: an explicit frontmatter key or a canonical inline
// `graduated to` line is a marker; a bare #N anywhere in prose is NOT.
func TestGraduationMarker(t *testing.T) {
	hash := refHash + refN // "#111"
	wb := refWB + refWBN   // "wb#37"
	url := "https://" + "github.com/o/r/issues/9"
	cases := []struct {
		name, text, want string
	}{
		{"frontmatter key, quoted", "---\ntype: analysis\ngraduated_to: \"" + hash + "\"\n---\nbody\n", hash},
		{"frontmatter key, wb value", "---\ngraduated_to: \"" + wb + "\"\n---\nbody\n", wb},
		{"frontmatter key wins over inline", "---\ngraduated_to: \"" + hash + "\"\n---\ngraduated to " + wb + "\n", hash},
		{"inline canonical line, no colon", "no frontmatter\ngraduated to " + hash + "\n", hash},
		{"inline canonical line, with colon", "graduated to: " + wb + "\n", wb},
		{"inline canonical line, url", "graduated to: " + url + "\n", url},
		{"inline canonical line, bare number is opaque pointer text (spec-verbatim)", "graduated to: 42\n", "42"},
		{"bare ref in prose is NOT a marker", "this doc references " + hash + " as evidence\n", ""},
		{"graduated-to mid-sentence is NOT a marker", "the plan graduated to " + hash + " earlier this week\n", ""},
		{"no marker at all", "---\ntype: analysis\n---\njust prose, nothing filed\n", ""},
		{"empty", "", ""},
	}
	for _, c := range cases {
		if got := graduationMarker(c.text); got != c.want {
			t.Errorf("%s: graduationMarker = %q, want %q", c.name, got, c.want)
		}
	}
}

// TestScanFileNoMarkerNoGraduatedTo is the #92/#78 RED regression: a short doc that merely
// CITES an issue number in prose — with NO graduation marker — must NOT populate graduated_to,
// and (being short) must not fire R5. This FAILS on the pre-fix leftmost-bare-#N heuristic
// (which set graduated_to to the quoted ref) and PASSES once graduated_to is marker-gated.
func TestScanFileNoMarkerNoGraduatedTo(t *testing.T) {
	prose := "This analysis references filed issue " + refHash + refN + " as supporting evidence.\n"
	content := "---\ntype: analysis\ncreated: 2026-07-15\nupdated: 2026-07-15\ntags: []\nsynopsis: \"cites an issue as evidence\"\n---\n" + prose
	row := scanTempMD(t, "analyses/cite-as-evidence.md", content)
	if row.GraduatedTo != "" {
		t.Fatalf("a doc that only CITES %s in prose must leave graduated_to empty, got %q", refHash+refN, row.GraduatedTo)
	}
	if _, _, hit := r5Check(row.DirKind, content); hit {
		t.Fatalf("R5 must not fire on a doc that only cites an issue as evidence")
	}
}

// TestScanFileFrontmatterMarkerSetsGraduatedToAndR5Fires is the regression that an EXPLICIT
// `graduated_to:` frontmatter marker populates graduated_to regardless of length, and that a
// graduated-but-still-long doc (>40 lines) fires R5 as designed.
func TestScanFileFrontmatterMarkerSetsGraduatedToAndR5Fires(t *testing.T) {
	marker := refHash + refN
	content := "---\ntype: analysis\ncreated: 2026-07-15\nupdated: 2026-07-15\ntags: []\nsynopsis: \"graduated but not collapsed\"\ngraduated_to: \"" + marker + "\"\n---\n" + padLines(45)
	row := scanTempMD(t, "analyses/graduated-not-collapsed.md", content)
	if row.GraduatedTo != marker {
		t.Fatalf("frontmatter graduated_to marker must populate the column, got %q want %q", row.GraduatedTo, marker)
	}
	if _, _, hit := r5Check(row.DirKind, content); !hit {
		t.Fatalf("R5 must fire on a graduated (marker present) doc that is still >40 lines")
	}
}

// TestScanFileInlineMarkerSetsGraduatedTo pins the canonical inline `graduated to <ref>` line
// (the verify.sh F-R5 fixture shape). It must populate graduated_to and, on a >40-line doc,
// fire R5 — this is what keeps the integration gate's F-R5 fixture firing once R5 is wired to
// the shared marker gate.
func TestScanFileInlineMarkerSetsGraduatedTo(t *testing.T) {
	marker := refHash + refN
	body := "graduated to " + marker + "\n" + padLines(45)
	content := "---\ntype: analysis\ncreated: 2026-07-15\nupdated: 2026-07-15\ntags: []\nsynopsis: \"graduated stub\"\n---\n" + body
	row := scanTempMD(t, "analyses/inline-graduated.md", content)
	if row.GraduatedTo != marker {
		t.Fatalf("canonical inline `graduated to` line must populate the column, got %q want %q", row.GraduatedTo, marker)
	}
	if _, _, hit := r5Check(row.DirKind, content); !hit {
		t.Fatalf("R5 must fire on an inline-marked graduated doc that is still >40 lines")
	}
}

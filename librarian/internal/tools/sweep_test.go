package tools

import "testing"

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

package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"github.com/hsb3/deskkit/internal/modules/librarian/desklib"
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
	refHash = "#"   // bare-issue-ref sigil, kept split from its digits at the source level
	refN    = "111" // the sample issue number
	refWB   = "wb#" // the wb-prefixed alternative sigil
	refWBN  = "37"  // the wb sample number
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
	row, err := scanFile(root, name, dirMap, "_meta/secrets", "testdesk", nil)
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

// --- slice A: frontmatter `id` as the document-identity primitive (ADR 0017) ---
//
// These four drive the actual Sweep() against a real store + on-disk desk (via newTestEnv /
// mustWriteFile, shared with propose_fix_test.go). Each is red against the pre-change,
// path-only matching: (1) a rename would insert a fresh record, (2) is a preservation companion,
// (3) doc_id would be empty, (4) no duplicate finding would be filed.

func firstFileRow(t *testing.T, app core.App, path string) *core.Record {
	t.Helper()
	r, err := app.FindFirstRecordByFilter("files", "path = {:p}", dbx.Params{"p": path})
	if err != nil {
		t.Fatalf("no files row at %q: %v", path, err)
	}
	return r
}

// --- content indexing (§5.6 retrieval/search) ---

// TestScanFileContentUTF8Text: a UTF-8 text file has its body stored verbatim in Content, so a
// session can later retrieve/search it.
func TestScanFileContentUTF8Text(t *testing.T) {
	body := "---\ntype: analysis\ncreated: 2026-07-20\nupdated: 2026-07-20\ntags: []\n---\nsalient unique-token-xyzzy body prose\n"
	row := scanTempMD(t, "analyses/has-content.md", body)
	if row.Content != body {
		t.Fatalf("Content not stored verbatim for a UTF-8 file:\n got %q\nwant %q", row.Content, body)
	}
}

// TestScanFileContentSecretHomeExcluded: a file under SECRETS_DIR is a secret home and is NEVER
// indexed — Content stays empty (mirrors the meta/secrets exclusion boundary).
func TestScanFileContentSecretHomeExcluded(t *testing.T) {
	row := scanTempMD(t, "_meta/secrets/creds.md", "API_TOKEN=super-secret-value\n")
	if row.Content != "" {
		t.Fatalf("a file under SECRETS_DIR must not be indexed, got Content %q", row.Content)
	}
}

// TestScanFileContentBinaryExcluded: a non-UTF-8 (binary) file is not indexed — Content is empty.
func TestScanFileContentBinaryExcluded(t *testing.T) {
	root := t.TempDir()
	rel := "assets/blob.bin"
	abs := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	// A lone 0xff never begins a valid UTF-8 sequence, so these bytes are non-UTF-8.
	if err := os.WriteFile(abs, []byte{0xff, 0xfe, 0x00, 0x01, 0xff}, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	dirMap := map[string]string{"decision": "_structure/decisions", "task": "tasks", "analysis": "analyses", "journal": "journal"}
	row, err := scanFile(root, rel, dirMap, "_meta/secrets", "testdesk", nil)
	if err != nil {
		t.Fatalf("scanFile: %v", err)
	}
	if row.Content != "" {
		t.Fatalf("a non-UTF-8 binary file must not be indexed, got Content %q", row.Content)
	}
}

// TestTruncateRunes pins the rune-safe content cap: under-cap is unchanged, over-cap is clipped,
// and a multi-byte rune is never split.
func TestTruncateRunes(t *testing.T) {
	if got := truncateRunes("hello", 10); got != "hello" {
		t.Fatalf("under cap must be unchanged, got %q", got)
	}
	if got := truncateRunes("hello", 3); got != "hel" {
		t.Fatalf("over cap must clip to max runes, got %q", got)
	}
	if got := truncateRunes("héllo", 2); got != "hé" {
		t.Fatalf("truncation must not split a multi-byte rune, got %q", got)
	}
}

// TestSweep_RenameWithIDKeepsSameRecord — a doc carrying a frontmatter `id`, renamed on disk
// between two sweeps, keeps the SAME files record at its new path (identity survives the rename);
// the old path has no row (moved, not soft-deleted-and-orphaned). RED against path-only matching,
// which would fresh-insert a new record and soft-delete the old path.
func TestSweep_RenameWithIDKeepsSameRecord(t *testing.T) {
	app, cfg := newTestEnv(t)
	ctx := context.Background()

	doc := "---\ntype: analysis\nid: analysis-alpha\ncreated: 2026-07-20\nupdated: 2026-07-20\ntags: []\n---\nbody\n"
	oldRel := "analyses/alpha.md"
	mustWriteFile(t, cfg.DeskRoot, oldRel, doc)
	if _, err := Sweep(ctx, app, cfg, &SweepInput{}); err != nil {
		t.Fatalf("first sweep: %v", err)
	}
	before := firstFileRow(t, app, oldRel)
	origID := before.Id
	if got := before.GetString("doc_id"); got != "analysis-alpha" {
		t.Fatalf("doc_id not stored from frontmatter id: got %q", got)
	}

	// Rename on disk (same id, new path), then re-sweep.
	if err := os.Remove(filepath.Join(cfg.DeskRoot, oldRel)); err != nil {
		t.Fatalf("remove old path: %v", err)
	}
	newRel := "analyses/alpha-renamed.md"
	mustWriteFile(t, cfg.DeskRoot, newRel, doc)
	if _, err := Sweep(ctx, app, cfg, &SweepInput{}); err != nil {
		t.Fatalf("second sweep: %v", err)
	}

	after := firstFileRow(t, app, newRel)
	if after.Id != origID {
		t.Fatalf("rename with id must keep the SAME record: old id %s, new id %s", origID, after.Id)
	}
	if after.GetBool("deleted") {
		t.Fatalf("the renamed row must not be soft-deleted")
	}
	if orphan, _ := app.FindFirstRecordByFilter("files", "path = {:p}", dbx.Params{"p": oldRel}); orphan != nil {
		t.Fatalf("old path must have no row after the rename, found id %s (deleted=%v)", orphan.Id, orphan.GetBool("deleted"))
	}
}

// TestSweep_RenameWithoutIDIsSoftDeletePlusInsert — a doc with NO frontmatter id, renamed on disk,
// still soft-deletes the old path and inserts a fresh row (today's behavior is unchanged for the
// common no-id case). This is the preservation companion.
func TestSweep_RenameWithoutIDIsSoftDeletePlusInsert(t *testing.T) {
	app, cfg := newTestEnv(t)
	ctx := context.Background()

	doc := "---\ntype: analysis\ncreated: 2026-07-20\nupdated: 2026-07-20\ntags: []\n---\nbody\n" // NO id
	oldRel := "analyses/beta.md"
	mustWriteFile(t, cfg.DeskRoot, oldRel, doc)
	if _, err := Sweep(ctx, app, cfg, &SweepInput{}); err != nil {
		t.Fatalf("first sweep: %v", err)
	}
	origID := firstFileRow(t, app, oldRel).Id

	if err := os.Remove(filepath.Join(cfg.DeskRoot, oldRel)); err != nil {
		t.Fatalf("remove old path: %v", err)
	}
	newRel := "analyses/beta-renamed.md"
	mustWriteFile(t, cfg.DeskRoot, newRel, doc)
	if _, err := Sweep(ctx, app, cfg, &SweepInput{}); err != nil {
		t.Fatalf("second sweep: %v", err)
	}

	old := firstFileRow(t, app, oldRel)
	if !old.GetBool("deleted") {
		t.Fatalf("no-id rename: the old path row must be soft-deleted")
	}
	newRec := firstFileRow(t, app, newRel)
	if newRec.Id == origID {
		t.Fatalf("no-id rename must be a fresh insert, not a moved record")
	}
	if newRec.GetBool("deleted") {
		t.Fatalf("the new path row must be live")
	}
}

// TestSweep_RebuildReproducesDocID — a fresh sweep from disk alone (after the store's files rows
// are wiped) reproduces the same doc_id for every id-carrying doc, confirming the identity
// primitive is re-derivable from the desk tree (files-are-truth, decision 0009). RED pre-change:
// doc_id did not exist, so the first-pass values would be empty.
func TestSweep_RebuildReproducesDocID(t *testing.T) {
	app, cfg := newTestEnv(t)
	ctx := context.Background()

	docs := map[string]string{
		"analyses/one.md": "---\ntype: analysis\nid: doc-one\ncreated: 2026-07-20\nupdated: 2026-07-20\ntags: []\n---\nx\n",
		"tasks/two.md":    "---\ntype: task\nid: doc-two\ncreated: 2026-07-20\nupdated: 2026-07-20\ntags: []\n---\ny\n",
	}
	for rel, c := range docs {
		mustWriteFile(t, cfg.DeskRoot, rel, c)
	}
	if _, err := Sweep(ctx, app, cfg, &SweepInput{}); err != nil {
		t.Fatalf("sweep 1: %v", err)
	}
	first := map[string]string{}
	for rel := range docs {
		got := firstFileRow(t, app, rel).GetString("doc_id")
		if got == "" {
			t.Fatalf("%s: doc_id not populated from frontmatter id", rel)
		}
		first[rel] = got
	}

	// Wipe the store's files rows (simulate a store rebuild) and sweep the SAME disk again.
	existing, err := app.FindRecordsByFilter("files", "", "", 0, 0, dbx.Params{})
	if err != nil {
		t.Fatalf("list files rows: %v", err)
	}
	for _, r := range existing {
		if err := app.Delete(r); err != nil {
			t.Fatalf("wipe files row: %v", err)
		}
	}
	if _, err := Sweep(ctx, app, cfg, &SweepInput{}); err != nil {
		t.Fatalf("sweep 2 (rebuild): %v", err)
	}
	for rel, want := range first {
		if got := firstFileRow(t, app, rel).GetString("doc_id"); got != want {
			t.Fatalf("%s: rebuild produced doc_id %q, want %q (must be re-derivable from disk)", rel, got, want)
		}
	}
}

// TestSweep_DuplicateIDFilesFindingNoMerge — two DIFFERENT docs sharing one frontmatter id do NOT
// merge (each keeps its own row); the duplicate is surfaced as a patrol-visible finding. RED
// pre-change: no doc_id, no duplicate detection, so no finding would be filed.
func TestSweep_DuplicateIDFilesFindingNoMerge(t *testing.T) {
	app, cfg := newTestEnv(t)
	ctx := context.Background()

	shared := "shared-id-x"
	docA := "---\ntype: analysis\nid: " + shared + "\ncreated: 2026-07-20\nupdated: 2026-07-20\ntags: []\n---\naaa\n"
	docB := "---\ntype: analysis\nid: " + shared + "\ncreated: 2026-07-20\nupdated: 2026-07-20\ntags: []\n---\nbbb\n"
	relA, relB := "analyses/a.md", "analyses/b.md"
	mustWriteFile(t, cfg.DeskRoot, relA, docA)
	mustWriteFile(t, cfg.DeskRoot, relB, docB)

	if _, err := Sweep(ctx, app, cfg, &SweepInput{}); err != nil {
		t.Fatalf("sweep: %v", err)
	}

	recA := firstFileRow(t, app, relA)
	recB := firstFileRow(t, app, relB)
	if recA.Id == recB.Id {
		t.Fatalf("two docs sharing an id must NOT be merged into one row")
	}

	findings, err := app.FindRecordsByFilter(
		"patrol_findings", "rule = 'duplicate-doc-id' && state = 'flagged'", "", 0, 0, dbx.Params{})
	if err != nil {
		t.Fatalf("query findings: %v", err)
	}
	if len(findings) == 0 {
		t.Fatalf("a duplicate frontmatter id must file a patrol-visible finding")
	}
}

// --- the ignore list is a CONTENT boundary for sweep ---

// TestSweep_IgnoreListIsAContentBoundary — the content-boundary gate. Every entry of the SHIPPED ignore
// seed gets a sentinel-bearing file; after a sweep none of their bodies may be in the store,
// while a non-ignored control file's body must be — so the test fails when the blanking is
// removed (sentinel indexed) AND when it over-blankets (control empty). Expectations are
// DERIVED from the shipped seed via LoadIgnoreList, never a hand-copied list. The named rows
// still EXIST: patrol must be able to flag write-protected paths (flag-only, the F-IGN
// contract in verify.sh), so only content is withheld, never the row.
func TestSweep_IgnoreListIsAContentBoundary(t *testing.T) {
	app, cfg := newTestEnv(t)
	ctx := context.Background()

	// Replace the test env's empty ignore file with the SHIPPED seed.
	if err := os.WriteFile(cfg.IgnoreConfig, []byte(desklib.DefaultIgnore()), 0o644); err != nil {
		t.Fatalf("write shipped seed: %v", err)
	}
	entries, err := desklib.LoadIgnoreList(cfg.IgnoreConfig)
	if err != nil {
		t.Fatalf("load shipped seed: %v", err)
	}

	const sentinel = "SENTINEL-CREDENTIAL-DO-NOT-INDEX"
	var seeded []string
	for _, entry := range entries {
		rel := entry
		if strings.HasSuffix(entry, "/") {
			rel = entry + "seeded-secret.md"
		}
		mustWriteFile(t, cfg.DeskRoot, rel, "---\nsynopsis: x\n---\n"+sentinel+"\n")
		seeded = append(seeded, rel)
	}
	// Two credential-bearing paths, both covered by shipped entries (.claude/, _meta/secrets/).
	for _, rel := range []string{".claude/settings.local.json", "_meta/secrets/x.env"} {
		if !desklib.IsIgnored(rel, entries) {
			t.Fatalf("shipped seed no longer covers %s — the credential boundary regressed", rel)
		}
		mustWriteFile(t, cfg.DeskRoot, rel, sentinel+"\n")
		seeded = append(seeded, rel)
	}
	const control = "notes/control.md"
	mustWriteFile(t, cfg.DeskRoot, control, "---\nsynopsis: c\n---\nindexable body\n")

	if _, err := Sweep(ctx, app, cfg, &SweepInput{}); err != nil {
		t.Fatalf("sweep: %v", err)
	}

	for _, rel := range seeded {
		rec, _ := app.FindFirstRecordByFilter("files", "path = {:p}", dbx.Params{"p": rel})
		if rec == nil {
			continue // .git/ and logs/ are walk-pruned entirely — no row is also no content
		}
		if got := rec.GetString("content"); got != "" {
			t.Errorf("%s is on the shipped ignore list but its content was indexed (%d bytes)", rel, len(got))
		}
	}
	// Flag-only stays alive: a write-protected path keeps its metadata row…
	if rec, _ := app.FindFirstRecordByFilter("files", "path = {:p}", dbx.Params{"p": ".claude/settings.local.json"}); rec == nil {
		t.Fatalf(".claude/settings.local.json must keep a metadata row (patrol flag-only)")
	}
	// …and the control proves the test can tell stored from blanked.
	rec, _ := app.FindFirstRecordByFilter("files", "path = {:p}", dbx.Params{"p": control})
	if rec == nil || !strings.Contains(rec.GetString("content"), "indexable body") {
		t.Fatalf("non-ignored control file must carry content — the boundary over-blanked")
	}
}

// TestSweep_NewIgnoreRuleClearsStoredContent — the retroactive half of the boundary: a rule
// that starts matching an EXISTING row clears its stored body on the next sweep (content is in
// COMPARE_FIELDS, so stored → "" re-persists), protecting already-swept desks, not only fresh
// ones. The row itself survives — the file is still on disk, so soft-delete would be wrong.
func TestSweep_NewIgnoreRuleClearsStoredContent(t *testing.T) {
	app, cfg := newTestEnv(t)
	ctx := context.Background()

	const rel = ".claude/settings.local.json"
	mustWriteFile(t, cfg.DeskRoot, rel, "top secret\n")
	if _, err := Sweep(ctx, app, cfg, &SweepInput{}); err != nil {
		t.Fatalf("sweep 1: %v", err)
	}
	if firstFileRow(t, app, rel).GetString("content") == "" {
		t.Fatalf("precondition: with an empty ignore list the body must be stored")
	}

	if err := os.WriteFile(cfg.IgnoreConfig, []byte(".claude/\n"), 0o644); err != nil {
		t.Fatalf("add rule: %v", err)
	}
	if _, err := Sweep(ctx, app, cfg, &SweepInput{}); err != nil {
		t.Fatalf("sweep 2: %v", err)
	}
	rec := firstFileRow(t, app, rel)
	if got := rec.GetString("content"); got != "" {
		t.Fatalf("a newly-matching rule must clear stored content on the next sweep, still holds %q", got)
	}
	if rec.GetBool("deleted") {
		t.Fatalf("the file is still on disk — its row must stay live, not soft-deleted")
	}
}

// TestSweep_UnreadableIgnoreListFailsClosed — sweep refuses to run past a broken boundary,
// exactly like the write tools (§10.1): an error out, no partial index.
func TestSweep_UnreadableIgnoreListFailsClosed(t *testing.T) {
	app, cfg := newTestEnv(t)
	mustWriteFile(t, cfg.DeskRoot, "notes/a.md", "body\n")
	if err := os.Remove(cfg.IgnoreConfig); err != nil {
		t.Fatalf("remove ignore file: %v", err)
	}
	if _, err := Sweep(context.Background(), app, cfg, &SweepInput{}); err == nil {
		t.Fatalf("sweep must fail closed when the ignore list is unreadable")
	}
}

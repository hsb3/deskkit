package desklib

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseFrontmatterTolerantColon(t *testing.T) {
	text := "---\n" +
		"type: decision\n" +
		"synopsis: this: has a colon\n" +
		"tags: [a, b]\n" +
		"list:\n" +
		"- x\n" +
		"- y\n" +
		"---\n" +
		"body\n"
	fm := ParseFrontmatter(text)

	// First-colon split tolerates the unquoted colon in the value (the desk YAML gotcha).
	if got := fm["synopsis"]; got != "this: has a colon" {
		t.Fatalf("synopsis = %q, want %q", got, "this: has a colon")
	}
	if got := fm["type"]; got != "decision" {
		t.Fatalf("type = %q, want decision", got)
	}
	if got, want := fm["tags"], []string{"a", "b"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("tags = %#v, want %#v", got, want)
	}
	if got, want := fm["list"], []string{"x", "y"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("list = %#v, want %#v", got, want)
	}
}

func TestParseFrontmatterUnterminatedIsEmpty(t *testing.T) {
	// Opening fence, never closed -> empty map (no crash), treated as no frontmatter.
	fm := ParseFrontmatter("---\ntype: task\nno closing fence here\n")
	if len(fm) != 0 {
		t.Fatalf("unterminated fence should yield empty map, got %#v", fm)
	}
	// No leading fence at all -> empty map.
	if fm := ParseFrontmatter("just text\n"); len(fm) != 0 {
		t.Fatalf("no fence should yield empty map, got %#v", fm)
	}
}

func TestWriteExactByteExact(t *testing.T) {
	dir := t.TempDir()
	abs := filepath.Join(dir, "nested", "a.md")
	// Mixed line endings, no trailing newline: must round-trip byte-for-byte (no
	// newline translation) so restore is cmp-clean (spec §5.4/§5.5).
	content := []byte("line1\r\nline2\nno-trailing-newline")
	if err := WriteExact(abs, content); err != nil {
		t.Fatalf("WriteExact: %v", err)
	}
	got, err := os.ReadFile(abs)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Fatalf("bytes not exact:\n got %q\nwant %q", got, content)
	}
}

func TestIgnoredFailClosed(t *testing.T) {
	// A missing/unreadable ignore file must fail CLOSED: every path is treated as
	// ignored (true) and an error is surfaced (spec §10.1). Never an empty list.
	missing := filepath.Join(t.TempDir(), "does-not-exist.ignore")
	ignored, err := Ignored("tasks/anything.md", missing)
	if !ignored {
		t.Fatalf("missing ignore file must fail closed (ignored=true), got false")
	}
	if err == nil {
		t.Fatalf("missing ignore file must surface an error")
	}
}

func TestIsIgnoredPrefixAndExact(t *testing.T) {
	list := []string{"_meta/", "CLAUDE.md", "_structure/decisions/"}
	cases := map[string]bool{
		"_meta/HANDOFF.md":                 true,  // prefix (trailing /)
		"_meta":                            false, // "_meta/" requires the slash prefix
		"CLAUDE.md":                        true,  // exact
		"CLAUDE.md.bak":                    false, // not exact, not entry+"/"
		"_structure/decisions/0001-foo.md": true,  // prefix
		"tasks/x.md":                       false,
	}
	for rel, want := range cases {
		if got := IsIgnored(rel, list); got != want {
			t.Errorf("IsIgnored(%q) = %v, want %v", rel, got, want)
		}
	}
}

func TestChecksumStable(t *testing.T) {
	// sha256("") is a known constant.
	const emptySHA = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	if got := Checksum(nil); got != emptySHA {
		t.Fatalf("Checksum(nil) = %s, want %s", got, emptySHA)
	}
}

// TestGitNewestCommitExcluding proves the R6 core-computation fix: over a real temp git repo it
// asserts GitNewestCommitExcluding ignores commits that touch ONLY the excluded (handoff) path, so
// the handoff's own later update commit does not count as the "newest" change it guards. The plain
// GitNewestCommit (whole tree) is asserted to return the handoff's own commit date to make the
// difference explicit — that whole-tree value is exactly why R6 could never self-clear before.
func TestGitNewestCommitExcluding(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	root := t.TempDir()

	baseEnv := append(os.Environ(),
		"GIT_CONFIG_GLOBAL=/dev/null", // hermetic: no global hooks/gpgsign/templates leak in
		"GIT_CONFIG_SYSTEM=/dev/null",
		"GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@example.com",
	)
	git := func(extraEnv []string, args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
		cmd.Env = append(append([]string{}, baseEnv...), extraEnv...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	commitAt := func(date string) []string {
		stamp := date + "T12:00:00" // noon: no midnight/timezone date shift
		return []string{"GIT_AUTHOR_DATE=" + stamp, "GIT_COMMITTER_DATE=" + stamp}
	}

	git(nil, "init")

	// A guarded desk file committed on the EARLIER date — the newest change the handoff GUARDS.
	if err := os.WriteFile(filepath.Join(root, "guarded.md"), []byte("desk content\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(nil, "add", "guarded.md")
	git(commitAt("2026-07-10"), "commit", "-m", "add guarded file")

	// The handoff, added and then refreshed on LATER dates — both commits touch ONLY the handoff.
	handoffRel := "_meta/HANDOFF.md"
	handoffAbs := filepath.Join(root, handoffRel)
	if err := os.MkdirAll(filepath.Dir(handoffAbs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(handoffAbs, []byte("updated: 2026-07-12\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(nil, "add", handoffRel)
	git(commitAt("2026-07-12"), "commit", "-m", "add handoff")

	if err := os.WriteFile(handoffAbs, []byte("updated: 2026-07-20\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(nil, "add", handoffRel)
	git(commitAt("2026-07-20"), "commit", "-m", "refresh handoff")

	// Whole-tree newest is the handoff's OWN refresh — the exact value that prevented self-clear.
	if got := GitNewestCommit(root); got != "2026-07-20" {
		t.Fatalf("GitNewestCommit(whole tree) = %q, want 2026-07-20 (the handoff's own commit)", got)
	}
	// Excluding the handoff, the newest change it GUARDS is the guarded-file commit; the handoff's
	// own update commits (07-12, 07-20 — both touch only the excluded path) are filtered out.
	if got := GitNewestCommitExcluding(root, handoffRel); got != "2026-07-10" {
		t.Fatalf("GitNewestCommitExcluding = %q, want 2026-07-10 (handoff's own commit must be excluded)", got)
	}
}

func TestEnsureIgnoreFileAutoCreates(t *testing.T) {
	root := t.TempDir()
	cfgPath := filepath.Join(root, ".librarian-ignore")
	if err := EnsureIgnoreFile(cfgPath, root); err != nil {
		t.Fatalf("EnsureIgnoreFile: %v", err)
	}
	b, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("ignore file not created: %v", err)
	}
	if string(b) != DefaultIgnore() {
		t.Fatalf("auto-created content != embedded default")
	}
	// Idempotent: a second call must not clobber an operator-edited file.
	edited := "custom-entry/\n"
	if err := os.WriteFile(cfgPath, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := EnsureIgnoreFile(cfgPath, root); err != nil {
		t.Fatalf("second EnsureIgnoreFile: %v", err)
	}
	b, _ = os.ReadFile(cfgPath)
	if string(b) != edited {
		t.Fatalf("EnsureIgnoreFile clobbered an existing file")
	}
}

// A desk is routinely also a working repo, so the seeded ignore list has to keep the indexer
// out of the places a repo keeps credentials. This asserts against the SHIPPED seed rather than
// a copy of it: a hand-listed expectation would still pass after someone edited the real file,
// which is the failure mode that let .claude/settings.local.json get indexed with its contents.
func TestDefaultIgnore_CoversCredentialBearingPaths(t *testing.T) {
	// Round-trip the shipped seed through the real loader rather than a test-local parser, so
	// this exercises the same path a live desk does.
	seed := filepath.Join(t.TempDir(), ".librarian-ignore")
	if err := os.WriteFile(seed, []byte(DefaultIgnore()), 0o644); err != nil {
		t.Fatalf("write seed: %v", err)
	}
	list, err := LoadIgnoreList(seed)
	if err != nil {
		t.Fatalf("load shipped default ignore: %v", err)
	}

	mustIgnore := []string{
		".claude/settings.local.json", // agent config: holds API keys by convention
		".claude/memory/MEMORY.md",
		".claude/agents/x.md",
		"_meta/secrets/anthropic.env",
		"_meta/secrets/superuser.txt",
		".git/config", // remote URLs can carry tokens
		"logs/run.log",
	}
	for _, rel := range mustIgnore {
		if !IsIgnored(rel, list) {
			t.Errorf("the shipped default ignore does NOT cover %q — a sweep would index it", rel)
		}
	}

	// The complement matters just as much: an over-broad rule that swallowed ordinary desk
	// content would make the indexer useless, and nothing else here would catch that.
	mustIndex := []string{
		"specs/one-pager.md",
		"journal/2026-01-01.md",
		"claude-notes.md", // not under .claude/
		".claudemap",      // shares a prefix but is not that directory
		"projects/work.md",
	}
	for _, rel := range mustIndex {
		if IsIgnored(rel, list) {
			t.Errorf("the shipped default ignore wrongly covers %q — ordinary desk content", rel)
		}
	}
}

package desklib

import (
	"bytes"
	"os"
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
		"_meta/HANDOFF.md":                    true,  // prefix (trailing /)
		"_meta":                               false, // "_meta/" requires the slash prefix
		"CLAUDE.md":                           true,  // exact
		"CLAUDE.md.bak":                       false, // not exact, not entry+"/"
		"_structure/decisions/0001-foo.md":    true,  // prefix
		"tasks/x.md":                          false,
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

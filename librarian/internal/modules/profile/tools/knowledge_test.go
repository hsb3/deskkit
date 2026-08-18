package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// write creates dir/name (with parents) holding body.
func write(t *testing.T, dir, name, body string) {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatalf("mkdir for %s: %v", p, err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", p, err)
	}
}

// TestKnowledgeIndex_WalkSortExcludeAndWords pins the walk contract: recursive, *.md only, the
// profile files themselves excluded (including profile.md), entries sorted path-ascending,
// exact word counts, and included content equal to the raw file text.
func TestKnowledgeIndex_WalkSortExcludeAndWords(t *testing.T) {
	root := t.TempDir()
	write(t, root, "profile.yaml", "schema_version: 1\n")
	write(t, root, "profile.example.yaml", "schema_version: 1\n")
	write(t, root, "profile.md", "---\nschema_version: 1\n---\n")
	write(t, root, "notes.md", "  one two   three \n")
	write(t, root, "empty.md", "   \n\t\n")
	write(t, root, "nested/deep/inner.md", "alpha beta")
	write(t, root, "ignored.txt", "not markdown")
	write(t, root, ".dotted.md", "dot file counts")

	idx := KnowledgeIndex(root, DefaultKnowledgeBudget)

	var got []string
	for _, e := range idx.Entries {
		got = append(got, e.Path)
	}
	want := []string{".dotted.md", "empty.md", "nested/deep/inner.md", "notes.md"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("entries = %v, want %v (sorted, profile.* excluded, .txt excluded)", got, want)
	}
	if idx.FileCount != len(want) {
		t.Errorf("FileCount = %d, want %d", idx.FileCount, len(want))
	}
	if idx.Root != root {
		t.Errorf("Root = %q, want %q", idx.Root, root)
	}
	if idx.Budget != DefaultKnowledgeBudget {
		t.Errorf("Budget = %d, want %d", idx.Budget, DefaultKnowledgeBudget)
	}

	byPath := map[string]KnowledgeEntry{}
	for _, e := range idx.Entries {
		byPath[e.Path] = e
	}
	if e := byPath["notes.md"]; e.Words != 3 || e.Bytes != len("  one two   three \n") ||
		!e.ContentIncluded || e.Content != "  one two   three \n" {
		t.Errorf("notes.md entry = %+v, want 3 words, raw bytes, content included verbatim", e)
	}
	if e := byPath["empty.md"]; e.Words != 0 {
		t.Errorf("whitespace-only file should count 0 words, got %d", e.Words)
	}
	if e := byPath["nested/deep/inner.md"]; e.Words != 2 || e.Content != "alpha beta" {
		t.Errorf("nested entry = %+v, want 2 words and its content", e)
	}
}

// TestKnowledgeIndex_BudgetKeepsScanning pins the budget algorithm: an over-budget file is
// emitted metadata-only and the scan CONTINUES against the unchanged running total, so a later
// file that still fits is included.
func TestKnowledgeIndex_BudgetKeepsScanning(t *testing.T) {
	root := t.TempDir()
	fifty := strings.Repeat("x", 50)
	write(t, root, "a.md", fifty)
	write(t, root, "b.md", fifty)
	write(t, root, "c.md", fifty)

	idx := KnowledgeIndex(root, 60)

	if idx.FileCount != 3 || len(idx.Entries) != 3 {
		t.Fatalf("FileCount = %d / %d entries, want 3 (over-budget files stay in the index)",
			idx.FileCount, len(idx.Entries))
	}
	if idx.Entries[0].Path != "a.md" || idx.Entries[1].Path != "b.md" || idx.Entries[2].Path != "c.md" {
		t.Fatalf("entries out of sorted order: %+v", idx.Entries)
	}
	if !idx.Entries[0].ContentIncluded || idx.Entries[0].Content != fifty {
		t.Errorf("a.md should be included with content, got %+v", idx.Entries[0])
	}
	for _, e := range idx.Entries[1:] {
		if e.ContentIncluded || e.Content != "" {
			t.Errorf("%s should be metadata-only, got %+v", e.Path, e)
		}
		if e.Bytes != 50 || e.Words != 1 {
			t.Errorf("%s metadata should still be exact, got %+v", e.Path, e)
		}
	}
	if idx.BytesIncluded != 50 {
		t.Errorf("BytesIncluded = %d, want 50 (only the included file counts)", idx.BytesIncluded)
	}
}

// TestKnowledgeIndex_MissingRoot: a nonexistent root is an empty index, never an error.
func TestKnowledgeIndex_MissingRoot(t *testing.T) {
	idx := KnowledgeIndex(filepath.Join(t.TempDir(), "no", "such", "dir"), DefaultKnowledgeBudget)
	if idx.FileCount != 0 || len(idx.Entries) != 0 || idx.BytesIncluded != 0 {
		t.Fatalf("missing root should yield an empty index, got %+v", idx)
	}
}

// TestKnowledgeIndex_ContentOmittedFromJSON proves the wire shape: an excluded entry carries no
// `content` key at all (the field is omitempty), so a consumer cannot mistake "" for a body.
func TestKnowledgeIndex_ContentOmittedFromJSON(t *testing.T) {
	root := t.TempDir()
	write(t, root, "a.md", strings.Repeat("x", 50))
	idx := KnowledgeIndex(root, 10)
	b := mustJSON(t, idx.Entries[0])
	if strings.Contains(b, `"content"`) {
		t.Errorf("an excluded entry must omit `content` entirely, got %s", b)
	}
	if !strings.Contains(b, `"contentIncluded":false`) {
		t.Errorf("entry JSON should carry contentIncluded:false, got %s", b)
	}
}

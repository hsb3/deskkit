package tools

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// DefaultKnowledgeBudget bounds what the index auto-loads so a large personalization folder
// cannot blow a model's context window: 64 KiB of file CONTENT (metadata is always returned).
const DefaultKnowledgeBudget = 65536

// profileExcludeRe matches the files that ARE the profile rather than background prose. It
// deliberately covers the .md extension too, so a markdown-frontmatter profile is excluded from
// the prose index even though the index is otherwise markdown-only.
var profileExcludeRe = regexp.MustCompile(`(?i)^profile(\.example)?\.(ya?ml|json|md)$`)

// KnowledgeEntry is one indexed background file. Content is present ONLY when it fit the
// budget: `omitempty` keeps an excluded entry's payload absent from the wire shape entirely, so
// a consumer can never mistake an empty string for an empty document.
type KnowledgeEntry struct {
	// Path is slash-separated and relative to the index root.
	Path            string `json:"path"`
	Bytes           int    `json:"bytes"`
	Words           int    `json:"words"`
	ContentIncluded bool   `json:"contentIncluded"`
	Content         string `json:"content,omitempty"`
}

// KnowledgeIndexResult is the whole index. FileCount counts EVERY entry (budget-excluded ones
// included); BytesIncluded sums only the files whose content was actually returned.
type KnowledgeIndexResult struct {
	Root          string           `json:"root"`
	Budget        int              `json:"budget"`
	FileCount     int              `json:"fileCount"`
	BytesIncluded int              `json:"bytesIncluded"`
	Entries       []KnowledgeEntry `json:"entries"`
}

// KnowledgeIndex indexes root (a personalization root) into a deterministic, budget-bounded
// listing of its markdown background files.
//
// Determinism: files are visited in ascending path order, so the same tree always yields the
// same index. Budget: a running included-byte total grows only when a file's content is
// returned, and a file that does not fit is emitted metadata-only WITHOUT stopping the scan —
// so a later, smaller file is still measured against the unchanged total and may be included.
// A missing or unreadable root (or subdirectory) yields fewer entries, never an error: the
// index is a best-effort briefing surface, not a gate.
func KnowledgeIndex(root string, budget int) KnowledgeIndexResult {
	files := collectMarkdown(root)
	sort.Strings(files)

	entries := make([]KnowledgeEntry, 0, len(files))
	bytesIncluded := 0
	for _, abs := range files {
		b, err := os.ReadFile(abs)
		if err != nil {
			continue // unreadable file — skip entirely rather than fail the whole index
		}
		content := string(b)
		rel, relErr := filepath.Rel(root, abs)
		if relErr != nil {
			continue
		}
		e := KnowledgeEntry{
			Path:  filepath.ToSlash(rel),
			Bytes: len(b),
			Words: countWords(content),
		}
		if bytesIncluded+e.Bytes <= budget {
			bytesIncluded += e.Bytes
			e.ContentIncluded = true
			e.Content = content
		}
		entries = append(entries, e)
	}
	return KnowledgeIndexResult{
		Root:          root,
		Budget:        budget,
		FileCount:     len(entries),
		BytesIncluded: bytesIncluded,
		Entries:       entries,
	}
}

// collectMarkdown walks root recursively and returns the absolute paths of every *.md file that
// is not a profile file. A directory that cannot be read is skipped, not fatal.
func collectMarkdown(root string) []string {
	var out []string
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		name := d.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".md") || profileExcludeRe.MatchString(name) {
			return nil
		}
		out = append(out, path)
		return nil
	})
	return out
}

// countWords counts whitespace-delimited tokens in trimmed text (a whitespace-only document is
// zero words, not one).
func countWords(text string) int { return len(strings.Fields(text)) }

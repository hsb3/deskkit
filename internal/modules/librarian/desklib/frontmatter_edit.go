package desklib

import (
	"fmt"
	"strings"
)

// SetFrontmatterField returns content with the top-level frontmatter key set to value,
// touching ONLY that key's line: every other byte of the document (body, other keys,
// comments, spacing) is preserved exactly. This is the server-side primitive behind
// field-level editing — the browser never rewrites YAML itself.
//
// Semantics mirror ParseFrontmatter's tolerance: the first non-empty line must open a
// `---` fence and a closing fence must exist, else the document has no frontmatter and
// the edit is refused. An existing scalar key is replaced in place (its indentation and
// any trailing \r kept); a missing key is inserted immediately before the closing fence.
// A key currently holding a block array is refused rather than silently flattened.
func SetFrontmatterField(content []byte, key, value string) ([]byte, error) {
	if strings.TrimSpace(key) == "" {
		return nil, fmt.Errorf("set frontmatter: empty key")
	}
	if strings.ContainsAny(value, "\n\r") {
		return nil, fmt.Errorf("set frontmatter: value for %q must be a single line", key)
	}
	lines := strings.Split(string(content), "\n")
	if len(lines) == 0 || strings.TrimSpace(strings.TrimSuffix(lines[0], "\r")) != "---" {
		return nil, fmt.Errorf("set frontmatter: document has no frontmatter fence")
	}
	// An unterminated fence means no valid frontmatter (ParseFrontmatter's rule) — check
	// before editing anything, so a key inside a broken fence is never touched.
	closed := false
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(strings.TrimSuffix(lines[i], "\r")) == "---" {
			closed = true
			break
		}
	}
	if !closed {
		return nil, fmt.Errorf("set frontmatter: frontmatter fence never closes")
	}
	for i := 1; i < len(lines); i++ {
		line := lines[i]
		cr := ""
		if strings.HasSuffix(line, "\r") {
			cr = "\r"
			line = strings.TrimSuffix(line, "\r")
		}
		if strings.TrimSpace(line) == "---" {
			// Closing fence with the key never seen: insert before it.
			inserted := key + ": " + value + cr
			lines = append(lines[:i], append([]string{inserted}, lines[i:]...)...)
			return []byte(strings.Join(lines, "\n")), nil
		}
		k, _, found := strings.Cut(line, ":")
		if !found || strings.TrimSpace(k) != key || strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		// A block-array key (empty value, `- item` lines follow) is not a scalar edit.
		if i+1 < len(lines) && strings.TrimSpace(strings.TrimSuffix(lines[i+1], "\r")) != "" &&
			strings.HasPrefix(strings.TrimSpace(strings.TrimSuffix(lines[i+1], "\r")), "- ") {
			return nil, fmt.Errorf("set frontmatter: key %q holds a block array, not a scalar", key)
		}
		indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
		lines[i] = indent + key + ": " + value + cr
		return []byte(strings.Join(lines, "\n")), nil
	}
	return nil, fmt.Errorf("set frontmatter: frontmatter fence never closes")
}

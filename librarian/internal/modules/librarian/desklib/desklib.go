// Package desklib holds the pure, DB-free helpers ported from the PoC desklib.py:
// checksum, tolerant frontmatter parse, git metadata, byte-exact write, and the
// write-protection ignore boundary (is_ignored + fail-closed load + first-run
// auto-create). These are the primitives the seven tools (later slices) call; keeping
// them here (no PocketBase import) makes them unit-testable in isolation.
package desklib

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// defaultIgnore is the embedded seed for a desk's .librarian-ignore (spec §10.1). It is
// identity-neutral and protects the binding docs by default. `_knowledge/` is included
// per the M-05 / build-brief punch-list #1 reconciliation (write-excluded / flag-only,
// exactly as `_meta/` is) so the librarian never proposes a fix against a profile or
// freeform-background file. Inline comments are deliberately absent: LoadIgnoreList only
// strips whole-line `#` comments, so a trailing comment would corrupt an entry.
//
//go:embed default-librarian-ignore
var defaultIgnore string

// Checksum returns the lowercase sha256 hex of raw bytes (spec §5.1 checksum field).
func Checksum(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// WriteExact writes exactly content to abs, creating parent dirs, with NO newline
// translation (spec §5.4). This byte-exactness is what makes restore cmp-clean.
func WriteExact(abs string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return err
	}
	return os.WriteFile(abs, content, 0o644)
}

// ParseFrontmatter is the hand-rolled, tolerant markdown-frontmatter parser (spec §5.1;
// ported verbatim from desklib.py parse_frontmatter). It is NOT a strict YAML library:
//   - requires the first non-empty line to be `---`;
//   - splits each key line on the FIRST colon only, so an unquoted `synopsis: a: b` is
//     tolerated (the desk's YAML-colon gotcha) where strict YAML would fail;
//   - supports `[a, b]` inline arrays and empty-value-opens-a-block-array (`- item`);
//   - strips surrounding quotes from scalar values;
//   - an UNTERMINATED / malformed fence returns an empty map (treated as no frontmatter),
//     never a crash.
//
// Returned values are string or []string.
func ParseFrontmatter(text string) map[string]any {
	lines := strings.Split(text, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return map[string]any{}
	}
	fm := map[string]any{}
	currentKey := ""
	for i := 1; i < len(lines); i++ {
		line := lines[i]
		if strings.TrimSpace(line) == "---" {
			return fm // closed fence -> valid frontmatter
		}
		stripped := strings.TrimSpace(line)
		if currentKey != "" && strings.HasPrefix(stripped, "- ") {
			item := trimQuotes(strings.TrimSpace(stripped[2:]))
			arr, _ := fm[currentKey].([]string)
			fm[currentKey] = append(arr, item)
			continue
		}
		currentKey = ""
		if stripped != "" && !strings.HasPrefix(stripped, "#") && strings.Contains(line, ":") {
			key, value, _ := strings.Cut(line, ":") // FIRST colon only
			key = strings.TrimSpace(key)
			value = strings.TrimSpace(value)
			switch {
			case strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]"):
				inner := strings.TrimSpace(value[1 : len(value)-1])
				if inner == "" {
					fm[key] = []string{}
				} else {
					parts := strings.Split(inner, ",")
					arr := make([]string, 0, len(parts))
					for _, p := range parts {
						arr = append(arr, trimQuotes(strings.TrimSpace(p)))
					}
					fm[key] = arr
				}
			case value == "":
				fm[key] = []string{} // may become a block array
				currentKey = key
			default:
				fm[key] = trimQuotes(value)
			}
		}
	}
	return map[string]any{} // unterminated fence -> not valid frontmatter
}

// trimQuotes mirrors Python's value.strip("\"'"): strip any leading/trailing single or
// double quote characters from both ends.
func trimQuotes(s string) string { return strings.Trim(s, "\"'") }

// --- git metadata (spec §5.1). Returns "<hash>|<yyyy-mm-dd>" or "" on any failure. ---

// GitOrigin returns the first-add commit meta for rel (git --diff-filter=A). See the
// first-vs-latest-add note in the handoff: spec §5.1 specifies `-1`; the sweep-tool owner
// may add `--reverse` if strict first-add ordering is required.
func GitOrigin(root, rel string) string { return gitMeta(root, rel, true) }

// GitLastCommit returns the last-commit meta for rel (spec §5.1).
func GitLastCommit(root, rel string) string { return gitMeta(root, rel, false) }

// GitNewestCommit returns the newest commit date (yyyy-mm-dd) across the whole tree, for
// R6 staleness (spec §5.2). "" when git yields nothing.
func GitNewestCommit(root string) string {
	out, err := exec.Command("git", "-C", root, "log", "-1", "--format=%cs").Output()
	if err != nil {
		return ""
	}
	return firstLine(strings.TrimSpace(string(out)))
}

func gitMeta(root, rel string, firstAdd bool) string {
	args := []string{"-C", root, "log", "--format=%H|%cs", "-1"}
	if firstAdd {
		args = append(args, "--diff-filter=A")
	}
	args = append(args, "--", rel)
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return ""
	}
	return firstLine(strings.TrimSpace(string(out)))
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

// --- ignore boundary (spec §5.3 is_ignored + §10.1 fail-closed / auto-create). ---

// LoadIgnoreList reads an ignore file: one entry per line, blank lines and whole-line
// `#` comments skipped, entries DESK_ROOT-relative. Returns an error when the file is
// absent or unreadable — callers MUST fail closed on that error (see Ignored).
func LoadIgnoreList(path string) ([]string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var entries []string
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			entries = append(entries, line)
		}
	}
	return entries, nil
}

// IsIgnored reports whether rel matches any entry (spec §5.3, ported exactly):
//   - entry ending in "/"        -> prefix match: strings.HasPrefix(rel, entry)
//   - entry without trailing "/" -> exact (rel == entry) OR prefix rel starts entry+"/"
func IsIgnored(rel string, ignoreList []string) bool {
	for _, entry := range ignoreList {
		if strings.HasSuffix(entry, "/") {
			if strings.HasPrefix(rel, entry) {
				return true
			}
		} else if rel == entry || strings.HasPrefix(rel, entry+"/") {
			return true
		}
	}
	return false
}

// Ignored is the write-protection boundary decision for rel, loading the ignore file at
// cfgPath. It FAILS CLOSED (spec §10.1): if the ignore file is absent or unreadable,
// every path is treated as ignored (returns true, err) so no write can proceed past a
// broken boundary. It never falls through to an empty list.
func Ignored(rel, cfgPath string) (bool, error) {
	list, err := LoadIgnoreList(cfgPath)
	if err != nil {
		return true, err // fail closed
	}
	return IsIgnored(rel, list), nil
}

// EnsureIgnoreFile auto-creates the ignore boundary from the embedded defaults on first
// run if it is absent (spec §10.1), so the boundary exists before any tool can write.
// The embedded copy is the seed for the on-disk file, never a silent runtime substitute
// (that path is Ignored's fail-closed behavior). An existing file is left untouched.
func EnsureIgnoreFile(cfgPath, deskRoot string) error {
	if cfgPath == "" {
		cfgPath = filepath.Join(deskRoot, ".librarian-ignore")
	}
	if _, err := os.Stat(cfgPath); err == nil {
		return nil // already present
	} else if !os.IsNotExist(err) {
		return err // a real stat error (permissions) — surface it
	}
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(cfgPath, []byte(defaultIgnore), 0o644)
}

// DefaultIgnore returns the embedded default ignore contents (for tests / tooling).
func DefaultIgnore() string { return defaultIgnore }

package config

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// ProfileRootDir is the single profile-root directory name. scripts/check-profile-root.mjs fails
// if it diverges from schema/paths.yaml; every path join to the personalization root flows from it.
const ProfileRootDir = "_knowledge"

// DiscoverProfile walks up from startDir looking for a single personalization root
// `_knowledge/profile.{yaml,yml,json,md}` (M-05 surface (i); the same walk-up a .env or
// .git gets). It returns the first match and true, or ("", false). Extension precedence
// on a tie within one directory: yaml, yml, json, md.
func DiscoverProfile(startDir string) (string, bool) {
	names := []string{"profile.yaml", "profile.yml", "profile.json", "profile.md"}
	dir := startDir
	for {
		for _, name := range names {
			p := filepath.Join(dir, ProfileRootDir, name)
			if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
				return p, true
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

// LoadProfile parses a profile file into a nested map, selecting the codec by extension
// (M-05 "Format & location"): .yaml/.yml -> YAML, .json -> JSON, .md -> the YAML
// frontmatter block. Plain scalars and maps only.
func LoadProfile(path string) (map[string]any, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".yaml", ".yml":
		return unmarshalYAML(b)
	case ".json":
		var m map[string]any
		if err := json.Unmarshal(b, &m); err != nil {
			return nil, fmt.Errorf("profile %s: %w", path, err)
		}
		return m, nil
	case ".md":
		return unmarshalYAML(extractFrontmatter(b))
	default:
		return nil, fmt.Errorf("profile %s: unsupported extension", path)
	}
}

func unmarshalYAML(b []byte) (map[string]any, error) {
	var m map[string]any
	if err := yaml.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("profile yaml: %w", err)
	}
	if m == nil {
		m = map[string]any{}
	}
	return m, nil
}

// extractFrontmatter returns the bytes between the first pair of `---` fences in a
// markdown-with-frontmatter file; empty if there is no fence.
func extractFrontmatter(b []byte) []byte {
	lines := strings.Split(string(b), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return nil
	}
	var body []string
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			return []byte(strings.Join(body, "\n"))
		}
		body = append(body, lines[i])
	}
	return nil // unterminated fence
}

// profileScalar resolves a dotted path into the profile tree and returns its scalar value
// as a string, or "" if absent, empty, or not a scalar (map/list).
func profileScalar(profile map[string]any, dotted string) string {
	if profile == nil {
		return ""
	}
	return scalarString(indexTree(profile, strings.Split(dotted, ".")))
}

func indexTree(m map[string]any, parts []string) any {
	var cur any = m
	for _, p := range parts {
		mm, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur = mm[p]
	}
	return cur
}

func scalarString(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case bool:
		return strconv.FormatBool(t)
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t, 10)
	case float64:
		if t == math.Trunc(t) && !math.IsInf(t, 0) {
			return strconv.FormatInt(int64(t), 10)
		}
		return strconv.FormatFloat(t, 'g', -1, 64)
	default:
		return "" // maps and lists are not substitutable scalars
	}
}

// ProfileScalar is the exported wrapper over profileScalar: it resolves a dotted path into the
// profile tree and returns its scalar value as a string, or "" if absent, empty, or not a
// scalar. Exported for the profile-module tools, which resolve the same dotted grammar the
// {{profile.…}} substitution does and must not re-derive it.
func ProfileScalar(profile map[string]any, dotted string) string {
	return profileScalar(profile, dotted)
}

// IndexTree is the exported wrapper over indexTree: it walks parts into the profile tree and
// returns whatever sits there (a scalar, a nested map, or nil). Exported so a caller that must
// report the keys AVAILABLE under the deepest resolvable parent — not just the missing leaf —
// can inspect the intermediate node.
func IndexTree(m map[string]any, parts []string) any { return indexTree(m, parts) }

// placeholderRe matches {{profile.<dotted.path>}} / {{env.<VAR>}} with an optional
// `|| "default"` (M-05 substitution convention). Group 1 = kind, 2 = path/var,
// 3 = the whole `|| "…"` clause (presence = has default), 4 = the default text.
var placeholderRe = regexp.MustCompile(`\{\{\s*(profile|env)\.([A-Za-z0-9_.]+)\s*(\|\|\s*"((?:[^"\\]|\\.)*)")?\s*\}\}`)

// Substitute resolves every {{profile.…}} / {{env.…}} placeholder in text against the
// profile (and process env). Missing-key rule (M-05, fail LOUD, not silent): a placeholder
// with NO `|| default` whose value is absent/empty is a hard error — Substitute returns
// the partially-resolved text plus a non-nil error naming every offending placeholder,
// never a silent empty substitution. Optional placeholders must carry a `|| "…"` default.
func Substitute(text string, profile map[string]any) (string, error) {
	var missing []string
	out := placeholderRe.ReplaceAllStringFunc(text, func(match string) string {
		sub := placeholderRe.FindStringSubmatch(match)
		kind, path := sub[1], sub[2]
		hasDefault := sub[3] != ""
		def := unescape(sub[4])

		var val string
		if kind == "env" {
			val = os.Getenv(path)
		} else {
			val = profileScalar(profile, path)
		}
		if val == "" {
			if hasDefault {
				return def
			}
			missing = append(missing, match)
			return match // left in place; the returned error is authoritative
		}
		return val
	})
	if len(missing) > 0 {
		return out, fmt.Errorf(
			"profile substitution: missing required key(s) with no default: %s",
			strings.Join(missing, ", "))
	}
	return out, nil
}

func unescape(s string) string {
	s = strings.ReplaceAll(s, `\"`, `"`)
	s = strings.ReplaceAll(s, `\\`, `\`)
	return s
}

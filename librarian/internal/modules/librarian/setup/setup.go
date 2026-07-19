// Package setup implements the zero-export desk onramp: `deskkit init`
// scaffolds the minimal _knowledge/profile.yaml a folder needs to work as a desk, and
// the first-run prompt decision that offers to run it on an interactive terminal.
//
// Identity-neutral: the profile written at runtime derives its desk name purely from the
// target directory's basename (never a person, org, or repo). The templates below carry no
// deployment identity — schema-generic comments only.
package setup

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/example/pocket-librarian/internal/core/config"
	"github.com/example/pocket-librarian/internal/core/store"
)

// profileTemplate is the least a folder needs to work as a desk with zero exports (a
// schema-compatible profile.yaml: schema/profile.schema.yaml, additionalProperties:false,
// schema_version required). %s is the YAML-quoted, basename-derived desk name.
const profileTemplate = `# Minimal desk profile scaffolded by deskkit init.
# The least a folder needs to work as a desk with zero exports.
# Full personalization options (identity, repos, board, models, preferences, ...)
# are documented in the profile schema: schema/profile.schema.yaml
schema_version: 1
desk:
  name: %s
  root: "."
`

// envStub is the optional .env written under --with-env. LLM_API_KEY_ENV names (never holds)
// the env var carrying the LLM API key; the canonical per-provider names are the same ones
// the provider adapter reads.
const envStub = `# LLM API-key indirection: LLM_API_KEY_ENV names the env var that holds your key.
LLM_API_KEY_ENV=ANTHROPIC_API_KEY
# Set the key it names (uncomment and paste your key):
# ANTHROPIC_API_KEY=
# Other providers: OPENAI_API_KEY / GEMINI_API_KEY (match LLM_API_KEY_ENV to your provider).
`

// FirstRunPrompt is the exact first-run onramp question shown on an interactive terminal when
// config is unresolvable. Empty input defaults to yes.
const FirstRunPrompt = "Set up this folder as a desk? [Y/n] "

// profileNames are the discovery-order profile basenames under _knowledge/ (mirrors
// config.DiscoverProfile's precedence).
var profileNames = []string{"profile.yaml", "profile.yml", "profile.json", "profile.md"}

// InitOptions controls the init scaffold.
type InitOptions struct {
	Force   bool // overwrite an existing profile/.env and allow a nested desk
	WithEnv bool // additionally write a .env stub
}

// InitResult reports what init wrote (or would have). StorePath is where the desk's data will
// land; init never creates it.
type InitResult struct {
	Dir             string // absolute target directory
	DeskName        string // basename-derived desk name written into the profile
	ProfilePath     string // the profile.yaml written
	EnvPath         string // the .env stub written ("" unless --with-env)
	StorePath       string // resolved store dir (NOT created by init)
	AncestorProfile string // an ancestor desk's profile, when a nested desk was created
}

// WriteSummary prints a human-readable summary of what init wrote to w.
func (r *InitResult) WriteSummary(w io.Writer) {
	fmt.Fprintf(w, "Wrote %s\n", r.ProfilePath)
	if r.EnvPath != "" {
		fmt.Fprintf(w, "Wrote %s\n", r.EnvPath)
	}
	fmt.Fprintf(w, "Desk name: %s\n", r.DeskName)
	if r.StorePath != "" {
		fmt.Fprintf(w, "Store path (created on first use): %s\n", r.StorePath)
	}
}

// InitProfile scaffolds <dir>/_knowledge/profile.yaml (and, with --with-env, <dir>/.env). It
// is idempotent-refusing: an existing profile (or, with --with-env, an existing .env) is a
// hard error unless opts.Force. When <dir> already sits inside another desk (an ancestor
// profile is discoverable), creating a nested desk is refused unless opts.Force or
// confirmNested returns true — so a nested desk is always deliberate. confirmNested may be nil
// (treated as a decline). init writes nothing outside <dir>/_knowledge and <dir>/.env.
func InitProfile(dir string, opts InitOptions, confirmNested func(deskName, ancestorPath string) (bool, error)) (*InitResult, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}

	// --- preflight (no writes until every check passes) ---

	// This dir already owns a profile.
	if existing, ok := existingProfileIn(abs); ok && !opts.Force {
		return nil, fmt.Errorf("%s already exists; pass --force to overwrite", existing)
	}

	// Nested-desk guard: an ancestor profile means this folder is already inside a desk.
	var ancestor string
	if ap, ok := ancestorProfileOf(abs); ok {
		ancestor = ap
		if !opts.Force {
			name := ancestorDeskName(ap)
			allowed := false
			if confirmNested != nil {
				allowed, err = confirmNested(name, ap)
				if err != nil {
					return nil, err
				}
			}
			if !allowed {
				return nil, fmt.Errorf(
					"this folder is already inside desk %q (%s); pass --force to create a nested desk", name, ap)
			}
		}
	}

	// --with-env clobber check (before any write, so init never half-applies).
	envPath := filepath.Join(abs, ".env")
	if opts.WithEnv && !opts.Force {
		if _, statErr := os.Stat(envPath); statErr == nil {
			return nil, fmt.Errorf("%s already exists; pass --force to overwrite", envPath)
		}
	}

	// --- writes ---

	deskName := filepath.Base(abs)
	knowledgeDir := filepath.Join(abs, "_knowledge")
	if err := os.MkdirAll(knowledgeDir, 0o755); err != nil {
		return nil, err
	}
	profilePath := filepath.Join(knowledgeDir, "profile.yaml")
	if err := os.WriteFile(profilePath, []byte(fmt.Sprintf(profileTemplate, yamlQuote(deskName))), 0o644); err != nil {
		return nil, err
	}

	res := &InitResult{
		Dir:             abs,
		DeskName:        deskName,
		ProfilePath:     profilePath,
		AncestorProfile: ancestor,
	}
	// Where this desk's data will land (StoreDir only computes the path; it never creates it).
	if storePath, serr := store.StoreDir(deskName); serr == nil {
		res.StorePath = storePath
	}

	if opts.WithEnv {
		// 0o600: this stub holds a real LLM API key once filled in, so it is owner-only from birth.
		if err := os.WriteFile(envPath, []byte(envStub), 0o600); err != nil {
			return nil, err
		}
		res.EnvPath = envPath
	}
	return res, nil
}

// FirstRunDecision is the pure decision for the first-run onramp: prompt on w and read a Y/n
// answer from r only when interactive (isTTY) and input is allowed (!noInput); otherwise
// decline WITHOUT emitting anything (byte-identical to the pre-existing fail-closed path).
// Empty input defaults to yes; y/yes (any case) accept; anything else declines. A read
// error / EOF declines (fail closed).
func FirstRunDecision(r io.Reader, w io.Writer, isTTY, noInput bool) (bool, error) {
	if noInput || !isTTY {
		return false, nil
	}
	if _, err := fmt.Fprint(w, FirstRunPrompt); err != nil {
		return false, err
	}
	sc := bufio.NewScanner(r)
	if !sc.Scan() {
		return false, sc.Err() // EOF / read error -> fail closed
	}
	switch strings.ToLower(strings.TrimSpace(sc.Text())) {
	case "", "y", "yes":
		return true, nil
	default:
		return false, nil
	}
}

// ConfirmNested prompts (default NO) whether to create a desk nested inside an existing one.
// Non-interactive (isTTY false) declines without prompting.
func ConfirmNested(r io.Reader, w io.Writer, isTTY bool, deskName, ancestorPath string) (bool, error) {
	if !isTTY {
		return false, nil
	}
	fmt.Fprintf(w, "This folder is already inside desk %q (%s).\nCreate a nested desk here anyway? [y/N] ", deskName, ancestorPath)
	sc := bufio.NewScanner(r)
	if !sc.Scan() {
		return false, sc.Err()
	}
	switch strings.ToLower(strings.TrimSpace(sc.Text())) {
	case "y", "yes":
		return true, nil
	default:
		return false, nil
	}
}

// existingProfileIn reports the profile file THIS directory already owns, if any.
func existingProfileIn(abs string) (string, bool) {
	for _, name := range profileNames {
		p := filepath.Join(abs, "_knowledge", name)
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			return p, true
		}
	}
	return "", false
}

// ancestorProfileOf discovers a profile in a STRICT ancestor of abs (the walk-up starts at
// abs's parent, so abs's own _knowledge is never mistaken for an ancestor).
func ancestorProfileOf(abs string) (string, bool) {
	parent := filepath.Dir(abs)
	if parent == abs {
		return "", false
	}
	return config.DiscoverProfile(parent)
}

// ancestorDeskName reads desk.name from an ancestor profile for the nested-desk notice,
// falling back to the basename of the directory that owns the profile.
func ancestorDeskName(profilePath string) string {
	if m, err := config.LoadProfile(profilePath); err == nil {
		if desk, ok := m["desk"].(map[string]any); ok {
			if n, ok := desk["name"].(string); ok && n != "" {
				return n
			}
		}
	}
	return filepath.Base(filepath.Dir(filepath.Dir(profilePath)))
}

// yamlQuote returns s as a YAML double-quoted scalar (escaping backslash, double-quote, and the
// control chars newline/carriage-return), so a basename with spaces, special characters, or even
// an embedded newline round-trips through the YAML parser unchanged. Backslash is escaped first so
// the escapes emitted below are not themselves re-escaped; the raw \n / \r are turned into their
// YAML escape sequences (a literal newline in a double-quoted scalar folds to a space — silently
// wrong — so it must never reach the emitted YAML).
func yamlQuote(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\r", `\r`)
	return `"` + s + `"`
}

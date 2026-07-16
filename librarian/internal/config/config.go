// Package config ports the PoC's pb_config.py env discovery into Go and adds the M-05
// data-surface loader (D7): a walk-up profile discovery + a {{profile.…}}/{{env.…}}
// substitution resolver that fails LOUD on a missing required key. Config is the single
// identity source: DESK_ROOT / DESK_NAME / path constants / model come from env, then a
// discovered _knowledge/profile.* fills anything still unset, then built-in neutral
// defaults — env ALWAYS wins (spec §3.4; M-05 "How the pocket-librarian consumes surface (i)").
//
// Identity-neutral: DESK_ROOT and DESK_NAME have NO built-in default. If neither env nor
// a profile supplies them, Load returns an error rather than inventing a personal path.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Config is the resolved runtime configuration. Secrets (API keys, superuser password)
// are NOT stored here — they are read from the environment at provider-construction /
// bootstrap time (spec §6.3, §10.3), so this struct carries nothing secret.
type Config struct {
	PBURL               string        // PB_URL
	PBSuperuserEmail    string        // PB_SUPERUSER_EMAIL (created on first run if set)
	PBSuperuserPassword string        // PB_SUPERUSER_PASSWORD (secret; env only)
	DeskRoot            string        // DESK_ROOT (required; no default)
	DeskName            string        // DESK_NAME (required; no default)
	DecisionsDir        string        // DECISIONS_DIR
	TasksDir            string        // TASKS_DIR
	AnalysesDir         string        // ANALYSES_DIR
	JournalDir          string        // JOURNAL_DIR
	SecretsDir          string        // SECRETS_DIR
	IgnoreConfig        string        // IGNORE_CONFIG (default <DeskRoot>/.librarian-ignore)
	HandoffPath         string        // HANDOFF_PATH
	AutonomousWrites    bool          // LIBRARIAN_AUTONOMOUS_WRITES (registration-time gate, §5.4)
	ClaimerPollInterval time.Duration // CLAIMER_POLL_INTERVAL (§2.4; Phase 2)
	LLMProvider         string        // LLM_PROVIDER
	LLMModel            string        // LLM_MODEL
	LLMMaxTokens        int           // LLM_MAX_TOKENS
	AgentMaxStep        int           // AGENT_MAX_STEP
}

// EntityDirMap returns the frontmatter-type -> configured-directory map the sweep/patrol
// tools (later slices) consume for dir_kind / R3 derivation (spec §5.1, §5.2).
func (c *Config) EntityDirMap() map[string]string {
	return map[string]string{
		"decision": c.DecisionsDir,
		"task":     c.TasksDir,
		"analysis": c.AnalysesDir,
		"journal":  c.JournalDir,
	}
}

// Load resolves configuration with strict env > profile > default precedence. It first
// loads a walk-up .env (never overriding an already-set process env var), then discovers
// a _knowledge/profile.* to fill unset fields, then applies neutral defaults. Returns an
// error if DESK_ROOT or DESK_NAME cannot be resolved from env or profile.
func Load() (*Config, error) {
	if wd, err := os.Getwd(); err == nil {
		_ = LoadDotEnv(wd)
	}

	// Discover + load a profile once; nil map when none is found.
	var profile map[string]any
	var profileRoot string
	if wd, err := os.Getwd(); err == nil {
		if path, ok := DiscoverProfile(wd); ok {
			if p, perr := LoadProfile(path); perr == nil {
				profile = p
				// DESK_ROOT default-from-profile = the repo root that owns _knowledge/,
				// i.e. the parent of the _knowledge dir (M-05 "surface (i)"). An explicit
				// profile.desk.root, when a non-"." value, overrides.
				profileRoot = filepath.Dir(filepath.Dir(path))
				if r := profileScalar(profile, "desk.root"); r != "" && r != "." {
					if filepath.IsAbs(r) {
						profileRoot = r
					} else {
						profileRoot = filepath.Join(profileRoot, r)
					}
				}
			}
		}
	}

	ps := func(dotted string) string { return profileScalar(profile, dotted) }

	c := &Config{
		PBURL:               pick("PB_URL", "", "http://127.0.0.1:8090"),
		PBSuperuserEmail:    pick("PB_SUPERUSER_EMAIL", "", ""),
		PBSuperuserPassword: pick("PB_SUPERUSER_PASSWORD", "", ""),
		DeskRoot:            pick("DESK_ROOT", profileRoot, ""),
		DeskName:            pick("DESK_NAME", ps("desk.name"), ""),
		DecisionsDir:        pick("DECISIONS_DIR", ps("desk.paths.decisions"), "_structure/decisions"),
		TasksDir:            pick("TASKS_DIR", ps("desk.paths.tasks"), "tasks"),
		AnalysesDir:         pick("ANALYSES_DIR", ps("desk.paths.analyses"), "analyses"),
		JournalDir:          pick("JOURNAL_DIR", ps("desk.paths.journal"), "journal"),
		SecretsDir:          pick("SECRETS_DIR", ps("desk.paths.secrets"), "_meta/secrets"),
		HandoffPath:         pick("HANDOFF_PATH", ps("desk.paths.handoff"), "_meta/HANDOFF.md"),
		LLMProvider:         pick("LLM_PROVIDER", ps("models.provider"), "anthropic"),
		LLMModel:            pick("LLM_MODEL", ps("models.model"), "claude-opus-4-8"),
	}
	c.AutonomousWrites = envBool("LIBRARIAN_AUTONOMOUS_WRITES", false)
	c.LLMMaxTokens = envInt("LLM_MAX_TOKENS", 4096)
	c.AgentMaxStep = envInt("AGENT_MAX_STEP", 12)
	c.ClaimerPollInterval = envDuration("CLAIMER_POLL_INTERVAL", 5*time.Second)

	// IGNORE_CONFIG default depends on the resolved DeskRoot; env still overrides.
	c.IgnoreConfig = pick("IGNORE_CONFIG", "", "")
	if c.IgnoreConfig == "" && c.DeskRoot != "" {
		c.IgnoreConfig = filepath.Join(c.DeskRoot, ".librarian-ignore")
	}

	var missing []string
	if c.DeskRoot == "" {
		missing = append(missing, "DESK_ROOT")
	}
	if c.DeskName == "" {
		missing = append(missing, "DESK_NAME")
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf(
			"config: required %s not set via env or _knowledge/profile.*; "+
				"the identity-neutral binary has no personal default",
			strings.Join(missing, ", "))
	}
	return c, nil
}

// pick applies env > profile > default precedence. An env var that is set but empty is
// treated as unset (so it can fall through to profile/default), matching the PoC's
// os.environ.get(key) or default idiom.
func pick(envKey, profileVal, def string) string {
	if v, ok := os.LookupEnv(envKey); ok && v != "" {
		return v
	}
	if profileVal != "" {
		return profileVal
	}
	return def
}

func envBool(key string, def bool) bool {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}

func envInt(key string, def int) int {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func envDuration(key string, def time.Duration) time.Duration {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}

// LoadDotEnv walks up from startDir to the filesystem root, and on the first `.env` it
// finds, applies its KEY=VALUE lines to the process environment WITHOUT overriding any
// already-set var (spec §3.4). Blank lines and `#` comments are skipped; surrounding
// quotes are stripped. Missing .env is not an error.
func LoadDotEnv(startDir string) error {
	dir := startDir
	for {
		p := filepath.Join(dir, ".env")
		if _, err := os.Stat(p); err == nil {
			return applyEnvFile(p)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return nil // reached root, no .env
		}
		dir = parent
	}
}

func applyEnvFile(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.Trim(strings.TrimSpace(val), `"'`)
		if _, exists := os.LookupEnv(key); exists {
			continue // never override an already-set process env var
		}
		_ = os.Setenv(key, val)
	}
	return nil
}

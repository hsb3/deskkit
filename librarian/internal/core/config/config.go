// Package config ports the PoC's pb_config.py env discovery into Go and adds the M-05
// data-surface loader (D7): a walk-up profile discovery + a {{profile.…}}/{{env.…}}
// substitution resolver that fails LOUD on a missing required key. Config is the single
// identity source: DESK_ROOT / DESK_NAME / path constants / model come from env, then a
// discovered _knowledge/profile.* fills anything still unset, then the machine-wide central
// config (central.go) for the few fields that have such a leg, then built-in neutral
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

// Resolution sources recorded in Config.Sources, in precedence order.
const (
	SourceEnv     = "env"     // a set, non-empty environment variable (a walk-up .env counts: it populates the env)
	SourceProfile = "profile" // the discovered per-desk _knowledge/profile.*
	SourceCentral = "central" // the machine-wide central config (see central.go)
	SourceDefault = "default" // the built-in neutral default
)

// Config is the resolved runtime configuration. The LLM API key is NEVER held here — only
// the NAME of the env var that may hold it (LLMAPIKeyEnv); the value is read at
// provider-construction time from that env var or the central config (spec §6.3).
// PBSuperuserPassword is the one secret this struct does carry: PocketBase bootstrap needs
// it in process at startup (spec §10.3), and it comes from the environment only.
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
	LLMAPIKeyEnv        string        // secrets_ref.llm_api_key — NAME of the env var holding the LLM API key (profile indirection; empty falls back to the per-provider default var)
	LLMMaxTokens        int           // LLM_MAX_TOKENS
	LLMContextWindow    int           // LLM_CONTEXT_WINDOW / profile models.context_window — token budget for ctx% (0 = unset; the TUI's per-model table default applies)
	AgentMaxStep        int           // AGENT_MAX_STEP
	PMEnabled           bool          // PM_ENABLED / profile modules.pm.enabled (spec §2.9; default ON since 1.0 — ADR 0008 amendment 2026-07-21)
	PMClaimTTL          time.Duration // PM_CLAIM_TTL — pm claim horizon (spec §3.6; default 30m)
	PMAutonomousWrites  bool          // PM_AUTONOMOUS_WRITES (spec §5.1/§13 item 9; default ON — the document gate is the real safety)
	PMStalledDays       int           // PM_STALLED_DAYS — get_context stalled threshold (spec §5.2; default 14)

	// Sources records where each resolved field's value came from: one of "env", "profile",
	// "central", "default" (the Source* constants). It is keyed by the env-var name the field
	// is documented under (e.g. "LLM_PROVIDER", "DESK_NAME") so a display surface needs no
	// second lookup table. Written by the resolver AS it decides — never re-derived, so it
	// cannot drift from the values above.
	Sources map[string]string
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

// Load resolves configuration with strict env > profile > central > default precedence. It
// first loads a walk-up .env (never overriding an already-set process env var), then discovers
// a _knowledge/profile.* to fill unset fields, then the machine-wide central config (three
// fields only: LLM_PROVIDER, LLM_MODEL, DESK_NAME via default_desk), then neutral defaults.
// Every resolved field's winning leg is recorded in Config.Sources. Returns an error if
// DESK_ROOT or DESK_NAME cannot be resolved from any of them.
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

	// Central (machine-wide) config: the leg between profile and default for the three fields
	// that have one. Loaded EXACTLY ONCE. An unreadable/malformed central file is not fatal —
	// it is treated as absent — but it is reported on stderr rather than silently swallowed,
	// because a config that is wrong AND quiet is the worst outcome here.
	central, cerr := LoadCentral()
	if cerr != nil {
		fmt.Fprintf(os.Stderr, "deskkit: ignoring central config: %v\n", cerr)
		central = &Central{}
	}

	r := &resolver{sources: map[string]string{}}

	c := &Config{
		Sources:             r.sources,
		PBURL:               r.pick("PB_URL", "", "", "http://127.0.0.1:8090"),
		PBSuperuserEmail:    r.pick("PB_SUPERUSER_EMAIL", "", "", ""),
		PBSuperuserPassword: r.pick("PB_SUPERUSER_PASSWORD", "", "", ""),
		DeskRoot:            r.pick("DESK_ROOT", profileRoot, "", ""),
		// default_desk is the central leg: it makes an otherwise unresolvable DESK_NAME
		// resolve, but only when the operator actually set one.
		DeskName:     r.pick("DESK_NAME", ps("desk.name"), central.DefaultDesk, ""),
		DecisionsDir: r.pick("DECISIONS_DIR", ps("desk.paths.decisions"), "", "_structure/decisions"),
		TasksDir:     r.pick("TASKS_DIR", ps("desk.paths.tasks"), "", "tasks"),
		AnalysesDir:  r.pick("ANALYSES_DIR", ps("desk.paths.analyses"), "", "analyses"),
		JournalDir:   r.pick("JOURNAL_DIR", ps("desk.paths.journal"), "", "journal"),
		SecretsDir:   r.pick("SECRETS_DIR", ps("desk.paths.secrets"), "", "_meta/secrets"),
		HandoffPath:  r.pick("HANDOFF_PATH", ps("desk.paths.handoff"), "", "_meta/HANDOFF.md"),
		LLMProvider:  r.pick("LLM_PROVIDER", ps("models.provider"), central.LLM.Provider, "anthropic"),
		LLMModel:     r.pick("LLM_MODEL", ps("models.model"), central.LLM.Model, "claude-opus-4-8"),
		// secrets_ref.llm_api_key names (never contains) the env var holding the API key.
		// Env still wins so an operator can override the indirection without a profile edit.
		LLMAPIKeyEnv: r.pick("LLM_API_KEY_ENV", ps("secrets_ref.llm_api_key"), "", ""),
	}
	c.AutonomousWrites = r.envBool("LIBRARIAN_AUTONOMOUS_WRITES", "", false)
	// PM feature gate (spec §2.9): env PM_ENABLED > profile modules.pm.enabled > default ON
	// (owner-ruled 2026-07-21, ADR 0008 amendment: PM ships default-on for 1.0). The profile
	// leg is THREE-STATE so it can still override the ON default: an explicit
	// modules.pm.enabled renders "true"/"false" via profileScalar and decides; ABSENT (empty)
	// falls through to the ON default. envBool keeps PM_ENABLED's exact prior semantics
	// (ParseBool on the env leg; unset or invalid falls through to profile-then-default), so
	// PM_ENABLED=false and modules.pm.enabled: false both still cleanly disable the module.
	c.PMEnabled = r.envBool("PM_ENABLED", ps("modules.pm.enabled"), true)
	c.PMClaimTTL = r.envDuration("PM_CLAIM_TTL", 30*time.Minute)
	// PM surface write gate (spec §5.1, §13 item 9): DEFAULT ON — PM tools write only the
	// store (never desk files), and the real safety is transition_item's document gates; a
	// desk that wants agents read-only over the graph sets PM_AUTONOMOUS_WRITES=false.
	c.PMAutonomousWrites = r.envBool("PM_AUTONOMOUS_WRITES", "", true)
	c.PMStalledDays = r.envInt("PM_STALLED_DAYS", "", 14)
	c.LLMMaxTokens = r.envInt("LLM_MAX_TOKENS", "", 4096)
	// Context-window budget for the TUI's ctx% gauge: env LLM_CONTEXT_WINDOW > profile
	// models.context_window > 0. 0 means "unset" — the TUI falls back to its per-model table
	// default.
	c.LLMContextWindow = r.envInt("LLM_CONTEXT_WINDOW", ps("models.context_window"), 0)
	c.AgentMaxStep = r.envInt("AGENT_MAX_STEP", "", 12)
	c.ClaimerPollInterval = r.envDuration("CLAIMER_POLL_INTERVAL", 5*time.Second)

	// IGNORE_CONFIG default depends on the resolved DeskRoot; env still overrides.
	c.IgnoreConfig = r.pick("IGNORE_CONFIG", "", "", "")
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

// resolver applies the precedence chain AND records, per env-var key, which leg won. Source
// recording happens inside the same branch that returns the value, so Config.Sources cannot
// disagree with Config's values — the alternative (a second pass re-deriving sources) drifts.
type resolver struct{ sources map[string]string }

func (r *resolver) mark(key, source string) { r.sources[key] = source }

// pick applies env > profile > central > default precedence. An env var that is set but empty
// is treated as unset (so it can fall through), matching the PoC's os.environ.get(key) or
// default idiom. Pass "" for a leg the field does not have.
func (r *resolver) pick(envKey, profileVal, centralVal, def string) string {
	if v, ok := os.LookupEnv(envKey); ok && v != "" {
		r.mark(envKey, SourceEnv)
		return v
	}
	if profileVal != "" {
		r.mark(envKey, SourceProfile)
		return profileVal
	}
	if centralVal != "" {
		r.mark(envKey, SourceCentral)
		return centralVal
	}
	r.mark(envKey, SourceDefault)
	return def
}

// envBool takes the profile leg as the raw profileScalar rendering: any non-empty value
// decides, and only the exact "true" means true (an explicit profile key must be able to
// override an ON default). An env var that is set but unparseable falls through to it.
func (r *resolver) envBool(key, profileVal string, def bool) bool {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			r.mark(key, SourceEnv)
			return b
		}
	}
	if profileVal != "" {
		r.mark(key, SourceProfile)
		return profileVal == "true"
	}
	r.mark(key, SourceDefault)
	return def
}

// envInt: env > profile > default. An unparseable value on either leg falls through.
func (r *resolver) envInt(key, profileVal string, def int) int {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			r.mark(key, SourceEnv)
			return n
		}
	}
	if profileVal != "" {
		if n, err := strconv.Atoi(profileVal); err == nil {
			r.mark(key, SourceProfile)
			return n
		}
	}
	r.mark(key, SourceDefault)
	return def
}

func (r *resolver) envDuration(key string, def time.Duration) time.Duration {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			r.mark(key, SourceEnv)
			return d
		}
	}
	r.mark(key, SourceDefault)
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

package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Every test here pins XDG_CONFIG_HOME to a temp dir: without it these would read (and
// SaveCentral would WRITE) the real user's central config, making the suite machine-dependent.

func TestCentralPathHonorsXDG(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	p, err := CentralPath()
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(dir, "deskkit", "config.yaml"); p != want {
		t.Fatalf("CentralPath() = %q, want %q", p, want)
	}
	// Resolving a path must never create anything.
	if _, err := os.Stat(filepath.Join(dir, "deskkit")); !os.IsNotExist(err) {
		t.Fatalf("CentralPath must not create the config dir (stat err = %v)", err)
	}
}

func TestLoadCentralMissingFileIsNotAnError(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	c, err := LoadCentral()
	if err != nil {
		t.Fatalf("LoadCentral on a missing file must not error, got %v", err)
	}
	if c == nil || *c != (Central{}) {
		t.Fatalf("LoadCentral on a missing file must return a zero-value Central, got %+v", c)
	}
}

func TestSaveCentralModes(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	c := &Central{}
	if err := c.Set("llm.api_key", "sk-scratch-abcd1234"); err != nil {
		t.Fatal(err)
	}
	if err := SaveCentral(c); err != nil {
		t.Fatalf("SaveCentral: %v", err)
	}

	p, err := CentralPath()
	if err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(p)
	if err != nil {
		t.Fatalf("central file not written: %v", err)
	}
	if got := fi.Mode().Perm(); got != 0o600 {
		t.Errorf("central file mode = %04o, want 0600", got)
	}
	di, err := os.Stat(filepath.Dir(p))
	if err != nil {
		t.Fatal(err)
	}
	if got := di.Mode().Perm(); got != 0o700 {
		t.Errorf("central dir mode = %04o, want 0700", got)
	}

	// Round-trips through the file.
	got, err := LoadCentral()
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := got.Get("llm.api_key"); v != "sk-scratch-abcd1234" {
		t.Errorf("llm.api_key round-trip = %q", v)
	}
}

func TestSaveCentralRewriteKeepsMode(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	if err := SaveCentral(&Central{}); err != nil {
		t.Fatal(err)
	}
	p, err := CentralPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(p, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := SaveCentral(&Central{DefaultDesk: "example-desk"}); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	if got := fi.Mode().Perm(); got != 0o600 {
		t.Errorf("rewritten central file mode = %04o, want 0600 (a widened mode must be tightened back)", got)
	}
}

func TestCentralSetGetRoundTrip(t *testing.T) {
	c := &Central{}
	vals := map[string]string{
		"llm.provider": "openai",
		"llm.model":    "example-model",
		"llm.api_key":  "sk-example-wxyz",
		"default_desk": "example-desk",
	}
	for _, k := range CentralKeys() {
		v, ok := vals[k]
		if !ok {
			t.Fatalf("CentralKeys() returned unexpected key %q", k)
		}
		if err := c.Set(k, v); err != nil {
			t.Fatalf("Set(%q): %v", k, err)
		}
	}
	for k, want := range vals {
		got, ok := c.Get(k)
		if !ok || got != want {
			t.Errorf("Get(%q) = %q,%v; want %q,true", k, got, ok, want)
		}
	}
	if _, ok := c.Get("nope"); ok {
		t.Error("Get on an unknown key must report ok=false")
	}
	err := c.Set("nope", "x")
	if err == nil {
		t.Fatal("Set on an unknown key must error")
	}
	for _, k := range CentralKeys() {
		if !strings.Contains(err.Error(), k) {
			t.Errorf("unknown-key error must name the valid key %q, got %q", k, err.Error())
		}
	}
}

func TestMaskSecret(t *testing.T) {
	if got := MaskSecret(""); got != "(unset)" {
		t.Errorf("MaskSecret(\"\") = %q, want (unset)", got)
	}
	short := "abc123"
	if got := MaskSecret(short); strings.Contains(got, "abc") || strings.Contains(got, "123") {
		t.Errorf("MaskSecret(%q) = %q — a short secret must expose no tail", short, got)
	}
	// The boundary itself: at exactly 8 characters a 4-char tail is HALF the secret, which is as
	// identifying as it is at 7. The cut-off has to include 8, not stop just below it.
	boundary := "abcd1234"
	if got := MaskSecret(boundary); got != "(set)" {
		t.Errorf("MaskSecret(%q) = %q — an 8-char secret must expose no tail (a 4-char tail is half of it)", boundary, got)
	}
	if got := MaskSecret("abcde1234"); !strings.HasSuffix(got, "1234)") {
		t.Errorf("MaskSecret on a 9-char secret = %q, want a last-4 tail (9 is the first length that gets one)", got)
	}
	long := "sk-test-0123456789abcdef"
	got := MaskSecret(long)
	if strings.Contains(got, long) {
		t.Errorf("MaskSecret leaked the whole secret: %q", got)
	}
	if !strings.HasSuffix(got, "cdef)") {
		t.Errorf("MaskSecret(%q) = %q, want a last-4 tail", long, got)
	}
	// Never more than the last 4 characters.
	if strings.Contains(got, "abcdef") {
		t.Errorf("MaskSecret leaked more than the last 4 chars: %q", got)
	}
}

func TestResolveAPIKeyPrecedence(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	const envName = "EXAMPLE_LLM_KEY"
	t.Setenv(envName, "")

	// Nothing set anywhere.
	if key, source := ResolveAPIKey(envName); key != "" || source != "" {
		t.Fatalf("unresolved key = %q (%s), want empty/empty", key, source)
	}

	// Central supplies it.
	c := &Central{}
	mustSet(t, c, "llm.api_key", "sk-central-fake-0000")
	if err := SaveCentral(c); err != nil {
		t.Fatal(err)
	}
	key, source := ResolveAPIKey(envName)
	if key != "sk-central-fake-0000" || source != SourceCentral {
		t.Fatalf("key = %q (%s), want the central key (central)", key, source)
	}

	// Env wins.
	t.Setenv(envName, "sk-env-fake-1111")
	key, source = ResolveAPIKey(envName)
	if key != "sk-env-fake-1111" || source != SourceEnv {
		t.Fatalf("key = %q (%s), want the env key (env)", key, source)
	}
}

func TestLoadCentralMalformedYAMLErrors(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	p, err := CentralPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("llm: [unclosed\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadCentral(); err == nil {
		t.Fatal("LoadCentral must surface a malformed central file rather than silently ignoring it")
	}
}

// Precedence: env > profile > central > default, asserted on both the value and the
// recorded source, for the two model fields that gain a central leg.
func TestLoadCentralPrecedence(t *testing.T) {
	deskDir := t.TempDir()
	restore := chdir(t, deskDir)
	defer restore()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("DESK_ROOT", deskDir)
	t.Setenv("DESK_NAME", "example-desk")
	os.Unsetenv("LLM_PROVIDER")
	os.Unsetenv("LLM_MODEL")

	// No central file: the built-in defaults, sourced "default".
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.LLMProvider != "anthropic" || cfg.Sources["LLM_PROVIDER"] != SourceDefault {
		t.Errorf("no central: provider = %q (%s), want anthropic (default)", cfg.LLMProvider, cfg.Sources["LLM_PROVIDER"])
	}

	// Central beats the default.
	central := &Central{}
	mustSet(t, central, "llm.provider", "openai")
	mustSet(t, central, "llm.model", "example-central-model")
	if err := SaveCentral(central); err != nil {
		t.Fatal(err)
	}
	cfg, err = Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.LLMProvider != "openai" || cfg.Sources["LLM_PROVIDER"] != SourceCentral {
		t.Errorf("central provider = %q (%s), want openai (central)", cfg.LLMProvider, cfg.Sources["LLM_PROVIDER"])
	}
	if cfg.LLMModel != "example-central-model" || cfg.Sources["LLM_MODEL"] != SourceCentral {
		t.Errorf("central model = %q (%s), want example-central-model (central)", cfg.LLMModel, cfg.Sources["LLM_MODEL"])
	}

	// Profile beats central.
	if err := os.MkdirAll(filepath.Join(deskDir, "_knowledge"), 0o755); err != nil {
		t.Fatal(err)
	}
	profile := "desk:\n  name: example-desk\nmodels:\n  provider: gemini\n  model: example-profile-model\n"
	if err := os.WriteFile(filepath.Join(deskDir, "_knowledge", "profile.yaml"), []byte(profile), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err = Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.LLMProvider != "gemini" || cfg.Sources["LLM_PROVIDER"] != SourceProfile {
		t.Errorf("profile provider = %q (%s), want gemini (profile)", cfg.LLMProvider, cfg.Sources["LLM_PROVIDER"])
	}

	// Env beats everything.
	t.Setenv("LLM_PROVIDER", "anthropic")
	t.Setenv("LLM_MODEL", "example-env-model")
	cfg, err = Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.LLMProvider != "anthropic" || cfg.Sources["LLM_PROVIDER"] != SourceEnv {
		t.Errorf("env provider = %q (%s), want anthropic (env)", cfg.LLMProvider, cfg.Sources["LLM_PROVIDER"])
	}
	if cfg.LLMModel != "example-env-model" || cfg.Sources["LLM_MODEL"] != SourceEnv {
		t.Errorf("env model = %q (%s), want example-env-model (env)", cfg.LLMModel, cfg.Sources["LLM_MODEL"])
	}
}

// default_desk supplies DESK_NAME when neither env nor a profile does — and only then.
func TestLoadCentralDefaultDesk(t *testing.T) {
	deskDir := t.TempDir()
	restore := chdir(t, deskDir)
	defer restore()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("DESK_ROOT", deskDir)
	os.Unsetenv("DESK_NAME")

	// No central default_desk: still unresolvable.
	if _, err := Load(); err == nil {
		t.Fatal("Load must still error when DESK_NAME is unresolvable and central sets no default_desk")
	}

	central := &Central{}
	mustSet(t, central, "default_desk", "example-desk")
	if err := SaveCentral(central); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatalf("central default_desk must resolve DESK_NAME: %v", err)
	}
	if cfg.DeskName != "example-desk" || cfg.Sources["DESK_NAME"] != SourceCentral {
		t.Errorf("DeskName = %q (%s), want example-desk (central)", cfg.DeskName, cfg.Sources["DESK_NAME"])
	}

	// Env still wins.
	t.Setenv("DESK_NAME", "env-desk")
	cfg, err = Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DeskName != "env-desk" || cfg.Sources["DESK_NAME"] != SourceEnv {
		t.Errorf("DeskName = %q (%s), want env-desk (env)", cfg.DeskName, cfg.Sources["DESK_NAME"])
	}
}

// A malformed central file must not break Load: it is treated as absent (and reported on
// stderr), never silently substituted for a resolved value.
func TestLoadToleratesMalformedCentral(t *testing.T) {
	deskDir := t.TempDir()
	restore := chdir(t, deskDir)
	defer restore()
	cfgHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgHome)
	t.Setenv("DESK_ROOT", deskDir)
	t.Setenv("DESK_NAME", "example-desk")
	os.Unsetenv("LLM_PROVIDER")

	p, err := CentralPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("llm: [unclosed\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatalf("a malformed central file must not fail Load: %v", err)
	}
	if cfg.LLMProvider != "anthropic" || cfg.Sources["LLM_PROVIDER"] != SourceDefault {
		t.Errorf("provider = %q (%s), want the anthropic default", cfg.LLMProvider, cfg.Sources["LLM_PROVIDER"])
	}
}

// Every source Load records must be one of the four legs. The keys are taken from Sources
// ITSELF, never hand-listed: a hand-copied list silently stops covering a field the moment one
// is added (it drifted to 22 of 24 exactly that way). The other half of this contract — that the
// Sources key set and `config show`'s rows are the SAME set, in both directions — is asserted in
// the cmd package, where the display table lives.
func TestSourcesCoverResolvedFields(t *testing.T) {
	deskDir := t.TempDir()
	restore := chdir(t, deskDir)
	defer restore()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("DESK_ROOT", deskDir)
	t.Setenv("DESK_NAME", "example-desk")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Sources) == 0 {
		t.Fatal("Load recorded no sources at all")
	}
	for k, source := range cfg.Sources {
		switch source {
		case SourceEnv, SourceProfile, SourceCentral, SourceDefault:
		default:
			t.Errorf("Sources[%q] = %q, want one of env/profile/central/default", k, source)
		}
	}
}

// IGNORE_CONFIG has no leg of its own: absent an env var it is DERIVED from the resolved
// DeskRoot. Its recorded source must therefore be the leg DESK_ROOT won on — a row whose value
// visibly traces to the profile while the SOURCE column says "default" is exactly the
// value/source disagreement Sources exists to make impossible.
func TestIgnoreConfigSourceFollowsDeskRoot(t *testing.T) {
	deskDir := t.TempDir()
	restore := chdir(t, deskDir)
	defer restore()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("DESK_NAME", "example-desk")
	os.Unsetenv("IGNORE_CONFIG")

	// DESK_ROOT from the profile => the derived IGNORE_CONFIG is sourced "profile".
	os.Unsetenv("DESK_ROOT")
	if err := os.MkdirAll(filepath.Join(deskDir, "_knowledge"), 0o755); err != nil {
		t.Fatal(err)
	}
	profile := "desk:\n  name: example-desk\n  root: \".\"\n"
	if err := os.WriteFile(filepath.Join(deskDir, "_knowledge", "profile.yaml"), []byte(profile), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(cfg.DeskRoot, ".deskkitignore"); cfg.IgnoreConfig != want {
		t.Fatalf("IgnoreConfig = %q, want the DeskRoot-derived %q", cfg.IgnoreConfig, want)
	}
	if got, deskRoot := cfg.Sources["IGNORE_CONFIG"], cfg.Sources["DESK_ROOT"]; got != deskRoot {
		t.Errorf("Sources[IGNORE_CONFIG] = %q but the path it was derived from came from %q",
			got, deskRoot)
	}

	// DESK_ROOT from the env => the same derived value is sourced "env".
	t.Setenv("DESK_ROOT", deskDir)
	cfg, err = Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Sources["DESK_ROOT"] != SourceEnv {
		t.Fatalf("setup: DESK_ROOT source = %q, want env", cfg.Sources["DESK_ROOT"])
	}
	if got := cfg.Sources["IGNORE_CONFIG"]; got != SourceEnv {
		t.Errorf("Sources[IGNORE_CONFIG] = %q, want env (it is derived from an env-supplied DESK_ROOT)", got)
	}

	// An explicit IGNORE_CONFIG still reports its own env leg, not DESK_ROOT's.
	t.Setenv("IGNORE_CONFIG", filepath.Join(deskDir, "custom-ignore"))
	cfg, err = Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.IgnoreConfig != filepath.Join(deskDir, "custom-ignore") || cfg.Sources["IGNORE_CONFIG"] != SourceEnv {
		t.Errorf("explicit IGNORE_CONFIG = %q (%s), want the env value (env)",
			cfg.IgnoreConfig, cfg.Sources["IGNORE_CONFIG"])
	}
}

func mustSet(t *testing.T, c *Central, key, val string) {
	t.Helper()
	if err := c.Set(key, val); err != nil {
		t.Fatal(err)
	}
}

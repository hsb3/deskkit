package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hsb3/desk-standard/librarian/internal/core/config"
)

// fakeAPIKey is a made-up, identity-neutral secret: long enough that MaskSecret shows a tail,
// and distinctive enough to grep the whole rendered output for.
const fakeAPIKey = "sk-fake-key-for-tests-0000wxyz"

// isolate redirects both XDG homes to throwaway dirs and gives the process a resolvable desk.
// Every config test calls it: nothing here may read or write the operator's real config/store.
func isolate(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("DESK_ROOT", t.TempDir())
	t.Setenv("DESK_NAME", "d")
}

// writeCentral seeds the machine-wide config file through the real writer.
func writeCentral(t *testing.T, mutate func(c *config.Central)) {
	t.Helper()
	c := &config.Central{}
	mutate(c)
	if err := config.SaveCentral(c); err != nil {
		t.Fatalf("seed central config: %v", err)
	}
}

// showRow returns the rendered `config show` row for a setting key, split into fields.
func showRow(t *testing.T, out, key string) []string {
	t.Helper()
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), key+" ") {
			return strings.Fields(line)
		}
	}
	t.Fatalf("`config show` has no %s row:\n%s", key, out)
	return nil
}

// TestConfigShow_EnvBeatsCentralThenFallsBack proves the precedence chain is REPORTED, not
// guessed: the same key set in both places shows the env value with source "env"; unset the env
// var and the same run falls to the central value with source "central".
func TestConfigShow_EnvBeatsCentralThenFallsBack(t *testing.T) {
	isolate(t)
	writeCentral(t, func(c *config.Central) { c.LLM.Model = "model-from-central" })
	t.Setenv("LLM_MODEL", "model-from-env")

	out := runCmdOut(t, newConfigCmd(), "show")
	row := showRow(t, out, "LLM_MODEL")
	if row[1] != "model-from-env" || row[2] != config.SourceEnv {
		t.Fatalf("env must win: LLM_MODEL row = %v, want [LLM_MODEL model-from-env env]", row)
	}

	if err := os.Unsetenv("LLM_MODEL"); err != nil { // t.Setenv restores it after the test
		t.Fatalf("unset LLM_MODEL: %v", err)
	}
	out = runCmdOut(t, newConfigCmd(), "show")
	row = showRow(t, out, "LLM_MODEL")
	if row[1] != "model-from-central" || row[2] != config.SourceCentral {
		t.Fatalf("central must win once env is unset: LLM_MODEL row = %v", row)
	}
}

// TestConfigSet_RoundTripsThroughShow: the write surface and the read surface agree.
func TestConfigSet_RoundTripsThroughShow(t *testing.T) {
	isolate(t)

	setOut := runCmdOut(t, newConfigCmd(), "set", "llm.model", "round-trip-model")
	if !strings.Contains(setOut, "round-trip-model") {
		t.Errorf("`config set` should confirm the value it stored:\n%s", setOut)
	}

	row := showRow(t, runCmdOut(t, newConfigCmd(), "show"), "LLM_MODEL")
	if row[1] != "round-trip-model" || row[2] != config.SourceCentral {
		t.Fatalf("set value did not round-trip through show: %v", row)
	}
}

// TestConfigSet_WritesA0600File: the central file may hold the API key, so its mode is part of
// the contract.
func TestConfigSet_WritesA0600File(t *testing.T) {
	isolate(t)

	runCmdOut(t, newConfigCmd(), "set", "llm.provider", "anthropic")

	path, err := config.CentralPath()
	if err != nil {
		t.Fatalf("CentralPath: %v", err)
	}
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatalf("central config was not created: %v", err)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Fatalf("central config mode = %v, want 0600", fi.Mode().Perm())
	}
}

// TestConfigNeverPrintsTheRawAPIKey greps every surface that could leak it: `show` with the key
// in the central file AND in the environment, and `set`'s own confirmation line.
func TestConfigNeverPrintsTheRawAPIKey(t *testing.T) {
	isolate(t)
	writeCentral(t, func(c *config.Central) { c.LLM.APIKey = fakeAPIKey })
	t.Setenv("ANTHROPIC_API_KEY", fakeAPIKey+"-env")

	show := runCmdOut(t, newConfigCmd(), "show")
	if strings.Contains(show, fakeAPIKey) {
		t.Fatalf("`config show` printed the raw API key:\n%s", show)
	}
	if !strings.Contains(show, config.MaskSecret(fakeAPIKey+"-env")) {
		t.Errorf("`config show` should render the key masked:\n%s", show)
	}

	set := runCmdOut(t, newConfigCmd(), "set", "llm.api_key", fakeAPIKey)
	if strings.Contains(set, fakeAPIKey) {
		t.Fatalf("`config set llm.api_key` echoed the raw value back:\n%s", set)
	}
}

// TestConfigShow_SourceColumnCoversEveryResolvedKey: the table must render every key the
// resolver records a source for — a field added to Config must not silently vanish from `show`.
func TestConfigShow_SourceColumnCoversEveryResolvedKey(t *testing.T) {
	isolate(t)
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	shown := map[string]bool{}
	for _, r := range configRows(cfg) {
		shown[r.key] = true
	}
	for key := range cfg.Sources {
		if !shown[key] {
			t.Errorf("`config show` omits %s (add it to configRows)", key)
		}
	}
}

// TestConfigShow_ReportsStoreAndCentralPaths: `show` answers "where does my data live" too.
func TestConfigShow_ReportsStoreAndCentralPaths(t *testing.T) {
	isolate(t)
	out := runCmdOut(t, newConfigCmd(), "show")

	centralPath, err := config.CentralPath()
	if err != nil {
		t.Fatalf("CentralPath: %v", err)
	}
	if !strings.Contains(out, centralPath) {
		t.Errorf("`config show` omits the central config path:\n%s", out)
	}
	storeDir := filepath.Join(os.Getenv("XDG_DATA_HOME"), "deskkit", "d")
	if !strings.Contains(out, storeDir) {
		t.Errorf("`config show` omits the resolved store path %s:\n%s", storeDir, out)
	}
	if !strings.Contains(out, "deskkit gui") || !strings.Contains(out, "deskkit config path") {
		t.Errorf("`config show` must point at the visual browser and `config path`:\n%s", out)
	}
}

// TestConfigShow_UnresolvedDeskStillReports: outside a desk, `show` still reports the central
// config rather than failing — it is the command you run to find out what is set.
func TestConfigShow_UnresolvedDeskStillReports(t *testing.T) {
	isolate(t)
	t.Setenv("DESK_ROOT", "")
	t.Setenv("DESK_NAME", "")
	writeCentral(t, func(c *config.Central) { c.LLM.Provider = "openai" })

	out := runCmdOut(t, newConfigCmd(), "show")
	if !strings.Contains(out, "openai") {
		t.Errorf("`config show` outside a desk must still show the central config:\n%s", out)
	}
	if !strings.Contains(out, "deskkit init") {
		t.Errorf("`config show` outside a desk must say how to make one:\n%s", out)
	}
}

// TestConfigPath_PrintsPathAndWhetherItExists
func TestConfigPath_PrintsPathAndWhetherItExists(t *testing.T) {
	isolate(t)
	path, err := config.CentralPath()
	if err != nil {
		t.Fatalf("CentralPath: %v", err)
	}

	out := runCmdOut(t, newConfigCmd(), "path")
	if !strings.Contains(out, path) {
		t.Fatalf("`config path` must print %s:\n%s", path, out)
	}
	if !strings.Contains(out, "not created yet") {
		t.Errorf("`config path` must say the file does not exist yet:\n%s", out)
	}

	runCmdOut(t, newConfigCmd(), "set", "llm.provider", "anthropic")
	out = runCmdOut(t, newConfigCmd(), "path")
	if !strings.Contains(out, "exists") || strings.Contains(out, "not created yet") {
		t.Errorf("`config path` must say the file exists once written:\n%s", out)
	}
}

// TestConfigSet_UnknownKeyNamesTheValidOnes
func TestConfigSet_UnknownKeyNamesTheValidOnes(t *testing.T) {
	isolate(t)
	cmd := newConfigCmd()
	cmd.SetArgs([]string{"set", "llm.nope", "x"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("`config set` accepted an unknown key")
	}
	for _, key := range config.CentralKeys() {
		if !strings.Contains(err.Error(), key) {
			t.Errorf("the unknown-key error must name %s: %v", key, err)
		}
	}
}

// TestConfigEdit_WithoutAnEditorNamesTheEnvVars: the fallback has to be actionable.
func TestConfigEdit_WithoutAnEditorNamesTheEnvVars(t *testing.T) {
	isolate(t)
	for _, k := range []string{"VISUAL", "EDITOR"} {
		t.Setenv(k, "")
		if err := os.Unsetenv(k); err != nil {
			t.Fatalf("unset %s: %v", k, err)
		}
	}

	cmd := newConfigCmd()
	cmd.SetArgs([]string{"edit"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("`config edit` with no editor set must error")
	}
	if !strings.Contains(err.Error(), "VISUAL") || !strings.Contains(err.Error(), "EDITOR") {
		t.Fatalf("the no-editor error must name $VISUAL and $EDITOR: %v", err)
	}
}

// TestConfigShow_CreatesNoStore mirrors the desks guarantee: reading configuration must not
// materialize a store directory.
func TestConfigShow_CreatesNoStore(t *testing.T) {
	isolate(t)
	runCmdOut(t, newConfigCmd(), "show")
	if _, err := os.Stat(filepath.Join(os.Getenv("XDG_DATA_HOME"), "deskkit", "d")); err == nil {
		t.Fatal("`config show` created a store directory; it must only read")
	}
}

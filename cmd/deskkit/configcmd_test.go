package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/tests"

	"github.com/hsb3/deskkit/internal/core/config"
	"github.com/hsb3/deskkit/internal/core/settings"
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

// writeMalformedCentral puts a file that does NOT parse at the central config path, through the
// real path resolver, and returns that path.
func writeMalformedCentral(t *testing.T) string {
	t.Helper()
	path, err := config.CentralPath()
	if err != nil {
		t.Fatalf("CentralPath: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("create central config dir: %v", err)
	}
	if err := os.WriteFile(path, []byte("llm: [unclosed\n"), 0o600); err != nil {
		t.Fatalf("write malformed central config: %v", err)
	}
	return path
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

// TestConfigShow_RowsAndSourcesAreTheSameSet cross-checks the two artifacts against EACH OTHER,
// in both directions, with neither list written out here: every key the resolver recorded a
// source for has a display row, and every display row has a recorded source. So a field added to
// Config can neither silently vanish from `show` nor appear in it with an empty SOURCE column.
func TestConfigShow_RowsAndSourcesAreTheSameSet(t *testing.T) {
	isolate(t)
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	rows := configRows(cfg)
	if len(rows) == 0 || len(cfg.Sources) == 0 {
		t.Fatalf("nothing to compare: %d rows, %d sources", len(rows), len(cfg.Sources))
	}

	shown := map[string]bool{}
	for _, r := range rows {
		shown[r.key] = true
		if r.source == "" {
			t.Errorf("`config show` renders %s with an empty SOURCE column "+
				"(resolve it through the resolver so its leg is recorded)", r.key)
		}
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

// TestConfigShow_UnresolvedDeskWithMalformedCentralStillReports is the both-broken case: no desk
// resolves here AND the machine-wide file does not parse. That is precisely when an operator runs
// this command, so it must still succeed and name the file to fix — the previous behaviour
// returned LoadCentral's error, exiting 1 with nothing actionable, while the SAME broken file on a
// resolvable desk exited 0 with a warning.
func TestConfigShow_UnresolvedDeskWithMalformedCentralStillReports(t *testing.T) {
	isolate(t)
	t.Setenv("DESK_ROOT", "")
	t.Setenv("DESK_NAME", "")
	path := writeMalformedCentral(t)

	// runCmdOut fails the test on a non-nil error from Execute — i.e. on the exit-1 behaviour.
	out := runCmdOut(t, newConfigCmd(), "show")

	if !strings.Contains(out, path) {
		t.Errorf("`config show` must name the unparseable central file %s:\n%s", path, out)
	}
	if !strings.Contains(out, "unparseable") {
		t.Errorf("`config show` must mark the central file as unparseable:\n%s", out)
	}
	if !strings.Contains(out, "deskkit config edit") && !strings.Contains(out, "deskkit config set") {
		t.Errorf("`config show` must still say how to change a machine-wide value:\n%s", out)
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

// TestConfigSet_MalformedFileErrorNamesTheRepair: `set` cannot rewrite a file it could not parse
// (that would silently discard whatever the operator meant to keep), so its error has to hand
// over the one command that CAN repair it — `config edit` stats the path and opens the raw file,
// never going through the parser.
func TestConfigSet_MalformedFileErrorNamesTheRepair(t *testing.T) {
	isolate(t)
	path := writeMalformedCentral(t)

	cmd := newConfigCmd()
	cmd.SetArgs([]string{"set", "llm.model", "x"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("`config set` over a malformed central file must error rather than overwrite it")
	}
	if !strings.Contains(err.Error(), "deskkit config edit") {
		t.Errorf("the error must point at the command that can repair the file: %v", err)
	}
	if !strings.Contains(err.Error(), path) {
		t.Errorf("the error must name the file to repair (%s): %v", path, err)
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

// TestReadOnlyCommandsCreateNoStore_Subprocess guards the headline safety property of `config`
// and `desks`: asking what is configured, or which desks exist, must materialize NOTHING under
// the data home.
//
// It has to be a subprocess. The property does not come from the commands themselves — it comes
// from main() intercepting them (isInterceptedInvocation) BEFORE PocketBase's Bootstrap
// MkdirAll-creates a store dir, because Bootstrap runs for every KNOWN registered command. An
// in-process test calling newConfigCmd()/newDesksCmd() never constructs PocketBase at all, so it
// cannot fail even with the interception deleted — it was proving nothing, and the two such tests
// this replaces are gone. Building and running the REAL binary is the only shape that goes red
// when the guard does (same rationale as TestPMActorBeforeLeaf_Subprocess in pm_test.go).
func TestReadOnlyCommandsCreateNoStore_Subprocess(t *testing.T) {
	binPath := filepath.Join(t.TempDir(), "deskkit-readonly-test")
	build := exec.Command("go", "build", "-o", binPath, ".")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build deskkit: %v\n%s", err, out)
	}

	// Both XDG homes are private to this run: the subprocess inherits os.Environ(), so without
	// these it would read — and a regression would WRITE — the operator's real ones.
	deskDir := t.TempDir()
	dataHome := t.TempDir()

	for _, args := range [][]string{{"config", "show"}, {"config", "path"}, {"desks"}} {
		label := strings.Join(args, " ")
		cmd := exec.Command(binPath, args...)
		cmd.Dir = deskDir
		cmd.Env = append(os.Environ(),
			"XDG_DATA_HOME="+dataHome,
			"XDG_CONFIG_HOME="+t.TempDir(),
			// A resolvable desk is the DANGEROUS case: an unresolvable one has no store
			// location to create in the first place, so it would pass either way.
			"DESK_ROOT="+deskDir,
			"DESK_NAME=ghost",
		)
		var out, errOut bytes.Buffer
		cmd.Stdout, cmd.Stderr = &out, &errOut
		if err := cmd.Run(); err != nil {
			t.Fatalf("deskkit %s: %v\nstdout: %s\nstderr: %s", label, err, out.String(), errOut.String())
		}

		entries, err := os.ReadDir(dataHome)
		if err != nil {
			t.Fatalf("read data home: %v", err)
		}
		if len(entries) != 0 {
			var names []string
			for _, e := range entries {
				names = append(names, e.Name())
			}
			t.Fatalf("`deskkit %s` created %v under the data home; it must only read "+
				"(main() must intercept it before PocketBase bootstraps a store)", label, names)
		}
	}
}

// seedStoreSettings materializes a REAL store at the resolved store path, carrying the given LLM
// settings. It builds the store through the migration runner (so the settings collection and its
// seeded singleton are the shipped ones), closes it, and copies the whole data dir into place —
// `config show` then reads it exactly as it would read an operator's own desk.
func seedStoreSettings(t *testing.T, provider, model, apiKey string) {
	t.Helper()
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("tests.NewTestApp: %v", err)
	}
	rec, err := app.FindRecordById(settings.Collection, settings.RecordID)
	if err != nil {
		app.Cleanup()
		t.Fatalf("find the seeded settings row: %v", err)
	}
	rec.Set(settings.FieldProvider, provider)
	rec.Set(settings.FieldModel, model)
	rec.Set(settings.FieldAPIKey, apiKey)
	if err := app.Save(rec); err != nil {
		app.Cleanup()
		t.Fatalf("save settings: %v", err)
	}
	src := app.DataDir()
	// Close the DB before copying: a live WAL would leave the copy missing the just-written row.
	if err := app.ResetBootstrapState(); err != nil {
		t.Fatalf("close the seeded store: %v", err)
	}
	dst := filepath.Join(os.Getenv("XDG_DATA_HOME"), "deskkit", os.Getenv("DESK_NAME"))
	if err := os.CopyFS(dst, os.DirFS(src)); err != nil {
		t.Fatalf("copy the seeded store into place: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(src) })
}

// TestConfigShow_StoreLegBeatsCentral is the resolution-order proof at the display surface: with a
// value in BOTH the desk's store and the machine-wide file, `config show` reports the store's — and
// says so in the SOURCE column, so an operator can see which leg won.
func TestConfigShow_StoreLegBeatsCentral(t *testing.T) {
	isolate(t)
	writeCentral(t, func(c *config.Central) {
		c.LLM.Provider = "gemini"
		c.LLM.Model = "model-from-central"
		c.LLM.APIKey = fakeAPIKey
	})
	seedStoreSettings(t, "openai", "model-from-store", "sk-store-key-0000abcd")

	out := runCmdOut(t, newConfigCmd(), "show")

	for _, key := range []string{"LLM_PROVIDER", "LLM_MODEL"} {
		row := showRow(t, out, key)
		if row[len(row)-1] != "store" {
			t.Errorf("%s won on %q, want the store leg:\n%s", key, row[len(row)-1], out)
		}
	}
	if !strings.Contains(out, "model-from-store") || strings.Contains(out, "model-from-central") {
		t.Errorf("`config show` must report the store's model, not the central one:\n%s", out)
	}
	if !strings.Contains(out, "(set, …abcd)") {
		t.Errorf("the API key row must resolve to the store's key, masked:\n%s", out)
	}
	if strings.Contains(out, "sk-store-key-0000abcd") || strings.Contains(out, fakeAPIKey) {
		t.Fatalf("`config show` printed a raw API key:\n%s", out)
	}
}

// TestConfigShow_EnvAndProfileStillBeatStore: a value the operator exported, or one the desk
// declares in its own profile, is not overridden by whatever a browser last wrote to the store.
func TestConfigShow_EnvAndProfileStillBeatStore(t *testing.T) {
	isolate(t)
	seedStoreSettings(t, "openai", "model-from-store", "")
	t.Setenv("LLM_MODEL", "model-from-env")

	out := runCmdOut(t, newConfigCmd(), "show")

	row := showRow(t, out, "LLM_MODEL")
	if row[len(row)-1] != "env" {
		t.Errorf("the store overrode the env leg (source %q):\n%s", row[len(row)-1], out)
	}
	if !strings.Contains(out, "model-from-env") {
		t.Errorf("`config show` must report the env model:\n%s", out)
	}
	// The provider, which env does NOT set, still comes from the store — proving the two legs are
	// resolved per field rather than all-or-nothing.
	if row := showRow(t, out, "LLM_PROVIDER"); row[len(row)-1] != "store" {
		t.Errorf("LLM_PROVIDER won on %q, want the store leg:\n%s", row[len(row)-1], out)
	}
}

// TestConfigShow_NoStoreYetConsultsNothing: a desk whose store has never been created must not
// have a "store" row invented for it — the command may only report a leg it actually consulted.
func TestConfigShow_NoStoreYetConsultsNothing(t *testing.T) {
	isolate(t)
	out := runCmdOut(t, newConfigCmd(), "show")

	for _, key := range []string{"LLM_PROVIDER", "LLM_MODEL"} {
		if row := showRow(t, out, key); row[len(row)-1] == "store" {
			t.Errorf("%s claims the store leg with no store on disk:\n%s", key, out)
		}
	}
	if !strings.Contains(out, "env > profile > store > central > default") {
		t.Errorf("`config show` must state the full precedence chain:\n%s", out)
	}
}

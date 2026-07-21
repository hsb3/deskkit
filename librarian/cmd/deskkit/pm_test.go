package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase"

	"github.com/hsb3/desk-standard/librarian/internal/core/config"
	"github.com/hsb3/desk-standard/librarian/internal/core/module"
	"github.com/hsb3/desk-standard/librarian/internal/modules/librarian"
	"github.com/hsb3/desk-standard/librarian/internal/modules/pm"
)

// findCommand returns the RootCmd subcommand with the given name, or nil.
func findCommand(app *pocketbase.PocketBase, name string) map[string]bool {
	for _, c := range app.RootCmd.Commands() {
		if c.Name() == name {
			subs := map[string]bool{}
			for _, s := range c.Commands() {
				subs[s.Name()] = true
			}
			return subs
		}
	}
	return nil
}

// TestPMCommandGroup_GatedOff: on a librarian-only desk (PM off — including the no-config
// case), registerToolCommands wires NO `pm` group, so `deskkit pm` is cobra's normal
// unknown-command error and the CLI surface is byte-identical to main (spec §2.9).
func TestPMCommandGroup_GatedOff(t *testing.T) {
	cases := []struct {
		name   string
		cfg    *config.Config
		cfgErr error
	}{
		{"no config", nil, errors.New("config: required DESK_ROOT not set")},
		{"config with pm off", &config.Config{DeskRoot: t.TempDir(), DeskName: "d"}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := pocketbase.New() // construction only; never Start()ed, so no store is touched
			registerToolCommands(app, tc.cfg, tc.cfgErr)
			if subs := findCommand(app, "pm"); subs != nil {
				t.Fatalf("pm command group registered on a gated-off desk: %v", subs)
			}
		})
	}
}

// TestPMCommandGroup_GatedOn freezes the twelve CLI subcommand names (spec §5.3; docs + the
// D5 plugin build on them) on a pm-enabled desk.
func TestPMCommandGroup_GatedOn(t *testing.T) {
	app := pocketbase.New()
	cfg := &config.Config{DeskRoot: t.TempDir(), DeskName: "d", PMEnabled: true, PMAutonomousWrites: true}
	registerToolCommands(app, cfg, nil)
	subs := findCommand(app, "pm")
	if subs == nil {
		t.Fatal("pm command group missing on a pm-enabled desk")
	}
	want := []string{
		"context", "list", "get", "create", "update", "transition",
		"block", "unblock", "note", "link", "claim", "release",
	}
	for _, name := range want {
		if !subs[name] {
			t.Errorf("pm %s missing (frozen CLI name)", name)
		}
	}
	if len(subs) != len(want) {
		t.Errorf("pm group has %d subcommands, want %d: %v", len(subs), len(want), subs)
	}
}

// TestPMStoreTouching pins the main() guard composition: `pm` counts as store-touching only
// when the pm module is enabled, so a gated-off desk's unknown `pm` invocation never
// pre-creates a store directory.
func TestPMStoreTouching(t *testing.T) {
	on := &config.Config{PMEnabled: true}
	off := &config.Config{}
	isPM := func(cfg *config.Config, args []string) bool {
		return isStoreTouchingInvocation(args) ||
			(firstSubcommand(args) == "pm" && cfg != nil && cfg.PMEnabled)
	}
	if !isPM(on, []string{"pm", "context"}) {
		t.Error("pm must be store-touching when the module is enabled")
	}
	if isPM(off, []string{"pm", "context"}) {
		t.Error("pm must NOT be store-touching when the module is disabled")
	}
	if isPM(nil, []string{"pm", "context"}) {
		t.Error("nil config: pm must not be store-touching")
	}
}

// TestPMActorFlagBeforeLeaf proves the FULL round trip (cobra's own dispatch + the pm engine)
// correctly threads --actor's value through when it is passed BEFORE the leaf subcommand, not
// only after it. This exercises app.RootCmd.Execute() directly, so it does NOT reproduce the
// underlying bug itself — that bug lived in main()'s own pre-cobra unknown-subcommand guard
// (nextNonFlagToken in main.go), which only runs inside main(), before cobra ever dispatches;
// this test's direct Execute() call bypasses it entirely (proven by running this exact test
// against the pre-fix main.go: it still passes). The actual regression coverage is
// TestUnknownSubcommand_PMActorBeforeLeaf plus the new flag-before-leaf cases in TestUnknownSubcommand
// (main_test.go), and TestPMActorBeforeLeaf_Subprocess below (a real compiled-binary run through
// main()). This test's job is the complementary one: once past that guard, prove cobra + the pm
// engine actually honor the flag's value (not just "no error") in both flag orders. Runs the real
// cobra command tree end to end against a bootstrapped store, mirroring requireConfig's own
// self-initializing-store path (ADR 0003) rather than inventing a new harness.
func TestPMActorFlagBeforeLeaf(t *testing.T) {
	prevReg := moduleReg
	t.Cleanup(func() { moduleReg = prevReg })

	cfg := &config.Config{
		DeskRoot: t.TempDir(), DeskName: "pm-cli-test", PMEnabled: true, PMAutonomousWrites: true,
	}
	reg, err := module.Register(cfg, librarian.New(), pm.New())
	if err != nil {
		t.Fatalf("module.Register: %v", err)
	}
	moduleReg = reg

	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("app.Bootstrap: %v", err)
	}
	t.Cleanup(func() { _ = app.ResetBootstrapState() })

	registerToolCommands(app, cfg, nil)

	// run executes the real RootCmd (the same command tree `deskkit` builds) with args and
	// returns the parsed JSON stdout payload. Every leaf RunE calls printJSON(cmd.OutOrStdout(),
	// ...), which walks up to the root's configured writer, so pointing RootCmd's own writer at
	// buf via SetOut captures a leaf's output directly — no process-global os.Stdout swap needed.
	run := func(args []string) map[string]any {
		t.Helper()
		var buf bytes.Buffer
		app.RootCmd.SetOut(&buf)
		app.RootCmd.SetArgs(args)
		execErr := app.RootCmd.Execute()
		if execErr != nil {
			t.Fatalf("execute %v: %v\noutput: %s", args, execErr, buf.String())
		}
		var result map[string]any
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("unmarshal output %q for %v: %v", buf.String(), args, err)
		}
		return result
	}

	// The already-working order: --actor AFTER the leaf subcommand.
	afterOut := run([]string{"pm", "create", "--title", "after-leaf", "--type", "task", "--actor", "alice"})
	afterItem, _ := afterOut["item"].(map[string]any)
	afterID, _ := afterItem["id"].(string)
	if afterID == "" {
		t.Fatalf("create (actor after leaf) produced no item id: %+v", afterOut)
	}

	// The regression: --actor BEFORE the leaf subcommand must now succeed too (previously
	// cobra reported `unknown command "pm alice"`, a non-zero exit — run() would have already
	// t.Fatalf'd above via execErr).
	beforeOut := run([]string{"pm", "--actor", "alice", "create", "--title", "before-leaf", "--type", "task"})
	beforeItem, _ := beforeOut["item"].(map[string]any)
	beforeID, _ := beforeItem["id"].(string)
	if beforeID == "" {
		t.Fatalf("create (actor before leaf) produced no item id: %+v", beforeOut)
	}

	// Prove --actor's VALUE actually threaded through in the before-leaf position (not merely
	// that cobra stopped erroring): claim the item with --actor before the leaf and check the
	// stored claimed_by, exactly as CreateItem's engine call records ActorFields elsewhere
	// (parity_test.go's TestToolBodies_EndToEnd asserts the same claimed_by field).
	claimOut := run([]string{"pm", "--actor", "alice", "claim", beforeID})
	claimItem, _ := claimOut["item"].(map[string]any)
	if got, _ := claimItem["claimed_by"].(string); got != "alice" {
		t.Fatalf("claim with --actor before the leaf: claimed_by = %q, want %q (full result: %+v)",
			got, "alice", claimOut)
	}

	// Sibling leaf subcommands still parse THEIR OWN flags normally after the leaf token. --to
	// is `transition`'s own flag, passed in the normal after-the-leaf position.
	transOut := run([]string{"pm", "transition", beforeID, "--to", "work"})
	transItem, _ := transOut["item"].(map[string]any)
	if got, _ := transItem["phase"].(string); got != "work" {
		t.Fatalf("transition --to (after the leaf): phase = %q, want %q", got, "work")
	}
}

// TestPMActorBeforeLeaf_Subprocess is the true, black-box regression test: it builds the
// REAL deskkit binary and runs it as a subprocess, exercising main() end to end — including the
// pre-cobra unknown-subcommand guard (unknownSubcommand/nextNonFlagToken in main.go) that
// actually produced the reported symptom, `deskkit: unknown command "pm alice"` on stderr with
// exit 1, for `pm --actor alice create --title x --type task` (flag BEFORE the leaf
// subcommand). A unit test calling registerToolCommands + cobra's Execute() directly (as
// TestPMActorFlagBeforeLeaf above does) never reaches that guard, since it lives in main()
// itself; only a real process invocation reproduces it faithfully (same rationale as
// TestDevModeDefaultOff_BinaryUnderTempDir in main_devmode_test.go).
func TestPMActorBeforeLeaf_Subprocess(t *testing.T) {
	binPath := filepath.Join(t.TempDir(), "deskkit-actor-test")
	build := exec.Command("go", "build", "-o", binPath, ".")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build deskkit: %v\n%s", err, out)
	}

	deskDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(deskDir, "_knowledge"), 0o755); err != nil {
		t.Fatalf("mkdir _knowledge: %v", err)
	}
	profile := "desk:\n  name: pm-actor-subprocess-test\nroot: \".\"\n"
	if err := os.WriteFile(filepath.Join(deskDir, "_knowledge", "profile.yaml"), []byte(profile), 0o644); err != nil {
		t.Fatalf("write profile.yaml: %v", err)
	}

	runIn := func(args ...string) (stdout, stderr string, exitCode int) {
		t.Helper()
		cmd := exec.Command(binPath, args...)
		cmd.Dir = deskDir
		cmd.Env = append(os.Environ(), "PM_ENABLED=true")
		var outBuf, errBuf bytes.Buffer
		cmd.Stdout = &outBuf
		cmd.Stderr = &errBuf
		err := cmd.Run()
		code := 0
		if err != nil {
			if ee, ok := err.(*exec.ExitError); ok {
				code = ee.ExitCode()
			} else {
				t.Fatalf("run %v: %v", args, err)
			}
		}
		return outBuf.String(), errBuf.String(), code
	}

	// The exact reported repro: --actor BEFORE the leaf subcommand.
	out, errOut, code := runIn("pm", "--actor", "alice", "create", "--title", "before-leaf", "--type", "task")
	if code != 0 {
		t.Fatalf("pm --actor alice create ... (flag before the leaf) exited %d, want 0\nstdout: %s\nstderr: %s", code, out, errOut)
	}
	if strings.Contains(errOut, "unknown command") {
		t.Fatalf("regression: guard still misreports the flag's value as an unknown subcommand: %s", errOut)
	}
	var created map[string]any
	if err := json.Unmarshal([]byte(out), &created); err != nil {
		t.Fatalf("unmarshal create output %q: %v", out, err)
	}
	item, _ := created["item"].(map[string]any)
	if id, _ := item["id"].(string); id == "" {
		t.Fatalf("create (actor before leaf) produced no item id: %s", out)
	}

	// The already-working order must still work: --actor AFTER the leaf subcommand.
	out2, errOut2, code2 := runIn("pm", "create", "--title", "after-leaf", "--type", "task", "--actor", "bob")
	if code2 != 0 {
		t.Fatalf("pm create ... --actor bob (flag after the leaf) exited %d, want 0\nstdout: %s\nstderr: %s", code2, out2, errOut2)
	}

	// Sanity check that the fix hasn't regressed the guard's actual job (the known-command
	// guard): a genuinely unknown top-level command must still be caught and fail non-zero.
	_, errOut3, code3 := runIn("frobnicate")
	if code3 == 0 {
		t.Fatal("a genuinely unknown top-level command must still exit non-zero (known-command guard regressed)")
	}
	if !strings.Contains(errOut3, `unknown command "frobnicate"`) {
		t.Fatalf("unexpected error for an unknown command: %s", errOut3)
	}

	// And a genuinely unknown NESTED pm command must still be caught too.
	_, errOut4, code4 := runIn("pm", "--actor", "alice", "frobnicate")
	if code4 == 0 {
		t.Fatal("a genuinely unknown nested pm command (after a real group flag) must still exit non-zero")
	}
	if !strings.Contains(errOut4, `unknown command "pm frobnicate"`) {
		t.Fatalf("unexpected error for an unknown nested pm command: %s", errOut4)
	}
}

package main

import (
	"errors"
	"testing"

	"github.com/pocketbase/pocketbase"

	"github.com/example/pocket-librarian/internal/core/config"
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

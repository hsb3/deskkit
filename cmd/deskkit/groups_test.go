package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
	"github.com/spf13/cobra"

	"github.com/hsb3/deskkit/internal/core/config"
)

// wantGroupTitles is the required help section order (the "alphabetic wall of commands" fix).
var wantGroupTitles = []string{
	"Setup & config:", "Inspect:", "Fix:", "Work graph:", "Agent:", "Admin:",
}

// buildFullRoot builds the SAME command tree main() builds — migratecmd, the tool commands, and
// finalizeCommandTree (serve/superuser + the display groups) — so these assertions describe the
// real binary's help, not a hand-assembled lookalike. The app is constructed but never Started,
// so no store is opened; the XDG homes are redirected regardless so nothing can touch the real
// ones.
func buildFullRoot(t *testing.T) *pocketbase.PocketBase {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	app := pocketbase.New()
	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{Automigrate: true})
	registerToolCommands(app, &config.Config{DeskRoot: t.TempDir(), DeskName: "d", PMEnabled: true}, nil)
	finalizeCommandTree(app)
	return app
}

// renderHelp executes the root command with --help against a buffer. Executing (rather than
// calling Help()) is deliberate: it is the path cobra takes in the real binary, so it also
// registers the built-in help/completion commands and runs checkCommandGroups — which PANICS on
// a GroupID that was never registered.
func renderHelp(t *testing.T, root *cobra.Command) string {
	t.Helper()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"--help"})
	if err := root.Execute(); err != nil {
		t.Fatalf("root --help: %v", err)
	}
	return buf.String()
}

// TestCommandGroups_EveryCommandIsGrouped is the definition-of-done assertion: no command may
// fall into cobra's "Additional Commands:" catch-all, which is exactly what happens when a new
// subcommand is registered without an entry in commandGroupIDs.
func TestCommandGroups_EveryCommandIsGrouped(t *testing.T) {
	app := buildFullRoot(t)
	out := renderHelp(t, app.RootCmd)

	if !app.RootCmd.AllChildCommandsHaveGroup() {
		var ungrouped []string
		for _, c := range app.RootCmd.Commands() {
			if c.IsAvailableCommand() && c.GroupID == "" {
				ungrouped = append(ungrouped, c.Name())
			}
		}
		t.Fatalf("commands without a display group (add them to commandGroupIDs): %v", ungrouped)
	}
	if strings.Contains(out, "Additional Commands:") {
		t.Fatalf("help renders an \"Additional Commands:\" section — some command is ungrouped:\n%s", out)
	}
}

// TestCommandGroups_RenderInOrder pins the section order in the rendered help.
func TestCommandGroups_RenderInOrder(t *testing.T) {
	app := buildFullRoot(t)
	out := renderHelp(t, app.RootCmd)

	prev := -1
	for _, title := range wantGroupTitles {
		at := strings.Index(out, "\n"+title)
		if at < 0 {
			t.Fatalf("help is missing the %q section:\n%s", title, out)
		}
		if at < prev {
			t.Fatalf("help section %q is out of order (want %v):\n%s", title, wantGroupTitles, out)
		}
		prev = at
	}
}

// TestCommandGroups_RegisteredInOrder pins the registered groups themselves: cobra renders them
// in AddGroup order, so this is the order the help section test above depends on.
func TestCommandGroups_RegisteredInOrder(t *testing.T) {
	app := buildFullRoot(t)
	groups := app.RootCmd.Groups()
	if len(groups) != len(wantGroupTitles) {
		t.Fatalf("registered %d groups, want %d", len(groups), len(wantGroupTitles))
	}
	for i, g := range groups {
		if g.Title != wantGroupTitles[i] {
			var got []string
			for _, x := range groups {
				got = append(got, x.Title)
			}
			t.Fatalf("group order: got %v, want %v", got, wantGroupTitles)
		}
	}
}

// TestCommandGroups_KeepFlatCommandIndent guards the grouped help's two-space command indent:
// the integration gate greps `deskkit --help` for '^  chat ' (verify.sh), so grouping
// must not re-indent command lines.
func TestCommandGroups_KeepFlatCommandIndent(t *testing.T) {
	app := buildFullRoot(t)
	out := renderHelp(t, app.RootCmd)
	for _, name := range []string{"chat", "desks", "config", "gui", "serve", "superuser", "migrate", "pm"} {
		if !strings.Contains(out, "\n  "+name+" ") {
			t.Errorf("help does not list %q at the expected two-space indent:\n%s", name, out)
		}
	}
}

// TestCommandGroups_GuiIsDiscoverable pins the GUI discoverability requirement: `gui` sits in
// the Inspect group and its one-liner says what it actually is.
func TestCommandGroups_GuiIsDiscoverable(t *testing.T) {
	app := buildFullRoot(t)
	for _, c := range app.RootCmd.Commands() {
		if c.Name() != "gui" {
			continue
		}
		if c.GroupID != groupInspect {
			t.Fatalf("gui GroupID = %q, want %q", c.GroupID, groupInspect)
		}
		if !strings.Contains(strings.ToLower(c.Short), "visual data browser") {
			t.Fatalf("gui help must name the visual data browser; got %q", c.Short)
		}
		return
	}
	t.Fatal("gui command is not registered")
}

// TestCommandGroups_WorkGraphHiddenWithoutPM: with the pm module gated off there is no `pm`
// command, so the Work graph section must not render as an empty heading.
func TestCommandGroups_WorkGraphHiddenWithoutPM(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	app := pocketbase.New()
	registerToolCommands(app, &config.Config{DeskRoot: t.TempDir(), DeskName: "d"}, nil)
	finalizeCommandTree(app)
	out := renderHelp(t, app.RootCmd)
	if strings.Contains(out, "Work graph:") {
		t.Fatalf("empty Work graph section rendered on a pm-disabled desk:\n%s", out)
	}
}

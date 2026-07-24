package main

import (
	"bytes"
	"os"
	"testing"

	"github.com/spf13/cobra"
)

// TestNoStaleSevenToolClaim guards against the return of the false "seven-tool core" help string:
// the librarian MCP default surface exposes 5 tools (6 with LIBRARIAN_AUTONOMOUS_WRITES, +12 under
// PM_ENABLED) and the CLI carries far more than seven subcommands — the authoritative map is
// docs/development/specs/tool-surface.md. RED before the fix (the mcp-serve Short said "seven-tool core"), green after.
func TestNoStaleSevenToolClaim(t *testing.T) {
	src, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}
	if bytes.Contains(src, []byte("seven-tool core")) {
		t.Error(`main.go still contains the false "seven-tool core" claim; describe the surface accurately (see docs/development/specs/tool-surface.md)`)
	}
}

func TestHasDirFlag(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want bool
	}{
		{"absent", []string{"sweep"}, false},
		{"value form", []string{"sweep", "--dir", "/tmp/store"}, true},
		{"equals form", []string{"sweep", "--dir=/tmp/store"}, true},
		{"valueless trailing", []string{"sweep", "--dir"}, true},
		{"before the subcommand", []string{"--dir", "/tmp/store", "sweep"}, true},
		{"equals form before the subcommand", []string{"--dir=/tmp/store", "sweep"}, true},
		{"unrelated flag only", []string{"sweep", "--dev"}, false},
		{"unrelated flag with similar prefix does not match", []string{"sweep", "--directory=/x"}, false},
		{"empty args", []string{}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := hasDirFlag(c.args); got != c.want {
				t.Errorf("hasDirFlag(%v) = %v, want %v", c.args, got, c.want)
			}
		})
	}
}

func TestFirstSubcommand(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want string
	}{
		{"bare subcommand", []string{"sweep"}, "sweep"},
		{"subcommand with trailing flags", []string{"sweep", "--dir", "/tmp/store"}, "sweep"},
		{"flags before the subcommand, value form", []string{"--dir", "/tmp/store", "serve"}, "serve"},
		{"flags before the subcommand, equals form", []string{"--dir=/tmp/store", "serve"}, "serve"},
		{"boolean global flag before the subcommand", []string{"--dev", "serve"}, "serve"},
		{"multiple value flags before the subcommand", []string{"--dir", "/x", "--queryTimeout", "5", "query"}, "query"},
		{"no subcommand (bare invocation)", []string{}, ""},
		{"only flags, no subcommand", []string{"--dev"}, ""},
		// Regression coverage for the --hooksDir bypass: an unregistered-in-this-build but
		// recognized value flag must still have its value skipped, not mistaken for the
		// subcommand — this is exactly how "--hooksDir /some/path serve" previously resolved to
		// "/some/path" (not store-touching) and let PocketBase Bootstrap create a stray data dir.
		{"hooksDir value form before the subcommand", []string{"--hooksDir", "/some/path", "serve"}, "serve"},
		{"hooksDir equals form before the subcommand", []string{"--hooksDir=/some/path", "serve"}, "serve"},
		{"hooksWatch value form before the subcommand", []string{"--hooksWatch", "true", "serve"}, "serve"},
		{"hooksPool value form before the subcommand", []string{"--hooksPool", "25", "serve"}, "serve"},
		{"multiple jsvm-style value flags before the subcommand", []string{"--hooksDir", "/x", "--hooksPool", "25", "sweep"}, "sweep"},
		// The narrated residual gap: a value flag NOT on globalValueFlags still shadows the
		// subcommand token — its value is treated as the subcommand. Pinned as current behavior
		// (documented in the globalValueFlags comment as "a genuinely novel unrecognized
		// value-flag ... could still shadow the subcommand token the same way"), not endorsed.
		{"unrecognized value flag shadows the subcommand", []string{"--novel", "/x", "serve"}, "/x"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := firstSubcommand(c.args); got != c.want {
				t.Errorf("firstSubcommand(%v) = %q, want %q", c.args, got, c.want)
			}
		})
	}
}

func TestSubcommandIndex(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want int
	}{
		{"bare subcommand", []string{"sweep"}, 0},
		{"value flag then subcommand", []string{"--dir", "/x", "serve"}, 2},
		{"equals form does not skip a value token", []string{"--dir=/x", "serve"}, 1},
		{"boolean global flag before the subcommand", []string{"--dev", "serve"}, 1},
		{"two value flags before the subcommand", []string{"--dir", "/x", "--queryTimeout", "5", "query"}, 4},
		{"no subcommand (bare invocation)", []string{}, -1},
		{"only flags, no subcommand", []string{"--dev"}, -1},
		{"trailing value flag with no value", []string{"--dir"}, -1},
		{"unrecognized value flag shadows the subcommand", []string{"--novel", "/x", "serve"}, 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := subcommandIndex(c.args); got != c.want {
				t.Errorf("subcommandIndex(%v) = %d, want %d", c.args, got, c.want)
			}
		})
	}
}

func TestIsInitInvocation(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want bool
	}{
		{"bare init", []string{"init"}, true},
		{"value flag before init", []string{"--dir", "/x", "init"}, true},
		{"init with force flag", []string{"init", "--force"}, true},
		{"init with help long flag", []string{"init", "--help"}, false},
		{"init with help short flag anywhere", []string{"init", "-h"}, false},
		{"help short flag before init", []string{"-h", "init"}, false},
		{"a different subcommand", []string{"serve"}, false},
		{"bare invocation", []string{}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := isInitInvocation(c.args); got != c.want {
				t.Errorf("isInitInvocation(%v) = %v, want %v", c.args, got, c.want)
			}
		})
	}
}

func TestHasNoInputFlag(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want bool
	}{
		{"bare no-input", []string{"init", "--no-input"}, true},
		{"equals true", []string{"init", "--no-input=true"}, true},
		{"equals false still present", []string{"init", "--no-input=false"}, true},
		{"absent", []string{"init", "--force"}, false},
		{"similar prefix does not match", []string{"--no-inputs"}, false},
		{"empty args", []string{}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := hasNoInputFlag(c.args); got != c.want {
				t.Errorf("hasNoInputFlag(%v) = %v, want %v", c.args, got, c.want)
			}
		})
	}
}

func TestStripNoInput(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want []string
	}{
		{"strips bare token", []string{"--no-input", "/dir"}, []string{"/dir"}},
		{"strips equals form", []string{"--no-input=true", "/dir"}, []string{"/dir"}},
		{"leaves other flags", []string{"--force", "/dir"}, []string{"--force", "/dir"}},
		{"strips only the no-input token", []string{"--no-input", "--force", "/dir"}, []string{"--force", "/dir"}},
		{"nothing to strip", []string{}, []string{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := stripNoInput(c.args)
			if len(got) != len(c.want) {
				t.Fatalf("stripNoInput(%v) = %v, want %v", c.args, got, c.want)
			}
			for i := range got {
				if got[i] != c.want[i] {
					t.Fatalf("stripNoInput(%v) = %v, want %v", c.args, got, c.want)
				}
			}
		})
	}
}

func TestUnknownSubcommand(t *testing.T) {
	// A representative registered set: leaf commands, groups (pm/findings/completion/migrate) with
	// their children, and the pbLateCommands (serve/superuser) the guard seeds explicitly.
	known := knownCommandSet{
		top: map[string]bool{
			"serve": true, "superuser": true, "migrate": true, "sweep": true,
			"query": true, "pm": true, "findings": true, "completion": true, "help": true,
		},
		groups: map[string]map[string]bool{
			"pm":         {"create": true, "show": true},
			"findings":   {"dispose": true},
			"completion": {"bash": true, "zsh": true},
			"migrate":    {"up": true, "down": true},
		},
		// pm's own persistent --actor flag: the group's OWN value-taking flags,
		// as buildKnownCommandSet would derive them from the live pm command's flag set.
		groupValueFlags: map[string]map[string]bool{
			"pm": {"--actor": true},
		},
	}
	cases := []struct {
		name     string
		args     []string
		wantName string
		wantBad  bool
	}{
		// The core regression: a bare unknown subcommand must be reported (was exit 0 before).
		{"unknown top-level command", []string{"frobnicate"}, "frobnicate", true},
		{"unknown nested pm command", []string{"pm", "frobnicate"}, "pm frobnicate", true},
		{"unknown nested findings command", []string{"findings", "frobnicate"}, "findings frobnicate", true},
		{"unknown nested completion command", []string{"completion", "frobnicate"}, "completion frobnicate", true},
		{"unknown nested migrate command", []string{"migrate", "frobnicate"}, "migrate frobnicate", true},
		{"unknown after leading global value flag", []string{"--dir", "/x", "frobnicate"}, "frobnicate", true},
		// Valid invocations must never be flagged.
		{"known leaf command", []string{"sweep"}, "", false},
		{"pocketbase-late serve", []string{"serve"}, "", false},
		{"pocketbase-late superuser", []string{"superuser"}, "", false},
		{"known leaf with an arg is not a nested lookup", []string{"query", "findings"}, "", false},
		{"known nested pm command", []string{"pm", "create"}, "", false},
		// Regression: a group's OWN persistent flag (pm's --actor), passed with its
		// value BEFORE the leaf subcommand, must not have that value mistaken for the
		// subcommand name. Before the fix this reported ("pm alice", true) — an exit-1 false
		// positive on a perfectly valid invocation.
		{"pm --actor VALUE before a known leaf", []string{"pm", "--actor", "alice", "create"}, "", false},
		{"pm --actor=VALUE (equals form) before a known leaf", []string{"pm", "--actor=alice", "create"}, "", false},
		{"pm --actor VALUE after a known leaf (already worked)", []string{"pm", "create", "--actor", "alice"}, "", false},
		// A GENUINE unknown nested command must still be caught even with --actor's value
		// correctly skipped first — the fix must not blanket-suppress real unknowns.
		{"pm --actor VALUE before a REAL unknown nested command", []string{"pm", "--actor", "alice", "frobnicate"}, "pm frobnicate", true},
		{"known findings dispose with id and flags", []string{"findings", "dispose", "abc123", "--as", "wont-fix"}, "", false},
		{"known migrate subcommand", []string{"migrate", "up"}, "", false},
		{"known completion subcommand", []string{"completion", "bash"}, "", false},
		{"group invoked bare prints its usage", []string{"pm"}, "", false},
		{"findings group invoked bare", []string{"findings"}, "", false},
		{"leading global value flag then known command", []string{"--dir", "/x", "sweep"}, "", false},
		// Help/version fast paths short-circuit before dispatch — never flagged.
		{"help long flag", []string{"--help"}, "", false},
		{"help short flag", []string{"-h"}, "", false},
		{"version long flag", []string{"--version"}, "", false},
		{"help flag alongside an unknown token", []string{"frobnicate", "--help"}, "", false},
		{"bare invocation", []string{}, "", false},
		{"only flags no subcommand", []string{"--dev"}, "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			gotName, gotBad := unknownSubcommand(c.args, known)
			if gotBad != c.wantBad || gotName != c.wantName {
				t.Errorf("unknownSubcommand(%v) = (%q, %v), want (%q, %v)", c.args, gotName, gotBad, c.wantName, c.wantBad)
			}
		})
	}
}

func TestIsStoreTouchingInvocation(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want bool
	}{
		{"store-touching subcommand", []string{"sweep"}, true},
		{"serve", []string{"serve"}, true},
		{"migrate", []string{"migrate", "up"}, true},
		{"help long flag excluded even with a store-touching subcommand present", []string{"sweep", "--help"}, false},
		{"help short flag excluded", []string{"-h"}, false},
		{"version short flag excluded", []string{"-v"}, false},
		{"version long flag excluded", []string{"--version"}, false},
		{"help before the subcommand excluded", []string{"--help", "serve"}, false},
		{"unknown command is not store-touching", []string{"frobnicate"}, false},
		{"bare invocation is not store-touching", []string{}, false},
		{"flags before the subcommand still resolve", []string{"--dir", "/x", "sweep"}, true},
		{"completion-style unknown subcommand not store-touching", []string{"completion", "bash"}, false},
		// The --hooksDir bypass, end to end: before globalValueFlags recognized these, this case
		// resolved firstSubcommand to "/some/path" and returned false here, silently skipping the
		// unresolved-location guard for a real `serve` invocation.
		{"hooksDir before serve does not bypass the guard", []string{"--hooksDir", "/some/path", "serve"}, true},
		{"hooksWatch before serve does not bypass the guard", []string{"--hooksWatch", "true", "serve"}, true},
		{"hooksPool before serve does not bypass the guard", []string{"--hooksPool", "25", "serve"}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := isStoreTouchingInvocation(c.args); got != c.want {
				t.Errorf("isStoreTouchingInvocation(%v) = %v, want %v", c.args, got, c.want)
			}
		})
	}
}

// TestGroupCommandValueFlags pins the live-cobra-introspection half of the group-value-flag fix:
// groupCommandValueFlags must find a group command's OWN persistent flag (declared via
// PersistentFlags(), the shape pm's --actor uses) as well as one declared via Flags() (a local
// flag directly on the group command itself), and must NOT report a boolean-style flag (whose
// mere presence is a complete value, NoOptDefVal set) as value-taking.
func TestGroupCommandValueFlags(t *testing.T) {
	group := &cobra.Command{Use: "widget"}
	var actor, localOnly string
	var verbose bool
	group.PersistentFlags().StringVar(&actor, "actor", "", "audit actor")
	group.Flags().StringVar(&localOnly, "local-only", "", "a flag declared directly on the group, not persistent")
	group.Flags().BoolVar(&verbose, "verbose", false, "presence alone is a complete value")

	got := groupCommandValueFlags(group)
	if !got["--actor"] {
		t.Error("groupCommandValueFlags must include the group's own PERSISTENT value-taking flag (--actor)")
	}
	if !got["--local-only"] {
		t.Error("groupCommandValueFlags must include the group's own LOCAL value-taking flag (--local-only)")
	}
	if got["--verbose"] {
		t.Error("groupCommandValueFlags must NOT include a boolean flag (--verbose): its presence alone is a complete value, never followed by a separate value token")
	}
}

// TestBuildKnownCommandSet_GroupValueFlags proves buildKnownCommandSet wires a real, registered
// group command's persistent flags into groupValueFlags automatically — no manual enumeration
// (unlike globalValueFlags) — so a future persistent flag on pm (or any other group) is picked
// up without touching this guard again.
func TestBuildKnownCommandSet_GroupValueFlags(t *testing.T) {
	root := &cobra.Command{Use: "deskkit"}
	pmGroup := &cobra.Command{Use: "pm"}
	var actor string
	pmGroup.PersistentFlags().StringVar(&actor, "actor", "operator", "audit actor")
	pmGroup.AddCommand(&cobra.Command{Use: "create"})
	root.AddCommand(pmGroup)

	ks := buildKnownCommandSet(root)
	if !ks.groupValueFlags["pm"]["--actor"] {
		t.Fatalf("buildKnownCommandSet did not capture pm's --actor as a value flag: %+v", ks.groupValueFlags)
	}
}

// TestUnknownSubcommand_PMActorBeforeLeaf is the focused --actor-before-leaf regression, isolated from
// the broader TestUnknownSubcommand table: before the fix, unknownSubcommand had no way to know
// that pm's --actor takes a value, so nextNonFlagToken(args, idx+1) saw "alice" (--actor's
// value) as the first bare token and reported it as an attempted (unknown) nested pm
// subcommand — exactly the reported symptom, `deskkit: unknown command "pm alice"`, printed by
// main() right before os.Exit(1) (see the guard call site in main()).
func TestUnknownSubcommand_PMActorBeforeLeaf(t *testing.T) {
	known := knownCommandSet{
		top:    map[string]bool{"pm": true},
		groups: map[string]map[string]bool{"pm": {"create": true}},
		groupValueFlags: map[string]map[string]bool{
			"pm": {"--actor": true},
		},
	}
	name, bad := unknownSubcommand([]string{"pm", "--actor", "alice", "create", "--title", "x", "--type", "task"}, known)
	if bad {
		t.Fatalf("pm --actor alice create ... (flag before the leaf) flagged as unknown: %q", name)
	}
}

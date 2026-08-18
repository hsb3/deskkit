// Command deskkit is the single Go binary: it serves PocketBase, runs the agent loop under
// `serve`, and exposes the tool core as CLI subcommands. This spine wires pocketbase.New(),
// migratecmd (automigrate), the blank-imported migrations, first-run seeding (ignore boundary +
// system prompt) on serve, and the Cobra subcommands routed through the tools seam. main builds
// the enabled module set (librarian always on; pm on unless PM_ENABLED=false or profile
// modules.pm.enabled: false) via module.Register, which merges each module's tools into the
// shared toolcore registry before any surface builds.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/mattn/go-isatty"
	"github.com/pocketbase/pocketbase"
	// pbcmd is aliased: `cmd` is the parameter name of every RunE closure in this file.
	pbcmd "github.com/pocketbase/pocketbase/cmd"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/hsb3/deskkit/internal/core/config"
	"github.com/hsb3/deskkit/internal/core/mcp"
	"github.com/hsb3/deskkit/internal/core/migrate"
	"github.com/hsb3/deskkit/internal/core/module"
	"github.com/hsb3/deskkit/internal/core/spa"
	"github.com/hsb3/deskkit/internal/core/store"
	"github.com/hsb3/deskkit/internal/modules/librarian/agent"
	"github.com/hsb3/deskkit/internal/modules/librarian/desklib"
	"github.com/hsb3/deskkit/internal/modules/librarian/prompt"
	"github.com/hsb3/deskkit/internal/modules/librarian/setup"
	"github.com/hsb3/deskkit/internal/modules/librarian/tools"
	"github.com/hsb3/deskkit/internal/modules/librarian/trigger"
	"github.com/hsb3/deskkit/internal/modules/librarian/tui"
	"github.com/hsb3/deskkit/internal/modules/librarian/web"

	"github.com/hsb3/deskkit/internal/modules/librarian"
	"github.com/hsb3/deskkit/internal/modules/pm"
	"github.com/hsb3/deskkit/internal/modules/profile"

	// Blank-import registers the librarian's Go migrations. The core-owned
	// module_schema_versions meta migration self-registers from internal/core/migrate's init(),
	// which the normal import above already runs.
	_ "github.com/hsb3/deskkit/internal/modules/librarian/collections"
)

// version is stamped at build time via ldflags from the repo-root VERSION file. A bare
// `go build` leaves "dev", so `deskkit --version` is truthful only for make/release builds.
var version = "dev"

// moduleReg holds the enabled module set, populated once in main() by module.Register before the
// app is constructed. OnServe and requireConfig both read it.
var moduleReg *module.Registry

func main() {
	// `init` scaffolds a desk profile with filesystem writes ONLY, and must never trigger
	// PocketBase's Bootstrap, which MkdirAll-creates a store data dir. Bootstrap runs for every
	// KNOWN registered command (skipBootstrap only spares --help/--version/unknown), so a
	// registered `init` reaching app.Start() would materialize a stray pb_data — handle it here,
	// before the app exists. `init --help`/`-h` deliberately fall through to cobra instead: the
	// help path skips Bootstrap and prints the registered command's usage.
	if isInitInvocation(os.Args[1:]) {
		os.Exit(runInit(os.Args[1:]))
	}

	// `desks` and `config` are read-only orientation commands that open NO store, intercepted for
	// the same reason `init` is: a normally-dispatched `desks` would still reach Bootstrap and
	// MkdirAll a stray store dir just for asking which desks exist. Both stay registered on
	// app.RootCmd below for discovery and `--help`, whose path skips Bootstrap.
	if isInterceptedInvocation(os.Args[1:], "desks") {
		os.Exit(runIntercepted(newDesksCmd(), os.Args[1:]))
	}
	if isInterceptedInvocation(os.Args[1:], "config") {
		os.Exit(runIntercepted(newConfigCmd(), os.Args[1:]))
	}

	// Config resolves BEFORE the app is constructed: the store location derives from DESK_NAME and
	// seeds PocketBase's --dir default. serve/migrate still run without full config, but the store
	// LOCATION must resolve — via DESK_NAME (the XDG default) or an explicit --dir, enforced just
	// below. Tool subcommands additionally require full config (requireConfig).
	cfg, cfgErr := config.Load()

	// --dir explicitly overrides the XDG default (both `--dir <path>` and `--dir=<path>`); when
	// present the unresolved-location guard below does not apply.
	explicitDir := hasDirFlag(os.Args[1:])
	// `pm` is store-touching only when the pm module is enabled: with PM off the group is not even
	// registered, so guarding (and pre-creating a store dir for) an unknown command would be wrong.
	storeTouching := isStoreTouchingInvocation(os.Args[1:]) ||
		(firstSubcommand(os.Args[1:]) == "pm" && cfg != nil && cfg.PMEnabled)
	// --no-input is read manually here because the first-run onramp is decided in main() BEFORE
	// cobra parses flags. Non-TTY or --no-input means fail closed.
	noInput := hasNoInputFlag(os.Args[1:])

	// Absent --dir, the store defaults to $XDG_DATA_HOME/deskkit/<DESK_NAME>/ (falling back to
	// ~/.local/share/...), replacing PocketBase's cwd/exe-relative pb_data.
	var defaultDataDir string
	var locErr error
	if !explicitDir {
		if dir, derr := resolveStoreLocation(cfg, cfgErr); derr != nil {
			locErr = derr
		} else {
			defaultDataDir = dir
			// Pre-create the store dir 0700 so it is not group/world-readable — PocketBase's own
			// bootstrap would otherwise MkdirAll it 0777. Only when actually about to open the
			// store: a non-store command (e.g. --help) must not materialize a data dir.
			// The legacy-store auto-migration runs FIRST: it is a no-op unless the new home is
			// absent, so it must look before this MkdirAll materializes that home.
			if storeTouching {
				if mgErr := store.MigrateLegacyStoreDir(cfg.DeskName); mgErr != nil {
					locErr = mgErr
				} else if mkErr := os.MkdirAll(dir, 0o700); mkErr != nil {
					locErr = fmt.Errorf("cannot create the store directory %s: %w; pass --dir <path>", dir, mkErr)
				}
			}
		}
	}

	// Fail closed on an unresolvable store LOCATION — no silent fallback to a cwd/exe-relative
	// pb_data. Enforced HERE, before app.Start()/Bootstrap, because PocketBase opens (and
	// MkdirAll-creates) the data dir during Bootstrap, earlier than any cobra PersistentPreRunE,
	// so a post-Bootstrap guard would already have written the stray dir. Gated to real
	// store-touching subcommands so it never breaks --help, `completion`, or usage output.
	if locErr != nil && storeTouching {
		// First-run onramp (interactive only): when the location is unresolvable purely because
		// config is missing (cfgErr, not a StoreDir/mkdir failure), offer to scaffold this folder as
		// a desk. It lives HERE, not in requireConfig, because the guard os.Exit's before cobra ever
		// dispatches the store-touching command — a prompt in requireConfig would be unreachable for
		// exactly the commands that hit this guard. On accept, the cwd is init'd and config
		// re-resolved so the original command proceeds in the same process; on decline / non-TTY /
		// --no-input, FirstRunDecision emits nothing and the fail-closed error below stands.
		if cfgErr != nil {
			tty := isatty.IsTerminal(os.Stdin.Fd()) && isatty.IsTerminal(os.Stdout.Fd())
			if accept, derr := setup.FirstRunDecision(os.Stdin, os.Stdout, tty, noInput); derr == nil && accept {
				wd, _ := os.Getwd()
				res, ierr := setup.InitProfile(wd, setup.InitOptions{}, nil)
				if ierr != nil {
					fmt.Fprintf(os.Stderr, "deskkit: %v\n", ierr)
					os.Exit(1)
				}
				res.WriteSummary(os.Stdout)
				// Recompute + pre-create the store location at 0700, matching the guard above.
				if newCfg, nerr := config.Load(); nerr == nil {
					cfg, cfgErr = newCfg, nil
					if dir, serr := resolveStoreLocation(cfg, nil); serr != nil {
						locErr = serr
					} else if mgErr := store.MigrateLegacyStoreDir(cfg.DeskName); mgErr != nil {
						locErr = mgErr
					} else if mkErr := os.MkdirAll(dir, 0o700); mkErr != nil {
						locErr = fmt.Errorf("cannot create the store directory %s: %w; pass --dir <path>", dir, mkErr)
					} else {
						defaultDataDir, locErr = dir, nil
					}
				}
			}
		}
		if locErr != nil {
			fmt.Fprintf(os.Stderr, "deskkit: %v\n", locErr)
			os.Exit(1)
		}
	}

	// DefaultDataDir seeds the --dir flag default; empty falls back to PocketBase's exe-relative
	// pb_data, reached only when --dir is passed (and overrides it) or for non-store commands.
	//
	// DefaultDev is deliberately always false, NOT pocketbase.New()'s own IsProbablyGoRun()
	// heuristic: that fires whenever os.Args[0] sits under os.TempDir() or the Go build cache, so
	// any release binary staged in a temp dir silently lands in dev mode, whose SQL debug lines go
	// to STDOUT and corrupt mcp-serve's JSON-RPC stream and every tool command's JSON output.
	// `--dev` stays a real root flag, so interactive development still opts in explicitly.
	//
	// The enabled module set is built here so each module's tools merge into the shared toolcore
	// registry BEFORE any surface (agent/mcp) builds and before the migration runner executes.
	// Safe with a nil cfg: profile and librarian ignore it, pm treats nil as off. A registration
	// error is a collection-ownership collision — a build-time bug — so it is fatal.
	reg, regErr := module.Register(cfg, profile.New(), librarian.New(), pm.New())
	if regErr != nil {
		log.Fatalf("deskkit: module registration: %v", regErr)
	}
	moduleReg = reg

	app := pocketbase.NewWithConfig(pocketbase.Config{
		DefaultDev:     false,
		DefaultDataDir: defaultDataDir,
	})

	// Override PocketBase's own RootCmd.Version so `deskkit --version` reports THIS binary.
	app.RootCmd.Version = version

	// Registered so it appears in --help and cobra accepts it; the onramp decision in main() reads
	// it manually from os.Args, because that decision fires before cobra parses flags.
	app.RootCmd.PersistentFlags().Bool("no-input", false, "never prompt; fail closed when config is unresolved (scripts/CI)")

	// Automigrate on startup; also `deskkit migrate up`. migrate is schema-only and deliberately
	// skips the desk open-guard: it writes no desk rows, so running it against another desk's
	// store is harmless.
	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		Automigrate: true,
	})

	// On serve start, after bootstrap/migrations: ensure the ignore boundary exists and seed the
	// system prompt on first run. These run only under serve.
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		// Exposure mode is derived from EVERY resolved bind address, not just the one the serve event
		// reports: with --https set, that address is the HTTPS one and hides a separate --http
		// listener. A loopback bind keeps the local UX; any exposed address puts the whole process in
		// public mode (auth prerequisites, route auth, same-origin CSRF). Derived from the exposure
		// rather than a --public flag, because a forgotten flag on an 0.0.0.0 bind fails OPEN.
		// --origins rides the same read so hardenPublicCORS can tell an allowlist from the default
		// wildcard.
		serveAddrs, serveOrigins := serveAddrsAndOrigins(app.RootCmd, e.Server.Addr)
		publicMode := isPublicBind(serveAddrs...)

		// A self-contradictory --origins (bare "*" mixed with explicit origins) is fatal on a public
		// bind rather than silently resolved either way. Checked here, with the same os.Exit
		// fail-closed shape as the auth gate below, so nothing ever binds.
		if err := ValidatePublicOrigins(publicMode, serveOrigins); err != nil {
			fmt.Fprintf(os.Stderr, "deskkit: %v\n", err)
			os.Exit(1)
		}

		// Fail-closed auth prerequisites, FIRST: this hook runs before the listener opens, so exiting
		// here means nothing was ever bound. os.Exit, never `return err` — serve is a system command
		// registered inside app.Start(), which runs RootCmd.Execute() in a goroutine and DISCARDS its
		// error, so a returned RunE error prints but still exits 0, and a refusal that exits 0 is one
		// nothing downstream can detect. In public mode this call also PROVISIONS the PB_SUPERUSER_*
		// account and re-verifies one exists, so a rejected password fails the boot.
		superuserCreated, err := store.CheckServeAuthPrereqs(e.App, cfg, publicMode)
		if err != nil {
			fmt.Fprintf(os.Stderr, "deskkit: %v\n", err)
			os.Exit(1)
		}
		if superuserCreated {
			app.Logger().Info("created superuser from PB_SUPERUSER_* env", "email", cfg.PBSuperuserEmail)
		}

		// Drop the dependency's default wildcard CORS middleware on a public bind, unless the operator
		// supplied an --origins allowlist, which the dependency binds into that very same middleware.
		// Router-wide, so it covers /api/* too; a no-op on a loopback bind. Must happen BEFORE
		// e.Next(), which builds the mux from the router's current middleware set.
		hardenPublicCORS(e.Router, publicMode, serveOrigins)

		// Embedded SPA at `/` (admin console stays /_/, data API /api/). Registered outside the cfgErr
		// gate below: the static shell and the loopback token bootstrap need only the store, not a
		// resolved desk config. In public mode the bootstrap route is not registered at all and the
		// shell still serves, so the login form loads while every data call behind it stays
		// token-gated — the admin console's shell-loads-data-doesn't stance.
		spa.Register(e.Router, publicMode)

		if cfgErr == nil {
			// Desk open-guard: refuse to serve a store that already belongs to a different desk.
			// gui re-execs `serve`, so it is covered here too. Print + os.Exit, never `return err`,
			// for the reason given at the auth-prereq gate above: a serve RunE error still exits 0.
			if err := store.CheckDeskGuard(e.App, cfg.DeskName); err != nil {
				fmt.Fprintf(os.Stderr, "deskkit: %v\n", err)
				os.Exit(1)
			}
			// Migrations have already run by OnServe, so GuardDowngrade refuses to serve a store a
			// newer binary already migrated ahead, and StampModules records each enabled module's
			// applied version by observation. Stamping is non-fatal bookkeeping.
			if err := migrate.GuardDowngrade(e.App, moduleReg.MigrateModules()); err != nil {
				fmt.Fprintf(os.Stderr, "deskkit: %v\n", err)
				os.Exit(1)
			}
			if err := migrate.StampModules(e.App, moduleReg.MigrateModules()); err != nil {
				app.Logger().Error("stamp module schema versions", "err", err)
			}
			if err := desklib.EnsureIgnoreFile(cfg.IgnoreConfig, cfg.DeskRoot); err != nil {
				app.Logger().Error("ensure .librarian-ignore", "err", err)
			}
		} else {
			app.Logger().Warn("config not fully resolved; skipping .librarian-ignore auto-create", "err", cfgErr)
		}
		if err := prompt.Seed(e.App); err != nil {
			app.Logger().Error("seed system prompt", "err", err)
		}
		// First-run superuser auto-create: only when both PB_SUPERUSER_* env vars are set;
		// idempotent and non-fatal — a failure logs but never blocks serve.
		if cfgErr == nil {
			if created, err := store.EnsureSuperuser(e.App, cfg); err != nil {
				app.Logger().Error("ensure superuser", "err", err)
			} else if created {
				app.Logger().Info("created superuser from PB_SUPERUSER_* env", "email", cfg.PBSuperuserEmail)
			}
		}

		// Wake layer (spec §2.4) — registered ONLY under serve, so one-shot tool commands
		// (which also create records) never enqueue tasks. Event hooks + cron enqueue tasks
		// rows; a single background claimer polls the queue at CLAIMER_POLL_INTERVAL and runs
		// each task (deterministic kinds call the tool directly; agentic kinds run the loop
		// via the injected action below). Config is required for the tools, so gate on cfgErr.
		if cfgErr == nil {
			// Each enabled module wires its own serve-time record hooks + cron via RegisterHooks.
			// StartClaimer stays in main because it needs agentAction: moving it into the module would
			// pull the agent package into trigger.
			for _, mod := range moduleReg.Enabled {
				if err := mod.RegisterHooks(e.App, cfg); err != nil {
					app.Logger().Error("register module hooks", "module", mod.Name(), "err", err)
				}
				// Realtime is serve-only: wiring each module's optional RealtimeSource here is what keeps
				// one-shot CLI commands from emitting events.
				if rs, ok := mod.(module.RealtimeSource); ok {
					if err := rs.RegisterRealtime(e.App); err != nil {
						app.Logger().Error("register module realtime", "module", mod.Name(), "err", err)
					}
				}
			}
			trigger.StartClaimer(context.Background(), e.App, cfg, agentAction)

			// Browser session surface: routes driving the SAME multi-turn session the `chat` REPL
			// exposes — same *agent.Session, same gated tool slice, same write boundary. Serve-only and
			// gated on cfgErr, because the session factory needs resolved config. A loopback bind stays
			// unauthenticated by design (on-demand, single operator); on a non-loopback bind
			// web.Register puts every route behind RequireAuth and switches to strict same-origin.
			webCleanup := web.Register(e.Router, func(ctx context.Context) (web.Streamer, error) {
				s, serr := agent.NewSession(ctx, app, cfg)
				if serr != nil {
					return nil, serr // return a nil interface on error, never a typed-nil session
				}
				return s, nil
			}, publicMode)
			e.App.OnTerminate().BindFunc(func(te *core.TerminateEvent) error {
				webCleanup(context.Background())
				return te.Next()
			})
		} else {
			app.Logger().Warn("config not resolved; wake layer (hooks/cron/claimer) not started", "err", cfgErr)
		}
		return e.Next()
	})

	registerToolCommands(app, cfg, cfgErr)

	// PocketBase's Execute() runs RootCmd.Execute() in a goroutine and discards its error, so a
	// failing RunE would exit 0. Wrap every registered subcommand's RunE — RECURSIVELY, since the
	// `pm` group nests a level down — to record the first error and exit non-zero after Start()
	// returns, post-cleanup.
	var cmdErr error
	var wrapRunE func(cmds []*cobra.Command)
	wrapRunE = func(cmds []*cobra.Command) {
		for _, c := range cmds {
			if f := c.RunE; f != nil {
				c.RunE = func(cmd *cobra.Command, args []string) error {
					err := f(cmd, args)
					if err != nil {
						err = annotateLockErr(err)
						if cmdErr == nil {
							cmdErr = err
						}
					}
					return err
				}
			}
			wrapRunE(c.Commands())
		}
	}
	wrapRunE(app.RootCmd.Commands())

	// serve/superuser + the display groups. Registering the two PocketBase system commands here
	// (what app.Start() would do) is the only way they can carry a GroupID: Start() adds them
	// after the last point a caller could set one. Deliberately AFTER wrapRunE, so their RunE
	// stays unwrapped and their error handling matches Start()'s.
	finalizeCommandTree(app)

	// Unknown-subcommand guard. Because Execute() discards RootCmd.Execute()'s error, cobra's own
	// "unknown command" error would exit 0. Detect an unrecognized (nested) subcommand HERE —
	// after every registration has populated RootCmd and BEFORE app.Execute() dispatches — and
	// exit non-zero directly, since a returned RunE error is invisible for a goroutine-run
	// Execute.
	if name, unknown := unknownSubcommand(os.Args[1:], buildKnownCommandSet(app.RootCmd)); unknown {
		fmt.Fprintf(os.Stderr, "deskkit: unknown command %q\nRun 'deskkit --help' for usage.\n", name)
		os.Exit(1)
	}

	// app.Execute(), not app.Start(): Start() is exactly "add superuser + serve, then Execute()",
	// and finalizeCommandTree has already added those two (with their groups). Calling Start()
	// here would register them a second time.
	if err := app.Execute(); err != nil {
		log.Fatal(annotateLockErr(err))
	}
	if cmdErr != nil {
		os.Exit(1)
	}
}

// storeTouchingCommands are the subcommands that open the desk's store and therefore require a
// resolvable store location. Maintained explicitly rather than derived from RootCmd, because
// this guard runs in main() BEFORE the app (and its command tree) exists. `init`, `desks`, and
// `config` are deliberately absent: they open no store.
var storeTouchingCommands = map[string]bool{
	"serve": true, "migrate": true, "superuser": true,
	"sweep": true, "patrol": true, "propose-fix": true, "apply-fix": true,
	"restore": true, "query": true, "record-feedback": true, "agent": true, "chat": true,
	"mcp-serve": true, "gui": true, "findings": true,
}

// Command display groups (`deskkit --help`). The IDs are internal; the Titles are printed
// literally by cobra, trailing colon included.
const (
	groupSetup   = "setup"
	groupInspect = "inspect"
	groupFix     = "fix"
	groupWork    = "work"
	groupAgent   = "agent"
	groupAdmin   = "admin"
)

// commandGroups is the help's section order: cobra renders groups in AddGroup call order, so
// this slice IS the display order.
var commandGroups = []cobra.Group{
	{ID: groupSetup, Title: "Setup & config:"},
	{ID: groupInspect, Title: "Inspect:"},
	{ID: groupFix, Title: "Fix:"},
	{ID: groupWork, Title: "Work graph:"},
	{ID: groupAgent, Title: "Agent:"},
	{ID: groupAdmin, Title: "Admin:"},
}

// commandGroupIDs maps a top-level command name to its display group. A command missing from
// this map renders under cobra's "Additional Commands:" catch-all, which the groups test fails
// on, so adding a subcommand means adding it here. help/completion are cobra built-ins added
// during Execute(); they get their group from the two setters in applyCommandGroups.
var commandGroupIDs = map[string]string{
	"init":   groupSetup,
	"config": groupSetup,

	"desks":           groupInspect,
	"sweep":           groupInspect,
	"patrol":          groupInspect,
	"query":           groupInspect,
	"findings":        groupInspect,
	"record-feedback": groupInspect,
	"gui":             groupInspect,

	"propose-fix": groupFix,
	"apply-fix":   groupFix,
	"restore":     groupFix,

	"pm": groupWork,

	"agent":     groupAgent,
	"chat":      groupAgent,
	"mcp-serve": groupAgent,

	"serve":     groupAdmin,
	"migrate":   groupAdmin,
	"superuser": groupAdmin,
}

// finalizeCommandTree completes the root command tree: it registers the two system commands
// PocketBase would otherwise add inside Start() (so they can carry a group), then applies the
// display groups. Called once from main() after every other registration, and by the groups
// test, so the test asserts the tree main() actually builds. NewServeCommand's second argument
// is Start()'s `!hideStartBanner`.
func finalizeCommandTree(app *pocketbase.PocketBase) {
	app.RootCmd.AddCommand(pbcmd.NewSuperuserCommand(app))
	app.RootCmd.AddCommand(pbcmd.NewServeCommand(app, true))
	applyCommandGroups(app.RootCmd)
}

// applyCommandGroups assigns each registered top-level command its display group and registers
// only the groups that ended up with a member — an empty group renders as a bare heading, and
// `pm` is absent when the module is gated off. cobra validates the result at Execute() time:
// checkCommandGroups panics on an id that was never registered.
func applyCommandGroups(root *cobra.Command) {
	used := map[string]bool{groupAdmin: true} // help/completion always land in Admin
	for _, c := range root.Commands() {
		if id, ok := commandGroupIDs[c.Name()]; ok {
			c.GroupID = id
			used[id] = true
		}
	}
	for _, g := range commandGroups {
		if used[g.ID] && !root.ContainsGroup(g.ID) {
			root.AddGroup(&cobra.Group{ID: g.ID, Title: g.Title})
		}
	}
	// PocketBase suppresses both of cobra's built-ins, so neither is LISTED — but the hidden help
	// command is still a child with an empty GroupID, which alone makes cobra render an empty
	// "Additional Commands:" heading. SetHelpCommandGroupID groups that existing command, which is
	// what removes the heading. The completion setter is the same arrangement for the day
	// PocketBase stops disabling that command.
	root.SetHelpCommandGroupID(groupAdmin)
	root.SetCompletionCommandGroupID(groupAdmin)
}

// resolveStoreLocation answers WHERE this desk's store lives, with no side effects: no mkdir, no
// legacy-store migration, no PocketBase. main() pairs it with those side effects on the startup
// path; the read-only `desks` and `config show` surfaces call it for the location alone. One
// function so the two can never disagree about the answer.
func resolveStoreLocation(cfg *config.Config, cfgErr error) (string, error) {
	if cfgErr != nil {
		return "", fmt.Errorf(
			"cannot resolve the store location: %w; set DESK_NAME (env or _knowledge/profile.*) or pass --dir <path>",
			cfgErr)
	}
	dir, err := store.StoreDir(cfg.DeskName)
	if err != nil {
		return "", fmt.Errorf("cannot resolve the store location: %w; pass --dir <path>", err)
	}
	return dir, nil
}

// hasDirFlag reports whether an explicit --dir override is present (both `--dir <path>` and
// `--dir=<path>`) — the signal that the XDG default is bypassed.
func hasDirFlag(args []string) bool {
	for _, a := range args {
		if a == "--dir" || strings.HasPrefix(a, "--dir=") {
			return true
		}
	}
	return false
}

// isStoreTouchingInvocation reports whether os.Args names a subcommand that opens the store, so
// the unresolved-location guard fires there but never on --help, --version, the bare usage
// output, or an unknown command (which cobra reports itself).
func isStoreTouchingInvocation(args []string) bool {
	// Help/version short-circuit inside cobra before any command body runs; never guard them.
	for _, a := range args {
		switch a {
		case "-h", "--help", "-v", "--version":
			return false
		}
	}
	return storeTouchingCommands[firstSubcommand(args)]
}

// knownCommandSet is the recognized-command lookup the unknown-subcommand guard consults.
// top is every top-level command + alias name; groups maps each command that HAS subcommands to
// its child (+alias) names, so a nested unknown like `pm frobnicate` is caught too.
// groupValueFlags maps that same group name to the GROUP COMMAND'S OWN value-taking flags.
type knownCommandSet struct {
	top             map[string]bool
	groups          map[string]map[string]bool
	groupValueFlags map[string]map[string]bool
}

// buildKnownCommandSet snapshots app.RootCmd — by then FULLY populated, since
// finalizeCommandTree registers serve/superuser itself rather than leaving them to app.Start()
// — into the lookup the guard needs. Every command is read from root.Commands(), so nothing is
// hardcoded here.
func buildKnownCommandSet(root *cobra.Command) knownCommandSet {
	ks := knownCommandSet{
		top: map[string]bool{}, groups: map[string]map[string]bool{},
		groupValueFlags: map[string]map[string]bool{},
	}
	for _, c := range root.Commands() {
		names := append([]string{c.Name()}, c.Aliases...)
		for _, n := range names {
			ks.top[n] = true
		}
		if c.HasSubCommands() {
			children := map[string]bool{}
			for _, sub := range c.Commands() {
				children[sub.Name()] = true
				for _, a := range sub.Aliases {
					children[a] = true
				}
			}
			// groupCommandValueFlags needs no enumerated whitelist (unlike globalValueFlags below):
			// by the time this runs the group command's own flag definitions exist and are consulted
			// directly, so a future persistent flag on any group is picked up automatically.
			vf := groupCommandValueFlags(c)
			for _, n := range names {
				ks.groups[n] = children
				ks.groupValueFlags[n] = vf
			}
		}
	}
	return ks
}

// groupCommandValueFlags returns the "--name" form of every flag REGISTERED DIRECTLY ON c that
// requires an explicit value, mirroring cobra's own hasNoOptDefVal check: an unset NoOptDefVal
// means the very next token is the flag's value, never a subcommand name.
//
// It visits c.Flags() AND c.PersistentFlags() separately on purpose. This runs BEFORE
// RootCmd.Execute(), and cobra merges persistent flags into Flags() only lazily during Execute,
// so at this call site c.Flags() alone does NOT contain a persistent flag declared on c. The
// double-visit is harmless once a merge has happened — do not "simplify" it to c.Flags() alone.
func groupCommandValueFlags(c *cobra.Command) map[string]bool {
	vf := map[string]bool{}
	collect := func(fs *pflag.FlagSet) {
		fs.VisitAll(func(f *pflag.Flag) {
			if f.NoOptDefVal == "" {
				vf["--"+f.Name] = true
			}
		})
	}
	collect(c.Flags())
	collect(c.PersistentFlags())
	return vf
}

// nextNonFlagToken returns the first non-flag token in args at/after start (a group's nested
// subcommand name), or ("", false) when only flags/nothing remain. valueFlags is that group's
// OWN value-taking flags: a recognized "--flag value" pair (space form, not "--flag=value")
// skips the value token too, so a group persistent flag passed BEFORE the leaf subcommand is
// never mistaken for the subcommand name. A flag this group does NOT know about still shadows
// the token — see globalValueFlags for why an enumerated fallback cannot close that fully.
func nextNonFlagToken(args []string, start int, valueFlags map[string]bool) (string, bool) {
	for i := start; i < len(args); i++ {
		a := args[i]
		if strings.HasPrefix(a, "-") {
			if !strings.Contains(a, "=") && valueFlags[a] {
				i++ // the next token is this flag's value, not a subcommand name
			}
			continue
		}
		return a, true
	}
	return "", false
}

// unknownSubcommand reports the offending token and true when args name a command (or, under a
// known command GROUP, a nested subcommand) that is not registered — the case the process must
// exit non-zero on, since PocketBase discards cobra's own unknown-command error. It returns
// ("", false) for the bare invocation, any -h/--help/-v/--version request, a known leaf command,
// a known group invoked bare, and a known group + known child.
func unknownSubcommand(args []string, known knownCommandSet) (string, bool) {
	for _, a := range args {
		switch a {
		case "-h", "--help", "-v", "--version":
			return "", false
		}
	}
	idx := subcommandIndex(args)
	if idx < 0 {
		return "", false // bare invocation -> cobra prints root usage (exit 0)
	}
	top := args[idx]
	if !known.top[top] {
		return top, true
	}
	children, isGroup := known.groups[top]
	if !isGroup {
		return "", false // known leaf command; its own args/flags are cobra's business
	}
	sub, ok := nextNonFlagToken(args, idx+1, known.groupValueFlags[top])
	if !ok {
		return "", false // group invoked bare -> cobra prints the group usage (exit 0)
	}
	if !children[sub] {
		return top + " " + sub, true
	}
	return "", false
}

// globalValueFlags are every value-taking flag this manual pre-parse must recognize so its value
// token is never mistaken for the subcommand name. Get this wrong and `--someflag /path serve`
// resolves the subcommand to "/path", which is not store-touching, so the location guard is
// skipped and PocketBase's Bootstrap MkdirAll-creates a stray exe-relative data dir before cobra
// ever reports the bad flag.
//
// --dir/--encryptionEnv/--queryTimeout are the root persistent flags eagerParseFlags registers
// on RootCmd in this build — re-audit that function on every dependency bump; migratecmd
// registers none. --hooksDir/--hooksWatch/--hooksPool are NOT registered (the jsvm plugin that
// adds them is not imported) but are listed defensively: this scan runs before app construction
// and cannot consult cobra's flag definitions, so an unregistered flag is exactly as dangerous
// here as a registered one. It stays an enumerated whitelist, so a novel value-flag not listed
// can still shadow the subcommand token.
var globalValueFlags = map[string]bool{
	"--dir": true, "--encryptionEnv": true, "--queryTimeout": true,
	"--hooksDir": true, "--hooksWatch": true, "--hooksPool": true,
}

// subcommandIndex returns the index in args of the invoked subcommand token, skipping any
// leading global flags and the values of the value-taking ones so a form like `--dir /x serve`
// resolves to the "serve" index. Returns -1 for the bare (no-subcommand) usage.
func subcommandIndex(args []string) int {
	for i := 0; i < len(args); i++ {
		a := args[i]
		if strings.HasPrefix(a, "-") {
			if !strings.Contains(a, "=") && globalValueFlags[a] {
				i++ // its value is the next token, not the subcommand name
			}
			continue
		}
		return i
	}
	return -1
}

// firstSubcommand returns the first non-flag token in args (the invoked subcommand name),
// skipping any leading global flags and the values of the value-taking ones so a form like
// `--dir /x serve` still resolves to "serve". Returns "" for the bare (no-subcommand) usage.
func firstSubcommand(args []string) string {
	if i := subcommandIndex(args); i >= 0 {
		return args[i]
	}
	return ""
}

// hasNoInputFlag reports whether the root --no-input flag is present (both `--no-input` and
// `--no-input=<bool>`). Read manually because the first-run onramp is decided before cobra
// parses flags.
func hasNoInputFlag(args []string) bool {
	for _, a := range args {
		if a == "--no-input" || strings.HasPrefix(a, "--no-input=") {
			return true
		}
	}
	return false
}

// isInterceptedInvocation reports whether os.Args invokes `name` for REAL execution — the
// signal to run it standalone in main(), before PocketBase's Bootstrap can create a store dir.
// `name --help` / `-h` deliberately returns false: those fall through to cobra, whose help path
// skips Bootstrap and prints the registered command's usage.
func isInterceptedInvocation(args []string, name string) bool {
	for _, a := range args {
		switch a {
		case "-h", "--help":
			return false
		}
	}
	return firstSubcommand(args) == name
}

// isInitInvocation reports whether os.Args invokes `init` for real execution (not `init --help`
// / `-h`, which fall through to cobra's Bootstrap-skipping help path).
func isInitInvocation(args []string) bool { return isInterceptedInvocation(args, "init") }

// runIntercepted executes one command standalone BEFORE the PocketBase app exists, so it never
// triggers Bootstrap (no stray store dir). It returns the process exit code. The trailing args
// (the command's own flags/subcommand/operands) are everything after the command's own token;
// the global --no-input token is stripped since it belongs to the root, not the subcommand.
func runIntercepted(cmd *cobra.Command, args []string) int {
	idx := subcommandIndex(args)
	var tail []string
	if idx >= 0 {
		tail = args[idx+1:]
	}
	cmd.SetArgs(stripNoInput(tail))
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "deskkit: %v\n", err)
		return 1
	}
	return 0
}

// runInit executes `init` standalone (see runIntercepted) and returns the process exit code.
func runInit(args []string) int { return runIntercepted(newInitCmd(), args) }

// stripNoInput removes the root persistent --no-input token from a subcommand's args: the
// standalone init command below does not define it and would otherwise reject it as unknown.
func stripNoInput(args []string) []string {
	out := make([]string, 0, len(args))
	for _, a := range args {
		if a == "--no-input" || strings.HasPrefix(a, "--no-input=") {
			continue
		}
		out = append(out, a)
	}
	return out
}

// newInitCmd builds the `init` subcommand: it scaffolds <dir>/_knowledge/profile.yaml (and, with
// --with-env, <dir>/.env) so a folder works as a desk with zero exports. Registered on the app
// RootCmd for discovery/`init --help`, executed standalone by runInit for the real run.
// SilenceErrors/Usage keep the standalone execution's error reporting to runInit's single line.
func newInitCmd() *cobra.Command {
	var force, withEnv bool
	cmd := &cobra.Command{
		Use:           "init [dir]",
		Short:         "Scaffold the minimal profile so a folder works as a desk (zero exports)",
		Args:          cobra.MaximumNArgs(1),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := "."
			if len(args) == 1 {
				dir = args[0]
			}
			confirm := func(deskName, ancestorPath string) (bool, error) {
				tty := isatty.IsTerminal(os.Stdin.Fd()) && isatty.IsTerminal(os.Stdout.Fd())
				return setup.ConfirmNested(os.Stdin, os.Stdout, tty, deskName, ancestorPath)
			}
			res, err := setup.InitProfile(dir, setup.InitOptions{Force: force, WithEnv: withEnv}, confirm)
			if err != nil {
				return err
			}
			res.WriteSummary(os.Stdout)
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "overwrite an existing profile/.env and allow a nested desk")
	cmd.Flags().BoolVar(&withEnv, "with-env", false, "also write a .env stub naming the LLM API-key env var")
	return cmd
}

func requireConfig(app core.App, cfg *config.Config, cfgErr error) (*config.Config, error) {
	if cfgErr != nil {
		return nil, cfgErr
	}
	// Self-initialize the store: only serve and `migrate up` trigger a migration run, so a one-shot
	// tool command against a never-initialized store would find no collections and leak
	// sql.ErrNoRows. Applied before the desk-guard, so that guard consults now-existing (empty)
	// collections and a first run passes by construction. Cheap once current.
	if err := app.RunAppMigrations(); err != nil {
		return nil, fmt.Errorf("initialize store schema: %w", err)
	}
	// Module-schema versioning at the one-shot-command entry point too: guard against a store a
	// newer binary migrated ahead, then stamp each enabled module's applied version by observation.
	// Stamping is non-fatal bookkeeping.
	if err := migrate.GuardDowngrade(app, moduleReg.MigrateModules()); err != nil {
		return nil, err
	}
	if err := migrate.StampModules(app, moduleReg.MigrateModules()); err != nil {
		app.Logger().Error("stamp module schema versions", "err", err)
	}
	// Desk open-guard: refuse if this store already belongs to another desk. Checked before any
	// seeding/write so a mismatched desk never mutates the wrong store; the choke point every tool
	// + agent/chat/mcp-serve/gui RunE reaches first.
	if err := store.CheckDeskGuard(app, cfg.DeskName); err != nil {
		return nil, err
	}
	// First-run auto-creation applies to every entry point, not just serve: one-shot CLI tools
	// would otherwise fail closed on the missing default ignore file. A present-but-unreadable
	// file still fails closed in desklib.
	if err := desklib.EnsureIgnoreFile(cfg.IgnoreConfig, cfg.DeskRoot); err != nil {
		return nil, err
	}
	// Seed the editable system prompt here too, not only under serve: a desk driven purely via
	// one-shot CLI + MCP would never materialize the prompts row, leaving "edit the system prompt
	// in the DB" a silent no-op. Not a safety gate — the agent falls back to the embedded default
	// — so a seed failure logs but never blocks the command.
	if err := prompt.Seed(app); err != nil {
		app.Logger().Error("seed system prompt", "err", err)
	}
	return cfg, nil
}

// annotateLockErr adds a hint to a SQLite "database is locked" failure, which is what a store
// open hits when another process — typically a concurrent `serve` — already holds the desk's
// store. Detection is a case-insensitive substring match rather than a typed sqlite error,
// because the underlying error crosses several wrapping layers before reaching here.
func annotateLockErr(err error) error {
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "locked") {
		return err
	}
	return fmt.Errorf("%w; is another deskkit process (e.g. `serve`) already running against this desk?", err)
}

// registerToolCommands wires the librarian tool subcommands + gui onto the PocketBase RootCmd.
// serve, migrate, and superuser come from PocketBase / migratecmd. Each tool command routes
// through the same tools.* function the agent calls.
func registerToolCommands(app *pocketbase.PocketBase, cfg *config.Config, cfgErr error) {
	// init — scaffold a desk profile. Registered for discovery and `init --help`; the REAL run is
	// intercepted in main() before app.Start() so it never triggers Bootstrap. NOT in
	// storeTouchingCommands: it opens no store.
	app.RootCmd.AddCommand(newInitCmd())

	// desks / config — read-only orientation over the XDG homes. Registered for discovery and
	// `--help`; like `init`, the REAL run is intercepted in main() so neither materializes a store
	// just for answering a question. Neither is store-touching.
	app.RootCmd.AddCommand(newDesksCmd())
	app.RootCmd.AddCommand(newConfigCmd())

	// pm — the work-graph command group, registered ONLY when the pm module is enabled; with PM
	// off, `deskkit pm` is cobra's normal unknown-command error.
	if cfg != nil && cfg.PMEnabled {
		registerPMCommands(app, cfg, cfgErr)
	}

	// sweep
	app.RootCmd.AddCommand(&cobra.Command{
		Use:   "sweep",
		Short: "Reindex the desk tree into the files collection",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := requireConfig(app, cfg, cfgErr)
			if err != nil {
				return err
			}
			res, serr := tools.Sweep(cmd.Context(), app, c, &tools.SweepInput{})
			return printJSON(cmd.OutOrStdout(), res, serr)
		},
	})

	// patrol
	var patrolPath string
	patrolCmd := &cobra.Command{
		Use:   "patrol",
		Short: "Dry-run: file rule findings (R1–R6) + one log row; NO fs writes",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := requireConfig(app, cfg, cfgErr)
			if err != nil {
				return err
			}
			res, serr := tools.Patrol(cmd.Context(), app, c, &tools.PatrolInput{Path: patrolPath})
			return printJSON(cmd.OutOrStdout(), res, serr)
		},
	}
	patrolCmd.Flags().StringVar(&patrolPath, "path", "", "restrict patrol to this file/subtree")
	app.RootCmd.AddCommand(patrolCmd)

	// propose-fix
	var proposeRun string
	var proposeRules []string
	proposeCmd := &cobra.Command{
		Use:   "propose-fix",
		Short: "Plan mechanical fixes and record originals to revisions; NO fs writes",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := requireConfig(app, cfg, cfgErr)
			if err != nil {
				return err
			}
			res, serr := tools.ProposeFix(cmd.Context(), app, c, &tools.ProposeFixInput{RunID: proposeRun, Rules: proposeRules})
			return printJSON(cmd.OutOrStdout(), res, serr)
		},
	}
	proposeCmd.Flags().StringVar(&proposeRun, "run", "", "scope to a patrol run id")
	proposeCmd.Flags().StringSliceVar(&proposeRules, "rules", nil, "rule filter (default R1,R2,R3)")
	app.RootCmd.AddCommand(proposeCmd)

	// apply-fix (supervised commit; never a Makefile/CI default target)
	var applyRun string
	var applyRevs []string
	applyCmd := &cobra.Command{
		Use:   "apply-fix",
		Short: "Commit recorded revisions byte-exact (supervised; writes desk files)",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := requireConfig(app, cfg, cfgErr)
			if err != nil {
				return err
			}
			res, serr := tools.ApplyFix(cmd.Context(), app, c, &tools.ApplyFixInput{RunID: applyRun, RevisionIDs: applyRevs})
			return printJSON(cmd.OutOrStdout(), res, serr)
		},
	}
	applyCmd.Flags().StringVar(&applyRun, "run", "", "apply this run's recorded, un-applied revisions")
	applyCmd.Flags().StringSliceVar(&applyRevs, "revision-ids", nil, "explicit revision ids to apply")
	app.RootCmd.AddCommand(applyCmd)

	// restore
	var restoreRev, restorePath string
	restoreCmd := &cobra.Command{
		Use:   "restore",
		Short: "Reverse a change to the exact recorded original",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := requireConfig(app, cfg, cfgErr)
			if err != nil {
				return err
			}
			res, serr := tools.Restore(cmd.Context(), app, c, &tools.RestoreInput{RevisionID: restoreRev, Path: restorePath})
			return printJSON(cmd.OutOrStdout(), res, serr)
		},
	}
	restoreCmd.Flags().StringVar(&restoreRev, "revision", "", "the revisions row id to reverse")
	restoreCmd.Flags().StringVar(&restorePath, "by-path", "", "resolve to the latest applied, unrestored revision for this path")
	app.RootCmd.AddCommand(restoreCmd)

	// query <kind>
	var queryDays int
	var queryPretty bool
	var queryIncludeDisposed bool
	var queryTerm string
	var queryLimit int
	var queryPath string
	var queryShowIndex bool
	queryCmd := &cobra.Command{
		Use:   "query <kind>",
		Short: "Read-only queries: live_files recent orphans uncollapsed findings summary adoption feedback search content",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := requireConfig(app, cfg, cfgErr)
			if err != nil {
				return err
			}
			raw, qerr := tools.Query(cmd.Context(), app, c, &tools.QueryInput{
				Kind: args[0], Days: queryDays, IncludeDisposed: queryIncludeDisposed,
				Term: queryTerm, Limit: queryLimit, Path: queryPath, ShowIndex: queryShowIndex,
			})
			if qerr != nil {
				return qerr
			}
			// --pretty renders an aligned table for the human-supervised workflow; raw JSON
			// (the agent/scripting contract) stays the default and is the fallback for any kind
			// the renderer does not format.
			if queryPretty {
				if out, ok := prettyQuery(args[0], raw); ok {
					fmt.Println(out)
					return nil
				}
			}
			fmt.Println(string(raw))
			return nil
		},
	}
	queryCmd.Flags().IntVar(&queryDays, "days", 7, "window for 'recent'")
	queryCmd.Flags().BoolVar(&queryPretty, "pretty", false, "render an aligned table instead of raw JSON (human-supervised workflow)")
	// --include-disposed widens the `findings` query from the live-only default (open findings)
	// to include acknowledged/triaged/wont_fix history; no effect on other query kinds.
	queryCmd.Flags().BoolVar(&queryIncludeDisposed, "include-disposed", false, "for `query findings`: also show disposed (acknowledged/triaged/wont-fix) findings, not just open ones")
	// --term/--limit drive `query search` (substring retrieval over indexed content); --path drives
	// `query content` (fetch one file's stored body). Inert for the other kinds.
	queryCmd.Flags().StringVar(&queryTerm, "term", "", "for `query search`: substring to find in indexed file content")
	queryCmd.Flags().IntVar(&queryLimit, "limit", 20, "for `query search`: max results (hard-capped at 200)")
	queryCmd.Flags().StringVar(&queryPath, "path", "", "for `query content`: desk-relative file path whose stored body to return")
	// --show-index opts the by-design-unreferenced index/entry files (CLAUDE.md, READMEs, INDEX.md)
	// back into `query orphans`; they are hidden by default as noise (an entry/index doc is what
	// other docs point at, so nothing references it). No effect on other query kinds.
	queryCmd.Flags().BoolVar(&queryShowIndex, "show-index", false, "for `query orphans`: also show by-design-unreferenced index/entry files (CLAUDE.md, READMEs, INDEX.md)")
	app.RootCmd.AddCommand(queryCmd)

	// findings — supervised disposition lifecycle for patrol findings. A finding's disposition
	// (open|acknowledged|triaged|wont_fix) is ORTHOGONAL to its state, so a live-only
	// `query findings` stops surfacing a disposed item while it survives re-baseline: patrol
	// dedupes on (file,rule,checksum) and inherits a prior non-open disposition, with its
	// provenance, onto a re-created finding. Disposition is owner-supervised, so it is a CLI
	// subcommand and deliberately NOT an MCP tool, like restore. tools.DisposeFinding normalizes
	// and validates the value. Optional --by/--reason record who/why; there is no default actor,
	// and moving back to `open` clears any recorded provenance.
	findingsCmd := &cobra.Command{
		Use:   "findings",
		Short: "Manage patrol findings (disposition lifecycle)",
	}
	var disposeAs, disposeBy, disposeReason string
	disposeCmd := &cobra.Command{
		Use:   "dispose <finding-id>",
		Short: "Set a finding's disposition: open, acknowledged, triaged, or wont-fix",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := requireConfig(app, cfg, cfgErr)
			if err != nil {
				return err
			}
			res, serr := tools.DisposeFinding(cmd.Context(), app, c, args[0], disposeAs, disposeBy, disposeReason)
			return printJSON(cmd.OutOrStdout(), res, serr)
		},
	}
	disposeCmd.Flags().StringVar(&disposeAs, "as", "", "disposition to set: open, acknowledged, triaged, or wont-fix (required)")
	_ = disposeCmd.MarkFlagRequired("as")
	disposeCmd.Flags().StringVar(&disposeBy, "by", "", "who is disposing this finding (optional; recorded as provenance)")
	disposeCmd.Flags().StringVar(&disposeReason, "reason", "", "why (optional; recommended when --as wont-fix)")
	findingsCmd.AddCommand(disposeCmd)
	app.RootCmd.AddCommand(findingsCmd)

	// record-feedback — write one entry to the store-native feedback log, through the same
	// tools.RecordFeedback the agent/chat/MCP surfaces call. DB-only write: unlike apply-fix it
	// touches no desk file, so it carries no write gate.
	var fbKind, fbSummary, fbDetail, fbContext, fbSource string
	recordFeedbackCmd := &cobra.Command{
		Use:   "record-feedback",
		Short: "Record a problem or feedback entry to the store's feedback log",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := requireConfig(app, cfg, cfgErr)
			if err != nil {
				return err
			}
			res, serr := tools.RecordFeedback(cmd.Context(), app, c, &tools.RecordFeedbackInput{
				Kind:    fbKind,
				Summary: fbSummary,
				Detail:  fbDetail,
				Context: fbContext,
				Source:  fbSource,
			})
			return printJSON(cmd.OutOrStdout(), res, serr)
		},
	}
	recordFeedbackCmd.Flags().StringVar(&fbKind, "kind", "", "entry type: problem or feedback")
	recordFeedbackCmd.Flags().StringVar(&fbSummary, "summary", "", "one-line summary")
	recordFeedbackCmd.Flags().StringVar(&fbDetail, "detail", "", "optional longer detail")
	recordFeedbackCmd.Flags().StringVar(&fbContext, "context", "", "optional note on what the agent was doing")
	recordFeedbackCmd.Flags().StringVar(&fbSource, "source", "agent", "who originated the entry: agent or user")
	// Declared required at the cobra layer so --help marks them and a missing flag fails fast
	// with cobra's standard error (the tool re-validates values either way).
	_ = recordFeedbackCmd.MarkFlagRequired("kind")
	_ = recordFeedbackCmd.MarkFlagRequired("summary")
	app.RootCmd.AddCommand(recordFeedbackCmd)

	// agent <instruction> — the MANUAL trigger for the ReAct loop (agent_runs.trigger="manual").
	// A one-shot separate process, like sweep/patrol.
	agentCmd := &cobra.Command{
		Use:   "agent <instruction>",
		Short: "Run the agent loop once on an instruction (manual trigger)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := requireConfig(app, cfg, cfgErr)
			if err != nil {
				return err
			}
			final, err := agent.Run(cmd.Context(), app, c, "manual", args[0])
			if err != nil {
				return err
			}
			if final != "" {
				fmt.Println(final)
			}
			return nil
		},
	}
	app.RootCmd.AddCommand(agentCmd)

	// chat — interactive multi-turn stewardship session over the agent loop. On a real terminal it
	// runs the full-screen TUI; runChat below is the line-oriented REPL fallback for piped stdio or
	// --plain. Like `agent`, it self-initializes the store via requireConfig, so no prior
	// `migrate up` or `serve` is required. The session inherits the gated tool set (restore never
	// exposed; apply_fix only under LIBRARIAN_AUTONOMOUS_WRITES) and the data-backed system prompt.
	chatCmd := &cobra.Command{
		Use:   "chat",
		Short: "Interactive multi-turn librarian session (full-screen TUI on a terminal; REPL when piped or --plain)",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := requireConfig(app, cfg, cfgErr)
			if err != nil {
				return err
			}
			plain, _ := cmd.Flags().GetBool("plain")
			// Route to the line REPL when --plain is set or either stdio end is not a terminal: the
			// full-screen TUI requires an interactive TTY on both stdin AND stdout. requireConfig has
			// already run above, so nothing here draws before config is known good.
			if plain || !isatty.IsTerminal(os.Stdin.Fd()) || !isatty.IsTerminal(os.Stdout.Fd()) {
				return runChat(cmd.Context(), app, c)
			}
			// Resolve the color theme ONCE here, in the safe window before the Bubble Tea program starts:
			// the "auto" path probes the terminal background, and that query corrupts input if it fires
			// after the program's stdin reader is running. Precedence: --theme > LIBRARIAN_THEME >
			// auto-detect, consulting the flag only when set so an unset flag defers to the env override.
			themeFlag := ""
			if cmd.Flags().Changed("theme") {
				// GetString only errors on an unregistered/wrong-type flag; "theme" is registered as a
				// string below, so the error is structurally impossible here — discard it explicitly.
				themeFlag, _ = cmd.Flags().GetString("theme") //nolint:errcheck // "theme" is a registered string flag
			}
			// Module TUI views are collected lazily against the LIVE app/config: the pm views when that
			// module is enabled, none otherwise.
			return tui.Run(cmd.Context(), app, c, tui.ResolveTheme(themeFlag), moduleReg.TUIViews(app, c))
		},
	}
	chatCmd.Flags().Bool("plain", false, "force the line-oriented REPL instead of the full-screen TUI")
	chatCmd.Flags().String("theme", "auto", "color theme for the full-screen TUI: light, dark, or auto (detect the terminal background once at startup)")
	app.RootCmd.AddCommand(chatCmd)

	// mcp-serve — expose the tool core as an MCP stdio server. The model-facing tool set carries
	// the SAME registration-time write gate as the agent loop (apply_fix only when
	// LIBRARIAN_AUTONOMOUS_WRITES=true), and restore is NEVER exposed: it stays supervised CLI-only.
	// It holds the DB open for the whole session, so it MUST NOT run concurrently with `serve`
	// (single-writer SQLite). The stdio server returns when the client closes stdin.
	app.RootCmd.AddCommand(&cobra.Command{
		Use:   "mcp-serve",
		Short: "Expose the librarian tool core as an MCP stdio server (model-facing; the exposed tool set is gated per §5.4, distinct from the fuller CLI subcommand set)",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := requireConfig(app, cfg, cfgErr)
			if err != nil {
				return err
			}
			return mcp.Serve(cmd.Context(), app, c)
		},
	})

	// gui — convenience: open the admin GUI then serve (spawns `serve` as a child so it
	// does not depend on PocketBase serve-command internals; single-writer rule holds).
	app.RootCmd.AddCommand(&cobra.Command{
		Use:   "gui",
		Short: "Open the visual data browser (admin console) in a browser",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Abort BEFORE opening a browser or spawning the serve child when config or the desk guard
			// fails: otherwise a mismatched-desk store pops a browser tab and re-execs `serve`, which
			// hits its own OnServe guard exit only after the browser opened against a store gui had no
			// business touching. requireConfig runs the same desk-guard check as every other subcommand.
			c, err := requireConfig(app, cfg, cfgErr)
			if err != nil {
				return err
			}
			openBrowser(strings.TrimRight(c.PBURL, "/") + "/_/")
			bin, xerr := os.Executable()
			if xerr != nil {
				return xerr
			}
			// Forward the resolved --dir to the child so it opens the EXACT SAME store gui just
			// desk-guarded. --dir is a process flag, not an env var, so unlike
			// DESK_ROOT/DESK_NAME/XDG_DATA_HOME (which inherit via exec.Command's nil Env) it does NOT
			// propagate on its own. GetString("dir") reads cobra's resolved value — the explicit --dir
			// when passed, else the XDG default main() handed to NewWithConfig — so forwarding it
			// unconditionally keeps parent and child aimed at the same store either way.
			dir, derr := cmd.Flags().GetString("dir")
			if derr != nil {
				return derr
			}
			childArgs := append([]string{"serve", "--dir", dir}, args...)
			child := exec.Command(bin, childArgs...)
			child.Stdout, child.Stderr, child.Stdin = os.Stdout, os.Stderr, os.Stdin
			return child.Run()
		},
	})
}

// agentAction adapts agent.Run to trigger.AgentAction (the claimer's agentic dispatch): a
// query/custom task runs the eino loop and the final text is discarded (the transcript is
// persisted to the messages/agent_runs collections either way). Keeping this in the command
// layer is what lets internal/trigger stay free of an internal/agent import.
func agentAction(ctx context.Context, app core.App, cfg *config.Config, trig, input string) error {
	_, err := agent.Run(ctx, app, cfg, trig, input)
	return err
}

// runChat drives a line-oriented REPL over one agent.Session. Each input line is one turn; the
// session's growing history keeps the exchange multi-turn. Exit with "exit", "quit", or EOF.
// The write boundary is the Session's, inherited from the gated tool set — nothing here opens a
// new capability.
func runChat(ctx context.Context, app core.App, cfg *config.Config) error {
	sess, err := agent.NewSession(ctx, app, cfg)
	if err != nil {
		return err
	}
	defer func() { _ = sess.Close(ctx) }()

	fmt.Printf("deskkit session for desk %q — type a request; 'exit', 'quit', or Ctrl-D to end.\n", cfg.DeskName)
	sc := bufio.NewScanner(os.Stdin)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024) // allow long pasted requests
	for {
		fmt.Print("\nlibrarian> ")
		if !sc.Scan() {
			break // EOF (Ctrl-D) or read error (checked below)
		}
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		if line == "exit" || line == "quit" {
			break
		}
		reply, terr := sess.Turn(ctx, line)
		if terr != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", terr)
			continue
		}
		fmt.Println(reply)
	}
	fmt.Println("session ended.")
	return sc.Err()
}

// printJSON marshals a tool's typed result (or returns its error) to w. Every cobra caller
// passes cmd.OutOrStdout() so a test can redirect a leaf command's output via cmd.SetOut/
// RootCmd.SetOut without touching the process-global os.Stdout (cmd.OutOrStdout() on a
// subcommand walks up to the root's configured writer, defaulting to os.Stdout only when
// nothing was set).
func printJSON[T any](w io.Writer, res T, err error) error {
	if err != nil {
		return err
	}
	b, merr := json.MarshalIndent(res, "", "  ")
	if merr != nil {
		return merr
	}
	fmt.Fprintln(w, string(b))
	return nil
}

func openBrowser(url string) {
	var c *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		c = exec.Command("open", url)
	case "windows":
		c = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		c = exec.Command("xdg-open", url)
	}
	_ = c.Start()
}

// Command pocket-librarian is the single Go binary that serves PocketBase, runs the agent
// loop under `serve` (later slice), and exposes the six tools as CLI subcommands. This
// spine wires: pocketbase.New(), migratecmd (automigrate), the blank-imported migrations,
// first-run seeding (ignore boundary + system prompt) on serve, and the Cobra subcommands
// routed through the tools seam. Tool bodies + the eino loop + MCP are later slices.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/mattn/go-isatty"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
	"github.com/pocketbase/pocketbase/tools/osutils"
	"github.com/spf13/cobra"

	"github.com/example/pocket-librarian/internal/agent"
	"github.com/example/pocket-librarian/internal/bootstrap"
	"github.com/example/pocket-librarian/internal/config"
	"github.com/example/pocket-librarian/internal/desklib"
	"github.com/example/pocket-librarian/internal/mcp"
	"github.com/example/pocket-librarian/internal/prompt"
	"github.com/example/pocket-librarian/internal/setup"
	"github.com/example/pocket-librarian/internal/tools"
	"github.com/example/pocket-librarian/internal/trigger"
	"github.com/example/pocket-librarian/internal/tui"

	// Blank-import registers all Go migrations (spec §4.11).
	_ "github.com/example/pocket-librarian/migrations"
)

// version is stamped at build time via ldflags (-X main.version=<VERSION>), wired from the
// repo-root VERSION file by the librarian Makefile and the release workflow. A bare `go build`
// leaves the default "dev", so `pocket-librarian --version` prints the real version only for
// make/release builds.
var version = "dev"

func main() {
	// `init` scaffolds a desk profile with filesystem writes ONLY: it needs neither resolved
	// config nor PocketBase, and — critically — must never trigger PocketBase's Bootstrap, which
	// MkdirAll-creates a store data dir. Bootstrap runs for every KNOWN registered command
	// (skipBootstrap only spares --help/--version/unknown), so a registered `init` reaching
	// app.Start() would still materialize a stray pb_data. Handle it HERE, before the app exists,
	// and exit. `init --help` / `init -h` are deliberately NOT matched (isInitInvocation returns
	// false): they fall through to cobra, whose help path skips Bootstrap and prints the
	// registered command's usage (init is also registered in registerToolCommands for discovery).
	if isInitInvocation(os.Args[1:]) {
		os.Exit(runInit(os.Args[1:]))
	}

	// Config is resolved BEFORE the app is constructed: the store location is derived from
	// DESK_NAME and seeds PocketBase's --dir default (ADR 0002 §2). config.Load has no
	// PocketBase dependency, so this reorder is safe. serve/migrate (schema ops) may still run
	// even before DESK_ROOT/DESK_NAME are fully set, but the store LOCATION must be resolvable —
	// via DESK_NAME (→ the XDG default) or an explicit --dir (enforced just below). The tool
	// subcommands additionally require full config (requireConfig).
	cfg, cfgErr := config.Load()

	// --dir is the explicit store override that wins over the XDG default (both `--dir <path>`
	// and `--dir=<path>` forms). When present, the location is whatever the operator chose and
	// the unresolved-location guard does not apply.
	explicitDir := hasDirFlag(os.Args[1:])
	storeTouching := isStoreTouchingInvocation(os.Args[1:])
	// --no-input (root persistent flag; also registered on app.RootCmd below for help/usage) is
	// read manually here because the first-run onramp is decided in main() BEFORE cobra parses
	// flags. Non-TTY or --no-input => today's fail-closed behavior, byte-identical error.
	noInput := hasNoInputFlag(os.Args[1:])

	// Absent --dir, the store defaults to $XDG_DATA_HOME/pocket-librarian/<DESK_NAME>/ (falling
	// back to ~/.local/share/...), replacing PocketBase's cwd/exe-relative pb_data (ADR 0002 §2).
	var defaultDataDir string
	var locErr error
	if !explicitDir {
		if cfgErr != nil {
			locErr = fmt.Errorf(
				"cannot resolve the store location: %w; set DESK_NAME (env or _knowledge/profile.*) or pass --dir <path>",
				cfgErr)
		} else if dir, derr := config.StoreDir(cfg.DeskName); derr != nil {
			locErr = fmt.Errorf("cannot resolve the store location: %w; pass --dir <path>", derr)
		} else {
			defaultDataDir = dir
			// Pre-create the store dir 0700 so it is not group/world-readable — PocketBase's own
			// bootstrap would otherwise MkdirAll it 0777. Only when actually about to open the
			// store: a non-store command (e.g. --help) must not materialize a data dir.
			if storeTouching {
				if mkErr := os.MkdirAll(dir, 0o700); mkErr != nil {
					locErr = fmt.Errorf("cannot create the store directory %s: %w; pass --dir <path>", dir, mkErr)
				}
			}
		}
	}

	// Fail closed on an unresolvable store LOCATION. This narrows the old "serve/migrate run
	// config-free" tolerance: they still run without full config, but the store LOCATION must
	// now resolve (ADR 0002 §2) — no silent fallback to a cwd/exe-relative pb_data. Enforced
	// HERE, before app.Start()/Bootstrap, because PocketBase opens (and MkdirAll-creates) the
	// data dir during Bootstrap — earlier than any cobra PersistentPreRunE — so a post-Bootstrap
	// guard would already have written the stray dir. Gated to real store-touching subcommands
	// so it never breaks --help, `completion`, the bare usage output, or unknown-command errors.
	if locErr != nil && storeTouching {
		// First-run onramp (interactive only): when the store LOCATION is unresolvable purely
		// because config is missing (cfgErr, not a StoreDir/mkdir failure), offer to scaffold this
		// folder as a desk instead of only erroring. This lives HERE, not in requireConfig, because
		// this guard os.Exit's before cobra ever dispatches the store-touching command — so a prompt
		// hooked in requireConfig would be unreachable for exactly the commands that hit this guard.
		// On accept we init the cwd, re-resolve config, recompute the store location, and fall
		// through so the original command proceeds seamlessly in this same process. On decline /
		// non-TTY / --no-input, FirstRunDecision emits nothing and the fail-closed error below is
		// byte-identical to before.
		if cfgErr != nil {
			tty := isatty.IsTerminal(os.Stdin.Fd()) && isatty.IsTerminal(os.Stdout.Fd())
			if accept, derr := setup.FirstRunDecision(os.Stdin, os.Stdout, tty, noInput); derr == nil && accept {
				wd, _ := os.Getwd()
				res, ierr := setup.InitProfile(wd, setup.InitOptions{}, nil)
				if ierr != nil {
					fmt.Fprintf(os.Stderr, "pocket-librarian: %v\n", ierr)
					os.Exit(1)
				}
				res.WriteSummary(os.Stdout)
				// Re-resolve config now that the profile exists; recompute + pre-create the store
				// location (0700, matching the guard above) so the requested command can continue.
				if newCfg, nerr := config.Load(); nerr == nil {
					cfg, cfgErr = newCfg, nil
					if dir, serr := config.StoreDir(cfg.DeskName); serr != nil {
						locErr = fmt.Errorf("cannot resolve the store location: %w; pass --dir <path>", serr)
					} else if mkErr := os.MkdirAll(dir, 0o700); mkErr != nil {
						locErr = fmt.Errorf("cannot create the store directory %s: %w; pass --dir <path>", dir, mkErr)
					} else {
						defaultDataDir, locErr = dir, nil
					}
				}
			}
		}
		if locErr != nil {
			fmt.Fprintf(os.Stderr, "pocket-librarian: %v\n", locErr)
			os.Exit(1)
		}
	}

	// DefaultDataDir seeds the --dir flag default (override-if---dir-passed-else-XDG); empty
	// falls back to PocketBase's exe-relative pb_data, reached only when --dir is passed (and
	// overrides it) or for non-store commands. DefaultDev mirrors pocketbase.New()'s go-run
	// detection so `go run` still defaults to dev mode.
	app := pocketbase.NewWithConfig(pocketbase.Config{
		DefaultDev:     osutils.IsProbablyGoRun(),
		DefaultDataDir: defaultDataDir,
	})

	// Override PocketBase's own RootCmd.Version (defaults to its embedded "(untracked)") with the
	// ldflags-stamped repo version, so `pocket-librarian --version` reports THIS binary's release.
	app.RootCmd.Version = version

	// --no-input suppresses the first-run onramp prompt (fail closed on unresolved config, for
	// scripts/CI). Registered here so it appears in --help and cobra accepts it; the onramp
	// decision in main() reads it manually from os.Args (it fires before cobra parses flags).
	app.RootCmd.PersistentFlags().Bool("no-input", false, "never prompt; fail closed when config is unresolved (scripts/CI)")

	// Automigrate on startup; also `pocket-librarian migrate up`. See §11.3 open item 1:
	// confirm the automigrate generated-migration behavior in the run environment. migrate is
	// schema-only and deliberately skips the desk open-guard (ADR 0002 §3): it writes no desk
	// rows, so running it against another desk's store is harmless.
	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		Automigrate: true,
	})

	// On serve start (after bootstrap/migrations): ensure the ignore boundary exists and
	// seed the system prompt on first run (spec §10.1, §6.1). These run only under serve.
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		if cfgErr == nil {
			// Desk open-guard (ADR 0002 §3): refuse to serve a store that already belongs to a
			// different desk. gui re-execs `serve`, so it is covered here too.
			//
			// Print + os.Exit rather than `return err`: serve/superuser are PocketBase system
			// commands registered inside app.Start(), which runs RootCmd.Execute() in a
			// goroutine and discards its error (upstream: "leave to the commands to decide
			// whether to print their error") — so a returned RunE error here would print via
			// PocketBase's own error writer but the process would still exit 0. Every other
			// guarded surface (requireConfig's cmdErr wrapper; the unresolved-location guard's
			// direct os.Exit above) fails closed with a non-zero exit; mirror that here since
			// normal RunE-error propagation is invisible for this command.
			if err := bootstrap.CheckDeskGuard(e.App, cfg.DeskName); err != nil {
				fmt.Fprintf(os.Stderr, "pocket-librarian: %v\n", err)
				os.Exit(1)
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
		// First-run superuser auto-create (spec §10.3): only when both PB_SUPERUSER_* env
		// vars are set; idempotent and non-fatal — a failure logs but never blocks serve.
		if cfgErr == nil {
			if created, err := bootstrap.EnsureSuperuser(e.App, cfg); err != nil {
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
			if err := trigger.RegisterHooks(e.App, cfg); err != nil {
				app.Logger().Error("register record hooks", "err", err)
			}
			if err := trigger.RegisterCron(e.App, cfg); err != nil {
				app.Logger().Error("register cron", "err", err)
			}
			trigger.StartClaimer(context.Background(), e.App, cfg, agentAction)
		} else {
			app.Logger().Warn("config not resolved; wake layer (hooks/cron/claimer) not started", "err", cfgErr)
		}
		return e.Next()
	})

	registerToolCommands(app, cfg, cfgErr)

	// PocketBase's Execute() runs RootCmd.Execute() in a goroutine and discards its
	// error (upstream: "leave to the commands to decide whether to print their error"),
	// so a failing RunE would exit 0. Wrap every registered subcommand's RunE to record
	// the first error, and exit non-zero after Start() returns (post-cleanup).
	var cmdErr error
	for _, c := range app.RootCmd.Commands() {
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
	}

	if err := app.Start(); err != nil {
		log.Fatal(annotateLockErr(err))
	}
	if cmdErr != nil {
		os.Exit(1)
	}
}

// storeTouchingCommands are the subcommands that open the desk's store and therefore require a
// resolvable store location (ADR 0002 §2). serve/superuser are registered by PocketBase inside
// Start(), migrate by migratecmd, the tool commands by registerToolCommands — so the set is
// maintained here explicitly rather than derived from RootCmd (not fully populated pre-Start).
var storeTouchingCommands = map[string]bool{
	"serve": true, "migrate": true, "superuser": true,
	"sweep": true, "patrol": true, "propose-fix": true, "apply-fix": true,
	"restore": true, "query": true, "record-feedback": true, "agent": true, "chat": true,
	"mcp-serve": true, "gui": true,
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

// globalValueFlags are every value-taking flag this manual pre-parse must recognize so its
// value token is never mistaken for the subcommand name (the specific bug this list closes:
// `--hooksDir /some/path serve` previously resolved firstSubcommand to "/some/path", which is
// not in storeTouchingCommands, so isStoreTouchingInvocation silently returned false and
// PocketBase's Bootstrap — which runs and MkdirAll-creates the (wrong, exe-relative) data dir
// BEFORE cobra even reaches the "unknown flag" parse error for `serve` — proceeded unguarded).
//
// --dir/--encryptionEnv/--queryTimeout are pocketbase.go's actual registered root persistent
// flags in THIS build (verified against the vendored source: eagerParseFlags in
// github.com/pocketbase/pocketbase/pocketbase.go registers them on RootCmd — re-audit that
// function on every dependency bump; migratecmd registers none).
// --hooksDir/--hooksWatch/--hooksPool are NOT currently registered (the jsvm plugin that adds
// them is not imported here) but are added defensively: they are real PocketBase-ecosystem
// root flags a future dependency bump could wire in, and — as the bug above shows — an
// UNREGISTERED flag is exactly as dangerous to this pre-parse as a registered one, since this
// scan runs before app construction and cannot consult cobra's own flag definitions. This
// remains an enumerated whitelist, not a structural fix: a genuinely novel unrecognized
// value-flag not on this list could still shadow the subcommand token the same way.
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

// isInitInvocation reports whether os.Args invokes `init` for real execution (not `init --help`
// / `-h`, which fall through to cobra's Bootstrap-skipping help path).
func isInitInvocation(args []string) bool {
	for _, a := range args {
		switch a {
		case "-h", "--help":
			return false
		}
	}
	return firstSubcommand(args) == "init"
}

// runInit executes `init` as a standalone cobra command BEFORE the PocketBase app exists, so it
// never triggers Bootstrap (no stray store dir). It returns the process exit code. The trailing
// args (init's own flags + the optional dir) are everything after the `init` token; the global
// --no-input token is stripped since it belongs to the root, not init.
func runInit(args []string) int {
	idx := subcommandIndex(args)
	var tail []string
	if idx >= 0 {
		tail = args[idx+1:]
	}
	cmd := newInitCmd()
	cmd.SetArgs(stripNoInput(tail))
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "pocket-librarian: %v\n", err)
		return 1
	}
	return 0
}

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
// --with-env, <dir>/.env) so a folder works as a desk with zero exports. It is registered on the
// app RootCmd for discovery/`init --help`, and executed standalone by runInit for the real run.
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
	// Self-initialize the store (ADR 0003): one-shot tool commands do NOT trigger a migration
	// run on their own (only serve and `migrate up` do), so a command against a never-initialized
	// store would otherwise find no collections and leak sql.ErrNoRows. Apply pending app
	// migrations idempotently here — before the desk-guard, so it consults now-existing (empty)
	// collections and a first run passes by construction. Cheap (a _migrations check) once current.
	if err := app.RunAppMigrations(); err != nil {
		return nil, fmt.Errorf("initialize store schema: %w", err)
	}
	// Desk open-guard (ADR 0002 §3): refuse if this store already belongs to another desk.
	// Checked before any seeding/write so a mismatched desk never mutates the wrong store; the
	// choke point every tool + agent/chat/mcp-serve/gui RunE reaches first.
	if err := bootstrap.CheckDeskGuard(app, cfg.DeskName); err != nil {
		return nil, err
	}
	// "First run" auto-creation (spec §10.1) applies to every entry point, not just
	// serve: one-shot CLI tools would otherwise fail closed on the missing default
	// ignore file. A present-but-unreadable file still fails closed in desklib.
	if err := desklib.EnsureIgnoreFile(cfg.IgnoreConfig, cfg.DeskRoot); err != nil {
		return nil, err
	}
	// Seed the editable system prompt on first run here too, not only under serve (spec
	// §4.10): a desk driven purely via one-shot CLI + MCP would otherwise never materialize
	// the prompts row, leaving "edit the system prompt in the DB" a silent no-op. Unlike the
	// ignore boundary this is not a safety gate — the agent falls back to the embedded
	// default — so a seed failure logs but never blocks the command (mirrors the serve hook).
	if err := prompt.Seed(app); err != nil {
		app.Logger().Error("seed system prompt", "err", err)
	}
	return cfg, nil
}

// annotateLockErr adds a short hint to a SQLite "database is locked" failure — the shape
// requireConfig's store-open/migration step (and PocketBase's own Bootstrap, reached from
// app.Start() before any RunE) hits when another process, typically a concurrently running
// `serve`, already holds the desk's store open. Detection is a simple case-insensitive
// substring match on "locked" rather than a typed sqlite error, since the underlying error
// crosses several wrapping layers (dbx, mattn/go-sqlite3) before reaching here; a non-lock
// error is returned unchanged.
func annotateLockErr(err error) error {
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "locked") {
		return err
	}
	return fmt.Errorf("%w; is another pocket-librarian process (e.g. `serve`) already running against this desk?", err)
}

// registerToolCommands wires the six tool subcommands + gui onto the PocketBase RootCmd.
// serve, migrate, and superuser are provided by PocketBase / migratecmd. Each tool command
// routes through the same tools.* function the agent will call (spec §2.6, §3.3). Until the
// tool-body slice lands these return ErrNotImplemented — expected for the spine.
func registerToolCommands(app *pocketbase.PocketBase, cfg *config.Config, cfgErr error) {
	// init — scaffold a desk profile (zero exports). Registered for discovery (`--help` listing)
	// and `init --help`; the REAL run is intercepted in main() before app.Start() so it never
	// triggers Bootstrap (see runInit). NOT added to storeTouchingCommands: it opens no store.
	app.RootCmd.AddCommand(newInitCmd())

	// sweep
	app.RootCmd.AddCommand(&cobra.Command{
		Use:   "sweep",
		Short: "Reindex the desk tree into the files collection",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := requireConfig(app, cfg, cfgErr)
			if err != nil {
				return err
			}
			return printJSON(tools.Sweep(cmd.Context(), app, c, &tools.SweepInput{}))
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
			return printJSON(tools.Patrol(cmd.Context(), app, c, &tools.PatrolInput{Path: patrolPath}))
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
			return printJSON(tools.ProposeFix(cmd.Context(), app, c, &tools.ProposeFixInput{RunID: proposeRun, Rules: proposeRules}))
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
			return printJSON(tools.ApplyFix(cmd.Context(), app, c, &tools.ApplyFixInput{RunID: applyRun, RevisionIDs: applyRevs}))
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
			return printJSON(tools.Restore(cmd.Context(), app, c, &tools.RestoreInput{RevisionID: restoreRev, Path: restorePath}))
		},
	}
	restoreCmd.Flags().StringVar(&restoreRev, "revision", "", "the revisions row id to reverse")
	restoreCmd.Flags().StringVar(&restorePath, "by-path", "", "resolve to the latest applied, unrestored revision for this path")
	app.RootCmd.AddCommand(restoreCmd)

	// query <kind>
	var queryDays int
	var queryPretty bool
	queryCmd := &cobra.Command{
		Use:   "query <kind>",
		Short: "Read-only queries: live_files recent orphans uncollapsed findings summary adoption feedback",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := requireConfig(app, cfg, cfgErr)
			if err != nil {
				return err
			}
			raw, qerr := tools.Query(cmd.Context(), app, c, &tools.QueryInput{Kind: args[0], Days: queryDays})
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
	app.RootCmd.AddCommand(queryCmd)

	// record-feedback — write one entry to the store-native feedback log. Routes through the
	// same tools.RecordFeedback the agent/chat/MCP surfaces call (spec §2.6). DB-only write:
	// unlike apply-fix it touches no desk file, so it carries no write gate.
	var fbKind, fbSummary, fbDetail, fbContext, fbSource string
	recordFeedbackCmd := &cobra.Command{
		Use:   "record-feedback",
		Short: "Record a problem or feedback entry to the store's feedback log",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := requireConfig(app, cfg, cfgErr)
			if err != nil {
				return err
			}
			return printJSON(tools.RecordFeedback(cmd.Context(), app, c, &tools.RecordFeedbackInput{
				Kind:    fbKind,
				Summary: fbSummary,
				Detail:  fbDetail,
				Context: fbContext,
				Source:  fbSource,
			}))
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

	// agent <instruction> — Phase-1 MANUAL trigger for the eino ReAct loop (spec §6;
	// agent_runs.trigger="manual"). One-shot separate process, like sweep/patrol: requires the
	// DB already migrated (a prior `serve` or `migrate up`). INTERPRETATION (see handoff): the
	// §3.3 CLI table lists no agent subcommand, but §8 Phase 1 requires a driveable loop and the
	// trigger enum includes "manual"; confirm this surface against §8 Phase 1.
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

	// chat — interactive multi-turn stewardship session over the eino loop (ADR 0001: terminal
	// surface first). On a real terminal this runs the full-screen TUI (internal/tui, ADR 0004);
	// runChat below is the line-oriented REPL fallback used when stdio is piped or --plain is
	// passed.
	// Like `agent`, it self-initializes the store on first run via requireConfig (ADR 0003) — no
	// prior `migrate up` or `serve` is required. Scope stays desk stewardship: the session
	// inherits the gated tool set (restore never exposed; apply_fix only when
	// LIBRARIAN_AUTONOMOUS_WRITES is set) and the data-backed system prompt — not a general chat.
	chatCmd := &cobra.Command{
		Use:   "chat",
		Short: "Interactive multi-turn librarian session (full-screen TUI on a terminal; REPL when piped or --plain)",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := requireConfig(app, cfg, cfgErr)
			if err != nil {
				return err
			}
			plain, _ := cmd.Flags().GetBool("plain")
			// Route to the line REPL when --plain is set, or when either stdio end is not a
			// terminal (piped input/output): the full-screen TUI requires an interactive TTY on
			// both stdin AND stdout. requireConfig has already run (and printed any self-init /
			// open-guard notice) above, so this decision — and any REPL prompt or the alternate
			// screen — happens only after config is known good.
			if plain || !isatty.IsTerminal(os.Stdin.Fd()) || !isatty.IsTerminal(os.Stdout.Fd()) {
				return runChat(cmd.Context(), app, c)
			}
			// Resolve the color theme ONCE here, in the safe window before the Bubble Tea program
			// starts — the "auto" path probes the terminal background (a query that would corrupt
			// input if fired after the program's stdin reader is running; see ADR 0004). Precedence:
			// --theme flag > LIBRARIAN_THEME env > auto-detect. Only consult the flag when the user
			// set it, so an unset flag defers to the env override rather than shadowing it.
			themeFlag := ""
			if cmd.Flags().Changed("theme") {
				// GetString only errors on an unregistered/wrong-type flag; "theme" is registered as a
				// string below, so the error is structurally impossible here — discard it explicitly.
				themeFlag, _ = cmd.Flags().GetString("theme") //nolint:errcheck // "theme" is a registered string flag
			}
			return tui.Run(cmd.Context(), app, c, tui.ResolveTheme(themeFlag))
		},
	}
	chatCmd.Flags().Bool("plain", false, "force the line-oriented REPL instead of the full-screen TUI")
	chatCmd.Flags().String("theme", "auto", "color theme for the full-screen TUI: light, dark, or auto (detect the terminal background once at startup)")
	app.RootCmd.AddCommand(chatCmd)

	// mcp-serve — expose the tool core as an MCP stdio server (spec §7.2 outbound dual-surface;
	// build-brief §5 / punch-list 4). The model-facing tool set is tools.AgentTools(cfg): the
	// SAME §5.4 registration-time write gate as the eino loop (apply_fix only when
	// LIBRARIAN_AUTONOMOUS_WRITES=true), and restore is NEVER exposed (§5.5, supervised CLI-only).
	// Like the other one-shot tool commands it opens the DB directly; because it holds the DB open
	// for the session, it MUST NOT run concurrently with `serve` (single-writer SQLite, §10.4).
	// Termination: the stdio server returns when the MCP client closes stdin (EOF), the normal
	// stdio-transport lifecycle.
	app.RootCmd.AddCommand(&cobra.Command{
		Use:   "mcp-serve",
		Short: "Expose the six-tool core as an MCP stdio server (model-facing; gated per §5.4)",
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
		Short: "Serve the DB and open the admin GUI in a browser",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Abort BEFORE opening a browser or spawning the serve child when config/the desk
			// guard fails: gui previously only used requireConfig's error to pick a fallback URL
			// and pressed on regardless, which meant a mismatched-desk store still popped a
			// browser tab and re-exec'd `serve` (which then hit its own OnServe guard exit, but
			// only after the browser had already opened against a store gui had no business
			// touching). requireConfig runs the same desk-guard check as every other subcommand.
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
			// desk-guarded and is about to display in the browser (chosen over the alternative of
			// having gui refuse an explicit --dir outright: forwarding is strictly more useful and
			// gui otherwise behaves like every other subcommand w.r.t. --dir). --dir is a process
			// flag, not an env var, so unlike DESK_ROOT/DESK_NAME/XDG_DATA_HOME (which DO inherit
			// via exec.Command's default nil Env) it does NOT propagate to the child on its own.
			// cmd.Flags().GetString("dir") reads cobra's own resolved value — the operator's
			// explicit --dir when passed, or otherwise the XDG default this process's main()
			// computed and handed to pocketbase.NewWithConfig as DefaultDataDir — so forwarding
			// unconditionally (not only when --dir was passed explicitly) keeps parent and child
			// aimed at the same store in both cases; there is no scenario where they should diverge.
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

// runChat drives a line-oriented REPL over one agent.Session. Each input line is one
// conversational turn; the session's growing history keeps the exchange multi-turn (the model
// sees prior turns). Exit with "exit", "quit", or EOF (Ctrl-D). The write boundary and the
// stewardship scope are the Session's, inherited from the gated tool set — nothing here opens a
// new capability.
func runChat(ctx context.Context, app core.App, cfg *config.Config) error {
	sess, err := agent.NewSession(ctx, app, cfg)
	if err != nil {
		return err
	}
	defer func() { _ = sess.Close(ctx) }()

	fmt.Printf("pocket-librarian session for desk %q — type a request; 'exit', 'quit', or Ctrl-D to end.\n", cfg.DeskName)
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

// printJSON marshals a tool's typed result (or returns its error) to stdout.
func printJSON[T any](res T, err error) error {
	if err != nil {
		return err
	}
	b, merr := json.MarshalIndent(res, "", "  ")
	if merr != nil {
		return merr
	}
	fmt.Println(string(b))
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

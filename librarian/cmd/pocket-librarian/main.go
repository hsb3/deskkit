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

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
	"github.com/spf13/cobra"

	"github.com/example/pocket-librarian/internal/agent"
	"github.com/example/pocket-librarian/internal/bootstrap"
	"github.com/example/pocket-librarian/internal/config"
	"github.com/example/pocket-librarian/internal/desklib"
	"github.com/example/pocket-librarian/internal/mcp"
	"github.com/example/pocket-librarian/internal/prompt"
	"github.com/example/pocket-librarian/internal/tools"
	"github.com/example/pocket-librarian/internal/trigger"

	// Blank-import registers all Go migrations (spec §4.11).
	_ "github.com/example/pocket-librarian/migrations"
)

func main() {
	app := pocketbase.New()

	// Automigrate on startup; also `pocket-librarian migrate up`. See §11.3 open item 1:
	// confirm the automigrate generated-migration behavior in the run environment.
	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		Automigrate: true,
	})

	// Config is loaded lazily-tolerant: serve/migrate (schema ops) can run even before
	// DESK_ROOT/DESK_NAME are set, but the tool subcommands require them (requireConfig).
	cfg, cfgErr := config.Load()

	// On serve start (after bootstrap/migrations): ensure the ignore boundary exists and
	// seed the system prompt on first run (spec §10.1, §6.1). These run only under serve.
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		if cfgErr == nil {
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
				if err != nil && cmdErr == nil {
					cmdErr = err
				}
				return err
			}
		}
	}

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
	if cmdErr != nil {
		os.Exit(1)
	}
}

func requireConfig(app core.App, cfg *config.Config, cfgErr error) (*config.Config, error) {
	if cfgErr != nil {
		return nil, cfgErr
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

// registerToolCommands wires the six tool subcommands + gui onto the PocketBase RootCmd.
// serve, migrate, and superuser are provided by PocketBase / migratecmd. Each tool command
// routes through the same tools.* function the agent will call (spec §2.6, §3.3). Until the
// tool-body slice lands these return ErrNotImplemented — expected for the spine.
func registerToolCommands(app *pocketbase.PocketBase, cfg *config.Config, cfgErr error) {
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
		Short: "Read-only queries: live_files recent orphans uncollapsed findings summary adoption",
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

	// chat — interactive multi-turn stewardship session (REPL) over the eino loop (ADR 0001:
	// terminal surface first). Like `agent`, a one-shot process that opens the DB directly, so
	// it requires a prior `migrate up` (or serve). Scope stays desk stewardship: the session
	// inherits the gated tool set (restore never exposed; apply_fix only when
	// LIBRARIAN_AUTONOMOUS_WRITES is set) and the data-backed system prompt — not a general chat.
	app.RootCmd.AddCommand(&cobra.Command{
		Use:   "chat",
		Short: "Interactive multi-turn librarian session (REPL over the agent loop)",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := requireConfig(app, cfg, cfgErr)
			if err != nil {
				return err
			}
			return runChat(cmd.Context(), app, c)
		},
	})

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
			c, err := requireConfig(app, cfg, cfgErr)
			url := "http://127.0.0.1:8090/_/"
			if err == nil {
				url = strings.TrimRight(c.PBURL, "/") + "/_/"
			}
			openBrowser(url)
			bin, xerr := os.Executable()
			if xerr != nil {
				return xerr
			}
			child := exec.Command(bin, append([]string{"serve"}, args...)...)
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

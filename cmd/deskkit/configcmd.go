package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/hsb3/deskkit/internal/core/config"
	"github.com/hsb3/deskkit/internal/core/settings"
)

// newConfigCmd builds the `config` group: what is resolved, where each value came from, where
// the machine-wide file lives, and how to change it. Like `desks` it opens no store and creates
// nothing (except the central config file, on `set`/`edit`) — it is registered on the RootCmd
// for discovery, and main() intercepts the real run before PocketBase can bootstrap a store.
func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "config",
		Short:         "Show the resolved configuration and edit the machine-wide config file",
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	cmd.AddCommand(newConfigShowCmd(), newConfigPathCmd(), newConfigEditCmd(), newConfigSetCmd())
	return cmd
}

func newConfigShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Print every resolved setting with the source that won (env, profile, store, central, default)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return writeConfigShow(cmd.OutOrStdout())
		},
	}
}

func newConfigPathCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Print the path of the machine-wide config file",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			path, err := config.CentralPath()
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s  %s\n", path, existsNote(path))
			return nil
		},
	}
}

func newConfigEditCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "edit",
		Short: "Open the machine-wide config file in $VISUAL or $EDITOR",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			path, err := config.CentralPath()
			if err != nil {
				return err
			}
			editor := strings.TrimSpace(os.Getenv("VISUAL"))
			if editor == "" {
				editor = strings.TrimSpace(os.Getenv("EDITOR"))
			}
			if editor == "" {
				return fmt.Errorf("no editor configured: set $VISUAL or $EDITOR, or edit %s directly", path)
			}
			// Create the file (0700 dir, 0600 file) before handing it to the editor, so the
			// editor never has to create it with a looser umask-derived mode.
			if _, serr := os.Stat(path); errors.Is(serr, fs.ErrNotExist) {
				c, lerr := config.LoadCentral()
				if lerr != nil {
					return lerr
				}
				if serr := config.SaveCentral(c); serr != nil {
					return serr
				}
			}
			// $VISUAL/$EDITOR conventionally may carry flags ("code -w"), so split on spaces.
			argv := append(strings.Fields(editor), path)
			ed := exec.Command(argv[0], argv[1:]...) // #nosec G204 -- the operator's own $EDITOR
			ed.Stdin, ed.Stdout, ed.Stderr = os.Stdin, os.Stdout, os.Stderr
			return ed.Run()
		},
	}
}

func newConfigSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a key in the machine-wide config file: " + strings.Join(config.CentralKeys(), ", "),
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key, value := args[0], args[1]
			c, err := config.LoadCentral()
			if err != nil {
				// `set` must not rewrite a file it could not parse — that would silently discard
				// whatever else the operator had in it. `edit` is the way out: it stats the path
				// and hands the raw file to $EDITOR without ever parsing it. The error already
				// names the file.
				return fmt.Errorf("%w\nrepair it by hand with `deskkit config edit`, then re-run `deskkit config set`", err)
			}
			if err := c.Set(key, value); err != nil {
				return err
			}
			if err := config.SaveCentral(c); err != nil {
				return err
			}
			path, err := config.CentralPath()
			if err != nil {
				return err
			}
			// The API key is confirmed MASKED — a shell transcript (or a screen share) must never
			// carry the secret back out of the tool that just accepted it.
			shown := value
			if key == "llm.api_key" {
				shown = config.MaskSecret(value)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "set %s = %s in %s\n", key, shown, path)
			return nil
		},
	}
}

// configRow is one line of `config show`: the setting's env-var name, its resolved value, and
// the leg of the precedence chain that supplied it.
type configRow struct{ key, value, source string }

// configRows renders every resolved setting. The SOURCE column comes STRAIGHT from cfg.Sources —
// which the resolver wrote as it decided — and is never re-derived here: a second copy of the
// precedence rules is a copy that drifts from the real one, which is the whole failure this
// command exists to prevent. Secrets are masked at this boundary, so no caller can print one.
func configRows(cfg *config.Config) []configRow {
	row := func(key, value string) configRow {
		if value == "" {
			value = "(unset)"
		}
		return configRow{key: key, value: value, source: cfg.Sources[key]}
	}
	b := func(v bool) string { return strconv.FormatBool(v) }
	n := func(v int) string { return strconv.Itoa(v) }

	return []configRow{
		row("DESK_NAME", cfg.DeskName),
		row("DESK_ROOT", cfg.DeskRoot),
		row("DECISIONS_DIR", cfg.DecisionsDir),
		row("TASKS_DIR", cfg.TasksDir),
		row("ANALYSES_DIR", cfg.AnalysesDir),
		row("JOURNAL_DIR", cfg.JournalDir),
		row("SECRETS_DIR", cfg.SecretsDir),
		row("HANDOFF_PATH", cfg.HandoffPath),
		row("IGNORE_CONFIG", cfg.IgnoreConfig),

		row("LLM_PROVIDER", cfg.LLMProvider),
		row("LLM_MODEL", cfg.LLMModel),
		row("LLM_API_KEY_ENV", cfg.LLMAPIKeyEnv),
		row("LLM_MAX_TOKENS", n(cfg.LLMMaxTokens)),
		row("LLM_CONTEXT_WINDOW", n(cfg.LLMContextWindow)),

		row("AGENT_MAX_STEP", n(cfg.AgentMaxStep)),
		row("LIBRARIAN_AUTONOMOUS_WRITES", b(cfg.AutonomousWrites)),
		row("CLAIMER_POLL_INTERVAL", cfg.ClaimerPollInterval.String()),

		row("PM_ENABLED", b(cfg.PMEnabled)),
		row("PM_CLAIM_TTL", cfg.PMClaimTTL.String()),
		row("PM_AUTONOMOUS_WRITES", b(cfg.PMAutonomousWrites)),
		row("PM_STALLED_DAYS", n(cfg.PMStalledDays)),

		row("PB_URL", cfg.PBURL),
		row("PB_SUPERUSER_EMAIL", cfg.PBSuperuserEmail),
		// The one secret Config carries (PocketBase needs it in process at startup).
		row("PB_SUPERUSER_PASSWORD", config.MaskSecret(cfg.PBSuperuserPassword)),
	}
}

// writeConfigShow renders the resolved configuration. An unresolvable desk is NOT an error: this
// is the command an operator runs to find out what is set, so it degrades to the machine-wide
// half plus a pointer at `deskkit init`.
func writeConfigShow(w io.Writer) error {
	centralPath, err := config.CentralPath()
	if err != nil {
		return err
	}

	cfg, cfgErr := config.Load()
	if cfgErr != nil {
		central, lerr := config.LoadCentral()
		centralNote := existsNote(centralPath)
		if lerr != nil {
			// Both halves broken — no desk here AND an unparseable file — is EXACTLY when this
			// command gets run, so it degrades the same way Load does (warn, treat as absent)
			// instead of exiting 1 with nothing actionable. The marker names the file to fix.
			central, centralNote = &config.Central{}, "(unparseable — fix it with `deskkit config edit`)"
		}
		fmt.Fprintf(w, "No desk resolves here: %v\n\n", cfgErr)
		fmt.Fprintf(w, "Machine-wide config file: %s  %s\n\n", centralPath, centralNote)
		tw := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
		fmt.Fprintf(tw, "KEY\tVALUE\n")
		for _, key := range config.CentralKeys() {
			v, _ := central.Get(key)
			if key == "llm.api_key" {
				v = config.MaskSecret(v)
			} else if v == "" {
				v = "(unset)"
			}
			fmt.Fprintf(tw, "%s\t%s\n", key, v)
		}
		if err := tw.Flush(); err != nil {
			return err
		}
		fmt.Fprintf(w, "\nMake this folder a desk with `deskkit init`, then re-run `deskkit config show`.\n")
		fmt.Fprintf(w, "Change a machine-wide value with `deskkit config set <key> <value>`.\n")
		return nil
	}

	storeDir, locErr := resolveStoreLocation(cfg, nil)
	// The store leg of the precedence chain (env > profile > store > central > default). `config
	// show` opens no store and creates nothing — a PocketBase bootstrap would MkdirAll the data dir
	// and run system migrations against it — so the row is read straight out of an EXISTING
	// data.db. A desk with no store yet simply has nothing on this leg, which is the honest answer
	// rather than a guessed one.
	var stored *settings.Settings
	if locErr == nil {
		var serr error
		if stored, serr = settings.LoadFromDir(storeDir); serr != nil {
			// Never silently drop a leg: an unreadable store would otherwise show as "no value
			// stored", which is a different fact from "could not look".
			fmt.Fprintf(w, "Could not read this desk's store settings: %v\n\n", serr)
			stored = nil
		}
		config.ApplySettings(cfg, stored)
	} else {
		storeDir = "(unresolved: " + locErr.Error() + ")"
	}
	fmt.Fprintf(w, "Desk %q\n", cfg.DeskName)
	fmt.Fprintf(w, "  store        %s\n", storeDir)
	fmt.Fprintf(w, "  config file  %s  %s\n\n", centralPath, existsNote(centralPath))

	tw := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
	fmt.Fprintf(tw, "SETTING\tVALUE\tSOURCE\n")
	for _, r := range configRows(cfg) {
		fmt.Fprintf(tw, "%s\t%s\t%s\n", r.key, r.value, r.source)
	}
	// The LLM API key is never held in Config: it resolves at use time from the named env var, this
	// desk's store, or the central file, so it gets its own row from the one resolver both surfaces
	// call — always masked.
	envName := config.APIKeyEnvName(cfg)
	key, keySource := config.ResolveAPIKeySettings(stored, envName)
	if keySource == "" {
		keySource = "(unresolved)"
	}
	fmt.Fprintf(tw, "%s (LLM API key)\t%s\t%s\n", envName, config.MaskSecret(key), keySource)
	if err := tw.Flush(); err != nil {
		return err
	}

	fmt.Fprintf(w, "\nSOURCE is the leg that won: env > profile > store > central > default.\n")
	fmt.Fprintf(w, "\"store\" is this desk's own settings, editable from the browser at `deskkit serve`.\n")
	fmt.Fprintf(w, "Change a machine-wide value with `deskkit config set <key> <value>` (file: `deskkit config path`).\n")
	fmt.Fprintf(w, "Browse this desk's data visually with `deskkit gui`.\n")
	return nil
}

// existsNote annotates a path with whether it is actually there — a config file that does not
// exist yet is the normal state, and saying so beats printing a path that looks authoritative.
func existsNote(path string) string {
	if _, err := os.Stat(path); err == nil {
		return "(exists)"
	}
	return "(not created yet)"
}

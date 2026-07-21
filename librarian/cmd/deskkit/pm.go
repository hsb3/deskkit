// The `pm` command group (spec §5.3): the owner/script CLI surface over the PM tool family.
// Registered ONLY when the pm module is enabled (spec §2.9: with PM off, PM tools are absent
// from every surface — `deskkit pm` is then cobra's normal unknown-command error). Every
// subcommand is a thin adapter over the SAME modules/pm/tools functions the MCP server and
// the TUI views call (§10.10 one core, three surfaces); raw JSON on stdout is the default
// machine contract, mirroring the librarian tools' JSON-first output.
//
// Frozen command names (docs + the D5 plugin build on them):
//
//	pm context · pm list · pm get · pm create · pm update · pm transition · pm block ·
//	pm unblock · pm note · pm link · pm claim · pm release
//
// Audit identity (§3.6): --actor (persistent; default $USER, else "operator") with
// actor_kind "human" — the CLI is the human surface; agents come in over MCP.
//
// Concurrency ergonomics: the engine's optimistic version check (R2.6) binds every mutation.
// --version passes the token explicitly (refused on mismatch); omitting it makes the CLI
// read the item's current version first — the supervised-operator convenience, documented as
// a read-then-write (scripts wanting the strict check pass --version).
package main

import (
	"fmt"
	"os"

	"github.com/pocketbase/pocketbase"
	pbcore "github.com/pocketbase/pocketbase/core"
	"github.com/spf13/cobra"

	"github.com/hsb3/desk-standard/librarian/internal/core/config"
	"github.com/hsb3/desk-standard/librarian/internal/core/schema"
	pmtools "github.com/hsb3/desk-standard/librarian/internal/modules/pm/tools"
)

// pmValidator resolves the document validator captured at module registration (the
// librarian's); read lazily so registration order never matters.
func pmValidator() schema.DocumentValidator {
	if moduleReg == nil {
		return nil
	}
	return moduleReg.Validator
}

// defaultCLIActor is the CLI's audit identity default: $USER, else a neutral "operator".
func defaultCLIActor() string {
	if u := os.Getenv("USER"); u != "" {
		return u
	}
	return "operator"
}

// registerPMCommands wires the `pm` group onto the RootCmd. Callers gate on cfg.PMEnabled —
// this function assumes the module is on.
func registerPMCommands(app *pocketbase.PocketBase, cfg *config.Config, cfgErr error) {
	var actorFlag string

	pmCmd := &cobra.Command{
		Use:   "pm",
		Short: "The desk work graph: document-gated items, transitions, dependencies",
	}
	pmCmd.PersistentFlags().StringVar(&actorFlag, "actor", defaultCLIActor(),
		"audit actor recorded on writes (default $USER)")

	actor := func() pmtools.ActorFields {
		return pmtools.ActorFields{Actor: actorFlag, ActorKind: "human"}
	}

	// resolveVersion returns the explicit --version when set (>= 0), else the item's current
	// version (CLI convenience; see the package comment). The lookup is DESK-SCOPED (review
	// finding): an item id from another desk must read as "not on this desk", never leak its
	// version across the boundary. cfg is non-nil here — callers run requireConfig first.
	resolveVersion := func(app pbcore.App, c *config.Config, itemID string, flag int) (int, error) {
		if flag >= 0 {
			return flag, nil
		}
		recs, err := app.FindRecordsByFilter("items", "id = {:id} && desk = {:desk}", "", 1, 0,
			map[string]any{"id": itemID, "desk": c.DeskName})
		if err != nil || len(recs) == 0 {
			return 0, fmt.Errorf("no item %q on desk %q", itemID, c.DeskName)
		}
		return recs[0].GetInt("version"), nil
	}

	// pm context
	var stalledDays int
	contextCmd := &cobra.Command{
		Use:   "context",
		Short: "Single-call cold-start briefing: active, blocked, stalled, recent transitions",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := requireConfig(app, cfg, cfgErr)
			if err != nil {
				return err
			}
			res, perr := pmtools.GetContext(cmd.Context(), app, c, pmValidator(),
				&pmtools.GetContextInput{StalledDays: stalledDays})
			return printJSON(cmd.OutOrStdout(), res, perr)
		},
	}
	contextCmd.Flags().IntVar(&stalledDays, "stalled-days", 0, "stalled threshold in days (default 14 / PM_STALLED_DAYS)")
	pmCmd.AddCommand(contextCmd)

	// pm list
	var lf pmtools.ListItemsInput
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "Filtered work-graph query",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := requireConfig(app, cfg, cfgErr)
			if err != nil {
				return err
			}
			res, perr := pmtools.ListItems(cmd.Context(), app, c, pmValidator(), &lf)
			return printJSON(cmd.OutOrStdout(), res, perr)
		},
	}
	listCmd.Flags().StringVar(&lf.Phase, "phase", "", "filter: queue, work, review, terminal")
	listCmd.Flags().StringVar(&lf.Court, "court", "", "filter: owner, desk, crew, vendor, external-session")
	listCmd.Flags().StringVar(&lf.Type, "type", "", "filter: schema-v1/kit item type")
	listCmd.Flags().StringVar(&lf.Blocked, "blocked", "", "filter the blocked flag: true or false")
	listCmd.Flags().StringVar(&lf.Parent, "parent", "", "filter to direct children of this item id")
	pmCmd.AddCommand(listCmd)

	// pm get <id>
	pmCmd.AddCommand(&cobra.Command{
		Use:   "get <id>",
		Short: "One item with notes, dependencies, transitions, ancestors",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := requireConfig(app, cfg, cfgErr)
			if err != nil {
				return err
			}
			res, perr := pmtools.GetItem(cmd.Context(), app, c, pmValidator(),
				&pmtools.GetItemInput{ItemID: args[0]})
			return printJSON(cmd.OutOrStdout(), res, perr)
		},
	})

	// pm create
	var ci pmtools.CreateItemInput
	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Add a work item to the graph (phase starts at queue)",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := requireConfig(app, cfg, cfgErr)
			if err != nil {
				return err
			}
			ci.ActorFields = actor()
			res, perr := pmtools.CreateItem(cmd.Context(), app, c, pmValidator(), &ci)
			return printJSON(cmd.OutOrStdout(), res, perr)
		},
	}
	createCmd.Flags().StringVar(&ci.Title, "title", "", "the item title")
	createCmd.Flags().StringVar(&ci.Type, "type", "", "schema-v1/kit item type (e.g. decision, task)")
	createCmd.Flags().StringVar(&ci.Parent, "parent", "", "parent item id (omit for a root)")
	createCmd.Flags().StringVar(&ci.Court, "court", "", "owner, desk, crew, vendor, external-session")
	createCmd.Flags().StringVar(&ci.Pointer, "pointer", "", "doc path / issue URL / other locus")
	createCmd.Flags().StringVar(&ci.Body, "body", "", "long-form body: narrative, acceptance criteria, or spec, stored inline")
	createCmd.Flags().StringVar(&ci.Severity, "severity", "", "low, medium, high")
	createCmd.Flags().IntVar(&ci.Priority, "priority", 0, "ordinal within a court/queue")
	_ = createCmd.MarkFlagRequired("title")
	pmCmd.AddCommand(createCmd)

	// pm update <id>
	var ui pmtools.UpdateItemInput
	var updateVersion int
	updateCmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Edit an item's first-class fields (empty flag = unchanged)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := requireConfig(app, cfg, cfgErr)
			if err != nil {
				return err
			}
			ver, verr := resolveVersion(app, c, args[0], updateVersion)
			if verr != nil {
				return verr
			}
			ui.ItemID, ui.Version, ui.ActorFields = args[0], ver, actor()
			res, perr := pmtools.UpdateItem(cmd.Context(), app, c, pmValidator(), &ui)
			return printJSON(cmd.OutOrStdout(), res, perr)
		},
	}
	updateCmd.Flags().IntVar(&updateVersion, "version", -1, "version token you read, >= 1 (omit = use the item's current version)")
	updateCmd.Flags().StringVar(&ui.Title, "title", "", "new title")
	updateCmd.Flags().StringVar(&ui.Type, "type", "", "new schema-v1/kit type")
	updateCmd.Flags().StringVar(&ui.Court, "court", "", "new court")
	updateCmd.Flags().StringVar(&ui.Pointer, "pointer", "", "new document pointer")
	updateCmd.Flags().StringVar(&ui.Body, "body", "", "new body (empty = unchanged)")
	updateCmd.Flags().StringVar(&ui.Severity, "severity", "", "new severity")
	updateCmd.Flags().IntVar(&ui.Priority, "priority", 0, "new priority (0 = unchanged)")
	updateCmd.Flags().StringVar(&ui.Properties, "properties", "", "new properties JSON object")
	updateCmd.Flags().StringVar(&ui.StatusLabel, "status-label", "", "new status label (a different phase's label is a gated transition request)")
	pmCmd.AddCommand(updateCmd)

	// pm transition <id> --to <phase>
	var toPhase string
	var transitionVersion int
	transitionCmd := &cobra.Command{
		Use:   "transition <id>",
		Short: "Request a phase transition (advance/demote/reopen); gates may refuse",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := requireConfig(app, cfg, cfgErr)
			if err != nil {
				return err
			}
			ver, verr := resolveVersion(app, c, args[0], transitionVersion)
			if verr != nil {
				return verr
			}
			res, perr := pmtools.TransitionItem(cmd.Context(), app, c, pmValidator(),
				&pmtools.TransitionItemInput{
					ItemID: args[0], TargetPhase: toPhase, Version: ver, ActorFields: actor(),
				})
			return printJSON(cmd.OutOrStdout(), res, perr)
		},
	}
	transitionCmd.Flags().StringVar(&toPhase, "to", "", "target phase: queue, work, review, terminal")
	transitionCmd.Flags().IntVar(&transitionVersion, "version", -1, "version token you read, >= 1 (omit = use the item's current version)")
	_ = transitionCmd.MarkFlagRequired("to")
	pmCmd.AddCommand(transitionCmd)

	// pm block <id> / pm unblock <id>
	var blockReason string
	var blockVersion int
	blockCmd := &cobra.Command{
		Use:   "block <id>",
		Short: "Set the blocked side-state (preserves the phase)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := requireConfig(app, cfg, cfgErr)
			if err != nil {
				return err
			}
			ver, verr := resolveVersion(app, c, args[0], blockVersion)
			if verr != nil {
				return verr
			}
			res, perr := pmtools.BlockItem(cmd.Context(), app, c, pmValidator(),
				&pmtools.BlockItemInput{ItemID: args[0], Version: ver, Reason: blockReason, ActorFields: actor()})
			return printJSON(cmd.OutOrStdout(), res, perr)
		},
	}
	blockCmd.Flags().StringVar(&blockReason, "reason", "", "why the item is blocked (audit detail)")
	blockCmd.Flags().IntVar(&blockVersion, "version", -1, "version token you read, >= 1 (omit = use the item's current version)")
	pmCmd.AddCommand(blockCmd)

	var unblockReason string
	var unblockVersion int
	unblockCmd := &cobra.Command{
		Use:   "unblock <id>",
		Short: "Clear the blocked side-state (restores the held phase)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := requireConfig(app, cfg, cfgErr)
			if err != nil {
				return err
			}
			ver, verr := resolveVersion(app, c, args[0], unblockVersion)
			if verr != nil {
				return verr
			}
			res, perr := pmtools.UnblockItem(cmd.Context(), app, c, pmValidator(),
				&pmtools.UnblockItemInput{ItemID: args[0], Version: ver, Reason: unblockReason, ActorFields: actor()})
			return printJSON(cmd.OutOrStdout(), res, perr)
		},
	}
	unblockCmd.Flags().StringVar(&unblockReason, "reason", "", "why the block clears (audit detail)")
	unblockCmd.Flags().IntVar(&unblockVersion, "version", -1, "version token you read, >= 1 (omit = use the item's current version)")
	pmCmd.AddCommand(unblockCmd)

	// pm note <id>
	var noteKey, noteBody string
	noteCmd := &cobra.Command{
		Use:   "note <id>",
		Short: "Attach a phase-scoped keyed note (the lighter artifact)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := requireConfig(app, cfg, cfgErr)
			if err != nil {
				return err
			}
			res, perr := pmtools.AddNote(cmd.Context(), app, c, pmValidator(),
				&pmtools.AddNoteInput{ItemID: args[0], Key: noteKey, Body: noteBody, ActorFields: actor()})
			return printJSON(cmd.OutOrStdout(), res, perr)
		},
	}
	noteCmd.Flags().StringVar(&noteKey, "key", "", "the note key (e.g. rationale, handoff)")
	noteCmd.Flags().StringVar(&noteBody, "body", "", "the note body")
	_ = noteCmd.MarkFlagRequired("key")
	_ = noteCmd.MarkFlagRequired("body")
	pmCmd.AddCommand(noteCmd)

	// pm link <from> <to>
	var li pmtools.LinkItemsInput
	linkCmd := &cobra.Command{
		Use:   "link <from> <to>",
		Short: "Create a typed dependency edge between two items",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := requireConfig(app, cfg, cfgErr)
			if err != nil {
				return err
			}
			li.From, li.To, li.ActorFields = args[0], args[1], actor()
			res, perr := pmtools.LinkItems(cmd.Context(), app, c, pmValidator(), &li)
			return printJSON(cmd.OutOrStdout(), res, perr)
		},
	}
	linkCmd.Flags().StringVar(&li.Kind, "kind", "", "blocks, is-blocked-by, or relates-to")
	linkCmd.Flags().StringVar(&li.UnblockAt, "unblock-at", "", "for blocks edges: work, review, or terminal")
	linkCmd.Flags().StringVar(&li.Cascade, "cascade", "", "for blocks edges: auto, manual, auto-reopen, permanent")
	_ = linkCmd.MarkFlagRequired("kind")
	pmCmd.AddCommand(linkCmd)

	// pm claim <id> / pm release <id>
	var claimVersion int
	claimCmd := &cobra.Command{
		Use:   "claim <id>",
		Short: "Claim an item with a TTL so two agents never double-work it",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := requireConfig(app, cfg, cfgErr)
			if err != nil {
				return err
			}
			ver, verr := resolveVersion(app, c, args[0], claimVersion)
			if verr != nil {
				return verr
			}
			res, perr := pmtools.ClaimItem(cmd.Context(), app, c, pmValidator(),
				&pmtools.ClaimItemInput{ItemID: args[0], Version: ver, ActorFields: actor()})
			return printJSON(cmd.OutOrStdout(), res, perr)
		},
	}
	claimCmd.Flags().IntVar(&claimVersion, "version", -1, "version token you read, >= 1 (omit = use the item's current version)")
	pmCmd.AddCommand(claimCmd)

	var releaseVersion int
	releaseCmd := &cobra.Command{
		Use:   "release <id>",
		Short: "Release an item's claim",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := requireConfig(app, cfg, cfgErr)
			if err != nil {
				return err
			}
			ver, verr := resolveVersion(app, c, args[0], releaseVersion)
			if verr != nil {
				return verr
			}
			res, perr := pmtools.ReleaseItem(cmd.Context(), app, c, pmValidator(),
				&pmtools.ReleaseItemInput{ItemID: args[0], Version: ver, ActorFields: actor()})
			return printJSON(cmd.OutOrStdout(), res, perr)
		},
	}
	releaseCmd.Flags().IntVar(&releaseVersion, "version", -1, "version token you read, >= 1 (omit = use the item's current version)")
	pmCmd.AddCommand(releaseCmd)

	// A gate refusal (or any engine refusal) is an EXPECTED domain outcome on this JSON-first
	// surface: print the refusal line, never a usage dump. Cobra only honors SilenceUsage on
	// the root or the executed leaf, so set it on every leaf here.
	for _, c := range pmCmd.Commands() {
		c.SilenceUsage = true
	}

	app.RootCmd.AddCommand(pmCmd)
}

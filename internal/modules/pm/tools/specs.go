package tools

import (
	"context"

	"github.com/pocketbase/pocketbase/core"

	cfgpkg "github.com/hsb3/deskkit/internal/core/config"
	"github.com/hsb3/deskkit/internal/core/schema"
	"github.com/hsb3/deskkit/internal/core/toolcore"
)

// Specs returns the PM module's twelve tools as toolcore.ToolSpecs (spec §5.1). The tool
// NAMES are frozen — docs and the D5 plugin build on them:
//
//	get_context, list_items, get_item, create_item, update_item, transition_item,
//	block_item, unblock_item, add_note, link_items, claim_item, release_item
//
// validator is a lazy getter for the DocumentValidator the module receives at registration
// (registration populates it AFTER Tools() is collected, so the closure must read late).
//
// Gate encoding (spec §5.1, §13 item 9): PM tools write only the STORE, never desk files
// (WritesFiles=false on all twelve; the §5.4 LIBRARIAN_AUTONOMOUS_WRITES gate does not apply).
// The read tools are AgentDefault always; the write tools are AgentDefault only while
// PM_AUTONOMOUS_WRITES (default ON) holds — writesEnabled false makes agents read-only over
// the graph while transition_item's document gates remain the real safety.
func Specs(validator func() schema.DocumentValidator, writesEnabled bool) []toolcore.ToolSpec {
	const mod = "pm"
	w := writesEnabled
	return []toolcore.ToolSpec{
		toolcore.New[GetContextInput](mod, "get_context",
			"Single-call cold-start briefing: active, blocked, stalled, recent transitions, ancestors, counts for this desk.",
			false, true, false,
			func(ctx context.Context, app core.App, cfg *cfgpkg.Config, in *GetContextInput) (any, error) {
				return GetContext(ctx, app, cfg, validator(), in)
			}),
		toolcore.New[ListItemsInput](mod, "list_items",
			"Filtered work-graph query by phase, court, type, blocked flag, or parent.",
			false, true, false,
			func(ctx context.Context, app core.App, cfg *cfgpkg.Config, in *ListItemsInput) (any, error) {
				return ListItems(ctx, app, cfg, validator(), in)
			}),
		toolcore.New[GetItemInput](mod, "get_item",
			"One work item with its notes, dependency edges, recent transitions, and ancestor chain.",
			false, true, false,
			func(ctx context.Context, app core.App, cfg *cfgpkg.Config, in *GetItemInput) (any, error) {
				return GetItem(ctx, app, cfg, validator(), in)
			}),
		toolcore.New[CreateItemInput](mod, "create_item",
			"Add a work item to the graph (phase starts at queue).",
			false, w, false,
			func(ctx context.Context, app core.App, cfg *cfgpkg.Config, in *CreateItemInput) (any, error) {
				return CreateItem(ctx, app, cfg, validator(), in)
			}),
		toolcore.New[UpdateItemInput](mod, "update_item",
			"Edit an item's first-class fields (version-checked); a status label of a different phase is a gated transition request.",
			false, w, false,
			func(ctx context.Context, app core.App, cfg *cfgpkg.Config, in *UpdateItemInput) (any, error) {
				return UpdateItem(ctx, app, cfg, validator(), in)
			}),
		toolcore.New[TransitionItemInput](mod, "transition_item",
			"Request any legal phase transition (advance, demote, reopen); the server refuses until the phase's required documents validate.",
			false, w, false,
			func(ctx context.Context, app core.App, cfg *cfgpkg.Config, in *TransitionItemInput) (any, error) {
				return TransitionItem(ctx, app, cfg, validator(), in)
			}),
		toolcore.New[BlockItemInput](mod, "block_item",
			"Set an item's blocked side-state (preserves its phase).",
			false, w, false,
			func(ctx context.Context, app core.App, cfg *cfgpkg.Config, in *BlockItemInput) (any, error) {
				return BlockItem(ctx, app, cfg, validator(), in)
			}),
		toolcore.New[UnblockItemInput](mod, "unblock_item",
			"Clear an item's blocked side-state (restores its held phase).",
			false, w, false,
			func(ctx context.Context, app core.App, cfg *cfgpkg.Config, in *UnblockItemInput) (any, error) {
				return UnblockItem(ctx, app, cfg, validator(), in)
			}),
		toolcore.New[AddNoteInput](mod, "add_note",
			"Attach a phase-scoped keyed note to an item (the lighter artifact).",
			false, w, false,
			func(ctx context.Context, app core.App, cfg *cfgpkg.Config, in *AddNoteInput) (any, error) {
				return AddNote(ctx, app, cfg, validator(), in)
			}),
		toolcore.New[LinkItemsInput](mod, "link_items",
			"Create a typed dependency edge (blocks, is-blocked-by, relates-to) with unblock-at and cascade semantics.",
			false, w, false,
			func(ctx context.Context, app core.App, cfg *cfgpkg.Config, in *LinkItemsInput) (any, error) {
				return LinkItems(ctx, app, cfg, validator(), in)
			}),
		toolcore.New[ClaimItemInput](mod, "claim_item",
			"Claim an item with a TTL so two agents never double-work it.",
			false, w, false,
			func(ctx context.Context, app core.App, cfg *cfgpkg.Config, in *ClaimItemInput) (any, error) {
				return ClaimItem(ctx, app, cfg, validator(), in)
			}),
		toolcore.New[ReleaseItemInput](mod, "release_item",
			"Release an item's claim (only the holder may release a live claim).",
			false, w, false,
			func(ctx context.Context, app core.App, cfg *cfgpkg.Config, in *ReleaseItemInput) (any, error) {
				return ReleaseItem(ctx, app, cfg, validator(), in)
			}),
	}
}

// ToolNames lists the frozen PM tool ids in registration order (test + docs anchor).
func ToolNames() []string {
	return []string{
		"get_context", "list_items", "get_item", "create_item", "update_item",
		"transition_item", "block_item", "unblock_item", "add_note", "link_items",
		"claim_item", "release_item",
	}
}

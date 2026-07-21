// The twelve tool bodies: thin adapters mapping frozen inputs onto the engine's core
// functions and its results onto stable JSON shapes. Every surface (MCP via specs.go, the CLI
// pm group, the TUI views) routes through these — one core, three surfaces (R4.1).
package tools

import (
	"context"
	"time"

	"github.com/pocketbase/pocketbase/core"

	cfgpkg "github.com/hsb3/desk-standard/librarian/internal/core/config"
	"github.com/hsb3/desk-standard/librarian/internal/core/schema"
	"github.com/hsb3/desk-standard/librarian/internal/modules/pm/engine"
)

// newEngine binds one call's engine: the surface's app + cfg, and the DocumentValidator the
// module received at registration (nil = documented gates fail closed, §2.5).
func newEngine(app core.App, cfg *cfgpkg.Config, v schema.DocumentValidator) *engine.Engine {
	return &engine.Engine{App: app, Cfg: cfg, Validator: v}
}

// actorOf maps the frozen actor fields to the engine's audit identity. Unset defaults to an
// agent actor — the model-facing surfaces are agent-driven; the CLI passes explicit values.
func actorOf(a ActorFields) engine.Actor {
	actor := engine.Actor{Name: a.Actor, Kind: a.ActorKind, DelegationParent: a.DelegationParent}
	if actor.Name == "" {
		actor.Name = "agent"
	}
	if actor.Kind == "" {
		actor.Kind = "agent"
	}
	return actor
}

// itemResult is the stable mutation-result shape: the item's post-operation summary.
type itemResult struct {
	Item engine.ItemSummary `json:"item"`
}

// edgeResult is link_items' stable result: the canonical stored edge.
type edgeResult struct {
	Edge engine.DependencyRow `json:"edge"`
}

// noteResult is add_note's stable result.
type noteResult struct {
	Note engine.NoteRow `json:"note"`
}

// GetContext — §5.2 the cold-start briefing.
func GetContext(ctx context.Context, app core.App, cfg *cfgpkg.Config, v schema.DocumentValidator, in *GetContextInput) (*engine.ContextResult, error) {
	return newEngine(app, cfg, v).GetContext(ctx, in.StalledDays)
}

// ListItems — §5.1 filtered graph query.
func ListItems(ctx context.Context, app core.App, cfg *cfgpkg.Config, v schema.DocumentValidator, in *ListItemsInput) ([]engine.ItemSummary, error) {
	return newEngine(app, cfg, v).ListItems(ctx, engine.ListFilter{
		Phase: in.Phase, Court: in.Court, Type: in.Type, Blocked: in.Blocked, Parent: in.Parent,
	})
}

// GetItem — §5.1 one item + notes, deps, recent transitions, ancestors.
func GetItem(ctx context.Context, app core.App, cfg *cfgpkg.Config, v schema.DocumentValidator, in *GetItemInput) (*engine.ItemDetail, error) {
	return newEngine(app, cfg, v).GetItem(ctx, in.ItemID)
}

// CreateItem — §5.1 add a work item.
func CreateItem(ctx context.Context, app core.App, cfg *cfgpkg.Config, v schema.DocumentValidator, in *CreateItemInput) (*itemResult, error) {
	rec, err := newEngine(app, cfg, v).CreateItem(ctx, engine.CreateItemInput{
		Title: in.Title, Type: in.Type, Parent: in.Parent, Court: in.Court,
		Pointer: in.Pointer, Body: in.Body, Severity: in.Severity, Priority: in.Priority,
		Actor: actorOf(in.ActorFields),
	})
	if err != nil {
		return nil, err
	}
	return &itemResult{Item: engine.Summarize(rec)}, nil
}

// UpdateItem — §5.1 edit first-class fields. Optionality is signaled by presence, not value
// (see types.go): the optional string fields are already *string, so they pass straight through
// to the engine — nil stays nil (unchanged), a non-nil pointer (including &"") writes verbatim, so
// a present empty string clears the value. Priority keeps the value convention (0 = unchanged).
func UpdateItem(ctx context.Context, app core.App, cfg *cfgpkg.Config, v schema.DocumentValidator, in *UpdateItemInput) (*itemResult, error) {
	up := engine.UpdateItemInput{
		ItemID: in.ItemID, Version: in.Version, Actor: actorOf(in.ActorFields),
		Title: in.Title, Type: in.Type, Court: in.Court, Pointer: in.Pointer,
		Body: in.Body, Severity: in.Severity, Properties: in.Properties, StatusLabel: in.StatusLabel,
	}
	if in.Priority != 0 {
		up.Priority = &in.Priority
	}
	rec, err := newEngine(app, cfg, v).UpdateItem(ctx, up)
	if err != nil {
		return nil, err
	}
	return &itemResult{Item: engine.Summarize(rec)}, nil
}

// TransitionItem — §4.1 the one generic transition path (machine → blocked → claim → gates).
func TransitionItem(ctx context.Context, app core.App, cfg *cfgpkg.Config, v schema.DocumentValidator, in *TransitionItemInput) (*itemResult, error) {
	rec, err := newEngine(app, cfg, v).Transition(ctx, engine.TransitionInput{
		ItemID: in.ItemID, TargetPhase: in.TargetPhase, Version: in.Version,
		Actor: actorOf(in.ActorFields),
	})
	if err != nil {
		return nil, err
	}
	return &itemResult{Item: engine.Summarize(rec)}, nil
}

// BlockItem — §3.2 set the blocked side-state.
func BlockItem(ctx context.Context, app core.App, cfg *cfgpkg.Config, v schema.DocumentValidator, in *BlockItemInput) (*itemResult, error) {
	rec, err := newEngine(app, cfg, v).Block(ctx, in.ItemID, in.Version, actorOf(in.ActorFields), in.Reason)
	if err != nil {
		return nil, err
	}
	return &itemResult{Item: engine.Summarize(rec)}, nil
}

// UnblockItem — §3.2 clear the blocked side-state.
func UnblockItem(ctx context.Context, app core.App, cfg *cfgpkg.Config, v schema.DocumentValidator, in *UnblockItemInput) (*itemResult, error) {
	rec, err := newEngine(app, cfg, v).Unblock(ctx, in.ItemID, in.Version, actorOf(in.ActorFields), in.Reason)
	if err != nil {
		return nil, err
	}
	return &itemResult{Item: engine.Summarize(rec)}, nil
}

// AddNote — §3.7 attach a phase-scoped note.
func AddNote(ctx context.Context, app core.App, cfg *cfgpkg.Config, v schema.DocumentValidator, in *AddNoteInput) (*noteResult, error) {
	rec, err := newEngine(app, cfg, v).AddNote(ctx, in.ItemID, in.Key, in.Body, actorOf(in.ActorFields))
	if err != nil {
		return nil, err
	}
	return &noteResult{Note: engine.NoteRow{
		Phase: rec.GetString("phase"), Key: rec.GetString("key"), Body: rec.GetString("body"),
		Actor: rec.GetString("actor"), ActorKind: rec.GetString("actor_kind"),
		// At mirrors GetItem's note rows (review finding: it was left empty here).
		At: rec.GetDateTime("created").Time().UTC().Format(time.RFC3339),
	}}, nil
}

// LinkItems — §3.4 create a typed dependency edge.
func LinkItems(ctx context.Context, app core.App, cfg *cfgpkg.Config, v schema.DocumentValidator, in *LinkItemsInput) (*edgeResult, error) {
	rec, err := newEngine(app, cfg, v).Link(ctx, engine.LinkInput{
		From: in.From, To: in.To, Kind: in.Kind, UnblockAt: in.UnblockAt, Cascade: in.Cascade,
		Actor: actorOf(in.ActorFields),
	})
	if err != nil {
		return nil, err
	}
	return &edgeResult{Edge: engine.DependencyRow{
		From: rec.GetString("from"), To: rec.GetString("to"), Kind: rec.GetString("kind"),
		UnblockAt: rec.GetString("unblock_at"), Cascade: rec.GetString("cascade"),
	}}, nil
}

// ClaimItem — §3.6 claim with TTL.
func ClaimItem(ctx context.Context, app core.App, cfg *cfgpkg.Config, v schema.DocumentValidator, in *ClaimItemInput) (*itemResult, error) {
	rec, err := newEngine(app, cfg, v).Claim(ctx, in.ItemID, in.Version, actorOf(in.ActorFields))
	if err != nil {
		return nil, err
	}
	return &itemResult{Item: engine.Summarize(rec)}, nil
}

// ReleaseItem — §3.6 release a claim.
func ReleaseItem(ctx context.Context, app core.App, cfg *cfgpkg.Config, v schema.DocumentValidator, in *ReleaseItemInput) (*itemResult, error) {
	rec, err := newEngine(app, cfg, v).Release(ctx, in.ItemID, in.Version, actorOf(in.ActorFields))
	if err != nil {
		return nil, err
	}
	return &itemResult{Item: engine.Summarize(rec)}, nil
}

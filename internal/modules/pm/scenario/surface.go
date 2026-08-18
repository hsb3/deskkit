package scenario

import (
	"context"

	pbcore "github.com/pocketbase/pocketbase/core"

	"github.com/hsb3/deskkit/internal/core/config"
	"github.com/hsb3/deskkit/internal/core/schema"
	"github.com/hsb3/deskkit/internal/modules/pm/engine"
	pmtools "github.com/hsb3/deskkit/internal/modules/pm/tools"
)

// Surface is the set of PM mutations a scenario step drives. Two implementations exist so the
// harness can prove thin-surface parity (spec §10.10): every operation must produce the same
// observable outcome whether driven through the engine core directly or through the model/CLI
// tool bodies that wrap it. NEITHER implementation reimplements transition/gate/cascade logic —
// they are the same one core, reached two ways (R4.1).
type Surface interface {
	Name() string
	Create(ctx context.Context, in CreateArgs) (id string, err error)
	Transition(ctx context.Context, id, target string, version int, a engine.Actor) error
	Block(ctx context.Context, id string, version int, a engine.Actor, reason string) error
	Unblock(ctx context.Context, id string, version int, a engine.Actor, reason string) error
	Claim(ctx context.Context, id string, version int, a engine.Actor) error
	Release(ctx context.Context, id string, version int, a engine.Actor) error
	Link(ctx context.Context, from, to, kind, unblockAt, cascade string, a engine.Actor) error
	AddNote(ctx context.Context, id, key, body string, a engine.Actor) error
	Update(ctx context.Context, id string, version int, severity string, priority int, a engine.Actor) error
}

// CreateArgs mirrors the create inputs both surfaces accept.
type CreateArgs struct {
	Title, Type, Parent, Court, Pointer, Severity string
	Priority                                      int
	Actor                                         engine.Actor
}

// --- engine surface: the core the CLI's read/print path and the TUI views call directly ---

type engineSurface struct{ eng *engine.Engine }

func (engineSurface) Name() string { return "engine" }

func (s engineSurface) Create(ctx context.Context, in CreateArgs) (string, error) {
	rec, err := s.eng.CreateItem(ctx, engine.CreateItemInput{
		Title: in.Title, Type: in.Type, Parent: in.Parent, Court: in.Court,
		Pointer: in.Pointer, Severity: in.Severity, Priority: in.Priority, Actor: in.Actor,
	})
	if err != nil {
		return "", err
	}
	return rec.Id, nil
}

func (s engineSurface) Transition(ctx context.Context, id, target string, version int, a engine.Actor) error {
	_, err := s.eng.Transition(ctx, engine.TransitionInput{ItemID: id, TargetPhase: target, Version: version, Actor: a})
	return err
}
func (s engineSurface) Block(ctx context.Context, id string, version int, a engine.Actor, reason string) error {
	_, err := s.eng.Block(ctx, id, version, a, reason)
	return err
}
func (s engineSurface) Unblock(ctx context.Context, id string, version int, a engine.Actor, reason string) error {
	_, err := s.eng.Unblock(ctx, id, version, a, reason)
	return err
}
func (s engineSurface) Claim(ctx context.Context, id string, version int, a engine.Actor) error {
	_, err := s.eng.Claim(ctx, id, version, a)
	return err
}
func (s engineSurface) Release(ctx context.Context, id string, version int, a engine.Actor) error {
	_, err := s.eng.Release(ctx, id, version, a)
	return err
}
func (s engineSurface) Link(ctx context.Context, from, to, kind, unblockAt, cascade string, a engine.Actor) error {
	_, err := s.eng.Link(ctx, engine.LinkInput{From: from, To: to, Kind: kind, UnblockAt: unblockAt, Cascade: cascade, Actor: a})
	return err
}
func (s engineSurface) AddNote(ctx context.Context, id, key, body string, a engine.Actor) error {
	_, err := s.eng.AddNote(ctx, id, key, body, a)
	return err
}
func (s engineSurface) Update(ctx context.Context, id string, version int, severity string, priority int, a engine.Actor) error {
	// Mirror the tools body's "empty/zero = unchanged" semantics so the two surfaces are
	// behaviourally identical field-for-field (thin-surface parity, §10.10).
	in := engine.UpdateItemInput{ItemID: id, Version: version, Actor: a}
	if severity != "" {
		in.Severity = &severity
	}
	if priority != 0 {
		in.Priority = &priority
	}
	_, err := s.eng.UpdateItem(ctx, in)
	return err
}

// --- tools surface: the model-facing / CLI tool bodies (tools.*) over the same engine ---

type toolsSurface struct {
	app pbcore.App
	cfg *config.Config
	val schema.DocumentValidator
}

func (toolsSurface) Name() string { return "tools" }

func af(a engine.Actor) pmtools.ActorFields {
	return pmtools.ActorFields{Actor: a.Name, ActorKind: a.Kind, DelegationParent: a.DelegationParent}
}

func (s toolsSurface) Create(ctx context.Context, in CreateArgs) (string, error) {
	r, err := pmtools.CreateItem(ctx, s.app, s.cfg, s.val, &pmtools.CreateItemInput{
		Title: in.Title, Type: in.Type, Parent: in.Parent, Court: in.Court,
		Pointer: in.Pointer, Severity: in.Severity, Priority: in.Priority, ActorFields: af(in.Actor),
	})
	if err != nil {
		return "", err
	}
	return r.Item.ID, nil
}

func (s toolsSurface) Transition(ctx context.Context, id, target string, version int, a engine.Actor) error {
	_, err := pmtools.TransitionItem(ctx, s.app, s.cfg, s.val, &pmtools.TransitionItemInput{
		ItemID: id, TargetPhase: target, Version: version, ActorFields: af(a),
	})
	return err
}
func (s toolsSurface) Block(ctx context.Context, id string, version int, a engine.Actor, reason string) error {
	_, err := pmtools.BlockItem(ctx, s.app, s.cfg, s.val, &pmtools.BlockItemInput{ItemID: id, Version: version, Reason: reason, ActorFields: af(a)})
	return err
}
func (s toolsSurface) Unblock(ctx context.Context, id string, version int, a engine.Actor, reason string) error {
	_, err := pmtools.UnblockItem(ctx, s.app, s.cfg, s.val, &pmtools.UnblockItemInput{ItemID: id, Version: version, Reason: reason, ActorFields: af(a)})
	return err
}
func (s toolsSurface) Claim(ctx context.Context, id string, version int, a engine.Actor) error {
	_, err := pmtools.ClaimItem(ctx, s.app, s.cfg, s.val, &pmtools.ClaimItemInput{ItemID: id, Version: version, ActorFields: af(a)})
	return err
}
func (s toolsSurface) Release(ctx context.Context, id string, version int, a engine.Actor) error {
	_, err := pmtools.ReleaseItem(ctx, s.app, s.cfg, s.val, &pmtools.ReleaseItemInput{ItemID: id, Version: version, ActorFields: af(a)})
	return err
}
func (s toolsSurface) Link(ctx context.Context, from, to, kind, unblockAt, cascade string, a engine.Actor) error {
	_, err := pmtools.LinkItems(ctx, s.app, s.cfg, s.val, &pmtools.LinkItemsInput{From: from, To: to, Kind: kind, UnblockAt: unblockAt, Cascade: cascade, ActorFields: af(a)})
	return err
}
func (s toolsSurface) AddNote(ctx context.Context, id, key, body string, a engine.Actor) error {
	_, err := pmtools.AddNote(ctx, s.app, s.cfg, s.val, &pmtools.AddNoteInput{ItemID: id, Key: key, Body: body, ActorFields: af(a)})
	return err
}
func (s toolsSurface) Update(ctx context.Context, id string, version int, severity string, priority int, a engine.Actor) error {
	// The optional string fields are now *string (presence, not value, signals a write), so guard
	// severity into a pointer — mirroring the engineSurface.Update just above so the two surfaces
	// stay behaviourally identical field-for-field (thin-surface parity, §10.10).
	in := pmtools.UpdateItemInput{ItemID: id, Version: version, Priority: priority, ActorFields: af(a)}
	if severity != "" {
		in.Severity = &severity
	}
	_, err := pmtools.UpdateItem(ctx, s.app, s.cfg, s.val, &in)
	return err
}

// Package scenario is a light, declarative harness for validating that the PM artifacts and
// workflows behave sensibly end-to-end (foreman amendment to spec §10, owner-directed): a
// scenario is a named, ordered sequence of PM operations with an expected outcome after each
// step. Every step runs THROUGH THE REAL ENGINE (via a Surface) — the harness is a
// test/observation layer, never a reimplementation of transition, gate, or cascade logic.
//
// Two design properties matter:
//
//   - Runner/fixtures separation. This file is the engine-agnostic runner; the starter
//     scenarios live in fixtures.go. The runner takes any []Step, so it is reusable by D8,
//     which will replay THIS desk's real thread data (imported into a scratch store) through
//     the same runner — see Runner.Bind, which pre-seeds imported item keys so a scenario can
//     reference items the importer created rather than only ones a Create step makes.
//   - Surface parity. The runner drives an injected Surface (engine or tools; surface.go), so
//     the same scenario proves the engine core and the model/CLI tool bodies agree (§10.10).
//
// Document gates are exercised through a controllable in-memory validator (docStub): a set_doc
// step flips a pointer's verdict between "missing", a wrong status, and satisfied, so a gate can
// be observed refusing-then-admitting exactly as a real librarian verdict would drive it — with
// no filesystem or librarian dependency (the narrow seam, §2.5).
package scenario

import (
	"context"
	"fmt"
	"strings"

	pbcore "github.com/pocketbase/pocketbase/core"

	"github.com/hsb3/deskkit/internal/core/config"
	"github.com/hsb3/deskkit/internal/core/schema"
	"github.com/hsb3/deskkit/internal/modules/pm/engine"
)

// Op is the operation a step performs.
type Op string

const (
	Create     Op = "create"
	Transition Op = "transition"
	Block      Op = "block"
	Unblock    Op = "unblock"
	Claim      Op = "claim"
	Release    Op = "release"
	Link       Op = "link"
	AddNote    Op = "add_note"
	Update     Op = "update"
	SetDoc     Op = "set_doc" // control the harness validator's verdict for a pointer
)

// Actor re-exports the engine actor so fixtures read cleanly.
type Actor = engine.Actor

// Step is one operation plus the outcome it must produce. Only the fields an Op uses are read.
type Step struct {
	Name string
	Op   Op

	// Item is the acting item's scenario key. A Create step binds it to the new record id;
	// every other item op resolves it to a bound id.
	Item string

	// create
	Title, Type, Court, Pointer, Severity string
	Priority                              int
	Parent                                string // parent scenario key

	// transition
	To string

	// block / unblock
	Reason string

	// actor + optional stale-version injection (concurrency cases)
	Actor        Actor
	StaleVersion *int

	// link (From/LinkTo are scenario keys)
	From, LinkTo, Kind, UnblockAt, Cascade string

	// add_note
	NoteKey, NoteBody string

	// update
	UpdSeverity string
	UpdPriority int

	// set_doc: define Pointer's verdict. Missing overrides Status/Valid ("does not exist").
	DocStatus  string
	DocValid   bool
	DocMissing bool

	Expect Expect
}

// Expect is the asserted outcome of a step.
type Expect struct {
	Refused         bool     // the op must return an engine refusal
	RefusalContains string   // the refusal message must contain this substring
	Phase           string   // the acting item's resulting phase (checked when set)
	StatusLabel     string   // the acting item's resulting status_label (checked when set)
	Blocked         *bool    // the acting item's blocked flag (checked when set)
	AuditEvent      string   // this event must appear in the acting item's audit trail
	AutoUnblocked   []string // these scenario keys must be blocked==false after the step
	StillBlocked    []string // these scenario keys must be blocked==true after the step
}

// Scenario is a named, ordered sequence of steps.
type Scenario struct {
	Name  string
	Steps []Step
}

// Runner drives one scenario over one store through a chosen Surface. It resolves scenario
// keys to record ids, reads live versions (the CLI's read-then-write convenience, R2.6), and
// owns the controllable document validator the gate steps flip.
type Runner struct {
	app     pbcore.App
	cfg     *config.Config
	val     *docStub
	surface Surface
	ids     map[string]string // scenario key → record id
}

// NewEngineRunner builds a runner that drives the engine core directly.
func NewEngineRunner(app pbcore.App, cfg *config.Config) *Runner {
	val := newDocStub()
	return &Runner{
		app: app, cfg: cfg, val: val, ids: map[string]string{},
		surface: engineSurface{eng: &engine.Engine{App: app, Cfg: cfg, Validator: val}},
	}
}

// NewToolsRunner builds a runner that drives the model/CLI tool bodies (same engine underneath).
func NewToolsRunner(app pbcore.App, cfg *config.Config) *Runner {
	val := newDocStub()
	return &Runner{
		app: app, cfg: cfg, val: val, ids: map[string]string{},
		surface: toolsSurface{app: app, cfg: cfg, val: val},
	}
}

// Bind pre-seeds a scenario key → record id mapping. D8 calls this for every item the importer
// created (importer.Result.IDs), so a scenario can operate on imported thread data without a
// Create step for each.
func (r *Runner) Bind(key, id string) { r.ids[key] = id }

// SurfaceName reports which surface this runner drives (for test messages).
func (r *Runner) SurfaceName() string { return r.surface.Name() }

// Run executes every step in order, asserting each expectation. The first failing expectation
// returns an error naming the scenario, the step, and what diverged.
func (r *Runner) Run(ctx context.Context, s Scenario) error {
	for i, step := range s.Steps {
		if err := r.runStep(ctx, s.Name, i, step); err != nil {
			return err
		}
	}
	return nil
}

func (r *Runner) runStep(ctx context.Context, scenario string, idx int, step Step) error {
	where := fmt.Sprintf("scenario %q step %d (%s/%s)", scenario, idx+1, step.Op, step.Name)

	if step.Op == SetDoc {
		if step.DocMissing {
			r.val.remove(step.Pointer)
		} else {
			r.val.set(step.Pointer, step.DocStatus, step.DocValid)
		}
		return nil
	}

	if step.Op == Create {
		id, err := r.surface.Create(ctx, CreateArgs{
			Title: step.Title, Type: step.Type, Parent: r.ids[step.Parent], Court: step.Court,
			Pointer: step.Pointer, Severity: step.Severity, Priority: step.Priority, Actor: r.actor(step),
		})
		if err != nil {
			return fmt.Errorf("%s: create failed: %w", where, err)
		}
		r.ids[step.Item] = id
		return r.checkState(ctx, where, step)
	}

	// A mutating op on an existing item: resolve the acting id + version, then dispatch.
	err := r.dispatch(ctx, step)
	if step.Expect.Refused {
		if err == nil {
			return fmt.Errorf("%s: expected a refusal, got success", where)
		}
		if !engine.IsRefusal(err) {
			return fmt.Errorf("%s: expected a refusal, got a non-refusal error: %v", where, err)
		}
		if step.Expect.RefusalContains != "" && !strings.Contains(err.Error(), step.Expect.RefusalContains) {
			return fmt.Errorf("%s: refusal %q must contain %q", where, err.Error(), step.Expect.RefusalContains)
		}
		return r.checkState(ctx, where, step) // phase/blocked must be unchanged after a refusal
	}
	if err != nil {
		return fmt.Errorf("%s: unexpected error: %w", where, err)
	}
	return r.checkState(ctx, where, step)
}

// dispatch resolves the acting item's live version (unless a stale version is injected) and
// calls the matching Surface op.
func (r *Runner) dispatch(ctx context.Context, step Step) error {
	a := r.actor(step)
	switch step.Op {
	case Link:
		from, fok := r.ids[step.From]
		to, tok := r.ids[step.LinkTo]
		if !fok {
			return fmt.Errorf("scenario: unbound key %q (From)", step.From)
		}
		if !tok {
			return fmt.Errorf("scenario: unbound key %q (LinkTo)", step.LinkTo)
		}
		return r.surface.Link(ctx, from, to, step.Kind, step.UnblockAt, step.Cascade, a)
	case AddNote:
		id, ok := r.ids[step.Item]
		if !ok {
			return fmt.Errorf("scenario: unbound key %q", step.Item)
		}
		return r.surface.AddNote(ctx, id, step.NoteKey, step.NoteBody, a)
	}
	id, ok := r.ids[step.Item]
	if !ok {
		return fmt.Errorf("scenario: unbound key %q", step.Item)
	}
	version := r.version(id)
	if step.StaleVersion != nil {
		version = *step.StaleVersion
	}
	switch step.Op {
	case Transition:
		return r.surface.Transition(ctx, id, step.To, version, a)
	case Block:
		return r.surface.Block(ctx, id, version, a, step.Reason)
	case Unblock:
		return r.surface.Unblock(ctx, id, version, a, step.Reason)
	case Claim:
		return r.surface.Claim(ctx, id, version, a)
	case Release:
		return r.surface.Release(ctx, id, version, a)
	case Update:
		return r.surface.Update(ctx, id, version, step.UpdSeverity, step.UpdPriority, a)
	}
	return fmt.Errorf("scenario: unknown op %q", step.Op)
}

// actor defaults an unset step actor to a neutral human operator.
func (r *Runner) actor(step Step) engine.Actor {
	a := step.Actor
	if a.Name == "" {
		a.Name, a.Kind = "operator", "human"
	}
	if a.Kind == "" {
		a.Kind = "human"
	}
	return a
}

// version reads an item's current version token (0 when absent).
func (r *Runner) version(id string) int {
	rec, err := r.app.FindRecordById("items", id)
	if err != nil {
		return 0
	}
	return rec.GetInt("version")
}

// checkState asserts the acting item's post-step phase/status_label/blocked and audit event,
// plus any AutoUnblocked / StillBlocked cascade effects on named keys.
func (r *Runner) checkState(ctx context.Context, where string, step Step) error {
	e := step.Expect
	if step.Item != "" && (e.Phase != "" || e.StatusLabel != "" || e.Blocked != nil || e.AuditEvent != "") {
		rec, err := r.app.FindRecordById("items", r.ids[step.Item])
		if err != nil {
			return fmt.Errorf("%s: reload acting item: %w", where, err)
		}
		if e.Phase != "" && rec.GetString("phase") != e.Phase {
			return fmt.Errorf("%s: phase = %q, want %q", where, rec.GetString("phase"), e.Phase)
		}
		if e.StatusLabel != "" && rec.GetString("status_label") != e.StatusLabel {
			return fmt.Errorf("%s: status_label = %q, want %q", where, rec.GetString("status_label"), e.StatusLabel)
		}
		if e.Blocked != nil && rec.GetBool("blocked") != *e.Blocked {
			return fmt.Errorf("%s: blocked = %v, want %v", where, rec.GetBool("blocked"), *e.Blocked)
		}
		if e.AuditEvent != "" {
			if err := r.assertAuditEvent(r.ids[step.Item], e.AuditEvent); err != nil {
				return fmt.Errorf("%s: %w", where, err)
			}
		}
	}
	for _, key := range e.AutoUnblocked {
		if err := r.assertBlocked(key, false); err != nil {
			return fmt.Errorf("%s: %w", where, err)
		}
	}
	for _, key := range e.StillBlocked {
		if err := r.assertBlocked(key, true); err != nil {
			return fmt.Errorf("%s: %w", where, err)
		}
	}
	return nil
}

func (r *Runner) assertBlocked(key string, want bool) error {
	rec, err := r.app.FindRecordById("items", r.ids[key])
	if err != nil {
		return fmt.Errorf("reload %q: %w", key, err)
	}
	if rec.GetBool("blocked") != want {
		return fmt.Errorf("item %q blocked = %v, want %v", key, rec.GetBool("blocked"), want)
	}
	return nil
}

func (r *Runner) assertAuditEvent(id, event string) error {
	rows, err := r.app.FindRecordsByFilter("transitions",
		"item = {:i} && event = {:e}", "", 1, 0, map[string]any{"i": id, "e": event})
	if err != nil {
		return fmt.Errorf("audit query: %w", err)
	}
	if len(rows) == 0 {
		return fmt.Errorf("expected an audit row with event %q, found none", event)
	}
	return nil
}

// --- controllable document validator (the gate seam, §2.5) ---

// docStub is the harness's DocumentValidator: a pointer with no set verdict "does not exist",
// mirroring a real librarian verdict for an absent document. set/remove let a set_doc step
// flip a gate open or closed between steps.
type docStub struct{ verdicts map[string]schema.Verdict }

func newDocStub() *docStub { return &docStub{verdicts: map[string]schema.Verdict{}} }

func (d *docStub) set(pointer, status string, valid bool) {
	// Satisfied is intentionally left zero: Verdict recomputes it per-requirement below (it
	// depends on the gate's RequiredStatus, which set() cannot know), so storing a value here
	// would be dead and misleading.
	d.verdicts[pointer] = schema.Verdict{
		Exists: true, FrontmatterValid: valid, Status: status,
	}
}

func (d *docStub) remove(pointer string) { delete(d.verdicts, pointer) }

// Verdict answers per the gate engine's contract (spec §2.5): compute Satisfied against the
// requirement's RequiredStatus so a "wrong status" doc refuses and names actual-vs-required,
// exactly as the librarian's real validator does.
func (d *docStub) Verdict(_ context.Context, pointer string, req schema.ArtifactRequirement) (schema.Verdict, error) {
	v, ok := d.verdicts[pointer]
	if !ok {
		return schema.Verdict{Missing: []string{
			`required document (type=` + req.Type + `, status=` + req.RequiredStatus + `) at "` + pointer + `" does not exist`,
		}}, nil
	}
	satisfied := v.Exists && v.FrontmatterValid && (req.RequiredStatus == "" || v.Status == req.RequiredStatus)
	out := schema.Verdict{Exists: v.Exists, FrontmatterValid: v.FrontmatterValid, Status: v.Status, Satisfied: satisfied}
	if !satisfied {
		switch {
		case !v.Exists:
			out.Missing = []string{`required document (type=` + req.Type + `, status=` + req.RequiredStatus + `) at "` + pointer + `" does not exist`}
		case !v.FrontmatterValid:
			out.Missing = []string{`required document (type=` + req.Type + `) at "` + pointer + `" has invalid frontmatter`}
		default:
			out.Missing = []string{`required document (type=` + req.Type + `, status=` + req.RequiredStatus + `) at "` + pointer + `" is at status "` + v.Status + `", needs "` + req.RequiredStatus + `"`}
		}
	}
	return out, nil
}

package engine

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"

	"github.com/hsb3/deskkit/internal/core/config"
	"github.com/hsb3/deskkit/internal/core/schema"
	"github.com/hsb3/deskkit/internal/modules/pm/collections"
	"github.com/hsb3/deskkit/internal/modules/pm/gates"
)

// stubValidator drives the gate engine with no librarian internals (test lane §10.5: the PM
// module needs only the core/schema seam).
type stubValidator struct {
	verdicts map[string]schema.Verdict
}

func (s *stubValidator) Verdict(_ context.Context, pointer string, req schema.ArtifactRequirement) (schema.Verdict, error) {
	if v, ok := s.verdicts[pointer]; ok {
		return v, nil
	}
	return schema.Verdict{Missing: []string{
		`required document (type=` + req.Type + `, status=` + req.RequiredStatus + `) at "` + pointer + `" does not exist`}}, nil
}

// newEngine boots a fresh test app, applies the pm migrations DIRECTLY (their Up funcs — no
// global registration, so this cannot pollute other tests' migration lists), and returns an
// engine over a stub validator.
func newEngine(t *testing.T, sv schema.DocumentValidator) *Engine {
	t.Helper()
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("tests.NewTestApp: %v", err)
	}
	t.Cleanup(app.Cleanup)
	for _, mig := range collections.Migrations() {
		if mig.SelfRegistered || mig.Up == nil {
			t.Fatalf("pm migration %q must be programmatic with a real Up (spec §2.8a)", mig.Basename)
		}
		if err := mig.Up(app); err != nil {
			t.Fatalf("apply pm migration %q: %v", mig.Basename, err)
		}
	}
	return &Engine{
		App:       app,
		Cfg:       &config.Config{DeskRoot: t.TempDir(), DeskName: "test-desk", PMClaimTTL: 30 * time.Minute},
		Validator: sv,
	}
}

var human = Actor{Name: "owner", Kind: "human"}

func mustCreate(t *testing.T, e *Engine, in CreateItemInput) *core.Record {
	t.Helper()
	rec, err := e.CreateItem(context.Background(), in)
	if err != nil {
		t.Fatalf("CreateItem(%+v): %v", in, err)
	}
	return rec
}

func mustTransition(t *testing.T, e *Engine, rec *core.Record, target string) *core.Record {
	t.Helper()
	out, err := e.Transition(context.Background(), TransitionInput{
		ItemID: rec.Id, TargetPhase: target, Version: rec.GetInt("version"), Actor: human,
	})
	if err != nil {
		t.Fatalf("Transition(%s -> %s): %v", rec.Id, target, err)
	}
	return out
}

func transitionErr(e *Engine, rec *core.Record, target string) error {
	_, err := e.Transition(context.Background(), TransitionInput{
		ItemID: rec.Id, TargetPhase: target, Version: rec.GetInt("version"), Actor: human,
	})
	return err
}

// --- §10.1 gate red-ability ---

// TestGateRefusedThenSatisfied drives the default decision gate (review->terminal needs an
// accepted decision doc) through absent → wrong-status → satisfied, asserting each refusal
// names exactly what is missing and the audit records gate_refused rows.
func TestGateRefusedThenSatisfied(t *testing.T) {
	sv := &stubValidator{verdicts: map[string]schema.Verdict{}}
	e := newEngine(t, sv)
	ptr := "_structure/decisions/0001-x.md"
	item := mustCreate(t, e, CreateItemInput{Title: "rule the thing", Type: "decision", Pointer: ptr})
	item = mustTransition(t, e, item, "work")
	item = mustTransition(t, e, item, "review")

	// Document absent: refused, names the pointer.
	err := transitionErr(e, item, "terminal")
	if !IsRefusal(err) || !strings.Contains(err.Error(), ptr) || !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("absent doc should refuse naming the pointer, got %v", err)
	}

	// Wrong status: refused, names actual vs required.
	sv.verdicts[ptr] = schema.Verdict{
		Exists: true, FrontmatterValid: true, Status: "proposed",
		Missing: []string{`required document (type=decision, status=accepted) at "` + ptr + `" is at status "proposed", needs "accepted"`},
	}
	err = transitionErr(e, item, "terminal")
	if !IsRefusal(err) || !strings.Contains(err.Error(), `needs "accepted"`) {
		t.Fatalf("wrong-status doc should refuse naming the statuses, got %v", err)
	}

	// Refusals are observable: gate_refused rows exist (§4.1).
	refused, ferr := e.App.FindRecordsByFilter("transitions",
		"item = {:i} && event = 'gate_refused'", "", 0, 0, map[string]any{"i": item.Id})
	if ferr != nil || len(refused) != 2 {
		t.Fatalf("expected 2 gate_refused audit rows, got %d (%v)", len(refused), ferr)
	}

	// Phase unchanged by refusals.
	item, _ = e.loadItem(item.Id)
	if item.GetString("phase") != "review" {
		t.Fatalf("refused item must stay in review, got %s", item.GetString("phase"))
	}

	// Satisfy the document: the same advance succeeds.
	sv.verdicts[ptr] = schema.Verdict{Exists: true, FrontmatterValid: true, Status: "accepted", Satisfied: true}
	item = mustTransition(t, e, item, "terminal")
	if item.GetString("phase") != "terminal" || item.GetString("status_label") != "done" {
		t.Fatalf("satisfied advance should land terminal/done, got %s/%s",
			item.GetString("phase"), item.GetString("status_label"))
	}
}

// TestGateFailsClosedWithoutValidator: no DocumentValidator registered => documented gates
// refuse naming the absence (§2.5).
func TestGateFailsClosedWithoutValidator(t *testing.T) {
	e := newEngine(t, nil)
	item := mustCreate(t, e, CreateItemInput{Title: "x", Type: "decision", Pointer: "d.md"})
	item = mustTransition(t, e, item, "work")
	item = mustTransition(t, e, item, "review")
	err := transitionErr(e, item, "terminal")
	if !IsRefusal(err) || !strings.Contains(err.Error(), "no document validator") {
		t.Fatalf("documented gate with no validator must fail closed, got %v", err)
	}
}

// TestUngatedTypeAdvancesFreely: a type with no bound rules passes gates trivially, and
// demote/reopen are ungated by default (§3.2, §4.1).
func TestUngatedTypeAdvancesFreely(t *testing.T) {
	e := newEngine(t, nil) // even with no validator: ungated edges never consult it
	item := mustCreate(t, e, CreateItemInput{Title: "free", Type: "analysis"})
	item = mustTransition(t, e, item, "work")
	item = mustTransition(t, e, item, "review")
	item = mustTransition(t, e, item, "terminal")
	item = mustTransition(t, e, item, "work")  // reopen: ungated by default
	item = mustTransition(t, e, item, "queue") // demote: ungated by default
	if item.GetString("phase") != "queue" {
		t.Fatalf("expected queue after reopen+demote, got %s", item.GetString("phase"))
	}
}

// TestCreateItemRejectsUnknownType: CreateItem hard-refuses an items.type outside the
// schema-v1 vocabulary (ADR 0012), closing the one write path that previously accepted any
// string unchecked — the identical schema.Vocab().KnownType check gates.ParseRules already
// applies to desk_config.rules. An absent type stays legal (the deliberate empty-type-passes
// scope call: `type` remains optional across every caller-facing shape), and every known
// vocabulary type is still accepted.
func TestCreateItemRejectsUnknownType(t *testing.T) {
	t.Run("unknown type is refused and creates no row", func(t *testing.T) {
		e := newEngine(t, nil)
		before, err := e.deskItems()
		if err != nil {
			t.Fatalf("deskItems: %v", err)
		}

		_, err = e.CreateItem(context.Background(), CreateItemInput{Title: "x", Type: "no-such-type"})
		if err == nil {
			t.Fatal("expected a refusal for an unknown item type, got nil error")
		}
		if !IsRefusal(err) {
			t.Fatalf("expected a Refusal, got %T: %v", err, err)
		}
		if !strings.Contains(err.Error(), "no-such-type") {
			t.Fatalf("refusal should name the offending type, got %v", err)
		}
		if !strings.Contains(err.Error(), "analysis, ") {
			t.Fatalf("refusal should list the known types comma-separated, got %v", err)
		}
		if !strings.Contains(err.Error(), "doctypes.yaml") {
			t.Fatalf("refusal should point at the vocabulary source, got %v", err)
		}

		after, err := e.deskItems()
		if err != nil {
			t.Fatalf("deskItems: %v", err)
		}
		if len(after) != len(before) {
			t.Fatalf("expected no items row created, had %d before and %d after", len(before), len(after))
		}
	})

	t.Run("absent type stays legal", func(t *testing.T) {
		e := newEngine(t, nil)
		before, err := e.deskItems()
		if err != nil {
			t.Fatalf("deskItems: %v", err)
		}
		mustCreate(t, e, CreateItemInput{Title: "x"}) // no Type set
		after, err := e.deskItems()
		if err != nil {
			t.Fatalf("deskItems: %v", err)
		}
		if len(after) != len(before)+1 {
			t.Fatalf("expected exactly one item row created, had %d before and %d after", len(before), len(after))
		}
	})

	t.Run("every known vocabulary type is still accepted", func(t *testing.T) {
		vocab, err := schema.Vocab()
		if err != nil {
			t.Fatalf("schema.Vocab(): %v", err)
		}
		e := newEngine(t, nil)
		for _, typ := range vocab.TypeNames() {
			typ := typ
			t.Run(typ, func(t *testing.T) {
				mustCreate(t, e, CreateItemInput{Title: "x", Type: typ})
			})
		}
	})
}

// TestEmptyTypeCannotSkipDocumentGate (engine policy decision): an untyped item
// must be refused — not silently waved through — on an edge the desk's config gates for at
// least one known item type. Under the shipped default rules (gates/defaults.go), "task" gates
// work->review; an item created with NO type reaches that same edge and, before this fix, would
// advance with no document present at all. Once ANY recognized type
// is assigned (even "analysis", which this desk never gates on this edge), the edge opens up
// again — the fix targets undetectable typelessness, not the assigned type's own gate rules.
func TestEmptyTypeCannotSkipDocumentGate(t *testing.T) {
	e := newEngine(t, nil)
	item := mustCreate(t, e, CreateItemInput{Title: "untyped"}) // no Type set
	// queue->work is not gated for any type under the default rules, so this is unaffected.
	item = mustTransition(t, e, item, "work")

	err := transitionErr(e, item, "review")
	if !IsRefusal(err) || !strings.Contains(err.Error(), "no type set") {
		t.Fatalf("untyped item must be refused on a document-gated edge, got %v", err)
	}
	if reloaded, lerr := e.loadItem(item.Id); lerr != nil || reloaded.GetString("phase") != "work" {
		t.Fatalf("refused transition must not move the item, got %v (err %v)", reloaded, lerr)
	}
	// The refusal is recorded as an observable gate_refused audit row, same as an ordinary
	// gate refusal (§4.1) — this is philosophically the same category of "no" even though no
	// DocumentValidator was consulted.
	rows, rerr := e.App.FindRecordsByFilter("transitions",
		"item = {:i} && event = 'gate_refused'", "", 0, 0, map[string]any{"i": item.Id})
	if rerr != nil {
		t.Fatal(rerr)
	}
	if len(rows) != 1 {
		t.Fatalf("expected exactly one gate_refused audit row, got %d", len(rows))
	}

	// Assigning ANY recognized type — even "analysis", which this desk's default config never
	// gates on work->review — unblocks the edge.
	typ := "analysis"
	item, uerr := e.UpdateItem(context.Background(), UpdateItemInput{
		ItemID: item.Id, Version: item.GetInt("version"), Type: &typ, Actor: human,
	})
	if uerr != nil {
		t.Fatalf("UpdateItem(type=analysis): %v", uerr)
	}
	item = mustTransition(t, e, item, "review")
	if item.GetString("phase") != "review" {
		t.Fatalf("expected review after assigning a type, got %s", item.GetString("phase"))
	}
}

// TestDeskConfigOverridesDefaults: a stored desk_config rules YAML replaces the shipped
// default (here: analysis becomes gated), and an INVALID stored config is a loud error, not a
// silent fallback (§4.2).
func TestDeskConfigOverridesDefaults(t *testing.T) {
	e := newEngine(t, nil)
	col, err := e.App.FindCollectionByNameOrId("desk_config")
	if err != nil {
		t.Fatal(err)
	}
	rec := core.NewRecord(col)
	rec.Set("desk", "test-desk")
	rec.Set("rules", "schema_version: 1\ngates:\n  analysis:\n    \"work->review\":\n      documents: [{ type: analysis, status: approved }]\n")
	if err := e.App.Save(rec); err != nil {
		t.Fatal(err)
	}

	item := mustCreate(t, e, CreateItemInput{Title: "a", Type: "analysis", Pointer: "a.md"})
	item = mustTransition(t, e, item, "work")
	if err := transitionErr(e, item, "review"); !IsRefusal(err) {
		t.Fatalf("desk-configured gate should refuse (no validator => fail closed), got %v", err)
	}

	// Corrupt the stored rules directly (bypassing hooks — one-shot process case): the engine
	// must fail LOUD on read, not silently drop the desk's gates.
	rec.Set("rules", "schema_version: 1\ngates:\n  not-a-type:\n    \"work->review\":\n      documents: [{ type: analysis }]\n")
	if err := e.App.Save(rec); err != nil {
		t.Fatal(err)
	}
	item2 := mustCreate(t, e, CreateItemInput{Title: "b", Type: "analysis"})
	_, terr := e.Transition(context.Background(), TransitionInput{
		ItemID: item2.Id, TargetPhase: "work", Version: item2.GetInt("version"), Actor: human,
	})
	if terr == nil || IsRefusal(terr) || !strings.Contains(terr.Error(), "invalid gate rules") {
		t.Fatalf("invalid stored rules must be a loud error, got %v", terr)
	}
}

// TestTraitCompositionThroughFrontmatter: a trait matching a pointed-doc frontmatter field
// (via the seam's FrontmatterReader) composes its requirement onto the transition (§4.2).
type stubValidatorWithFM struct {
	stubValidator
	fm map[string]map[string]any
}

func (s *stubValidatorWithFM) Frontmatter(_ context.Context, pointer string) (map[string]any, error) {
	if m, ok := s.fm[pointer]; ok {
		return m, nil
	}
	return map[string]any{}, nil
}

func TestTraitCompositionThroughFrontmatter(t *testing.T) {
	sv := &stubValidatorWithFM{
		stubValidator: stubValidator{verdicts: map[string]schema.Verdict{}},
		fm:            map[string]map[string]any{"doc.md": {"governs": "desk-operations"}},
	}
	e := newEngine(t, sv)
	col, _ := e.App.FindCollectionByNameOrId("desk_config")
	rec := core.NewRecord(col)
	rec.Set("desk", "test-desk")
	rec.Set("rules", `schema_version: 1
traits:
  - name: governs-desk-operations
    match: { field: governs, equals: desk-operations }
    on: "review->terminal"
    documents: [{ type: decision, status: accepted }]
`)
	if err := e.App.Save(rec); err != nil {
		t.Fatal(err)
	}

	// analysis has no per-type rule; the trait alone gates it because its pointed doc's
	// frontmatter matches.
	item := mustCreate(t, e, CreateItemInput{Title: "governed", Type: "analysis", Pointer: "doc.md"})
	item = mustTransition(t, e, item, "work")
	item = mustTransition(t, e, item, "review")
	err := transitionErr(e, item, "terminal")
	if !IsRefusal(err) || !strings.Contains(err.Error(), "type=decision") {
		t.Fatalf("matching trait must compose the decision-doc requirement, got %v", err)
	}

	// A non-matching item of the same type stays ungated.
	free := mustCreate(t, e, CreateItemInput{Title: "ungoverned", Type: "analysis", Pointer: "other.md"})
	free = mustTransition(t, e, free, "work")
	free = mustTransition(t, e, free, "review")
	if free = mustTransition(t, e, free, "terminal"); free.GetString("phase") != "terminal" {
		t.Fatal("non-matching trait must not gate")
	}
}

// --- §10.2 (engine half): blocked blocks advance; illegal edges refused before gates ---

func TestBlockedRefusesAdvanceOnly(t *testing.T) {
	e := newEngine(t, nil)
	item := mustCreate(t, e, CreateItemInput{Title: "b", Type: "analysis"})
	item = mustTransition(t, e, item, "work")
	item, err := e.Block(context.Background(), item.Id, item.GetInt("version"), human, "waiting on vendor")
	if err != nil {
		t.Fatal(err)
	}
	if !item.GetBool("blocked") || item.GetString("restore_phase") != "work" {
		t.Fatalf("block must set the flag and record restore_phase, got %v/%s",
			item.GetBool("blocked"), item.GetString("restore_phase"))
	}
	if err := transitionErr(e, item, "review"); !IsRefusal(err) || !strings.Contains(err.Error(), "blocked") {
		t.Fatalf("advance while blocked must refuse, got %v", err)
	}
	// Demote while blocked is legal (§3.2 blocks ADVANCE only).
	item = mustTransition(t, e, item, "queue")
	if item.GetString("phase") != "queue" {
		t.Fatalf("demote while blocked should pass, got %s", item.GetString("phase"))
	}
	item, err = e.Unblock(context.Background(), item.Id, item.GetInt("version"), human, "")
	if err != nil {
		t.Fatal(err)
	}
	if item.GetBool("blocked") {
		t.Fatal("unblock must clear the flag")
	}
}

func TestIllegalEdgeRefusedBeforeGates(t *testing.T) {
	e := newEngine(t, nil)
	item := mustCreate(t, e, CreateItemInput{Title: "skip", Type: "decision", Pointer: "d.md"})
	err := transitionErr(e, item, "terminal") // queue->terminal is not an edge
	if !IsRefusal(err) || !strings.Contains(err.Error(), "no legal transition") {
		t.Fatalf("illegal edge must refuse via the machine, got %v", err)
	}
	// The machine refused BEFORE gates: no gate_refused audit row.
	rows, _ := e.App.FindRecordsByFilter("transitions",
		"item = {:i} && event = 'gate_refused'", "", 0, 0, map[string]any{"i": item.Id})
	if len(rows) != 0 {
		t.Fatal("machine refusals must not be recorded as gate_refused")
	}
}

// --- §10.3 cascade ---

// linkAndAdvanceEnv builds A blocks B with the given cascade/unblock_at.
func linkEnv(t *testing.T, e *Engine, cascade, unblockAt string) (a, b *core.Record) {
	t.Helper()
	a = mustCreate(t, e, CreateItemInput{Title: "A", Type: "analysis"})
	b = mustCreate(t, e, CreateItemInput{Title: "B", Type: "analysis"})
	if _, err := e.Link(context.Background(), LinkInput{
		From: a.Id, To: b.Id, Kind: "blocks", UnblockAt: unblockAt, Cascade: cascade, Actor: human,
	}); err != nil {
		t.Fatalf("Link: %v", err)
	}
	var lerr error
	if b, lerr = e.loadItem(b.Id); lerr != nil {
		t.Fatal(lerr)
	}
	return a, b
}

func TestCascadeAuto(t *testing.T) {
	e := newEngine(t, nil)
	a, b := linkEnv(t, e, "auto", "review")
	if !b.GetBool("blocked") {
		t.Fatal("creating an unsatisfied blocks edge must block the target")
	}
	// A reaches work: not yet the release phase (§10.3: releases AT the named phase, not before).
	a = mustTransition(t, e, a, "work")
	b, _ = e.loadItem(b.Id)
	if !b.GetBool("blocked") {
		t.Fatal("unblock_at=review must not release at work")
	}
	// A reaches review: B auto-clears.
	a = mustTransition(t, e, a, "review")
	b, _ = e.loadItem(b.Id)
	if b.GetBool("blocked") {
		t.Fatal("auto cascade must clear B when A reaches review")
	}
	// One-shot: A regressing does not re-block B.
	a = mustTransition(t, e, a, "work")
	b, _ = e.loadItem(b.Id)
	if b.GetBool("blocked") {
		t.Fatal("auto is one-shot: regression must not re-block")
	}
}

func TestCascadeAutoReopen(t *testing.T) {
	e := newEngine(t, nil)
	a, b := linkEnv(t, e, "auto-reopen", "review")
	a = mustTransition(t, e, a, "work")
	a = mustTransition(t, e, a, "review")
	b, _ = e.loadItem(b.Id)
	if b.GetBool("blocked") {
		t.Fatal("auto-reopen clears like auto when A reaches the release phase")
	}
	// A regresses below review: B re-blocks (standing-workstream semantics).
	a = mustTransition(t, e, a, "work")
	b, _ = e.loadItem(b.Id)
	if !b.GetBool("blocked") {
		t.Fatal("auto-reopen must re-block B when A regresses below the release phase")
	}
}

func TestCascadeManualAndPermanent(t *testing.T) {
	e := newEngine(t, nil)
	a, b := linkEnv(t, e, "manual", "work")
	a = mustTransition(t, e, a, "work")
	b, _ = e.loadItem(b.Id)
	if !b.GetBool("blocked") {
		t.Fatal("manual cascade must not auto-clear; a human/agent clears it explicitly")
	}
	// Explicit unblock works.
	if _, err := e.Unblock(context.Background(), b.Id, b.GetInt("version"), human, "manually cleared"); err != nil {
		t.Fatal(err)
	}

	p, q := linkEnv(t, e, "permanent", "work")
	p = mustTransition(t, e, p, "work")
	p = mustTransition(t, e, p, "review")
	p = mustTransition(t, e, p, "terminal")
	q, _ = e.loadItem(q.Id)
	if !q.GetBool("blocked") {
		t.Fatal("permanent cascade never auto-clears, even at terminal")
	}
	_ = p
}

func TestCascadeMultiBlocker(t *testing.T) {
	e := newEngine(t, nil)
	a1, b := linkEnv(t, e, "auto", "work")
	a2 := mustCreate(t, e, CreateItemInput{Title: "A2", Type: "analysis"})
	if _, err := e.Link(context.Background(), LinkInput{
		From: a2.Id, To: b.Id, Kind: "blocks", UnblockAt: "work", Cascade: "auto", Actor: human,
	}); err != nil {
		t.Fatal(err)
	}
	// First blocker releases; the second is still short: B stays blocked.
	a1 = mustTransition(t, e, a1, "work")
	b, _ = e.loadItem(b.Id)
	if !b.GetBool("blocked") {
		t.Fatal("B must stay blocked while a second blocker is unreleased")
	}
	// Second blocker releases: B clears.
	a2 = mustTransition(t, e, a2, "work")
	b, _ = e.loadItem(b.Id)
	if b.GetBool("blocked") {
		t.Fatal("B must clear once every gating edge is satisfied")
	}
}

func TestLinkIsBlockedByCanonicalizes(t *testing.T) {
	e := newEngine(t, nil)
	a := mustCreate(t, e, CreateItemInput{Title: "blocker", Type: "analysis"})
	b := mustCreate(t, e, CreateItemInput{Title: "blocked", Type: "analysis"})
	// "b is-blocked-by a" must store as "a blocks b".
	edge, err := e.Link(context.Background(), LinkInput{
		From: b.Id, To: a.Id, Kind: "is-blocked-by", UnblockAt: "terminal", Cascade: "auto", Actor: human,
	})
	if err != nil {
		t.Fatal(err)
	}
	if edge.GetString("kind") != "blocks" || edge.GetString("from") != a.Id || edge.GetString("to") != b.Id {
		t.Fatalf("is-blocked-by must canonicalize to the inverse blocks edge, got %s %s->%s",
			edge.GetString("kind"), edge.GetString("from"), edge.GetString("to"))
	}
}

// --- §10.4 concurrency / atomicity ---

// TestTransitionRollsBackOnLaterFailure is the atomicity regression guard for the
// optimistic-concurrency TOCTOU fix: a mutating method's load->version-check->mutate->save->
// audit->cascade sequence must commit or roll back as ONE unit. It forces a failure AFTER the
// phase write (via the txFailpoint seam, standing in for a failing audit/cascade write) and
// asserts the item's phase AND version are fully restored — no half-applied mutation. Without
// the enclosing transaction the phase Save persists while the later step fails, leaving the item
// half-mutated (this test fails); with it, the phase and version roll back (this test passes).
func TestTransitionRollsBackOnLaterFailure(t *testing.T) {
	e := newEngine(t, nil)
	item := mustCreate(t, e, CreateItemInput{Title: "atomic", Type: "analysis"})
	beforePhase := item.GetString("phase")
	beforeVersion := item.GetInt("version")

	txFailpoint = func() error { return errors.New("forced failure after the phase write") }
	t.Cleanup(func() { txFailpoint = nil })

	_, err := e.Transition(context.Background(), TransitionInput{
		ItemID: item.Id, TargetPhase: "work", Version: beforeVersion, Actor: human,
	})
	if err == nil {
		t.Fatal("expected the forced mid-sequence failure to surface as an error")
	}

	reloaded, lerr := e.loadItem(item.Id)
	if lerr != nil {
		t.Fatal(lerr)
	}
	if got := reloaded.GetString("phase"); got != beforePhase {
		t.Fatalf("phase must roll back to %q on a mid-sequence failure, got %q (half-applied write)", beforePhase, got)
	}
	if got := reloaded.GetInt("version"); got != beforeVersion {
		t.Fatalf("version must roll back to %d on a mid-sequence failure, got %d", beforeVersion, got)
	}
	// The advance audit row must not have committed either (whole sequence rolled back).
	rows, _ := e.App.FindRecordsByFilter("transitions",
		"item = {:i} && event = 'advance'", "", 0, 0, map[string]any{"i": item.Id})
	if len(rows) != 0 {
		t.Fatalf("no advance audit row may commit when the transition rolls back, got %d", len(rows))
	}

	// With the failpoint cleared, the same transition succeeds and bumps the version — proving
	// the seam only affects the forced-failure path, not the happy path.
	txFailpoint = nil
	moved := mustTransition(t, e, reloaded, "work")
	if moved.GetString("phase") != "work" || moved.GetInt("version") != beforeVersion+1 {
		t.Fatalf("post-rollback retry must advance and bump once, got %s/%d",
			moved.GetString("phase"), moved.GetInt("version"))
	}
}

func TestVersionMismatchRefused(t *testing.T) {
	e := newEngine(t, nil)
	item := mustCreate(t, e, CreateItemInput{Title: "v", Type: "analysis"})
	_, err := e.Transition(context.Background(), TransitionInput{
		ItemID: item.Id, TargetPhase: "work", Version: 99, Actor: human,
	})
	if !IsRefusal(err) || !strings.Contains(err.Error(), "changed since you read it") {
		t.Fatalf("stale version must refuse, got %v", err)
	}
	// Every successful mutation bumps the token.
	before := item.GetInt("version")
	item = mustTransition(t, e, item, "work")
	if item.GetInt("version") != before+1 {
		t.Fatalf("version must bump on transition: %d -> %d", before, item.GetInt("version"))
	}
}

func TestClaimSemantics(t *testing.T) {
	e := newEngine(t, nil)
	alice := Actor{Name: "agent-alice", Kind: "agent"}
	bob := Actor{Name: "agent-bob", Kind: "agent", DelegationParent: "session-1"}

	item := mustCreate(t, e, CreateItemInput{Title: "c", Type: "analysis"})
	item, err := e.Claim(context.Background(), item.Id, item.GetInt("version"), alice)
	if err != nil {
		t.Fatal(err)
	}
	if item.GetString("claimed_by") != "agent-alice" || item.GetDateTime("claim_expires").Time().IsZero() {
		t.Fatal("claim must set holder + expiry")
	}

	// A live foreign claim blocks bob's advance, claim, and release.
	if _, err := e.Transition(context.Background(), TransitionInput{
		ItemID: item.Id, TargetPhase: "work", Version: item.GetInt("version"), Actor: bob,
	}); !IsRefusal(err) || !strings.Contains(err.Error(), "agent-alice") {
		t.Fatalf("foreign live claim must refuse advance naming the holder, got %v", err)
	}
	if _, err := e.Claim(context.Background(), item.Id, item.GetInt("version"), bob); !IsRefusal(err) {
		t.Fatalf("foreign live claim must refuse re-claim, got %v", err)
	}
	if _, err := e.Release(context.Background(), item.Id, item.GetInt("version"), bob); !IsRefusal(err) {
		t.Fatalf("foreign live claim must refuse release, got %v", err)
	}

	// The holder may advance.
	item, err = e.Transition(context.Background(), TransitionInput{
		ItemID: item.Id, TargetPhase: "work", Version: item.GetInt("version"), Actor: alice,
	})
	if err != nil {
		t.Fatal(err)
	}

	// An EXPIRED claim is treated as free (§3.6).
	item.Set("claim_expires", time.Now().Add(-time.Minute))
	if err := e.App.Save(item); err != nil {
		t.Fatal(err)
	}
	item, _ = e.loadItem(item.Id)
	if _, err := e.Claim(context.Background(), item.Id, item.GetInt("version"), bob); err != nil {
		t.Fatalf("expired claim must be claimable by another actor, got %v", err)
	}
}

// TestClaimTTLFromDeskConfig: desk_config.claim_ttl_minutes overrides the default horizon.
func TestClaimTTLFromDeskConfig(t *testing.T) {
	e := newEngine(t, nil)
	col, _ := e.App.FindCollectionByNameOrId("desk_config")
	rec := core.NewRecord(col)
	rec.Set("desk", "test-desk")
	rec.Set("claim_ttl_minutes", 120)
	if err := e.App.Save(rec); err != nil {
		t.Fatal(err)
	}
	item := mustCreate(t, e, CreateItemInput{Title: "ttl", Type: "analysis"})
	item, err := e.Claim(context.Background(), item.Id, item.GetInt("version"), human)
	if err != nil {
		t.Fatal(err)
	}
	expires := item.GetDateTime("claim_expires").Time()
	if until := time.Until(expires); until < 110*time.Minute || until > 130*time.Minute {
		t.Fatalf("claim_ttl_minutes=120 should land ~2h out, got %v", until)
	}
}

// TestClaimGatesBlockUnblock pins ADR 0020 (a live claim is authoritative over every direct
// mutation) for the Block/Unblock paths: a non-holder is refused naming the holder — even on
// the idempotent no-op paths, so the claim boundary wins — while the holder, and anyone acting
// on an expired or absent claim, still proceed. The cascade/auto (un)block paths are unaffected
// (they call setBlocked directly, not these public methods).
func TestClaimGatesBlockUnblock(t *testing.T) {
	e := newEngine(t, nil)
	alice := Actor{Name: "agent-alice", Kind: "agent"}
	bob := Actor{Name: "agent-bob", Kind: "agent"}

	item := mustCreate(t, e, CreateItemInput{Title: "contested", Type: "analysis"})
	item = mustTransition(t, e, item, "work")
	item, err := e.Claim(context.Background(), item.Id, item.GetInt("version"), alice)
	if err != nil {
		t.Fatal(err)
	}

	// A non-holder's Block is refused, naming the holder; nothing is mutated.
	if _, err := e.Block(context.Background(), item.Id, item.GetInt("version"), bob, "trample"); !IsRefusal(err) || !strings.Contains(err.Error(), "agent-alice") {
		t.Fatalf("foreign live claim must refuse Block naming the holder, got %v", err)
	}
	if item, _ = e.loadItem(item.Id); item.GetBool("blocked") {
		t.Fatal("refused foreign Block must not set the blocked flag")
	}

	// The holder may Block.
	item, err = e.Block(context.Background(), item.Id, item.GetInt("version"), alice, "waiting")
	if err != nil || !item.GetBool("blocked") {
		t.Fatalf("the claim holder must be able to Block, got %v (blocked=%v)", err, item.GetBool("blocked"))
	}

	// The claim owner boundary wins over the idempotent no-op: a non-holder's Block on the
	// ALREADY-blocked item is still refused, not a silent success.
	if _, err := e.Block(context.Background(), item.Id, item.GetInt("version"), bob, "trample again"); !IsRefusal(err) {
		t.Fatalf("foreign Block on an already-blocked item must still refuse, got %v", err)
	}

	// A non-holder's Unblock is refused too; the item stays blocked.
	if _, err := e.Unblock(context.Background(), item.Id, item.GetInt("version"), bob, ""); !IsRefusal(err) || !strings.Contains(err.Error(), "agent-alice") {
		t.Fatalf("foreign live claim must refuse Unblock naming the holder, got %v", err)
	}
	if item, _ = e.loadItem(item.Id); !item.GetBool("blocked") {
		t.Fatal("refused foreign Unblock must leave the item blocked")
	}

	// The holder may Unblock.
	item, err = e.Unblock(context.Background(), item.Id, item.GetInt("version"), alice, "")
	if err != nil || item.GetBool("blocked") {
		t.Fatalf("the claim holder must be able to Unblock, got %v (blocked=%v)", err, item.GetBool("blocked"))
	}

	// Idempotent no-op the other way: a non-holder's Unblock on the NOT-blocked item is still refused.
	if _, err := e.Unblock(context.Background(), item.Id, item.GetInt("version"), bob, ""); !IsRefusal(err) {
		t.Fatalf("foreign Unblock on a not-blocked item must still refuse, got %v", err)
	}

	// An EXPIRED claim frees both paths for anyone (§3.6).
	item.Set("claim_expires", time.Now().Add(-time.Minute))
	if err := e.App.Save(item); err != nil {
		t.Fatal(err)
	}
	item, _ = e.loadItem(item.Id)
	if item, err = e.Block(context.Background(), item.Id, item.GetInt("version"), bob, "expired-free"); err != nil {
		t.Fatalf("expired claim must let a non-holder Block, got %v", err)
	}
	if _, err := e.Unblock(context.Background(), item.Id, item.GetInt("version"), bob, ""); err != nil {
		t.Fatalf("expired claim must let a non-holder Unblock, got %v", err)
	}

	// An ABSENT claim (never claimed) lets anyone Block/Unblock.
	fresh := mustCreate(t, e, CreateItemInput{Title: "unclaimed", Type: "analysis"})
	fresh = mustTransition(t, e, fresh, "work")
	if fresh, err = e.Block(context.Background(), fresh.Id, fresh.GetInt("version"), bob, ""); err != nil {
		t.Fatalf("absent claim must let anyone Block, got %v", err)
	}
	if _, err := e.Unblock(context.Background(), fresh.Id, fresh.GetInt("version"), bob, ""); err != nil {
		t.Fatalf("absent claim must let anyone Unblock, got %v", err)
	}
}

// --- §3.3 label routing ---

func TestSetStatusLabelRoutesThroughMachine(t *testing.T) {
	e := newEngine(t, nil)
	item := mustCreate(t, e, CreateItemInput{Title: "l", Type: "analysis"})
	// Same-phase relabel: plain write.
	item, err := e.SetStatusLabel(context.Background(), item.Id, "next", item.GetInt("version"), human)
	if err != nil || item.GetString("status_label") != "next" || item.GetString("phase") != "queue" {
		t.Fatalf("same-phase relabel: %v %s/%s", err, item.GetString("status_label"), item.GetString("phase"))
	}
	// Cross-phase label: a transition request through the machine.
	item, err = e.SetStatusLabel(context.Background(), item.Id, "active", item.GetInt("version"), human)
	if err != nil || item.GetString("phase") != "work" {
		t.Fatalf("cross-phase label must transition: %v phase=%s", err, item.GetString("phase"))
	}
	// A label that would need an illegal edge refuses (queue<-work is legal; work->terminal "done" is not).
	if _, err := e.SetStatusLabel(context.Background(), item.Id, "done", item.GetInt("version"), human); !IsRefusal(err) {
		t.Fatalf("label implying an illegal edge must refuse, got %v", err)
	}
	// Unknown label refuses.
	if _, err := e.SetStatusLabel(context.Background(), item.Id, "someday", item.GetInt("version"), human); !IsRefusal(err) {
		t.Fatalf("unknown label must refuse, got %v", err)
	}
}

// --- audit trail (§3.6) ---

func TestAuditTrail(t *testing.T) {
	e := newEngine(t, nil)
	agent := Actor{Name: "agent-1", Kind: "agent", DelegationParent: "lead-session"}
	item := mustCreate(t, e, CreateItemInput{Title: "audit", Type: "analysis"})
	if _, err := e.Transition(context.Background(), TransitionInput{
		ItemID: item.Id, TargetPhase: "work", Version: item.GetInt("version"), Actor: agent,
	}); err != nil {
		t.Fatal(err)
	}
	rows, err := e.App.FindRecordsByFilter("transitions",
		"item = {:i} && event = 'advance'", "", 0, 0, map[string]any{"i": item.Id})
	if err != nil || len(rows) != 1 {
		t.Fatalf("expected one advance row, got %d (%v)", len(rows), err)
	}
	row := rows[0]
	if row.GetString("actor") != "agent-1" || row.GetString("actor_kind") != "agent" ||
		row.GetString("delegation_parent") != "lead-session" ||
		row.GetString("from_phase") != "queue" || row.GetString("to_phase") != "work" {
		t.Fatalf("audit row fields wrong: %v", row.PublicExport())
	}
}

// TestReleaseClearsClaim: the holder releasing a live claim actually clears it (holder +
// expiry), and the item is claimable by someone else immediately after.
func TestReleaseClearsClaim(t *testing.T) {
	e := newEngine(t, nil)
	alice := Actor{Name: "agent-alice", Kind: "agent"}
	bob := Actor{Name: "agent-bob", Kind: "agent"}
	item := mustCreate(t, e, CreateItemInput{Title: "r", Type: "analysis"})
	item, err := e.Claim(context.Background(), item.Id, item.GetInt("version"), alice)
	if err != nil {
		t.Fatal(err)
	}
	item, err = e.Release(context.Background(), item.Id, item.GetInt("version"), alice)
	if err != nil {
		t.Fatal(err)
	}
	if item.GetString("claimed_by") != "" || !item.GetDateTime("claim_expires").Time().IsZero() {
		t.Fatalf("release must clear holder + expiry, got %q / %v",
			item.GetString("claimed_by"), item.GetDateTime("claim_expires"))
	}
	if _, err := e.Claim(context.Background(), item.Id, item.GetInt("version"), bob); err != nil {
		t.Fatalf("released item must be claimable, got %v", err)
	}
}

// TestSetStatusLabelVersionSingleBump: a cross-phase label write is ONE logical mutation —
// the caller's version advances exactly once (the Transition's bump; the label pin adds none).
func TestSetStatusLabelVersionSingleBump(t *testing.T) {
	e := newEngine(t, nil)
	item := mustCreate(t, e, CreateItemInput{Title: "vb", Type: "analysis"})
	before := item.GetInt("version")
	item, err := e.SetStatusLabel(context.Background(), item.Id, "active", before, human)
	if err != nil {
		t.Fatal(err)
	}
	if item.GetInt("version") != before+1 {
		t.Fatalf("cross-phase SetStatusLabel must bump exactly once: %d -> %d", before, item.GetInt("version"))
	}
	if item.GetString("status_label") != "active" || item.GetString("phase") != "work" {
		t.Fatalf("label/phase wrong: %s/%s", item.GetString("status_label"), item.GetString("phase"))
	}
}

// TestDeskConfigHookRejectsBadLabels drives the write-time validation the pm module's hooks
// enforce (validateDeskConfigRecord is exercised via gates.ParseLabels here — the hook
// binding itself is a one-line BindFunc over the same function).
func TestDeskConfigHookRejectsBadLabels(t *testing.T) {
	if _, err := gates.ParseLabels(`{"someday": "not-a-phase"}`); err == nil {
		t.Fatal("status_labels with an unknown phase must be refused")
	}
	if _, err := gates.ParseLabels(`{"someday": "queue"}`); err != nil {
		t.Fatalf("valid status_labels must parse: %v", err)
	}
	// And through a stored desk_config row: the engine's loader fails LOUD on read.
	e := newEngine(t, nil)
	col, _ := e.App.FindCollectionByNameOrId("desk_config")
	rec := core.NewRecord(col)
	rec.Set("desk", "test-desk")
	rec.Set("status_labels", `{"someday": "not-a-phase"}`)
	if err := e.App.Save(rec); err != nil { // hooks not bound in this bare test app
		t.Fatal(err)
	}
	item := mustCreate(t, e, CreateItemInput{Title: "h", Type: "analysis"})
	_, terr := e.Transition(context.Background(), TransitionInput{
		ItemID: item.Id, TargetPhase: "work", Version: item.GetInt("version"), Actor: human,
	})
	if terr == nil || IsRefusal(terr) || !strings.Contains(terr.Error(), "status_labels") {
		t.Fatalf("invalid stored status_labels must be a loud engine error, got %v", terr)
	}
}

// TestDemoteGatedWhenConfigBindsOne is the other half of §10.2 (spec §3.2/§4.1): demote and
// reopen edges are ungated BY DEFAULT (TestUngatedTypeAdvancesFreely), but are gated when —
// and only when — the desk's config binds a rule to them. A desk_config binds a gate to the
// demote edge review->work; the demote is then refused while the gate document is missing or
// at the wrong status, passes once satisfied, and the reopen edge (unbound in the same
// config) stays ungated throughout.
func TestDemoteGatedWhenConfigBindsOne(t *testing.T) {
	sv := &stubValidator{verdicts: map[string]schema.Verdict{}}
	e := newEngine(t, sv)
	col, _ := e.App.FindCollectionByNameOrId("desk_config")
	rec := core.NewRecord(col)
	rec.Set("desk", "test-desk")
	rec.Set("rules", `schema_version: 1
gates:
  analysis:
    "review->work":
      documents:
        - type: analysis
          status: approved
          pointer: item
`)
	if err := e.App.Save(rec); err != nil {
		t.Fatal(err)
	}

	ptr := "analyses/walkback-rationale.md"
	item := mustCreate(t, e, CreateItemInput{Title: "gated walkback", Type: "analysis", Pointer: ptr})
	item = mustTransition(t, e, item, "work")
	item = mustTransition(t, e, item, "review") // work->review unbound in this config: ungated

	// Gate doc missing: the BOUND demote edge refuses, naming the pointer.
	err := transitionErr(e, item, "work")
	if !IsRefusal(err) || !strings.Contains(err.Error(), ptr) {
		t.Fatalf("bound demote with missing doc must refuse naming the pointer, got %v", err)
	}

	// Wrong status: still refused, naming actual vs required.
	sv.verdicts[ptr] = schema.Verdict{
		Exists: true, FrontmatterValid: true, Status: "draft",
		Missing: []string{`required document (type=analysis, status=approved) at "` + ptr + `" is at status "draft", needs "approved"`},
	}
	err = transitionErr(e, item, "work")
	if !IsRefusal(err) || !strings.Contains(err.Error(), `needs "approved"`) {
		t.Fatalf("bound demote with wrong-status doc must refuse, got %v", err)
	}

	// The refusals were gate refusals: recorded as gate_refused audit rows, phase unchanged.
	rows, _ := e.App.FindRecordsByFilter("transitions",
		"item = {:i} && event = 'gate_refused'", "", 0, 0, map[string]any{"i": item.Id})
	if len(rows) != 2 {
		t.Fatalf("expected 2 gate_refused rows for the bound demote, got %d", len(rows))
	}
	item, _ = e.loadItem(item.Id)
	if item.GetString("phase") != "review" {
		t.Fatalf("refused demote must leave the item in review, got %s", item.GetString("phase"))
	}

	// Satisfied: the same demote passes.
	sv.verdicts[ptr] = schema.Verdict{Exists: true, FrontmatterValid: true, Status: "approved", Satisfied: true}
	item = mustTransition(t, e, item, "work")
	if item.GetString("phase") != "work" {
		t.Fatalf("satisfied bound demote must pass, got %s", item.GetString("phase"))
	}

	// "Only when the config binds one": the reopen edge is unbound in this config, so after
	// completing (review->terminal is also unbound here), terminal->work passes with the gate
	// doc removed — no gate consulted.
	delete(sv.verdicts, ptr)
	item = mustTransition(t, e, item, "review")
	item = mustTransition(t, e, item, "terminal")
	item = mustTransition(t, e, item, "work") // reopen: unbound => ungated
	if item.GetString("phase") != "work" {
		t.Fatalf("unbound reopen must stay ungated, got %s", item.GetString("phase"))
	}
}

// TestCreateItemBody proves CreateItem persists the inline long-form body (spec §3.1): the
// value is written to the items row and reads back byte-for-byte after reload.
func TestCreateItemBody(t *testing.T) {
	e := newEngine(t, &stubValidator{})
	const body = "## Acceptance\n- items.body stores narrative + acceptance criteria inline\n- reads back verbatim"
	item := mustCreate(t, e, CreateItemInput{Title: "with body", Type: "task", Body: body})

	reloaded, err := e.loadItem(item.Id)
	if err != nil {
		t.Fatalf("loadItem: %v", err)
	}
	if got := reloaded.GetString("body"); got != body {
		t.Fatalf("CreateItem body round-trip: got %q, want %q", got, body)
	}
}

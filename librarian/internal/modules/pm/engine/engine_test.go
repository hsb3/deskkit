package engine

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"

	"github.com/example/pocket-librarian/internal/core/config"
	"github.com/example/pocket-librarian/internal/core/schema"
	"github.com/example/pocket-librarian/internal/modules/pm/collections"
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

// --- §10.4 concurrency ---

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

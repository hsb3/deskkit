package scenario

// TestAdoptionDryRun is the D8 adoption dry-run (spec §8.1 adoption R7.3, §11 D8 row): it seeds
// a SCRATCH store from neutral "desk thread data" and OBSERVES the whole workflow end-to-end
// without ever writing a live desk. It reuses the D6 seam exactly as TestRunner_ReplaysImportedItems
// does — importer.Import → Runner.Bind(res.IDs) → Runner.Run — never a bespoke import/runner path.
//
// The test renders four labeled proofs via t.Log (run with -v to read them):
//
//	PROOF 4  live desk never written — the scratch desk's on-disk store home never materializes.
//	PROOF 1  get_context cold-start  — one call surfaces the freshly-seeded active/blocked/stalled/
//	                                    recent_transitions sets.
//	PROOF 2  gate refused-then-satisfied — a decision item is refused entry to terminal while its
//	                                    decision doc is absent (the refusal NAMES the missing doc),
//	                                    then admitted once the doc is accepted.
//	PROOF 3  dependency auto-unblock  — advancing the blocker to its unblock_at phase auto-releases
//	                                    the dependent.
//
// Every observation is a real assertion: the test fails loudly if any of them is wrong.

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	pbcore "github.com/pocketbase/pocketbase/core"

	"github.com/example/pocket-librarian/internal/core/store"
	"github.com/example/pocket-librarian/internal/modules/pm/engine"
	"github.com/example/pocket-librarian/internal/modules/pm/importer"
)

// adoptionSeed is the neutral manifest standing in for a desk's handoff threads-index / plan
// rows. It is identity-neutral (R5.3): generic keys and titles, no real person/org/desk/repo.
// It carries the three shapes the dry-run must observe:
//   - a parent (plan-onboarding) with three children,
//   - a blocking dep with unblock_at + cascade:auto (task-draft blocks task-review) for the
//     auto-unblock proof,
//   - a gated item (thread-alpha, a decision) whose review->terminal advance needs a decision
//     document that starts absent, for the refusal proof.
func adoptionSeed(decPtr string) importer.Manifest {
	return importer.Manifest{
		Items: []importer.ManifestItem{
			{Key: "plan-onboarding", Title: "Onboarding plan", Type: "analysis", Court: "desk"},
			{Key: "thread-alpha", Title: "Ratify the onboarding approach", Type: "decision", Court: "owner", Parent: "plan-onboarding", Pointer: decPtr},
			{Key: "task-draft", Title: "Draft the onboarding checklist", Type: "analysis", Court: "crew", Parent: "plan-onboarding"},
			{Key: "task-review", Title: "Review the onboarding checklist", Type: "analysis", Court: "crew", Parent: "plan-onboarding"},
		},
		Deps: []importer.ManifestDep{
			{From: "task-draft", To: "task-review", Kind: "blocks", UnblockAt: "review", Cascade: "auto"},
		},
	}
}

func TestAdoptionDryRun(t *testing.T) {
	ctx := context.Background()

	// A neutral scratch-desk name (R5.3): no real desk/person/org. Distinctive enough that its
	// real store home is guaranteed absent on any machine, so PROOF 4's "does not exist" check is
	// meaningful.
	const scratchDesk = "adoption-dryrun-scratch"
	decPtr := "_structure/decisions/0099-adoption-example.md"

	// ─────────────────────────────────────────────────────────────────────────────────────────
	// PROOF 4 — live desk never written.
	// Redirect the XDG data home into the test's temp dir up front, so even if some code path
	// resolved the canonical store location it would land under temp, never the user's real home.
	// Then assert the scratch desk's store home never materializes — the dry-run runs entirely in
	// the in-memory/temp test app; no live PocketBase store dir is created for this desk anywhere
	// a real deskkit binary would put one.
	tmpXDG := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmpXDG)

	scratchStore, err := store.StoreDir(scratchDesk)
	if err != nil {
		t.Fatalf("resolve scratch StoreDir: %v", err)
	}
	if !strings.HasPrefix(scratchStore, tmpXDG) {
		t.Fatalf("scratch store %q is not under the test temp dir %q", scratchStore, tmpXDG)
	}
	// The canonical real-home default, computed independent of the XDG override — this is the path
	// a normally-configured desk would use. It must not exist: a dry-run never touches it.
	realHomeStore := ""
	if home, herr := os.UserHomeDir(); herr == nil {
		realHomeStore = filepath.Join(home, ".local", "share", store.AppDirName, scratchDesk)
	}

	app, cfg := newApp(t)
	cfg.DeskName = scratchDesk // drive the whole dry-run against the neutral scratch desk

	t.Log("=== DRY-RUN PROOF 4: live desk never written ===")
	t.Logf("scratch desk                : %q", scratchDesk)
	t.Logf("XDG data home (temp)        : %s", tmpXDG)
	t.Logf("resolved scratch store home : %s", scratchStore)
	t.Logf("canonical real-home default : %s", realHomeStore)
	// Setup guard: neither store home exists yet (the import below hasn't run). The load-bearing
	// assertion is the post-run re-check registered just below — that is what proves the dry-run
	// itself wrote no live desk. mustNotExist distinguishes "exists" from an unrelated stat error
	// (a bare !os.IsNotExist would mask e.g. a permission error as a "must not exist" failure).
	mustNotExist(t, "scratch store home", scratchStore)
	mustNotExist(t, "canonical real-home store", realHomeStore)
	t.Logf("SETUP OK: neither store home exists before import.")
	// PROOF 4 (post-condition): after the ENTIRE dry-run has run, re-assert neither home ever
	// materialized. Registered now so it runs last, closing the proof against the real operations.
	t.Cleanup(func() {
		mustNotExist(t, "scratch store home (post-run)", scratchStore)
		mustNotExist(t, "canonical real-home store (post-run)", realHomeStore)
		t.Logf("OBSERVED (post-run): neither store home exists — the dry-run wrote no live desk on disk.")
	})

	// ─────────────────────────────────────────────────────────────────────────────────────────
	// SEED — populate the scratch engine via the import seam (the same path §10.8 and D8 use).
	eng := &engine.Engine{App: app, Cfg: cfg}
	res, err := importer.Import(ctx, eng, adoptionSeed(decPtr))
	if err != nil {
		t.Fatalf("import seed: %v", err)
	}
	t.Log("=== DRY-RUN SEED: scratch store seeded from desk thread data ===")
	t.Logf("created items=%d deps=%d (skipped items=%d deps=%d)",
		res.CreatedItems, res.CreatedDeps, res.SkippedItems, res.SkippedDeps)
	if res.CreatedItems != 4 || res.CreatedDeps != 1 {
		t.Fatalf("seed counts = items %d deps %d, want items 4 deps 1", res.CreatedItems, res.CreatedDeps)
	}
	for _, k := range []string{"plan-onboarding", "thread-alpha", "task-draft", "task-review"} {
		if res.IDs[k] == "" {
			t.Fatalf("seed produced no id for key %q", k)
		}
	}

	// ─────────────────────────────────────────────────────────────────────────────────────────
	// PROOF 1 — get_context cold-start. One call on the freshly-seeded store returns the four sets.
	cold, err := eng.GetContext(ctx, 0)
	if err != nil {
		t.Fatalf("get_context cold-start: %v", err)
	}
	t.Log("=== DRY-RUN PROOF 1: get_context cold-start ===")
	t.Logf("active (%d):", len(cold.Active))
	for _, it := range cold.Active {
		t.Logf("  - %s  phase=%s court=%s title=%q", it.ID, it.Phase, it.Court, it.Title)
	}
	t.Logf("blocked (%d):", len(cold.Blocked))
	for _, b := range cold.Blocked {
		t.Logf("  - %s  phase=%s blocked_by=%v title=%q", b.ID, b.Phase, b.BlockingItems, b.Title)
	}
	t.Logf("stalled (%d):", len(cold.Stalled))
	for _, s := range cold.Stalled {
		t.Logf("  - %s  days_since=%d", s.ID, s.DaysSinceLastTransition)
	}
	t.Logf("recent_transitions (%d):", len(cold.RecentTransitions))
	for _, tr := range cold.RecentTransitions {
		t.Logf("  - item=%s event=%s from=%s to=%s actor=%s", tr.Item, tr.Event, tr.From, tr.To, tr.Actor)
	}
	if b, merr := json.Marshal(cold.Counts); merr != nil {
		t.Fatalf("marshal cold-start counts: %v", merr)
	} else {
		t.Logf("counts: %s", string(b))
	}

	// Assertions: the seed's shape must be surfaced correctly.
	//  - all four items sit in queue,
	//  - exactly the dependent (task-review) is blocked, and it names its blocker,
	//  - the other three non-terminal items are active,
	//  - nothing is stalled on a freshly-seeded store,
	//  - the block edge shows up as the one recent transition.
	if got := cold.Counts.ByPhase["queue"]; got != 4 {
		t.Fatalf("cold-start counts.by_phase[queue] = %d, want 4", got)
	}
	if len(cold.Blocked) != 1 || cold.Blocked[0].ID != res.IDs["task-review"] {
		t.Fatalf("cold-start blocked = %+v, want exactly task-review (%s)", cold.Blocked, res.IDs["task-review"])
	}
	if !contains(cold.Blocked[0].BlockingItems, res.IDs["task-draft"]) {
		t.Fatalf("blocked task-review must name blocker task-draft (%s), got %v",
			res.IDs["task-draft"], cold.Blocked[0].BlockingItems)
	}
	if len(cold.Active) != 3 {
		t.Fatalf("cold-start active = %d items, want 3", len(cold.Active))
	}
	for _, want := range []string{"plan-onboarding", "thread-alpha", "task-draft"} {
		if !activeHas(cold.Active, res.IDs[want]) {
			t.Fatalf("cold-start active must include %q (%s); got %+v", want, res.IDs[want], cold.Active)
		}
	}
	if len(cold.Stalled) != 0 {
		t.Fatalf("cold-start stalled = %d, want 0 on a freshly-seeded store", len(cold.Stalled))
	}
	if !transitionsHave(cold.RecentTransitions, res.IDs["task-review"], "block") {
		t.Fatalf("cold-start recent_transitions must include the block on task-review; got %+v", cold.RecentTransitions)
	}
	t.Logf("OBSERVED: 4 queued, task-review blocked by task-draft, 3 active, 0 stalled, block audited.")

	// ─────────────────────────────────────────────────────────────────────────────────────────
	// Bind the imported items into a runner (exactly what D8 does with importer.Result.IDs).
	r := NewEngineRunner(app, cfg)
	for key, id := range res.IDs {
		r.Bind(key, id)
	}

	// ─────────────────────────────────────────────────────────────────────────────────────────
	// PROOF 2 — gate refused-then-satisfied.
	// Walk thread-alpha (a decision) to review through the runner (its queue->work and work->review
	// edges are ungated). At review, review->terminal is gated on an accepted decision doc.
	if err := r.Run(ctx, Scenario{
		Name: "adoption-advance-decision-to-review",
		Steps: []Step{
			{Name: "start the decision", Op: Transition, Item: "thread-alpha", To: "work", Expect: Expect{Phase: "work"}},
			{Name: "decision into review", Op: Transition, Item: "thread-alpha", To: "review", Expect: Expect{Phase: "review"}},
		},
	}); err != nil {
		t.Fatalf("advance thread-alpha to review: %v", err)
	}

	t.Log("=== DRY-RUN PROOF 2: gate refused-then-satisfied ===")

	// Capture the EXACT refusal message via the runner's own engine (the doc is absent by
	// default — r.val holds no verdict for decPtr yet). This is the same engine + validator the
	// runner drives; the direct call is a read-only observation of the refusal text.
	// Invariant this relies on: a REFUSED transition does not bump the item's optimistic-concurrency
	// version, so r.version(threadAlphaID) below stays valid for the subsequent runner steps. If the
	// engine ever starts versioning refused attempts, this probe must re-read the version afterward.
	es, ok := r.surface.(engineSurface)
	if !ok {
		t.Fatalf("engine runner surface is %T, want engineSurface", r.surface)
	}
	threadAlphaID := res.IDs["thread-alpha"]
	_, refErr := es.eng.Transition(ctx, engine.TransitionInput{
		ItemID: threadAlphaID, TargetPhase: "terminal", Version: r.version(threadAlphaID),
		Actor: engine.Actor{Name: "operator", Kind: "human"},
	})
	if refErr == nil {
		t.Fatal("gate must refuse review->terminal while the decision doc is absent, got success")
	}
	if !engine.IsRefusal(refErr) {
		t.Fatalf("expected a gate refusal, got a non-refusal error: %v", refErr)
	}
	if !strings.Contains(refErr.Error(), "does not exist") || !strings.Contains(refErr.Error(), decPtr) {
		t.Fatalf("refusal must name the missing doc %q with \"does not exist\"; got: %s", decPtr, refErr.Error())
	}
	t.Logf("REFUSED (doc absent): %s", refErr.Error())

	// Now assert the same refused-then-satisfied arc through the runner (which also checks the
	// phase is unchanged after refusal, then admits once the doc is accepted).
	if err := r.Run(ctx, Scenario{
		Name: "adoption-gate-refuse-then-satisfy",
		Steps: []Step{
			{Name: "doc still absent", Op: SetDoc, Pointer: decPtr, DocMissing: true},
			{Name: "complete refused: doc absent", Op: Transition, Item: "thread-alpha", To: "terminal",
				Expect: Expect{Refused: true, RefusalContains: "does not exist", Phase: "review", AuditEvent: "gate_refused"}},
			{Name: "accept the decision doc", Op: SetDoc, Pointer: decPtr, DocStatus: "accepted", DocValid: true},
			{Name: "complete admitted", Op: Transition, Item: "thread-alpha", To: "terminal",
				Expect: Expect{Phase: "terminal", StatusLabel: "done", AuditEvent: "advance"}},
		},
	}); err != nil {
		t.Fatalf("gate refuse-then-satisfy: %v", err)
	}
	satisfied, err := app.FindRecordById("items", threadAlphaID)
	if err != nil {
		t.Fatalf("reload thread-alpha: %v", err)
	}
	t.Logf("SATISFIED (doc accepted): thread-alpha advanced to phase=%s status_label=%s",
		satisfied.GetString("phase"), satisfied.GetString("status_label"))

	// ─────────────────────────────────────────────────────────────────────────────────────────
	// PROOF 3 — dependency auto-unblock.
	// task-draft blocks task-review with unblock_at=review, cascade=auto. Advancing task-draft to
	// work leaves task-review blocked; advancing it to review auto-releases task-review.
	t.Log("=== DRY-RUN PROOF 3: dependency auto-unblock ===")
	t.Logf("before: task-review blocked=%v (blocker task-draft not yet at unblock_at=review)",
		blockedOf(t, app, res.IDs["task-review"]))
	if err := r.Run(ctx, Scenario{
		Name: "adoption-auto-unblock",
		Steps: []Step{
			{Name: "blocker to work: dependent stays blocked", Op: Transition, Item: "task-draft", To: "work",
				Expect: Expect{Phase: "work", StillBlocked: []string{"task-review"}}},
			{Name: "blocker to review: dependent auto-unblocks", Op: Transition, Item: "task-draft", To: "review",
				Expect: Expect{Phase: "review", AutoUnblocked: []string{"task-review"}}},
		},
	}); err != nil {
		t.Fatalf("auto-unblock: %v", err)
	}
	t.Logf("after: task-draft reached review -> task-review blocked=%v (auto-unblocked)",
		blockedOf(t, app, res.IDs["task-review"]))
	if blockedOf(t, app, res.IDs["task-review"]) {
		t.Fatal("task-review must be auto-unblocked once task-draft reaches its unblock_at phase")
	}
	t.Logf("OBSERVED: task-draft reaching review auto-unblocked task-review.")
}

func contains(xs []string, want string) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
}

func activeHas(active []engine.ItemSummary, id string) bool {
	for _, it := range active {
		if it.ID == id {
			return true
		}
	}
	return false
}

func transitionsHave(rows []engine.TransitionRow, item, event string) bool {
	for _, tr := range rows {
		if tr.Item == item && tr.Event == event {
			return true
		}
	}
	return false
}

// mustNotExist fails the test if path exists, and separately fails on any non-IsNotExist stat
// error (so a permission error can't masquerade as a clean "does not exist"). Empty path is a
// no-op (the canonical real-home path is "" when os.UserHomeDir fails).
func mustNotExist(t *testing.T, label, path string) {
	t.Helper()
	if path == "" {
		return
	}
	switch _, err := os.Stat(path); {
	case err == nil:
		t.Fatalf("%s %q must not exist (dry-run must not write a live store)", label, path)
	case !os.IsNotExist(err):
		t.Fatalf("stat %s %q: unexpected error: %v", label, path, err)
	}
}

func blockedOf(t *testing.T, app pbcore.App, id string) bool {
	t.Helper()
	rec, err := app.FindRecordById("items", id)
	if err != nil {
		t.Fatalf("reload item %q: %v", id, err)
	}
	return rec.GetBool("blocked")
}

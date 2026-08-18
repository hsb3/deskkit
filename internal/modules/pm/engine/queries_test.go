package engine

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/hsb3/deskkit/internal/core/schema"
)

// backdate rewrites an item's created timestamp via raw SQL — autodate fields ignore explicit
// sets, and the stalled computation (§5.2) needs an old item. The string-interpolated SQL is
// INTENTIONAL and test-only (controlled literal inputs; PB's filter APIs cannot bypass the
// autodate guard) — do not copy this pattern into production code.
func backdate(t *testing.T, e *Engine, table, id string, ago time.Duration) {
	t.Helper()
	ts := time.Now().Add(-ago).UTC().Format("2006-01-02 15:04:05.000Z")
	if _, err := e.App.DB().NewQuery(
		"UPDATE " + table + " SET created = '" + ts + "' WHERE id = '" + id + "'").Execute(); err != nil {
		t.Fatalf("backdate %s/%s: %v", table, id, err)
	}
}

// TestGetContext_FourSets drives the §5.2 briefing: active, blocked (with blocking items +
// reason), stalled (threshold), recent_transitions, ancestors, counts — the D4 acceptance
// "get_context returns the four sets".
func TestGetContext_FourSets(t *testing.T) {
	e := newEngine(t, &stubValidator{})

	// A parent (root) and its child: the ancestors chain surfaces root..parent.
	root := mustCreate(t, e, CreateItemInput{Title: "the epic", Type: "task", Court: "desk"})
	child := mustCreate(t, e, CreateItemInput{Title: "the slice", Type: "task", Parent: root.Id, Court: "crew", Priority: 2})
	child = mustTransition(t, e, child, "work") // active, court crew

	// A blocker edge: blocker (queue) blocks victim until review → victim is blocked.
	blocker := mustCreate(t, e, CreateItemInput{Title: "the blocker", Type: "task", Court: "desk", Priority: 1})
	victim := mustCreate(t, e, CreateItemInput{Title: "the victim", Type: "task", Court: "owner"})
	if _, err := e.Link(context.Background(), LinkInput{
		From: blocker.Id, To: victim.Id, Kind: "blocks", UnblockAt: "review", Cascade: "auto", Actor: human,
	}); err != nil {
		t.Fatalf("Link: %v", err)
	}

	// A stalled item: created long ago, no transitions since.
	old := mustCreate(t, e, CreateItemInput{Title: "the stale one", Type: "task", Court: "desk"})
	backdate(t, e, "items", old.Id, 30*24*time.Hour)

	// A terminal item: counted by phase, surfaced nowhere. Typed "analysis" (a type the
	// shipped default gate rules never bind), so it can walk to terminal ungated. An
	// UNTYPED item can no longer walk a document-gated edge like task's work->review — see
	// engine_test.go TestEmptyTypeCannotSkipDocumentGate — so this fixture must carry a real,
	// ungated type rather than none, to keep testing "an ungated item walks freely" and not
	// the now-closed empty-type loophole.)
	done := mustCreate(t, e, CreateItemInput{Title: "the done one", Type: "analysis"})
	done = mustTransition(t, e, done, "work")
	done = mustTransition(t, e, done, "review")
	done = mustTransition(t, e, done, "terminal")

	res, err := e.GetContext(context.Background(), 0)
	if err != nil {
		t.Fatalf("GetContext: %v", err)
	}
	if res.Desk != "test-desk" || res.GeneratedAt == "" {
		t.Errorf("desk/generated_at malformed: %q %q", res.Desk, res.GeneratedAt)
	}

	activeIDs := map[string]bool{}
	for _, it := range res.Active {
		activeIDs[it.ID] = true
	}
	for _, want := range []string{root.Id, child.Id, blocker.Id, old.Id} {
		if !activeIDs[want] {
			t.Errorf("active set missing %s (have %v)", want, activeIDs)
		}
	}
	if activeIDs[victim.Id] || activeIDs[done.Id] {
		t.Error("blocked/terminal items must not be in active")
	}

	if len(res.Blocked) != 1 || res.Blocked[0].ID != victim.Id {
		t.Fatalf("blocked set = %+v, want exactly the victim", res.Blocked)
	}
	if len(res.Blocked[0].BlockingItems) != 1 || res.Blocked[0].BlockingItems[0] != blocker.Id {
		t.Errorf("victim's blocking_items = %v, want [%s]", res.Blocked[0].BlockingItems, blocker.Id)
	}
	if !strings.Contains(res.Blocked[0].BlockedReason, blocker.Id) {
		t.Errorf("blocked_reason %q should name the blocker", res.Blocked[0].BlockedReason)
	}

	if len(res.Stalled) != 1 || res.Stalled[0].ID != old.Id {
		t.Fatalf("stalled set = %+v, want exactly the backdated item", res.Stalled)
	}
	if res.Stalled[0].DaysSinceLastTransition < 29 {
		t.Errorf("days_since_last_transition = %d, want ~30", res.Stalled[0].DaysSinceLastTransition)
	}

	if len(res.RecentTransitions) == 0 {
		t.Fatal("recent_transitions is empty despite advances/blocks")
	}
	if res.RecentTransitions[0].At == "" || res.RecentTransitions[0].Event == "" {
		t.Errorf("recent transition rows malformed: %+v", res.RecentTransitions[0])
	}

	if chain, ok := res.Ancestors[child.Id]; !ok || len(chain) != 1 || chain[0] != root.Id {
		t.Errorf("ancestors[%s] = %v, want [%s]", child.Id, res.Ancestors[child.Id], root.Id)
	}

	if res.Counts.ByPhase["terminal"] != 1 || res.Counts.ByPhase["work"] != 1 || res.Counts.ByPhase["queue"] != 4 {
		t.Errorf("counts.by_phase = %v", res.Counts.ByPhase)
	}
	if res.Counts.ByCourt["desk"] != 3 || res.Counts.ByCourt["crew"] != 1 || res.Counts.ByCourt["owner"] != 1 {
		t.Errorf("counts.by_court = %v", res.Counts.ByCourt)
	}
}

// TestGetContext_ActiveOrdering pins the §5.2 active ordering: (court, priority).
func TestGetContext_ActiveOrdering(t *testing.T) {
	e := newEngine(t, &stubValidator{})
	b2 := mustCreate(t, e, CreateItemInput{Title: "b2", Court: "desk", Priority: 2})
	a9 := mustCreate(t, e, CreateItemInput{Title: "a9", Court: "crew", Priority: 9})
	b1 := mustCreate(t, e, CreateItemInput{Title: "b1", Court: "desk", Priority: 1})
	res, err := e.GetContext(context.Background(), 0)
	if err != nil {
		t.Fatalf("GetContext: %v", err)
	}
	got := []string{res.Active[0].ID, res.Active[1].ID, res.Active[2].ID}
	want := []string{a9.Id, b1.Id, b2.Id}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("active order = %v, want %v", got, want)
		}
	}
}

// TestListItems_Filters exercises every §5.1 filter: phase, court, type, blocked, parent.
func TestListItems_Filters(t *testing.T) {
	e := newEngine(t, &stubValidator{})
	parent := mustCreate(t, e, CreateItemInput{Title: "parent", Type: "task", Court: "desk"})
	kid := mustCreate(t, e, CreateItemInput{Title: "kid", Type: "decision", Parent: parent.Id, Court: "owner"})
	moved := mustCreate(t, e, CreateItemInput{Title: "moved", Type: "task", Court: "desk"})
	moved = mustTransition(t, e, moved, "work")
	if _, err := e.Block(context.Background(), kid.Id, kid.GetInt("version"), human, "waiting"); err != nil {
		t.Fatalf("Block: %v", err)
	}

	cases := []struct {
		name   string
		filter ListFilter
		want   []string
	}{
		{"phase work", ListFilter{Phase: "work"}, []string{moved.Id}},
		{"court owner", ListFilter{Court: "owner"}, []string{kid.Id}},
		{"type decision", ListFilter{Type: "decision"}, []string{kid.Id}},
		{"blocked true", ListFilter{Blocked: "true"}, []string{kid.Id}},
		{"blocked false", ListFilter{Blocked: "false"}, []string{parent.Id, moved.Id}},
		{"parent", ListFilter{Parent: parent.Id}, []string{kid.Id}},
		{"all", ListFilter{}, []string{parent.Id, kid.Id, moved.Id}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := e.ListItems(context.Background(), tc.filter)
			if err != nil {
				t.Fatalf("ListItems: %v", err)
			}
			ids := map[string]bool{}
			for _, it := range got {
				ids[it.ID] = true
			}
			if len(got) != len(tc.want) {
				t.Fatalf("got %d items (%v), want %d", len(got), ids, len(tc.want))
			}
			for _, w := range tc.want {
				if !ids[w] {
					t.Errorf("missing %s in %v", w, ids)
				}
			}
		})
	}
}

// TestGetItem_Detail asserts the §5.1 get_item shape: summary + notes + both-direction deps
// + recent transitions + ancestor chain.
func TestGetItem_Detail(t *testing.T) {
	e := newEngine(t, &stubValidator{})
	root := mustCreate(t, e, CreateItemInput{Title: "root", Type: "task"})
	mid := mustCreate(t, e, CreateItemInput{Title: "mid", Type: "task", Parent: root.Id})
	leaf := mustCreate(t, e, CreateItemInput{Title: "leaf", Type: "task", Parent: mid.Id, Pointer: "docs/x.md"})
	other := mustCreate(t, e, CreateItemInput{Title: "other", Type: "task"})
	leaf = mustTransition(t, e, leaf, "work")
	if _, err := e.AddNote(context.Background(), leaf.Id, "rationale", "because", human); err != nil {
		t.Fatalf("AddNote: %v", err)
	}
	if _, err := e.Link(context.Background(), LinkInput{
		From: leaf.Id, To: other.Id, Kind: "relates-to", Actor: human,
	}); err != nil {
		t.Fatalf("Link: %v", err)
	}

	d, err := e.GetItem(context.Background(), leaf.Id)
	if err != nil {
		t.Fatalf("GetItem: %v", err)
	}
	if d.Title != "leaf" || d.Phase != "work" || d.Pointer != "docs/x.md" {
		t.Errorf("summary malformed: %+v", d.ItemSummary)
	}
	if len(d.Notes) != 1 || d.Notes[0].Key != "rationale" || d.Notes[0].Phase != "work" {
		t.Errorf("notes = %+v", d.Notes)
	}
	if len(d.Dependencies) != 1 || d.Dependencies[0].Kind != "relates-to" {
		t.Errorf("dependencies = %+v", d.Dependencies)
	}
	if len(d.RecentTransitions) != 1 || d.RecentTransitions[0].Event != "advance" {
		t.Errorf("recent transitions = %+v", d.RecentTransitions)
	}
	if len(d.Ancestors) != 2 || d.Ancestors[0] != root.Id || d.Ancestors[1] != mid.Id {
		t.Errorf("ancestors = %v, want [%s %s] (root..parent)", d.Ancestors, root.Id, mid.Id)
	}
}

// TestUpdateItem covers §5.1 update_item: version check, field edits, empty-title refusal,
// phase-changing status_label routed through the machine + gates.
func TestUpdateItem(t *testing.T) {
	e := newEngine(t, &stubValidator{})
	item := mustCreate(t, e, CreateItemInput{Title: "before", Type: "task", Court: "desk"})

	// Version mismatch is refused (R2.6).
	title := "after"
	if _, err := e.UpdateItem(context.Background(), UpdateItemInput{
		ItemID: item.Id, Version: 99, Title: &title, Actor: human,
	}); !IsRefusal(err) {
		t.Fatalf("stale version should refuse, got %v", err)
	}

	// Field edits + version bump.
	court := "owner"
	sev := "high"
	prio := 7
	updated, err := e.UpdateItem(context.Background(), UpdateItemInput{
		ItemID: item.Id, Version: item.GetInt("version"),
		Title: &title, Court: &court, Severity: &sev, Priority: &prio, Actor: human,
	})
	if err != nil {
		t.Fatalf("UpdateItem: %v", err)
	}
	if updated.GetString("title") != "after" || updated.GetString("court") != "owner" ||
		updated.GetString("severity") != "high" || updated.GetInt("priority") != 7 {
		t.Errorf("edits not applied: %v", updated.PublicExport())
	}
	if updated.GetInt("version") != item.GetInt("version")+1 {
		t.Errorf("version not bumped: %d", updated.GetInt("version"))
	}

	// An empty title is refused.
	empty := "  "
	if _, err := e.UpdateItem(context.Background(), UpdateItemInput{
		ItemID: item.Id, Version: updated.GetInt("version"), Title: &empty, Actor: human,
	}); !IsRefusal(err) {
		t.Fatalf("empty title should refuse, got %v", err)
	}

	// A status_label of a DIFFERENT phase is a transition request through the machine (§3.3):
	// "active" maps to work, and queue->work is a legal ungated edge, so it advances.
	label := "active"
	moved, err := e.UpdateItem(context.Background(), UpdateItemInput{
		ItemID: item.Id, Version: updated.GetInt("version"), StatusLabel: &label, Actor: human,
	})
	if err != nil {
		t.Fatalf("UpdateItem(status_label): %v", err)
	}
	if moved.GetString("phase") != "work" || moved.GetString("status_label") != "active" {
		t.Errorf("label-driven transition failed: phase=%s label=%s",
			moved.GetString("phase"), moved.GetString("status_label"))
	}
	// And an ILLEGAL label-driven edge is refused by the machine (work -> terminal directly).
	doneLabel := "done"
	if _, err := e.UpdateItem(context.Background(), UpdateItemInput{
		ItemID: item.Id, Version: moved.GetInt("version"), StatusLabel: &doneLabel, Actor: human,
	}); !IsRefusal(err) {
		t.Fatalf("work->terminal via label should refuse, got %v", err)
	}
}

// TestUpdateItemRejectsUnknownType: update_item must refuse an unknown
// items.type with the exact same schema-v1 vocabulary check create_item already applies
// (TestCreateItemRejectsUnknownType), closing the one remaining write path that previously
// accepted any string unchecked into a field gates.Effective binds on. Clearing the type back
// to "" stays legal (mirrors create_item's "absent type stays legal" scope call, ADR 0012).
func TestUpdateItemRejectsUnknownType(t *testing.T) {
	t.Run("unknown type is refused and leaves the field unchanged", func(t *testing.T) {
		e := newEngine(t, &stubValidator{})
		item := mustCreate(t, e, CreateItemInput{Title: "x", Type: "task"})

		bogus := "no-such-type"
		_, err := e.UpdateItem(context.Background(), UpdateItemInput{
			ItemID: item.Id, Version: item.GetInt("version"), Type: &bogus, Actor: human,
		})
		if err == nil {
			t.Fatal("expected a refusal for an unknown item type, got nil error")
		}
		if !IsRefusal(err) {
			t.Fatalf("expected a Refusal, got %T: %v", err, err)
		}
		if !strings.Contains(err.Error(), "no-such-type") {
			t.Fatalf("refusal should name the offending type, got %v", err)
		}
		if !strings.Contains(err.Error(), "doctypes.yaml") {
			t.Fatalf("refusal should point at the vocabulary source, got %v", err)
		}
		if reloaded, lerr := e.loadItem(item.Id); lerr != nil || reloaded.GetString("type") != "task" {
			t.Fatalf("a refused type update must not touch the stored type, got %v (err %v)",
				reloaded, lerr)
		}
	})

	t.Run("clearing the type back to empty stays legal", func(t *testing.T) {
		e := newEngine(t, &stubValidator{})
		item := mustCreate(t, e, CreateItemInput{Title: "x", Type: "task"})
		empty := ""
		updated, err := e.UpdateItem(context.Background(), UpdateItemInput{
			ItemID: item.Id, Version: item.GetInt("version"), Type: &empty, Actor: human,
		})
		if err != nil {
			t.Fatalf("clearing type should be legal, got %v", err)
		}
		if updated.GetString("type") != "" {
			t.Fatalf("expected type cleared, got %q", updated.GetString("type"))
		}
	})

	t.Run("every known vocabulary type is still accepted", func(t *testing.T) {
		vocab, err := schema.Vocab()
		if err != nil {
			t.Fatalf("schema.Vocab(): %v", err)
		}
		e := newEngine(t, &stubValidator{})
		item := mustCreate(t, e, CreateItemInput{Title: "x"})
		for _, typ := range vocab.TypeNames() {
			typ := typ
			t.Run(typ, func(t *testing.T) {
				current, lerr := e.loadItem(item.Id)
				if lerr != nil {
					t.Fatal(lerr)
				}
				if _, err := e.UpdateItem(context.Background(), UpdateItemInput{
					ItemID: item.Id, Version: current.GetInt("version"), Type: &typ, Actor: human,
				}); err != nil {
					t.Fatalf("UpdateItem(type=%s): %v", typ, err)
				}
			})
		}
	})
}

// TestUpdateItemGatedByClaim pins ADR 0020 (a live claim is authoritative over every direct
// mutation) for update_item: a live foreign claim refuses a non-holder's field edit AND its
// status_label path (which delegates to SetStatusLabel), naming the holder, while the holder
// and an expired claim proceed. The absent-claim success path is already covered by
// TestUpdateItem (an unclaimed item edited by `human`).
func TestUpdateItemGatedByClaim(t *testing.T) {
	e := newEngine(t, &stubValidator{})
	alice := Actor{Name: "agent-alice", Kind: "agent"}
	bob := Actor{Name: "agent-bob", Kind: "agent"}

	item := mustCreate(t, e, CreateItemInput{Title: "contested", Type: "analysis", Court: "desk"})
	item, err := e.Claim(context.Background(), item.Id, item.GetInt("version"), alice)
	if err != nil {
		t.Fatal(err)
	}

	// A non-holder's field edit is refused, naming the holder; nothing is written.
	newTitle := "hijacked"
	if _, err := e.UpdateItem(context.Background(), UpdateItemInput{
		ItemID: item.Id, Version: item.GetInt("version"), Title: &newTitle, Actor: bob,
	}); !IsRefusal(err) || !strings.Contains(err.Error(), "agent-alice") {
		t.Fatalf("foreign live claim must refuse UpdateItem naming the holder, got %v", err)
	}
	if item, _ = e.loadItem(item.Id); item.GetString("title") != "contested" {
		t.Fatalf("refused foreign UpdateItem must not edit the field, got title %q", item.GetString("title"))
	}

	// The status_label path is gated the same way (it delegates to SetStatusLabel): a
	// non-holder's label change is refused up front, before any transition.
	label := "active"
	if _, err := e.UpdateItem(context.Background(), UpdateItemInput{
		ItemID: item.Id, Version: item.GetInt("version"), StatusLabel: &label, Actor: bob,
	}); !IsRefusal(err) || !strings.Contains(err.Error(), "agent-alice") {
		t.Fatalf("foreign live claim must refuse a status_label UpdateItem, got %v", err)
	}
	if item, _ = e.loadItem(item.Id); item.GetString("phase") != "queue" {
		t.Fatalf("refused foreign status_label UpdateItem must not transition, got phase %q", item.GetString("phase"))
	}

	// The holder may edit.
	holderTitle := "renamed by holder"
	item, err = e.UpdateItem(context.Background(), UpdateItemInput{
		ItemID: item.Id, Version: item.GetInt("version"), Title: &holderTitle, Actor: alice,
	})
	if err != nil || item.GetString("title") != "renamed by holder" {
		t.Fatalf("the claim holder must be able to UpdateItem, got %v (title %q)", err, item.GetString("title"))
	}

	// An EXPIRED claim frees the path for a non-holder (§3.6).
	item.Set("claim_expires", time.Now().Add(-time.Minute))
	if err := e.App.Save(item); err != nil {
		t.Fatal(err)
	}
	item, _ = e.loadItem(item.Id)
	freeTitle := "edited after expiry"
	if _, err := e.UpdateItem(context.Background(), UpdateItemInput{
		ItemID: item.Id, Version: item.GetInt("version"), Title: &freeTitle, Actor: bob,
	}); err != nil {
		t.Fatalf("expired claim must let a non-holder UpdateItem, got %v", err)
	}
}

// TestItemBody_UpdateAndProjection covers the §5.1 body surface end-to-end at the engine layer:
// GetItem projects items.body into ItemDetail.Body, UpdateItem edits it, and a non-nil pointer to
// an empty string is a deliberate clear (mirroring the other *string fields).
func TestItemBody_UpdateAndProjection(t *testing.T) {
	e := newEngine(t, &stubValidator{})
	const initial = "initial narrative + acceptance criteria"
	item := mustCreate(t, e, CreateItemInput{Title: "body item", Type: "task", Body: initial})

	// GetItem projects the stored body.
	d, err := e.GetItem(context.Background(), item.Id)
	if err != nil {
		t.Fatalf("GetItem: %v", err)
	}
	if d.Body != initial {
		t.Fatalf("GetItem body projection: got %q, want %q", d.Body, initial)
	}

	// UpdateItem edits the body.
	const revised = "revised spec text stored inline on the item"
	body := revised
	updated, err := e.UpdateItem(context.Background(), UpdateItemInput{
		ItemID: item.Id, Version: item.GetInt("version"), Body: &body, Actor: human,
	})
	if err != nil {
		t.Fatalf("UpdateItem(body): %v", err)
	}
	if updated.GetString("body") != revised {
		t.Fatalf("UpdateItem body edit: got %q, want %q", updated.GetString("body"), revised)
	}
	d, err = e.GetItem(context.Background(), item.Id)
	if err != nil {
		t.Fatalf("GetItem after update: %v", err)
	}
	if d.Body != revised {
		t.Fatalf("GetItem body after update: got %q, want %q", d.Body, revised)
	}

	// A non-nil empty string clears the body deliberately (omitempty then drops it from JSON).
	empty := ""
	cleared, err := e.UpdateItem(context.Background(), UpdateItemInput{
		ItemID: item.Id, Version: updated.GetInt("version"), Body: &empty, Actor: human,
	})
	if err != nil {
		t.Fatalf("UpdateItem(clear body): %v", err)
	}
	if cleared.GetString("body") != "" {
		t.Fatalf("UpdateItem body clear: got %q, want empty", cleared.GetString("body"))
	}
}

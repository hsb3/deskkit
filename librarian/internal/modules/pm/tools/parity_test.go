package tools

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	pbcore "github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"

	"github.com/example/pocket-librarian/internal/core/config"
	"github.com/example/pocket-librarian/internal/modules/pm/collections"
	"github.com/example/pocket-librarian/internal/modules/pm/engine"
	pmtui "github.com/example/pocket-librarian/internal/modules/pm/tui"
)

// newPMApp boots a fresh test app with the pm collections applied directly (no global
// registration — cannot pollute other tests' migration lists) and returns it with a resolved
// test config.
func newPMApp(t *testing.T) (pbcore.App, *config.Config) {
	t.Helper()
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("tests.NewTestApp: %v", err)
	}
	t.Cleanup(app.Cleanup)
	for _, mig := range collections.Migrations() {
		if err := mig.Up(app); err != nil {
			t.Fatalf("apply pm migration %q: %v", mig.Basename, err)
		}
	}
	cfg := &config.Config{
		DeskRoot: t.TempDir(), DeskName: "test-desk",
		PMEnabled: true, PMAutonomousWrites: true, PMClaimTTL: 30 * time.Minute,
	}
	return app, cfg
}

// normalizeCtx clears the per-call timestamp so three same-second calls compare equal.
func normalizeCtx(r *engine.ContextResult) *engine.ContextResult {
	r.GeneratedAt = ""
	return r
}

// TestGetContext_SurfaceParity is test lane §10.10 (D4 acceptance): the SAME get_context
// result is reachable via (1) the engine core the CLI calls, (2) the registered tool spec's
// model-facing invoke path (the eino/MCP binding), and (3) the TUI landing view — one core,
// three surfaces.
func TestGetContext_SurfaceParity(t *testing.T) {
	app, cfg := newPMApp(t)
	eng := &engine.Engine{App: app, Cfg: cfg}
	ctx := context.Background()

	// Seed a small graph: an active item, a blocked pair, a note.
	a, err := eng.CreateItem(ctx, engine.CreateItemInput{Title: "active one", Court: "desk"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := eng.Transition(ctx, engine.TransitionInput{
		ItemID: a.Id, TargetPhase: "work", Version: a.GetInt("version"),
		Actor: engine.Actor{Name: "owner", Kind: "human"},
	}); err != nil {
		t.Fatal(err)
	}
	b, err := eng.CreateItem(ctx, engine.CreateItemInput{Title: "blocker", Court: "desk"})
	if err != nil {
		t.Fatal(err)
	}
	v, err := eng.CreateItem(ctx, engine.CreateItemInput{Title: "victim", Court: "owner"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := eng.Link(ctx, engine.LinkInput{
		From: b.Id, To: v.Id, Kind: "blocks", UnblockAt: "terminal", Cascade: "auto",
		Actor: engine.Actor{Name: "owner", Kind: "human"},
	}); err != nil {
		t.Fatal(err)
	}

	// Surface 1: the engine core (what the `pm context` CLI prints).
	direct, err := eng.GetContext(ctx, 0)
	if err != nil {
		t.Fatalf("engine.GetContext: %v", err)
	}

	// Surface 2: the registered tool spec's invoke path (the eino/MCP binding built from the
	// SAME spec toolcore registers on both model-facing surfaces).
	var spec = Specs(noValidator, true)[0] // get_context (frozen order; see specs_test)
	einoTool, err := spec.NewEinoTool(app, cfg)
	if err != nil {
		t.Fatalf("NewEinoTool: %v", err)
	}
	raw, err := einoTool.InvokableRun(ctx, `{}`)
	if err != nil {
		t.Fatalf("InvokableRun(get_context): %v", err)
	}
	viaTool := &engine.ContextResult{}
	if err := json.Unmarshal([]byte(raw), viaTool); err != nil {
		t.Fatalf("unmarshal tool result: %v", err)
	}

	// Surface 3: the TUI landing view (spec §5.3) loads through the same engine call.
	views := pmtui.Views(app, cfg)
	if len(views) != 3 {
		t.Fatalf("pm mounts %d TUI views, want 3", len(views))
	}
	landing := views[0]
	if landing.Name() != pmtui.ContextViewName {
		t.Fatalf("first view = %q, want the context landing view", landing.Name())
	}
	_ = landing.Init()
	res, ok := landing.(interface{ Result() *engine.ContextResult })
	if !ok {
		t.Fatal("context view does not expose its loaded Result")
	}
	viaTUI := res.Result()
	if viaTUI == nil {
		t.Fatal("context view loaded no result")
	}

	// One core, three surfaces: identical payloads (timestamps normalized).
	d := normalizeCtx(direct)
	tl := normalizeCtx(viaTool)
	tv := normalizeCtx(viaTUI)
	if !reflect.DeepEqual(d, tl) {
		t.Errorf("engine vs tool-path results differ:\n%+v\nvs\n%+v", d, tl)
	}
	if !reflect.DeepEqual(d, tv) {
		t.Errorf("engine vs TUI-view results differ:\n%+v\nvs\n%+v", d, tv)
	}

	// The briefing itself carries the expected sets for this seed.
	if len(d.Active) != 2 || len(d.Blocked) != 1 || d.Blocked[0].ID != v.Id {
		t.Errorf("briefing sets wrong: active=%d blocked=%+v", len(d.Active), d.Blocked)
	}
}

// TestToolBodies_EndToEnd drives the twelve tool functions over one store — the model-facing
// adapter layer over the engine (create → claim → note → link → transition-refused-by-claim →
// release → transition → update → block/unblock → get/list/context).
func TestToolBodies_EndToEnd(t *testing.T) {
	app, cfg := newPMApp(t)
	ctx := context.Background()
	me := ActorFields{Actor: "crew-1", ActorKind: "agent"}

	created, err := CreateItem(ctx, app, cfg, nil, &CreateItemInput{
		Title: "the item", Court: "desk", ActorFields: me,
	})
	if err != nil {
		t.Fatalf("CreateItem: %v", err)
	}
	id := created.Item.ID

	claimed, err := ClaimItem(ctx, app, cfg, nil, &ClaimItemInput{ItemID: id, Version: created.Item.Version, ActorFields: me})
	if err != nil {
		t.Fatalf("ClaimItem: %v", err)
	}
	if claimed.Item.ClaimedBy != "crew-1" {
		t.Fatalf("claimed_by = %q", claimed.Item.ClaimedBy)
	}

	// A FOREIGN actor's transition is refused while the claim is live (R2.6).
	if _, err := TransitionItem(ctx, app, cfg, nil, &TransitionItemInput{
		ItemID: id, TargetPhase: "work", Version: claimed.Item.Version,
		ActorFields: ActorFields{Actor: "crew-2", ActorKind: "agent"},
	}); err == nil {
		t.Fatal("foreign transition during a live claim must refuse")
	}

	if _, err := AddNote(ctx, app, cfg, nil, &AddNoteInput{ItemID: id, Key: "handoff", Body: "state", ActorFields: me}); err != nil {
		t.Fatalf("AddNote: %v", err)
	}

	other, err := CreateItem(ctx, app, cfg, nil, &CreateItemInput{Title: "other", ActorFields: me})
	if err != nil {
		t.Fatal(err)
	}
	edge, err := LinkItems(ctx, app, cfg, nil, &LinkItemsInput{
		From: other.Item.ID, To: id, Kind: "is-blocked-by", UnblockAt: "work", Cascade: "auto", ActorFields: me,
	})
	if err != nil {
		t.Fatalf("LinkItems: %v", err)
	}
	// is-blocked-by canonicalizes: stored as id -blocks-> other (§3.4).
	if edge.Edge.Kind != "blocks" || edge.Edge.From != id || edge.Edge.To != other.Item.ID {
		t.Fatalf("edge not canonicalized: %+v", edge.Edge)
	}

	released, err := ReleaseItem(ctx, app, cfg, nil, &ReleaseItemInput{ItemID: id, Version: claimed.Item.Version, ActorFields: me})
	if err != nil {
		t.Fatalf("ReleaseItem: %v", err)
	}
	moved, err := TransitionItem(ctx, app, cfg, nil, &TransitionItemInput{
		ItemID: id, TargetPhase: "work", Version: released.Item.Version, ActorFields: me,
	})
	if err != nil {
		t.Fatalf("TransitionItem: %v", err)
	}
	if moved.Item.Phase != "work" {
		t.Fatalf("phase = %q, want work", moved.Item.Phase)
	}
	// The advance cascaded the auto edge (id reached work): other auto-unblocked.
	detail, err := GetItem(ctx, app, cfg, nil, &GetItemInput{ItemID: other.Item.ID})
	if err != nil {
		t.Fatal(err)
	}
	if detail.Blocked {
		t.Error("auto cascade should have unblocked the dependent item at unblock_at=work")
	}

	upd, err := UpdateItem(ctx, app, cfg, nil, &UpdateItemInput{
		ItemID: id, Version: moved.Item.Version, Severity: "high", Priority: 3, ActorFields: me,
	})
	if err != nil {
		t.Fatalf("UpdateItem: %v", err)
	}
	if upd.Item.Severity != "high" || upd.Item.Priority != 3 {
		t.Fatalf("update not applied: %+v", upd.Item)
	}

	blocked, err := BlockItem(ctx, app, cfg, nil, &BlockItemInput{ItemID: id, Version: upd.Item.Version, Reason: "waiting on review", ActorFields: me})
	if err != nil {
		t.Fatalf("BlockItem: %v", err)
	}
	if !blocked.Item.Blocked {
		t.Fatal("block did not set the flag")
	}
	unblocked, err := UnblockItem(ctx, app, cfg, nil, &UnblockItemInput{ItemID: id, Version: blocked.Item.Version, ActorFields: me})
	if err != nil {
		t.Fatalf("UnblockItem: %v", err)
	}
	if unblocked.Item.Blocked || unblocked.Item.Phase != "work" {
		t.Fatalf("unblock did not restore: %+v", unblocked.Item)
	}

	list, err := ListItems(ctx, app, cfg, nil, &ListItemsInput{Phase: "work"})
	if err != nil {
		t.Fatalf("ListItems: %v", err)
	}
	if len(list) != 1 || list[0].ID != id {
		t.Fatalf("list = %+v", list)
	}
	briefing, err := GetContext(ctx, app, cfg, nil, &GetContextInput{})
	if err != nil {
		t.Fatalf("GetContext: %v", err)
	}
	if len(briefing.Active) != 2 {
		t.Fatalf("briefing.active = %+v", briefing.Active)
	}
	// The audit trail carries the acting agent (R2.5).
	found := false
	for _, tr := range briefing.RecentTransitions {
		if tr.Actor == "crew-1" && tr.ActorKind == "agent" {
			found = true
		}
	}
	if !found {
		t.Error("no audit row records the agent actor")
	}
}

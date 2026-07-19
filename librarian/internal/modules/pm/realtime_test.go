package pm

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	pbcore "github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/subscriptions"

	"github.com/example/pocket-librarian/internal/core/config"
	"github.com/example/pocket-librarian/internal/modules/pm/collections"
	"github.com/example/pocket-librarian/internal/modules/pm/engine"
)

// newRealtimeApp boots a test app with the pm collections applied directly and the pm
// realtime hook registered (as main's OnServe wiring does for an enabled module).
func newRealtimeApp(t *testing.T) (pbcore.App, *engine.Engine) {
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
	if err := (&Mod{}).RegisterRealtime(app); err != nil {
		t.Fatalf("RegisterRealtime: %v", err)
	}
	eng := &engine.Engine{App: app, Cfg: &config.Config{
		DeskRoot: t.TempDir(), DeskName: "test-desk", PMClaimTTL: 30 * time.Minute,
	}}
	return app, eng
}

// collect reads up to n messages off a client channel (delivery is async — the module sends
// per client in a goroutine), then confirms a short quiet window follows (no extra events).
func collect(t *testing.T, c subscriptions.Client, n int) []subscriptions.Message {
	t.Helper()
	var out []subscriptions.Message
	deadline := time.After(5 * time.Second)
	for len(out) < n {
		select {
		case m := <-c.Channel():
			out = append(out, m)
		case <-deadline:
			t.Fatalf("timed out waiting for message %d/%d", len(out)+1, n)
		}
	}
	select {
	case m := <-c.Channel():
		t.Fatalf("unexpected extra realtime message: %s", m.Data)
	case <-time.After(150 * time.Millisecond):
	}
	return out
}

// TestRealtime_EmitsOnTransitions is the D4 acceptance "realtime emits on transitions"
// (spec §5.4, R4.3): every transitions write — advance AND the dependency-driven cascade's
// block/unblock rows — broadcasts one pm/transitions message to subscribed clients, and a
// client without the subscription receives nothing.
func TestRealtime_EmitsOnTransitions(t *testing.T) {
	app, eng := newRealtimeApp(t)
	ctx := context.Background()
	actor := engine.Actor{Name: "owner", Kind: "human"}

	sub := subscriptions.NewDefaultClient()
	sub.Subscribe(RealtimeTopic)
	app.SubscriptionsBroker().Register(sub)
	other := subscriptions.NewDefaultClient() // no subscription: must stay silent
	app.SubscriptionsBroker().Register(other)

	// Seed: A blocks B until work (auto); creating the edge blocks B (a block audit row).
	a, err := eng.CreateItem(ctx, engine.CreateItemInput{Title: "A", Actor: actor})
	if err != nil {
		t.Fatal(err)
	}
	b, err := eng.CreateItem(ctx, engine.CreateItemInput{Title: "B", Actor: actor})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := eng.Link(ctx, engine.LinkInput{
		From: a.Id, To: b.Id, Kind: "blocks", UnblockAt: "work", Cascade: "auto", Actor: actor,
	}); err != nil {
		t.Fatal(err)
	}
	afterLink := collect(t, sub, 1)
	if afterLink[0].Name != RealtimeTopic {
		t.Fatalf("message on topic %q, want %q", afterLink[0].Name, RealtimeTopic)
	}
	var blockEv map[string]any
	if err := json.Unmarshal(afterLink[0].Data, &blockEv); err != nil {
		t.Fatalf("unmarshal event: %v", err)
	}
	if blockEv["event"] != "block" || blockEv["item"] != b.Id {
		t.Errorf("block event payload = %v", blockEv)
	}

	// Advance A to work: one advance event for A + the cascade's unblock event for B.
	if _, err := eng.Transition(ctx, engine.TransitionInput{
		ItemID: a.Id, TargetPhase: "work", Version: a.GetInt("version"), Actor: actor,
	}); err != nil {
		t.Fatal(err)
	}
	events := collect(t, sub, 2)
	kinds := map[string]string{}
	for _, m := range events {
		var ev map[string]any
		if err := json.Unmarshal(m.Data, &ev); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		kinds[ev["event"].(string)] = ev["item"].(string)
	}
	if kinds["advance"] != a.Id || kinds["unblock"] != b.Id {
		t.Errorf("expected advance(A) + unblock(B), got %v", kinds)
	}

	// The unsubscribed client received nothing (its channel never even gets a sender).
	select {
	case m := <-other.Channel():
		t.Errorf("unsubscribed client received a message: %s", m.Data)
	case <-time.After(150 * time.Millisecond):
	}
}

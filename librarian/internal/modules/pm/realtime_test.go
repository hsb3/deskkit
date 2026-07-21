package pm

import (
	"context"
	"encoding/json"
	"strconv"
	"testing"
	"time"

	pbcore "github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/subscriptions"

	"github.com/hsb3/desk-standard/librarian/internal/core/config"
	"github.com/hsb3/desk-standard/librarian/internal/modules/pm/collections"
	"github.com/hsb3/desk-standard/librarian/internal/modules/pm/engine"
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

// orderProbeClient embeds a real DefaultClient but delays each Send by an amount INVERSELY
// proportional to the message's sequence (carried in `detail`): message 0 waits longest, the
// last message waits least. Under the OLD per-message `go client.Send` fan-out every message is
// delivered by its own racing goroutine, so a later message that waits less overtakes an
// earlier one — a deterministic reordering the ordering assertion catches. Under the bounded
// dispatcher a SINGLE drain goroutine calls Send in turn, so the per-message delay only paces
// delivery; written order is preserved. The delay is what makes this a reliable regression
// guard: without it, real sequential writes are spaced far enough apart that even the old
// fan-out's goroutines happen to block in order.
type orderProbeClient struct {
	*subscriptions.DefaultClient
	n    int
	unit time.Duration
}

func (c *orderProbeClient) Send(m subscriptions.Message) {
	if seq, ok := probeSeq(m); ok {
		time.Sleep(time.Duration(c.n-seq) * c.unit) // later messages wait less
	}
	c.DefaultClient.Send(m)
}

// probeSeq recovers the monotonic sequence a test stamped into a message's `detail` field.
func probeSeq(m subscriptions.Message) (int, bool) {
	var ev map[string]any
	if json.Unmarshal(m.Data, &ev) != nil {
		return 0, false
	}
	s, ok := ev["detail"].(string)
	if !ok {
		return 0, false
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0, false
	}
	return v, true
}

// TestRealtime_PerClientOrdering proves the bounded realtime dispatch preserves per-client FIFO
// order under concurrent delivery pressure: a subscribed client receives transitions in exactly
// the order the rows were written, never a scrambled order. It is the regression guard for the
// old per-message `go client.Send` fan-out, which raced one goroutine per message onto the
// client's channel and offered NO per-client ordering guarantee.
//
// Method: emit n transitions rows through the real save hook, each stamped with a strictly
// increasing sequence in `detail`; the subscribed client (orderProbeClient) delays each Send
// inversely to that sequence so the old fan-out's independent sender goroutines finish out of
// order (later messages overtake earlier ones). The bounded dispatcher serializes all sends
// through one goroutine, so the sequence still arrives strictly 0..n-1. n is kept below
// realtimeQueueSize so every message fits the per-client buffer and none is dropped.
func TestRealtime_PerClientOrdering(t *testing.T) {
	app, eng := newRealtimeApp(t)
	ctx := context.Background()
	actor := engine.Actor{Name: "owner", Kind: "human"}

	item, err := eng.CreateItem(ctx, engine.CreateItemInput{Title: "seq", Actor: actor})
	if err != nil {
		t.Fatal(err)
	}

	const n = 12 // < realtimeQueueSize (64): every message fits the buffer, so none is dropped
	sub := &orderProbeClient{DefaultClient: subscriptions.NewDefaultClient(), n: n, unit: 15 * time.Millisecond}
	sub.Subscribe(RealtimeTopic)
	app.SubscriptionsBroker().Register(sub)

	col, err := app.FindCollectionByNameOrId("transitions")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < n; i++ {
		rec := pbcore.NewRecord(col)
		rec.Set("item", item.Id)
		rec.Set("event", "advance")
		rec.Set("actor", actor.Name)
		rec.Set("actor_kind", actor.Kind)
		rec.Set("detail", strconv.Itoa(i)) // the strictly increasing ordering key
		if err := app.Save(rec); err != nil {
			t.Fatalf("write transitions row %d: %v", i, err)
		}
	}

	msgs := collect(t, sub, n)
	got := make([]int, len(msgs))
	for i, m := range msgs {
		v, ok := probeSeq(m)
		if !ok {
			t.Fatalf("message %d carried no integer detail: %s", i, m.Data)
		}
		got[i] = v
	}
	for i, v := range got {
		if v != i {
			t.Fatalf("per-client ordering violated at position %d: got %d, want %d\nfull received order: %v",
				i, v, i, got)
		}
	}
}

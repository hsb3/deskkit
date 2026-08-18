package pm

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/subscriptions"
)

// realtimeQueueSize bounds each client's pending-message buffer. Realtime is an observer
// channel (spec §5.4): under sustained backpressure from a slow client we DROP rather than
// stall a transition, so a small fixed buffer is deliberate — it absorbs a normal burst
// without letting one non-draining subscriber pin unbounded memory.
const realtimeQueueSize = 64

// drainIdleRecheck bounds how long a per-client drain goroutine can sit idle before it
// re-checks whether its client was discarded. The reaper (see dispatch) closes a gone
// client's queue on the NEXT broadcast, which is the common teardown path; this periodic
// recheck is the belt-and-suspenders that lets an idle goroutine exit when its client is
// discarded during a lull with no further broadcasts, so no goroutine outlives its client.
const drainIdleRecheck = 5 * time.Second

// realtimeManager fans transitions rows out to subscribed clients with a bounded, per-client
// ORDERED dispatch. Each client is served by exactly ONE serialized sender — a single drain
// goroutine reading a buffered queue — so:
//
//   - per-client FIFO is guaranteed by construction (one goroutine, one queue, delivered in
//     enqueue order); nothing races to write a given client's channel;
//   - a live-but-not-draining client costs at most ONE goroutine plus realtimeQueueSize
//     buffered messages, never one goroutine per message (the old fan-out's unbounded growth);
//   - enqueue is NON-BLOCKING, so the transition hook never waits on realtime.
//
// DROP POLICY: drop-newest-on-full. When a client's queue is full the incoming message is
// discarded FOR THAT CLIENT ONLY (other clients are unaffected). Realtime is an observer
// channel (§5.4) — a dropped event under sustained backpressure is acceptable; a blocked or
// failed transition is not.
//
// State lives on the *Mod (see RegisterRealtime), so each app/test gets its own manager and
// no dispatcher state leaks across apps.
type realtimeManager struct {
	mu          sync.Mutex
	dispatchers map[string]*clientDispatcher
}

func newRealtimeManager() *realtimeManager {
	return &realtimeManager{dispatchers: map[string]*clientDispatcher{}}
}

// clientDispatcher is the single serialized sender for one subscription client: a bounded
// queue drained in FIFO order by exactly one goroutine.
type clientDispatcher struct {
	client subscriptions.Client
	queue  chan subscriptions.Message
}

// broadcastTransition marshals one transitions row and enqueues it for every subscribed, live
// client. A marshal failure is swallowed — realtime is an observer channel and must never fail
// a transition (§5.4).
func (m *realtimeManager) broadcastTransition(app core.App, rec *core.Record) {
	payload, err := json.Marshal(map[string]any{
		"item":       rec.GetString("item"),
		"event":      rec.GetString("event"),
		"from_phase": rec.GetString("from_phase"),
		"to_phase":   rec.GetString("to_phase"),
		"actor":      rec.GetString("actor"),
		"actor_kind": rec.GetString("actor_kind"),
		"detail":     rec.GetString("detail"),
		"created":    rec.GetDateTime("created").String(),
	})
	if err != nil {
		return
	}
	m.dispatch(app, subscriptions.Message{Name: RealtimeTopic, Data: payload})
}

// dispatch reaps dispatchers whose client has gone away, then enqueues msg for every live,
// subscribed client — lazily creating a per-client dispatcher (buffer + single drain
// goroutine) on first use. It never blocks: a full queue drops the message for that client.
func (m *realtimeManager) dispatch(app core.App, msg subscriptions.Message) {
	clients := app.SubscriptionsBroker().Clients()

	m.mu.Lock()
	defer m.mu.Unlock()

	// Reap: any dispatcher whose client is no longer registered or has been discarded is torn
	// down. Closing its queue lets the drain goroutine finish any buffered sends and exit —
	// this is the primary "exit when IsDiscarded" path (no goroutine leak).
	for id, d := range m.dispatchers {
		if c, ok := clients[id]; !ok || c.IsDiscarded() {
			close(d.queue)
			delete(m.dispatchers, id)
		}
	}

	for id, client := range clients {
		if client.IsDiscarded() || !client.HasSubscription(RealtimeTopic) {
			continue
		}
		d := m.dispatchers[id]
		if d == nil {
			d = &clientDispatcher{
				client: client,
				queue:  make(chan subscriptions.Message, realtimeQueueSize),
			}
			m.dispatchers[id] = d
			go d.drain()
		}
		// Non-blocking enqueue: drop-newest when this client's buffer is full (see DROP POLICY
		// on realtimeManager). The transition hook is never allowed to wait on a slow client.
		select {
		case d.queue <- msg:
		default:
		}
	}
}

// drain is a client's single serialized sender: it delivers queued messages in FIFO order via
// client.Send (an unbuffered write that only the client's live SSE writer drains). Exactly one
// drain goroutine exists per client, so per-client ordering holds by construction. It exits
// when the queue is closed (the client was reaped) or the client is discarded, so a goroutine
// never outlives its client. Send recovers a closed-channel panic internally, so a client
// discarded mid-send is safe.
func (d *clientDispatcher) drain() {
	timer := time.NewTimer(drainIdleRecheck)
	defer timer.Stop()
	for {
		select {
		case msg, ok := <-d.queue:
			if !ok {
				return // queue closed by the reaper → clean exit
			}
			if !d.client.IsDiscarded() {
				d.client.Send(msg)
			}
		case <-timer.C:
			if d.client.IsDiscarded() {
				return // discarded during a lull, no broadcast to reap us → exit anyway
			}
		}
		// Go 1.23+ timer semantics: Reset after a possible fire delivers no stale value.
		timer.Reset(drainIdleRecheck)
	}
}

package agent

import (
	"testing"

	"github.com/cloudwego/eino/schema"

	"github.com/example/pocket-librarian/internal/core/config"
	"github.com/example/pocket-librarian/internal/core/toolcore"
	"github.com/example/pocket-librarian/internal/modules/librarian/tools"
)

// TestDeltaMessages_NoDoublePersist simulates the cumulative model inputs across a ReAct run
// (each model call sees the whole conversation so far) and asserts the high-water-mark delta
// yields every message EXACTLY ONCE, in order — the invariant the single-callback persistence
// mechanism relies on to never write a message twice.
func TestDeltaMessages_NoDoublePersist(t *testing.T) {
	sys := &schema.Message{Role: schema.System, Content: "sys"}
	user := &schema.Message{Role: schema.User, Content: "fix X"}
	asst1 := &schema.Message{Role: schema.Assistant, Content: "", ToolCalls: []schema.ToolCall{{ID: "c1"}}}
	tool1 := &schema.Message{Role: schema.Tool, Content: "r1", ToolCallID: "c1"}
	asst2 := &schema.Message{Role: schema.Assistant, Content: "", ToolCalls: []schema.ToolCall{{ID: "c2"}}}
	tool2 := &schema.Message{Role: schema.Tool, Content: "r2", ToolCallID: "c2"}

	// The sequence of cumulative inputs the callback sees on successive model calls.
	calls := [][]*schema.Message{
		{sys, user},
		{sys, user, asst1, tool1},
		{sys, user, asst1, tool1, asst2, tool2},
	}

	hwm := 0
	var persisted []*schema.Message
	for _, in := range calls {
		delta, newHwm := deltaMessages(in, hwm)
		persisted = append(persisted, delta...)
		hwm = newHwm
	}

	want := []*schema.Message{sys, user, asst1, tool1, asst2, tool2}
	if len(persisted) != len(want) {
		t.Fatalf("persisted %d messages, want %d (double-persist or gap)", len(persisted), len(want))
	}
	for i := range want {
		if persisted[i] != want[i] {
			t.Fatalf("persisted[%d] = %q, want %q (order/identity mismatch)", i, persisted[i].Content, want[i].Content)
		}
	}
	if hwm != len(want) {
		t.Fatalf("final hwm = %d, want %d", hwm, len(want))
	}
}

// TestDeltaMessages_NoNewMessages: a repeated input with no growth persists nothing.
func TestDeltaMessages_NoNewMessages(t *testing.T) {
	msgs := []*schema.Message{{Role: schema.User, Content: "x"}}
	delta, newHwm := deltaMessages(msgs, 1)
	if len(delta) != 0 {
		t.Fatalf("expected no delta when hwm == len, got %d", len(delta))
	}
	if newHwm != 1 {
		t.Fatalf("hwm should stay 1, got %d", newHwm)
	}
}

// TestAgentTools_GateComposition verifies the §5.4 registration-time gate as consumed by the
// agent slice: the autonomous set always carries the read/flag/propose tools; apply_fix only
// when LIBRARIAN_AUTONOMOUS_WRITES is on; restore never.
func TestAgentTools_GateComposition(t *testing.T) {
	// The agent loop's buildTools now iterates the shared toolcore registry (post-refactor), so
	// the gate composition is asserted against toolcore populated with the librarian's specs —
	// exactly the set main wires at startup. Reset first for isolation from other tests.
	toolcore.Reset()
	toolcore.Register(tools.Specs()...)
	check := func(t *testing.T, autonomous bool, wantApply bool) {
		t.Helper()
		cfg := &config.Config{AutonomousWrites: autonomous}
		names := map[string]bool{}
		for _, n := range toolcore.ToolNames(toolcore.AgentTools(cfg)) {
			names[n] = true
		}
		for _, always := range []string{"sweep", "patrol", "propose_fix", "query"} {
			if !names[always] {
				t.Errorf("autonomous=%v: expected %q in the agent tool set", autonomous, always)
			}
		}
		if names["restore"] {
			t.Errorf("autonomous=%v: restore must never be in the agent tool set", autonomous)
		}
		if names["apply_fix"] != wantApply {
			t.Errorf("autonomous=%v: apply_fix present=%v, want %v", autonomous, names["apply_fix"], wantApply)
		}
	}
	check(t, false, false)
	check(t, true, true)
}

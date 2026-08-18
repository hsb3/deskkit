package statemachine

import "testing"

// TestEdge_Legality is test lane §10.2's machine half: every one of the 16 phase pairs is
// checked against the spec §3.2 table — the six legal edges carry the right event, every
// other pair (including self-loops and skips like queue→review or queue→terminal) is refused.
func TestEdge_Legality(t *testing.T) {
	want := map[[2]Phase]Event{
		{Queue, Work}:      Advance,
		{Work, Review}:     Advance,
		{Review, Terminal}: Advance,
		{Review, Work}:     Demote,
		{Work, Queue}:      Demote,
		{Terminal, Work}:   Reopen,
	}
	for _, from := range Phases() {
		for _, to := range Phases() {
			ev, ok := Edge(from, to)
			expected, legal := want[[2]Phase{from, to}]
			if legal != ok {
				t.Errorf("Edge(%s,%s): legal=%v, want %v", from, to, ok, legal)
				continue
			}
			if legal && ev != expected {
				t.Errorf("Edge(%s,%s): event=%s, want %s", from, to, ev, expected)
			}
		}
	}
}

func TestParsePhase(t *testing.T) {
	for _, p := range Phases() {
		if got, err := ParsePhase(string(p)); err != nil || got != p {
			t.Errorf("ParsePhase(%s): %v, %v", p, got, err)
		}
	}
	if _, err := ParsePhase("done"); err == nil {
		t.Error("ParsePhase should refuse a status label that is not a phase")
	}
}

func TestParseEdgeKey(t *testing.T) {
	from, to, err := ParseEdgeKey("review->terminal")
	if err != nil || from != Review || to != Terminal {
		t.Errorf("ParseEdgeKey(review->terminal) = %s,%s,%v", from, to, err)
	}
	for _, bad := range []string{"queue->terminal", "work→review", "review", "queue->queue", "nope->work"} {
		if _, _, err := ParseEdgeKey(bad); err == nil {
			t.Errorf("ParseEdgeKey(%q) should fail", bad)
		}
	}
}

func TestDefaultStatusLabels(t *testing.T) {
	labels := DefaultStatusLabels()
	if labels["backlog"] != Queue || labels["next"] != Queue || labels["active"] != Work ||
		labels["in-review"] != Review || labels["done"] != Terminal ||
		labels["dropped"] != Terminal || labels["superseded"] != Terminal {
		t.Errorf("default labels drifted from spec §3.3: %v", labels)
	}
	if len(labels) != 7 {
		t.Errorf("expected exactly the 7 seeded phase labels, got %d", len(labels))
	}
	if _, isLabel := labels["blocked"]; isLabel {
		t.Error("blocked is the side-state flag, never a phase label")
	}
	for _, p := range Phases() {
		if labels[DefaultLabelFor(p)] != p {
			t.Errorf("DefaultLabelFor(%s)=%q does not map back to %s", p, DefaultLabelFor(p), p)
		}
	}
}

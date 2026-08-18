// Package statemachine is the rigid, code-owned phase machine (spec §3.2): four phases, a
// fixed legal-edge table (data-in-code — extending it is a code change, not a config change,
// deliberately; spec §9/R1.4), and the phase<->status-label mapping helpers (§3.3). The
// `blocked` side-state is a bool on the item, NOT a phase; the engine (modules/pm/engine)
// owns its handling — this package only answers "is this edge legal, and what event is it?".
package statemachine

import "fmt"

// Phase is one of the four rigid phases.
type Phase string

const (
	Queue    Phase = "queue"
	Work     Phase = "work"
	Review   Phase = "review"
	Terminal Phase = "terminal"
)

// Phases lists the machine's phases in rank order (the items.phase select values).
func Phases() []Phase { return []Phase{Queue, Work, Review, Terminal} }

// ParsePhase validates a phase string.
func ParsePhase(s string) (Phase, error) {
	switch Phase(s) {
	case Queue, Work, Review, Terminal:
		return Phase(s), nil
	}
	return "", fmt.Errorf("unknown phase %q (phases: queue, work, review, terminal)", s)
}

// Rank orders phases for the dependency unblock_at comparison (§3.5): a blocker "reaches"
// phase P when Rank(current) >= Rank(P).
func Rank(p Phase) int {
	switch p {
	case Queue:
		return 0
	case Work:
		return 1
	case Review:
		return 2
	case Terminal:
		return 3
	}
	return -1
}

// Event is the audit label a legal edge carries (the transitions.event vocabulary also has
// block/unblock/claim/release/gate_refused, which are engine events, not machine edges).
type Event string

const (
	Advance Event = "advance"
	Demote  Event = "demote"
	Reopen  Event = "reopen"
)

// legalEdges is THE machine table (§3.2): queue→work, work→review, review→terminal (advance);
// review→work, work→queue (demote); terminal→work (reopen). The spec's diagram lists
// review→work under both demote and reopen; one edge needs one canonical audit label, and
// this machine records it as DEMOTE (walking an in-review item back to work), reserving
// REOPEN for the terminal→work resurrection. All other pairs are refused before gates are
// even consulted.
var legalEdges = map[[2]Phase]Event{
	{Queue, Work}:      Advance,
	{Work, Review}:     Advance,
	{Review, Terminal}: Advance,
	{Review, Work}:     Demote,
	{Work, Queue}:      Demote,
	{Terminal, Work}:   Reopen,
}

// Edge returns the event kind for from→to, or ok=false when the edge is illegal.
func Edge(from, to Phase) (Event, bool) {
	ev, ok := legalEdges[[2]Phase{from, to}]
	return ev, ok
}

// EdgeKey is the gate-config transition key for an edge (§4.2), e.g. "review->terminal".
func EdgeKey(from, to Phase) string { return string(from) + "->" + string(to) }

// ParseEdgeKey parses a gate-config transition key ("from->to") and validates it names a
// LEGAL machine edge (a gate bound to an impossible edge is a config error).
func ParseEdgeKey(key string) (from, to Phase, err error) {
	i := indexArrow(key)
	if i < 0 {
		return "", "", fmt.Errorf("transition key %q is not of the form \"from->to\"", key)
	}
	f, t := key[:i], key[i+2:]
	fp, ferr := ParsePhase(f)
	if ferr != nil {
		return "", "", fmt.Errorf("transition key %q: %w", key, ferr)
	}
	tp, terr := ParsePhase(t)
	if terr != nil {
		return "", "", fmt.Errorf("transition key %q: %w", key, terr)
	}
	if _, ok := Edge(fp, tp); !ok {
		return "", "", fmt.Errorf("transition key %q names an illegal edge", key)
	}
	return fp, tp, nil
}

func indexArrow(s string) int {
	for i := 0; i+1 < len(s); i++ {
		if s[i] == '-' && s[i+1] == '>' {
			return i
		}
	}
	return -1
}

// DefaultStatusLabels is the seeded label→phase vocabulary (spec §3.3, owner-ruled default —
// do NOT invent labels): backlog/next (queue), active (work), in-review (review),
// done/dropped/superseded (terminal). `blocked`/`waiting` are NOT phase labels — they surface
// the blocked FLAG regardless of phase and are handled by the surfaces, not stored as labels.
// Freely editable per desk via desk_config.status_labels.
func DefaultStatusLabels() map[string]Phase {
	return map[string]Phase{
		"backlog":    Queue,
		"next":       Queue,
		"active":     Work,
		"in-review":  Review,
		"done":       Terminal,
		"dropped":    Terminal,
		"superseded": Terminal,
	}
}

// DefaultLabelFor returns the canonical default label an engine write uses when a transition
// lands an item in a phase its current label no longer maps to (§3.3: the label and the
// machine cannot drift).
func DefaultLabelFor(p Phase) string {
	switch p {
	case Queue:
		return "backlog"
	case Work:
		return "active"
	case Review:
		return "in-review"
	case Terminal:
		return "done"
	}
	return ""
}

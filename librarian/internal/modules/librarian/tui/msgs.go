// Bubble Tea messages that carry the streaming engine's events onto the model's single
// goroutine. The engine (agent.Session.StreamTurn) runs the ReAct loop on its own goroutines
// and pushes agent.Event values down a channel; pump.go turns each received value into one of
// these messages, so every mutation of the model happens inside Update — race-free by
// construction (the plan's "all model mutation on the bubbletea goroutine" invariant).
package tui

import "github.com/hsb3/desk-standard/librarian/internal/modules/librarian/agent"

// eventMsg wraps one live engine event (token, tool_start, tool_end, final, or error). The
// terminal event (final|error) arrives as an eventMsg too; the channel-closed signal that
// follows it is delivered separately as turnDoneMsg.
type eventMsg struct {
	ev agent.Event
}

// turnDoneMsg signals that the engine channel has closed — the turn is fully over and no
// further events will arrive. It is the single place the model flips streaming off and
// re-enables input, so it is emitted exactly once per turn (when <-ch reports !ok).
type turnDoneMsg struct{}

// toastExpireMsg clears the transient footer toast (a copy confirmation or error) after its
// display window elapses. seq stamps the toast that scheduled it: a later toast bumps the model's
// sequence, so a stale expiry for a superseded toast is ignored and never clears a newer one.
type toastExpireMsg struct {
	seq int
}

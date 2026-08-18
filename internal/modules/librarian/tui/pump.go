// The event pump: the re-issued waitForEvent command that drains the engine channel one
// value at a time onto the Bubble Tea runtime. This is the ONLY place a goroutine other than
// the model's reads the channel, and it reads exactly one value per invocation before handing
// control back to Update, which re-issues it — so the model never blocks the UI and never
// touches its own fields off the main goroutine (the race-free-by-construction pattern from
// the plan, Part 2 pump.go).
package tui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/hsb3/deskkit/internal/modules/librarian/agent"
)

// waitForEvent returns a command that blocks on the next engine event and reports it. On a
// received value it yields eventMsg; when the channel is closed (the turn's terminal event has
// already been delivered) it yields turnDoneMsg exactly once. Update re-issues this command
// after every eventMsg, so the whole turn is drained promptly — satisfying the engine's
// contract that a full buffer must not stall the agent's goroutines.
func waitForEvent(ch <-chan agent.Event) tea.Cmd {
	return func() tea.Msg {
		ev, ok := <-ch
		if !ok {
			return turnDoneMsg{}
		}
		return eventMsg{ev: ev}
	}
}

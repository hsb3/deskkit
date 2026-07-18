// Package tui is the full-screen Bubble Tea chat surface for `pocket-librarian chat` on a real
// terminal. It drives the SAME agent.Session the line REPL uses (one conversation, one gated
// tool set, one persistence path); the REPL remains the piped / non-TTY / --plain fallback. The
// package lives outside cmd/ so the model's Update is unit-testable without a terminal.
//
// Run owns the Session lifecycle around the program. The engine contract is strict about
// shutdown: the caller must never Close the Session until the event channel of any in-flight
// turn has closed. Run therefore drains a still-streaming turn (cancelling it first) before
// Close, and uses a fresh context for Close when the caller's context is already canceled so the
// agent_runs row is still finalized on ctrl+c / parent cancellation.
package tui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/pocketbase/pocketbase/core"

	"github.com/example/pocket-librarian/internal/config"
)

// Run opens a Session eagerly (so a build/config failure surfaces before the alternate screen
// takes over the terminal), runs the Bubble Tea program on the alternate screen bound to ctx,
// then drains any in-flight turn and closes the Session. It returns the program's error, or a
// Close error when the program itself succeeded.
func Run(ctx context.Context, app core.App, cfg *config.Config) error {
	provider := &agentProvider{app: app, cfg: cfg}

	// Open the initial session eagerly (via the provider), so a build/config failure surfaces
	// before the alternate screen takes over the terminal.
	sess, err := provider.fresh(ctx)
	if err != nil {
		return err
	}

	p := tea.NewProgram(newModel(ctx, sess, provider, cfg), tea.WithAltScreen(), tea.WithContext(ctx))
	final, runErr := p.Run()

	// The final model may hold a DIFFERENT session than the one opened above: the user may have
	// resumed a conversation or started a new one, each of which swapped m.sess (closing the OLD
	// session inside Update at swap time). So exactly ONE session is live at program end — the
	// final model's — and that is the one to drain and close.
	finalSess := sess
	if fm, ok := final.(model); ok {
		if fm.sess != nil {
			finalSess = fm.sess
		}
		// Drain a turn that was still streaming when the program exited (e.g. the context was
		// canceled out from under it): cancel it, then read the channel to close. The engine never
		// drops events, so this always terminates once the terminal event lands.
		if fm.events != nil {
			if fm.cancelTurn != nil {
				fm.cancelTurn()
			}
			for range fm.events { //nolint:revive // intentional drain to channel close
			}
		}
	}

	// Finalize the agent_runs row. If the caller's context is already canceled, Close on it would
	// be a no-op path for anything context-sensitive, so use a short fresh context instead.
	closeCtx := ctx
	if ctx.Err() != nil {
		var cancel context.CancelFunc
		closeCtx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
	}
	if cerr := provider.closeSession(closeCtx, finalSess); cerr != nil && runErr == nil {
		runErr = cerr
	}
	return runErr
}

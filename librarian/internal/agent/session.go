// Multi-turn interactive session over the SAME eino ReAct loop that Run() drives
// (spec §6). A Session opens ONE agent_runs row and reuses one built *react.Agent across
// several Turns, carrying a growing conversation history so the model sees prior turns.
//
// The stewardship boundary is inherited, not re-declared: the tool slice comes from the
// existing buildTools/AgentTools gating (restore never exposed; apply_fix only under the
// autonomous-writes gate) and the system prompt comes from newAgent's MessageModifier
// (systemPrompt). Nothing here mutates desk files except through the existing
// propose -> apply tool path.
package agent

import (
	"context"
	"errors"
	"sync"

	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
	"github.com/pocketbase/pocketbase/core"

	"github.com/example/pocket-librarian/internal/config"
)

// maxHistoryMessages bounds the conversation context replayed to the model on each Turn.
// history holds only the user + final-assistant message of each turn (intermediate tool-call
// messages are persisted to the transcript, not kept here), so it is a clean alternating
// sequence; once it exceeds this many messages the oldest turns are dropped with a simple
// sliding window (no summarization). Even so a whole turn is kept intact at the boundary. This
// caps the per-Turn token cost and prevents unbounded growth in a long-running session.
const maxHistoryMessages = 40

// Session is a multi-turn conversation bound to one agent_runs row and one built agent.
// history accumulates the user + final-assistant message of each turn (bounded by
// maxHistoryMessages) and is replayed on each Turn, which is what makes the exchange
// multi-turn: the model sees the recent turns every time.
type Session struct {
	app     core.App
	cfg     *config.Config
	agent   *react.Agent
	run     *core.Record
	rc      *runCtx
	history []*schema.Message
	last    *schema.Message // most recent assistant output, used to finalize the run row

	// mu guards the single-turn concurrency state. busy rejects overlapping turns; termErr
	// carries the terminal turn's real error object (StreamTurn's Event is serializable-only,
	// so this is how Turn preserves context.Canceled identity for callers).
	mu      sync.Mutex
	busy    bool
	termErr error
}

// NewSession builds the chat model, the gated tool slice, and the ReAct agent exactly as
// Run() does, and opens ONE agent_runs row (trigger "manual") for the whole session. On any
// build failure the run row is failed like Run() does its pre-loop failures.
func NewSession(ctx context.Context, app core.App, cfg *config.Config) (*Session, error) {
	run, err := createAgentRun(app, "manual", "", cfg)
	if err != nil {
		return nil, err
	}
	rc := &runCtx{app: app, cfg: cfg, runID: run.Id}

	chatModel, err := chatModelFactory(ctx, cfg)
	if err != nil {
		return nil, failRun(app, run, rc, err)
	}
	toolset, err := buildTools(app, cfg) // §5.4 gating: restore absent, apply_fix gated
	if err != nil {
		return nil, failRun(app, run, rc, err)
	}
	ag, err := newAgent(ctx, app, chatModel, toolset, cfg) // MessageModifier prepends systemPrompt
	if err != nil {
		return nil, failRun(app, run, rc, err)
	}

	return &Session{
		app:   app,
		cfg:   cfg,
		agent: ag,
		run:   run,
		rc:    rc,
	}, nil
}

// Turn drives one full ReAct loop and returns the final assistant text (or the terminal
// error). It is a thin blocking drain over StreamTurn, so the REPL and the streaming TUI share
// ONE persistence/history code path. The user message, per-turn hwm baseline, final/partial
// persistence, history append, and rollback all happen inside StreamTurn (stream.go). A loop
// error is returned to the caller with its identity preserved (a canceled turn returns
// context.Canceled), matching the pre-streaming contract.
func (s *Session) Turn(ctx context.Context, userInput string) (string, error) {
	var final string
	var sawErr bool
	var errText string
	for ev := range s.StreamTurn(ctx, userInput) {
		switch ev.Kind {
		case EventFinal:
			final = ev.Content
		case EventError:
			sawErr = true
			errText = ev.Err
		}
	}
	if sawErr {
		s.mu.Lock()
		te := s.termErr
		s.mu.Unlock()
		if te != nil {
			return "", te
		}
		return "", errors.New(errText)
	}
	return final, nil
}

// capHistory keeps at most maxHistoryMessages of the most recent conversation, dropping the
// oldest messages with a sliding window (no summarization). Called after a turn's assistant
// reply is appended, so the retained tail always ends on a complete turn.
func capHistory(h []*schema.Message) []*schema.Message {
	if len(h) <= maxHistoryMessages {
		return h
	}
	return append([]*schema.Message(nil), h[len(h)-maxHistoryMessages:]...)
}

// Close finalizes the session's agent_runs row to its terminal state using the last output.
// Errors are non-fatal (logged, returned) so a caller can ignore them at shutdown.
func (s *Session) Close(ctx context.Context) error {
	if err := finishRun(s.app, s.run, s.rc, s.last, nil); err != nil {
		s.app.Logger().Error("finalize session run", "run", s.run.Id, "err", err)
		return err
	}
	return nil
}

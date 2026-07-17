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

	"github.com/cloudwego/eino/compose"
	einoagent "github.com/cloudwego/eino/flow/agent"
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

// Turn appends the user input to the running history, drives one full ReAct loop over the
// ENTIRE history (so prior turns are in context), appends the assistant reply back onto the
// history, persists the final assistant message (as Run does for its final out), and returns
// the assistant text. A loop error is returned to the caller; the transcript up to the last
// model call is already persisted by the callback.
func (s *Session) Turn(ctx context.Context, userInput string) (string, error) {
	s.history = append(s.history, schema.UserMessage(userInput))

	handler := s.rc.persistHandler() // the ONE persistence mechanism (persist.go)
	out, genErr := s.agent.Generate(ctx, s.history,
		einoagent.WithComposeOptions(compose.WithCallbacks(handler)))
	if genErr != nil {
		return "", genErr
	}
	if out != nil {
		s.history = append(s.history, out)
		s.history = capHistory(s.history)
		s.last = out
		// The final assistant message never appears in a later model INPUT, so the
		// input-side callback does not capture it; persist it here exactly as Run() does.
		if perr := s.rc.persist(out); perr != nil {
			s.app.Logger().Error("persist final message", "run", s.run.Id, "err", perr)
		}
		return out.Content, nil
	}
	return "", nil
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

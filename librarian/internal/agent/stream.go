// Streaming engine for the interactive chat surface (spec §6, chat-TUI plan Part 1). A
// Session drives the SAME eino ReAct loop as Turn()/Run(), but exposes a live event stream so
// a caller (the TUI, or a future SSE route) can render tokens, tool steps, and terminal state
// as they happen instead of blocking on a whole reply.
//
// Why tokens come from a callback, not the agent output stream: under the anthropic
// StreamToolCallChecker (agent.go), react.Agent.Stream's output reader only releases the final
// answer AFTER the checker reads ahead — a burst, not live tokens. Live tokens therefore come
// from cbutils.ModelCallbackHandler.OnEndWithStreamOutput (an independent stream copy); the
// agent output stream (ConcatMessageStream) is used only as the authoritative final message and
// completion signal. Callbacks fire on agent goroutines, so a stream copy is drained by a
// wg-tracked reader goroutine (never synchronously inside the callback, which would deadlock),
// and wg.Wait() gates the terminal event so all tokens precede final|error.
package agent

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	einoagent "github.com/cloudwego/eino/flow/agent"
	"github.com/cloudwego/eino/schema"
	cbutils "github.com/cloudwego/eino/utils/callbacks"
)

// EventKind enumerates the streamed event families. Values are stable JSON strings so the
// Event type can be marshaled directly onto a future SSE route without a translation layer.
type EventKind string

const (
	EventToken     EventKind = "token"      // content delta (+ Step ordinal)
	EventToolStart EventKind = "tool_start" // Tool, CallID, Args
	EventToolEnd   EventKind = "tool_end"   // Tool, CallID, Result (or error text)
	EventFinal     EventKind = "final"      // Content: full assistant answer
	EventError     EventKind = "error"      // Err, Canceled, Partial
)

// Event is a flat, JSON-taggable record of one streaming occurrence. It is deliberately
// serializable-only (no error values, no channels): the actual terminal error is preserved for
// in-process callers via the Session's unexported termErr field, keeping this type reusable as
// the SSE payload for the deferred webapp. Exactly one terminal event (final|error) is emitted
// per turn, after which the channel closes.
type Event struct {
	Kind     EventKind `json:"kind"`
	Step     int       `json:"step,omitempty"`
	Token    string    `json:"token,omitempty"`
	Tool     string    `json:"tool,omitempty"`
	CallID   string    `json:"call_id,omitempty"`
	Args     string    `json:"args,omitempty"`
	Result   string    `json:"result,omitempty"`
	Content  string    `json:"content,omitempty"`
	Err      string    `json:"err,omitempty"`
	Canceled bool      `json:"canceled,omitempty"`
	Partial  string    `json:"partial,omitempty"`
}

// errSessionBusy rejects an overlapping turn: a Session drives one turn at a time.
var errSessionBusy = errors.New("session busy: a turn is already in progress")

// StreamTurn drives one full ReAct loop over the running history and returns a buffered channel
// (cap 64) of live events. Exactly one terminal event (final|error) is emitted, then the channel
// is closed; the caller MUST drain to close (cancel ctx to abort — draining continues until the
// terminal event lands). The mutex-guarded busy flag rejects overlapping turns.
//
// Persistence and history semantics are identical to Turn() because Turn() is now a thin drain
// over this method: one code path, so the REPL and the TUI share exactly the same transcript
// behavior.
func (s *Session) StreamTurn(ctx context.Context, userInput string) <-chan Event {
	ch := make(chan Event, 64)

	s.mu.Lock()
	if s.busy {
		s.mu.Unlock()
		// Reject without touching termErr (the in-flight turn owns it): the busy error rides
		// in the Event, and Turn falls back to it when termErr is unset.
		ch <- Event{Kind: EventError, Err: errSessionBusy.Error()}
		close(ch)
		return ch
	}
	s.busy = true
	s.termErr = nil
	s.mu.Unlock()

	go s.runTurn(ctx, userInput, ch)
	return ch
}

// runTurn is the single-goroutine engine behind StreamTurn (plan Part 1 flow steps 1-7).
func (s *Session) runTurn(ctx context.Context, userInput string, ch chan<- Event) {
	defer close(ch)
	defer func() {
		s.mu.Lock()
		s.busy = false
		s.mu.Unlock()
	}()

	// Step 1 — per-turn hwm baseline (the persistence bug fix). Each turn's first model input
	// is [system] + history + [userN]; the baseline is the count of those already-persisted
	// leading messages so the first delta is exactly [userN]. On the very first turn (nothing
	// persisted yet, rc.seq == 0) the baseline is 0 so the system row persists exactly once.
	// See persist.go for the invariant this restores.
	s.rc.mu.Lock()
	firstTurn := s.rc.seq == 0
	if firstTurn {
		s.rc.hwm = 0
	} else {
		s.rc.hwm = 1 + len(s.history)
	}
	s.rc.mu.Unlock()

	// Step 2 — append the user message (remember the pre-append length for rollback), and on
	// the first turn backfill agent_runs.input_summary (picker title source) if still empty.
	preLen := len(s.history)
	s.history = append(s.history, schema.UserMessage(userInput))
	if firstTurn && s.run.GetString("input_summary") == "" {
		s.run.Set("input_summary", summarize(userInput))
		if err := s.app.Save(s.run); err != nil {
			s.app.Logger().Error("backfill input_summary", "run", s.run.Id, "err", err)
		}
	}

	// Step 3 — callbacks: the existing persistence handler PLUS a live-events handler.
	te := &turnEvents{ch: ch}
	persist := s.rc.persistHandler()
	events := te.handler()

	// Step 4 — drive the loop; the agent output stream is the authoritative final message.
	sr, streamErr := s.agent.Stream(ctx, s.history,
		einoagent.WithComposeOptions(compose.WithCallbacks(persist, events)))
	var out *schema.Message
	var runErr error
	if streamErr != nil {
		runErr = streamErr
	} else {
		out, runErr = schema.ConcatMessageStream(sr)
	}

	// Ordering guarantee: all token/tool events are emitted before the terminal event.
	te.wg.Wait()

	if runErr == nil && out != nil {
		// Step 5 — success.
		if perr := s.rc.persist(out); perr != nil {
			s.app.Logger().Error("persist final message", "run", s.run.Id, "err", perr)
		}
		s.history = append(s.history, out)
		s.history = capHistory(s.history)
		s.last = out
		ch <- Event{Kind: EventFinal, Content: out.Content}
		return
	}

	// Step 6 — error / cancel. Snapshot whatever the last model step streamed as the partial.
	partial := te.partial()
	canceled := ctx.Err() != nil || errors.Is(runErr, context.Canceled)
	if partial != "" {
		// Persist a PLAIN assistant row carrying the partial text (no synthetic marker — the DB
		// transcript stays model-safe; the TUI renders the "(interrupted)" badge). Appending it
		// keeps the history alternation valid, so the next turn is consistent.
		pm := schema.AssistantMessage(partial, nil)
		if perr := s.rc.persist(pm); perr != nil {
			s.app.Logger().Error("persist partial message", "run", s.run.Id, "err", perr)
		}
		s.history = append(s.history, pm)
		s.history = capHistory(s.history)
		s.last = pm
	} else {
		// No partial: roll the history back to its pre-turn length so no dangling user message
		// is left behind (this also fixes today's dangling-user-message bug on a failed Turn).
		s.history = s.history[:preLen]
	}

	// Preserve context.Canceled identity for callers (Turn returns termErr). Event stays
	// serializable-only: the error text rides in Err, the sentinel in termErr.
	s.mu.Lock()
	if canceled {
		s.termErr = context.Canceled
	} else {
		s.termErr = runErr
	}
	s.mu.Unlock()

	errText := ""
	if runErr != nil {
		errText = runErr.Error()
	}
	ch <- Event{Kind: EventError, Err: errText, Canceled: canceled, Partial: partial}
}

// turnEvents fans the eino model/tool callbacks of one turn into Event values on ch. Token
// readers run on their own goroutines (tracked by wg); lastText accumulates the CURRENT model
// step's text (reset per step) as the source of the cancel/error Partial. stepDone lets the
// tool callback wait for the preceding step's tokens to flush, so tool events land strictly
// between step-1 and step-2 tokens rather than racing them.
type turnEvents struct {
	ch       chan<- Event
	wg       sync.WaitGroup
	mu       sync.Mutex
	step     int
	lastText strings.Builder
	stepDone chan struct{}
}

// partial returns the current/last step's accumulated text under the lock.
func (te *turnEvents) partial() string {
	te.mu.Lock()
	defer te.mu.Unlock()
	return te.lastText.String()
}

// handler builds the events callback (ChatModel stream output + Tool start/end/error).
func (te *turnEvents) handler() callbacks.Handler {
	modelHandler := &cbutils.ModelCallbackHandler{
		OnEndWithStreamOutput: func(ctx context.Context, _ *callbacks.RunInfo, output *schema.StreamReader[*model.CallbackOutput]) context.Context {
			te.mu.Lock()
			te.step++
			myStep := te.step
			te.lastText.Reset() // Partial tracks only the latest step's tokens.
			done := make(chan struct{})
			te.stepDone = done
			te.mu.Unlock()

			te.wg.Add(1)
			go func() {
				defer te.wg.Done()
				defer close(done)
				defer output.Close()
				for {
					chunk, err := output.Recv()
					if err != nil {
						return // io.EOF or a cancel/stream error ends this step's tokens.
					}
					if chunk == nil || chunk.Message == nil || chunk.Message.Content == "" {
						continue
					}
					delta := chunk.Message.Content
					te.mu.Lock()
					te.lastText.WriteString(delta)
					te.mu.Unlock()
					te.ch <- Event{Kind: EventToken, Step: myStep, Token: delta}
				}
			}()
			return ctx
		},
	}

	toolHandler := &cbutils.ToolCallbackHandler{
		OnStart: func(ctx context.Context, info *callbacks.RunInfo, input *tool.CallbackInput) context.Context {
			// Flush the preceding step's tokens before announcing the tool, so ordering is
			// deterministic (step-1 tokens, then tool_start/tool_end, then step-2 tokens).
			te.mu.Lock()
			done := te.stepDone
			te.mu.Unlock()
			if done != nil {
				<-done
			}
			args := ""
			if input != nil {
				args = input.ArgumentsInJSON
			}
			te.ch <- Event{Kind: EventToolStart, Tool: toolName(info), CallID: compose.GetToolCallID(ctx), Args: args}
			return ctx
		},
		OnEnd: func(ctx context.Context, info *callbacks.RunInfo, output *tool.CallbackOutput) context.Context {
			result := ""
			if output != nil {
				result = output.Response
			}
			te.ch <- Event{Kind: EventToolEnd, Tool: toolName(info), CallID: compose.GetToolCallID(ctx), Result: result}
			return ctx
		},
		OnError: func(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
			text := ""
			if err != nil {
				text = err.Error()
			}
			// A tool error still closes the step: the error text rides in Result.
			te.ch <- Event{Kind: EventToolEnd, Tool: toolName(info), CallID: compose.GetToolCallID(ctx), Result: text}
			return ctx
		},
	}

	return cbutils.NewHandlerHelper().ChatModel(modelHandler).Tool(toolHandler).Handler()
}

// toolName reads the tool name from the callback RunInfo (the tool node sets RunInfo.Name to
// the invoked tool's name).
func toolName(info *callbacks.RunInfo) string {
	if info == nil {
		return ""
	}
	return info.Name
}

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
	EventToolEnd   EventKind = "tool_end"   // Tool, CallID, Result on success; Err on failure
	EventFinal     EventKind = "final"      // Content: full assistant answer
	EventError     EventKind = "error"      // Err, Canceled, Partial
)

// Event is a flat, JSON-taggable record of one streaming occurrence. It is deliberately
// serializable-only (no error values, no channels): the actual terminal error is preserved for
// in-process callers via the Session's unexported termErr field, so this type serializes cleanly
// as the SSE payload for the browser session surface (internal/modules/librarian/web). Exactly
// one terminal event (final|error) is emitted per turn, after which the channel closes.
type Event struct {
	Kind     EventKind `json:"kind"`
	Step     int       `json:"step,omitempty"`
	Token    string    `json:"token,omitempty"`
	Tool     string    `json:"tool,omitempty"`
	CallID   string    `json:"call_id,omitempty"`
	Args     string    `json:"args,omitempty"`
	Result   string    `json:"result,omitempty"`
	Content  string    `json:"content,omitempty"`
	Err      string    `json:"err,omitempty"` // terminal error text; on tool_end, a failed call's error
	Canceled bool      `json:"canceled,omitempty"`
	Partial  string    `json:"partial,omitempty"`

	// Token accounting, set on the terminal event only (final|error). PromptTokens is the LAST
	// model step's prompt count — that step's prompt already includes the full replayed history +
	// tool results, so it IS the live thread's current context size; CompletionTokens sums the
	// generated tokens across the turn's model steps; TotalTokens is their sum. Zero (omitted)
	// when the provider reported no usage. Serializable-only, so this rides the future SSE payload.
	PromptTokens     int `json:"prompt_tokens,omitempty"`
	CompletionTokens int `json:"completion_tokens,omitempty"`
	TotalTokens      int `json:"total_tokens,omitempty"`
}

// errSessionBusy rejects an overlapping turn: a Session drives one turn at a time.
var errSessionBusy = errors.New("session busy: a turn is already in progress")

// StreamTurn drives one full ReAct loop over the running history and returns a buffered channel
// (cap 64) of live events. Exactly one terminal event (final|error) is emitted, then the channel
// is closed; the caller MUST drain to close (cancel ctx to abort — draining continues until the
// terminal event lands), and must drain PROMPTLY: once the buffer fills, event sends block the
// agent's own goroutines (tool callbacks and the token reader), stalling the loop until the
// consumer catches up. Events are never dropped — a stalled consumer stalls the turn, it does
// not corrupt the step record. The mutex-guarded busy flag rejects overlapping turns.
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

	// Step 3 — callbacks: the existing persistence handler PLUS a live-events handler. The events
	// handler also buffers the current tool round into s.rc.pending so an abort right after a tool
	// can flush it (see the error branch below), which is why it carries the run context.
	te := &turnEvents{ch: ch, rc: s.rc}
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

	// Ordering guarantee: all token/tool events are emitted before the terminal event. After the
	// wait every step's token reader has folded its usage in, so the accumulated counts are final.
	te.wg.Wait()
	promptTok, completionTok, totalTok := te.usage()

	if runErr == nil && out != nil {
		// Step 5 — success.
		if perr := s.rc.persist(out); perr != nil {
			s.app.Logger().Error("persist final message", "run", s.run.Id, "err", perr)
		}
		s.history = append(s.history, out)
		s.history = capHistory(s.history)
		s.last = out
		ch <- Event{
			Kind: EventFinal, Content: out.Content,
			PromptTokens: promptTok, CompletionTokens: completionTok, TotalTokens: totalTok,
		}
		return
	}

	// Step 6 — error / cancel. Snapshot whatever the last model step streamed as the partial.
	partial := te.partial()
	canceled := ctx.Err() != nil || errors.Is(runErr, context.Canceled)

	// A completed tool round buffered this turn (turnEvents fills s.rc.pending) is present only when
	// the loop aborted right after a tool executed — MaxStep, cancellation, or another error — before
	// the next model input carried the round into the transcript. Recovering it here closes the same
	// audit-trail gap the one-shot Run() closes: the executed tool call is recorded even though the
	// turn did not finish.
	s.rc.mu.Lock()
	hasPending := len(s.rc.pending) > 0
	s.rc.mu.Unlock()

	switch {
	case hasPending:
		// Flush the buffered round (assistant tool-call message + its tool result[s]). Its assistant
		// message already carries this step's streamed content, so a separate partial row would
		// duplicate it — skip the partial. Roll the in-turn user message out of the in-memory history
		// (like the no-partial case) so the replayed alternation stays clean; the persisted transcript
		// keeps the complete round.
		s.rc.flushPending()
		s.history = s.history[:preLen]
		partial = "" // the round is fully captured in the transcript; nothing partial remains to render
	case partial != "":
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
	default:
		// No partial and no buffered round: roll the history back to its pre-turn length so no
		// dangling user message is left behind (this also fixes the dangling-user-message case on a
		// failed Turn).
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
	// Report whatever usage was counted even on a partial/canceled turn (0 is omitted).
	ch <- Event{
		Kind: EventError, Err: errText, Canceled: canceled, Partial: partial,
		PromptTokens: promptTok, CompletionTokens: completionTok, TotalTokens: totalTok,
	}
}

// turnEvents fans the eino model/tool callbacks of one turn into Event values on ch. Token
// readers run on their own goroutines (tracked by wg); lastText accumulates the CURRENT model
// step's text (reset per step) as the source of the cancel/error Partial. stepDone lets the
// tool callback wait for the preceding step's tokens to flush, so tool events land strictly
// between step-1 and step-2 tokens rather than racing them.
//
// It also mirrors the current tool round into rc.pending so an abort right after a tool can flush
// it (audit-trail integrity for the streaming path). stepAsst is the reconstructed assistant
// message of the current step (built from the streamed chunks — the streaming model node fires
// only OnEndWithStreamOutput, never OnEnd, so the round can't be captured the way the non-stream
// captureHandler does); asstRecorded guards against pushing it twice when one step makes several
// tool calls. rc may be nil (the buffering is then a no-op).
type turnEvents struct {
	ch           chan<- Event
	rc           *runCtx
	wg           sync.WaitGroup
	mu           sync.Mutex
	step         int
	lastText     strings.Builder
	stepDone     chan struct{}
	stepAsst     *schema.Message
	asstRecorded bool

	// Accumulated token usage across the turn's model steps (guarded by mu; folded in by each
	// step's token reader as it drains). promptTokens holds the LAST step's prompt count (the
	// current context size); completionTokens sums generated tokens across steps; totalTokens is
	// their sum. Read once, after wg.Wait(), via usage().
	promptTokens     int
	completionTokens int
	totalTokens      int
}

// recordPending buffers the current round into rc.pending: this step's assistant message once,
// then this tool's result. Called from the tool OnEnd, which the stepDone gate orders after the
// assistant message was reconstructed, so [assistant, tool] ordering holds. An abort right after
// the tool — before the next model OnStart resets the buffer — leaves the round for the flush.
func (te *turnEvents) recordPending(ctx context.Context, info *callbacks.RunInfo, result string) {
	if te.rc == nil {
		return
	}
	te.mu.Lock()
	asst := te.stepAsst
	already := te.asstRecorded
	te.asstRecorded = true
	te.mu.Unlock()

	te.rc.mu.Lock()
	if !already && asst != nil {
		te.rc.pending = append(te.rc.pending, asst)
	}
	te.rc.pending = append(te.rc.pending, &schema.Message{
		Role:       schema.Tool,
		Content:    result,
		ToolCallID: compose.GetToolCallID(ctx),
		ToolName:   toolName(info),
	})
	te.rc.mu.Unlock()
}

// partial returns the current/last step's accumulated text under the lock.
func (te *turnEvents) partial() string {
	te.mu.Lock()
	defer te.mu.Unlock()
	return te.lastText.String()
}

// usage returns the turn's accumulated token counts under the lock. Call only after wg.Wait(),
// once every step's token reader has folded its usage in.
func (te *turnEvents) usage() (prompt, completion, total int) {
	te.mu.Lock()
	defer te.mu.Unlock()
	return te.promptTokens, te.completionTokens, te.totalTokens
}

// handler builds the events callback (ChatModel stream output + Tool start/end/error).
func (te *turnEvents) handler() callbacks.Handler {
	modelHandler := &cbutils.ModelCallbackHandler{
		// OnStart resets the per-step reconstruction state AND the pending-round buffer. The buffer
		// is cleared HERE (model start, not stream end) because the input-side persistHandler flushes
		// the prior round via its delta at this same model OnStart — so once a later model call
		// begins, the prior round is already in the transcript and must not be flushed again. A model
		// call that never begins (the loop aborted right after a tool) leaves the buffer intact for
		// the abort flush. Resetting at stream end instead would miss a mid-loop model error, whose
		// OnStart persists the round via the delta but whose OnEndWithStreamOutput never fires.
		OnStart: func(ctx context.Context, _ *callbacks.RunInfo, _ *model.CallbackInput) context.Context {
			te.mu.Lock()
			te.stepAsst = nil
			te.asstRecorded = false
			te.mu.Unlock()
			if te.rc != nil {
				te.rc.mu.Lock()
				te.rc.pending = nil
				te.rc.mu.Unlock()
			}
			return ctx
		},
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
				// Track the last non-nil usage this step reports (typically only the final chunk
				// carries it; schema.ConcatMessageStream max-merges, but the live source is here).
				var stepUsage *schema.TokenUsage
				var chunks []*schema.Message
				for {
					chunk, err := output.Recv()
					if err != nil {
						break // io.EOF or a cancel/stream error ends this step's tokens.
					}
					if chunk == nil || chunk.Message == nil {
						continue
					}
					if rm := chunk.Message.ResponseMeta; rm != nil && rm.Usage != nil {
						stepUsage = rm.Usage // both pointers may be nil — checked above.
					}
					chunks = append(chunks, chunk.Message) // keep every chunk for the round reconstruction
					if chunk.Message.Content == "" {
						continue // tool-call chunks carry no content — no token to emit
					}
					delta := chunk.Message.Content
					te.mu.Lock()
					te.lastText.WriteString(delta)
					te.mu.Unlock()
					te.ch <- Event{Kind: EventToken, Step: myStep, Token: delta}
				}
				// Fold this step's usage into the turn totals under the lock. Steps fold in order —
				// the tool callback waits on `done` (closed by this goroutine's defer), so step N's
				// fold completes before step N+1's model call — so "last step's prompt wins" is
				// deterministic even though folds happen on separate goroutines.
				if stepUsage != nil {
					te.mu.Lock()
					te.promptTokens = stepUsage.PromptTokens
					te.completionTokens += stepUsage.CompletionTokens
					te.totalTokens = te.promptTokens + te.completionTokens
					te.mu.Unlock()
				}
				// Reconstruct this step's assistant message (content + any tool calls) from the streamed
				// chunks, so a tool round aborted before the next model input can still be recovered into
				// the transcript. Set before close(done) fires (deferred), so a tool callback that waits
				// on stepDone observes it. A stream that errored part-way yields a partial content message
				// with no tool call; recordPending only pushes it on a real tool OnEnd, so it is harmless.
				if len(chunks) > 0 {
					if msg, err := schema.ConcatMessages(chunks); err == nil {
						te.mu.Lock()
						te.stepAsst = msg
						te.mu.Unlock()
					}
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
			// Buffer the round (assistant tool-call message + this result) for abort recovery before
			// announcing the tool_end, so a cancellation observed right after does not race the record.
			te.recordPending(ctx, info, result)
			te.ch <- Event{Kind: EventToolEnd, Tool: toolName(info), CallID: compose.GetToolCallID(ctx), Result: result}
			return ctx
		},
		OnError: func(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
			text := ""
			if err != nil {
				text = err.Error()
			}
			// A tool error still closes the step, but rides in Err (not Result) so a renderer
			// can distinguish a failed call from a successful response.
			te.ch <- Event{Kind: EventToolEnd, Tool: toolName(info), CallID: compose.GetToolCallID(ctx), Err: text}
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

// Package agent is the eino ReAct loop slice (spec §6). It wires the provider-selected
// chat model, the enabled modules' tools (registered as eino InvokableTools via the merged
// toolcore registry, gated per §5.4 by exclusion from the slice), the DB-backed system prompt
// (§4.10/§6.1), and the single transcript-persistence callback (persist.go). Run() is the
// manual Phase-1 entry point.
//
// Scope note (interpretation, flagged in handoff): the spec envisioned the run-time system
// prompt RESOLVER living in internal/prompt (Resolve added to that package). internal/prompt
// is out of this slice's file scope, so the resolver is implemented here as systemPrompt(),
// reading the active prompts row and falling back to the exported prompt.Embedded() seed.
package agent

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	einoagent "github.com/cloudwego/eino/flow/agent"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"github.com/hsb3/deskkit/internal/core/config"
	"github.com/hsb3/deskkit/internal/core/toolcore"
	"github.com/hsb3/deskkit/internal/modules/librarian/prompt"
	"github.com/hsb3/deskkit/internal/modules/librarian/provider"
)

// chatModelFactory builds the provider chat model; overridable in tests to inject a stub
// so the loop can be exercised without a live API key.
var chatModelFactory = provider.NewChatModel

// systemPrompt resolves the ACTIVE prompt from the prompts collection (spec §4.10/§6.1),
// falling back to the embedded default seed, then prepends a short desk-facts preamble
// interpolated from config. Loaded at RUN START so GUI/REST edits apply to the next run.
// Nothing person-specific is compiled in — the preamble uses only configured values.
func systemPrompt(ctx context.Context, app core.App, cfg *config.Config) string {
	text := prompt.Embedded()
	recs, err := app.FindRecordsByFilter(
		"prompts", "key = {:k} && active = true", "-version", 1, 0,
		dbx.Params{"k": "librarian.system"})
	if err == nil && len(recs) > 0 {
		if c := recs[0].GetString("content"); c != "" {
			text = c
		}
	}
	return interpolateDeskFacts(text, cfg)
}

// interpolateDeskFacts prepends one preamble line naming the configured desk + root above
// whichever prompt text was resolved. Identity-neutral: values come from config only.
func interpolateDeskFacts(text string, cfg *config.Config) string {
	preamble := fmt.Sprintf(
		"You are the librarian for the desk %q, rooted at %s.\n\n",
		cfg.DeskName, cfg.DeskRoot)
	return preamble + text
}

// claudeToolCallChecker buffers the stream until a tool call is seen or the stream ends.
// Anthropic does not emit the tool call in the first streaming chunk, so eino's default
// checker can prematurely conclude "no tool call" (spec §6.1). Only used for Claude; nil for
// OpenAI/Gemini uses the default.
func claudeToolCallChecker(ctx context.Context, sr *schema.StreamReader[*schema.Message]) (bool, error) {
	defer sr.Close()
	for {
		msg, err := sr.Recv()
		if err == io.EOF {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		if len(msg.ToolCalls) > 0 {
			return true, nil
		}
	}
}

// newAgent builds the eino ReAct agent (spec §6.1). The MessageModifier loads the system
// prompt per run from the DB; MaxStep bounds the loop; the Claude checker is supplied only
// for the anthropic provider.
func newAgent(ctx context.Context, app core.App, chatModel model.ToolCallingChatModel, toolset []tool.BaseTool, cfg *config.Config) (*react.Agent, error) {
	acfg := &react.AgentConfig{
		ToolCallingModel: chatModel,
		ToolsConfig:      compose.ToolsNodeConfig{Tools: toolset},
		MessageModifier: func(ctx context.Context, in []*schema.Message) []*schema.Message {
			return append([]*schema.Message{schema.SystemMessage(systemPrompt(ctx, app, cfg))}, in...)
		},
		MaxStep: cfg.AgentMaxStep, // default 12
	}
	if cfg.LLMProvider == "anthropic" {
		acfg.StreamToolCallChecker = claudeToolCallChecker
	}
	return react.NewAgent(ctx, acfg)
}

// buildTools builds eino InvokableTools for the merged registry's gated tool set
// (toolcore.AgentTools(cfg)): the §5.4 LIBRARIAN_AUTONOMOUS_WRITES gate is enforced by
// EXCLUSION FROM THE SLICE (apply_fix absent unless gated on; restore never present) — not
// re-implemented here. On top of that gate, the slice is further narrowed to the librarian
// module only (toolcore.SelectByModules(..., "librarian")): the in-binary eino loop's system
// prompt covers the librarian tool family exclusively, so when PM is enabled and its twelve
// tools join the merged registry, they must NOT reach this loop (ADR 0014(c)) — the PM tools
// are surfaced only via the MCP/CLI surfaces, which build their own module-appropriate slices.
// Each spec's NewEinoTool derives the tool's JSON schema from its input struct's jsonschema tags
// via eino's InferTool (moved into toolcore.New).
func buildTools(app core.App, cfg *config.Config) ([]tool.BaseTool, error) {
	var out []tool.BaseTool
	for _, spec := range toolcore.SelectByModules(toolcore.AgentTools(cfg), "librarian") {
		t, err := spec.NewEinoTool(app, cfg)
		if err != nil {
			return nil, fmt.Errorf("register tool %q: %w", spec.Name, err)
		}
		out = append(out, argNormalizingTool{InvokableTool: t})
	}
	return out, nil
}

// argNormalizingTool normalizes an empty/whitespace-only ArgumentsInJSON to "{}" before the
// wrapped InferTool tool unmarshals it. A zero-argument tool call (e.g. sweep, patrol) streams
// no argument deltas, so under the STREAMING path the concatenated arguments are "" and
// json.Unmarshal("") fails inside eino's InferTool wrapper ("[LocalFunc] failed to unmarshal
// arguments in json"), killing the whole turn. (The non-streaming Generate path received "{}"
// from the provider's complete message, so this was a streaming-introduced regression.) The
// wrapped InvokableTool is embedded so Info() and every other interface surface pass through
// unchanged; only InvokableRun is intercepted — and because these tools implement solely
// InvokableTool, ToolsNode derives its stream endpoint from InvokableRun too, so this single
// point fixes both paths. Applied to whatever slice AgentTools returns, so the §5.4 write-gating
// is untouched (it still governs which tools are in the slice).
type argNormalizingTool struct {
	tool.InvokableTool
}

func (t argNormalizingTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	if strings.TrimSpace(argumentsInJSON) == "" {
		argumentsInJSON = "{}"
	}
	return t.InvokableTool.InvokableRun(ctx, argumentsInJSON, opts...)
}

// Run executes one agent run end-to-end (spec §6.5). It opens an agent_runs row (status
// running), builds the provider model + gated tool slice + agent, registers the single
// persistence callback, drives the ReAct loop with agent.Generate, persists the final
// assistant message, and finalizes the run row (succeeded | failed | blocked, step_count).
// Run executes one agent loop. On success the final assistant text is returned
// so callers (the CLI) can print it; the full transcript is in the messages
// collection either way.
func Run(ctx context.Context, app core.App, cfg *config.Config, trigger, input string) (string, error) {
	// Same per-run store re-resolve NewSession does: under serve the claimer drives Run from a
	// long-lived Config, so a provider/model saved from the browser must reach the next run.
	cfg = config.WithStore(app, cfg)

	run, err := createAgentRun(app, trigger, input, cfg)
	if err != nil {
		return "", err
	}
	rc := &runCtx{app: app, cfg: cfg, runID: run.Id}

	chatModel, err := chatModelFactory(ctx, app, cfg)
	if err != nil {
		return "", failRun(app, run, rc, err)
	}
	toolset, err := buildTools(app, cfg)
	if err != nil {
		return "", failRun(app, run, rc, err)
	}
	ag, err := newAgent(ctx, app, chatModel, toolset, cfg)
	if err != nil {
		return "", failRun(app, run, rc, err)
	}

	handler := rc.persistHandler() // the ONE persistence mechanism (persist.go)
	capture := rc.captureHandler() // in-memory current-round buffer, flushed on any abort
	out, genErr := ag.Generate(ctx, []*schema.Message{schema.UserMessage(input)},
		einoagent.WithComposeOptions(compose.WithCallbacks(handler, capture)))

	// The final assistant message (out) is the loop's output; it never appears in a
	// subsequent model INPUT, so it is not captured by the input-side callback. Persist it
	// here on success.
	if genErr == nil && out != nil {
		if perr := rc.persist(out); perr != nil {
			app.Logger().Error("persist final message", "run", run.Id, "err", perr)
		}
	} else if genErr != nil {
		// The loop aborted right after a tool may have executed. The input-side callback flushes a
		// round only on the NEXT model call, which never came, so the assistant tool-call message
		// and its tool result may still be buffered in rc.pending. Flush them so the transcript
		// records the tool call whose effect (e.g. a revisions row) really landed — the
		// audit-trail integrity fix.
		//
		// The flush fires on ANY abort, not just MaxStep: a context cancellation that lands between
		// a tool's OnEnd and the next model's OnStart (the graph runner aborts before submitting the
		// next model task) leaves the same round buffered. The invariant is "a real tool call
		// executed and was not yet flushed", not which error ended the loop. flushPending is a no-op
		// when the buffer is empty (a fresh model OnStart reset it, or no tool ran), so a pre-tool
		// failure persists nothing spurious.
		rc.flushPending()
	}
	final := ""
	if genErr == nil && out != nil {
		final = out.Content
	}
	return final, finishRun(app, run, rc, out, genErr)
}

// Package agent is the eino ReAct loop slice (spec §6). It wires the provider-selected
// chat model, the six tools (registered as eino InvokableTools, gated per §5.4 by
// exclusion from the slice), the DB-backed system prompt (§4.10/§6.1), and the single
// transcript-persistence callback (persist.go). Run() is the manual Phase-1 entry point.
//
// Scope note (interpretation, flagged in handoff): the spec envisioned the run-time system
// prompt RESOLVER living in internal/prompt (Resolve added to that package). internal/prompt
// is out of this slice's file scope, so the resolver is implemented here as systemPrompt(),
// reading the active prompts row and falling back to the exported prompt.Embedded() seed.
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	toolutils "github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
	einoagent "github.com/cloudwego/eino/flow/agent"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"github.com/example/pocket-librarian/internal/config"
	"github.com/example/pocket-librarian/internal/prompt"
	"github.com/example/pocket-librarian/internal/provider"
	"github.com/example/pocket-librarian/internal/tools"
)

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

// buildTools wraps the frozen tools.* core functions as eino InvokableTools via
// utils.InferTool, marshaling each typed result to a JSON string. The per-run tool slice is
// selected by tools.AgentTools(cfg): the §5.4 LIBRARIAN_AUTONOMOUS_WRITES gate is enforced by
// EXCLUSION (apply_fix absent unless gated on; restore never present) — not re-implemented
// here. InferTool derives each tool's JSON schema from the input struct's jsonschema tags.
func buildTools(app core.App, cfg *config.Config) ([]tool.BaseTool, error) {
	builders := map[string]func(name, desc string) (tool.InvokableTool, error){
		"sweep": func(name, desc string) (tool.InvokableTool, error) {
			return toolutils.InferTool(name, desc, func(ctx context.Context, in tools.SweepInput) (string, error) {
				return jsonResult(tools.Sweep(ctx, app, cfg, &in))
			})
		},
		"patrol": func(name, desc string) (tool.InvokableTool, error) {
			return toolutils.InferTool(name, desc, func(ctx context.Context, in tools.PatrolInput) (string, error) {
				return jsonResult(tools.Patrol(ctx, app, cfg, &in))
			})
		},
		"propose_fix": func(name, desc string) (tool.InvokableTool, error) {
			return toolutils.InferTool(name, desc, func(ctx context.Context, in tools.ProposeFixInput) (string, error) {
				return jsonResult(tools.ProposeFix(ctx, app, cfg, &in))
			})
		},
		"apply_fix": func(name, desc string) (tool.InvokableTool, error) {
			return toolutils.InferTool(name, desc, func(ctx context.Context, in tools.ApplyFixInput) (string, error) {
				return jsonResult(tools.ApplyFix(ctx, app, cfg, &in))
			})
		},
		"restore": func(name, desc string) (tool.InvokableTool, error) {
			return toolutils.InferTool(name, desc, func(ctx context.Context, in tools.RestoreInput) (string, error) {
				return jsonResult(tools.Restore(ctx, app, cfg, &in))
			})
		},
		"query": func(name, desc string) (tool.InvokableTool, error) {
			return toolutils.InferTool(name, desc, func(ctx context.Context, in tools.QueryInput) (string, error) {
				raw, err := tools.Query(ctx, app, cfg, &in)
				if err != nil {
					return "", err
				}
				return string(raw), nil // query already returns a JSON document
			})
		},
	}

	var out []tool.BaseTool
	for _, spec := range tools.AgentTools(cfg) {
		b, ok := builders[spec.Name]
		if !ok {
			continue // spec not registered as an agent tool (defensive; should not happen)
		}
		t, err := b(spec.Name, spec.Description)
		if err != nil {
			return nil, fmt.Errorf("register tool %q: %w", spec.Name, err)
		}
		out = append(out, t)
	}
	return out, nil
}

// jsonResult marshals a tool's typed result to a JSON string (empty string on error).
func jsonResult[T any](r T, err error) (string, error) {
	if err != nil {
		return "", err
	}
	b, merr := json.Marshal(r)
	if merr != nil {
		return "", merr
	}
	return string(b), nil
}

// Run executes one agent run end-to-end (spec §6.5). It opens an agent_runs row (status
// running), builds the provider model + gated tool slice + agent, registers the single
// persistence callback, drives the ReAct loop with agent.Generate, persists the final
// assistant message, and finalizes the run row (succeeded | failed | blocked, step_count).
func Run(ctx context.Context, app core.App, cfg *config.Config, trigger, input string) error {
	run, err := createAgentRun(app, trigger, input, cfg)
	if err != nil {
		return err
	}
	rc := &runCtx{app: app, cfg: cfg, runID: run.Id}

	chatModel, err := provider.NewChatModel(ctx, cfg)
	if err != nil {
		return failRun(app, run, rc, err)
	}
	toolset, err := buildTools(app, cfg)
	if err != nil {
		return failRun(app, run, rc, err)
	}
	ag, err := newAgent(ctx, app, chatModel, toolset, cfg)
	if err != nil {
		return failRun(app, run, rc, err)
	}

	handler := rc.persistHandler() // the ONE persistence mechanism (persist.go)
	out, genErr := ag.Generate(ctx, []*schema.Message{schema.UserMessage(input)},
		einoagent.WithComposeOptions(compose.WithCallbacks(handler)))

	// The final assistant message (out) is the loop's output; it never appears in a
	// subsequent model INPUT, so it is not captured by the input-side callback. Persist it
	// here on success. On error/blocked, the transcript up to the last model call is already
	// persisted by the callback.
	if genErr == nil && out != nil {
		if perr := rc.persist(out); perr != nil {
			app.Logger().Error("persist final message", "run", run.Id, "err", perr)
		}
	}
	return finishRun(app, run, rc, out, genErr)
}

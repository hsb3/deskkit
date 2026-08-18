package tools

import (
	"context"

	"github.com/pocketbase/pocketbase/core"

	"github.com/hsb3/deskkit/internal/core/config"
	"github.com/hsb3/deskkit/internal/core/toolcore"
)

// Module is the owning module name every profile ToolSpec carries. It is the value a shared MCP
// mount declares in MCP_MODULES to be narrowed to this family alone.
const Module = "profile"

// Specs returns the profile module's four tool specs, in a stable order. The tool NAMES are
// frozen — skills, docs, and host configurations build on them.
//
// Gate encoding: all four are READ-ONLY (WritesFiles=false — none of them touches a desk file
// or the store), so each is AgentDefault unconditionally and none is AgentGated. The write
// gates that narrow the other modules' surfaces have nothing to act on here.
func Specs() []toolcore.ToolSpec {
	return []toolcore.ToolSpec{
		toolcore.New[GetInput](Module, "profile_get",
			"Resolve a dotted key of the desk's personalization profile to its scalar value; fails loudly, naming the available keys, when the key is absent or empty.",
			false, true, false, asInvoke(ProfileGet)),
		toolcore.New[ValidateInput](Module, "profile_validate",
			"Validate the desk's personalization profile (or a given file) against schema v1, returning the verdict, the violations, and the profile path.",
			false, true, false, asInvoke(ProfileValidate)),
		toolcore.New[RenderInput](Module, "template_render",
			"Render a template by substituting its {{profile.<key>}} / {{env.<VAR>}} placeholders; a placeholder with no default that resolves empty is refused.",
			false, true, false, asInvoke(TemplateRender)),
		toolcore.New[IndexInput](Module, "knowledge_index",
			"Index the desk's background-knowledge folder: every markdown file with its size and word count, plus file content up to a byte budget.",
			false, true, false, asInvoke(KnowledgeIndexTool)),
	}
}

// asInvoke adapts a typed tool implementation to toolcore's `any`-returning invoke signature.
// It returns an UNTYPED nil on error, so a failed call never hands the surfaces a typed nil
// pointer boxed in a non-nil interface.
func asInvoke[I, O any](fn func(context.Context, core.App, *config.Config, *I) (*O, error)) func(context.Context, core.App, *config.Config, *I) (any, error) {
	return func(ctx context.Context, app core.App, cfg *config.Config, in *I) (any, error) {
		out, err := fn(ctx, app, cfg, in)
		if err != nil {
			return nil, err
		}
		return out, nil
	}
}

// ToolNames lists the frozen profile tool ids in registration order (test + docs anchor).
func ToolNames() []string {
	return []string{"profile_get", "profile_validate", "template_render", "knowledge_index"}
}

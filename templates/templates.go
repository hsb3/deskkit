// Package templates holds the librarian's embedded, approved content sources: the two
// fixer templates (frontmatter-universal, pointer-stub) and the embedded default system
// prompt. Content that lands on disk during a fix comes ONLY from these templates
// (spec §5.4 "templates only" boundary); the system prompt is the //go:embed'd seed for
// the prompts collection (spec §6.1). Embeds live here (not in cmd/) because //go:embed
// cannot reach across `..` — a single package co-located with the files keeps the seam clean.
package templates

import (
	_ "embed"
	"strings"
)

//go:embed frontmatter-universal.md
var FrontmatterUniversal string

//go:embed pointer-stub.md
var PointerStub string

//go:embed librarian-system-prompt.txt
var SystemPrompt string

// Render strips a leading HTML comment block (the source annotation) and substitutes
// {{key}} placeholders from subs. Ported from the PoC render_template: content comes
// from the template only — callers never synthesize prose.
func Render(tmpl string, subs map[string]string) string {
	if strings.HasPrefix(tmpl, "<!--") {
		if end := strings.Index(tmpl, "-->"); end >= 0 {
			tmpl = strings.TrimLeft(tmpl[end+3:], "\n")
		}
	}
	for k, v := range subs {
		tmpl = strings.ReplaceAll(tmpl, "{{"+k+"}}", v)
	}
	return tmpl
}

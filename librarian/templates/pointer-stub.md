<!--
template: pointer-stub.md
Universal frontmatter fence (type/created/updated/tags) plus the desk's `synopsis`
convention. `type: pointer` marks a collapsed/relocated doc per the desk's
"collapses to a pointer" rule. Body is exactly one line; the fixer substitutes
{{placeholders}} only — no prose. The leading comment block is stripped before the
stub is written (see templates.Render).
-->
---
type: pointer
created: {{date}}
updated: {{date}}
tags: []
synopsis: "Pointer stub — content moved to {{new_path}}"
---
Moved to {{new_path}}.

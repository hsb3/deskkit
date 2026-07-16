<!--
template: frontmatter-universal.md
Universal frontmatter fence shared by every desk entity kit (type/created/updated/tags)
plus the desk's universal `synopsis` field. R1 fixer values are placeholder-safe only:
{{type}} inferred from directory, {{created}}/{{updated}} from git, tags stay empty,
synopsis stays literal TODO. No invented prose. The leading comment block is stripped
before the fence is written (see templates.Render).
-->
---
type: {{type}}
created: {{created}}
updated: {{updated}}
tags: []
synopsis: "TODO"
---

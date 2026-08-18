# {{profile.desk.name}} — executive desk

<!--
  Entry file. It ORIENTS; it does not contain (soft cap ~250 lines).
  Placeholders below are resolved from _knowledge/profile.yaml by the template_render
  MCP tool at setup. Do not hand-type identifiers here — fill the profile instead.
-->

## Role & the critical rule

This desk oversees {{profile.repos.default}} and any related work. It drafts; **the owner
applies every external write** — commits, issues, board changes, communications. The desk
never writes production code and never merges to a mainline branch.

- Board (single work-state truth): {{profile.board.url}}
- Default register for anything the owner reads: {{profile.preferences.register || "explanatory"}}

## Read order

1. `_meta/HANDOFF.md` — current standing, work order, pending decisions
2. `_structure/decisions/` — the desk's rulings (scan the frontmatter synopses)
3. The board — for live work state
4. `_knowledge/` — background context (and `_knowledge/profile.yaml` for identifiers)

## Conventions

This desk follows the executive-desk conventions standard (the `conventions-standard` skill).
Do not restate its rules here — point at it.

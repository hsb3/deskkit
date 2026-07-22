---
type: analysis
status: active
created: 2026-07-21
updated: 2026-07-22
tags: [model-simulations, greenfield, desk-setup]
synopsis: Scenario 3 walkthrough — a greenfield desk from scaffold through first sweep/patrol/PM-item, traced against the v1 model; scaffold+template_render steps tabletop, the store tail scripted.
---

*Scenario 3 of the #126 model simulations (v1 half). Follows the shipped Greenfield runbook (K23,
`plugin/claude-plugin/skills/desk-setup/SKILL.md`). The scaffold copy and `template_render`
placeholder materialization run in the TS/MCP lane (tabletop here); the store-touching tail —
first sweep/patrol/PM-item — is scripted against a throwaway conformant desk.*

Status: active (2026-07-21)

## Operator story

An operator stands up a brand-new executive desk: names and places it outside the repos it
oversees, copies the standard-free scaffold, fills a profile, renders the placeholders, completes
the entry brief, then runs the desk's first librarian baseline and creates its first PM item.

## Coverage

- **Tabletop** — K23 steps 1–7 (name/place, fix goal+role, copy scaffold, create profile,
  `template_render`, fill entry brief + `_meta/` instruments, thin app-instructions). These are
  human-authoring + TS/MCP-tool steps; no `deskkit` store operation, no LLM key.
- **Scripted** — the desk's first store operations once it is scaffolded and conformant
  (`probes/probe-s3-greenfield.sh`, output `probes/output/out-s3.txt`): a minimal conformant desk
  (CLAUDE.md `type: home`, `_meta/README` `type: readme`, `_meta/HANDOFF`, `_structure/decisions/
  README`) is swept, patrolled, and given its first PM item.

## Step-by-step trace

| # | Operator action | Surface behavior | Store entities / fields | v1 verdict | v2 delta |
|---|---|---|---|---|---|
| 1 | Name + place the desk outside overseen repos (K23.1) | procedural — desk on a cloud drive, never nested in a watched repo | none | **OK** (tabletop) | — |
| 2 | Fix goal + role; critical rule "desk drafts, owner applies" (K23.2) | procedural | none | **OK** (tabletop) | — |
| 3 | Copy the standard-free scaffold (K23.3) | `assets/template/` skeleton copied; no conventions prose baked in (K25) | none | **OK** (tabletop) | — |
| 4 | Create + fill `_knowledge/profile.yaml` (K23.4) | identifiers live only here | validated against `schema/profile.schema.yaml` (TS lane) | **OK** (tabletop) | — |
| 5 | `template_render` MCP tool (K23.5) | resolves `{{profile.…}}`; **fail-loud** on a required placeholder with no default | TS/MCP lane; no PocketBase store | **OK** (tabletop — TS/MCP, not scripted here) | — |
| 6 | Fill entry brief + `_meta/` instruments (K23.6) | CLAUDE.md orient-only (K8), HANDOFF, README | none | **OK** (tabletop) | — |
| 7 | Thin app/project-instructions (K23.7) | host instruction field points at the desk files | none | **OK** (tabletop) | — |
| 8 | First store touch (`query summary`) on a never-migrated store | `{"files_total":0,"open_findings_total":0}` — self-inits, no bare `sql.ErrNoRows` | store auto-migrates at `requireConfig` | **OK** [scripted] | — |
| 9 | First `sweep` | `{"total":5,"created":5}` | `files` (5 rows) | **OK** [scripted] | — |
| 10 | First `patrol` | `{"findings_new":0,"by_rule":{}}` — a conformant scaffold is clean | `patrol_findings` (none), `patrol_log` (1 row) | **OK** [scripted] | — |
| 11 | First PM item (`pm create --type task`) | item created; `pm context` → `{"active":1,"blocked":0}` | `items`, `transitions` | **OK** [scripted] | — |

## Findings from this scenario

**No new deficiencies.** The greenfield path is clean: a conformant scaffold produces zero patrol
findings, self-init works, and the first PM item lands. Two observations carried forward, neither
gate-bearing:

- **The greenfield desk starts with no `desk_config` row**, so PM gates fall back to the shipped
  `DefaultRulesYAML` (only `decision review→terminal` and `task work→review`). This minimal
  default is self-documented in `defaults.go` as a "KNOWN UNAUTHORED DESIGN GAP" pending an owner
  ruling — already flagged in source, not a new finding, but the v2 model should settle the
  default gate set.
- **Two lanes, two setup acts.** `template_render` (TS/MCP) materializes the desk; the librarian
  baseline (Go binary) is a separate step requiring `deskkit` on PATH and — for the PM tools over
  MCP — a fresh session. The runbooks state this; noted so the v2 model's setup story accounts
  for the lane boundary.

## v2 assessment (this scenario against `docs/element-model-v2-draft.md`)

The `v2 delta` column reads "—": greenfield setup mechanics (scaffold copy, `template_render`, the
first sweep/patrol/PM-item) are model-agnostic. But the greenfield path is where the v2 model's
**content vocabulary** would first bite:

- **V2-D3 (Low)** — a v2 desk holds ~15 net-new element/doc types (`goal`, `source`, `deliverable`,
  `engineering-spec`, …). The greenfield runbook produces a *conformant* scaffold with **zero patrol
  findings** (step 10) precisely because every scaffold doc has a known `dir_kind` + type. The v2
  model names no directory-placement or patrol/`entity_type` classification for its new types, so a
  v2 scaffold cannot yet be made conformant the same way. Disposition: amendment-needed (§13
  librarian-taxonomy line); build-time work §13 already defers, but the model should *name* it. Not
  blocking.
- **Default gate set (pre-existing).** The greenfield desk starts with no `desk_config`, falling back
  to the minimal shipped gate set — the same "KNOWN UNAUTHORED DESIGN GAP" **V2-D1 (#197)** asks the
  v2 model to settle. No new finding here; folded into #197.

No greenfield-specific v2 deficiency beyond these. The setup *flow* is unchanged by v2.

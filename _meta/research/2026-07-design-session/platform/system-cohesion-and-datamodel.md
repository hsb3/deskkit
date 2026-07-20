---
type: plan
status: active
created: 2026-07-20
updated: 2026-07-20
tags: [dev-tooling, extenders, estate, cohesion, datamodel, pocketbase, kb, pm, frontend, desk-standard, naming]
synopsis: "Macro review of the extender estate as one cohesive system (dotfiles-agents + desk-standard + this desk), triggered by desk-standard becoming a core co-product. Three threads: (1) bundle/naming cohesion across the two products; (2) the datamodel — broadened to the exec desk's KB/PM state beyond GitHub issues, which settles the store on the already-committed PocketBase (0009/0015; census stays YAML-derived; DuckDB out); (3) the KB/PM frontend, not yet built (design open). Extends decision 0021-F7's cohesion+value eval from desk-standard-internal to the whole estate."
---

# System cohesion + estate datamodel — macro review

_Status: active — 2026-07-20. Step-back review before shipping more extenders. Trigger: `desk-standard`
is now a core part of the toolset, so the estate must be evaluated as ONE system, not two lineups.
Broadens decision `0021`-F7 (cohesion + value eval, currently scoped to desk-standard) to the estate._

## The estate today (3 surfaces)

| Surface | Role | Ships | Targeting |
| --- | --- | --- | --- |
| **dotfiles-agents** | CC extender marketplace | code-desk · exec-desk · foreman-kit · github-project-board + standalones | Claude-Code-only; **opencode planned but deferred** (mirrors desk-standard #12) |
| **desk-standard** | Executive-desk *governor* | `plugin/` (4 skills + MCP) · `desk-pm` plugin · `librarian`/deskkit Go binary (embedded PocketBase) · schema v1 | **Multi-target already** (claude-plugin + opencode); heading to 1.0.0 (`0021`) |
| **dev-tooling-desk** | The exec-desk *instance* | this planning desk · `_knowledge/extenders.yaml` (census) · live `pb_data/` | — |

Cutover is complete (2026-07-18): both dotfiles-agents desk bundles + foreman-kit are installed and LIVE
(per `extenders.yaml` status). So the build is done; this review is about *coherence*, not delivery.

## Thread 1 — bundle / naming cohesion

Questions to rule (feed the naming of the three new skills #137/#138/#139 — hold their bundle-homes until settled):

- **PM ownership overlap.** desk-standard ships `desk-pm` (pm-triage / session-open / advance-item + PM
  module in deskkit); dotfiles-agents ships `github-project-board` + `board-triage`. Two "work-tracking"
  surfaces. Which owns PM, and where's the seam? (extenders.yaml already flags "exec desk needs a broader
  PM system — not yet built".)
- **Governor vs driver split.** desk-standard *states* conventions (`conventions-standard`); dotfiles-agents
  *drives* workflows. #137 `decision-loop` sits exactly on this seam — confirm it cross-references rather
  than restates the governor.
- **Post-#147 standalones.** `diagrams` / `obsidian-toolkit` / `owner-signoff` were added beyond the 0020
  table. Do they satisfy the 0020 naming principle (cross-desk ⇒ standalone), or need re-homing?
- **Targeting parity.** desk-standard is multi-target now; dotfiles-agents is CC-only-for-now. `gate:cross-tool`
  ("opencode parity") is therefore a **live deferred promise**, not dead intent — the root `plugins/` layout
  (which replaced `targets/claude-code/plugins` at the CC-only rebuild) will need revisiting when opencode lands.

## Thread 2 — datamodel (broadened 2026-07-20: it's KB/PM, not just the census)

Reframed by Henry: the exec desk's project management **spans well beyond GitHub issues**. The desk *sets*
release requirements that graduate to the GitHub board for the code desk to build (per `0021`); it does not run
day-to-day off that board. Its own PM/KM is broader — action items with no repo ("schedule a meeting with Tony
re: duckdb vs pocketbase"), people and their opinions, meetings, decisions, knowledge notes.

**Two data natures — don't conflate them:**

- **Derived census** (extenders, bundles, repos) — regenerable from manifests (`extenders.yaml` schema v2 +
  dotfiles-agents `primitives-core.yaml` + `gh` state). Truth stays in the files; a DB is a rebuildable cache.
- **Authored PM/KM state** (tasks incl. non-code, people, meetings, decisions, notes) — lives nowhere else; it
  is edited, stateful, workflow-bearing. Needs an authoritative store **with a write/edit UI**.

**Store conclusion:** the authored PM/KM half settles the question — it wants **PocketBase**, already the
committed desk backend (`0009`/`0015`, embedded in deskkit), already carrying the PM work-graph (desk-pm / the
document-gated phase machine), already slated for a **webapp in 1.0 scope** (`0021`-F3, fed by the ADR-0004
streaming layer). DuckDB is out (read-only analytical; no write/UI story). A derived SQLite is fine only as an
*inspector for the census half*, never as the PM/KM store. Net: one PocketBase — census synced in read-mostly,
PM/KM authored natively.

**Unifying primitive: the typed entity + typed relation.** Person · topic · decision · task · meeting ·
extender, each an entity; edges like *task—about→topic*, *meeting—with→person*, *decision—rules→topic*. That is
what makes "meet with Tony about duckdb-vs-pocketbase" link the person, the topic, the pending decision, and the
task. PocketBase expresses it as relations; the markdown desk already expresses it as `[[wikilinks]]` — shared
entity IDs bridge the two.

## Thread 3 — the KB/PM frontend (not yet built; design open, ideas welcome)

> **Expanded 2026-07-20:** the frontend is one surface of a multi-surface **agent-integration toolkit** (one
> data model + workflows, exposed via harness persona / CLI / REPL / TUI / webapp). Design tracked separately
> in [../desk-platform/plan.md](../desk-platform/plan.md); the options below feed its webapp surface.

The missing piece. Store is effectively decided; the human surface is the real build. Options:

- **A · PB Admin UI (zero build)** — interim "see/edit today" over the collections; generic, no PM/KM shape.
- **B · React SPA served by PocketBase** — the `0021`-F3 webapp; rich kanban / calendar / entity-graph views;
  most work, most durable. React-vs-Go-route is an open sub-decision (`0021`-F3; ADR-0001 preferred a Go route).
- **C · Go + htmx/templ embedded in the PB binary** — single-binary, no npm build, ADR-0001's lean; ideal for a
  single-user local desk; culturally consistent with deskkit (`0009`: no TS transpile, no npm deps). Recommended
  starting point for the PM half.
- **D · Split surfaces** — **Obsidian as the KB frontend** over the markdown desk (graph, backlinks, search for
  free — `obsidian-toolkit` already ships) + the PB app for structured PM only; joined by shared entity IDs.
  Avoids rebuilding KB navigation that already exists. Recommended for the KB half.

**Design idea — capture is omni-surface, storage is one.** NL intake ("schedule a meeting with Tony…") parsed
**chat-to-schema** into a structured record — from the webapp *and* from within Claude Code (a skill writing to
PB's REST API). desk-pm is the *agent* surface on the work graph; the webapp is the *human* surface; PocketBase
is the shared store. One unified board shows GitHub-synced code work (read-only mirror cards) beside native desk
items (editable) — directly answering "PM spans more than GitHub." The census/cohesion analysis (Thread 1)
becomes read-only dashboards in the same app.

**Recommendation:** PB Admin UI now (see/edit today) → **C for the PM app + D for the KB** (Obsidian), reserving
B (React) only if PM views get genuinely rich. **[Owner input welcome: B vs C, and whether to split KB to D.]**

## Near-term dotfiles-agents execution

Live issue state is a generated view — run `_meta/plans/dotfiles-agents/_utils/sequence.py` / `reconcile.py`;
do not freeze a list here. Actionable now:

- **Close-out** — run #111's install-smoke proof, then close #111 / #134 / #121 / #135; refresh the stale
  dotfiles-agents HANDOFF.
- **Forward skill wave** (independent; names pending Thread 1): **#138 release-loop** (code-desk) ·
  **#137 decision-loop** (exec-desk) · **#139 claude-code-expertise** (standalone). Per-skill build plans land
  in `_meta/plans/dotfiles-agents/<slug>/plan.md` when each starts.

## Provenance

Trigger: session review 2026-07-20. Related rulings: `0020` (desk-set restructure) · `0021`-F3/F7
(desk-standard 1.0.0 — webapp + cohesion/value eval) · `0016` (extenders-estate fresh start) · `0009`/`0015`
(PocketBase desk backend). Census: `_knowledge/extenders.yaml`.

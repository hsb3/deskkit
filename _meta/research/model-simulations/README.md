---
type: analysis
status: active
created: 2026-07-21
updated: 2026-07-22
tags: [model-simulations, design-session, schema-v2]
synopsis: Index for the #126 model-simulations work — the scenario set, method, coverage, and where the deficiency roll-up lives. Both halves complete: v1 (shipped model, scripted+tabletop) and v2 (draft element model, tabletop).
---

*Purpose: walk realistic project scenarios through both the shipped v1 element model and the draft
v2 element model to surface deficiencies in the data model or its components before the v2 model
finalizes (#126). **Both halves are now complete** — the v1 half (scripted + tabletop against the
running system) and the v2 half (tabletop against `docs/element-model-v2-draft.md`, unblocked once
`element-model-revision` #125 merged the model). See "v2 half" below.*

Status: active (2026-07-22) — v1 + v2 halves complete.

## What this is

Issue #126 (`gate:v2-final`, parent epic #130) carries the owner directive attached to ADR 0009
decision (3): run simulations against both the v1 and v2 data models to surface deficiencies
before v2 is finalized. The owner blessed all four recommended defaults as written (sign-off
batch 2026-07-21, comment on #126):

1. **Scenario set** — four flows derived from the real workflows
   (`../2026-07-design-session/workflows.md`) + the shipped setup runbooks.
2. **Method** — tabletop (paper) walkthrough first; scripted v1 probes where cheap, against a
   throwaway scratch desk built on the `librarian/verify.sh` pattern (never a real desk).
3. **Deficiency report** at `deficiency-report.md` (this folder).
4. **Pass bar** — mirrors the schema-v2 epic close-when: every deficiency is either filed as its
   own issue or explicitly recorded as accepted-risk with rationale, before #125 is finalized.

## The scenario set

The four shipped-flow scenarios are walked against **both** models (v1 in the main columns; v2 in
each doc's `v2 delta` column + a per-scenario "v2 assessment" section). Scenarios 5 & 6 are the v2
model's **own** native walkthroughs (draft §10), the v2-native pair the issue's Deliverable A
requires.

| # | Scenario | Walkthrough doc | Derived from |
|---|---|---|---|
| 1 | Librarian chain: sweep → patrol → propose_fix → apply_fix → restore + findings disposition | [scenario-1-librarian-chain.md](scenario-1-librarian-chain.md) | workflows.md §1.1–1.3, §1.5 |
| 2 | PM gated transition + cascade, with a gate refusal | [scenario-2-pm-gated-cascade.md](scenario-2-pm-gated-cascade.md) | workflows.md §2.2–2.3 |
| 3 | Greenfield desk setup (K23) | [scenario-3-greenfield.md](scenario-3-greenfield.md) | desk-setup SKILL "Greenfield runbook (K23)" |
| 4 | Brownfield adoption (9-phase) | [scenario-4-brownfield.md](scenario-4-brownfield.md) | brownfield-adoption SKILL "Phase runbook" |
| 5 | **v2 software slice** (tabletop) | [scenario-5-v2-software.md](scenario-5-v2-software.md) | `docs/element-model-v2-draft.md` §10 (software), §3–§7 |
| 6 | **v2 research slice** (tabletop) | [scenario-6-v2-research.md](scenario-6-v2-research.md) | `docs/element-model-v2-draft.md` §10 (research), §5/§8/§9 |

## Method & coverage (no silent caps)

Each walkthrough is a step-by-step tabletop trace against the v1 element model, every step
marked **OK / FRICTION / DEFICIENCY**. Where a step is a deterministic, non-interactive
`deskkit` operation, it was **scripted against a throwaway scratch desk** and the actual output
is cited; where it needs human judgment, an LLM key, or the TS/MCP `template_render` lane, it is
**tabletop-only** and marked as such.

- **Scripted** (real `deskkit` runs, outputs under `probes/output/`): the entire librarian chain
  + disposition surface (S1); the entire PM gated-transition + cascade lifecycle (S2); the
  greenfield store-touching tail — first sweep/patrol/PM-item (S3); the brownfield Phase 8
  librarian baseline (S4).
- **Tabletop-only** (stated per step): scaffold copy + `template_render` placeholder
  materialization (S3 K23 steps 3–7, TS/MCP lane); every human-judgment phase of brownfield
  (S4 phases 1–7, 9 — lock, inventory, disposition table, approval gate, init, author
  instruments, take stock); all agent-loop / LLM-driven steps (no LLM key available — never
  faked).

Probes are reproducible: `bash probes/<probe>.sh /path/to/deskkit`. Each builds and tears down
its own `mktemp` scratch desk with a hermetic `XDG_DATA_HOME`, exactly as `librarian/verify.sh`
does. No probe touches a real desk or the operator's real store. The binary used was built from
this worktree (`go build -o deskkit ./cmd/deskkit`, go 1.25.8).

## Results roll-up

Full detail + dispositions in [deficiency-report.md](deficiency-report.md). Headline:

**v1 half (shipped model):**

- **2 DEFICIENCIES** (gate-bearing), both **filed as issues**:
  D1 (#184) — the finding-disposition surface has no id-bearing read path; D2 (#185) —
  `update_item` skips the `type` vocabulary check that `create_item` enforces, and an untyped item
  bypasses document gates.
- **1 FRICTION** (feeds the harvest loop, not the gate): F1 — cascade-initiated block/unblock
  audit rows are attributed to the triggering human actor.
- **4 accuracy observations**, three of which are **stale claims in
  `../2026-07-design-session/data-model.md`** that the live behavior contradicts (recorded with
  evidence; correcting that dossier is an out-of-scope follow-up — see the report).

**v2 half (draft `docs/element-model-v2-draft.md`, tabletop):**

- **4 DEFICIENCIES**, each dispositioned: V2-D1 (#197, filed) — the software spec phase-machine is
  unreconciled with the shipped PM phase-machine and §6.4's verification gate names no gate rule;
  V2-D2 — §11 overclaims the ADR 0012 type-validation inheritance (rides #185); V2-D3 — the net-new
  v2 doc types have no directory/patrol classification; V2-D4 — the `claim` status read surface
  risks inheriting D1's missing-id gap (rides #184).
- **3 accepted-risk / by-design** (V2-A1 research plane is paper by ADR 0009 two-track ordering;
  V2-A2 output triggers deferred to #127; V2-A3 the element model is a content reframe, not a
  redefinition of the librarian/PM infrastructure it consumes).
- **#125 → #126 blocked-by relationship recorded on the board and discharged** (AC5).

## v2 half (unblocked by #125, now walked)

The v2 element-model half of #126 was **blocked by `element-model-revision` (#125)**: running
scenarios against the pre-revision draft would have surfaced gaps the revision already intended to
fix. #125 has since merged the reviewed model to `docs/element-model-v2-draft.md` (folding ADR 0018
Q1–Q4 + both adversarial reviews' 18 gaps), discharging that blocked-by relationship (recorded on
the board and in the deficiency report, AC5). The v2 pass is now complete:

- **Method: tabletop.** The v2 model is a DRAFT with no storage or code (ADR 0009 "truth regime is
  STAGED"; draft §14 forbids v2 storage ahead of finalization), so every v2 step is a paper
  coherence trace, not a scripted run — exactly the recommended default (deliverable B) for a model
  with nothing to script against.
- **Scenario set.** The four v1 scenarios re-read against the v2 model (each doc's `v2 delta` column
  + a "v2 assessment" section), **plus** the model's own two native walkthroughs — Scenario 5 (v2
  software) and Scenario 6 (v2 research), tracing draft §10.
- **Framing.** The v2 element model is a *content/knowledge* reframe; it does not redefine the
  librarian operational collections or the PM engine — those are infrastructure it consumes (§11).
  So the v1 scenarios' *operational* steps are unchanged by v2 (their `v2 delta` = "—"); what v2
  changes is the *content vocabulary* those engines carry. This is why the issue's stop-and-report
  trigger ("missing a v2-equivalent of a v1 collection") does not fire — see V2-A3 in the report.

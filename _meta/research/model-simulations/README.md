---
type: analysis
status: active
created: 2026-07-21
updated: 2026-07-21
tags: [model-simulations, design-session, schema-v2]
synopsis: Index for the #126 model-simulations work — the v1-half scenario set, method, coverage, and where the deficiency roll-up lives.
---

*Purpose: walk realistic project scenarios through the shipped v1 element model to surface
deficiencies in the data model or its components before the v2 model finalizes (#126). This is
the **v1 half**; the v2 half is blocked by `element-model-revision` (#125) and is staged for
later — see "v2 is deferred" below.*

Status: active (2026-07-21) — v1 half complete; v2 columns intentionally left open.

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

| # | Scenario | Walkthrough doc | Derived from |
|---|---|---|---|
| 1 | Librarian chain: sweep → patrol → propose_fix → apply_fix → restore + findings disposition | [scenario-1-librarian-chain.md](scenario-1-librarian-chain.md) | workflows.md §1.1–1.3, §1.5 |
| 2 | PM gated transition + cascade, with a gate refusal | [scenario-2-pm-gated-cascade.md](scenario-2-pm-gated-cascade.md) | workflows.md §2.2–2.3 |
| 3 | Greenfield desk setup (K23) | [scenario-3-greenfield.md](scenario-3-greenfield.md) | desk-setup SKILL "Greenfield runbook (K23)" |
| 4 | Brownfield adoption (9-phase) | [scenario-4-brownfield.md](scenario-4-brownfield.md) | brownfield-adoption SKILL "Phase runbook" |

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

- **2 DEFICIENCIES** (gate-bearing), both filed-as-issue (drafts in the report):
  D1 — the finding-disposition surface has no id-bearing read path; D2 — `update_item` skips the
  `type` vocabulary check that `create_item` enforces, and an untyped item bypasses document
  gates.
- **1 FRICTION** (feeds the harvest loop, not the gate): F1 — cascade-initiated block/unblock
  audit rows are attributed to the triggering human actor.
- **4 accuracy observations**, three of which are **stale claims in
  `../2026-07-design-session/data-model.md`** that the live behavior contradicts (recorded with
  evidence; correcting that dossier is an out-of-scope follow-up — see the report).

## v2 is deferred (blocked by #125)

The v2 element-model half of #126 is **blocked by `element-model-revision` (#125)**: running
scenarios against the current pre-revision draft would surface gaps the revision already intends
to fix. Every walkthrough table carries a **"v2 (deferred)"** column, left as `— (blocked by
#125)`, so the v2 pass can be added in place against the same scenario steps once #125 lands and
produces a testable model. The report's disposition table likewise reserves a v2 section. Do not
run v2 walkthroughs here until #125 exists.

---
type: analysis
status: active
created: 2026-07-22
updated: 2026-07-22
tags: [model-simulations, schema-v2, element-model, research]
synopsis: Scenario 6 walkthrough — the v2 element model's own research-project slice (docs/element-model-v2-draft.md §10, over §5/§8/§9) traced tabletop against the draft, every element/relation marked OK / FRICTION / DEFICIENCY. The v2-native research counterpart the issue's Deliverable A requires.
---

*Scenario 6 of the #126 model simulations (**v2 half**). Walks the research-project slice the v2
element model draws for itself (`docs/element-model-v2-draft.md` §10, realized by the evidence model
§5/§9 and the research loop §8) against that draft. **Method: tabletop** — same rationale as Scenario
5: the v2 model is a DRAFT with no storage/code (ADR 0009 STAGED truth; §14). The research plane is,
by the draft's own §13/§14, the most-deferred build; this walkthrough is therefore primarily a
coherence check that the model *names* an end-to-end research slice that holds together.*

Status: active (2026-07-22)

## Operator story (from the draft's own §10 research walkthrough)

A researcher opens a `project(type=research)`, gathers inputs (`source`s with subkinds, `dataset`s,
`literature-note`s, `note`s), runs the reviewed loop
(`research-brief ⇄ investigation-plan ⇄ gather/experiment ⇄ synthesize`), builds first-class evidence
(`claim`s `supported-by` `citation`s `from` `source`s; `experiment`s `supports` claims), integrates
into a `research-synthesis` (spine), and — on a trigger — surfaces a `deliverable(kind:
research-report)`. Findings raise new `open-question`s that re-enter the loop.

## Step-by-step trace

| # | Slice step (draft §10 / §) | Elements + relations exercised | v2 verdict |
|---|---|---|---|
| 1 | Open `project(type=research)` (§3) | `project` entity, `type: research`; optional `workstream` tag | **OK** |
| 2 | Gather `source`s (subkinds), `dataset`s, `literature-note`s, `note`s [input] (§5.1–5.2) | `source` (subkind + credibility); `dataset` split out (computed-over, versioned); `literature-note` (one per source); `note` | **OK (model-level)** — build deferred (§13/§14), V2-A1 |
| 3 | Reviewed loop: `research-brief ⇄ investigation-plan ⇄ gather/experiment ⇄ synthesize` (§8) | loop redrawn (R4 fix); "source-gathering" dropped as a discrete phase; `research-brief`/`investigation-plan` docs | **OK** — the waterfall→loop fix (Q3) is coherent |
| 4 | Build evidence: `claim —supported-by→ citation —from→ source` (§5.3, §9) | `claim`/`finding` with status axis `supported\|contradicted\|open`; `citation` (passage + locator); modeled on findings-disposition (ADR 0013, §11) | **DEFICIENCY (V2-D4)** — the `claim` **status read surface**, built on the findings-disposition pattern, risks inheriting v1-D1/#184's missing-id read gap |
| 5 | `experiment`/`investigation-run` `—supports→ claim`; reproducibility hook (§5.4) | `experiment` (config + as-run protocol + result; seed/env/`dataset` version/code-ref); `experiment —supports→ claim` | **OK (model-level)** — named; build deferred (§13) |
| 6 | `goal`/`milestone`/`task`s + `analysis` of options [activity/loop] (§6, §9) | activity-plane entities; `analysis` (options → recommendation) `→ decision` | **OK** — clean `analysis → decision` pair kept (R7) |
| 7 | `synthesize → research-synthesis` (spine) (§3, §8, §9) | `research-synthesis` promoted into the spine (R5 / shared-defect fix) — the research core knowledge artifact | **OK** — spine can now yield a research artifact (shared defect closed) |
| 8 | Surface `deliverable(kind: research-report)` [output, on trigger] (§7) | `research-report` is a `deliverable` **view** (`kind`), not a doctype (R7); `deliverable —assembles→ {synthesis, finding, source, …}` | **OK (relations)** — output un-exercisable, trigger deferred to #127 (V2-A2) |
| 9 | `finding`s raise new `open-question`s that re-enter the loop; `open-question —resolved-by→ {decision\|finding\|deliverable}` (§7, §8) | `open-question` as the loop re-entry point (Q3); registry→entity promotion; resolve edge specified | **OK** — the re-entry point + its exit edge are both modeled |

## Findings from this scenario

### DEFICIENCY V2-D4 — the `claim` status read surface risks inheriting the D1 missing-id gap (step 4)

§5.3 and §11 model the `claim` evidentiary-status axis (`supported/contradicted/open`) deliberately on
the shipped **findings-disposition** pattern (ADR 0013) — "adopted rather than reinvented," carrying
the same actor/reason/when provenance. That is a sound reuse. But the v1 half surfaced (v1-D1, filed
#184) that the findings-disposition *read* surface emits `{path, detail}` with **no record id**, so
the CLI disposition workflow — which resolves by id — cannot be completed from the CLI. A `claim`
status surface built on the same pattern will reproduce that gap unless the model states that the
evidence-triple's status read surface must expose the record id.

**Severity low** — forward-looking; the research plane is build-deferred, so nothing is broken today,
but the model is the right place to record the constraint so the D1 mistake is not repeated.
**Disposition: amendment-needed (note), depends on #184** — add a "the claim-status read surface must
expose the record id" constraint to §5.3/§11. Not separately filed: rides #184.

## By-design / accepted-risk observations (not deficiencies)

- **V2-A1 — the entire research plane is paper.** The evidence triple, `experiment`, `dataset`,
  `literature-note` collections are, by §13/§14 and ADR 0009's two-track ordering, deferred until after
  the model finalizes; the draft only *names* them. So Scenario 6 is a **coherence check, not a
  behavior test** — and it passes coherence: the slice names an end-to-end research flow
  (`source → literature-note → claim/citation → synthesis → research-report`, §9) with no missing link.
  Accepted-risk, by-design. (This is the correct state per the issue: "building the research plane …
  is downstream v2-track work.")
- **V2-A2 — output trigger deferred to #127** (same as Scenario 5): the `research-report` deliverable
  view is trigger-gated; the trigger cannot be walked here.
- **Evidence model completeness (positive result).** The research-review gaps R1–R7 that made the v1
  spine unable to run research (no evidence triple, flat `source`, no empirical element, waterfall
  loop, spine can't yield a research artifact, no literature-note, boundary redundancy) are each traced
  to a folded-in element/relation in this walkthrough — the §12 disposition table checks out against a
  live trace. No *new* research-model deficiency beyond V2-D4.
</content>

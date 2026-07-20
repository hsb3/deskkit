# 0018 · Element-model direction — goal shape, planes, research loop, triggered outputs

_Settles the platform stream's four open questions; steers the schema-v2 element track._

- **Status:** Accepted
- **Date:** 2026-07-20
- **Raised by:** `_meta/research/2026-07-design-session/platform/spec-element-model.md` §8 (questions imported by ADR 0009's reboot)

## Context

The migrated element model (three planes — input · activity · output — over a beefed spine)
left four owner-preference questions open, previously pending on the retired desk. Its two
adversarial reviews returned real gaps (no evidence triple, no code/PR or bug entity, the
spine deferring each project type's core artifact), so the model itself remains a draft the
v2 track must revise; these rulings steer that revision.

## Decision

Owner-ruled 2026-07-20:

- **Q1 — simple `goal`** (measurable target + deadline + measure), not OKR structure now;
  OKR-style objective/key-results may layer later as goal-to-goal relations if ever wanted.
- **Q2 — workstream stays as an optional tag** (product / engineering / pm) alongside the
  plane model; planes are the structure, workstream is a label, droppable later if unused.
- **Q3 — research is the reviewed loop**, not a waterfall: brief ⇄ plan ⇄ gather/experiment
  ⇄ synthesize → findings → new open questions → re-enter, with `open-question` as the
  re-entry point; the evidence triple (claim / citation / source) folds in when the v2 track
  builds the research plane.
- **Q4 — exec outputs: exec-summary + stakeholder-update first; ON-DEMAND until triggers
  are defined.** The owner's constraint, verbatim in spirit: outputs must be driven by
  triggers, not produced speculatively — "i don't want to produce documents that aren't
  needed." Candidate triggers recorded for the trigger-design work: **(a) a meeting,
  (b) reaching a milestone/marker in task completion.** Demo-prep waits for a real demo.

## Consequences

- The v2 element track (ADR 0009) revises the model under these rulings + its review fixes,
  then runs the owner-directed **simulations** (ADR 0009) before finalization.
- Trigger design becomes a named v2-track work item; until it lands, no automation produces
  exec outputs unprompted.
- Nothing in the shipped v1 schema changes on account of this ADR alone.

## Affects

The schema-v2 element track (`_meta/research/2026-07-design-session/platform/
spec-element-model.md` revision) · the output-plane / comms integration design · the
trigger-design work item · the Phase-4 build plan.

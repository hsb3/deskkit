> **Tracking:** #130, rollup over the schema-v2 element track (ADRs 0009 + 0018, 2026-07-20 design session). The element model revised, simulated, and versioned before anything v2 ships.

## Why

ADR 0009's two-track ruling sends the element model (three planes over a beefed spine, migrated
from the desk-platform stream) down its own track: it stays DRAFT until it is revised under the
ADR 0018 Q1-Q4 rulings plus its two adversarial reviews' gaps, and the owner directed that
**v1 + v2 model simulations** — realistic project scenarios walked through both models — run
before the v2 model is finalized. This epic tracks that arc plus its substrate (versioning the
shared `schema/` contract) and the two v2-track design items the session named (trigger design
for exec outputs, ADR 0018 Q4; centralized prompt tuning, ADR 0015).

This epic deliberately carries **no milestone**: it is the next arc, not part of the 1.0.0
promise. Nothing here changes the shipped v1 schema on its own (ADR 0018).

## Children

- [ ] #124 `schema-versioning` — ADR 0009: version the shared `schema/` contract; both lanes' loaders + drift guards
- [ ] #125 `element-model-revision` — ADR 0018 + 0009: revise the element model under Q1-Q4 + the review gaps
- [ ] #126 `model-simulations` — ADR 0009 owner directive: walk realistic scenarios through v1 AND draft v2; the deficiency report gates finalization
- [ ] #127 `trigger-design` — ADR 0018 Q4: triggers for exec outputs (candidates: meeting; milestone/marker); on-demand until it lands
- [ ] #128 `prompt-tuning-centralized` — ADR 0015 owner requirement: one canonical prompt set tuned in one place

## Close when

- The revised element model is FINALIZED: every Q1-Q4 ruling reflected, every adversarial-review
  gap addressed or explicitly deferred with reason, and the model-simulations deficiency report
  exists with each deficiency either fixed in the model or recorded as accepted.
- `schema/` carries a version both lanes' loaders read and assert.
- The trigger design and the centralized-tuning design are recorded (ADR or reviewed design
  doc), each naming its build slices.
- No v2 storage/migration work has shipped ahead of the finalized model (the track's own
  ordering rule).

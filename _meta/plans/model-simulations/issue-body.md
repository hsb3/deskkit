> **Tracking:** #TBD, ADR 0009 / ADR 0018 (2026-07-20 design session). Simulate realistic
> project scenarios against both the shipped v1 schema and the draft v2 element model, to
> surface deficiencies before the v2 model finalizes.

## Problem

ADR 0009's decision (3) sends the element model down its own schema-v2 track, and attaches a
named owner directive to it (`docs/decisions/0009-platform-frame.md`, "Owner directive attached
to (3)"): run **simulations against both the v1 and v2 data models** - walk realistic project
scenarios through the model and its components to surface deficiencies - before the v2 model is
finalized. The owner's own signoff note is more specific than the ADR prose
(`_meta/signoff/2026-07-20-design-session-rulings/answers.json`, item `D0c`): "this makes
sense. more design work needs to be done to ensure the new model holds together. for both v1
and v2, i propose that we run some simulations to identify any deficiencies in the data model
or related components."

No such simulation work exists today. The v2 element model is still a draft with named
completeness gaps from its own adversarial reviews (`_meta/research/2026-07-design-session/
platform/spec-element-model.md`, "R3 review findings" section - e.g. no evidence triple, no
code/PR or bug entity, the spine deferring each project type's core artifact), and the sibling
epic already conditions the v2 track's own "FINALIZED" close-when bullet on this deficiency
report existing (`_meta/plans/epic-schema-v2-track/issue-body.md`, "Close when" - "the
model-simulations deficiency report exists with each deficiency either fixed in the model or
recorded as accepted"). This issue is that child, made ownable on its own.

**The v1 model** is the shipped, running schema - not paper. It has two authoritative
inventories to walk scenarios against:
- The persisted collections: `_meta/research/2026-07-design-session/data-model.md` section 1
  "Collections - librarian lane" (`files`, `patrol_findings`, `patrol_log`, `revisions`,
  `adoption_log`, `agent_runs`, `messages`, `tasks`, `prompts`, `feedback`) and section 2
  "Collections - PM lane" (`items`, `dependencies`, `transitions`, `notes`, `desk_config`).
- The workflows those collections serve: `_meta/research/2026-07-design-session/workflows.md`
  section 1 "Librarian lane" (sweep -> patrol -> propose_fix -> apply_fix -> restore, sections
  1.1-1.3; the findings disposition lifecycle, section 1.5) and section 2 "PM lane" (the phase
  state machine 2.1, gated transitions 2.2, the block/unblock cascade 2.3, claim/release 2.4).

**The v2 model** is the revised element model (three planes - input / activity / output - over
a beefed spine): `_meta/research/2026-07-design-session/platform/spec-element-model.md`. This
issue is **blocked by `element-model-revision` (#TBD)** - the v2 side of the simulation needs a
model that has already absorbed the ADR 0018 Q1-Q4 rulings and both adversarial reviews' fixes;
running scenarios against the current, pre-revision draft would surface gaps the revision is
already going to fix, wasting the exercise. The v1 side has no such dependency and can start
immediately.

## Deliverables

- **A - The scenario set.** Recommended default: derive scenarios from
  `_meta/research/2026-07-design-session/workflows.md`'s real flows plus
  `_meta/research/2026-07-design-session/platform/spec-element-model.md` section 6
  "Walkthroughs", so each scenario exercises collections/entities AND at least one PM engine
  phase/gate AND the findings lifecycle where relevant. At minimum:
  - One scenario drawn from the shipped sweep -> patrol -> propose_fix -> apply_fix -> restore
    chain (workflows.md sections 1.1-1.3), including a findings-disposition step (section 1.5).
  - One scenario drawn from the PM gated-transition + cascade cycle (workflows.md sections
    2.2-2.3): an item moves queue -> work -> review -> terminal, with at least one gate refusal
    and one cascaded block/unblock.
  - One **greenfield** desk scenario, following the shipped Greenfield runbook (K23)
    (`plugin/claude-plugin/skills/desk-setup/SKILL.md`, "Greenfield runbook (K23)" section) -
    walk a brand-new desk from scaffold through first sweep/patrol/PM-item creation.
  - One **brownfield** desk scenario, following the shipped brownfield-adoption skill's 9-phase
    runbook (`plugin/claude-plugin/skills/brownfield-adoption/SKILL.md`, "Phase runbook"
    section) - an existing, messy desk through inventory -> disposition -> the librarian
    baseline gate (its phase 8).
  - Software AND research project walkthroughs on the v2 side, per
    `spec-element-model.md` section 6 (its own software/research walkthrough pair) once the
    revision lands.
  The exact scenario list is an **open item** for the assignee to finalize - the above is a
  recommended default, not a closed list.
- **B - The simulation method.** Recommended default: **tabletop (paper) walkthrough first**,
  for both models, since the v2 model has no storage or code to script against yet
  (`docs/decisions/0009-platform-frame.md`, "Truth regime is STAGED" - the element model stays
  draft). Where the v1 side can run for real, script it against a real scratch desk via a
  `make verify`-style harness: `librarian/verify.sh` already builds and migrates the binary,
  seeds a **throwaway** scratch desk under `mktemp`, and drives the tool chain end to end
  (`librarian/verify.sh` header comment, lines 2-6) - the same harness shape (never a real desk)
  is the recommended substrate for any scripted v1 scenario, not a new mechanism. Exact
  method/tooling per scenario is an **open item** - the above is the recommended default.
- **C - The deficiency report.** The output artifact. Recommended default location:
  `_meta/research/model-simulations/deficiency-report.md` (new folder, sibling to but distinct
  from the dated `_meta/research/2026-07-design-session/` dossier, since this work spans past
  that one session). Alternate: graduate to `docs/` if the report proves durable reference
  material rather than working notes. Exact target is an **open item**; the default above should
  be used absent a stronger reason to diverge.
- **D - The pass/fail bar gating v2 finalization**, mirroring the schema-v2 epic's own
  close-when bullet (`_meta/plans/epic-schema-v2-track/issue-body.md`): the report exists; every
  deficiency it lists is either filed as its own issue or explicitly recorded as
  accepted-risk with rationale; the v2 model (`element-model-revision`, #TBD) is amended to
  close each deficiency, or the deficiency is recorded accepted-risk, BEFORE that issue is
  marked finalized.

## Acceptance criteria

- [ ] A scenario set is recorded (in this deliverable's own working doc or directly in the
      deficiency report) covering at minimum: one flow from the shipped sweep/patrol/
      propose_fix/apply_fix/restore chain, one flow from the PM gated-transition + cascade
      cycle, one greenfield desk walkthrough (per the desk-setup skill's Greenfield runbook),
      and one brownfield desk walkthrough (per the brownfield-adoption skill's phase runbook).
- [ ] Both the v1 (shipped) model and the revised v2 (element-model) draft are walked through
      the same scenario set, with results recorded per scenario per model.
- [ ] A deficiency report artifact exists at the agreed location, listing every deficiency
      surfaced, each with a disposition: filed as its own issue (linked), or explicitly
      recorded as accepted-risk with a stated rationale.
- [ ] Every deficiency against the v2 model is either closed by an amendment to
      `element-model-revision` (#TBD) or recorded accepted-risk, before that issue is marked
      finalized - matching `_meta/plans/epic-schema-v2-track/issue-body.md`'s own "Close when"
      bar.
- [ ] `element-model-revision` (#TBD) carries a recorded blocked-by/depends-on relationship to
      this issue (or its inverse) so the ordering is enforced on the board, not just in prose.

## Dependencies & gates

- Blocked by `element-model-revision` (#TBD): the v2 side of the simulation needs a model that
  has already absorbed the ADR 0018 Q1-Q4 rulings and the two adversarial reviews' fixes
  (`_meta/research/2026-07-design-session/platform/spec-element-model.md`, "R3 review
  findings"). The v1 side is unblocked and can start immediately.
- Parent: `epic-schema-v2-track` (#TBD), `_meta/plans/epic-schema-v2-track/issue-body.md`.
- Gate menu (`_meta/plans/_config.md`): this is docs/research work by default - `make check`,
  `make test`, and `make verify` do NOT fire unless deliverable B's scripted v1 harness adds a
  new script under `librarian/` or `scripts/`, in which case: any new script under `librarian/`
  is subject to identity-neutrality (`node scripts/check-neutrality.mjs`) and must only ever
  touch a throwaway scratch desk, never a real one, per `librarian/verify.sh`'s own convention.
  No PocketBase migration and no shipped-tree behavior change ships from this issue alone.

## Out of scope

- Finalizing the v2 model itself - that is `element-model-revision` (#TBD), a separate act
  gated BY this issue's deficiency report, not performed here.
- Building v2 storage or migrations - explicitly out of order per the schema-v2 epic's own
  ordering rule (`_meta/plans/epic-schema-v2-track/issue-body.md`, "no v2 storage/migration
  work has shipped ahead of the finalized model").

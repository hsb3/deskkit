> **Tracking:** #TBD, ADR 0009 / ADR 0018 (2026-07-20 design session). Design what a trigger is
> and how it fires exec outputs, so the on-demand default can graduate without producing
> documents nobody asked for.

## Problem

ADR 0018 Q4 ruled the exec-output element types (`exec-summary` + `stakeholder-update` first,
per `_meta/research/2026-07-design-session/platform/spec-element-model.md` section 4 "Output
plane - communication packages") stay **ON-DEMAND until triggers are defined**
(`docs/decisions/0018-element-model-direction.md`, "Q4"). The owner's constraint is explicit and
is worth quoting verbatim from the raw signoff record
(`_meta/signoff/2026-07-20-design-session-rulings/answers.json`, item `Q4`): "these need to be
driven by triggers. we have to define those triggers. for now, they're just on demand. i'm open
to defining triggers but don't want to produce documents that aren't needed. i'll say that
candidate triggers are a) a meeting b) reaching some milestone/marker in task completion." ADR
0018's own Consequences section names this as a standing work item: "Trigger design becomes a
named v2-track work item; until it lands, no automation produces exec outputs unprompted."

No trigger design exists today. This issue IS that design item - not an implementation, a
recorded design (ADR or design doc) answering four questions, each grounded in what the shipped
system does or does not already have:

1. **What a trigger IS in the model** - an event on which plane? Both candidates are
   activity-plane events (a meeting is an interaction/spine entity; "reaching a
   milestone/marker in task completion" is squarely the PM engine's phase-transition
   machinery) - the design should state this plainly rather than leave "trigger" undefined as a
   floating concept.
2. **How the two candidates map onto shipped/planned mechanisms** - verified, not assumed:
   - **Milestone/marker**: the PM engine knows **nothing** about milestones today. The shipped
     PM-lane collections are exactly `items, dependencies, transitions, notes, desk_config`
     (`_meta/research/2026-07-design-session/data-model.md` section 2, "Collections - PM lane")
     - there is no `milestone` collection or field. `milestone` is a **v2-track NET-NEW/partial**
     spine entity in the draft element model
     (`_meta/research/2026-07-design-session/platform/spec-element-model.md` section 1, row
     `milestone` - "partial (roadmap phases, `target_ship_date`)"). Today, `items.type` is
     free-text and **unvalidated** - `CreateItem` sets it directly from caller input with zero
     vocabulary check (`data-model.md` section 4, `items.type` row) - so a caller could already
     label an item's type `"milestone"` with **zero engine awareness** of what that means. The
     closest existing v1 mechanism to "reaching a marker" is the cascade engine's `auto`/
     `auto-reopen` unblock-at-phase-rank comparison
     (`_meta/research/2026-07-design-session/workflows.md` section 2.3, `tryAutoUnblock`) - a
     plausible substrate for "an item crossing a phase threshold fires something" once
     `milestone` is a real, validated element, but nothing today interprets phase-crossing as a
     milestone event.
   - **Meeting**: no `meeting` collection exists in either lane's shipped schema
     (`data-model.md` sections 1-2). `meeting-notes` is a shipped **document doctype** -
     frontmatter-validated, required `meeting_date`/`attendees` (`schema/doctypes.yaml:81`) - but
     that is a file convention, not a queryable entity or a scheduling surface. There is
     **no calendar/scheduling integration** anywhere in the profile contract today - a grep for
     "calendar", "meeting", and "trigger" against `schema/profile.schema.yaml` returns zero
     matches. Honestly: a meeting trigger has nothing to hook into today beyond a human manually
     authoring a `meeting-notes` file: the design must state this gap rather than imply an
     integration exists.
3. **What firing produces** - which output, delivered how. Per ADR 0018 Q4 and
   `spec-element-model.md` section 4, the candidates are `exec-summary` and
   `stakeholder-update`; the shipped/planned assembly path is a `deliverable` entity assembling
   input+activity artifacts, rendered via the exec-desk **comms** skill
   (`spec-element-model.md` section 4, "Workflow" - "a `deliverable` assembles content... ->
   renders via the exec-desk comms skill (deck-builder / pptx-themes / audio) -> delivered").
   None of that comms-skill wiring exists in this repo today (the comms skill and its
   deck/audio pipeline are the desk-platform stream's tooling, not desk-standard's) - the design
   should name this as a dependency, not assume it is already reachable.
4. **The manual-override path** - on-demand generation stays available regardless of whether a
   trigger fires; ADR 0018 Q4 keeps this explicit, and the design must carry it forward as a
   hard constraint, not an implicit fallback.

## Deliverables

- **A - The design record itself.** Recommended default: draft it as a design doc first, not an
  ADR directly - `_meta/research/trigger-design/design.md` (new folder; sibling in spirit to
  `_meta/research/2026-07-design-session/platform/spec-element-model.md`, which this design
  feeds), then graduate the ratified shape into an ADR once reviewed, following this repo's
  existing ADR practice of settling direction only after review
  (`docs/decisions/0018-element-model-direction.md` is itself the graduated form of exactly such
  a research-stage document). Exact target/format is an **open item** - the recommendation above
  should be used absent a stronger reason to diverge.
- **B - The trigger definition** (design question 1 above): what a trigger is, on which plane,
  recorded in the design doc with the activity-plane framing stated and justified or revised.
- **C - The candidate-to-mechanism mapping** (design question 2 above): the milestone/marker
  mapping (citing the free-text `items.type` gap and the cascade `auto`/`auto-reopen`
  unblock-at-phase mechanism as the plausible substrate) and the meeting mapping (citing the
  `meeting-notes` doctype and the confirmed absence of any calendar/scheduling integration),
  both carried into the design record with their citations intact.
- **D - The firing-to-output mapping** (design question 3 above): which trigger produces which
  of `exec-summary`/`stakeholder-update`, assembled by which mechanism (the `deliverable` +
  comms-skill path named in `spec-element-model.md` section 4), delivered to where - and an
  explicit note if the comms-skill substrate does not yet exist in this repo.
- **E - The manual-override statement** (design question 4 above): on-demand generation stays
  available regardless of trigger state, recorded as a hard constraint in the design, not an
  afterthought.
- **F - Build slices**, named with rough scope or explicitly deferred with a stated reason - the
  design should not leave a silent gap between "designed" and "buildable."

## Acceptance criteria

- [ ] A design doc (or ADR) exists that records: what a trigger is and which plane it lives on;
      the mapping for both candidate triggers (milestone/marker, meeting) onto shipped or
      planned mechanisms, each claim cited to source; what firing produces and how it is
      delivered; and the manual-override path.
- [ ] The no-speculative-outputs rule is stated as a hard constraint in the design record,
      quoting or faithfully paraphrasing the owner's Q4 signoff note (`_meta/signoff/
      2026-07-20-design-session-rulings/answers.json`, item `Q4`).
- [ ] The record states plainly, with citation, that no calendar/scheduling integration exists
      today for a meeting trigger to hook into (`schema/profile.schema.yaml` has zero matches
      for "calendar"/"meeting"/"trigger"), and that `items.type` is free-text/unvalidated today
      so a "milestone" has no engine-level meaning yet (`_meta/research/2026-07-design-session/
      data-model.md` section 4) - so neither candidate is claimed to already work.
- [ ] The design is reviewed (adversarial pass or owner signoff) before any build slice is
      treated as ready to pick up.
- [ ] Build slices are either named (with rough scope) or explicitly deferred with a stated
      reason - no silent gap between "designed" and "buildable."

## Dependencies & gates

- Docs-only until build: no code, schema, or migration change ships from this issue alone.
  `make check`, `make test`, and `make verify` do NOT fire.
- Relation to `element-model-revision` (#TBD): a prose note, not a hard edge. This design steers
  the eventual shape of the output plane / `deliverable` entity
  (`_meta/research/2026-07-design-session/platform/spec-element-model.md` section 4), so the two
  should stay mutually aware, but no dependency in either direction was found that requires one
  to complete before the other - flag this again if the element-model revision changes the
  output-plane shape enough to invalidate part C/D above.
- Parent: `epic-schema-v2-track` (#TBD), `_meta/plans/epic-schema-v2-track/issue-body.md`.
- Not blocked by anything. Blocks any future "implement triggers" build issue, to be filed once
  this design is reviewed.

## Out of scope

- Implementing triggers (the automation itself) - a separate, later build slice per deliverable
  F above.
- Demo-prep - ADR 0018 explicitly defers it: "Demo-prep waits for a real demo."
  (`docs/decisions/0018-element-model-direction.md`, "Q4").

---
title: "Trigger design for exec outputs -- build plan (ADR 0018 Q4)"
type: spec
status: planned
created: 2026-07-23
purpose: "Plan the trigger-design work item (#127): what a trigger is, how the two owner-named candidate triggers map onto shipped/planned mechanisms, what firing produces, and the manual-override constraint the design record ADR 0018 Q4 requires before exec-output generation can graduate off on-demand-only."
notes: "Issue #127 (open, unscheduled backlog, label enhancement, no milestone, no assignee). Companion issue body: issue-body.md in this folder (verified SAME as the live GitHub body, modulo one trailing newline). Docs-only design item; no code/schema/migration ships from this plan alone."
---

# Trigger design for exec outputs (#127)

_Plans the design-record work ADR 0018 Q4 named as a standing v2-track item: what a trigger is
and which plane it lives on, how the two owner-named candidates (a meeting; a milestone/marker in
task completion) map onto shipped or planned mechanisms, what firing produces and how it is
delivered, and the manual-override constraint that must survive regardless of trigger state.
RESIDUAL: none -- no trigger design exists anywhere in this repo today (grep-confirmed below);
this plan is entirely forward-looking, not a partial-work continuation._

Status: draft
Date: 2026-07-23

## Tracking

**#127** `trigger-design`, label `enhancement`, OPEN, no milestone, no assignee (unscheduled
backlog -- see "Scheduling status" below). Raised by ADR 0018 Q4
(`docs/decisions/0018-element-model-direction.md:29-33`), whose Consequences section names this
"a named v2-track work item; until it lands, no automation produces exec outputs unprompted"
(`0018-element-model-direction.md:39-40`). ADR 0009 (`docs/decisions/0009-platform-frame.md`)
sets the two-track schema split (mechanisms vs. the schema-v2 element model) this work lives
inside. Parent: `epic-schema-v2-track` **#130**
(`_meta/plans/epic-schema-v2-track/issue-body.md`), whose "Close when" section requires "the
trigger design ... recorded (ADR or reviewed design doc), each naming its build slices" as one of
its two remaining open bullets (the other is model finalization, gated on **#197** -- see
"Dependencies" below). Related, not blocking either direction: `element-model-revision` **#125**
(closed/merged -- produced the v2 draft this design steers) and `model-simulations` **#126**
(closed -- already logged the deferred trigger as an accepted, by-design gap, cited below).
Companion staged body: `issue-body.md` in this folder -- verified against the live GitHub body
(`gh issue view 127 --json body -q .body`); the two are **SAME**, differing only by one trailing
newline the live fetch adds (`diff` output: `132a133 > ` -- a blank line, not a content change).

## The problem, grounded in source

**No trigger design exists anywhere in this repo today.** `rg -i "trigger" schema/` and a search
of `plugin/`, `librarian/`, `docs/` for a trigger *definition* (as opposed to the unrelated
librarian wake-layer package, see below) return nothing that answers "what is a trigger" --
confirmed by `docs/development/specs/element-model-v2-draft.md` itself disclaiming the question three separate
times: section 7 ("trigger design is deferred to the separate `trigger-design` epic-child
(#127)... This document states only the Q4 constraint; it does not design the triggers",
`element-model-v2-draft.md:75-77`), section 13 ("all trigger design -- deferred: ... trigger
design is the separate `trigger-design` epic-child (#127)", `:452-454`), and section 14, Out of
scope ("Designing the exec-output triggers -- `trigger-design` (#127)", `:462`).

**What IS already ruled (the constraint the design must carry forward, not re-derive):**

- ADR 0018 Q4: exec outputs ship `exec-summary` + `stakeholder-update` first, **on-demand until
  triggers are defined** (`0018-element-model-direction.md:29-33`). Verbatim owner note from the
  raw signoff record (`_meta/signoff/2026-07-20-design-session-rulings/answers.json:60-63`,
  confirmed by direct read): "these need to be driven by triggers. we have to define those
  triggers. for now, they're just on demand. i'm open to defining triggers but don't want to
  produce documents that aren't needed. i'll say that candidate triggers are a) a meeting b)
  reaching some milestone/marker in task completion."
- `docs/development/specs/element-model-v2-draft.md` section 7 names the same two candidates and states the
  constraint as "Deliverables are produced on a trigger, never speculatively" (`:273-274`), and
  section 7's Rendering paragraph: "A `deliverable` renders via the exec-desk **comms** skill
  (deck-builder / pptx-themes / audio) -> delivered. The comms skill is the output-plane engine"
  (`:280-281`).
- `model-simulations` (#126) already priced in the deferred-trigger gap as **accepted, by-design**
  risk, not a v2-model defect: "V2-A2 | Output-plane deliverables are trigger-gated (Q4) but the
  trigger design is deferred to `trigger-design` (#127) ... **By-design.** ... Accepted-risk --
  tracked as a separate issue, not a v2-model gap"
  (`_meta/research/model-simulations/deficiency-report.md:246`). This means the design work does
  not need to re-justify the deferral -- it is already ratified; the design's job is to actually
  answer the four questions.

**Mechanism facts, verified against current source (not the issue-body's point-in-time claims --
see the correction below):**

- The PM-lane collections are exactly `items, dependencies, transitions, notes, desk_config`
  (`_meta/research/2026-07-design-session/data-model.md:202,211,238,253,270,284` -- section
  headers verified by direct read); there is no `milestone` collection or field anywhere in either
  lane's shipped schema. `milestone` is a v2-track spine entity, not yet built
  (`docs/development/specs/element-model-v2-draft.md:92`, row `milestone`, "partial (roadmap phases,
  `target_ship_date`)"). `schema/doctypes.yaml` has zero `milestone` entries (grep-confirmed,
  exit 1) -- there is no `milestone` doctype string in the vocabulary today.
- **Correction to the issue-body's claim.** The staged `issue-body.md` (and the live #127 body,
  identical) states "`items.type` is free-text and unvalidated -- `CreateItem` sets it directly
  from caller input with zero vocabulary check" (`issue-body.md:36-38`). **This is now false,
  verified against current source, not just the corrected dossier.** `data-model.md` itself
  carries a dated correction: "the #126 v1 model simulations empirically falsified three claims
  below ... `items.type` IS validated at create (`create_item` refuses an unknown type); the
  asymmetric `update_item` gap this section originally described was closed by issue #185
  (`update_item` now applies the identical vocabulary check...)" (`data-model.md:16-22`, dated
  2026-07-21 -- one day after the #127 issue body was staged). Confirmed directly in the engine
  source: `CreateItem` calls `vocab.KnownType(in.Type)`
  (`librarian/internal/modules/pm/engine/engine.go:142`) and `UpdateItem` applies the identical
  check (`librarian/internal/modules/pm/engine/queries.go:538`). **The corrected, still-true
  conclusion the design record should carry instead:** because `milestone` is not in the
  validated vocabulary, `CreateItem` would **refuse** an attempt to set `items.type = "milestone"`
  today (not silently accept it as the issue-body's "zero engine awareness" framing implies) --
  so a milestone item cannot even be created under v1 today, let alone one the engine interprets.
  The practical upshot is the same (milestone has zero engine-level meaning today) but the
  *mechanism* is refusal-by-missing-vocabulary-entry, not free-text passthrough -- the design
  record should state the corrected mechanism, not repeat the now-superseded claim.
- The closest existing v1 mechanism to "crossing a marker" is the cascade engine's `auto`/
  `auto-reopen` unblock-at-phase-rank comparison, `tryAutoUnblock`
  (`_meta/research/2026-07-design-session/workflows.md:362-385`,
  `librarian/internal/modules/pm/engine/engine.go:747-786`) -- a plausible substrate for "an item
  crossing a phase threshold fires something" once `milestone` is a real, validated element, but
  nothing today interprets phase-crossing as a milestone event, and this mechanism is entirely
  separate from any firing/dispatch layer (it only unblocks a dependent item's `blocked` state;
  it enqueues nothing).
- **New finding beyond the issue body: a real "wake layer" already exists, but is not wired to
  the PM module.** `librarian/internal/modules/librarian/trigger/` (package `trigger`) implements
  exactly the spec's "how PocketBase hooks wake the agent" section
  (`docs/development/specs/pocket-librarian-v1-spec.md:158`, the `tasks` queue spec'd at `:590`): a record hook
  (`RegisterHooks`, bound today only to the `files` collection's create event) and an hourly cron
  (`RegisterCron`) both `enqueue` a row into the `tasks` collection; a single claimer goroutine
  (`StartClaimer`/`ClaimOnce`) polls and dispatches by `tasks.kind`
  (`librarian/internal/modules/librarian/trigger/trigger.go:35-96,104-227`). `tasks.kind` is a
  fixed `SelectField` enum -- `sweep, patrol, propose_fix, apply_fix, restore, query, custom`
  (`librarian/internal/modules/librarian/collections/0008_tasks.go:17-18`) -- and the `query`/
  `custom` cases already dispatch to an injected `AgentAction` (an LLM call) rather than a fixed
  tool function (`trigger.go:219-223`), so a first build slice could plausibly reuse the existing
  `custom` kind (no enum migration) to carry an exec-output generation request as payload.
  **But this wake layer is entirely inside the `librarian` module and has zero PM-module
  awareness today**: the PM module's own hook registration (`RegisterHooks`,
  `librarian/internal/modules/pm/module.go:98-118`) binds only `desk_config` write validation and
  `transitions` append-only hardening -- it never calls the trigger package's `enqueue`, and the
  trigger package never listens on `items` or `transitions`. Wiring a milestone-crossing PM event
  into this wake layer is net-new cross-module wiring, not something already connected.
- No `meeting` collection exists in either lane's shipped schema (`data-model.md` sections 1-2,
  headers at lines 31 and 202 -- neither lists `meeting`). `meeting-notes` is a shipped document
  doctype, frontmatter-required `meeting_date`/`attendees`
  (`schema/doctypes.yaml:97` -- **note:** the issue-body cites `:81`; the doctypes file has
  shifted since the body was staged, a citation-drift, not a substantive error -- the doctype
  itself is unchanged). This is a file convention, not a queryable entity or scheduling surface.
  `rg -i "calendar|meeting|trigger" schema/profile.schema.yaml` returns zero matches
  (exit 1, confirmed directly) -- no calendar/scheduling integration exists anywhere in the
  profile contract today.
- No comms-skill / deck-builder / pptx-themes implementation exists under `plugin/`, `librarian/`,
  or `kits/` -- a repo-wide grep for `deck-builder`/`pptx-themes`/`comms.skill` returns hits only
  in prose docs (`docs/development/specs/element-model-v2-draft.md`, `_meta/research/...`,
  `.claude/memory/decision-briefings-outside-terminal.md`), never a shipped implementation. The
  `deliverable -> comms skill -> delivered` rendering path named in
  `docs/development/specs/element-model-v2-draft.md:280-281` is unbuilt in this repo (per that same document, the
  comms skill is "the desk-platform stream's tooling, not desk-standard's" per the issue body's
  own framing, `issue-body.md:60-62` -- unchanged by the correction above, independently
  confirmed by the grep).

## Design questions the deliverable must answer

Numbered here (not reusing ADR 0018's own Q1-Q4 labels, to avoid confusion) with a recommended
default where one can be grounded. These map 1:1 onto the issue body's four questions but restate
them against the verified/corrected facts above rather than the issue body's point-in-time
claims.

1. **What IS a trigger in the model, and which plane?** Both candidates are activity-plane
   events -- a meeting is an interaction/spine entity (`docs/development/specs/element-model-v2-draft.md:94`, spine
   row `meeting`); "reaching a milestone/marker" is squarely the PM engine's phase-transition
   machinery (the cascade `tryAutoUnblock` mechanism above). **Recommended default: state this
   plainly as the answer** -- a trigger is an activity-plane event, not a floating concept or a
   fourth plane. High confidence; the source material does not support any other reading.
2. **How do the two candidates map onto shipped/planned mechanisms?** Not a single
   yes/no call -- the design record must carry both mappings with citations, corrected per
   above:
   - **Milestone/marker:** `milestone` is a named-but-unbuilt v2 spine entity
     (`element-model-v2-draft.md:92`); `schema/doctypes.yaml` has no `milestone` entry, so
     `CreateItem`'s vocabulary check would refuse the type today (corrected mechanism, not
     "free-text/unvalidated" -- see above); the cascade `auto`/`auto-reopen` unblock-at-phase-rank
     comparison (`tryAutoUnblock`) is the plausible substrate for interpreting a phase-crossing as
     an event, but nothing today wires it to any firing/dispatch layer. **Recommended default:**
     the design record should name `tryAutoUnblock`'s phase-rank comparison as the substrate
     *concept*, but state explicitly that realizing it requires (a) `milestone` entering the
     validated vocabulary (blocked on model finalization, see Dependencies) and (b) new wiring
     from a phase-crossing to a firing event -- neither exists today.
   - **Meeting:** `meeting-notes` is a shipped document doctype (a file convention), not a
     queryable entity or scheduling surface; zero calendar/scheduling integration exists in the
     profile contract (`schema/profile.schema.yaml` grep, confirmed). **Recommended default:** the
     design record should state this gap plainly rather than imply an integration exists -- a
     meeting trigger has nothing to hook into today beyond a human manually authoring a
     `meeting-notes` file (which could itself be a legitimate v1 firing mechanism: a hook on
     `meeting-notes` file creation, mirroring the existing `files`-collection hook pattern in the
     wake layer -- worth naming as the more buildable near-term path, since it needs no new
     validated vocabulary entry, unlike the milestone path).
3. **What does firing produce, and how is it delivered?** Per ADR 0018 Q4 and
   `element-model-v2-draft.md` section 7, the candidates are `exec-summary` and
   `stakeholder-update`, assembled by a `deliverable` entity and rendered via the exec-desk comms
   skill. **Recommended default:** name this as the shipped/planned assembly path, and state
   explicitly (as the issue body already does, now independently confirmed by grep) that the
   comms-skill substrate does not exist in this repo -- it is an external dependency, not
   something this design or its build slices can close alone. **New grounding to add beyond the
   issue body:** the *firing mechanism itself* (event/cron -> enqueue -> claimer dispatch) already
   exists in-repo as the librarian wake layer (`librarian/internal/modules/librarian/trigger/`),
   and its `custom` task kind could plausibly carry a first exec-output request without a schema
   migration -- but it has zero PM-module wiring today (see above), so this is a *candidate build
   substrate to name*, not a claim that firing is already possible.
4. **The manual-override path.** ADR 0018 Q4 keeps on-demand generation available regardless of
   trigger state, explicit and non-negotiable. **Recommended default:** carry this forward
   verbatim as a hard constraint in the design record -- no judgment call needed; this is a direct
   ADR carry-forward, not new design work.

## Deliverable shape

**Recommendation: a design doc first, not an ADR directly** -- `_meta/research/trigger-design/design.md`
(new folder; confirmed empty today, `command ls _meta/research/` lists no `trigger-design` entry),
then graduate the ratified shape into a new ADR (next available slot: `0022`, advisory only --
confirm the true next-free number at landing time, `docs/decisions/` currently runs 0001-0021)
once reviewed. This matches the issue body's own recommendation (`issue-body.md:69-76`), and is
independently grounded in two repo precedents, verified directly rather than taken on the issue
body's word:

- `docs/decisions/0018-element-model-direction.md` is itself "Raised by
  `_meta/research/2026-07-design-session/platform/spec-element-model.md` §8" (`0018:7`) -- i.e.
  ADR 0018 is the graduated form of exactly this research-doc-first pattern.
- `docs/development/specs/element-model-v2-draft.md`'s own Provenance section: "Graduated 2026-07-22 from the R3
  proposal ... (3 read-only scouts ... two opus adversarial reviews)" (`element-model-v2-draft.md:469-473`)
  -- a second, more recent instance of the same research-doc -> reviewed `docs/`-landed artifact
  pattern, this time landing as a `docs/` spec rather than an ADR (showing the pattern is not
  ADR-only; the graduated *shape* depends on what the content is).

Given trigger-design produces a **ruling** (what triggers are, hard constraints) more than a
**spec** (element inventories), an ADR is the better graduated shape than a `docs/` spec doc --
matching ADR 0018's own precedent for exactly this kind of owner-constraint-carrying content,
not `docs/development/specs/element-model-v2-draft.md`'s precedent (a spec-shaped inventory document). Exact
filename/number is confirmed only at landing time (see Open questions).

**Required out-of-scope follow-up (not owned by this plan folder):** once the design ratifies,
`docs/development/specs/element-model-v2-draft.md` sections 7 (`:276-278`), 13 (`:452-454`), and 14 (`:462`) each
currently say the trigger question is deferred to "the separate `trigger-design` epic-child
(#127)" -- those three spots need a small follow-up edit pointing at the landed ADR/design doc
instead of "deferred." That file is owned by whoever holds `element-model-revision`/model-track
docs, not this plan folder; name it as a build-slice dependency, do not edit it here.

## Deliverables

- **A -- The design record itself.** `_meta/research/trigger-design/design.md` (new), reviewed,
  then graduated to a new ADR (advisory `docs/decisions/0022-trigger-design.md`, confirm number
  at landing). Acceptance: file exists at each stage; the ADR is Accepted (not Proposed) before
  any build slice is treated as ready.
- **B -- The trigger definition** (design question 1). Acceptance: the design record states,
  with citation, that a trigger is an activity-plane event, both candidates included.
- **C -- The candidate-to-mechanism mapping** (design question 2). Acceptance: both mappings
  present with citations, including the corrected `items.type`-validation mechanism (not the
  issue-body's superseded "free-text/unvalidated" framing) and the wake-layer/PM-module wiring
  gap named above.
- **D -- The firing-to-output mapping** (design question 3). Acceptance: names which trigger
  produces which of `exec-summary`/`stakeholder-update`, the `deliverable` + comms-skill assembly
  path, and states plainly that the comms-skill substrate is an external, unbuilt dependency; also
  names the in-repo wake-layer as a candidate firing substrate (new finding, not required by the
  issue body but strengthens deliverable F's feasibility).
- **E -- The manual-override statement** (design question 4). Acceptance: recorded as a hard
  constraint, ADR 0018 Q4 carried forward verbatim or faithfully paraphrased.
- **F -- Build slices**, named with rough scope or explicitly deferred with a stated reason.
  Acceptance: no silent gap between "designed" and "buildable." Recommended candidate slices to
  name (not commit to building here): (i) a `meeting-notes`-file-creation hook mirroring the
  wake layer's existing `files`-hook pattern, reusing the `custom` task kind -- lower-cost, no
  vocabulary change needed; (ii) `milestone` entering the validated vocabulary + a
  phase-crossing-to-firing wiring on `tryAutoUnblock` -- explicitly **blocked on v2 model
  finalization** per ADR 0009's ordering rule ("Implementing any v2 collection, migration, or
  store change" is forbidden "ahead of the finalized model",
  `docs/development/specs/element-model-v2-draft.md:458`), so this slice cannot start until the model (currently
  gated on #197, see Dependencies) finalizes; (iii) the comms-skill rendering path itself --
  explicitly out of this repo's scope, a cross-repo/cross-stream dependency, name and defer, do
  not attempt.

## Acceptance criteria

Restated from the issue body's own criteria, corrected where the source has moved since staging:

- [ ] A design doc (then ADR) exists recording: what a trigger is and which plane (design
      question 1); both candidate mappings with citations (design question 2); what firing
      produces and how it's delivered (design question 3); the manual-override path (design
      question 4).
- [ ] The no-speculative-outputs rule is stated as a hard constraint, quoting or faithfully
      paraphrasing the owner's Q4 signoff note (`_meta/signoff/2026-07-20-design-session-rulings/answers.json:60-63`,
      verified verbatim above).
- [ ] The record states, with citation, that no calendar/scheduling integration exists today for
      a meeting trigger (`schema/profile.schema.yaml` zero matches, confirmed) -- **and** the
      corrected `items.type` mechanism: type validation exists (`CreateItem`/`UpdateItem` both
      check `vocab.KnownType`), but `milestone` is not in the vocabulary, so a milestone item
      cannot be created today, not "free-text/unvalidated." (This corrects, not repeats, the
      issue body's acceptance-criteria wording at `issue-body.md:103-107` -- see the Problem
      section's Correction above.)
- [ ] The design is reviewed (adversarial pass or owner signoff) before any build slice is
      treated as ready to pick up.
- [ ] Build slices are named (with rough scope) or explicitly deferred with a stated reason --
      verified against the F deliverable above: at minimum the meeting-hook slice, the
      milestone-vocabulary slice (with its finalization-gate dependency stated), and the
      comms-skill slice (named and deferred, not attempted) are each addressed.
- [ ] (New, not in the issue body) The three `docs/development/specs/element-model-v2-draft.md` cross-reference
      spots (`:276-278`, `:452-454`, `:462`) are named as a required follow-up in the design
      record or its landing PR description, so the graduated design does not leave the v2 draft
      pointing at a still-open issue.

## Dependencies

**Independent of `#197`, not behind it.** Verified directly, not inferred from the issue body
(which predates #197 and does not mention it): `#197` ("reconcile the software spec phase-machine
with the PM item phase-machine and name the building->shipped gate rule") is scoped to
`docs/development/specs/element-model-v2-draft.md` sections **6.1/6.4** -- the **activity-plane** spec-document
lifecycle (`draft -> in-review -> approved -> building -> shipped`) versus PM `items.phase`, and
the `test-run -> verified-by ->` gate rule. `#127` is scoped to section **7**, the **output
plane**. Different plane, different mechanism family (spec-document status axis vs. exec-output
firing), confirmed by reading `#197`'s full body directly (`gh issue view 197 --json body`) --
it never references triggers, `deliverable`, or exec outputs. **Decisive evidence from the
repo's own gate-label taxonomy:** `#197` carries label `gate:v2-final` ("blocks schema-v2
element-model finalization"); `#127` carries only `enhancement` -- no `gate:v2-final` label
(`gh issue view <n> --json labels`, confirmed for both). The repo's own labeling convention
already encodes that trigger-design is not one of the items blocking the v2 element model's
finalization gate.

Both are, however, sibling gates on epic `#130`'s "Close when" section, which lists model
finalization (behind #197) and trigger-design (#127) as two **separate** bullets -- neither
blocks the other's own completion, but both must land before the epic itself can close. This
plan can proceed in parallel with #197's resolution.

**One real coupling exists, already flagged by the issue body and confirmed above (not a
blocking dependency, a steering one):** the milestone-vocabulary build slice (F-ii above) cannot
ship ahead of v2 model finalization per ADR 0009's ordering rule -- and model finalization is
itself gated on #197. So while the *design record* (deliverables A-E) has no dependency on #197,
one of its *named build slices* (F-ii) transitively does, once #197 resolves and the model
finalizes. State this in the design record so a future builder does not attempt F-ii early.

## Gate & contract hygiene

| Gate | Fires? | Note |
| ---- | ------ | ---- |
| Repo checks (`make check`) | NO | no change under `plugin/` or `librarian/`; the neutrality scanner's scope is `plugin/` + `librarian/` only, `docs/` and `_meta/` are exempt |
| Unit tests (`make test`) | NO | no code change |
| Librarian integration (`make verify`) | NO | no `librarian/` change |
| Bundle drift guard (`make package`) | NO | no `plugin/core`, `plugin/mcp`, or `schema/` change |
| Identity neutrality (`check-neutrality.mjs`) | NO | design doc lands in `_meta/research/` (exempt) and, once graduated, `docs/decisions/` (exempt) -- bare `#127` references are fine in both locations, unlike in `plugin/`/`librarian/` |
| Version sync / kits drift | NO | no `VERSION` or `kits/` change |
| CHANGELOG | CONDITIONAL | per `CHANGELOG.md:23-24` precedent, the v2 element model draft's `docs/` graduation DID get an entry; recommended default: add an `[Unreleased]` entry when the design graduates to `docs/decisions/00NN-trigger-design.md` (the ADR), not for interim `_meta/research/trigger-design/design.md` work-in-progress -- matching how other research-stage docs in `_meta/research/` were not individually logged, only their `docs/`-landed graduations were |
| DB migration discipline | NO | no PocketBase collection touched by this design work itself (F-ii's future migration is out of scope here, and explicitly blocked on model finalization) |
| Regression-test bar | NO | not a behavior change |
| Adversarial review | YES (owner-decided, not a CI gate) | the design's own acceptance criteria requires "reviewed (adversarial pass or owner signoff)" before build slices are ready -- this is a process gate stated in the design record, not an automated CI check |
| Pre-commit (lefthook) | YES, always | mirrors CI lanes; a docs-only change still runs it, trivially passes (no code touched) |

Explicitly: **no code, schema, or migration gate fires from this plan alone** -- matches the
issue body's own Dependencies & gates section (`issue-body.md:115-116`), independently confirmed
against the actual gate scopes above rather than taken on the issue body's word.

## Parallelism + landing order

Trivial for this plan: a single design-doc deliverable (A), with B-E as its required content and
F as its closing section, has no meaningful parallel-file-scope split -- one author drafts the
whole design record, since B/C/D/E are sections of the same document, not independently
implementable code units. No timelines; the shape is:

1. Draft `_meta/research/trigger-design/design.md` answering all four design questions (B-E),
   naming build slices (F).
2. Adversarial review or owner signoff (the design's own acceptance criteria requires this before
   any build slice is "ready to pick up").
3. Graduate the reviewed shape into a new ADR (advisory `0022`, confirm true next-free number at
   landing).
4. File the `docs/development/specs/element-model-v2-draft.md` cross-reference follow-up (out-of-scope for this
   plan's own files -- a required handoff to whoever owns that doc) and the epic `#130`
   Close-when checkbox update.
5. Only after (3): pick up any of the named build slices (F), each gated individually by its own
   stated dependency (F-i is buildable now; F-ii is blocked on model finalization/#197; F-iii is
   out of this repo's scope).

## Scheduling status

**Unscheduled backlog.** `#127` carries no milestone and no assignee (`gh issue view 127`,
confirmed). Epic `#130` itself "deliberately carries no milestone: it is the next arc, not part
of the 1.0.0 promise" (`_meta/plans/epic-schema-v2-track/issue-body.md`, "Why" section). The
repo's own standing notes confirm the schema-v2 arc is between cycles: "schema-v2 arc closed for
this cycle" and "The owner decision queue is EMPTY" (`_meta/HANDOFF.md:33-39`), with `#197` named
as the schema-v2 track's next-in-consequence open item ahead of `#127`
(`_meta/HANDOFF.md:53-56`) -- not because `#127` is blocked by it (see Dependencies above), but
because the HANDOFF's own ordering reflects unscheduled-backlog priority, not a hard gate.

No timeline is stated or implied by this plan, per project convention. **What would make this
schedulable:** (1) an owner or crew picks it up as the next schema-v2-track item -- nothing
technical blocks starting immediately, since deliverables A-E have zero dependency on #197 or any
other open item; (2) alternatively, if the owner chooses to sequence deliberately, landing #197
first would let build-slice F-ii's dependency note (above) be written against a finalized model
rather than a projected one, a minor drafting convenience, not a technical blocker. Either
sequencing is valid; this plan does not recommend one over the other.

## Open questions / owner decisions

1. **Design-doc-first vs. ADR-direct.** Recommended default: design doc first
   (`_meta/research/trigger-design/design.md`), graduate to ADR -- see "Deliverable shape" above,
   grounded in two repo precedents (0018's own provenance; the v2 draft's provenance). Real
   decision the owner or drafting session could override if a shorter path is preferred, since the
   content here (four largely-settled questions with grounded defaults) is less exploratory than
   the full element-model revision was.
2. **ADR number.** Advisory `0022` (next free slot as of 2026-07-23, `docs/decisions/` runs
   0001-0021). **Confirm the true next-free number at landing time** -- a parallel ADR-producing
   effort landing first would consume it first.
3. **Whether build slice F-i (the `meeting-notes`-hook) is worth naming as "near-term buildable"
   in the design record, given it still depends on the comms-skill rendering path (F-iii, out of
   repo scope) to actually produce a delivered output.** Recommended default: name it anyway, but
   scope it explicitly as "wires the firing signal only; rendering/delivery stays blocked on the
   comms-skill dependency" -- so the design record does not imply an end-to-end capability that
   is not actually deliverable without the external comms-skill work.
4. **Whether to reuse the wake layer's existing `custom` task kind for a first firing
   implementation, versus adding a new dedicated enum value to `tasks.kind` later.** Not a call
   for the design record itself -- flagged here as a build-slice-level default a future builder
   can start from: **reuse `custom` for the first slice** (no `0008_tasks.go` migration needed),
   revisit a dedicated kind only if `custom`'s free-form payload proves insufficient once real
   usage exists.

## Unverified / flagged for the build/review session

- **Whether any off-repo executive-desk decision record (per this repo's paired-desk convention,
  `CLAUDE.md`'s "No in-repo dogfooding" section) already has draft trigger-design content this
  plan should have found.** Not checked -- this plan is scoped to the desk-standard repo's own
  tracked sources; if a paired desk holds relevant prior art, the design-drafting session should
  check it before starting from zero.
- **Whether `#197`'s eventual resolution changes anything about the activity-plane framing this
  plan's design question 1 rests on** (both candidates classified as activity-plane events). The
  two issues are independent as verified above, but #197's phase-machine reconciliation could in
  principle touch PM `items.phase` semantics in a way that affects how "reaching a milestone/marker"
  is described mechanically -- worth a quick re-check when #197 lands, not a blocking concern now.


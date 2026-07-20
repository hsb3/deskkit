> **Tracking:** #TBD, ADR 0018 + 0009 (2026-07-20 design session). Revise the draft element
> model under the Q1-Q4 owner rulings and its two adversarial reviews' gaps, so it stops being a
> proposal with open questions and becomes the schema-v2 track's reviewed draft.

## Problem

`_meta/research/2026-07-design-session/platform/spec-element-model.md` (the R3 element-model
proposal: three planes — input / activity / output — over a beefed spine, `:16-24`) is the
model ADR 0009 sends down the schema-v2 track (`docs/decisions/0009-platform-frame.md:36-38`).
ADR 0009's Consequences are explicit: "The element model stays draft until its review fixes land
and the simulations pass" (`:49`). ADR 0018 settled the four owner-preference questions the doc
itself raised (its own §8, `spec-element-model.md:138-144`) and restates the gap: "Its two
adversarial reviews returned real gaps ... so the model itself remains a draft the v2 track must
revise" (`docs/decisions/0018-element-model-direction.md:14-17`). Neither the rulings nor the
review fixes have been folded into the document yet — this issue is that fold-in.

**ADR 0018's four rulings**, each answering an open question left in the source doc:

- **Q1** (`spec-element-model.md:140`, "Is `goal` the right unit, or ... OKR-style
  `objective + key-results`?"): ruled **simple `goal`** — measurable target + deadline +
  measure, not OKR structure now; OKR may layer later as goal-to-goal relations
  (`docs/decisions/0018-element-model-direction.md:21-22`). The doc's current `goal` row
  (`spec-element-model.md:38`, "measurable target + deadline") does not yet name "measure" or
  the future OKR escape hatch.
- **Q2** (`:141`, "Keep `_headcase`'s workstream tag ... or drop it?"): ruled **keep it as an
  optional tag** (product/engineering/pm) alongside the plane model — planes are the structure,
  workstream a droppable label (`0018:23-24`). Not yet stated anywhere in the current doc.
- **Q3** (`:142-143`, "is `brief -> investigation-plan -> synthesis -> report` the right
  spine ...?"): ruled **research is the reviewed loop, not a waterfall** — brief ⇄ plan ⇄
  gather/experiment ⇄ synthesize → findings → new open-questions → re-enter, with
  `open-question` as the re-entry point; **the evidence triple (claim / citation / source)
  folds in** when the v2 track builds the research plane (`0018:25-28`). This directly answers
  the research review's gap #4 (loop, not waterfall) and gap #1 (evidence triple) below.
- **Q4** (`:144`, "Which of the deferred exec outputs ... matter first?"): ruled
  **exec-summary + stakeholder-update first, ON-DEMAND until triggers are defined** — outputs
  must be trigger-driven, never speculative; candidate triggers (a meeting, a milestone/marker)
  are recorded for the separate `trigger-design` work item, and demo-prep waits for a real demo
  (`0018:29-33`). Trigger design itself is a named v2-track work item, tracked as its own
  epic-child issue (`trigger-design`) — NOT this issue's scope.

**The two adversarial reviews' gaps** (`spec-element-model.md:146-201`), confirmed accurate
against source in the reviews' own provenance statements:

**Research review** (verdict: "runs research *intake* but not research end-to-end",
`:151-170`):
1. `:153-155` — **no evidence triple**: add `finding`/`claim` (status
   supported\|contradicted\|open) + `citation`/`excerpt`, relations
   `claim -supported-by-> citation -from-> source`. (Q3-mandated — must fold in.)
2. `:156-157` — **`source` too flat**: add subkinds (paper/url/vendor-doc/interview/internal) +
   excerpt/access_date/version/credibility; split `dataset` out of `source`.
3. `:158-160` — **no empirical element**: add `experiment`/`investigation-run` (config + as-run
   protocol + result; reproducibility hook), linked to the claims it supports.
4. `:161-163` — **research is a loop, not a waterfall**: redraw
   `brief <-> plan <-> gather/experiment <-> synthesize -> findings -> new open-questions ->
   back to brief`; `open-question` is the re-entry point; drop "source-gathering" as a discrete
   phase. (Q3-mandated — must fold in.)
5. `:164-165` — **v1 spine can't run research**: un-defer `research-synthesis` (the one FREE
   research doc); add finding+citation, else a research slice yields sources+notes+decision but
   no research artifact. (Also the shared defect below — must fold in.)
6. `:166-167` — **no literature-note**: the per-source reading artifact, chain
   `source -> literature-note -> claim/finding -> synthesis`.
7. `:168-169` — **boundary redundancy**: `research-report` should be a `deliverable` VIEW
   (output plane), not a doc type; keep the clean `analysis -> decision` pair; add a
   "when to use which" table.
8. `:170` — **minor**: `postmortem` SOP omitted from the §5 free list; schema is **32** types,
   not 31.

**Software review** (verdict: "runs the planning + comms arc, NOT the build-and-ship core",
`:172-195`):
1. `:174-176` — **no code/PR element**: build phase is invisible; add a `change`/`pull-request`
   element (or an explicit code-boundary statement) + an `implemented_by` relation from
   `engineering-spec`/`task` to the code.
2. `:177-178` — **no bug/issue entity**: only undifferentiated `task` + raid's issues leg; add
   `bug`/`issue` related to the spec/release it affects. Flagged in the review as "the clearest
   registry->entity win the doc missed."
3. `:179-181` — **`goal` floats**: wired only downward (`goal->milestone->task`); add
   `goal <-> one-pager` (measured-by) + `goal -> {product-spec\|deliverable}`; state which is
   canonical.
4. `:182-183` — **v1 spine omits every spec type**: a software slice can't state what it builds;
   promote `engineering-spec` into v1 spine (carries DoD + test-plan link). (Also the shared
   defect below — must fold in.)
5. `:184-186` — **§3 decomposition chain is WRONG**: `feature-spec`/`ux-spec` are SIBLINGS of
   `engineering-spec` (children of `product-spec`), not parents; §3 must be fixed to match §6
   (already correct there).
6. `:187-188` — **output-plane relations undefined**: `deliverable` assembles nothing (no
   relation); `release-notes` covers nothing; `source` has no cites edge; `open-question` has no
   resolved-by. Specify all four.
7. `:189-190` — **dropped types + postmortem**: `user-story`/`mockup` in no plane; `postmortem`
   SOP (incident review -> retro `variant: incident`) unmentioned.
8. `:191-192` — **no verification/QA-result element**: `test-plan` is only the plan; nothing
   records execution/sign-off gating `approved->building->shipped`; add `test-run`/
   `verification`.
9. `:193-194` — **redundancy**: product/feature/ux-spec undifferentiated (add
   when-to-use-which); output plane sprawls ~10 forms (collapse to `deliverable` container + a
   `kind` enum); `one-pager` double-placed.
10. `:195` — **minor**: 32 types not 31; `milestone` in §1 spine but not §7; `daily-note` is an
    SOP/template, not a schema type.

**Shared defect** (`:197-201`), flagged by BOTH reviews: the v1 spine defers the core artifact
of each project type (`engineering-spec` for software, `research-synthesis` for research), so
neither type can be demonstrated on the spine. The source doc's own stated fix: "promote one
build/knowledge artifact per type into the spine."

## Deliverables

- **A revised element-model document** folding in all four ADR 0018 rulings (Q1-Q4 above) and
  addressing every numbered review gap (research 1-8, software 1-10) — either folded in (with
  the model change named) or explicitly deferred with a stated one-line reason. No gap silently
  dropped.
- **Location — open question, recommended default stated here, confirm at build time.** Revise
  in place at `_meta/research/2026-07-design-session/platform/spec-element-model.md` (this
  repo's doc-revision discipline: "revise in place, git is the version history — never `vN`
  filenames"), OR graduate it to a new `docs/` location now that it has moved from "research
  proposal" to "the schema-v2 track's reviewed draft" (the `_meta/README.md` taxonomy: "graduate
  findings into `docs/` or issues"). **Recommended default: graduate**, to
  `docs/element-model-v2-draft.md` — following the existing `<product>-v1-spec.md` naming
  precedent (`docs/pocket-librarian-v1-spec.md`, `docs/pm-system-v1-spec.md`) while keeping
  `-draft` in the name so it isn't mistaken for a finalized `-spec.md`. Its own status header
  stays `status: draft` either way — graduating to `docs/` does NOT finalize the model;
  finalization is gated on `model-simulations` (ADR 0009). Note this "v2" is the schema-track
  label (ADR 0009), not a document-revision version counter, so graduating to a `v2`-named
  `docs/` file does not conflict with the never-`vN`-filenames rule (that rule targets revising
  the SAME document under a new suffix, not a research-to-docs graduation of a schema-versioned
  artifact).
- **A "consumed v1 mechanisms" subsection** naming which shipped v1 mechanism each cross-cutting
  concept in the revision borrows:
  - **ADR 0011** (typed cross-reference contract: kind + target + optional desk-relative
    qualifier) — the substrate for every "typed relation" the frame already claims
    (`spec-element-model.md:18`, "connected by typed relations"); state explicitly that the
    plane model's relations (e.g. `implemented_by`, `measured-by`, `cites`, `resolved-by` named
    above) are instances of this one contract, not ad hoc per-relation designs.
  - **ADR 0012** (`items.type` validated at creation against the shared vocabulary, hard reject
    unknown) — state that when the v2 element list becomes live PM `items.type` values, new
    element types inherit the identical `CreateItem`-time vocabulary check for free (per 0012's
    own text: "the check reads the vocabulary as it stands at any time — schema v1 doctypes now,
    the v2 element list when the two-track lands," `docs/decisions/0012-item-type-validation.md:21-22`).
  - **ADR 0013** (findings disposition: a human-judgment axis orthogonal to workflow state, with
    actor/reason/when provenance) — cite as design precedent for the research review's `claim`
    status axis (supported/contradicted/open, gap #1 above): a claim's evidentiary status is
    exactly the same shape as a finding's disposition — an orthogonal judgment axis, not a
    lifecycle `state`. State this explicitly rather than reinventing the shape.
  - **ADR 0017** (frontmatter `id` as the document-identity primitive, surviving rename +
    store-rebuild) — state that every spine entity-that-is-also-a-document (`person`, `source`,
    `decision`, `open-question`, `note`, etc.) hangs its cross-plane relation identity off this
    primitive, since files-are-truth (ADR 0009) means no store-side identity can be assumed to
    survive a rebuild.
- **The revision's own tracking line states it blocks `model-simulations (#TBD)`** — the
  ADR 0009 owner directive names simulations as the gate before the v2 model is finalized
  (`docs/decisions/0009-platform-frame.md:40-42`), and `epic-schema-v2-track`'s own child list
  already sequences `model-simulations` after `element-model-revision`
  (`_meta/plans/epic-schema-v2-track/issue-body.md:19-20`).

## Acceptance criteria

- [ ] The revision states `goal` as a measurable target + deadline + **measure**, explicitly NOT
      OKR structure, with a documented future escape hatch (OKR-style objective/key-results as
      goal-to-goal relations) — Q1.
- [ ] The revision states workstream (product/engineering/pm) stays an **optional tag**
      alongside the plane structure, explicitly droppable later — Q2.
- [ ] The revision redraws research as the loop `brief <-> plan <-> gather/experiment <->
      synthesize -> findings -> new open-questions -> re-enter`, names `open-question` as the
      re-entry point, and includes the evidence triple (`finding`/`claim` +
      `citation`/`excerpt` + `source`, with the `claim -supported-by-> citation -from->
      source` relation chain) — Q3 + research-review gaps #1/#4.
- [ ] The revision states output-plane deliverables are trigger-gated / on-demand, shipping only
      `exec-summary` + `stakeholder-update` now, with `demo-prep` and trigger design explicitly
      deferred to the separate `trigger-design` epic-child issue — Q4.
- [ ] Every one of the 8 numbered research-review gaps and 10 numbered software-review gaps
      (cited by line above) appears in the revision as either folded in (model change named) or
      explicitly deferred with a one-line reason — checkable via a side-by-side table in the
      revision's own provenance/changelog section.
- [ ] The shared defect is closed: the spine promotes one build/knowledge artifact per project
      type — `engineering-spec` for software, `research-synthesis` for research.
- [ ] The revision contains a "consumed v1 mechanisms" subsection naming ADR 0011 (typed
      relations), ADR 0012 (type-vocabulary validation for new v2 element types), ADR 0013 (the
      disposition/provenance pattern as precedent for the claim-status axis), and ADR 0017
      (frontmatter `id` as entity-identity substrate).
- [ ] The revision's tracking/provenance section states it blocks `model-simulations (#TBD)`.
- [ ] The revision is still marked `status: draft` wherever it lands (`docs/` or `_meta/research/`)
      — this issue does not finalize the model; finalization is gated on `model-simulations`.
- [ ] `node scripts/check-neutrality.mjs` passes bare (docs/ and `_meta/` are exempt from the
      scan by scope, but the check confirms nothing under `plugin/`/`librarian/` regressed
      incidentally).

## Dependencies & gates

- This is a docs/`_meta`-only change (a revised planning/spec document, no code). Per the
  `pointer-grammar-spec` sibling issue's precedent for docs-only scope statements, the following
  do **not** fire:
  - Librarian integration (`make verify`, `librarian/verify.sh`) — no `librarian/` code change.
  - Bundle drift guard (`make package` + `git diff --exit-code` on `plugin/claude-plugin/`) —
    no `plugin/core`, `plugin/mcp`, or `schema/` change.
  - Version sync (`check-version-sync.mjs`) and kits drift (`check-kits.mjs`) — `VERSION` and
    `kits/` untouched.
  - DB migration discipline — no PocketBase collection touched.
- Gates that DO fire because they always fire: `make check` (neutrality + self-test, kit drift,
  scaffold frontmatter, core purity, actionlint) and `make test` (plugin `bun test` + librarian
  `go test ./...`); both should be no-ops relative to `main` for a docs-only diff. Pre-commit
  (lefthook) mirrors the same and additionally runs markdown/YAML formatting checks that DO
  apply to a new/edited `.md` file.
- CHANGELOG: no shipped product behavior changes, so `check-changelog.mjs` (which hard-gates
  only at the release tag) is not blocked by this alone; still record the revision under
  `[Unreleased]` per this repo's provenance practice.
- **Blocks `model-simulations (#TBD)`** — ADR 0009's owner directive requires simulations
  against both v1 and v2 data models before the v2 model is finalized
  (`docs/decisions/0009-platform-frame.md:40-42`); `model-simulations` cannot walk a coherent v2
  model until this revision exists.
- Child of `epic-schema-v2-track` (`_meta/plans/epic-schema-v2-track/issue-body.md:19`).
- Depends on nothing new from `schema-versioning` (the versioning MECHANISM issue) — that issue
  ships the version marker; this issue produces v2-track CONTENT that will eventually be
  versioned by it, but the two are independently draftable/reviewable.

## Out of scope

- Implementing any v2 collection, migration, or store change — this issue is a document
  revision only; ADR 0009's own two-track ordering rule forbids v2 storage work landing ahead of
  the finalized model.
- **Finalizing** the model — ADR 0009: "the element model stays draft until its review fixes
  land and the simulations pass." This issue lands the review fixes and the Q1-Q4 rulings; it
  does not and cannot finalize, since `model-simulations` has not yet run.
- Designing the exec-output triggers (Q4's "which triggers, exactly") — tracked separately as
  the `trigger-design` epic-child issue; this issue states only the Q4 constraint
  (trigger-gated, on-demand, exec-summary + stakeholder-update first).
- Centralized prompt tuning (ADR 0015) — an unrelated epic-child issue
  (`prompt-tuning-centralized`), not touched here.
- Versioning the `schema/` contract itself — that is the sibling `schema-versioning` issue's
  scope; this issue produces content for the v2 track, it does not build the version marker.
- Running the model-simulations walkthroughs — a separate epic-child issue
  (`model-simulations`) that consumes this revision as an input; not performed here.

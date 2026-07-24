---
type: spec
status: draft
created: 2026-07-20
updated: 2026-07-22
tags: [dev-tooling, desk-platform, data-model, elements, planes, software, research, schema-v2]
synopsis: "Reviewed draft of the schema-v2 element model: every entity, document template, and workflow the platform should carry, organized into three planes (input / activity / output) over a beefed spine, for two project types (software + research). Graduated from the R3 research proposal and revised under ADR 0018's four owner rulings (Q1-Q4) and both adversarial reviews' 18 numbered gaps. Names which shipped v1 mechanisms each cross-cutting concept consumes (ADR 0011/0012/0013/0017). Still a DRAFT: finalization is gated on model-simulations (#126) per ADR 0009."
---

# Desk Platform — element model (schema-v2 draft)

_Status: **draft**. This is the schema-v2 track's reviewed element model — graduated from the R3
research proposal at
[`_meta/research/2026-07-design-session/platform/spec-element-model.md`](../../../_meta/research/2026-07-design-session/platform/spec-element-model.md)
(which is frozen as provenance) after folding in ADR 0018's Q1–Q4 owner rulings and both adversarial
reviews' 18 numbered gaps. It is **not finalized**: ADR 0009 holds the model draft "until its review
fixes land and the simulations pass" (`docs/decisions/0009-platform-frame.md:49`). This document
lands the review fixes and the rulings; finalization is gated on **`model-simulations` (#126)**._

> **Naming note.** "v2" here is the **schema-track label** (ADR 0009's two-track split), not a
> document-revision version counter. Graduating the R3 proposal to a `v2`-named `docs/` file does not
> conflict with the repo's never-`vN`-filenames rule — that rule targets revising the *same* document
> under a new suffix, not a research-to-`docs/` graduation of a schema-versioned artifact. The
> `-draft` suffix keeps it distinct from a finalized `-spec.md`.

## 0. Tracking & provenance

- **Resolves** `#125` (`element-model-revision`) — the fold-in of ADR 0018 + the two reviews.
- **Blocks `model-simulations` (#126)** — ADR 0009's owner directive requires simulations against
  both the v1 and v2 data models before the v2 model is finalized
  (`docs/decisions/0009-platform-frame.md:40-42`); `model-simulations` cannot walk a coherent v2
  model until this revision exists, and `epic-schema-v2-track`'s child list sequences
  `model-simulations` after `element-model-revision`
  (`_meta/plans/epic-schema-v2-track/issue-body.md:19-20`).
- **Child of** `epic-schema-v2-track` (`#130`).
- **Rulings folded:** ADR 0018 (Q1–Q4), ADR 0009 (two-track, staged truth, simulations gate).
- **Reviews folded:** the R3 research review (8 gaps) + software review (10 gaps) + the shared defect,
  as embedded in the frozen source doc's "R3 review findings" section.
- The gap-by-gap disposition table is in §12; the consumed-v1-mechanisms subsection is in §11.

## 1. The frame

Every element is a **typed entity** or a **document** hanging off one, connected by **typed
relations**. (Every relation named in this document — `implemented_by`, `measured-by`, `cites`,
`resolved-by`, `supported-by`, `from`, `assembles`, and the rest — is an instance of the one shipped
typed-reference contract, ADR 0011; see §11.) Elements sort into three **planes** that map 1:1 to the
exec desk's three jobs, over a shared **spine**:

- **Input plane** — refine information (sources → synthesized knowledge).
- **Activity plane** — set goals/targets/deadlines and do the work (the PM layer).
- **Output plane** — assemble communication packages (deliverables), **trigger-gated** (§7, Q4).
- **Spine** — the cross-cutting backbone every project uses, and the `document + revision` mirror
  substrate.

**Workstream is an optional tag, not the structure (Q2).** `_headcase` organized by *workstream*
(product / engineering / pm; executive deferred). The plane model is the reframe: **planes are the
structure; workstream stays as an optional label** (`product` / `engineering` / `pm`) that a desk may
carry on any element and may drop later if it goes unused (ADR 0018 Q2, `0018:23-24`). It is a
classification convenience, never a required axis — nothing in the model keys off it.

## 2. The four rulings, folded in (ADR 0018 Q1–Q4)

- **Q1 — `goal` is a simple measurable target, not OKR.** A `goal` is **measurable target +
  deadline + measure** (the metric the target is read against), explicitly **not** an OKR
  `objective + key-results` structure now. **Future escape hatch:** if OKRs are ever wanted, they
  layer on top as **goal-to-goal relations** (an objective goal with child key-result goals) rather
  than a new nested schema — no structural commitment is made today (`0018:21-22`). See §4 (spine
  `goal` row) and §6 (goal wiring).
- **Q2 — workstream is an optional tag** alongside the plane structure, droppable later (§1 above,
  `0018:23-24`).
- **Q3 — research is the reviewed loop, not a waterfall**, and the **evidence triple folds in** with
  the research plane (§8, §9; `0018:25-28`).
- **Q4 — output-plane deliverables are trigger-gated / on-demand.** `exec-summary` +
  `stakeholder-update` ship first; outputs are produced only on a trigger, never speculatively;
  `demo-prep` waits for a real demo; **trigger design is deferred to the separate `trigger-design`
  epic-child issue (#127)** (§7; `0018:29-33`). This document states only the Q4 *constraint*; it
  does not design the triggers.

## 3. Spine (cross-cutting backbone) — beefed, with the shared defect closed

The old v1 spine (`task / meeting / person / decision`) is **entirely in the activity plane** — it
cannot even demonstrate the three-plane model. The beefed spine touches all three planes and both PM
pillars, and **closes the shared defect** by promoting one **core build/knowledge artifact per
project type** into the spine so each type is demonstrable on it: `engineering-spec` (software) and
`research-synthesis` (research). See §12 "shared defect" row.

| Element | Kind | From _headcase? | Role |
| --- | --- | --- | --- |
| `project` | entity | yes (`project`) | container; `type: software \| research`; owner, status, repo/forge links, dates; optional `workstream` tag |
| `person` | entity | net-new (implied by author/owner/attendees) | stakeholder / collaborator / contact |
| `goal` | entity | **net-new** | **measurable target + deadline + measure** (Q1); the "set goals/targets/deadlines" pillar |
| `milestone` | entity | partial (roadmap phases, `target_ship_date`) | dated commitment under a goal |
| `task` | entity | yes (`task` lightweight) | action item incl. non-code ("meet with Tony") |
| `meeting` | entity + doc | yes (`meeting-notes`) | interaction + captured notes/decisions/actions |
| `decision` | entity + doc | yes (`decision`) | ADR-style record; promoted from the flat DECISIONS registry |
| `open-question` | entity | yes (OPEN_QUESTIONS registry) | promote registry → queryable entity; **the research loop's re-entry point** (§8) |
| `source` | entity | **net-new** | an input reference (§5); carries subkinds + credibility |
| `note` | entity | yes (`journal`/`idea`) | lightweight capture (`daily-note` is an SOP/template, not a schema type — §12 sw-#10) |
| `engineering-spec` | doc | yes | **software** core build artifact (carries DoD + `test-plan` link) — **promoted (shared defect)** |
| `research-synthesis` | doc | yes | **research** core knowledge artifact — **promoted (shared defect)** |
| `deliverable` | entity | **net-new** (formalizes "comm packet") | an output-plane container with a `kind` enum (§7) |

**Spine and the document mirror.** Every spine entity that is also a document (`person`, `source`,
`decision`, `open-question`, `note`, `meeting`, both promoted specs, …) hangs its cross-plane relation
identity off the frontmatter `id` primitive (ADR 0017), because files-are-truth (ADR 0009) means no
store-side row id can be assumed to survive a rebuild. See §11.

## 4. Input plane — refine information

| Element | Kind | From _headcase? | Software / Research |
| --- | --- | --- | --- |
| `source` | entity | net-new | both (research-heavy) — **subkinds + credibility (§5.1)** |
| `dataset` | entity | **net-new (split from `source`)** | both — computed over, not cited (§5.1) |
| `literature-note` | doc | **net-new** | research — one reading note bound to one `source` (§5.2) |
| `finding` / `claim` | entity | **net-new** | research — the evidence unit (§5.3, §9) |
| `citation` | entity | **net-new** | research — passage + locator into a `source` (§5.3, §9) |
| `experiment` / `investigation-run` | entity + doc | **net-new** | research — config + as-run protocol + result (§5.4, §9) |
| `note` / `idea` / `journal` | entity | yes | both |
| `research-synthesis` | doc | yes | **research** (now spine, §3) |
| `analysis` | doc | yes (options A/B/C → recommendation) | both |
| `project-brief` | doc | yes (north-star framing) | both |
| `user-journey` | doc | yes | product/research |
| `technical-design` | doc | yes | **software** (architecture reference) |
| `runbook` | doc | yes | **software** (operational knowledge) |

### 5.1 `source` gets subkinds + credibility; `dataset` splits out

`source` was too flat (one undifferentiated reference). It now carries:

- **subkind** enum: `paper` \| `url` \| `vendor-doc` \| `interview` \| `internal`;
- **fields:** `excerpt`, `access_date`, `version`, **credibility** (`primary` \| `secondary`).

**`dataset` is split out of `source`** as its own input element: a dataset is *computed over*, not
*cited* — it carries a version and is referenced by an `experiment`/`investigation-run`
(reproducibility hook, §5.4), not by a `citation`.

### 5.2 `literature-note` — the per-source reading artifact

One `literature-note` binds to exactly one `source` and records
argument / method / limitations / quotes. It is the bridge from raw source to evidence:
**`source → literature-note → claim/finding → research-synthesis`**.

### 5.3 The evidence triple (Q3-mandated)

Research needs first-class, queryable evidence — not "cited research" as un-queryable prose (which
would contradict the registry→entity thesis). The triple:

- **`finding` / `claim`** — an assertion carrying a **status axis**: `supported` \| `contradicted`
  \| `open`. This status is a **human-judgment axis orthogonal to any workflow `state`**, modelled on
  the shipped findings-disposition pattern (ADR 0013) — see §11, not reinvented here.
- **`citation`** — a `passage/excerpt` + `locator` drawn from a `source`.
- **`source`** — the input reference (§5.1).

**Relation chain:** `claim —supported-by→ citation —from→ source`. A `claim` may also be
`—contradicted-by→ citation` and rolls up into a `research-synthesis`.

### 5.4 `experiment` / `investigation-run` — the empirical element

An `experiment` (a.k.a. `investigation-run`) records **config + as-run protocol + result**, with a
**reproducibility hook** (seed / env / `dataset` version / code-ref). It is the realization of the
planned method: an `investigation-plan` (§8) is the *planned* method; the run is what actually
executed. It links to the claims it supports: `experiment —supports→ claim`.

> Scope note: this document **names** `experiment`/`dataset`/`citation`/`literature-note` as v2 model
> elements (folding the gaps in at the model level). *Building* the research plane — live collections,
> validation, tooling — is downstream v2-track work per ADR 0009's two-track ordering, and the
> evidence triple "folds in when the v2 track builds the research plane" (`0018:27-28`).

## 6. Activity plane — set goals/targets/deadlines + do the work

| Element | Kind | From _headcase? | Software / Research |
| --- | --- | --- | --- |
| `project`, `goal`, `milestone`, `task`, `meeting` | entity | mixed (goal/milestone net-new) | both |
| `raid` | doc/entity | yes (risks/assumptions/issues/deps) | both |
| `roadmap` | doc | yes (themes/phases) | both |
| `weekly-checkin`, `retro` | doc | yes (cadence) | both |
| `postmortem` → `retro (variant: incident)` | doc | yes (SOP, previously omitted — §12) | both (incident path) |
| `one-pager` | doc | yes (gate before >2 days) | both — **home plane = activity** (§7 clarifies the output *view*) |
| `product-spec` | doc | yes | **software/product** |
| `feature-spec`, `ux-spec` | doc | yes | **software/product** — **siblings** of `engineering-spec` (§6.1) |
| `engineering-spec` | doc | yes | **software** — now spine (§3); carries DoD + `test-plan` link |
| `test-plan` | doc | yes | **software** — the plan; execution is `test-run`/`verification` (§6.4) |
| `test-run` / `verification` | entity + doc | **net-new** | **software** — records execution/sign-off (§6.4) |
| `change` / `pull-request` | entity | **net-new** | **software** — the build artifact (§6.2) |
| `bug` / `issue` | entity | **net-new** | **software** — defect entity (§6.3) |
| `user-story`, `mockup` | doc | yes (previously in no plane — §12) | **software/product** — refinement artifacts under `product-spec`/`feature-spec` |
| `research-brief` / `investigation-plan` | doc | net-new | **research** (§8) |

### 6.1 Software decomposition — the §3-vs-§6 defect fixed

The R3 draft's §3 mis-drew the chain, making `feature-spec`/`ux-spec` *parents* of
`engineering-spec`. **Corrected:** they are **siblings**, all children of `product-spec`:

```
one-pager → product-spec → { feature-spec, ux-spec, engineering-spec } → test-plan → test-run
```

Phase machine (unchanged): `draft → in-review → approved → building → shipped`, with the PM
">2 weeks in-progress" hard-flag and the weekly-checkin / retro cadence.

**When to use which spec** (redundancy fix): `product-spec` = *what* and *why* (the product decision);
`feature-spec` = one feature's behavior/acceptance; `ux-spec` = interaction/visual design; `mockup`
and `user-story` are refinement inputs to those; `engineering-spec` = *how* it is built (DoD +
`test-plan` link). A slice needs a `product-spec`; the sibling specs are used as the work demands.

### 6.2 `change` / `pull-request` — the build phase, made visible

The build phase was invisible: `project` carried repo/forge links but nothing hung off them. Add a
`change` / `pull-request` element and an **`implemented_by`** relation:
`engineering-spec —implemented-by→ change` and `task —implemented-by→ change`.

**Code boundary (stated explicitly).** The *code itself* lives in git/the forge, referenced by
pointer (a typed reference, ADR 0011), **not** copied into the store. The `change` element is the
desk-side handle (title, status, forge URL, the specs/tasks it implements) — the model tracks the
*relation to* the code, files-are-truth for everything derived (ADR 0009).

### 6.3 `bug` / `issue` — the defect entity

`_headcase` had only undifferentiated `task` plus raid's issues *leg*. Add a first-class `bug` /
`issue` entity (priority enum already exists), related to what it affects:
`bug —affects→ { engineering-spec \| release-notes \| change }`. (Flagged in the software review as
"the clearest registry→entity win the doc missed.")

### 6.4 `test-run` / `verification` — execution and sign-off

`test-plan` is only the *plan*; nothing recorded execution/sign-off, yet the phase machine gates
`approved → building → shipped` on verification. Add a `test-run` / `verification` element:
`test-plan —verified-by→ test-run`, and `test-run` carries the pass/fail + sign-off that the
`building → shipped` transition reads.

### 6.5 `goal` wiring — no longer floating; canonical source named

The R3 `goal` was wired only downward (`goal → milestone → task`) and duplicated the one-pager's
Success Metrics. Fixed:

- `goal —measured-by→ one-pager` — the one-pager *frames/pitches* the goal;
- `goal → { product-spec \| deliverable }` — the goal points at what realizes it.

**Canonical statement of the measure is the `goal` entity itself** (Q1: target + deadline +
*measure*). The one-pager's "Success Metrics" section becomes a **view/reference onto the goal via
`measured-by`, not a second source of truth** — this resolves the duplication in favour of the
queryable entity, consistent with the registry→entity thesis (a floating metric buried in a doc is
exactly what §5-gap-5 promotes out). *(Judgment call recorded: the review said "state which is
canonical" without ruling it; the entity is canonical because the whole thesis is
prose-section → queryable-record, and Q1 puts the measure on the goal.)*

## 7. Output plane — communication packages (trigger-gated)

**Collapsed to one container + a `kind` enum** (redundancy fix — the R3 output plane sprawled ~10
undifferentiated forms). A **`deliverable`** is the single output-plane container; its `kind` enum
selects the form:

`exec-summary` \| `stakeholder-update` \| `release-notes` \| `sow` \| `research-report` \|
`briefing` / `status-deck` \| `demo-prep`.

- **`research-report` is a `deliverable` VIEW** (`kind: research-report`), **not** its own doc type
  (research review gap #7) — it composes a `research-synthesis` + its findings/claims for an
  audience.
- **`one-pager` is home in the activity plane** (§6); the output plane may *surface* it as a
  deliverable view, but its canonical home is activity — resolving the R3 double-placement.

**Output-plane relations, now specified** (software review gap #6):

- `deliverable —assembles→ { source, finding, analysis, research-synthesis, milestone, … }` — a
  deliverable *assembles* named input/activity artifacts (it no longer assembles "nothing").
- `release-notes —covers→ { change, milestone }` — release-notes *covers* the changes/milestones it
  reports.
- `source —cites→` / inverse `deliverable —cites→ source` — the citation edge into the evidence model
  (§5.3).
- `open-question —resolved-by→ { decision \| finding \| deliverable }` — the re-entry point's exit
  edge (also feeds the research loop, §8).

**Trigger gating (Q4).** Deliverables are **produced on a trigger, never speculatively**
("i don't want to produce documents that aren't needed", `0018:31`). Shipping first:
`exec-summary` + `stakeholder-update`. **Deferred:** `demo-prep` (waits for a real demo) and the
**trigger design itself** — which triggers, exactly (candidates recorded in ADR 0018: a meeting; a
milestone/marker in task completion) — is the separate **`trigger-design` epic-child (#127)**, not
this document's scope. This document states only the constraint.

**Rendering.** A `deliverable` renders via the exec-desk **comms** skill (deck-builder /
pptx-themes / audio) → delivered. The comms skill is the output-plane engine.

## 8. Research as a loop (Q3 + research-review gap #4)

The R3 draft drew research as a waterfall
(`research-brief → investigation-plan → source-gathering → research-synthesis → findings/decision`).
**Redrawn as the reviewed loop** — "source-gathering" is dropped as a discrete phase (gathering
happens inside the loop):

```
research-brief  ⇄  investigation-plan  ⇄  gather / experiment  ⇄  synthesize
      ↑                                                              │
      │                                                              ▼
   re-enter  ←──  new open-questions  ←──────────────────────  findings
```

- **`open-question` is the re-entry point** (Q3): a synthesis produces findings, which raise **new
  `open-question`s**, which re-enter the loop (back to brief/plan). `open-question —resolved-by→
  { decision \| finding }` closes the ones that resolve.
- **`gather / experiment`** covers both citation-gathering (§5.3) and empirical
  `experiment`/`investigation-run`s (§5.4) — the plan is the method, the run realizes it.
- **`synthesize`** produces a `research-synthesis` (spine, §3), which the evidence triple (§5.3) makes
  queryable rather than prose.

## 9. Evidence model — research end-to-end (research-review gaps #1/#2/#3/#5/#6)

Putting §5 together, a research slice now runs **end-to-end**, not just intake:

```
source (subkind, credibility)
   │  from
   ├── citation ──supported-by── claim/finding (status: supported|contradicted|open)
   │                                   │  supports
   ├── literature-note (one per source)│
   │                                   ▼
   └── dataset ──(reproducibility)── experiment/investigation-run ──supports── claim
                                                                       │
                                                              rolls up ▼
                                                          research-synthesis (spine)
                                                                       │  view
                                                                       ▼
                                                          deliverable (kind: research-report)
```

This closes research-review gaps #1 (evidence triple), #2 (`source` subkinds + `dataset` split),
#3 (empirical element), #5 (spine can now yield a research artifact — `research-synthesis` promoted +
finding/citation added), and #6 (`literature-note`).

**When to use which knowledge doc** (research-review gap #7):

| Artifact | Use when |
| --- | --- |
| `analysis` | weighing options A/B/C toward a recommendation → feeds a `decision` |
| `research-synthesis` | integrating findings/claims across sources into durable knowledge |
| `decision` | recording the chosen path (ADR-style), the terminal of `analysis` |

The clean `analysis → decision` pair is kept; `research-report` is the output *view* of a synthesis
(§7), not a fourth doc type.

## 10. Walkthroughs (does the model run a real project?)

**Software project.** `project(type=software)` → capture `source`/`note`, write `project-brief`
[input] → set `goal` (target + deadline + measure) + `milestone`s;
`one-pager → product-spec → { feature-spec, ux-spec, engineering-spec } → test-plan`, `task`s,
`meeting`s, `raid`, weekly-checkin/retro [activity]; build via `change` (`implemented_by`), defects as
`bug`, execution as `test-run` gating `building → shipped` → `release-notes` (`covers` the changes) +
a status `deliverable` deck [output, on trigger]. Decisions logged as `decision`; incidents via
`postmortem` → `retro(variant: incident)`; blockers via the >2-week flag.

**Research project.** `project(type=research)` → gather `source`s (subkinds), `dataset`s,
`literature-note`s, `note`s [input] → `research-brief` ⇄ `investigation-plan` ⇄ gather/`experiment` ⇄
`synthesize`, with `claim`s `supported-by` `citation`s `from` `source`s, `goal`/`milestone`/`task`s,
`analysis` of options [activity/loop] → `research-synthesis` (spine) → `deliverable`
(`kind: research-report`) [output, on trigger]. `finding`s raise new `open-question`s that re-enter
the loop; decisions + open-questions tracked as entities.

## 11. Consumed v1 mechanisms

Each cross-cutting concept in this revision **borrows a shipped v1 mechanism** rather than inventing a
new one. This keeps the v2 element model a *composition* of settled contracts, not a parallel design.

- **ADR 0011 — typed cross-reference contract** (kind + target + optional desk-relative qualifier).
  This is the substrate for every "typed relation" the frame claims. **Every relation named in this
  document — `implemented_by`, `measured-by`, `cites`/`covers`/`assembles`, `resolved-by`,
  `supported-by`/`from`, `affects`, `verified-by` — is an instance of this one contract, not an ad hoc
  per-relation design.** No default qualifier ships (identity-neutral); under files-are-truth (ADR
  0009) qualifiers resolve at read time from the profile.
- **ADR 0012 — `items.type` validated at creation against the shared vocabulary, hard reject
  unknown.** When the v2 element list becomes live PM `items.type` values, **every new element type
  inherits the identical `CreateItem`-time vocabulary check for free** — per 0012's own text, "the
  check reads the vocabulary as it stands at any time — schema v1 doctypes now, the v2 element list
  when the two-track lands" (`docs/decisions/0012-item-type-validation.md:21-22`). The count in §12
  (32 doctypes) is that vocabulary as it stands today; the v2 elements extend it and are validated the
  same way.
- **ADR 0013 — findings disposition** (a human-judgment axis orthogonal to workflow `state`, with
  actor / reason / when provenance). This is the **design precedent for the `claim` status axis**
  (`supported` / `contradicted` / `open`, §5.3): a claim's evidentiary status is exactly the same
  shape as a finding's disposition — an **orthogonal judgment axis, not a lifecycle `state`**, and it
  carries the same actor/reason/when provenance. Adopted deliberately rather than reinvented. (Under
  the standing files-are-truth regime, such judgment provenance is store-only supervisor state and
  re-opens only if the ADR 0009 gate flips — same caveat 0013 records for itself.)
- **ADR 0017 — frontmatter `id` as the document-identity primitive** (human-copyable, stable across
  rename, surviving store-rebuild-from-disk). **Every spine entity-that-is-also-a-document** —
  `person`, `source`, `decision`, `open-question`, `note`, `meeting`, `engineering-spec`,
  `research-synthesis`, and the rest — **hangs its cross-plane relation identity off this primitive**,
  because files-are-truth (ADR 0009) means no store-side id can be assumed to survive a rebuild. The
  typed relations (ADR 0011) therefore target frontmatter `id`s, not store row ids.

## 12. Gap-by-gap disposition (research 1–8, software 1–10, shared defect)

Every numbered review gap is either **folded in** (with the model change named) or **deferred** (with
a one-line reason). Nothing is silently dropped.

### Research review

| # | Gap | Disposition | Where |
| --- | --- | --- | --- |
| R1 | No evidence triple | **Folded in** — `finding`/`claim` (status axis) + `citation`/`excerpt` + `source`; chain `claim —supported-by→ citation —from→ source` (Q3-mandated) | §5.3, §9 |
| R2 | `source` too flat | **Folded in** — subkinds (paper/url/vendor-doc/interview/internal) + excerpt/access_date/version/credibility; `dataset` split out | §5.1 |
| R3 | No empirical element | **Folded in** — `experiment`/`investigation-run` (config + as-run protocol + result; reproducibility hook), `—supports→ claim` | §5.4, §9 |
| R4 | Research is a loop, not a waterfall | **Folded in** — redrawn loop; `open-question` is the re-entry point; "source-gathering" dropped as a discrete phase (Q3-mandated) | §8 |
| R5 | v1 spine can't run research | **Folded in** — `research-synthesis` un-deferred into the spine + finding/citation added (also the shared defect) | §3, §9 |
| R6 | No literature-note | **Folded in** — per-source reading artifact; chain `source → literature-note → claim/finding → synthesis` | §5.2, §9 |
| R7 | Boundary redundancy | **Folded in** — `research-report` becomes a `deliverable` view (`kind`), not a doc type; `analysis → decision` kept; when-to-use table added | §7, §9 |
| R8 | Minor: `postmortem` omitted; 32 types not 31 | **Folded in** — `postmortem` added (→ `retro(variant: incident)`); free-vocabulary count corrected to **32** | §6, §12-note |

### Software review

| # | Gap | Disposition | Where |
| --- | --- | --- | --- |
| S1 | No code/PR element | **Folded in** — `change`/`pull-request` element + `implemented_by` relation + explicit code boundary (code in forge, linked by pointer) | §6.2 |
| S2 | No bug/issue entity | **Folded in** — `bug`/`issue` entity `—affects→` spec/release/change | §6.3 |
| S3 | `goal` floats | **Folded in** — `goal —measured-by→ one-pager` + `goal → {product-spec\|deliverable}`; **goal entity named canonical** | §6.5 |
| S4 | v1 spine omits every spec type | **Folded in** — `engineering-spec` promoted into the spine (DoD + test-plan link) (also the shared defect) | §3 |
| S5 | §3 decomposition chain WRONG | **Folded in** — corrected to `product-spec → {feature-spec, ux-spec, engineering-spec}` (siblings) | §6.1 |
| S6 | Output-plane relations undefined | **Folded in** — `assembles` / `covers` / `cites` / `resolved-by` all specified | §7 |
| S7 | Dropped types + postmortem | **Folded in** — `user-story`/`mockup` placed under product/feature specs; `postmortem` → incident `retro` path added | §6 |
| S8 | No verification/QA-result element | **Folded in** — `test-run`/`verification` gates `building → shipped` via `verified-by` | §6.4 |
| S9 | Redundancy | **Folded in** — spec when-to-use table; output plane collapsed to `deliverable` container + `kind` enum; `one-pager` double-placement resolved (activity home) | §6.1, §7 |
| S10 | Minor: 32 not 31; `milestone` in §1 not §7; `daily-note` is an SOP/template not a schema type | **Folded in** — count corrected to 32; `milestone` present in both spine (§3) and v1 slice (§13); `daily-note` reclassified as SOP/template (spine `note` row) | §3, §13, §12-note |

### Shared defect (both reviews)

| Gap | Disposition | Where |
| --- | --- | --- |
| v1 spine defers each type's core artifact | **Folded in (closed)** — spine promotes one build/knowledge artifact per type: `engineering-spec` (software) + `research-synthesis` (research) | §3, §13 |

> **§12-note — the "32 not 31" correction.** Both reviews found the `_headcase` free vocabulary is
> **32 doctypes**, not the 31 the R3 draft claimed; the omitted type was the **`postmortem`** SOP
> (incident review → `retro(variant: incident)`). This document uses 32 and folds `postmortem` into
> the activity plane. (This is the *shipped-vocabulary* count that ADR 0012's `CreateItem` check reads
> today; the net-new v2 elements named above extend it and inherit the same validation, §11.)

## 13. v1 build slice (the beefed spine) + deferred

- **v1 spine slice:** `project · person · source · note · goal · milestone · task · meeting ·
  decision · open-question · engineering-spec · research-synthesis · deliverable` — spans all three
  planes and both PM pillars (vs the old activity-only four), and carries **one core artifact per
  project type** so both types are demonstrable (shared defect closed). `milestone` is now present
  here *and* in the spine table (§3), fixing S10's "in §1 spine but not §7" mismatch.
- **Deferred to later increments (with reasons):**
  - The full research plane *build* — live `finding`/`claim`/`citation`/`experiment`/`dataset`/
    `literature-note` collections — **deferred:** ADR 0009's two-track ordering builds storage only
    after the model finalizes; the model *names* them now (§5), building is downstream.
  - The full software spec-decomposition chain beyond the promoted `engineering-spec`
    (`feature-spec`/`ux-spec`/`user-story`/`mockup`) — **deferred:** added once the spine loop proves
    out, to keep the first slice minimal.
  - `change`/`bug`/`test-run` live tracking — **deferred:** named in the model (§6.2–6.4); the build
    slice lands them after the spine loop, and the code boundary (forge-linked) has no store cost.
  - The cadence docs (`weekly-checkin`/`retro`/`postmortem`) — **deferred:** cadence layers on after
    the core loop.
  - The deferred exec outputs beyond `exec-summary` + `stakeholder-update`, and **all trigger
    design** — **deferred:** Q4 ships those two first and on-demand; `demo-prep` waits for a real
    demo; trigger design is the separate `trigger-design` epic-child (#127).

## 14. Out of scope (restated from #125)

- Implementing any v2 collection, migration, or store change (ADR 0009 forbids v2 storage ahead of
  the finalized model).
- **Finalizing** the model — gated on `model-simulations` (#126); this document lands the fixes and
  rulings only.
- Designing the exec-output triggers — `trigger-design` (#127).
- Versioning the `schema/` contract itself — `schema-versioning` (#124).
- Running the model-simulations walkthroughs — `model-simulations` (#126), which consumes this
  document as input.

## Provenance

Graduated 2026-07-22 from the R3 proposal
[`_meta/research/2026-07-design-session/platform/spec-element-model.md`](../../../_meta/research/2026-07-design-session/platform/spec-element-model.md)
(3 read-only scouts over `_headcase` @ 2026-07-20; frame + spine the foreman's synthesis; two opus
adversarial reviews). Rulings: ADR 0018 (Q1–Q4), ADR 0009 (two-track, staged truth, simulations
gate). Consumed mechanisms: ADR 0011/0012/0013/0017 (§11). Resolves #125; blocks #126; child of #130.
</content>
</invoke>

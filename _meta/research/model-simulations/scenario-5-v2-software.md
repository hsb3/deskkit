---
type: analysis
status: active
created: 2026-07-22
updated: 2026-07-22
tags: [model-simulations, schema-v2, element-model, software]
synopsis: Scenario 5 walkthrough — the v2 element model's own software-project slice (docs/element-model-v2-draft.md §10) traced tabletop against the draft, every element/relation/phase marked OK / FRICTION / DEFICIENCY. The v2-native counterpart the issue's Deliverable A requires.
---

*Scenario 5 of the #126 model simulations (**v2 half**). Walks the software-project slice the v2
element model draws for itself (`docs/element-model-v2-draft.md` §10, over §3–§7) against that draft
model. **Method: tabletop** — the v2 model is a DRAFT with no storage or code to script against
(ADR 0009 "truth regime is STAGED"; §14 forbids v2 storage ahead of finalization), so every step is
a paper trace asking one question per element: does the model name the entity/document/relation the
slice needs, and does it hold together with the shipped mechanics it claims to consume (§11)?*

Status: active (2026-07-22)

## What "walking a paper model" means here

The v1 scenarios drove a running system; the v2 model has no system yet. So a v2 step is **OK** when
the draft names the element + its relations coherently and its build is either shippable on the v1
mechanics (§11) or explicitly deferred with a reason; **FRICTION** when the model is coherent but a
builder would hit an unstated choice; **DEFICIENCY** when the model is internally inconsistent,
silently omits something the slice requires, or asserts a safety property the consumed v1 mechanism
does not actually provide. Deficiencies roll up to `deficiency-report.md` (§"v2 model deficiencies").

## Operator story (from the draft's own §10 software walkthrough)

A team stands up a `project(type=software)`, refines inputs (`source`, `note`, `project-brief`),
sets a measurable `goal` + `milestone`s, decomposes the work
(`one-pager → product-spec → {feature-spec, ux-spec, engineering-spec} → test-plan`), does it
(`task`, `meeting`, `raid`, cadence docs), builds it (`change`, `implemented_by`), tracks defects
(`bug`), verifies it (`test-run` gating `building → shipped`), and — on a trigger — ships outputs
(`release-notes` covering the changes, a status `deliverable` deck).

## Step-by-step trace

| # | Slice step (draft §10 / §) | Elements + relations exercised | Consumes (v1 mechanism §11) | v2 verdict |
|---|---|---|---|---|
| 1 | Create `project(type=software)` (§3 spine) | `project` entity; `type: software\|research`; owner/status/repo-links/dates; optional `workstream` tag (Q2) | ADR 0012 type-vocab at create; ADR 0011 for repo/forge links | **OK** — spine entity fully specified |
| 2 | Capture `source` + `note` [input] (§4, §5.1) | `source` (subkind enum + credibility); `note` (spine lightweight) | ADR 0017 frontmatter `id` for cross-plane identity | **OK** |
| 3 | Write `project-brief` [input] (§4) | `project-brief` doc hung off `project` | ADR 0011 relation to `project` | **OK** |
| 4 | Set `goal` (target + deadline + measure, Q1) + `milestone`s (§3, §6.5) | `goal` entity (measure canonical here); `milestone` under goal; `goal —measured-by→ one-pager`; `goal → {product-spec\|deliverable}` | ADR 0011 relations | **OK** — Q1 measure + canonical-source ruling both landed (§6.5) |
| 5 | Decompose: `one-pager → product-spec → {feature-spec, ux-spec, engineering-spec} → test-plan` [activity] (§6.1) | the corrected sibling chain (S5 fix); `engineering-spec` now spine, carries DoD + `test-plan` link; `one-pager` home = activity | ADR 0011; ADR 0012 for each doc type | **OK** — the §3-vs-§6 chain defect is fixed; when-to-use table (§6.1) disambiguates the specs |
| 6 | `task`s, `meeting`s, `raid`, weekly-checkin / retro [activity] (§6) | activity-plane entities/docs; the `>2 weeks in-progress` PM hard-flag | shipped PM engine (items/transitions) | **OK** |
| 7 | Build via `change` (`implemented_by`) (§6.2) | `change`/`pull-request` entity; `engineering-spec —implemented-by→ change`; `task —implemented-by→ change`; code lives in forge, linked by pointer | ADR 0011 (forge pointer); ADR 0012 (`change` as a new `items.type`) | **FRICTION → DEFICIENCY (V2-D2)** — build deferred (§13, expected); but §11's claim that a new type like `change` inherits the type check "for free" is only *create*-complete (see step 7 note) |
| 8 | Track defects as `bug` (§6.3) | `bug`/`issue` entity; `bug —affects→ {engineering-spec\|release-notes\|change}` | ADR 0011; ADR 0012 | **OK** (named; build deferred §13) |
| 9 | Verify: `test-run` gates `building → shipped` (§6.4) | `test-plan —verified-by→ test-run`; `test-run` carries pass/fail + sign-off the transition reads | shipped PM **gate engine** (`desk_config` rules) | **DEFICIENCY (V2-D1)** — `building → shipped` is a *spec-document status* edge (§6.1's `draft→in-review→approved→building→shipped`), not an edge in the shipped PM item phase-machine (`queue→work→review→terminal`) where gates actually bind; the model names no gate rule realizing `verified-by`, and the two lifecycles are never reconciled |
| 10 | Ship `release-notes` (covers) + status `deliverable` deck [output, on trigger] (§7) | `release-notes —covers→ {change, milestone}`; `deliverable —assembles→ {source, finding, analysis, milestone, …}`; `kind` enum selects form | comms skill (output engine); ADR 0011 | **OK (relations)** but output is **un-exercisable** — trigger design deferred to #127 (V2-A2, accepted-risk) |
| 11 | `decision` log; incidents via `postmortem → retro(variant: incident)`; blockers via >2-week flag (§6) | `decision` entity+doc; `retro` incident variant (R8/S7 fix) | shipped PM | **OK** — `postmortem` omission (R8/S7) is folded in |

## Findings from this scenario

### DEFICIENCY V2-D1 — the software phase-machine is unreconciled with the shipped PM phase-machine (step 9)

§6.1 states the software lifecycle as `draft → in-review → approved → building → shipped` and §6.4
says "`test-run` gates `building → shipped`." But the shipped PM engine — the only thing that
actually *binds gates* — runs a different phase vocabulary, `queue → work → review → terminal`
(Scenario 2, verified live), and binds gate documents on `items.type` at specific edges
(`defaults.go`; only `task work→review` and `decision review→terminal` ship today, self-documented as
a "KNOWN UNAUTHORED DESIGN GAP"). The draft never says whether `draft→…→shipped` is (a) the spec
**document's own `status` axis** (frontmatter), orthogonal to the PM item's `phase` — the reading its
"(unchanged)" parenthetical implies — or (b) a replacement PM phase vocabulary. Nor does it name
which gate rule realizes §6.4's `verified-by` gating. A builder cannot implement the verification gate
from the model as written.

Notably, the draft is scrupulous about exactly this axis-separation elsewhere: §5.3 and §11 spell out
that a `claim`'s status (`supported/contradicted/open`) is "a human-judgment axis **orthogonal to any
workflow `state`**." The same explicitness is simply missing for the spec phase-machine.

**Severity medium** — blocks a coherent v2 activity-plane build; the verification gate (S8's own
folded-in fix) is not implementable as specified. **Disposition: amendment-needed, filed as a
model-track issue** (see report). Ties to the pre-existing default-gate-set "KNOWN UNAUTHORED DESIGN
GAP."

### DEFICIENCY V2-D2 — §11 overclaims the ADR 0012 type-validation "inherited for free" (step 7)

§11 argues each new v2 element type (`change`, `bug`, `test-run`, …) "inherits the identical
`CreateItem`-time vocabulary check for free." True at *create* — but the v1 half proved (v1-D2, filed
#185) that the check is **create-only**: `update_item` sets `type` unvalidated, and an item created
with **no** type advances through document gates ungated. So a new v2 type does **not** get complete
gate-binding safety "for free": a supervised `update` can retype it to a gate-less value, or an
untyped instance can skip the very verification gate §6.4 relies on.

**Severity medium** — the model asserts a safety property the consumed mechanism only partially
provides. **Disposition: amendment-needed (note), depends on #185** — scope §11's claim to *create*
and reference #185's update-parity + empty-type policy. Not separately filed: the engine fix is
already tracked as #185; the doc amendment rides its resolution.

### Cross-scenario deficiencies also implicated

- **V2-D3** (new v2 doc types have no directory / patrol classification) — a `change`/`bug`/
  `engineering-spec` created here has no named `dir_kind` home, so Scenario 1/4's patrol would surface
  it as an orphan or misfiled. See report; amendment-note.

## By-design / accepted-risk observations (not deficiencies)

- **V2-A2** — the output-plane deliverables (step 10) are trigger-gated (Q4) but the trigger design is
  deferred to `trigger-design` (#127). The relations (`assembles`/`covers`) are fully specified; the
  *trigger* cannot be walked here. Accepted-risk, by-design (explicitly its own epic-child).
- **Build deferral is expected, not a gap** — `change`/`bug`/`test-run` live tracking is deferred by
  §13 under ADR 0009's two-track ordering (storage only after finalization). The model *naming* them
  (which it does) is exactly the deliverable at this stage.
</content>
</invoke>

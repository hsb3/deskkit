---
type: plan
status: draft
created: 2026-07-20
updated: 2026-07-20
tags: [dev-tooling, desk-platform, data-model, elements, planes, headcase, software, research, r3]
synopsis: "R3 element-model proposal: every entity, document template, and workflow the platform should carry, organized into the three planes (input / activity / output), for two project types (software + research). Grounded in a 3-scout mining of the _headcase Obsidian knowledge-work OS (31-type schema, ~22 SOP templates, 4 personas, the one-pager->product-spec->engineering-spec->test-plan decomposition + draft->shipped phase machine). Marks what _headcase gives free vs the 5 net-new gaps, beefs up the v1 spine from 4 activity-only entities to a cross-plane backbone, and walks a software and a research project through the planes. Pending adversarial completeness review."
---

# Desk Platform - element model proposal (R3)

_Status: **draft**, pending adversarial completeness review. Sources: 3 read-only scouts over
`/Users/henry/Developer/functionform-headcase/packages/obsidian/data/_headcase` (schema, SOPs, personas),
2026-07-20. Frame + spine are the foreman's synthesis; the reviewers stress-test coverage next._

## The frame

Every element is a **typed entity** or a **document** hanging off one, connected by **typed relations**.
Elements sort into three **planes** that map 1:1 to the exec desk's three jobs, over a shared **spine**:

- **Input plane** - refine information (sources -> synthesized knowledge)
- **Activity plane** - set goals/targets/deadlines and do the work (the PM layer)
- **Output plane** - assemble communication packages (deliverables)
- **Spine** - the cross-cutting backbone every project uses, and the `document + revision` mirror substrate.

`_headcase` has no "plane" concept today; it organizes by *workstream* (product / engineering / pm / +executive
deferred). The plane model is the reframe. Most `_headcase` types slot cleanly; the gaps are named in §5.

## 1. Spine (cross-cutting backbone) - beefed up

The old v1 spine (`task / meeting / person / decision`) is **entirely in the activity plane** - it cannot even
demonstrate the three-plane model. Beefed spine (touches all three planes + both PM pillars):

| Element | Kind | From _headcase? | Role |
| --- | --- | --- | --- |
| `project` | entity | yes (`project`) | container; `type: software \| research`; owner, status, repo links, dates |
| `person` | entity | net-new (implied by author/owner/attendees) | stakeholder / collaborator / contact |
| `goal` | entity | **NET-NEW** | measurable target + deadline - the "set goals/targets/deadlines" pillar |
| `milestone` | entity | partial (roadmap phases, `target_ship_date`) | dated commitment under a goal |
| `task` | entity | yes (`task` lightweight) | action item incl. non-code ("meet with Tony") |
| `meeting` | entity + doc | yes (`meeting-notes`) | interaction + captured notes/decisions/actions |
| `decision` | entity + doc | yes (`decision`) | ADR-style record; promoted from the flat DECISIONS registry |
| `open-question` | entity | yes (OPEN_QUESTIONS registry) | promote registry -> queryable entity |
| `source` | entity | **NET-NEW** | an input reference: doc / url / dataset / paper |
| `note` | entity | yes (`journal`/`idea`/`daily-note`) | lightweight capture |
| `deliverable` | entity | **NET-NEW** (formalizes "comm packet") | an output package (§4) |

## 2. Input plane - refine information

| Element | Kind | From _headcase? | Software / Research |
| --- | --- | --- | --- |
| source | entity | net-new | both (research-heavy) |
| note / idea / journal | entity | yes | both |
| `research-synthesis` | doc | yes | **research** |
| `analysis` | doc | yes (options A/B/C -> recommendation) | both |
| `project-brief` | doc | yes (north-star framing) | both |
| `user-journey` | doc | yes | product/research |
| `technical-design` | doc | yes | **software** (architecture reference) |
| `runbook` | doc | yes | **software** (operational knowledge) |

**Workflow (refine-information pipeline):** capture `source`/`note` -> `research-synthesis` (research) or
`analysis` (options) -> `decision`. Software adds `technical-design` as durable architecture reference.

## 3. Activity plane - set goals/targets/deadlines + do the work

| Element | Kind | From _headcase? | Software / Research |
| --- | --- | --- | --- |
| project, goal, milestone, task, meeting | entity | mixed (goal/milestone net-new) | both |
| `raid` | doc/entity | yes (risks/assumptions/issues/deps) | both |
| `roadmap` | doc | yes (themes/phases, no dates) | both |
| `weekly-checkin`, `retro` | doc | yes (cadence) | both |
| `one-pager` | doc | yes (gate before >2 days) | both (pitch/frame) |
| `product-spec`, `feature-spec`, `ux-spec` | doc | yes | **software/product** |
| `engineering-spec`, `test-plan` | doc | yes | **software** |
| `research-brief` / `investigation-plan` | doc | **NET-NEW** | **research** (see §5) |

**Workflows:**
- **Software decomposition (from _headcase):** `one-pager -> product-spec -> {feature-spec, ux-spec} ->
  engineering-spec -> test-plan`; phase machine `draft -> in-review -> approved -> building -> shipped`;
  the PM ">2 weeks in-progress" hard-flag; weekly-checkin/retro cadence.
- **Research decomposition (NET-NEW, the research analog):** `research-brief (question) -> investigation-plan
  -> source-gathering (tasks) -> research-synthesis -> findings/decision`.
- **Goal tracking (NET-NEW):** `goal -> milestone -> task`, with deadline surfacing and the >2-week flag.

## 4. Output plane - communication packages

Thinnest in `_headcase` (comm-packets have no schema type; the Executive workstream - `exec-summary`,
`stakeholder-update`, `demo-prep` - is explicitly deferred). The platform formalizes it:

| Element | Kind | From _headcase? | Software / Research |
| --- | --- | --- | --- |
| `deliverable` / comm-package | entity | net-new (formalizes "comm packet") | both |
| `release-notes` | doc | yes (composite) | **software** |
| `sow` | doc | yes (client contract) | both |
| `one-pager` | doc | yes (doubles as pitch) | both |
| `exec-summary`, `stakeholder-update`, `demo-prep` | doc | **NET-NEW** (deferred in _headcase) | both |
| `research-report` | doc | **NET-NEW** | **research** |
| briefing / status-deck | doc | via the exec-desk `comms` skill | both |

**Workflow:** a `deliverable` assembles content from input + activity artifacts -> renders via the exec-desk
**comms** skill (deck-builder / pptx-themes / audio) -> delivered. The comms skill is the output-plane engine.

## 5. What _headcase gives free vs the 5 net-new gaps

**Free:** the 31-type schema vocabulary; ~22 SOP document templates (the "templates" leg of the north-star);
the persona model (product/engineering/pm/general); the software decomposition contract + phase machine;
frontmatter/wikilink/decomposition conventions (the relation model).

**Net-new (the gaps the platform must fill):**
1. **`goal` entity** - measurable target + deadline. `_headcase` buries success metrics inside one-pagers and
   has no first-class goal/OKR object. Required by pillar 2.
2. **Output plane** - formalize `deliverable`/comm-package + the deferred exec/stakeholder outputs, wired to
   the comms skill.
3. **Research-project decomposition** - `_headcase` has one research *document* (`research-synthesis`) but no
   research *project* structure. Add `research-brief -> investigation-plan -> ... -> findings`.
4. **`source` entity** - first-class references/citations; `_headcase` only has a Sources *section* in a doc.
5. **Registries -> entities** - promote the flat append-only `DECISIONS / TASKS / OPEN_QUESTIONS` registries to
   queryable records. This IS the "structured data in PB" thesis.

## 6. Walkthroughs (does the model run a real project?)

**Software project.** `project(type=software)` -> capture `source`/`note`, write `project-brief` [input] ->
set `goal` + `milestone`s; `one-pager -> product-spec -> engineering-spec -> test-plan`, `task`s, `meeting`s,
`raid`, weekly-checkin/retro [activity] -> `release-notes` + a status `deliverable` deck [output]. Decisions
logged as `decision`; blockers via the >2-week flag.

**Research project.** `project(type=research)` -> gather `source`s, `note`s [input] -> `research-brief` +
`investigation-plan`, `goal`/`milestone`/`task`s, `analysis` of options [activity] -> `research-synthesis` ->
`research-report` + briefing `deliverable` [output]. Decisions + open-questions tracked as entities.

## 7. v1 build slice (the beefed spine) + deferred

- **v1 spine:** `project · person · source · note · goal · task · meeting · decision · open-question ·
  deliverable` - spans all three planes and both PM pillars (vs the old activity-only four).
- **Deferred to later increments:** the full software spec-decomposition chain, the research decomposition,
  the cadence docs (weekly-checkin/retro), and the deferred exec outputs - added once the spine loop proves out.

## 8. Open questions for Henry (feeds R3 q4)

- Is `goal` the right unit, or do you want OKR-style `objective + key-results`?
- Keep `_headcase`'s workstream tag (product/engineering/pm) alongside the plane model, or drop it?
- Research decomposition: is `research-brief -> investigation-plan -> synthesis -> report` the right spine,
  or do you run research differently?
- Which of the deferred exec outputs (`exec-summary`, `stakeholder-update`, `demo-prep`) matter first?

## R3 review findings (adversarial, 2026-07-20)

Two opus reviewers stress-tested this proposal. **Research review returned; software review still running** —
fold BOTH before presenting to Henry. The research plane needs the most work.

### Research review — verdict: runs research *intake* but not research end-to-end
Verified all "from _headcase" vs "net-new" claims as accurate (gaps 1/3/4 confirmed against source). Top gaps:
1. **No evidence triple** — add `finding`/`claim` (status supported|contradicted|open) + `citation`/`excerpt`
   (passage + locator from a `source`); relations `claim -supported-by-> citation -from-> source`. Without it,
   "cited research" is un-queryable prose — contradicts the registry->entity thesis (§5 gap 5).
2. **`source` too flat** — give it subkinds (paper/url/vendor-doc/interview/internal) + excerpt, access_date,
   version, credibility (primary/secondary). Split **`dataset`** out of `source` (it is computed over, not cited).
3. **No empirical element** — add `experiment`/`investigation-run` (config + as-run protocol + result;
   reproducibility hook: seed/env/dataset-version/code-ref), linked to the claims it supports. `investigation-plan`
   = the planned method the run realizes.
4. **Research is a loop, not a waterfall** — redraw `brief <-> plan <-> gather/experiment <-> synthesize ->
   findings -> new open-questions -> back to brief`; make `open-question` the re-entry point; drop
   "source-gathering" as a discrete phase.
5. **v1 spine can't run research** — it defers `research-synthesis` (the one FREE research doc). Un-defer it +
   add finding+citation, else a research slice yields sources+notes+decision but no research artifact.
6. **No literature-note** — the per-source reading artifact (one note bound to one source:
   argument/method/limitations/quotes). Chain: `source -> literature-note -> claim/finding -> synthesis`.
7. **Boundary redundancy** — `research-report` should be a `deliverable` VIEW (output plane), not a doc type;
   keep the clean `analysis -> decision` pair; add a "when to use which" table (analysis/synthesis/decision).
8. **Minor** — `postmortem` SOP omitted from the §5 free list; schema is **32** types, not 31.

### Software review — verdict: runs the planning + comms arc, NOT the build-and-ship core
Provenance claims re-verified accurate (goal/source net-new; registries real; exec outputs deferred). Top gaps:
1. **No code/PR element** — build phase is invisible; `project` has repo links but nothing hangs off them. Add
   a `change`/`pull-request` element (or an explicit "code in GitHub, linked by URL" boundary) + an
   `implemented_by` relation from `engineering-spec`/`task` to the code.
2. **No bug/issue entity** — only undifferentiated `task` + raid's issues leg. Add `bug`/`issue` (priority enum
   already exists), related to the spec/release it affects. The clearest registry->entity win the doc missed.
3. **`goal` floats** — wired only downward (`goal->milestone->task`); no edge up to one-pager/product-spec/
   deliverable; and duplicates the one-pager's Success Metrics. Add `goal <-> one-pager` (measured-by) +
   `goal -> {product-spec|deliverable}`; state which is canonical.
4. **v1 spine omits every spec type** — a software slice can't state what it builds. Promote `engineering-spec`
   (carries DoD + test-plan link) into v1; consider trading `person` out for minimality.
5. **§3 decomposition chain is WRONG** — `feature-spec`/`ux-spec` are SIBLINGS of `engineering-spec` (all
   children of `product-spec`), not parents. Fix to `product-spec -> {feature-spec, ux-spec, engineering-spec}`;
   reconcile §3 with §6 (which is already correct).
6. **Output-plane relations undefined** — `deliverable` assembles nothing (no relation); `release-notes` covers
   nothing; `source` has no cites edge; `open-question` no resolved-by. Specify all four.
7. **Dropped types + postmortem** — `user-story`/`mockup` in no plane; `postmortem` SOP (incident review ->
   retro `variant: incident`) unmentioned. Fold in or justify; add an incident path.
8. **No verification/QA-result element** — `test-plan` is only the plan; nothing records execution/sign-off
   gating `approved->building->shipped`. Add `test-run`/`verification`.
9. **Redundancy** — product/feature/ux-spec undifferentiated (add when-to-use); output plane sprawls ~10 forms
   (collapse to `deliverable` container + a `kind` enum); `one-pager` double-placed (activity + output).
10. **Minor** — 32 types not 31; `milestone` in §1 spine but not §7; `daily-note` is an SOP/template, not a schema type.

### Shared defect flagged by BOTH reviews
The v1 spine defers the core artifact of each project type (`engineering-spec` for software, `research-synthesis`
for research) — so neither type can be demonstrated on the spine. **Fix:** promote one build/knowledge artifact
per type into the spine. **Next action:** fold both reviews into a revised model, then produce the review deck
for Henry + the 4 open questions (§8).

## Provenance

Mining: 3 read-only scouts over `_headcase` @ 2026-07-20 (schema.yaml / shared / sops / agents). Parent design:
[plan.md](plan.md). Store + planes: [../extenders-estate/system-cohesion-and-datamodel.md](../extenders-estate/system-cohesion-and-datamodel.md).

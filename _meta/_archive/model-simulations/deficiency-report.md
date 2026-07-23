---
type: analysis
status: active
created: 2026-07-21
updated: 2026-07-22
tags: [model-simulations, deficiency-report, schema-v2, gate]
synopsis: The #126 model-simulations deficiency roll-up (v1 + v2 halves) — every DEFICIENCY with severity, source scenario/step, and a disposition per the pass bar; FRICTIONs listed separately; the v2-model section is now walked (draft docs/element-model-v2-draft.md, #125 merged).
---

*The output artifact of #126 (deliverable C). Rolls up every deficiency both halves of the
walkthroughs surfaced, each with a disposition satisfying the pass bar. FRICTIONs (harvest-loop
input, not gate) are listed separately. The v2 section is now filled — `element-model-revision`
(#125) merged the draft to `docs/element-model-v2-draft.md`, unblocking the v2 pass.*

Status: active (2026-07-22) — **v1 + v2 halves complete.**

## #125 → #126 blocked-by relationship (acceptance-criterion 5)

`model-simulations` (#126) was **blocked by** `element-model-revision` (#125): the v2 pass needed a
model that had already absorbed the ADR 0018 Q1–Q4 rulings and both adversarial reviews' fixes.
**That relationship is recorded on the board and now satisfied:** #125 carries `blocking: #126`;
#126 carries `blocked-by: #125` (both confirmed via `gh issue view`, both now CLOSED — #125 merged
the model to `docs/element-model-v2-draft.md`, graduated from the frozen R3 source doc). The
ordering was enforced on the board, not just in prose, exactly as the criterion requires, and it is
now discharged: the v2 walkthroughs below were run against the merged draft.

## Pass bar (deliverable D, mirroring the schema-v2 epic close-when)

> Every deficiency listed here is either **filed as its own issue** (linked) or **explicitly
> recorded as accepted-risk with a stated rationale**, before `element-model-revision` (#125) is
> marked finalized. Each v2 deficiency (added later) must additionally be closed by an amendment
> to #125 or recorded accepted-risk before #125 finalizes.

**v1-half assessment: PASS.** Both v1 deficiencies below have a disposition (both filed as issues —
D1 → #184, D2 → #185). No deficiency is left undispositioned.

**v2-half assessment: PASS.** Four v2 deficiencies (V2-D1…V2-D4) each carry a disposition — one
filed as a model-track amendment issue (V2-D1 → #197), three recorded as amendment-needed notes
(two of which ride already-filed engine issues #185 / #184). Three items are recorded accepted-risk
/ by-design (V2-A1…V2-A3) with rationale. No v2 deficiency is left undispositioned; every one is
either "closed by an amendment to the v2 model" (tracked as an issue/note) or accepted-risk with a
stated rationale, as the pass bar requires **before the model is marked finalized**. The v2 model
(`docs/element-model-v2-draft.md`) remains `status: draft`; finalizing it is out of scope here (that
is the epic-#130 act this report gates, not performed in #126).

## Deficiencies (gate-bearing)

| ID | Title | Severity | Surfaced in | Disposition |
|---|---|---|---|---|
| D1 | Finding-disposition surface has no id-bearing read path | Medium | Scenario 1, steps 4–5 | **Filed → #184** (draft below) |
| D2 | `update_item` skips the `type` vocab check `create_item` enforces; untyped items bypass gates | Medium | Scenario 2, Part A / D2 probes | **Filed → #185** (draft below) |

### D1 — Finding-disposition surface has no id-bearing read path

**What.** `deskkit findings dispose <finding-id> --as <...>` (0.8.0, CLI-only supervised,
workflows.md §1.5) resolves its target by record id (`FindRecordById`, `dispose.go`). But the
finding read surfaces `query findings` and `query uncollapsed` emit only
`findingBrief{path, detail}` (`internal/modules/librarian/tools/query.go:99-102`) — **no id**.
The `feedback` query, by contrast, emits an id (`feedbackBrief`, `query.go:167-168`). So the id a
supervisor needs is obtainable only from the PocketBase admin GUI or raw REST — the CLI
disposition workflow cannot be completed from the CLI.

**Evidence.** `probes/output/out-s1.txt` step 4: `query findings` returns
`{"kind":"findings","count":2,"by_rule":{"R1":[{"path":"tasks/needs-fm.md","detail":"..."}],...}}`
with no id. Engine is sound (7 tests, `dispose_test.go`); this is a missing read surface only.

**Severity.** Medium — a shipped feature is not operable from its own intended (CLI) surface.

**Drafted issue (ready to file):**

- **Title:** `fix(librarian): expose finding record id in query findings/uncollapsed so 'findings dispose' is usable from the CLI`
- **Body:**
  > `deskkit findings dispose <finding-id>` (the supervised disposition lifecycle, §1.5) takes a
  > patrol-finding **record id**, but no read surface emits one: `query findings` and
  > `query uncollapsed` reduce each finding to `{path, detail}` (`query.go:99-102`), unlike the
  > `feedback` query which includes `id` (`query.go:167-168`). Today the only way to get the id is
  > the admin GUI / REST, so the CLI-only disposition workflow can't be completed from the CLI.
  >
  > **Fix:** add the finding `id` to `findingBrief` (and thus to `query findings` /
  > `query uncollapsed` / the `--include-disposed` view), matching `feedbackBrief`. Keep `path` +
  > `detail`. No migration; JSON-additive. Update the tool-surface doc + `verify.sh` to assert the
  > id is present and round-trips through `findings dispose`.
  >
  > Surfaced by the #126 model simulations, Scenario 1 (`_meta/research/model-simulations/
  > scenario-1-librarian-chain.md`). Identity-neutral.

### D2 — `type` validation is asymmetric; untyped items bypass document gates

**What.** Document gates key on `items.type` (`engine.go:367`). Two verified supervised
operations leave an item with a type the gate cannot bind, silently disabling it:

1. **`update_item` does not validate `type`** although `create_item` does. Create refuses an
   unknown type (`engine.go:142` `if !vocab.KnownType(in.Type)`); update sets it blindly
   (`queries.go:530-531`). Verified: `pm create --type tsak` → `Error: unknown item type "tsak"`;
   `pm update <id> --type tsak` → succeeds, type becomes `tsak` (`probes/output/out-d3d.txt`).
2. **An item created with no `type`** advances through gates ungated (empty type binds nothing).
   Verified: a no-type item advanced `work→review` `rc=0 {"phase":"review"}` with no gate doc on
   disk (`probes/output/out-d3c.txt`).

**Severity.** Medium — the gate system's core safety rests on `type`; an ordinary update or an
omitted type removes the gate without any signal.

**Drafted issue (ready to file):**

- **Title:** `fix(pm): validate items.type in update_item (parity with create_item) and decide the empty-type gate policy`
- **Body:**
  > Document gates bind on `items.type` (`engine.go:367`). `create_item` validates `type` against
  > the schema vocabulary and refuses an unknown one (`engine.go:134-144`), but `update_item` sets
  > it with no check (`queries.go:530-531`), and an item created with no type at all advances
  > through every document gate ungated. Both let a supervised operation silently disable a gate.
  >
  > Repro (verified in #126 sims): `pm create --type tsak` is rejected; `pm update <id> --type
  > tsak` is accepted (type becomes `tsak`); a no-type item advances `work→review` with no gate
  > document present.
  >
  > **Fix:** (a) apply the same `vocab.KnownType` check in `update_item` that `create_item` uses;
  > (b) decide + enforce the empty-`type` policy — either forbid transitioning an untyped item
  > through a gated edge, or require `type` at create. Add engine tests for both. Reconcile the
  > stale claim in `_meta/research/2026-07-design-session/data-model.md` §4 ("type is
  > unvalidated") in the same change. Surfaced by #126 Scenario 2. Identity-neutral.

## Frictions (harvest-loop input — NOT gate-bearing)

Listed separately per the brief: frictions feed the improvement/harvest loop, they do not gate
v2 finalization.

| ID | Friction | Severity | Surfaced in | Disposition |
|---|---|---|---|---|
| F1 | Cascade-initiated block/unblock is attributed to the triggering human actor | Low | Scenario 2, step B6 | Harvest-loop item; optional accepted-risk |

**F1 detail.** An engine-cascade auto-unblock records `actor:"operator", actor_kind:"human"` — the
identity of whoever advanced the blocker — with only the prose `detail` distinguishing it from a
human action (`probes/output/out-frictions.txt`). Filtering the audit by `actor_kind` misclassifies
cascade rows. Low-severity audit-fidelity nuance; the cascade works correctly. Suggested harvest
action: attribute cascade-initiated `transitions` rows to a system/agent actor kind (or add a
`cascade: true` flag) so automated writes are distinguishable on a queryable axis. If not taken,
**accepted-risk**: the `detail` string already records the cause, so provenance is not lost, only
harder to query.

## Accuracy observations (recorded with evidence; not gate-bearing)

Three are **stale claims in `../2026-07-design-session/data-model.md`** (commit 51235f6) that the
live tree contradicts. Correcting that dossier is an **out-of-scope follow-up** (this work owns
only `_meta/research/model-simulations/`); recorded here so the corrections are not lost.

| ID | Observation | Evidence |
|---|---|---|
| O1 | data-model.md §4 says `items.type` is unvalidated ("a typo'd type advances ungated"). **Stale** — create validates (`engine.go:142`). Real residual is D2 (update + empty type). | `out-d3d.txt`, `out-d3c.txt` |
| O2 | data-model.md §5.3 (C4 residue: `query summary`/`uncollapsed` count disposed findings). **Stale/resolved** — the 0.8.0 bug floor made all three surfaces filter `disposition='open'` (`query.go:483,505,522`). | source read (`query.go:495-527`) |
| O3 | data-model.md Gaps flags `transitions.event = gate_refused` as maybe-dead. **Live** — every gate refusal writes a `gate_refused` row. | `out-s2.txt` step A7 |
| O4 | Un-frontmattered files escape patrol's typed rules but are caught by `query orphans` (working-as-intended). Baseline procedure should run `query orphans` alongside `patrol`. | `out-frictions.txt` F2 section |

## v2 model deficiencies (WALKED — draft `docs/element-model-v2-draft.md`, #125 merged)

_The v2 side is now run. `element-model-revision` (#125) merged the reviewed model to
`docs/element-model-v2-draft.md` (folding ADR 0018 Q1–Q4 + both reviews' 18 gaps), unblocking this
pass. **Method: tabletop** — the v2 model is a DRAFT with no storage or code (ADR 0009 STAGED truth;
draft §14 forbids v2 storage ahead of finalization), so each walkthrough is a paper coherence trace,
not a scripted run. The scenario set: the four v1 scenarios re-read against the v2 model (their `v2
delta` columns + per-scenario "v2 assessment" sections), **plus** the model's own two native
walkthroughs the issue's Deliverable A requires — Scenario 5 (v2 software) and Scenario 6 (v2
research), tracing draft §10._

**Framing (not a deficiency), recorded so it is not misread as one — V2-A3.** The v2 element model
is a **content/knowledge-domain reframe** (three planes over a beefed spine: `project`, `goal`,
`source`, `claim`, `deliverable`, …). It does **not** redefine the librarian operational collections
(`files`, `patrol_findings`, `revisions`, `patrol_log`, `adoption_log`) or the PM engine mechanics —
those are model-agnostic infrastructure it explicitly *consumes* (draft §11: ADR 0011/0012/0013/0017).
So Scenarios 1 & 2's *operational* steps (sweep→patrol→apply→restore; the gated-transition/cascade
engine) are **unchanged** by v2 — what v2 changes is the *content vocabulary* those engines carry.
The v2 model "not having" the librarian collections is therefore **by design, not a gap** (the
issue's stop-and-report trigger — "missing a v2-equivalent of a v1 collection" — does not fire: the
operational collections are infrastructure the element model sits atop, not content it should mirror).

### Deficiencies

| ID | Title | Severity | Surfaced in | Disposition |
|---|---|---|---|---|
| V2-D1 | Software spec phase-machine (`draft→in-review→approved→building→shipped`, §6.1) is unreconciled with the shipped PM item phase-machine (`queue→work→review→terminal`); §6.4's "`test-run` gates `building→shipped`" names no gate rule and no gated edge in the live PM vocab | Medium | Scenario 5 step 9; Scenario 2 v2 assessment | **Amendment-needed → filed #197** (model-track) |
| V2-D2 | §11 overclaims ADR 0012 type-validation "inherited for free" — v1-D2/#185 shows it is create-only (`update_item` unvalidated; empty-type bypasses gates), so new v2 element types do NOT get complete gate-binding safety for free | Medium | Scenario 5 step 7; carryover of D2/#185 | **Amendment-needed (note); rides #185** |
| V2-D3 | The ~15 net-new v2 element/doc types (`change`, `bug`, `engineering-spec`, `literature-note`, `deliverable`, …) have no directory-placement (`dir_kind`) or patrol/`entity_type` classification named; Scenarios 1 & 4 rely on patrol classifying docs by type/dir, so new types surface as orphans/misfiled until the librarian taxonomy is extended | Low | Scenario 1 & 4 v2 assessment; Scenario 5/6 | **Amendment-needed (note)** — add a librarian-taxonomy line to §13 deferred list |
| V2-D4 | The `claim` evidentiary-status read surface (modeled on findings-disposition, §5.3/§11) risks inheriting v1-D1/#184's missing-id read-path gap when the evidence triple is built | Low | Scenario 6 step 4 | **Amendment-needed (note); rides #184** |

### V2-D1 — spec phase-machine unreconciled with the PM phase-machine (filed #197)

**What.** §6.1 names the software lifecycle `draft → in-review → approved → building → shipped` and
§6.4 says "`test-run` gates `building → shipped`." But the shipped PM engine — the only thing that
binds gate documents — runs `queue → work → review → terminal` (Scenario 2, verified live) and binds
gates on `items.type` at named edges (`defaults.go`; only `task work→review` + `decision
review→terminal` ship, self-documented as a "KNOWN UNAUTHORED DESIGN GAP"). The draft never says
whether `draft→…→shipped` is the spec **document's `status` axis** (orthogonal to `items.phase`, the
reading its "(unchanged)" parenthetical implies) or a replacement PM phase vocabulary, and names no
gate rule realizing §6.4's `verified-by`. The draft is scrupulous about this exact axis-separation
for `claim` status (§5.3/§11, "orthogonal to any workflow `state`") — the same note is just missing
for the spec machine.

**Severity** medium — blocks a coherent v2 activity-plane build; the S8 verification-gate fix is not
implementable as specified. **Disposition: amendment-needed, filed #197.** The amendment (make the
axis explicit + name the gate rule, or record it blocked on the default-gate-set owner ruling) is a
`docs/`-scoped edit to `docs/element-model-v2-draft.md`, on the model's own follow-up track — **not**
performed here (that file is out of this deliverable's ownership).

### V2-D2 — §11 overclaims the ADR 0012 type-validation inheritance (rides #185)

**What.** §11 argues each new v2 element type "inherits the identical `CreateItem`-time vocabulary
check for free." True at *create* — but v1-D2 (#185) proved the check is create-only: `update_item`
sets `type` unvalidated, and a no-type item advances through document gates ungated. So a new v2 type
does not get complete gate-binding safety "for free": a supervised `update` can retype it to a
gate-less value, or an untyped instance can skip the very verification gate §6.4 relies on.

**Severity** medium (doc-accuracy of a safety claim). **Disposition: amendment-needed (note), rides
#185.** No separate issue: the engine fix is #185; the model amendment (scope §11's claim to *create*,
reference #185's update-parity + empty-type policy) rides its resolution.

### V2-D3 — new v2 doc types have no directory / patrol classification

**What.** The v2 model names ~15 net-new element/document types but says nothing about where they
live on disk (`dir_kind`) or how patrol's mechanical rules (R1 frontmatter, R3 type/dir match) and
the `entity_type` classifier recognize them. Scenarios 1 & 4 depend on patrol classifying docs by
type/dir; without a taxonomy extension a `change`/`engineering-spec`/`literature-note` would surface
as a `query orphans` hit or an R3 misfile.

**Severity** low — build-time; the plane builds are already deferred (§13). **Disposition:
amendment-needed (note)** — the model should *name* the librarian directory-taxonomy + patrol-rule
extension in its §13 deferred list so the completeness item is tracked, rather than leaving it
implicit. Non-blocking; folded into the V2-D1 issue's "coherent build" scope if the owner prefers one
tracker.

### V2-D4 — `claim` status read surface risks the D1 gap (rides #184)

**What.** §5.3/§11 model the `claim` status axis (`supported/contradicted/open`) on the shipped
findings-disposition pattern (ADR 0013) — a sound reuse. But v1-D1 (#184) showed that pattern's read
surface emits `{path, detail}` with no record id, so its CLI disposition workflow is inoperable. A
`claim`-status surface built the same way reproduces the gap unless the model states its status read
surface must expose the record id.

**Severity** low — forward-looking; the research plane is build-deferred (nothing broken today).
**Disposition: amendment-needed (note), rides #184** — add a "status read surface must expose the
record id" constraint to §5.3/§11.

### v2 accepted-risk / by-design (not deficiencies)

| ID | Item | Rationale (accepted-risk / by-design) |
|---|---|---|
| V2-A1 | The entire research plane (evidence triple, `experiment`, `dataset`, `literature-note` collections) is paper — no runnable behavior | **By-design.** ADR 0009's two-track ordering builds storage only after the model finalizes; §13/§14 defer the research-plane build explicitly. The model *names* the elements (which is the deliverable at this stage); Scenario 6 confirms the slice is coherent end-to-end (§9 chain has no missing link). Accepted-risk. |
| V2-A2 | Output-plane deliverables are trigger-gated (Q4) but the trigger design is deferred to `trigger-design` (#127); the output walkthroughs name `assembles`/`covers` relations but cannot exercise a trigger | **By-design.** Q4 ships `exec-summary` + `stakeholder-update` on-demand and defers all trigger design to its own epic-child (#127). Accepted-risk — tracked as a separate issue, not a v2-model gap. |
| V2-A3 | The v2 element model does not redefine the librarian operational collections or PM engine mechanics | **By-design framing** (see the box above §Deficiencies). Not a gap — recorded so "missing librarian collections" is not misread as a v2 deficiency. |

## Board follow-ups (out of scope for this deliverable, listed for the owner)

- ✅ **D1 and D2 filed** — #184 (finding-id read path) and #185 (`update_item` type validation +
  empty-type gate policy). Linked in the tables above.
- ✅ **#125 → #126 blocked-by relationship recorded on the board** (AC5) — see the top of this
  report; #125 carries `blocking: #126`, #126 carries `blocked-by: #125`, both now CLOSED.
- ✅ **V2-D1 filed → #197** (model-track amendment: reconcile the spec/PM phase-machines + name the
  `building→shipped` gate rule).
- **Amend `docs/element-model-v2-draft.md` (model's own follow-up track, NOT this deliverable):**
  V2-D1 (#197); V2-D2 §11 create-only scoping (rides #185); V2-D3 §13 librarian-taxonomy line;
  V2-D4 §5.3/§11 claim-status read-surface constraint (rides #184). Each is a `docs/`-scoped edit to
  a file this deliverable does not own; performed when the model is revised toward "finalized."
- **Before the v2 model is marked FINALIZED** (epic #130 close-when / this report's pass bar): every
  V2-D deficiency above must be either amended into the model or the amendment-note re-recorded as
  accepted-risk. All four currently carry an amendment disposition; none is undispositioned.
- Correct data-model.md §4 (O1), §5.3 (O2), and Gaps/`gate_refused` (O3) — the change that closes
  #185 is the natural home for the O1 correction.

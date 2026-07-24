---
title: Reconcile the software spec phase-machine with the PM item phase-machine
type: spec
status: planned
created: 2026-07-23
purpose: >
  Build plan for #197 (label gate:v2-final): amend docs/development/specs/element-model-v2-draft.md so the
  §6.1 software-spec lifecycle (draft -> in-review -> approved -> building -> shipped) is
  explicitly reconciled with the shipped PM item phase-machine (queue -> work -> review ->
  terminal), and §6.4's building -> shipped verification gate is either named as a concrete
  gate rule or explicitly recorded as blocked on the owner's pending default-gate-set ruling.
  Model-track, docs-only; closing #197 clears the last blocker on epic #130.
notes: >
  Tracks issue #197 (the one buildable-now item on the schema-v2 track; closing it closes
  epic #130). Provenance: #126 model simulations v2 half, Scenario 5 step 9 (V2-D1); the
  model itself landed as #125. See _meta/plans/phase-machine-reconciliation/issue-body.md
  for the staged issue body this plan builds against.
---

# Reconcile the software spec phase-machine with the PM item phase-machine

_Amend `docs/development/specs/element-model-v2-draft.md` to state explicitly which axis its
`draft -> in-review -> approved -> building -> shipped` software lifecycle is, and to name (or
explicitly defer) the PM gate rule that realizes §6.4's `building -> shipped` verification gate.
**Residual: none of this has shipped** — the draft currently states the lifecycle and the gate
claim but never reconciles either against the shipped PM machine; this plan is the full scope,
not a remainder of prior work. Docs-only; no storage, no code (ADR 0009's staged two-track
ruling, restated as the draft's own SS14 Out-of-scope, `docs/development/specs/element-model-v2-draft.md:456-459`)._

Status: draft (this plan) · Date: 2026-07-23

## Tracking

- **#197** (label `gate:v2-final`) — the tracking issue this plan builds against. Filed
  2026-07-22 by the wave-v4 model-simulations crew
  (`_meta/HANDOFF.md:54-56`). Staged body: `_meta/plans/phase-machine-reconciliation/issue-body.md`.
- **Epic #130** (`epic: schema-v2 element track`) — #197 is the last open item blocking its
  close (`_meta/HANDOFF.md:54-56`, `_meta/plans/README.md:39`).
- **Provenance:** `#126` (v1 + v2 model simulations, CLOSED) — Scenario 5 step 9 flags the
  deficiency as **V2-D1**
  (`_meta/research/model-simulations/scenario-5-v2-software.md:49`, disposition detail
  `:55-70`; rolled up at `_meta/research/model-simulations/deficiency-report.md:179`,
  `:184-201`). `#125` (`element-model-revision`, CLOSED) — landed the draft this plan amends.
- **Contract impact:** none. The change surface is a single `docs/`-scoped prose file
  (`docs/development/specs/element-model-v2-draft.md`). No API, DB, codegen, or other source-of-truth surface is
  touched — see Gate & contract hygiene below.

## The problem (grounded in source)

**What the draft states today.**

- SS6.1 names the software lifecycle: "Phase machine (unchanged): `draft -> in-review ->
  approved -> building -> shipped`, with the PM '>2 weeks in-progress' hard-flag and the
  weekly-checkin / retro cadence." (`docs/development/specs/element-model-v2-draft.md:198`)
- SS6.4 states the verification gate: "`test-plan` is only the *plan*; nothing recorded
  execution/sign-off, yet the phase machine gates `approved -> building -> shipped` on
  verification. Add a `test-run` / `verification` element: `test-plan -verified-by-> test-run`,
  and `test-run` carries the pass/fail + sign-off that the `building -> shipped` transition
  reads." (`docs/development/specs/element-model-v2-draft.md:224-229`) The SS12 disposition table marks gap S8
  "Folded in" on this basis (`docs/development/specs/element-model-v2-draft.md:418`).
- SS5.3 and SS11 establish a **precedent pattern** elsewhere in the same document: a `claim`'s
  status (`supported`/`contradicted`/`open`) is explicitly called "a **human-judgment axis
  orthogonal to any workflow `state`**" (`docs/development/specs/element-model-v2-draft.md:148-150`), and SS11
  restates it as "the design precedent for the `claim` status axis... an **orthogonal judgment
  axis, not a lifecycle `state`**" (`docs/development/specs/element-model-v2-draft.md:375-381`), modelled
  explicitly on ADR 0013's findings-disposition mechanism.

**What is missing.** The draft never applies the SS5.3/SS11 orthogonality-note pattern to its own
SS6.1 lifecycle. It does not state whether `draft -> ... -> shipped` is the spec document's own
`status` axis (orthogonal to the PM item's `phase`) or a replacement PM phase vocabulary, and SS6.4
names no PM gate rule realizing `verified-by`.

**What the shipped PM system actually does (the machine that would enforce this gate).** The PM
item phase machine is `queue -> work -> review -> terminal` (`docs/development/specs/pm-system-v1-spec.md:398`,
full machine `docs/development/specs/pm-system-v1-spec.md:471-501`), explicitly "a small, code-enforced machine,
deliberately separate from `status_label`" (`docs/development/specs/pm-system-v1-spec.md:473`) and "data-in-code;
extending it... is a code change, not a config change -- deliberate" (`docs/development/specs/pm-system-v1-spec.md
:500-501`). Gates bind on `items.type` at specific edges via `desk_config` YAML rules
(`docs/development/specs/pm-system-v1-spec.md:494-498`); the shipped default set carries exactly two rules -- `task
work->review` and `decision review->terminal` -- self-documented as a "KNOWN UNAUTHORED DESIGN
GAP" the owner has not yet ruled on wholesale
(`librarian/internal/modules/pm/gates/defaults.go:12-19`, rules at `:20-37`).

**A decisive, previously uncited piece of evidence: the mechanism already exists in schema v1.**
`schema/doctypes.yaml:54-55` defines a `spec` status family:
`[draft, in-review, approved, building, shipped, shelved]` -- byte-identical to SS6.1's five
values, plus a sixth (`shelved`) the draft's SS6.1 prose never mentions. Both `engineering-spec`
and `test-plan` -- the exact documents SS6.1's chain names -- are typed against this family today
(`schema/doctypes.yaml:70-71`: `engineering-spec: { status: spec, ... }`,
`test-plan: { status: spec, ... }`). This is very likely what SS6.1's "(unchanged)" parenthetical
refers to: a document frontmatter `status` field, already shipped and validated in schema v1,
that is orthogonal to any PM item's `phase`. Neither the draft's SS11 ("Consumed v1 mechanisms")
nor the #126 simulation deficiency report cites `schema/doctypes.yaml`'s status family as the
resolving mechanism -- SS11 lists ADR 0011/0012/0013/0017 but not this schema-file fact
(`docs/development/specs/element-model-v2-draft.md:357-387` mentions `doctypes` only in the ADR 0012 type-count
context at :371/:373; no `status family` hit -- it never cites the status-family mechanism as
the phase-axis resolver). And the family is not inert contract data: the librarian actively
enforces it (`librarian/internal/core/schema/doctypes.go:138-193`, `StatusAllowed` +
`ValidateFrontmatter` refuse a spec doc whose `status` falls outside the family). This is new
grounding this plan is contributing, not a restatement of the issue's own text.

**Secondary gap found in the same sweep:** the draft's five-value SS6.1 chain omits the sixth
`status.spec` value `shelved` (`schema/doctypes.yaml:55`) -- worth a one-line note in the
amendment so the two stay in sync, though not a blocker (no acceptance criterion in the staged
issue body requires it; flagged here as a minor completeness item a builder can fold in cheaply).

## Recommendation on the central question (argued from source; decision left OPEN)

**Question:** is `draft -> in-review -> approved -> building -> shipped` (a) the spec document's
own `status` axis (frontmatter, orthogonal to `items.phase`), or (b) a replacement PM phase
vocabulary?

**Recommended default: (a), the document's own `status` axis — orthogonal to `items.phase`.**
Argued from four independent source signals, no single one alone dispositive:

1. **The mechanism already ships.** `schema/doctypes.yaml:54-55`'s `status.spec` enum is
   byte-identical (plus `shelved`) to SS6.1's chain, and the two documents SS6.1 names
   (`engineering-spec`, `test-plan`) already carry it (`schema/doctypes.yaml:70-71`). SS6.1's
   own word "(unchanged)" (`docs/development/specs/element-model-v2-draft.md:198`) reads naturally as "this
   frontmatter field, already shipped, is not being redesigned here" — not as a claim about the
   PM item machine, which SS6.1 never mentions.
2. **The PM machine is stated as architecturally closed to this kind of extension.** It is
   "data-in-code," and swapping or extending its four-phase vocabulary is explicitly "a code
   change, not a config change -- deliberate" (`docs/development/specs/pm-system-v1-spec.md:500-501`). Reading (b)
   would require the draft to be silently proposing a rigid-machine redesign; nothing in SS6,
   SS12, or SS14 (out-of-scope) says that, and SS14 explicitly forbids shipping v2 storage/code
   changes at all (`docs/development/specs/element-model-v2-draft.md:456-459`). Interpretation (b) is the more
   disruptive reading and has zero supporting text; (a) requires no new mechanism.
3. **The document is internally consistent with itself on the adjacent case.** SS5.3/SS11 already
   establish the pattern "an entity/document carries its own judgment-status axis, orthogonal to
   PM `state`" for `claim` (`docs/development/specs/element-model-v2-draft.md:148-150`, `:375-381`), modelled on
   ADR 0013. Reading SS6.1's lifecycle the same way is the parsimonious, self-consistent choice
   -- the draft simply forgot to say so explicitly, which is exactly what V2-D1 flags.
4. **The simulation that raised the deficiency already reads it as (a).** Scenario 5 step 9's own
   verdict row characterizes the edge as "a *spec-document status* edge (SS6.1's
   `draft->in-review->approved->building->shipped`), **not** an edge in the shipped PM item
   phase-machine" (`_meta/research/model-simulations/scenario-5-v2-software.md:49`). The
   deficiency is that the draft never SAYS this, not that the answer is unknown — the simulation
   crew's own working interpretation, arrived at independently, already lands on (a).

**Why this stays an OPEN owner decision despite the recommendation:** none of the four signals is
a ruling. The draft's author never confirmed reading (a) over (b); ADR 0018/0009 did not settle
this specific question; and the SS12 disposition table marks S8 "Folded in" without addressing the
ambiguity (`docs/development/specs/element-model-v2-draft.md:418`) — so silence should not be mistaken for an
existing ruling. See Open question 1 below.

## Deliverables

Two disjoint units, both edits to the same single file
(`docs/development/specs/element-model-v2-draft.md`) — they serialize on that file (see Parallelism below) but are
independently reviewable slices.

**A -- Axis statement (the orthogonality note).**
State explicitly, at or near SS6.1 (beside the "(unchanged)" parenthetical,
`docs/development/specs/element-model-v2-draft.md:198`) and cross-referenced from SS6.4, that
`draft -> in-review -> approved -> building -> shipped` is the spec document's own frontmatter
`status` axis (`schema/doctypes.yaml:54-55`'s `spec` family), orthogonal to the PM item's `phase`
-- mirroring the SS5.3/SS11 `claim`-status wording exactly (same "orthogonal to any workflow
state/phase" phrasing). Fold in the `shelved` sixth value found in this plan's residual sweep so
the two enumerations match.
- *Acceptance:* grep `docs/development/specs/element-model-v2-draft.md` for a sentence naming the axis choice
  explicitly (issue-body criterion 1); grep for an orthogonality clause referencing `items.phase`
  or `phase` near the SS6.1/SS6.4 text (criterion 2); the SS6.1 chain and
  `schema/doctypes.yaml:54-55` list the same status values (criterion 5's "no contradiction" plus
  the shelved-value fix).

**B -- Gate-rule naming for SS6.4's `verified-by` edge.**
Because `items.phase` and document `status` are orthogonal per Deliverable A, the natural binding
point for the verification gate is the PM item's own advance edge for whatever item type carries
the `engineering-spec` (or its future v2 `items.type`) -- e.g. `review->terminal`, mirroring the
shipped `decision review->terminal` pattern (`librarian/internal/modules/pm/gates/defaults.go
:23-27`). Recommended concrete rule to NAME in the doc (not ship to `defaults.go` -- see gate
hygiene below):
```
engineering-spec:
  "review->terminal":
    documents:
      - type: test-run
        status: pass
        pointer: item
```
This cannot be shipped as a real `defaults.go` rule today: `test-run` is a net-new v2 doctype with
no `schema/doctypes.yaml` entry or status family yet (build deferred per
`docs/development/specs/element-model-v2-draft.md:433-448`), `engineering-spec` is not yet a live PM `items.type`
(ADR 0012 extends the vocabulary only "when the two-track lands,"
`docs/development/specs/element-model-v2-draft.md:368-373`), and the owner has not ruled the wholesale default gate
set (`librarian/internal/modules/pm/gates/defaults.go:12-19`). So the amendment **names the rule
as a proposal** in the `defaults.go` YAML shape, explicitly marked "proposed -- not yet authored
into `defaults.go`; blocked on the SS2 schema build (test-run doctype) and the owner's wholesale
gate-set ruling." This satisfies the issue body's acceptance criterion 3 (which accepts a named
rule OR a blocked-on-ruling marker) by doing both at once: naming the concrete shape while marking
it explicitly not-yet-shippable.
- *Acceptance:* grep `docs/development/specs/element-model-v2-draft.md` for the named rule (item type + phase edge +
  document type/status/pointer triple) or the blocked marker (criterion 3); confirm the doc does
  not claim the rule is live in `defaults.go` today (it is not -- verified against
  `librarian/internal/modules/pm/gates/defaults.go:20-37`, which carries only `decision` and
  `task` rules).

## Gate & contract hygiene

Change surface: **`docs/development/specs/element-model-v2-draft.md` only** -- a single `docs/`-scoped prose file.
Cross-checked against the gate menu in `_meta/plans/_config.md:21-40`.

| Gate | Fires? | Why |
| ---- | ------ | --- |
| Repo checks (`make check`: neutrality + self-test, kit drift, scaffold frontmatter, core purity, actionlint) | No | No `plugin/`, `librarian/`, `kits/`, or scaffold-instrument file touched |
| Unit tests (`make test`: plugin `bun test` + librarian `go test ./...`) | No | No `plugin/` or `librarian/` source changed; Deliverable B's rule is named in prose only, not compiled into `defaults.go` |
| Librarian integration (`make verify`) | No | Fires on anything under `librarian/`; this change is `docs/`-only |
| Bundle drift guard (`make package` + `git diff --exit-code`) | No | Fires on `plugin/core`, `plugin/mcp`, `schema/`; none touched |
| Identity neutrality (`node scripts/check-neutrality.mjs`) | No (exempt) | Scope is `plugin/` + `librarian/` recursively; `docs/` is exempt (`CLAUDE.md` "the one rule that matters"; also stated in the staged issue-body's own Dependencies section). A `#197`/`#125`/`#130`-style bare ref inside this doc is fine under the exemption -- but must never migrate into a Go comment/test on any follow-up build |
| Version sync (`node scripts/check-version-sync.mjs`) | No | No `VERSION` or shipped manifest change |
| Kits drift (`node scripts/check-kits.mjs`) | No | No `kits/` or `kits.yaml` change |
| CHANGELOG gate (`check-changelog.mjs`) | No, at merge time; advisory only pre-release | Fires as a hard release-tag gate, not a per-PR gate; a docs-only model-track amendment is a defensible `[Unreleased]` entry but not enforced until a tag cut |
| DB migration discipline | No | No PocketBase collection touched |
| Regression-test bar (the #82 bar) | No | No behavior change -- prose only, no code path altered |
| Pre-commit (lefthook) | Yes, trivially | Mirrors CI lanes; a `docs/`-only diff clears it with nothing to run |
| Prompt-drift (`scripts/check-prompt-drift.mjs`) | No | Explicitly scoped to `librarian/templates/librarian-system-prompt.txt` vs `docs/development/specs/pocket-librarian-v1-spec.md` (`scripts/check-prompt-drift.mjs:29`) -- a different spec file entirely; confirmed by reading the script, it has no `element-model-v2-draft.md` reference (a repo-wide grep for the path across `scripts/`, `plugin/`, `librarian/`, `schema/`, `.github/` returned zero hits) |

**Net: this is one of the lowest-friction gate surfaces in the repo** -- only lefthook's mirrored
CI lanes apply, and they pass trivially on a prose-only diff. No follow-up PR is required to keep
a generated artifact in sync.

## Parallelism + landing order

| Unit | Parallelizes? | Why |
| ---- | -------------- | --- |
| A (axis statement) | Serializes with B | Both edit `docs/development/specs/element-model-v2-draft.md`; a single-file change is not a parallel-writer scenario -- one pass, one PR |
| B (gate-rule naming) | Serializes with A | Same file; B's prose also depends on A's orthogonality framing being in place first (B references "per Deliverable A" reasoning) |
| Open question 1 (axis ruling) | Blocks A's exact wording, not its existence | A can land with the recommended default stated as the plan's working answer; if the owner later rules (b) instead, the amendment's specific sentence needs a follow-up edit, but the *presence* of an explicit axis statement (criterion 1) does not change |
| Open question 2 (gate-rule shape) | Does not block B | B's acceptance criterion (name a rule OR a blocked marker) is satisfiable either way; the concrete rule text is this plan's recommendation, not a hard requirement |

**Safe landing order:** a single PR, single commit, editing `docs/development/specs/element-model-v2-draft.md` only
-- Deliverable A first (the axis statement, syntactically anchored at SS6.1), then Deliverable B
immediately after (the gate-rule naming at SS6.4), since B's phrasing leans on A's orthogonality
framing. No epic/parent-issue edits are in scope of this plan (the epic #130 "Close when" body is
out of scope here; closing #197 on GitHub after this PR merges is a separate, later step for
whoever runs issue mode / the reconciliation script). No other file owner is touched, so there is
no cross-file serialization to plan around.

## Open questions / owner decisions

1. **Axis question: is `draft -> in-review -> approved -> building -> shipped` the spec document's
   own `status` axis (orthogonal to `items.phase`), or a replacement PM phase vocabulary?**
   **Default / recommendation: (a), the document's own `status` axis** -- argued above from four
   source signals (the `schema/doctypes.yaml:54-55` mechanism already shipping, the PM machine's
   stated code-only extensibility, the SS5.3/SS11 internal precedent, and the simulation's own
   working reading). Confirm-or-override before Deliverable A lands its exact wording.
2. **Gate-rule shape: should the amendment name the concrete candidate rule (`engineering-spec
   review->terminal` requiring a passing `test-run` document), or fall back to a pure
   blocked-on-owner-ruling marker with no concrete shape at all?**
   **Default / recommendation: name the concrete shape, explicitly marked not-yet-shippable** --
   this gives a future builder a starting point without pretending the rule is live, and both
   satisfy issue-body acceptance criterion 3. If the owner would rather the model document stay
   fully silent on the exact rule shape (deferring entirely to the `defaults.go` wholesale
   ruling), Deliverable B narrows to just the blocked marker -- a strict subset of the
   recommended text, so no rework if the owner picks this instead.
3. **Timing: should this amendment be a standalone PR that only touches
   `docs/development/specs/element-model-v2-draft.md`'s SS6.1/SS6.4, or should it be folded into a larger doc pass
   if one is already planned for the model before the `#87` 1.0.0 maturity call?**
   **Default / recommendation: standalone.** #197 is explicitly the one buildable-now item
   (`_meta/HANDOFF.md:54-56`); nothing else on the schema-v2 track is currently active, and a
   narrow single-purpose PR keeps the change trivially reviewable against the five criteria in
   `_meta/plans/phase-machine-reconciliation/issue-body.md`.

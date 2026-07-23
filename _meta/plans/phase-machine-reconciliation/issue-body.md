> **Tracking:** #197, the one buildable-now item on the schema-v2 track (label `gate:v2-final`; closing it clears epic #130's last blocker). Provenance: the #126 model simulations, v2 half — Scenario 5 (v2 software) step 9, rolled up as V2-D1; the model itself landed as #125. Reconcile the software-spec phase-machine with the shipped PM item phase-machine and name (or explicitly defer) the `building → shipped` gate rule. Model-track only: a `docs/`-scoped amendment to the draft, no storage or code.

## Problem

`docs/element-model-v2-draft.md` §6.1 states the software lifecycle as
`draft → in-review → approved → building → shipped` (`docs/element-model-v2-draft.md:198`), and §6.4
says a `test-run` / `verification` element gates `building → shipped` via `verified-by`
(`docs/element-model-v2-draft.md:224-229`, disposition row S8 at `:418`).

But the shipped PM engine — the only thing that actually **binds gate documents** — runs a
*different* phase vocabulary: `queue → work → review → terminal`
(`docs/pm-system-v1-spec.md:398`, machine at `:471-501`; verified live in the #126 v1 half,
Scenario 2). Gates bind on `items.type` at named phase edges, and only two rules ship today —
`task work→review` and `decision review→terminal` — self-documented as a
"KNOWN UNAUTHORED DESIGN GAP" (`librarian/internal/modules/pm/gates/defaults.go:12-37`).

The draft never reconciles the two lifecycles, and names no gate rule realizing §6.4's
`verified-by` gating. A builder cannot implement the verification gate from the model as written,
because the model does not say which axis `draft → … → shipped` is:

- **(a)** the spec **document's own `status` axis** (frontmatter), orthogonal to the PM item's
  `phase` — the reading the "(unchanged)" parenthetical at `docs/element-model-v2-draft.md:198`
  implies, and the reading the simulation itself reached (it calls `building → shipped` "a
  *spec-document status* edge", `_meta/research/model-simulations/scenario-5-v2-software.md:49`); or
- **(b)** a **replacement PM phase vocabulary** — which would mean redefining the rigid,
  code-owned state machine (`docs/pm-system-v1-spec.md:500-501`).

Notably the draft is scrupulous about exactly this axis-separation elsewhere: §5.3 and §11 spell out
that a `claim`'s status (`supported` / `contradicted` / `open`) is "a human-judgment axis
**orthogonal to any workflow `state`**" (`docs/element-model-v2-draft.md:148-150`, `:375-381`),
modelled on the shipped findings-disposition pattern (ADR 0013). The same explicitness is simply
missing for the spec phase-machine.

## Deliverables

A docs-only amendment to `docs/element-model-v2-draft.md` (model-track; no storage, no code). Two
edits:

1. **Axis statement.** State explicitly which axis `draft → in-review → approved → building →
   shipped` is — the spec document's own `status` axis orthogonal to `items.phase`, or a
   replacement PM phase vocabulary — with a one-line orthogonality note mirroring the §5.3/§11
   `claim`-status precedent (`docs/element-model-v2-draft.md:148-150`). Land it at §6.1 (beside the
   "(unchanged)" parenthetical) and/or §6.4.
2. **Gate-rule naming.** Name which PM gate rule realizes §6.4's `test-run —verified-by→` gating —
   which `items.type`, which phase edge, which required document (type + status + pointer), in the
   shape of the shipped `defaults.go` rules — **or** record it as blocked on the pending owner
   ruling for the default gate set, referencing the `defaults.go` "KNOWN UNAUTHORED DESIGN GAP"
   (`librarian/internal/modules/pm/gates/defaults.go:12-19`).

The build plan (`_meta/plans/phase-machine-reconciliation/plan.md`) carries the residual analysis,
the recommended default on axis + gate, and the landing order.

## Acceptance criteria

Independently verifiable by someone who did not write the edit:

1. `docs/element-model-v2-draft.md` states explicitly, in prose, **which axis**
   `draft → … → shipped` is — either "the spec document's own `status` axis, orthogonal to
   `items.phase`" or "a replacement PM phase vocabulary". A reader can point to the sentence; the
   ambiguity the simulation flagged is gone.
2. That statement carries an **orthogonality note mirroring §5.3/§11** — i.e. it names the
   relationship to the PM item's `phase`/`state` the way the `claim`-status note names its
   relationship to workflow `state` (`docs/element-model-v2-draft.md:148-150`).
3. §6.4's `building → shipped` gate is realized by **either** a **named gate rule** (a concrete
   `items.type` + phase edge + required-document triple in the `defaults.go` shape) **or** an
   **explicit blocked-on-owner-ruling marker** that cites the `defaults.go` "KNOWN UNAUTHORED DESIGN
   GAP" — not left silent. Grep for the marker or the rule finds one of the two.
4. The amendment does **not** contradict the shipped PM machine: it does not assert
   `draft/in-review/approved/building/shipped` are `items.phase` values without also stating that
   redefining the rigid machine is a code change (`docs/pm-system-v1-spec.md:500-501`).
5. `docs/element-model-v2-draft.md` frontmatter stays `status: draft` — the flip to final is an
   owner call, not part of this amendment (see Out of scope).

## Dependencies & gates

Change surface: **`docs/element-model-v2-draft.md` only** (a `docs/`-scoped prose edit).

- **Identity neutrality** (`node scripts/check-neutrality.mjs`) — scope is `plugin/` + `librarian/`
  recursively; `docs/` is **exempt** (`CLAUDE.md`, the neutrality rule). A `#197`-style ref in this
  doc is fine. **Does not fire against the edit** — but a bare `#N` must never migrate from `docs/`
  into a Go comment/test on any follow-up.
- **Repo checks / unit tests / librarian integration** (`make check`, `make test`, `make verify`) —
  **do not fire**: no `plugin/`, `librarian/`, `schema/`, `kits/`, `VERSION`, or manifest change.
  No new gate rule ships to `defaults.go` under this issue (that is deferred / owner-ruled), so no
  Go test bar attaches here.
- **Prompt-drift** (`scripts/check-prompt-drift.mjs`) — **does not fire**: it guards only
  `librarian/templates/librarian-system-prompt.txt` against `docs/pocket-librarian-v1-spec.md`
  (`scripts/check-prompt-drift.mjs:29`), not this draft.
- **Bundle / version-sync / kits / DB-migration** gates — **do not fire** (no generated artifact,
  manifest, kit, or PocketBase collection touched).
- **Pre-commit (lefthook)** — mirrors CI; a `docs/`-only diff clears it.

## Out of scope

- **Shipping any v2 storage or code** — no v2 collection, migration, gate-engine change, or store
  edit. ADR 0009 forbids v2 storage ahead of the finalized model
  (`docs/decisions/0009-platform-frame.md:29-49`), restated in the draft's own §14
  (`docs/element-model-v2-draft.md:456-459`). This is a model-track amendment only.
- **Flipping the model to final.** The doc stays `status: draft`. The `#126` simulations bar is
  passed (v2 half, wave-v4), so the remaining flip-to-final is folded into the owner's `#87`
  (1.0.0 maturity) call — not this issue.
- **Building the software / research planes** — live collections, patrol classification, the
  `change`/`bug`/`test-run` entities' storage — deferred per the draft §13 build slice
  (`docs/element-model-v2-draft.md:433-448`).
- **Authoring the full default gate set.** If Deliverable 2 lands as a blocked-on-owner-ruling
  marker, ruling and shipping the software gate rule is the owner's separate `defaults.go` call.

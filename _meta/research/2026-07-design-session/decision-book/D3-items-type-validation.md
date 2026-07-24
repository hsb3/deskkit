---
type: analysis
status: draft
created: 2026-07-20
updated: 2026-07-20
tags: [design-session, decision-book, data-model, items-type, validation, D3]
synopsis: Decision brief D3 — whether CreateItem should validate items.type against the
  schema-v1 doctype vocabulary at birth (hard reject, soft warn) or whether an unmatched
  type's resulting ungated advance should be ruled deliberate and merely documented.
---

_Decision book brief D3 (`../README.md` index, §3 C5). Informs the design session; does not
itself rule. Evidence resolves to the Phase-0 dossiers beside `../README.md`; dossier claims
are hypotheses until re-derived where a ruling binds._

**Layer:** model · **Depends on:** none · **Seeds:** exec-desk agenda §3.5 · prep-doc §3 C5

> **Platform-stream interaction (2026-07-20 reboot — see `D0-platform-frame.md`):** platform
> R3 redesigns the type vocabulary wholesale (three-plane element model, beefed spine —
> `../platform/spec-element-model.md`), and its plan calls the shipped schema "a first draft,
> not binding." Under D0.c's two-track hypothesis, D3 rules the validation **mechanism**
> (check-at-birth vs deliberate ungated advance) against shipped v1; the **vocabulary** the
> check reads becomes the v2 element list when R3 lands. Don't freeze the v1 type list as
> part of this ruling.

# D3 — `items.type` validation

## The question

Should `CreateItem` validate `items.type` against the shared schema-v1 doctype vocabulary
before persisting a new PM item, and if so how strictly — or should the current behavior
(an unrecognized type simply matches no gate rule and advances ungated) be ruled deliberate,
documented behavior instead of a defect?

Verbatim exec-desk seed: "`items.type` validation — enforce the doctype vocabulary at item
birth or rule that unknown types deliberately advance ungated."

## Why now

Post-#112, this is untouched: `items.type` is a plain `TextField` with zero vocabulary check
at `CreateItem`, and the gate engine keys its `(type, edge)` lookup directly off that
unvalidated string — so a typo'd type doesn't fail loudly, it just silently matches zero gate
requirements and advances as if no gate had ever been configured for it. This is not a
hypothetical failure mode; it is the direct, evidenced consequence of how `Effective` looks up
gate config (Evidence #3 below), and it is the exact behavior `document-model-gaps.md` re-ran
against PR #112 and found still live (Evidence #5).

It is also cheap relative to the other model-layer decisions in this book: the vocabulary
check itself (`schema.Vocab().KnownType`) already exists and already validates a different
write path in the same collection family — `desk_config.rules`' per-type gate config is
checked against the identical vocabulary at config-save time (Evidence #2). The open
question is not "does a checker exist" but "should `CreateItem` call it, and how hard." The
prep doc's own hypothesis (§4) names "Validate `items.type` at birth" as the current best
guess — that hypothesis is Option A below, not a ruling.

## Evidence

- `data-model.md § 4. Typed vs stringly — cross-reference-shaped fields` — `items.type` is a
  bare `TextField` (no `Values`); validated at write? "**No** — `CreateItem` sets it directly
  from caller input with zero vocabulary check; a typo'd type advances ungated"
  (`librarian/internal/modules/pm/collections/collections.go:56`;
  `librarian/internal/modules/pm/engine/engine.go:146`).
- `data-model.md § 6. Shared schema contract (schema/)` — `desk_config.rules`' gate-rule
  item-type strings ARE already validated against the same vocabulary at config-write time:
  "`gates.ParseRules` validates `schema_version==1`, every gate's item-type against
  `schema.Vocab().KnownType`..." (`librarian/internal/modules/pm/gates/gates.go:57-123`;
  hooks at `librarian/internal/modules/pm/module.go:97-108,160-173`). The checker exists and
  is already load-bearing elsewhere in the PM lane — only `CreateItem` skips it.
- `workflows.md § 2.2 Gated transitions — the §4.1 sequence (machine → blocked → claim →
  gates → write)` — step 5: "The gate engine evaluates whatever the desk's config binds to
  `(item.type, edge key)` — forward edges gate by default; demote/reopen gate only when the
  config explicitly names them" (`gates.Effective`,
  `librarian/internal/modules/pm/gates/gates.go:133-150`, called at `engine.go:345-352`). This
  is the mechanism that turns a typo into silent ungated advance: an unrecognized type matches
  no config entry, so `Effective` returns zero requirements and the edge passes trivially.
- `workflows.md § 2.7 The importer` — item creation always routes through one path: "Every
  mutation goes through `engine.CreateItem`/`engine.Link` — the importer is a thin driver,
  owning no transition/gate/cascade logic itself" (`importer.go:18-19`). A birth-time check
  added to `CreateItem` is inherited by the deterministic-rebuild importer path (and its
  manifest-driven seed data) for free — one write path to change, none to keep in sync.
- `document-model-gaps.md § Gap F — items.type unconstrained (unfiled)` — direct
  re-verification against PR #112: "**Verdict F — UNADDRESSED (as expected).** `items.type` is
  still unvalidated at birth; unknown types advance ungated. Maps to agenda item 5 (validate at
  birth vs rule ungated-advance deliberate)." (`engine.go:130-132` — only `Title` is
  non-empty-checked; `:146` — `rec.Set("type", in.Type)`, no vocabulary check; `:352` — the
  gate lookup keyed off the unvalidated string).
- `data-model.md § 3. Identity & keying` — PM item keying is "Auto-generated PocketBase id, or
  a caller-pinned `ID` for the deterministic import path (rebuild reproducibility)"
  (`librarian/internal/modules/pm/engine/engine.go:111-116,141-143`) — unlike the librarian's
  `files` collection, PM items are not rebuilt from a disk walk; a live store's rows are the
  only copy of their `type` value, which bears on migration reality below.

## Options

Hypotheses are marked; none of these is a ruling.

### Option A — Validate at birth, hard reject (the prep-doc hypothesis)

`CreateItem` calls `schema.Vocab()` and refuses the write (no record created) when
`!vocab.KnownType(in.Type)`, mirroring the check `gates.ParseRules` already runs for
`desk_config.rules` (Evidence #2). No PocketBase schema migration is required — the field
stays a bare `TextField`; the guard is an application-level check, the same pattern already
proven for gate-rule validation.

- Closes the ungated-advance hole completely, and for every surface at once — MCP, CLI, TUI,
  and the importer all funnel through the one `engine.CreateItem` (Evidence #4): one place to
  add the check, none to fall out of sync.
- Symmetric with how a document's own frontmatter `type:` is already validated against the
  identical vocabulary — `items.type` stops being the one write path the vocabulary misses.
- Cheap: reuses an existing, already-tested vocabulary function; no new collection or field.
- Closed-vocabulary tension: `schema/doctypes.yaml`'s ~28 types are the *shared* contract both
  lanes read, not desk-personalizable. A desk wanting an ad hoc PM-only item type has no
  self-service path under this option; adding one requires a `schema/` change shipped to both
  lanes.
- Import/rebuild risk: any existing manifest (including the D8 adoption-seed path, Evidence
  #4) carrying a typo'd or deliberately out-of-vocabulary type now hard-fails import instead
  of silently creating an ungated item — needs auditing before it lands (unverified whether any
  current fixture is affected — see Uncertainties).
- Not retroactive: a live store's already-created, mistyped items are untouched (items aren't
  rebuilt from disk the way `files` are, Evidence #3). Fixing them needs a manual data
  correction — a data-fix script, not a schema migration ("forward migrations only" wall).

### Option B — Validate at birth, soft (warn, don't block)

`CreateItem` still calls `KnownType`, but on a miss it proceeds with the write and surfaces
the mismatch rather than refusing it.

- No import/rebuild breakage — existing manifests and ad hoc desk-local types keep working.
- Makes the drift visible to whoever calls `CreateItem` without hard-blocking a legitimate
  desk-specific type the shared vocabulary hasn't caught up to.
- Does **not** close the ungated-advance hole by itself: the gate engine keys `Effective`
  purely off the type string (Evidence #3) — a warned-but-accepted mismatched type still
  matches zero gate rules and still advances ungated. To actually answer the seed's "enforce…
  at item birth" framing, a soft warning needs pairing with some other enforcement — on its
  own this option answers "make it visible," not "enforce."
- No evidenced mechanism for surfacing a "soft" warning on `CreateItem`. The closest
  dossier-cited analog, `patrol_findings`, operates only over the librarian's `files`
  collection via a fixed R1–R6 rule set (`workflows.md § 1.2 Patrol — flag rule violations
  (dry-run)`), not PM `items` — a warning channel here is new workflow-layer surface area, not
  reuse of anything evidenced in this pass.

### Option C — Rule ungated-advance deliberate (document only, no code change)

Leave `CreateItem` as-is. `docs/development/specs/pm-system-v1-spec.md` and/or the gate-rule documentation state
explicitly that a type with no bound gate config — whether because it is genuinely
untracked-by-design or because it is a typo — advances ungated by design, extending the
general "no bound rule ⇒ no gate" behavior that already governs unconfigured demote/reopen
edges (Evidence #3: "an edge with no bound gate passes trivially" per `gates.go` — cited via
`Evaluate`'s own doc comment at the workflows.md-sourced call chain).

- Zero build cost, zero migration or import risk, zero closed-vocabulary tension — desks keep
  full freedom over what string a PM item's `type` holds.
- Consistent with the gate philosophy already in force elsewhere (unconfigured edges already
  pass ungated by design) — extends "unconfigured ⇒ ungated" from edges to types rather than
  introducing a new enforcement class.
- Does not distinguish a deliberate untyped/ad hoc item from a typo — exactly the risk the
  exec-desk seed calls out. Neither an agent nor a human can tell, after the fact, which one
  happened. Ruling this deliberate bets that an occasional silent typo costs less than a
  closed vocabulary plus import-breakage risk (Option A) or new warning plumbing (Option B).
- Leaves Gap F (Evidence #5) formally closed only by documentation, not by narrowing the
  actual failure mode the dossier flagged.

A workflow-layer companion — a patrol-style post-hoc detector scanning PM `items` for
out-of-vocabulary types — is not itself an option here: patrol today reaches only the
librarian's `files` collection (`workflows.md § 1.2`), so extending it to `items` is new
workflow-layer scope this brief surfaces but does not decide (see Out of scope).

## Decision criteria

Any ruling on this brief must satisfy, citing the prep-doc constraint walls (`../README.md`
§7) that bind here:

- **Forward migrations only; never edit an applied migration.** If a chosen mechanism needs a
  schema change (e.g. a future move to a `SelectField`), it must be additive, and existing
  live-store rows must be left as-is by the migration itself — correction of already-wrong
  rows is a separate, explicitly-called-out data-fix, not smuggled into the migration.
- **No unattainable retroactive fix.** An option that implies live desks must correct existing
  typo'd items must say so explicitly and describe it as a manual, optional data-fix, not a
  precondition for adopting the ruling.
- **`schema/` is the shared, product-neutral contract both lanes read.** An option that ties
  `items.type`'s vocabulary to `schema/doctypes.yaml` inherits that contract's shared change
  cost; an option inventing a second, PM-only vocabulary needs to justify why the existing
  shared one doesn't suffice (the vocabulary engine's own scope comment already states a
  preference against inventing "a second, divergent notion of validity" — `data-model.md § 6`).
- **Gates read disk directly, never sweep-dependent — unaffected here.** `items.type`
  validation is a DB-side, at-birth concern; note the boundary explicitly so it isn't
  conflated with the unrelated gate-verdict disk-read invariant.
- **No ruling smuggled in as fact.** If the chosen option changes importer/rebuild behavior
  (Option A) or spec text (Option C), the ruling must say so explicitly, not leave it implicit
  in a code or docs diff.

## Blast radius

**Option A (hard reject):** Code — `librarian/internal/modules/pm/engine/engine.go`
(`CreateItem`, ~lines 129-178) adds a `schema.Vocab()` + `KnownType` check before
`rec.Set("type", in.Type)`. No PocketBase migration required as an application-level check
(`TextField` unchanged) — same pattern already proven for `desk_config.rules`; a later move to
a DB-level `SelectField` enum instead IS a forward migration and needs an explicit data-fix
plan for any live-store row already outside the new enum. Spec — `docs/development/specs/pm-system-v1-spec.md`
needs the new invariant stated explicitly (not confirmed either way by the dossiers whether
it's currently silent — see Uncertainties). Test lane — importer/rebuild-reproducibility tests
and any manifest-driven seed data (`workflows.md § 2.7`) need auditing for out-of-vocabulary
`type` fixtures before this lands. `schema/doctypes.yaml` — no change required by this option
itself, but it becomes the de facto ceiling on what `items.type` may ever be; a desk wanting a
new PM-only type routes through a `schema/` change shipped to both lanes.

**Option B (soft warn):** Same `CreateItem` code point, plus a new (currently unevidenced)
warning-surfacing mechanism — scope TBD; likely the `transitions`/audit write path or a
tool-response field, with no existing dossier-cited construct today. Spec needs to document the
warning contract precisely so every surface (MCP/CLI/TUI) treats it identically — a symmetry
concern that echoes D5/C1.

**Option C (document only):** Docs only — `docs/development/specs/pm-system-v1-spec.md` gains an explicit
sentence on the "unconfigured type ⇒ ungated" behavior. No code, no migration, no test impact.
Possible companion (out of scope for this brief): a workflow-layer detector, new patrol-style
scope over PM `items` — patrol today only reaches the librarian's `files` collection
(`workflows.md § 1.2`), not `items`.

## Out of scope / interactions

- This brief rules only the birth-time (`CreateItem`) enforcement question. Whether
  `items.type` is mutable post-creation (an `update_item`/`UpdateItem` path exists at
  `queries.go:544-548`, per `workflows.md § 2.5`, but confirmed only for `status_label`
  writes) is out of scope here — see Uncertainties for the update-time gap this could leave.
- D2 (typed cross-references) is adjacent but distinct: D2 concerns reference-shaped fields
  (`graduated_to`, gate pointers) gaining a kind+target+qualifier primitive. `items.type` is a
  plain classification string, not a reference — no shared mechanism is assumed between them.
- D8 (backlog) owns the `files.entity_type` naming collision (`data-model.md § 5.2`) — a
  different field in the same "doctype string, weak typing" family, but a separate collision
  this brief does not resolve.
- Whether `schema/doctypes.yaml` itself should grow a versioning story (so a desk could pin an
  older/newer vocabulary) is the prep doc's "migration reality" tough spot (§5) at the
  schema-contract level generally, not specific to this decision — this brief assumes the
  current single, unversioned `schema/doctypes.yaml` as given.
- A workflow-layer, patrol-style post-hoc detector for mistyped PM items (Option C's
  companion note) is surfaced here but not scoped or decided by this brief.

## Uncertainties

- Whether `items.type` is mutable after creation (via the `update_item` tool /
  `queries.go:544-548` `UpdateItem`) is not confirmed by either dossier — `workflows.md § 2.5`
  cites that function only for `status_label` writes. If `type` is also settable there, a
  birth-only validation (any option here) leaves a same-shaped gap at update time. Needs a
  targeted read of `queries.go` before a ruling finalizes on "birth-time is sufficient."
- Whether `docs/development/specs/pm-system-v1-spec.md` currently documents (or is silent on) the "type with no
  bound gate rule advances ungated" behavior is not dossier-confirmed either way —
  `workflows.md`'s own Gaps section flags that `gates/defaults.go`'s default gate bindings
  were not read, and neither dossier quotes the spec's `CreateItem`-adjacent section directly.
- Whether any live desk's importer manifest or seed data (the D8 adoption-seed path,
  `workflows.md § 2.7`) currently carries an out-of-vocabulary `type` string is unverified —
  this determines whether Option A's import-breakage consequence is theoretical or immediate
  for existing tooling.
- `core/schema/doctypes.go`'s `KnownType`/`Vocab()` internals were read directly by
  `data-model.md` (cited at `doctypes.go:151-158` for `ValidateFrontmatter`, and the file's own
  scope comment at `:7-11`), but `workflows.md`'s own Gaps section separately flags that its
  author did not read that file. The two dossiers do not conflict, but the vocabulary engine's
  full behavior (e.g. how `Vocab()`'s process-wide caching would interact with any future
  desk-local extension) rests on `data-model.md`'s read, not `workflows.md`'s.

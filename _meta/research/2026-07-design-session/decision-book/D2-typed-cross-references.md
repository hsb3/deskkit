---
type: analysis
status: draft
created: 2026-07-20
updated: 2026-07-20
tags: [design-session, decision-book, data-model, cross-references, schema, 1.0.0]
synopsis: D2 decision brief — whether one first-class reference primitive in schema/ (kind +
  target + optional desk/repo qualifier) should replace today's untyped graduated_to text field,
  with gate pointers and future refs migrating onto it; the desk-relative-qualification tension
  with identity-neutrality; options, criteria, blast radius. Informs the session; does not rule.
---

_Decision-book brief D2 (layer: **model**; depends on **D1** — pointer grammar). Frames the
decision; does not rule. Evidence resolves to the Phase-0 dossiers beside `../README.md` §6;
**dossier claims are hypotheses** re-derived in the session where a ruling binds. Rulings land as
ADRs in `docs/decisions/`._

Status: draft (2026-07-20)

> **Platform-stream interaction (2026-07-20 reboot — see `D0-platform-frame.md`):** the
> migrated platform stream independently converged on *typed entity + typed relation* as the
> unifying primitive (Grist-style references — `../platform/system-cohesion-and-datamodel.md`
> Thread 2, `../platform/plan.md` R2). Under D0.c's two-track hypothesis, D2's reference
> primitive becomes the typed-relation substrate the schema-v2 element model consumes — a
> reason to favor the specify-the-contract-now option class. The qualifier-storage axis is
> regime-dependent (D0.b): state the answer under files-are-truth AND what flips at the gate.

# D2 — Typed cross-references

## The question

**Should there be one first-class reference primitive in the shared `schema/` contract — a
typed value of `kind + target + optional desk/repo qualifier` — that today's `graduated_to`,
the gate document pointers, and future references all become instances of, or does each
reference-shaped field stay a separate, locally-typed text column?**

Exec-desk agenda seed (verbatim): _"Cross-reference typing ruling — `graduated_to` (and future
refs): typed kind + repo-qualified identity vs today's bare substring. Gates #92's proper fix."_

Post-#112 state: extraction is fixed (#92 — an explicit marker now populates `graduated_to`,
never a bare substring), but the model gap under it is untouched — `graduated_to` is still one
untyped text column, a bare `42` is a spec-pinned valid value, and no reference carries repo
qualification. The candidate ruling shape (a hypothesis, prep §4, not a ruling): one reference
primitive in `schema/`, consumed by both lanes, with `graduated_to` and gate pointers migrated
onto it.

## Why now

- **#92's "proper fix" is blocked on this ruling, by the seed's own words.** #112 closed the
  extraction *symptom* (marker-only), but left `graduated_to` a single untyped column whose
  spec-pinned grammar knowingly accepts a bare, repo-unqualified `42` — so the same stored value
  means different issues on different desks while looking resolved. That is the one place a
  *shipped, spec-pinned* value is wrong at the model layer.
- **Every future reference re-invents the same untyped-string pattern.** `graduated_to`,
  `items.pointer`, and the gate `DocRequirement.Pointer` are three independent text notions of
  "a pointer to somewhere," each with its own ad-hoc grammar and its own (or no) validation.
  Without a shared primitive, D1's pointer grammar and any later cross-doc link each land as a
  fourth and fifth one-off.
- **Left unruled**, the model layer ships 1.0.0 with reference identity that is desk-ambiguous
  by construction and un-unifiable later without a second contract break.

## Evidence

- `document-model-gaps.md § Gap B — untyped, repo-unqualified cross-refs (#92)` — extraction is
  now explicit-marker-only, but the field is still one untyped `text` column with no repo
  qualification; the "gap under the bug" is untouched (`collections/0001_files.go:23`;
  `sweep.go:117,164`; spec pin `pocket-librarian-v1-spec.md:477,680`).
- `document-model-gaps.md § Gap B — untyped, repo-unqualified cross-refs (#92)` — the
  spec-pinned regex accepts a bare number: `#?\d+` makes `#` optional, so `graduated to: 42`
  captures `"42"` as opaque pointer text, pinned verbatim by the spec (`sweep.go:359`;
  `pocket-librarian-v1-spec.md:911,999`; test `sweep_test.go:276`). Sanity-checked in source: the
  regex `(?im)^\s*graduated to:?\s+(wb#\d+|#?\d+|https?://\S+)` and its intent comment sit at
  `librarian/internal/modules/librarian/tools/sweep.go:356-359`.
- `document-model-gaps.md § Gap B — untyped, repo-unqualified cross-refs (#92)` — nothing on the
  write path consults `repos.shorthand.issue_default` or any qualifier, so `42` / `#42` / `wb#42`
  are stored as-is and mean different issues on different desks (grep: no `issue_default` /
  `repos.shorthand` reference on the graduated-to path in `sweep.go`).
- `data-model.md § 4. Typed vs stringly — cross-reference-shaped fields` — the full inventory of
  reference-shaped fields and their (non-)validation: `files.graduated_to` opaque pointer text,
  validated at write? **No** (`sweep.go:349-376`); `items.pointer` desk-relative path + advisory
  anchor, validated only at gate time (`pm/collections/collections.go:67`; `module.go:104-175`);
  gate `DocRequirement.Pointer` a closed 2-value grammar `""`/`"item"`/`"note:<key>"`, validated
  at `ParseRules` (`gates.go:25,109-123`).
- `data-model.md § 1.1 \`files\` (0001, altered by 0012)` — `graduated_to` is a bare `TextField`
  riding the implicit 5000-char cap; `files` carries no `RelationField` of its own, it is a
  relation target only (`0001_files.go`; relation from `patrol_findings.file`,
  `0002_patrol_findings.go:18`).
- `data-model.md § 6. Shared schema contract (\`schema/\`)` — `schema/` is the shared contract
  with two drift guards: `doctypes.yaml` must be byte-identical to its vendored Go copy
  (`doctypes_test.go:10-29`), and `profile.schema.yaml` is copied into `plugin/claude-plugin/`
  with CI failing on any diff (`plugin/package.json:17`; `.github/workflows/ci.yml:85`). The
  profile schema already declares `repos.shorthand.issue_default` — the per-desk qualifier source
  (`schema/profile.schema.yaml:1-201`). Sanity-checked in source: `shorthand:` /
  `issue_default:` at `schema/profile.schema.yaml:67-72`.
- `data-model.md § 6. Shared schema contract (\`schema/\`)` — the TS lane does **not** enforce
  `formats:`/`enums:` today, and the Go engine's own comment says the value-format checks are
  "NOT enforced by this engine yet" — so a reference-kind enum would be new enforcement in both
  lanes, not a tightening of an existing check (`doctypes.go:7-11`; zero-match grep under
  `plugin/core/*.ts`).
- `data-model.md § 5.2 The \`entity_type\` naming collision (files vs schema)` — a live warning
  for any new typed vocabulary: `files.entity_type` (a store column holding a doctype) and
  `schema`'s `entity_type` enum (`[person, company, technology, product, service]`) already
  collide by name with disjoint value spaces (`0001_files.go:16`; `sweep.go:248`;
  `doctypes.yaml:29`). A reference `kind` enum must not repeat this.
- `data-model.md § 3. Identity & keying` — `graduated_to` is derived at sweep time from the desk
  file's explicit marker (the file text is source-of-truth; the column is a re-derivable
  projection), whereas `items.pointer` is caller-supplied into the PM store (`sweep.go:49-52`
  path-keyed derivation vs PM item keying `engine.go:111-116,141-143`). The two migrate very
  differently (see Blast radius).
- `document-model-gaps.md § Gap A — no sub-file addressing (#102)` — the pointer grammar (D1's
  remit) is implemented but specified nowhere; a `§ heading` anchor is tolerated-and-ignored, a
  `#` anchor fails closed with a hint (`module.go:122,136-138,209-214`). Establishes the D1↔D2
  boundary: same latent "typed, qualified reference" shape.
- `document-model-gaps.md § Gaps & uncertainties` — the dossier's own read: cross-reference
  typing is "the most consequential open modeling question," and it generalizes — "gate pointers
  (Gap A) and `graduated_to` are the same latent 'typed, qualified reference' primitive, so ruling
  B well subsumes part of A."
- `document-model-gaps.md § Analysis §3 agenda → current status` — agenda item 3 maps here: a
  typed reference primitive (kind + target + repo/desk qualifier), with `graduated_to` still one
  untyped column and the spec-pinned regex accepting a bare unqualified `42`.
- `workflows.md § 1.1 Sweep — reindex the tree` — `graduated_to` is extracted **only** from an
  explicit marker inside one `RunInTransaction`, writing the `files` collection only; a re-sweep
  re-derives it (`sweep.go:26-102,368-376`) — so the store side of a graduated_to representation
  change is cheap to migrate.

## Options

Options 2 and 3 both carry the prep §4 hypothesis ("one typed reference primitive in `schema/`");
they differ only on whether the qualifier is persisted. No option is stated as a ruling.

**Option 1 — Field-local hygiene, no shared primitive.** Keep `graduated_to`, `items.pointer`,
and the gate pointer as three separate text notions; only tighten each field's own write-time
validation — e.g. require the graduation marker to carry `#N`/`wb#N`/URL and reject the bare
number. No `schema/` contract change, no cross-lane primitive, no stored qualifier.
- _Consequences:_ cheapest; closes the "bare `42` silently accepted" surprise at the field where
  it lives; a spec delta only (the regex is spec-pinned). But reference identity stays
  desk-ambiguous (a stored `#42` still means different issues on different desks), the three
  notions stay divergent, and #92's "proper fix" — a *typed, qualified* reference — does not land.
  Future refs keep re-inventing the pattern.

**Option 2 — One shared reference primitive; qualifier resolved at read time, never persisted
(prep §4 hypothesis, identity-neutral variant).** Define a first-class reference type in `schema/`:
`{kind, target}` where `kind ∈ {issue, doc-pointer, url, …}` and `target` is the raw ref exactly
as authored. `files.graduated_to`, `items.pointer`, and gate `DocRequirement.Pointer` become
instances. The desk/repo qualifier is **not** stored — it is applied at display/link time by the
consuming surface from the profile's `repos.shorthand.issue_default`.
- _Consequences:_ gives #92 a structural fix and unifies the notion across both lanes; stores
  nothing desk-specific, so identity-neutrality holds by construction and the store stays trivially
  rebuildable (raw `target` lives in the desk file/manifest). But a stored reference is never
  self-describing — resolving `#42` always requires the desk's profile, so a store snapshot carried
  to another desk resolves differently (arguably *correct* — references are desk-relative — but it
  means no portable resolved link). It is a `schema/` v2: both lanes and both drift guards move
  together.

**Option 3 — One shared reference primitive; qualifier resolved-and-frozen at write time
(prep §4 hypothesis, self-describing variant).** Same primitive as Option 2, but the value also
carries a `qualifier` slot populated *at write time* from the author's profile shorthand — a
snapshot of the resolution.
- _Consequences:_ references become self-describing and portable (a `wb#42` that already knows its
  repo resolves the same everywhere), which is what "repo-qualified identity" most literally means.
  But it stores a profile-derived value in the user's own store — legal under identity-neutrality
  (that wall governs *shipped* code, not a user's desk data) **only if** no default qualifier is
  ever baked into shipped code and an unqualified author simply stores none. The store is still
  rebuildable *if* the qualifier is also written into the desk file marker (so a re-sweep can
  re-derive it); if it lives only in the store column, a rebuild-from-disk loses it — a real
  constraint-wall check. Heavier migration: `items.pointer` is a genuine forward migration.

**Option 4 — Specify now, migrate later (contract-first, deferred rollout).** Land the reference
type in the `schema/` contract and add write-time validation guards immediately (reject the bare
number, reject malformed refs), but defer migrating the live `graduated_to`/`items.pointer` columns
onto the structured shape until a concrete consumer (the D8 seeder, a cross-desk link surface)
demands it.
- _Consequences:_ closes the shipped-wrong-value hole and fixes the contract shape without a live
  forward migration this cycle; keeps 1.0.0's blast radius small. But it books a known follow-up
  break (the eventual column migration) and risks the contract type sitting unexercised — a
  vocabulary that no code reads is the exact failure `document-model-gaps.md`'s unwired-enum
  findings warn about.

## Decision criteria

Any ruling must satisfy the constraint walls (`../README.md` §7) that bind here:

- **Identity-neutrality (CI-enforced, `check-neutrality.mjs`).** No default desk/repo qualifier
  may be baked into shipped `plugin/` or `librarian/` code. Qualification is desk-relative by
  construction — it can *only* come from the profile's `repos.shorthand.issue_default`
  (`data-model.md § 6`). A ruling that stores a qualifier (Option 3) must source it from the
  profile at write time and store nothing for an unqualified author.
- **Store rebuilds from disk; document bodies are never persisted.** The desk file / import
  manifest is source-of-truth. Any information a reference carries (target, and a qualifier if
  stored) must survive a store-rebuild-from-disk — i.e. it must live in the desk file marker, not
  only in a store column (`data-model.md § 3`; `workflows.md § 1.1`). This is the sharp test that
  separates Option 3's two sub-variants.
- **Forward migrations only on shipped collections; never edit an applied migration.** Live stores
  exist (the owner's own desks). A structured column change to `files`/`items` is a new forward
  migration. Note the asymmetry: `graduated_to` is sweep-derived, so its store representation is
  re-derivable by re-sweep (cheap); `items.pointer` is caller-supplied, so its migration is real.
- **`schema/` is the shared contract — a v2 moves both lanes at once.** Whether the contract gets
  *versioned* is part of this decision's blast radius. Both drift guards must stay green: the
  byte-identical `doctypes.yaml` vendored copy and the `profile.schema.yaml → claude-plugin` copy
  (`data-model.md § 6`).
- **Harness-purity of `plugin/core` (`check-core-purity.mjs`).** If the TS lane consumes the
  primitive (it currently enforces no enums/formats at all), the added validation must import no
  harness/runtime API.
- **No naming collision.** A reference `kind` enum must not repeat the `entity_type` double-naming
  (`data-model.md § 5.2`).

## Blast radius

**Option 1 (field-local):** no collection or `schema/` change. Code: `sweep.go` graduation regex +
its comment. Spec: `docs/development/specs/pocket-librarian-v1-spec.md` §5.1 graduated_to precedence and §5.2 R5 —
both pin the regex verbatim, so tightening it is a spec delta. Docs: the harvest-loop skill's
graduation guidance.

**Options 2/3 (shared primitive):**
- _`schema/` contract:_ a new reference type in the shared schema (likely a new `schema/`
  artifact or an addition to `doctypes.yaml`) + its byte-identical vendored Go copy under
  `librarian/internal/core/schema/` (guarded by `doctypes_test.go`) + the
  `profile.schema.yaml → plugin/claude-plugin/schema/` copy if the profile side gains fields.
  This is the "does the contract get versioned" call.
- _Collections/migrations:_ `files` — forward migration for `graduated_to`'s new shape (store side
  cheap: a re-sweep re-derives it). `items` — forward migration for `pointer` (real: caller-supplied,
  not re-derivable). Gate `DocRequirement.Pointer` grammar lives in `desk_config.rules` YAML,
  validated at `ParseRules` — a change there ripples to every desk's gate config.
- _Code:_ `sweep.go` (`graduationMarker`/regex → structured parse); `module.go`
  `Verdict`/`sectionFilePart` (D1 territory — the pointer's `target` grammar); pm engine pointer
  resolution (`engine.go` `pointerResolver`); `plugin/core` TS if the primitive is validated there
  (new enforcement).
- _Spec:_ `docs/development/specs/pocket-librarian-v1-spec.md` (graduated_to §5.1, R5 §5.2), `docs/development/specs/pm-system-v1-spec.md`
  (pointer definition §3.1 / R2.3, `:396`/`:744`).
- _Docs/skills/ADR:_ a new ADR in `docs/decisions/`; harvest-loop + conventions-standard skills.
- _Option 3 adds:_ the qualifier must also be written into the desk file marker (else it fails the
  rebuild-from-disk wall), so the marker grammar itself changes — a larger spec + authoring-surface
  delta than Option 2.

**Option 4 (specify-now):** the `schema/` contract + validation-guard slice of Options 2/3 lands;
the `files`/`items` column migrations are deferred and booked as a follow-up. Smaller live radius,
one known future break.

## Out of scope / interactions

- **D1 — pointer grammar (the depends-on).** D1 owns *what a pointer may be*, what each form
  satisfies gate-wise, fail-closed rules, and sub-file addressing (the advisory-heading tolerance).
  D2 owns *the reference primitive itself* and *whether* `graduated_to`, `items.pointer`, and future
  refs are instances of it. If the primitive is adopted (Options 2/3/4), D1's grammar becomes the
  `target` grammar of the `doc-pointer` kind — so the two rulings compose, they do not overlap.
  `document-model-gaps.md § Gaps & uncertainties` argues ruling the primitive "subsumes part of"
  the pointer grammar; the index's dependency ordering nonetheless lists D2 *after* D1 (flagged in
  Uncertainties).
- **D3 — `items.type` validation.** A different vocabulary (doctypes at item birth), not a
  reference. Adjacent (both are "validate a stringly field against a shared vocabulary") but its
  own brief.
- **D8 — document identity / rename history.** The identity of *a document across renames* is a
  distinct question from the identity of *a reference to a document*; D8 owns the former. A ruling
  here does not decide rename history.
- **Spec ↔ reality (D7)** and the `entity_type` column collision (D8 backlog) are noted as
  cautions above, not decided here.

## Uncertainties

- **Seed-internal ordering conflict (carried, not resolved).** The decision-book index lists D2 as
  *depends-on D1* (D1 first), yet the same index's Known-interactions note says "the pointer grammar
  is plausibly an instance of the reference primitive … rule the primitive first" (`../README.md`
  lines 26, 39-43). These disagree on which to rule first. This brief respects the packet's stated
  `depends-on: D1` and frames the primitive as the more fundamental shape; the session should
  resolve the ordering explicitly.
- **Whether `items.pointer` should be an instance at all.** It is a desk-relative file path with an
  advisory section anchor (D1's grammar), semantically closer to a "doc-pointer" than an "issue
  ref." Folding it into the same primitive as `graduated_to` (an issue/URL ref) may over-unify two
  genuinely different reference kinds; the alternative is a primitive whose `kind` enum is broad
  enough to hold both without conflating them. Not decided here.
- **Gate `DocRequirement.Pointer` as an instance.** Its grammar is a closed 2-value set
  (`""`/`"item"`/`"note:<key>"`) that addresses *store-internal* loci, not external issues — it may
  not belong under the same primitive as an outward reference. Flagged, not judged.
- **Whether the contract gets versioned.** The brief lays out the blast radius of a `schema/` v2 but
  does not assume one is required — a purely additive contract change might avoid a version bump.
  This is a session call.
- **Dead pre-#112 extraction helpers.** `document-model-gaps.md § Gaps & uncertainties` flags
  `issueRefFind`/`hashRefRe`/`findBareHashRef` (`sweep.go:378-427`) as possibly-dead code still
  exercised by a test; whether a reference-typing change should remove them is unverified.
- **TS-lane consumption is unbuilt.** The plugin lane enforces no enums/formats today
  (`data-model.md § 6`), so "consumed by both lanes" is currently aspirational on the TS side; the
  cost of adding harness-pure reference validation to `plugin/core` was not scoped in the dossiers.

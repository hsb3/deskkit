---
type: analysis
status: draft
created: 2026-07-20
updated: 2026-07-20
tags: [design-session, decision-book, data-model, pointer-grammar, gates, 1.0.0]
synopsis: Decision brief D1 (layer model, depends on nothing; D2 depends on it) — what a PM
  `items.pointer` may be, what each form satisfies gate-wise, the fail-closed rules, and the
  hard center — sub-file addressing vs rename tolerance (the `§` anchor is advisory by ruling).
  Options only; no ruling.
---

_Phase-1 decision brief (`README.md`). Informs the session; does not rule. Evidence resolves to
Phase-0 dossier sections beside this book; dossier claims are hypotheses until re-derived where a
ruling binds. Options are labeled; none is a ruling._

Status: draft (2026-07-20)

# D1 — Pointer grammar

## The question

**What may a PM `items.pointer` (and a gate rule's document pointer) be, what does each form
satisfy gate-wise, and what are the fail-closed rules — including whether the `§`-anchored
sub-file locus is ever really addressable without breaking rename tolerance?**

Two distinct grammars hide under the word "pointer" and the ruling must separate them:

1. **The `items.pointer` field grammar** — the string an item carries (a desk-relative file path,
   today optionally suffixed with an advisory `§ heading`). This is what a gate resolves to disk.
2. **The gate `DocRequirement.Pointer` selector** — the *closed* grammar in the gate-rules YAML
   that names *which* document a gate consults: `""`/`"item"` (the item's own pointer field) or
   `"note:<key>"` (a keyed note's body). This is a selector over the item, not a locus on disk.

The agenda seed names three deliverables inside this one decision: (a) what a pointer may be and
what each form satisfies gate-wise, with fail-closed rules; (b) **#102's *proper* fix** — #112
shipped a symptom fix (`§` tolerated-and-ignored), leaving open whether real sub-file addressing
is the proper fix; (c) **the seeder design** — the adoption/import seeder is the concrete use case
that would emit section refs and force the sub-file question.

## Why now

The pointer grammar is **implemented but specified nowhere**. #112 (the 0.8.0 bug floor) made the
verdict path *tolerate* a `§ heading` anchor by resolving only the file part and gave `#`-anchored
pointers an actionable failure — closing the #102 symptom (a seeded section-ref item no longer
fails its first gated transition). But the spec's only definition of a pointer is still a broad
one-liner, so nothing states the delimiter, the advisory-heading rule, the `#`/`://` rejections, or
what a sub-file locus would ever satisfy. Every gate in both lanes reads this field; a grammar the
spec doesn't pin is a grammar every future surface, seeder, and skill will guess at differently.
The hard center is real, not cosmetic: the `§` anchor is advisory **by ruling** — gates ignore it
precisely so heading renames can't break a transition — so "just resolve the section" would
reintroduce exactly the fragility the current design avoids.

## Evidence

- `document-model-gaps.md § Gap A — no sub-file addressing (#102)` — #112 made the verdict path
  tolerate a `§ heading` anchor by resolving only the FILE part (`sectionFilePart`,
  `module.go:209-214`; called by `Verdict` at `module.go:122`); the heading is advisory and
  **never checked** (`module.go:116-121`, proven by `TestVerdict_ToleratesSectionAnchorSuffix`,
  `module_test.go:198-259` — an absent heading still passes).
- `document-model-gaps.md § Gap A — no sub-file addressing (#102)` — a `file.md#heading` pointer
  fails closed with a named cause pointing at `§` (`module.go:136-138`, proven by
  `TestVerdict_HashAnchorNotStripped`, `module_test.go:261-290`); a `://` pointer still fails
  closed (`module.go:123-124`).
- `document-model-gaps.md § Gap A — no sub-file addressing (#102)` — the grammar is implemented,
  not specified: the spec's only pointer definition is the broad one-liner
  (`pm-system-v1-spec.md:396`, `| pointer | text | doc path / issue URL / other locus (R2.3) |`)
  plus the "MAY carry a pointer to a GitHub issue URL" prose (`:744`); nothing states that `§` is
  the delimiter, that the heading is advisory, that `#` is rejected, or what a sub-file locus
  satisfies gate-wise.
- `document-model-gaps.md § Gap A — no sub-file addressing (#102)` — real sub-file addressing (the
  seeder's need to emit section refs) requires **stable section identity without reintroducing
  rename-fragility**; today the anchor is tolerated-and-ignored, not resolved to a section, so the
  latent-failure prediction is deferred, not resolved.
- `data-model.md § 4. Typed vs stringly — cross-reference-shaped fields` — `items.pointer` is a
  bare `TextField` holding a desk-relative file path with an optional advisory `§ heading` suffix;
  a `#heading` suffix is not stripped and fails resolution; it is **not validated at PM write
  time** — only at gate-evaluation time by the librarian's `Verdict()` (declared
  `collections.go:67`; resolution `module.go:104-175`, `sectionFilePart` `module.go:203-214`).
- `data-model.md § 4. Typed vs stringly — cross-reference-shaped fields` — the gate
  `DocRequirement.Pointer` is a **separate, closed** grammar: `""`/`"item"` (the item's own
  pointer) or `"note:<key>"` (a note's body), validated at `ParseRules` time by
  `validateDocRequirement` (`gates.go:25,109-123`).
- `data-model.md § 4. Typed vs stringly — cross-reference-shaped fields` — `desk_config.rules`
  (the gate-rules YAML) is validated by `gates.ParseRules`, which checks every doc requirement's
  type/status/**pointer grammar**, and is hook-bound so an invalid config is never saved
  (`gates.go:57-123`; hooks `pm/module.go:97-108,160-173`).
- `data-model.md § 2.4 ` + "`notes` (`collections.go:135-147`)" — the `note:<key>` gate-pointer
  grammar addresses the `notes.key` field; `notes.body` is the content the gate reads and is
  content-bearing on the implicit 5000-char cap (`collections.go:135-147`).
- `workflows.md § 2.2 Gated transitions — the §4.1 sequence (machine → blocked → claim → gates → write)`
  — `Evaluate` resolves each requirement's pointer via `pointerResolver` (`"item"` → the item's
  `pointer` field; `"note:<key>"` → a keyed note's body, `engine.go:419-443`, called at
  `engine.go:345-352`); with **no validator registered every documented gate fails closed**
  (`gates.go:183-185`).
- `workflows.md § 2.2 Gated transitions — the §4.1 sequence (machine → blocked → claim → gates → write)`
  — the verdict reads **DISK, not the store**: it resolves the pointer against `DESK_ROOT` with a
  containment check (`resolveDeskPath`, `module.go:220-231`), does a direct `os.ReadFile`
  (`module.go:130`), and validates frontmatter type/status — never a `files`-collection lookup, so
  a verdict never depends on sweep freshness (`module.go:96-103`).
- `workflows.md § 2.7 The importer` — the importer is the adoption/seed path (programmatic only;
  test lane §10.8) and routes every mutation through `engine.CreateItem`/`engine.Link` — the one
  write path — so any section-ref pointers a seeder emits land in `items.pointer` through it
  (`importer.go:87-174`).
- `data-model.md § 6. Shared schema contract (`schema/`)` — issue qualification lives in the
  **per-desk profile** (`repos.shorthand.issue_default`), not in shipped code — bearing directly on
  whether an issue-URL/issue-ref pointer form could ever be desk-neutral (`schema/profile.schema.yaml:1-201`).

## Options

Each is a hypothesis, not a ruling. A, B, C differ on the **hard center** (does a `§` locus ever
really resolve?); they are largely orthogonal to the **fail-closed rules** and the
**external/issue-locus** sub-decision, which every option must settle (see Decision criteria).

### Option A — Ratify the current behavior; `§` stays advisory-only (specify, don't build)

Write the spec to match what #112 shipped: `items.pointer` = a desk-relative file path, optionally
suffixed with an advisory `§ Heading`; the gate resolves and requires **only the file part**;
`#`-anchored and `://` pointers fail closed with their existing hints; the **whole file** is the
only thing a gate ever asserts. Sub-file addressing is explicitly **not supported** — `§` is a
human/reader wayfinding hint the gate never checks.

- **Consequences.** Zero migration; maximal rename tolerance (heading renames can never break a
  transition, by design). The seeder may emit `§` section refs, but they are decoration — a gate
  can never assert "section X of file Y is in status Z"; it only ever asserts file-level
  existence + frontmatter type/status. #102's "proper fix" is declared to be *this* (the symptom
  fix is the design), and the latent sub-file need is closed as out-of-scope until a use case
  forces it (feeds D8's backlog framing).

### Option B — Stable section identity via explicit anchor ids (real sub-file addressing)

Sections carry an **explicit, stable id** that survives heading text renames — e.g. a
frontmatter-declared section map, or an inline anchor token authored into the document. The gate
resolves `file § #<id>` to a real section on disk and can evaluate section-scoped predicates
(existence, and — if defined — a section-level status).

- **Consequences.** Real sub-file addressing the seeder can rely on. But it trades rename-fragility
  for **id-maintenance-fragility**: a deleted or typo'd id breaks the transition (fail-closed), and
  authors/agents must maintain ids. Resolution must still read disk directly (constraint wall), and
  now must parse section structure — heavier than existence+frontmatter. Heading *text* renames no
  longer break it; a missing *id* does. Needs an authoring convention in skills/kits and a spec for
  the id grammar. No `items.pointer` column change if the id rides inside the string
  (`file § #id`); the store stays rebuildable because the id lives in the desk tree, not the store.

### Option C — Tolerant section resolution with graceful file-level fallback

The gate attempts to resolve the `§ Heading` by **tolerant matching** (e.g. normalized heading
text) and, if no section matches, **falls back to the file-level verdict** rather than failing.

- **Consequences.** Sub-file addressing that never regresses rename tolerance — a renamed heading
  silently degrades to a file-level check. But the section-level assertion is **soft**: you cannot
  rely on it being enforced, because a miss downgrades rather than refuses. This is a poor fit for a
  gate whose whole value is a hard fail-closed guarantee (a "checked" section ref that quietly stops
  being checked is arguably worse than Option A's honest "never checked"). Cheapest authoring
  burden; weakest guarantee.

### Option D — Fold the field grammar into D2's typed-reference primitive

Instead of a bare string, `items.pointer` becomes an **instance of D2's typed reference**
(kind ∈ {`desk-doc`, `issue`, `external`}; target = path/URL; qualifier = section-id / repo). The
grammar's meaning is then read off the kind: a `desk-doc` ref with a section-id qualifier is the
Option-B checkable case; an `issue`/`external` ref satisfies a *documented* external-locus gate
class or fails closed by policy. This realizes the prep-doc §4 hypothesis (one primitive consumed
by both lanes; `graduated_to`, gate pointers, and `pointer` all migrate onto it).

- **Consequences.** Unifies three cross-ref-shaped fields under one contract and makes the
  external/issue-locus question a *typed* decision rather than a string-parse. But it **depends on
  D2 ruling first**, is the largest change (a `schema/` contract move touches **both lanes at
  once**), and turns `items.pointer` from a `TextField` into a structured value — a forward
  migration on the live `items` collection whose existing free-text pointers were never
  write-validated. The sub-file mechanism (A/B/C) is still an open sub-choice *inside* the
  `desk-doc` kind, so D doesn't eliminate the hard center — it relocates it under a typed wrapper.

## Decision criteria

Any ruling must satisfy the constraint walls (`README.md` §7) that bind here:

- **Gates read disk directly, never sweep-dependent.** Any section-resolution (B/C) must resolve
  against `DESK_ROOT` via a direct read, never a `files`-collection lookup — the verdict must not
  regain a store dependency it deliberately lacks (evidence: `workflows.md` §2.2; `module.go:96-103`).
- **Store rebuildable from disk.** Section identity (B) must live in the **desk tree** (frontmatter
  / inline token) or be derivable from disk — never store-only — so a rebuild reproduces the same
  resolution. This forbids "section ids assigned by the store."
- **Fail-closed, and say why.** The grammar must define the exact refusal set and keep every
  refusal actionable: `://` and scheme-bearing strings, `#heading` (with the `§` hint), a
  path escaping `DESK_ROOT` (containment), an unreadable/missing file, and — the framework rule —
  **no validator registered ⇒ every documented gate fails closed**. A gate must never *pass* on an
  unresolvable or ambiguous pointer (evidence: `document-model-gaps.md` §Gap A; `workflows.md` §2.2).
- **Identity-neutrality.** If `issue`/`external` pointer forms are admitted, the grammar must bake
  **no default qualifier** into shipped code — a bare `#N` means different issues on different
  desks, and qualification lives in the per-desk profile (`repos.shorthand.issue_default`). The
  neutral options are: an issue/external pointer is *item metadata that satisfies no gate* (gate
  always needs a desk-doc), or a documented external-locus gate class whose qualifier is
  profile-supplied (evidence: `data-model.md` §6).
- **Forward migrations only; live stores exist.** If `items.pointer` changes shape (D), it is a
  forward migration on a shipped collection holding **never-validated** free-text values — the
  migration must define how existing pointers map, and the `schema/` contract move must version
  both lanes together (this is the migration reality, not a greenfield grammar).
- **Two grammars stay separate but consistent.** The ruling must state both the `items.pointer`
  field grammar and the gate `DocRequirement.Pointer` selector grammar, and keep the selector's
  closed set (`item`/`note:<key>`) coherent with whatever the field admits.

## Blast radius

Concrete artifacts, by option class:

- **Spec (`docs/pm-system-v1-spec.md`).** All options add the missing grammar section: pointer
  forms, per-form gate satisfaction, fail-closed set, the `§`-advisory rule, and the
  `DocRequirement.Pointer` selector grammar. The current broad one-liner (`:396`) and the
  "GitHub issue URL" prose (`:744`) are **corrected in place** to match the ruling (they currently
  advertise issue-URL pointers the gate rejects — a spec/reality tension the ruling must resolve).
- **Code — librarian (`module.go`).** A/C touch `Verdict`/`sectionFilePart`; B adds a
  section-resolution reader (parse section structure, still disk-direct); D reshapes `Verdict`'s
  input to a typed reference. All keep the `resolveDeskPath` containment guard.
- **Code — PM (`gates.go`, `engine.go`).** If the selector grammar widens (unlikely under A;
  possible under D), `validateDocRequirement`/`ParseRules` (`gates.go`) and `pointerResolver`
  (`engine.go`) change; `desk_config.rules` YAML validation is hook-bound, so any new selector value
  must be accepted there or existing configs break.
- **`schema/` contract.** Untouched under A/B/C (pointer stays a lane-local field). Under D the
  reference primitive lives in `schema/` → a v2 contract move affecting **both** lanes at once.
- **Collections / migrations / live stores.** A/B/C: `items.pointer` stays a `TextField`, **no
  migration**. D: a forward migration on `items` turning free-text (never-validated) pointers into
  a structured value — the owner's own desks hold existing values that must map or be tolerated.
- **Skills / personas / kits.** B (and D's `desk-doc` kind) needs an authoring convention (how a
  human/agent/seeder writes a stable section id) surfaced in the pm-operator persona, the
  conventions skill, and any SOP kit that emits pointers. The **seeder/importer** design (agenda
  deliverable c) is downstream of whichever section grammar is chosen — it must emit only forms the
  gate can satisfy.
- **Docs.** The neutrality allowlist and any doc citing the pointer grammar; `README.md` C2 row.

## Out of scope / interactions

- **D2 owns the typed-reference primitive** (kind + target + qualifier). This brief covers the
  pointer *grammar* and *gate semantics*; **Option D is the only place D1 proposes making the
  grammar an instance of D2's primitive**, and it is explicitly conditioned on D2 ruling first
  (`README.md` known-interaction: "the pointer grammar is plausibly an instance of the reference
  primitive; rule the primitive first"). Options A–C keep the grammar lane-local and leave the
  primitive question entirely to D2.
- **Reference identity across desks** (a bare `#N` / `42` meaning different issues per desk, and
  `graduated_to`'s repo-unqualified value) is **D2 / tough-spot 2**, not decided here. D1 only
  states the *neutrality boundary* for issue/external pointer forms; how a reference is *qualified*
  is D2's.
- **Document identity across renames** (frontmatter id vs store-side inference) is **D8** (backlog).
  It overlaps B's "stable section identity" — a whole-document id and a stable section id may share a
  mechanism — but D1 decides only section-level addressing for gate pointers; whole-document
  rename history is D8's.
- **`items.type` validation** (that a gate's `(type, edge)` binding matches a real doctype) is
  **D3**; D1 assumes the type side is settled elsewhere and rules only the *pointer* side of a
  `DocRequirement`.

## Uncertainties

- **Whether the seeder actually emits `§` section refs today, or only would.** Gap A frames the
  seeder's section-ref need as *latent* ("latent until the seeder emits section refs"); the
  importer dossier (`workflows.md` §2.7) confirms the importer is the seed path and routes through
  the single write path, but does **not** show it emitting section-anchored pointers. Whether any
  shipped seeder path emits `§` refs — vs. this being a designed-for-future need — is unverified
  here and should be checked before ruling B/C worth the cost.
- **Exact spec §3.1 / R2.3 pointer text.** The dossiers cite `pm-system-v1-spec.md:396` and `:744`;
  the fuller §3.1 definition and the R2.3 rule the `:396` line references were not read in this
  brief. The correction-in-place blast-radius item assumes those are the only two spec loci
  advertising issue-URL pointers — unconfirmed.
- **"D8 seeder" terminology collision.** The agenda seed's "D8 seeder" and this book's brief **D8**
  (backlog: identity/hygiene) are cited from different numbering systems (the seeder term traces to
  the librarian spec's internal decisions / the adoption seed path, e.g. `restore`'s "decision
  0014"). This brief treats "the seeder" as the importer/adoption path (`workflows.md` §2.7); the
  precise identity of "D8 seeder" as a named artifact was not independently pinned.
- **Section-level status semantics under B.** Whether a *section* can carry its own frontmatter-like
  status (so a gate could assert section status, not just section existence) has no evidence in the
  dossiers — the schema-v1 vocabulary is document-level (`data-model.md` §6). If B is pursued, "what
  does a section-level verdict even assert" is an open sub-question.
- **Existing live-store pointer values.** `items.pointer` was never write-validated
  (`data-model.md` §4), so the owner's desks may hold arbitrary strings. What forms actually exist
  in live stores was not verified (the dossiers verify source, not running stores) and bears on any
  option that narrows the admitted grammar.

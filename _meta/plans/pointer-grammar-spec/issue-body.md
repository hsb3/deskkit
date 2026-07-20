> **Tracking:** #115, ADR 0010 (2026-07-20 design session). Ratify the shipped pointer grammar with a normative spec section — no code change.

## Problem

The `items.pointer` grammar is implemented but specified nowhere. `items.pointer` is a bare
`TextField` (declared `librarian/internal/modules/pm/collections/collections.go:67`, never
write-validated) whose resolution rules live only in code:

- `Verdict` (`librarian/internal/modules/librarian/module.go:96-175`, the `schema.DocumentValidator`
  every PM gate calls) resolves a pointer's FILE part via `sectionFilePart`
  (`module.go:203-214`): a pointer is a desk-relative file path, optionally suffixed
  `§ <heading>`. The heading is a human wayfinding hint and is **never checked**
  (`module.go:116-121` states this; the file part alone must exist and validate).
- A `://`-bearing pointer fails closed (`module.go:123-124`: `"pointer %q is not a desk file; a
  document gate needs a file path"`).
- A `#`-anchored pointer (the markdown convention, e.g. `file.md#heading`) is **not** stripped —
  only `§` delimits a section anchor — and fails closed with an actionable hint naming the
  supported `§` form (`module.go:136-138`).
- This behavior is pinned by tests: `TestVerdict_ToleratesSectionAnchorSuffix`
  (`librarian/internal/modules/librarian/module_test.go:198-258` — an absent heading still
  passes, a genuinely missing file still fails, a URL with a section anchor still fails) and
  `TestVerdict_HashAnchorNotStripped` (`module_test.go:260-292` — a `#`-anchored pointer fails
  closed and the failure names `§`).

Meanwhile the spec's only definition of a pointer is a stale one-liner —
`docs/pm-system-v1-spec.md:396`: `` | `pointer` | text | doc path / issue URL / other locus (R2.3) | ``
— which still advertises an issue-URL pointer form the gate rejects outright (any `://`-bearing
string fails per `module.go:123-124`). Nothing in the spec states the `§` delimiter, the
advisory-suffix rule, or the two fail-closed forms above.

**The true residual, stated precisely:** PR #113 (commit `e5aee59`) already corrected the
*other* spec/reality contradiction ADR 0010 names — `docs/pm-system-v1-spec.md:743-750` (§7,
R6.1) now reads "per ADR 0010 an issue URL is **not** a gate `pointer` (the pointer grammar is a
desk-relative file path; gates fail `://` closed)". That correction is done; verified against
current `docs/pm-system-v1-spec.md`. What is still missing is (a) the stale §3.1 field-table row
at `:396`, which was not touched by #113 and still contradicts the R6.1 sentence three lines'
worth of prose away from it in review order, and (b) a normative section that actually defines
the grammar (delimiter, advisory rule, fail-closed set) rather than referencing it obliquely.
This issue closes both.

Separately, the gate's `DocRequirement.Pointer` **selector** (`librarian/internal/modules/pm/gates/gates.go:25`,
values `""`/`"item"`/`"note:<key>"`, validated by `ParseRules`/`validateDocRequirement`
`gates.go:57-107`/`:109-123`) answers a different question — *which* document a gate reads — and
must stay visibly distinct in the spec from the `items.pointer` field's *where-on-disk* grammar
this issue defines.

Issue and URL references are **not** gate pointers under this grammar — they are a
cross-reference, typed per ADR 0011 (`docs/decisions/0011-typed-reference-contract.md`); this
issue does not define that contract, only states the boundary.

## Deliverables

- A normative pointer-grammar subsection in `docs/pm-system-v1-spec.md`, placed immediately
  after §3.1 (`items` — the universal work item). The spec's existing precedent for out-of-order
  subsection numbering (`### 2.10 D2b — the chassis rename` appears before `### 2.8 The migration
  framework`, both owner-approved insertions) means a new subsection — e.g. "3.1a `pointer`
  grammar (ADR 0010)" — can be added without renumbering anything. (Top-level sections 1-13 are
  listed in the Table of contents; sub-sections like 2.8/2.10/3.1 are not, so no TOC edit is
  needed.) Content required:
  - A pointer is a desk-relative file path.
  - It may carry an advisory `§ <heading>` suffix; the heading is never checked by a gate —
    cite `module.go:116-121`, `:203-214` (`sectionFilePart`).
  - A `#`-anchored pointer and a `://`-scheme-bearing pointer both fail closed, each with an
    actionable hint — cite `module.go:123-124` and `:136-138`.
  - Cite the pinning tests by path: `module_test.go:198-258`
    (`TestVerdict_ToleratesSectionAnchorSuffix`) and `:260-292`
    (`TestVerdict_HashAnchorNotStripped`).
  - Name ADR 0010 as the ruling this section ratifies.
- Correct the stale row at `docs/pm-system-v1-spec.md:396` (currently
  `doc path / issue URL / other locus (R2.3)`) so it no longer advertises an issue-URL pointer
  form, and instead points to the new subsection.
  **Sequencing note:** `_meta/plans/typed-reference-contract/issue-body.md` (ADR 0011) also edits
  this row; whichever lands first leaves the other a one-line forward pointer.
- One sentence distinguishing the gate `DocRequirement.Pointer` selector (`gates.go:25`,
  `""`/`"item"`/`"note:<key>"`) from the `items.pointer` field grammar, so a reader does not
  conflate "which document" with "where the document lives."
- No code change. `Verdict`/`sectionFilePart` (`module.go`) and `DocRequirement`/
  `validateDocRequirement` (`gates.go`) are cited, not modified — their behavior is pinned by the
  tests above and is ratified as-is by ADR 0010.

## Acceptance criteria

- [ ] `docs/pm-system-v1-spec.md` contains a subsection stating: a pointer is a desk-relative
      file path; a `§ heading` suffix is advisory and never checked by a gate; a `#`-anchored or
      `://`-scheme-bearing pointer fails closed.
- [ ] That subsection cites the pinning implementation and tests by path: `module.go:96-175` and
      `:203-214`; `module_test.go:198-258` and `:260-292`; and names ADR 0010.
- [ ] The field-table row at `docs/pm-system-v1-spec.md:396` no longer describes "issue URL" as
      a pointer form, and points readers to the new subsection.
- [ ] The spec text keeps the gate `DocRequirement.Pointer` selector (`gates.go:25`) explicitly
      distinct from the `items.pointer` field grammar — a reader cannot conflate them.
- [ ] `git diff --stat -- librarian/ plugin/` on the closing PR is empty (no code change under
      either shipped lane).
- [ ] `node scripts/check-neutrality.mjs` passes bare (confirms nothing under `plugin/`/
      `librarian/` regressed; `docs/` itself is exempt from the scan).
- [ ] `make check` and `make test` pass bare and are no-ops relative to `main` (the edit is
      docs-only, so both gates should be unaffected).

## Dependencies & gates

- Depends on nothing new: ADR 0010 is Accepted and the shipped behavior it ratifies
  (`module.go`, the two pinning tests) already exists on `main`.
- This is a docs-only change scoped to `docs/pm-system-v1-spec.md`. The following gates do
  **not** fire for this change surface:
  - Librarian integration (`make verify`, `librarian/verify.sh`) — no `librarian/` code change.
  - Bundle drift guard (`make package` + `git diff --exit-code` on `plugin/claude-plugin/`) —
    no `plugin/core`, `plugin/mcp`, or `schema/` change.
  - Version sync (`check-version-sync.mjs`) and kits drift (`check-kits.mjs`) — `VERSION` and
    `kits/` untouched.
  - DB migration discipline — no PocketBase collection touched.
- Gates that DO fire because they always fire: `make check` (neutrality + self-test, kit drift,
  scaffold frontmatter, core purity, actionlint) and `make test` (plugin `bun test` + librarian
  `go test ./...`); pre-commit (lefthook) mirrors the same. All should be no-ops relative to
  `main` for a docs-only diff.
- CHANGELOG: no shipped behavior changes, so `check-changelog.mjs` (which hard-gates only at the
  release tag) is not blocked by this alone; the closing PR should still add a short
  `[Unreleased]` entry recording the spec correction, matching the provenance precedent set by
  PR #113's own design-session record.
- Adjacent, not absorbed: #101 ("Document which PM transitions gate") will want to say, per
  transition, that a gate resolves its required document via the grammar this issue specifies.
  #101's per-transition table (which transitions gate, plus doctype + `updated`-key requirements)
  is downstream of this issue's grammar section but is a separate deliverable, tracked on #101.
- ADR 0011 (typed-reference contract, `docs/decisions/0011-typed-reference-contract.md`) owns
  the issue/URL reference side once ruled; this issue states only the boundary ("not a gate
  pointer"), not the contract itself.

## Out of scope

- Sub-file addressing (stable section identity / anchor ids). ADR 0010 defers this until a use
  case pulls it (`_meta/research/2026-07-design-session/decision-book/D1-pointer-grammar.md`,
  Option A); the `§` suffix stays advisory-only.
- Any behavior change to `Verdict`/`sectionFilePart` (`module.go`) or to the gate
  `DocRequirement.Pointer` selector (`gates.go`) — both are pinned by tests and ratified as-is.
- The typed-reference contract for issue/URL references — that is ADR 0011's scope; this issue
  only states that such references are not gate pointers.
- #101's per-transition gate documentation (which transitions gate, doctype + `updated`-key
  requirements) — adjacent, tracked separately on #101.

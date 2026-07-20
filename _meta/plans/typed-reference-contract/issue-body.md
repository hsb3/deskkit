> **Tracking:** #TBD, ADR 0011 (2026-07-20 design session). Specify the typed cross-reference
> contract — kind + target + optional desk-relative qualifier — in `schema/`, with a validation
> guard in both lanes; no field migrates onto it yet.

## Problem

`schema/` today has exactly two dimensions — `profile.schema.yaml` (the personalization block)
and `doctypes.yaml` (the doc-type vocabulary) — and neither defines a reference-shaped value.
Every reference-shaped field in the store is a bare, unvalidated string, and the one field ADR
0011 names first, `files.graduated_to`, is spec-pinned to stay that way:

- `files.graduated_to` is a plain `TextField` with no `Values`/enum
  (`librarian/internal/modules/librarian/collections/0001_files.go:23`), populated only from an
  explicit graduation marker — never a bare reference merely quoted in prose
  (`graduationMarker`, `librarian/internal/modules/librarian/tools/sweep.go:368-376`). The
  marker's target grammar is one regex with three alternative forms:
  `(?im)^\s*graduated to:?\s+(wb#\d+|#?\d+|https?://\S+)`
  (`sweep.go:359`) — an issue-shaped ref (`wb#N`, `#N`, or a bare number) OR a raw URL, and the
  regex is **pinned verbatim by the spec**
  (`docs/pocket-librarian-v1-spec.md:911`, R5 rule table `:999`) — a bare unqualified number is a
  deliberately valid target, not a bug.
- Nothing on that write path resolves or stores a repo qualifier. A repo-wide grep for
  `issue_default`/`shorthand` under `librarian/` returns zero matches — confirmed in this pass.
  The qualifier source already exists on the profile side —
  `schema/profile.schema.yaml:67-75` declares `repos.shorthand.issue_default` (`owner/repo`
  pattern) as "the repo a bare `#N` resolves against" — but no code reads it. The same stored
  value (`wb#42`) therefore means a different issue on every desk, silently.
- Neither lane enforces a value-format/enum check on any field today, so adding one here is new
  ground, not a tightening of an existing check. The Go engine says so in its own doc comment:
  "the `formats:`/`enums:` value-format checks are NOT enforced by this engine yet"
  (`librarian/internal/core/schema/doctypes.go:7-11`). The TS lane's only schema-driven
  validation is the profile's own JSON-Schema constraints (`ajv`, `plugin/core/schema.ts`); there
  is no reference-kind or doctype-enum check anywhere under `plugin/core/*.ts` — confirmed by a
  zero-match search for the schema's `entity_type` enum in that tree.
- A new vocabulary here must not repeat the repo's live naming collision: `files.entity_type`
  (a store column holding the doc's frontmatter `type`, `0001_files.go:16`,
  `sweep.go:248`) and `schema/doctypes.yaml`'s own unrelated `entity_type` enum
  (`[person, company, technology, product, service]`, `schema/doctypes.yaml:29`) already share a
  name with disjoint value spaces. A reference `kind` enum must live in its own namespace.

**The true residual, stated precisely:** #92/#112 already fixed the *extraction* symptom (a
graduation must be explicitly marked, never inferred from a bare `#N` in prose). The *model*
underneath is untouched — `graduated_to` is still one untyped column with no repo
qualification, and every future reference-shaped field (`items.pointer`, gate
`DocRequirement.Pointer`) reinvents its own ad hoc string grammar with no shared primitive. ADR
0011 rules that a contract now closes this shape; this issue is that contract's build slice.

## Deliverables

- **The contract** — `schema/references.yaml`, a new third dimension of schema v1 (sibling to
  `doctypes.yaml`/`profile.schema.yaml`), defining the reference primitive: a closed `kind` enum
  seeded with `issue` and `url` (the two target-grammar forms `graduated_to`'s marker already
  accepts, `sweep.go:359` — see Out of scope for why the enum stops there), a `target` slot (raw
  string, the ref exactly as authored, no format constraint), and a documented (not stored)
  `qualifier` slot: per ADR 0011, a qualifier resolves at READ time from
  `profile.repos.shorthand.issue_default` and is never part of the persisted `{kind, target}`
  shape — state this rule in the file's own header comment so a future resolver has a contract
  to consume. Update `schema/README.md`'s "What's in it now" section to name this third
  dimension.
- **Go-lane guard** — `librarian/internal/core/schema/references.go`: `//go:embed` a vendored
  copy at `librarian/internal/core/schema/references.yaml` (mirrors the existing
  `doctypes.go`/`doctypes.yaml` embed pattern, `doctypes.go:23-24`), exposing
  `ReferenceVocab() (*ReferenceVocabulary, error)`, `(*ReferenceVocabulary) KnownKind(kind
  string) bool`, and `ValidateReference(kind, target string) error` (rejects an unknown kind and
  an empty/whitespace-only target; takes no qualifier parameter — the persisted shape never
  carries one). Tests in `librarian/internal/core/schema/references_test.go`, including a
  byte-identical drift guard `TestReferencesEmbeddedCopy_MatchesRepoRoot` analogous to
  `TestDoctypesEmbeddedCopy_MatchesRepoRoot` (`doctypes_test.go:10-29`).
- **Go-lane neutrality allowlist** — add an entry to `schema/neutrality-lint.allow` exempting the
  new vendored copy, mirroring the existing `doctypes.yaml` entry
  (`schema/neutrality-lint.allow:30-35`): `SCAN_DIRS` covers `plugin`/`librarian`/`kits`
  (`scripts/check-neutrality.mjs:52`), not `schema/` itself, so the canonical file is unscanned
  but its vendored copy under `librarian/` is — needed if the copy's provenance comments name any
  issue ordinal, matching the doctypes.yaml precedent.
- **TS-lane guard** — `plugin/core/references.ts`, mirroring `schema.ts`'s discovery pattern
  (`plugin/core/schema.ts:33-58`) to load `schema/references.yaml` and expose an equivalent
  `validateReference(kind: string, target: string, vocab: ReferenceVocabulary):
  ValidationResult`. Must import no harness/runtime API — plain Node builtins + `yaml` only,
  matching every existing file in `plugin/core/` (AC2 purity rule,
  `plugin/scripts/check-core-purity.mjs:1-21`). Tests in `plugin/core/references.test.ts`.
- **Bundle copy** — extend `plugin/package.json`'s `package` script (currently `cp
  ../schema/profile.schema.yaml claude-plugin/schema/profile.schema.yaml`, `plugin/package.json:17`)
  to also copy `references.yaml`, so `plugin/claude-plugin/schema/references.yaml` exists for a
  marketplace install (the plugin cache carries only `plugin/claude-plugin/`, per CLAUDE.md).
- **Spec prose** — a short forward-pointer note in each spec, citing ADR 0011, stating plainly
  that no extraction/resolution behavior changes yet:
  - `docs/pocket-librarian-v1-spec.md`: insert after the `graduated_to` precedence paragraph ends
    (`:918`, before the "Line counting" paragraph at `:920`) — a new paragraph, not touching the
    existing "Updated 2026-07-20 (0.8.0)" callout (`:904-905`) or the precedence prose itself.
  - `docs/pm-system-v1-spec.md`: a note near the `pointer` field-table row (`:396`). **Sequencing
    note:** `_meta/plans/pointer-grammar-spec/issue-body.md` (ADR 0010) also edits this same row
    and inserts a new pointer-grammar subsection immediately after §3.1 — whichever issue lands
    first should leave the other a one-line forward pointer rather than restating its content
    (that issue's own body already commits to naming ADR 0011 as the reference-contract owner,
    `_meta/plans/pointer-grammar-spec/issue-body.md:132-133`).
- **CHANGELOG** — an `[Unreleased]` entry recording the new contract + guard (schema-content
  addition, both lanes), matching the provenance convention already used for design-session
  deliverables (`CHANGELOG.md` `[Unreleased]` section, current top entry).

## Acceptance criteria

- [ ] `schema/references.yaml` exists, parses, and declares a `kind` enum containing at least
      `issue` and `url`, a `target` slot, and header prose stating the qualifier is resolved at
      read time from the profile and never persisted.
- [ ] `schema/README.md` documents `references.yaml` as schema v1's third dimension.
- [ ] `go test ./internal/core/schema/...` (run from `librarian/`) passes, including
      `TestReferencesEmbeddedCopy_MatchesRepoRoot` failing if the vendored copy diverges from
      the repo-root file (prove this by editing one copy locally and re-running — it must fail).
- [ ] Named Go tests in `references_test.go` assert: `ValidateReference("issue", "wb#42")` is
      nil; `ValidateReference("not-a-kind", "wb#42")` is a non-nil error; `ValidateReference("issue",
      "")` is a non-nil error.
- [ ] `cd plugin && bun test core/references.test.ts` passes, asserting the equivalent three
      cases against `validateReference`.
- [ ] `cd plugin && bun run check:purity` passes bare with `plugin/core/references.ts` present —
      confirms no harness/runtime import was introduced into `core/`.
- [ ] `cd plugin && bun run package` then `git diff --exit-code -- claude-plugin/` is clean on the
      closing PR (the new `claude-plugin/schema/references.yaml` copy is committed, not only
      locally generated).
- [ ] `node scripts/check-neutrality.mjs` passes bare, including the new vendored copy under
      `librarian/internal/core/schema/` (requires the new `schema/neutrality-lint.allow` entry
      only if that copy's provenance comments carry a bare issue ordinal, matching the
      `doctypes.yaml` precedent; otherwise no entry is needed).
- [ ] `git diff --stat -- librarian/internal/modules` is empty on the closing PR — no change to
      `sweep.go`'s graduation extraction, `module.go`'s pointer resolution, or any PocketBase
      collection file; today's `graduated_to`/`items.pointer` behavior is provably unchanged.
- [ ] `docs/pocket-librarian-v1-spec.md` and `docs/pm-system-v1-spec.md` each carry the
      forward-pointer note described above, citing ADR 0011.
- [ ] `make check` and `make test` pass bare.

## Dependencies & gates

- Depends on nothing new to start work — ADR 0011 is Accepted. Adjacent to
  `_meta/plans/pointer-grammar-spec/issue-body.md` (ADR 0010), which already states the boundary:
  issue/URL references are not gate pointers, they are this contract's concern
  (`docs/decisions/0011-typed-reference-contract.md`) — see the sequencing note above for the
  one place both issues touch the same spec line.
- **Fires — always:** `make check` (neutrality + self-test, kit drift, scaffold frontmatter, core
  purity, actionlint) and `make test` (plugin `bun test` + librarian `go test ./...`); pre-commit
  (lefthook) mirrors the same.
- **Fires — bundle drift guard** (`make package` then `git diff --exit-code -- claude-plugin/`,
  CI at `.github/workflows/ci.yml:81-85`): this issue's `schema/` addition plus the new
  `plugin/core/references.ts` both touch the copied bundle surface (the `plugin/package.json:17`
  `cp` step is extended to include `references.yaml`).
- **Fires — identity neutrality** (`node scripts/check-neutrality.mjs`, `SCAN_DIRS =
  plugin/librarian/kits`, `scripts/check-neutrality.mjs:52`): on `plugin/core/references.ts`, the
  new `plugin/claude-plugin/schema/references.yaml` bundle copy, and the new
  `librarian/internal/core/schema/references.yaml` vendored copy (needs its own
  `schema/neutrality-lint.allow` entry only if that copy's provenance comments carry a bare issue
  ordinal, matching the `doctypes.yaml` precedent — `schema/` itself is unscanned, but a copy
  landing under `librarian/` is).
- **Fires, but WEAK signal — librarian integration** (`make verify`, `librarian/verify.sh`):
  fires under the mechanical "any change under `librarian/`" trigger (this issue adds
  `librarian/internal/core/schema/references.go`). `verify.sh`'s scratch-desk CLI flow (sweep →
  patrol → propose-fix → apply-fix → restore) never calls the new guard — it isn't wired to any
  collection or CLI surface this cycle — so a green `make verify` here proves only that nothing
  else regressed, not that the guard works. The named unit tests above are the real coverage.
- **CHANGELOG** (`check-changelog.mjs` hard-gates only at the release tag): not blocked pre-merge,
  but the closing PR should still add an `[Unreleased]` entry — this is a product-visible
  addition (new schema contract + guard in both lanes), matching the always-add convention.
- **Does NOT fire — DB migration discipline:** no PocketBase collection is touched. This issue is
  explicitly scoped to ship no store migration (ADR 0011: "No immediate store migration... The
  contract + guard land first and are testable alone").
- **Does NOT fire — version sync** (`check-version-sync.mjs`): `VERSION` and the three shipped
  manifests (`plugin.json`, `plugin/package.json`'s version field, `marketplace.json`) are
  untouched; this is a schema-content addition, not a release-version bump.
- **Does NOT fire — kits drift** (`check-kits.mjs`): `kits/`/`kits.yaml` untouched.
- **Does NOT fire — scaffold frontmatter:** no scaffold template touched.

## Out of scope

- **Migrating `files.graduated_to` or `items.pointer` onto the new structured shape.** ADR 0011:
  "`graduated_to`, gate pointers, and future refs become instances of the primitive LATER via the
  v2 track." No `files`/`items` migration lands in this issue.
- **A resolver that actually applies `profile.repos.shorthand.issue_default` at read time.** This
  issue documents the rule (qualifier resolves at read time, never persisted) in the contract's
  own prose; it ships no resolver function. Wiring one is deferred to when a concrete consumer
  needs it (Option 4 — a cross-desk link surface or the D8 seeder), not booked here.
- **Write-time qualifier freezing** (the D2 Option 3 "self-describing" variant — a qualifier
  resolved and stored at author time). Re-opens only if the ADR 0009 files-are-truth trust gate
  flips.
- **Broadening the `kind` enum beyond `issue`/`url`** to cover `items.pointer` (a candidate
  `doc-pointer` kind) or the gate `DocRequirement.Pointer` selector. Both are genuinely open per
  the decision book: "whether `items.pointer` should be an instance at all... Not decided here";
  "Gate `DocRequirement.Pointer` as an instance... Flagged, not judged"
  (`_meta/research/2026-07-design-session/decision-book/D2-typed-cross-references.md`,
  Uncertainties). A builder needing a third kind opens a follow-up rather than expanding this
  issue's PR.
- **Versioning the `schema/` contract as a whole.** The decision book flags this as an
  undecided session call ("a purely additive contract change might avoid a version bump");
  `schema-versioning` is already its own tracked deliverable
  (`_meta/plans/epic-schema-v2-track/issue-body.md`). This issue's addition is treated as
  additive and versionless.

**Open owner decision, kept open with a default:** ADR 0011's own Consequences note — "if v2 is
long-delayed, revisit whether `graduated_to` should migrate early" — is not resolved here.
Default absent a ruling: it does **not** migrate early; Option 4 was chosen specifically to
decouple the contract from any migration, so a slow v2 track alone is not license to fold the
`graduated_to` migration into this issue or a quick follow-up without a fresh ruling.

_Provenance: ADR 0011 (`docs/decisions/0011-typed-reference-contract.md`), decision book D2
(`_meta/research/2026-07-design-session/decision-book/D2-typed-cross-references.md`), data-model
dossier §4/§6 (`_meta/research/2026-07-design-session/data-model.md`) — 2026-07-20 design
session._

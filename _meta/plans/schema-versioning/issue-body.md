> **Tracking:** #TBD, ADR 0009 (2026-07-20 design session). Version the shared `schema/`
> contract so both lanes can tell v1 from v2 and refuse an unknown version loud, instead of
> silently misreading it.

## Problem

ADR 0009's two-track ruling (`docs/decisions/0009-platform-frame.md:35-38`) fixes session
mechanisms against shipped schema v1 while the element model proceeds as the schema-v2 track,
and states plainly: "The shared `schema/` contract gets versioned" (line 38); under
Consequences, "`schema/` versioning becomes a build work item touching both lanes' loaders and
drift guards" (line 47). Today `schema/` carries **no version marker of any kind** — that is
the residual this issue closes.

**`schema/` today** (5 files, `command ls schema/`):
- `doctypes.yaml` — the doc-type vocabulary (universal keys, formats, enums, status families,
  ~28 types); no `version:`/`schema_version:` key anywhere in the file (`schema/doctypes.yaml:1-93`).
- `profile.schema.yaml` — the JSON-Schema-in-YAML profile validator; declares `$schema`/`$id`
  (`schema/profile.schema.yaml:1-2`) but no schema-CONTRACT version. Its own `schema_version`
  key (`:14-17`, `const: 1`) is a different thing entirely — it constrains the *profile
  instance* (`_knowledge/profile.yaml`) to declare itself version 1; it says nothing about
  which version of `profile.schema.yaml` itself is in force. **Reusing that same key name for
  the contract's own version would recreate the live `files.entity_type` vs `entity_type`-enum
  collision that ADR 0017 has ruled must be renamed** (rename ruled, not yet shipped) — same
  name, disjoint value spaces, no shared code path
  (`_meta/research/2026-07-design-session/data-model.md:324-343`). This issue's version marker
  must NOT be named `schema_version` for that reason.
- `README.md` — prose describing the two dimensions above; not contract data.
- `neutrality-lint.allow` — the identity-neutrality scanner's allowlist; tooling config, not
  contract data.
- `.gitkeep` — empty.

Only the two content dimensions (`doctypes.yaml`, `profile.schema.yaml`) are "the contract" a
version marker needs to cover.

**How each lane loads `schema/` today** (neither checks a version — there is none to check):

- **Go lane (librarian).** `librarian/internal/core/schema/doctypes.go` embeds a **vendored
  copy** at build time (`//go:embed doctypes.yaml`, lines 23-24, embedding
  `librarian/internal/core/schema/doctypes.yaml` — go:embed cannot reach outside its own
  module, so it is NOT the repo-root file directly). `Vocab()` (lines 57-60) parses it once per
  process via `sync.Once`; `parseDoctypes` (lines 62-91) unmarshals YAML into
  `Vocabulary{Universal, StatusFamilies, Types}` and fails loud on a structural defect (missing
  `universal`/`types`, lines 67-69; a type naming an unknown status family, lines 80-84) but has
  no concept of a version field to check. The embedded copy's byte-identity to the repo-root
  source is asserted by `TestDoctypesEmbeddedCopy_MatchesRepoRoot`
  (`librarian/internal/core/schema/doctypes_test.go:13-29`) — a raw byte comparison, not a
  version check.
  - `schema/profile.schema.yaml` is NOT read by the Go lane at all — a grep across
    `librarian/` finds it referenced only in comments
    (`librarian/internal/modules/librarian/setup/setup.go:23,28`;
    `librarian/internal/core/schema/doctypes.yaml:1`, byte-identical to the repo-root
    `schema/doctypes.yaml:1` per the embedded-copy drift guard above), never loaded by Go code.
    The Go lane's only `schema/` dependency is `doctypes.yaml`.
- **TS lane (plugin).** `plugin/core/schema.ts` walks up from cwd/module-dir looking for
  `schema/profile.schema.yaml` on disk (`discoverSchema`, lines 33-46; `defaultSchemaPath`,
  lines 53-58 — env override `DESK_SCHEMA_PATH` checked first). `loadSchemaObject` (61-67)
  parses the YAML into a plain object; `compileValidator`/`getValidator` (76-88) compile it
  with ajv draft-2020-12. No version field is read or asserted anywhere in this file.
  - The **bundled copy**, `plugin/claude-plugin/schema/profile.schema.yaml`, is produced by
    `bun run package` (`plugin/package.json:17`:
    `cp ../schema/profile.schema.yaml claude-plugin/schema/profile.schema.yaml`) and is
    drift-guarded in CI: both `.github/workflows/ci.yml:81-85` and
    `.github/workflows/release.yml:76-80` run `bun run package` then
    `git diff --exit-code -- claude-plugin/`, failing the build if the committed bundle
    differs from a fresh regeneration.
  - `schema/doctypes.yaml` is NOT read by the TS lane at all — a zero-match grep confirms no
    `plugin/core/*.ts` file references `doctypes.yaml` or its `entity_type`/`formats`/`enums`
    value-format checks (`_meta/research/2026-07-design-session/data-model.md:340-343`).

So the two lanes are already asymmetric consumers (Go reads only `doctypes.yaml`, TS reads only
`profile.schema.yaml`) — a version marker has to work for a lane reading either file alone, not
assume both are always present together.

**Distinct from an already-shipped, differently-scoped versioning mechanism.**
`docs/pm-system-v1-spec.md` §8.3 (R7.1, lines 797-806) already specifies
`module_schema_versions` — a PocketBase migration-version mechanism gating whether a STORE's PM
module schema is ahead/behind the binary. That is unrelated to this issue: R7.1 versions the
DATABASE migrations; ADR 0009 asks for a version on the shared `schema/` CONTRACT files
themselves (the doc-type vocabulary + profile JSON-Schema), which today have no version concept
at all. Do not conflate the two in the build.

**Where a version marker would have to be honored:**
- Both lanes' loaders (`librarian/internal/core/schema/doctypes.go` `parseDoctypes`;
  `plugin/core/schema.ts` `loadSchemaObject`/`compileValidator`) — each needs to read the
  marker and reject (or warn — open question, see Deliverables) a version it does not know.
- The embedded-copy drift guard (`doctypes_test.go:13-29`) — no logic change needed (still
  byte-identical), but the version key becomes part of what "byte-identical" covers.
- The bundle drift guard (`plugin/package.json:17`, CI `ci.yml:81-85`/`release.yml:76-80`) —
  same: the copied file's version marker travels with the copy; `make package` regeneration
  must still produce a byte-identical result.
- `schema/README.md` and the two build specs (`docs/pocket-librarian-v1-spec.md`,
  `docs/pm-system-v1-spec.md`) wherever they describe `schema/`'s consumption — need a sentence
  stating the version contract.

## Deliverables

- A version marker added to both `schema/doctypes.yaml` and `schema/profile.schema.yaml`.
  **Format is an open question — recommended default below, confirm at build time:**
  - **Recommended default: a top-level key inside each file** (e.g. `contract_version: 1` in
    `doctypes.yaml`; a parallel top-level `x-contract-version: 1` — NOT `schema_version`, see
    Problem — in `profile.schema.yaml`, exempt from `additionalProperties: false` since it is a
    schema-meta key, not an instance property being validated). Lowest-invasiveness option: no
    path/embed/copy-command changes in either lane, no CI step changes — just a new field both
    loaders read.
  - **Alternative considered: a versioned directory** (`schema/v1/doctypes.yaml`,
    `schema/v2/...`). Cleanly partitions the two-track model (ADR 0009) at the filesystem
    level, but is more invasive now: it changes the `//go:embed` path, the TS
    `SCHEMA_REL`/`discoverSchema` walk-up target, `plugin/package.json:17`'s copy command, and
    both CI drift-guard working directories — a bigger surface for a mechanism-only issue.
    Recommend deferring this shape until the v2 track actually needs a second,
    simultaneously-loadable schema tree (decide again when `element-model-revision` lands its
    v2 content — not now).
  - Coordination note: if `schema/references.yaml` (the ADR 0011 typed-reference contract,
    tracked separately as `_meta/plans/typed-reference-contract/`) has landed by build time, the
    version marker covers all three contract files, not two.
- `librarian/internal/core/schema/doctypes.go` (`parseDoctypes`, `Vocab()`): reads the version
  marker and returns an error naming the unrecognized version when it does not match a known set
  (today: `{1}`), the same fail-loud pattern the function already uses for a missing
  `universal`/`types` section (lines 67-69).
- `plugin/core/schema.ts` (`loadSchemaObject`/`compileValidator`/`getValidator`): reads the
  marker and rejects an unrecognized version with a clear error, mirroring the existing
  "profile violates schema v1" error shape (`loadAndValidateProfile`, lines 120-128).
- Drift-guard updates: `TestDoctypesEmbeddedCopy_MatchesRepoRoot`
  (`librarian/internal/core/schema/doctypes_test.go:13-29`) needs no logic change (still a byte
  comparison) but gains a comment noting the version key is now part of what it pins; a
  companion test (Go and TS) covers "loader rejects a value not in the known-version set" (see
  Acceptance).
- Spec prose: `schema/README.md` gets a paragraph stating the version contract;
  `docs/pocket-librarian-v1-spec.md` and `docs/pm-system-v1-spec.md` each get a sentence
  distinguishing this contract version from the already-shipped `module_schema_versions`
  store-migration mechanism (§8.3/R7.1), so a reader does not conflate the two.

## Acceptance criteria

- [ ] `schema/doctypes.yaml` and `schema/profile.schema.yaml` both carry a version marker
      (format per the Deliverables default, or whatever the build session confirms).
- [ ] A Go test demonstrates `schema.Vocab()`/`parseDoctypes` returns a non-nil error naming the
      unrecognized version when a `doctypes.yaml` byte slice's version marker is not in the
      loader's known set (the test can construct a modified byte slice in-memory; it does not
      need to edit the shipped file).
- [ ] A TS test (bun test) demonstrates `loadSchemaObject`/`compileValidator`/`getValidator`
      rejects (or warns — per whichever policy the build session picks, stated explicitly in
      the PR) a `profile.schema.yaml` fixture carrying an unrecognized version marker.
- [ ] `TestDoctypesEmbeddedCopy_MatchesRepoRoot`
      (`librarian/internal/core/schema/doctypes_test.go:13-29`) still passes — the embedded
      copy stays byte-identical to `schema/doctypes.yaml`, including the new version key.
- [ ] `make package` (`plugin/package.json:17`) regenerates
      `plugin/claude-plugin/schema/profile.schema.yaml` byte-identical to the versioned
      `schema/profile.schema.yaml`; `git diff --exit-code -- claude-plugin/` passes bare (the
      CI drift guard, `ci.yml:81-85`/`release.yml:76-80`, is unchanged in mechanism and still
      green).
- [ ] `go test ./...` (librarian) and `bun test` (plugin) are both green.
- [ ] `schema/README.md` states the version contract in prose; `docs/pocket-librarian-v1-spec.md`
      and `docs/pm-system-v1-spec.md` each carry a sentence distinguishing this contract version
      from `module_schema_versions` (§8.3/R7.1).

## Dependencies & gates

- Bundle drift guard (`make package` then `git diff --exit-code` on `plugin/claude-plugin/`):
  FIRES — this issue touches `schema/` directly (the `_config.md` gate table lists `schema/` as
  a trigger).
- Unit tests (`make test`): FIRES — `go test ./...` and `bun test`, including the new loader
  tests, are the real acceptance bar.
- Repo checks (`make check`): fires always (neutrality + self-test, kit drift, scaffold
  frontmatter, core purity, actionlint); no bare `#N` issue refs in any new/edited Go or TS file
  under `librarian/`/`plugin/` — this issue body's ADR references stay in `docs/`/`_meta/` prose
  only.
- Librarian integration (`make verify`, `librarian/verify.sh`, scratch desk): FIRES — this
  issue's Go changes are under `librarian/internal/core/schema/`, and the gate table's trigger
  is "anything under `librarian/`"; no specific `verify.sh` check is expected to exercise the
  new version field directly, but the gate still runs.
- Version sync (`check-version-sync.mjs`): does NOT fire — no root `VERSION` or shipped-manifest
  change (the new key is a schema-CONTRACT version, not the product `VERSION`).
- Kits drift: does NOT fire — `kits/`/`kits.yaml` untouched.
- DB migration discipline: does NOT fire — no PocketBase collection touched; this is a
  build-time YAML/JSON-Schema contract, not a store migration.
- CHANGELOG: an entry under `[Unreleased]` is required before a tagged release ships this
  (`check-changelog.mjs` gates the tag, not this PR).
- Substrate, not a blocker: this issue is a child of `epic-schema-v2-track`; it produces no v2
  content itself, but the `element-model-revision` issue's eventual v2 IMPLEMENTATION (not its
  docs-only revision) will need the versioning mechanism this issue ships.

## Out of scope

- Any v2 element-model CONTENT in `schema/` — this issue is the versioning MECHANISM only; the
  element model's own revision is tracked separately (`element-model-revision`, ADR 0018/0009).
- Choosing between the version-key and versioned-directory formats with finality — the
  Deliverables section states a recommended default and defers the final call to the build
  session.
- A migration/back-compat story for a hypothetical pre-version `schema/` snapshot already on
  some consumer's disk — there is exactly one shipped schema state today (implicitly "v1",
  unversioned); this issue adds the marker going forward, it does not need to detect or migrate
  an unmarked past state at runtime.
- Enforcing the `formats:`/`enums:` value-format checks in `doctypes.yaml` that are today
  "declared but not enforced"
  (`_meta/research/2026-07-design-session/data-model.md:374-378`) — a pre-existing, unrelated
  gap, not touched by versioning.
- Changing `module_schema_versions` (`docs/pm-system-v1-spec.md` §8.3/R7.1) — that mechanism is
  unrelated and already shipped; this issue only adds a cross-reference distinguishing it.

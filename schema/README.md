<!-- Status header: one-line purpose + status, per house convention -->
_Schema v1 — the product-neutral rule/structure contract the deskkit binary reads._
Status: active

# schema/

**What schema v1 is.** One product-neutral schema, and the single rule/structure source for the
`deskkit` binary. The plugin bundle carries no runtime of its own — it mounts the binary — so this
tree has exactly one consumer.

**The binary carries drift-guarded copies.** `go:embed` cannot reach outside the Go module's own
directory tree, so `profile.schema.yaml`, `doctypes.yaml`, and `references.yaml` each have a copy
under `internal/core/schema/`. **Edit the file here first, then re-copy it.** The copies are held
byte-identical by `TestProfileSchemaEmbeddedCopy_MatchesRepoRoot`,
`TestDoctypesEmbeddedCopy_MatchesRepoRoot`, and `TestReferencesEmbeddedCopy_MatchesRepoRoot`
(`make test`), which fail loudly on a one-sided edit.

**Contract version (ADR 0009).** The two content-bearing contract files each carry a top-level
version marker naming the version of the contract *file itself*: `doctypes.yaml` declares
`contract_version: 1` and `profile.schema.yaml` declares `x-contract-version: 1` (an `x-` schema-meta
key, stripped before schema compilation, exempt from `additionalProperties: false`). The loaders in
`internal/core/schema` (`parseDoctypes`/`Vocab()`, `compileProfileSchema`) read the marker and **fail
loud, naming the unrecognized value, when it is not in the known set** (today `{1}`). This lets a
reader tell one contract shape from the next instead of silently misreading it, and is the substrate
the schema-v2 track builds on. Two things it is deliberately **not**: (1) `profile.schema.yaml`'s
own `schema_version` key, which const-1 pins a profile *instance* to schema v1 (a different value
space, hence the marker is *not* named `schema_version`); (2) the store-side
`module_schema_versions` mechanism (`docs/development/specs/pm-system-v1-spec.md` §8.3 / R7.1),
which versions a desk's PocketBase PM migrations per install. Same word "version", disjoint
mechanisms.

**What's in it now.** Three schema dimensions, plus a shared path-constants file:

- `profile.schema.yaml` — the personalization profile block: identity, repos,
  board, desk paths, machines, models, secrets_ref, preferences, and the open-ended
  `custom` escape hatch. It validates a `_knowledge/profile.{yaml,json,md}` file. Only
  `schema_version` is required at the top level; nested objects require their own core
  keys when present (e.g. `repos.default`, `board.number`, `machines[].role`) — see the
  schema file for the exact contract.
- `doctypes.yaml` — the **doc-type dimension**: every doc `type` a filled SOP kit (`kits/`)
  emits, its status family, and required/optional fields. The contract as data. Port + gap
  dispositions: ADR 0006.
- `references.yaml` — the **reference dimension**: a typed cross-reference primitive — a
  `{kind, target}` shape with a closed `kind` enum (seeded `issue`, `url`) and a raw `target`
  string. The desk-relative repo qualifier resolves at read time from the profile
  (`repos.shorthand.issue_default`) and is never persisted. Enforced by
  `ReferenceVocab`/`ValidateReference` in `internal/core/schema`. Contract + rationale: ADR 0011.
- `paths.yaml` — the **filesystem path constants**. Its `profile_root` key names the single
  directory that holds the personalization profile (`profile.{yaml,yml,json,md}`);
  `internal/core/config`'s `ProfileRootDir` pins the same value, so a rename is a
  two-definition change. The `check-profile-root.mjs` guard fails if the two diverge.

**How validation is consumed.** A profile is validated against this schema before it is written —
one that violates the schema is rejected, not shipped forward. The neutrality lint
(`scripts/check-neutrality.mjs`) loads a reference profile that must itself validate, since the
profile doubles as the lint's denylist.

**The `custom:` escape rule.** Everywhere else in the schema is closed
(`additionalProperties: false`) so an unplanned key is rejected rather than silently
accepted. A key the schema does not yet define goes under `custom:`, which stays
open (`additionalProperties: true`) — agents never invent new top-level keys; a
genuinely recurring need graduates into the schema instead (a schema bump, not a
`custom:` workaround).

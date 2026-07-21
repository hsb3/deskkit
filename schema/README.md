<!-- Status header: one-line purpose + status, per house convention -->
_Schema v1 — the single shared schema `plugin/` and `librarian/` both consume._
Status: active

# schema/

**What schema v1 is.** One shared, product-neutral schema both the desk-standard plugin
and the deskkit binary read as their single rule/structure source
(`_meta/build-brief.md` §3.3(a); `_structure/decisions/0013` item 8). It is the seed of
a single estate-wide schema (`0013` item 4).

**What's in it now.** Three schema dimensions, plus a shared path-constants file:

- `profile.schema.yaml` — the M-05 personalization profile block
  (`_meta/m-05-data-surfaces.md`, "Field set (schema v1 profile block)"): identity, repos,
  board, desk paths, machines, models, secrets_ref, preferences, and the open-ended
  `custom` escape hatch. It validates a `_knowledge/profile.{yaml,json,md}` file. Only
  `schema_version` is required at the top level; nested objects require their own core
  keys when present (e.g. `repos.default`, `board.number`, `machines[].role`) — see the
  schema file for the exact contract.
- `doctypes.yaml` — the **doc-type dimension**: every doc `type` a filled SOP kit (`kits/`)
  emits, its status family, and required/optional fields. Product-neutral successor of the
  origin vault's `types:` model (0013 items 4 + 8). The contract as data; the runtime
  validation engine that consumes it is the PM-system build's job. Port + gap dispositions:
  [`docs/decisions/0006-kit-port-schema-reconciliation.md`](../docs/decisions/0006-kit-port-schema-reconciliation.md).
- `references.yaml` — the **reference dimension**: the typed cross-reference primitive both
  lanes share — a `{kind, target}` shape with a closed `kind` enum (seeded `issue`, `url`) and
  a raw `target` string. The desk-relative repo qualifier is documented as read-time-resolved
  from the profile (`repos.shorthand.issue_default`) and is never persisted. A validation guard
  lands in each lane (`librarian` `ReferenceVocab`/`ValidateReference`, `plugin`
  `validateReference`); no field migrates onto the shape yet. Contract + rationale:
  [`docs/decisions/0011-typed-reference-contract.md`](../docs/decisions/0011-typed-reference-contract.md).
- `paths.yaml` — the **filesystem path constants** both lanes read. Its `profile_root` key
  names the single directory that holds the personalization profile
  (`profile.{yaml,yml,json,md}`); each lane pins a per-lane constant to it (`plugin/core`
  `PROFILE_ROOT_DIR`, `librarian` config `ProfileRootDir`), so a rename is a one-definition
  change here plus both constants. The `check-profile-root.mjs` guard fails if the canonical
  value and either lane's constant diverge.

**How validation is consumed.** The M-05 substitution loader (build-brief D7) validates
an agent-written profile against this schema before write — a profile that violates the
schema is rejected, not shipped forward. The neutrality lint (D8) loads a reference
profile that must itself validate, since the profile doubles as the lint's denylist. A
CI check wiring schema validation into the required-checks list lands with D8.

**The `custom:` escape rule.** Everywhere else in the schema is closed
(`additionalProperties: false`) so an unplanned key is rejected rather than silently
accepted. A key the schema does not yet define goes under `custom:`, which stays
open (`additionalProperties: true`) — agents never invent new top-level keys; a
genuinely recurring need graduates into the schema instead (a schema bump, not a
`custom:` workaround).

# 0017 · Document identity & schema hygiene — frontmatter id, column rename, cap guard

_Settles the three promoted backlog gaps: rename identity, the `entity_type` collision,
implicit text caps._

- **Status:** Accepted
- **Date:** 2026-07-20
- **Raised by:** decision book `_meta/research/2026-07-design-session/decision-book/D8-backlog-identity-and-hygiene.md` (promoted from pull-only backlog at the owner's 2026-07-20 scope sign-off)

## Context

(a) A file rename is soft-delete + fresh insert — history is discarded, and any identity must
survive store-rebuild-from-disk. (b) The `files.entity_type` column and the schema's
`entity_type` enum share a name with disjoint value spaces — a genuine confusion trap that is
migration-cheap to clear. (c) Several content-bearing `TextField`s silently ride PocketBase's
5000-char default (the CLAUDE.md gotcha), with no guard against recurrence.

## Decision

Owner-ruled 2026-07-20 as a package (D8):

- **(a) Frontmatter `id`** is the document identity primitive: human-copyable, stable across
  renames, matched at sweep time alongside `path`. It is the only candidate that survives
  both a store rebuild (files-are-truth, ADR 0009) and the platform's git-agnostic mirror
  loop. Optional per document; absent id = today's behavior. Store-side checksum inference
  and git-based identity are rejected as primary mechanisms.
- **(b) Rename the `files.entity_type` column** (e.g. to `doctype`) via a forward migration;
  nothing depends on the name, only the value. Applied migrations stay untouched.
- **(c) The cap sweep + a recurrence guard**: one forward migration setting explicit `Max`
  on the flagged fields (0011/0013 sizing precedent), plus a CI check failing any new
  librarian-collection content `TextField` without an explicit `Max` (short label fields
  exempt by list).

## Consequences

- Rename stops discarding history for documents that carry an id; retrofitting existing docs
  is optional and incremental.
- The guard closes the cap class, matching the repo's drift-guard pattern; exemption list
  must be explicit so the check stays loud but not noisy.
- Under a future ADR 0009 gate flip, store-side identity could supplement (not replace) the
  frontmatter id.

## Affects

`librarian/.../tools/sweep.go` (id matching) · `schema/` (the `id` frontmatter key) · a new
forward migration (column rename + caps) · `scripts/` (the cap guard) · `query`/CLI output
field names · `docs/development/specs/pocket-librarian-v1-spec.md` identity prose.

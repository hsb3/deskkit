# 0015 · Prompt governance — git is truth; DB rows are a re-seeded cache

_Settles which copy of an agent's instructions wins across the embed, the DB, and markdown._

- **Status:** Accepted
- **Date:** 2026-07-20
- **Raised by:** decision book `_meta/research/2026-07-design-session/decision-book/D6-prompt-governance.md`

## Context

Agent instructions lived in three stores with no governing rule: the Go embed (seed), the
runtime-editable DB `prompts` collection, and version-controlled plugin markdown — soon four,
with the ADR 0014 bundle. A runtime GUI/REST prompt edit silently vanishes on a store
rebuild (independently re-derived: `prompts` is store-native and embedded-seeded; no code
path persists an edit to the desk tree — Verification record, claim 4). The stale shipped
librarian prompt was the live proof the split was ungoverned.

## Decision

**Git-is-truth under the standing regime** (D6 Option A, owner-ruled 2026-07-20):

- The version-controlled sources (Go embed / plugin markdown / bundle markdown) are
  canonical for shipped persona instructions. The DB `prompts` row is a **re-seeded cache**.
- Runtime edits are **ephemeral by documented rule** — that they do not survive a rebuild is
  now by-design, not an accident. "Reset to shipped" = clear the row; the embed re-seeds.
- The durable customization path is `_knowledge/` personalization (the profile), never DB
  rows and never edits to shipped artifacts.
- **Owner requirement recorded for the build:** prompt *tuning* must become possible in a
  **centralized** fashion — one canonical prompt set tuned in one place, never divergent
  per-project/per-store prompt versions. The tuning mechanism is a v2-track design item.
- Re-opens toward DB-as-truth (with a disk-import path) only if the ADR 0009 trust gate flips.

## Consequences

- The multi-source drift problem reduces to the repo's existing generated-artifact +
  drift-guard pattern (embed ↔ bundle markdown can be guarded; the DB row no longer competes).
- The stale-prompt class of bug becomes a guardable regression rather than a governance hole.
- Users who edited DB rows lose those edits at reset — acceptable and now documented.

## Affects

`librarian/internal/modules/librarian/prompt/prompt.go` (`Seed` semantics documented) ·
`collections/0009_prompts.go` prose · the ADR 0014 bundle's prompt sources · a drift guard
across the version-controlled prompt copies · `docs/pocket-librarian-v1-spec.md` prompt prose.

# 0012 · `items.type` validated at creation — hard reject unknown types

_Closes the ungated-advance hole for typo'd or unknown PM item types._

- **Status:** Accepted
- **Date:** 2026-07-20
- **Raised by:** decision book `_meta/research/2026-07-design-session/decision-book/D3-items-type-validation.md` (exec-desk agenda §3.5)

## Context

`CreateItem` performed no vocabulary check: an unknown `items.type` matches no gate rules,
yields zero effective requirements, and advances through every phase ungated (independently
re-derived; decision-book Verification record, claim 5). The checker exists
(`schema.Vocab().KnownType`) but guarded only gate configuration — an asymmetry, since gate
*rules* reject unknown types while item *births* accept them.

## Decision

**Validate at creation, hard reject** (D3 Option A, owner-ruled 2026-07-20): `CreateItem`
rejects a type not present in the shared vocabulary, with an actionable error naming the
known types. The check reads the vocabulary *as it stands at any time* — schema v1 doctypes
now, the v2 element list when the two-track (ADR 0009) lands. Existing rows are untouched;
only new creations are checked.

## Consequences

- A typo'd type fails loudly at the only write path (the importer inherits the check for
  free); the ungated-advance path closes for new items.
- Seed manifests carrying an out-of-vocabulary type will fail at import — the error must
  name the offending type and the vocabulary source so the fix is obvious.
- Post-creation type mutation (if any path allows it) must gain the same check when touched —
  flagged in the brief as unverified; verify during the build.

## Affects

`librarian/internal/modules/pm/engine/engine.go` (`CreateItem`) ·
`librarian/internal/core/schema/doctypes.go` (`KnownType` call site) · the PM importer ·
`docs/pm-system-v1-spec.md` (item-creation prose) · a red-able regression test per the #82 bar.

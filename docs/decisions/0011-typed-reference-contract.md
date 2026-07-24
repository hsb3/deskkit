# 0011 · Typed cross-reference contract — specify now, migrate on the v2 track

_Settles how cross-references (`graduated to:`, gate pointers, future refs) become typed._

- **Status:** Accepted
- **Date:** 2026-07-20
- **Raised by:** decision book `_meta/research/2026-07-design-session/decision-book/D2-typed-cross-references.md` (exec-desk agenda §3.3; #92's proper fix)

## Context

`graduated_to` is one untyped text field; a bare `42` is a valid, spec-pinned value with no
repo qualification. Qualification can only come from the per-desk profile
(`repos.shorthand`), so reference resolution is desk-relative by construction, and
identity-neutrality forbids any shipped default qualifier. The platform stream independently
converged on typed entity + typed relation (Grist-style references) as the unifying
primitive — the v2 element model needs exactly this substrate.

## Decision

**Specify the reference contract in `schema/` now; migrate fields with the v2 track**
(D2 Option 4, owner-ruled 2026-07-20):

- One reference primitive: **kind + target + optional desk-relative qualifier**, with a
  validation guard in the shared contract. No default qualifier ships (identity-neutral).
- Under the standing files-are-truth regime (ADR 0009), qualifiers resolve at **read time**
  from the profile and are not persisted; the write-time-freeze variant re-opens only if the
  0009 trust gate flips.
- `graduated_to`, gate pointers (per ADR 0010), and future refs become instances of the
  primitive; their field migrations ride the schema-v2 track rather than standalone
  migrations.

## Consequences

- The v2 element model's typed relations get their substrate from this contract — one
  definition, both lanes.
- No immediate store migration; shipped extraction/validation keeps working until the v2
  migration lands. The contract + guard land first and are testable alone.
- If v2 is long-delayed, revisit whether `graduated_to` should migrate early.

## Affects

`schema/` (the reference contract + versioning per ADR 0009) · both lanes' validators ·
`librarian/.../tools/sweep.go` graduation extraction (eventual instance migration) ·
`docs/development/specs/pocket-librarian-v1-spec.md` / `docs/development/specs/pm-system-v1-spec.md` reference prose.

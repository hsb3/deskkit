# 0009 · The platform frame — grow deskkit, staged truth regime, schema two-track

_Settles the frame the 2026-07-20 design session's other rulings bind to, and adopts the
migrated desk-platform stream's requirements rulings into this repo._

- **Status:** Accepted
- **Date:** 2026-07-20
- **Raised by:** decision book `_meta/research/2026-07-design-session/decision-book/D0-platform-frame.md`; owner reboot directive 2026-07-20

## Context

A parallel design stream (desk-platform, on the dev-tooling executive desk) ruled that its
agent-integration toolkit is desk-standard evolving — "grow deskkit" — and designed top-down
(surfaces, element model, KB storage) while this repo's design session worked bottom-up from
shipped reality. Its stated direction "PocketBase becomes truth once trust is established"
collides with this repo's store-rebuilds-from-disk constraint wall, and its element-model
redesign collides with rulings grounded on the shipped schema. The stream's working docs
migrated to `_meta/research/2026-07-design-session/platform/` (originals frozen).

## Decision

Owner-ruled 2026-07-20 (`_meta/signoff/2026-07-20-design-session-rulings/answers.json`):

1. **Ratify the platform R1 rulings as binding here**: the platform IS deskkit growing (one
   repo, max reuse); the coding-agent-harness **persona bundle is the v1 proof surface**;
   the non-negotiables hold (single-user local-first, single binary, MIT/permissive borrows
   only, identity-neutral, files-are-truth for derived data). Carve-out: R1's 4-entity spine
   content is superseded by the element-model reviews and lands via (3).
2. **Truth regime is STAGED.** Files-are-truth stands NOW; every current ruling binds to it.
   "PocketBase becomes truth" is a recorded *direction*, not a standing ruling — flipping it
   requires a future ADR at a **named trust gate** (to be specified concretely when the
   mirror/held-diff loop ships: a supervised operating period, proven byte-exact restore, an
   incident-free bar). Regime-dependent rulings (0011, 0013, 0015, 0017) state what changes
   if the gate flips.
3. **Two-track schema.** Session rulings fix *mechanisms* against shipped schema v1; the
   element model (three planes over a beefed spine) proceeds as the **schema v2 track**,
   folding in its adversarial-review fixes and consuming the session's mechanisms. The
   shared `schema/` contract gets versioned.

**Owner directive attached to (3):** run **simulations against both the v1 and v2 data
models** — walk realistic project scenarios through the model and its components to surface
deficiencies — before the v2 model is finalized. This is a named deliverable of the build plan.

## Consequences

- The desk-platform R3→R4 work re-homes in this repo's orbit; the other desk's stream is closed.
- `schema/` versioning becomes a build work item touching both lanes' loaders and drift guards.
- No ruling may silently assume PB-as-truth; the gate ADR is the only path to that regime.
- The element model stays draft until its review fixes land and the simulations pass.

## Affects

`schema/` (versioning) · the constraint-wall statements in `docs/CHARTER.md` and the specs ·
`_meta/research/2026-07-design-session/platform/*` (adopted record) · ADRs 0011/0013/0015/0017
(regime-parameterized) · the Phase-4 build plan (simulations deliverable).

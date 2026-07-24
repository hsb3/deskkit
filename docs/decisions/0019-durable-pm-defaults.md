# 0019 · Durable PM defaults — autonomous-writes-on and claim-TTL-30m recorded from spec §13

_Records the two durable build-contract PM defaults that shipped from `pm-system-v1-spec.md` §13,
so the spec→record chain is reconcilable from the ADR set rather than from PR archaeology._

- **Status:** Accepted
- **Date:** 2026-07-21
- **Raised by:** issue #103 (R-3); `docs/development/specs/pm-system-v1-spec.md` §13 items 7 & 9

## Context

Two durable build-contract PM defaults were taken by the PM spec and shipped in v0.7.0 as
defaults/seeds (spec §13 "As shipped (v0.7.0)"), but they lived **only** in spec §13 with no
decision record: **PM writes default autonomous-on** (§13 item 9) and **claim TTL default 30 min**
(§13 item 7). Because they were never captured as an ADR, the decision behind each build-contract
knob was recoverable only by reading PR titles — not from the decision record itself.

A related concurrency rule — **SQLite single-writer** — is deliberately **out of scope here**: it
is already recorded in [ADR 0002](0002-multi-desk-topology-store-per-desk.md) ("SQLite single-writer
is an operating rule"; store-per-desk means N stores never contend). This ADR does not restate it.

## Decision

Record the two shipped defaults as the durable build contract. They remain **owner-re-rulable
defaults, not bindings** — consistent with spec §13's "Shipping a default did not convert it into a
binding decision"; the value here is traceability (what the default is, why, and the override knob):

1. **PM writes default autonomous-on** — `PM_AUTONOMOUS_WRITES` defaults `true`. The real safety is
   the document gate on `advance`, not a write flag, so agents may drive the graph by default; a
   desk sets `PM_AUTONOMOUS_WRITES=false` to make agents read-only over the graph. PM tools write
   only the store, never desk files. (`librarian/internal/core/config/config.go:121`
   `envBool("PM_AUTONOMOUS_WRITES", true)`; `librarian/internal/modules/pm/module.go:79`.)
2. **Claim TTL default 30 minutes** — `PM_CLAIM_TTL` overrides it (per-desk `desk_config` may also
   override), with version-token optimistic concurrency as the "light" concurrency bar (R2.6).
   (`librarian/internal/core/config/config.go:117` `envDuration("PM_CLAIM_TTL", 30*time.Minute)`;
   `librarian/internal/modules/pm/engine/engine.go:67`.)

## Consequences

- The env-var overrides (`PM_AUTONOMOUS_WRITES`, `PM_CLAIM_TTL`) are the supported knobs; per-desk
  `desk_config` also overrides the claim TTL. Changing either shipped default is a build-contract
  change that updates this ADR and spec §13 in the same edit.
- The epic→record chain for these two defaults is now reconcilable from the ADR set; PR archaeology
  is no longer required to recover why each default ships as it does.
- Single-writer stays owned by ADR 0002; this ADR must never restate or re-decide it.

## Affects

`docs/development/specs/pm-system-v1-spec.md` §13 items 7 & 9 (now cite this ADR) ·
`librarian/internal/core/config/config.go` (`PMAutonomousWrites`, `PMClaimTTL`) ·
`librarian/internal/modules/pm/module.go`, `.../pm/engine/engine.go` (default application) ·
[ADR 0002](0002-multi-desk-topology-store-per-desk.md) (single-writer, out of scope here).

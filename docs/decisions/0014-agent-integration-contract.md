# 0014 · The agent integration contract — one contract, composed desk persona, gated mount

_Settles agent-surface parity: which librarian/PM asymmetries are policy, which were debt._

- **Status:** Accepted
- **Date:** 2026-07-20
- **Raised by:** decision book `_meta/research/2026-07-design-session/decision-book/D5-agent-contract-and-parity.md` (exec-desk agenda §3.1; the owner's headline concern)

## Context

The librarian (in-binary Go eino loop) and PM (plugin-markdown agent) integrations grew as
different kinds of objects on different layers, with unclaimed tools on every surface: the
librarian's own system prompt was stale independent of PM; ride-along tools sat on both the
PM mount and the eino loop; 3 of 4 TS plugin tools were claimed by no skill; PM `import` and
the admin console were claimed by no persona at all. Platform ADR 0009 makes the persona
bundle the v1 proof surface, raising the stakes of the mount question.

## Decision

Owner-ruled 2026-07-20 as a package (D5): **symmetry is a surface property of a shared
core.** One **integration contract** — persona instructions · tool mount · wake layer ·
write-gate policy — defined once; each agent instantiates it. The deliberate asymmetries
(inverted write-gate defaults; no in-binary PM brain) survive as *documented contract
parameters*. The four calls:

- **(a) Packaging:** ship ONE **desk-persona Claude Code bundle** that composes the
  librarian + PM module personas under the contract (not a separate librarian bundle; the
  platform's "desk agent" is the composition, not a third instantiation).
- **(b) Mount:** keep one shared `mcp-serve` mount, add **tool-level gating** so tools no
  persona claims are unreachable; fail-loud + stderr mount signal (the #79 precedent)
  applies to every mount. PM `import` and the admin console get named owners: `import`
  becomes a persona-claimed tool or is documented supervised-CLI-only; the admin console is
  documented as a human maintenance surface, not an agent surface.
- **(c) In-binary loop:** stays **librarian-only** — PM tools are excluded from the eino
  slice; the stale librarian system prompt content is fixed regardless (per ADR 0015's
  mechanism).
- **(d) Instructions:** the contract names **exactly one instruction source per surface**;
  the sourcing/sync mechanism is ADR 0015's.

Chat-to-schema capture (the platform's differentiator skill) rides this same contract as a
new claimed tool when it lands.

## Consequences

- The contract gets written down (spec section) and every current/future surface is audited
  against it; unclaimed-tool findings become CI-checkable in principle.
- Per-module mounts are ruled out for now; revisit only if tool-level gating proves
  insufficient in practice.
- The bundle is a new shipped artifact: it enters the neutrality-lint surface and needs its
  own packaging + drift guard (the marketplace copies only `plugin/claude-plugin/`).

## Affects

A new spec section (the contract) · `librarian/internal/core/mcp/server.go` (tool gating) ·
`librarian/templates/librarian-system-prompt.txt` (staleness fix) · the eino `buildTools`
slice · the new desk-persona bundle artifacts · `docs/tool-surface.md` · PM `import` +
admin-console docs.

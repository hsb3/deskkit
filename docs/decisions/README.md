_Index of Architecture Decision Records for deskkit._
Status: active

# Decision records

Append-only, zero-padded sequential ADRs. Status lifecycle:
`Proposed → Accepted | Rejected | Superseded-by-NNNN`. Cite the relevant ADR from code/docs
where the decision binds; never delete a record — supersede or correct in place.

| ADR | Title | Status |
|---|---|---|
| [0001](0001-interactive-surface-tui-first.md) | Interactive surface: terminal session first, PocketBase-served webapp deferred | Accepted (2026-07-16) |
| [0002](0002-multi-desk-topology-store-per-desk.md) | Multi-desk topology: store-per-desk, XDG store home, `desk` as open-guard | Accepted (2026-07-17) |
| [0003](0003-tool-commands-self-initialize-store.md) | Store initialization: tool commands self-initialize (auto-run app migrations) | Accepted (2026-07-17) |
| [0004](0004-chat-full-screen-tui.md) | `chat`: full-screen Bubble Tea TUI, streaming event layer, resume | Accepted (2026-07-18) |
| [0005](0005-versioning-and-changelog.md) | Versioning policy, CHANGELOG, and a missing-bump guard | Accepted (2026-07-18) |
| [0006](0006-kit-port-schema-reconciliation.md) | SOP kit port + schema-v1 doc-type reconciliation | Accepted (2026-07-18) |
| [0007](0007-tui-charm-v2-stack.md) | Chat TUI moves to the Charm v2 stack (terminal background retained) | Accepted (2026-07-18) |
| [0008](0008-pm-core-modules-architecture.md) | PM system architecture: core + compile-time modules (R5.5) — narrow validation seam, module-scoped migrations, per-desk feature gating | Accepted (2026-07-19) |
| [0009](0009-platform-frame.md) | The platform frame — grow deskkit, staged truth regime (files-are-truth + named gate), schema two-track + model simulations | Accepted (2026-07-20) |
| [0010](0010-pointer-grammar.md) | Pointer grammar — ratify shipped behavior; URL refs are not gate pointers | Accepted (2026-07-20) |
| [0011](0011-typed-reference-contract.md) | Typed cross-reference contract — specify in `schema/` now, migrate on the v2 track | Accepted (2026-07-20) |
| [0012](0012-item-type-validation.md) | `items.type` validated at creation — hard reject unknown types | Accepted (2026-07-20) |
| [0013](0013-disposition-completion-adoption-log.md) | Findings lifecycle completed — provenance on the finding, adoption log shrunk to writer-backed events | Accepted (2026-07-20) |
| [0014](0014-agent-integration-contract.md) | The agent integration contract — composed desk persona bundle, gated shared mount, librarian-only in-binary loop | Accepted (2026-07-20) |
| [0015](0015-prompt-governance.md) | Prompt governance — git is truth, DB rows a re-seeded cache, centralized tuning requirement | Accepted (2026-07-20) |
| [0016](0016-ts-boundary-deskkit-proxy.md) | TS plugin boundary — extend via a designed deskkit proxy; drift-guarded tool-surface truth | Accepted (2026-07-20) |
| [0017](0017-document-identity-and-hygiene.md) | Document identity & hygiene — frontmatter id, `entity_type` column rename, text-cap sweep + guard | Accepted (2026-07-20) |
| [0018](0018-element-model-direction.md) | Element-model direction — simple goal, optional workstream tag, research loop, trigger-gated exec outputs | Accepted (2026-07-20) |
| [0019](0019-durable-pm-defaults.md) | Durable PM defaults — autonomous-writes-on and claim-TTL-30m recorded from spec §13 | Accepted (2026-07-21) |
| [0020](0020-pm-claim-semantics.md) | PM claim semantics — a live claim is authoritative over every direct mutation | Accepted (2026-07-21) |

> **Tracking:** #129, rollup over the ADR 0010-0017 build slices (2026-07-20 design session). Every v1-track mechanism, contract, and guard the session ruled, landed against shipped schema v1.

## Why

The 2026-07-20 design session ruled direction as ADRs 0009-0018 (`docs/decisions/`), under the
ADR 0009 two-track frame: session rulings fix mechanisms against shipped schema v1 while the
element model proceeds separately as the schema-v2 track. This epic is the v1 side — the
mechanism slices (pointer grammar, typed references, type validation, findings lifecycle,
identity + hygiene), the agent integration contract with its desk-persona bundle, and the
governance guards (prompt copies, tool surface). Grouping them as one promise: when this epic
closes, every Affects surface named by ADRs 0010-0017 has shipped or been explicitly re-scoped
by a new ADR, and the shipped system matches what the session ruled.

Sequencing context: the 0.8.0 bug floor merged first (PR #112), then the session ruled
(PR #113). These slices are the feature work the sequencing directive was holding.

Coordination rule — the librarian migration chain: `findings-lifecycle-completion` and
`document-identity-hygiene` each add three forward migrations to the same
`librarian/internal/modules/librarian/collections/` chain. Migration numbers in their plans
are PROVISIONAL: the merging agent assigns the real sequence (basenames + the `module.go`
`SchemaVersion()` / manifest twin) from true current HEAD at landing time — whichever issue
lands first takes the next free numbers; the second continues from the true next free number.
Never dispatch both concurrently without serializing the migration commits.

## Children

- [ ] #115 `pointer-grammar-spec` — ADR 0010: normative spec section for the shipped pointer grammar
- [ ] #116 `typed-reference-contract` — ADR 0011: the reference primitive + validation guard in `schema/`
- [ ] #117 `item-type-validation` — ADR 0012: `CreateItem` hard-rejects unknown `items.type`
- [ ] #118 `findings-lifecycle-completion` — ADR 0013: dismissed retirement, disposition-aware counts, provenance, adoption-log shrink
- [ ] #114 `agent-integration-contract` — ADR 0014: contract spec section, tool gating, prompt fix, surface audit
- [ ] #119 `desk-persona-bundle` — ADR 0014: the composed librarian + PM Claude Code bundle (v1 proof surface per ADR 0009)
- [ ] #120 `prompt-governance` — ADR 0015: git-is-truth semantics + prompt-copy drift guard
- [ ] #121 `tool-surface-drift-guard` — ADR 0016: `docs/tool-surface.md` pinned by a guard
- [ ] #122 `ts-proxy-design` — ADR 0016: the TS `plugin/mcp` proxy design item (precedes implementation)
- [ ] #123 `document-identity-hygiene` — ADR 0017: frontmatter id, doctype rename, cap guard

## Close when

- Every child issue is closed, each with its PR/evidence linked.
- The ADR 0014 contract audit finds zero unclaimed tools across the agent surfaces
  (librarian eino loop, `mcp-serve` mount, TS plugin tools, the bundle).
- Every Affects surface named in ADRs 0010-0017 either shipped in a child's PR or is
  re-scoped by a recorded ADR (no silent drops).
- `make check`, `make test`, and `make verify` are green on main with all children merged.

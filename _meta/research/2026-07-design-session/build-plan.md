# Phase 4 — the build plan (structure derivation)

_How ADRs 0009-0018 slice into epics and issues. This is the session's Phase-4 record and the
shared fact-base for the drafting wave; the staged bodies under `_meta/plans/<slug>/` and the
filed issues are the live tracker._
Status: active (2026-07-20)

## Derivation rule

One issue per coherent build/design unit named by an ADR's **Decision + Affects**; an ADR
yields multiple issues only where its deliverables are independently ownable (0014's contract
vs its bundle; 0016's guard vs its proxy design). Two epics split the work by track, per the
ADR 0009 two-track ruling: v1 mechanisms bind to shipped schema v1; the element model proceeds
as the schema-v2 track.

## Epic A — design-session v1 build-out (milestone 1.0.0)

| Slug | ADR | Unit | Gate label |
| ---- | --- | ---- | ---------- |
| `pointer-grammar-spec` | 0010 | Normative spec section for the shipped pointer grammar; no code change; cites the pinning tests | gate:1.0.0 |
| `typed-reference-contract` | 0011 | Reference primitive (kind + target + optional desk-relative qualifier) + validation guard in `schema/`; field migrations ride v2 | gate:1.0.0 |
| `item-type-validation` | 0012 | `CreateItem` hard-rejects unknown `items.type`; importer inherits; post-creation mutation paths verified | gate:1.0.0 |
| `findings-lifecycle-completion` | 0013 | Retire `dismissed` (fwd migration); disposition-aware `summary`/`uncollapsed`; provenance on `patrol_findings`; shrink `adoption_log` to writer-backed `fix` | gate:1.0.0 |
| `agent-integration-contract` | 0014 | The contract spec section; tool-level gating on `mcp-serve`; librarian prompt staleness fix; eino slice stays librarian-only; `import` + admin-console ownership docs | gate:1.0.0 |
| `desk-persona-bundle` | 0014 | The composed librarian+PM desk-persona Claude Code bundle: new shipped artifact, packaging + drift guard + neutrality surface | gate:1.0.0 |
| `prompt-governance` | 0015 | Git-is-truth documented (`Seed` semantics, reset-to-shipped); drift guard across version-controlled prompt copies | gate:1.0.0 |
| `tool-surface-drift-guard` | 0016 | `docs/tool-surface.md` pinned by a drift guard in `scripts/` | gate:1.0.0 |
| `ts-proxy-design` | 0016 | Design item preceding implementation: TS `plugin/mcp` proxy to deskkit `mcp-serve` (lifecycle, availability, tool surface, 0014 gating); fallback recorded | none |
| `document-identity-hygiene` | 0017 | Frontmatter `id` + sweep matching; `files.entity_type` column rename; cap sweep + TextField-Max CI guard | gate:1.0.0 |

Epic close-when: children closed; every 0010-0017 Affects surface shipped or re-scoped by a
new ADR; the 0014 contract audit finds zero unclaimed tools.

## Epic B — schema-v2 element track (no milestone; own arc)

| Slug | ADR | Unit | Gate label |
| ---- | --- | ---- | ---------- |
| `schema-versioning` | 0009 | Version the shared `schema/` contract; both lanes' loaders + drift guards read the version | gate:v2-final |
| `element-model-revision` | 0018 + 0009 | Revise the element model under Q1-Q4 rulings + the two adversarial reviews' gaps (evidence triple, code/PR + bug entity, per-project-type core artifact) | gate:v2-final |
| `model-simulations` | 0009 | Owner-directed: walk realistic project scenarios through v1 AND draft v2; deficiency report gates v2 finalization | gate:v2-final |
| `trigger-design` | 0018 Q4 | Trigger mechanism for exec outputs (candidates: meeting; milestone/marker); until it lands, outputs stay on-demand | none |
| `prompt-tuning-centralized` | 0015 | Design item: one canonical prompt set tuned in one place, never per-store divergence | none |

Epic close-when: v2 element model finalized only after simulations pass; `schema/` versioned;
trigger + tuning designs recorded as ADRs or design docs.

## Native dependency edges (set at filing)

- `desk-persona-bundle` blocked-by `agent-integration-contract` (the bundle instantiates the contract).
- `ts-proxy-design` blocked-by `agent-integration-contract` (proxied tools gate per the contract).
- `model-simulations` blocked-by `element-model-revision` (needs the draft v2 to simulate).

Prose-only (no hard edge): prompt-governance's drift guard extends to bundle sources once the
bundle exists; element-model-revision consumes schema-versioning's mechanics when finalizing.

## Reconcile with the existing backlog

- **#99 (populate or retire `query adoption`) is ruled by ADR 0013** — shrink to writer-backed
  `fix`. Close #99 with a comment citing the ADR + the `findings-lifecycle-completion` issue.
- **#95 (actor/identity param on PM write tools)** — adjacent to 0013's provenance (actor on
  dispositions), distinct surface (PM write tools). Stays open; named in Known overlaps.
- **#101 (document which PM transitions gate)** — adjacent to 0010's spec section and 0014's
  contract section; stays open, named in Known overlaps.
- **#103 (spec-to-durable-record traceability)** — ADRs 0009-0018 partially deliver it for the
  PM defaults; stays open for the CHANGELOG-epic half.
- **#83 (PM default-on lane) is now unblocked** — #79 merged and the design session ruled; it
  remains existing-lane work, not part of this wave.

## Defaults applied (flag to owner, one-command fixes)

1. Epic A + children ride the **1.0.0 milestone**; Epic B rides **no milestone** (own arc, not
   a 1.0.0 promise). 2. New labels: `gate:1.0.0` (blocks the public-launch promise) and
   `gate:v2-final` (blocks v2-model finalization) per the gate-label standard. 3. No new
   milestone created for the v2 track; the epic tracks it.

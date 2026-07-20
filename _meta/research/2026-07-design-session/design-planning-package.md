_The design + planning package for the ADR 0009–0018 wave — the full corpus assembled for
adversarial review before the build lanes open. Status: active (2026-07-20)._

# Design + planning package — desk-standard v1 build-out

This indexes everything the build rests on: the binding decisions (ADRs), the reasoning and
evidence behind them (decision book + dossiers + platform stream), and the plans/issue bodies
that turn them into work. It is the review corpus and the corpus map. The **authoritative
layer is the ADRs**; the decision book and dossiers are the record; the plans are provisional
until built.

## A — Binding decisions (ADRs 0009–0018) · `docs/decisions/`

| ADR | Title | Binds |
|---|---|---|
| 0009 | platform-frame | truth regime (files-are-truth now; DB-becomes-truth gated), element-model as schema-v2 track |
| 0010 | pointer-grammar | what a work-item pointer may be |
| 0011 | typed-reference-contract | the typed cross-reference primitive |
| 0012 | item-type-validation | `items.type` validation |
| 0013 | disposition-completion-adoption-log | findings disposition + adoption-log completion |
| 0014 | agent-integration-contract | librarian vs PM agent contract & parity |
| 0015 | prompt-governance | prompt source & governance |
| 0016 | ts-boundary-deskkit-proxy | the TS-plugin ↔ deskkit boundary |
| 0017 | document-identity-and-hygiene | rename identity + document hygiene |
| 0018 | element-model-direction | the schema-v2 element model direction |

## B — Decision book (the reasoning) · `_meta/research/2026-07-design-session/decision-book/`

`README.md` (index + verification record) · `D0-platform-frame` · `D1-pointer-grammar` ·
`D2-typed-cross-references` · `D3-items-type-validation` · `D4-disposition-and-adoption-log` ·
`D5-agent-contract-and-parity` · `D6-prompt-governance` · `D7-spec-reality-reconciliation` ·
`D8-backlog-identity-and-hygiene`. Each brief: the question, the options, the evidence
headline, the recommendation. D0–D8 map to ADRs 0009–0017; the element model is 0018.

## C — Evidence dossiers · `_meta/research/2026-07-design-session/`

`data-model.md` (store/document model) · `workflows.md` (the workflow surface) ·
`surface-matrix.md` (MCP/CLI/TUI/skills projections) · `agent-symmetry.md` (the two agent
surfaces) · `document-model-gaps.md` (the modeling gaps #92/#93/#102 symptomize). Each cited
`path:line` against merged main with an explicit gaps section.

## D — Platform stream (merged in at the reboot) · `_meta/research/2026-07-design-session/platform/`

`README.md` (pointer) · `plan.md` (grow-deskkit, persona bundle as v1 proof) ·
`spec-element-model.md` (three-plane element model) · `system-cohesion-and-datamodel.md`.

## E — Deep plans + derivation · `_meta/plans/*/plan.md` + `_meta/research/.../build-plan.md`

`findings-lifecycle-completion/plan.md` · `agent-integration-contract/plan.md` ·
`desk-persona-bundle/plan.md` · `document-identity-hygiene/plan.md` · plus `build-plan.md`
(the ADR→issue derivation). Every load-bearing claim cited `path:line` against merged main.

## F — Staged issue bodies (17) · `_meta/plans/*/issue-body.md`

Epics: `epic-design-session-v1` (#129), `epic-schema-v2-track` (#130). Children (#114–#128):
`findings-lifecycle-completion`, `agent-integration-contract`, `desk-persona-bundle`,
`document-identity-hygiene`, `pointer-grammar-spec`, `typed-reference-contract`,
`item-type-validation`, `prompt-governance`, `ts-proxy-design`, `tool-surface-drift-guard`,
`schema-versioning`, `element-model-revision`, `model-simulations`, `trigger-design`,
`prompt-tuning-centralized`. Config: `_config.md`, `README.md`.

## G — Ground-truth specs (design corrected passages here) · `docs/`

`pocket-librarian-v1-spec.md` (§7.2 corrected) · `pm-system-v1-spec.md` (R6.1 corrected).
The plans' citations resolve against these + the shipped source under `plugin/` and `librarian/`.

## Review dimensions (what the adversarial pass hunts)

1. **Traceability** — every ADR ruling has a plan/issue; no orphan plans; coverage complete.
2. **Cross-document consistency** — ADRs ↔ book ↔ dossiers ↔ plans agree (truth regime,
   element-model vocabulary, pointer grammar, migration numbering, disposition semantics).
3. **Spec-vs-reality & citation integrity** — plan `path:line` citations resolve against
   current main; the two spec corrections are internally consistent.
4. **Buildability** — acceptance criteria testable, gates real, dependency/sequencing claims
   correct (#114 blocks #119/#122; #118+#123 migration commits serialize); hidden coupling.
5. **Identity-neutrality & generated-artifact risk** — no planned change risks the neutrality
   lint, the load-bearing spec/ADR paths, or the generated-artifact drift guards.
6. **Optimistic/unverified claims** — re-derive the flagged risks (UpdateItem unguarded `type`
   mutation; the librarian "kept verbatim" prompt block already stale; PocketBase reserves
   `id` → `doc_id`; `MCP_MODULES` unset must mean all-enabled) and any other stated-as-fact.

## Review output

Findings report at `_meta/research/2026-07-design-session/adversarial-review-2026-07-20.md`
(this folder). The build does not start until the BLOCKER findings are cleared or waived.

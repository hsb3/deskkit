---
type: analysis
status: draft
created: 2026-07-20
updated: 2026-07-20
tags: [design-session, decision-book, data-model, workflows, agent-symmetry, 1.0.0]
synopsis: Phase-1 decision book for the pre-feature design session — one brief per decision
  (question, options, dossier-grounded evidence, criteria, blast radius), dependency-ordered
  model → workflow → surface. The owner signs off on THIS LIST (scope) before any answers.
---

_Phase 1 of the design-session prep (`../README.md` §2). Informs the session; does not rule.
Rulings land as ADRs in `docs/decisions/`. Every brief's evidence resolves to a Phase-0
dossier section beside this book; dossier claims are hypotheses until re-derived where a
ruling binds._

Status: active (2026-07-20) — all eight briefs written, reviewed (see Verification record),
and the owner **signed off on the list** (scope sign-off 2026-07-20, recorded at
`_meta/signoff/2026-07-20-decision-book-scope/answers.json`): D1–D7 included as-is; **D8
promoted from pull-only backlog to a full session decision**; no missing decisions named.
One owner-input note rides D7 (TS-plugin-seam / server-backed-capabilities preference —
recorded in the brief; input to weigh, not a ruling).

**REBOOT (2026-07-20, owner directive):** the parallel **desk-platform** stream
(dev-tooling-desk) migrated into this session — its R1 ruled "grow deskkit", so its
decisions bind this repo. Working copies in `../platform/`; reconciliation brief
**`D0-platform-frame.md`** added (truth regime · element-model two-track · R1 ratification ·
the platform's four open questions); D2/D3/D4/D5/D6/D7/D8 carry platform-interaction
annotations. The other desk's pending R3 approval gate is superseded by this session. The
combined decide phase covers **D0 + D1–D8 + Q1–Q4**.

**RULED (2026-07-20 13:54Z):** the owner ruled the full combined set via the rulings form —
every recommendation accepted (`_meta/signoff/2026-07-20-design-session-rulings/answers.json`),
with three attached notes: v1+v2 **model simulations** before v2 finalizes (D0.c);
**centralized prompt tuning**, never per-store divergence (D6); exec outputs **trigger-gated**,
on-demand until triggers are defined — candidates: a meeting, a milestone/marker (Q4).
Recorded as **ADRs 0009–0018** in `docs/decisions/` (Phase 3); the two contradicted spec
passages corrected in place (pm-spec R6.1 pointer sentence; librarian-spec §7.2 TS-boundary
clause). This book is now the session's historical record; the ADRs are what binds.

# Decision book — index

## The decisions, dependency-ordered (model → workflow → surface)

| ID | Decision | Layer | Seeds | Depends on | Brief |
|---|---|---|---|---|---|
| D0 | The platform frame — ratify the migrated desk-platform R1 rulings; fix the truth regime (files-are-truth vs PB-becomes-truth); element-model two-track vs shipped schema; import platform questions Q1–Q4 | frame (rule first) | desk-platform R1/R2/R3 (migrated `../platform/`) · reboot directive 2026-07-20 | — | `D0-platform-frame.md` |
| D1 | Pointer grammar — what a pointer may be, what each form satisfies gate-wise, fail-closed rules, sub-file addressing vs rename tolerance | model | agenda §3.2 · C2 · tough spot 1 | — | `D1-pointer-grammar.md` |
| D2 | Typed cross-references — a first-class reference primitive (kind + target + qualifier); `graduated_to` and gate pointers migrate onto it | model | agenda §3.3 · C3 · tough spot 2 | D1 | `D2-typed-cross-references.md` |
| D3 | `items.type` validation — enforce the doctype vocabulary at item birth, or rule ungated advance deliberate | model | agenda §3.5 · C5 | — | `D3-items-type-validation.md` |
| D4 | Findings disposition + the adoption log — complete the disposition sub-machine (dead `dismissed`, disposition-blind counts, actor/reason), rule what `adoption_log` is FOR (5 of 6 event values writerless) | model + workflow | agenda §3.4 · C4 · Phase-0 finding (unwired enums) | — | `D4-disposition-and-adoption-log.md` |
| D5 | Agent integration contract & surface parity — one contract (instructions + mount + wake + write-gate), two instantiations; the four asymmetries policy-vs-debt; unclaimed tools on every surface (both agent loops, 3/4 TS plugin tools, PM `import`, admin console) | surface | agenda §3.1 · C1 · Phase-0 finding (unclaimed surfaces) | D1–D4 | `D5-agent-contract-and-parity.md` |
| D6 | Prompt governance / single-sourcing — one source of truth (or a documented split) across the Go embed, DB `prompts` rows, and plugin markdown | surface | agenda §3.1 (item 4) · C6 · tough spot 4 | D5 | `D6-prompt-governance.md` |
| D7 | Spec ↔ reality reconciliation — what the TS `plugin/mcp` boundary promises vs the 4 shipped profile tools; where tool-surface truth lives | surface | C7 | D5 | `D7-spec-reality-reconciliation.md` |
| D8 | Identity & hygiene — document identity / rename history, `entity_type` column collision, TextField implicit caps (**promoted to full decision at the 2026-07-20 scope sign-off**; authored as pull-only) | model | agenda §3.6 · C8 · tough spot 3 | — | `D8-backlog-identity-and-hygiene.md` |

**Adjacent but already queued elsewhere (NOT in this book):** the Lane B items
(`desk_config` gate-rule schema, `status_label` vocabulary, D5-artifact content, Obs-1
`_knowledge/` root, #81 retrofit defaults) and the 1.0.0 issues they neighbor (#94 #96
#101 #88). The session may pull one in, but the book doesn't duplicate their queues.

**Known interactions** (each brief states its side): D0→all — D0.b's truth regime
parameterizes D2/D4/D6/D8's criteria and D0.c's two-track split parameterizes D3's
vocabulary; D0.a's persona ruling pre-answers part of D5.a. D1↔D2 — the pointer grammar is
plausibly an instance of the reference primitive (or vice versa); D1 rules the grammar
and gate semantics first, then D2 rules whether refs (pointers included) migrate onto
one primitive — matching the table's order. D5↔D6 — the contract names *that* each surface has one instruction source (D5);
D6 rules the *mechanism*. D5→D7 — what the TS boundary should promise is a contract
parameter.

## Brief template (every brief follows it, sections in this order)

1. **Frontmatter** — the `_meta` analysis convention (type/status/created/updated/tags/synopsis).
2. **The question** — one sentence, then the elaboration.
3. **Why now** — what breaks or stays unruled without it; which shipped issues were symptoms.
4. **Evidence** — bullets of the form `dossier.md § Heading — claim (path:line)`. Every
   bullet resolves to a real dossier section; `path:line` cites are lifted from the dossier,
   never invented. Discrepancies between seed and dossier go to Uncertainties, not silently
   resolved.
5. **Options** — 2–4, each with consequences. Hypotheses may be marked (prep-doc §4 style);
   no option stated as a ruling.
6. **Decision criteria** — what any ruling must satisfy, citing the constraint walls
   (`../README.md` §7) that bind here.
7. **Blast radius** — concrete artifacts per option class: collections/migrations, spec
   sections, `schema/` contract, skills/personas, docs. Live stores exist — migration
   reality is part of the radius.
8. **Out of scope / interactions** — what this brief deliberately does not decide; which
   sibling brief owns the boundary.
9. **Uncertainties** — open evidence gaps carried forward explicitly.

## Verification record (2026-07-20)

Two independent report-only reviews ran after the eight briefs landed; findings were fixed or
recorded here — the briefs themselves are otherwise as their authors left them.

**Review 1 — conformance & coverage.** DoD items 2–5 all PASS with evidence: every Evidence
bullet resolves to a real dossier heading; ≥3 sampled `path:line` cites per brief were lifted,
none invented; the seed matrix (agenda §3.1–.6, C1–C8, three Phase-0 findings) has no uncovered
seed and one double-cover (stale librarian prompt: D5 as debt, D6 as canary) with the boundary
stated on both sides; all four cross-brief boundaries agree from both sides; no timelines, no
smuggled rulings; dependency order coherent. Item 1 initially FAILED on D4 only (leaked
tool-envelope tags at EOF + an out-of-template extra section) — both fixed in place. Cosmetic,
left as-is: section-numbering style varies per brief (unnumbered / §1-based / §2-based).

**Review 2 — adversarial re-derivation of the five beyond-dossier claims** (each was
plan-changing, so each was re-derived from source, not trusted):

1. **CONFIRMED, stronger than claimed** — `adoption_log` has live readers: `queryAdoption`
   (`librarian/.../tools/query.go:449-450`) surfaced as the `query adoption` CLI kind
   (`cmd/deskkit/main.go:778`) AND via the MCP `query` tool + TUI (shared tool core,
   `tools/specs.go:55`); it is also third in deskguard's desk-carrying set
   (`core/store/deskguard.go:13`) for desk-collision detection. D4's kill-option blast radius
   includes all query surfaces, the guard, and a migration.
2. **CONFIRMED and broader** — `docs/development/specs/pm-system-v1-spec.md:744` promises GitHub-issue-URL
   pointers while `Verdict` fails `://` closed (`librarian/internal/modules/librarian/module.go:123-124`;
   D1's bare `module.go` cites mean this file — no `pm/gates/module.go` exists). Broader: the
   shipped default gates (`pm/gates/defaults.go:23-36`) seed `pointer: item` for decision/task,
   so a URL-pointer item fails its default transition today. Raises the weight of D1's
   issue-URL sub-decision.
3. **CONFIRMED facts, softened reading** — Go `mcp-serve` exposes the librarian core (true);
   the TS `plugin/mcp` boundary ships exactly 4 profile tools (true); but the spec sentence
   (`pocket-librarian-v1-spec.md:1785-1787`) is ambiguous and leans a client-caller reading —
   D7's Option A framing ("amend to reality") holds; a "hard broken promise" framing would
   overstate it.
4. **CONFIRMED** — runtime (GUI/REST) prompt edits are ephemeral across a store rebuild:
   `prompts` is store-native and embedded-seeded (`prompt/prompt.go:24-40`,
   `collections/0009_prompts.go:9-12`); rebuild = delete the data dir + re-init; no code path
   persists a prompt to the desk tree. D6's premise is safe to rule on.
5. **CONFIRMED** — an unknown `items.type` yields zero effective requirements and advances
   ungated (`pm/gates/gates.go:133-150,180-181`); `CreateItem` does no vocabulary check
   (`pm/engine/engine.go:129-168`) while `KnownType` guards gate config only
   (`gates.go:70,110`). D3's mechanism claim is exact.

## Definition of done (Phase-1 gate)

- [ ] All eight briefs exist, template-conformant, one file each.
- [ ] Every Evidence bullet names a dossier file + section heading that exists, with cites
      lifted from that dossier (reviewer-verified).
- [ ] No overlap or gap against the seeds: exec-desk agenda §3 (six items) + prep-doc §3
      C1–C8 + the Phase-0 surfaced findings (unwired `adoption_log`, unclaimed surfaces,
      stale librarian prompt).
- [ ] Dependency ordering coherent — no brief depends on a later one.
- [ ] No timelines anywhere; no ruling smuggled in as fact.
- [x] Owner has signed off on the decision LIST (this index) — scope only, not answers.
      (2026-07-20, `_meta/signoff/2026-07-20-decision-book-scope/answers.json`; D8 promoted.)

---
type: analysis
status: draft
created: 2026-07-20
updated: 2026-07-20
tags: [design-session, data-model, workflows, agent-symmetry, access-surfaces, 1.0.0]
synopsis: Prep for the pre-feature design session ruled by the 2026-07-20 sequencing directive
  (bug floor → design session → features). Records the design lens (data model + workflows at
  the core, surfaces as projections), the process outline, what we already know we want to
  change, current best-guess choices, the tough spots, and the grounding-dossier index.
---

_Design-session prep — the process we'll run and the working material for it. Informs the
session; does not itself rule. Rulings land as ADRs in `docs/decisions/`. Seeded by the
exec-desk analysis `EXECUTIVE_DESK/Projects/desk-standard-desk/analyses/desk-standard-agent-symmetry-and-document-model-2026-07-20.md`
(written pre-PR-112 — its claims are delta-verified by the dossiers below)._

Status: draft (2026-07-20)

# Design-session prep

## 1. The design lens (owner framing, 2026-07-20)

The core of the product is **(1) a data model and (2) workflows over it**. Everything else is
an **access surface** projecting that core: the MCP server(s), the CLI, the skills, the agent
personas, the chat TUI, the admin console, and direct human file editing. "Fit to built-in,
bring your own harness — and easy for a human to get involved."

Two consequences the session should hold to:

- **Decide at the right layer, in order: model → workflows → surfaces.** A surface question
  that keeps resurfacing (the agent-symmetry concern) usually has an unruled model or
  workflow question underneath it.
- **Symmetry is a surface property of a shared core.** The KM (librarian) and PM agents
  should be two instantiations of one *integration contract* (persona instructions + tool
  mount + wake layer + write-gate policy), not necessarily one implementation.

## 2. Process outline

Deliverables and criteria only — no timelines.

| Phase | Work | Done when |
|---|---|---|
| **0. Ground** | Five evidence dossiers (§6), every claim cited `path:line`, exec-desk analysis delta-verified against post-PR-112 main | Dossiers exist beside this file; uncertainties explicit; foreman spot-checked |
| **1. Frame** | The **decision book**: one brief per decision — question, options, evidence links into the dossiers, decision criteria, blast radius — dependency-ordered model → workflow → surface. Seeds: exec-desk agenda §3 (six items) + the owner's symmetry priority + anything Phase 0 surfaces | Every brief's evidence resolves to a dossier section; owner has signed off on the *list* of decisions (scope), not yet the answers |
| **2. Decide** | The session itself. Package per owner preference: deck/PDF (+ audio) in `_meta/briefings/`, decisions collected via the owner-signoff form or live | Every decision in the book has a ruling or an explicit deferral |
| **3. Record** | Each ruling → ADR in `docs/decisions/` (append-only, cited where it binds); spec deltas drafted (`docs/development/specs/pocket-librarian-v1-spec.md`, `docs/development/specs/pm-system-v1-spec.md`, `schema/`); charter updated if direction moves | ADRs merged; specs no longer contradict rulings |
| **4. Plan the build** | Planning desk: epics/issues with acceptance criteria + gate labels; parallelism map. Lane 2 (PM default-on, #83) first unless rulings resequence | Issues conformant; `gate:` labels answer "what blocks 1.0.0" |

## 3. Things we know we want to change

From the exec-desk analysis (§1/§2/§3), the bug-floor experience (PR #112), and the owner's
framing. "Post-#112 status" = what the bug floor already did at the symptom level.

| # | Item | Post-#112 status | What the session must rule |
|---|---|---|---|
| C1 | **Agent-surface parity** (the owner's headline) | Untouched (delta-verified: 14/15 analysis claims hold) | Which of the four unruled asymmetries are policy vs debt: librarian Claude Code bundle or CLI/TUI-first ruling; per-module vs shared MCP mount (ride-along tools); in-binary loop gets a PM prompt or loses PM tools; prompt governance per surface (DB-seeded vs markdown). Surface-matrix widened it: the unclaimed-tool problem exists on BOTH agent surfaces (the librarian's own system prompt is stale independent of PM), 3 of 4 TS plugin tools are claimed by no skill, and PM `import` + the admin console are claimed by no persona at all |
| C2 | **Pointer grammar** | `§ heading` anchors resolve by file part (#102); `#` anchors fail with an actionable hint | What a pointer *may be* (whole file / sub-file locus / issue URL / external), what each satisfies gate-wise, fail-closed rules. The current grammar is implemented but specified nowhere |
| C3 | **Typed cross-references** | Extraction fixed (#92: explicit marker only) — but `graduated_to` remains one untyped text field; a bare `42` is a valid, spec-pinned value; no repo qualification | A first-class reference type: kind + target + qualifier. `graduated_to`, gate pointers, and future refs become instances |
| C4 | **Findings disposition model** | #93 shipped `disposition` (open/acknowledged/triaged/wont_fix) orthogonal to `state`, with re-patrol inheritance | Residue: `state` still declares `dismissed` with no setter; `query summary`/`uncollapsed` counts ignore disposition; `adoption_log` is worse than "false_positive detached" — 5 of its 6 event values (`patrol/revert/false_positive/friction/note`) have NO writer; even restore logs nothing. Rule the complete sub-machine + what the adoption log is actually FOR |
| C5 | **`items.type` validation** | Untouched — plain TextField, unvalidated at `CreateItem`; a typo'd type advances ungated | Enforce doctype vocabulary at item birth, or rule ungated-advance deliberate |
| C6 | **Prompt governance / single-sourcing** | Untouched | One source of truth for agent instructions across the Go embed, the DB `prompts` collection, and plugin markdown — or a documented split |
| C7 | **Spec/reality residue** | #94 shipped `docs/development/specs/tool-surface.md` with verified counts | Spec still promises librarian tools over the TS `plugin/mcp` boundary that ships only 4 profile tools; reconcile spec → reality |
| C8 | Backlog (pull only if a use case demands) | Untouched | Rename identity / doc history (store is rebuildable — any identity must survive rebuild); TextField implicit 5000-char caps; `files.entity_type` column colliding with the schema's `entity_type` enum |

## 4. What looks like the right choice (hypotheses to ratify — not rulings)

- **Model-first sequencing.** Rule C2/C3/C4/C5 (model) before C1/C6 (surfaces). The
  asymmetry ruling gets easier once the model rulings say what every surface must expose.
- **Symmetry as one contract, two instantiations.** Define the integration contract
  (instructions + mount + wake + write-gate) once; librarian and PM each instantiate it.
  Don't force the PM into an in-binary loop or the librarian out of one — the deliberate
  asymmetries (inverted write-gate defaults, no in-binary PM brain) survive as *documented
  contract parameters*.
- **Ship a librarian Claude Code bundle** mirroring desk-pm's D5 shape (agent + skills +
  hook + `.mcp.json`). The "fit to built-in" story is incomplete while the librarian's only
  Claude Code path is a README snippet.
- **Per-module MCP surfaces** (e.g. `mcp-serve --module=pm`) over today's
  everything-enabled mount, so no surface carries tools no persona claims. Apply #79's
  fail-loud ruling to every future mount.
- **One typed reference primitive** in `schema/` (kind + target + optional desk/repo
  qualifier), consumed by both lanes; `graduated_to` and gate pointers migrate onto it.
- **Validate `items.type` at birth** against the shared vocabulary — cheap, closes the
  ungated-advance hole, symmetric with how doc frontmatter is already validated.
- **Preserve two load-bearing invariants** while changing the model: gates read disk
  directly (never sweep-dependent), and the store stays rebuildable from the desk tree.

## 5. Tough spots

- **Sub-file addressing vs rename tolerance.** The `§ heading` anchor is *advisory by
  ruling* — gates ignore it precisely so heading renames can't break transitions. Real
  sub-file addressing needs stable section identity (explicit ids? tolerant resolution?)
  without reintroducing that fragility. This is C2's hard center.
- **Reference identity across desks.** A bare `#N` (or `42`) means different things on
  different desks; qualification needs `repos.shorthand` from the *profile*, which is
  per-desk personalization — so reference resolution is desk-relative by construction.
  Identity-neutrality forbids baking any default qualifier into shipped code.
- **Where document identity lives** (C8): frontmatter id (human-copyable, survives rebuild),
  store-side rename inference (checksum heuristics), or git. Must survive
  store-rebuild-from-disk; today rename = soft-delete + fresh insert, history discarded.
- **Prompt single-sourcing across three stores** — Go-embedded seed, DB `prompts` rows
  (GUI-editable at runtime), plugin markdown (version-controlled). The repo already has the
  pattern for this (generated artifacts + drift guards), but runtime-editable DB prompts
  fundamentally can't be drift-guarded against a git source — the governance question is
  real, not mechanical.
- **One mount vs per-module mounts.** Per-module is cleaner but multiplies processes and
  `.mcp.json` entries per session; a shared mount needs tool-level gating so unclaimed
  tools aren't silently reachable. Interacts with the in-binary loop question (C1 item 3).
- **Migration reality.** Live stores exist (the owner's own desks). Every model change is a
  forward migration under record-original-first; `schema/` is the *shared* contract — a v2
  moves both lanes at once. Decide whether the contract gets versioned.
- **Disposition coherence across surfaces** (C4): "open findings" must mean one thing in
  `query findings`, `query summary`, MCP, and the TUI — today the summary counts disagree
  by design-gap.

## 6. Grounding dossiers (Phase 0)

Each dossier is owned by one read-only-except-own-file agent; every claim cites `path:line`
against post-PR-112 main (`51235f6`). **Dossier claims are hypotheses** — anything a ruling
will bind on gets re-derived in the session.

| File | Question | Status / headline |
|---|---|---|
| `data-model.md` | Every collection/field/constraint both lanes persist; the `schema/` contract; what's typed vs stringly; unwired enum values, implicit caps, identity/keying | **done.** Unwired enums broader than known: `state.dismissed` AND 5 of 6 `adoption_log.event` values (even `revert` — restore logs nothing; foreman-verified). `items.type` unvalidated confirmed. `entity_type` collision genuine (disjoint value spaces). 9 explicit gaps |
| `workflows.md` | Every workflow end-to-end (sweep, patrol, propose/apply/restore, feedback, eino loop; PM machine, gates, claims, cascade, importer) — trigger → steps → writes → invariants; where KM and PM meet | **done.** 18 workflows. Correction: patrol has SIX rules (R1–R6, HANDOFF staleness included). Gate-reads-disk invariant confirmed. No undocumented spec/code divergence. 9 explicit gaps |
| `surface-matrix.md` | Operations × surfaces matrix (MCP by gate, CLI, TS plugin tools, skills, personas, TUI, admin console, human file paths) — who can do what from where, and its gating | **done.** Counts match `docs/development/specs/tool-surface.md` exactly. Six unclaimed-surface findings: ride-along on the PM mount AND on the eino loop (the librarian system prompt is stale even without PM); 3 of 4 TS plugin tools claimed by no skill; PM `import` and the admin console claimed by no persona at all |
| `agent-symmetry.md` | Delta-verification of exec-desk analysis §1 against current main: each claim CONFIRMED / STALE / CHANGED with fresh citations | **done.** 14/15 CONFIRMED, 1 CHANGED: #79 resolved by fail-loud serve + mount signal, not symmetric self-gating. The symmetry concern survives PR #112 intact |
| `document-model-gaps.md` | Delta-verification of exec-desk analysis §2 (gaps A–F) against current main: what #112 actually closed vs what remains | **done** (proved with a live green test run). D closed (9 tx-wrapped methods). A/B symptom-closed, model-open. C lifecycle shipped, residue stands (dead `dismissed`, disposition-blind summary counts, detached `false_positive`, no actor/reason). E/F untouched |

## 7. Constraint walls (any design must respect)

- **Identity-neutrality** of everything shipped — CI-enforced (`scripts/check-neutrality.mjs`).
- **Harness-purity** of `plugin/core` — enforced (`scripts/check-core-purity.mjs`).
- **Record-original-first** write boundary; every fix byte-exact reversible via `restore`.
- **Store self-initializes and rebuilds from disk**; document bodies are never persisted
  (sole exception: the revisions pre-image ledger).
- **Forward migrations only** on shipped collections; never edit an applied migration.
- **PocketBase bootstraps before cobra** — anything preventing store creation runs in
  `main()`; fail-closed serve paths need `os.Exit(1)`.
- **Makefile is the task interface; gates run bare, never piped.**
- Personalization only via `_knowledge/profile.yaml` — never by editing shipped artifacts.

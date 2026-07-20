---
type: analysis
status: draft
created: 2026-07-20
updated: 2026-07-20
tags: [design-session, decision-book, platform, truth-regime, element-model, reconciliation]
synopsis: D0 — the platform frame. Reconciles the migrated desk-platform stream (R1 rulings,
  element model R3, store/cohesion review) with this decision book. Rules the truth regime the
  other decisions bind to, the element-model's relationship to the shipped schema, and imports
  the platform's four open questions into this session's scope.
---

_Decision book brief D0 (`README.md`). Added at the 2026-07-20 reboot when the parallel
desk-platform stream migrated into this desk (owner directive). Unlike D1–D8, its evidence
base is the migrated platform docs in `../platform/` (primary sources, not Phase-0 dossiers) —
plus the cross-stream review recorded in the session handoff. Informs the session; does not
itself rule._

Status: draft (2026-07-20)

# D0 — The platform frame

## 1. The question

Under what platform frame do D1–D8 get ruled? The parallel stream's R1 (closed with the owner
2026-07-20) decided **"grow deskkit"** — the desk-platform toolkit IS desk-standard evolving.
That makes its rulings binding context for this book. D0 ratifies or amends them as they bind
this repo, and rules the two meta-questions they raise: **(b) the truth regime** and **(c) the
element model's relationship to the shipped schema**. It also imports the platform's four open
questions (d) into this session.

## 2. Why now

Without D0, the two streams contradict: four of this book's briefs (D2, D4, D6, D8) derive
decision criteria from the *store-rebuilds-from-disk / files-are-truth* constraint wall, while
platform R1 decision 4 states a direction of travel — *"PB becomes truth once trust is
established"* — that eventually inverts that wall. And D3 validates item types against a
vocabulary that platform R3 is redesigning wholesale ("the current PB schema is a first draft,
not binding"). The session must fix the frame before those rulings can be stable.

## 3. Evidence (primary sources — the migrated platform docs)

- `../platform/plan.md` — R1 decisions 1–5 (grow deskkit; persona bundle as v1 proof surface;
  4-entity spine, later beefed; files-mirrored-with-held-diffs KB model + the PB-becomes-truth
  direction; the five non-negotiables incl. identity-neutral + single-binary). R2 stack
  (fsnotify + go-diff, PB `revisions`, Grist-style typed relations). R3 status: element model
  drafted, both adversarial reviews back, owner approval pending — now superseded by this session.
- `../platform/spec-element-model.md` — the three-plane element model over a beefed spine
  (~10 entity types), the 5 net-new gaps, and both reviews' findings (shared defect: the v1
  spine deferred each project type's core artifact).
- `../platform/system-cohesion-and-datamodel.md` — the store ruling (one PocketBase; census
  synced read-mostly, PM/KM authored natively), the **two data natures** (derived vs authored),
  and the typed entity + typed relation unifying primitive; frontend options A–D.
- This repo's constraint walls: `../README.md` §7 (store self-initializes and rebuilds from
  disk; document bodies never persisted except the revisions ledger; identity-neutrality;
  record-original-first).

## 4. Sub-decisions and options

### 0.a — Ratify R1's product rulings as they bind this repo

R1 decisions 1 (grow deskkit), 2 (persona bundle = v1 proof surface), 3 (spine scope), and
5 (non-negotiables) were owner-ruled on the other desk and are consistent with this repo's
walls. Ratifying them here (as an ADR) makes them citable where they bind. *Hypothesis:
ratify as-written; decision 3's spine content is superseded by the R3 reviews and lands via
0.c, not as-frozen.* The alternative — treating them as advisory until re-derived — costs a
round-trip and contradicts the owner having already ruled them.

### 0.b — The truth regime (the load-bearing call)

What is authoritative for **authored knowledge content** (documents), now and later?

- **(i) Files-are-truth, permanent.** The current walls stand forever; PB is always a
  rebuildable index + staging layer (held diffs, revisions). Simplest; forecloses the
  platform's stated direction.
- **(ii) Staged regime with an explicit future gate (hypothesis — recommended).** Files are
  truth NOW and every ruling in this book binds to that regime; *PB-becomes-truth* is
  recorded as a direction, not a standing ruling — flipping it requires a future ADR at a
  named trust gate (e.g. after the mirror/held-diff loop has run supervised for a period,
  with restore proven). Preserves both streams' intent; makes regime-dependent rulings
  (D2 qualifier storage, D4 provenance durability, D6 prompt durability, D8 identity)
  explicitly *regime-parameterized* — each states what changes if the gate flips.
- **(iii) PB-is-truth now.** Matches the direction immediately but breaks the walls, the
  librarian's record-original-first boundary, and the trust rationale the owner himself
  gave for the mirror phase ("on-disk files are a trust comfort until the system earns it").

Note the **two-data-natures** boundary from the cohesion review is orthogonal and already
settled: *derived* census data is always regenerable (files/manifests are truth); *authored
PM/KM entities* (tasks, meetings, people, decisions) are already store-native today — the
regime question is specifically about authored **document content**.

### 0.c — The element model vs the shipped schema

R3's element model redesigns the vocabulary top-down (three planes, beefed spine, the
reviewers' additions). The shipped `schema/` v1 + PB collections are what D1–D8 ground on.

- **(i) Two-track (hypothesis — recommended):** D1–D8 rule the *mechanisms* against shipped
  v1 (pointer grammar, reference primitive, validation-at-birth, disposition machine,
  contract); the element model proceeds as the **schema v2 track** (R3→R4), consuming those
  mechanisms — the reference primitive (D2) becomes v2's typed-relation substrate, the
  doctype vocabulary (D3) becomes v2's element list, the document+revision mirror (R2 stack)
  extends the librarian boundary. `schema/` gets versioned (this answers the "does the
  contract get versioned" question D2 carried).
- **(ii) Pause D1–D8 model rulings until R3 lands.** Avoids any rework but blocks the bug-floor
  follow-ons (#92/#93/#102 modeling) on a design track that failed its first completeness
  review — sequencing the settled behind the unsettled.
- **(iii) Fold R3 into this book now as full decisions.** One session rules everything, but
  the element model's own reviewers judged it not yet able to run a project end-to-end —
  it isn't decision-ripe beyond its IA skeleton and open questions.

### 0.d — Import the platform's four open questions

From `spec-element-model.md` §8, previously awaiting the owner on the other desk — now this
session's items: **Q1** goal vs OKR-style objective+key-results; **Q2** keep the `_headcase`
workstream tag alongside planes or drop it; **Q3** the research decomposition shape (the
research review argues loop-not-waterfall, with `open-question` as the re-entry point);
**Q4** which deferred exec outputs matter first. These are owner-preference calls with light
evidence; the session answers them directly.

## 5. Decision criteria

- Any regime ruling must keep the librarian's **record-original-first / byte-exact restore**
  boundary intact through the transition it defines (wall, and R1's own rationale).
- **Identity-neutrality** binds every platform artifact that ships (R1 non-negotiable = the
  repo's CI-enforced rule; convergent, but v2 schema work must stay inside it).
- Ratifications must not silently re-open what the owner already ruled (R1), nor freeze what
  its own reviews falsified (the R3 spine).
- Whatever 0.b picks, each regime-dependent brief (D2/D4/D6/D8) must state its answer under
  the standing regime AND what flips at the gate — no ruling may be silently regime-blind.

## 6. Blast radius

- **ADRs:** D0's rulings land as the first ADR(s) of the session set; R1 ratification cites
  the migrated `plan.md`.
- **`schema/`:** versioning (v1 frozen contract + v2 track) — touches both lanes' loaders and
  the drift-guard set.
- **Docs:** `docs/CHARTER.md` gains the platform direction (charter precedence rule applies);
  the spec set gains the two-track note.
- **Desks:** dev-tooling-desk's desk-platform thread is closed (migration pointers in place);
  this repo's handoff carries the merged session; the R3→R4 work re-homes here.
- **The decide phase:** the combined form covers D0.a–d + D1–D8 + Q1–Q4.

## 7. Out of scope / interactions

- D0 does not rule any D1–D8 question on its merits — it fixes the frame they're ruled in.
- The persona-bundle pre-answer flows to **D5.a** (annotation there); the mount question
  stays D5.b's; the TS-seam reconciliation stays D7's (three-sided note there).
- Estate questions outside this repo (dotfiles-agents naming cohesion, PM ownership overlap
  with github-project-board, frontend option A–D for the webapp lane) stay on their desks;
  only what binds desk-standard migrated.

## 8. Uncertainties

- The R1 record is the migrated plan.md's own text; no separate signed artifact exists for
  it (the rounds were conversational). Ratification (0.a) is partly *creating* the durable
  record — flagged, not a defect.
- The trust-gate criteria for 0.b(ii) are sketched, not specified — if chosen, the ADR must
  name them concretely (supervised period, restore proof, incident-free bar).
- Whether the platform persona (the "desk agent") is a third contract instantiation or the
  composition of the librarian + PM personas is a D5 design question the session should
  answer while ruling D5.a.

---
type: plan
status: draft
created: 2026-07-20
updated: 2026-07-20
tags: [dev-tooling, desk-platform, agent-integration, data-model, workflows, pocketbase, tui, webapp, mcp, rounds]
synopsis: "Design space (draft, iterated in rounds) for a toolkit that integrates AI agents with ONE unified data model + workflows, exposed through many surfaces: a coding-agent harness assuming a desk persona (skills + MCP, or CLI), plus a built-in agent over CLI/REPL, TUI, and eventually a webapp. Store settled on PocketBase (per system-cohesion-and-datamodel.md); this doc is the build. Method: rounds — requirements -> tech/borrow survey -> data+workflow spec -> prototypes -> integrate. Obsidian deferred; prefer borrowing MIT-licensed projects."
---

# Desk platform — agent-integration toolkit over a unified data model + workflows

_Status: **draft**, iterated in rounds (Henry, 2026-07-20). The store question is settled in
[../extenders-estate/system-cohesion-and-datamodel.md](../extenders-estate/system-cohesion-and-datamodel.md)
(PocketBase); this doc designs the toolkit around it. Nothing here is locked until a round closes._

## Vision

Provide a **toolkit with many ways to integrate AI agents against one unified data model + workflows** —
not a single app. The data model + workflows are the core; agents plug in through whichever surface fits.

## North-star acceptance test (Henry, 2026-07-20)

The whole system — **data models + templates + workflows + tooling + UI together** — must be sufficient to
**run most knowledge-work projects**, *excluding* the developer-specific workflows of active development (the
code desk's domain). The executive desk it serves does three things: **(1) refine information · (2) set
goals / targets / deadlines · (3) prepare communication packages.** The data model + workflows are designed
**top-down from these three pillars**; the current PB schema is a **first draft, not binding** — redesign
freely. This is the platform's own instance of desk-standard `0021`-F7 (cohesion + value eval).

## Surfaces (integration modes)

| Surface | How an agent integrates | Status today |
| --- | --- | --- |
| **Coding-agent harness as the desk agent** | A **persona/agent definition + skills + MCP server** (or CLI) that make Claude Code / opencode *become* the desk agent | deskkit ships MCP + CLI; desk-pm is an agent surface |
| **Built-in agent — CLI / REPL** | Ships configured; talk to it in a shell | deskkit CLI exists |
| **Built-in agent — TUI** | Terminal UI over the same core | not built |
| **Built-in agent — webapp** | Browser UI (PocketBase-served) | 1.0 lane #19 (stub) |
| **Direct data access** | MCP tools · CLI · PB REST for scripted/human use | partial (deskkit) |

## Core (shared by every surface)

- **Unified data model** — typed **entities** (person · topic · decision · task · meeting · extender · note)
  + typed **relations** (task—about→topic, meeting—with→person, decision—rules→topic). PocketBase-backed;
  authored PM/KM native, derived census synced in read-mostly.
- **Workflows** — the document-gated phase machine (desk-pm); **chat-to-schema capture** ("meet with Tony
  re: duckdb vs pocketbase" → structured record); census sync; KB linking.

## Product-boundary question (Round 1)

Is this **desk-standard/deskkit growing** (it already = PocketBase + MCP + CLI + PM module + a webapp lane),
a **new layer on top**, or a **distinct product**? Resolve before tech choices — it decides the repo home.

## Method — rounds

1. **R1 · Requirements** (closed 2026-07-20) — decisions recorded below.
2. **R2 · Tech stack + MIT-project survey** (active) — borrow candidates (file↔DB mirror/diff engines,
   PocketBase-backed PM/KB apps, entity-graph patterns, Go TUI later). License gate: MIT/permissive only.
3. **R3 · Data-model + workflow spec** (active) — designed top-down from the three exec-desk pillars; PB
   collections, relations, workflows, capture. **Delivered as decks / explainers (comms), not raw markdown.**
   Stays on dev-tooling-desk for now; migrates to the desk-standard desk later (co-occurring work), not yet.
4. **R4 · Prototype(s)** — one thin vertical slice (capture → store → view) on one surface.
5. **R5 · Integrate** — iterate across surfaces until it hangs together.

## R1 — decisions (closed 2026-07-20)

1. **Product boundary — grow deskkit.** Evolve desk-standard's deskkit (already PocketBase + MCP + CLI + PM
   module + webapp lane). The toolkit = persona packaging + new surfaces on top; one repo, max reuse.
2. **v1 proof surface — the coding-agent-harness persona (skills + MCP).** Claude Code / opencode assumes the
   desk persona via a skills bundle wired to the existing deskkit MCP server. Most differentiated, least new
   UI. TUI and webapp follow as later surfaces.
3. **v1 data-model scope — the 4-entity spine:** `task · meeting · person · decision` + their relations —
   enough to run the "meet with Tony re: a decision" flow end-to-end. `topic / note / extender` deferred.
4. **KB storage — files mirrored into PB with held diffs (transitional to PB-as-truth).** The on-disk file is
   canonical *for now*; deskkit **ingests and mirrors** it into PB; **agent rewrites occur in PB and are held
   as diffs** (reviewed, then applied back to disk). Rationale (Henry): structured data in PB is better for
   manipulation; on-disk files are a trust comfort until the system earns "everything in the DB." Extends
   deskkit's existing record-original-first / byte-exact-reversible librarian boundary. **Direction of travel:
   PB becomes truth once trust is established.**
5. **Non-negotiables (confirmed):** single-user local-first · single-binary / no-heavy-npm (deskkit ethos) ·
   **MIT/permissive borrow only** · identity-neutral (estate rule) · files-are-truth for derived data and (for
   now) for KB, with PB as the working/staging layer.

## R2 — tech stack + MIT-borrow survey (closed 2026-07-20)

**Determined by R1 (grow deskkit):** Go · embedded PocketBase · MCP server (exists) · a Claude Code plugin as
the persona (skills bundle wired to the MCP). Not re-litigated.

**Open tech decisions:**

- **File ↔ PB mirror + diff-hold engine** — watch/ingest, content hashing, where held diffs live (a
  `revisions` / `pending_diffs` collection), apply-back + restore mechanics. Reuse the librarian's
  record-original / restore core as the base?
- **PB schema for the 4-entity spine** + typed relations (relation fields vs join collections).
- **chat-to-schema capture** — the skill / MCP tool that turns "meet with Tony…" into a spine record.

**Borrow-survey targets (MIT/permissive only):** file↔DB mirror/diff engines · PocketBase-based PM/KB apps ·
entity-graph / backlinks patterns · (later) Go TUI frameworks. Survey complete 2026-07-20 — results below.

### R2 — borrow results + recommended stack (2026-07-20)

Licenses verified against repos. **Headline: no turnkey permissive engine for the file↔DB diff loop — assemble
it from Go primitives; strong architecture references exist.** Recommended stack:

- **Mirror/diff engine (decision #1).** Per the R1 KB model, **PB is the staging layer** — it mirrors the
  on-disk file and holds pending agent edits as diffs. **Fork resolved (Henry, 2026-07-20): git-agnostic** —
  no `go-git`. Engine = **`fsnotify`** (BSD-3, watch disk → re-ingest) + **`sergi/go-diff`** (MIT, *fuzzy*
  patch-apply — apply a held edit onto a file that has drifted, the hard part). History/audit lives in a PB
  `revisions` collection, not git; portable to non-git desks. Architecture refs: **Dolt** (Apache-2.0, Go —
  accumulate-then-reconcile) and **TinaCMS** (Apache-2.0 — files=truth, DB=derived+editable, write-back = our
  exact loop). Generalizes deskkit's record-original / byte-exact-reversible boundary.
- **Go + embedded-PB app patterns.** **Beszel** (MIT, 23.6k★, Go — production app embedding PB as a library
  with custom collections/hooks/migrations, single binary) is the reference; **spinspire/pocketbase-sveltekit-
  starter** (MIT) for the scaffold. Confirms "grow deskkit" is sound.
- **Schema / typed relations (decision #2).** PB relation fields, disciplined by **Grist**'s typed
  Reference / Reference-List model (Apache-2.0 — cleanest permissive typed-relation reference); **JustJot**
  (MIT) to mine for a notes schema.
- **chat-to-schema capture (decision #3).** Authored as a skill + MCP tool (our differentiator; no vendorable
  candidate — SilverBullet's markdown-object extraction is a loose reference).
- **Later — TUI surface.** The **Charm** stack (Bubble Tea / Bubbles / Lip Gloss / **Glamour** markdown render,
  all MIT) + **dstask** (MIT, Go, git-backed note-per-task) as UX reference. Banked for the TUI round.
- **Ideas-only (copyleft, no code).** Trilium (AGPL — richest attribute/relation model), Logseq, SiYuan (AGPL).
  Study, don't vendor.

## R3 — inputs + crew (2026-07-20)

- **Planes confirmed** as **input · activity · output** (Henry's naming, cleaner than knowledge/work/output).
- **v1 project-type focus: software + research** (answers R3 q3). Design the activity + output planes for those two.
- **Spine to be beefed up** beyond `task/meeting/person/decision` (Henry: too weak).
- **Input plane mined from `_headcase`** — an Obsidian knowledge-work OS at
  `/Users/henry/Developer/functionform-headcase/packages/obsidian/data/_headcase`: `schema.yaml`, four agent
  personas (pm/product/engineering/general), and ~22 SOP document templates (analysis · decision · roadmap ·
  research-synthesis · engineering-spec · feature-spec · technical-design · test-plan · raid · retro ·
  project-brief · one-pager · sow · runbook · postmortem · weekly-checkin · release-notes · user-journey ·
  ux-spec · product-spec · meeting-notes · daily-note).
- **q4 (missing elements)** left open by Henry.
- **Crew (foreman, standard):** mine `_headcase` (scouts) → design each plane for software+research (builders)
  → session synthesizes the proposal → adversarial completeness review (can it run a real software / research
  project end-to-end?). Deliverable: an element proposal across the three planes + a beefed v1 spine.
- **Status (2026-07-20):** mining done (3 scouts); proposal drafted → `spec-element-model.md`; adversarial
  review — **research back** (7 substantive gaps; research plane needs the most work — evidence triple,
  experiment element, un-defer research-synthesis), **software back** too (build-phase gaps: no code/PR or bug
  entity, wrong §3 decomposition chain, spine omits every spec type). Both reviews captured in
  `spec-element-model.md` § "R3 review findings". **Progress deck delivered**
  (`_meta/briefings/2026-07-20-desk-platform-progress/`) presenting the 5 major IA changes + 4 open questions as
  an owner approval gate. **Awaiting Henry's approval** (standing directive: he signs off major IA before it's
  finalized) before folding the reviews into the final model → R4 prototype.

## Provenance

Vision stated by Henry 2026-07-20. Store decision + two-data-natures: `../extenders-estate/system-cohesion-and-datamodel.md`.
Related rulings: desk-standard 1.0 `_structure/decisions/0021` (PM default-on, webapp lane, cohesion+value eval);
PocketBase backend `0009`/`0015`.

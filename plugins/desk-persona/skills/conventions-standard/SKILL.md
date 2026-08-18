---
name: conventions-standard
description: >-
  Reference for the executive-desk conventions standard v1 (K1–K29): the frontmatter
  contract, the two status vocabularies, decision-record rules (append-only, supersession
  vs correction, collision repair), the naming regime, the closed instrument set, the
  secrets home, the decision spine, and the storage-model taxonomy. Consult when
  authoring, reviewing, structuring, or setting up a desk, or when checking a desk
  against the standard (the adherence checklist at the end). This is reference content,
  not an executable checker.
---

# Executive-desk conventions standard v1

An **executive desk** is a planning surface that oversees one or more projects and keeps the
planning clutter — decisions, analyses, status, communication drafts — out of the projects it
watches. The defining move is a separation of writing rights: **the desk drafts, and the owner
applies every external write** (commits, issues, board changes, communications). Not every
overseen project is software; a desk can oversee a portfolio of work of any kind.

This skill is the consultable rule set. It states *what the standard requires*; the
`desk-setup` skill applies it to a new or existing desk, and the `harvest-loop` skill evolves
it. Mechanical enforcement (scan, flag, auto-fix, restore) is **not** this skill's job — it
belongs to the deskkit, reached through the MCP tools. This skill defines the rules
those tools enforce.

## The critical rule (K1)

The desk drafts; the owner applies **every** external write — commits, issues, board changes,
communications. Everything else in the standard exists to protect this separation. A desk never
writes production code and never merges to a mainline branch.

## Quick reference — the convergent core and its refinements (K1–K29)

| # | Rule |
|---|---|
| K1 | Desk drafts; owner applies every external write. |
| K2 | The board/issue tracker is the single work-state truth; the desk reads live state, never snapshots or forks a parallel backlog. Doc-vs-reality gaps become tracked register rows, not restated counts. |
| K3 | A frontmatter contract governs every `.md` (see "Frontmatter contract" below). Enforced by schema v1. |
| K4 | Two status vocabularies — one for docs, one for decisions (see "Status vocabularies"). Status lives in frontmatter, never free text. |
| K5 | Decision records: one file per decision, `NNNN-short-title.md`, append-only, indexed newest-first; graduate to a code repo's ADR when a decision implies a code change (see "Decision records"). |
| K6 | Naming: kebab-case by topic/type, never by sequence (no `v2`); no date prefixes on living docs; dated names only mark point-in-time artifacts. |
| K7 | `analyses/` (working-out) and `deliverables/` (finished, rendered, dated folders) are strictly separate. Markdown is canonical; rendered outputs are disposable and regenerated, never hand-edited. |
| K8 | The entry file orients, it does not contain (soft cap ~250 lines): role, boundaries, the critical rule, a numbered read order. The handoff is the session bridge; run the clearability test ("what do I know that isn't written down?" → "nothing") at every boundary. |
| K9 | Start flat / plan-don't-stub: add structure only when a live thread needs it. |
| K10 | Sensitive material has exactly one untracked home; never in tracked files, memory, or communications (refined by K28–K29). |
| K11 | No timelines: express work as deliverables, criteria, and parallelism; owner deadlines are constraints, never schedules. |
| K12 | One concept per file; one owner per fact. When a fact is owned elsewhere, the local copy collapses to a pointer. |
| K13 | A desk is not a code repo: no build/test gate; a connected repo's own agent-instruction files govern code only and never cross-apply to the desk. |
| K14 | Cross-desk transfer happens via communication packages delivered into the receiving repo's inbox; the handoff is intra-desk only, never a delivery channel (see "Cross-desk transfer"). |
| K15 | Five-layer skeleton: **entry** → **working desk** (`_meta/`) → **decision spine** (`_structure/`) → **workstreams** (one top-level dir each) → **output shelves** (`deliverables/`, `communications/`). Structure grows into these layers, not sideways into ad-hoc top-level dirs. |
| K16 | Closed ALL-CAPS instrument set (see "The instrument set"). |
| K17 | Every absorbed workstream gets a scope doc (`_structure/<workstream>-scope.md`) stating what it is, what it is not, and where its truth lives. |
| K18 | Per-folder naming regimes may differ but are declared where they apply (e.g. communications as `YYYY-MM-DD-sender-topic.md`). |
| K19 | De-noising is trigger-based, never scheduled: cleanup fires on an event (a workstream closes, a rule change touches 2+ files, a duplicate is noticed) and follows archive-don't-accumulate. |
| K20 | Two decision-repair mechanisms, chosen by what changed (see "Decision records → repair"). |
| K21 | Self-maintenance through one file, `_meta/improvement-log.md`: the maintenance backlog, a pass log, and the friction ledger. |
| K22 | Communications lifecycle with an index ledger: markdown is canonical in `communications/outbound/`; inbound items are dated; `communications/INDEX.md` tracks status with audience-register discipline. |
| K23 | Greenfield setup is an ordered runbook (owned by the `desk-setup` skill). |
| K24 | Brownfield adoption is a distinct mode with its own status track and a user-approval gate before any write (owned by the `desk-setup` skill). Retrofitting frozen source material into conformance is a supported disposition (vs. declaring it exempt), governed by ruled field-preservation defaults (see "Brownfield retrofit defaults"). |
| K25 | Template assets are deliberately standard-free: the copyable skeleton ships without the standard's prose baked in. |
| K26 | App/project-instructions are a thin pointer only; substance lives in version-visible files. |
| K27 | The harvest loop is the lifecycle mechanism that evolves the standard (owned by the `harvest-loop` skill). |
| K28 | Storage-model taxonomy: a desk and a code repo live in physically different places with different backup and exposure properties (see "Storage model"). |
| K29 | Exactly one never-committed secrets folder per project, one name across the estate; on the desk surface it stays cloud-synced even though it is git-excluded (see "Secrets home"). |

The rest of this skill expands the rules that require a precise, ruled answer.

## Frontmatter contract (K3, ruled by schema v1)

Every desk `.md` carries frontmatter validated by **schema v1** (the shared `schema/` in this
repo is the enforcement substrate — do not restate the full field/type list here; it lives
there under one owner, K12). The universal field set is:

```yaml
type: <one of the schema's document types>
status: <from the matching status vocabulary — see below>
created: YYYY-MM-DD
updated: YYYY-MM-DD
tags: [ ... ]
synopsis: "One-sentence scannable abstract."
```

- `synopsis` is **required on the desk surface** and optional elsewhere. The document's `# H1`
  is its title and the `synopsis` its abstract — there is **no** `title` or `summary` field
  (they were dropped from the written contract because no real desk doc carried them).
- **Quote any frontmatter scalar that contains `: ` (colon-space)** — an unquoted
  `synopsis: foo: bar` breaks strict YAML parsing. This applies to every frontmatter file,
  including profiles.
- Keys the schema does not define do not become new top-level fields; unplanned metadata goes
  under the schema's open-ended escape (`custom`), so the schema stays the contract.

**Frontmatter-exemption classes.** A small allowlist is exempt from the contract: the entry
file (`CLAUDE.md`), `_meta/HANDOFF.md`, `_meta/README.md`, a **directory-index `README.md`
inside an entity dir** (`_structure/decisions/`, tasks, analyses, journal, `_knowledge/` — the
index orients the folder, it is not an entity record; the deskkit enforces this by basename, so
patrol's R1/R4 never fire on it), template files (the standard-free scaffold), briefing sidecar
files (e.g. `sources.md`), and drafted GitHub issue bodies (the desk drafts these for the owner
to file; they follow the issue tracker's shape, not the desk contract).

## Status vocabularies (K4)

Status is a frontmatter field, never prose. Two families:

- **Desk documents:** `draft → in-review → active → decided → pushed → superseded`, plus
  `archived`. Use the value that names the doc's real lifecycle state.
- **Decisions:** `proposed | accepted | rejected | superseded`. A decision also carries
  `decided_by`; `governs` and any `affects_workstreams`/`Affects` linkage are optional.

A status value valid on the desk surface (e.g. `active`) is not necessarily valid on another
surface (e.g. a knowledge vault). When a document moves between surfaces, its status is
re-stamped to the destination surface's vocabulary — the tooling does this, not free-hand.

## Brownfield retrofit defaults (K24)

Frozen or graduated source material entering a desk under brownfield adoption (K24) has two
sanctioned dispositions, chosen at the approval gate: **declare it exempt** (mark it frozen-source
and skip the contract) or **retrofit it into conformance** (author the frontmatter contract, fix
naming). **Retrofit-over-exempt is a supported disposition** — a desk may choose to bring real
source material under the standard rather than fence it off, and when it does, these defaults make
the retrofit reproducible instead of a per-adopter judgment call.

**The preservation principle.** A retrofit that overwrites a frontmatter field which carried a
real source value **preserves the original under `custom.original_<field>`** — symmetric across
every overwritten field, so nothing an adopter classified by hand is silently lost. `custom` is
the schema's open-ended escape (K3), so preserved originals never become new top-level fields.
The three ruled defaults are instances of this one principle:

- **`type` — overwrite, preserve the original.** A source `type` that conflicts with the retrofit's
  required type is overwritten to the required value; the original is recorded under
  `custom.original_type` (symmetric with `custom.original_status`, where a non-vocabulary source
  `status` is preserved when the retrofit re-stamps it).
- **`created` — derive from the source's own date fields, in a fixed order.** Take the first that
  is present: explicit frontmatter `date_created` → frontmatter `last_updated` → a prose
  "Last Updated" line → the file's modification time. The consulted source date fields are **also**
  preserved under `custom` for audit, not just consumed.
- **`tags` — merge and dedupe, never replace.** The required tags are merged with the source's own
  tags and de-duplicated, with the required tags **always present**. Real source classification
  (a domain tag, a severity marker) survives the retrofit rather than being overwritten by the
  required set alone.

## Decision records (K5, K20)

- **One file per decision**, `NNNN-short-title.md` (zero-padded, kebab-case title), living on
  the **decision spine** (see below). Append-only, indexed newest-first in the spine's index.
- A decision that implies a code or infrastructure change **graduates** to the owning repo's
  ADR plus an issue, linked from an `Affects` section; the desk record stays as the
  portfolio-level trace. The desk record and the code ADR are different surfaces (K13) — the
  desk never re-specifies what the code ADR owns.

**Repair — pick the mechanism by what changed (K20):**

- The *decision* changed → **supersession.** Write a new record; set the old record's status to
  `superseded`; leave its text in place for provenance.
- A *factual premise* the decision rested on was wrong → **dated in-place correction.** Add a
  `> Correction (YYYY-MM-DD): …` callout, strike the false sentence inline, and set the status
  to read "corrected". A falsified claim is never left readable as current truth.

**Number-collision repair (ruled: always renumber the later-filed doc).** If two records share a
number, **renumber the later-filed doc** to the next free number, and leave a **dated redirect
note** at the old identity so any pointer that already cited it still resolves. (This is the
cleanest final state; the redirect note pays for the append-only tradeoff.) The collision check
is mechanically decidable and belongs to the enforcement tooling, not to hand inspection.

## Naming and the instrument set (K6, K16)

- **Kebab-case by topic or type**, never by sequence. No `v2` filenames — git is the version
  history; revise in place and archive superseded material with a pointer. No date prefixes on
  living docs; dated names (`YYYY-MM-DD-subject`) are reserved for point-in-time artifacts.
- **The closed ALL-CAPS instrument set.** Exactly these filenames may be ALL-CAPS, and each owns
  one cross-cutting instrument:

  `README.md · CLAUDE.md · AGENTS.md · MEMORY.md · HANDOFF.md · STATUS.md · TRACKER.md · INDEX.md`

  Everything else is lowercase-kebab. The set is **closed**: adding an instrument means editing
  this rule, which keeps proliferation visible. A file that is ALL-CAPS but not in this set is a
  violation (a mechanically checkable rule).

## Secrets home (K10, K29)

Exactly one never-committed folder per project holds secrets and live-ops material:
**`_meta/secrets/`**. The folder itself is tracked as keep-only (a `.gitkeep`); its **contents
are never committed**. One name across the whole estate.

- Never put secret values in tracked files, memory, or communications — only in `_meta/secrets/`
  or in environment variables referenced indirectly by name.
- Reserve the words "secrets"/"operations" for this untracked folder; tracked current-state
  reference material lives elsewhere (a workstream dir or `_knowledge/`), so the two never
  collide.

## Storage model (K28)

Git-tracking status is **not** the same as backup status or exposure status; the standard
reasons about them separately per surface:

- **A desk lives on a cloud drive:** cloud-backed-up, local git, **no** remote host. On this
  surface `_meta/secrets/` is git-excluded **yet still cloud-synced** — "git-excluded" protects
  it from history and from any (non-existent) remote, but the cloud drive still backs it up.
- **A code repo lives on local disk:** it has a remote host and no cloud backup, so its
  equivalent folder is neither committed nor cloud-backed.

Because not every overseen project is software, a desk may oversee non-code work and still
follow the desk surface's rules.

## The decision spine and the five-layer skeleton (K15, K3-of-structure)

- **Decision spine = `_structure/decisions/`.** This is where a desk's constitution lives,
  separated from the churny working desk (`_meta/`). A small/flat desk may collapse the spine
  into `_meta/decisions/`. A code repo keeps its ADRs at `docs/decisions/` **deliberately** —
  a different standard for a different surface (K13), not an inconsistency to reconcile.
- **Five layers** (K15): entry (`CLAUDE.md`/`README.md`) → working desk (`_meta/`) → decision
  spine (`_structure/`) → workstreams (one top-level dir each) → output shelves
  (`deliverables/`, `communications/`). Grow into these layers, not sideways.

## Cross-desk transfer (K14)

A desk sends work to a code repo as a **communication package** dropped into that repo's inbox
(conventionally `_meta/plans/inbox/`). The **package format is owned by the companion code-repo
standard, not by this desk standard** — reference it, never re-specify it (K12). The handoff
file is intra-desk only and is never a cross-desk delivery channel.

## What the desk deliberately does not do

- No work-state tracking, backlog, or status/count snapshots — the board is the single truth
  (K2). Docs point at it; they never restate counts.
- No build/test gate, no cross-applied code-repo instructions (K13).
- No timelines (K11).

---

# Adherence — the conformance checklist (Item D, spec only)

This section defines **what conformance means** as testable rule statements, plus a
**non-mutating** procedure an agent (or human) runs to report deltas. It ships **no executable
checker and no auto-fix**: mechanical enforcement — scan, flag, `--fix`, restore, numbering-
collision detection, retrieval — is the **deskkit's** job (reached through the MCP
tools), specified here so the two never drift and never own the same fact twice (K12). Running
this checklist changes nothing; it produces a report.

## Testable rule statements

Each is a claim a checker can decide against a desk tree:

1. Every non-exempt `.md` carries `type, status, created, updated, tags`, and — on the desk
   surface — `synopsis`. (Exempt: the frontmatter-exemption classes listed above.)
2. Every frontmatter scalar containing `: ` is quoted.
3. `status` values come from the matching vocabulary (desk-doc family or decision family); no
   free-text status.
4. No filename is ALL-CAPS unless it is in the closed instrument set.
5. Decision records live under `_structure/decisions/` (or the allowed `_meta/decisions/`
   collapse), are named `NNNN-short-title.md`, and no two share a number.
6. Exactly one secrets home exists (`_meta/secrets/`), it is keep-only, and no secret value
   appears in any tracked file, memory, or communication.
7. Filenames are kebab-case; no `v2`-style sequence names; date prefixes appear only on
   point-in-time artifacts.
8. No document restates board counts or work-state (K2); plans state deliverables, criteria,
   and parallelism, never timelines (K11).
9. The entry file stays within its orienting role (soft cap ~250 lines) and carries a numbered
   read order (K8).

## The check procedure (read-only)

1. Enumerate the desk's `.md` files, excluding the exemption classes and any dated frozen
   snapshots (dated briefing/research folders are point-in-time and are not re-checked).
2. For each, evaluate rules 1–4 and 7 above; record any file that fails, with the specific rule.
3. Scan filenames across the tree for rule 4 (ALL-CAPS outside the set) and rule 7 (naming).
4. Inspect `_structure/decisions/` for rule 5 (numbering, collisions, location).
5. Confirm rule 6 (one keep-only `_meta/secrets/`; no tracked secret values).
6. Spot-check plans and status docs for rules 8–9.
7. **Report deltas only — never edit.** Present each finding as `path — rule — what is wrong`.
   Remediation is proposed, not applied: applying fixes is the deskkit's supervised,
   record-original-first job, not this checklist's.

The checklist is trigger-based, never scheduled (K19): run it when a desk is being set up,
adopted, audited, or when a rule change lands — not on a calendar.

---
name: brownfield-adoption
description: >-
  The hardened, phase-by-phase runbook for bringing an existing, non-conformant desk
  under the standard. Use when adopting a real pre-existing planning folder — not a blank
  greenfield — and you need the full field-tested procedure behind the K24 brownfield mode:
  lock, read-only inventory, an approved disposition table, staged migration, instrument
  authoring, and a librarian baseline as the final gate. Extends the desk-setup skill's K24
  states; owns the procedure, not the rules (see conventions-standard).
---

# brownfield-adoption

This is the **K24 runbook**, hardened by a real field-test adoption. K24 defines brownfield
mode — the status track `untouched → inventoried → approved → reconciled`, the mandatory
user-approval gate, and the never-delete-source invariant. This skill does not redefine any of
that: the `desk-setup` skill owns the mode definition and the greenfield scaffold, and the
`conventions-standard` skill owns every rule cited here as `(K##)`. What this skill adds is the
field-tested *procedure* — the ordering, the traps, and the verification — that turns the K24
states into a repeatable move.

## Parameters and prerequisites

Settle these before Phase 1; they bind every phase.

- **Subject desk path** — always **absolute**. The adopting agent may be **running inside the
  subject desk**, so relative paths are ambiguous and dangerous; use absolute paths throughout.
  **Serialize all writers**: never run parallel agents mutating the desk during the move — one
  writer at a time, or the lock and the inventory stop meaning anything.
- **Staging path** — a **parameter, not a constant** (a sibling folder alongside the desk is the
  convention, but the caller sets it). **WARN:** if the desk lives inside a cloud-synced tree
  (iCloud, Dropbox), a bulk move churns the sync engine for the whole tree — suggest a
  **non-synced staging path** for a large desk so the migration does not thrash sync.
- **This plugin installed** from its marketplace, for the `template_render` tool and the
  greenfield runbook. If it is not installable in the environment, the skill **degrades to a
  docs-driven mode**: follow the same phases by hand against the `conventions-standard` docs,
  rendering placeholders manually instead of via the tool.
- **deskkit installed**, for the final baseline. The recommended path is **downloading
  the published release artifact and verifying its checksum** before first use. Release-built binaries
  report their release version via `--version`; a binary that prints `dev` was built from
  source without the version stamp — pin such a build from its source commit, not `--version`.
- **The desk-pm MCP tool surface needs `deskkit` ON PATH and a fresh session.** The PM tools are
  served by `deskkit mcp-serve` (launched by the desk-pm plugin's `.mcp.json`), so they exist
  **only** once `deskkit` is **installed AND resolvable on the session's PATH** — a GUI-launched
  host may not see `~/.local/bin`, in which case the plain `deskkit` command is **silently dropped**
  — **and the Claude Code session has been restarted**. MCP servers wire at session start: a
  mid-session install does **not** mount the surface, so the PM tools stay absent until the next
  session. If they are missing, confirm `command -v deskkit` resolves, then **start a fresh
  session** (or wire `deskkit mcp-serve` as a stdio MCP server by its absolute path in your own
  settings) rather than assuming the module is broken. When the surface does mount, `mcp-serve`
  now prints a one-line signal to stderr (visible in the host's MCP server log) naming the tools it
  exposed — its **absence** is the diagnostic.

## Phase runbook

The phases map onto the K24 status track. The **librarian baseline is deliberately the FINAL
gate** — it grades the *finished* state and is never run mid-migration.

### 1. Lock (before anything else)

**Zip the intact desk, INCLUDING `.git/`, before moving or writing a single file.** Then
**verify the zip** — list its entries and sanity-check the count against the disk — so the lock
exists even if the move is later interrupted. This is the non-negotiable first step; every later
phase is re-runnable from this archive.

### 2. Inventory — `untouched → inventoried` (read-only)

Survey without writing anything. The disposition table's spine **must come from a literal disk
enumeration** (`ls -A` at the desk root), then be annotated — **never authored from memory, a
handoff, or a scout report**. A real field-test table, built from a handoff, silently missed a
top-level directory exactly this way; only the disk enumeration catches everything.

Then run `git status --ignored`. Gitignored content splits two ways, and **every ignored item
gets an explicit keep/drop call**:

- **Load-bearing** — operational skill dirs, local settings that wire memory and tooling. A naive
  tracked-files-only migration silently loses these.
- **Litter** — `.DS_Store`, logs, caches. Dropped, but named as dropped.

Build the desk's real status from **repo evidence** (git log, the tracker, an existing handoff),
never from stale docs — if a status doc disagrees with reality, reality wins (K24 invariant).

### 3. Disposition table as a file

Write **one row per enumerated top-level item**: keep-as-is / move / rename / archive / drop, plus
the target location under the standard. Ship this as a file the user can read — the
`assets/adoption-plan.md` template gives the table skeleton plus a slot for the gate questions.

**Two rows are mandatory and explicit, never buried:**

- **(a) The `.gitignore` semantics flip.** Adopting the template moves the desk from
  *ignore-by-default-with-negations* to *track-by-default-except-secrets*. That is a real
  storage-semantics change — it must be **its own approved row**, never folded into a vague
  "reconcile gitignore".
- **(b) Ratified decision records are adopted as-is.** A consolidated, ratified decisions record
  is **never retroactively exploded** into per-decision files — respect what was already ruled
  (K24 invariant). This is its own row so the decision is explicit.

### 4. GATE — `inventoried → approved`

**Mandatory user approval before ANY write, move, or delete (K24).** Package the gate exactly:
the **disposition table as a file** the user can read, plus **2–3 explicit residual judgment
questions** — never a wall of prose. **Always include** the fresh-git-history-vs-preserve-history
question. No file is touched until the user approves.

### 5. Init — `approved → reconciled` begins

Move the desk's contents to the staging path, then **init the fresh skeleton in the now-empty
desk** via the `desk-setup` skill's greenfield runbook — the template scaffold plus profile-driven
`template_render` rendering. The desk is now a clean standard skeleton; the old content is staged.

### 6. Migrate

Apply the **approved dispositions** from staging into the new structure, row by row. Every step
stays **re-runnable from the locked zip**, and **the source stays intact** — the user deletes
staging and the zip when satisfied; **this skill never deletes them** (K24).

### 7. Author instruments

Write `_knowledge/profile.yaml` and **validate it against the shipped schema before proceeding**.
Three known authoring traps:

- **(a) The repos section models a default / by-role / shorthand shape.** A desk with several
  first-class repos overflows that shape — the extras belong in the **`custom` block**, not
  forced into `by_role`.
- **(b) `secrets_ref.llm_api_key` expects an env-var NAME**, while the desk paths key for secrets
  takes a **directory**. The near-identical names invite mis-mapping — a name is not a path.
- **(c) The models section wants a literal model id.** A tiering *doctrine* ("smart decides, cheap
  executes") has **no slot** there and belongs in the **`custom` block**.

### 8. Librarian baseline (the final gate)

With `DESK_ROOT` and `DESK_NAME` set: `migrate up` → `sweep` → `patrol` into a **FRESH store**.
**Record the patrol run id and the store path** in the adoption record so a later re-baseline is
comparable. These commands are **deterministic and need no LLM key**.

Triage findings three ways:

- **Template / upstream noise** — report upstream, do not fix locally.
- **Genuine pre-existing debt** — real, but not this migration's fault.
- **Judgment calls** — flagged for a human.

**Supervised fixes are a SEPARATE sitting**, bounded by the librarian's enforcement boundary:
**mechanical-only** (missing frontmatter, journal naming, type-vs-dir), **record-original-first**,
**templates-only content**. Everything else — graduated-doc collapse, staleness, decision-status
checks — **stays flagged for a human**. **Single-writer rule:** never run one-shot commands while the
librarian's serve/GUI is up. **Known behavior:** patrol does **not** resolve an open finding that
merely stops firing (a rule change, a hand-fix, a deletion) — so after upgrading the librarian,
**re-baseline into a fresh store** rather than trusting the old store's open count.

### 9. Take stock

The finish line is **"git status clean + gitignore covers the litter"**, **not** "the directory is
pristine" — the adopting agent's own tooling recreates machine-local litter (logs, telemetry)
during the migration, and that is expected. Leave the old folder and the zip **for the user to
delete** (K24). Hand the result to the `conventions-standard` skill's adherence checklist to
confirm conformance, mirroring how `desk-setup` closes.

## What stays manual

The skill guides these; it does not automate them. Judgment migrations — where a piece of content
belongs, what to archive — are the adopter's call. Non-mechanical findings stay flagged, never
auto-fixed. Decision-record curation is human work. Honest scope: this runbook makes the move
safe and repeatable, not hands-free.

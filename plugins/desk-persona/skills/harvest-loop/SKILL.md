---
name: harvest-loop
description: >-
  The lifecycle procedure that evolves the conventions standard and this plugin from real
  usage. Use when running a harvest pass — reading the friction ledgers desks accumulate,
  folding transferable lessons into a versioned plugin/standard revision, and marking the
  absorbed entries resolved. Also defines the plugin's version + changelog discipline and
  ships the per-desk friction-ledger template (assets/improvement-log.md).
---

# harvest-loop

This is how the standard improves without ambient drift. Each desk keeps a **friction ledger**
(its `_meta/improvement-log.md`, per K21); periodically a harvest pass reads those ledgers
across every live desk, folds the transferable lessons into a **versioned revision** of the
standard/plugin, and marks the absorbed entries resolved. The loop is trigger-based, never
scheduled (K19) — run it when enough friction has accumulated or a theme recurs across desks.

## The friction ledger (what gets harvested)

Every desk's `_meta/improvement-log.md` carries three ledgers in one file:

- **Maintenance backlog (M-nn)** — desk-local upkeep items. Mostly **not** harvested; these are
  that desk's own to-dos.
- **Pass log** — a dated record of what each maintenance/de-noising pass did.
- **Instruction-friction ledger (INS-nn)** — the harvest target: places where a convention
  rubbed against real work. Each entry names what rubbed and what rule change would fix it.

The template for this file ships at `assets/improvement-log.md`; the `desk-setup` skill seeds a
copy into every new desk.

## The harvest procedure

1. **Collect.** Read the INS-nn ledger from every live desk's `_meta/improvement-log.md`. (When
   the deskkit is available, it can gather these across desks through the MCP tools;
   otherwise read them directly.)
2. **Triage — transferable vs desk-local.** An entry is **transferable** when its lesson
   generalizes to any desk (a convention is ambiguous, missing, or wrong). It is **desk-local**
   when it only concerns one deployment's data or history — those stay as M-nn items and are not
   harvested.
3. **Fold into a revision.** Draft one versioned revision of the standard/plugin that absorbs the
   transferable entries. The revision **cites the absorbed entry IDs** (e.g. "absorbs INS-04,
   INS-07") so the provenance from friction to rule change is traceable.
4. **Mark absorbed entries resolved.** In each source ledger, mark the harvested entry resolved
   with a **pointer back to the revision** that absorbed it. Absorbed entries are batons, not
   journal lines — resolve them, don't accumulate them.
5. **Version + changelog.** Bump the plugin's single authoritative version field and add a
   changelog entry (below). A revision that changes a rule and does not bump the version is a
   bug — the version is how consumers know the standard moved.

A lesson that recurs across **multiple** desks is the strongest harvest signal; a single-desk
one-off is more often a desk-local quirk than a standard defect — weigh accordingly before
changing a rule everyone depends on.

## Version + changelog discipline

The plugin carries **exactly one** authoritative version field (in its manifest) and a changelog
— never a version that drifts from the revision notes. Each harvest revision:

- increments the version,
- adds a newest-first changelog entry naming **what rule changed** and **which INS-nn entries it
  absorbed**,
- and, when a rule change is breaking for existing desks, says so plainly so adopters know a
  migration is implied.

The changelog is the standard's audit trail: a reader can trace any current rule back to the
friction that shaped it.

## Boundary

The harvest loop evolves the **standard and this plugin** — it does not touch desk work-state
(the board owns that, K2) and does not mutate desk files itself. Applying a rule change to an
existing desk (re-scaffolding, migrating) is the `desk-setup` skill's and the deskkit's
job; harvest-loop produces the revised rules, not the desk edits.

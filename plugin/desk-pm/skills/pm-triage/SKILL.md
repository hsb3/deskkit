---
name: pm-triage
description: >-
  File new work into the PM graph and wire it up: create items with the right court, type, and
  priority; link typed dependencies (blocks / is-blocked-by / relates-to) with unblock-at and
  cascade; set and clear the blocked side-state; and reprioritize a court's queue. Use when
  intaking new work, grooming the backlog, recording that one item blocks another, marking
  something blocked/unblocked, or reordering what comes next. Triggers on "file this", "add an
  item", "triage the backlog", "this depends on that", "mark it blocked", "reprioritize".
---

# pm-triage — intake, dependencies, and priority

Triage turns incoming work into graph items and keeps the graph's edges and ordering honest. It
uses the write side of the PM tool family. All of it writes only the PM store — never desk files,
never a code repo.

## Create an item

**`create_item`** adds a work item; it starts in the `queue` phase. Fields:

| Field | Values / meaning |
|---|---|
| `title` | the item's name |
| `type` | a schema-v1 / kit item type (e.g. `decision`, `task`) — this is what the gate rules key on |
| `court` | who holds it: `owner`, `desk`, `crew`, `vendor`, `external-session` |
| `pointer` | the doc path / issue URL / other locus it tracks (a plain string) |
| `severity` | `low` / `medium` / `high` |
| `priority` | ordinal within its court/queue (lower = sooner) |
| `parent` | a parent item id (omit for a root item) |

CLI: `deskkit pm create --title ... --type ... --court ... [--parent ...] [--pointer ...]
[--severity ...] [--priority N]`.

Pick the **type** deliberately: it determines which document gate applies when the item is later
advanced (see pm-advance-item). Pick the **court** so the item lands in the right queue —
`get_context`'s active list is ordered by `(court, priority)`.

## Link dependencies

**`link_items <from> <to>`** creates a typed edge:

- **`kind`** — `blocks`, `is-blocked-by`, or `relates-to`.
- For a `blocks` edge, also set:
  - **`unblock-at`** — the phase at which the blocker stops blocking: `work`, `review`, or
    `terminal` (e.g. a blocker that clears once it reaches `review`).
  - **`cascade`** — how the block propagates/clears: `auto` (standing — re-applies while the
    condition holds), `auto-reopen` (one-shot resurrection), `manual`, or `permanent`.

CLI: `deskkit pm link <from> <to> --kind blocks --unblock-at review --cascade auto`.

Use `relates-to` for non-blocking association; use `blocks` / `is-blocked-by` to express real
sequencing. Cascade + unblock-at are what let a dependency auto-clear a downstream block when the
upstream item advances far enough — model them, don't hand-manage the blocked flag where a
dependency edge captures the truth.

## Block and unblock

The blocked flag is a **side-state** independent of phase — it holds an item in place without
demoting it.

- **`block_item <id> --reason "..."`** sets it (the reason is audit detail).
- **`unblock_item <id> --reason "..."`** clears it and restores the held phase.

Prefer a dependency edge with `cascade: auto` when the block is *caused by another item*; use a
bare `block_item` only for an external reason with no in-graph blocker (e.g. waiting on an
owner decision that is not itself an item). Both are version-checked — pass the `version` you
read from `get_item`.

## Reprioritize

To reorder a court's queue, **`update_item <id> --priority N`** (lower sorts sooner). Priority is
an ordinal *within* a court/queue, so reprioritize relative to the other items in the same court.
`update_item` is version-checked and edits only first-class fields; it never moves an item's phase
(a `status_label` naming a different phase's label is instead a gated transition request — use
pm-advance-item for that).

## See the queue

**`list_items`** filters the graph: `--phase` (`queue`/`work`/`review`/`terminal`), `--court`,
`--type`, `--blocked` (`true`/`false`), `--parent` (direct children of an id). Use it to find the
un-triaged (`--phase queue`), audit a court, or list what a parent contains before wiring edges.

## Boundaries

Triage never authors the documents that gates require and never writes to a code repo. For
code-repo work the GitHub board stays the single source of truth; a PM item carries a `pointer`
to the issue, not a copy of it. If the write tools are unavailable, the desk is read-only for
agents (`PM_AUTONOMOUS_WRITES=false`) — propose the triage and let the owner apply it via
`deskkit pm ...`.

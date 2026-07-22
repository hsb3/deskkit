---
name: pm-session-open
description: >-
  Open a work session against a desk's PM work graph: call get_context once to pull the
  desk's live working state — what is active, what is blocked and on what, what has gone
  stale, what changed recently — render it as a cold-start briefing, and pick (and claim)
  the next item to work. Use at the start of a session on a desk that has the PM module
  enabled, whenever you would otherwise read a hand-maintained handoff threads index, or
  when asked "what should I work on", "what's the state of the desk", or "brief me".
---

# pm-session-open — the cold-start briefing

The PM module keeps the desk's working state in a store, not a hand-maintained document.
`get_context` is the single call that returns it. This skill replaces reading a flat threads
index: one tool call gives you the same picture, always current.

## Prerequisite — is PM enabled on this desk?

The PM tools exist only when the desk has the PM module enabled (`PM_ENABLED`, or the
`modules.pm.enabled` profile key). On a desk where PM is off, `get_context` is not present and
the CLI `deskkit pm ...` group is unregistered.

- If the PM tools are unavailable, PM is not enabled here. Say so plainly and stop — do not
  fabricate a briefing. Enabling it is a desk-level adoption step (enable the module, run
  `deskkit migrate` so the PM collections and their stable ids are created, then seed
  `desk_config`); that is out of scope for a normal work session.

## The one call

Call **`get_context`** (no arguments needed; it is desk-scoped). The equivalent owner/script
surface is `deskkit pm context` (add `--stalled-days N` to change the stale threshold).

It returns:

| Field | What it is | How to read it |
|---|---|---|
| `active` | non-terminal, non-blocked items, ordered by `(court, priority)` | the workable queue — start here |
| `blocked` | items with `blocked=true`, each with its blocking items resolved | waiting on something; do not pick these |
| `stalled` | non-terminal items untouched longer than the threshold (default 14 days) | candidates to nudge, re-scope, or drop |
| `recent_transitions` | the latest phase changes with actor + timestamp | what moved since you were last here |
| `ancestors` | root→parent chain per item id | where an item sits in the tree |
| `counts` | totals `by_phase` and `by_court` | the shape of the backlog at a glance |

## Render the briefing

Summarize for the user in plain language, grounded entirely in the returned data — never invent
items or status:

1. **Where the desk stands** — the `counts`, one line.
2. **Active queue** — the `active` list, in returned order (court then priority), each with its
   phase and `status_label` and its `pointer` (the doc/issue it tracks) if present.
3. **Blocked** — each blocked item and, from its resolved blocking items, *what* it waits on.
4. **Stalled** — flag anything in `stalled` as needing a decision (nudge / re-scope / drop).
5. **Recently moved** — a couple of lines from `recent_transitions` so the user sees momentum.

## Pick — and claim — the next item

To choose what to work: take the top of `active` (already ordered by court then priority);
prefer the `owner` court, then `desk`, then the rest, unless the user directs otherwise.

If you are going to actually work an item (not just report), **`claim_item`** it first so a
second agent on the same desk never double-works it — the claim carries a TTL (default 30 min,
`PM_CLAIM_TTL`). Release it with **`release_item`** when you stop, or let the TTL lapse.

A live foreign claim is distinct from — and stronger than — a version conflict: it refuses
**every** direct mutation of the item (transition, block, unblock, update) for anyone but the
holder, until the TTL lapses or the holder releases. Claim before you plan to mutate, not after.

Pass your agent id as `actor` (and `delegation_parent` if you are acting under delegation) on
every write tool call — `claim_item` included — so the audit trail and any later claim-refusal
message name who actually holds the item. Unset, writes are attributed to a generic `"agent"`.

- If `claim_item` / `release_item` and the other write tools are absent while the read tools
  are present, the desk has set `PM_AUTONOMOUS_WRITES=false`: you are **read-only** over the
  graph. Brief and recommend, but the owner drives the mutations via `deskkit pm ...`.

## Hand off to the right next skill

- To move an item forward a phase → **pm-advance-item**.
- To file new work, wire dependencies, or reprioritize → **pm-triage**.

## Boundaries

`get_context` reads only the PM store; it writes nothing. For code-repo work the GitHub board
stays the single source of truth — a PM item may carry a `pointer` to an issue URL, but the PM
system never forks a copy of board state.

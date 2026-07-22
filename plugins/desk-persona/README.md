# desk-persona — the platform v1 proof surface

A companion Claude Code plugin (distinct from the `desk-standard` plugin, shared marketplace) that
instantiates the **agent integration contract**
(`docs/agent-integration-contract-v1-spec.md`, ADR 0014, ADR 0015) as a single composed mount:
one `deskkit` MCP server exposing both the librarian and PM tool families together, plus the
agents, skills, and briefing hook that operate them. It proves the contract's general case — a desk
that has *both* modules registered on one mount at once. The PM surfaces (the `pm-operator` agent,
the three `pm-*` skills, the SessionStart hook) folded in from the retired standalone `desk-pm`
bundle (owner ruling 2026-07-21; ADR 0014(a) one composed bundle).

## What it ships

| Artifact | Kind | Encodes |
|---|---|---|
| `librarian-operator` | agent | grounds every claim with `query`, reindexes with `sweep`, flags rule violations with `patrol`, computes record-original-first fixes with `propose_fix`, logs problems/feedback with `record_feedback`; never authors or rewrites prose |
| `pm-operator` | agent | operates the PM work graph end-to-end over the twelve PM tools; never authors gate documents or writes a repo |
| `pm-session-open` | skill | one `get_context` call → render the cold-start briefing, pick and claim the next item |
| `pm-advance-item` | skill | `get_item` → `transition_item`; handle a gate refusal by routing the missing document to the authoring path, then retry |
| `pm-triage` | skill | `create_item`, `link_items`, `block_item`/`unblock_item`, reprioritize, `list_items` — intake and wire the graph |
| `.mcp.json` | MCP wiring | launches `deskkit mcp-serve` with `PM_ENABLED=true` and `MCP_MODULES=librarian,pm`, composing both tool families onto one mount |
| `hooks/session-briefing.sh` | SessionStart hook | injects the `deskkit pm context` cold-start briefing at session start; a silent no-op when PM is off or `deskkit` is absent |

This bundle mounts the `deskkit` Go binary directly — it carries no generated TypeScript server and
is not part of `make package`.

## The composed mount

`MCP_MODULES=librarian,pm` tells the mount to register both tool families rather than one, per
the contract's mount invariant (`docs/agent-integration-contract-v1-spec.md` §4). The result is a
single MCP server exposing 17 tools, zero phantom entries:

- **5 librarian tools** (always-on slice): `query`, `sweep`, `patrol`, `propose_fix`,
  `record_feedback`. `apply_fix` is withheld unless `LIBRARIAN_AUTONOMOUS_WRITES=true`; `restore`
  is never exposed over MCP.
- **12 PM tools**: `get_context`, `list_items`, `get_item`, `create_item`, `update_item`,
  `transition_item`, `block_item`, `unblock_item`, `add_note`, `link_items`, `claim_item`,
  `release_item`.

## Prerequisites

1. **The `deskkit` binary on `PATH`** — build/install it from this repo (`make install` puts it
   in `~/.local/bin`). The `.mcp.json` invokes `deskkit mcp-serve`.
2. **PM enabled + migrated on the desk** — enable the module (`PM_ENABLED` or the
   `modules.pm.enabled` profile key), run `deskkit migrate` so the PM collections and their
   stable ids are created, then seed `desk_config` (gate rules + `status_label` vocabulary).

## The gates

- **`MCP_MODULES`** — which registered tool families the mount surfaces. This bundle sets
  `librarian,pm`; unset or a single value (e.g. `pm`) narrows the mount to just that family.
- **`PM_ENABLED`** — whether the PM tools exist at all. Off → the PM tools and `deskkit pm ...`
  CLI group are absent; the PM skills and `pm-operator` say so and stop.
- **`PM_AUTONOMOUS_WRITES=false`** — withholds the nine PM write tools, leaving only the three
  read tools (`get_context`, `list_items`, `get_item`). The PM skills and `pm-operator` fall back
  to read-only: brief, diagnose, and recommend `deskkit pm ...` commands for the owner to run.
  `transition_item`'s document gate is the real safety either way.
- **`LIBRARIAN_AUTONOMOUS_WRITES`** — gates `apply_fix` specifically; unset or false, the
  librarian side of the mount stays read/flag-only (`query`, `sweep`, `patrol`,
  `record_feedback`).

## Operational notes

- `deskkit mcp-serve` opens the store directly, so it must not run concurrently with
  `deskkit serve` (single-writer store).
- **Identity-neutral by construction**: no artifact hardcodes a person, org, repo, issue, or desk
  name. All desk identity flows through the desk profile and `desk_config`.

## Boundaries

The PM tools write only their own store — never a desk file, never a code repo. For code-repo
work the GitHub board stays the single source of truth; a PM item carries a `pointer` to an issue
URL, not a copy of its state. The librarian tools never author or rewrite prose and never fix a
path on the ignore list; a mechanical fix always records the file's original content
(record-original-first) before it writes.

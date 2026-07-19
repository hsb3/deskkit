# desk-pm — the PM work graph as an agent surface

A complementary Claude Code plugin (distinct from the `desk-standard` plugin, shared
marketplace) that turns a desk's **PM work graph** into skills, an agent, and a session-start
briefing. It is the agent-facing layer over the `deskkit` binary's PM MCP tools; the data and
runtime live in the one binary, feature-gated behind `PM_ENABLED` (spec §6, §2.9).

## What it ships

| Artifact | Kind | Encodes |
|---|---|---|
| `pm-session-open` | skill | one `get_context` call → render the cold-start briefing, pick and claim the next item |
| `pm-advance-item` | skill | `get_item` → `transition_item`; handle a gate refusal by routing the missing document to the authoring path, then retry |
| `pm-triage` | skill | `create_item`, `link_items`, `block/unblock`, reprioritize, `list_items` — intake and wire the graph |
| `pm-operator` | agent | operates the graph end-to-end over the twelve PM tools; never authors gate documents or writes a repo |
| `session-briefing.sh` | SessionStart hook | injects `deskkit pm context` at session start; silent no-op when PM is off or deskkit is absent |
| `.mcp.json` | MCP wiring | launches `deskkit mcp-serve` with `PM_ENABLED=true`, exposing the PM tool family |

The PM tool family (frozen in D4): `get_context`, `list_items`, `get_item`, `create_item`,
`update_item`, `transition_item`, `block_item`, `unblock_item`, `add_note`, `link_items`,
`claim_item`, `release_item`. Owner/script equivalents are the `deskkit pm <sub>` commands.

## Prerequisites

1. **The `deskkit` binary on `PATH`** — build/install it from this repo (`make install` puts it
   in `~/.local/bin`). The `.mcp.json` invokes `deskkit mcp-serve`.
2. **PM enabled + migrated on the desk** — the adoption path (spec §8.1): enable the module
   (`PM_ENABLED` or the `modules.pm.enabled` profile key), run `deskkit migrate` so the PM
   collections and their stable ids are created, then seed `desk_config` (gate rules +
   `status_label` vocabulary). The plugin's `.mcp.json` sets `PM_ENABLED=true` for the
   MCP-serve view, but the store must already carry the PM collections.

## The gated reality

- **PM off** → the PM tools/commands do not exist; the skills say so and stop, and the hook
  no-ops. Nothing here fabricates a briefing.
- **`PM_AUTONOMOUS_WRITES=false`** → the nine write tools are withheld from agents; only the
  three read tools (`get_context`, `list_items`, `get_item`) are exposed. The skills and agent
  fall back to read-only: brief, diagnose, and recommend `deskkit pm ...` commands for the
  owner to apply. `transition_item`'s document gate is the real safety either way (spec §13).

## Operational notes

- `deskkit mcp-serve` and the hook's `deskkit pm context` open the store directly, so they must
  not run concurrently with `deskkit serve` (single-writer SQLite). When serve holds the DB the
  hook errors and no-ops.
- **Identity-neutral by construction** (spec §6.2): no artifact hardcodes a person, org, repo,
  issue, or desk name. All desk identity flows through the desk profile and `desk_config`.

## Boundaries (spec §7)

The PM system writes only its own store — never a desk file, never a code repo. For code-repo
work the GitHub board stays the single source of truth; a PM item carries a `pointer` to an
issue URL, not a copy of its state.

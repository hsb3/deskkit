_The PM module as a daily surface: what the document-gated work graph is, how to turn it on, and
the CLI / MCP / TUI / plugin surfaces over it — every command, tool, and flag as it actually
ships._
Status: active

# The PM work graph — user guide

The **PM module** is `deskkit`'s second capability, alongside the librarian. It is a
document-gated work graph: work items move through a rigid phase machine, and a phase advance is
**refused until the document that phase requires exists and validates** against schema v1. One
binary, one per-desk store, three surfaces (CLI, MCP, TUI) over one engine — plus a complementary
Claude Code plugin (`desk-pm`).

Design and rationale: `pm-system-v1-spec.md` and
`decisions/0008-pm-core-modules-architecture.md`.

## It's off until you enable it

The PM module is **feature-gated OFF by default**. On a fresh desk, `deskkit` is the librarian and
nothing else — no `pm` command, no PM tools, no PM collections in the store. You opt a desk in:

```bash
export PM_ENABLED=true        # or set modules.pm.enabled: true in _knowledge/profile.yaml
deskkit migrate up            # runs the PM migrations, creates the five PM collections
```

Enablement resolves by config layering, env winning: `PM_ENABLED` > profile
`modules.pm.enabled` > default off. With it off, `deskkit pm ...` is just cobra's normal
unknown-command error, and the store carries no PM tables at all — physical omission, not inert
tables. Turning it on is what runs the PM migrations and stamps the module's schema version.

| Env var | Default | Effect |
|---|---|---|
| `PM_ENABLED` | `false` | Turns the whole PM module on for this desk. |
| `PM_AUTONOMOUS_WRITES` | `true` | When `false`, agents get only the three read tools over MCP; the nine write tools are withheld. Owner/script writes via the `deskkit pm` CLI are unaffected. The document gate is the real safety either way. |
| `PM_STALLED_DAYS` | `14` | How many days without a transition marks an item "stalled" in `get_context`. |
| `PM_CLAIM_TTL` | `30m` | How long a claim holds before another actor may treat the item as free. |

The store lives at `$XDG_DATA_HOME/deskkit/<DESK_NAME>/` (fallback
`~/.local/share/deskkit/<DESK_NAME>/`) — the same per-desk store the librarian uses (ADR 0002).
Upgrading from `pocket-librarian` (v0.6.0 or earlier)? A store still at the old
`$XDG_DATA_HOME/pocket-librarian/<DESK_NAME>/` home is moved to the new home automatically on
first startup, with one logged line — no desk loses its store across the rename.

## The model in one screen

- **Phases** (rigid, rank-ordered): `queue → work → review → terminal`. A new item starts at
  `queue`.
- **Legal transitions:** advance (`queue→work`, `work→review`, `review→terminal`), demote
  (`work→queue`, `review→work`), reopen (`terminal→work`). Every other edge is refused by the
  machine before any gate runs.
- **`blocked`** is a side-state, not a phase: it preserves the item's phase and prevents advance
  until cleared.
- **Gates** hang off transitions. A gate demands a document of a given schema-v1 / kit **type**
  (optionally at a required **status**, e.g. `accepted`); the transition is refused, naming
  exactly what is missing, until that document validates. Gate rules are per-desk editable YAML in
  the `desk_config` collection — the shipped defaults are a seed the desk owner re-rules.
- **Court** (who holds the item): `owner`, `desk`, `crew`, `vendor`, `external-session`.
- **`body`** is a dedicated long-form text field on an item (narrative, acceptance criteria,
  inline spec) — distinct from the `pointer` external-doc reference and from `notes`. Set it via
  `create`/`update`; `get_item` returns it (the list/summary shape omits it).
- **Dependencies** are typed edges: `blocks`, `is-blocked-by`, `relates-to`. A `blocks` edge
  carries an `unblock-at` phase (`work` / `review` / `terminal`) and a cascade mode (`auto`,
  `manual`, `auto-reopen`, `permanent`).
- **Concurrency:** every mutation is version-checked (optimistic concurrency); `claim`/`release`
  give an item a TTL'd claim. A live foreign claim is **authoritative over every direct mutation**
  of the item — transition, block, unblock, and update are all refused for a non-holder while the
  claim is live — until it lapses (default 30 min, `PM_CLAIM_TTL`) or the holder releases it; the
  holder's own writes proceed as normal. Cascade/auto-unblock is derived graph state, not a direct
  call, so it is unaffected by claims. See [ADR 0020](decisions/0020-pm-claim-semantics.md).
- **Audit:** every transition appends an immutable row; a refusal lands as a `gate_refused` audit
  entry. Every write records who acted via optional `actor`/`actor_kind`/`delegation_parent`
  fields (unset → actor `agent`, kind `agent`); the CLI instead defaults `--actor` to `$USER`,
  kind `human`.

## Which transitions gate

Gates are **transition-specific, not universal**: a transition gates only when the desk's
`desk_config` rules name it, keyed by item **type** and the specific **edge**. The shipped
default ruleset (a seed — a desk re-rules it) gates exactly two edges:

| Item type | Edge | Requires |
|---|---|---|
| `decision` | `review→terminal` | a `decision` document, status `accepted`, resolved at the item's `pointer` |
| `task` | `work→review` | a `task` document, status `active`, resolved at the item's `pointer` |

**Every other transition is ungated by default — including `queue→work`.** Demote
(`work→queue`, `review→work`) and reopen (`terminal→work`) gate only if the desk's config
explicitly binds a rule to them.

A gated transition demands a document that **resolves** at the required pointer (the item's own
`pointer` by default, or a `note:<key>`), is of the required **doctype**, is at the required
**status**, and carries **valid frontmatter**: every schema-v1 universal key present — `type`,
`status`, `created`, `updated`, `tags` (`status` optional on lightweight types) — plus the
doctype's own required fields (`schema/doctypes.yaml`). `updated` is a universal key like the
rest: a gated document missing it is refused exactly as one missing a doctype-specific field.

**Worked example.** A `task` item's `queue→work` transition is **ungated** — it always succeeds
(subject only to the machine and any live claim). The same item's `work→review` transition **is**
gated: it is refused until a `task` document exists at the item's pointer, validates (including
carrying `updated`), and is at status `active`.

Authoritative version: [`pm-system-v1-spec.md` §4](pm-system-v1-spec.md#4-gates--the-spine-r3).

## The `pm` CLI — the owner / script surface

Registered only when the module is enabled. Every subcommand is a thin adapter over the same
engine the MCP server and TUI call; **raw JSON on stdout is the default** (the machine contract),
and a gate/engine refusal prints its refusal line, never a usage dump. Audit identity defaults to
`--actor $USER` with actor kind `human`.

| Command | What it does | Key flags |
|---|---|---|
| `pm context` | Single-call cold-start briefing: active, blocked, stalled, recent transitions | `--stalled-days N` |
| `pm list` | Filtered work-graph query | `--phase` `--court` `--type` `--blocked true\|false` `--parent <id>` |
| `pm get <id>` | One item with notes, dependencies, transitions, ancestors | — |
| `pm create` | Add a work item (starts at `queue`) | `--title` (required) `--type` `--parent` `--court` `--pointer` `--body` `--severity low\|medium\|high` `--priority N` |
| `pm update <id>` | Edit first-class fields (empty flag = unchanged; `--priority 0` also = unchanged — `0` is the zero-value sentinel, not a settable priority) | `--title` `--type` `--court` `--pointer` `--body` `--severity` `--priority` `--properties <json>` `--status-label` `--version N` |
| `pm transition <id>` | Request a phase transition; gates may refuse | `--to queue\|work\|review\|terminal` (required) `--version N` |
| `pm block <id>` | Set the blocked side-state (preserves the phase) | `--reason` `--version N` |
| `pm unblock <id>` | Clear the blocked side-state | `--reason` `--version N` |
| `pm note <id>` | Attach a phase-scoped keyed note | `--key` (required) `--body` (required) |
| `pm link <from> <to>` | Create a typed dependency edge | `--kind blocks\|is-blocked-by\|relates-to` (required) `--unblock-at work\|review\|terminal` `--cascade auto\|manual\|auto-reopen\|permanent` |
| `pm claim <id>` | Claim an item with a TTL | `--version N` |
| `pm release <id>` | Release an item's claim | `--version N` |

`--version` is the optimistic-concurrency token you read. Omitting it on a mutating command makes
the CLI read the item's current version first (a supervised-operator convenience, done
desk-scoped); scripts that want the strict check pass `--version` explicitly and get a refusal on
mismatch.

Example — create, wire a dependency, try to advance, satisfy the gate, advance:

```bash
deskkit pm create --title "Ship the widget" --type task --court crew
# → {"id":"...","phase":"queue","version":1, ...}
deskkit pm transition <id> --to work        # refused if `work` gates a missing document — names it
deskkit pm context                          # what's active / blocked / stalled right now
```

## The MCP tools — the agent surface

`deskkit mcp-serve` exposes the tool core over stdio MCP. With the PM module enabled, the **twelve
PM tools** join the librarian's tools on the one server:

`get_context`, `list_items`, `get_item`, `create_item`, `update_item`, `transition_item`,
`block_item`, `unblock_item`, `add_note`, `link_items`, `claim_item`, `release_item`.

Each maps to the identically-scoped `pm <sub>` command above. The three read tools
(`get_context`, `list_items`, `get_item`) are always available to agents; the nine write tools are
available to agents only while `PM_AUTONOMOUS_WRITES` is on (default). PM tools write **only the
store, never desk files** — `transition_item`'s document gate is the safety, not a write flag.
`get_context` is the cold-start briefing tool: one call returns the active / blocked / stalled sets
and recent transitions for the desk, the same result reachable via `deskkit pm context` and the
TUI landing view (one core, three surfaces).

The nine write tools each carry optional `actor` / `actor_kind` / `delegation_parent` fields,
recorded verbatim on the audit trail (§3.6 of the spec); left unset they default to actor
`"agent"`, kind `"agent"`. `create_item` and `update_item` additionally accept an optional `body`
— the item's long-form narrative, acceptance criteria, or spec, stored inline; `get_item` returns
it (the `list_items` summary shape omits it).

Wire it into a Claude Code project (or use the `desk-pm` plugin, below):

```json
{
  "mcpServers": {
    "deskkit-pm": {
      "command": "/path/to/deskkit",
      "args": ["mcp-serve"],
      "env": { "PM_ENABLED": "true", "DESK_ROOT": "/path/to/desk", "DESK_NAME": "my-desk" }
    }
  }
}
```

## The TUI views

When the module is enabled, three PM views mount into `deskkit`'s full-screen TUI, over the same
engine as the CLI and MCP surfaces:

- **`pm context`** — the landing view: the `get_context` briefing rendered on entry (the same call
  the tool and the CLI make).
- **`pm board`** — the work graph by phase / court.
- **`pm item`** — one item's detail: notes, dependencies, transitions, ancestors.

## The `desk-pm` plugin

A separate Claude Code plugin (shared marketplace) turns the graph into an agent-facing layer over
the PM MCP tools: session-open briefing, advance-an-item, and triage skills; a `pm-operator` agent
that runs the graph over the twelve tools but never authors gate documents or writes a repo; and a
`SessionStart` hook that injects `deskkit pm context` at session start (silent no-op when PM is off
or `deskkit` is absent). The data and runtime stay in the one binary — the plugin is the surface,
not a second store. Details: `../plugin/desk-pm/README.md`.

## Adopting the PM graph on a real desk

The generic adoption path (spec §8.1), identity-neutral:

1. **Enable + migrate** — `PM_ENABLED=true` (or the profile key), then `deskkit migrate up` so the
   PM collections and their stable ids exist.
2. **Seed `desk_config`** — write the gate rules (which document type/status gates which
   transition per item type) and the `status_label` vocabulary for the desk's types. The shipped
   defaults are a starting seed, meant to be re-ruled to the desk's own workflow.
3. **Import existing work surfaces** — the handoff threads index, standing plans/tasks become
   `items` rows with their court, type, pointer, and phase (idempotent, desk-scoped).
4. **Retire the old surfaces to pointers** — the flat index collapses to a pointer to
   `get_context`, matching the desk's graduation discipline.
5. **Never write the live desk during a dry-run** — import into a scratch store first, observe
   `get_context`, then adopt.

## Boundaries

The PM system writes only its own store — never a desk file, never a code repo (spec §7). For
code-repo work the GitHub board stays the single source of truth; a PM item carries a `pointer` to
an issue URL, not a copy of its state. When a gate needs a document produced, that document is
authored through the normal (human or librarian-supervised) path — the gate reads verdicts, it
never writes documents.

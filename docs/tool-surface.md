# Tool surface — the authoritative map

_Enumerates every tool/command desk-standard exposes, across all three surfaces, with the exact
gate that controls each and the empirically-verified count per surface. This is the reference that
replaces the informal (and wrong) "seven-tool core" shorthand._

Status: active
Date: 2026-07-20

desk-standard is **two products over one shared schema**. Tools reach a caller through three
distinct surfaces, and the count differs on each — so "how many tools are there?" has no single
answer without naming the surface and its gate:

1. **Librarian CLI** — the `deskkit` binary's Cobra subcommands (the fullest set).
2. **Librarian MCP server** — `deskkit mcp-serve`, a *gated subset* of the tool core, model-facing.
3. **Plugin TS MCP server** — `plugin/mcp/server.ts`, a *separate* server over the TypeScript core.

The librarian CLI and the librarian MCP server are two faces of **one Go tool core**
(`internal/core/toolcore`); the plugin TS server is an entirely separate codebase with its own
tools. They are not the same set and must not be conflated.

> **Why counts were disputed.** The old `mcp-serve` help string said "seven-tool core"; the issue
> text guessed "5 default"; a scout counted 8. All three were derived by reading, not running. The
> numbers below are derived **empirically** — see [How the counts were derived](#how-the-counts-were-derived).

---

## 1. Librarian CLI subcommands (`deskkit`)

Registered in `librarian/cmd/deskkit/main.go` (`registerToolCommands`) plus the PocketBase- and
migratecmd-provided system commands. The CLI is the **only** surface that carries `restore` and the
supervised `apply-fix`/`findings dispose` actions.

| Subcommand | Source | Notes |
|---|---|---|
| `init` | `registerToolCommands` / `newInitCmd` | Scaffold a desk profile; intercepted in `main()` before PocketBase bootstraps. |
| `sweep` | tool core | Reindex the desk tree. |
| `patrol` | tool core | File rule findings R1–R6; no fs writes. |
| `propose-fix` | tool core | Plan mechanical fixes; record originals. |
| `apply-fix` | tool core | Supervised byte-exact commit; writes desk files. |
| `restore` | tool core | Reverse a change to the recorded original. **CLI-only — never exposed over MCP.** |
| `query` | tool core | Read-only queries. `--include-disposed` widens `findings` past the live-only default. |
| `record-feedback` | tool core | Write one feedback-log entry. |
| `findings dispose <id> --as <disposition>` | `registerToolCommands` | Supervised disposition lifecycle (`open`/`acknowledged`/`triaged`/`wont_fix`). CLI-only, like `restore`. |
| `agent` | `registerToolCommands` | Run the eino loop once (manual trigger). |
| `chat` | `registerToolCommands` | Interactive session (TUI or REPL). |
| `mcp-serve` | `registerToolCommands` | Start the librarian MCP server (surface 2 below). |
| `gui` | `registerToolCommands` | Serve the DB + open the admin console. |
| `serve` | PocketBase | Web server + wake layer. |
| `migrate` | migratecmd | Schema migrations (`up`/`down`/…). |
| `superuser` | PocketBase | Manage superusers. |
| `help`, `completion` | Cobra built-ins | — |
| `pm` (group) | `registerPMCommands` | **Only when `PM_ENABLED`** (the PM module is off by default). |

Verify live: `deskkit --help` (add `PM_ENABLED=true` to see the `pm` group).

---

## 2. Librarian MCP server tool surface (`deskkit mcp-serve`)

Model-facing, so it uses the **§5.4 registration-time gate** (`internal/core/mcp/server.go` →
`toolcore.ExposedTools(cfg)`); the per-tool gate flags live in
`librarian/internal/modules/librarian/tools/specs.go` (`AgentDefault` / `AgentGated`). The gate has
two independent switches, so there are four combinations:

| Environment | Tool count | Tools |
|---|---:|---|
| **default** (neither flag) | **5** | `sweep`, `patrol`, `propose_fix`, `query`, `record_feedback` |
| `LIBRARIAN_AUTONOMOUS_WRITES=true` | **6** | the 5 above **+ `apply_fix`** |
| `PM_ENABLED=true` | **17** | the 5 above **+ 12 PM tools** (below) |
| both flags | **18** | the 6 (with `apply_fix`) **+ 12 PM tools** |

The 12 PM tools (added only under `PM_ENABLED`, from the PM module's specs under
`librarian/internal/modules/pm/`): `get_context`, `list_items`, `get_item`, `create_item`,
`update_item`, `transition_item`, `block_item`, `unblock_item`, `add_note`, `link_items`,
`claim_item`, `release_item`.

**Gate rules that make the count differ from the CLI:**

- `apply_fix` is `AgentGated` — exposed over MCP **only** when `LIBRARIAN_AUTONOMOUS_WRITES=true`
  (checked again at execution time).
- `restore` is **never** exposed over MCP — recovery is a supervised CLI action (§5.5). Its
  exclusion is structural: it is neither `AgentDefault` nor `AgentGated`, and `ExposedTools` also
  filters it defensively.
- `findings dispose`, `init`, `agent`, `chat`, `gui`, `serve`, `migrate`, `superuser` are CLI
  concerns and have no MCP tool.

The server also prints a one-line mount signal to **stderr** naming the exact exposed set — e.g.
`deskkit mcp-serve: mounted "deskkit" v1; 5 tool(s) exposed: sweep, patrol, propose_fix, query,
record_feedback` — fed the same `ExposedTools(cfg)` it registers, so it is a faithful count.

---

## 3. Plugin TypeScript MCP server (`plugin/mcp/server.ts`)

A **separate** stdio MCP server over the TypeScript core — not the Go tool core, not gated by the
librarian's env flags. Its tools are the `TOOLS` array in `plugin/core/tools.ts` (line ~240).

| Tool | Source |
|---|---|
| `profile_get` | `plugin/core/tools.ts` |
| `profile_validate` | `plugin/core/tools.ts` |
| `template_render` | `plugin/core/tools.ts` |
| `knowledge_index` | `plugin/core/tools.ts` |

**Count: 4**, fixed (no gate). The packaged copy at `plugin/claude-plugin/mcp/server.js` is
generated and drift-guarded — same four tools.

---

## Summary

| Surface | Count | Gate |
|---|---:|---|
| Librarian CLI subcommands | 16 base (+ `pm` group under `PM_ENABLED`) | — |
| Librarian MCP — default | 5 | none |
| Librarian MCP — `+LIBRARIAN_AUTONOMOUS_WRITES` | 6 | adds `apply_fix` |
| Librarian MCP — `+PM_ENABLED` | 17 | adds 12 PM tools |
| Librarian MCP — both | 18 | both |
| Plugin TS MCP server | 4 | none |

There is no single "core" count. The nearest thing to the old "seven" was never right for any
surface.

---

## How the counts were derived

Empirically, against a build of the current tree — not by reading source:

**Librarian MCP (surface 2)** — build the binary, then for each env combination speak
newline-delimited JSON-RPC over stdio (`initialize`, then `tools/list`) to `deskkit mcp-serve` and
count the returned `result.tools`, against a throwaway store:

```
go build -o /tmp/deskkit ./cmd/deskkit
printf '%s\n%s\n%s\n' \
  '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"p","version":"0"}}}' \
  '{"jsonrpc":"2.0","method":"notifications/initialized"}' \
  '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}' \
  | DESK_ROOT=<desk> DESK_NAME=probe /tmp/deskkit --dir <fresh-store> mcp-serve
```

Read the `id:2` response and count `result.tools`. Repeat with `LIBRARIAN_AUTONOMOUS_WRITES=true`
and/or `PM_ENABLED=true`. The server's stderr mount signal corroborates each count. Both methods
agreed exactly: 5 / 6 / 17 / 18.

**Librarian CLI (surface 1)** — `deskkit --help` (and again with `PM_ENABLED=true`).

**Plugin TS server (surface 3)** — the `TOOLS` array in `plugin/core/tools.ts`.

To re-verify after any tool add/remove, re-run the probe above; the numbers in this doc must match.

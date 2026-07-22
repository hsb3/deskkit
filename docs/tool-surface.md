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
| `query` | tool core | Read-only queries. `--include-disposed` widens `findings` past the live-only default; the `search` (`--term`/`--limit`) and `content` (`--path`) kinds retrieve indexed file bodies. |
| `record-feedback` | tool core | Write one feedback-log entry. |
| `findings dispose <id> --as <disposition> [--by <actor>] [--reason <text>]` | `registerToolCommands` | Supervised disposition lifecycle (`open`/`acknowledged`/`triaged`/`wont_fix`); optional `--by`/`--reason` record who/why (no baked default actor) and are cleared when re-disposed to `open`. CLI-only, like `restore`. |
| `agent` | `registerToolCommands` | Run the eino loop once (manual trigger). |
| `chat` | `registerToolCommands` | Interactive session (TUI or REPL). |
| `mcp-serve` | `registerToolCommands` | Start the librarian MCP server (surface 2 below). |
| `gui` | `registerToolCommands` | Serve the DB + open the admin console. |
| `serve` | PocketBase | Web server + wake layer. |
| `migrate` | migratecmd | Schema migrations (`up`/`down`/…). |
| `superuser` | PocketBase | Manage superusers. |
| `help`, `completion` | Cobra built-ins | — |
| `pm` (group) | `registerPMCommands` | Registered when the PM module is enabled — **on by default** since 1.0 (ADR 0008 amendment); `PM_ENABLED=false` (or profile `modules.pm.enabled: false`) disables it and `pm` becomes cobra's unknown-command error. |

Verify live: `deskkit --help` (the `pm` group shows by default; add `PM_ENABLED=false` to hide it).

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

The column labels are a **gate truth-table** — the count as a function of the two flags, not the
runtime default. Since 1.0 `PM_ENABLED` **defaults on** (ADR 0008 amendment 2026-07-21), so a
fresh desk's live MCP surface is the `PM_ENABLED=true` row (**17**, or **18** with
`LIBRARIAN_AUTONOMOUS_WRITES`) unless the desk opts out with `PM_ENABLED=false`.

The 12 PM tools (present whenever the PM module is enabled — the default; from the PM module's
specs under `librarian/internal/modules/pm/`):

| Tool | Writes | Notes |
|---|---|---|
| `get_context` | no | single-call cold-start briefing |
| `list_items` | no | filtered graph query |
| `get_item` | no | one item + notes/deps/transitions/ancestors, including its `body` (the list/summary shape omits it) |
| `create_item` | yes | accepts an optional long-form `body`; carries the optional actor fields below |
| `update_item` | yes | accepts an optional `body` (omit to leave unchanged; pass an empty string to clear); carries the optional actor fields below; refused by a live foreign claim ([ADR 0020](decisions/0020-pm-claim-semantics.md)) |
| `transition_item` | yes | carries the optional actor fields below; refused by a live foreign claim (ADR 0020) |
| `block_item` / `unblock_item` | yes | carry the optional actor fields below; refused by a live foreign claim (ADR 0020) |
| `add_note` | yes | carries the optional actor fields below |
| `link_items` | yes | carries the optional actor fields below |
| `claim_item` / `release_item` | yes | carry the optional actor fields below |

Every write tool's input carries optional `actor` / `actor_kind` / `delegation_parent` fields
(`librarian/internal/modules/pm/tools/types.go` `ActorFields`), recorded verbatim on the audit
row (pm-system-v1-spec.md §3.6); unset, they default to actor `"agent"`, kind `"agent"` — the
model-facing surfaces are agent-driven by default. The CLI instead defaults its persistent
`--actor` flag to `$USER` (else `"operator"`), kind `human` (`librarian/cmd/deskkit/pm.go`).

**Gate rules that make the count differ from the CLI:**

- `apply_fix` is `AgentGated` — exposed over MCP **only** when `LIBRARIAN_AUTONOMOUS_WRITES=true`
  (checked again at execution time).
- `restore` is **never** exposed over MCP — recovery is a supervised CLI action (§5.5). Its
  exclusion is structural: it is neither `AgentDefault` nor `AgentGated`, and `ExposedTools` also
  filters it defensively.
- `findings dispose`, `init`, `agent`, `chat`, `gui`, `serve`, `migrate`, `superuser` are CLI
  concerns and have no MCP tool.

The server also prints a one-line mount signal to **stderr** naming the gated module set and the
exact exposed tool set — e.g. `deskkit mcp-serve: mounted "deskkit" v1; modules: all; 5 tool(s)
exposed: sweep, patrol, propose_fix, query, record_feedback` — fed the same `ExposedTools(cfg)` it
registers, so it is a faithful count. The `modules:` segment reads `all` when `MCP_MODULES` is unset
and names the declared set otherwise (see §2.1).

### 2.1 Module gating on a shared mount (`MCP_MODULES`)

A single `deskkit mcp-serve` process can be narrowed to specific modules with the `MCP_MODULES`
environment variable, so a **shared MCP mount exposes only the tools that mount is meant to carry**.
This is a **third, orthogonal axis** layered on top of the two §5.4 switches above — the gate keys on
each tool's `ToolSpec.Module` (`internal/core/mcp/server.go` → `toolcore.SelectByModules` over
`toolcore.ExposedSpecs(cfg)`). Three cases, kept deliberately distinct:

- **`MCP_MODULES` unset** → no module filter; every tool the §5.4 gate exposes is served. **The
  5 / 6 / 17 / 18 counts in the table above are all the unset case** — unchanged behavior.
- **`MCP_MODULES` set, non-empty** (e.g. `pm`, or `librarian,pm`) → the exposed set is filtered to
  tools whose owning module is in the declared set. The **desk-pm mount** shape — a mount that
  declares `MCP_MODULES=pm` alongside `PM_ENABLED=true` — exposes
  **exactly the 12 PM tools** and **none** of the 5 librarian ride-alongs (`sweep`, `patrol`,
  `propose_fix`, `query`, `record_feedback`). A partially-matching set (`librarian,bogus`) keeps the
  matched subset and serves.
- **`MCP_MODULES` set but resolving to nothing** → **fail loud** (exit 1), never a silent fallback
  to "all". Two sub-cases: an explicitly empty / whitespace-only declaration (`""`, `" , "` — names
  no module after splitting and trimming), or an unresolvable set that yields zero exposed tools (a
  typo, or a module not registered/enabled on this desk — e.g. `MCP_MODULES=pm` without
  `PM_ENABLED`). Both print an actionable stderr line naming `MCP_MODULES` and exit non-zero,
  because a shared mount that silently served the wrong (or full) surface would defeat its purpose.

| Mount | Env | Tool count | Tools |
|---|---|---:|---|
| Librarian MCP (default) | *(none)* | 5 | the 5 librarian defaults |
| **desk-pm mount** | `PM_ENABLED=true`, `MCP_MODULES=pm` | **12** | the 12 PM tools only (no ride-alongs) |

The mount signal names the gated set, so the axis is legible in the host's log:
`deskkit mcp-serve: mounted "deskkit" v1; modules: pm; 12 tool(s) exposed: get_context, list_items,
…`.

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

**Count: 4**, fixed (no gate). The packaged copy at `plugins/desk-standard/mcp/server.js` is
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
agreed exactly: 5 / 6 / 17 / 18. These four are the **`MCP_MODULES`-unset** baseline.

**Module-gating axis (§2.1)** — re-run the same probe with `MCP_MODULES` set to prove the third
axis. A trailing `sleep` keeps stdin open so the response flushes before the EOF shutdown races it:

```
DESKDIR=$(mktemp -d); STORE=$(mktemp -d)
{ printf '%s\n%s\n%s\n' \
  '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"p","version":"0"}}}' \
  '{"jsonrpc":"2.0","method":"notifications/initialized"}' \
  '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}'; sleep 1; } \
  | DESK_ROOT="$DESKDIR" DESK_NAME=probe PM_ENABLED=true MCP_MODULES=pm /tmp/deskkit --dir "$STORE" mcp-serve
```

The desk-pm mount variant (`PM_ENABLED=true MCP_MODULES=pm`) returns **exactly 12** — the PM tools
and no librarian ride-alongs — and its stderr reads `modules: pm; 12 tool(s) exposed: …`. Drop
`MCP_MODULES` and the same env yields 17 (unset = all). Set `MCP_MODULES=""` (or an unresolvable
name) on a resolved desk and the process **exits 1** with an actionable stderr line — proving the
fail-loud contract rather than a silent fallback.

> **Drift-guard note.** The [ADR-0016](decisions/0016-ts-boundary-deskkit-proxy.md) tool-surface
> drift guard's current framing is
> **two axes / four combinations** (`LIBRARIAN_AUTONOMOUS_WRITES` × `PM_ENABLED` → 5/6/17/18); that
> framing **predates module gating** and models only the `MCP_MODULES`-unset baseline. `MCP_MODULES`
> is a genuine **third axis** — the guard must be extended to model it (at minimum the desk-pm
> `MCP_MODULES=pm` → 12 mount) so the count table above stays enforced, not just documented.

**Librarian CLI (surface 1)** — `deskkit --help` (and again with `PM_ENABLED=true`).

**Plugin TS server (surface 3)** — the `TOOLS` array in `plugin/core/tools.ts`.

To re-verify after any tool add/remove, re-run the probe above; the numbers in this doc must match.

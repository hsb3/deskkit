# Tool surface — the authoritative map

_Enumerates every tool/command deskkit exposes, across both of its tool-bearing surfaces, with the
exact gate that controls each and the empirically-verified count per surface. This is the reference
that replaces the informal (and wrong) "seven-tool core" shorthand._

Status: active
Date: 2026-07-20 (framing refreshed 2026-08-18 for the single-binary consolidation)

deskkit is **one binary and one plugin bundle over one shared schema**. Tools reach a caller
through two distinct surfaces, and the count differs on each — so "how many tools are there?" has
no single answer without naming the surface and its gate:

1. **Librarian CLI** — the `deskkit` binary's Cobra subcommands (the fullest set).
2. **Librarian MCP server** — `deskkit mcp-serve`, a *gated subset* of the tool core, model-facing.

Both are faces of **one Go tool core** (`internal/core/toolcore`), fed by the registered modules
(`profile`, `librarian`, `pm`). They are not the same set and must not be conflated.

> **The four profile tools used to be a third surface.** `profile_get`, `profile_validate`,
> `template_render`, and `knowledge_index` were served by a separate TypeScript stdio MCP server
> over a TypeScript core. That server is gone: the four are now the Go **`profile` module** (§2.2)
> on the same `deskkit mcp-serve` process. One binary carries the whole surface.

> **Why counts were disputed.** The old `mcp-serve` help string said "seven-tool core"; the issue
> text guessed "5 default"; a scout counted 8. All three were derived by reading, not running. The
> numbers below are derived **empirically** — see [How the counts were derived](#how-the-counts-were-derived).

---

## 1. Librarian CLI subcommands (`deskkit`)

Registered in `cmd/deskkit/main.go` (`registerToolCommands`, plus `finalizeCommandTree`
for the PocketBase system commands and `migratecmd` for `migrate`). The CLI is the **only** surface
that carries `restore` and the supervised `apply-fix`/`findings dispose` actions. `--help` renders
them in labelled groups (Setup & config / Inspect / Fix / Work graph / Agent / Admin) rather than
one alphabetic list; a command without a group is a bug, caught by `TestCommandGroups_*`.

| Subcommand | Source | Notes |
|---|---|---|
| `init` | `registerToolCommands` / `newInitCmd` | Scaffold a desk profile; intercepted in `main()` before PocketBase bootstraps. |
| `desks` | `registerToolCommands` / `newDesksCmd` | List the desks this machine has a store for, marking the one the current directory resolves to. Read-only over the XDG store home; intercepted in `main()` like `init`, so asking which desks exist never creates one. |
| `config` (group) | `registerToolCommands` / `newConfigCmd` | `show` (every resolved setting + the source that won: env/profile/central/default), `path`, `edit`, `set <key> <value>` for the machine-wide config file. Secrets render masked. Intercepted in `main()` like `init`; creates no store. |
| `sweep` | tool core | Reindex the desk tree. |
| `patrol` | tool core | File rule findings R1–R6; no fs writes. |
| `propose-fix` | tool core | Plan mechanical fixes; record originals. |
| `apply-fix` | tool core | Supervised byte-exact commit; writes desk files. |
| `restore` | tool core | Reverse a change to the recorded original. **CLI-only — never exposed over MCP.** |
| `query` | tool core | Read-only queries. `--include-disposed` widens `findings` past the live-only default; the `search` (`--term`/`--limit`) and `content` (`--path`) kinds retrieve indexed file bodies. Each `findings`/`uncollapsed` finding brief carries the patrol-finding record `id` alongside `path`/`detail` (matching `feedback`'s shape), so its output feeds directly into `findings dispose <id>` — no admin-GUI/REST detour needed. |
| `record-feedback` | tool core | Write one feedback-log entry. |
| `findings dispose <id> --as <disposition> [--by <actor>] [--reason <text>]` | `registerToolCommands` | Supervised disposition lifecycle (`open`/`acknowledged`/`triaged`/`wont_fix`); optional `--by`/`--reason` record who/why (no baked default actor) and are cleared when re-disposed to `open`. CLI-only, like `restore`. |
| `agent` | `registerToolCommands` | Run the eino loop once (manual trigger). |
| `chat` | `registerToolCommands` | Interactive session (TUI or REPL). |
| `mcp-serve` | `registerToolCommands` | Start the librarian MCP server (surface 2 below). |
| `gui` | `registerToolCommands` | Serve the DB + open the admin console. |
| `serve` | PocketBase (`cmd.NewServeCommand`, registered by `finalizeCommandTree`) | Web server + wake layer. |
| `migrate` | migratecmd | Schema migrations (`up`/`down`/…). |
| `superuser` | PocketBase (`cmd.NewSuperuserCommand`, registered by `finalizeCommandTree`) | Manage superusers. |
| `help`, `completion` | Cobra built-ins | — |
| `pm` (group) | `registerPMCommands` | Registered when the PM module is enabled — **on by default** since 1.0 (ADR 0008 amendment); `PM_ENABLED=false` (or profile `modules.pm.enabled: false`) disables it and `pm` becomes cobra's unknown-command error. |

> **`agent` is ratified surface, not an oversight.** It was added during Phase 1 because the
> eino loop needed a way to be driven and tested outside `serve`, and the gap between that and
> the spec's originally-scoped six-tool core (`sweep`/`patrol`/`propose_fix`/`apply_fix`/
> `restore`/`query`) was never explicitly closed by a ruling — only by the command shipping and
> staying real, exercised surface (the manual agent harness, `examples/agent-loop.sh`, drives it
> directly). **Owner ruling 2026-08-18 (DESK-69): keep it.** The shipped surface led and this
> spec followed — that is the honest framing, not a claim that `agent` was always documented as
> deliberate. Nothing depends on its absence; removing a working, tested command to satisfy a
> document would have been the wrong way round.

Verify live: `deskkit --help` (the `pm` group shows by default; add `PM_ENABLED=false` to hide it).

---

## 2. Librarian MCP server tool surface (`deskkit mcp-serve`)

Model-facing, so it uses the **§5.4 registration-time gate** (`internal/core/mcp/server.go` →
`toolcore.ExposedTools(cfg)`); the per-tool gate flags live in
`internal/modules/librarian/tools/specs.go` (`AgentDefault` / `AgentGated`). The gate has
two independent switches, so there are four combinations:

| Environment | Tool count | Tools |
|---|---:|---|
| **default** (neither flag) | **9** | the 4 ungated `profile` tools (§2.2) **+** `sweep`, `patrol`, `propose_fix`, `query`, `record_feedback` |
| `LIBRARIAN_AUTONOMOUS_WRITES=true` | **10** | the 9 above **+ `apply_fix`** |
| `PM_ENABLED=true` | **21** | the 9 above **+ 12 PM tools** (below) |
| both flags | **22** | the 10 (with `apply_fix`) **+ 12 PM tools** |

Every count in this doc is a **live total** — what a `tools/list` against the running binary
returns. The always-on `profile` module (§2.2) contributes a constant 4 to each row: it has no env
gate, so it is part of the total, not a footnote to it. The librarian × pm contribution alone is
5 / 6 / 17 / 18.

The column labels are a **gate truth-table** — the count as a function of the two flags, not the
runtime default. Since 1.0 `PM_ENABLED` **defaults on** (ADR 0008 amendment 2026-07-21), so a
fresh desk's live MCP surface is the `PM_ENABLED=true` row (**21**, or **22** with
`LIBRARIAN_AUTONOMOUS_WRITES`) unless the desk opts out with `PM_ENABLED=false` (**9** / **10**).

The 12 PM tools (present whenever the PM module is enabled — the default; from the PM module's
specs under `internal/modules/pm/`):

| Tool | Writes | Notes |
|---|---|---|
| `get_context` | no | single-call cold-start briefing |
| `list_items` | no | filtered graph query |
| `get_item` | no | one item + notes/deps/transitions/ancestors, including its `body` (the list/summary shape omits it) |
| `create_item` | yes | accepts an optional long-form `body`; carries the optional actor fields below |
| `update_item` | yes | accepts an optional `body` (omit to leave unchanged; pass an empty string to clear); carries the optional actor fields below; refused by a live foreign claim (ADR 0020, DESK-41) |
| `transition_item` | yes | carries the optional actor fields below; refused by a live foreign claim (ADR 0020) |
| `block_item` / `unblock_item` | yes | carry the optional actor fields below; refused by a live foreign claim (ADR 0020) |
| `add_note` | yes | carries the optional actor fields below |
| `link_items` | yes | carries the optional actor fields below |
| `claim_item` / `release_item` | yes | carry the optional actor fields below |

Every write tool's input carries optional `actor` / `actor_kind` / `delegation_parent` fields
(`internal/modules/pm/tools/types.go` `ActorFields`), recorded verbatim on the audit
row (pm-system-v1-spec.md §3.6); unset, they default to actor `"agent"`, kind `"agent"` — the
model-facing surfaces are agent-driven by default. The CLI instead defaults its persistent
`--actor` flag to `$USER` (else `"operator"`), kind `human` (`cmd/deskkit/pm.go`).

**Gate rules that make the count differ from the CLI:**

- `apply_fix` is `AgentGated` — exposed over MCP **only** when `LIBRARIAN_AUTONOMOUS_WRITES=true`
  (checked again at execution time).
- `restore` is **never** exposed over MCP — recovery is a supervised CLI action (§5.5). Its
  exclusion is structural: it is neither `AgentDefault` nor `AgentGated`, and `ExposedTools` also
  filters it defensively.
- `findings dispose`, `init`, `agent`, `chat`, `gui`, `serve`, `migrate`, `superuser` are CLI
  concerns and have no MCP tool.

The server also prints a one-line mount signal to **stderr** naming the gated module set and the
exact exposed tool set — e.g. `deskkit mcp-serve: mounted "deskkit" v1; modules: librarian; 5
tool(s) exposed: sweep, patrol, propose_fix, query, record_feedback` — fed the same `ExposedTools(cfg)` it
registers, so it is a faithful count. The `modules:` segment reads `all` when `MCP_MODULES` is unset
and names the declared set otherwise (see §2.1).

### 2.1 Module gating on a shared mount (`MCP_MODULES`)

A single `deskkit mcp-serve` process can be narrowed to specific modules with the `MCP_MODULES`
environment variable, so a **shared MCP mount exposes only the tools that mount is meant to carry**.
This is a **third, orthogonal axis** layered on top of the two §5.4 switches above — the gate keys on
each tool's `ToolSpec.Module` (`internal/core/mcp/server.go` → `toolcore.SelectByModules` over
`toolcore.ExposedSpecs(cfg)`). Three cases, kept deliberately distinct:

- **`MCP_MODULES` unset** → no module filter; every tool the §5.4 gate exposes is served. **The
  9 / 10 / 21 / 22 counts in the table above are all the unset case** — unchanged behavior.
- **`MCP_MODULES` set, non-empty** (e.g. `pm`, or `librarian,pm`) → the exposed set is filtered to
  tools whose owning module is in the declared set. The **pm-only mount** (`MCP_MODULES=pm`) shape
  — a mount that declares `MCP_MODULES=pm` alongside `PM_ENABLED=true` — exposes
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
| Librarian MCP (default; PM_ENABLED implicit) | *(none)* | 21 | the 4 profile + 5 librarian + 12 PM defaults |
| **pm-only mount** (`MCP_MODULES=pm`) | `PM_ENABLED=true`, `MCP_MODULES=pm` | **12** | the 12 PM tools only (no ride-alongs) |

The pm-only mount carries **no profile tools either**: `profile` being ungated means it rides
every *unfiltered* mount, not that it overrides a mount's declared module set.

The mount signal names the gated set, so the axis is legible in the host's log:
`deskkit mcp-serve: mounted "deskkit" v1; modules: pm; 12 tool(s) exposed: get_context, list_items,
…`.

### 2.2 The `profile` module's four tools (ungated)

The `profile` module reads a desk's personalization surfaces — the `_knowledge/profile.*` profile
and the `_knowledge/` background folder — and contributes four **read-only** tools
(`internal/modules/profile/tools/specs.go`). They are the Go home of the tool family
that previously ran as a separate TypeScript stdio server, so one binary now carries the
whole surface.

| Tool | Writes | Notes |
|---|---|---|
| `profile_get` | no | resolve a dotted profile key to its scalar value; fails loud, naming the keys available under the deepest resolved parent |
| `profile_validate` | no | validate the profile against schema v1; returns `{ valid, errors, profilePath }` — an absent or unparseable profile is a `valid:false` **result**, not an error |
| `template_render` | no | substitute `{{profile.<key>}}` / `{{env.<VAR>}}` placeholders (optional `\|\| "default"`); a placeholder with no default that resolves empty is refused |
| `knowledge_index` | no | index `_knowledge/**/*.md` with per-file `path`/`bytes`/`words`, plus content up to a byte budget (default 65536); over-budget files are metadata-only |

There is **no env gate**: all four are `AgentDefault`, none is `AgentGated`, none writes a desk
file, and the module is always enabled — a desk always has a personalization surface, even when
it is empty (the tools then answer "this desk declares nothing" instead of vanishing). So the
count is a fixed **4** on every combination of the §2 flags.

Discovery starts at the **resolved desk root** (`--dir` / `DESK_ROOT`), falling back to the
process working directory only when no desk root resolved — the binary is normally launched from
an arbitrary cwd, so a cwd-only walk-up would find nothing in exactly the cases that matter.

| Mount | Env | Tool count | Tools |
|---|---|---:|---|
| **profile-only mount** | `MCP_MODULES=profile` | **4** | `profile_get`, `profile_validate`, `template_render`, `knowledge_index` |

---

## Summary

| Surface | Count | Gate |
|---|---:|---|
| Librarian CLI subcommands | 18 base (+ `pm` group under `PM_ENABLED`) | — |
| Librarian MCP — default | 9 | none |
| Librarian MCP — `+LIBRARIAN_AUTONOMOUS_WRITES` | 10 | adds `apply_fix` |
| Librarian MCP — `+PM_ENABLED` | 21 | adds 12 PM tools |
| Librarian MCP — both | 22 | both |
| **pm-only mount** (§2.1) | 12 | `MCP_MODULES=pm` |
| **profile-only mount** (§2.2) | 4 | `MCP_MODULES=profile` |

The four `Librarian MCP` rows are live totals: each already includes the ungated `profile`
module's constant 4 (§2.2). The two mount rows are the `MCP_MODULES` axis, not additions to the
rows above.

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
agreed exactly: **9 / 10 / 21 / 22** — `PM_ENABLED=false` yields 9 (10 with
`LIBRARIAN_AUTONOMOUS_WRITES`) and the default PM-on mount yields 21 (22). These four are the
**`MCP_MODULES`-unset** baseline, and each already includes the always-registered `profile`
module's 4. Adding `MCP_MODULES=profile` returns **exactly 4** and its stderr reads
`modules: profile; 4 tool(s) exposed: profile_get, profile_validate, template_render,
knowledge_index`.

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

The pm-only mount variant (`PM_ENABLED=true MCP_MODULES=pm`) returns **exactly 12** — the PM tools
and no librarian or profile ride-alongs — and its stderr reads `modules: pm; 12 tool(s) exposed: …`.
Drop `MCP_MODULES` and the same env yields 21 (unset = all). Set `MCP_MODULES=""` (or an unresolvable
name) on a resolved desk and the process **exits 1** with an actionable stderr line — proving the
fail-loud contract rather than a silent fallback.

> **Drift-guard note.** Every count above is **enforced, not just documented**. The ADR-0016 guard
> lives in `internal/core/mcp/tool_surface_doc_test.go` on the `go test ./...` (`make
> test`) lane: it parses the tables in this doc and re-derives each number from source.
> `TestToolSurfaceDoc_MCPCounts` covers the MCP counts against the real `toolcore` gate over the
> registered `profile` + `librarian` + `pm` specs — the live totals (9 / 10 / 21 / 22) and both
> gated mounts (`MCP_MODULES=pm` → 12, `MCP_MODULES=profile` → 4), the mounts by tool NAME as well
> as count so a tool changing modules is caught. `TestToolSurfaceDoc_CLICount` covers the CLI base
> count against the command registrations in `cmd/deskkit/main.go`.

**Librarian CLI (surface 1)** — `deskkit --help` (and again with `PM_ENABLED=true`).

To re-verify after any tool add/remove, re-run the probe above; the numbers in this doc must match.

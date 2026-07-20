# TS `plugin/mcp` → deskkit `mcp-serve` proxy — design

_The implementation design ADR 0016 calls for: how the harness-pure TypeScript `plugin/mcp`
boundary gains server-backed (librarian) capabilities by **proxying** deskkit's Go `mcp-serve`,
without reimplementing librarian logic in TS. A builder should be able to pick up the build slices
in §5 cold from this document._
Status: active (2026-07-20)

> **Scope note.** This is a *design pass*, not a new architectural ruling. ADR
> [0016](../decisions/0016-ts-boundary-deskkit-proxy.md) already made the call (extend the TS
> boundary via a designed proxy, not amend-spec-to-reality) and named the design of "process
> lifecycle, availability behavior, which tools surface, gating per ADR 0014" a **scoped work
> item** that precedes implementation (`docs/decisions/0016-ts-boundary-deskkit-proxy.md:23-26`).
> This doc elaborates that ruling; it does not amend it. Per the issue's Deliverable A, a design
> that *elaborates* ADR 0016 lands as a `docs/development/` doc (this file); only a design that
> *materially amends* ADR 0016 would fold back into the ADR as a dated amendment — this one does
> not, so the separate-doc path is correct.

---

## 1. What this designs, and what it does not

**Designs:** the composition mechanism, process lifecycle, failure behavior, exposed-tool set,
gating, harness-purity boundary, mount signal, fallback trigger, and the ownable build slices —
one answer per Deliverable B question of issue #122.

**Does not:** implement anything. No code under `plugin/core/` or `plugin/mcp/server.ts` changes
in this issue; the proxy module, the `server.ts` wiring, the claiming skill, and the drift-guard
extension are all follow-up slices (§5). This is a `docs/` file, so no shipped-tree gate
(neutrality, purity, version-sync, bundle drift) fires for it
(`scripts/check-neutrality.mjs:52` scopes the scan to `plugin` / `librarian` / `kits`).

## 2. Ground truth today (verified against this tree)

The TS server and the Go server are two structurally separate, **un-composed** mounts. Neither
calls the other; they coexist only by both appearing in a session's MCP config.

- **The TS server is a thin adapter over the harness-pure TS core.** `plugin/mcp/server.ts`
  imports `TOOLS` from `../core/index.js` (`plugin/mcp/server.ts:18`), maps them 1:1 in its
  `ListTools` handler (`plugin/mcp/server.ts:29-35`), dispatches `CallTool` to each tool's own
  handler and turns a thrown handler error into an MCP `isError` result rather than a protocol
  crash (`plugin/mcp/server.ts:37-54`), and serves over stdio (`plugin/mcp/server.ts:59-63`). It
  re-declares no schema and calls no other process. Its tools are the fixed four
  `profile_get`, `profile_validate`, `template_render`, `knowledge_index`
  (`plugin/core/tools.ts:240`; `docs/tool-surface.md:128-142`, "Count: 4, fixed (no gate)").
- **The plugin bundle mounts only the TS server.** `plugin/claude-plugin/.mcp.json` runs
  `bun ${CLAUDE_PLUGIN_ROOT}/mcp/server.js` (`plugin/claude-plugin/.mcp.json:1-6`) — the
  generated, self-contained bundle copy of `server.ts`.
- **The librarian surface is mounted separately, by the desk-pm bundle.**
  `plugin/desk-pm/.mcp.json` starts `deskkit mcp-serve` directly with
  `PM_ENABLED=true` + `MCP_MODULES=pm` (`plugin/desk-pm/.mcp.json:1-10`), so that mount carries
  **exactly the 12 PM tools and none of the 5 librarian ride-alongs**
  (`docs/tool-surface.md:117-120`; `docs/agent-integration-contract-v1-spec.md:100`).
- **`deskkit mcp-serve` is a gated Go surface.** Default = 5 (`sweep`, `patrol`, `propose_fix`,
  `query`, `record_feedback`); `+LIBRARIAN_AUTONOMOUS_WRITES` = 6 (adds `apply_fix`);
  `+PM_ENABLED` = 17; both = 18 (`docs/tool-surface.md:59-92`). `MCP_MODULES` is an orthogonal
  **third axis**: a mount declaring `MCP_MODULES=librarian` surfaces only librarian-module tools
  even when `PM_ENABLED=true` (`docs/tool-surface.md:94-125`;
  `docs/agent-integration-contract-v1-spec.md:79-90`; module name `librarian` is real —
  `librarian/internal/modules/librarian/tools/specs.go:23`).

The composition question this design settles: **does `plugin/mcp/server.ts` itself start calling
the Go binary at runtime (an in-process proxy — the reading ADR 0016's "the TS boundary gains …
by proxying" language implies), and if so, how.** This design answers *yes, in-process, as a
spawned stdio child*, and specifies the how below.

## 3. Design decisions (Deliverable B)

Each subsection states the question, the options, their trade-offs, and lands a recommendation
with rationale.

### 3.1 Process lifecycle — eager stdio child, parent-owned

**Question.** Does the TS server spawn `deskkit mcp-serve` as a long-lived child at its own
startup, spawn it lazily on the first proxied `CallTool`, or call an already-running
`deskkit serve`/`mcp-serve` over some IPC? Who owns start/stop, and what happens to the child if
the TS server exits or crashes?

| Option | Mechanism | Trade-offs |
|---|---|---|
| **A. Eager child at startup** *(recommended)* | `server.ts` spawns `deskkit mcp-serve` in `main()`, holds a stdio JSON-RPC **client** to it for the session | Matches the "MCP servers wire at session start" contract (`brownfield-adoption/SKILL.md:46-47`); lets the mount signal report native + proxied tools in one startup line; clean shutdown is free (below). Cost: one child process per session even if no proxied tool is called. |
| B. Lazy child on first proxied `CallTool` | Spawn only when a proxied tool is first invoked | Defers process cost, but Claude Code calls `tools/list` **at session start** — and the proxy cannot answer `ListTools` for the proxied set without either the child running or a hardcoded tool list (drift risk, the exact rot `docs/tool-surface.md` exists to prevent). So "lazy on CallTool" collapses to "eager on the first ListTools" in practice, buying little and costing signal legibility. |
| C. Attach to an external running server over IPC | Assume an operator ran `deskkit serve`/`mcp-serve` separately; connect over a socket/HTTP | `mcp-serve` is **stdio-only** (`librarian/internal/core/mcp/server.go:139`, `mcp.StdioTransport{}`); `serve` is the PocketBase web server, not an MCP endpoint. There is no HTTP MCP transport to attach to today — this option requires building new Go transport, out of scope and heavier, and pushes lifecycle onto the operator. |

**Recommendation: Option A.** The TS server spawns `deskkit mcp-serve` as a stdio child in
`main()` and acts as a plain MCP **client** to it (speaks `initialize` → `tools/list`, then
forwards `tools/call`). The proxy learns the child's exposed tool set by *asking it*, never by
hardcoding a list — so the proxied surface can never drift from the Go gate.

**Lifecycle ownership.** The parent (TS server) owns start and stop:

- **Start:** spawn in `main()` before `server.connect(transport)` returns to the host, so the
  aggregate mount signal (§3.6) can be emitted once with the true native + proxied counts.
- **Stop / orphan safety:** because the child is a stdio server, **closing the parent's write end
  of the child's stdin is itself a clean shutdown** — the Go server classifies stdin-EOF as a
  normal disconnect and exits 0/silent (`librarian/internal/core/mcp/server.go:97-101, 226-234`).
  So the primary reaping mechanism is "parent dies → pipe closes → child sees EOF → child exits."
  Belt-and-suspenders: register a cleanup handler on the parent's `exit`/`SIGINT`/`SIGTERM` that
  calls `child.kill()`, and spawn without `detached` so the child stays in the parent's process
  group. Residual orphan risk (parent `SIGKILL`, which runs no handler) is bounded by the
  stdin-EOF path and is the standard risk of any stdio-child MCP topology — acceptable.
- **Mid-session child crash:** a proxied `CallTool` after the child has died returns an MCP
  `isError` result naming the child's death (mirroring `server.ts:48-53`'s posture), and the
  proxy attempts **one** respawn (no retry storm); if respawn also fails, the proxied surface
  stays degraded-and-loud (§3.2) while the four native tools keep serving.

### 3.2 Availability behavior — fail *loud*, degrade the proxy, keep native tools alive

**Question.** What happens when `deskkit` is not installed, not on PATH, or the spawn fails? The
#79 precedent and ADR 0014 require **fail loud with a stderr mount signal**
(`docs/decisions/0014-agent-integration-contract.md:30-31`), not the silent drop a PATH-less
`deskkit` produces for the desk-pm mount today
(`plugin/claude-plugin/skills/brownfield-adoption/SKILL.md:44-45`).

**The judgment call this question forces.** The Go `mcp-serve` fails loud by `os.Exit(1)`
(`librarian/internal/core/mcp/server.go:112-115, 123-128`) — correct *there*, because if the desk
is unresolved the Go server has **nothing** to serve. The TS server is different: it always has
four native profile tools that do **not** depend on deskkit. So "fail loud" here must **not** mean
"exit 1 and take the native tools down with it" — that would regress a working surface. The right
reading of ADR 0014's invariant is **loud vs. silent**, not **process-death vs. survival**: the
proxy's failure must be observable, named, and actionable on stderr; it need not kill an
independently-healthy surface. The desk-pm silent-drop is bad because *nothing names why the tools
vanished* — our proxy fixes exactly that, without conflating "loud" with "fatal to the whole
process."

**Recommendation.** At startup the proxy attempts the spawn + handshake. On failure —
`ENOENT` (deskkit not on PATH), non-zero early child exit, or a handshake timeout — the proxy:

1. keeps the four native tools mounted and serving;
2. exposes **zero** proxied tools (never a stale/guessed set);
3. emits a **loud, named** stderr line (§3.6) — e.g. `proxy: deskkit mcp-serve UNAVAILABLE
   (deskkit not found on PATH) — 0 proxied; install deskkit and restart the session`;
4. **forwards the child's own stderr** verbatim when the child started but exited — so when the
   child itself fails loud (desk unresolved → its `requireResolvedConfig` line at
   `librarian/internal/core/mcp/server.go:112-115`, or an `MCP_MODULES` failure at `:123-128`),
   the operator sees the child's actionable message, not a swallowed one.

This composes cleanly: often the proxy's failure reason is just a passthrough of the child's own
fail-loud line.

### 3.3 Which tools surface — the 5 librarian defaults via `MCP_MODULES=librarian`; write-gate passthrough

**Question.** Does the TS boundary proxy the full gated set (5/6/17/18), a fixed named subset, or
something configurable? What is the default gate state, and do `PM_ENABLED` /
`LIBRARIAN_AUTONOMOUS_WRITES` pass through, or does the proxy impose its own default?

**Recommendation: the proxy imposes `MCP_MODULES=librarian` on the child, surfacing exactly the
5 librarian default tools** (`sweep`, `patrol`, `propose_fix`, `query`, `record_feedback`), and
**passes through `LIBRARIAN_AUTONOMOUS_WRITES`** from the proxy process's own environment (so
`apply_fix` appears as the 6th tool iff the operator set that flag). **PM tools never surface
through the TS proxy.**

Rationale, separating the two independent env axes:

- **Module axis (proxy's job, fixed to `librarian`).** The TS/`claude-plugin` bundle is the
  librarian-surface composition; the **desk-pm bundle already owns the PM mount**
  (`plugin/desk-pm/.mcp.json:1-10`, `MCP_MODULES=pm`). Proxying PM tools through the TS server too
  would double-mount them and re-open the ride-along/unclaimed-tool problem the #114 contract just
  closed (`docs/agent-integration-contract-v1-spec.md:100`). Setting `MCP_MODULES=librarian` on
  the child guarantees the librarian slice even if `PM_ENABLED` happens to be ambient in the
  environment — the module gate excludes PM tools regardless
  (`docs/agent-integration-contract-v1-spec.md:79-81`; module name confirmed at
  `librarian/internal/modules/librarian/tools/specs.go:23`). The proxy does **not** set
  `PM_ENABLED`.
- **Write-gate axis (operator policy, passed through).** `LIBRARIAN_AUTONOMOUS_WRITES` is a
  deliberate, operator-set policy with a default of **off** (ADR 0014 parameter 5;
  `docs/agent-integration-contract-v1-spec.md:33`). The proxy must not invent a second default —
  it passes the flag through from its own environment to the child, so the write-gate stays a
  single source of truth. Default off ⇒ **5 tools**; with `LIBRARIAN_AUTONOMOUS_WRITES=true` in
  the session env ⇒ **6 tools** (adds `apply_fix`, itself re-checked at execution time in the Go
  server).

So the proxied set is **not** "configurable" in the open-ended sense (no ability to pull in PM or
arbitrary modules through this mount); it is a **fixed module (`librarian`) whose write-tool count
follows one passed-through policy flag** — the smallest surface that satisfies ADR 0016's
"server-backed capabilities" without duplicating the desk-pm mount.

### 3.4 Gating per ADR 0014 — a claiming skill is a required co-slice

**Question.** ADR 0014 requires "tool-level gating so tools no persona claims are unreachable"
(`docs/decisions/0014-agent-integration-contract.md:29-30`). Which Claude Code skill claims the
proxied tools?

**Finding.** In the plugin's skill layer today, `template_render` is claimed by `desk-setup`
(`plugin/claude-plugin/skills/desk-setup/SKILL.md:8,68`) and `brownfield-adoption`; the other
three TS tools are **no-persona-by-design** (`docs/agent-integration-contract-v1-spec.md:131-140`).
The 5 librarian tools are today claimed only by the **librarian system prompt** (the Go/in-binary
persona) — *not* by any Claude Code skill in the plugin bundle. Surfacing them through the TS
plugin without a matching skill would repeat the unclaimed-tool problem "at larger scale," which
the issue explicitly forbids (Deliverable B, citing D7 finding 4:
`_meta/research/2026-07-design-session/decision-book/D7-spec-reality-reconciliation.md`).

**Recommendation.** A **new/extended Claude Code skill that claims the 5 proxied librarian tools
is a required co-slice of the proxy** — the proxy must not ship without it. Preferred shape: a
**dedicated skill** (working name `desk-librarian` — a skill file, NOT the dedicated
`desk-librarian` *bundle* the contract explicitly ruled out for v1; the name is reused only for
the loop it describes) covering the daily librarian loop
(sweep → patrol → propose_fix → query → record_feedback), because that loop is semantically
distinct from the setup runbooks `desk-setup`/`brownfield-adoption` already own, and the
`librarian-guide.md` prose already describes it. Folding the claims into `conventions-standard` is
the lighter alternative if a fifth skill is judged too many. Final naming/placement is a build-slice
detail (§5, slice 3); the binding requirement is that the proxied tools land **with** their
claiming instructions in the same implementation wave, never ahead of them.

### 3.5 Harness-purity boundary — a new `plugin/mcp/proxy.ts`, never `plugin/core/`

**Question.** Confirm proxy code lives in `plugin/mcp/`, never `plugin/core/`; name the exact new
file(s).

**Recommendation.** The proxy is a new module **`plugin/mcp/proxy.ts`**, beside `server.ts`, plus
minimal wiring inside `server.ts` (a future slice — not touched by this issue). Nothing lands in
`plugin/core/`.

This is settled by ADR 0016's consequence line — "the proxy honors harness-purity: `plugin/core`
stays pure; the proxy lives at the server boundary"
(`docs/decisions/0016-ts-boundary-deskkit-proxy.md:34-35`) — and it sidesteps the purity gate by
construction rather than needing an exemption: `check-core-purity.mjs` scans **only**
`plugin/core/` (its `coreDir` is `../core`, `plugin/scripts/check-core-purity.mjs:12`; the file
walk is rooted there, `:23-34`). A `plugin/mcp/proxy.ts` using `node:child_process`,
`node:net`, or `fetch` is entirely outside that scan path. (Notably the purity RULES ban the MCP
SDK, Bun globals, and harness APIs but **do not** ban process/socket/fetch primitives,
`plugin/scripts/check-core-purity.mjs:15-21` — but that is moot here, since the proxy is not in
`core/` at all.) The proxy will import the MCP SDK as a client, which is already legitimate in
`mcp/` (`server.ts:12-17` imports it today).

### 3.6 Fail-loud + mount signal — one aggregate TS line, native vs. proxied, plus forwarded child signal

**Question.** How does the TS server's own stderr signal (mirroring the Go server's
`deskkit mcp-serve: mounted …` line, `docs/tool-surface.md:88-90`) report the proxy's mount state
alongside its four native tools, so a caller can tell native from proxied and whether the proxy
mounted?

**Recommendation.** The TS server emits **one aggregate stderr mount signal** at startup, in the
Go server's shape (`librarian/internal/core/mcp/server.go:217-224`), naming both slices — e.g. on
success:

```
desk-standard-plugin: mounted 0.1.0; native 4: profile_get, profile_validate, template_render, knowledge_index; proxy deskkit mcp-serve (modules: librarian) 5: sweep, patrol, propose_fix, query, record_feedback
```

and on proxy failure:

```
desk-standard-plugin: mounted 0.1.0; native 4: profile_get, profile_validate, template_render, knowledge_index; proxy deskkit mcp-serve UNAVAILABLE (deskkit not found on PATH) — 0 proxied; install deskkit and restart the session
```

Rules:

- **Stderr only** — stdout is the JSON-RPC channel; a stray byte corrupts the protocol (same
  invariant as `librarian/internal/core/mcp/server.go:134-138`).
- **Native vs. proxied is explicit** in the line, so a reader can attribute each tool.
- The child's **own** `deskkit mcp-serve: mounted …` signal (and any fail-loud line) is
  **forwarded** to the host log too (§3.2), so both the aggregate view and the child's
  first-hand signal are visible. Two lines is intentional legibility, not noise.
- The proxied-tool NAMES in the signal come from the child's `tools/list` response, never a
  hardcoded list — the signal stays faithful to the Go gate the same way the Go signal is fed the
  same `ExposedTools(cfg)` it registers (`docs/tool-surface.md:88-90`).

## 4. Fallback trigger (Deliverable B; ADR 0016 escape hatch)

**What counts as "the design surfaced a blocker."** Any of:

1. **The host forbids the spawn.** If a Claude Code plugin's stdio MCP server cannot spawn a
   grandchild subprocess (a host sandbox/permission constraint), the entire in-process-proxy model
   (§3.1 Option A) is dead and Options B/C do not rescue it (B collapses to A; C needs Go transport
   that does not exist). This is the most plausible real blocker and should be probed **first** in
   implementation, before any other slice.
2. **Lifecycle coupling that cannot be made safe** — e.g. child processes that cannot be reliably
   reaped and accumulate across sessions, or a respawn path that cannot avoid a storm.
3. **No fail-loud that preserves the native tools** — if, contrary to §3.2, the only way to signal
   proxy failure is to take the whole TS server down (regressing the four native tools), the
   contract and the surface are in irreconcilable tension.

**The fallback path is fixed, not improvised.** On any blocker above, **reopen ADR 0016 with
Option A (amend-spec-to-reality)** (`docs/decisions/0016-ts-boundary-deskkit-proxy.md:37-38`):
correct the spec's promise to describe the shipped **four-tool** TS boundary and route
librarian-tool access **only** through the Go `mcp-serve`/CLI surfaces, dropping the proxy plan.
Record it in the ADR; do not invent a workaround (e.g. a TS reimplementation of librarian tools —
explicitly banned by ADR 0016, "never by reimplementing librarian logic in TS,"
`docs/decisions/0016-ts-boundary-deskkit-proxy.md:23-24`).

## 5. Build slices (ownable implementation units)

A future implementation issue can pick these up without rediscovery. Ordered by dependency;
slice 0 is the go/no-go probe.

| # | Slice | Owned files | Notes / gate |
|---|---|---|---|
| **0** | **Spawn-feasibility probe** | *(spike, no owned prod file)* | Confirm a Claude Code plugin stdio MCP server may spawn `deskkit` as a subprocess in the target host. If not → §4 blocker → Option A fallback. **Do this first.** |
| **1** | Proxy module | **new** `plugin/mcp/proxy.ts` | Spawn `deskkit mcp-serve` with `MCP_MODULES=librarian` (+ passthrough `LIBRARIAN_AUTONOMOUS_WRITES`); MCP **client** handshake (`initialize`→`tools/list`); `CallTool` forwarding; lifecycle/cleanup (§3.1); fail-loud + child-stderr forwarding (§3.2). Harness-pure boundary respected by living in `mcp/`, not `core/` (§3.5). |
| **2** | Server wiring | `plugin/mcp/server.ts` **(edited — out of scope for #122)** | Merge native `TOOLS` with the proxy's tool list in `ListTools`; route `CallTool` to native handler or proxy by name; emit the aggregate mount signal (§3.6). |
| **3** | Claiming skill | **new/extended** under `plugin/claude-plugin/skills/` (e.g. `desk-librarian/SKILL.md`, or `conventions-standard`) | Claim the 5 proxied librarian tools so none is unclaimed (§3.4; ADR 0014). **Must ship in the same wave as slices 1–2.** |
| **4** | Tool-surface truth | `docs/tool-surface.md` **§3 + Summary**, and the `tool-surface-drift-guard` (tracked **separately**) | Update surface-3 count from 4 to 4 + proxied (5/6); extend the drift guard to count the proxied tools. This issue only **notes** the guard must absorb them — it does not implement/extend the guard here. |
| **5** | Bundle regeneration | generated `plugin/claude-plugin/mcp/server.js` via `make package` | The marketplace copies only `plugin/claude-plugin/`, so `proxy.ts` **must be bundled into** the self-contained `server.js`; the existing bundle drift guard (`git diff --exit-code`) then covers it. Verify the packaging step includes the new module. |
| **6** | Spec reconciliation *(optional, same wave)* | `docs/pocket-librarian-v1-spec.md` §7.2 | Move the TS-boundary promise from "planned" to "shipped" once the proxy lands (ADR 0016 kept it "planned" until then, `docs/decisions/0016-ts-boundary-deskkit-proxy.md:36`). |

CHANGELOG and version-sync do not fire for *this* docs-only issue; the CHANGELOG entry belongs to
the implementation issue that ships the proxy.

## 6. Open questions (owners named)

- **Host spawn permission (blocking).** Does the Claude Code plugin host permit a plugin's stdio
  MCP server to spawn `deskkit` as a subprocess? — *Owner: the slice-0 implementer.* This gates
  the whole design (§4); resolve before slices 1–2.
- **Claiming-skill shape.** Dedicated `desk-librarian` skill vs. folding the 5 tool claims into
  `conventions-standard`. — *Owner: the skill-author of slice 3*, with the plugin-skills reviewer.
  Recommendation leans dedicated (§3.4); either satisfies ADR 0014's no-unclaimed-tool rule.
- **`apply_fix` over the proxy at all.** §3.3 passes `LIBRARIAN_AUTONOMOUS_WRITES` through, which
  would let `apply_fix` surface through the TS mount when the operator opts in. Whether the
  desk-standard bundle should *ever* expose the byte-exact file-writing tool over the plugin mount
  (vs. keeping `apply_fix` a Go-mount / CLI concern only) is a policy call. — *Owner: the ADR 0014
  write-gate owner.* Default recommendation: pass through (single source of truth), but flag for
  explicit sign-off given the blast radius.
- **Two mount signals in the host log.** §3.6 forwards the child's signal *and* emits the
  aggregate — confirm the host's MCP log renders both legibly rather than interleaving them
  confusingly. — *Owner: slice-2 implementer,* verified against a live session.

## 7. Citations (verified against this tree)

Current-behavior claims, each spot-checked at the cited line:

- `plugin/mcp/server.ts:18` — TS server imports `TOOLS` from `../core/index.js`.
- `plugin/mcp/server.ts:29-35` — `ListTools` maps core tools 1:1.
- `plugin/mcp/server.ts:37-54` — `CallTool` dispatch; thrown handler error → `isError` result
  (`:48-53`).
- `plugin/mcp/server.ts:59-63` — serves over `StdioServerTransport`.
- `plugin/core/tools.ts:240` — the fixed four-tool `TOOLS` array.
- `plugin/claude-plugin/.mcp.json:1-6` — bundle mounts only the TS server
  (`bun ${CLAUDE_PLUGIN_ROOT}/mcp/server.js`).
- `plugin/desk-pm/.mcp.json:1-10` — desk-pm mounts `deskkit mcp-serve` with `PM_ENABLED=true`,
  `MCP_MODULES=pm`.
- `plugin/scripts/check-core-purity.mjs:12` — purity gate scans only `plugin/core/`
  (`coreDir = ../core`); `:15-21` — RULES ban SDK/Bun/harness imports, not process/net/fetch;
  `:23-34` — the file walk is rooted at `coreDir`.
- `scripts/check-neutrality.mjs:52` — neutrality scan scoped to `plugin` / `librarian` / `kits`;
  `docs/` exempt.
- `docs/tool-surface.md:59-92` — surface-2 gate table (5/6/17/18) + mount signal (`:88-90`);
  `:94-125` — §2.1 `MCP_MODULES` third axis; `:117-120` — desk-pm mount = 12 PM tools only;
  `:128-142` — surface 3 = fixed 4.
- `docs/agent-integration-contract-v1-spec.md:79-90` — module gating as a mount contract
  parameter; `MCP_MODULES=librarian` excludes PM even under `PM_ENABLED`. `:100` — desk-pm mount
  owns the 12 PM tools. `:131-140` — the three TS tools are no-persona-by-design; `template_render`
  skill-claimed.
- `docs/decisions/0016-ts-boundary-deskkit-proxy.md:23-26` — proxy is a scoped work item preceding
  implementation; `:34-35` — harness-purity + fail-loud apply; `:36` — "planned" until it ships;
  `:37-38` — fallback = reopen with Option A.
- `docs/decisions/0014-agent-integration-contract.md:29-30` — tool-level gating so no tool is
  unclaimed; `:30-31` — fail-loud + stderr mount signal applies to every mount.
- `plugin/claude-plugin/skills/brownfield-adoption/SKILL.md:42-52` — deskkit must be on PATH,
  wires at session start, silent-drop is the failure being replaced, stderr signal is the
  diagnostic.
- `plugin/claude-plugin/skills/desk-setup/SKILL.md:8,68` — `template_render` is skill-claimed.
- `librarian/internal/core/mcp/server.go:97-101, 226-234` — stdin-EOF is a clean shutdown;
  `:112-115` — `requireResolvedConfig` fail-loud `os.Exit(1)`; `:123-128` — `MCP_MODULES` gate
  fail-loud; `:134-139` — stderr-only mount signal + stdio transport; `:145-186` — the three
  `MCP_MODULES` cases; `:217-224` — mount-signal format.
- `librarian/internal/modules/librarian/tools/specs.go:23` — the module name `librarian`.
- `_meta/research/2026-07-design-session/decision-book/D7-spec-reality-reconciliation.md` — D7
  finding 4 (the existing TS tools are under-claimed) — the precedent the claiming-skill slice
  (§3.4) exists to avoid repeating.

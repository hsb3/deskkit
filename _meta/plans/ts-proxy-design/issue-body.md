> **Tracking:** #122, ADR 0016 (2026-07-20 design session). Design — not implement — how the TS
> `plugin/mcp` boundary gains server-backed capabilities by proxying deskkit's Go `mcp-serve`,
> so a builder can pick up the implementation cold once this design is reviewed.

## Problem

ADR 0016 rules that the TS `plugin/mcp` boundary "gains server-backed capabilities by **proxying
deskkit's Go `mcp-serve`** — never by reimplementing librarian logic in TS," and that "the proxy
design (process lifecycle, availability behavior, which tools surface, gating per ADR 0014) is a
**scoped work item** that precedes any implementation"
(`docs/decisions/0016-ts-boundary-deskkit-proxy.md:23-26`). No such design exists yet — today the
TS server and the Go server are two structurally separate, un-composed mounts (see below); this
issue is the design pass ADR 0016 calls for, producing a document a builder can implement from,
not the proxy itself.

Current shipped reality, ground-truthed directly:

- `plugin/mcp/server.ts` is a thin adapter over the harness-pure TS core: it imports `TOOLS` from
  `../core/index.js` (`plugin/mcp/server.ts:18`), maps them 1:1 in its `ListTools` handler
  (`plugin/mcp/server.ts:29-35`), and dispatches `CallTool` to each tool's own handler
  (`plugin/mcp/server.ts:37-54`) — it re-declares no schema and calls no other process. The four
  tools it serves are `profile_get`, `profile_validate`, `template_render`, `knowledge_index`
  (`plugin/core/tools.ts:240`, confirmed against `docs/tool-surface.md` §3) — there is no proxy
  code today.
- The Go `mcp-serve` surface is mounted **separately today, by a different plugin bundle**, not
  by the TS server: `plugin/desk-pm/.mcp.json` starts `deskkit mcp-serve` directly as its own
  stdio server (`command: "deskkit"`, `args: ["mcp-serve"]`, `env: {"PM_ENABLED": "true"}`).
  `plugin/claude-plugin/.mcp.json` mounts only the TS server (`command: "bun"`,
  `args: ["${CLAUDE_PLUGIN_ROOT}/mcp/server.js"]`). These are two independent stdio processes
  today, composed only by both being present in a session's MCP config — neither calls the
  other. A proxy design has to decide whether `plugin/mcp/server.ts` itself starts spawning or
  calling the Go binary at runtime (an in-process proxy, matching ADR 0016's "the TS boundary
  gains ... by proxying" language) or some other composition — the former is this issue's
  working assumption; flag if a reviewer disagrees.
- `deskkit mcp-serve`'s existing operational contract (the #79 precedent) is documented in the
  `brownfield-adoption` skill's prerequisites: `deskkit` must be installed **and resolvable on
  the session's PATH**, and **MCP servers wire at session start** — "a mid-session install does
  **not** mount the surface... the PM tools stay absent until the next session"
  (`plugin/claude-plugin/skills/brownfield-adoption/SKILL.md:42-49`). The server "prints a
  one-line signal to stderr... naming the tools it exposed — its **absence** is the diagnostic"
  (`plugin/claude-plugin/skills/brownfield-adoption/SKILL.md:50-52`; the same signal is described
  from the Go side in `docs/tool-surface.md:88-90`). ADR 0014 generalizes this as a standing
  rule: "fail-loud + stderr mount signal (the #79 precedent) applies to every mount"
  (`docs/decisions/0014-agent-integration-contract.md:30-31`), and ADR 0016 says it "applies to the
  proxied surface" too (`docs/decisions/0016-ts-boundary-deskkit-proxy.md:35`) — any proxy design
  inherits this contract, it does not get to invent a quieter failure mode.
- Harness-purity: `plugin/scripts/check-core-purity.mjs` (wired into `make check` via
  `cd plugin && bun run check:purity`) scans `plugin/core/` non-test source and fails on MCP SDK
  imports, Bun-specific globals/builtins, and OpenCode/Claude-hook API imports
  (`plugin/scripts/check-core-purity.mjs:15-21`). It does **not** ban `node:child_process`,
  `node:net`, or `fetch` — a subprocess-spawn or HTTP-call proxy mechanism would not by itself
  trip this gate. ADR 0016's own consequence line resolves where the proxy code should live
  regardless: "the proxy honors harness-purity: `plugin/core` stays pure; the proxy lives at the
  server boundary" (`docs/decisions/0016-ts-boundary-deskkit-proxy.md:34-35`) — i.e. proxy code
  belongs in `plugin/mcp/` (the adapter layer `server.ts` already occupies), not `plugin/core/`,
  which sidesteps the purity gate's scan path entirely rather than requiring it to police a new
  exemption.
- Fallback, if this design hits a blocker (ADR 0016's own escape hatch): "If the proxy design
  surfaces a blocker (e.g. lifecycle coupling), the fallback is ADR 0016 re-opened with Option A
  (amend-to-reality only) — record it, don't improvise"
  (`docs/decisions/0016-ts-boundary-deskkit-proxy.md:37-38`). Option A means: correct the spec's
  promise to describe the shipped four-tool TS boundary and route librarian-tool access only
  through the Go `mcp-serve`/CLI surfaces, dropping the proxy plan rather than forcing a broken
  design through.

## Deliverables

- A — a design document answering the named questions below, reviewed before any implementation
  PR opens. *Recommended target (default, open to change):* `docs/development/ts-proxy-design.md`
  — `docs/development/` already holds contributor-facing design/survey docs in this shape (e.g.
  `docs/development/chat-tui-ux-survey.md`), and this is a design pass, not a new architectural
  ruling, so it does not need to be a new ADR. *Open alternative:* if the design materially
  amends ADR 0016 itself (rather than elaborating it), fold the outcome into ADR 0016 as a
  dated amendment instead of a separate doc — keep this contested, default to the separate
  design-doc path first.
- B — the design must explicitly answer or explicitly defer (with a stated reason) each of:
  - **Process lifecycle** — does `plugin/mcp/server.ts` spawn `deskkit mcp-serve` as a
    long-lived child process at its own startup, spawn it lazily on first proxied `CallTool`, or
    call an already-running `deskkit serve`/`mcp-serve` over some IPC? Who owns start/stop, and
    what happens to the child if the TS server exits or crashes (orphan process risk)?
  - **Availability behavior** — what happens when `deskkit` is not installed, not on PATH, or the
    spawned process fails to start? Per the #79 precedent and ADR 0014, this must fail loud with
    a stderr mount signal (`docs/decisions/0014-agent-integration-contract.md:30-31`), not silently
    drop the proxied tools the way a PATH-less `deskkit` today silently drops the desk-pm mount
    (`plugin/claude-plugin/skills/brownfield-adoption/SKILL.md:44-45`) — name the exact signal
    text and where it's checked.
  - **Which tools surface** — does the TS boundary proxy the librarian MCP server's full gated
    set (5/6/17/18 per `docs/tool-surface.md` §2), a fixed named subset, or something
    configurable? State the default gate state (does `PM_ENABLED`/
    `LIBRARIAN_AUTONOMOUS_WRITES` pass through from the TS proxy's own environment, or does the
    proxy impose its own default independent of the Go server's env?).
  - **Gating per ADR 0014** — ADR 0014 requires "tool-level gating so tools no persona claims are
    unreachable" (`docs/decisions/0014-agent-integration-contract.md:29-30`). Name which Claude
    Code skill(s) would claim the proxied tools (today, 3 of the 4 existing TS tools are already
    unclaimed by any skill —
    `_meta/research/2026-07-design-session/decision-book/D7-spec-reality-reconciliation.md`
    citing `surface-matrix.md § 6` finding 4 — so the proxy must not repeat this at larger scale
    without a matching skill-instruction update).
  - **Harness-purity boundary** — confirm proxy code lives in `plugin/mcp/` (or a new
    `plugin/mcp/proxy.ts`-style module beside `server.ts`), never in `plugin/core/`, per the
    reasoning above; name the exact new file(s).
  - **Fail-loud + mount-signal rules** — how the TS server's own stderr signal (mirroring the Go
    server's `deskkit mcp-serve: mounted ...` line, `docs/tool-surface.md:88-90`) reports the
    proxy's mount state alongside its existing four native tools, so a caller can tell which
    tools are native vs. proxied and whether the proxy mounted successfully.
  - **Fallback trigger** — state plainly what would count as "the design surfaced a blocker" per
    ADR 0016's escape hatch, and confirm the fallback path (reopen ADR 0016, Option A,
    amend-to-reality) rather than improvising a workaround.
  - **Build slices** — name the ownable implementation units the design implies (e.g.: proxy
    process-management module; tool-list merge/re-export; new skill-instruction updates to claim
    the proxied tools; `docs/tool-surface.md` + its drift guard (`tool-surface-drift-guard`,
    tracked separately) extended to count the new tools; `make package` regeneration of
    `plugin/claude-plugin/mcp/server.js` once the proxy ships, already covered by the existing
    bundle drift guard).
- C — a review pass on the design doc (an independent read against ADR 0016's consequences
  section and ADR 0014's gating rules) before any slice above is picked up for implementation.

## Acceptance criteria

- [ ] The design is recorded in a committed doc (default `docs/development/ts-proxy-design.md`;
      note in the doc if a different target was used instead).
- [ ] Each named question in Deliverables B is either answered in the doc or explicitly marked
      deferred with a stated reason (no silently-dropped question).
- [ ] The fallback trigger (what counts as "a blocker") is stated in the doc, along with
      confirmation that the fallback path is reopening ADR 0016 with Option A — not an
      improvised alternative.
- [ ] The doc names concrete, ownable build slices (per Deliverables B's last bullet) that a
      future implementation issue can pick up without rediscovery.
- [ ] The design doc has had an independent review pass and any resulting corrections are
      folded in before this issue closes.
- [ ] No code under `plugin/core/` or `plugin/mcp/server.ts` changes as part of this issue —
      this issue's output is a document, not an implementation (see Out of scope).

## Dependencies & gates

- **Blocked by:** agent-integration-contract (#114) — ADR 0014's mount/gating rules
  (persona instructions, tool mount, wake layer, write-gate policy — one contract, each agent
  instantiates it, `docs/decisions/0014-agent-integration-contract.md:20-24`) are a direct input
  to "which tools surface" and "gating per ADR 0014" above; this design should read that issue's
  outcome before finalizing those two answers, though drafting can start in parallel.
- Docs-only until a follow-up implementation issue is opened from this design's build slices —
  no code gate fires from this issue alone.
- `make check`, `make test`, `make verify`, bundle drift (`make package`), identity-neutrality,
  version-sync, and kit-drift do **not** fire for this issue's own change if the deliverable stays
  a `docs/` file — none of `plugin/`, `librarian/`, `kits/`, `VERSION`, or a shipped manifest is
  touched. Identity-neutrality (`scripts/check-neutrality.mjs`) in particular is scoped to
  `plugin/ + librarian/ + kits/` only and does not scan `docs/`
  (`scripts/check-neutrality.mjs:52`; confirmed in `_meta/plans/_config.md`'s gate menu) — so
  this design doc does not need identity-neutral phrasing to pass CI, though it should stay
  consistent with the rest of the repo's docs.
- CHANGELOG: a design-only doc is arguably not itself a "product change" per
  `_meta/plans/_config.md`'s CHANGELOG gate row; defer the CHANGELOG entry to whichever
  implementation issue actually ships the proxy.

## Out of scope

- Implementing the proxy — no changes to `plugin/core/`, `plugin/mcp/server.ts`, or any new
  proxy module land under this issue; that is separate, follow-up work sliced from this design.
- Changing the four shipped TS tools (`profile_get`, `profile_validate`, `template_render`,
  `knowledge_index`) — they stay exactly as they are; the design only adds new, proxied tools
  alongside them.
- The `tool-surface-drift-guard` issue (tracked separately) — this issue notes that guard will
  need to absorb the proxy's tools once built, but does not implement or extend that guard here.
- Re-litigating ADR 0016's core decision (extend-via-proxy over amend-spec-to-reality) — that
  call is already made; this issue only designs the "how," with the stated fallback as the sole
  escape hatch if the "how" proves infeasible.

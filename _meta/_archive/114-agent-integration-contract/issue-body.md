> **Tracking:** #114, ADR 0014 (2026-07-20 design session). Write the agent integration contract
> down once, then make the shared `mcp-serve` mount, the in-binary loop, and the librarian prompt
> actually conform to it — so no tool is reachable that no persona claims.

## Problem

The librarian (in-binary Go eino loop) and PM (plugin-markdown agent) integrations grew as
different objects on different layers, and every agent surface today carries tools no
instruction claims. ADR 0014 rules that symmetry is a surface property of one shared contract
(persona instructions, tool mount, wake layer, write-gate policy); the contract is not written
down anywhere, and three conformance gaps are live in current source:

- **The shared mount rides along.** `deskkit mcp-serve` registers `toolcore.ExposedTools(cfg)`,
  the merged registry of every enabled module
  (`librarian/internal/core/mcp/server.go:69-81`; `librarian/internal/core/toolcore/toolcore.go:147-156`).
  With `PM_ENABLED` set, the desk-pm mount (`plugin/desk-pm/.mcp.json:1-9`, only `PM_ENABLED=true`)
  exposes 17 tools: the 12 PM tools plus the 5 default librarian tools
  (`sweep patrol propose_fix query record_feedback`), which neither the `pm-operator` allowlist
  (`plugin/desk-pm/agents/pm-operator.md:13-26`) nor any desk-pm skill claims (count corroborated
  by `docs/tool-surface.md:70`).
- **The in-binary eino loop gets PM tools with no PM discipline.** `buildTools` builds the eino
  slice from `toolcore.AgentTools(cfg)` (`librarian/internal/modules/librarian/agent/agent.go:107-109`),
  which iterates the whole merged registry (`toolcore.go:133-142`); under `PM_ENABLED`,
  `module.Register` merges PM's 12 tools into that registry
  (`librarian/internal/core/module/module.go:95`; PM `Enabled`/`Tools`
  `librarian/internal/modules/pm/module.go:63,78`). But the resolved system prompt only ever loads
  `librarian.system` (`agent.go:42-53`) — no `pm.system` is seeded anywhere.
- **The librarian prompt is stale even with PM off.** The embedded prompt
  (`librarian/templates/librarian-system-prompt.txt`, `//go:embed`'d at
  `librarian/templates/templates.go` into `SystemPrompt`) unconditionally lists `apply_fix` and
  `restore` in its tool block. `restore` is never in the agent slice under any config
  (`toolcore.go:147-156`, and it is neither `AgentDefault` nor `AgentGated`); `apply_fix` is
  absent unless `LIBRARIAN_AUTONOMOUS_WRITES=true` (default off). So a default desk's agent claims
  two tools it does not hold.

Two more surfaces are claimed by no persona at all: the PM manifest `import` seam has no CLI, MCP,
TUI, or console surface (`librarian/internal/modules/pm/importer/importer.go:1-20`; its only caller
is the test harness), and the PocketBase admin console at `/_/` is reachable by any superuser,
named by no skill or persona (`_meta/research/2026-07-design-session/surface-matrix.md` finding 6).
And on the TS plugin MCP server, three of the four tools — `profile_get`, `profile_validate`,
`knowledge_index` — are claimed by no shipped skill (only `template_render` is, via the `desk-setup`
and `brownfield-adoption` skills), so the ADR 0014 audit's "TS plugin tools" surface has an
unclaimed hole too.

## Deliverables

- **A - Contract spec section.** Write the contract down: the 5-parameter table (persona
  instructions source+body, tool mount, Claude Code packaging, wake layer, write-gate policy),
  each agent's instantiation, and each parameter's policy-vs-debt verdict, plus the #79 fail-loud
  + stderr mount-signal invariant every mount inherits. Target: a new standalone doc
  `docs/agent-integration-contract-v1-spec.md`, cited from `docs/pocket-librarian-v1-spec.md`,
  `docs/pm-system-v1-spec.md`, and the neutrality allowlist (see Open questions for the
  new-section-in-librarian-spec alternative). Ratify the two deliberate asymmetries (inverted
  write-gate defaults; no in-binary PM brain / no in-binary PM wake) as documented parameters.
- **B - Tool-level gating on the shared mount.** On `librarian/internal/core/mcp/server.go`, gate
  the exposed set so tools no persona claims are unreachable: a mount declares which module(s) it
  intends, and the server filters the merged registry by `ToolSpec.Module`
  (`toolcore.go:36`, the field already exists) to that set. Fail loud on an empty/unresolvable
  intended set (extend the `requireResolvedConfig` -> `os.Exit(1)` pattern, `server.go:102-105`);
  the existing `emitMountSignal` (`server.go:114,148-151`) then names the gated set. Update
  `plugin/desk-pm/.mcp.json` to declare the `pm` module so its mount drops the 5 librarian
  ride-along tools (17 -> 12).
- **C - Fix the stale librarian prompt.** Make `librarian/templates/librarian-system-prompt.txt`
  reflect the actually-gated slice (config-aware wording, or drop the two phantom `apply_fix` /
  `restore` lines). Per ADR 0015 the embed is the git-truth source; existing stores hold a stale
  `prompts` row until cleared (documented-ephemeral, not a migration). **In the SAME edit**, move
  the fenced prompt block this mirrors in `docs/pocket-librarian-v1-spec.md` (the `:1435-1464`
  block + the `:2316` Decisions bullet) so the two stay byte-identical — `prompt-governance` (#120)
  adds `scripts/check-prompt-drift.mjs` to pin them, and per the epic coordination rule **#114
  lands before #120** so that guard is built against the corrected pair.
- **D - Keep the eino `buildTools` slice librarian-only.** Filter the eino slice to
  `ToolSpec.Module == "librarian"` (PM `Name()` returns "pm", `pm/module.go:56`) so the in-binary
  loop never receives PM tools it has no prompt for (ADR 0014(c), decision-book option C1). The
  shared `toolcore` registry stays the single registry — filter it, never fork it.
- **E - Name owners for the remaining unclaimed surfaces: `import`, the admin console, and the
  three unclaimed TS plugin tools.** Document PM `import` as a supervised / D8-reserved maintenance
  path with no agent surface (matching `restore` / `findings dispose`), and the admin console as an
  acknowledged human maintenance surface, not an agent surface. Also account for the TS plugin MCP
  tools no skill claims — `profile_get`, `profile_validate`, `knowledge_index` (only
  `template_render` is claimed, via `desk-setup` + `brownfield-adoption`): document each as either
  claimed by a skill/persona or no-persona-by-design (a setup/validation seam the skills call, not
  an agent-reachable tool), so the audit finds zero unclaimed tools on the TS surface too. All
  recorded in the contract spec section (docs-only; see Open questions on an interim `import` CLI
  subcommand).

## Acceptance criteria

- [ ] `docs/agent-integration-contract-v1-spec.md` exists with the 5-parameter contract table,
      each agent's instantiation, and a policy-vs-debt verdict per parameter; it is cited from
      both build specs and added to the neutrality allowlist.
- [ ] With the desk-pm mount (`PM_ENABLED=true`, no other flag) the `tools/list` probe returns
      exactly the 12 PM tools and none of `sweep patrol propose_fix query record_feedback`; the
      stderr mount signal names that gated set. (Probe per `docs/tool-surface.md` "How the counts
      were derived".)
- [ ] A mount with an empty/unresolvable intended module set exits non-zero with an actionable
      stderr message (the #79 fail-loud pattern), never a degenerate surface. Default:
      `MCP_MODULES` UNSET = all-enabled modules mounted (today's behavior — the documented bare
      probe in `docs/tool-surface.md:135-146` keeps yielding 5/6/17/18); only an EXPLICITLY
      declared module set that is empty or unresolvable fails loud. Module declaration adds a
      third gate axis beyond the two env gates; the ADR-0016 tool-surface guard must model it (its
      current two-axis four-combination framing predates gating).
- [ ] `librarian/templates/librarian-system-prompt.txt` no longer claims a tool the default-desk
      agent lacks: on a default desk the prompt's tool list is a subset of `ExposedTools(cfg)`; a
      red-able regression test pins this. The mirrored spec block
      (`docs/pocket-librarian-v1-spec.md:1435-1464`) is updated in the same PR so #120's drift
      guard will pass.
- [ ] Under `PM_ENABLED=true`, `buildTools` returns zero tools whose `Module != "librarian"`; a
      test asserts the eino slice is librarian-only regardless of PM state.
- [ ] The contract spec section names an owner for `import`, the admin console, and the three
      unclaimed TS plugin tools (`profile_get` / `profile_validate` / `knowledge_index`);
      `docs/tool-surface.md` is updated so its counts match the gated mount surface.

## Dependencies & gates

- **Fires.** `make verify` (`librarian/verify.sh`) and `make test` (librarian `go test ./...` +
  the tool-count drift test in `librarian/cmd/deskkit/main_test.go`) for B/C/D; `make check`
  neutrality (`scripts/check-neutrality.mjs`, scope `plugin/`+`librarian/`) for any shipped-tree
  text touched; a manual `docs/tool-surface.md` update in the same PR (the ADR-0016 drift guard is
  not yet built; if the guard lands first, update its expected numbers too). Regression-test bar
  (#82): B/C/D each carry a red-able test.
- **Does NOT fire.** No DB migration (the prompt fix edits the `//go:embed` source, not an applied
  migration; the `prompts` row re-seeds per ADR 0015). No version-sync change (no manifest version
  bump). No kit-drift, no scaffold-frontmatter. The spec doc (A) and the `import`/console docs (E)
  are `docs/` — exempt from neutrality, no code gate.
- **Depends on.** ADR 0015 (`docs/decisions/0015-prompt-governance.md`) governs the instruction-source
  mechanism deliverable A names and the re-seed rule C relies on.
- **Lands before #120.** Slice C edits the prompt embed + its mirrored spec block
  (`docs/pocket-librarian-v1-spec.md:1435-1464` / `:2316`) that `prompt-governance` (#120) will
  pin with `scripts/check-prompt-drift.mjs`; #114 must merge first (epic coordination rule 2) or
  #120's guard fails against a stale pair.

## Out of scope

- The desk-persona bundle itself — its own issue (`desk-persona-bundle`), blocked by this contract.
- Per-module mounts (`--module=pm` as separate processes) — ruled out for now (ADR 0014
  Consequences); revisit only if tool-level gating proves insufficient.
- Chat-to-schema capture — rides this contract later as a new claimed tool.
- Promoting `import` to a real adoption surface — reserved for D8.
- A composed in-binary PM prompt (decision-book option C2) — this issue rules C1 (exclusion).

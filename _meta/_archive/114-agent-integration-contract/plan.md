---
title: "Agent integration contract — build plan"
type: spec
status: shipped
created: 2026-07-20
purpose: "Write the agent integration contract down once, then make the shared mount, the in-binary loop, and the librarian prompt conform to it."
notes: "Issue #114 (closed). ADR 0014 (D5, Accepted 2026-07-20); also binds ADR 0015."
---

# Plan - agent integration contract

_Build plan for the agent integration contract: write it down once, then make the shared mount,
the in-binary loop, and the librarian prompt conform to it._

Status: draft (2026-07-20)

## Tracking

- Issue: #114 (this plan's tracking issue)
- ADR: `docs/decisions/0014-agent-integration-contract.md` (D5, Accepted 2026-07-20)
- Also binds: `docs/decisions/0015-prompt-governance.md` (instruction-source mechanism)
- Evidence: `_meta/research/2026-07-design-session/decision-book/D5-agent-contract-and-parity.md`,
  `_meta/research/2026-07-design-session/agent-symmetry.md`,
  `_meta/research/2026-07-design-session/surface-matrix.md`, `docs/tool-surface.md`

## Problem (grounded in source)

ADR 0014 rules that the librarian and PM integrations are two instantiations of one contract
(persona instructions, tool mount, wake layer, write-gate policy), but the contract is unwritten
and three conformance gaps are live on `main`:

- The shared `mcp-serve` mount registers the merged registry of all enabled modules
  (`librarian/internal/core/mcp/server.go:69-81` -> `librarian/internal/core/toolcore/toolcore.go:147-156`);
  the desk-pm mount therefore rides along 5 librarian tools no persona claims
  (`plugin/desk-pm/.mcp.json:1-9`; allowlist `plugin/desk-pm/agents/pm-operator.md:13-26`;
  `docs/tool-surface.md:70`, PM_ENABLED -> 17).
- The in-binary eino loop builds its slice from the same merged registry
  (`librarian/internal/modules/librarian/agent/agent.go:107-109` -> `toolcore.go:133-142`), so under
  `PM_ENABLED` it holds 12 PM tools (`module.Register` merge `librarian/internal/core/module/module.go:95`;
  PM `Enabled`/`Tools` `librarian/internal/modules/pm/module.go:63,78`) while its prompt resolves
  only `librarian.system` (`agent.go:42-53`; no `pm.system` seeded).
- The librarian prompt is stale even with PM off: `librarian/templates/librarian-system-prompt.txt`
  lists `apply_fix` and `restore` unconditionally, but `restore` is never exposed and `apply_fix`
  needs `LIBRARIAN_AUTONOMOUS_WRITES` (`toolcore.go:147-156`).

Plus two personaless surfaces: `import` (`librarian/internal/modules/pm/importer/importer.go:1-20`,
test-harness caller only) and the admin console (`surface-matrix.md` finding 6).

Enabler already in place: `ToolSpec` carries a `Module` field (`toolcore.go:36`), so both the mount
gating and the librarian-only eino slice are a filter over the existing merged registry — no
struct change, no second registry, honoring the "filter the one registry, never fork it" invariant
(D5 decision criteria 5).

## Deliverables (file scopes + per-unit acceptance)

### A - Contract spec section

- **Scope:** new `docs/agent-integration-contract-v1-spec.md`; citations added in
  `docs/pocket-librarian-v1-spec.md` and `docs/pm-system-v1-spec.md`; allowlist entry in
  `scripts/check-neutrality.mjs` (allow-path, not scan-dir).
- **Content:** the 5-parameter contract table; each agent's instantiation; policy-vs-debt verdict
  per parameter; the two ratified deliberate asymmetries (inverted write-gate defaults; no
  in-binary PM brain / wake); the #79 fail-loud + stderr mount-signal invariant; the surface
  audit result (which tools each persona claims) as the reference the CI-check open question keys
  on.
- **Acceptance:** the doc exists with all five parameters and a verdict each; both build specs cite
  it; it is in the neutrality allowlist.

### B - Tool-level mount gating

- **Scope:** `librarian/internal/core/mcp/server.go` (gate the exposed set), a filter helper in
  `librarian/internal/core/toolcore/toolcore.go` (filter `ExposedTools` by `Module`),
  `plugin/desk-pm/.mcp.json` (declare the `pm` module), `docs/tool-surface.md` (counts).
- **Mechanism:** the mount declares intended module(s) (env var or serve flag); the server filters
  `ExposedTools(cfg)` to specs whose `Module` is intended; empty/unresolvable intended set ->
  `os.Exit(1)` with an actionable stderr message (extend `requireResolvedConfig`); `emitMountSignal`
  reports the gated set. Default: `MCP_MODULES` UNSET = all-enabled modules mounted (today's
  behavior — the documented bare probe in `docs/tool-surface.md:135-146` keeps yielding
  5/6/17/18); only an EXPLICITLY declared module set that is empty or unresolvable fails loud.
  Module declaration adds a third gate axis beyond the two env gates; the ADR-0016 tool-surface
  guard must model it (its current two-axis four-combination framing predates gating).
- **Acceptance:** desk-pm mount `tools/list` returns exactly the 12 PM tools; empty intended set
  fails loud; mount signal names the gated set.

### C - Librarian prompt staleness fix

- **Scope:** `librarian/templates/librarian-system-prompt.txt` (embed source); a regression test
  under `librarian/internal/modules/librarian/` asserting the prompt's tool list is a subset of
  `ExposedTools(cfg)` on a default desk.
- **Mechanism:** edit the git-truth embed (ADR 0015); document the existing-store re-seed reality
  (runtime rows are ephemeral).
- **Acceptance:** default-desk prompt claims no tool the agent lacks; red-able test pins it.

### D - Librarian-only eino slice

- **Scope:** `librarian/internal/modules/librarian/agent/agent.go` (`buildTools`), optionally a
  helper in `toolcore.go`; a test asserting the slice is librarian-only under `PM_ENABLED`.
- **Mechanism:** filter the eino slice to `ToolSpec.Module == "librarian"` (decision-book C1).
- **Acceptance:** under `PM_ENABLED=true`, `buildTools` returns zero non-librarian tools.

### E - import + admin-console owners

- **Scope:** docs only, in the contract spec section (A). No code.
- **Mechanism:** document `import` as supervised / D8-reserved (no agent surface) and the admin
  console as an acknowledged human maintenance surface.
- **Acceptance:** the contract section names an owner for each.

## Gate and contract hygiene

| Gate | Fires? | Why |
| ---- | ------ | --- |
| make check (neutrality + self-test) | yes | B/C/D touch shipped-tree text under plugin/ + librarian/ |
| make test (plugin + librarian go test) | yes | B/C/D behavior changes; tool-count drift test in main_test.go |
| make verify (librarian integration) | yes | B/C/D touch librarian/ |
| docs/tool-surface.md count truth (ADR 0016) | manual | manual `docs/tool-surface.md` update in the same PR (the ADR-0016 drift guard is not yet built; if the guard lands first, update its expected numbers too) |
| Regression-test bar (#82) | yes | B/C/D each carry a red-able test |
| DB migration discipline | no | prompt fix edits the go:embed source; prompts row re-seeds (ADR 0015) |
| Version sync | no | no manifest version bump |
| Bundle drift guard (claude-plugin) | no | no plugin/core, plugin/mcp, or schema change |
| Kits drift | no | kits/ untouched |
| CHANGELOG | yes | product change needs an Unreleased entry |

## Parallelism and landing order

- **Parallel wave (independent file scopes):**
  - A - contract spec doc (docs-only; no code dependency)
  - C - prompt fix (`librarian-system-prompt.txt` + the mirrored spec block + its test)
  - D - eino librarian-only slice (`agent.go` + its test)
  These three touch disjoint files and can land in any order.
- **Serialized after the contract is defined:**
  - B - mount gating depends on A naming which module a mount intends (the gating declaration is a
    contract parameter). B and D both filter the merged registry by `Module`; land D's filter first
    if a shared toolcore helper is introduced, so B reuses it rather than duplicating.
- **E** is folded into A (docs).
- **Landing order:** A (+ C, D in parallel) -> B -> update `docs/tool-surface.md` counts + CHANGELOG.
- **Shared-file caution:** `toolcore.go` is touched by both B and D if a filter helper is shared —
  serialize those two edits (one owner for the toolcore helper), do not fan them out in parallel.
- **Cross-issue (#120):** slice C also moves the mirrored prompt block in
  `docs/pocket-librarian-v1-spec.md` (`:1435-1464` + the `:2316` Decisions bullet) in lockstep with
  `librarian-system-prompt.txt`, because `prompt-governance` (#120) adds
  `scripts/check-prompt-drift.mjs` pinning the two byte-identical. #114 lands before #120 (epic
  coordination rule 2), so C's file scope includes that spec block, not just the `.txt` + test.

## Open questions (recommended defaults)

- **Spec target (A).** Standalone `docs/agent-integration-contract-v1-spec.md` vs a new section in
  `docs/pocket-librarian-v1-spec.md`. **Default: standalone** — the contract is cross-product
  (librarian + PM + TS plugin surfaces); a standalone doc gives the mount gating, the bundle, and
  the surface audit one citable home and keeps the librarian spec librarian-scoped. Alternative
  kept open: a `## Agent integration contract` section in the librarian spec if the maintainer
  prefers not to add a third citable spec path.
- **Mount intent declaration (B).** Env var (e.g. `MCP_MODULES=pm`) vs a `mcp-serve --modules` flag.
  **Default: an env var** — `.mcp.json` entries already pass config via `env` (see
  `plugin/desk-pm/.mcp.json`), so an env var needs no new flag plumbing and stays declarative in the
  bundle manifest. Default: `MCP_MODULES` UNSET = all-enabled modules mounted (today's behavior —
  the documented bare probe in `docs/tool-surface.md:135-146` keeps yielding 5/6/17/18); only an
  EXPLICITLY declared module set that is empty or unresolvable fails loud. Module declaration adds
  a third gate axis beyond the two env gates; the ADR-0016 tool-surface guard must model it (its
  current two-axis four-combination framing predates gating).
- **CI check for unclaimed tools.** ADR 0014 says unclaimed-tool findings become CI-checkable in
  principle. **Default: no new CI check in this issue.** Runtime gating (B) already makes unmounted
  tools unreachable, which is stronger than a lint; a persona-claims-vs-mounted-tools CI check needs
  a machine-readable persona->tool map that does not exist yet. Record the audit as the manual
  reference in the spec (A) and add the check when the desk-persona bundle lands (its persona files
  are the natural claim source). Open.
- **import owner (E).** Document supervised / D8-reserved (default) vs add an interim
  `deskkit pm import` CLI subcommand now. **Default: document only** — a CLI subcommand is a real
  surface promotion, which D5 decision criteria 4 reserves for D8. The subcommand stays an open
  option if the maintainer wants `import` reachable by a supervised human path before D8.
- **Prompt fix shape (C).** Config-aware prompt wording vs simply dropping the two phantom lines.
  **Default: drop the two lines** — simplest, and the agent learns its real slice from the tool
  list at run start; config-aware wording reintroduces the exact drift ADR 0015 wants guarded.

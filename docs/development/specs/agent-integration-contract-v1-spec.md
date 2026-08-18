# Agent integration contract — one contract, two instantiations

_Names the single cross-product agent-integration contract — persona instructions, tool mount,
Claude Code packaging, wake layer, write-gate policy — that the librarian (KM) agent and the PM
agent each instantiate, states which asymmetries between them are ratified policy versus debt,
and audits every tool-bearing surface against it._

Status: active
Date: 2026-07-20

---

## 1. The contract, in one line

Agent symmetry is a **surface property of a shared core**, not a requirement that the librarian
and PM integrations be identical implementations. `desk-standard` defines **one integration
contract** — five named parameters — and each agent (the librarian's in-binary eino loop; the
PM module's plugin-markdown operator) **instantiates** that contract. Where an instantiation
differs from its sibling, this document states whether the difference is **ratified policy** (a
deliberate, documented asymmetry that stands) or **debt** (a gap this issue closes). The ruling
is ADR 0014 (DESK-36); the prompt-sourcing mechanism
behind parameter 1 is ADR 0015 (DESK-37); the empirical tool
inventory this document audits against is [`docs/development/specs/tool-surface.md`](tool-surface.md).

## 2. The five-parameter contract table

| # | Parameter | Librarian (KM) instantiation | PM instantiation | Verdict |
|---|---|---|---|---|
| 1 | **Persona instructions** (source + body) | DB `prompts` row `librarian.system` — the git-embedded seed is truth (ADR 0015); the row is a re-seeded cache, live-read per run | Version-controlled plugin markdown (`pm-operator.md` + the three desk-pm skills); no `pm.system` row exists anywhere | Asymmetric **by ratified policy**: the contract names exactly **one** instruction source per surface (ADR 0014(d)) — a DB row for the librarian, markdown for PM, never both for one surface. **Debt this issue fixes:** the librarian prompt's phantom `apply_fix`/`restore` claims — it unconditionally names two tools the default-configuration agent does not hold (§5). |
| 2 | **Tool mount** (what is reachable) | `deskkit mcp-serve` over the merged `toolcore` registry; the in-binary eino loop reads the same registry | `plugin/desk-pm/.mcp.json` mounts the identical server binary with `PM_ENABLED=true` — same server, same registry | Shared mount **by ratified policy** (ADR 0014(b) keeps one `mcp-serve`, not per-module processes; per-module mounts are ruled out for v1). **Debt this issue fixes:** tool-level module gating, so a mount that intends only one module's tools does not also surface the other module's tools to a persona that never claimed them (§4). |
| 3 | **Claude Code packaging** | None shipped — the only Claude Code path is a hand-authored `.mcp.json` snippet in `librarian/README.md` | Full bundle: agent persona + 3 skills + SessionStart hook + `.mcp.json` + marketplace entry, under `plugin/desk-pm/` | Asymmetric **by ratified policy**: the composed **desk-persona bundle is the v1 proof surface** (ADR 0009 platform R1 decision 2; ADR 0014(a)) — it composes the librarian + PM module personas rather than existing as a third, separate instantiation. A dedicated `desk-librarian` bundle mirroring desk-pm's shape is explicitly ruled out for v1 and stays out of this issue's scope. |
| 4 | **Wake layer** | Hourly cron plus a claimer goroutine, registered only under `serve` — a one-shot CLI call never enqueues | No in-binary wake at all; the plugin's SessionStart hook is PM's wake, and it calls the CLI directly, not an MCP tool | **Deliberate asymmetry, ratified as policy** (ADR 0014) — the PM module registers tools, TUI views, hooks, and a realtime source, but no loop and no wake. This is not scheduled to converge. |
| 5 | **Write-gate policy** | `LIBRARIAN_AUTONOMOUS_WRITES` — default **off**; gates `apply_fix` at tool-registration time and again at execution time; `restore` is structurally excluded from every agent surface (never gated by a flag at all) | `PM_AUTONOMOUS_WRITES` — default **on**; gates the 8 PM write tools. Every PM write tool mutates only the store, never a desk file, so the phase machine's document gates — not this flag — are the real safety boundary | **Deliberate asymmetry, ratified as policy**, recorded in a code comment before this issue and reaffirmed by ADR 0014: inverted defaults reflect inverted blast radius (a byte-exact file write vs. a store-only, gate-checked write). |

## 3. The two ratified deliberate asymmetries

Two of the five parameters above are not merely "currently different" — the design session
named them explicitly as asymmetries that **survive as documented contract parameters, not as
debt to erase** (ADR 0014):

1. **Inverted write-gate defaults** (parameter 5). `LIBRARIAN_AUTONOMOUS_WRITES` defaults off
   because an ungated `apply_fix` performs a byte-exact write to a desk file; `PM_AUTONOMOUS_WRITES`
   defaults on because every PM write tool mutates only the store, and the phase machine refuses
   an illegal transition regardless of which flag is set. The two flags protect different blast
   radii — making them symmetric would either lock PM agents out of the graph by default or
   loosen the librarian's file-write default, and neither is wanted.
2. **No in-binary PM brain, no in-binary PM wake** (parameters 1 and 4 together). The PM module
   registers tools, TUI views, hooks, and a realtime source — never an agent loop. There is no
   `pm.system` prompt and no cron/claimer wake for PM inside the binary; PM's only "brain" is the
   plugin-markdown `pm-operator` persona, and its only wake is the plugin's SessionStart hook.
   This is deliberate, not an oversight: the PM engine is deterministic state-machine code, and
   giving it an in-binary loop would duplicate the `pm-operator` discipline into a second,
   harder-to-keep-in-sync instruction source — reopening parameter 1's "exactly one source"
   rule. The in-binary eino loop stays librarian-only (ADR 0014(c)); PM tools are excluded from
   its tool slice regardless of `PM_ENABLED`.

## 4. The mount invariant every mount inherits

Every mount this contract produces or gates — present or future — inherits the fail-loud plus
observable-mount-signal precedent already shipped for the librarian MCP server: an unresolved or
degenerate desk configuration MUST fail loud (`os.Exit(1)` with an actionable stderr message
naming what is unresolved), never register a silent or empty tool surface. On success, the mount
prints a one-line stderr **mount signal** naming the exact exposed tool set, e.g.:

```
deskkit mcp-serve: mounted "deskkit" v1; modules: all; 5 tool(s) exposed: sweep, patrol, propose_fix, query, record_feedback
```

Any new mount — a future per-module surface, a librarian-specific bundle, or any other shape —
reuses this pattern rather than re-deriving its own failure mode.

**Module gating is a contract parameter of the mount**, not merely an implementation detail of
parameter 2 (tool mount). A mount declares which module(s) it intends to expose via the
`MCP_MODULES` environment variable:

- **Unset** — every enabled module's tools are mounted (today's behavior, unchanged; equivalent
  to declaring all enabled modules).
- **Set, non-empty, resolvable** (e.g. `MCP_MODULES=pm`) — the mount filters the merged
  `toolcore` registry down to the declared module(s)' tools only. A librarian-only desk
  declaring `MCP_MODULES=librarian` never surfaces PM tools even when `PM_ENABLED=true`; a
  `desk-pm` mount declaring `MCP_MODULES=pm` never surfaces the 5 default librarian tools.
- **Explicitly declared but empty or unresolvable** (a typo'd or unknown module name, or an
  explicit empty set) — the mount fails loud (`os.Exit(1)`), inheriting the invariant above,
  rather than silently falling back to "all modules" or degenerating to zero tools.

This is the **third gate axis**, alongside the two existing env gates — `PM_ENABLED` (which
module registers its tools at all) and `LIBRARIAN_AUTONOMOUS_WRITES` / `PM_AUTONOMOUS_WRITES`
(which write tools a registered module exposes). `MCP_MODULES` controls which *registered*
module's tools a given **mount instance** actually surfaces, independent of what is registered
process-wide.

## 5. Surface audit — a verdict for every tool-bearing surface

For each surface: which tools it exposes (post-fix, where this issue changes the answer), which
persona's instructions claim those tools, and the verdict.

| Surface | Tools exposed | Claimed by | Verdict |
|---|---|---|---|
| **Librarian MCP mount** (`deskkit mcp-serve`, `MCP_MODULES` unset or declaring `librarian`) | 5 default (`sweep`, `patrol`, `propose_fix`, `query`, `record_feedback`) + `apply_fix` under `LIBRARIAN_AUTONOMOUS_WRITES`; `restore` never | The librarian system prompt (`librarian/templates/librarian-system-prompt.txt`), corrected to name exactly the gated slice it actually holds | **Claimed by the librarian persona.** Debt (the phantom `apply_fix`/`restore` claims) is fixed this issue per ADR 0015's mechanism. |
| **desk-pm MCP mount** (`plugin/desk-pm/.mcp.json`, declaring `MCP_MODULES=pm`) | Exactly the 12 PM tools (`get_context`, `list_items`, `get_item`, `create_item`, `update_item`, `transition_item`, `block_item`, `unblock_item`, `add_note`, `link_items`, `claim_item`, `release_item`); the 5 librarian tools are gated OUT by `MCP_MODULES` | `pm-operator.md`'s `tools:` frontmatter — already a clean 12-name claim, zero librarian names | **Claimed by the PM persona; the ride-along is resolved.** Before this issue the mount exposed 17 tools (the 12 PM tools plus 5 unclaimed librarian ride-alongs); module gating (§4) closes the gap without touching the persona. |
| **In-binary eino loop** (`deskkit agent` / `deskkit chat`, driving the chat TUI) | Librarian-only tool slice — PM tools are excluded from the loop's tool slice regardless of `PM_ENABLED` (ADR 0014(c)) | The (corrected) librarian system prompt | **Claimed by the librarian persona; no PM ride-along.** Before this issue, `PM_ENABLED` merged all 12 PM tools into the loop's own tool slice while its prompt stayed librarian-only — a stronger instance of the mount ride-along, since it is the model's own instructions going stale, not a mount an operator chose to attach. |
| **TS plugin MCP server** (`plugin/mcp/server.ts`) | Fixed 4: `profile_get`, `profile_validate`, `template_render`, `knowledge_index` — no env gate | `template_render` is claimed by the `desk-setup` and `brownfield-adoption` skills. The other three — see §6 | **Split verdict, resolved by design (§6):** `template_render` is claimed; `profile_get` / `profile_validate` / `knowledge_index` are **no-persona-by-design**, not an unclaimed gap. |
| **PM `import` seam** (`librarian/internal/modules/pm/importer/importer.go`) | A manifest → graph-seed importer; no CLI subcommand, no MCP tool, no TUI affordance, no admin-console path | No persona; its only caller today is the test/scenario harness | **No-persona-by-design (supervised, D8-reserved).** See §6 — matches `restore`/`findings dispose`'s supervised-only posture; not an oversight. |
| **Admin console** (PocketBase `/_/`, via `deskkit gui` / `deskkit serve`) | Raw record CRUD over every collection, gated only by superuser login — no per-operation gate | No skill, agent persona, or system prompt names it | **No-persona-by-design (acknowledged human maintenance surface).** See §6. |

*Surfaces out of this audit's scope by construction:* the librarian CLI (`deskkit <subcommand>`)
is the human/script owner surface and carries no persona claim by design — it is not model-facing
and is not subject to the write-gate flags at all (the CLI always has `apply-fix`, unlike the
model-facing surfaces). The chat TUI inherits the in-binary eino loop's tool slice and prompt
verbatim, so its verdict is identical to that row and is not repeated as a separate line.

## 6. Named owners for the previously-unclaimed surfaces

Three surfaces had no claiming persona at all before this issue. Each now has a named,
documented owner:

- **`import`** (`librarian/internal/modules/pm/importer/importer.go`) is documented as a
  **supervised, D8-reserved maintenance path** — the same posture as `restore` and
  `findings dispose`: real, tested, and load-bearing for rebuild-from-scratch reproducibility,
  but deliberately given **no agent surface** (no MCP tool, no CLI subcommand, no TUI
  affordance). Its only sanctioned caller today is the test/scenario harness; promoting it to an
  interactive adoption path is reserved for a future, not-yet-built maintenance track and is out
  of this issue's scope.
- **The admin console** (PocketBase `/_/`) is documented as an **acknowledged human maintenance
  surface, not an agent surface**. It is reachable by any superuser and gated by nothing beyond
  that login — no per-collection state-machine hook runs there, so a human operator can bypass
  the phase machine's document gates or the record-original-first fix pipeline by editing a
  record directly. This is accepted as a *convention*, the same posture as any other direct
  human file/GUI edit every desk already permits — it is not a gap this contract closes, and no
  persona should ever be instructed to use it.
- **The three TS plugin tools `profile_get`, `profile_validate`, and `knowledge_index`** are each
  documented as **no-persona-by-design**: they are a setup/validation seam the `desk-setup` and
  `brownfield-adoption` **skills** call as procedural steps in a runbook, not a tool set an agent
  is instructed to reach for autonomously. Only `template_render` is skill-claimed by name
  (`desk-setup/SKILL.md`, `brownfield-adoption/SKILL.md`), because template rendering is the one
  operation a skill's own instructions need to invoke mid-runbook; reading the active profile,
  validating it, or indexing the `_knowledge/` background files are steps a human or the skill's
  own procedural logic performs, not steps a persona autonomously decides to take. Under this
  framing the TS surface finds **zero unclaimed tools**, matching the librarian and PM surfaces
  above.

## 7. Citations

- ADR 0014 — the agent integration contract (DESK-36) —
  the ruling this document elaborates: one contract, five parameters, four sub-decisions (a)–(d).
- ADR 0015 — prompt governance (DESK-37) — the mechanism behind
  parameter 1 (persona instructions): the version-controlled source (Go embed / plugin markdown)
  is truth, and the DB `prompts` row is a re-seeded cache.
- [`docs/development/specs/tool-surface.md`](tool-surface.md) — the empirically-derived tool inventory and gate
  table this audit (§5) is built against.

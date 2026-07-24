---
type: analysis
status: draft
created: 2026-07-20
updated: 2026-07-20
tags: [design-session, agent-symmetry]
synopsis: Delta-verification of the exec-desk analysis §1 (librarian vs PM agent integrations are not mirror images) against post-PR-112 main — 14 of 15 claims still hold; only the #79 mount-failure claim materially changed (fix shipped).
---

_Grounding dossier for the pre-feature design session — an adversarial re-derivation of Section 1
("the two agent integrations are not mirror images") of the exec-desk analysis
`EXECUTIVE_DESK/Projects/desk-standard-desk/analyses/desk-standard-agent-symmetry-and-document-model-2026-07-20.md`,
which was written BEFORE PR 112 merged. Each claim is re-located in current source (`main` @ `51235f6`)
and marked CONFIRMED / STALE / CHANGED with a fresh `path:line`. Informs the session; does not itself
rule. Dossier claims are hypotheses — anything a ruling binds on gets re-derived in the session._

Status: draft (2026-07-20)

# Agent-integration symmetry — exec-desk §1 delta-verified

## Method note

The analysis cited line numbers from a pre-PR-112 tree; per the brief I did not trust them and
re-located every claim in current source. Where only the line moved (substance intact) the verdict is
CONFIRMED with the fresh citation. **Verdict counts: 14 CONFIRMED · 0 STALE · 1 CHANGED** (across the
7 comparison-table rows, 2 "defensible" asymmetries, and 6 "accidental/unruled" asymmetries).

## Verdict table — the 7-row comparison table

| Claim (analysis) | Analysis said | Current reality | Verdict | Citation (fresh) |
|---|---|---|---|---|
| **Brain** | Librarian = in-binary Go eino ReAct loop; PM = Claude Code markdown agent, no in-binary PM loop | Exactly so. The eino loop (`newAgent`/`Run`) is Go in the binary; the PM module contributes only Tools/TUIViews/Hooks/Realtime — no loop, no brain | CONFIRMED | `librarian/internal/modules/librarian/agent/agent.go:87-100,148-186`; `plugin/desk-pm/agents/pm-operator.md:31-35`; `librarian/internal/modules/pm/module.go:78-129` |
| **Tools** | Librarian 7, MCP-default 5; PM 12 (test-guarded) | Librarian `Specs()` returns exactly 7 (sweep, patrol, propose_fix, apply_fix, restore, query, record_feedback); AgentDefault=true on 5 → MCP-default 5. PM `Specs()`/`ToolNames()` = 12 | CONFIRMED | `librarian/…/librarian/tools/specs.go:22-72`; `…/core/toolcore/toolcore.go:133-156` (gate); `librarian/…/pm/tools/specs.go:27-104`, `ToolNames` 107-113 |
| **Instructions** | Librarian = embedded prompt seeded into DB `prompts` collection, DB-editable; PM = plugin markdown, no `prompts` row | `prompt.Seed` writes `templates.SystemPrompt` to `prompts` key `librarian.system` on first run; `systemPrompt` reloads the active row per run (GUI/REST-editable). PM instructions live in markdown; **no `pm.system` row anywhere** (grep negative) | CONFIRMED | `librarian/…/librarian/prompt/prompt.go:16-40`; `agent.go:42-53`; `plugin/desk-pm/agents/pm-operator.md` + `skills/*` |
| **Claude Code mount** | Librarian = none shipped, only a `.mcp.json` README snippet; PM = full bundle (agent + 3 skills + SessionStart hook + `.mcp.json` + marketplace entry) | No `desk-librarian` plugin exists; librarian's only Claude Code path is a hand-authored README snippet. desk-pm ships agent + 3 skills (pm-advance-item, pm-session-open, pm-triage) + SessionStart hook + `.mcp.json` + marketplace entry | CONFIRMED | `librarian/README.md:212-227`; `plugin/desk-pm/` tree; `.claude-plugin/marketplace.json:9-23` (lists desk-standard + desk-pm only) |
| **Write gate** | `LIBRARIAN_AUTONOMOUS_WRITES` default **off** (`config.go:112`); `PM_AUTONOMOUS_WRITES` default **on** (`config.go:122`) | Both defaults exactly as claimed. Line moved: PM gate is now `config.go:121` (analysis said 122) | CONFIRMED | `librarian/internal/core/config/config.go:112` (librarian off), `:121` (PM on) |
| **Safety model** | Librarian = record-original-first byte-exact, `restore` CLI-only; PM = store-only writes + document gates + phase machine + version tokens | `restore` is neither AgentDefault nor AgentGated → never in the agent/MCP slice (CLI-only); all 12 PM tools `WritesFiles=false` (store-only); `transition_item` refuses until phase docs validate; `update_item` version-checked | CONFIRMED | `specs.go:49-54` + `toolcore.go:144-156`; `pm/tools/specs.go:22-27,61-66`; `update_item` desc `pm/tools/specs.go:55-60` |
| **TUI** | Librarian = chat transcript is the view; PM = structured views via `Module.TUIViews` | Loop persists the transcript to the `messages` collection (the librarian's only view); PM contributes 3 structured views via `TUIViews` | CONFIRMED | `agent.go:141-180`; `librarian/internal/modules/pm/module.go:83-88` |

## Verdict table — the 2 "deliberate & defensible" asymmetries

| Claim (analysis) | Analysis said | Current reality | Verdict | Citation (fresh) |
|---|---|---|---|---|
| **Inverted write-gate defaults are ruled** | Code comment at `config.go:119-121` records the ruling (PM tools write only the store; the gates are the real safety); PM spec §5.1 prose says "mirroring the librarian pattern" while shipping the opposite default — the prose needs reconciling | The ruling comment is intact, moved to `config.go:118-120` (plus the struct-field doc at `:46`). Substance unchanged | CONFIRMED | `config.go:118-121` (ruling + gate); struct doc `:46` |
| **No in-binary PM brain / no PM TUI chat is deliberate** | The PM engine is deterministic state-machine code; the "agent" is genuinely just an operator over its tools | Confirmed by structure: the PM module registers tools/views/hooks/realtime but no loop; its TUI contribution is structured views, not a chat transcript | CONFIRMED | `pm/module.go:38-88` (no loop; `TUIViews` = structured views) |

## Verdict table — the 6 "accidental / unruled" asymmetries

| # | Claim (analysis) | Analysis said | Current reality | Verdict | Citation (fresh) |
|---|---|---|---|---|---|
| 1 | **No librarian Claude Code bundle** | PM spec D5 (skills/agent/hook/`.mcp.json`) has no librarian counterpart; librarian's only Claude Code path is a README snippet | Still true. Marketplace lists only `desk-standard` (the 4-profile-tool plugin) and `desk-pm`; no `desk-librarian`. Librarian Claude Code wiring is a README `.mcp.json` snippet | CONFIRMED | `.claude-plugin/marketplace.json:9-23`; `librarian/README.md:216-227` |
| 2 | **Ride-along tools on the PM mount** | `deskkit mcp-serve` exposes *every* enabled module's tools, so mounting desk-pm also surfaces ~5 librarian tools no persona claims | Still true. `NewServer` loops over `toolcore.ExposedTools(cfg)` = the merged registry of all enabled modules; with `PM_ENABLED` the mount exposes **17** (5 librarian default + 12 PM). Now *documented* (not resolved) in the new `docs/development/specs/tool-surface.md` | CONFIRMED | `core/mcp/server.go:69-81`; `toolcore.go:133-156`; corroborated `docs/development/specs/tool-surface.md` §2 (PM_ENABLED → 17) |
| 3 | **In-binary loop gets PM tools without PM discipline** | With `PM_ENABLED` the eino loop receives the 12 PM tools via the merged registry, but its resolved system prompt is librarian-only; pm-operator's read-before-write / gate-routing / claim-release discipline lives only in the plugin lane | Still true. `buildTools` → `toolcore.AgentTools(cfg)` returns the merged (librarian+PM) slice; `systemPrompt` resolves only `librarian.system` (no `pm.system` is ever seeded). The PM discipline is in `pm-operator.md`, not in any in-binary prompt | CONFIRMED | `pm/module.go:78-81` → `agent.go:107-117` → `toolcore.go:133-142`; prompt `agent.go:42-53`; discipline `pm-operator.md:52-55` |
| 4 | **Prompt-governance split** | Librarian instructions DB-backed/GUI-editable; PM instructions version-controlled markdown; neither model chosen for both | Still true and unchanged: two governance models, one per surface | CONFIRMED | `prompt.go:16-40` + `agent.go:42-53` (DB) vs `pm-operator.md` + `skills/*` (markdown) |
| 5 | **Spec-promised route that doesn't exist** | Spec says librarian tools are callable via the TS `plugin/mcp` boundary; the shipped TS server exposes only the 4 profile tools. Also stale librarian counts ("four tools", "six-tool core") vs the real 7/5 | Still true. Spec still promises the plugin/mcp boundary can "call the librarian's tools directly"; the TS server exposes only `TOOLS` from `plugin/core` = 4 profile tools. Spec counts still stale ("six-tool core" TOC/§3.3 table; "default set … four" in §7.2). #94 shipped a NEW authoritative doc (`docs/development/specs/tool-surface.md`) rather than editing the spec, so the spec residue persists | CONFIRMED | spec `docs/development/specs/pocket-librarian-v1-spec.md:1784-1790` (route), `:399` + TOC `:14` ("six-tool"), `:1789` ("four"); `plugin/mcp/server.ts:29-35` + `plugin/core/tools.ts:65,127,175,205` (4 tools) |
| 6 | **Mount failure modes diverge inside one bundle (= #79)** | The desk-pm SessionStart hook self-gates on `command -v deskkit`; the `.mcp.json` mount does not — #79, already 0.8.0 | **#79 fix SHIPPED in PR 112.** The hook still self-gates (silent no-op if binary absent). The `.mcp.json` is still a static mount, but the divergence is now addressed on the *server* side, not by symmetric self-gating: `mcp.Serve` calls `requireResolvedConfig(cfg)` and `os.Exit(1)` with an actionable "desk not resolved…" message when config doesn't resolve (fail-loud, not silent-absent), and emits a one-line **mount signal** to stderr naming the exposed tool set on success | CHANGED | hook `plugin/desk-pm/hooks/session-briefing.sh:21`; static mount `plugin/desk-pm/.mcp.json`; fix `core/mcp/server.go:92-119` (fail-loud 102-105, mount signal 114), `requireResolvedConfig` 126-142, `emitMountSignal` 148-151 |

## New since the analysis (PR 112 — bears on the symmetry decision)

- **#79 fail-loud serve + mount signal (the one material delta).** `deskkit mcp-serve` no longer
  registers a degenerate surface against an unresolved desk: `requireResolvedConfig` → `os.Exit(1)`
  with a named-identity, actionable message; and on success `emitMountSignal` prints
  `mounted "deskkit" v1; N tool(s) exposed: …` to **stderr** (stdout is the JSON-RPC channel). The
  design session's "apply #79's ruling to any future librarian bundle" agenda item should treat #79
  as *resolved by fail-loud + observable mount*, not as an open self-gating gap. Note the residual:
  the fail-loud only fires once the binary runs — a *missing* `deskkit` binary named in `.mcp.json`
  still fails at the host level (the hook's `command -v` guard has no server-side twin).
  `core/mcp/server.go:92-151`.
- **`docs/development/specs/tool-surface.md` is now the authoritative surface inventory (#94).** Status `active`,
  dated 2026-07-20, counts derived empirically (JSON-RPC `tools/list` probe, not by reading source):
  Librarian CLI 16 base (+`pm` group under `PM_ENABLED`); Librarian MCP 5 / 6 / 17 / 18 by env combo;
  Plugin TS server 4. It documents the ride-along (asymmetry #2) explicitly but does not resolve it,
  and it does **not** correct the spec's stale "six-tool core" / "four default" language (asymmetry #5
  residue). `docs/development/specs/tool-surface.md:1-152`.
- **Ride-along is now countable, not just implied.** The tool-surface table quantifies the
  merged-mount blast radius (PM_ENABLED → 17 tools on one server; both flags → 18), which sharpens the
  session's "per-module vs shared mount" decision (design-prep C1). `docs/development/specs/tool-surface.md:64-70`.

## Gaps & uncertainties

- **Comparison-table row count.** The brief referenced an "8-row comparison table"; the analysis table
  has **7 dimension rows** (Brain, Tools, Instructions, Claude Code mount, Write gate, Safety model,
  TUI) under a header row — 8 counting the header. All 7 data rows are verified above.
- **Config line-number drift.** The write-gate lines moved by one (PM gate `:121`, not `:122`; ruling
  comment `:118-120`, not `:119-121`). Substance is identical; flagged only so the session doesn't
  re-flag it as a discrepancy.
- **Spec line for the plugin/mcp promise.** The analysis cited `spec:1775-1776`; the actual sentence is
  at `~:1784-1790` in current main (the outbound-MCP paragraph). Same claim, re-located.
- **pm-operator discipline vs in-binary prompt — verified asymmetrically.** I confirmed the load-bearing
  half directly (the in-binary prompt resolves only `librarian.system`; no `pm.system` is seeded) and
  spot-confirmed pm-operator carries read-before-write/claim/gate discipline (`pm-operator.md:52-55`).
  I did not exhaustively diff the two instruction bodies clause-by-clause — if the session rules on a
  composed/per-module in-binary prompt, re-derive the exact discipline delta then.
- **"record_feedback" makes the librarian README itself stale.** `librarian/README.md:214-215` lists
  the mcp-serve default as `sweep, patrol, propose_fix, query` (4) — it omits `record_feedback`, so the
  real default is 5 (as `docs/development/specs/tool-surface.md` and `specs.go` confirm). This is a second stale-count
  site beyond the spec ones the analysis named; minor, noted for the spec-reconcile lane, not fixed here.
- **Marketplace "entry" for desk-pm.** Confirmed desk-pm is listed in `.claude-plugin/marketplace.json`
  and carries its own `.claude-plugin/plugin.json`; I did not exercise an actual `claude plugin install`
  to prove the bundle mounts end-to-end (out of scope for a source delta-check).

---
type: analysis
status: draft
created: 2026-07-20
updated: 2026-07-20
tags: [design-session, decision-book, agent-symmetry, access-surfaces, surface-parity, 1.0.0]
synopsis: D5 decision brief — one integration contract (instructions + tool mount + wake layer +
  write-gate policy), two instantiations (KM/librarian and PM). Packages four sub-decisions
  (librarian bundle · per-module vs shared mount · in-binary PM prompt vs PM-tool exclusion ·
  instruction source per surface) plus the full Phase-0 unclaimed-surface set. Informs the
  session; does not rule.
---

_Phase-1 decision book (`../../README.md` §2, index `./README.md`). Layer: **surface**;
depends on **D1–D4** — the model rulings say what every surface must expose, so parity is
decidable only once they land. Evidence resolves to the Phase-0 dossiers beside the prep doc;
dossier claims are hypotheses until re-derived where a ruling binds. No ruling here —
hypotheses are labelled._

Status: draft (2026-07-20)

> **Platform-stream interaction (2026-07-20 reboot — see `D0-platform-frame.md`):** platform
> R1 decision 2 (owner-ruled) makes the **coding-agent-harness persona bundle the v1 proof
> surface** — largely pre-answering 5.a in the ship-a-bundle direction, with a nuance the
> session must settle: the platform persona is "the desk agent" (whole-desk), so decide
> whether it is a third contract instantiation or the composition of the librarian + PM
> personas. It also raises 5.b's stakes: the persona wires to "the existing deskkit MCP
> server" (`../platform/plan.md` R1/R2) — the everything-enabled mount whose ride-along
> problem 5.b exists to fix. Chat-to-schema capture (the platform's differentiator skill)
> would ride this same contract as a new claimed tool.

# D5 — Agent integration contract & surface parity

## The question

**Is agent symmetry a single integration contract with two instantiations — and for each of
the four asymmetries in prep §1, is the gap policy (document it) or debt (fix it)?**

The owner's framing (prep §1): symmetry is a *surface property of a shared core*. The KM
(librarian) and PM agents should be two instantiations of one **integration contract** —
`(persona instructions + tool mount + wake layer + write-gate policy)` — not necessarily one
implementation. Deliberate asymmetries (inverted write-gate defaults; no in-binary PM brain)
survive as **documented contract parameters**, not as debt to erase. This brief packages the
four sub-decisions the exec-desk agenda groups under one doc — **5.a** librarian Claude Code
bundle or not, **5.b** per-module vs shared MCP mounting, **5.c** in-binary PM prompt vs
PM-tool exclusion, **5.d** instruction source per surface — under one contract framing, and
widens each to the full Phase-0 unclaimed-surface set (ride-along tools on both agent loops;
3 of 4 TS plugin tools claimed by no skill; PM `import` and the admin console claimed by no
persona at all).

## Why now

The agent-symmetry concern is the owner's headline (prep §3 C1) and it survived the 0.8.0 bug
floor (PR #112) essentially untouched: 14 of 15 exec-desk analysis claims re-derived as
CONFIRMED against post-PR-112 main, 1 CHANGED (#79, resolved by fail-loud serve, not by
symmetric self-gating). Nothing forces a ruling — but the cost of *not* ruling compounds in
two directions the dossiers made concrete:

- **A mounted tool no instruction claims is a live footgun, not a cosmetic gap.** Enabling PM
  puts 12 PM tools into the same process-global registry the in-binary eino loop reads, while
  that loop's system prompt stays librarian-only (finding 2). Independently, on a
  *default-configuration* desk the librarian's own prompt claims two tools (`apply_fix`,
  `restore`) it does not hold (finding 3). Both are stale *the moment* they matter — the first
  when a second module turns on, the second always.
- **The "fit to built-in, bring your own harness" story is asymmetric where it is most
  visible.** desk-pm ships a full Claude Code bundle (agent + 3 skills + SessionStart hook +
  `.mcp.json` + marketplace entry); the librarian's only Claude Code path is a hand-authored
  README snippet (asymmetry #1). One product, two very different front doors.

Shipped-issue lineage: **#79** (mount failure modes diverging inside one bundle) is the one
asymmetry already resolved — its fail-loud + mount-signal fix is the precedent every future
mount decision here should inherit. **#94** shipped `docs/tool-surface.md` (authoritative tool
inventory) but did not resolve the ride-along it documents, nor correct the spec residue —
i.e. #94 quantified the problem this brief must rule on. The four asymmetries themselves are
unruled: no issue closed them.

## Evidence

Every bullet resolves to a real dossier section; `path:line` cites are lifted from the
dossier, not re-derived. Two load-bearing cites were spot-confirmed in source (noted in
Uncertainties). Dossier files live at `../` (one level up from this book).

**The contract has two clean instantiations, and two deliberate asymmetries are already ruled:**

- `agent-symmetry.md § Verdict table — the 7-row comparison table` — across all seven
  dimensions (brain, tools, instructions, Claude Code mount, write gate, safety model, TUI) the
  librarian and PM integrations differ; the eino loop is in-binary Go, the PM "agent" is a
  markdown operator with no in-binary loop (`librarian/internal/modules/librarian/agent/agent.go:87-100,148-186`;
  `plugin/desk-pm/agents/pm-operator.md:31-35`; `librarian/internal/modules/pm/module.go:78-129`).
- `agent-symmetry.md § Verdict table — the 2 "deliberate & defensible" asymmetries` — the
  **inverted write-gate defaults are already ruled** in a code comment: PM tools write only the
  store, so the document gates (not a registration gate) are the real safety; `LIBRARIAN_AUTONOMOUS_WRITES`
  defaults off, `PM_AUTONOMOUS_WRITES` defaults on (`config.go:118-121`; struct doc `:46`).
  The **no-in-binary-PM-brain** asymmetry is likewise deliberate — the PM engine is
  deterministic state-machine code, its module registers tools/views/hooks but no loop
  (`pm/module.go:38-88`). These two are the "documented contract parameters" the framing preserves.

**5.a — the librarian has no Claude Code bundle:**

- `agent-symmetry.md § Verdict table — the 6 "accidental / unruled" asymmetries` (#1) — no
  `desk-librarian` plugin exists; the marketplace lists only `desk-standard` (the 4-profile-tool
  plugin) and `desk-pm`; the librarian's only Claude Code wiring is a README `.mcp.json` snippet
  (`.claude-plugin/marketplace.json:9-23`; `librarian/README.md:216-227`).

**5.b — ride-along tools on shared mounts, and the mirror gap (capabilities with no mount):**

- `agent-symmetry.md § Verdict table — the 6 "accidental / unruled" asymmetries` (#2) — the
  `deskkit mcp-serve` server loops over the *merged* registry of all enabled modules; with
  `PM_ENABLED` the mount exposes **17** tools (5 default librarian + 12 PM), now documented but
  not resolved (`core/mcp/server.go:69-81`; `toolcore.go:133-156`; `docs/tool-surface.md` §2,
  PM_ENABLED → 17).
- `surface-matrix.md § 6. Unclaimed-surface findings (the ride-along problem)` (finding 1) —
  `plugin/desk-pm/.mcp.json` sets only `PM_ENABLED=true`, so the PM mount surfaces the 5 default
  librarian tools (`sweep patrol propose_fix query record_feedback`); neither the `pm-operator`
  `tools:` allowlist (12 PM names, zero librarian) nor any desk-pm skill references them
  (`plugin/desk-pm/.mcp.json:1-9`; `agents/pm-operator.md:13-26`).
- `surface-matrix.md § 6. Unclaimed-surface findings (the ride-along problem)` (finding 5) —
  the mirror gap: the PM manifest **importer** is real, tested, and load-bearing for the
  rebuild-reproducibility guarantee, but has no CLI subcommand, no MCP tool, no TUI affordance,
  no admin-console path — its only caller is the test harness, reserved for a not-yet-built D8
  adoption path (`importer.go:1-20`; `pm/scenario/scenario.go:144`; `importer.go:8-10`).
- `surface-matrix.md § 6. Unclaimed-surface findings (the ride-along problem)` (finding 6) —
  the admin console (PocketBase `/_/`) is reachable by any superuser, gated by nothing beyond
  login, and named by no skill or persona — a human-only path by *convention*, not restriction
  (`collections.go:12`).
- `agent-symmetry.md § New since the analysis (PR 112 — bears on the symmetry decision)` — #79
  is resolved by **fail-loud serve + observable mount signal**, not symmetric self-gating:
  `requireResolvedConfig` → `os.Exit(1)` with an actionable message, and on success
  `emitMountSignal` prints the exposed tool set to stderr; the residual is a *missing* binary
  named in `.mcp.json` still failing at the host level (`core/mcp/server.go:92-151`). This is
  the precedent to apply to any future mount.

**5.c — the in-binary loop's tools and its prompt diverge:**

- `agent-symmetry.md § Verdict table — the 6 "accidental / unruled" asymmetries` (#3) — with
  `PM_ENABLED` the eino loop receives all 12 PM tools via the merged registry, but its resolved
  system prompt only ever loads `librarian.system` (no `pm.system` is seeded anywhere); the PM
  read-before-write / gate-routing / claim-release discipline lives only in `pm-operator.md`
  (`pm/module.go:78-81` → `agent.go:107-117` → `toolcore.go:133-142`; prompt `agent.go:42-53`;
  discipline `pm-operator.md:52-55`).
- `surface-matrix.md § 6. Unclaimed-surface findings (the ride-along problem)` (finding 2) — the
  ride-along exists *inside the binary*, one level worse: `module.Register` merges PM's 12 tools
  into the process-global registry both the eino loop (S4) and the MCP server (S1) read, while
  the loop's prompt (`librarian/templates/librarian-system-prompt.txt`) lists exactly 7
  librarian tools and never the PM ones — the model's own instructions go stale the moment a
  second module is enabled (`internal/core/module/module.go:74-95`; `agent.go:109`;
  `librarian-system-prompt.txt:1-33`).
- `surface-matrix.md § 6. Unclaimed-surface findings (the ride-along problem)` (finding 3) — the
  librarian prompt is stale **even with PM off**: it unconditionally lists `apply_fix` and
  `restore`, but `restore` is never in the agent slice under any config and `apply_fix` is absent
  unless `LIBRARIAN_AUTONOMOUS_WRITES=true` (default off) — so a default desk's agent claims 2
  tools it does not have (`librarian-system-prompt.txt:5-14`; `toolcore.go:144-150`).
- `workflows.md § 1.6 The eino ReAct agent loop` — the system prompt is a **live DB read at
  RUN START** (`key="librarian.system" && active=true`, newest version, falling back to the
  embedded seed), and the write gate / `restore` exclusion are enforced purely by *which tools
  are in the slice*, never by a runtime check in the loop (`agent.go:42-62,91-93`;
  `agent.go:38-41`; `toolcore.go:133-142`). Whatever 5.c rules, the prompt-source and the
  tool-slice source are two independent decisions the loop already reads separately.

**5.d — instruction source per surface, and skills that claim no tools:**

- `agent-symmetry.md § Verdict table — the 6 "accidental / unruled" asymmetries` (#4) — the
  prompt-governance split is real and unchanged: librarian instructions are DB-backed /
  GUI-editable, PM instructions are version-controlled markdown; neither model was chosen for
  both (`prompt.go:16-40` + `agent.go:42-53` vs `pm-operator.md` + `skills/*`).
- `surface-matrix.md § 6. Unclaimed-surface findings (the ride-along problem)` (finding 4) — of
  the 4 TS plugin tools, only `template_render` is named by any skill; `profile_get` and
  `knowledge_index` are mounted and listed in every `tools/list` but no skill instructs an agent
  to call them, and `profile_validate` is implied once but never named
  (`plugin/claude-plugin/.mcp.json:1-6`; `brownfield-adoption/SKILL.md:120`).
- `surface-matrix.md § 7. Persona inventory` — the full instruction-source map: the in-binary
  eino prompt (7 tools, 2 possibly-absent), `pm-operator` (12 PM tools, a clean claim), the
  three desk-pm skills, the four plugin skills (two name only `template_render`; `harvest-loop`
  and `conventions-standard` name no tool at all, referencing "the MCP tools" generically), and
  the SessionStart hook (calls the **CLI**, not an MCP tool) (`plugin/desk-pm/agents/pm-operator.md`
  lines 13-26; `harvest-loop/SKILL.md:35`; `conventions-standard/SKILL.md:24`).

**Cross-cutting mechanism (grounds all four):**

- `workflows.md § 3. Cross-lane touch points` — `module.Register` is the single seam: it merges
  every enabled module's `Tools()` into one shared `toolcore` registry, and the MCP server loops
  that merged registry rather than a per-lane switch — so a module's tools appear on *every*
  surface with zero server changes (`module.go:74-116`; `server.go:57-81`; `toolcore.go:35-88`).
  This is *why* the ride-along is structural, not a bug in any one file.
- `workflows.md § 4. Feature gating` — the three flags and where they are read: `PM_ENABLED`
  (module registration; `pm/module.go:63`, `config.go:113-116`), `LIBRARIAN_AUTONOMOUS_WRITES`
  (registration-time slice inclusion of `apply_fix` + dispatch-time wake re-check; `toolcore.go:129-142`,
  `config.go:112`, `trigger.go:201-213`), `PM_AUTONOMOUS_WRITES` (PM write-tool `AgentDefault`,
  default ON; `pm/module.go:78-81`, `config.go:118-121`).
- `workflows.md § 1.7 Wake layer / cron hooks` — the librarian's wake layer (hourly cron +
  claimer goroutine) is registered **only under `serve`**, so one-shot CLI never enqueues; PM
  has no in-binary wake at all (its plugin wake is the SessionStart hook) (`trigger.go:52-61`;
  `main.go:264-269,286`). "Wake layer" is thus a live contract parameter with an already-deliberate
  asymmetry.
- `surface-matrix.md § 4. TS plugin lane — operations x surfaces` — the TS MCP server exposes a
  fixed 4 tools, no env gate; it is a separate mount from the Go binary entirely
  (`plugin/core/tools.ts:56-65,127,175,205,240`; `plugin/mcp/server.ts:18,29-34,37-60`).

## Options

**One contract, two instantiations (the framing all four sub-decisions sit under).** Define
the integration contract as a named tuple every agent surface instantiates:

| Contract parameter | Librarian (KM) instantiation | PM instantiation | Status |
|---|---|---|---|
| **Persona instructions** (source + body) | DB `prompts` row `librarian.system`, GUI-editable, live-read per run | plugin markdown (`pm-operator.md` + skills), version-controlled; no `pm.system` row | asymmetric — **5.c/5.d rule policy-vs-debt** |
| **Tool mount** (what is reachable) | Go MCP / eino loop over the merged registry | desk-pm `.mcp.json` mounts the same server with `PM_ENABLED` | shared mount ⇒ ride-along — **5.b rules** |
| **Claude Code packaging** | none (README snippet only) | full bundle (agent + skills + hook + marketplace entry) | asymmetric — **5.a rules** |
| **Wake layer** | cron + claimer goroutine (serve-only) | SessionStart hook (plugin); no in-binary wake | **deliberate asymmetry — document as policy** |
| **Write-gate policy** | `LIBRARIAN_AUTONOMOUS_WRITES` off | `PM_AUTONOMOUS_WRITES` on (store-only writes; gates are safety) | **already ruled (config comment) — document as policy** |

The framing choice itself is a hypothesis to ratify (prep §4, "Symmetry as one contract, two
instantiations"): **adopt the contract as the organizing artifact** — a short spec section that
names these five parameters and, for each agent, states its value and whether the value is
policy or debt. The alternative is to rule each asymmetry ad hoc with no shared vocabulary;
that is what produced the current drift. The two bottom rows are the "deliberate & defensible"
asymmetries — under this framing they are *ratified as documented parameters*, not reopened.
The four sub-decisions below each rule one of the top-three (contested) parameters.

### 5.a — Librarian Claude Code bundle, or CLI/TUI-first?

- **A1 — Ship a `desk-librarian` bundle** mirroring desk-pm's shape (agent persona + skills +
  SessionStart hook + `.mcp.json` + marketplace entry). *(Hypothesis, prep §4 "Ship a librarian
  Claude Code bundle".)* Completes the "fit to built-in" story symmetrically; but it forces a
  librarian *persona* to exist in markdown, which reopens 5.d (two instruction sources for the
  same brain) and multiplies bundles to maintain.
- **A2 — Rule CLI/TUI-first deliberate; keep the README `.mcp.json` snippet as the only Claude
  Code path.** Documents the asymmetry as policy: the librarian is a binary-native agent, its
  Claude Code story is "mount the MCP server", not "install a plugin". Cheaper; but leaves the
  front-door asymmetry the owner flagged, and a bare `.mcp.json` snippet has no SessionStart
  affordance and no #79-style host-level guard for a missing binary.
- **A3 — Ship a *minimal* bundle: `.mcp.json` + a mount-signal-aware hook, no separate persona**
  — the librarian's brain stays in-binary (the DB prompt), the bundle only wires the mount and
  a fail-loud check. Narrows A1's persona-duplication risk while closing the front-door gap.

### 5.b — Per-module MCP surfaces, or one shared mount with tool-level gating?

- **B1 — Per-module mounts** (e.g. `mcp-serve --module=pm`, `--module=librarian`) so no surface
  carries tools no persona claims. *(Hypothesis, prep §4 "Per-module MCP surfaces".)* Kills the
  ride-along at the source; but multiplies processes and `.mcp.json` entries per session, and
  interacts with 5.c (the in-binary loop still merges the registry unless the module gate also
  scopes *its* slice).
- **B2 — Keep the shared mount, add tool-level gating** so unclaimed tools aren't silently
  reachable — a mount declares which module's tools it intends and the server filters the merged
  registry to that set. Fewer processes; but needs a new gating dimension in `toolcore` and a
  per-mount declaration, and the "shared mount" tough spot (prep §5) warns unclaimed tools are
  reachable until that filter exists.
- **B3 — Rule the ride-along policy (document it), not debt** — a mount that enables PM
  *intends* to also expose the 5 default librarian tools, and the personas are updated to claim
  them. Cheapest; but contradicts the `pm-operator` allowlist's clean 12-name intent and leaves
  the in-binary finding-2 case (the prompt can't "claim" tools a human didn't choose to mount).
- **Every option inherits #79's precedent:** whatever mount shape wins, apply fail-loud +
  observable mount signal to it (and consider the missing-binary host-level residual).
- **Mirror gap (rule alongside):** `import` and the admin console are *capabilities/surfaces
  with no claiming persona*. Options: (i) give `import` a supervised CLI subcommand now vs.
  leave it test-harness-only pending D8; (ii) name the admin console in the contract as an
  acknowledged human-only path vs. leave it convention-only. These are the inverse of the
  ride-along and belong to the same "mount composition" ruling.

### 5.c — In-binary PM prompt, or PM-tool exclusion from the in-binary loop?

- **C1 — Exclude PM tools from the in-binary eino slice.** The librarian brain stays
  librarian-only; PM tools are reachable only through the PM (plugin) instantiation. Restores
  tool/prompt coherence with the smallest change (scope the merge per surface, or filter
  `AgentTools` to the librarian module); but means an in-binary `chat`/`agent` on a PM-enabled
  desk cannot drive PM — a capability some may want.
- **C2 — Give the in-binary loop a composed PM prompt** — seed a `pm.system` row (or compose
  `librarian.system` + a PM section) whenever `PM_ENABLED`, so the mounted 12 PM tools come with
  their read-before-write / gate-routing / claim-release discipline. Makes the in-binary loop a
  true two-module operator; but duplicates the pm-operator discipline into a second instruction
  source (reopens 5.d) and needs the exact discipline delta re-derived (dossier flagged it was
  not clause-by-clause diffed).
- **C3 — Fix the *librarian* prompt's own staleness regardless** — the finding-3 case
  (`apply_fix`/`restore` claimed on a default desk) is orthogonal to PM and is debt under any
  option: make the prompt reflect the actual gated slice (config-aware, or drop the two lines).
  Independent of C1/C2 and cheaper than either.

### 5.d — Instruction source per surface (the contract *names* it; D6 rules the mechanism)

- **D-i — The contract names exactly one instruction source per surface**, and this brief rules
  only *that* each surface has one, plus which of today's stale/absent claims are debt: the
  in-binary prompt's phantom tools (finding 3, debt), the desk-pm mount's unclaimed ride-along
  (finding 1, tied to 5.b), the 3 TS plugin tools no skill names (finding 4, debt — the skills
  should name what they expect an agent to call), the two skills that name no tool at all. The
  **single-sourcing mechanism** (Go embed ↔ DB rows ↔ plugin markdown) is D6's.
- **D-ii — Defer the per-surface source question entirely to D6.** Simpler boundary; but leaves
  the parity contract without a named instruction-source parameter, which is exactly the field
  the framing table needs to be complete — so the contract would ship with a hole D6 fills
  later.
- Recommended boundary *(not a ruling)*: D-i. The contract *names* an instruction source per
  surface as a parameter (making the parity table complete); D6 rules how those sources stay in
  sync across the three stores.

## Decision criteria

Any ruling must satisfy the constraint walls (`../../README.md` §7) that bind here:

1. **Identity-neutrality (CI-enforced).** A librarian bundle (5.a) or any new persona/skill must
   ship zero deployment identity — no person/org/repo/issue. A bundle persona is markdown under
   `plugin/`, in scope for `scripts/check-neutrality.mjs`; write it identity-neutral or it fails CI.
2. **Harness-purity of `plugin/core`.** 5.d touching the TS plugin skills must not push
   harness/runtime imports into `plugin/core` (`scripts/check-core-purity.mjs`). Skills are
   markdown; the constraint binds only if a ruling adds core code.
3. **Fail-loud / observable mounts (#79 precedent).** Any mount shape 5.a/5.b produces must
   fail loud on an unresolved desk (`os.Exit(1)`, not a degenerate surface) and emit the mount
   signal — the pattern `core/mcp/server.go` already ships. Note the missing-binary host-level
   residual has no server-side twin.
4. **The write-gate and `restore`/`import` postures are already deliberate.** Any per-module or
   tool-level gating (5.b) must preserve: `restore` and `findings dispose` supervised-CLI-only
   (structurally excluded from the agent slice), the inverted autonomous-writes defaults, and
   `import`'s reserved-for-D8 status unless 5.b deliberately promotes it.
5. **One write path, no second gate.** The contract must not introduce a parallel tool-dispatch
   or gate path — the shared `toolcore` registry and the single `transitionCore` are load-bearing
   invariants; per-module mounting must *filter* the one registry, not fork it.
6. **Depends on D1–D4.** Parity is "what every surface must expose"; the model rulings
   (pointer grammar, typed refs, `items.type`, disposition/adoption-log) change the tool
   *contracts*, so 5.b/5.d must consume D1–D4's outputs, not pre-empt them.

## Blast radius

Concrete artifacts per option class. Live stores exist (the owner's desks) — migration reality
is part of the radius.

- **5.a — librarian bundle (A1/A3):** new `plugin/desk-librarian/` tree (agent persona, skills,
  hook, `.mcp.json`), a new `.claude-plugin/marketplace.json` entry, and package/regeneration
  wiring (the marketplace install copies only `plugin/claude-plugin/`; a new bundle needs its own
  packaging + drift guard). CI: neutrality scan now covers a new persona. A2 is docs-only
  (`librarian/README.md` + a charter/spec note ruling CLI/TUI-first).
- **5.b — mount shape (B1/B2):** `librarian/internal/core/mcp/server.go` (mount entry / flag),
  `toolcore` (a per-mount tool-filter dimension for B2, or a `--module` scoping for B1),
  `plugin/desk-pm/.mcp.json` (and any librarian `.mcp.json`), and `docs/tool-surface.md` (counts
  change). No collection/migration change unless `import` gets a real surface (then a CLI
  subcommand in `cmd/deskkit/main.go`, no new collection). Admin-console posture is docs-only.
- **5.c — in-binary prompt/slice (C1/C2/C3):** C1 = a scoping change in
  `module.Register`/`AgentTools` (the merged-registry seam) — no store change. C2 = a new
  `pm.system` prompt seed (Go embed + a `prompts` row on `PM_ENABLED` desks — a forward-only
  seed, never edit an applied migration) + the composition logic in `agent.go`'s `systemPrompt`.
  C3 = edit `librarian/templates/librarian-system-prompt.txt` (+ the embedded seed and any DB
  `librarian.system` row — a re-seed / migration reality for live stores).
- **5.d — instruction sources:** plugin skills (`plugin/claude-plugin/skills/*`, `plugin/desk-pm/skills/*`)
  named-tool edits; the parity contract spec section (new). Interacts with D6 (mechanism) and D7
  (the TS boundary's promised tool set).
- **The contract spec itself:** a new section in `docs/pocket-librarian-v1-spec.md` and/or
  `docs/pm-system-v1-spec.md` (and possibly the charter if direction moves), plus an ADR in
  `docs/decisions/` recording the parity ruling and each asymmetry's policy-vs-debt verdict.

## Out of scope / interactions

- **D6 owns the prompt-governance MECHANISM** — how a single instruction source is kept in sync
  across the Go embed, the DB `prompts` rows, and plugin markdown (the drift-guard question the
  runtime-editable DB prompt makes genuinely hard). D5 rules only *that the contract names an
  instruction source per surface* (5.d), not how it syncs. The `prompts` collection's
  GUI-editability vs. git-source tension is D6's tough spot (prep §5), not settled here.
- **D7 owns spec ↔ reality reconciliation of the TS `plugin/mcp` boundary** — the spec promises
  librarian tools over a boundary that ships only 4 profile tools, and carries stale "six-tool
  core" / "four default" counts (asymmetry #5). D5 treats "what the TS boundary *should*
  promise" as a contract parameter and hands the reconciliation to D7.
- **D1–D4 (this brief depends on them):** the model/workflow rulings determine what each surface
  must expose. D5 must not re-decide pointer grammar, typed refs, `items.type` validation, or
  the disposition/adoption-log sub-machine — it consumes their outputs into the tool contracts.
- **Backlog (D8):** `import`'s promotion to a real adoption surface is reserved for D8; D5 only
  flags it as the mirror gap and may rule an interim CLI affordance under 5.b.
- **Not decided here:** whether any asymmetry is *correct* is the session's call — this brief
  frames the contract and the options, it does not rule (prep §2 phase 2). The two "deliberate &
  defensible" asymmetries are ratified as documented parameters, not reopened.

## Uncertainties

- **pm-operator discipline vs. a composed in-binary prompt (5.c C2).** The dossier confirmed the
  load-bearing half (the in-binary prompt resolves only `librarian.system`; no `pm.system` is
  seeded) but did **not** diff the two instruction bodies clause-by-clause
  (`agent-symmetry.md § Gaps & uncertainties`). If C2 is ruled, the exact discipline delta to
  port must be re-derived in the session.
- **`profile_validate`'s "claimed vs. unclaimed" status is a judgment call, not a bright line**
  (`surface-matrix.md § 8. Gaps & uncertainties`) — `brownfield-adoption/SKILL.md:120` implies
  validation without naming the tool. Whether 5.d counts that as "claimed" is exactly the
  surface-vs-persona question the session resolves.
- **Admin-console gate-bypass is stated structurally, not proven live** (`surface-matrix.md §
  8.`) — nil API rules ⇒ superuser bypass is standard PocketBase, and no per-collection state-
  machine hook was found, but no live write was attempted. If the contract rules on admin-console
  posture, confirm empirically first.
- **`import`'s "reserved for D8" is the package author's stated intent** (`importer.go:8-10`),
  not a ruled roadmap commitment (`surface-matrix.md § 8.`). Treat 5.b's `import` option as
  ruling that intent, not discovering it.
- **Tool counts are source-verified, not re-run empirically a second time** (`surface-matrix.md
  § 0. Anchor and how this dossier extends it`; `agent-symmetry.md § New since the analysis`) —
  the dossiers re-read the source behind `docs/tool-surface.md`'s counts and found no
  discrepancy, but did not rebuild the binary / re-run the JSON-RPC probe. If a count is
  load-bearing to a ruling, re-run the probe.
- **Config line-number drift** — the write-gate lines moved by one from the exec-desk analysis
  (PM gate `config.go:121` not `:122`; ruling comment `:118-120`); substance identical, flagged
  so the session doesn't re-flag it (`agent-symmetry.md § Gaps & uncertainties`).
- **Two cites spot-confirmed in source** (permitted sanity-check, no fresh research): the
  librarian prompt does list `apply_fix`/`restore` and no PM tools
  (`librarian/templates/librarian-system-prompt.txt`), and `server.go` does carry
  `requireResolvedConfig` → `os.Exit(1)` + `emitMountSignal` — both consistent with the dossier
  cites used above.

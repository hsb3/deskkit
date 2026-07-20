---
type: analysis
status: draft
created: 2026-07-20
updated: 2026-07-20
tags: [design-session, access-surfaces]
synopsis: Operations x access-surfaces matrix for the librarian, PM, and TS-plugin tool
  families, citing path:line for every populated cell and flagging tools reachable on a
  mount that no persona's instructions claim.
---

_Design-session dossier — for every operation the product supports, which surface(s) expose
it, under what gating, and which surfaces claim it in their own instructions. Grounds C1
(agent-surface parity) and C6 (prompt governance) in `README.md`. Informs the session; does
not itself rule._

Status: draft (2026-07-20)

# Surface matrix

## 0. Anchor and how this dossier extends it

[`docs/tool-surface.md`](../../../docs/tool-surface.md) is the shipped, empirically-derived
map of the three tool-bearing surfaces (librarian CLI, librarian MCP, plugin TS MCP) and their
counts (16 base CLI / 5·6·17·18 MCP by gate / 4 TS, fixed). This dossier's source-level
re-check of those claims **found no discrepancy**:

- Librarian CLI: reading `registerToolCommands`/`registerPMCommands` directly gives the same
  16 base subcommands (`init sweep patrol propose-fix apply-fix restore query findings-dispose
  record-feedback agent chat mcp-serve gui` + PocketBase's `serve migrate superuser`), plus the
  gated `pm` group of 12 — matches `docs/tool-surface.md` table 1 exactly.
- Librarian MCP: `toolcore.AgentTools`/`ExposedTools`
  (`librarian/internal/core/toolcore/toolcore.go:133-156`) and the two module `Specs()`
  functions (`librarian/internal/modules/librarian/tools/specs.go`,
  `librarian/internal/modules/pm/tools/specs.go`) reproduce the same 5/6/17/18 gate logic the
  doc describes — this dossier did **not** re-run the doc's JSON-RPC probe (no build step was
  in scope; see §8), so the counts are source-verified, not re-run empirically a second time.
- Plugin TS MCP: `plugin/core/tools.ts:240`'s `TOOLS` array is exactly the 4 the doc names.

What this dossier adds beyond the anchor: the **CLI** and **MCP** are only two of nine access
surfaces the product actually has. This matrix adds the **chat TUI**, the **in-binary eino
agent** (a fourth tool-bearing surface `docs/tool-surface.md` does not cover — its tool set is
gate-identical to the MCP server, §2), the **admin console**, the **four Claude Code plugin
skills**, the **desk-pm bundle** (persona + skills + hook), and **direct human file/GUI paths**
— then cross-references which of the nine actually *claims* each tool in its own instructions
(§6-7), which is where the gaps live.

## 1. Surfaces legend

| # | Surface | What it is | Tool set source |
|---|---|---|---|
| S1 | **Go MCP server** | `deskkit mcp-serve`, stdio, model-facing | `toolcore.ExposedTools(cfg)` — 5/6/17/18 by gate (§0) |
| S2 | **CLI** | `deskkit <subcommand>`, human/script-facing | `registerToolCommands`/`registerPMCommands`, full set always (no gate) |
| S3 | **Chat TUI / REPL** | `deskkit chat`, full-screen (ADR 0004) or line REPL when piped/`--plain` | same tool slice as S4 (`agent.NewSession`, `librarian/internal/modules/librarian/agent/session.go:53-56`) |
| S4 | **In-binary eino agent** | `deskkit agent <instruction>` (manual trigger) and the loop underlying S3 | `toolcore.AgentTools(cfg)` — **identical gate/set to S1**, since both call the same function (`librarian/internal/modules/librarian/agent/agent.go:103-109`) |
| S5 | **Admin console** | PocketBase's built-in web UI at `/_/`, reached via `deskkit gui`/`deskkit serve` + a superuser login | raw record CRUD over every collection; **not tool-gated at all** (§5) |
| S6 | **Claude Code plugin skills** | `plugin/claude-plugin/skills/{desk-setup,conventions-standard,harvest-loop,brownfield-adoption}` | prose instructions naming the TS MCP tools (S7) and, secondarily, the librarian MCP tools (S1) by generic reference |
| S7 | **TS MCP server** | `plugin/mcp/server.ts`, stdio, over the harness-pure TS core | `TOOLS` array, `plugin/core/tools.ts:240` — 4 tools, fixed, no gate |
| S8 | **desk-pm bundle** | `plugin/desk-pm/` — `pm-operator` agent persona + 3 skills + a SessionStart hook, mounting S1 with `PM_ENABLED=true` | the persona's `tools:` frontmatter + the 3 skills' prose (§7) |
| S9 | **Direct human paths** | editing a desk file by hand; `deskkit restore --by-path`; the admin console (S5) as a human path | no tool core — the human bypasses every surface |

## 2. Librarian (KM) lane — operations x surfaces

| Operation | S1 Go MCP (gate) | S2 CLI | S3 Chat TUI | S4 eino agent | S5 Admin console | S6 Plugin skills | S9 Direct human | Gating |
|---|---|---|---|---|---|---|---|---|
| **sweep** (reindex desk tree) | `sweep`, default-exposed. Spec: `librarian/internal/modules/librarian/tools/specs.go:26-32` | `sweep`, `librarian/cmd/deskkit/main.go:691-701` | via agent tool-call (S4 tool set) | same spec as S1 | reindex bypassed — files collection editable directly, but `sweep` itself has no admin-console equivalent (it's a filesystem walk, not a record edit) | not named in any of the 4 skills (grep: 0 hits) | re-run `sweep` is the only recovery from a stale index; no file-edit equivalent | none (`AgentDefault=true` unconditionally) |
| **patrol** (flag R1-R6 findings) | `patrol`, default-exposed. `specs.go:31-35` | `patrol [--path]`, `main.go:704-717` | via agent tool-call | same spec as S1 | findings viewable/editable as raw records (bypasses the rule engine) | not named in any of the 4 skills | — | none |
| **query** (8 read kinds: `live_files recent orphans uncollapsed findings summary adoption feedback`, `librarian/internal/modules/librarian/tools/query.go:38-56`) | `query`, default-exposed. `specs.go:56-63` | `query <kind> [--days] [--pretty] [--include-disposed]`, `main.go:772-807` | via agent tool-call | same spec as S1 | any collection browsable directly (equivalent read, no kind grouping) | not named in any of the 4 skills | read desk files directly (equivalent to `live_files`/`orphans` but unindexed) | none. `--include-disposed` / disposition filtering is CLI/MCP-only; the admin console shows raw rows regardless of disposition |
| **propose-fix** (plan a mechanical fix; record original) | `propose_fix`, default-exposed. `specs.go:36-40` | `propose-fix [--run] [--rules]`, `main.go:720-735` | via agent tool-call | same spec as S1 | revisions row insertable by hand (not the intended path) | not named in any of the 4 skills | — | none |
| **apply-fix** (commit a proposed fix, byte-exact) | `apply_fix`, **AgentGated** — exposed only when `LIBRARIAN_AUTONOMOUS_WRITES=true`. `specs.go:41-45` | `apply-fix [--run] [--revision-ids]`, `main.go:738-753`, no gate — CLI always has it | via agent tool-call, gated same as S1 | same spec as S1 (gated) | a desk file editable directly, **bypassing record-original-first entirely** | not named in any of the 4 skills | editing the file by hand is the ungated equivalent write | `LIBRARIAN_AUTONOMOUS_WRITES` (registration-time; checked again at execution, per `docs/tool-surface.md` L80-81) |
| **restore** (reverse to recorded original) | **never exposed** — neither `AgentDefault` nor `AgentGated`; `ExposedTools` also filters it defensively. `specs.go:46-50`, `toolcore.go:144-150` | `restore [--revision \| --by-path]`, `main.go:756-770` — **CLI-only** | not available (not in the S4 tool set, so not in S3 either) | not available — same exclusion as S1 | a prior state is recoverable only if the admin console still holds the `revisions` row (raw, unassisted) | not named in any of the 4 skills | `deskkit restore --by-path <path>` **is** the sanctioned human recovery path | structural exclusion, not an env flag — supervised-only by design |
| **findings dispose** (`open\|acknowledged\|triaged\|wont_fix`) | **no MCP tool** (like `restore`, deliberately CLI-only) | `findings dispose <id> --as <disposition>`, `main.go:809-837` | not available | not available | disposition field editable directly on the `findings`/`patrol_findings` record (bypasses the normalize/validate step in `tools.DisposeFinding`) | not named in any of the 4 skills | — | supervised-only, no flag |
| **record-feedback** (append feedback-log entry) | `record_feedback`, default-exposed. `specs.go:61-67` | `record-feedback --kind --summary [--detail] [--context] [--source]`, `main.go:839-869` | via agent tool-call | same spec as S1 | `feedback` collection insertable directly | not named in any of the 4 skills | — | none — DB-only write, no file-write gate applies |
| **chat** (interactive multi-turn session) | n/a (chat is a surface, not a tool) | `chat [--plain] [--theme]`, `main.go:897-940` — the surface itself | **is** S3 | opens/drives one `agent.Session` (`session.go:53-56`) | n/a | not named in any of the 4 skills | — | inherits the S1 gate for every tool call it makes (§0) |
| **agent** (manual one-shot loop trigger) | n/a (this *is* S4's CLI entry point) | `agent <instruction>`, `main.go:871-895` | n/a | **is** S4 | n/a | not named in any of the 4 skills | — | same S1 gate |

## 3. PM lane — operations x surfaces

PM is feature-gated OFF by default (`PM_ENABLED`, `config.go:44,116`); every row below is
**absent from every surface** unless the desk has it on.

| Operation | S1 Go MCP (only under `PM_ENABLED`) | S2 CLI (`deskkit pm ...`) | S3 Chat TUI | S4 eino agent | S5 Admin console | S8 desk-pm bundle | Gating |
|---|---|---|---|---|---|---|---|
| **get_context / list_items / get_item** (queries) | `get_context` `specs.go:31-36`, `list_items` `:37-42`, `get_item` `:43-48` — all `AgentDefault=true` unconditionally (read tools ignore `PM_AUTONOMOUS_WRITES`) | `pm context [--stalled-days]` `pm.go:86-99`, `pm list [--phase --court --type --blocked --parent]` `:101-119`, `pm get <id>` `:122-134` | read-only PM views: `pm context`/`pm board`/`pm item`, `librarian/internal/modules/pm/tui/views.go:24-45` (thin reads over the same `engine` calls) | in the S4 tool set whenever `PM_ENABLED` (merged registry, `module.go:74-95` `Register`, gate check `:78`, merge `:95`) — **but the eino system prompt never mentions them**, see §7 | items/dependencies/notes/transitions collections browsable directly | `pm-operator` persona claims all three by name (`agents/pm-operator.md:15-17`); `pm-session-open` skill is built entirely around `get_context` | none (always `AgentDefault`) |
| **create_item** | `create_item`, `specs.go:49-54`, `AgentDefault=w` (see gating) | `pm create --title [--type --parent --court --pointer --severity --priority]`, `pm.go:136-158` | not writable from a TUI keypress (views are read-only, §2 of `views.go`) | in S4 tool set under `PM_ENABLED`, same write gate as S1 | insertable directly, **bypassing** `desk_config` court/type conventions | claimed by `pm-operator` (`:18`) and `pm-triage` skill | `PM_AUTONOMOUS_WRITES` (default **ON**, `config.go:46,121`) |
| **update_item** (incl. **set-status-label**) | `update_item`, `specs.go:55-60` | `pm update <id> --version [--title --type --court --pointer --severity --priority --properties --status-label]`, `pm.go:160-189` | not writable from TUI | in S4 tool set | direct field edit, **bypasses** the `SetStatusLabel` phase-gate routing entirely (see below) | claimed by `pm-operator` (`:19`), `pm-triage` (reprioritize/pointer) | `PM_AUTONOMOUS_WRITES` |
| — *set-status-label detail* | a `status_label` naming a *different* phase's label is not a plain field write: `UpdateItem` routes it through `engine.SetStatusLabel` (`librarian/internal/modules/pm/engine/queries.go:546-547`), which applies the same document-gate discipline as `transition_item` (`engine.go:811-815`) whenever the label implies a phase change | same CLI flag as above | — | — | **the admin console writing `status_label` directly does not run `SetStatusLabel` — it is a raw column edit with no gate check at all** | `pm-advance-item` documents this coupling explicitly (`SKILL.md:44-45`) | same as `update_item`, plus the phase gate when a phase change is implied |
| **transition_item** (advance / demote / reopen) | `transition_item`, `specs.go:61-66` | `pm transition <id> --to <phase> --version`, `pm.go:191-216` | not writable from TUI | in S4 tool set | direct `phase` field edit **bypasses the state machine and every document gate** — the sharpest ungated-surface finding in this dossier (§5) | claimed by `pm-operator` (`:20`) and is the entire subject of the `pm-advance-item` skill | `PM_AUTONOMOUS_WRITES`; the document gate itself is unconditional and cannot be bypassed through S1/S2/S3/S4 |
| **block_item / unblock_item** | `block_item` `specs.go:67-72`, `unblock_item` `:73-78` | `pm block <id> --reason --version` `pm.go:218-240`, `pm unblock <id> --reason --version` `:242-263` | not writable from TUI | in S4 tool set | direct `blocked` field edit, bypasses reason/version audit trail | claimed by `pm-operator` (`:21-22`) and `pm-triage` | `PM_AUTONOMOUS_WRITES` |
| **add_note** | `add_note`, `specs.go:79-84` | `pm note <id> --key --body`, `pm.go:265-284` | not writable from TUI | in S4 tool set | `notes` collection insertable directly | claimed by `pm-operator` (`:23`); used in `pm-advance-item` step 4 | `PM_AUTONOMOUS_WRITES` |
| **link_items** | `link_items`, `specs.go:85-90` | `pm link <from> <to> --kind [--unblock-at --cascade]`, `pm.go:286-305` | not writable from TUI | in S4 tool set | `dependencies` collection insertable directly, bypassing kind/cascade validation | claimed by `pm-operator` (`:24`) and `pm-triage` | `PM_AUTONOMOUS_WRITES` |
| **claim_item / release_item** | `claim_item` `specs.go:91-96`, `release_item` `:97-102` | `pm claim <id> --version` `pm.go:307-327`, `pm release <id> --version` `:329-348` | not writable from TUI | in S4 tool set | claim fields editable directly, **defeats the double-work TTL guard entirely** | claimed by `pm-operator` (`:25-26`) and `pm-session-open` (claim before working, release after) | `PM_AUTONOMOUS_WRITES` |
| **import** (manifest -> graph seed) | **no MCP tool** | **no CLI subcommand** | not available | not available | not applicable (would be a bulk direct insert) | not referenced by any skill or the persona | **not wired to any interactive surface at all** — see §8 |

## 4. TS plugin lane — operations x surfaces

| Operation | S7 TS MCP server | S6 Plugin skills | Direct human |
|---|---|---|---|
| **profile_get** (dotted-path read of the active profile) | `profileGet`, `plugin/core/tools.ts:56-65`; registered into `TOOLS`, `:240` | **not referenced by name in any of the 4 skills** (grep across `desk-setup`, `conventions-standard`, `harvest-loop`, `brownfield-adoption`: 0 hits) | reading `_knowledge/profile.yaml` directly |
| **profile_validate** (schema-v1 validate the active profile) | `profileValidate`, `:127` | **not referenced by name in any of the 4 skills** — `brownfield-adoption/SKILL.md:120` says "validate it against the shipped schema before proceeding" but never names the tool, and `desk-setup` never mentions validation at all despite authoring the profile in the same runbook | eyeballing the YAML against `schema/profile.schema.yaml` |
| **template_render** (resolve `{{profile.…}}` placeholders in the scaffold) | `templateRender`, `:175` | **the only TS tool any skill names**: `desk-setup/SKILL.md:8,68`, `brownfield-adoption/SKILL.md:34,110` | hand-editing every placeholder in the scaffold |
| **knowledge_index** (index `_knowledge/*.md` background files) | `knowledgeIndex`, `:205` | **not referenced by name in any of the 4 skills** | reading the `_knowledge/` tree directly |

Server registration for all four: `plugin/mcp/server.ts:18` (imports `TOOLS`), `:29-34`
(`ListTools` handler enumerates them verbatim), `:37-60` (`CallTool` dispatches by name). No
env-var gate exists for this surface — fixed set of 4, always all four.

## 5. Gating column — what flips a cell

| Flag / mechanism | Default | Affects | Where read |
|---|---|---|---|
| `LIBRARIAN_AUTONOMOUS_WRITES` | `false` | `apply_fix` on S1 (Go MCP) and S4 (eino agent/chat). Never affects S2 (CLI always has `apply-fix`) | `config.go:112`; consumed by `toolcore.AgentTools`, `toolcore.go:133-142` |
| `PM_AUTONOMOUS_WRITES` | **`true`** | the 8 PM write tools (`create_item update_item transition_item block_item unblock_item add_note link_items claim_item release_item`) on S1/S4. Read tools (`get_context list_items get_item`) are unaffected. Never affects S2 (`pm create`, `pm transition`, ... always registered when `PM_ENABLED`) | `config.go:46,121`; each PM `ToolSpec`'s `AgentDefault` is literally set to this flag, `pm/tools/specs.go:27-29` (`w := writesEnabled`) |
| `PM_ENABLED` | `false` | the entire `pm` CLI group (S2, `main.go:686-688`) and all 12 PM tools on S1/S4 (`module.go:78` gates `mod.Enabled(cfg)`, `:95` merges `mod.Tools()` into the shared registry); the 3 PM TUI views on S3 (`main.go:935`, `pm/module.go:86`) | `config.go:44,116` |
| Supervised-CLI-only (no flag) | — | `restore` (S1/S4: structurally excluded, never gated by an env var) and `findings dispose` (no `ToolSpec` exists at all) | `specs.go:46-50` (restore), `main.go:809-837` (dispose has no MCP counterpart) |
| Admin-console auth (superuser login) | — | S5 in its entirety — every collection's API rules are `nil` ("superuser-only"), so reaching `/_/` at all requires a superuser account; once authenticated, **no per-operation gate applies** (§6) | `librarian/internal/modules/pm/collections/collections.go:12` ("API rules stay nil => superuser-only, matching every librarian collection"); superuser bootstrap: `main.go:254-260` (first-run auto-create from `PB_SUPERUSER_*` env), `deskkit superuser` (PocketBase-provided) |

## 6. Unclaimed-surface findings (the ride-along problem)

1. **The desk-pm MCP mount ride-along.** `plugin/desk-pm/.mcp.json:1-9` starts
   `deskkit mcp-serve` with only `PM_ENABLED=true` set — neither `LIBRARIAN_AUTONOMOUS_WRITES`
   nor `PM_AUTONOMOUS_WRITES` is touched, so per the gate table (`docs/tool-surface.md` row 3)
   this mount exposes **17 tools**: the 12 PM tools *plus the 5 default librarian tools*
   (`sweep patrol propose_fix query record_feedback`). Neither the `pm-operator` agent's
   `tools:` allowlist (`agents/pm-operator.md:13-26`, all 12 PM names, zero librarian names)
   nor any of the 3 desk-pm skills (`pm-session-open`, `pm-advance-item`, `pm-triage` — grepped,
   0 hits for `sweep|patrol|propose_fix|record_feedback`) references any of the 5 ride-along
   tools. They are reachable by whatever agent/session has this mount active but claimed by no
   instructions at all.
2. **The same ride-along exists *inside the binary*, one level worse.** When `PM_ENABLED=true`,
   `module.Register` (`internal/core/module/module.go:74-95`, gate check `:78`, merge `:95`;
   comment at `:1-6`: "lets a second module's tools appear on every surface... without those
   surfaces knowing the module exists") merges the PM module's 12 tools into the same
   process-global registry S4 (`agent.go:109`)
   and S1 both read from. So the **in-binary eino agent** — the same loop that drives `chat`
   (S3) — receives all 12 PM tools whenever PM is on. But its system prompt
   (`librarian/templates/librarian-system-prompt.txt:1-33`) is the **librarian-only** prompt: it
   lists exactly 7 tools (`query sweep patrol propose_fix apply_fix restore record_feedback`)
   and never mentions `get_context`/`create_item`/`transition_item`/etc. This is a stronger
   instance of finding 1 — not a mount surface the operator chose to attach, but the *model's
   own instructions* silently going stale the moment a second module is enabled. This directly
   grounds README.md's C6 (prompt governance) and is new evidence for C1.
3. **The librarian system prompt is stale even with PM off.** Independent of finding 2, the
   same prompt (`librarian-system-prompt.txt:5-14`) unconditionally lists `apply_fix` and
   `restore` as available tools. `restore` is **never** given to the agent under any
   configuration (`toolcore.go:144-150`), and `apply_fix` is absent unless
   `LIBRARIAN_AUTONOMOUS_WRITES=true` (default off). So on a **default-configuration desk**
   the agent's own prompt claims 2 tools it does not have.
4. **`profile_get` and `knowledge_index` are reachable but claimed by nobody.** All 4 Claude
   Code plugin skills were grepped for all 4 TS tool names; only `template_render` appears
   (`desk-setup`, `brownfield-adoption`). `profile_get` and `knowledge_index` are mounted
   (`plugin/claude-plugin/.mcp.json:1-6`) and listed in every `tools/list` response (S7's fixed
   4), but no shipped skill's instructions tell an agent to call them. `profile_validate` is
   functionally implied once (`brownfield-adoption/SKILL.md:120`, "validate it against the
   shipped schema") but never named — an agent following that line has to guess the tool exists
   and find its name from the MCP tool list itself, not from the skill.
5. **`import` is reachable by no surface, human or agent.** The PM manifest importer
   (`librarian/internal/modules/pm/importer/importer.go:1-20`) is real, tested
   (`importer_test.go`, `rebuild_test.go`), and load-bearing for the §10.8 rebuild-reproducibility
   guarantee — but it has no CLI subcommand, no MCP tool, no TUI affordance, and no admin-console
   equivalent beyond raw bulk inserts. Its only caller today is the test/scenario harness
   (`pm/scenario/scenario.go:144`). This is the mirror-image gap of findings 1-4: a *capability*
   with zero claiming surface at all, reserved explicitly for a not-yet-built D8 adoption path
   (see the package doc comment, `importer.go:8-10`).
6. **The admin console claims everything and is claimed by no persona.** S5 is not named in
   any skill, the `pm-operator` persona, or the librarian system prompt — it is a human-only
   path by convention, not by any technical restriction beyond requiring a superuser login. See
   §7 for why that matters more than an ordinary ride-along.

## 7. Persona inventory

| Persona / instruction set | File | Tools it actually references (by name) |
|---|---|---|
| **In-binary eino agent system prompt** (drives S3 chat + S4 `agent`) | `librarian/templates/librarian-system-prompt.txt` | `query sweep patrol propose_fix apply_fix restore record_feedback` (7) — no PM tools ever, even when mounted (finding 2); includes 2 tools (`apply_fix`, `restore`) it may not actually hold (finding 3) |
| **`pm-operator` agent persona** | `plugin/desk-pm/agents/pm-operator.md` (frontmatter `tools:`, lines 13-26) | all 12 PM tools by name, zero librarian tools — a clean claim of exactly its intended surface |
| **`pm-session-open` skill** | `plugin/desk-pm/skills/pm-session-open/SKILL.md` | `get_context` (the whole skill), `claim_item`/`release_item` (picking the next item) |
| **`pm-advance-item` skill** | `plugin/desk-pm/skills/pm-advance-item/SKILL.md` | `get_item`, `transition_item`, `update_item` (for `pointer`), `add_note` |
| **`pm-triage` skill** | `plugin/desk-pm/skills/pm-triage/SKILL.md` | `create_item`, `link_items`, `block_item`/`unblock_item`, `update_item` (priority), `list_items` |
| **`desk-setup` skill** | `plugin/claude-plugin/skills/desk-setup/SKILL.md` | `template_render` only (`:8,68`) |
| **`brownfield-adoption` skill** | `plugin/claude-plugin/skills/brownfield-adoption/SKILL.md` | `template_render` (`:34,110`); validation implied but unnamed (`:120`) |
| **`harvest-loop` skill** | `plugin/claude-plugin/skills/harvest-loop/SKILL.md` | no tool named; references "the MCP tools" generically (`:35`) for cross-desk ledger gathering, unspecified which server |
| **`conventions-standard` skill** | `plugin/claude-plugin/skills/conventions-standard/SKILL.md` | no tool named; references "the MCP tools" generically (`:24`) as how the deskkit's rules are reached |
| **desk-pm `SessionStart` hook** | `plugin/desk-pm/hooks/session-briefing.sh` | calls the **CLI** directly (`deskkit pm context`, line 26/28) — not an MCP tool call at all; degrades silently (exit 0) if PM is off or `deskkit` is not on PATH |

## 8. Gaps & uncertainties

- **Empirical re-run not performed.** This dossier verified `docs/tool-surface.md`'s counts by
  re-reading the same source it cites (`toolcore.go`, both `specs.go` files, `registerToolCommands`/
  `registerPMCommands`, `plugin/core/tools.ts`) and found no discrepancy, but did **not** rebuild
  the `deskkit` binary or re-run the doc's JSON-RPC probe — no build step was taken in this
  file-scoped slice. If the design session wants a fresh empirical count, re-run the probe in
  `docs/tool-surface.md`'s "How the counts were derived" section.
- **TUI write-affordance check is code-level, not behavioral.** §3 states the 3 PM TUI views
  (`views.go`) bind no write keypress beyond `r` (refresh) — verified by reading every
  `tea.KeyPressMsg` branch in `views.go`, but the chat TUI's underlying agent loop can still
  invoke PM write tools via natural-language tool-calling if the model chooses to (S4's tool
  set, gated only by `PM_AUTONOMOUS_WRITES`) — the TUI *keybindings* are read-only; the
  *session* is not, because it is the same eino loop as `chat`.
- **`profile_validate`'s "implied but unnamed" status is a judgment call, not a bright line.**
  `brownfield-adoption/SKILL.md:120` tells an agent to validate the profile without naming the
  tool; whether that counts as "claimed" is exactly the kind of surface-vs-persona question this
  matrix exists to surface for the session, not resolve unilaterally here.
- **Admin-console reachability of gated PM writes (§3, §5) is stated as a structural fact**
  (nil API rules -> superuser bypass is standard PocketBase behavior, and no per-collection
  hook enforcing the state machine was found in `pm/collections/collections.go`), but this
  dossier did not attempt a live write against a running store to empirically confirm a
  superuser-authenticated console edit skips `engine.Transition`'s gate check — that would
  require standing up a store, which was out of scope for a file-only slice.
- **`import`'s D8 status is read from the importer package's own doc comment**
  (`importer.go:1-20`), not from a roadmap document — treat "reserved for D8" as the package
  author's stated intent, not a ruled commitment.
- **Scope boundary.** This dossier does not evaluate whether any of these asymmetries is
  *correct* (several are clearly deliberate — `restore`/`findings dispose` supervised-only by
  design) — it only maps what exists and what is claimed where, per the brief. The
  model/workflow rulings (C1-C8 in `README.md`) determine which findings above are policy and
  which are debt.

_First agent-led integration test pass across both desk-standard products. Answers "does this
actually work when a real agent drives it", not just "do the unit tests / verify.sh pass."_

Status: active
Date: 2026-07-20

# Agent-led integration testing — first pass

## 1. Why this exists

Before this session, desk-standard had thorough coverage of two kinds: unit tests (bun 75,
go 594) and a scripted CLI integration gate (`librarian/verify.sh`, 48 checks — build, sweep,
patrol, propose-fix, apply-fix, restore, store self-init, XDG resolution, desk open-guard).
None of that ever drove the product **the way an actual agent or user would**: through the
plugin's MCP tools as a live Claude Code session would call them, through the librarian's own
real LLM-driven agent loop (`deskkit agent`), through a live multi-message MCP protocol session,
or through the PM module's work-graph tools in a realistic multi-step sequence. This was a real
gap, not a paperwork one — several of the bugs below only exist at the intersection of "real LLM
in the loop" and "process built/run from an unusual path," which no unit test or scripted CLI
gate would ever hit.

## 2. Method

Foreman-led crew, two phases:

- **Phase 1 — 3 parallel testers**, each owning a disjoint surface, working only in isolated
  scratch dirs plus exactly one new repo file each (no edits to any existing tracked file, no
  commits): Track 1 (plugin/ TS MCP surface + the 4 Claude Code skills), Track 2 (librarian core
  — the real `deskkit agent` LLM loop + a live MCP protocol session), Track 3 (PM module — a
  realistic block/unblock/transition/note/claim/release work-graph session).
- **Phase 2 — 2 independent adversarial reviewers**, each re-deriving/re-running the highest-impact
  claims from the testers' reports from scratch (own builds, own scratch desks) rather than
  trusting the prose. 9 claims reviewed: 8 CONFIRMED, 1 PLAUSIBLE, 0 REFUTED.

Repo hygiene held throughout: `git status --porcelain` at the end shows exactly 3 new untracked
files (`plugin/scripts/verify-mcp-protocol.sh`, `librarian/dogfood-agent.sh`,
`librarian/dogfood-pm.sh`) and zero modified tracked files; `node scripts/check-neutrality.mjs`
passes clean including the 3 new files (their `.sh` extension is in-scope).

## 3. Track 1 — plugin/ TS MCP surface + skills

**Connectivity finding (open question, not a confirmed desk-standard bug):** this repo's
`.claude/settings.json` has `"desk-standard@desk-standard": true` in `enabledPlugins`, resolving
via `.claude-plugin/marketplace.json` to `./plugin/claude-plugin`, which declares an MCP server
exposing the plugin's 4 tools. Despite that, **neither the original tester nor either of the two
independent reviewers could get those 4 tools to surface as agent-callable tools in a live
session working in this exact repo** — one saw `ToolSearch` return "No matching deferred tools
found," another saw `ToolSearch` itself unavailable in their session. Both point the same
direction (enabled in settings ≠ actually usable by an agent) but via different failure modes,
and nobody diagnosed *why* (harness plugin-loading timing, `bun`-launch issue, or something else)
— this needs a from-scratch Claude Code session restart to pin down, not just file-reading.
**This is the single most important finding in this whole pass**, because it means the plugin's
entire value proposition — an agent using these tools inside Claude Code — has not been proven to
work in the one place it matters most (this repo, with its own plugin enabled).

**Tool-level correctness (fully confirmed, twice independently):** with native connectivity
unavailable, both the tester and a reviewer drove the real, git-tracked
`plugin/claude-plugin/mcp/server.js` directly over stdio JSON-RPC (harness now saved at
`plugin/scripts/verify-mcp-protocol.sh`). All 4 tools work correctly on both happy and fail-loud
paths:
- `profile_get` — resolves real dotted-path values; on a missing key, fails loud and **names the
  available sibling keys** (found-twice-identical behavior).
- `profile_validate` — passes a valid profile; on an injected unknown top-level key, correctly
  reports the schema violation.
- `template_render` — resolves `{{profile.*}}` placeholders and `||` fallbacks; on a required
  placeholder with no default and no matching profile key, fails loud rather than silently
  substituting an empty string.
- `knowledge_index` — indexes real markdown content, with correct greedy budget-fit behavior
  (`contentIncluded:false` on files that don't fit).

One design note (not a bug): the server resolves the profile from the **server process's cwd**,
not any client-supplied context — `createServer()` is called with no context in `main()`. Fine
for a per-desk stdio-spawned server; a sharp edge if ever spawned from a fixed location while the
user works elsewhere.

**Skills walked for real:** the `desk-setup` greenfield runbook was executed step-by-step against
a scratch desk (steps 1-6 completed for real via the tools above — including a full round-trip of
`template_render` against the actual scaffold's `{{profile.*}}` placeholders, confirmed zero
remain afterward; steps 7-8 are inherently no-ops for a throwaway desk with no real host app or
overseen repo). The `conventions-standard` adherence checklist was then run against the resulting
desk (8 of 9 rules clean pass; 1 rule surfaced the frontmatter-exemption gap below).
`brownfield-adoption` and `harvest-loop` were checked for dangling references — none found, but
one internal-consistency smell: `harvest-loop/assets/improvement-log.md` and
`desk-setup/assets/template/_meta/improvement-log.md` are two divergently-drifted copies of what
should probably be one owned template.

## 4. Track 2 — librarian core, the real agent loop

**The real `deskkit agent` LLM loop** (real ANTHROPIC_API_KEY, eino ReAct loop) was driven against
a scratch desk seeded with 5 fixtures (R1/R2/R3/R4/R5, deliberately distinct from `verify.sh`'s
own fixtures). Tool-call sequence chosen by the model: `sweep → patrol → query(findings) →
query(summary) → propose_fix` — a sane sequence. First run hit `AGENT_MAX_STEP`'s default (12)
and aborted (see the transcript-persistence bug below); raising it to 30 let the model
self-correct (it had chained `propose_fix --run <this patrol's id>`, which is idempotent-empty
since patrol had already run, and independently reissued `propose_fix` without a run filter to
pick up the real open findings). With `LIBRARIAN_AUTONOMOUS_WRITES=true`, a real fix was applied
and independently verified byte-exact reversible via `restore --by-path` (sha256 before-fix ==
sha256 after-restore, both differing from the intermediate modified hash) — confirmed twice, once
via the agent path and once via a reviewer's plain-CLI repro.

**A live, multi-message MCP protocol session** (not a one-shot `tools/list` probe) was driven
against `deskkit mcp-serve`: `initialize → notifications/initialized → tools/list → tools/call`
for all 5 default tools (`sweep`, `patrol`, `propose_fix`, `query`, `record_feedback`) and, with
`LIBRARIAN_AUTONOMOUS_WRITES=true`, the 6th (`apply_fix`), which was confirmed to actually write
to disk over the wire. Tool counts matched `docs/tool-surface.md` exactly (5/6).

## 5. Track 3 — PM module, real-agent workflow

**Architectural finding, confirmed via code + the existing ADR:** `deskkit agent`'s eino loop
hardcodes `toolcore.SelectByModules(toolcore.AgentTools(cfg), "librarian")` — it can **never**
call PM tools, even with `PM_ENABLED=true`, by design (`docs/decisions/0014-agent-integration-
contract.md`, clause (c)). The tester tried the real-agent path first per the brief; the model
correctly refused to fabricate PM behavior and only logged feedback about the mismatch — the
right behavior for a scoped-out capability, not a bug. Fell back (as instructed) to a
deterministic MCP protocol session against `deskkit mcp-serve` with `PM_ENABLED=true
MCP_MODULES=pm`.

**Full realistic work-graph sequence, driven and verified over the wire:** create item A, create
item B, link `A --blocks--> B` (B comes up `blocked=true` immediately as a cascade side-effect,
not deferred) → `transition_item(B)` while blocked is **genuinely refused** (`isError:true`, named
reason) → `unblock_item(B)` succeeds → `transition_item(B)` now succeeds (phase `queue→work`) →
`add_note` → `claim_item` (`claimed_by:"agent"`) → `release_item` (cleared). Full version-counter
progression and audit trail confirmed via `get_item` at each step.

**Module gating confirmed empirically, twice:** `MCP_MODULES=pm` exposes exactly the 12 documented
PM tools and **none** of the 5 librarian ride-alongs — matches `docs/tool-surface.md` §2.1 exactly.

## 6. Confirmed bugs filed as issues

Ranked by severity; all 4 were independently re-derived by a reviewer (code citation and/or live
reproduction), not just the original tester's claim.

| Severity | Issue | One-line |
|---|---|---|
| High | [#148](https://github.com/hsb3/desk-standard/issues/148) | Building/running `deskkit` from a path under `$TMPDIR` (e.g. `mktemp -d`) silently triggers PocketBase dev-mode, dumping raw SQL to stdout — would corrupt a real MCP client's JSON-RPC stream, not just scripts. Reproduced twice independently (676 / 664 corrupted lines). |
| Medium | [#149](https://github.com/hsb3/desk-standard/issues/149) | The real agent loop can silently omit a state-mutating tool call from its own `messages` transcript when it hits `AGENT_MAX_STEP` right after that call — the store's `revisions` table shows the mutation happened, the transcript doesn't. An audit-trail integrity gap, reproduced live (code path + a targeted low-MaxStep repro). |
| Low | [#150](https://github.com/hsb3/desk-standard/issues/150) | `deskkit pm --help` shows `--actor` as if usable before the leaf subcommand; it must go after (cobra has no `TraverseChildren`). |
| Low | [#151](https://github.com/hsb3/desk-standard/issues/151) | `conventions-standard`'s frontmatter-exemption prose checklist doesn't name `_knowledge/README.md` (or `_knowledge/` as an exempt entity dir), even though the plugin's own shipped scaffold ships that exact unfrontmattered file — the machine gate (patrol, by-basename) is already correct, only the human checklist prose is out of sync. |

## 7. Findings not yet filed (backlog candidates — judgment calls, not clear-cut bugs)

- **Plugin MCP non-connectivity in a live session** (§3) — the most consequential open question
  in this report, but not filed as a desk-standard code bug because it's unclear whether the
  cause is in this repo at all (could be Claude Code plugin-loading/session-timing behavior
  outside desk-standard's control). Worth a from-scratch session test before deciding whether
  there's anything here to fix.
- **`propose_fix --run <id>` scoping trap** (`internal/modules/librarian/tools/propose_fix.go:34-37`)
  — findings are stamped with the patrol run that *created* them; a fresh idempotent re-patrol
  doesn't restamp existing open findings, so an agent naturally chaining "patrol → propose_fix
  --run &lt;that patrol's id&gt;" gets a silently empty result even though real open findings exist.
  Observed live (the Track 2 agent hit exactly this on its first run) but not one of the 9 claims
  put through the adversarial review pass.
- **`AGENT_MAX_STEP` default (12) is tight** — a plain 3-tool-call task with normal reasoning
  turns can exhaust it; the dogfood scripts raise it to 30.
- **LLM classification non-determinism** — across two otherwise-identical runs, the model
  classified the same fixture (R2/R3-shaped) as mechanical-and-fixed in one run and
  judgment-requiring in another, even though the tool's own `fixableRules` set treats both as
  deterministic. Not a tool bug — a product characteristic worth knowing if run-to-run
  consistency is ever assumed.
- **Two divergent `improvement-log.md` template copies** (`harvest-loop/assets/` vs.
  `desk-setup/assets/template/_meta/`) — drifted from each other; possibly deliberate
  (reference template vs. seeded stub) but worth harvest-loop's owner taking a look.

## 8. What's left behind, reusable

Three new scripts (all git-status-clean, all pass `check-neutrality.mjs`, all pass `shellcheck`
with only cosmetic findings):

- `plugin/scripts/verify-mcp-protocol.sh` — deterministic, no LLM needed, drives the real plugin
  MCP server over stdio JSON-RPC against a target desk root.
- `librarian/dogfood-agent.sh` — **manual, needs a real `ANTHROPIC_API_KEY`, makes real LLM
  calls** — not wired into `make verify`/CI on purpose (cost + non-determinism); reproduces the
  sweep/patrol/real-agent-loop/restore/MCP-protocol sequence from §4. 19/19 checks passing.
- `librarian/dogfood-pm.sh` — deterministic (MCP-protocol-only, no LLM), reproduces the PM
  work-graph sequence + module-gating check from §5. 13/13 checks passing.

None of these are wired into `make check`/`make verify`/CI — that's a deliberate follow-on
decision, not an oversight: the deterministic two could reasonably join a gate, the real-LLM one
should stay a manual/opt-in dogfood check given cost and non-determinism.

---
title: "Dual-format fan-out: Claude + OpenCode plugin instances from a common core"
type: spec
status: parked
created: 2026-07-23
purpose: "Capture the residual design/build work for #12 in scheduleable shape while it stays parked, so an owner can pick it up cold when the >= v1.0.0 gate lifts."
notes: "Tracks #12. Parked by owner ruling (Henry, 2026-07-17) until >= v1.0.0 (_meta/HANDOFF.md:73; docs/decisions/0021-desk-standard-1-0-0-direction.md:102-104; docs/CHARTER.md:38). This plan does not unpark the issue — it documents current state and the unpark condition."
---

# Dual-format fan-out (Claude + OpenCode from one core)

*Scope: the RESIDUAL only — a common-core-authored `plugin/core` + `plugin/mcp` already ship
(harness-pure, tested); the Claude Code production instance already ships self-contained
(`plugins/desk-standard/`, drift-guarded). What's missing is a production step that emits a second,
OpenCode instance from that same core, plus the design call on whether/how the frozen adapter spike
feeds it.*

Status: draft (parked)
Date: 2026-07-23

## Tracking

#12, "desk-standard dual-format: produce Claude + OpenCode plugin instances from a common core
(fan-out build)". Originally relocated from `hsb3/dotfiles-agents-workbench#50` (executive-desk
decision 0016). **Parked by owner ruling (Henry, 2026-07-17) until >= v1.0.0** — recorded at
`_meta/HANDOFF.md:73` ("OpenCode #12 parked (Henry, 2026-07-17) until >= v1.0.0") and reaffirmed by
`docs/decisions/0021-desk-standard-1-0-0-direction.md:102-104` ("the Claude-only v1 scope (Claude +
OpenCode dual-format stays post-1.0)") and `docs/CHARTER.md:38` ("Scope is Claude Code only in v1;
OpenCode is a deferred, separate fan-out build"). No epic parent found for #12 (not listed under
either `epic-schema-v2-track` (#130) or the 1.0.0 wave epic (#129) at the time of this refresh).
This plan does not change the park; it exists so an unparking owner has schedule-ready detail
instead of a stale, template-nonconformant body. Companion issue body: `issue-body.md` in this
folder.

## The problem (grounded in source)

**What EXISTS today toward the fan-out** (verified against the tree, 2026-07-23):

- **The authored core is already harness-pure and shared.** `plugin/core` (profile
  discovery/loading, schema-v1 validation, `{{profile.…}}` substitution, `_knowledge/` indexing)
  imports no MCP/OpenCode/Claude-hook/Bun-specific API, enforced by
  `scripts/check-core-purity.mjs` (`plugin/README.md:5-8`; CLAUDE.md:114). `plugin/mcp` is a thin
  stdio server exposing exactly four tools whose schemas/behavior are defined once in `core/`
  (`plugin/README.md:9-11`).
- **The Claude instance already IS a production step's output, not a hand-adapter.**
  `cd plugin && bun run package` (`plugin/package.json:17`) builds `plugin/mcp/server.ts` with Bun
  into a self-contained `plugins/desk-standard/mcp/server.js` (committed, 757KB — bundles its own
  deps) and copies `schema/profile.schema.yaml` + `schema/references.yaml` into
  `plugins/desk-standard/schema/`. The shipped `.mcp.json` wires it with
  `${CLAUDE_PLUGIN_ROOT}/mcp/server.js` (`plugins/desk-standard/.mcp.json:2-4`) — no repo-relative
  path, no `.ts` source dependency at install time. A CI step regenerates and diffs it
  (`.github/workflows/ci.yml:139-147`, `git diff --exit-code`) so a hand-edited generated artifact
  fails the build. This resolves what the pre-refresh issue body still listed as an open packaging
  deferral.
- **The old path is gone.** `plugin/claude-plugin/` (the path the pre-refresh body cited) moved to
  `plugins/desk-standard/` in PR #191 (`59556b9`, "extract marketplace bundles to top-level
  plugins/ tree"); the old directory survives on disk only as an empty folder holding a stray
  `.DS_Store`.
- **An OpenCode spike exists, frozen, as a design reference — not a production step.**
  `plugin/opencode/plugin.ts` (98 lines) is a hand-written `Plugin`-typed OpenCode module already
  demonstrating the target shape for tool registration: `resolvePluginRoot()` (env override then
  module-relative path, `plugin.ts:47-52`), `mcpServerCommand()` (`plugin.ts:59-61`), and
  `registerMcpServer()` — idempotent, "never double-registers" (`plugin.ts:68-77`) — wired into a
  `DeskStandard: Plugin` export that registers the shared MCP server via OpenCode's `config` hook,
  fail-open on error (`plugin.ts:85-96`). Its own README states the disposition plainly: "descoped
  2026-07-16… treat it as an artifact of record, not a surface to extend"
  (`plugin/opencode/README.md:3,12-16`) -- though the README's "not wired into the package
  scripts, the manifest, or CI" wording is stale on two of three counts: `plugin/package.json:13`
  scopes `"test"` to `bun test core mcp opencode desk-persona`, so its 8 tests run under
  `make test` and CI (`.github/workflows/ci.yml:151`); only "not the manifest" holds. It
  typechecks, its tests pass under the default suite, but it ships nothing (root
  `README.md:33-35`: "`plugin/opencode/` holds a frozen, unwired adapter spike… and ships
  nothing").
- **The old citation surface for this issue is gone.** `docs/build-brief.md` (cited by the
  pre-refresh body for AC1/AC3, D4/D5) does not exist at that path; `_meta/build-brief.md` (the
  file other in-repo docs actually cite historically, e.g. `docs/development/README.md:62`,
  `schema/README.md:9,54`) is also missing from the current tree. `docs/CHARTER.md` is the
  canonical direction page now (CLAUDE.md:1-3, precedence rule) and is the correct citation target
  going forward.

**What is MISSING (the true residual):**

- No production step emits an OpenCode instance. The `package` script in `plugin/package.json:17`
  only builds the Claude side.
- No decision recorded on how the frozen `plugin/opencode/plugin.ts` spike relates to that future
  production step — promoted wholesale as the generated shape, regenerated from scratch with the
  spike kept only as a reference, or something in between. The spike's own README explicitly
  defers this ("kept as reference for the fan-out design", not itself the answer).
- No OpenCode-side manifest/packaging shape has been designed (there is no OpenCode equivalent of
  a `plugin.json`/marketplace descriptor in-tree today).
- **A found, unrelated staleness** (not this plan's to fix — outside this plan's owned files):
  root `README.md:33-34` still points the OpenCode-scope tracking pointer at the retired
  `hsb3/dotfiles-agents-workbench#50` rather than `desk-standard#12`, even though #12's own body
  records the relocation ("Relocated from `hsb3/dotfiles-agents-workbench#50` per executive-desk
  decision 0016 — this work serves desk-standard, not the retired workbench"). Flagged here as a
  residual doc fix for whoever owns `README.md` next; not performed by this plan.

## Deliverables

Mirrors `issue-body.md`'s Deliverables A-D (not restated in full here); the residual design work
this plan adds on top:

| Slice | What it resolves | File scope (primary) |
| ----- | ----------------- | --------------------- |
| Design note | The two open judgment calls: (1) production-step shape - extend `plugin/package.json`'s existing `package` script vs. a new sibling script/target; (2) the frozen spike's fate - promote / regenerate-fresh / reference-only | new short design note (location owner's choice at unpark time, e.g. `_meta/plans/dual-format-fanout/design.md`) |
| A/B (issue-body) | The production step + OpenCode instance registration | `plugin/package.json`, a new `plugin/opencode/` production path (or its replacement), `plugins/` (new sibling to `desk-standard/`) |
| C (issue-body) | Fidelity-loss documentation | same design note or a dedicated doc |
| D (issue-body) | Drift guard + neutrality coverage for the new artifacts | `.github/workflows/ci.yml`, `scripts/check-neutrality.mjs` scope (already covers `plugin/` recursively - no script change expected, verify at build time) |

## Gate & contract hygiene

| Gate | Fires? | Note |
| ---- | ------ | ---- |
| Repo checks (`make check`) | YES, always | neutrality + self-test, core-purity, etc. |
| Unit tests (`make test`) | YES, always | plugin `bun test`; new OpenCode-instance tests join it |
| Librarian integration (`make verify`) | NO | nothing under `librarian/` is touched by this issue |
| Bundle drift guard (`make package` + `git diff --exit-code`) | YES, once the new production step exists | extend the existing Claude-side pattern (`.github/workflows/ci.yml:139-147`) to the new OpenCode output rather than inventing a second mechanism |
| Identity neutrality (`check-neutrality.mjs` + self-test) | YES, always | scope already covers `plugin/` recursively; no scope-widening needed, just new files landing inside it |
| Version sync (`check-version-sync.mjs`) | CONDITIONAL | only if an OpenCode-side manifest is added to the shipped-manifest list; unresolved until the design note picks a shape |
| Kits drift | NO | no `kits/` change implied |
| CHANGELOG | YES, at release | gates the tag, not the PR - applies whenever this eventually ships |
| Regression-test bar | YES | new production step + registration logic need a red-able test, matching `plugin/opencode/plugin.ts`'s existing `plugin.test.ts` pattern |

## Parallelism + landing order

Not applicable while parked — no build work is authorized. When unparked: the design note (the
production-step shape + the spike's fate) should land FIRST and serialize ahead of any code, since
both later deliverables (the production step itself, and the OpenCode registration) depend on that
call. Once the design note lands, the production-step build and the fidelity-loss documentation
deliverable can proceed as one slice (small surface, same PR) rather than split further; no
meaningful parallelism to plan across separate builders for a change this size.

## Open questions / owner decisions

1. **Production-step shape** — extend the existing `plugin/package.json:17` `package` script to
   also emit an OpenCode instance, or add a separate sibling script (e.g. `package:opencode`)?
   No default recommended here — genuinely open, deferred to the unparking owner/builder, since it
   depends on how independently the two instances need to be regenerable (e.g. CI wanting to gate
   them separately vs. together).
2. **Fate of `plugin/opencode/plugin.ts`** — promote it wholesale into the generated instance,
   regenerate from scratch and demote the spike to pure documentation, or something in between?
   **Recommended default: promote its registration logic** (`resolvePluginRoot`,
   `mcpServerCommand`, `registerMcpServer`) as the seed of the generated module — it already
   satisfies the "port the capability, not the implementation" contract (fail-open, idempotent,
   config-hook-only, no re-implemented tools) and has passing tests today, so discarding it would
   re-derive already-verified logic for no stated benefit.
3. **Sequencing re-check** — the issue's live 2026-07-16 owner comment ordered #12 last, after
   #8 -> #13 -> #7, because "every earlier item still churns that core." Whoever unparks #12 should
   re-verify #8/#13/#7's current state before scheduling; this plan does not re-verify it (out of
   this refresh's scope — a park-time snapshot, not a live triage).

## Unpark condition

Do not schedule build work until the repo reaches >= v1.0.0 (per the governing dependency in
`issue-body.md`'s Dependencies & gates). At that point: re-run this plan's "What is MISSING"
section against the then-current tree (the Claude-side production step or the frozen spike may have
moved again), resolve Open question 1 and re-confirm/revise the Open question 2 default, and only
then slice into buildable PRs.

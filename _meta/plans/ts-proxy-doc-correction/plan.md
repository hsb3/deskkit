---
title: "ts-proxy-design.md stale-reference correction — build plan"
type: spec
status: planned
created: 2026-07-23
purpose: "Correct docs/development/ts-proxy-design.md's retired plugin/desk-pm/ path and 'desk-pm mount' label, and re-verify its proxying rationale against the current plugins/desk-persona/ bundle composition before any wording is fixed."
notes: "Issue #199 (open, label: documentation, owner-gated). Surfaced during wave-v4's reconciliation pass (2026-07-22); needs design-doc-owner judgment, not a blind rename."
---

# ts-proxy-doc-correction - build plan

_Correct the 11 retired `desk-pm` references in `docs/development/ts-proxy-design.md` (8 line
locations) and re-verify the doc's proxying rationale against the current
`plugins/desk-persona/` composition, which has drifted further than a rename since the design was
written -- its `.mcp.json` now mounts `MCP_MODULES=librarian,pm`, not the `pm`-only value the
design's Deliverable-B reasoning is built on. No RESIDUAL shipped work exists here; this is a
docs-only correction, not an implementation slice._

Status: draft
Date: 2026-07-23

## Tracking

- Issue #199 (`issue-body.md` in this folder) -- open, label `documentation`, owner-gated.
- Design doc under correction: `docs/development/ts-proxy-design.md` (Status: Active, 2026-07-20).
- Governing ADR: `docs/decisions/0016-ts-boundary-deskkit-proxy.md` (Accepted, 2026-07-20) -- this
  plan does not amend the ADR; it corrects a design doc that elaborates it.
- Predecessor renames this issue continues: PR #180 (desk-pm folded into desk-persona,
  2026-07-21), PR #191 (bundles moved to top-level `plugins/`, 2026-07-22), #181/#187 ("desk-pm
  mount" label renamed to "pm-only mount" across `docs/tool-surface.md`, the MCP server, its
  tests, and the e2e suite -- `CHANGELOG.md:50-52`).
- Gates: `_meta/HANDOFF.md:65-66` -- this issue blocks the ts-proxy implementation's slice 0
  (host spawn-feasibility probe), not the other way around.
- Contract impact: none. No API, DB, codegen, or other source-of-truth surface is touched; this
  edits one `docs/development/` file.

## The problem (grounded in source)

### What exists today: the stale-reference inventory (re-verified, not re-derived from the issue)

`grep -n "desk-pm" docs/development/ts-proxy-design.md` returns exactly 11 lines, at 8 distinct
locations:

| Line(s) | Content | Kind |
| --- | --- | --- |
| 50 | "the librarian surface is mounted separately, by the desk-pm bundle" | label |
| 51-52 | `plugin/desk-pm/.mcp.json` starts `deskkit mcp-serve` with `PM_ENABLED=true` + `MCP_MODULES=pm` | path + stale env value |
| 117 | "the desk-pm mount today" | label |
| 127 | "The desk-pm silent-drop is bad because..." | label |
| 161 | "the desk-pm bundle already owns the PM mount" | label (load-bearing rationale) |
| 162 | `(plugin/desk-pm/.mcp.json:1-10, MCP_MODULES=pm)` | path + stale env value |
| 181 | "without duplicating the desk-pm mount" | label |
| 335 | citations list: `plugin/desk-pm/.mcp.json:1-10` -- desk-pm mounts... | path + label |
| 343 | "desk-pm mount = 12 PM tools only" (citing `docs/tool-surface.md:117-120`) | label + drifted citation |
| 346 | "desk-pm mount owns the 12 PM tools" (citing `docs/agent-integration-contract-v1-spec.md:100`) | label + citation to an equally-stale doc |

**Line-number verification against the issue's filing.** #199's body cites "~50-52, 117, 127,
161-162, 181, 335, 343, 346" -- this is an EXACT match to the re-derived list above (11 lines
across 8 locations), no drift. The filer's numbers hold precisely against current `main`.

### What has ALSO changed since the design was written (the load-bearing new finding)

The design doc's ground-truth section (`docs/development/ts-proxy-design.md:50-62`) asserts: "The
librarian surface is mounted separately, by the desk-pm bundle... so that mount carries exactly
the 12 PM tools and none of the 5 librarian ride-alongs." This was true when written, but is no
longer the current state:

- `plugins/desk-persona/.mcp.json:1-8` (read directly):
  ```json
  {
    "desk-persona": {
      "command": "deskkit",
      "args": ["mcp-serve"],
      "env": { "PM_ENABLED": "true", "MCP_MODULES": "librarian,pm" }
    }
  }
  ```
  `MCP_MODULES` is `librarian,pm`, not `pm`. This exposes 17 tools: the 5 librarian defaults
  (`query`, `sweep`, `patrol`, `propose_fix`, `record_feedback`) plus the 12 PM tools --
  corroborated by `plugins/desk-persona/.claude-plugin/plugin.json` ("17 tools") and
  `plugins/desk-persona/README.md`'s own "## The composed mount" section, which states this
  explicitly: "a single MCP server exposing 17 tools, zero phantom entries... 5 librarian tools
  (always-on slice)... 12 PM tools."
- The 5 librarian tools are claimed, not a ride-along: `plugins/desk-persona/agents/librarian-operator.md:14-18`
  lists `mcp__desk-persona__query`, `sweep`, `patrol`, `propose_fix`, `record_feedback` in its
  `tools:` frontmatter. This is a real, claimed, marketplace-published surface --
  `.claude-plugin/marketplace.json` lists both `desk-standard` and `desk-persona` as installable
  plugins side by side (verified: `cat .claude-plugin/marketplace.json`).
- **Timeline, which explains why the design doc missed this rather than ignored it.** The design
  doc merged in PR #142 (commit `4356945d1`, 2026-07-20 15:52:50 -0400). The desk-persona bundle
  that set `MCP_MODULES=librarian,pm` merged in PR #143 (commit `ec1cd6acc`, 2026-07-20 16:03:18
  -0400) -- about ten minutes later, same day. `grep -c "desk-persona" docs/development/ts-proxy-design.md`
  returns 0: the design doc never mentions desk-persona at all. It was authored just ahead of its
  own sibling composition change and was never reconciled against it once that change landed.
  (Verified via `git log --follow` on both files; see Evidence below for exact commands.)

### Does the design's proxying argument hold, hold with amendment, or is it invalidated?

This is the design-doc owner's call -- the finding below is evidence, framed with a recommended
default, not a unilateral edit to the doc's reasoning.

**Evidence for "holds, with a named amendment" (the recommended default):**

1. ADR 0016's actual motivation is narrower than "does anything in the marketplace expose
   librarian tools" -- the owner's sign-off quote is specifically about the TS plugin seam:
   "we need to/should take advantage of the typescript plugin seam to enable server-backed
   capabilities" (`docs/decisions/0016-ts-boundary-deskkit-proxy.md:14-16`). `desk-persona`
   existing does not change what `desk-standard`'s own TS boundary can do.
2. `desk-standard` and `desk-persona` are two separately installable plugins
   (`.claude-plugin/marketplace.json`). A desk that installs only `desk-standard` (the lighter
   plugin -- 4 fixed profile/template/index tools, no PM prerequisites) gets nothing from
   desk-persona's mount. `desk-persona`'s README lists real prerequisites the lighter plugin does
   not need: `deskkit` on PATH, PM enabled + migrated, `desk_config` seeded
   (`plugins/desk-persona/README.md` "## Prerequisites"). The proxy's continuing value is
   "librarian capability inside the lighter, PM-free install," which desk-persona does not
   substitute for.
3. The design's own duplication analysis (`:161-181`) was already reasoning about avoiding
   double-mounting one tool family (PM) between two Claude Code surfaces. The same reasoning
   pattern extends cleanly to the newly-real second case (librarian tools now also live on
   desk-persona's mount): for a desk that installs BOTH bundles, the TS proxy's `query`/`sweep`/
   etc. and desk-persona's `query`/`sweep`/etc. are two independently-namespaced tool paths to the
   same underlying capability (`mcp__desk-standard__query` vs `mcp__desk-persona__query` -- MCP
   tool names are server-name-prefixed, so this is not a naming collision), not a contradiction --
   redundant, not broken. That redundancy is a fair thing for the doc to name and accept
   explicitly, rather than the silent gap it is today.

**What would flip the verdict to "invalidated":** if the design-doc owner judges that a desk
installing `desk-standard` without `desk-persona` is not a real/supported configuration worth
building a proxy for (e.g. if v1's practical guidance is "install desk-persona for any librarian
access"), the proxy loses its stated rationale and ADR 0016 would need to be reopened per its own
escape hatch (`docs/decisions/0016-ts-boundary-deskkit-proxy.md:37-38`, "reopen ADR 0016 with
Option A") rather than patched via this doc. (This flip condition is the plan's own framing; the
ADR's stated escape-hatch trigger is a design blocker such as lifecycle coupling -- either route
uses the same reopen mechanism.) Nothing found in this pass forces that reading, but
it was not ruled out either -- it is the genuine open call, not a foregone conclusion.

**Recommended default:** HOLDS WITH AMENDMENT. Keep the proxy design; add an explicit paragraph
(likely a new subsection near current section 3.3, `docs/development/ts-proxy-design.md:146-181`)
that (a) names `plugins/desk-persona/`'s current `MCP_MODULES=librarian,pm` composition as ground
truth, (b) states the proxy's distinct value (`desk-standard`-only installs), and (c) accepts the
desk-standard/desk-persona librarian-tool overlap as intentional redundancy across two
independently-namespaced mounts, not a defect to design around.

## Deliverables

- **A -- Verify (this plan's own output).** Re-derive the stale-reference inventory and the
  desk-persona ground truth from source; land the recommended default above. **Shipped by this
  plan and its `issue-body.md`** -- no further build action needed for this slice.
- **B -- Amend the reasoning.** A human or agent with design-doc-owner authority confirms (or
  overrides) the "holds with amendment" verdict above, and writes the amendment paragraph into
  `docs/development/ts-proxy-design.md` (recommended location: a new subsection after current
  section 3.3, before section 3.4). This is the one non-mechanical step; it must land BEFORE C,
  since C's rewritten wording depends on the confirmed verdict, not the draft one.
  - **Acceptance:** the doc contains an explicit "holds / holds-with-amendment / invalidated"
    statement naming `plugins/desk-persona/`'s actual `MCP_MODULES=librarian,pm` composition, not
    the retired `pm`-only value.
- **C -- Mechanical path/label sweep.** Once B's verdict is confirmed, replace all 11 stale
  `desk-pm` references (8 locations: `:50-52, 117, 127, 161-162, 181, 335, 343, 346`):
  `plugin/desk-pm/` to `plugins/desk-persona/`; "desk-pm mount" to "pm-only mount" or
  "desk-persona bundle" per context; `MCP_MODULES=pm` to `MCP_MODULES=librarian,pm` where the
  citation is describing desk-persona's actual mount (not a hypothetical `pm`-only mount, which
  remains a real, separately-documented shape at `docs/tool-surface.md:127-129, 142`).
  - **Acceptance:** `grep -n "plugin/desk-pm" docs/development/ts-proxy-design.md` and
    `grep -n "desk-pm" docs/development/ts-proxy-design.md` both return zero hits.
- **D -- Re-point drifted citations.** `:343`'s citation to `docs/tool-surface.md:117-120` no
  longer contains the "12 PM tools only" claim it cites (that content is now at
  `docs/tool-surface.md:127-129, 142`, the "pm-only mount" table row, after the #181/#187 rename
  pass touched the TARGET doc without touching this citing doc). Re-verify and re-point every
  citation touched by C at fix time, not from this plan's line numbers (which can drift again
  before the fix lands).
  - **Acceptance:** every citation in the corrected doc resolves to a line range that still
    contains the claim it cites, spot-checked by re-opening each cited `path:line`.

## Gate & contract hygiene

ASCII-only cells; gate commands run bare, per `_config.md`.

| Gate | Command / trigger | Fires? | Why |
| ---- | ----------------- | ------ | --- |
| Repo checks (aggregate) | `make check` | NO (docs-only) | `docs/` is not in the neutrality scan's `SCAN_DIRS` (`scripts/check-neutrality.mjs:54`); no kit/scaffold/purity/actionlint surface touched |
| Identity neutrality | `node scripts/check-neutrality.mjs` | NO | scope is `["plugin", "plugins", "librarian", "kits"]`; `docs/` exempt |
| Unit tests | `make test` | NO | no `plugin/` or `librarian/` code change |
| Librarian integration | `make verify` | NO | no `librarian/` change |
| Bundle drift guard | `make package` + `git diff` | NO | no `plugin/core`, `plugin/mcp`, or `schema/` change |
| Version sync | `check-version-sync.mjs` | NO | no `VERSION`/manifest change |
| Kits drift | `check-kits.mjs` | NO | no `kits/` change |
| CHANGELOG | `[Unreleased]` entry; `check-changelog.mjs` at release | NO | a docs-only citation fix to an unshipped design is not a product change; the eventual ts-proxy implementation issue carries its own entry |
| Pre-commit | lefthook | YES | always |
| Regression-test bar | red-able test per fix (#82 bar) | NO | not applicable to a prose/citation correction; the acceptance criteria (grep-zero, citation re-verification) are this change's test |

## Parallelism + landing order

**Nothing here parallelizes across owners.** This is a single small file
(`docs/development/ts-proxy-design.md`) with one dependency chain: B (the owner's verdict) must
land before C (the wording it drives) and D (the citation fixes C's edits expose). Splitting B/C/D
across parallel agents would mean C guessing at a verdict B has not yet confirmed -- the exact
"blind rename" #199 was filed to prevent.

**Landing order:**

1. **A** -- already done (this plan + `issue-body.md`).
2. **B** -- design-doc owner confirms or overrides the "holds with amendment" default and writes
   the amendment paragraph.
3. **C** -- mechanical sweep of all 11 stale lines, informed by B's confirmed wording.
4. **D** -- re-point the two collateral citations (`docs/tool-surface.md`,
   `docs/agent-integration-contract-v1-spec.md`), verified fresh at fix time.
5. Final gate: `grep -n "desk-pm" docs/development/ts-proxy-design.md` returns zero hits.

**Out-of-scope follow-up, not part of this landing order:** `docs/agent-integration-contract-v1-spec.md:30-31,
100` carries the same stale `plugin/desk-pm` / "desk-pm mount" pattern (found during this pass,
`grep -n "desk-pm" docs/agent-integration-contract-v1-spec.md`) but is not named by #199's scope.
Left for a separate filing; not a dependency of B/C/D above.

## Open questions / owner decisions

1. **Does the proxy's rationale hold, hold with amendment, or is it invalidated?**
   Recommended default: **holds with amendment** (see "Does the design's proxying argument hold"
   above for the full evidence). Owner: the design-doc owner (ts-proxy-design.md), confirmed
   before Deliverable C's wording is finalized.
2. **Where should the amendment paragraph live?** Recommended default: a new subsection after
   current section 3.3 (`:146-181`), since that is where the "avoid duplicating the desk-pm mount"
   reasoning already lives and the amendment extends the same argument. Owner: whoever lands
   Deliverable B; a different location is fine if it reads better in context.
3. **Should the out-of-scope `docs/agent-integration-contract-v1-spec.md` staleness be filed as
   its own issue now, or left for a later sweep?** Recommended default: file it separately rather
   than silently expanding this issue's scope, since #199 names only `ts-proxy-design.md` and the
   spec doc's staleness is a distinct, independently-scoped correction. Owner: whoever reviews this
   plan; not a blocker to B/C/D above.

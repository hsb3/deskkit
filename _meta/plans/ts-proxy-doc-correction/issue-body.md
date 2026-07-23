> **Tracking:** #199, filed 2026-07-22 during wave-v4's reconciliation pass (owner-gated:
> `documentation`, needs design-doc-owner judgment, not a mechanical rename).
> `docs/development/ts-proxy-design.md` still cites the retired `plugin/desk-pm/` path and the
> retired "desk-pm mount" label, and its proxying rationale needs re-verification against the
> current `plugins/desk-persona/` bundle composition before any wording is fixed.

## Problem

`docs/development/ts-proxy-design.md` (Status: Active, `docs/development/ts-proxy-design.md:8`)
is the design a future ts-proxy implementer is meant to pick up cold (its own framing,
`docs/development/ts-proxy-design.md:6`). It carries 11 references to the retired `desk-pm` name
across 8 line locations, re-verified by grep against the current tree:

- `docs/development/ts-proxy-design.md:50-52` -- ground-truth paragraph: "the librarian surface
  is mounted separately, by the desk-pm bundle," citing `plugin/desk-pm/.mcp.json` with
  `MCP_MODULES=pm`.
- `docs/development/ts-proxy-design.md:117` -- "the desk-pm mount" label, in the
  availability-behavior section.
- `docs/development/ts-proxy-design.md:127` -- "the desk-pm silent-drop," same section.
- `docs/development/ts-proxy-design.md:161-162` -- the load-bearing rationale line for
  Deliverable B / section 3.3: "the desk-pm bundle already owns the PM mount," citing
  `plugin/desk-pm/.mcp.json:1-10, MCP_MODULES=pm`.
- `docs/development/ts-proxy-design.md:181` -- "without duplicating the desk-pm mount."
- `docs/development/ts-proxy-design.md:335` -- citations section, `plugin/desk-pm/.mcp.json:1-10`.
- `docs/development/ts-proxy-design.md:343` -- "desk-pm mount = 12 PM tools only," citing
  `docs/tool-surface.md:117-120`.
- `docs/development/ts-proxy-design.md:346` -- "desk-pm mount owns the 12 PM tools," citing
  `docs/agent-integration-contract-v1-spec.md:100`.

This matches the issue's original approximate line list (~50-52, 117, 127, 161-162, 181, 335,
343, 346) exactly -- no drift found; the filer's line numbers hold precisely against current
`main`.

Two facts confirmed since the filing, both bearing on the fix:

1. **The rename target moved twice, so this is not a single find-replace.**
   `plugin/desk-pm/` folded into `plugin/desk-persona/` (PR #180, 2026-07-21 owner ruling), which
   then moved to `plugins/desk-persona/` when the marketplace bundles relocated out of `plugin/`
   (PR #191, 2026-07-22). The "desk-pm mount" label was separately renamed to "pm-only mount"
   across `docs/tool-surface.md`, the MCP server, its tests, and the e2e suite (#181, #187;
   `CHANGELOG.md:50-52`) -- but that rename pass's own file list did not include
   `docs/development/ts-proxy-design.md`, so this doc is a genuine miss, not a false positive.
2. **The cited `MCP_MODULES` value itself changed, not just the path.**
   `plugins/desk-persona/.mcp.json:1-8` sets `MCP_MODULES=librarian,pm` (confirmed live) -- not
   the `MCP_MODULES=pm`-only value the design doc's Deliverable-B rationale (`:161-162`) is built
   on. Git history shows this value has been `librarian,pm` since the bundle's creation (PR #143,
   commit `ec1cd6a`, 2026-07-20 16:03), which landed about ten minutes after the design doc itself
   merged (PR #142, commit `4356945`, 2026-07-20 15:52) -- the design was authored just before its
   own sibling change landed and was never reconciled against it. Practically:
   `plugins/desk-persona/` already exposes the same 5 default librarian tools (`query`, `sweep`,
   `patrol`, `propose_fix`, `record_feedback`) directly to Claude Code today, via the
   `librarian-operator` agent's `tools:` claim
   (`plugins/desk-persona/agents/librarian-operator.md:14-18`) over a direct Go `deskkit mcp-serve`
   mount -- a live, marketplace-published surface (`.claude-plugin/marketplace.json` lists
   `desk-standard` and `desk-persona` side by side). The design's proxying rationale (`:161-181`)
   reasons about avoiding one duplication (proxying PM tools, when desk-pm/desk-persona already
   owns the PM mount) but does not address a second, now-real one: whether proxying the 5
   librarian tools through the desk-standard TS boundary duplicates desk-persona's existing
   librarian-tool exposure, for any desk that installs both bundles.

This second point is why the fix needs the design-doc owner, not a find-replace: the doc's
reasoning, not just its wording, may need an added paragraph reconciling the proxy's continuing
rationale (a lighter `desk-standard`-only install path to librarian tools, versus the heavier
`desk-persona` composition) against the new fact that `desk-persona` already ships that
capability today via a different mount. See this folder's `plan.md` for the full evidence trail
and a recommended default.

## Deliverables

- **A -- Verify the proxying argument.** Confirm (or correct) whether ADR 0016's motivation
  (`docs/decisions/0016-ts-boundary-deskkit-proxy.md:14-16`, the desk-standard TS plugin seam
  specifically) still justifies the proxy now that `plugins/desk-persona/` independently exposes
  the 5 librarian tools via a direct Go mount. Land the verdict as an explicit paragraph in the
  design doc, not a silent assumption.
- **B -- Mechanical path/label sweep.** Replace every `plugin/desk-pm/` path reference with
  `plugins/desk-persona/`, and every "desk-pm mount" / "desk-pm bundle" label with "pm-only mount"
  or "desk-persona bundle" as contextually correct, across all 11 stale lines
  (`docs/development/ts-proxy-design.md:50-52, 117, 127, 161-162, 181, 335, 343, 346`).
- **C -- Correct the `MCP_MODULES` value.** Update the `MCP_MODULES=pm` citations (`:52, 162`) to
  the current `MCP_MODULES=librarian,pm`, and re-derive whatever downstream claim depended on the
  old value (chiefly `:161-181`'s "avoid duplicating the desk-pm mount" reasoning, per
  Deliverable A's verdict).
- **D -- Re-verify collateral citations.** `:343` cites `docs/tool-surface.md:117-120` for
  "desk-pm mount = 12 PM tools only"; that specific claim has since moved to
  `docs/tool-surface.md:127-129, 142` (the "pm-only mount" table row) as part of the same
  #181/#187 rename pass -- the citation's line range drifted even though the underlying fact is
  still true. Re-point the citation.

## Acceptance criteria

- [ ] `grep -n "plugin/desk-pm" docs/development/ts-proxy-design.md` returns zero hits.
- [ ] `grep -n "desk-pm" docs/development/ts-proxy-design.md` returns zero hits (covers the
      "desk-pm mount" / "desk-pm bundle" label too).
- [ ] The doc's Deliverable B / section 3.3 rationale names the CURRENT bundle
      (`plugins/desk-persona/`) and its actual `MCP_MODULES=librarian,pm` composition, not the
      retired `pm`-only value.
- [ ] The doc contains an explicit statement of whether the proxy's rationale holds, holds with
      the stated amendment, or is invalidated by desk-persona's existing librarian-tool exposure
      -- signed off by the design-doc owner, not left implicit.
- [ ] Every re-pointed citation (`docs/tool-surface.md`, `docs/agent-integration-contract-v1-spec.md`)
      resolves to a line range that still contains the cited claim, re-verified at fix time (line
      numbers may have drifted again since this filing).
- [ ] `node scripts/check-neutrality.mjs` is unaffected (docs/ is exempt; confirms no accidental
      scope creep into `plugin/` or `librarian/`).

## Dependencies & gates

This is a `docs/`-only change (`docs/development/ts-proxy-design.md`), so most repo gates do not
fire:

- Repo checks / neutrality / core purity / bundle drift / version sync / kits drift -- NO. None of
  `plugin/`, `librarian/`, `schema/`, `VERSION`, or `kits/` are touched; `docs/` is explicitly
  outside the neutrality scan's scanned dirs (`scripts/check-neutrality.mjs:54`,
  `SCAN_DIRS = ["plugin", "plugins", "librarian", "kits"]`).
- Unit tests / librarian integration (`make test`, `make verify`) -- NO. No code changes.
- CHANGELOG -- NO. A docs-only correction to an unshipped design's citations is not itself a
  product change; the eventual ts-proxy implementation issue
  (`docs/development/ts-proxy-design.md` section 5) carries its own CHANGELOG entry when it ships.
- Pre-commit (lefthook) -- YES, always.

**What this issue in turn gates:** `_meta/HANDOFF.md:65-66` records the ts-proxy
implementation's slice 0 (host spawn-feasibility probe) as blocked behind this issue's doc
correction -- this issue is the prerequisite for that go/no-go, not the other way around.

## Out of scope

- Implementing the ts-proxy itself (`plugin/mcp/proxy.ts`, slices 0-6 of
  `docs/development/ts-proxy-design.md` section 5) -- this issue only corrects the design doc's
  references and reasoning.
- Fixing the same stale `desk-pm` references in `docs/agent-integration-contract-v1-spec.md:30-31,
  100` -- found during this investigation but out of scope for #199, which names only
  `docs/development/ts-proxy-design.md`. Flagged as a related follow-up, not filed here.
- Extending or amending ADR 0016 itself -- if Deliverable A's verdict is "invalidated" rather than
  "holds with amendment," that is an ADR-level call (a correction or supersession per the repo's
  ADR discipline), handled separately from this doc-reference fix.

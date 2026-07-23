# _meta/plans

Build-ready planning docs — one subfolder per unit of work (`<slug>/plan.md`, plus
`<slug>/issue-body.md` and any screenshots/artifacts). The board tracks STATE; this desk holds the
build DETAIL.

This desk is **git-tracked** (`_meta/` is tracked by default), so the plans + `_utils/` scripts are visible
in a fresh clone, in worktrees, and in cloud sessions. A plan's
tracking issue is the first `#NNN` in its `plan.md` `## Tracking` section.

See `_config.md` for THIS project's gate menu, issue-template sections, and canonical docs to cite.

## Toolkit (`_utils/`)

Seven dependency-free scripts, each a generated VIEW over `gh` + disk (never hand-maintained state),
each with `--json` and a non-zero exit on findings so it can gate a wave. Run them from the main tree
where `gh` is authed: `python3 _meta/plans/_utils/<script>`.

| Script | What it checks | Mutates? |
| ------ | -------------- | -------- |
| `reconcile.py` | plan-table rows vs live issue state vs disk folders vs `plan.md` Tracking | no |
| `conformance.py` | every open issue body vs its `.github/ISSUE_TEMPLATE` required sections | no |
| `coverage.py` | open non-epic issues with NO plan folder (the planning backlog) | no |
| `sequence.py` | tiers ALL open issues into NOW / NEXT / BLOCKED / DEFERRED | no |
| `deps-suggest.py` | candidate native blocked-by edges harvested from prose `#refs` (advisory) | no |
| `sync-bodies.py` | each folder's `issue-body.md` vs the live GitHub body | only `--push`/`--pull` |
| `evidence-audit.py` | recently-closed issues that closed with no linked PR and no evidence | no |

## Status

The status tables below are hand-maintained, so they drift from live issue state. Run
`python3 _meta/plans/_utils/reconcile.py` to cross-check every row against GitHub and the folders on
disk before trusting the table. It exits non-zero on any drift.

ACTIVE plans (associated with an OPEN issue) — add a row per plan you draft:

| Plan | Issue(s) | Status |
| ---- | -------- | ------ |
| phase-machine-reconciliation | #197 | planned; gate:v2-final; epic #130; the one buildable-now item |
| ts-proxy-doc-correction | #199 | planned; owner-gated (design-doc judgment); gates ts-proxy slice 0 |
| trigger-design | #127 | planned; backlog, unscheduled; schema-v2 model track |
| prompt-tuning-centralized | #128 | planned; backlog, unscheduled; ADR 0015 track |
| sop-library-expansion | #36 | parked post-1.0 (owner, 2026-07-21); plan captures unpark state |
| dual-format-fanout | #12 | parked until >= v1.0.0 (owner, 2026-07-17); plan captures unpark state |

`epic-schema-v2-track/` additionally holds the staged body for open epic #130 (no plan.md — epics
are containers; the epic closes when #197 closes).

ARCHIVED (issue closed/merged; folder moved to `_meta/_archive/<issue>-<slug>/`, content intact):

| Plan (archived path) | Issue(s) | Why archived |
| -------------------- | -------- | ------------ |
| _archive/114-agent-integration-contract/ | #114 | shipped; epic #129 v1 build wave |
| _archive/115-pointer-grammar-spec/ | #115 | shipped; epic #129 v1 build wave |
| _archive/116-typed-reference-contract/ | #116 | shipped; epic #129 v1 build wave |
| _archive/117-item-type-validation/ | #117 | shipped; epic #129 v1 build wave |
| _archive/118-findings-lifecycle-completion/ | #118 | shipped; epic #129 v1 build wave |
| _archive/119-desk-persona-bundle/ | #119 | shipped; epic #129 v1 build wave |
| _archive/120-prompt-governance/ | #120 | shipped; epic #129 v1 build wave |
| _archive/121-tool-surface-drift-guard/ | #121 | shipped; epic #129 v1 build wave |
| _archive/122-ts-proxy-design/ | #122 | shipped; design doc landed (docs/development/ts-proxy-design.md) |
| _archive/123-document-identity-hygiene/ | #123 | shipped; epic #129 v1 build wave |
| _archive/124-schema-versioning/ | #124 | shipped wave-v4 (PR #193) |
| _archive/125-element-model-revision/ | #125 | shipped wave-v4 (PR #192) |
| _archive/126-model-simulations/ | #126 | shipped wave-v4 (PR #198); v2 residual filed as #197 |
| _archive/129-epic-design-session-v1/ | #129 | epic closed 2026-07-20 (PRs #136-#147) |
| _archive/189-tui-ux-onboarding/ | #189 | shipped wave-v4 (PR #195) |
| _archive/190-user-dev-docs-split/ | #190 | shipped wave-v4 (PR #195) |

## The loop (how plans get produced)

A repeatable cycle: **draft in parallel** (one agent per plan, every claim cited to `path:line`) →
**review adversarially** against the rubric (a read-only verifier per plan, told to re-derive and
refute, run on your strongest model tier) → **fix** (handed the verified facts) → **augment the
rubric + re-review** → **apply bodies + reconcile** (gate the wave on `reconcile.py` clean). Trust
the findings; treat any 1-5 scores as directional. A perfect sweep is a red flag, not a triumph.

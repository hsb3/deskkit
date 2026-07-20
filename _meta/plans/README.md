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

ARCHIVED (issue closed/merged; plan moved to `_meta/_archive/<issue>-<slug>.md`):

| Plan (archived path) | Issue(s) | Why archived |
| -------------------- | -------- | ------------ |

## The loop (how plans get produced)

A repeatable cycle: **draft in parallel** (one agent per plan, every claim cited to `path:line`) →
**review adversarially** against the rubric (a read-only verifier per plan, told to re-derive and
refute, run on your strongest model tier) → **fix** (handed the verified facts) → **augment the
rubric + re-review** → **apply bodies + reconcile** (gate the wave on `reconcile.py` clean). Trust
the findings; treat any 1-5 scores as directional. A perfect sweep is a red flag, not a triumph.

# Planning desk — project config

_The project-specific bindings the planning-desk authoring modes read. Written at setup by
detecting this repo's gates and templates; update it when the project's gates change._

Repo: `hsb3/desk-standard`  ·  Set up: `2026-07-20`

## Issue templates

The conformance gate keys on the load-bearing sections, not exact wording:

- **feature / bug** (non-epic): an **Acceptance criteria** section + a **Dependencies & gates** section.
- **epic / tracker**: a **Close when** section.

| Template | Path | Required sections present |
| -------- | ---- | ------------------------- |
| feature | `.github/ISSUE_TEMPLATE/feature.yml` | Tracking; Problem; Deliverables; Acceptance criteria; Dependencies & gates; Out of scope |
| bug | `.github/ISSUE_TEMPLATE/bug.yml` | Acceptance criteria; Dependencies & gates |
| epic | `.github/ISSUE_TEMPLATE/epic.yml` | Tracking; Why; Children; Close when |

## Gate menu

The gates a change must account for in an issue's **Dependencies & gates** and a plan's **Gate &
contract hygiene**. A plan picks the subset its change surface actually touches and is explicit
about which do NOT fire. Run gate commands **bare, never piped** (a pipe masks the exit code —
2026-07-17 incident).

| Gate | Command / trigger | Fires when the change touches... |
| ---- | ----------------- | ------------------------------ |
| Repo checks (aggregate) | `make check` (neutrality + self-test, kit drift, scaffold frontmatter, core purity, actionlint) | always |
| Unit tests | `make test` (plugin `bun test` + librarian `go test ./...`) | always |
| Librarian integration | `make verify` (`librarian/verify.sh`, scratch desk) | anything under `librarian/` |
| Bundle drift guard | `make package` then `git diff --exit-code` on `plugin/claude-plugin/` | `plugin/core`, `plugin/mcp`, `schema/` |
| Identity neutrality | `node scripts/check-neutrality.mjs` (scope: `plugin/` + `librarian/` recursively; bare `#N` refs in Go comments/tests FAIL; `docs/` exempt) | any shipped-tree text |
| Version sync | `node scripts/check-version-sync.mjs` | `VERSION` or a shipped manifest |
| Kits drift | `node scripts/check-kits.mjs` | `kits/` or `kits.yaml` |
| CHANGELOG | entry under `[Unreleased]`; `check-changelog.mjs` hard-gates the release tag | any product change |
| DB migration discipline | forward migration only, never edit an applied one; content `TextField`s need explicit `Max`; enum shrink needs a data-first down-remap (0010/0012 precedent) | a PocketBase collection |
| Regression-test bar | every fix carries a red-able regression test (the #82 bar); data-safety slices get independent adversarial review (PR #112 practice) | any behavior change |
| Pre-commit | lefthook mirrors CI lanes | always |

## Canonical docs to cite

A load-bearing claim cites `path:line` OR one of these spine docs:

- `docs/CHARTER.md` — canonical direction; precedence rule (charter wins).
- `docs/pocket-librarian-v1-spec.md` and `docs/pm-system-v1-spec.md` — the two build specs
  (paths are load-bearing; cited from code + the neutrality allowlist).
- `docs/decisions/` — ADRs 0001–0018; 0009–0018 are the 2026-07-20 design-session rulings
  every feature lane binds to.
- `docs/tool-surface.md` — empirically-verified tool counts (ADR 0016 makes it drift-guarded truth).
- `schema/` — the shared contract both lanes read.
- `CLAUDE.md` — the agent digest (commands, generated artifacts, cross-cutting gotchas).

## Conventions / gotchas

- ASCII-only inside markdown table cells; no literal `|` in cells.
- Trunk-based: branch `<type>/<short-name>`, squash-merge PRs, commit prefixes
  `feat|fix|docs|refactor|style|harden(scope):`.
- `Resolves #N` in a commit auto-closes on push to main — post the proof comment first.
- Identity-neutrality is the one rule that matters: nothing under `plugin/` or `librarian/`
  hardcodes a person/org/repo/issue. Issue-body prose lives in `_meta/` and `docs/` (exempt).
- Owner decision batches go through the owner-signoff HTML form (never chat questions);
  decision packages are deck/PDF, never terminal markdown.
- Run the `_utils/` scripts from the main working tree (they read live `gh` state + disk).
- `_meta/plans/inbox/` receives communication packages from the strategy desk (repo-meta-structure
  standard); it is not a plan folder.

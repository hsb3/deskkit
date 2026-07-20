# Plan - desk-persona bundle

_Build plan for the composed desk-persona Claude Code bundle (librarian + PM under the contract)._

Status: draft (2026-07-20)

## Tracking

- Issue: #119 (this plan's tracking issue)
- Blocked by: agent-integration-contract (#114)
- ADR: `docs/decisions/0014-agent-integration-contract.md` clause (a); binds
  `docs/decisions/0015-prompt-governance.md` (git-is-truth prompt copies)
- Evidence: `_meta/research/2026-07-design-session/decision-book/D5-agent-contract-and-parity.md`,
  `_meta/research/2026-07-design-session/agent-symmetry.md`,
  `_meta/research/2026-07-design-session/surface-matrix.md`

## Problem (grounded in source)

ADR 0014(a) rules ONE desk-persona bundle composing the librarian + PM personas under the contract
- the platform v1 proof surface (ADR 0009). Today the marketplace lists only `desk-standard`
(source `./plugin/claude-plugin`) and `desk-pm` (source `./plugin/desk-pm`), each versioned
(`.claude-plugin/marketplace.json`); the librarian has no Claude Code bundle, only a README
snippet. Consequences on `main`:

- the librarian tools a session needs are claimed by no persona or skill
  (`_meta/research/2026-07-design-session/surface-matrix.md` findings 1, 4);
- desk-pm is an authored bundle: `.claude-plugin/plugin.json`, `.mcp.json` (mounts
  `deskkit mcp-serve` with `PM_ENABLED=true`), `agents/pm-operator.md`,
  `skills/{pm-session-open,pm-advance-item,pm-triage}/SKILL.md`, `hooks/`, `README.md` — no
  generated TS `server.js`, no drift guard today (it mounts the Go binary).

Packaging facts the bundle must satisfy:

- `scripts/check-neutrality.mjs:52` scans `plugin/`, `librarian/`, `kits/` - a new
  `plugin/desk-persona/` is auto-in-scope.
- `scripts/check-version-sync.mjs` SOURCES enumerate both `plugin.json` files and both marketplace
  entries against root `VERSION`; a third bundle adds two SOURCES entries.
- `make package` (plugin `package.json` `package` script) only builds
  `plugin/claude-plugin/mcp/server.js` + copies the schema; a binary-mounting bundle like desk-pm
  is not part of it.
- Per ADR 0015, the bundle's librarian-persona markdown becomes a version-controlled prompt copy
  and needs a drift guard against the canonical `librarian/templates/librarian-system-prompt.txt`.

## Deliverables (file scopes + per-unit acceptance)

### A - Bundle layout

- **Scope:** new `plugin/desk-persona/` tree - `.claude-plugin/plugin.json`, `.mcp.json`,
  `agents/`, `skills/`, `README.md`.
- **Acceptance:** the tree mirrors desk-pm's shape; `.mcp.json` mounts `deskkit mcp-serve` with the
  module set the contract's gating expects.

### B - Composed persona / skill / agent

- **Scope:** `plugin/desk-persona/agents/*.md`, `plugin/desk-persona/skills/*/SKILL.md`.
- **Mechanism:** compose the librarian persona (claims `sweep patrol propose_fix query
  record_feedback`) with the PM contribution reused from desk-pm content - one source per surface.
- **Acceptance:** the persona names the previously unclaimed librarian tools; a grep/test asserts
  their presence.

### C - Packaging integration

- **Scope:** `.claude-plugin/marketplace.json` (new entry, source `./plugin/desk-persona`,
  versioned), `plugin/desk-persona/.claude-plugin/plugin.json`, `scripts/check-version-sync.mjs`
  (two new SOURCES entries).
- **Acceptance:** `node scripts/check-version-sync.mjs` passes with the new manifest + marketplace
  entry at `VERSION`.

### D - Drift guard

- **Scope:** a new guard script (e.g. `scripts/check-persona-drift.mjs`) wired into `make check`;
  the generated bundle librarian-persona body.
- **Mechanism:** generate the bundle's librarian-persona body from
  `librarian/templates/librarian-system-prompt.txt`; guard with regenerate + `git diff --exit-code`
  (the repo generated-artifact pattern).
- **Acceptance:** hand-editing the generated body fails the guard non-zero; regeneration passes.

### E - Neutrality + docs

- **Scope:** all `plugin/desk-persona/` text; `docs/README.md` pointer; the marketplace description.
- **Acceptance:** `node scripts/check-neutrality.mjs` passes with the tree present; `--self-test`
  still detects a seeded violation.

## Gate and contract hygiene

| Gate | Fires? | Why |
| ---- | ------ | --- |
| make check (neutrality + self-test) | yes | new persona/skill text under plugin/ |
| check-version-sync | yes | new plugin.json + marketplace entry |
| bundle persona drift guard (new, D) | yes | bundle markdown is a guarded prompt copy (ADR 0015) |
| make test (plugin bun test) | yes | if a bundle claim test is added as a bun test; default also touches `plugin/package.json` (append `desk-persona` to the test dir glob, `package.json:13`) — otherwise the check rides `scripts/`/Go and `package.json` is untouched |
| Regression-test bar (#82) | yes | the drift guard is the red-able check |
| make verify (librarian integration) | no | no Go code added; mounts the existing binary |
| DB migration discipline | no | no collection change |
| claude-plugin server.js drift guard | no | bundle carries no generated TS server |
| Kits drift | no | kits/ untouched |
| CHANGELOG | yes | new shipped artifact needs an Unreleased entry |

## Parallelism and landing order

- **Blocked by** agent-integration-contract - the bundle cannot land until the contract defines the
  persona claims, the mount's declared module set, and the prompt copy the drift guard targets.
- **Within this unit:** A (layout) and B (persona content) are sequential (B fills A's tree); C
  (packaging) and D (drift guard) can proceed in parallel once B's persona body exists; E
  (neutrality/docs) is a final sweep over the tree.
- **Landing order:** contract issue merges -> A -> B -> (C, D in parallel) -> E + CHANGELOG.
- **Shared-file caution:** `.claude-plugin/marketplace.json` and `scripts/check-version-sync.mjs`
  are repo-shared; if the desk-pm disposition (below) also touches the marketplace, serialize those
  edits under one owner - do not fan out marketplace edits in parallel.

## Open questions (recommended defaults)

- **Bundle location.** `plugin/desk-persona/` vs another root. **Default: `plugin/desk-persona/`** -
  mirrors `plugin/desk-pm/`, auto-in-neutrality-scope (`check-neutrality.mjs:52`), and the
  marketplace already sources bundles from `./plugin/*`. Not genuinely contested.
- **desk-pm disposition.** Does the composed bundle subsume desk-pm, or coexist? **Default: fold** -
  ADR 0014(a) says "ONE bundle ... not a third instantiation," so the PM contribution should be
  composed INTO desk-persona from the existing desk-pm content, and desk-pm-as-a-separate-marketplace-entry
  is superseded (one source, per ADR 0015). Folding/retiring desk-pm touches `plugin/desk-pm/` +
  the marketplace + version-sync SOURCES - flag it as an in-scope change to confirm with the
  maintainer before deleting anything (archive-not-delete per project protocol). Alternative
  (coexist) rejected: it duplicates the PM persona into two sources, violating single-sourcing.
- **Drift-guard relationship (D).** The canonical librarian prompt is an eino system prompt; the
  bundle persona is a Claude Code agent persona - not byte-identical genres. **Default: generate the
  bundle's librarian body from the canonical source** (include/transform), guarded by regenerate +
  `git diff --exit-code`, so there is exactly one editable source. Open detail: the transform (how
  much of the eino prompt maps to the persona body) is the design work of D; if a clean generation
  is infeasible, fall back to a structural check (the persona must name every tool in
  `ExposedTools(cfg)` for the librarian module and no phantom tool).
- **Mount module declaration.** The bundle's `.mcp.json` must declare the module set the contract's
  gating expects. **Default: declare both `librarian` and `pm`** (the composed desk agent owns
  both), gated via the same env-var mechanism the contract issue lands (see that plan's Open
  questions). This is why the bundle is blocked by the contract.
- **CHANGELOG entry.** New shipped artifact -> an `[Unreleased]` entry is required
  (`check-changelog.mjs` hard-gates the release tag). Default: add it in this issue.

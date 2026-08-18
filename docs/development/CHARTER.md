_The canonical page for desk-standard: what it is, what it governs, and the direction settled
for 1.0.0._
Status: active (2026-07-19)

> **Precedence.** If anything anywhere in this repo — a README, a spec, an ADR, a comment, a
> handoff — disagrees with this page, **this page wins.** Change the project's direction only
> here; everything else links to it rather than restating it.

# Charter

## What this is

**Two products over one shared schema, one personalization model.** Nothing that ships carries
a person, org, repo, or issue number — identity is injected at use time, never baked in.

- **`plugin/`** — the desk-standard Claude Code plugin: a harness-pure TypeScript core (profile
  loading, schema validation, `{{profile.…}}` substitution, `_knowledge/` indexing) behind a
  stdio MCP server, wrapped as a plugin with four skills.
- **`librarian/`** — **deskkit**, a single Go binary (embedded PocketBase) that indexes a desk,
  flags convention violations, and proposes/applies fixes under a record-original-first boundary
  (every applied fix is byte-exact reversible via `restore`). It also carries the **PM module**,
  a document-gated work graph. Surfaces: CLI, MCP server, and a chat TUI over one tool core.
  (Named `pocket-librarian` through v0.6.0; renamed **deskkit** in v0.7.0 — the binary is
  `deskkit`.)
- **`schema/`** — schema v1: the product-neutral contract both `plugin/` and `librarian/` read.

You personalize by filling `_knowledge/profile.yaml` (from `_knowledge/profile.example.yaml`) —
never by editing a shipped skill, template, or tool.

## What this governs

- **Identity-neutrality is a hard invariant**, enforced in CI by the neutrality lint over
  `plugin/` + `librarian/`. Shipped code names no deployment identity. (`docs/` and repo-root
  files are exempt and may cite issues/identity freely.)
- **One version, one repo.** Both products ship under the single root `VERSION`; a release tags
  that one version (SemVer policy: ADR 0005 (DESK-27)).
- **Decisions are ADRs**, append-only, filed on the project board as `DECISION` tasks and cited
  where they bind as a bare `ADR NNNN`.
- **Scope is Claude Code only** in v1; OpenCode is a deferred, separate fan-out build.

## Direction settled for 1.0.0

These are decided; open build detail lives in the roadmap and issues, not here.

- **1.0.0 is a maturity milestone, not a breaking-contract change.** The MAJOR bump signals the
  system is dogfooded and stable — a deliberate, documented deviation from strict SemVer,
  recorded when the release is cut (a note against ADR-0005).
- **The PM module ships default-on.** The 1.0 maturity flip (owner-ruled 2026-07-21; ADR-0008
  amendment) turns it on for every fresh desk; a desk opts out with `PM_ENABLED=false` or profile
  `modules.pm.enabled: false`.
- **The PocketBase-served webapp is in 1.0.0 scope.** The deferred chat surface moves out of
  post-1.0 deferral into a pre-1.0 build lane, amending
  ADR 0001 (DESK-23)'s TUI-first deferral; the
  serving-stack call is made when that lane starts, not here.
- **Public launch is deferred to ≥ 1.0.0.** The repo stays private until then; the public
  `curl | bash` install path is expected to 404 by design until launch — install from the
  private repo with authenticated `gh` in the meantime (see the root README).

## Where to go next

| For | Read |
|---|---|
| Working in the repo as an agent | [`CLAUDE.md`](../../CLAUDE.md) (command surface + the one rule) |
| The docs map (using vs developing) | [`README.md`](README.md) |
| The librarian's build spec | [`pocket-librarian-v1-spec.md`](specs/pocket-librarian-v1-spec.md) |
| The PM system's build spec | [`pm-system-v1-spec.md`](specs/pm-system-v1-spec.md) |
| Why a decision was made | the project board's `DECISION` tasks |

_Session-to-session bridge for desk-standard. Read this before working; update it at the end
of any substantial session. Secret-free — live URLs/credentials belong in `_meta/operations/`
(untracked) if that dir is ever created._
Status: active (2026-07-17)

# HANDOFF

## 0. Orientation

Two products over one shared schema, all identity-neutral (nothing shipped carries a
person/org/repo/issue): **`plugin/`** (harness-pure TS core + stdio MCP server, wrapped as a
Claude Code plugin with four skills) and **`librarian/`** (pocket-librarian: single Go
binary, embedded PocketBase, eino agent loop, record-original-first write boundary), with
**`schema/`** as the contract both read. Personalization is only via `_knowledge/profile.yaml`.
Root `README.md` is the front door; `docs/build-brief.md` is the build brief this repo was
built from; `docs/pocket-librarian-v1-spec.md` is the librarian's spec (build spec, not
operator docs — operator docs are `librarian/README.md`).

Siblings: `hsb3/dotfiles-agents` (the pattern source for workflows/templates); Henry's
executive desk at `~/Documents/EXECUTIVE_DESK/Projects/dev-tooling-desk` holds off-repo
decision records (e.g. 0013–0016) referenced from the build brief — annotated, not vendored.

## 1. Current standing + top priority

v1 (Claude Code only) is built, distributable, and CI-green on `main` — a live plugin
marketplace: `claude plugin marketplace add hsb3/desk-standard` →
`claude plugin install desk-standard@desk-standard` (proven end-to-end).

**PR #21 merged** (squash `e004f59`, 2026-07-17; four claude-review passes to a clean bill,
CI-green on main) — closed **#16 + #17**, the two correctness fixes from the 2026-07-16
dev-tooling-desk field evaluation (§2). Top priority is now **#18**.

Open backlog, ranked (field-eval findings interleaved with the distribution arc):
- **#18** — field-interaction UX batch (orphan noise from infra dirs, R4 severity ambiguity,
  mcp-serve EOF exit, JSON-only output); non-blocking cleanups. Two of the four are product
  rulings (orphan taxonomy; R4 mechanical-vs-judgment), two are clear fixes (EOF exit, pretty).
- **#20** — design session: multi-desk topology (7 desks → 7 stores vs 1; canonical store
  location; resolve the latent-but-forbidden `desk` field). A ruling, not code — gates any
  serious multi-desk story.
- **#7** — curl-able install.sh (absorbs librarian release binaries, configure, post-install
  verify; builds on the marketplace flow + `bun run package`).
- **#12** — dual-format common-core fan-out (Claude + OpenCode instances; consumes
  `bun run package` as the seed).
- **#19** — PB-served webapp chat surface (deferred interactive follow-on; ADR 0001).

## 2. Last session delivered (2026-07-17)

**#16 + #17** fixed, proven, and **merged** as **PR #21** (squash `e004f59`). Both were
findings from the 2026-07-16 dev-tooling-desk field evaluation. The review cycle ran four
claude-review passes (six findings total — 2 code tidy-ups, 2 test-gap fills, 2 doc/comment
accuracy) all resolved before merge; commits `2db9bf4` (hoist collection lookup + path in
errors + batch/seed tests), `4250219` (path-based test lookup + migration-comment asymmetry),
`59ae0d9` (ProposeFix docstring corrected to the per-file-tolerance contract).

- **#16** — record-original-first was silently capped at PocketBase's 5000-char TextField
  default: any desk file >5 KB could not have its byte-exact original recorded (the §5.4
  safety boundary failing on exactly the costly files), and one oversized file hard-aborted
  the whole propose-fix run. Fix: new migration `0011_widen_content_fields.go` widens the
  three content-bearing text fields (`revisions.original_content`, `messages.content`,
  `prompts.content`; `Max=50_000_000`) — applies to EXISTING stores on next migrate, via the
  0010 alter pattern. `propose_fix` now tolerates per-file failures (an `"error"` outcome
  mirroring `ApplyOutcome`) instead of aborting; boundary unweakened (an errored finding
  records no revision row → no fs write can follow). Regression test with a >5 KB fixture;
  forced-store-failure test rewritten to the tolerant contract. Live-proven: `propose-fix` on
  a 9512-char file returns `recorded` with the full original stored.
- **#17** — `prompt.Seed` was `OnServe`-only, so CLI/MCP-only desks never materialized the
  editable system-prompt row (§4.10). Moved the seed into `requireConfig` (the shared
  one-shot entry path), non-fatal; the `.librarian-ignore` half was already there. Live-proven:
  one-shot `sweep` seeds the `prompts` row.

The same field evaluation filed **#16/#17/#18/#20**; #18 and #20 remain open (§1 backlog).

## 2b. Earlier eras (full detail in merged PRs / closed issues / git)

- **2026-07-16 eve** — #13/#14/#15 via parallel worktree crews: `chat` REPL + trigger wake
  layer (ADR `docs/decisions/0001-interactive-surface-tui-first.md`, TUI-first; webapp
  deferred → #19), `secrets_ref.llm_api_key` indirection, superuser auto-create under serve,
  patrol stale-finding resolution (migration `0010`). Commits `d4c52aa`→`e851e5b`→merge
  `f148e04`; `verify.sh` 40→42 checks.
- **2026-07-16 day** — #8 (brownfield-adoption skill, plugin 0.3.0) + #9/#10/#11: README
  patrol exemption, in-repo marketplace + `bun run package` bundle (drift-guarded), issue/PR
  templates + live claude workflows, `librarian/README.md` operator docs.

## 3. Where to start building

Once PR #21 merges (closes #16/#17), the top code items are **#18** (UX cleanup batch —
safe, non-blocking) and the distribution arc **#7 → #12**. **#20** is a design session
(ruling first — store-per-desk vs shared, canonical store location, `desk`-field resolution —
then code). **#19** (webapp) is the deferred interactive follow-on; ADR 0001 records the
preferred shape (custom Go route, PB-served, no runtime frontend toolchain). Note the skill
files under `plugin/claude-plugin/skills/` are neutrality-lint-scanned (no bare issue refs,
GitHub URLs, or profile scalars in skill prose).

## 4. Conventions & gotchas

- **Gates** (run all before claiming done): `node scripts/check-neutrality.mjs` ·
  `cd plugin && bun run check:purity && bun test && bun run build` ·
  `cd librarian && go build ./... && go vet ./... && go test ./...` · `bash librarian/verify.sh`
  (42 checks) · `actionlint`. CI (`ci.yml`) is the aggregate required check.
- **Generated, never hand-edit**: `plugin/claude-plugin/mcp/server.js` and
  `plugin/claude-plugin/schema/profile.schema.yaml` — regen with `cd plugin && bun run package`;
  CI drift-guards them. Bundle output is byte-identical across macOS/linux (proven).
- **Neutrality lint scope** = `plugin/` + `librarian/` recursively. Bare issue refs (`#11`)
  in Go comments/tests inside `librarian/` FAIL the lint — write issue-free comments.
  `.claude-plugin/marketplace.json` (owner identity) is deliberately outside the surface
  (recorded in `docs/m-05-data-surfaces.md`).
- **Commits auto-close**: `Resolves #N` in a commit message closes the issue on push to
  main — post the proof comment first, or use `gh issue comment` after (close-with-comment
  fails on an already-closed issue).
- **Beware pre-staged files**: parallel agents may `git add` their scope; `git commit` with a
  pathspec still sweeps the whole index. Check `git status` before every commit.
- Librarian env: `DESK_ROOT`/`DESK_NAME` required by all tool commands; LLM only needed by
  `agent`/`chat`/MCP-driven calls (`LLM_PROVIDER` env → `profile.models` → anthropic; key
  from the env var named by `LLM_API_KEY_ENV` / `secrets_ref.llm_api_key`, else per-provider
  `ANTHROPIC_API_KEY`/`OPENAI_API_KEY`/`GEMINI_API_KEY`); `LIBRARIAN_AUTONOMOUS_WRITES=true`
  gates `apply_fix` (MCP and enqueued tasks — checked at execution time); `restore` is
  CLI-only. `serve` extras: `PB_SUPERUSER_EMAIL`/`PB_SUPERUSER_PASSWORD` (idempotent
  first-run superuser), `CLAIMER_POLL_INTERVAL` (wake-layer claimer).
- **PocketBase bare `TextField` silently caps at 5000 chars** (`Max==0` → default 5000, per
  `core.TextField`). Any field holding full file bodies / transcripts / editable prompts MUST
  set an explicit large `Max` or it truncates at 5 KB — the content fields are widened in
  migration `0011`. Set `Max` explicitly when adding a new content-bearing text field.
- **Altering a shipped collection: add a forward migration, don't edit the applied one**
  (precedent: `0010`, `0011`). Editing `000N`'s field decl only reaches fresh stores; a new
  migration that mutates the field via `FindCollectionByNameOrId` + `Save(c)` fixes existing
  stores on next migrate. Source decls carry a `// … widened in 0011` pointer comment.
- Go 1.25 floor (PocketBase's go.mod); Bun 1.3.14 pinned in CI.
- **Worktree provisioning**: untracked-and-load-bearing = `plugin/node_modules` (run
  `bun install --frozen-lockfile`) and `.claude/agent-memory/` (machine-local). Everything
  else a worktree agent needs is committed.
- `_meta/` here holds only this handoff so far — the full taxonomy (operations/ ignore
  stanza etc.) comes via the mise-en-place scaffold when needed.

## 5. Incident log

- 2026-07-16: first #11 commit accidentally swept 27 pre-staged pocketbase files into the
  fix commit (parallel agent had staged them); caught pre-push, reset --soft and recommitted
  per-stream. See the pre-staged-files gotcha above.

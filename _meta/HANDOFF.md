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

**All four dev-tooling-desk field-eval findings are now shipped**: **#18 merged** (PR #22,
squash `f0adc61`, 2026-07-17; CI-green on main, review "Ready to merge") closed the
field-interaction UX batch; **#16 + #17** landed earlier the same day (PR #21, §2b). Top
priority is now **#20** (a design ruling that needs your input — see §3) and, in parallel, the
distribution arc **#7 → #12** (pure build work, no ruling needed).

Open backlog, ranked:
- **#20** — design session: multi-desk topology (7 desks → 7 stores vs 1; canonical store
  location; resolve the latent-but-forbidden `desk` field). A ruling, not code — gates any
  serious multi-desk story. **Needs Henry.**
- **#7** — curl-able install.sh (absorbs librarian release binaries, configure, post-install
  verify; builds on the marketplace flow + `bun run package`).
- **#12** — dual-format common-core fan-out (Claude + OpenCode instances; consumes
  `bun run package` as the seed).
- **#19** — PB-served webapp chat surface (deferred interactive follow-on; ADR 0001).

## 2. Last session delivered (2026-07-17)

**#18** — field-interaction UX batch — fixed, proven, and **merged** as **PR #22** (squash
`f0adc61`). Four independent findings, one commit each; CI-green, three claude-review passes to
"Ready to merge". Two carried product rulings decided with Henry:

- **Item 1 — new `infra` dir_kind (RULING: add a dir_kind, not just filter).** `query orphans`
  was drowned in `.claude/**` / `.agents/**` / `.github/**` noise. `sweep` now buckets dotted
  infra dirs as `infra` (memory precedence for `.claude/memory/**` preserved); `isOrphan`
  excludes `dir_kind ∈ {meta, memory, infra}`. Migration `0012_dir_kind_add_infra.go` adds
  `infra` to the enum for existing stores (0001 decl carries it for fresh). Spec §5.1/§5.6 +
  schema table updated. Live-proven: infra/memory excluded, a genuine loose `.md` still flagged.
- **Item 2 — R4 reclassified mechanical → judgment (RULING: reclassify).** R4 detects
  mechanically but its remediation (which status to pick) is a judgment call; it was already
  flag-only (absent from `FIXABLE_RULES`). `mechanicalRules` → `{R1,R2,R3}`; patrol files R4 as
  `judgment`. Spec §5.2 updated.
- **Item 3 — `mcp-serve` clean EOF exit.** Client stdin-close gave exit 1 + `Error: server is
  closing: EOF` + usage dump. The SDK wraps its internal `ErrServerClosing` (`%w`) + `io.EOF`
  (`%v`), so `errors.Is(io.EOF)` misses it — `isShutdownEOF` matches the "server is closing"
  sentinel alone (deliberately: also covers a broken-pipe writeErr disconnect). Reproduced &
  fixed: exit 1 → exit 0.
- **Item 4 — `query --pretty` table output.** Presentation-only; raw JSON stays the default and
  is the fallback for any kind the renderer doesn't format.

Review cycle noted one accepted trade-off (migration 0012 down-path `return nil` mirrors the
0010/0011 precedent — swallows non-not-found errors; fine for a local tool).

## 2b. Earlier eras (full detail in merged PRs / closed issues / git)

- **2026-07-17 (earlier) — #16 + #17** (PR #21, squash `e004f59`; four review passes).
  **#16**: record-original-first was silently capped at PocketBase's 5000-char TextField
  default (>5 KB desk files couldn't be recorded; one oversized file aborted the run). Migration
  `0011_widen_content_fields.go` widens `revisions.original_content`/`messages.content`/
  `prompts.content` to `Max=50_000_000` (existing stores on next migrate); `propose_fix`
  tolerates per-file failures (boundary unweakened). **#17**: `prompt.Seed` was `OnServe`-only —
  moved into `requireConfig` so CLI/MCP-only desks materialize the editable system-prompt row.
- **2026-07-16 eve** — #13/#14/#15 via parallel worktree crews: `chat` REPL + trigger wake
  layer (ADR `docs/decisions/0001-interactive-surface-tui-first.md`, TUI-first; webapp
  deferred → #19), `secrets_ref.llm_api_key` indirection, superuser auto-create under serve,
  patrol stale-finding resolution (migration `0010`). Commits `d4c52aa`→`e851e5b`→merge
  `f148e04`; `verify.sh` 40→42 checks.
- **2026-07-16 day** — #8 (brownfield-adoption skill, plugin 0.3.0) + #9/#10/#11: README
  patrol exemption, in-repo marketplace + `bun run package` bundle (drift-guarded), issue/PR
  templates + live claude workflows, `librarian/README.md` operator docs.

## 3. Where to start building

With all field-eval findings shipped, the open work splits two ways. **Pure build (no ruling
needed):** the distribution arc **#7** (curl-able install.sh) **→ #12** (dual-format
common-core fan-out) — safe to start immediately. **Needs a ruling first: #20** is a design
session (store-per-desk vs shared, canonical store location, `desk`-field resolution — decide,
then code) — bring this to Henry before building. **#19** (webapp) is the deferred interactive
follow-on; ADR 0001 records the preferred shape (custom Go route, PB-served, no runtime
frontend toolchain). Note the skill
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
  (precedent: `0010`, `0011`, `0012`). Editing `000N`'s field decl only reaches fresh stores; a
  new migration that mutates the field via `FindCollectionByNameOrId` + `Save(c)` fixes existing
  stores on next migrate. Source decls carry a pointer comment (`// … widened in 0011`,
  `// "infra" added in 0012`). Two variants proven: **TextField Max** (`0011`) and **SelectField
  Values** enum-extension (`0012` adds `infra` to `dir_kind`). An enum-extension's DOWN migration
  must remap rows off the new value FIRST (`0012`: `infra`→`other`) before dropping it from the
  enum, or a rollback leaves a row outside its reverted enum (same data-first pattern as `0010`).
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
- 2026-07-17: a gate command piped to `tail` (`check-neutrality.mjs 2>&1 | tail -1 && git add`)
  masked neutrality's exit 1 — a pipeline's status is the LAST command's (`tail`=0), so the
  `&&` proceeded and a #18 commit landed locally with 4 bare-`#18` lint violations in
  `librarian/`. Caught on the next standalone run, `--amend`ed before push (nothing bad reached
  the remote). Lesson: when a command GATES a commit, run it bare and check its own exit code —
  never pipe it. (Neutrality scope reminder: `docs/` is exempt, so the spec's `#18` refs are
  fine; `librarian/` Go comments/tests are not.)

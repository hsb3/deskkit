_Session-to-session bridge for desk-standard. Read this before working; update it at the end
of any substantial session. Secret-free — live URLs/credentials belong in `_meta/operations/`
(untracked) if that dir is ever created._
Status: active (2026-07-16)

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

v1 (Claude Code only) is built, distributable, and CI-green on `main`. The repo is a live
plugin marketplace: `claude plugin marketplace add hsb3/desk-standard` →
`claude plugin install desk-standard@desk-standard` (proven end-to-end).

The interactive-surfaces arc (**#13/#14/#15**) shipped 2026-07-16 evening (§2). Open
backlog, ranked: **#7** (curl-able install.sh; absorbs remaining distribution work:
librarian release binaries, configure + post-install verify) → **#12** (dual-format
common-core fan-out — consumes `bun run package` as the seed of the production step) →
**#19** (PB-served webapp session — the deferred follow-on recorded in ADR 0001).

## 2. Last substantial session delivered (2026-07-16, evening)

Closed **#13, #14, #15** via two parallel worktree crews (foreman pattern: opus lead-driven
team for the coupled #13 chain, single opus builder for #14/#15; every diff independently
adversarially reviewed before landing — crew B 5/5 claims confirmed, crew A 7/7; all gates
re-run by the session). Commits `d4c52aa` (crew B) → `e851e5b` (crew A) → merge `f148e04`;
CI green on both pushes. Filed **#19** (deferred webapp follow-on).

- `d4c52aa` **#14 items 1–2 + #15** — `secrets_ref.llm_api_key` indirection implemented
  (plus `LLM_API_KEY_ENV` env override; per-provider vars remain the fallback, byte-identical
  when unset); idempotent superuser auto-create under `serve` (new `internal/bootstrap`;
  note: NOT on bare `migrate up` — no app-lifecycle hook there, recorded on #14); patrol now
  resolves open findings that stop firing (`state=resolved` + `resolved_run`, same
  transaction; scoped patrols resolve in-scope only; deleted-file findings resolve by
  design, test-pinned; migration `0010` down-func remaps resolved→flagged before stripping
  the enum — proven e2e against a real store).
- `e851e5b` **#13** — `pocket-librarian chat`: multi-turn desk-stewardship REPL over the
  eino loop (`internal/agent/session.go`, stubbed-provider tests, history cap 40, no new
  write path — tools only via the gated registry, `restore` never exposed, `apply_fix`
  gated at execution time too); trigger wake layer `internal/trigger` under `serve` (hourly
  cron patrol, files-create hook → scoped patrol, claimer finally consuming
  `ClaimerPollInterval`; transactional claim, panic-safe dispatch — a panicking task marks
  `failed`, serve survives); ADR **`docs/decisions/0001-interactive-surface-tui-first.md`**
  (§7.4 options evaluated; TUI now, PB-served webapp deferred → #19); spec §7.4 carries the
  supersession pointer; `verify.sh` 40 → 42 checks.
- Reviewer findings fixed pre-land: claimer panic recovery + chat history cap (crew A);
  0010 rollback data-remap + deleted-file resolution test (crew B). Known benign behavior:
  a bulk sweep enqueues one scoped patrol per new file (burst, not recursion — patrol never
  writes `files`).

## 2b. Earlier same day (2026-07-16)

Closed **#8** (brownfield-adoption skill, `eec43eb`, plugin 0.2.0 → 0.3.0: hardened K24
runbook, 9 phases, librarian baseline as final gate; field-test patrol evidence on the
issue) and **#9/#10/#11** + operator docs (`f90bb29`/`455769c`/`0e44935`/`5044405`):
directory-index README patrol exemption (greenfield template patrols at zero findings),
in-repo marketplace + self-contained plugin bundle (`bun run package`, drift-guarded),
issue/PR templates + live claude workflows, `librarian/README.md` operator sections
(LLM/keys, admin console via `make gui`, `mcp-serve` wiring). The discovery flow from those
sittings filed #13/#14/#15 — all now closed (§2).

## 3. Where to start building

Start with #7 (curl-able install.sh): absorb the remaining distribution work — librarian
release binaries, configure step, post-install verify — building on the proven marketplace
flow and `bun run package`. Then #12 (dual-format fan-out). #19 (webapp session) is the
deferred interactive follow-on; ADR 0001 records the preferred shape (custom Go route,
PB-served, no runtime frontend toolchain). Note the skill files under
`plugin/claude-plugin/skills/` are neutrality-lint-scanned (no bare issue refs, GitHub
URLs, or profile scalars in skill prose).

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

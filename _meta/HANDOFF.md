_Session-to-session bridge for desk-standard. Read this before working; update it at the end
of any substantial session. Secret-free — live URLs/credentials belong in `_meta/operations/`
(untracked) if that dir is ever created._
Status: active (2026-07-16)

# HANDOFF

## 0. Orientation

Two products over one shared schema, all identity-neutral (nothing shipped carries a
person/org/repo/issue): **`plugin/`** (harness-pure TS core + stdio MCP server, wrapped as a
Claude Code plugin with three skills) and **`librarian/`** (pocket-librarian: single Go
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

Open backlog, ranked (rationale posted as comments on each issue, 2026-07-16):
**#8** (brownfield-adoption skill — next) → **#13** (interactive librarian surfaces:
TUI/local webapp + the unbuilt trigger layer; fold **#14** config-drift fixes in) →
**#7** (curl-able install.sh; absorbs remaining distribution work: librarian release
binaries, configure + post-install verify) → **#12** (dual-format common-core fan-out —
last; consumes `bun run package` as the seed of the production step).

Before designing #8's migration steps: re-run the field-test patrol on the adopted
dev-tooling desk — 2 of its 9 recorded findings are resolved by the #11 rule fix.

## 2. Last substantial session delivered (2026-07-16)

Closed #9, #10, #11 and addressed the librarian-docs gap; four commits on `main`, CI green:

- `f90bb29` **#11** — patrol rules: a basename `README.md` inside an entity-mapped dir is a
  directory index, not an entity record (exempt from R1/R4/R5 via `isEntityDoc`/`r4Check`).
  Greenfield template now patrols at **zero findings** (proven e2e). Key discovery: there was
  no pre-existing exemption mechanism — `_meta/README.md` escapes only via `dir_kind=meta`;
  the conventions-standard skill's documented allowlist was unenforced prose, now reconciled.
- `455769c` **#9** — in-repo marketplace (`.claude-plugin/marketplace.json`) + self-contained
  plugin bundle: `bun run package` emits `plugin/claude-plugin/mcp/server.js` (core + npm deps
  inlined) + in-plugin schema copy the existing walk-up finds from the cache. Generated
  artifacts committed + CI drift-guarded. Plugin 0.1.0 → 0.2.0.
- `0e44935` **#10** — issue/PR templates + `claude.yml`/`claude-review.yml` (action
  v1.0.158, OAuth-preferred/API-key fallback; `CLAUDE_CODE_OAUTH_TOKEN` secret set by Henry,
  workflows live) + PocketBase dev skill tracked at `.claude/skills/pocketbase/`.
- `5044405` — `librarian/README.md` operator sections (LLM/provider selection, API keys,
  admin console via `make gui` → `127.0.0.1:8090/_/`, `mcp-serve` wiring snippet), written as
  actual current behavior. Filed **#13** (interactive agent surfaces — Henry's ruling: the
  librarian is an on-demand built-in agent, not CLI-only) and **#14** (declared-but-unread
  config: `secrets_ref.llm_api_key`, superuser auto-create, `ClaimerPollInterval`).

## 3. Where to start building

Start with #8 (its issue body carries the candidate flow + acceptance criteria; decision-0014
lineage in the spec §5 bounds the librarian's role). The skill ships inside
`plugin/claude-plugin/skills/` → must pass the neutrality lint. #13's design entry points:
spec §7.4 (three uncommitted surface options) and §1.3 (stewardship, not general chat —
Henry's ruling refines, doesn't discard, that boundary).

## 4. Conventions & gotchas

- **Gates** (run all before claiming done): `node scripts/check-neutrality.mjs` ·
  `cd plugin && bun run check:purity && bun test && bun run build` ·
  `cd librarian && go build ./... && go vet ./... && go test ./...` · `bash librarian/verify.sh`
  (40 checks) · `actionlint`. CI (`ci.yml`) is the aggregate required check.
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
  `agent`/MCP-driven calls (`LLM_PROVIDER` env → `profile.models` → anthropic;
  `ANTHROPIC_API_KEY`/`OPENAI_API_KEY`/`GEMINI_API_KEY` hardcoded per provider);
  `LIBRARIAN_AUTONOMOUS_WRITES=true` gates `apply_fix` over MCP; `restore` is CLI-only.
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

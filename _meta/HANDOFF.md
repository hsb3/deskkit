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
Root `README.md` is the front door; `_meta/build-brief.md` is the build brief this repo was
built from; `docs/pocket-librarian-v1-spec.md` is the librarian's spec (build spec, not
operator docs — operator docs are `librarian/README.md`).

Siblings: `hsb3/dotfiles-agents` (the pattern source for workflows/templates); Henry's
executive desk at `~/Documents/EXECUTIVE_DESK/Projects/dev-tooling-desk` holds off-repo
decision records (e.g. 0013–0016) referenced from the build brief — annotated, not vendored.

## 1. Current standing + top priority

v1 (Claude Code only) is built, distributable, and CI-green on `main` — a live plugin
marketplace: `claude plugin marketplace add hsb3/desk-standard` →
`claude plugin install desk-standard@desk-standard` (proven end-to-end).

**All rulings are in; everything open is now pure build work.** The field-eval findings
(#16/#17/#18) shipped earlier (PRs #21/#22, §2b), and **#20 closed 2026-07-17** — the
multi-desk design session was held and recorded as **ADR 0002** (commit `27a37a1`, §2).

**TOP PRIORITY — the release is PREPARED and waiting on one human command.** `make
release-prep` is green at **v0.4.0** (clean main, all gates rerun). To publish, run:
`git tag v0.4.0 && git push --tags` — the release workflow then asserts tag==VERSION, reruns
CI-equivalent gates (now including `verify.sh`, see §2), cross-compiles the four binaries, and
publishes a GitHub release + `checksums.txt`. This is the first *real* release (only a
`v0.0.1-alpha` pre-release exists). After it publishes, do the deferred live e2e of `install.sh`
(`curl | bash` against the real assets) — the one thing no PR could prove pre-release.

Open backlog, ranked (no ruling gates the buildable ones):
- **#12** — dual-format common-core fan-out (Claude + OpenCode instances; consumes
  `bun run package` as the seed). Architectural — likely needs an OpenCode-target ruling
  before fan-out; scope as its own phased effort, not a flat wave.
- **#34** — CI hardening: enforce shellcheck (incl. `install.sh`), actionlint, and a SHA-pin
  drift guard in CI (all exist locally/pre-commit but not in the pipeline). Follow-up from the
  release-prep audit.
- **#35** — test coverage: unit-test `requireConfig` self-init for non-`query` commands +
  a behavioral test for the TS MCP server. Follow-up from the release-prep audit.
- **#19** — PB-served webapp chat surface (deferred interactive follow-on; ADR 0001).

Shipped 2026-07-17: **#31** (PR #32, ADR 0003) + **docs/CI release-prep sweep** (PR #33), see
§2. Earlier same day: **#7 / #25 / #27** (PRs #29 / #30 / #28), **#23** (PR #24), **NOTE.md
punch list** (PR #26).

## 2. Last session delivered (2026-07-17, latest)

**#31 self-init ruling + docs/CI release-prep sweep, then release PREPARED.** Two threads:

- **#31 — store self-initialization** (PR #32, **ADR 0003**). Ruling taken: tool commands
  *self-initialize the store* (not merely translate the error). `requireConfig` now runs
  `app.RunAppMigrations()` idempotently before the desk-guard, so `sweep`/`query`/`patrol`/
  `chat`/etc. work on a fresh desk with no manual `migrate up`. Live-proven; `verify.sh` gained
  a self-init check (46→47). `migrate up` stays as the explicit path; ADR 0002 location
  fail-closed untouched. Rejected the translate-only option (keeps a papercut that serves no one).
- **Docs + tests sweep** (PR #33), driven by two read-only audits (docs-accuracy, test-coverage):
  - Docs: demoted `migrate up` from prerequisite → optional across all four user docs (stale
    after ADR 0003); dated **correction note on ADR 0001** (its "TUI" is a line-oriented REPL —
    `chat` = `bufio.Scanner` loop, zero TUI deps; "two commands" → one); de-pinned a stale
    `v0.4.0` install.sh example → `vX.Y.Z`.
  - Tests/CI: **wired `verify.sh` into `ci.yml` AND the `release.yml` gate** — the entire
    integration safety suite (self-init, record-original-first, byte-exact restore, open-guard)
    previously ran ONLY via a local `make verify`; the pipeline never exercised it (confirmed
    running green in CI on Linux). Activated 8 orphaned opencode tests (glob was `core mcp`,
    now `core mcp opencode`; `bun test` 37→45). Filed **#34** (CI hardening) + **#35** (coverage)
    as non-blocking follow-ups.
- **Release prepared:** `make release-prep` green at v0.4.0; awaiting `git tag v0.4.0 && git push
  --tags` (a human go/no-go, not auto-run — publishes a public release).

The `chat` interactive surface is a **line REPL, not a full-screen TUI** (ADR 0001 uses "TUI"
loosely; corrected in place). A real graphical/web surface is deferred → **#19**.

## 2-prev1. Distribution + hardening wave (2026-07-17)

**Distribution + hardening wave — #7, #25, #27** — built foreman-style as a flat fan-out: three
parallel worktree builders (disjoint file scopes), foreman adversarial verification + one
foreman-level correction, all three merged CI-green (`aca8b6e`/`fb039e7`→squash on main).
Local aggregate gates re-run on the integrated tree (version-sync, neutrality, shellcheck,
actionlint, full `go test`) — all green.

- **#27 — SHA-pin GitHub Actions** (PR #28). All 22 `uses:` refs across *four* workflows
  (`ci`, `release`, `claude`, `claude-review` — two more than expected) pinned to 40-char
  commit SHAs with `# vX` comments; annotated-tag refs (e.g. `claude-code-action@v1.0.158`)
  dereferenced to the underlying commit. Foreman independently re-resolved a SHA sample via
  `git ls-remote`. Landed FIRST so the release path is hardened before its first tag.
- **#7 — curl-able install.sh** (PR #29). Root `install.sh`: OS/arch detect → download the
  matching release binary + `checksums.txt` → sha256 verify (refuse on mismatch) → install to
  `~/.local/bin` (no root) → guide the marketplace plugin install; `--version`, `--prefix`,
  `--with-plugin`, `--dry-run`, `INSTALL_OS/ARCH` test hooks. Artifact names cross-checked
  line-by-line against `release.yml` (bare-version binary name, `sha256sum ./*` checksum
  format, `v<version>` tag). shellcheck clean; dry-run + error paths (exit 1) foreman-executed.
  `docs/getting-started.md` gained a *conditional* ("once a `v*` release is published")
  prebuilt-binary pointer. **Live e2e deferred to post-first-release** (nothing to download yet).
- **#25 — uninitialized-store message** (PR #30). `query` on a store whose collections were
  never created leaked PocketBase's bare `sql: no rows in result set`. Root cause: app
  migrations run only under `serve`/`migrate up`, never on a plain tool-command bootstrap.
  Fix translates `sql.ErrNoRows` (wrapped or bare) at the single `Query()` dispatch point
  (covers all 7 kinds + CLI/MCP/agent callers) into `store is not initialized — run
  \`librarian migrate up\` first`. **Foreman correction:** the builder first advised `sweep`;
  changed to `migrate up` after confirming against the documented first-run flow AND that
  `sweep` hits the same wall on a virgin store (sweep does not create collections). The
  broader leak from `sweep`/`patrol`/etc. is filed as **#31**.

## 2-prev0. NOTE.md punch list (2026-07-17)

**Henry's NOTE.md punch list** (user docs / tests visibility / VERSION / precommit / release
flow / Makefile / docs-vs-_meta) — built foreman-style (3 parallel builders + adversarial
reviewer + CI-review hardening round), merged as **PR #26** (squash `84d3b6e`). Rulings taken
with Henry: build-brief + m-05 → `_meta/` (spec + ADRs STAY in docs/); ONE repo version
(0.4.0, was plugin 0.3.0 / package.json 0.1.0 drift); guides + VHS recordings. Delivered:

- **Root `Makefile`** — canonical interface (`make help`): build/test/check/verify/package/
  media/setup/clean/release-prep. Use `make check` + `make test` + `make verify` as the gate
  suite now.
- **`VERSION` = 0.4.0** drives plugin.json/package.json/marketplace.json (drift-guarded by
  `scripts/check-version-sync.mjs` in CI + pre-commit) and the librarian binary via ldflags
  (`make`-built prints 0.4.0; bare `go build` prints `dev` — RootCmd.Version override in
  main.go, PocketBase's own Version var isn't ours to -X).
- **lefthook.yml** — fast pre-commit mirror (neutrality+self-test, version-sync; actionlint/
  purity path-scoped). `make setup` installs.
- **`.github/workflows/release.yml`** — tag `v*` → tag==VERSION gate + CI-equivalent checks →
  darwin/linux × amd64/arm64 pure-Go binaries (CGO_ENABLED=0; linux/arm64 cross-compile
  proven) → plugin bundle → gh release + sha256 checksums. Empty-dist guard; least-privilege
  permissions. THE SUBSTRATE FOR #7.
- **Three user guides** (docs/getting-started, plugin-guide, librarian-guide) in
  value-and-proof style — every transcript really run; reviewer independently reproduced the
  librarian guide's whole chain including byte-exact restore checksums. Four VHS
  tapes + GIFs (648K) in docs/media/, regenerable via `make media` (hermetic script).
- Review rounds caught: a stale `--version` claim in getting-started (contradicted the new
  stamp — fixed), a VHS tape teaching that CLI apply-fix needs LIBRARIAN_AUTONOMOUS_WRITES
  (it doesn't — env gates MCP/agent only; tape re-recorded), + 3 hardening nits (fixed).
  SHA-pinning actions deferred to **#27**.

## 2-prev. Earlier same day (2026-07-17, later still)

**#23** — ADR 0002 implementation, built foreman-style (scout → 2 builders + adversarial
reviewer → CI-review fix round), merged as **PR #24** (squash `b723d31`; CI + claude-review
green twice; verify.sh 42→46 checks). What shipped:

- **XDG store home**: no `--dir` → `$XDG_DATA_HOME/pocket-librarian/<DESK_NAME>/` (fallback
  `~/.local/share/…`, empty XDG = unset), pre-created `0700`. `config.Load()` now runs BEFORE
  app construction (PocketBase parses `--dir` eagerly inside `NewWithConfig`).
- **Fail-closed location**: unresolvable DESK_NAME + no `--dir` → exit 1 for every
  store-touching command (incl. serve/migrate), enforced by a pre-`Start()` argv scan in
  `main()` — cobra hooks are TOO LATE (PocketBase `Bootstrap()` creates the data dir before
  any RunE/PreRunE fires). The old "serve/migrate run config-free" tolerance is narrowed:
  store LOCATION must now resolve.
- **Desk open-guard** (`internal/bootstrap/deskguard.go`): mismatched `desk` rows
  (files → patrol_log → adoption_log) refuse with both names; empty store passes; `migrate`
  exempt. Sites: `requireConfig` + the OnServe hook.
- **gui** forwards the resolved `--dir` to its serve child unconditionally and aborts before
  opening a browser on any requireConfig error.
- verify.sh is hermetic (scratch `XDG_DATA_HOME` exported for the whole run) — never touches
  a real store.

Review cycle caught 5 real defects post-build (all fixed + live-proven): serve's open-guard
printed the refusal but exited 0 (PocketBase runs the command goroutine and DISCARDS RunE
errors for serve/superuser — they're registered inside `Start()`, after the cmdErr wrapper;
fail-closed there needs a direct `os.Exit(1)`); gui didn't forward `--dir` to its child
(would serve a different store than the browser targeted); `--hooksDir x serve` bypassed the
location guard via the argv scan's value-flag list (whitelist extended: hooksDir/hooksWatch/
hooksPool; residual known limitation — it's an enumerated whitelist by construction, noted
in code); a verify.sh scratch-dir leak on SIGINT; a missing adoption_log fallthrough test.

## 2a-bis. Earlier same day (2026-07-17, later)

**#20** — multi-desk design session held with Henry; four rulings, all recorded in
`docs/decisions/0002-multi-desk-topology-store-per-desk.md` (commit `27a37a1`, CI-green;
issue closed with the ruling comment, implementation split to **#23**):

1. **Store-per-desk** — portfolio view is read-only fan-out, never a shared write store.
2. **Canonical store home = XDG data home** (`$XDG_DATA_HOME/pocket-librarian/<DESK_NAME>/`,
   fallback `~/.local/share/…`) when `--dir` is absent; `--dir` stays the override. Stores
   live outside the iCloud-synced desk tree.
3. **`desk` field kept + promoted to open-guard** (refuse a store whose rows carry a different
   desk name). No composite `(desk, path)` key. Fact correction found during the session: the
   field was never latent — sweep/patrol/apply_fix already populate it with `DESK_NAME`.
4. **MCP stays one desk per process.**

Operational notes (version skew self-heals via per-store automigrate; patrol needs no
cross-desk coordination; backup/restore is per-store) are in the ADR, no code needed.

## 2a. Earlier same day (2026-07-17)

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

The distribution *script* (#7) is done; the natural next step is to **cut the first release**
(bump `VERSION` → `make release-prep` → follow its tag instructions) so `install.sh`'s live
`curl | bash` path is finally exercised — the one thing this wave could not prove because no
release exists yet. After that, two tracks, each needing a ruling before build: **#12**
(dual-format fan-out — decide the OpenCode-target shape) and **#31** (uninitialized-store
error — per-command translation vs. self-initializing auto-migrate). **#19** (webapp) is the
deferred interactive follow-on; ADR 0001 records the preferred shape (custom Go route,
PB-served, no runtime frontend toolchain). Note the skill files under
`plugin/claude-plugin/skills/` are neutrality-lint-scanned (no bare issue refs, GitHub URLs,
or profile scalars in skill prose).

## 4. Conventions & gotchas

- **Gates** (run all before claiming done — via the root Makefile since PR #26):
  `make check` (neutrality + self-test + purity + actionlint) · `make test` (bun 45 + go) ·
  `make verify` (verify.sh, 47 checks — **now also runs in CI + the release gate**, PR #33) ·
  `make package` (drift guard) · `node scripts/check-version-sync.mjs`. CI (`ci.yml`) is the
  aggregate required check. Note: shellcheck + actionlint are NOT yet CI-enforced (→ #34).
  Bumping any version = edit root `VERSION` + the three manifests (sync-guarded).
- **Generated, never hand-edit**: `plugin/claude-plugin/mcp/server.js` and
  `plugin/claude-plugin/schema/profile.schema.yaml` — regen with `cd plugin && bun run package`;
  CI drift-guards them. Bundle output is byte-identical across macOS/linux (proven).
- **Neutrality lint scope** = `plugin/` + `librarian/` recursively. Bare issue refs (`#11`)
  in Go comments/tests inside `librarian/` FAIL the lint — write issue-free comments.
  `.claude-plugin/marketplace.json` (owner identity) is deliberately outside the surface
  (recorded in `_meta/m-05-data-surfaces.md`).
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
- **PocketBase bootstrap runs before cobra**: `Execute()` calls `Bootstrap()` (which CREATES
  the data dir) before `RootCmd.Execute()` dispatches any RunE/PreRunE — anything that must
  prevent store creation has to run in `main()` before `app.Start()` (the argv-scan location
  guard does). And PocketBase registers `serve`/`superuser` INSIDE `Start()` then discards
  their RunE errors (goroutine) — fail-closed behavior in serve paths needs a direct
  `os.Exit(1)` (see the OnServe desk-guard), not a returned error.
- **Store location** (since PR #24): no `--dir` → `$XDG_DATA_HOME/pocket-librarian/
  <DESK_NAME>/`; unresolvable DESK_NAME + no `--dir` → exit 1 (serve/migrate included).
  verify.sh exports a scratch XDG_DATA_HOME — keep it hermetic when adding checks.
- **Store self-initializes** (ADR 0003, since PR #32): tool commands run `RunAppMigrations()`
  at the `requireConfig` choke point, so a fresh store needs no manual `migrate up`. When adding
  a new store-touching command routed through `requireConfig`, it inherits this automatically.
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

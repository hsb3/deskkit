_Session-to-session bridge for desk-standard. Read this before working; update it at the end
of any substantial session. Secret-free — live URLs/credentials belong in `_meta/operations/`
(untracked) if that dir is ever created._
Status: active (2026-07-19)

# HANDOFF

## 0. Orientation

Two products over one shared schema, all identity-neutral (nothing shipped carries a
person/org/repo/issue): **`plugin/`** (harness-pure TS core + stdio MCP server, wrapped as a
Claude Code plugin with four skills) and **`librarian/`** (pocket-librarian: single Go
binary, embedded PocketBase, eino agent loop, record-original-first write boundary), with
**`schema/`** as the contract both read. Personalization is only via `_knowledge/profile.yaml`.
Root `README.md` is the front door; `docs/README.md` indexes the docs (Using ┃ Development);
`_meta/build-brief.md` is the build brief this repo was built from; `docs/pocket-librarian-v1-spec.md`
is the librarian's spec (build spec, not operator docs — operator docs are `librarian/README.md`).

Siblings: `hsb3/dotfiles-agents` (the pattern source for workflows/templates); Henry's
executive desk at `~/Documents/EXECUTIVE_DESK/Projects/dev-tooling-desk` holds off-repo
decision records (e.g. 0013–0016) referenced from the build brief — annotated, not vendored.

## 1. Current standing + top priority

**v0.5.0 is released and CI-green on `main`.** Both products ship at 0.5.0 off the single root
`VERSION`: the plugin (live marketplace — `claude plugin marketplace add hsb3/desk-standard` →
`claude plugin install desk-standard@desk-standard`, proven) and the `pocket-librarian` binary
(release assets verified — downloads, runs, sha256 matches `checksums.txt`). All product rulings
are in; open work is pure build. Cutting the next release just follows
`docs/development/releasing.md` (§3, §4).

**Active blocker — the public `curl|bash` install path is gated on repo visibility.** The repo is
PRIVATE, so every unauthenticated URL `install.sh` needs (`raw…/install.sh`, `/releases/latest`,
`/releases/download/…`) 404s by design. install.sh's URL + checksum logic are proven correct
(authed assets verify); the only missing link is the unauthenticated fetch. Fix = make the repo
public (a deliberate launch decision, not autonomous) or add token/`gh` auth to install.sh.

**ON HOLD (Henry, 2026-07-17): going public AND OpenCode (#12) are parked** until Henry is
satisfied with the current build in practice — the focus is dogfooding/stabilizing shipped
Claude-Code, not opening or extending. Don't action either without his go-ahead. Buildable
follow-ups (#34, #35) can proceed.

**Small open follow-ons:** two accepted PR #40 review nits — the locked-store hint's substring
match could false-positive on a path containing "locked" (harmless; the hint is phrased as a
question), and the chat tapes' 30s answer sleep is tight on slow API days (only matters on
manual re-record; committed GIFs verified good). One accepted PR #48 nit: `ctrl+y` copies an
interrupted turn's partial text (judged correct — it's real answer text). The `[Unreleased]`
CHANGELOG now holds `make install` + the docs split + the whole chat-TUI pass; `make
version-status` advises a bump — cut the next release when Henry's ready.

**MERGED 2026-07-18: PR #48** (chat-TUI pass — #44 theming + #45 UX + #43 init/onramp +
Crush-style app chrome, look approved live) **and PR #54** (`record_feedback`, #50). Chat
GIFs re-recorded with the final chrome post-merge. Deferred UX items are recorded in
`docs/development/chat-tui-ux-survey.md`.

**RULED 2026-07-18 — all five TUI roadmap recommendations accepted verbatim** (memo, now
marked decided: `_meta/briefings/2026-07-18-tui-roadmap-rulings/README.md`; rulings also
commented on #51/#52/#53): #53 Charm-v2 migration GO with terminal background retained +
no-query rule kept pre-program-only (**ADR 0007**, committed on `feat/tui-charm-v2` @
57f1a76); #51 sessions-list-first launch + truncation titles + rename/delete v1; #52 defer
dollar-cost, add the `models.context_window` profile override key (schema change).

**DONE 2026-07-19 — the #53 v2 migration is PR #59: green, mergeable, awaiting Henry.**
Branch `feat/tui-charm-v2`: migration verified (build/vet/full tests, no-v1-imports grep,
make check/test), rebased onto the moved main, ci + claude-review pass, all four review
threads fixed (c17c802) and replied. **ADR renumbered 0006 → 0007** (PR #57, landed from a
parallel session, took 0006 for the kit-port ADR). Live parity proven in a REAL terminal
(tmux send-keys/capture-pane, full ready→streaming→ready cycle byte-clean) — that proof, not
re-recorded GIFs, is the PR's parity artifact.

**SEQUENCING DECISION IN HENRY'S COURT — PR #59 vs PR #60.** A parallel workstream (epic
#55) landed #56/#57 on main 2026-07-18 evening and opened **PR #60** (`refactor/core-modules`,
92 files) which MOVES `librarian/internal/tui/` → `librarian/internal/modules/librarian/tui/`.
Direct structural collision. Recommendation (commented on both PRs): merge **#59 first**;
#60 absorbs the v2 import swap in its rebase (its move rewrites those import paths anyway).
If #60 goes first instead, #59 must be re-targeted onto the moved paths and re-verified.

**#58 — chat-GIF re-record blocked by an xterm.js canvas-renderer bug (investigated,
disposition open).** Re-recording chat tapes on v2 deterministically shows a one-cell stale
glyph ("rready") — VHS-environment-only. Full evidence on the issue: our emitted bytes are
provably correct (raw pty capture replays clean through the exact xterm-headless version
ttyd bundles); VHS hardcodes `-t rendererType=canvas`, whose stale-glyph bug class is
documented upstream (xtermjs #3548/#3617); no reachable flag fixes it (dom renderer 4–9×
slower, never completed; customGlyphs/sixel flags don't help). Committed GIFs deliberately
stay v1-era. Disposition options ranked on #58 (accept / product-side full-line-repaint
dodge / toolchain pinning dig).

**Then:** #52's session-layer token plumbing (stack-independent) and #51 on v2 (both build
on whichever tree layout wins the #59/#60 sequencing).

**Release note:** `[Unreleased]` now holds the whole TUI pass + `record_feedback` + `make
install` — a solid 0.6.0. Cut per `docs/development/releasing.md` when Henry wants it.

**Open backlog, ranked** (no ruling gates the buildable ones):
- **#12** — dual-format Claude+OpenCode fan-out (consumes `bun run package` as the seed).
  Architectural; needs an OpenCode-target ruling before fan-out. **ON HOLD (above).**
- **#34** — CI hardening: enforce shellcheck (incl. `install.sh`) + actionlint + a SHA-pin drift
  guard in CI (all exist locally/pre-commit, not yet in the pipeline).
- **#35** — coverage: unit-test `requireConfig` self-init for non-`query` commands + a behavioral
  test for the TS MCP server.
- **#19** — PB-served webapp chat surface (deferred; ADR 0001 option b — substrate ready, see §3).
- **#36** — Template SOP library (v-next **major**): grow the 2 fixer templates to a full ~20-type
  SOP library from the headcase shared set (`_headcase/shared/sops/`). Needs a design ruling first
  (vendor-vs-sync, type↔dir_kind, planner template-selection, neutrality of vendored examples).
  Corrects the "librarian knows ~20 SOPs" premise — that catalog never existed; the 2-template
  boundary was intentional. A status-against-roadmap briefing deck lives at
  `_meta/briefings/2026-07-17-status-against-roadmap/`.

## 2. Recent deliveries (newest first — full detail in the cited PRs / ADRs / git)

- **2026-07-18 — record_feedback** (PR #54, open; issue #50): `feedback` collection
  (migration 0013), tool on all surfaces + CLI subcommand, `query feedback` kind,
  system-prompt nudge. Built in a worktree off main, independent of the TUI branch.
- **2026-07-18 — chat-TUI UX pass** (PR #48, open): #44 per-theme palettes resolved once
  pre-program (`--theme`/`LIBRARIAN_THEME`/auto-probe) + #45 survey
  (`docs/development/chat-tui-ux-survey.md`) and apply (gutters, per-turn `model · latency`
  footer, bubbles/help + ctrl+g, ctrl+y copy-raw-markdown, edge-row prompt history with draft
  stash, scroll-anchored streaming, NO_COLOR reduced motion) + #43 `pocket-librarian init` +
  interactive first-run onramp (`--no-input` keeps fail-closed) + Crush-style app chrome
  (full-width header/status bars, rounded input box that dims while streaming, user turns as
  `▌`-bordered tinted blocks, 120-col measure — `themeSurfaces` in styles.go). New
  `chat-light.tape`/GIF (light proof) + `init-onramp.tape`/GIF (offline onramp proof).
  Review hardening: lipgloss background cache pinned to the resolved theme in `tui.Run` (§4).
- **2026-07-18 — profile-first docs onramp** (PR #46). Fixed a getting-started
  self-inconsistency: §4 told non-devs to `export DESK_ROOT`/`DESK_NAME` even though step 2
  already fills `_knowledge/profile.yaml`, which the librarian auto-discovers by walk-up
  (`internal/config/config.go` `Load`→`DiscoverProfile`; a profile with `desk.name` +
  `root: "."` needs zero exports from inside the desk). Reframed env as the *override* (bare
  folder / dev build from `librarian/`), added a `chat` pointer. Mirrored in `librarian/README`
  + root README. Docs-only. Grew out of a UAT dogfooding session that also filed #43/#44/#45.
- **2026-07-18 — make install + docs dev/use split** (PR #42). `make install` (root →
  `librarian/Makefile`) builds the version-stamped binary into `~/.local/bin` (override `PREFIX=`);
  prompted by a stale on-PATH binary — `/plugin` updates the plugin, NOT the standalone binary
  (separate artifacts). Docs split Using vs Development: new `docs/README.md` index + `docs/development/`
  (overview, `releasing.md`, VHS `tapes/`); spec + ADRs kept at `docs/` top level for citation
  stability (§4 "Docs layout").
- **2026-07-18 — v0.5.0 release + versioning/changelog guard** (PR #41, **ADR 0005**). Bumped
  0.4.0→0.5.0, added `CHANGELOG.md`, and a two-tier missing-bump guard: `check-changelog.mjs`
  (hard release gate) + `check-version-status.mjs` (non-blocking drift advisory) — §4
  "Versioning/release". Tagged + released; assets verified.
- **2026-07-18 — chat full-screen TUI, all 5 phases** (PRs #37–#40, **ADR 0004**). Bubble Tea TUI
  with streaming, tool steps, markdown, cancel, resume (ctrl+o); REPL fallback when piped/`--plain`.
  Streaming substrate (`Session.StreamTurn`, JSON-taggable `Event`) is reusable for the deferred
  webapp SSE route. eino/bubbletea streaming gotchas → §4.
- **2026-07-17 — #31 store self-init** (PR #32, **ADR 0003**) + **docs/CI release-prep sweep**
  (PR #33): tool commands auto-run migrations at `requireConfig`; `verify.sh` wired into CI + the
  release gate (§4 "Store self-initializes").
- **2026-07-17 — distribution + hardening** #7/#25/#27 (PRs #29/#30/#28): curl-able `install.sh`,
  SHA-pinned Actions, actionable uninitialized-store message. **First release v0.4.0 cut** (§5).
- **2026-07-17 — NOTE.md punch list** (PR #26): root `Makefile`, unified `VERSION`, `lefthook.yml`,
  `release.yml`, three user guides + VHS media.
- **2026-07-17 — #23 XDG store home + desk open-guard** (PR #24, **ADR 0002**, from the #20 design
  session): store defaults to `$XDG_DATA_HOME/pocket-librarian/<DESK_NAME>/`; a mismatched-desk
  store refuses. PocketBase bootstrap/serve-RunE gotchas → §4.
- **2026-07-17 — #18 field-UX batch** (PR #22): `infra` dir_kind, R4→judgment, mcp-serve clean EOF,
  `query --pretty`. **#16/#17** (PR #21): content-field widen (migration 0011) + prompt seed moved
  into `requireConfig`.
- **2026-07-16 — #13/#14/#15** (**ADR 0001**): chat REPL + trigger wake layer, `secrets_ref`
  indirection, serve superuser auto-create, patrol stale-finding resolution (migration 0010).
  **#8–#11**: brownfield-adoption skill, in-repo marketplace + `bun run package` bundle, operator docs.

## 3. Where to start next

- **PR #48 is in Henry's court** (chat-TUI UX pass). After merge, the branch's deferred UX
  items live in `docs/development/chat-tui-ux-survey.md` if he wants a follow-up pass.
- **Cut the next release** when `[Unreleased]` warrants — follow `docs/development/releasing.md`
  (bump VERSION + 3 manifests → roll `[Unreleased]` into a dated CHANGELOG section →
  `make release-prep` → tag). `check-changelog` gates the tag; `make version-status` flags drift.
- **Repo-visibility decision** (§1) gates the public `curl|bash` path — Henry's call, on hold.
- **#19 webapp** (deferred; ADR 0001 option b): substrate is ready — `StreamTurn`'s JSON-taggable
  events (ADR 0004) marshal straight onto a PB-served SSE route (no runtime frontend toolchain).
- Any new streaming/TUI work: gates per §4; live proof via VHS (tapes in `docs/development/tapes/`,
  method in the §4 eino/bubbletea note).

## 4. Conventions & gotchas

- **Gates** (run all before claiming done — via the root Makefile since PR #26):
  `make check` (neutrality + self-test + purity + actionlint) · `make test` (bun 45 + go) ·
  `make verify` (verify.sh, 47 checks — **now also runs in CI + the release gate**, PR #33) ·
  `make package` (drift guard) · `node scripts/check-version-sync.mjs`. CI (`ci.yml`) is the
  aggregate required check. Note: shellcheck + actionlint are NOT yet CI-enforced (→ #34).
- **Versioning/release (ADR 0005, since PR #41 — runbook `docs/development/releasing.md`)**: SemVer on the
  single root `VERSION`; a bump = edit `VERSION` + the three manifests (sync-guarded) **and** move
  `[Unreleased]` CHANGELOG entries into a dated `## [<version>]` section. `check-changelog.mjs` is
  a **hard gate** (release-prep + release.yml) — a tag with no CHANGELOG section fails.
  `check-version-status.mjs` is a **non-blocking advisory** (`make version-status` + a CI step,
  always exit 0) that warns when `plugin/`/`librarian/` drifted past the last tag without a bump.
  `ci.yml` checks out `fetch-depth: 0` for it. Both scripts + CHANGELOG live outside the neutrality
  surface (repo root / `scripts/`). `make install` builds + drops the binary in `~/.local/bin`
  (override `PREFIX=`); `/plugin` updates the plugin only — the binary is a separate artifact.
- **Docs layout (since the dev/use split)**: `docs/` is indexed by `docs/README.md` in two
  tracks — **Using** (`getting-started`/`plugin-guide`/`librarian-guide` + `media/*.gif`) and
  **Development** (`docs/development/` = overview README + `releasing.md` + `tapes/*.tape`). The
  **spec** (`docs/pocket-librarian-v1-spec.md`) and **ADRs** (`docs/decisions/`) deliberately stay
  at the `docs/` top level — they're cited from code, skills, and the neutrality-lint allowlist, so
  their paths are kept stable (don't move them to `development/`). VHS **tape sources** live in
  `docs/development/tapes/`; their **`.gif` output** lands in `docs/media/` (the `Output` path is
  repo-root-relative, so tapes moved without touching it).
- **Generated, never hand-edit**: `plugin/claude-plugin/mcp/server.js` and
  `plugin/claude-plugin/schema/profile.schema.yaml` — regen with `cd plugin && bun run package`;
  CI drift-guards them. Bundle output is byte-identical across macOS/linux (proven).
- **Neutrality lint scope** = `plugin/` + `librarian/` recursively. Bare issue refs (`#11`)
  in Go comments/tests inside `librarian/` FAIL the lint — write issue-free comments. `docs/` is
  exempt. `.claude-plugin/marketplace.json` (owner identity) is deliberately outside the surface
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
  `os.Exit(1)` (see the OnServe desk-guard), not a returned error. Worked example since
  PR #48: `init` executes standalone in `main()` BEFORE the app exists (and stays out of
  `storeTouchingCommands`) precisely so Bootstrap can't create a stray store dir.
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
- **eino streaming gotchas** (bind any new streaming caller to these — origin: PRs #37–#38 / ADR 0004):
  agent output stream is bursty under the anthropic tool-call checker (use model-callback
  stream copies for live tokens); ctx-cancel doesn't abort a stuck provider stream; failed
  tools fire OnError only; zero-arg tool calls need the `argNormalizingTool` adapter (in
  place at buildTools — keep new tools behind it). **No terminal queries after bubbletea
  starts** (no glamour WithAutoStyle / lazy lipgloss adaptive colors) — responses leak into
  the textarea; regression-guarded in `internal/tui/defects_test.go`. Since PR #48 the chat
  theme resolves ONCE pre-program (`tui.ResolveTheme`) and `tui.Run` pins
  `lipgloss.SetHasDarkBackground` to it — load-bearing for embedded bubbles components whose
  DEFAULT styles use AdaptiveColor (the textarea): without the pin they're only safe via a
  bubbletea v1 init workaround that v2 removes. New TUI colors go in `newStyles`'s per-theme
  switch, never AdaptiveColor; new renderers take `(width, theme)`.
- **VHS chat tapes** (`chat.tape` dark + `chat-light.tape` light) both need
  `ANTHROPIC_API_KEY` — record with `ANTHROPIC_API_KEY="$(secret get ANTHROPIC_API_KEY)"
  bash scripts/record-media.sh`. Re-recording re-encodes ALL GIFs; `git restore` the
  non-chat ones when their tapes didn't change (byte churn, no content).
- **Worktree provisioning**: untracked-and-load-bearing = `plugin/node_modules` (run
  `bun install --frozen-lockfile`) and `.claude/agent-memory/` (machine-local). Everything
  else a worktree agent needs is committed.
- `_meta/` holds `build-brief.md`, `m-05-data-surfaces.md`, `briefings/`, and this handoff — the
  full taxonomy (operations/ ignore stanza etc.) comes via the mise-en-place scaffold when needed.

## 5. Incident log

- 2026-07-17: first real release (v0.4.0) cut successfully, but the live `install.sh` e2e
  surfaced that **the repo is private** — every unauthenticated URL the public install flow needs
  (`raw…/install.sh`, `/releases/latest`, `/releases/download/…`) returns 404. Initially looked
  like CDN propagation lag (persisted >6 min); root-caused by checking `gh repo view` visibility.
  Authed `gh release download` proved the assets are valid (binary runs, sha256 matches
  checksums.txt), so install.sh is correct — the blocker is purely repo visibility. Resolution:
  make the repo public when launching (§1), or add token auth to install.sh if it stays private.
  Lesson: a persistent (not transient) 404 on release assets that the authed API can see = check
  repo visibility before blaming propagation.
- 2026-07-18: the `pocket-librarian` on PATH was a stale build (no `chat`, `--version` printed
  `(untracked)`) — `/plugin` updates the plugin, NOT the standalone binary (separate artifacts).
  Fixed by installing the checksum-verified v0.5.0 release binary + adding `make install`. Lesson:
  after a release, update the binary separately (`make install` or `gh release download`).
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

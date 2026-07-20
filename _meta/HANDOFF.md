_Session-to-session bridge for desk-standard. Read this before working; update it at the end
of any substantial session. Secret-free — live URLs/credentials belong in `_meta/operations/`
(untracked) if that dir is ever created._
Status: active (2026-07-20)

# HANDOFF

## 0. Orientation

Two products over one shared schema, all identity-neutral (nothing shipped carries a
person/org/repo/issue): **`plugin/`** (harness-pure TS core + stdio MCP server, wrapped as a
Claude Code plugin with four skills) and **`librarian/`** (pocket-librarian: single Go
binary, embedded PocketBase, eino agent loop, record-original-first write boundary), with
**`schema/`** as the contract both read. Personalization is only via `_knowledge/profile.yaml`.
Root `README.md` is the front door; `CLAUDE.md` is the agent digest and `docs/CHARTER.md` the
canonical page (precedence rule — both since PR #105); `docs/README.md` indexes the docs (Using ┃
Development); `docs/pocket-librarian-v1-spec.md` is the librarian's spec (build spec, not operator
docs — operator docs are `librarian/README.md`). The build brief moved to the exec desk in PR #77.

Siblings: `hsb3/dotfiles-agents` (the pattern source for workflows/templates). The **paired
executive desk** is `~/Documents/EXECUTIVE_DESK/Projects/desk-standard-desk` (dedicated to this
product since the 2026-07-19 split): the 1.0.0 roadmap/design-review analyses, plans, briefings,
staged issue bodies, and the greenfield friction ledger live there. The off-repo decision
records the build brief cites (0013/0014/0015 — not 0016, which is dotfiles-agents — plus 0021,
the 1.0.0 direction) remain in `dev-tooling-desk`'s append-only spine, indexed from the new
desk's genesis record 0001.

## 1. Current standing + top priority

**v0.7.0 is released** (the PM system, ship-dark; adoption dry-run; PM spec/README reconcile)
and the CLI binary is **renamed `pocket-librarian` → `deskkit`** — release assets are
`deskkit_0.7.0_*`; `make install` drops `deskkit` into `~/.local/bin`. The stale
`pocket-librarian` 0.6.0 on PATH was removed 2026-07-19 (the 07-18 incident class — don't
resurrect it). **Lane 6 shipped** (PR #105 → `217b5e6`, epic #86 closed): repo-compliance-audit
**70 pass / 0 gap** (was 55/14); `CLAUDE.md` + `AGENTS.md` (symlink) + `docs/CHARTER.md`
authored; root `tests/` ruled a declared-home index (recorded in `_meta/mise-en-place.yml`).
**The deep-dive backlog is filed and triaged**: #91–#104 with milestones set (0.8.0 = bug floor
#67/#78/#79/#80 + #91/#92/#93/#102, #94 stretch — plan at the exec desk's
`_meta/plans/release-0.8.0.md`; the rest 1.0.0; onboarding epic #104 post-1.0); the #79
widening comment is posted; #81 rides the 1.0.0 milestone (R20 ruling). Cutting the next
release follows `docs/development/releasing.md`.

**Public launch is deferred until ≥ v1.0.0 (Henry, 2026-07-19) — this is settled, not an open
blocker.** The repo stays PRIVATE until then, so the public `curl|bash` path is expected to 404
by design (every unauthenticated URL `install.sh` needs — `raw…/install.sh`, `/releases/latest`,
`/releases/download/…`). install.sh's URL + checksum logic are already proven correct (authed
assets verify); the only missing link is the unauthenticated fetch, which will resolve for free
when the repo goes public at v1.0.0. **Until then, install from the private repo with `gh` auth**
(see §4 "Installing from the private repo"). Do NOT keep surfacing the private-repo 404 as a
blocker — it's a planned gate, not a bug.

**ON HOLD (Henry, 2026-07-17): going public AND OpenCode (#12) are parked** — public until
≥ v1.0.0 (above); the focus is dogfooding/stabilizing shipped Claude-Code, not opening or
extending. Don't action either without his go-ahead. Buildable follow-ups (#34, #35) can proceed.

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

**MERGED 2026-07-19 — #59 (v2 migration) then #60 (core+modules refactor), in that order.**
The #59-first sequencing was executed as recommended: #59 squash-merged clean (ci +
claude-review green), then #60 rebased onto the new main. The rebase's rename detection paired
#59's v2-modified tui files with #60's move correctly — v2 code landed at the new module path
`librarian/internal/modules/librarian/tui/`, zero v1 imports, old `internal/tui/` gone. **One
drift caught by `go vet`:** #59's new regression-guard test `defects_test.go` still imported the
pre-move `internal/config`; git took #59's copy so #60's path rewrite missed it. Fixed to
`internal/core/config` (config moved under `internal/core/` in #60), amended into the rebased
commit. Full gates green post-fix (check/test/verify-47/package/version-sync) and on merged
`main` (build+vet). **ADR renumbered 0006 → 0007** (PR #57 took 0006 for the kit-port ADR).
Live v2 parity was proven in a REAL terminal (tmux capture, byte-clean) — the PR's parity
artifact, not re-recorded GIFs (see #58). Current tree layout is the modules/ layout: any
new TUI work (#51/#52) targets `librarian/internal/modules/librarian/tui/`.

**#58 — CLOSED 2026-07-19 as won't-fix (upstream, cosmetic, docs unaffected).** One-more-pass
determination: re-recording was never actually necessary — ADR 0007 mandates the v2 migration
ship *zero visual change*, so the committed v1-era GIFs depict the v2 product accurately;
nothing in the docs is stale. The glitch (one-cell stale footer glyph "rready") is
VHS-environment-only — our emitted bytes replay clean through ttyd's exact xterm-headless
version; VHS hardcodes `-t rendererType=canvas`, whose stale-glyph bug (xtermjs #3548/#3617,
fixes in #4189/#4101) is newer than ttyd 1.7.7 bundles, with no newer ttyd stable to pin.
Product-side footer-repaint dodge rejected (global perf workaround defeating v2's cell-diff
renderer to satisfy a broken recording tool; real terminals clean). Auto-resolves for free on
a future VHS/ttyd xterm.js bump — reopen only if a real visual redesign needs fresh captures
before the toolchain catches up. Full rationale in the #58 close comment.

**Then:** #52's session-layer token plumbing (stack-independent) and #51 on v2 — both build on
the modules/ layout at `librarian/internal/modules/librarian/tui/`.

**Release note:** the TUI pass + `record_feedback` + `make install` + the SOP kit library +
the Charm v2 migration + the core+modules refactor all shipped in **v0.6.0** (tagged, released,
assets verified). `[Unreleased]` is now empty; the next bump starts accumulating from here.

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

- **2026-07-20 — Lane 6 conformance + exec-desk split + deep-dive triage.** PR #105 (squash
  `217b5e6`, closes #86): mise-en-place scaffold (`.claude/` skeleton, `.mcp.json`, dependabot
  npm@`/plugin` + gomod@`/librarian`, ADR 0000-template) + the three authored entry docs +
  the tests/ ruling; audit 70/0; docs adversarially reviewed 10/10; the dependabot directory
  bug was caught in PR review and fixed pre-merge. Off-repo, same session: the dedicated exec
  desk `desk-standard-desk` bootstrapped greenfield (friction ledger INS-01..10 — including the
  0.5.0-plugin vs 0.7.0-binary version skew and the `template_render` cwd walk-up gap), the
  content lift out of dev-tooling-desk completed, issues #91–#104 filed, 0.8.0 milestone created.
- **2026-07-19 — v0.6.0 release** (PR #61, tag `v0.6.0`). Minor bump 0.5.0→0.6.0: VERSION +
  3 manifests + `[Unreleased]` CHANGELOG rolled into a dated `[0.6.0]` section (SOP kit library,
  `make install`, `record_feedback`, chat-TUI UX pass, `init` onramp, Charm v2 migration,
  core+modules refactor). PR CI + claude-review green, squash-merged; `make release-prep` green
  from clean main; release.yml published all four platform binaries + plugin bundle + checksums;
  darwin/arm64 asset verified (runs, sha256 matches). On-PATH binary refreshed via `make install`.
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

- **Work the 0.8.0 milestone** (bug floor + Tier-1 deep-dive items; plan: exec desk
  `_meta/plans/release-0.8.0.md`). #79 gates the PM default-on lane (#83). Deferred TUI UX
  items remain in `docs/development/chat-tui-ux-survey.md`.
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
  `make check` (neutrality + self-test + purity + actionlint) · `make test` (bun 59 + go 314) ·
  `make verify` (`librarian/verify.sh`, 48 checks — **now also runs in CI + the release gate**, PR #33) ·
  `make package` (drift guard) · `node scripts/check-version-sync.mjs`. CI (`ci.yml`) is the
  aggregate required check. Note: shellcheck + actionlint are NOT yet CI-enforced (→ #34).
- **Versioning/release (ADR 0005, since PR #41 — runbook `docs/development/releasing.md`)**: SemVer on the
  single root `VERSION`; a bump = edit `VERSION` + the three manifests (sync-guarded) **and** move
  `[Unreleased]` CHANGELOG entries into a dated `## [<version>]` section. `check-changelog.mjs` is
  a **hard gate** (release-prep + release.yml) — a tag with no CHANGELOG section fails.
  `check-version-status.mjs` is a **non-blocking advisory** (`make version-status` + a CI step,
  always exit 0) that warns when `plugin/`/`librarian/` drifted past the last tag without a bump.
  `ci.yml` checks out `fetch-depth: 0` for it. Both scripts + CHANGELOG live outside the neutrality
  surface (repo root / `scripts/`). `make install` builds + drops the **`deskkit`** binary in `~/.local/bin`
  (override `PREFIX=`; renamed from `pocket-librarian` in 0.7.0); `/plugin` updates the plugin
  only — the binary is a separate artifact.
- **Installing from the private repo** (until the public launch at ≥ v1.0.0): the public
  `install.sh` / `curl|bash` path 404s by design while private — use authed `gh` instead. Two ways:
  - **From a local clone** (version-stamped, what a dev should use): `make install` (root or
    `librarian/`) → `~/.local/bin/deskkit`.
  - **Straight from the release** (no clone/build): download the platform asset with `gh`:
    ```bash
    gh release download v0.7.0 --repo hsb3/desk-standard \
      --pattern "deskkit_*_$(uname -s|tr '[:upper:]' '[:lower:]')_$(uname -m|sed 's/x86_64/amd64/;s/aarch64/arm64/')" \
      --output ~/.local/bin/deskkit --clobber && chmod +x ~/.local/bin/deskkit
    ```
    (Assets are `deskkit_<ver>_{darwin,linux}_{amd64,arm64}` since 0.7.0; verify against the
    release's `checksums.txt`.)
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
- **Store location** (since PR #24): no `--dir` → `$XDG_DATA_HOME/deskkit/
  <DESK_NAME>/` (dir renamed with the binary in 0.7.0); unresolvable DESK_NAME + no `--dir` →
  exit 1 (serve/migrate included).
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

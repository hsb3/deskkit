# Changelog

All notable changes to this repository — both the **plugin** (Claude Code plugin + MCP server)
and the **librarian** (`pocket-librarian` Go binary) — are recorded here. The two ship under one
repo version (the root `VERSION` file); a release tags that single version.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html). See
[`docs/development/releasing.md`](docs/development/releasing.md) for how a version is bumped and cut, and
[`docs/decisions/0005-versioning-and-changelog.md`](docs/decisions/0005-versioning-and-changelog.md)
for why this policy exists.

## [Unreleased]

### Added

- **SOP kit library (`kits/`) + schema-v1 doc-type dimension.** The 23 headcase SOP kits
  (guide/template/example per doc type) ported into the repo, neutralized to identity-neutral
  shipped artifacts (0013 S4(a), item 9). Indexed by the root `kits.yaml` manifest with a
  `scripts/check-kits.mjs` drift guard (in `make check` + CI). The kit frontmatter contract now
  lives in `schema/doctypes.yaml` (schema v1's doc-type dimension, successor of the vault
  `types:` model), with the seven previously-unschematized kit types added and the `user-defined`
  nonstandard types deliberately excluded. `kits/` is inside the neutrality-lint scan surface.
  Port + gap dispositions: `docs/decisions/0006-kit-port-schema-reconciliation.md`. The vault
  copies are frozen (read-only journal).
- **`make install`** — build the version-stamped librarian binary and install it to `~/.local/bin`
  (override with `make install PREFIX=/usr/local`). The one-command update-from-source path.
- **`record_feedback`** — the librarian can log feedback entries into its own store: a `problem`
  entry when a tool fails or a desk convention doesn't fit mid-task, or a `feedback` entry when
  the user asks it to record something. New `feedback` collection (migration 0013), the tool on
  all surfaces (agent, chat, MCP) plus a `record-feedback` CLI subcommand, and a `feedback` kind
  for `query` (open entries, newest first).
- **Chat TUI UX pass.** Per-role left gutters (a thick accent bar for the user, a faint thin bar
  for the librarian), a per-turn `model · latency` footer, and a fuller bubbles/help footer:
  `ctrl+g` toggles the full keybind help, `ctrl+y` copies the last answer's raw markdown (with a
  toast confirmation), and `up`/`down` at the textarea's edge rows walk prompt history, stashing
  and restoring the in-progress draft. Streaming shows a "▼ new output" hint when the transcript
  is scrolled up while new tokens arrive. `NO_COLOR` now also swaps the spinner for a static
  "working…" indicator, alongside the existing color-free rendering.
- **`pocket-librarian init [dir]`** — scaffolds the minimal `_knowledge/profile.yaml` a folder
  needs to work as a desk (desk name from the folder's basename, `root: "."`); idempotent,
  `--force` to overwrite, `--with-env` to also write a `.env` stub. It never creates the store. A
  store-touching command that can't resolve config on an interactive terminal now offers to run
  it ("Set up this folder as a desk? [Y/n]") and continues seamlessly on accept; the root
  `--no-input` flag (and a non-TTY) keeps the prior fail-closed error.

### Changed

- **Chat TUI migrated to the Charm v2 stack** (bubbletea v2, lipgloss v2, bubbles v2,
  glamour v2 — the `charm.land` modules), recorded as
  [ADR 0007](docs/decisions/0007-tui-charm-v2-stack.md). No feature or visual changes: the
  TUI keeps rendering on the terminal's own background, the theme is still resolved once
  pre-program (flag > env > one background probe), and no terminal query ever runs after
  startup — glamour v2 removing auto-style detection makes part of that guarantee
  structural. Drops the v1-era `termenv` dependency and the global background-cache pin.

### Fixed

- **Unreadable chat answers on light terminals.** `chat`'s full-screen TUI rendered with a single
  fixed dark palette. Add `chat --theme light|dark|auto` (default `auto`, a one-shot terminal-
  background probe run once before the Bubble Tea program starts — never at render time) and a
  `LIBRARIAN_THEME` env override, with precedence flag > env > auto-detect.

## [0.5.0] — 2026-07-18

The `chat` interactive surface graduates from a line REPL to a full-screen terminal UI, and the
streaming substrate underneath it becomes reusable. Recorded as
[ADR 0004](docs/decisions/0004-chat-full-screen-tui.md).

### Added

- **Full-screen chat TUI.** On a terminal (stdin **and** stdout are TTYs), `librarian chat` now
  opens a full-screen Bubble Tea UI: the answer streams token by token, a finished answer renders
  as markdown, and each tool call collapses to one faint line.
- **Conversation resume and switching.** `ctrl+o` opens a picker of prior conversations to
  resume; `ctrl+n` starts a fresh one — both no-ops while a turn is streaming. Resumed history is
  rehydrated for the model with orphaned user rows collapsed.
- **Cancel an in-flight turn.** `esc` interrupts a streaming turn (the reply is badged
  `(interrupted)`) or closes the resume picker; `ctrl+t` toggles tool-step detail.
- **Streaming event layer** (`agent.Session.StreamTurn`) emitting `token` / `tool_start` /
  `tool_end` / `final` / `error` events over a JSON-taggable `Event` type — the reusable
  substrate the deferred webapp SSE route (ADR 0001, option b) can marshal directly.

### Changed

- **`chat` auto-detects the terminal.** It launches the TUI when interactive and falls back to
  the original line REPL when either end is piped or `--plain` is passed — the non-TTY path is
  byte-for-byte the previous REPL.
- `Turn()` is now a thin drain over `StreamTurn`, so the REPL and the TUI share one persistence
  path.

### Fixed

- **Multi-turn transcript persistence.** The persistence high-water-mark was per-session, so
  multi-turn sessions duplicated the prior assistant row (no-tool turns) or dropped the new user
  row (tool turns). It is now re-baselined per turn and guarded by an exactly-once transcript
  regression test.
- **Zero-argument tool calls** (e.g. `sweep`, `patrol`) streamed no argument deltas, leaving
  `ArgumentsInJSON == ""` and killing the turn on unmarshal; a normalizing adapter now maps `""`
  → `"{}"` at tool registration.

## [0.4.0] — 2026-07-17

First tagged release — the distribution and hardening baseline. (Pre-`0.4.0` development history,
including the `v0.0.1-alpha` tag, lives in the git log and the merged PRs.)

### Added

- **Curl-able installer** (`install.sh`) and a tag-triggered **release workflow** that
  cross-compiles the librarian for darwin/linux × amd64/arm64 (pure-Go), builds the plugin
  bundle, and publishes a GitHub release with sha256 `checksums.txt`.
- **Unified repo version.** One canonical `VERSION` drives the librarian binary (via ldflags) and
  the three shipped manifests (`plugin.json`, `plugin/package.json`, `marketplace.json`),
  drift-guarded in CI and pre-commit.
- **Makefile task interface** (`make help`) as the canonical entry point, plus lefthook
  pre-commit hooks mirroring CI, and three user guides + demo media.
- **XDG store home + desk open-guard** ([ADR 0002](docs/decisions/0002-multi-desk-topology-store-per-desk.md)):
  stores default to `$XDG_DATA_HOME/pocket-librarian/<DESK_NAME>/`; a store refuses a mismatched
  desk name.
- **Line-REPL `chat` + trigger wake layer** ([ADR 0001](docs/decisions/0001-interactive-surface-tui-first.md)).

### Changed

- **Tool commands self-initialize the store** ([ADR 0003](docs/decisions/0003-tool-commands-self-initialize-store.md)):
  `sweep`/`query`/`patrol`/`chat`/etc. run the app migrations idempotently at first touch, so a
  fresh desk needs no manual `migrate up`.

### Fixed

- `query` on an uninitialized store now returns an actionable message instead of a bare
  `sql: no rows in result set`.
- Record-original-first is no longer capped at PocketBase's 5000-char default; content fields are
  widened so large desk files record and restore byte-exact.

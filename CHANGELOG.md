# Changelog

All notable changes to this repository — both the **plugin** (Claude Code plugin + MCP server)
and the **librarian** (`deskkit` Go binary; `pocket-librarian` through 0.6.0) — are recorded
here. The two ship under one repo version (the root `VERSION` file); a release tags that
single version.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html). See
[`docs/development/releasing.md`](docs/development/releasing.md) for how a version is bumped and cut, and
[`docs/decisions/0005-versioning-and-changelog.md`](docs/decisions/0005-versioning-and-changelog.md)
for why this policy exists.

## [Unreleased]

### Added

- **Sweep-time content indexing + `query search`/`content` kinds** (#89). Sweep now stores each
  file's body in a new `files.content` column (migration `0021`), so swept content is retrievable
  and searchable through a tool surface for the first time — previously the raw bytes were used
  only to compute a checksum and parse frontmatter, and nothing indexed was retrievable. Two new
  `query` **kinds** carry it (no new tool, no new CLI subcommand — the count surfaces are
  unchanged): `search` does substring/keyword retrieval over the indexed body via PocketBase's
  LIKE-contains operator (`content ~ term`, not FTS5; embeddings/vector search are out of scope),
  returning each match with a context snippet (`--term`, `--limit` default 20 / hard-capped 200);
  `content` returns one file's stored body by desk-relative `--path`. Indexing is UTF-8-only,
  never indexes a file under `SECRETS_DIR`, and is capped rune-safe at 1,000,000 chars; the body
  is re-derivable by a fresh sweep, so the store stays disposable. Store schema version 20 → 21.
- **Document identity + schema hygiene** (#123, ADR 0017). Three independent fixes shipped as
  one package. (A) **Frontmatter `id`** is now a recognized, OPTIONAL document-identity
  primitive: sweep reads it into a new `files.doc_id` column (migration `0018`) and matches an
  existing row by `doc_id` FIRST, falling back to `path`, so a renamed document that carries an
  `id` updates its existing record at the new path instead of being soft-deleted and
  re-inserted — rename stops discarding history. A document with no `id` keeps today's
  behavior unchanged, and two documents sharing one `id` within a sweep are never merged: the
  duplicate falls back to path-matching and is surfaced as a patrol-visible finding
  (`duplicate-doc-id`). (B) **`files.entity_type` is renamed to `files.doctype`**
  (migration `0019`, in place, reversible) — the old name collided with the schema's unrelated
  `entity_type` enum (a person/company classification); only the column name changed, the
  value did not, and every code/test/spec literal referring to the old column name moved with
  it. (C) **Explicit `Max` caps on seven content-bearing `TextField`s** that still rode
  PocketBase's implicit 5,000-char default (migration `0020`; `patrol_findings.detail` /
  `proposed_fix` widen to 50,000, five summary/detail/error fields tighten to 2,000), plus a
  new dependency-free `scripts/check-textfield-max.mjs` recurrence guard (wired into
  `make check` + CI) that fails any future uncapped content `TextField` by name. Store schema
  version 17 → 20.
- **Tool-surface drift guard** (#121, ADR 0016). `docs/tool-surface.md` — the authoritative,
  empirically-derived map of every tool-bearing surface (#94) — is now pinned to source by a
  mechanical guard, closing the manual "remember to re-run the probe" gap that let the old
  "seven-tool core" shorthand rot. A dependency-free `scripts/check-tool-surface.mjs` (wired into
  `make check` + CI next to `check-prompt-drift.mjs`) cross-checks the **Plugin TS MCP server** count
  (4, the `TOOLS` array in `plugin/core/tools.ts`) and the **Librarian CLI** base count (16, the
  `AddCommand` registrations in `librarian/cmd/deskkit/main.go` plus the framework system commands)
  against the numbers the doc states, with a `--self-test` proving it fails RED on a tool
  added/removed without a matching doc edit. The gate-dependent **Librarian MCP** counts
  (5 / 6 / 17 / 18 by `LIBRARIAN_AUTONOMOUS_WRITES` × `PM_ENABLED`, and the `MCP_MODULES=pm` desk-pm
  mount → 12) are pinned by a Go test (`TestToolSurfaceDoc_MCPCounts`) that reads the same doc counts
  and re-derives them from the real `toolcore` gate on the `go test ./...` lane — no reimplemented
  gate arithmetic to drift. The guard pins **counts**, not the doc's bytes, so an unrelated prose/row
  edit never trips it. A new gate only; no runtime behavior changes.
- **`desk-persona` — the composed librarian + PM Claude Code bundle** (#119, ADR 0014(a), ADR
  0015; the platform's v1 proof surface per ADR 0009). A new `plugin/desk-persona/` marketplace
  entry mounts one `deskkit mcp-serve` server with `MCP_MODULES=librarian,pm`, exposing all 17
  tools (the 5 librarian tools `sweep`/`patrol`/`propose_fix`/`query`/`record_feedback` plus the
  12 PM tools) behind two agents (`librarian-operator`, `pm-operator`) and the three PM skills
  (`pm-session-open`, `pm-advance-item`, `pm-triage`). The PM-sourced artifacts are copies of the
  existing `desk-pm` content with the MCP namespace rewritten `mcp__desk-pm__` →
  `mcp__desk-persona__` (one authored PM source; `desk-pm` coexists, unchanged); the
  librarian-operator body is generated from the canonical librarian instruction
  (`librarian/templates/librarian-system-prompt.txt`). A new drift guard
  (`scripts/check-persona-drift.mjs`, wired into `make check`) fails non-zero if either generated
  surface is hand-edited out of sync with its source. `plugin/desk-persona.test.ts` (`bun test`,
  `plugin/package.json`'s test glob extended) asserts the composed tool set by name, guards
  against phantom/invented tool identifiers across the bundle, and pins the marketplace/version
  wiring. `docs/README.md` gains a pointer to the bundle's `README.md`.
- **Findings lifecycle completed** (#118, ADR 0013). Finishes the disposition sub-machine #93
  opened. (1) The dead `state.dismissed` enum value is retired — migration `0015` remaps any
  residual row to `flagged` (data-first) then shrinks `patrol_findings.state` to
  `flagged`/`fixed`/`resolved`. (2) "Open findings" now means one thing everywhere: `query
  summary` and `query uncollapsed` became disposition-aware (`disposition = 'open'`), matching
  `query findings`' live default, so a disposed finding no longer inflates counts on some surfaces
  and not others. (3) Dispositions carry provenance — migration `0016` adds `actor` (max 200),
  `reason` (max 2000), and `disposed_at` (a plain date set at dispose time) to `patrol_findings`;
  `deskkit findings dispose <id> --as <disposition> [--by <actor>] [--reason <text>]` persists
  them (both flags optional, no baked default actor), and a re-fired finding inherits disposition
  AND provenance on `(file, rule, checksum)`. (4) `adoption_log.event` shrinks to writer-backed
  reality — migration `0017` retires the five writerless values, keeping only `fix` (its readers
  and deskguard role unchanged). Store schema version 14 → 17.
- **Prompt-copy drift guard + git-is-truth prompt governance** (#120, ADR 0015). A new
  dependency-free `scripts/check-prompt-drift.mjs` (wired into `make check` + CI next to
  `check-kits.mjs`) asserts the librarian system-prompt embed
  (`librarian/templates/librarian-system-prompt.txt`) and its "kept verbatim" quote in
  `docs/pocket-librarian-v1-spec.md` §6.1 stay **byte-identical**, closing the drift class that
  shipped a stale six-tool spec quote against the five-tool embed with nothing to catch it.
  `prompt.Seed`'s doc comments, spec §4.10/§6.1, and `librarian/README.md` now state ADR 0015
  in its own terms: the DB `prompts` row is a **re-seeded cache** (not canonical), GUI/REST
  edits are **ephemeral by rule**, and `_knowledge/` is the only durable customization path —
  with a documented **"reset to shipped"** affordance (delete the row via the admin console; the
  next command or a `serve` restart re-seeds it from the embed). Documentation + a new gate only;
  no runtime behavior changes (`Seed`'s seed-if-absent logic and the resolver are unchanged).
- **Tool-level MCP module gating on a shared mount** (`MCP_MODULES`; the agent integration
  contract — [ADR 0014](docs/decisions/0014-agent-integration-contract.md),
  `docs/agent-integration-contract-v1-spec.md`, #114 under epic #129). `deskkit mcp-serve` now filters its
  exposed tool set to the modules named in `MCP_MODULES`, keyed on each tool's `ToolSpec.Module`
  (`internal/core/mcp/server.go` → `toolcore.SelectByModules` over `toolcore.ExposedSpecs(cfg)`),
  so a shared MCP mount carries only the tools it is meant to expose. The **desk-pm** plugin mount
  (`plugin/desk-pm/.mcp.json`) declares `MCP_MODULES=pm` alongside `PM_ENABLED=true` and therefore
  exposes **exactly the 12 PM tools, dropping the 5 librarian ride-alongs (17 → 12)**. The
  semantics are three-way and deliberately non-collapsing: `MCP_MODULES` **unset** exposes every
  module (the 5 / 6 / 17 / 18 counts are unchanged); **set-but-empty** (`""`, `" , "`) or
  **unresolvable** (a typo, or a module not registered/enabled on this desk) **fails loud** with a
  direct `os.Exit(1)` and an actionable stderr line — never a silent fallback to "all". The
  `deskkit mcp-serve` mount signal now names the gated set (`modules: pm; 12 tool(s) exposed: …`).
  `docs/tool-surface.md` gains the module-gating axis (§2.1) and an extended count-derivation
  method; the eino agent loop stays librarian-only (ADR 0014(c)) and its prompt no longer names PM
  tools it never receives.
- **Typed cross-reference contract in `schema/`** (#116, ADR 0011). `schema/references.yaml`
  adds schema v1's third dimension — a `{kind, target}` reference primitive with a closed
  `kind` enum (seeded `issue`, `url`) and a raw `target` string. The desk-relative repo
  qualifier is documented as read-time-resolved from `profile.repos.shorthand.issue_default`
  and never persisted (no default qualifier ships — identity-neutral). A validation guard lands
  in both lanes — `ReferenceVocab`/`ValidateReference` (Go, `librarian/internal/core/schema/
  references.go`) and `validateReference` (TS, `plugin/core/references.ts`) — each drift-guarded
  byte-for-byte against the canonical file. No field migrates onto the shape and no store
  migration ships: `graduated_to` / `items.pointer` behavior is unchanged, and the field
  migrations ride the schema-v2 track. Forward-pointer notes added to
  `docs/pocket-librarian-v1-spec.md` and `docs/pm-system-v1-spec.md`.
- **Normative `pointer` grammar spec section** (#115). `docs/pm-system-v1-spec.md` §3.1a now
  defines the `items.pointer` grammar ADR 0010 ratified — desk-relative file path with an
  advisory `§ <heading>` suffix; URL and `#`-anchored forms fail closed — and corrects the
  field-table row that still described the pointer as "doc path / issue URL / other locus".
  Docs-only; the shipped, test-pinned behavior is unchanged.
- **Findings disposition lifecycle** (#93). Patrol findings now carry a `disposition`
  (`open`/`acknowledged`/`triaged`/`wont_fix`), orthogonal to `state`. `deskkit findings dispose
  <id> --as <disposition>` marks a finding; `query findings` defaults to live (undisposed) items
  with `--include-disposed` to show history. Dispositions survive re-patrol by inheriting on
  (file, rule, checksum); a finding whose evidence (checksum) changes re-opens automatically.
  Migration `0014_patrol_findings_disposition` backfills existing rows to `open`.
- **`items.type` validated at creation** (#117). `create_item` hard-refuses a non-empty `type`
  outside the schema-v1 vocabulary (ADR 0012), naming the offending value, the known types, and
  `schema/doctypes.yaml`; an absent type stays legal. The importer inherits the check with no
  importer-side code (regression-tested both places). Engine-level only — no DB migration;
  `items.type` stays a bare TextField.
- **Authoritative tool-surface document** (#94) — `docs/tool-surface.md` (linked from
  `docs/README.md`) enumerates every surface with empirically-verified counts: the librarian MCP
  tools by gate (5 default / 6 with `LIBRARIAN_AUTONOMOUS_WRITES` / 17 with `PM_ENABLED` / 18 both),
  the CLI subcommands, and the plugin's separate 4-tool TS MCP server. Replaces the false
  "seven-tool core" help string, which matched no real surface.
- **`deskkit mcp-serve` mount signal** (#79) — a one-line readiness line on stderr (server identity
  + tool count) so an absent PM tool surface is diagnosable instead of silent.
- **Scaffold-frontmatter drift guard** (#80) — `scripts/check-scaffold-frontmatter.mjs` (wired into
  `make check`) asserts scaffold-shipped instruments carry conformant frontmatter.

- **Repo-conformance pass to the code-desk standard** (Lane 6, #86). The repo now passes its own
  `repo-compliance-audit` with zero gaps:
  - **Root entry-doc set** — a `CLAUDE.md` agent-navigation guide (command surface, the one
    identity-neutrality rule, order-sensitive chains, config resolution), `AGENTS.md` (a symlink to
    `CLAUDE.md` — one source, no drift), and `docs/CHARTER.md`, the canonical page with an explicit
    precedence rule and the settled 1.0.0 direction.
  - **Scaffolded meta-structure** (`mise-en-place-scaffold`, additive-only) — `.claude/{agents,hooks,
    rules,memory}` with a tracked `settings.json` + `memory/MEMORY.md` index, root `.mcp.json`,
    `.github/dependabot.yml`, and `docs/decisions/0000-template.md`.
  - **`tests/` declared home** — `tests/README.md` documents that suites live with their products
    (`plugin/` bun, `librarian/` go, `librarian/verify.sh`), keeping `make test` / `make verify` as
    the canonical entries rather than hoisting suites to a root tree.

### Changed

- **`query orphans` hides by-design-unreferenced index/entry files by default** (#100). Basename
  `CLAUDE.md`, `README.md`, and `INDEX.md` (case-insensitive) are structural orphans — empty-doctype
  `.md` files outside `meta`/`memory`/`infra` — but an entry/index doc is what *other* docs point at,
  so it is never a misfiled orphan. The default view now filters them out (an ADDITIONAL filter on
  top of the unchanged structural `isOrphan` predicate) so `orphans` returns only genuine orphans;
  the new `--show-index` flag (`show_index`) opts them back in. No new tool or CLI subcommand.

### Fixed

- **R6 handoff-staleness self-clears on a handoff update, without a re-baseline** (#100). Patrol now
  measures the handoff against the newest change it GUARDS — the newest desk commit **excluding the
  handoff file itself** (`git log -1 --format=%cs -- . :(exclude)<HANDOFF_PATH>`, new
  `desklib.GitNewestCommitExcluding`). Previously `newest` was the whole-tree newest commit, which
  included the handoff's own update commit: the moment a handoff refresh was committed, that commit
  became the newest and the handoff could never be "current with" it, so the finding could only be
  cleared by a re-baseline. An updated handoff dated on/after the newest guarded change now clears
  R6 at the next patrol. The pure `r6Check(text, newest)` core is unchanged — only how `newest` is
  computed at the caller.

## [0.7.0] — 2026-07-19

### Added

- **PM module — collections, state machine, gates** (D3, `docs/pm-system-v1-spec.md` §3–§4,
  epic #55). The document-gated work graph lands as a feature-gated module (`PM_ENABLED` /
  profile `modules.pm.enabled`; OFF by default): five collections (`items`, `dependencies`,
  `transitions`, `notes`, `desk_config`) created by PROGRAMMATIC migrations only when the
  module is enabled — a librarian-only desk gets no PM collections, physically; the rigid
  queue→work→review→terminal machine with the `blocked` side-state; per-desk editable YAML
  gate rules (validated against the schema-v1/kit vocabulary, now embedded from
  `schema/doctypes.yaml` with a byte-identity drift guard) + cross-cutting traits; the §4.1
  transition path (machine → blocked → claim → gates → write + audit + cascade) with
  refusals that name exactly what is missing and land as `gate_refused` audit rows; typed
  dependency edges with `auto`/`auto-reopen`/`manual`/`permanent` cascade semantics;
  optimistic-concurrency version tokens + claim TTL (`PM_CLAIM_TTL`, default 30m). The
  librarian module's `DocumentValidator` seam is now fully wired (desk-file read +
  frontmatter parse + schema-v1 validation); gate evaluation consumes it and fails closed
  without it. The shipped default gate rules are a seed the owner may re-rule.
- **PM surfaces — CLI, MCP tools, TUI views** (D4, `docs/pm-system-v1-spec.md` §5, epic #55).
  The PM tool family is exposed on all three surfaces over one engine (parity asserted by a
  test): the **twelve PM MCP tools** (`get_context`, `list_items`, `get_item`, `create_item`,
  `update_item`, `transition_item`, `block_item`, `unblock_item`, `add_note`, `link_items`,
  `claim_item`, `release_item`) on `deskkit mcp-serve`; the matching `deskkit pm <sub>` CLI
  group (`context`, `list`, `get`, `create`, `update`, `transition`, `block`, `unblock`, `note`,
  `link`, `claim`, `release`) — JSON-first, present only when the module is enabled; and three
  TUI views (`pm context` landing / `pm board` / `pm item`) mounted into the full-screen TUI.
  `get_context` is the single-call cold-start briefing (active, blocked, stalled, recent
  transitions). The three read tools are always agent-available; the nine write tools are gated
  behind `PM_AUTONOMOUS_WRITES` (default ON) — a desk can make agents read-only over the graph
  while the document gate stays the real safety. PM tools write only the store, never desk files.
  Realtime events emit on transitions under `serve`. New PM env vars: `PM_AUTONOMOUS_WRITES`,
  `PM_STALLED_DAYS` (default 14).
- **`desk-pm` complementary plugin** (D5, `docs/pm-system-v1-spec.md` §6, epic #55). A separate
  Claude Code plugin (shared marketplace) that turns the PM graph into an agent surface over the
  MCP tools: the `pm-session-open`, `pm-advance-item`, and `pm-triage` skills; a `pm-operator`
  agent that operates the graph over the twelve tools but never authors gate documents or writes
  a repo; a `SessionStart` hook injecting `deskkit pm context` (silent no-op when PM is off or
  `deskkit` is absent); and a `.mcp.json` launching `deskkit mcp-serve` with `PM_ENABLED=true`.
  Identity-neutral by construction — no person, org, repo, issue, or desk name is hardcoded.
- Adoption dry-run (`TestAdoptionDryRun`): seeds a scratch store from a neutral manifest via the
  importer, observes `get_context` cold-start, a gate refused-then-satisfied, and a dependency
  auto-unblock, proving the live desk is never written (spec §8.1).

### Changed

- **Chassis rename: `pocket-librarian` → `deskkit`** (D2b, `docs/pm-system-v1-spec.md` §2.10,
  epic #55). The Go binary, its build/install/release artifact names, and the canonical store
  home (`$XDG_DATA_HOME/pocket-librarian/<DESK_NAME>/` → `$XDG_DATA_HOME/deskkit/<DESK_NAME>/`)
  are renamed. On startup, a store still at the old home is moved to the new one automatically
  (one logged line) — no desk loses its store across the rename. `install.sh` falls back to the
  pre-rename asset name for releases up to v0.6.0; ADR 0002's store-path literal carries a dated
  correction. The Go module path (`github.com/example/pocket-librarian`) is unchanged.

### Fixed

- dep-snapshot sort now tiebreaks on `Kind` so two edges sharing a (from,to) pair but differing
  in kind order deterministically, removing a flaky false-failure in the rebuild-reproducibility
  oracle (issue #71).

## [0.6.0] — 2026-07-18

### Added

- **SOP kit library (`kits/`) + schema-v1 doc-type dimension.** The 23 headcase SOP kits
  (guide/template/example per doc type) ported into the repo, neutralized to identity-neutral
  shipped artifacts (0013 S4(a), item 9). Indexed by the root `kits.yaml` manifest with a
  `scripts/check-kits.mjs` drift guard (in `make check` + CI). The kit frontmatter contract now
  lives in `schema/doctypes.yaml` (schema v1's doc-type dimension, successor of the vault
  `types:` model), with the seven previously-unschematized kit types added and the `user-defined`
  nonstandard types deliberately excluded. `kits/` is inside the neutrality-lint scan surface.
  Port + gap dispositions: `docs/decisions/0006-kit-port-schema-reconciliation.md` (epic #55,
  D1). The vault copies are frozen (read-only journal).
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

- **Core + modules architecture.** The librarian is refactored into a shared `internal/core/`
  (config, store, migrate, mcp, schema, module registry) with librarian-specific code moved under
  `internal/modules/librarian/`, making the librarian the first module on a reusable substrate
  (epic #55, D2). Internal reorganization only — no change to the CLI, MCP tool surface, schema,
  or store layout.
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

# CLAUDE.md

This file guides coding agents working in this repo. For the project's canonical direction
(what it is, what's settled for 1.0.0) see [`docs/development/CHARTER.md`](docs/development/CHARTER.md); if that page and
this one ever disagree about direction, the charter wins.

## Where the work lives

Tracked work lives on the Kaneo board (project `DESK`), not in this repo — never create
`backlog.md`, `TODO.md`, or a handoff file. The board replaced all three on 2026-08-18.

**Cold start: read task `DESK-21` first** — the `HANDOFF`-labelled task in **Documents**, the
lead lane. It is kept current in place rather than re-filed per session, and it is not work:
never claim it, never move it out. Decisions are tasks labelled `DECISION` in the same lane;
check them before contradicting one. Work comes from the top of **Up Next**; To Do is
untriaged. The `kaneo@dotfiles-agents` plugin reads its config (`KANEO_*`) from
`.claude/settings.local.json`, which is gitignored because the agent key is a credential.
This repo works the board as `deskkit` — follow the kaneo skill's claim ritual before working
a task.

GitHub issues stay the intake and the public record. The repo is linked to the board, so open
issues arrive in To Do, and branches move their own task: name one `desk-<taskNumber>`
(optional hyphen-prefixed suffix) and push → In Progress, PR open → In Review, merge → Done.
Board events also post to the Telegram topic `DESK`. The instance itself is operated from the
`kaneo-ops` repo, whose runbook is the operational contract — nothing about the instance is
configured from here.

## Project Overview

**desk-standard is one binary and one plugin bundle over one shared schema**, all identity-neutral
— nothing shipped carries a person, org, repo, or issue number. A desk is personalized by filling
its `_knowledge/profile.yaml` (copied from the desk-setup scaffold's
`_knowledge/profile.example.yaml`), never by editing a shipped skill, template, or tool. This repo
is NOT itself an exec desk — it ships no repo-root `_knowledge/`; the `_knowledge/` convention
belongs to the desks the tools stand up.

**Architecture — one runtime, one surface bundle, one contract:**

- **`librarian/`** — **deskkit**, a single Go binary with an **embedded PocketBase** store, and the
  entire runtime. Three modules feed one tool core:
  - **`profile`** (always on, pure-read, no collections) — desk personalization: `profile_get`,
    `profile_validate`, `template_render`, `knowledge_index`. Validates against a `go:embed`ed
    schema copy, including the `x-contract-version` gate (ADR 0009).
  - **`librarian`** — indexes a desk, flags rule violations, proposes/applies fixes under a
    record-original-first boundary (byte-exact reversible via `restore`).
  - **`pm`** — a document-gated work graph, on by default (opt out with `PM_ENABLED=false`).

  Surfaces over that one core: **CLI**, an **MCP server** (`deskkit mcp-serve`, narrowed by
  `MCP_MODULES`), a **chat TUI**, and a **browser session page** (`/desk/chat` on the embedded
  serve). Admin console (`make -C librarian gui`) serves PocketBase at `http://127.0.0.1:8090/_/`.
- **`plugins/desk-persona/`** — the ONE Claude Code plugin this marketplace ships: the agent-facing
  surface over that same binary. Seven skills (`desk-setup`, `conventions-standard`,
  `harvest-loop`, `brownfield-adoption`, `pm-session-open`, `pm-advance-item`, `pm-triage`), two
  agents (`librarian-operator`, `pm-operator`), a SessionStart briefing hook, and one `.mcp.json`
  launching `deskkit mcp-serve` with `MCP_MODULES=profile,librarian,pm`. It ships no runtime of its
  own.
- **`schema/`** — schema v1: the product-neutral contract, and the single source of truth for the
  copies embedded in the binary.

**Project structure** (every top-level entry annotated):

```
librarian/         the deskkit binary — embedded PocketBase; profile/librarian/pm modules;
                   CLI/MCP/TUI/web surfaces; verify.sh + e2e gates
plugins/           the marketplace-distributed bundle (a marketplace install copies ONLY this)
  desk-persona/    the only bundle: 7 skills, librarian-operator + pm-operator agents,
                   SessionStart hook, .mcp.json — authored in place, nothing generated
schema/            schema v1 — the source of truth for the binary's embedded copies
docs/              specs, the CHARTER, and using/developing guides (ADRs live on the board)
scripts/           repo-wide gate scripts (*.mjs) + record-media.sh
tests/             signpost only — suites live with their products; see tests/README.md
kits/ + kits.yaml  SOP template library + its drift-guarded manifest
.claude/           agent config: skills/, agents/, rules/, memory/, settings.json (tracked policy),
                   atelier.local.md (handoff lives on the DESK board, not in a file)
.github/           CI workflows + issue/PR templates + dependabot
Makefile           the canonical task interface — `make help` lists targets
VERSION            single source of truth for the release version (binary + bundle ship off it)
```

## The one rule that matters: identity-neutrality

**Nothing under `librarian/`, `plugins/`, or `kits/` may hardcode a deployment identity** — no
person, org, repo name, or bare issue reference (`#123`). This is the product's core promise and
the failure that rots worst: a hardcoded identity ships inside a distributed binary/plugin and
can't be pulled back. It is enforced in CI, not by convention:

```
# scope = librarian/ + plugins/ + kits/ recursively; docs/ and repo-root files are EXEMPT
node scripts/check-neutrality.mjs            # scans the shipped tree — FAILS on any hardcoded identity
node scripts/check-neutrality.mjs --self-test  # proves the scanner still detects a seeded violation
```

The classic trip: a bare `#18`-style issue ref in a **Go comment or test** under `librarian/`
fails the lint. Write issue-free comments there. In `docs/`, the spec's `#N` references are fine.

## Commands

The Makefile is the interface; run gate commands **bare, never piped** — a pipe (`| tail`) masks
the exit code and has let a failing gate through before (incident, 2026-07-17).

| Command | What it does |
|---|---|
| `make help` | List all targets (default goal) |
| `make setup` | `lefthook install` (git hooks) — there is no package-manager step |
| `make build` | Build the `deskkit` binary (version-stamped) |
| `make test` | Fast unit tests: `go test ./...` in `librarian/` |
| `make check` | Repo gates: neutrality + self-test, kit-drift, scaffold frontmatter, persona drift, textfield-max, query-kind drift + self-test, doc-link integrity + self-test, shellcheck, actionlint, workflow SHA-pin drift + self-test, profile-root drift + self-test |
| `make verify` | Librarian integration gate — `librarian/verify.sh` (throwaway scratch desk) |
| `make e2e` | End-to-end system-behaviour suite — whole system (cold-start → profile → librarian → PM → surfaces → release-shaped) on a throwaway desk; offline, no LLM key (`librarian/e2e/e2e.sh`) |
| `make package` | Informational no-op — this target generates nothing; the bundle's one generated file is written by `node scripts/check-persona-drift.mjs --write` |
| `make install` | Build + install the `deskkit` binary to `~/.local/bin` (override `PREFIX=`) |
| `node scripts/check-version-sync.mjs` | Assert root `VERSION` matches the shipped manifests |
| `make version-status` | Advisory (non-blocking): unreleased product changes since the last tag |
| `make release-prep` | Pre-tag gate (see order below) |

The librarian lane has its own `librarian/Makefile` (`make -C librarian build|test|fmt|sweep|patrol`; `fmt` is the gofmt gate, also run in CI);
`apply-fix` is deliberately **not** a target — it's supervised-only, run by hand, and every fix is
reversible with `deskkit restore --by-path <path>`.

The table above is `make` targets; `deskkit desks` and `deskkit config show|path|edit|set` are
plain binary subcommands (no store opened) — see "Configuration resolution" below for what they
show. `deskkit --help` groups the whole command menu into six sections instead of one alphabetic
list.

## Architecture notes

### Generated and copied artifacts — edit the source, not the copy

**The marketplace bundle is authored in place with one exception:
`plugins/desk-persona/agents/librarian-operator.md` is GENERATED** (it says so in a marker at its
top — never hand-edit it). `make package` generates nothing and only says so. What needs care is
the handful of files that are *derived from a canonical source*:

| File | Source of truth | Guard |
|---|---|---|
| `plugins/desk-persona/agents/librarian-operator.md` | `librarian/templates/librarian-system-prompt.txt` — regenerate with `node scripts/check-persona-drift.mjs --write` | `node scripts/check-persona-drift.mjs` (`make check`) |
| `librarian/internal/core/schema/profile.schema.yaml` | `schema/profile.schema.yaml` — `go:embed` can't reach outside the Go module, so the binary carries a copy | `TestProfileSchemaEmbeddedCopy_MatchesRepoRoot` (`make test`) |
| `librarian/internal/core/schema/references.yaml` | `schema/references.yaml` — same reason | `TestReferencesEmbeddedCopy_MatchesRepoRoot` (`make test`) |
| `kits/` tree | authored, but `kits.yaml` must match it | `node scripts/check-kits.mjs` |

Edit the repo-root `schema/` file first, then re-copy it into `librarian/internal/core/schema/`;
the guards are byte-for-byte and fail loudly on a one-sided edit.

### Architectural rules and their enforcing checks

| Rule | Enforced by |
|---|---|
| Identity-neutrality (shipped tree) | `scripts/check-neutrality.mjs` (+ `--self-test`) |
| Embedded schema copies stay byte-identical to `schema/` | `TestProfileSchemaEmbeddedCopy_MatchesRepoRoot` / `TestReferencesEmbeddedCopy_MatchesRepoRoot` (`make test`) |
| `VERSION` == shipped manifests | `scripts/check-version-sync.mjs` |
| `kits.yaml` == `kits/` tree | `scripts/check-kits.mjs` |
| `docs/development/specs/tool-surface.md` counts (CLI + gated MCP) match source (ADR 0016) | `TestToolSurfaceDoc_*` in `librarian/internal/core/mcp/tool_surface_doc_test.go` (`make test`) |
| Scaffold instruments carry conformant frontmatter | `scripts/check-scaffold-frontmatter.mjs` |
| Persona `librarian-operator` agent stays generated from the librarian prompt (ADR 0014/0015); PM surfaces are authored-in-place post-fold | `scripts/check-persona-drift.mjs` |
| The shipped bundle's authored artifacts (`.mcp.json` modules, agent `tools:`, skill tool refs, inventory) name only real modules/tools | `TestBundle*` in `librarian/internal/core/mcp/bundle_shape_test.go` (`make test`) |
| Content TextFields carry an explicit Max (ADR 0017) | `scripts/check-textfield-max.mjs` (+ `--self-test`) |
| Spec query-kind list == CLI/MCP registry (types.go ↔ spec §5.6 ↔ query.go switch) | `scripts/check-query-kinds.mjs` (+ `--self-test`) |
| Librarian Go tree stays gofmt-clean | `gofmt -l` via `make -C librarian fmt` (CI librarian lane) |
| A tagged release has a CHANGELOG section | `scripts/check-changelog.mjs` (release gate) |
| Every workflow `uses:` stays SHA-pinned (no mutable `@vN` tag) | `scripts/check-workflow-pins.mjs` (+ `--self-test`) |
| Profile root (`_knowledge`) pinned identically in `schema/paths.yaml` and the Go constant | `scripts/check-profile-root.mjs` (+ `--self-test`) |
| Shell entry points stay lint-clean (install.sh, verify.sh, dogfood-*.sh, sandbox/*, record-media, e2e/*) | `shellcheck` (CI + `make check`) |
| Every cited doc/media path on the published+shipped surface resolves (no dangling links; incident 2026-07-24) | `scripts/check-doc-links.mjs` (+ `--self-test`) |

### Order-sensitive chains

**`make release-prep`** runs in this order and aborts on the first failure — don't reorder:
1. clean working tree + on `main` (a release is cut from main only)
2. `check-version-sync` → `check-changelog` → `check-version-status` (cheap manifest/changelog gates first)
3. `make check` then `make test` (the expensive lanes last)
4. prints the tag/push commands — it never auto-tags.

**PocketBase bootstraps before cobra** (librarian): `Execute()` calls `Bootstrap()` — which
**creates the data dir** — before `RootCmd.Execute()` dispatches any `RunE`/`PreRunE`. Anything that
must *prevent* store creation has to run in `main()` before the app starts (the argv-scan location
guard and `init` do exactly this). Fail-closed behavior in `serve` paths needs a direct
`os.Exit(1)`, not a returned error (PocketBase discards serve-goroutine RunE errors).

## Configuration resolution

Every resolved field wins on one of four legs, in this order: **env > per-desk
`_knowledge/profile.*` > central config > built-in default** (`librarian/internal/core/config/config.go`).
The central leg is `$XDG_CONFIG_HOME/deskkit/config.yaml` (falling back to `~/.config/deskkit/config.yaml`),
a machine-wide file created 0600 in a 0700 dir via `deskkit config set/edit`. Only three fields
read it: `LLM_PROVIDER` (`llm.provider`), `LLM_MODEL` (`llm.model`), and `DESK_NAME`
(`default_desk`) — everything else stops at profile-or-default. `deskkit config show` prints
every resolved value with the leg that won.

**Librarian store / profile discovery** (first match wins):
1. explicit `--dir` (overrides everything)
2. `DESK_ROOT` + `DESK_NAME` env vars
3. `_knowledge/profile.yaml` discovered by walk-up from cwd (a profile with `desk.name` +
   `root: "."` needs no env vars when you run `deskkit` inside the desk)
4. the central config's `default_desk`, when set and no profile/env supplied `DESK_NAME`
5. no `--dir` and a resolvable `DESK_NAME` → store at `$XDG_DATA_HOME/deskkit/<DESK_NAME>/`
6. unresolvable `DESK_NAME` and no `--dir` → **exit 1** (serve/migrate included)

**LLM provider/key** (only `agent`/`chat`/MCP-driven calls need it):
- provider: `LLM_PROVIDER` env → `profile.models` → central `llm.provider` → `anthropic` (default)
- model: `LLM_MODEL` env → `profile.models` → central `llm.model` → the built-in default model
- key: the env var named by `LLM_API_KEY_ENV` / `secrets_ref.llm_api_key` (or the per-provider
  default `ANTHROPIC_API_KEY` / `OPENAI_API_KEY` / `GEMINI_API_KEY`) → else the central config's
  `llm.api_key`. The key never lives on the `Config` struct — it resolves at use time
  (`config.ResolveAPIKey`) — so setting it once via `deskkit config set llm.api_key <key>` is
  enough to run `agent`/`chat` with zero env vars set.
- `LIBRARIAN_AUTONOMOUS_WRITES=true` gates `apply_fix` (checked at execution time)

`deskkit desks` lists the desks this machine has a store for (marking which one the cwd
resolves to); `deskkit config show|path|edit|set` inspects/edits the central file above. Neither
opens a store. `deskkit --help` groups the command menu into six sections (Setup & config,
Inspect, Fix, Work graph, Agent, Admin) rather than one alphabetic list.

### No in-repo dogfooding (ruled 2026-07-22, "no dogfood")

This repo does not register its own MCP servers on itself — there is no root `.mcp.json` and
none should be added. The tools this repo builds (the `deskkit` binary and the `desk-persona`
plugin) are for coordinating *other* desks; that standard doesn't apply reflexively to the repo
that builds it, and the coordination tooling must live outside it — on a desk built to operate on
this repo, e.g. the paired executive desk (DESK-21 §0). In-repo verification instead
runs through `make verify` (`librarian/verify.sh`, a throwaway scratch desk) and `make e2e`, both
of which stand up disposable desks rather than pointing the binary at this repo's own tree.
`deskkit apply-fix` / `restore` stay `ask`-gated in `.claude/settings.json` regardless, matching
the librarian's supervised-write boundary wherever it runs.

## Cross-cutting gotchas

- **Store self-initializes** at the `requireConfig` choke point — a fresh store needs no manual
  `migrate up`. A new store-touching command routed through `requireConfig` inherits this.
- **Altering a shipped PocketBase collection: add a forward migration, never edit the applied one**
  (fresh stores only see the edit; existing stores need the new migration). Content-bearing text
  fields must set an explicit `Max` — a bare `TextField` silently caps at 5000 chars.
- **Doc paths are load-bearing — moving one means repointing its citations in the same change.**
  The build specs live in `docs/development/specs/` (`pocket-librarian-v1-spec.md`,
  `pm-system-v1-spec.md`, `tool-surface.md`, `agent-integration-contract-v1-spec.md`,
  `element-model-v2-draft.md`) and are read by CI gates + a Go test; they are cited from code,
  skills, and the neutrality allowlist. `scripts/check-doc-links.mjs` (in
  `make check`) fails on any dangling doc/media citation across the published+shipped surface, so a
  move that forgets a citation is caught — see `docs/development/docs-layout.md` for the full layout
  contract (what lives where, what's load-bearing, and how the working desk differs).
- **Toolchain floor:** Go `1.25` (PocketBase's `go.mod` floors it). Node is needed only to run the
  `scripts/*.mjs` gates — nothing shipped is built with it.

## Documentation

- **[`docs/development/CHARTER.md`](docs/development/CHARTER.md)** — canonical page (what it is, 1.0.0 direction, precedence rule).
- **[`docs/README.md`](docs/README.md)** — docs index, split Using vs Developing.
- **Kaneo tasks labelled `DECISION`** (Documents lane) — the ADRs, `ADR 0001` = `DESK-23` through
  `ADR 0022` = `DESK-43`, moved off disk on 2026-08-18. File a new decision as a board task, not as
  a file; code and docs cite it as a bare `ADR NNNN`, because a board id may not ship in an
  identity-neutral artifact. The deleted files are in git history (`git log -- docs/decisions/`).
- **Kaneo task `DESK-21`** — session-to-session bridge: current standing + deep gotchas. On the
  board, not in the repo (see "Where the work lives"); the pre-2026-08-18 file is in git history
  (`git log -- HANDOFF.md`).
- **[`CHANGELOG.md`](CHANGELOG.md)** — what changed per release.

## Keeping this file current

Update CLAUDE.md in the **same change** as the code it describes. A named command, path, or test
that no longer exists is worse than silence — if you rename a target or a gate, fix it here too.

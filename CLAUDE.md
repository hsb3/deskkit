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

**deskkit is one binary and one plugin bundle over one shared schema**, all identity-neutral
— nothing shipped carries a person, org, repo, or issue number. A desk is personalized by filling
its `_knowledge/profile.yaml` (copied from the desk-setup scaffold's
`_knowledge/profile.example.yaml`), never by editing a shipped skill, template, or tool. This repo
is NOT itself an exec desk — it ships no repo-root `_knowledge/`; the `_knowledge/` convention
belongs to the desks the tools stand up.

**Architecture — one runtime, one surface bundle, one contract:**

- **`cmd/deskkit` + `internal/`** — **deskkit**, a single Go binary with an **embedded PocketBase**
  store, and the entire runtime. The Go module is the repo itself (`github.com/hsb3/deskkit`); there
  is no container directory. Three modules feed one tool core:
  - **`profile`** (always on, pure-read, no collections) — desk personalization: `profile_get`,
    `profile_validate`, `template_render`, `knowledge_index`. Validates against a `go:embed`ed
    schema copy, including the `x-contract-version` gate (ADR 0009).
  - **`librarian`** — indexes a desk, flags rule violations, proposes/applies fixes under a
    record-original-first boundary (byte-exact reversible via `restore`).
  - **`pm`** — a document-gated work graph, on by default (opt out with `PM_ENABLED=false`).

  Surfaces over that one core: **CLI**, an **MCP server** (`deskkit mcp-serve`, narrowed by
  `MCP_MODULES`), a **chat TUI**, and a **browser SPA** at `/` on the embedded serve — a 34px
  rail of work modes (Queue, Library, Patrol, Work, Agent, Config) that is also the app's
  keyboard map (`web/src/lib/shell.ts` + `keys.ts`: every button has a key and every key has a
  button), over chat plus browse of files/findings/agent runs/PM items. Every collection is
  described declaratively in `web/src/lib/collections.ts` and the component reads that descriptor
  rather than branching on a collection name — adding an entity is a config entry, which is the
  design's own falsifiable test. A document's editable surface is its STRUCTURED part only
  (`status`, `type`); the prose body hands off to the URL template the desk declares in
  `preferences.editor_url`, because hardcoding an editor would ship a personalization. Writes and
  deletes go through the write-through path — `tools.WriteDoc` / `tools.DeleteDoc`:
  record-original-first, byte-exact, reversible via `restore`, compare-and-swap on the file
  checksum; `/desk/chat` still resolves via the SPA's index fallback. Admin console (`make gui`) serves PocketBase at
  `http://127.0.0.1:8090/_/`.
- **`plugins/deskkit/`** — the ONE Claude Code plugin this marketplace ships: the agent-facing
  surface over that same binary. Seven skills (`desk-setup`, `conventions-standard`,
  `harvest-loop`, `brownfield-adoption`, `pm-session-open`, `pm-advance-item`, `pm-triage`), two
  agents (`librarian-operator`, `pm-operator`), a SessionStart briefing hook, and one `.mcp.json`
  launching `deskkit mcp-serve` with `MCP_MODULES=profile,librarian,pm`. It ships no runtime of its
  own.
- **`schema/`** — schema v1: the product-neutral contract, and the single source of truth for the
  copies embedded in the binary.

**Project structure** (every top-level entry annotated):

```
go.mod / go.sum    the ONE Go module, at the repo root: github.com/hsb3/deskkit
cmd/deskkit/       the binary's main package — `go install github.com/hsb3/deskkit/cmd/deskkit@latest`
internal/          the whole runtime: core/ (store, config, mcp, schema, spa embed) +
                   modules/{profile,librarian,pm}
templates/         canonical prompt/text sources (librarian-system-prompt.txt generates the
                   librarian-operator agent) — inside the neutrality lint's scan scope
verify.sh          the librarian integration gate (throwaway scratch desk) — `make verify`
e2e/               end-to-end system-behaviour suite (throwaway desk, offline) — `make e2e`;
                   e2e/spa/ is the browser gate — real Chromium over `serve` — `make spa-verify`
sandbox/           sandbox-exec wrappers for the supervised write path
examples/          two manual walkthrough harnesses + their README; never in CI.
                   pm-walkthrough.sh is free/offline; agent-loop.sh makes REAL billed LLM calls
web/               the embedded SPA (Vite + Svelte + TypeScript + PocketBase JS SDK); `make build`
                   builds it into internal/core/spa/dist/ via go:embed; never committed
plugins/           the marketplace-distributed bundle (a marketplace install copies ONLY this)
  deskkit/         the only bundle: 7 skills, librarian-operator + pm-operator agents,
                   SessionStart hook, .mcp.json — authored in place, one generated file
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

**Nothing under `cmd/`, `internal/`, `templates/`, `e2e/`, `sandbox/`, `plugins/`, `kits/`, or `web/`
may hardcode a deployment identity** — no person, org, repo name, or bare issue reference (`#123`).
This is the product's core promise and the failure that rots worst: a hardcoded identity ships
inside a distributed binary/plugin and can't be pulled back. It is enforced in CI, not by
convention:

```
# scope = check-neutrality.mjs's SCAN_DIRS, recursively; docs/ and repo-root files are EXEMPT
node scripts/check-neutrality.mjs            # scans the shipped tree — FAILS on any hardcoded identity
node scripts/check-neutrality.mjs --self-test  # proves the scanner still detects a seeded violation
```

The scan is directory-scoped, so repo-root files (`README.md`, `install.sh`, `verify.sh`,
`go.mod`) are outside it — the module path `github.com/hsb3/deskkit` is
additionally token-allowlisted in `schema/neutrality-lint.allow` because a Go module path is
compile-time public API and cannot be templated.

The classic trip: a bare `#18`-style issue ref in a **Go comment or test** under `internal/` or
`cmd/` fails the lint. Write issue-free comments there. In `docs/`, the spec's `#N` references are
fine.

## Commands

The Makefile is the interface; run gate commands **bare, never piped** — a pipe (`| tail`) masks
the exit code and has let a failing gate through before (incident, 2026-07-17).

| Command | What it does |
|---|---|
| `make help` | List all targets (default goal) |
| `make setup` | `lefthook install` (git hooks) — there is no package-manager step |
| `make build` | Build the SPA (`web/`, via npm) then the `deskkit` binary (version-stamped), embedding the SPA dist via `go:embed` |
| `make test` | Fast unit tests: `go test ./...` at the repo root |
| `make check` | Repo gates: neutrality + self-test, kit-drift, scaffold frontmatter, persona drift, textfield-max, query-kind drift + self-test, doc-link integrity + self-test, shellcheck, actionlint, workflow SHA-pin drift + self-test, profile-root drift + self-test |
| `make verify` | Librarian integration gate — `verify.sh` (throwaway scratch desk) |
| `make e2e` | End-to-end system-behaviour suite — whole system (cold-start → profile → librarian → PM → surfaces → release-shaped) on a throwaway desk; offline, no LLM key (`e2e/e2e.sh`) |
| `make spa-verify` | SPA browser gate — drives the embedded SPA in a real Chromium against a throwaway desk, asserting against the bytes on disk as well as the DOM (`e2e/spa/run.sh`). Needs playwright installed once; **exits non-zero rather than skipping** when it is absent |
| `make package` | Informational no-op — this target generates nothing; the bundle's one generated file is written by `node scripts/check-persona-drift.mjs --write` |
| `make install` | Build + install the `deskkit` binary to `~/.local/bin` (override `PREFIX=`) |
| `node scripts/check-version-sync.mjs` | Assert root `VERSION` matches the shipped manifests |
| `make version-status` | Advisory (non-blocking): unreleased product changes since the last tag |
| `make release-prep` | Pre-tag gate (see order below) |

There is ONE Makefile, at the repo root — the old per-lane Makefile was folded into it, so
its targets are now plain root targets: `make build|test|vet|fmt|shellcheck|spa|serve|stop|gui|
sweep|patrol|propose-fix|findings|summary|adoption|orphans|uncollapsed|clean|media|
example-agent-loop` (`fmt` is the gofmt gate, also run in CI; `media` records the demo assets via
`scripts/record-media.sh`). That list plus the table above covers every target; `make help` is the
authoritative list. `apply-fix` is deliberately **not** a target — it's supervised-only,
run by hand, and every fix is reversible with `deskkit restore --by-path <path>`.
`make example-agent-loop` runs `examples/agent-loop.sh`, which makes REAL billed LLM calls and is
never part of CI; its free, offline sibling `examples/pm-walkthrough.sh` has no target — run it
directly. See `examples/README.md`.

The table above is `make` targets; `deskkit desks` and `deskkit config show|path|edit|set` are
plain binary subcommands (no store opened) — see "Configuration resolution" below for what they
show. `deskkit --help` groups the whole command menu into six sections instead of one alphabetic
list.

## Architecture notes

### Generated and copied artifacts — edit the source, not the copy

**The marketplace bundle is authored in place with one exception:
`plugins/deskkit/agents/librarian-operator.md` is GENERATED** (it says so in a marker at its
top — never hand-edit it). `make package` generates nothing and only says so. What needs care is
the handful of files that are *derived from a canonical source*:

| File | Source of truth | Guard |
|---|---|---|
| `plugins/deskkit/agents/librarian-operator.md` | `templates/librarian-system-prompt.txt` — regenerate with `node scripts/check-persona-drift.mjs --write` | `node scripts/check-persona-drift.mjs` (`make check`) |
| `internal/core/schema/profile.schema.yaml` | `schema/profile.schema.yaml` — `go:embed` can't reach outside the Go module, so the binary carries a copy | `TestProfileSchemaEmbeddedCopy_MatchesRepoRoot` (`make test`) |
| `internal/core/schema/references.yaml` | `schema/references.yaml` — same reason | `TestReferencesEmbeddedCopy_MatchesRepoRoot` (`make test`) |
| `kits/` tree | authored, but `kits.yaml` must match it | `node scripts/check-kits.mjs` |

Edit the repo-root `schema/` file first, then re-copy it into `internal/core/schema/`;
the guards are byte-for-byte and fail loudly on a one-sided edit. (The `go:embed`-can't-reach-out
constraint survives the module move: `schema/` is a sibling of `internal/`, not a child of it.)

### Architectural rules and their enforcing checks

| Rule | Enforced by |
|---|---|
| Identity-neutrality (shipped tree) | `scripts/check-neutrality.mjs` (+ `--self-test`) |
| Embedded schema copies stay byte-identical to `schema/` | `TestProfileSchemaEmbeddedCopy_MatchesRepoRoot` / `TestReferencesEmbeddedCopy_MatchesRepoRoot` (`make test`) |
| `VERSION` == shipped manifests | `scripts/check-version-sync.mjs` |
| `kits.yaml` == `kits/` tree | `scripts/check-kits.mjs` |
| `docs/development/specs/tool-surface.md` counts (CLI + gated MCP) match source (ADR 0016) | `TestToolSurfaceDoc_*` in `internal/core/mcp/tool_surface_doc_test.go` (`make test`) |
| Scaffold instruments carry conformant frontmatter | `scripts/check-scaffold-frontmatter.mjs` |
| Persona `librarian-operator` agent stays generated from the librarian prompt (ADR 0014/0015); PM surfaces are authored-in-place post-fold | `scripts/check-persona-drift.mjs` |
| The shipped bundle's authored artifacts (`.mcp.json` modules, agent `tools:`, skill tool refs, inventory) name only real modules/tools | `TestBundle*` in `internal/core/mcp/bundle_shape_test.go` (`make test`) |
| Content TextFields carry an explicit Max (ADR 0017) | `scripts/check-textfield-max.mjs` (+ `--self-test`) |
| Spec query-kind list == CLI/MCP registry (types.go ↔ spec §5.6 ↔ query.go switch) | `scripts/check-query-kinds.mjs` (+ `--self-test`) |
| Go tree stays gofmt-clean | `gofmt -l` via `make fmt` (CI go lane) |
| A tagged release has a CHANGELOG section | `scripts/check-changelog.mjs` (release gate) |
| Every workflow `uses:` stays SHA-pinned (no mutable `@vN` tag) | `scripts/check-workflow-pins.mjs` (+ `--self-test`) |
| Profile root (`_knowledge`) pinned identically in `schema/paths.yaml` and the Go constant | `scripts/check-profile-root.mjs` (+ `--self-test`) |
| Shell entry points stay lint-clean (install.sh, docker-entrypoint.sh, verify.sh, examples/*, sandbox/*, record-media, docker-smoke, e2e/*, e2e/spa/run.sh) | `shellcheck` via `make shellcheck` (CI + `make check`) |
| Every cited doc/media path on the published+shipped surface resolves (no dangling links; incident 2026-07-24) | `scripts/check-doc-links.mjs` (+ `--self-test`) |

### Order-sensitive chains

**`make release-prep`** runs in this order and aborts on the first failure — don't reorder:
1. clean working tree + on `main` (a release is cut from main only)
2. `check-version-sync` → `check-changelog` → `check-version-status` (cheap manifest/changelog gates first)
3. `make check` then `make test` (the expensive lanes last)
4. prints the tag/push commands — it never auto-tags.

**PocketBase bootstraps before cobra**: `Execute()` calls `Bootstrap()` — which
**creates the data dir** — before `RootCmd.Execute()` dispatches any `RunE`/`PreRunE`. Anything that
must *prevent* store creation has to run in `main()` before the app starts (the argv-scan location
guard and `init` do exactly this). Fail-closed behavior in `serve` paths needs a direct
`os.Exit(1)`, not a returned error (PocketBase discards serve-goroutine RunE errors).

## Configuration resolution

Every resolved field wins on one of five legs, in this order: **env > per-desk
`_knowledge/profile.*` > store settings > central config > built-in default**
(`internal/core/config/config.go`). The store leg is the `settings` collection — a migration-seeded
singleton row (`internal/modules/librarian/collections/0024_settings.go`) holding
`llm_provider`/`llm_model`/`llm_api_key`, written by the SPA's settings panel. The same row also
carries `sticky_finder` (`0025_settings_sticky_finder.go`), a browser preference no Go surface
reads — it rides the row because a per-desk preference has to outlive the browser that set it. It sits above the
central file because the store is per-desk while the file is machine-wide, and below the profile
because a desk's declared config still outranks runtime GUI state (decision: board task DESK-73,
no ADR number — post-0022 decisions are filed board-side without one). The API key
field is a PocketBase **hidden** TextField, so it is structurally absent from every API response
rather than filtered by hand; `llm_api_key_hint` carries the display suffix and is recomputed
server-side on save. Collection rules stay nil (superuser-only) — that IS the panel's auth posture.
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
- provider: `LLM_PROVIDER` env → `profile.models` → store `llm_provider` → central `llm.provider` →
  `anthropic` (default)
- model: `LLM_MODEL` env → `profile.models` → store `llm_model` → central `llm.model` → the
  built-in default model
- key: the env var named by `LLM_API_KEY_ENV` / `secrets_ref.llm_api_key` (or the per-provider
  default `ANTHROPIC_API_KEY` / `OPENAI_API_KEY` / `GEMINI_API_KEY`) → else the store's
  `llm_api_key` → else the central config's `llm.api_key`. The key never lives on the `Config`
  struct — it resolves at use time (`config.ResolveAPIKey` / `ResolveAPIKeySettings`) — so setting
  it once, via `deskkit config set llm.api_key <key>` OR the SPA settings panel, is enough to run
  `agent`/`chat` with zero env vars set.
- `LIBRARIAN_AUTONOMOUS_WRITES=true` gates `apply_fix` (checked at execution time)

**Editor hand-off.** `EDITOR_URL` env → `profile.preferences.editor_url` → unset. It is a URL
TEMPLATE, never a shell command — `{path}` (desk-relative) and `{abs}` (absolute) are substituted
and the browser renders an anchor, so nothing shells out and no route executes anything. A desk
that declares none simply gets no "open the body" control, which is also the hosted case.
`GET /desk/settings/resolved` reports it alongside `desk_root` (needed to expand `{abs}`).

**Public-mode serve.** `serve`'s auth posture is derived from every resolved `--http`/`--https`
bind address, never from a flag: a loopback bind (`127.0.0.1`/`localhost`/`::1`/empty) is today's
unauthenticated local UX, unchanged; anything else on either listener (a wildcard bind, a bare
`:PORT`, a routable IP, or a hostname that can't be proven loopback) makes the whole process
public mode, and an unprovable hostname classifies as public — fail closed. In public mode
`serve` refuses to open a listener at all unless it can *verify* an administrable superuser
exists after provisioning: it fatally creates the `PB_SUPERUSER_EMAIL`/`PB_SUPERUSER_PASSWORD`
account right there (rather than trusting a later non-fatal call) and re-counts superusers,
excluding the framework's own installer-placeholder row, which would otherwise satisfy a naive
count with no real account behind it. Setting exactly one of that env pair is a loud fatal error
in every mode. A self-contradictory `--origins` (a bare `*` mixed with explicit origins) also
refuses to start on a public bind. Public mode puts the two chat session routes
(`/desk/chat/stream` + `/desk/chat/reset`) behind `apis.RequireAuth` (401 with no token, 403
for a token from the wrong auth collection), leaves the loopback-only `/desk/bootstrap`
token-mint route unregistered entirely (the SPA shows a login form instead), serves the static
SPA shell without auth (shell loads, data doesn't — the admin-console stance),
and switches the CSRF check to strict same-origin; CORS drops the framework's default wildcard
unless an explicit `--origins` allowlist is set, in which case that allowlist is preserved (see
`docs/usage/deskkit-reference.md`'s "Browser session" section and `docs/pattern.md` for the full
model).

`deskkit desks` lists the desks this machine has a store for (marking which one the cwd
resolves to); `deskkit config show|path|edit|set` inspects/edits the central file above. Neither
opens a store. `deskkit --help` groups the command menu into six sections (Setup & config,
Inspect, Fix, Work graph, Agent, Admin) rather than one alphabetic list.

### No in-repo dogfooding (ruled 2026-07-22, "no dogfood")

This repo does not register its own MCP servers on itself — there is no root `.mcp.json` and
none should be added. The tools this repo builds (the `deskkit` binary and the `deskkit`
plugin bundle) are for coordinating *other* desks; that standard doesn't apply reflexively to the
repo that builds it, and the coordination tooling must live outside it — on a desk built to operate
on this repo, e.g. the paired executive desk (DESK-21 §0). In-repo verification instead
runs through `make verify` (`verify.sh`, a throwaway scratch desk), `make e2e`, and `make spa-verify`, all of
which stand up disposable desks rather than pointing the binary at this repo's own tree.
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
- **Toolchain floor:** Go `1.25` (PocketBase's `go.mod` floors it). Node is needed both to run the
  `scripts/*.mjs` gates and, now, to build the `web/` SPA that `make build` embeds into the
  binary — a plain `go build` (skipping the SPA step) still compiles and runs, serving a
  placeholder page at `/` instead of the SPA.

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

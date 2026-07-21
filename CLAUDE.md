# CLAUDE.md

This file guides coding agents working in this repo. For the project's canonical direction
(what it is, what's settled for 1.0.0) see [`docs/CHARTER.md`](docs/CHARTER.md); if that page and
this one ever disagree about direction, the charter wins.

## Project Overview

**desk-standard is two products over one shared schema**, all identity-neutral — nothing shipped
carries a person, org, repo, or issue number. You personalize by filling `_knowledge/profile.yaml`
(copied from `_knowledge/profile.example.yaml`), never by editing a shipped skill, template, or tool.

**Architecture — two lanes + one contract:**

- **`plugin/`** — a harness-pure TypeScript core behind a **stdio MCP server**
  (`plugin/mcp/server.ts`; tools `profile_get`, `profile_validate`, `template_render`,
  `knowledge_index`), wrapped as a Claude Code plugin with four skills (`desk-setup`,
  `conventions-standard`, `harvest-loop`, `brownfield-adoption`). "Harness-pure" = the core imports
  no harness/runtime APIs (enforced — see the critical rule).
- **`librarian/`** — **deskkit**, a single Go binary with an **embedded PocketBase** store. Indexes
  a desk, flags rule violations, and proposes/applies fixes under a record-original-first boundary
  (byte-exact reversible via `restore`). Surfaces over one tool core: **CLI**, an **MCP server**
  (`deskkit mcp-serve`), and a **chat TUI**. Admin console (`make gui`) serves PocketBase at
  `http://127.0.0.1:8090/_/`. It also carries the **PM module** (a document-gated work graph),
  feature-gated OFF by default.
- **`schema/`** — schema v1: the product-neutral contract both lanes read.

**Project structure** (every top-level entry annotated):

```
plugin/            TS lane — harness-pure core + MCP server, packaged as a Claude Code plugin
  core/            harness-pure domain library (profile, schema validation, templating, indexing)
  mcp/             stdio MCP server entry (server.ts)
  claude-plugin/   marketplace adapter: manifest, skills/, GENERATED mcp/server.js + schema copy
  desk-pm/         the desk-pm companion plugin (pm-operator agent + SessionStart hook)
  opencode/        frozen, unwired OpenCode spike — ships nothing in v1
librarian/         Go lane — the deskkit binary; embedded PocketBase; CLI/MCP/TUI; verify.sh gate
schema/            schema v1 — shared rule/structure source for both lanes
_knowledge/        your personalization root (profile.yaml + freeform background); real profile gitignored
docs/              specs, ADRs (docs/decisions/), the CHARTER, and using/developing guides
scripts/           repo-wide gate scripts (*.mjs) + record-media.sh
tests/             signpost only — suites live with their products; see tests/README.md
kits/ + kits.yaml  SOP template library + its drift-guarded manifest
_meta/             the working desk (HANDOFF, plans, briefings, research); tracked by default
.claude/           agent config: skills/, agents/, rules/, memory/, settings.json (tracked policy)
.github/           CI workflows + issue/PR templates + dependabot
Makefile           the canonical task interface — `make help` lists targets
VERSION            single source of truth for the release version (both products ship off it)
```

## The one rule that matters: identity-neutrality

**Nothing under `plugin/` or `librarian/` may hardcode a deployment identity** — no person, org,
repo name, or bare issue reference (`#123`). This is the product's core promise and the failure
that rots worst: a hardcoded identity ships inside a distributed binary/plugin and can't be pulled
back. It is enforced in CI, not by convention:

```
# scope = plugin/ + librarian/ recursively; docs/ and repo-root files are EXEMPT
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
| `make setup` | `bun install` (plugin) + `lefthook install` (git hooks) |
| `make build` | Build both lanes: plugin (`bun run build`) + librarian binary (version-stamped) |
| `make test` | Fast unit tests: plugin `bun test` (75) + librarian `go test ./...` (594) |
| `make check` | Repo gates: neutrality + self-test, kit-drift, prompt-drift, tool-surface drift + self-test, scaffold frontmatter, plugin core-purity, shellcheck, actionlint, workflow SHA-pin drift + self-test |
| `make verify` | Librarian integration gate — `librarian/verify.sh` (48 checks, throwaway scratch desk) |
| `make package` | Regenerate the marketplace bundle (`plugin/claude-plugin/` artifacts) |
| `make install` | Build + install the `deskkit` binary to `~/.local/bin` (override `PREFIX=`) |
| `node scripts/check-version-sync.mjs` | Assert root `VERSION` matches the shipped plugin manifests |
| `make version-status` | Advisory (non-blocking): unreleased product changes since the last tag |
| `make release-prep` | Pre-tag gate (see order below) |

The librarian lane has its own `librarian/Makefile` (`make -C librarian build|test|sweep|patrol`);
`apply-fix` is deliberately **not** a target — it's supervised-only, run by hand, and every fix is
reversible with `deskkit restore --by-path <path>`.

## Architecture notes

### Generated artifacts — never hand-edit

| File | Regenerate with | Guard |
|---|---|---|
| `plugin/claude-plugin/mcp/server.js` | `cd plugin && bun run package` (`make package`) | CI `git diff --exit-code` |
| `plugin/claude-plugin/schema/profile.schema.yaml` | same (copied from `schema/`) | CI `git diff --exit-code` |
| `plugin/claude-plugin/schema/references.yaml` | same (copied from `schema/`) | CI `git diff --exit-code` |
| `kits/` tree | authored, but `kits.yaml` must match it | `node scripts/check-kits.mjs` |

The marketplace install copies **only** `plugin/claude-plugin/`, so the bundled `server.js` must be
self-contained — that's why it's committed and drift-guarded.

### Architectural rules and their enforcing checks

| Rule | Enforced by |
|---|---|
| Identity-neutrality (shipped tree) | `scripts/check-neutrality.mjs` (+ `--self-test`) |
| `plugin/core` stays harness-pure (no harness imports) | `plugin` `bun run check:purity` → `scripts/check-core-purity.mjs` |
| `VERSION` == shipped manifests | `scripts/check-version-sync.mjs` |
| `kits.yaml` == `kits/` tree | `scripts/check-kits.mjs` |
| Prompt copies byte-identical (embed ↔ spec quote; ADR 0015) | `scripts/check-prompt-drift.mjs` |
| `docs/tool-surface.md` counts match source (ADR 0016) | `scripts/check-tool-surface.mjs` (+ `--self-test`; MCP gated counts by `TestToolSurfaceDoc_MCPCounts` on `make test`) |
| Scaffold instruments carry conformant frontmatter | `scripts/check-scaffold-frontmatter.mjs` |
| Persona bundle stays generated from its sources (ADR 0014) | `scripts/check-persona-drift.mjs` |
| Content TextFields carry an explicit Max (ADR 0017) | `scripts/check-textfield-max.mjs` (+ `--self-test`) |
| A tagged release has a CHANGELOG section | `scripts/check-changelog.mjs` (release gate) |
| Every workflow `uses:` stays SHA-pinned (no mutable `@vN` tag) | `scripts/check-workflow-pins.mjs` (+ `--self-test`) |
| Shell entry points stay lint-clean (install.sh, verify.sh, sandbox/*, record-media) | `shellcheck` (CI + `make check`) |

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

**Librarian store / profile discovery** (first match wins):
1. explicit `--dir` (overrides everything)
2. `DESK_ROOT` + `DESK_NAME` env vars
3. `_knowledge/profile.yaml` discovered by walk-up from cwd (a profile with `desk.name` +
   `root: "."` needs no env vars when you run `deskkit` inside the desk)
4. no `--dir` and a resolvable `DESK_NAME` → store at `$XDG_DATA_HOME/deskkit/<DESK_NAME>/`
5. unresolvable `DESK_NAME` and no `--dir` → **exit 1** (serve/migrate included)

**LLM provider/key** (only `agent`/`chat`/MCP-driven calls need it):
- provider: `LLM_PROVIDER` env → `profile.models` → `anthropic` (default)
- key: the env var named by `LLM_API_KEY_ENV` / `secrets_ref.llm_api_key` → else per-provider
  `ANTHROPIC_API_KEY` / `OPENAI_API_KEY` / `GEMINI_API_KEY`
- `LIBRARIAN_AUTONOMOUS_WRITES=true` gates `apply_fix` (checked at execution time)

## Cross-cutting gotchas

- **Store self-initializes** at the `requireConfig` choke point — a fresh store needs no manual
  `migrate up`. A new store-touching command routed through `requireConfig` inherits this.
- **Altering a shipped PocketBase collection: add a forward migration, never edit the applied one**
  (fresh stores only see the edit; existing stores need the new migration). Content-bearing text
  fields must set an explicit `Max` — a bare `TextField` silently caps at 5000 chars.
- **`docs/` spec + ADR paths are load-bearing** — `docs/pocket-librarian-v1-spec.md`,
  `docs/pm-system-v1-spec.md`, `docs/decisions/*` are cited from code, skills, and the neutrality
  allowlist. Don't move them.
- **Toolchain floors:** Go `1.25` (PocketBase's `go.mod` floors it), Bun `1.3.14`.

## Documentation

- **[`docs/CHARTER.md`](docs/CHARTER.md)** — canonical page (what it is, 1.0.0 direction, precedence rule).
- **[`docs/README.md`](docs/README.md)** — docs index, split Using vs Developing.
- **[`docs/decisions/`](docs/decisions/)** — ADRs (append-only; cited where they bind).
- **[`_meta/HANDOFF.md`](_meta/HANDOFF.md)** — session-to-session bridge: current standing + deep gotchas.
- **[`CHANGELOG.md`](CHANGELOG.md)** — what changed per release.

## Keeping this file current

Update CLAUDE.md in the **same change** as the code it describes. A named command, path, or test
that no longer exists is worse than silence — if you rename a target or a gate, fix it here too.

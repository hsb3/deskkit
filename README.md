# desk-standard

Profile-driven repo conventions and a self-repairing desk librarian, over one shared schema.

## What this is

Two products, one schema, one personalization model — nothing you install carries a name,
org, repo, or issue number:

- **`plugin/`** — the desk-standard plugin. A harness-pure core (profile loading, schema
  validation, `{{profile.…}}` substitution, `_knowledge/` indexing) behind a stdio MCP server
  (`profile_get`, `profile_validate`, `template_render`, `knowledge_index`), wrapped for Claude
  Code as a plugin with four skills (`desk-setup`, `conventions-standard`, `harvest-loop`,
  `brownfield-adoption`).
- **`librarian/`** — pocket-librarian, a single Go binary (embedded PocketBase) that indexes a
  desk's files, flags convention violations, and can propose + apply fixes under a
  record-original-first safety boundary — every applied fix is byte-exact reversible via
  `restore`. Exposes its tools as an MCP server (`restore` is deliberately CLI-only, and
  `apply_fix` is gated behind an env flag) and a CLI over one tool core.
- **`schema/`** — schema v1: the shared, product-neutral contract both `plugin/` and
  `librarian/` read as their rule/structure source.

You personalize by filling `_knowledge/profile.yaml` (copied from
`_knowledge/profile.example.yaml`) — never by editing a shipped skill, template, or tool.

## Scope of this build (v1)

**Claude Code only.** OpenCode support is deferred to a separate common-core fan-out build
(tracked in the workbench: `hsb3/dotfiles-agents-workbench#50`); `plugin/opencode/` holds a
frozen, unwired adapter spike kept as reference for that build (see its README) and ships nothing.

Harness versions tested: Claude Code `2.1.204` · Go `1.25.0` (pinned — PocketBase's own
`go.mod` floors it; a `go 1.23` directive will not compile the dependency graph) ·
Bun `1.3.14` / Node `26.5.0` (either runs `plugin/`).

Known, intentional gaps — not bugs:

- **OpenCode adapter** — not built (see above).
- **Hooks and agents surfaces** — the Claude plugin ships skills only; no `hooks/` or
  `agents/` definitions yet.

Marketplace packaging and schema distribution are now wired (see below); they are no longer gaps.

## Install as a marketplace plugin

This repo is its own Claude Code plugin marketplace (`.claude-plugin/marketplace.json`). From
any project:

```bash
claude plugin marketplace add hsb3/desk-standard
claude plugin install desk-standard@desk-standard
```

The install copies only `plugin/claude-plugin/` into the plugin cache, so the plugin is
self-contained: `bun run package` (in `plugin/`) bundles the MCP server and its npm + `plugin/core`
dependencies into the committed `plugin/claude-plugin/mcp/server.js` and copies the schema to
`plugin/claude-plugin/schema/profile.schema.yaml`. Both are generated artifacts — never
hand-edited; a CI drift guard regenerates and fails on any diff. For an installed plugin the
schema ships inside it (found by the same walk-up from the server module); running from source is
unchanged (walk-up to the repo `schema/`, overridable via `DESK_SCHEMA_PATH`).

For local development you can still point Claude Code straight at the source tree with
`claude --plugin-dir ./plugin/claude-plugin`.

## Quick start

```bash
# 1. Fill in your profile (never edit files under plugin/ or librarian/ to personalize)
cp _knowledge/profile.example.yaml _knowledge/profile.yaml
$EDITOR _knowledge/profile.yaml

# 2. Point Claude Code at the plugin
claude --plugin-dir ./plugin/claude-plugin
```

Steering a plain folder instead of the plugin scaffold? `pocket-librarian init [dir]`
(built in "Running the librarian" below) writes the minimal zero-export profile above —
desk name from the folder's basename — without the plugin's other placeholders; see
`docs/getting-started.md` §2.

Inside that session, the `desk-setup` skill scaffolds a new desk from your profile;
`conventions-standard` and `harvest-loop` run the standing checks and the periodic
improvement-log pass.

## Running the librarian

```bash
cd librarian
export DESK_ROOT=/path/to/your/desk
export DESK_NAME=my-desk
make build
make sweep    # index the desk tree
make patrol   # flag rule violations — dry-run, never writes
```

The store self-initializes on first run — `./pocket-librarian migrate up` is optional. The
exports above are needed here only because `make` runs from `librarian/`; with the binary on
your `PATH` and a `_knowledge/profile.yaml` in the desk (`desk.name` + `root: "."`), running
`pocket-librarian` from inside the desk needs no env vars. See `docs/getting-started.md` §4.

`apply-fix` is deliberately not a Makefile target — it's supervised-only, run by hand, and
every fix it applies can be reversed with `./pocket-librarian restore --by-path <path>`.

Beyond the CLI: `make gui` opens the embedded PocketBase admin console
(`http://127.0.0.1:8090/_/`) over the desk's index, and `mcp-serve` exposes the tool core to
agent sessions. LLM provider selection, API keys, console setup, and MCP wiring are covered
in `librarian/README.md`.

## Structure

```
plugin/
  core/             harness-pure TypeScript domain library
  mcp/              stdio MCP server (profile_get, profile_validate, template_render, knowledge_index)
  opencode/         frozen spike — not shipped in v1
  claude-plugin/    Claude Code adapter: manifest, .mcp.json, skills/
librarian/          pocket-librarian: Go binary, embedded PocketBase, MCP server + CLI
schema/             schema v1 — shared by plugin/ and librarian/
_knowledge/         your personalization root (profile + freeform background)
docs/               product specs, ADRs, and the getting-started / plugin / librarian guides
```

## Documentation

Docs split into two tracks — see **[docs/README.md](docs/README.md)** for the full index.

**Using it** — install and run:

- **[docs/getting-started.md](docs/getting-started.md)** — install, fill your profile, build the librarian, first sweep + patrol.
- **[docs/plugin-guide.md](docs/plugin-guide.md)** — the four skills as user journeys: when to reach for each, what you get.
- **[docs/librarian-guide.md](docs/librarian-guide.md)** — the daily loop: sweep → patrol → fix → byte-exact restore.
- `plugin/README.md`, `schema/README.md`, `librarian/README.md` — per-product operator detail.

**Developing it** — build, test, release:

- **[docs/development/](docs/development/)** — the contributor overview: build/test gates, media regeneration, and how to cut a release.
- `docs/pocket-librarian-v1-spec.md` — the librarian's product and technical spec.
- `docs/decisions/` — architecture decision records (interactive surface, multi-desk topology, store self-initialization, chat TUI, versioning policy).
- **[CHANGELOG.md](CHANGELOG.md)** — what changed in each release.
- `_meta/build-brief.md` — the build brief this repo was built from (repo shape, acceptance criteria, parallelism).
- `_meta/m-05-data-surfaces.md` — the profile / `_knowledge/` design and the neutrality lint.

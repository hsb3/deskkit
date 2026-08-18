# desk-standard

Profile-driven repo conventions and a self-repairing desk librarian, over one shared schema.

## What this is

Two products, one schema, one personalization model — nothing you install carries a name,
org, repo, or issue number:

- **The desk-standard plugin** (Claude Code) — stands up and maintains an executive desk.
  Four skills (`desk-setup`, `conventions-standard`, `harvest-loop`, `brownfield-adoption`)
  over a harness-pure core (profile loading, schema validation, `{{profile.…}}` substitution,
  `_knowledge/` indexing) behind a stdio MCP server (`profile_get`, `profile_validate`,
  `template_render`, `knowledge_index`).
- **deskkit** — a single Go binary (embedded PocketBase) that indexes a desk's files, flags
  convention violations, and can propose + apply fixes under a record-original-first safety
  boundary — every applied fix is byte-exact reversible via `restore`. One tool core surfaces
  as a CLI, an MCP server (`restore` is deliberately CLI-only; `apply_fix` is gated behind an
  env flag), and a chat TUI. The same binary carries the **PM module** — a document-gated work
  graph (items move through a rigid phase machine; a phase advance is refused until the
  document that phase requires validates), **on by default** — opt a desk out with
  `PM_ENABLED=false`. See [docs/usage/pm-guide.md](docs/usage/pm-guide.md) and the composed
  `desk-persona` plugin.
- **schema v1** — the shared, product-neutral contract both products read as their
  rule/structure source (`schema/`).

Both products are things you **install once and use in your own desks** — folders outside this
repo. You personalize a desk by filling its `_knowledge/profile.yaml` (copied from the
desk-setup scaffold's `_knowledge/profile.example.yaml`), never by editing a shipped skill,
template, or tool. This repo is not itself a desk; it ships no repo-root `_knowledge/`.

## Install

### The plugin

This repo is its own Claude Code plugin marketplace (`.claude-plugin/marketplace.json`). From
any project:

```bash
claude plugin marketplace add hsb3/desk-standard
claude plugin install desk-standard@desk-standard
```

The installed plugin is self-contained — its MCP server and schema ship inside the bundle, so
there is nothing else to wire up. (For local development you can point Claude Code straight at
the source tree with `claude --plugin-dir ./plugins/desk-standard`.)

### The `deskkit` binary

The release workflow publishes a prebuilt `deskkit` binary for macOS and Linux (amd64 + arm64)
on every `v*` tag — no Go toolchain needed.

**While this repo is private,** download the release asset with an authenticated `gh` (the
public `install.sh` one-liner below only works once the repo is public):

```bash
mkdir -p ~/.local/bin
os=$(uname -s | tr '[:upper:]' '[:lower:]')                 # darwin | linux
arch=$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')   # amd64 | arm64
gh release download --repo hsb3/desk-standard \
  --pattern "deskkit_*_${os}_${arch}" \
  --output ~/.local/bin/deskkit --clobber
chmod +x ~/.local/bin/deskkit
deskkit --version
```

`gh release download` with no tag picks the latest release; make sure `~/.local/bin` is on
your `PATH`.

**Once the repo is public,** the bundled installer does download + sha256-verify + install in
one step: `curl -fsSL https://raw.githubusercontent.com/hsb3/desk-standard/main/install.sh | bash`.
From a source checkout, `make install` builds and installs the version-stamped binary to
`~/.local/bin` — see [docs/development/install-and-build.md](docs/development/install-and-build.md).

## Use it in your desk

Everything below happens in **your desk** — a folder outside this repo — not in a checkout of
this repository.

**1. Stand up a desk.** In a Claude Code session with the plugin installed, ask for the
`desk-setup` skill: it scaffolds a conformant desk and fills `_knowledge/profile.yaml` with
your identifiers. Or start from a plain folder without the plugin:

```bash
cd /path/to/your/desk
deskkit init      # writes the minimal _knowledge/profile.yaml: desk name from the folder, root "."
```

**2. Sweep and patrol.** Run `deskkit` from inside the desk — it finds the profile by walking
up from your working directory, so there are no environment variables to export:

```bash
cd /path/to/your/desk
deskkit sweep     # index the desk tree (the store self-creates on first run)
deskkit patrol    # flag convention violations — a dry run; never writes your files
```

**3. Go deeper.** `deskkit chat` opens the TUI (needs an LLM key — see
[librarian/README.md](librarian/README.md)); the `conventions-standard` and `harvest-loop`
skills run the standing checks and the periodic improvement-log pass; the supervised
`propose-fix → apply-fix → restore` write loop is in
[docs/usage/librarian-guide.md](docs/usage/librarian-guide.md). The full walkthrough is
[docs/usage/getting-started.md](docs/usage/getting-started.md).

## Structure

Two similarly named top-level directories are a **source vs. distribution** split: `plugin/`
(singular) is the TypeScript *source* lane; `plugins/` (plural) holds the *built bundles* a
marketplace install actually copies — think `src/` vs `dist/`. The bundled
`plugins/desk-standard/mcp/server.js` and schema copy are generated by `bun run package`
(never hand-edited; a CI drift guard regenerates and fails on any diff).

```
plugin/             TS source — harness-pure core + stdio MCP server (the plugin's engine)
  core/             harness-pure TypeScript domain library
  mcp/              stdio MCP server (profile_get, profile_validate, template_render, knowledge_index)
  opencode/         frozen spike — not shipped in v1
plugins/            marketplace-distributed bundles (a marketplace install copies only these)
  desk-standard/    Claude Code adapter: manifest, .mcp.json, skills/, generated mcp/server.js
  desk-persona/     composed librarian+PM bundle: agents, PM skills, SessionStart hook
librarian/          deskkit source: Go binary, embedded PocketBase, MCP server + CLI + TUI
schema/             schema v1 — shared by plugin/ and librarian/
docs/               product specs, ADRs, and the getting-started / plugin / librarian guides
```

## Scope of this build (v1)

**Claude Code only.** OpenCode support is deferred to a separate common-core fan-out build
(tracked in #12, parked until ≥ v1.0.0); `plugin/opencode/` holds a frozen, unwired adapter
spike kept as reference for that build (see its README) and ships nothing.

Harness versions tested: Claude Code `2.1.204` · Go `1.25.0` (pinned — PocketBase's own
`go.mod` floors it; a `go 1.23` directive will not compile the dependency graph) ·
Bun `1.3.14` / Node `26.5.0` (either runs `plugin/`).

Known, intentional gaps — not bugs:

- **OpenCode adapter** — not built (see above).
- **Hooks and agents surfaces (`desk-standard` plugin)** — the `desk-standard` plugin itself
  ships skills only; no `hooks/` or `agents/` definitions. (The composed `desk-persona` plugin
  does ship the `librarian-operator` + `pm-operator` agents and a SessionStart briefing hook.)

## Developing

Building from source, running the gates, and the env-var / store-path lore live in the
developer track — start at [docs/development/](docs/development/) and
[docs/development/install-and-build.md](docs/development/install-and-build.md). The short
version, from a clone:

```bash
make setup    # bun install + git hooks
make build    # both lanes: plugin + version-stamped librarian binary
make test     # fast unit tests (bun test + go test)
make check    # repo gates (neutrality, drift guards, doc links, …)
make verify   # librarian integration gate on a throwaway scratch desk
```

The librarian lane also has its own `librarian/Makefile` (`make -C librarian build|test|sweep|patrol`)
for iterating on the binary against a desk named by `DESK_ROOT`/`DESK_NAME`; `apply-fix` is
deliberately not a target — it's supervised-only, run by hand, and every fix is reversible with
`deskkit restore --by-path <path>`. `make gui` there opens the embedded PocketBase admin
console (`http://127.0.0.1:8090/_/`).

## Documentation

The canonical page is **[docs/development/CHARTER.md](docs/development/CHARTER.md)** — what the project is and what's
settled for 1.0.0; if anything here disagrees with it, the charter wins. Agents working in the repo
start at **[CLAUDE.md](CLAUDE.md)**. Docs split into two tracks — see **[docs/README.md](docs/README.md)**
for the full index.

**Using it** — install and run:

- **[docs/usage/getting-started.md](docs/usage/getting-started.md)** — install, fill your profile, first sweep + patrol, meet the TUI.
- **[docs/usage/plugin-guide.md](docs/usage/plugin-guide.md)** — the four skills as user journeys: when to reach for each, what you get.
- **[docs/usage/librarian-guide.md](docs/usage/librarian-guide.md)** — the daily loop: sweep → patrol → fix → byte-exact restore.
- **[docs/usage/pm-guide.md](docs/usage/pm-guide.md)** — the PM work graph: enable it, the phase machine + gates, and the CLI / MCP / TUI / `desk-persona` plugin surfaces.
- `plugin/README.md`, `plugins/desk-persona/README.md`, `schema/README.md`, `librarian/README.md` — per-product operator detail.

**Developing it** — build, test, release:

- **[docs/development/](docs/development/)** — the contributor overview: build/test gates, media regeneration, and how to cut a release.
- `docs/development/specs/pocket-librarian-v1-spec.md` — the librarian's product and technical spec.
- `docs/development/specs/pm-system-v1-spec.md` — the PM system's product and technical spec (core+modules refactor, PM module, gates, surfaces, plugin).
- Architecture decision records (interactive surface, multi-desk topology, store self-initialization, chat TUI, versioning policy, kit port, Charm v2 stack, PM core+modules architecture) live on the project board as `DECISION` tasks, not in this repo; docs and code cite them as `ADR NNNN`.
- **[CHANGELOG.md](CHANGELOG.md)** — what changed in each release.

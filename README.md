# desk-standard

Profile-driven repo conventions and a self-repairing desk librarian, in one binary.

## What this is

One binary, one plugin, one schema, one personalization model — nothing you install carries a
name, org, repo, or issue number:

- **deskkit** — a single Go binary (embedded PocketBase) that is the whole runtime. It indexes a
  desk's files, flags convention violations, and can propose + apply fixes under a
  record-original-first safety boundary — every applied fix is byte-exact reversible via
  `restore`. Three modules feed one tool core: **profile** (personalization — `profile_get`,
  `profile_validate`, `template_render`, `knowledge_index`), **librarian** (the sweep → patrol →
  fix loop), and **PM** — a document-gated work graph (items move through a rigid phase machine;
  a phase advance is refused until the document that phase requires validates), **on by default**,
  opt a desk out with `PM_ENABLED=false`. The core surfaces as a CLI, an MCP server (`restore` is
  deliberately CLI-only; `apply_fix` is gated behind an env flag), a chat TUI, and a browser
  session page. See [docs/usage/pm-guide.md](docs/usage/pm-guide.md).
- **The `desk-persona` plugin** (Claude Code) — the agent-facing surface over that same binary:
  skills that stand up, check, evolve, and adopt an executive desk and drive the work graph, the
  `librarian-operator` and `pm-operator` agents, a SessionStart briefing hook, and one MCP mount.
  It ships no runtime of its own.
- **schema v1** — the shared, product-neutral rule/structure contract (`schema/`); the binary
  carries drift-guarded embedded copies of it.

Both are things you **install once and use in your own desks** — folders outside this repo. You
personalize a desk by filling its `_knowledge/profile.yaml` (copied from the desk-setup scaffold's
`_knowledge/profile.example.yaml`), never by editing a shipped skill, template, or tool. This repo
is not itself a desk; it ships no repo-root `_knowledge/`.

## Install

### The plugin

This repo is its own Claude Code plugin marketplace (`.claude-plugin/marketplace.json`), with one
plugin in it. From any project:

```bash
claude plugin marketplace add hsb3/desk-standard
claude plugin install desk-persona@desk-standard
```

The plugin drives the `deskkit` binary, so install that too (next section) and make sure it is on
your `PATH`. (For local development you can point Claude Code straight at the source tree with
`claude --plugin-dir ./plugins/desk-persona`.)

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

```
librarian/          deskkit source: Go binary, embedded PocketBase, profile/librarian/pm
                    modules, MCP server + CLI + TUI + web session page
plugins/            the marketplace-distributed bundle (a marketplace install copies only this)
  desk-persona/     the one bundle: skills, agents, SessionStart hook, .mcp.json —
                    authored in place, nothing generated
schema/             schema v1 — source of truth for the binary's embedded copies
docs/               product specs, the charter, and the getting-started / plugin / librarian guides
```

## Scope of this build (v1)

**Claude Code only.** OpenCode support is deferred to a separate common-core fan-out build
(tracked in #12, parked until ≥ v1.0.0); nothing for it ships today.

Harness versions tested: Claude Code `2.1.204` · Go `1.25.0` (pinned — PocketBase's own
`go.mod` floors it; a `go 1.23` directive will not compile the dependency graph). Node is needed
only to run the repo's `scripts/*.mjs` gates; nothing shipped is built with it.

## Developing

Building from source, running the gates, and the env-var / store-path lore live in the
developer track — start at [docs/development/](docs/development/) and
[docs/development/install-and-build.md](docs/development/install-and-build.md). The short
version, from a clone:

```bash
make setup    # git hooks (lefthook)
make build    # the version-stamped deskkit binary
make test     # fast unit tests (go test)
make check    # repo gates (neutrality, drift guards, doc links, …)
make verify   # librarian integration gate on a throwaway scratch desk
```

The binary also has its own `librarian/Makefile` (`make -C librarian build|test|fmt|sweep|patrol`)
for iterating against a desk named by `DESK_ROOT`/`DESK_NAME`; `apply-fix` is
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
- **[docs/usage/plugin-guide.md](docs/usage/plugin-guide.md)** — the desk skills as user journeys: when to reach for each, what you get.
- **[docs/usage/librarian-guide.md](docs/usage/librarian-guide.md)** — the daily loop: sweep → patrol → fix → byte-exact restore.
- **[docs/usage/pm-guide.md](docs/usage/pm-guide.md)** — the PM work graph: enable it, the phase machine + gates, and the CLI / MCP / TUI / `desk-persona` plugin surfaces.
- `plugins/desk-persona/README.md`, `schema/README.md`, `librarian/README.md` — operator detail per surface.

**Developing it** — build, test, release:

- **[docs/development/](docs/development/)** — the contributor overview: build/test gates, media regeneration, and how to cut a release.
- `docs/development/specs/pocket-librarian-v1-spec.md` — the librarian's product and technical spec.
- `docs/development/specs/pm-system-v1-spec.md` — the PM system's product and technical spec (core+modules refactor, PM module, gates, surfaces, plugin).
- Architecture decision records (interactive surface, multi-desk topology, store self-initialization, chat TUI, versioning policy, kit port, Charm v2 stack, PM core+modules architecture) live on the project board as `DECISION` tasks, not in this repo; docs and code cite them as `ADR NNNN`.
- **[CHANGELOG.md](CHANGELOG.md)** — what changed in each release.

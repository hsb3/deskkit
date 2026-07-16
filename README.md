# desk-standard

Profile-driven repo conventions and a self-repairing desk librarian, over one shared schema.

## What this is

Two products, one schema, one personalization model — nothing you install carries a name,
org, repo, or issue number:

- **`plugin/`** — the desk-standard plugin. A harness-pure core (profile loading, schema
  validation, `{{profile.…}}` substitution, `_knowledge/` indexing) behind a stdio MCP server
  (`profile_get`, `profile_validate`, `template_render`, `knowledge_index`), wrapped for Claude
  Code as a plugin with three skills (`desk-setup`, `conventions-standard`, `harvest-loop`).
- **`librarian/`** — pocket-librarian, a single Go binary (embedded PocketBase) that indexes a
  desk's files, flags convention violations, and can propose + apply fixes under a
  record-original-first safety boundary — every applied fix is byte-exact reversible via
  `restore`. Exposes its six tools as an MCP server and a CLI over one tool core.
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
- **Marketplace packaging** — wired for local development
  (`claude --plugin-dir ./plugin/claude-plugin`) only, not yet packaged for distribution.
- **Hooks and agents surfaces** — the Claude plugin ships skills only; no `hooks/` or
  `agents/` definitions yet.
- **Schema distribution** — the MCP server locates `schema/profile.schema.yaml` by walking
  up from the repo; a distributed build must ship or embed the schema instead.

## Quick start

```bash
# 1. Fill in your profile (never edit files under plugin/ or librarian/ to personalize)
cp _knowledge/profile.example.yaml _knowledge/profile.yaml
$EDITOR _knowledge/profile.yaml

# 2. Point Claude Code at the plugin
claude --plugin-dir ./plugin/claude-plugin
```

Inside that session, the `desk-setup` skill scaffolds a new desk from your profile;
`conventions-standard` and `harvest-loop` run the standing checks and the periodic
improvement-log pass.

## Running the librarian

```bash
cd librarian
export DESK_ROOT=/path/to/your/desk
export DESK_NAME=my-desk
make build && ./pocket-librarian migrate up
make sweep    # index the desk tree
make patrol   # flag rule violations — dry-run, never writes
```

`apply-fix` is deliberately not a Makefile target — it's supervised-only, run by hand, and
every fix it applies can be reversed with `./pocket-librarian restore --by-path <path>`. See
`librarian/README.md`.

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
docs/               build brief, product specs, data-surface design
```

## Source of truth

- `docs/build-brief.md` — the build brief this repo was built from (repo shape, acceptance
  criteria, parallelism).
- `docs/pocket-librarian-v1-spec.md` — the librarian's product and technical spec.
- `docs/m-05-data-surfaces.md` — the profile / `_knowledge/` design and the neutrality lint.
- `plugin/README.md`, `schema/README.md`, `librarian/README.md` — per-product detail.

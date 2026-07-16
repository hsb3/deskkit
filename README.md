# desk-standard

The desk-standard plugin (Claude Code **and** OpenCode) and **pocket-librarian**, in one repo over a shared schema.

> **Status: scaffolding.** This repo was stood up from a gated build brief; the crew builds
> `plugin/` and `librarian/` from `docs/build-brief.md`. The directory skeleton and the
> source-of-truth docs are in place; the code is not built yet.

## What this is

Two products that share one schema and one identity-neutral personalization model:

- **`plugin/`** — the desk-standard plugin, shipped in **both** harness formats. Authored once
  where possible (skill/prompt prose, the MCP tool contracts, the `core/` domain logic), with two
  thin harness adapters. Principle: *port the capability, not the implementation.*
  - `core/` — shared domain logic; imports neither harness.
  - `mcp/` — the shared tool boundary (MCP); both formats consume it.
  - `opencode/` — OpenCode adapter (TypeScript module on Bun).
  - `claude-plugin/` — Claude Code adapter (declarative `.claude-plugin/plugin.json` bundle).
- **`librarian/`** — pocket-librarian: a single Go binary that indexes and repairs desk files
  under a binding safety boundary. It exposes its tools as **an MCP server _and_ a CLI** over one
  tool core (the `outlook-mcp` dual-surface pattern) — the MCP server is how the plugin calls the
  librarian's capabilities.
- **`schema/`** — schema v1, shared by both products, including the personalization profile block.
- **`_knowledge/`** — the personalization root: a structured `profile.{yaml,json,md}` the system
  substitutes into prompts, plus freeform background the agent reads. See `docs/m-05-data-surfaces.md`.

## Identity-neutral by construction

Nothing about a person, org, repo, or issue is hardcoded. All personalization comes from
`_knowledge/` (never from code); a required CI lint fails the build on any hardcoded
name/org/repo/issue token. The shipped artifacts carry no personal identifiers.

## Build source of truth

- `docs/build-brief.md` — the W4 build brief (repo shape, dual-format contract, acceptance
  criteria, parallelism, punch-list). **Build from this.**
- `docs/pocket-librarian-v1-spec.md` — the librarian product & technical spec.
- `docs/m-05-data-surfaces.md` — the profile + `_knowledge/` design and the neutrality lint.

## Structure

```
plugin/{core,mcp,opencode,claude-plugin}/   librarian/   schema/   _knowledge/   docs/
```

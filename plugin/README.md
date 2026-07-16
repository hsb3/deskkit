# desk-standard plugin

The plugin-side implementation of the shared data-surface loader (D7) and its MCP surface.

- **`core/`** — a harness-pure TypeScript domain library: profile discovery/loading, schema-v1
  validation, `{{profile.…}}/{{env.…}}` substitution, and the `_knowledge/` index. It imports no
  MCP / OpenCode / Claude-hook / Bun-specific APIs (enforced by `scripts/check-core-purity.mjs`).
  It mirrors the Go reference in `librarian/internal/config`.
- **`mcp/`** — a thin stdio MCP server exposing exactly four tools (`profile_get`,
  `profile_validate`, `template_render`, `knowledge_index`) whose schemas + behavior are defined
  once in `core/` and imported here.

The `claude-plugin/` adapter wraps the same `core/` seam. **Scope change (2026-07-16, Henry):**
v1 ships Claude Code only — `opencode/` holds a frozen, unwired adapter spike (see its own
README), superseded by a separate common-core fan-out build tracked at
`hsb3/dotfiles-agents-workbench#50`.

## Commands (run from `plugin/`)

```sh
bun install                       # deps (bun; obeys the 7-day supply-chain cooldown)
bun test                          # unit tests
node scripts/check-core-purity.mjs  # AC2 core-purity guard
bun mcp/server.ts                 # run the MCP server under Bun
bun run build && node dist/mcp/server.js   # or compiled, under Node >=20 (dist/ is gitignored)
```

## Schema location

The validator loads `schema/profile.schema.yaml` (the repo's single source). At runtime it is
found via `DESK_SCHEMA_PATH`, else a walk-up from the module dir, else a walk-up from cwd.
**Packaging follow-up:** a distributed build must ship the schema alongside the plugin (or embed
it) — the walk-up only finds it when running inside the desk-standard repo.

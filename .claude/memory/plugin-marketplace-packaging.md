---
name: plugin-marketplace-packaging
description: "How Claude Code marketplace install constrains desk-standard packaging, and the bundle solution that landed for"
metadata: 
  node_type: memory
  type: project
  originSessionId: 839511c5-cb29-40d9-a20f-8b1272817785
---

Claude Code marketplace install facts (verified 2026-07-16, docs + live install):

- `claude plugin install` copies ONLY the plugin source dir to `~/.claude/plugins/cache/<marketplace>/<plugin>/<version>/`; `${CLAUDE_PLUGIN_ROOT}` points there. Paths traversing outside the plugin root (`../…`) break after install.
- No npm/bun install runs at install time — deps must be bundled, vendored, or hook-installed.
- Symlinks inside the plugin dir pointing elsewhere *within the same marketplace repo* get content-copied into the cache; targets outside the marketplace are skipped.
- Private-repo marketplaces work with existing gh/git auth (background auto-update caveat: HTTPS credential helpers disabled — SSH keys fine).

desk-standard's landed shape (#9, commit 455769c): `bun run package` in `plugin/` emits a self-contained node-target bundle `plugin/claude-plugin/mcp/server.js` (core + npm deps inlined, deterministic — proven byte-identical macOS vs linux CI) plus a schema copy at `plugin/claude-plugin/schema/profile.schema.yaml`, which the existing walk-up (`plugin/core/schema.ts defaultSchemaPath`) finds from the cache with zero code change. Committed generated artifacts are drift-guarded in ci.yml. `marketplace.json` owner identity is deliberately outside the neutrality-lint surface (recorded in docs/m-05-data-surfaces.md).

Consumed by [[#7 curl installer]] and #12 dual-format fan-out: the fan-out production step should generalize `bun run package`, not invent a new one.

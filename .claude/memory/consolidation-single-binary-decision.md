---
name: consolidation-single-binary-decision
description: "Owner ruling to collapse the two plugins/two MCP servers into ONE, everything on the Go deskkit binary — reverses ADR 0016; driven by early-user 'too confusing' feedback"
metadata:
  node_type: memory
  type: project
  originSessionId: 688b2df2-944b-44e6-95d9-abe6d3e20436
  modified: 2026-08-18T09:51:51.319Z
---

Early user feedback (2026-07-23) was "the plugin is super-confusing — why two plugins and two MCP
servers? just make it one." Henry ruled two design decisions in response:

1. **Consolidation target = "everything on deskkit."** Move the 4 TS profile tools (`profile_get`,
   `profile_validate`, `template_render`, `knowledge_index`) INTO the Go binary; retire the TS MCP
   server (`plugin/mcp/server.ts` → `plugins/desk-standard/mcp/server.js`); ship ONE plugin with ONE
   MCP server (`deskkit mcp-serve`, `MCP_MODULES=profile,librarian,pm`). This **reverses ADR 0016**
   (which chose to keep TS pure and reach librarian via a designed proxy) and **withdraws**
   `docs/development/ts-proxy-design.md` → needs a superseding ADR. Rationale: the Go binary is
   required for all real value (librarian/PM/data browser/store) anyway, so a separate TS server
   buys the user nothing. This **moots #199** (ts-proxy doc correction) and the unfilled ts-proxy
   build; **simplifies #12** (OpenCode fan-out — one Go server to expose).

2. **Central config stores the actual API key.** New machine-local `~/.config/deskkit/config.yaml`
   (perms 0600, never committed) holds `llm.provider` + `llm.model` + `llm.api_key` (value).
   Precedence becomes env > per-desk `_knowledge/profile.*` > central config > default. Keep the
   `Config` struct secret-free — have `resolveAPIKey` (provider/adapter.go) read the central file's
   key after env, before failing loud.

**Why:** owner architecture ruling — record so it survives a clear and isn't re-litigated.

**How to apply:** BOTH halves are now SHIPPED (2026-08-18): the collapse landed as PR #240
(DESK-48 — profile tools in Go, one bundle at `plugins/desk-persona/`, `plugin/` deleted) and the
central config as PR #241 (DESK-50 — `$XDG_CONFIG_HOME/deskkit/config.yaml`, 0600, precedence as
ruled). The superseding ADR is **0022 (Accepted 2026-07-23)**, on the board as **DESK-43**. Do not
re-open the two-plugin/ts-proxy architecture, and do not treat `plugin/` paths in older docs/memory
as live — they resolve only in git history.
Related: [[plugin-marketplace-packaging]] (the constraints the two-bundle split originally solved).

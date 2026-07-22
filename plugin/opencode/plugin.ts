// D4 / W2a — the thin OpenCode adapter. This plugin's capability is a shared MCP tool surface
// (four tools: profile_get / profile_validate / template_render / knowledge_index, defined once
// in plugin/core and served by plugin/mcp/server.ts) plus three skills. The Claude Code adapter
// (plugins/desk-standard) ships ZERO lifecycle hooks — it registers the MCP server via .mcp.json
// and auto-discovers skills. Porting the capability (not the implementation) therefore means the
// OpenCode adapter ALSO ships no behaviour-changing hooks: it registers the SAME MCP server via
// OpenCode's `config` hook (the in-process equivalent of .mcp.json — "translate registration
// format only", OPENCODE_TO_CLAUDE_CODE_GUIDE.md:38), and skills are copied into the OpenCode
// skills dir by the install path (scripts/install-opencode.mjs). No tool is re-implemented as an
// OpenCode custom tool; no capability is double-registered (AC3, GUIDE:236).
//
// This module is a Bun-native OpenCode plugin: it runs in-process under Bun straight from TS. The
// `@opencode-ai/plugin` import is TYPE-ONLY (erased at runtime), so the repo does not depend on
// that package; OpenCode provides it in its own runtime. A local minimal type shim
// (opencode-plugin.d.ts) satisfies the repo typecheck.

import type { Config, Plugin } from "@opencode-ai/plugin";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

/** The MCP server key — identity-neutral, matching the Claude adapter's .mcp.json entry. */
export const MCP_SERVER_NAME = "desk-standard";

/** Env var (identity-neutral) pointing at the installed `plugin/` root, for out-of-tree loads. */
export const PLUGIN_ROOT_ENV = "DESK_STANDARD_PLUGIN_ROOT";

/**
 * A structural view of just the slice of the OpenCode config this adapter touches, so the pure
 * registration helper is testable without importing the SDK's full Config type.
 */
export type McpLocalEntry = {
  type: "local";
  command: string[];
  enabled?: boolean;
  environment?: Record<string, string>;
};
export type McpConfigLike = {
  mcp?: Record<string, McpLocalEntry | { type: "remote"; url: string }>;
};

/**
 * resolvePluginRoot returns the absolute path to the `plugin/` directory (parent of `opencode/`).
 * Precedence: the `DESK_STANDARD_PLUGIN_ROOT` env override (for a plugin loaded from outside the
 * repo, e.g. copied into ~/.config/opencode/plugins/), else the location of this module — which is
 * correct whenever OpenCode loads the adapter by its in-repo path (the recommended install).
 */
export function resolvePluginRoot(env: NodeJS.ProcessEnv = process.env): string {
  const override = env[PLUGIN_ROOT_ENV];
  if (typeof override === "string" && override.trim() !== "") return override.trim();
  // opencode/plugin.ts -> opencode/ -> plugin/
  return dirname(dirname(fileURLToPath(import.meta.url)));
}

/**
 * mcpServerCommand builds the argv that launches the shared MCP server, mirroring the Claude
 * adapter's `.mcp.json` ("bun <root>/mcp/server.ts"). Bun resolves the server's node_modules by
 * walking up from the server file, so the command must point at the file inside the repo.
 */
export function mcpServerCommand(pluginRoot: string): string[] {
  return ["bun", join(pluginRoot, "mcp", "server.ts")];
}

/**
 * registerMcpServer registers the shared MCP server into an OpenCode config IN PLACE, idempotently.
 * Returns true if it added the entry, false if a server of that name already existed (never
 * double-registers — AC3). Pure and side-effect-free beyond the passed config object.
 */
export function registerMcpServer(
  config: McpConfigLike,
  name: string,
  command: string[],
): boolean {
  if (!config.mcp) config.mcp = {};
  if (config.mcp[name]) return false; // already registered — respect the user's / another install's entry
  config.mcp[name] = { type: "local", command, enabled: true };
  return true;
}

/**
 * The OpenCode plugin. Its only hook is `config`: it registers the shared MCP server so the four
 * tools are reachable in an OpenCode session without a manual config edit. No other lifecycle
 * behaviour — the capability lives in core/mcp (tools) and the skills (instructional), exactly as
 * on the Claude side. Fail-open: a registration error must never wedge session startup.
 */
export const DeskStandard: Plugin = async () => {
  const command = mcpServerCommand(resolvePluginRoot());
  return {
    config: async (config: Config) => {
      try {
        registerMcpServer(config as McpConfigLike, MCP_SERVER_NAME, command);
      } catch {
        // fail-open: never block startup on MCP registration
      }
    },
  };
};

export default DeskStandard;

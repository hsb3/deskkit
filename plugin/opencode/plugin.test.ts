import { test, expect } from "bun:test";
import {
  DeskStandard,
  MCP_SERVER_NAME,
  PLUGIN_ROOT_ENV,
  mcpServerCommand,
  registerMcpServer,
  resolvePluginRoot,
  type McpConfigLike,
} from "./plugin.js";

test("resolvePluginRoot honors the env override", () => {
  expect(resolvePluginRoot({ [PLUGIN_ROOT_ENV]: "/opt/desk/plugin" })).toBe("/opt/desk/plugin");
  expect(resolvePluginRoot({ [PLUGIN_ROOT_ENV]: "  /trimmed/plugin  " })).toBe("/trimmed/plugin");
});

test("resolvePluginRoot falls back to this module's plugin/ dir when no override", () => {
  const root = resolvePluginRoot({});
  // opencode/plugin.ts -> plugin/ ; the mcp server must sit under it.
  expect(root.endsWith("/plugin")).toBe(true);
});

test("mcpServerCommand mirrors the Claude .mcp.json (bun <root>/mcp/server.ts)", () => {
  expect(mcpServerCommand("/opt/desk/plugin")).toEqual([
    "bun",
    "/opt/desk/plugin/mcp/server.ts",
  ]);
});

test("registerMcpServer adds a local entry and reports true", () => {
  const config: McpConfigLike = {};
  const added = registerMcpServer(config, MCP_SERVER_NAME, ["bun", "/p/mcp/server.ts"]);
  expect(added).toBe(true);
  expect(config.mcp?.[MCP_SERVER_NAME]).toEqual({
    type: "local",
    command: ["bun", "/p/mcp/server.ts"],
    enabled: true,
  });
});

test("registerMcpServer never double-registers an existing name (AC3)", () => {
  const existing = { type: "remote" as const, url: "https://elsewhere.example/mcp" };
  const config: McpConfigLike = { mcp: { [MCP_SERVER_NAME]: existing } };
  const added = registerMcpServer(config, MCP_SERVER_NAME, ["bun", "/p/mcp/server.ts"]);
  expect(added).toBe(false);
  expect(config.mcp?.[MCP_SERVER_NAME]).toBe(existing); // untouched
});

test("registerMcpServer preserves other servers", () => {
  const other = { type: "local" as const, command: ["node", "x.js"] };
  const config: McpConfigLike = { mcp: { other } };
  registerMcpServer(config, MCP_SERVER_NAME, ["bun", "/p/mcp/server.ts"]);
  expect(config.mcp?.other).toBe(other);
  expect(config.mcp?.[MCP_SERVER_NAME]).toBeDefined();
});

test("the plugin exposes only a config hook and registers the shared server", async () => {
  const hooks = await DeskStandard({ directory: "/x", worktree: "/x" });
  // No behaviour-changing lifecycle hooks — port parity with the Claude adapter (zero hooks).
  expect(Object.keys(hooks)).toEqual(["config"]);
  const config: McpConfigLike = {};
  await hooks.config?.(config as never);
  const entry = config.mcp?.[MCP_SERVER_NAME];
  expect(entry).toBeDefined();
  expect((entry as { type: string }).type).toBe("local");
  const command = (entry as { command: string[] }).command;
  expect(command[0]).toBe("bun");
  expect(command[1]?.endsWith("/mcp/server.ts")).toBe(true);
});

test("the config hook is fail-open on a hostile config object", async () => {
  const hooks = await DeskStandard({ directory: "/x", worktree: "/x" });
  const frozen = Object.freeze({ mcp: Object.freeze({}) }) as McpConfigLike;
  // Mutating a frozen object throws in strict mode; the hook must swallow it, not reject.
  await expect(hooks.config?.(frozen as never)).resolves.toBeUndefined();
});

// Behavioral test for the TS MCP server (createServer). Mirrors the Go side
// (librarian/internal/core/mcp/server_test.go TestExposedTools_GateComposition /
// TestInputSchemaMap_MatchesStructs): drive the server over a real MCP transport pair
// instead of importing TOOLS directly, so the test exercises the actual
// registration/dispatch path (ListToolsRequestSchema + CallToolRequestSchema handlers),
// not just the core tool registry.

import { test, expect } from "bun:test";
import { mkdtempSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { InMemoryTransport } from "@modelcontextprotocol/sdk/inMemory.js";
import { Client } from "@modelcontextprotocol/sdk/client/index.js";
import { createServer } from "./server.ts";

const EXPECTED_TOOL_NAMES = ["profile_get", "profile_validate", "template_render", "knowledge_index"];

/** Spin up a linked in-memory client/server pair and hand back both, connected. */
async function connectedPair(ctx?: Parameters<typeof createServer>[0]) {
  const server = createServer(ctx);
  const client = new Client({ name: "server-test-client", version: "0" }, { capabilities: {} });
  const [clientTransport, serverTransport] = InMemoryTransport.createLinkedPair();
  await Promise.all([server.connect(serverTransport), client.connect(clientTransport)]);
  return { server, client };
}

test("listTools exposes exactly the four tools, each with a description and inputSchema", async () => {
  const { server, client } = await connectedPair();
  try {
    const { tools } = await client.listTools();
    expect(new Set(tools.map((t) => t.name))).toEqual(new Set(EXPECTED_TOOL_NAMES));
    expect(tools).toHaveLength(EXPECTED_TOOL_NAMES.length);
    for (const t of tools) {
      expect(typeof t.description).toBe("string");
      expect((t.description ?? "").length).toBeGreaterThan(0);
      expect(typeof t.inputSchema).toBe("object");
      expect(t.inputSchema).not.toBeNull();
    }
  } finally {
    await client.close();
    await server.close();
  }
});

test("callTool on an unknown tool name returns an isError result naming the tool, not a throw", async () => {
  const { server, client } = await connectedPair();
  try {
    const result = await client.callTool({ name: "nonexistent-tool", arguments: {} });
    expect(result.isError).toBe(true);
    const content = result.content as Array<{ type: string; text: string }>;
    expect(content[0]?.type).toBe("text");
    expect(content[0]?.text).toContain("unknown tool: nonexistent-tool");
  } finally {
    await client.close();
    await server.close();
  }
});

test("callTool round-trips knowledge_index to a non-error result (hermetic cwd, no _knowledge above it)", async () => {
  // knowledge_index never throws (a missing root yields an empty index rather than an error —
  // see plugin/core/tools.ts), so it is safe to call without any fixture desk. Pin cwd to a
  // freshly created temp dir so profile/knowledge discovery cannot walk up into this repo's
  // real _knowledge/ folder; the result must be deterministic and error-free.
  const cwd = mkdtempSync(join(tmpdir(), "desk-standard-mcp-server-test-"));
  const { server, client } = await connectedPair({ cwd });
  try {
    const result = await client.callTool({ name: "knowledge_index", arguments: {} });
    expect(result.isError).not.toBe(true);
    const content = result.content as Array<{ type: string; text: string }>;
    const parsed = JSON.parse(content[0]?.text ?? "null");
    expect(parsed).toEqual({ root: "", budget: 65536, fileCount: 0, bytesIncluded: 0, entries: [] });
  } finally {
    await client.close();
    await server.close();
  }
});

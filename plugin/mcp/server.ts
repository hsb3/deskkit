// D3 — stdio MCP server over the harness-pure core (plugin/core). This is a THIN adapter: it
// declares no tool schema and implements no tool logic. It imports TOOLS from core (the single
// source, AC3), advertises each tool's inputSchema verbatim, and dispatches CallTool to the
// core handler. A thrown handler error becomes an MCP isError RESULT (not a protocol crash),
// mirroring the Go outbound server's jsonContent posture (librarian/internal/mcp/server.go).
//
// Runnable directly under Bun (`bun mcp/server.ts`) and, after `bun run build` (or `tsc`),
// under Node as `node dist/mcp/server.js` (dist/ is gitignored — see the handoff VERIFICATION
// block). It uses only the stable low-level Server + setRequestHandler API, which is unchanged
// across @modelcontextprotocol/sdk 1.x.

import { Server } from "@modelcontextprotocol/sdk/server/index.js";
import { StdioServerTransport } from "@modelcontextprotocol/sdk/server/stdio.js";
import {
  CallToolRequestSchema,
  ListToolsRequestSchema,
} from "@modelcontextprotocol/sdk/types.js";
import { TOOLS, type ToolContext } from "../core/index.js";

const SERVER_NAME = "desk-standard-plugin"; // identity-neutral: no person/org/repo
const SERVER_VERSION = "0.1.0";

export function createServer(ctx?: ToolContext): Server {
  const server = new Server(
    { name: SERVER_NAME, version: SERVER_VERSION },
    { capabilities: { tools: {} } },
  );

  server.setRequestHandler(ListToolsRequestSchema, async () => ({
    tools: TOOLS.map((t) => ({
      name: t.name,
      description: t.description,
      inputSchema: t.inputSchema,
    })),
  }));

  server.setRequestHandler(CallToolRequestSchema, async (req) => {
    const tool = TOOLS.find((t) => t.name === req.params.name);
    if (!tool) {
      return {
        isError: true,
        content: [{ type: "text" as const, text: `unknown tool: ${req.params.name}` }],
      };
    }
    try {
      const result = tool.handler((req.params.arguments ?? {}) as never, ctx);
      return { content: [{ type: "text" as const, text: JSON.stringify(result) }] };
    } catch (e) {
      // Tool-execution failures surface as isError results so the model sees the failure text
      // (including template_render's full missing-placeholder offender list), not a crash.
      const text = e instanceof Error ? e.message : String(e);
      return { isError: true, content: [{ type: "text" as const, text }] };
    }
  });

  return server;
}

export async function main(): Promise<void> {
  const server = createServer();
  const transport = new StdioServerTransport();
  await server.connect(transport);
}

// Run when invoked as the entry point (works under both Bun and Node ESM).
if (import.meta.url === `file://${process.argv[1]}`) {
  main().catch((err) => {
    process.stderr.write(`desk-standard-plugin MCP server fatal: ${String(err)}\n`);
    process.exit(1);
  });
}

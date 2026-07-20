#!/usr/bin/env bash
# Manual stdio JSON-RPC harness for the desk-standard plugin's MCP server
# (plugin/claude-plugin/mcp/server.js). Written for a manual protocol pass when a live
# Claude Code session does not have this plugin's MCP tools natively connected (verify with
# ToolSearch first — see docs/tool-surface.md "How the counts were derived" for the sibling
# pattern this mirrors against the librarian's MCP server).
#
# The server's tool handlers resolve `_knowledge/profile.*` by walking up from the SERVER
# PROCESS's cwd (server.ts calls createServer() with no ctx, so cwdOf(ctx) falls back to
# process.cwd()) — not from the caller's cwd. This script cd's into the given desk root
# before exec'ing node so profile/knowledge discovery resolves against that desk.
#
# Usage:
#   plugin/scripts/verify-mcp-protocol.sh <desk-root> < requests.jsonl
#   printf '%s\n' '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}' \
#     | plugin/scripts/verify-mcp-protocol.sh /path/to/desk
#
# Requests are newline-delimited JSON-RPC, forwarded verbatim on stdin. A real client also
# sends `initialize` then `notifications/initialized` before any `tools/call` — see
# docs/tool-surface.md for the exact framing. Responses print to stdout as the server emits
# them (newline-delimited JSON), server diagnostics go to stderr.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SERVER="$SCRIPT_DIR/claude-plugin/mcp/server.js"

DESK_ROOT="${1:-$(pwd)}"

if [[ ! -f "$SERVER" ]]; then
  echo "verify-mcp-protocol: server not found at $SERVER" >&2
  exit 1
fi

if [[ ! -d "$DESK_ROOT" ]]; then
  echo "verify-mcp-protocol: desk root not found at $DESK_ROOT" >&2
  exit 1
fi

cd "$DESK_ROOT"
exec node "$SERVER"

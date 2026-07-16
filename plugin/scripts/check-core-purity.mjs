#!/usr/bin/env node
// AC2 core-purity guard. Scans plugin/core/ non-test source and fails (exit 1) if it imports
// the OpenCode plugin API, Claude hook payload types, the MCP SDK, or any Bun-specific API.
// core/ must be a harness-pure domain library the adapters (mcp/opencode/claude-plugin) wrap.
//
// Runs under plain Node (no TS/bun needed): `node scripts/check-core-purity.mjs`.

import { readdirSync, readFileSync, statSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const coreDir = join(dirname(fileURLToPath(import.meta.url)), "..", "core");

// Each rule: a label + a RegExp that, if matched on a non-test line, is a violation.
const RULES = [
  { label: "OpenCode plugin API import", re: /@opencode-ai(\/[\w.-]+)?/ },
  { label: "MCP SDK import (belongs in mcp/, not core/)", re: /@modelcontextprotocol\/sdk/ },
  { label: "Claude hook payload types import", re: /@anthropic-ai\/(claude|sdk-hooks)|claude-code[/-]hook/i },
  { label: "Bun global (Bun.*)", re: /(^|[^A-Za-z0-9_])Bun\./ },
  { label: "bun: builtin import (bun:test excluded)", re: /["']bun:(?!test)[\w.-]+["']/ },
];

function tsFiles(dir) {
  const out = [];
  for (const name of readdirSync(dir)) {
    const abs = join(dir, name);
    if (statSync(abs).isDirectory()) {
      out.push(...tsFiles(abs));
    } else if (name.endsWith(".ts") && !name.endsWith(".test.ts")) {
      out.push(abs);
    }
  }
  return out;
}

const violations = [];
for (const file of tsFiles(coreDir)) {
  const lines = readFileSync(file, "utf8").split("\n");
  lines.forEach((line, i) => {
    // Ignore comment-only lines so prose mentioning a forbidden name is not a false positive.
    const trimmed = line.trim();
    if (trimmed.startsWith("//") || trimmed.startsWith("*") || trimmed.startsWith("/*")) return;
    for (const rule of RULES) {
      if (rule.re.test(line)) {
        violations.push(`${file}:${i + 1}: ${rule.label}\n    ${trimmed}`);
      }
    }
  });
}

if (violations.length > 0) {
  console.error(`core-purity: FAIL — ${violations.length} violation(s):\n`);
  for (const v of violations) console.error(`  ${v}`);
  process.exit(1);
}
console.log(`core-purity: OK — ${tsFiles(coreDir).length} core source file(s) clean`);

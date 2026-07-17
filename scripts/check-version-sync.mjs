#!/usr/bin/env node
// Version-sync drift guard. The repo carries one canonical version in the root VERSION file;
// three shipped manifests must agree with it:
//   - plugin/claude-plugin/.claude-plugin/plugin.json   (the installed Claude plugin)
//   - plugin/package.json                               (the plugin build package)
//   - .claude-plugin/marketplace.json                   (plugins[].version in the marketplace)
// Exits 1 (listing every disagreement) if any of the three differs from VERSION; exits 0 when
// all four match. Runs under plain Node (no deps), like the other scripts/ guards.

import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const REPO_ROOT = join(dirname(fileURLToPath(import.meta.url)), "..");

const canonical = readFileSync(join(REPO_ROOT, "VERSION"), "utf8").trim();

// Each source: a label, its file, and an extractor pulling the version string out of the parsed JSON.
const SOURCES = [
  {
    label: "plugin/claude-plugin/.claude-plugin/plugin.json",
    file: "plugin/claude-plugin/.claude-plugin/plugin.json",
    pick: (j) => j.version,
  },
  {
    label: "plugin/package.json",
    file: "plugin/package.json",
    pick: (j) => j.version,
  },
  {
    label: ".claude-plugin/marketplace.json (plugins[0].version)",
    file: ".claude-plugin/marketplace.json",
    pick: (j) =>
      Array.isArray(j.plugins) && j.plugins.length === 0
        ? "<plugins array is empty>"
        : j.plugins?.[0]?.version,
  },
];

const mismatches = [];
for (const src of SOURCES) {
  let found;
  try {
    found = src.pick(JSON.parse(readFileSync(join(REPO_ROOT, src.file), "utf8")));
  } catch (err) {
    mismatches.push(`${src.label}: could not read version (${err.message})`);
    continue;
  }
  if (found !== canonical) {
    mismatches.push(`${src.label}: ${found} !== VERSION ${canonical}`);
  }
}

if (mismatches.length > 0) {
  console.error(`version-sync: FAIL — ${mismatches.length} manifest(s) disagree with VERSION (${canonical}):`);
  for (const m of mismatches) console.error(`  ${m}`);
  process.exit(1);
}

console.log(`version-sync: OK — VERSION + 3 manifests all at ${canonical}.`);

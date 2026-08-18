#!/usr/bin/env node
// Version-sync drift guard: without it, a shipped manifest could silently disagree with the root
// VERSION, so the marketplace listing and an installed plugin advertise two different version numbers:
//   - plugins/desk-persona/.claude-plugin/plugin.json   (the installed desk-persona plugin)
//   - .claude-plugin/marketplace.json                   (each plugins[].version in the marketplace)
// Exits 1 (listing every disagreement) if any manifest differs from VERSION; exits 0 when all
// match. Runs under plain Node (no deps), like the other scripts/ guards.

import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const REPO_ROOT = join(dirname(fileURLToPath(import.meta.url)), "..");

const canonical = readFileSync(join(REPO_ROOT, "VERSION"), "utf8").trim();

// Each source: a label, its file, and an extractor pulling the version string out of the parsed JSON.
const SOURCES = [
  {
    label: "plugins/desk-persona/.claude-plugin/plugin.json",
    file: "plugins/desk-persona/.claude-plugin/plugin.json",
    pick: (j) => j.version,
  },
  {
    label: ".claude-plugin/marketplace.json (desk-persona plugins[].version)",
    file: ".claude-plugin/marketplace.json",
    pick: (j) => marketplaceVersion(j, "desk-persona"),
  },
];

// Find a marketplace plugin's version by name (robust to reordering / new entries).
function marketplaceVersion(j, name) {
  if (!Array.isArray(j.plugins) || j.plugins.length === 0) return "<plugins array is empty>";
  const entry = j.plugins.find((p) => p?.name === name);
  return entry ? entry.version : `<no plugins[] entry named ${name}>`;
}

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

console.log(`version-sync: OK — VERSION + ${SOURCES.length} manifests all at ${canonical}.`);

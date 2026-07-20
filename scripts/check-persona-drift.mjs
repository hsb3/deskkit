#!/usr/bin/env node
// Persona drift guard (#119, ADR 0014(a) + ADR 0015). The composed `desk-persona` bundle is a
// GENERATED artifact: its persona/skill bodies are version-controlled copies of exactly one
// authored source per surface, so they cannot silently diverge from that source. This guard is
// the repo's generated-artifact pattern (regenerate + compare) applied to the bundle:
//
//   plugin/desk-persona/agents/librarian-operator.md   ← librarian/templates/librarian-system-prompt.txt
//                                                          (the canonical librarian instruction; ADR 0015)
//   plugin/desk-persona/agents/pm-operator.md          ← plugin/desk-pm/agents/pm-operator.md
//   plugin/desk-persona/skills/pm-session-open/SKILL.md ← plugin/desk-pm/skills/pm-session-open/SKILL.md
//   plugin/desk-persona/skills/pm-advance-item/SKILL.md ← plugin/desk-pm/skills/pm-advance-item/SKILL.md
//   plugin/desk-persona/skills/pm-triage/SKILL.md       ← plugin/desk-pm/skills/pm-triage/SKILL.md
//
// The PM-sourced files are copied from desk-pm — the ONE authored PM source (desk-pm coexists,
// untouched) — with the MCP server-name prefix rewritten `mcp__desk-pm__` → `mcp__desk-persona__`
// so the composed mount's tool namespace is correct. The librarian agent is derived from the
// corrected 5-tool eino system prompt: its `tools:` frontmatter and its embedded prompt body are
// both regenerated from that one file, so a tool added to / removed from the canonical prompt
// changes the persona and trips this guard.
//
// Usage (plain Node, no deps — like the other scripts/ guards):
//   node scripts/check-persona-drift.mjs           compare on-disk vs regenerated; exit 1 on drift
//   node scripts/check-persona-drift.mjs --write    (re)generate the derived files in place
//
// Drift is caught in BOTH directions: hand-editing a generated file (on-disk ≠ regenerated) and
// changing a source without regenerating (on-disk stale) both make the compare fail non-zero.

import { readFileSync, writeFileSync, existsSync, mkdirSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const REPO_ROOT = join(dirname(fileURLToPath(import.meta.url)), "..");
const r = (p) => readFileSync(join(REPO_ROOT, p), "utf8");

const LIBRARIAN_PROMPT = "librarian/templates/librarian-system-prompt.txt";
const PM_AGENT_SRC = "plugin/desk-pm/agents/pm-operator.md";
const PM_SKILL_NAMES = ["pm-session-open", "pm-advance-item", "pm-triage"];

// The single namespace rewrite: the composed bundle's MCP server key is `desk-persona`.
function toPersonaNamespace(text) {
  return text.split("mcp__desk-pm__").join("mcp__desk-persona__");
}

// Parse the ordered tool list out of the canonical librarian prompt's "You have these tools:"
// block — the list whose membership the frontmatter must track.
function librarianToolNames(prompt) {
  const lines = prompt.split("\n");
  const start = lines.findIndex((l) => /^You have these tools:/.test(l));
  if (start === -1) throw new Error(`${LIBRARIAN_PROMPT}: no "You have these tools:" block`);
  const tools = [];
  for (let i = start + 1; i < lines.length; i++) {
    const line = lines[i];
    if (/^\S/.test(line)) break; // first non-indented line ends the block
    const m = line.match(/^\s+-\s+([a-z_]+)\b/);
    if (m) tools.push(m[1]);
  }
  if (tools.length === 0) throw new Error(`${LIBRARIAN_PROMPT}: parsed zero tools`);
  return tools;
}

// Build the librarian-operator agent markdown: deterministic frontmatter (tools derived from the
// prompt) + the canonical prompt embedded verbatim as the persona body (ADR 0015 prompt copy).
function buildLibrarianAgent() {
  const prompt = r(LIBRARIAN_PROMPT).replace(/\s+$/, "");
  const tools = librarianToolNames(prompt);
  const toolLines = ["Read", ...tools.map((t) => `mcp__desk-persona__${t}`)]
    .map((t) => `  - ${t}`)
    .join("\n");
  const frontmatter = [
    "---",
    "name: librarian-operator",
    "description: >-",
    "  Operates a desk's documentation library through the librarian tool family: grounds every",
    "  claim with a read-only query, reindexes the tree with sweep, flags rule violations with",
    "  patrol, computes record-original-first mechanical fixes with propose_fix, and logs a",
    "  problem or feedback with record_feedback. Use when a session needs the desk indexed,",
    "  audited for rule violations, or mechanically repaired under the record-original-first",
    "  boundary. Never authors or rewrites prose; acts only through its tools.",
    "model: inherit",
    "color: green",
    "tools:",
    toolLines,
    "---",
  ].join("\n");
  const header = [
    "<!-- GENERATED — do not hand-edit. This persona body is a version-controlled copy of the",
    "     canonical librarian instruction (librarian/templates/librarian-system-prompt.txt, ADR 0015).",
    "     Regenerate with:  node scripts/check-persona-drift.mjs --write",
    "     Drift is guarded by scripts/check-persona-drift.mjs (wired into `make check`). -->",
  ].join("\n");
  return `${frontmatter}\n\n${header}\n\n# librarian-operator\n\n${prompt}\n`;
}

// The derived-file manifest: each target and how to (re)build its expected content from source.
const DERIVED = [
  {
    target: "plugin/desk-persona/agents/librarian-operator.md",
    build: buildLibrarianAgent,
  },
  {
    target: "plugin/desk-persona/agents/pm-operator.md",
    build: () => toPersonaNamespace(r(PM_AGENT_SRC)),
  },
  ...PM_SKILL_NAMES.map((name) => ({
    target: `plugin/desk-persona/skills/${name}/SKILL.md`,
    build: () => toPersonaNamespace(r(`plugin/desk-pm/skills/${name}/SKILL.md`)),
  })),
];

const write = process.argv.includes("--write");

if (write) {
  for (const { target, build } of DERIVED) {
    const abs = join(REPO_ROOT, target);
    mkdirSync(dirname(abs), { recursive: true });
    writeFileSync(abs, build());
  }
  console.log(`persona-drift: wrote ${DERIVED.length} generated file(s) under plugin/desk-persona/.`);
  process.exit(0);
}

const drifted = [];
for (const { target, build } of DERIVED) {
  const abs = join(REPO_ROOT, target);
  const expected = build();
  const actual = existsSync(abs) ? readFileSync(abs, "utf8") : null;
  if (actual === null) drifted.push(`${target}: MISSING (run --write)`);
  else if (actual !== expected) drifted.push(`${target}: differs from its source (run --write)`);
}

if (drifted.length > 0) {
  console.error(
    `persona-drift: FAIL — ${drifted.length} generated file(s) out of sync with their source:`,
  );
  for (const d of drifted) console.error(`  ${d}`);
  console.error(
    "\nThe desk-persona bundle is generated; edit the SOURCE (the eino prompt / the desk-pm content),",
    "then regenerate:  node scripts/check-persona-drift.mjs --write",
  );
  process.exit(1);
}

console.log(`persona-drift: OK — ${DERIVED.length} generated desk-persona file(s) in sync with source.`);

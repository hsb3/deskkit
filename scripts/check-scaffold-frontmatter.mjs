#!/usr/bin/env node
// Frontmatter-conformance guard for shipped scaffold INSTRUMENT assets: a missing required key
// here ships into every live desk instantiated from this asset, silently breaking that desk's
// frontmatter contract.
//
// Most files under a skill's assets/ are standard-free scaffold TEMPLATES (K25) — exempt from
// the frontmatter contract by design (e.g. desk-setup/assets/template/CLAUDE.md). But some
// assets are instruments meant to be copied verbatim into a live desk and become real, checked
// desk documents there — those ship WITH conformant frontmatter (Option B, #80) so the
// instantiated copy is clean by construction and never needs a one-off exemption (see
// conventions-standard SKILL.md, "Frontmatter contract" + adherence-checklist rule 1).
//
// Checks KEY PRESENCE only (schema/doctypes.yaml's `universal:` set + the desk-surface
// `synopsis` requirement) — not value format — because these are still-templated assets that
// intentionally ship bracket placeholders (e.g. `<YYYY-MM-DD>`) for a desk owner to fill in at
// copy time; strict value validation is the deskkit's job against an instantiated desk.
//
//   node scripts/check-scaffold-frontmatter.mjs

import { readFileSync, existsSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const REPO_ROOT = join(dirname(fileURLToPath(import.meta.url)), "..");

// Instrument assets that ship WITH conformant frontmatter. Add a path here whenever a new
// skill asset is designed to be copied (or template_render-materialized) verbatim into a desk
// as a real (non-template) document. The two improvement-log.md copies are siblings — the
// harvest-loop asset is copied by hand; the desk-setup one is the greenfield scaffold's actual
// `_meta/improvement-log.md` seed, resolved via template_render (desk-standard #80).
const INSTRUMENT_ASSETS = [
  "plugins/desk-persona/skills/harvest-loop/assets/improvement-log.md",
  "plugins/desk-persona/skills/desk-setup/assets/template/_meta/improvement-log.md",
];

// Universal frontmatter keys: schema/doctypes.yaml `universal: [type, status, created, updated,
// tags]` plus `synopsis`, required on the desk surface (conventions-standard SKILL.md,
// "Frontmatter contract").
const REQUIRED_KEYS = ["type", "status", "created", "updated", "tags", "synopsis"];

function parseFrontmatterKeys(text) {
  if (!text.startsWith("---")) return null;
  const end = text.indexOf("\n---", 3);
  if (end === -1) return null;
  const block = text.slice(3, end);
  const keys = new Set();
  for (const rawLine of block.split("\n")) {
    const m = rawLine.match(/^([A-Za-z_][A-Za-z0-9_]*):/);
    if (m) keys.add(m[1]);
  }
  return keys;
}

const problems = [];

for (const rel of INSTRUMENT_ASSETS) {
  const abs = join(REPO_ROOT, rel);
  if (!existsSync(abs)) {
    problems.push(`${rel}: listed instrument asset is missing on disk`);
    continue;
  }
  const text = readFileSync(abs, "utf8");
  const keys = parseFrontmatterKeys(text);
  if (keys === null) {
    problems.push(`${rel}: no frontmatter block found (expected a leading --- ... --- YAML block)`);
    continue;
  }
  const missing = REQUIRED_KEYS.filter((k) => !keys.has(k));
  if (missing.length > 0) {
    problems.push(`${rel}: missing frontmatter key(s): ${missing.join(", ")}`);
  }
}

if (problems.length > 0) {
  console.error(`check-scaffold-frontmatter: FAIL — ${problems.length} instrument asset(s) non-conformant:`);
  for (const p of problems) console.error(`  ${p}`);
  process.exit(1);
}

console.log(
  `check-scaffold-frontmatter: OK — ${INSTRUMENT_ASSETS.length} instrument asset(s) carry required frontmatter (${REQUIRED_KEYS.join(", ")}).`
);

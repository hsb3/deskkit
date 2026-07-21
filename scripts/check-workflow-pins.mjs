#!/usr/bin/env node
// GitHub Actions SHA-pin drift guard. After every `uses:` in .github/workflows/* was pinned to a
// full commit SHA (supply-chain hardening), nothing stopped a later PR from reintroducing a
// mutable `@vN` / `@branch` tag — which silently re-opens the "an upstream tag moves under us"
// risk the pinning closed. This asserts every third-party `uses:` reference is pinned to a full
// 40-hex commit SHA. Local (`./…`) and `docker://…` references are exempt (they cannot be
// SHA-pinned the same way). Runs under plain Node (no deps), like the other scripts/ guards.
//
//   node scripts/check-workflow-pins.mjs             # scan .github/workflows — FAIL on any unpinned uses:
//   node scripts/check-workflow-pins.mjs --self-test  # prove the guard still detects a seeded @vN tag
//
// Exit 1 on any problem (or, under --self-test, if the guard FAILS to flag a seeded violation).

import { readFileSync, readdirSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const REPO_ROOT = join(dirname(fileURLToPath(import.meta.url)), "..");
const WORKFLOW_DIR = join(REPO_ROOT, ".github", "workflows");

const SHA_RE = /^[0-9a-f]{40}$/;
// Capture the `uses:` value (a step key or a leading `- uses:`), stopping at whitespace so a
// trailing `# vN` comment is excluded.
const USES_RE = /^\s*(?:-\s*)?uses:\s*(\S+)/;

// Classify a single `uses:` value. Returns null when the reference is acceptably pinned or exempt,
// or a human-readable reason string when it is a violation.
function pinViolation(value) {
  if (value.startsWith("./") || value.startsWith("docker://")) return null; // local / docker — exempt
  const at = value.lastIndexOf("@");
  if (at === -1) return `unpinned (no @<sha>): "${value}"`;
  const ref = value.slice(at + 1);
  if (!SHA_RE.test(ref)) return `not pinned to a 40-hex commit SHA (got "@${ref}"): "${value}"`;
  return null;
}

// Scan raw workflow text; return [{ line, value, reason }] for every violating `uses:`.
function scanText(text) {
  const out = [];
  const lines = text.split("\n");
  for (let i = 0; i < lines.length; i++) {
    const m = lines[i].match(USES_RE);
    if (!m) continue;
    const reason = pinViolation(m[1]);
    if (reason) out.push({ line: i + 1, value: m[1], reason });
  }
  return out;
}

if (process.argv.includes("--self-test")) {
  // A seeded workflow with one good pin, one mutable @vN tag, one local ref, and one docker ref.
  // The guard must flag exactly the @vN line and leave the pinned/local/docker lines alone.
  const seeded = [
    "jobs:",
    "  a:",
    "    steps:",
    "      - uses: actions/checkout@34e114876b0b11c390a56381ad16ebd13914f8d5 # v4",
    "      - uses: actions/setup-go@v5",
    "      - uses: ./.github/actions/local",
    "      - uses: docker://alpine:3.20",
  ].join("\n");
  const found = scanText(seeded);
  const flaggedV5 = found.some((f) => f.value === "actions/setup-go@v5");
  if (found.length !== 1 || !flaggedV5) {
    console.error(
      `check-workflow-pins: SELF-TEST FAIL — expected exactly the @vN tag to be flagged, got ${JSON.stringify(found)}`,
    );
    process.exit(1);
  }
  console.log("check-workflow-pins: self-test OK — a mutable @vN tag is still detected (pinned/local/docker refs pass).");
  process.exit(0);
}

let files;
try {
  files = readdirSync(WORKFLOW_DIR).filter((f) => f.endsWith(".yml") || f.endsWith(".yaml"));
} catch (err) {
  console.error(`check-workflow-pins: FAIL — could not read ${WORKFLOW_DIR} (${err.message}).`);
  process.exit(1);
}

const problems = [];
let usesCount = 0;
for (const file of files.sort()) {
  const text = readFileSync(join(WORKFLOW_DIR, file), "utf8");
  for (const line of text.split("\n")) {
    if (USES_RE.test(line)) usesCount++;
  }
  for (const v of scanText(text)) {
    problems.push(`.github/workflows/${file}:${v.line}: ${v.reason}`);
  }
}

if (problems.length > 0) {
  console.error(`check-workflow-pins: FAIL — ${problems.length} unpinned action reference(s):`);
  for (const p of problems) console.error(`  ${p}`);
  console.error("  Pin every `uses:` to a full 40-hex commit SHA (keep the `# vN` comment for readability).");
  process.exit(1);
}

console.log(
  `check-workflow-pins: OK — all ${usesCount} action reference(s) across ${files.length} workflow file(s) are SHA-pinned (or local/docker-exempt).`,
);

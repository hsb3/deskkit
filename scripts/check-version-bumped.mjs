#!/usr/bin/env node
// Version-bump guard: every PR must move the root VERSION.
//
// A change worth merging is a change worth naming, so the version moves with the PR rather than in
// a periodic sweep. The advisory `version-status` target reports the same drift without blocking on
// it; this is that fact as a gate.
//
// Compares VERSION on HEAD against VERSION on the merge base with the base branch. Exits 1 when
// they match, 0 when HEAD's is different. Off a PR (no base to diff against) it exits 0 with a
// notice rather than inventing a comparison.
//
// Usage:  node scripts/check-version-bumped.mjs [base-ref]     (default: origin/main)
//         node scripts/check-version-bumped.mjs --self-test
// Runs under plain Node (no deps), like the other scripts/ guards.

import { execFileSync } from "node:child_process";
import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const REPO_ROOT = join(dirname(fileURLToPath(import.meta.url)), "..");

/** Run a git command, returning trimmed stdout, or null if git itself failed. */
function git(args) {
  try {
    return execFileSync("git", args, { cwd: REPO_ROOT, encoding: "utf8", stdio: ["ignore", "pipe", "ignore"] }).trim();
  } catch {
    return null;
  }
}

/** Parse a dotted version into comparable parts, or null if it is not one. */
function parse(v) {
  const m = /^(\d+)\.(\d+)\.(\d+)$/.exec(v);
  return m ? [Number(m[1]), Number(m[2]), Number(m[3])] : null;
}

/** Is `next` strictly greater than `prev`? Both must parse. */
export function isBump(prev, next) {
  const a = parse(prev);
  const b = parse(next);
  if (!a || !b) return false;
  for (let i = 0; i < 3; i++) {
    if (b[i] > a[i]) return true;
    if (b[i] < a[i]) return false;
  }
  return false;
}

function selfTest() {
  const cases = [
    ["0.9.0", "0.10.0", true, "minor bump"],
    ["0.9.0", "0.9.1", true, "patch bump"],
    ["0.9.0", "1.0.0", true, "major bump"],
    ["0.9.0", "0.9.0", false, "unchanged is not a bump"],
    ["0.10.0", "0.9.0", false, "going backwards is not a bump"],
    ["0.9.0", "0.9", false, "malformed next"],
    ["", "0.9.1", false, "malformed prev"],
    ["0.9.9", "0.10.0", true, "10 sorts above 9, not lexically below it"],
  ];
  let bad = 0;
  for (const [prev, next, want, why] of cases) {
    const got = isBump(prev, next);
    const ok = got === want;
    if (!ok) bad++;
    console.log(`  [${ok ? "OK" : "BAD"}] ${why}: ${prev || "(empty)"} -> ${next} => ${got}`);
  }
  if (bad) {
    console.error(`check-version-bumped: self-test FAILED (${bad} case(s))`);
    process.exit(1);
  }
  console.log("check-version-bumped: self-test OK — bumps, non-bumps and malformed input all classified.");
  process.exit(0);
}

if (process.argv.includes("--self-test")) selfTest();

const baseRef = process.argv[2] ?? "origin/main";
const head = readFileSync(join(REPO_ROOT, "VERSION"), "utf8").trim();

// The merge base, not the base tip: comparing against the tip would demand a fresh bump every time
// main moves ahead, which is a rebase, not a defect.
const mergeBase = git(["merge-base", "HEAD", baseRef]);
if (!mergeBase) {
  console.log(`check-version-bumped: SKIP — no merge base with ${baseRef} (not a PR build, or a shallow clone).`);
  process.exit(0);
}

const baseVersion = git(["show", `${mergeBase}:VERSION`]);
if (baseVersion === null) {
  console.log(`check-version-bumped: SKIP — ${baseRef} has no VERSION at the merge base.`);
  process.exit(0);
}

// A branch sitting exactly on the base has nothing to bump for.
if (git(["rev-parse", "HEAD"]) === mergeBase) {
  console.log("check-version-bumped: SKIP — HEAD is the merge base (no commits to release).");
  process.exit(0);
}

if (isBump(baseVersion, head)) {
  console.log(`check-version-bumped: OK — VERSION moved ${baseVersion} -> ${head} since ${baseRef}.`);
  process.exit(0);
}

console.error(`check-version-bumped: FAIL — VERSION is ${head}, same as ${baseRef} (${baseVersion}).`);
console.error("");
console.error("Every PR bumps the version. Pick the smallest true one:");
console.error("  patch  a fix or an internal change users will not notice");
console.error("  minor  anything a user could see — a new surface, flag, command or behaviour");
console.error("  major  a break");
console.error("");
console.error("Then keep the shipped manifests and the changelog with it:");
console.error("  edit VERSION, plugins/deskkit/.claude-plugin/plugin.json, .claude-plugin/marketplace.json");
console.error("  add the CHANGELOG.md section");
console.error("  node scripts/check-version-sync.mjs");
process.exit(1);

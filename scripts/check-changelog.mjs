#!/usr/bin/env node
// Changelog release gate. The root VERSION is the single source of truth; a release must be
// documented before it can be cut. This asserts CHANGELOG.md carries a section header for the
// current VERSION — `## [<version>]` (optionally followed by a date) — with at least one
// non-blank content line before the next `## ` header.
//
// Wired into `make release-prep` and the release.yml gate: bumping VERSION without adding its
// CHANGELOG section fails here, which is the forcing function that keeps the changelog honest.
// Runs under plain Node (no deps), like the other scripts/ guards. Exit 1 on any problem.

import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const REPO_ROOT = join(dirname(fileURLToPath(import.meta.url)), "..");

function fail(msg) {
  console.error(`check-changelog: FAIL — ${msg}`);
  process.exit(1);
}

const version = readFileSync(join(REPO_ROOT, "VERSION"), "utf8").trim();

let changelog;
try {
  changelog = readFileSync(join(REPO_ROOT, "CHANGELOG.md"), "utf8");
} catch (err) {
  fail(`could not read CHANGELOG.md (${err.message}).`);
}

const lines = changelog.split("\n");

// A version header looks like `## [0.5.0]` or `## [0.5.0] — 2026-07-18`. Match the current
// VERSION exactly (escape the dots so 0.5.0 doesn't also match 0x5x0).
const escaped = version.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
const headerRe = new RegExp(`^##\\s+\\[${escaped}\\](\\s|$)`);
const anyH2Re = /^##\s+/;

const start = lines.findIndex((l) => headerRe.test(l));
if (start === -1) {
  fail(
    `CHANGELOG.md has no section for VERSION ${version}. ` +
      `Add a "## [${version}] — <date>" section (move the [Unreleased] entries into it).`,
  );
}

// Collect content lines until the next `## ` header; require at least one non-blank line.
let hasContent = false;
for (let i = start + 1; i < lines.length; i++) {
  if (anyH2Re.test(lines[i])) break;
  if (lines[i].trim() !== "") {
    hasContent = true;
    break;
  }
}

if (!hasContent) {
  fail(`the "## [${version}]" section is empty — document what changed before releasing.`);
}

console.log(`check-changelog: OK — CHANGELOG.md documents version ${version}.`);

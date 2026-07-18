#!/usr/bin/env node
// Version-status advisory (NON-BLOCKING — always exits 0). Answers "have user-facing changes
// piled up since the last release without a version bump?" — the drift that let the whole chat
// TUI merge under a stale VERSION. Run by `make version-status` and as a non-required CI step;
// it only ever prints, never fails a build. The hard gate is check-changelog.mjs at release time.
//
// Logic: compare the root VERSION to the latest `v*` git tag.
//   VERSION >  last tag  → a release is staged; nudge if its CHANGELOG section is still missing.
//   VERSION == last tag  → warn if plugin/ or librarian/ changed since the tag (unreleased work).
//   VERSION <  last tag  → warn (VERSION is behind the last tag — unexpected).
// Degrades to a soft note when git history/tags aren't available (e.g. a shallow CI checkout).

import { execFileSync } from "node:child_process";
import { readFileSync as read } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const REPO_ROOT = join(dirname(fileURLToPath(import.meta.url)), "..");

// Run git with an argv array (no shell) — no interpolation surface for tag names.
function git(args) {
  return execFileSync("git", args, { cwd: REPO_ROOT, encoding: "utf8", stdio: ["ignore", "pipe", "ignore"] }).trim();
}

// Compare two "x.y.z" strings. Returns -1 / 0 / 1.
function cmpSemver(a, b) {
  const pa = a.split(".").map((n) => parseInt(n, 10) || 0);
  const pb = b.split(".").map((n) => parseInt(n, 10) || 0);
  for (let i = 0; i < 3; i++) {
    if ((pa[i] || 0) !== (pb[i] || 0)) return (pa[i] || 0) < (pb[i] || 0) ? -1 : 1;
  }
  return 0;
}

let version;
try {
  version = read(join(REPO_ROOT, "VERSION"), "utf8").trim();
} catch (err) {
  // Advisory must never fail the build — degrade to a soft note even if VERSION is unreadable.
  console.log(`version-status: (advisory) could not read VERSION (${err.message}) — skipping.`);
  process.exit(0);
}

let lastTag = "";
try {
  // Newest v-prefixed tag by semver order. Empty if there are no tags (or a tagless shallow clone).
  lastTag = git(["tag", "--list", "v*", "--sort=-v:refname"]).split("\n")[0].trim();
} catch {
  console.log("version-status: (advisory) git tags unavailable — skipping drift check.");
  process.exit(0);
}

if (!lastTag) {
  console.log(`version-status: (advisory) no v* release tag yet; VERSION is ${version}.`);
  process.exit(0);
}

const lastVersion = lastTag.replace(/^v/, "");
const c = cmpSemver(version, lastVersion);

if (c > 0) {
  // A bump is staged ahead of the last release. Nudge if it isn't documented yet.
  let documented = true;
  try {
    const escaped = version.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
    documented = new RegExp(`^##\\s+\\[${escaped}\\]`, "m").test(read(join(REPO_ROOT, "CHANGELOG.md"), "utf8"));
  } catch {
    documented = false;
  }
  console.log(`version-status: OK — VERSION ${version} is ahead of last tag ${lastTag} (a release is staged).`);
  if (!documented) {
    console.log(`  ⚠ CHANGELOG.md has no "## [${version}]" section yet — add one before tagging (check-changelog gates this).`);
  }
  process.exit(0);
}

if (c < 0) {
  console.log(`version-status: ⚠ VERSION ${version} is BEHIND the last tag ${lastTag} — unexpected; check the VERSION file.`);
  process.exit(0);
}

// VERSION == last tag: any product change since the tag is unreleased work with no bump.
let changed = [];
try {
  const out = git(["diff", "--name-only", `${lastTag}..HEAD`, "--", "plugin", "librarian"]);
  changed = out ? out.split("\n").filter(Boolean) : [];
} catch {
  console.log(`version-status: (advisory) can't diff ${lastTag}..HEAD (shallow clone?) — skipping drift check.`);
  process.exit(0);
}

if (changed.length === 0) {
  console.log(`version-status: OK — VERSION ${version} matches last tag ${lastTag}; no unreleased product changes.`);
} else {
  console.log(`version-status: ⚠ ${changed.length} file(s) under plugin/ or librarian/ changed since tag ${lastTag}, but VERSION is still ${version}.`);
  console.log(`  → Unreleased user-facing work is accumulating. Bump VERSION + the three manifests and add a CHANGELOG entry before the next release.`);
  console.log(`  (advisory only — this never fails the build.)`);
}
process.exit(0);

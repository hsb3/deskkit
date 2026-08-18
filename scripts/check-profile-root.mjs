#!/usr/bin/env node
// Profile-root drift guard. The single personalization-root directory name lives in exactly one
// canonical place — schema/paths.yaml's `profile_root` — and the product lane pins a constant to
// that value (librarian config.ProfileRootDir). Nothing otherwise stops that constant drifting
// from the canonical value, which would silently split the shipped binary off the schema's root.
// This asserts the two are byte-identical, so a rename is a one-definition change here plus the
// constant. Runs under plain Node (no deps), like the other scripts/ guards.
//
//   node scripts/check-profile-root.mjs             # compare canonical vs the lane constant — FAIL on divergence
//   node scripts/check-profile-root.mjs --self-test  # prove the guard still detects a seeded divergence
//
// Exit 1 on any problem — a divergence OR an un-extractable value (a guard that silently finds
// nothing is worse than none) — or, under --self-test, if the guard FAILS to flag a seeded divergence.

import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const REPO_ROOT = join(dirname(fileURLToPath(import.meta.url)), "..");

// Source of the canonical value and the lane's pinned constant (repo-relative labels for output).
const CANONICAL_PATH = "schema/paths.yaml";
const GO_PATH = "librarian/internal/core/config/profile.go";

// Extraction regexes. Canonical: an anchored `profile_root:` line (the `^` skips the commented
// mention of the same key), optional quotes, value stops at whitespace/quote/`#`. Go: a
// ProfileRootDir const tolerant of interpreted (") or raw (`) quotes.
const CANONICAL_RE = /^profile_root:\s*["']?([^"'\s#]+)/m;
const GO_RE = /ProfileRootDir\s*=\s*["`]([^"`]+)["`]/;

function matchOne(text, re) {
  const m = text.match(re);
  return m ? m[1] : null;
}

// Compare the canonical value against the lane constant over raw source text. Returns both
// extracted values plus a list of { source, reason } problems: an extraction failure for any value
// that could not be parsed, and a divergence if the lane constant differs from the canonical.
function scan({ canonicalText, goText }) {
  const canonical = matchOne(canonicalText, CANONICAL_RE);
  const go = matchOne(goText, GO_RE);
  const problems = [];

  if (canonical === null) problems.push({ source: CANONICAL_PATH, reason: "could not extract `profile_root` (missing file or no match)" });
  if (go === null) problems.push({ source: GO_PATH, reason: "could not extract ProfileRootDir (missing file or no match)" });

  // Divergence only means something once the canonical value is known.
  if (canonical !== null && go !== null && go !== canonical) {
    problems.push({ source: GO_PATH, reason: `ProfileRootDir="${go}" != canonical profile_root="${canonical}"` });
  }

  return { canonical, go, problems };
}

if (process.argv.includes("--self-test")) {
  // The canonical value is `_knowledge`; the Go const is DELIBERATELY seeded to a different value.
  // The guard must flag exactly that one Go divergence (and nothing else) over the injected strings.
  const seededGo = "_seeded_divergent_root";
  const { problems } = scan({
    canonicalText: "profile_root: _knowledge\n",
    goText: `const ProfileRootDir = "${seededGo}"\n`,
  });
  const flaggedGoDivergence = problems.some((p) => p.source === GO_PATH && p.reason.includes(seededGo));
  if (problems.length !== 1 || !flaggedGoDivergence) {
    console.error(
      `check-profile-root: SELF-TEST FAIL — expected exactly the seeded Go divergence to be flagged, got ${JSON.stringify(problems)}`,
    );
    process.exit(1);
  }
  console.log("check-profile-root: self-test OK — a divergent lane constant is still detected.");
  process.exit(0);
}

function read(rel) {
  try {
    return readFileSync(join(REPO_ROOT, rel), "utf8");
  } catch {
    return ""; // an empty string yields no regex match → an extraction-failure problem below
  }
}

const { canonical, problems } = scan({
  canonicalText: read(CANONICAL_PATH),
  goText: read(GO_PATH),
});

if (problems.length > 0) {
  console.error(`check-profile-root: FAIL — ${problems.length} problem(s):`);
  for (const p of problems) console.error(`  ${p.source}: ${p.reason}`);
  console.error(`  The canonical value (${CANONICAL_PATH} profile_root) and the lane constant must be identical — fix the divergent source.`);
  process.exit(1);
}

console.log(
  `check-profile-root: OK — profile_root="${canonical}" is pinned identically across ${CANONICAL_PATH} and ${GO_PATH}.`,
);

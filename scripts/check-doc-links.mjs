#!/usr/bin/env node
// Doc-link drift guard. Asserts that every citation of a repo doc/media path resolves to a file
// that actually exists. This is the gate that would have caught the 2026-07-24 docs reorg: a bulk
// `git mv` of the specs + guides + media out of docs/ left dozens of dangling `docs/…` references
// scattered across gate scripts, README, CLAUDE.md, shell/Go/TS comments, VHS tapes, and the active
// plans — none of which any check looked at, so the only symptom was a *different* gate crashing on
// ENOENT when it tried to READ one of the moved specs. A moved-but-still-cited doc now fails HERE,
// directly and by name, instead of silently.
//
// Two citation forms are checked over the git-tracked text tree (minus the exclusions below):
//   1. Root-relative doc references — `docs/….<ext>` (md/mdx/gif/png/svg/txt/tape/yaml). A trailing
//      `:123` line anchor or `#section` fragment is stripped before the existence check.
//   2. Relative markdown links to a sibling or parent .md file — resolved against the citing file's dir.
//
// Scope: this gate enforces link integrity on the PUBLISHED + SHIPPED surface — docs/, the root
// digests (README/CLAUDE/AGENTS), and the code tree (librarian/schema/plugins/scripts/kits).
// It deliberately skips the working desk (`_meta/`) and agent state (`.claude/`) — point-in-time
// snapshots, in-flight drafts, and curated memory whose refs are provenance, not maintained
// navigation — plus `CHANGELOG.md` (append-only) and test files (synthetic path fixtures). Keeping
// the working desk clean is a curation discipline (see docs/development/docs-layout.md), not a CI
// gate. A path that intentionally does not exist yet is allow-listed one-per-line in
// scripts/check-doc-links.allow as "<citing-file> -> <cited-path>".
//
//   node scripts/check-doc-links.mjs              scan; exit 1 on any dangling reference, 0 when clean
//   node scripts/check-doc-links.mjs --self-test   prove the guard still flags a seeded dangler
//
// House pattern (matches check-query-kinds.mjs): plain Node, no deps, a pure-function matcher
// shared by the live scan and the --self-test's synthetic filesystem.

import { readFileSync, existsSync, lstatSync } from "node:fs";
import { execFileSync } from "node:child_process";
import { dirname, join, relative, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const REPO_ROOT = join(dirname(fileURLToPath(import.meta.url)), "..");
const rel = (p) => relative(REPO_ROOT, p).split("\\").join("/");

// Text files worth scanning for citations (prose, code comments, config, tapes).
const SCAN_EXT = /\.(?:md|mdx|ts|tsx|js|mjs|cjs|go|sh|bash|json|ya?ml|tape|txt)$/i;
const EXCLUDE_PREFIX = ["_meta/", ".claude/", "librarian/vendor/"];
const EXCLUDE_EXACT = new Set(["CHANGELOG.md"]);
const EXCLUDE_FILE = /(?:_test\.go|\.test\.ts|\.spec\.ts|_test\.py|\.test\.js)$/;

// Root-relative doc-tree citation. The negative lookbehind rejects URL tails and longer paths
// (`…/docs/x.md`), so only a repo-root-relative `docs/…` citation matches; the char class stops at
// the extension via backtracking.
const ROOT_DOC_RE = /(?<![A-Za-z0-9._/-])docs\/[A-Za-z0-9._/-]+\.(?:md|mdx|gif|png|svg|txt|tape|ya?ml)/g;
// Relative markdown link to a .md(x) target — skip URLs, in-page anchors, mailto, and root-absolute.
const REL_MD_RE = /\]\((?!https?:|#|mailto:|\/)([^)\s#]+\.mdx?)(?:#[^)]*)?\)/g;

// Strip a trailing `:NNN` / `:NNN-MMM` line anchor (plan citations carry these).
const stripAnchor = (p) => p.replace(/:\d+(?:-\d+)?$/, "");

// Pure matcher: returns [{ ref, target }] for citations whose resolved target does NOT exist.
// `exists` is injected so the self-test can supply a synthetic filesystem.
function findDanglers(fileRel, content, exists = (abs) => existsSync(abs)) {
  const out = [];
  for (const m of content.matchAll(ROOT_DOC_RE)) {
    const target = stripAnchor(m[0]);
    if (!exists(join(REPO_ROOT, target))) out.push({ ref: m[0], target });
  }
  for (const m of content.matchAll(REL_MD_RE)) {
    const abs = resolve(REPO_ROOT, dirname(fileRel), m[1]);
    if (!exists(abs)) out.push({ ref: `](${m[1]})`, target: rel(abs) });
  }
  return out;
}

function loadAllow() {
  const p = join(REPO_ROOT, "scripts", "check-doc-links.allow");
  if (!existsSync(p)) return new Set();
  return new Set(
    readFileSync(p, "utf8")
      .split("\n")
      .map((l) => l.replace(/#.*$/, "").trim())
      .filter(Boolean),
  );
}

function scan() {
  const allow = loadAllow();
  const files = execFileSync("git", ["ls-files"], { cwd: REPO_ROOT, encoding: "utf8" })
    .split("\n")
    .filter(Boolean)
    .filter((f) => SCAN_EXT.test(f))
    .filter((f) => !EXCLUDE_EXACT.has(f))
    .filter((f) => !EXCLUDE_FILE.test(f))
    .filter((f) => !EXCLUDE_PREFIX.some((pre) => f.startsWith(pre)))
    // Skip symlinks (e.g. AGENTS.md -> CLAUDE.md) — the target is scanned on its own, so following
    // the link would just double-count every finding under both names.
    .filter((f) => {
      try {
        return !lstatSync(join(REPO_ROOT, f)).isSymbolicLink();
      } catch {
        return true;
      }
    });

  const problems = [];
  for (const f of files) {
    let content;
    try {
      content = readFileSync(join(REPO_ROOT, f), "utf8");
    } catch {
      continue;
    }
    for (const d of findDanglers(f, content)) {
      if (allow.has(`${f} -> ${d.target}`)) continue;
      problems.push({ file: f, ...d });
    }
  }

  if (problems.length) {
    console.error(`check-doc-links: FAIL — ${problems.length} dangling doc reference(s):`);
    for (const p of problems) console.error(`  ${p.file}: "${p.ref}" -> ${p.target} (no such file)`);
    console.error(
      `\nEvery cited doc/media path must resolve to a file that exists. If a path moved, repoint the\n` +
        `citation; if it was deleted, remove the reference. A path that intentionally does not exist yet\n` +
        `can be allow-listed in scripts/check-doc-links.allow as "<citing-file> -> <cited-path>".`,
    );
    process.exit(1);
  }
  console.log(`check-doc-links: OK — every doc/media citation across ${files.length} tracked file(s) resolves.`);
}

function selfTest() {
  // Synthetic filesystem — decoupled from the real tree, and built by concatenation so no scannable
  // literal of the seeded-bad path appears in this file (which the live scan reads).
  const specDir = ["docs", "development", "specs"].join("/");
  const realPath = specDir + "/tool-surface.md";
  const badPath = "docs/" + "gone-" + "xyz.md";
  const present = new Set([join(REPO_ROOT, realPath)]);
  const exists = (abs) => present.has(abs);

  const clean = "see `" + realPath + "` for the counts";
  const seeded = "see `" + badPath + "` and [gone](../" + "nope/missing.md)";
  const cleanHits = findDanglers("README.md", clean, exists);
  const seededHits = findDanglers("README.md", seeded, exists);

  const cleanPasses = cleanHits.length === 0;
  const seededCaught = seededHits.some((h) => h.target === badPath);
  if (!cleanPasses || !seededCaught) {
    console.error(
      `check-doc-links --self-test: FAIL — matcher regressed ` +
        `(clean-input flagged=${!cleanPasses}, seeded-dangler missed=${!seededCaught}).`,
    );
    process.exit(1);
  }
  console.log("check-doc-links --self-test: OK — clean input passes, seeded dangler is caught.");
}

process.argv.includes("--self-test") ? selfTest() : scan();

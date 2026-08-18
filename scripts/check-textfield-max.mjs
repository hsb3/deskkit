#!/usr/bin/env node
// TextField explicit-Max recurrence guard (ADR 0017 slice C). The repo convention (0011/0013):
// every content-bearing PocketBase TextField must state an explicit `Max` — a bare TextField
// validates at PocketBase's implicit 5,000-char default and silently rejects longer bodies. This
// guard scans the librarian collections and fails on any content TextField that neither carries an
// inline `Max:` nor is capped by a migration nor is on the exemption list, so a NEW uncapped
// content field is a red build that names the field.
//
//   node scripts/check-textfield-max.mjs             scan; exit 1 on any uncapped field, exit 0 clean
//   node scripts/check-textfield-max.mjs --self-test seed a fixture + assert detection (mirrors
//                                                    check-neutrality.mjs --self-test)
//
// Runs under plain Node (no deps), like check-kits.mjs / check-neutrality.mjs.
//
// ── how a field is considered "capped" ──
// A bare `core.TextField{Name: "x"}` PASSES iff any of:
//   1. inline `Max:` in the literal                    — the convention, stated at declaration.
//   2. a MIGRATION caps it at runtime — the field appears in a `[]struct{ coll, field ...}` cap
//      table inside a file that assigns `.Max =` (0011 widened three bodies; 0020 caps seven).
//      The DECLARATION stays bare by design (0011/0020 set Max at migrate time, on fresh stores
//      too), so the guard reads the cap tables rather than demanding a redundant inline Max.
//   3. it is on the two-category EXEMPTION list below.
//
// ── the two-category exemption list (source-documented, per plan.md Open Q4) ──
//   (1) PERMANENTLY-SHORT STRUCTURAL fields — never expected to carry a body, by role
//       (path, checksum, run ids, timestamps, provider/model labels, ...). Fine forever.
//   (2) KNOWN-BUT-DEFERRED CONTENT DEBT — content-adjacent `files` fields NOT swept by 0020's
//       migration (files.desk/doctype/status/synopsis/origin/graduated_to). Debt, pick it up
//       later; see the issue's Out of scope + plan.md.

import { readdirSync, readFileSync, existsSync, mkdtempSync, mkdirSync, writeFileSync, rmSync } from "node:fs";
import { dirname, join, relative } from "node:path";
import { fileURLToPath } from "node:url";
import { tmpdir } from "node:os";

const REPO_ROOT = join(dirname(fileURLToPath(import.meta.url)), "..");
const COLLECTIONS_DIR = join(
  REPO_ROOT,
  "internal", "modules", "librarian", "collections",
);

// (1) permanently-short structural fields — bare field names, exempt across every collection.
const STRUCTURAL_EXEMPT = new Set([
  "path", "checksum", "git_last_commit", "fm_created", "fm_updated", "run_id", "rule",
  "patrol_run", "desk", "provider", "model", "run_label", "tool_call_id", "tool_name",
  "source", "key", "name", "original_checksum", "new_path", "resolved_run",
]);

// (2) known-but-deferred content debt — qualified `collection.field`, NOT swept by 0020.
const DEBT_EXEMPT = new Set([
  "files.desk", "files.doctype", "files.status", "files.synopsis", "files.origin",
  "files.graduated_to",
]);

/** Collect (collection.field) pairs a migration caps at runtime: two-string tuples in the cap
 *  tables of any file that assigns `.Max =` (the widen/cap migrations 0011 + 0020). */
function collectMigrationCapped(text) {
  const capped = new Set();
  if (!/\.Max\s*=/.test(text)) return capped; // not a runtime cap migration → no cap table
  const tupleRe = /\{\s*"([^"]+)"\s*,\s*"([^"]+)"/g;
  let m;
  while ((m = tupleRe.exec(text)) !== null) capped.add(`${m[1]}.${m[2]}`);
  return capped;
}

/** Scan one collections/*.go file's TEXT for bare (Max-less) content TextFields. `capped` is the
 *  cross-file migration-capped set. Returns [{coll, field, line}]. */
function scanFileText(text, capped) {
  const out = [];
  let currentColl = "";
  const lines = text.split("\n");
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    // track the collection this field belongs to: nearest preceding create/lookup by name.
    const mColl = line.match(/NewBaseCollection\("([^"]+)"\)/) ||
      line.match(/FindCollectionByNameOrId\("([^"]+)"\)/);
    if (mColl) currentColl = mColl[1];

    // every core.TextField{...} literal on this line (declarations are single-line in this tree).
    const litRe = /core\.TextField\{([^}]*)\}/g;
    let lit;
    while ((lit = litRe.exec(line)) !== null) {
      const body = lit[1];
      if (/\bMax\s*:/.test(body)) continue; // (1) inline Max → satisfied
      const mName = body.match(/\bName\s*:\s*"([^"]+)"/);
      if (!mName) continue; // no static Name (unusual) — nothing to key on
      const field = mName[1];
      const qualified = `${currentColl}.${field}`;
      if (capped.has(qualified)) continue;         // (2) capped by a migration
      if (STRUCTURAL_EXEMPT.has(field)) continue;  // exemption cat. 1
      if (DEBT_EXEMPT.has(qualified)) continue;    // exemption cat. 2
      out.push({ coll: currentColl, field, line: i + 1 });
    }
  }
  return out;
}

/** scanDir scans every non-test *.go under dir. Two passes: build the migration-capped set across
 *  all files first, then flag bare content TextFields. Returns [{file, coll, field, line}]. */
function scanDir(dir) {
  const files = readdirSync(dir)
    .filter((n) => n.endsWith(".go") && !n.endsWith("_test.go"))
    .sort();
  const texts = new Map();
  const capped = new Set();
  for (const name of files) {
    const text = readFileSync(join(dir, name), "utf8");
    texts.set(name, text);
    for (const c of collectMigrationCapped(text)) capped.add(c);
  }
  const violations = [];
  for (const name of files) {
    for (const v of scanFileText(texts.get(name), capped)) {
      violations.push({ file: name, ...v });
    }
  }
  return violations;
}

function report(violations, dirLabel) {
  console.error(`check-textfield-max: FAIL — ${violations.length} uncapped content TextField(s) in ${dirLabel}:`);
  for (const v of violations) {
    console.error(`  ${v.file}:${v.line}: ${v.coll}.${v.field} has no explicit Max`);
    console.error(`      → set an explicit Max: in the literal, cap it in a migration, or (if structural/deferred) add it to the exemption list in scripts/check-textfield-max.mjs`);
  }
}

function runScan() {
  if (!existsSync(COLLECTIONS_DIR)) {
    console.error(`check-textfield-max: FAIL — collections dir not found at ${relative(REPO_ROOT, COLLECTIONS_DIR)}`);
    process.exit(1);
  }
  const violations = scanDir(COLLECTIONS_DIR);
  if (violations.length > 0) {
    report(violations, relative(REPO_ROOT, COLLECTIONS_DIR));
    process.exit(1);
  }
  console.log(`check-textfield-max: OK — every content TextField under ${relative(REPO_ROOT, COLLECTIONS_DIR)} is capped (inline Max, migration cap table, or a documented exemption).`);
}

/** runSelfTest seeds a throwaway collections dir with (a) a bare, non-exempt content TextField that
 *  MUST be flagged, (b) an inline-Max field, (c) a structural-exempt field, (d) a migration-capped
 *  field, all of which must PASS — then asserts the scanner flags exactly (a). Mirrors
 *  check-neutrality.mjs --self-test: prove seeded-violation detection without a red repo tree. */
function runSelfTest() {
  const root = mkdtempSync(join(tmpdir(), "ds-textfield-max-"));
  try {
    mkdirSync(root, { recursive: true });
    // A creation migration: one seeded uncapped field (must flag) alongside fields that must pass.
    writeFileSync(
      join(root, "0001_seed.go"),
      [
        "package collections",
        "func init() {",
        '\tc := core.NewBaseCollection("seed_coll")',
        '\tc.Fields.Add(&core.TextField{Name: "seeded_uncapped_body"})', // (a) MUST flag
        '\tc.Fields.Add(&core.TextField{Name: "inline", Max: 2000})',    // (b) inline Max → pass
        '\tc.Fields.Add(&core.TextField{Name: "checksum"})',             // (c) structural → pass
        '\tc.Fields.Add(&core.TextField{Name: "migrated_body"})',        // (d) migration-capped → pass
        "}",
        "",
      ].join("\n"),
    );
    // A cap migration that caps seed_coll.migrated_body at runtime.
    writeFileSync(
      join(root, "0002_seed_caps.go"),
      [
        "package collections",
        "var seedCaps = []struct{ coll, field string }{",
        '\t{"seed_coll", "migrated_body"},',
        "}",
        "func init() {",
        "\ttf.Max = 50000",
        "}",
        "",
      ].join("\n"),
    );

    const v = scanDir(root);
    const flagged = v.map((x) => `${x.coll}.${x.field}`);

    const expect = {
      "flags the seeded uncapped field": flagged.includes("seed_coll.seeded_uncapped_body"),
      "does NOT flag an inline-Max field": !flagged.includes("seed_coll.inline"),
      "does NOT flag a structural-exempt field": !flagged.includes("seed_coll.checksum"),
      "does NOT flag a migration-capped field": !flagged.includes("seed_coll.migrated_body"),
      "flags exactly one field": v.length === 1,
    };

    let ok = true;
    console.log("check-textfield-max --self-test:");
    for (const [k, pass] of Object.entries(expect)) {
      console.log(`  [${pass ? "OK" : "FAIL"}] ${k}`);
      ok = ok && pass;
    }
    if (!ok) {
      console.error("\nself-test FAILED — the guard did not behave as specified.");
      console.error(`  flagged: ${JSON.stringify(flagged)}`);
      process.exit(1);
    }
    console.log(`  → detection verified (1 seeded violation caught, 3 legitimate fields cleared).`);
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
}

if (process.argv.includes("--self-test")) runSelfTest();
else runScan();

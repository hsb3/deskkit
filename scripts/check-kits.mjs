#!/usr/bin/env node
// Kit-manifest drift guard. The SOP kit library lives under kits/, indexed by the root
// kits.yaml manifest (desk decision 0013 S4(a); ported per desk-standard #49). This guard fails
// if the manifest and the on-disk tree disagree in EITHER direction:
//   - a manifest kit whose dir is missing, or that lists a file not on disk   → "missing"
//   - a kit dir on disk not named in the manifest                            → "untracked dir"
//   - a *.md file on disk under kits/ not listed by its kit's `files`         → "untracked file"
// So removing a kit/file/manifest row, or dropping a stray file in, is a red build that NAMES
// the drift. Runs under plain Node (no deps), like the other scripts/ guards.
//
//   node scripts/check-kits.mjs   scan; exit 1 on any drift, exit 0 when manifest == tree

import { readFileSync, readdirSync, existsSync, statSync } from "node:fs";
import { dirname, join, relative } from "node:path";
import { fileURLToPath } from "node:url";

const REPO_ROOT = join(dirname(fileURLToPath(import.meta.url)), "..");
const KITS_DIR = join(REPO_ROOT, "kits");
const MANIFEST = join(REPO_ROOT, "kits.yaml");

// ── static origin-vault blocklist (the real neutrality backstop for #49) ──
// The neutrality lint's identity denylist derives from the *profile*, which ships as placeholders,
// so it cannot by itself catch a personal name or a private vault path carried verbatim from the
// origin vault (nine such wikilinks shipped in the first pass — ADR 0006 correction). This guard
// scans the shipped kit/schema surfaces for a fixed set of known origin-deployment identifiers and
// fails, naming the file, on any hit. Most are case-insensitive; `Burden` is case-sensitive so the
// common word "ops burden" is not a false positive.
const BLOCKLIST_SURFACES = ["kits", "schema"];
const BLOCKLIST_CI = ["headcase", "_headcase", "hsb-2026", "1_Engineering", "Henry", "hsb3", "FunctionForm", "ai_kb", "iCloud", "/Users/"];
const BLOCKLIST_CS = ["Burden"];
const BLOCKLIST_TEXT_EXT = new Set([".md", ".yaml", ".yml", ".json", ".txt"]);

// ── minimal parse of kits.yaml's deliberately-flat shape ──
// A list of `- dir: X` blocks under `kits:`, each with `type:`, optional flag lines, and a
// one-line flow list `files: [a.md, b.md]`. This is the exact structure the manifest commits to
// (see the header of kits.yaml); anything richer should graduate to a real YAML dependency.
function parseManifest(text) {
  const kits = [];
  let cur = null;
  for (const rawLine of text.split("\n")) {
    const line = rawLine.replace(/\s+#.*$/, ""); // strip trailing comments
    const mDir = line.match(/^\s*-\s*dir:\s*(\S+)\s*$/);
    if (mDir) {
      cur = { dir: mDir[1], files: [] };
      kits.push(cur);
      continue;
    }
    if (!cur) continue;
    const mFiles = line.match(/^\s*files:\s*\[([^\]]*)\]\s*$/);
    if (mFiles) {
      cur.files = mFiles[1].split(",").map((s) => s.trim()).filter(Boolean);
      continue;
    }
    const mType = line.match(/^\s*type:\s*(\S+)\s*$/);
    if (mType) cur.type = mType[1];
  }
  return kits;
}

const problems = [];

if (!existsSync(MANIFEST)) {
  console.error(`check-kits: FAIL — manifest not found at kits.yaml`);
  process.exit(1);
}
const manifest = parseManifest(readFileSync(MANIFEST, "utf8"));
if (manifest.length === 0) problems.push("kits.yaml parsed to zero kits (malformed manifest?)");

// forward: every manifest kit + file must exist on disk
const manifestDirs = new Set();
for (const kit of manifest) {
  if (!kit.dir) { problems.push("a manifest entry has no `dir`"); continue; }
  manifestDirs.add(kit.dir);
  if (!kit.type) problems.push(`${kit.dir}: manifest entry has no \`type\``);
  const dirPath = join(KITS_DIR, kit.dir);
  if (!existsSync(dirPath) || !statSync(dirPath).isDirectory()) {
    problems.push(`${kit.dir}: manifest kit dir is missing on disk (kits/${kit.dir}/)`);
    continue;
  }
  if (kit.files.length === 0) problems.push(`${kit.dir}: manifest lists no files`);
  for (const f of kit.files) {
    if (!existsSync(join(dirPath, f))) {
      problems.push(`${kit.dir}/${f}: listed in manifest but missing on disk`);
    }
  }
  // reverse (per kit): every *.md on disk must be listed
  const onDisk = readdirSync(dirPath).filter((n) => n.endsWith(".md"));
  for (const f of onDisk) {
    if (!kit.files.includes(f)) {
      problems.push(`${kit.dir}/${f}: present on disk but not listed in the manifest`);
    }
  }
}

// reverse (whole tree): every kit dir on disk must be in the manifest
if (existsSync(KITS_DIR)) {
  for (const name of readdirSync(KITS_DIR)) {
    const p = join(KITS_DIR, name);
    if (!statSync(p).isDirectory()) continue; // README.md etc. at kits/ root are fine
    if (!manifestDirs.has(name)) {
      problems.push(`kits/${name}/: kit dir on disk not named in the manifest`);
    }
  }
}

// ── static blocklist scan over the shipped kit/schema surfaces ──
function* walkText(dir) {
  let entries;
  try { entries = readdirSync(dir, { withFileTypes: true }); } catch { return; }
  for (const e of entries) {
    const abs = join(dir, e.name);
    if (e.isDirectory()) { yield* walkText(abs); continue; }
    if (!e.isFile()) continue;
    const dot = e.name.lastIndexOf(".");
    const ext = dot === -1 ? "" : e.name.slice(dot);
    if (BLOCKLIST_TEXT_EXT.has(ext)) yield abs;
  }
}

const blockHits = [];
for (const surface of BLOCKLIST_SURFACES) {
  const base = join(REPO_ROOT, surface);
  if (!existsSync(base)) continue;
  for (const abs of walkText(base)) {
    const rel = relative(REPO_ROOT, abs).split("\\").join("/");
    const lines = readFileSync(abs, "utf8").split("\n");
    for (let i = 0; i < lines.length; i++) {
      const line = lines[i];
      const lower = line.toLowerCase();
      for (const term of BLOCKLIST_CI) {
        if (lower.includes(term.toLowerCase())) blockHits.push(`${rel}:${i + 1}: blocked term "${term}"`);
      }
      for (const term of BLOCKLIST_CS) {
        if (line.includes(term)) blockHits.push(`${rel}:${i + 1}: blocked term "${term}"`);
      }
    }
  }
}

if (problems.length > 0 || blockHits.length > 0) {
  if (problems.length > 0) {
    console.error(`check-kits: FAIL — ${problems.length} manifest/tree drift(s):`);
    for (const p of problems) console.error(`  ${p}`);
  }
  if (blockHits.length > 0) {
    console.error(`check-kits: FAIL — ${blockHits.length} origin-vault blocklist hit(s) in ${BLOCKLIST_SURFACES.join(" + ")}/:`);
    for (const h of blockHits) console.error(`  ${h}`);
  }
  process.exit(1);
}

const fileCount = manifest.reduce((n, k) => n + k.files.length, 0);
console.log(`check-kits: OK — ${manifest.length} kits / ${fileCount} files; manifest == kits/ tree; blocklist clean over ${BLOCKLIST_SURFACES.join(" + ")}/.`);

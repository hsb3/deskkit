#!/usr/bin/env node
// D8 — the identity-neutrality lint (M-05 surface iii, "the graduation lint").
//
// Enforces 0013 item 9: no shipped artifact may hardcode a *deployment's* identity
// (a person, org, repo, issue, or board). Personalization comes from the M-05 profile;
// the remedy for a flagged literal is always "move it into _knowledge/profile.yaml and
// reference it via {{profile.<path>}}", never "delete the name".
//
// Scans the shipped surface — `plugins/`, `librarian/`, and `kits/` — for two token families:
//   1. Profile-value occurrences (self-closing denylist): every real-identity scalar in
//      _knowledge/profile.yaml (or, when only the example exists, the example's real values)
//      is a denylist entry; any occurrence in a shipped file fails.
//   2. Structural identifier patterns, independent of any profile:
//        - bare issue refs        #\d+   (reusing the librarian's ISSUE_REF_RE semantics:
//                                          reject a match preceded by a word char or `&`)
//        - github repo identity    https://github.com/o/r , git@github.com:o/r  (URL/SSH only —
//                                          see the "host form" note below)
//        - board/project refs      "project N" / "projects/N"
//
// A match PASSES (sanctioned escape) iff it is inside a {{profile.<path>}} / {{env.<VAR>}}
// placeholder, or lives in an allowlisted path / token (schema/neutrality-lint.allow).
//
// Runs under plain Node (no deps), like the other scripts/ guards:
//   node scripts/check-neutrality.mjs             scan the tree; exit 1 on any violation
//   node scripts/check-neutrality.mjs --self-test seed a fixture + assert detection (AC5 down)
//
// ── Deliberate deviation from the spec's literal pattern list (documented for review) ──
// m-05 lists a bare, scheme-less `github.com/owner/repo` pattern. In a Go repo that form is
// the *import-path* form: `github.com/hsb3/desk-standard/librarian/...` (the module's own,
// token-scoped-allowlisted, hosting path — schema/neutrality-lint.allow) and every third-party
// dependency (`github.com/pocketbase/pocketbase`, …) are
// bare `github.com/o/r`. Matching it flags hundreds of legitimate imports, making D8's
// "returns zero on a clean tree" unsatisfiable — the *identical* collision the spec itself
// invokes to drop bare hostless slugs (m-05 "Token patterns it flags", the owner/repo note).
// So the same accepted-residual-gap reasoning is extended one step: the scheme-less
// `github.com/o/r` import form is NOT structurally matched; URL and SSH-remote forms (which do
// not collide with imports) are. A deployment's real owner in bare form is still caught by the
// profile-value denylist (family 1), which is the spec's designated primary mechanism.
//
// Known latent false positive (review N1): family 2d (`owner/repo#N`) also matches numeric
// path anchors like `path/to/guide.md#42`. None exist in the scanned tree today; if a shipped
// doc ever needs one, add a targeted `neutrality-lint.allow` entry rather than loosening 2d.

import { readdirSync, readFileSync, statSync, existsSync, mkdtempSync, mkdirSync, writeFileSync, rmSync } from "node:fs";
import { dirname, join, relative } from "node:path";
import { fileURLToPath } from "node:url";
import { tmpdir } from "node:os";

const REPO_ROOT = join(dirname(fileURLToPath(import.meta.url)), "..");
// The shipped, identity-bearing surface: the distribution bundles (plugins/), the librarian, and
// the ported SOP kit library (kits/ — the 0013 S4(a) template surface). All carry the
// identity-neutrality constraint; scripts/ + CHANGELOG.md + docs/ are authoring surfaces and stay
// outside. `plugin` (the retired TS lane) stays listed deliberately: scanTree skips a missing dir,
// so the entry costs nothing and re-arms the scan if that path ever comes back.
const SCAN_DIRS = ["plugin", "plugins", "librarian", "kits"];
const ALLOW_FILE = join("schema", "neutrality-lint.allow");

const EXCLUDE_DIR_NAMES = new Set(["node_modules", "dist", ".git"]);
// Generated/dependency manifests + lockfiles: build inputs, not authored identity surface.
const EXCLUDE_BASENAMES = new Set(["go.sum", "go.mod", "bun.lock", "package-lock.json", "pnpm-lock.yaml", "yarn.lock"]);
const TEXT_EXT = new Set([
  ".ts", ".tsx", ".js", ".mjs", ".cjs", ".go", ".md", ".txt",
  ".yaml", ".yml", ".json", ".sh", ".toml",
]);

// Desk-convention vocabulary — enum values + default entity-dir paths from schema v1. These
// are the desk's own generic words (they legitimately appear across shipped code); they are
// NOT a deployment's identity, so they never become denylist entries even when a real profile
// happens to hold the default value.
const STRUCTURAL_VOCAB = new Set([
  "anthropic", "openai", "gemini", "github-projects",
  "primary", "secondary", "conventional", "explanatory", "terse",
  "_structure/decisions", "tasks", "analyses", "journal",
  "_meta/secrets", "_meta/handoff.md", "_knowledge", ".", "true", "false",
]);

// ───────────────────────────── denylist derivation ─────────────────────────────

/** Collect scalar string values from a JSON value tree, recursively. */
function collectJsonScalars(v, out) {
  if (v === null) return;
  if (typeof v === "string") out.push(v);
  else if (typeof v === "number" || typeof v === "boolean") out.push(String(v));
  else if (Array.isArray(v)) for (const x of v) collectJsonScalars(x, out);
  else if (typeof v === "object") for (const k of Object.keys(v)) collectJsonScalars(v[k], out);
}

/** Line-based scalar-value extraction for block YAML (profiles are shallow, schema-constrained). */
function collectYamlScalars(text, out) {
  for (let line of text.split("\n")) {
    // a "key: value" mapping, or a "- value" list item
    let m = line.match(/^\s*[\w.\-/]+:\s*(.+?)\s*$/) || line.match(/^\s*-\s+(.+?)\s*$/);
    if (!m) continue;
    let val = m[1];
    // unwrap a single layer of quotes
    if ((val.startsWith('"') && val.endsWith('"')) || (val.startsWith("'") && val.endsWith("'"))) {
      val = val.slice(1, -1);
    }
    // skip block/flow scalars, empty collections, nulls, anchors/aliases
    if (val === "" || val === "{}" || val === "[]" || val === "|" || val === ">" ||
        val === "~" || val === "null" || val.startsWith("{") || val.startsWith("[") ||
        val.startsWith("&") || val.startsWith("*") || val.startsWith("#")) continue;
    out.push(val);
  }
}

/**
 * buildDenylist reads a profile and returns the set of real-identity literals to flag.
 * Filters out placeholders, generic-example values, desk-convention vocabulary, and
 * collision-prone short/numeric tokens — so the derived denylist holds only values a real
 * deployment would personalize.
 */
function buildDenylist(profilePath) {
  const raw = readFileSync(profilePath, "utf8");
  const scalars = [];
  if (profilePath.endsWith(".json")) {
    collectJsonScalars(JSON.parse(raw), scalars);
  } else if (profilePath.endsWith(".md")) {
    // markdown-with-frontmatter: take the first --- fenced block as YAML
    const lines = raw.split("\n");
    if ((lines[0] ?? "").trim() === "---") {
      const body = [];
      for (let i = 1; i < lines.length; i++) {
        if ((lines[i] ?? "").trim() === "---") break;
        body.push(lines[i]);
      }
      collectYamlScalars(body.join("\n"), scalars);
    }
  } else {
    collectYamlScalars(raw, scalars);
  }

  const deny = new Set();
  for (const v of scalars) {
    if (typeof v !== "string") continue;
    if (v.length < 4) continue;                      // too short / collision-prone (kills ".")
    if (v.includes("<") || v.includes(">")) continue; // placeholder, e.g. "<display name>"
    if (/example/i.test(v)) continue;                 // generic example value
    if (/^owner\/repo(-[\w.-]+)?$/.test(v)) continue; // canonical placeholder slug family
    if (/^\d{1,3}$/.test(v)) continue;                // small number: too collision-prone
    if (STRUCTURAL_VOCAB.has(v.toLowerCase())) continue;
    deny.add(v);
  }
  return deny;
}

// ───────────────────────────── allowlist ─────────────────────────────

/** Parse schema/neutrality-lint.allow into { fileGlobs, tokenRules:[{glob,token}] }. */
function loadAllowlist(root) {
  const path = join(root, ALLOW_FILE);
  const fileGlobs = [];
  const tokenRules = [];
  if (!existsSync(path)) return { fileGlobs, tokenRules };
  for (const line of readFileSync(path, "utf8").split("\n")) {
    const s = line.trim();
    if (s === "" || s.startsWith("#")) continue;
    const bar = s.indexOf("|");
    if (bar === -1) {
      fileGlobs.push(globToRe(s));
    } else {
      tokenRules.push({ glob: globToRe(s.slice(0, bar).trim()), token: s.slice(bar + 1).trim() });
    }
  }
  return { fileGlobs, tokenRules };
}

/** Minimal glob → RegExp: supports ** (any path) and * (within a segment). Repo-relative, /. */
function globToRe(glob) {
  let re = "^";
  for (let i = 0; i < glob.length; i++) {
    const c = glob[i];
    if (c === "*") {
      if (glob[i + 1] === "*") { re += ".*"; i++; if (glob[i + 1] === "/") i++; }
      else re += "[^/]*";
    } else if (".+?()[]{}^$|\\".includes(c)) {
      re += "\\" + c;
    } else {
      re += c;
    }
  }
  return new RegExp(re + "$");
}

// ───────────────────────────── detectors ─────────────────────────────

const GH_RE = /(https?:\/\/github\.com\/[\w.-]+\/[\w.-]+|git@github\.com:[\w.-]+\/[\w.-]+)/g;
const PROJECT_RE = /\bproject\s+\d+|projects\/\d+/gi;
const PLACEHOLDER_RE = /\{\{\s*(?:profile|env)\.[^}]*\}\}/g;

/** Spans (start,end) of {{profile.…}} / {{env.…}} placeholders on a line — the escape zones. */
function placeholderSpans(line) {
  const spans = [];
  let m;
  PLACEHOLDER_RE.lastIndex = 0;
  while ((m = PLACEHOLDER_RE.exec(line)) !== null) spans.push([m.index, m.index + m[0].length]);
  return spans;
}
function inSpan(idx, spans) {
  return spans.some(([a, b]) => idx >= a && idx < b);
}

const SUGGEST = {
  issue: "avoid hardcoded issue numbers — reference the board via {{profile.repos.shorthand.issue_default}} / {{profile.board.number}}",
  github: "move the repo identity into {{profile.repos.default}}",
  project: "reference the board via {{profile.board.number}}",
  profile: "matches a profile value — move the literal into _knowledge/profile.yaml and reference {{profile.<path>}}",
};

/** Scan one line; push {line, col, token, suggestion} violations. */
function scanLine(text, lineNo, denylist, out) {
  const spans = placeholderSpans(text);

  // family 1: profile-value denylist
  for (const token of denylist) {
    let from = 0;
    for (;;) {
      const idx = text.indexOf(token, from);
      if (idx === -1) break;
      from = idx + token.length;
      // for purely-alphanumeric tokens, require non-word neighbours (reduce substring noise)
      if (/^[A-Za-z0-9]+$/.test(token)) {
        const before = text[idx - 1] ?? "";
        const after = text[idx + token.length] ?? "";
        if (/[A-Za-z0-9]/.test(before) || /[A-Za-z0-9]/.test(after)) continue;
      }
      if (inSpan(idx, spans)) continue;
      out.push({ line: lineNo, col: idx + 1, token, suggestion: SUGGEST.profile });
    }
  }

  // family 2a: bare issue refs #\d+ (RE2-style preceding-byte rejection)
  {
    const re = /#\d+/g;
    let m;
    while ((m = re.exec(text)) !== null) {
      const before = text[m.index - 1] ?? "";
      if (/[\w&]/.test(before)) continue; // e.g. wb#9, &#39; — not a bare issue ref
      if (inSpan(m.index, spans)) continue;
      out.push({ line: lineNo, col: m.index + 1, token: m[0], suggestion: SUGGEST.issue });
    }
  }

  // family 2b: github URL / SSH remote
  {
    GH_RE.lastIndex = 0;
    let m;
    while ((m = GH_RE.exec(text)) !== null) {
      if (inSpan(m.index, spans)) continue;
      out.push({ line: lineNo, col: m.index + 1, token: m[0], suggestion: SUGGEST.github });
    }
  }

  // family 2d: qualified issue refs owner/repo#N — a real identity even though the bare
  // hostless slug alone is deliberately unmatched (Go import-path collision, header note),
  // and family 2a skips the #N because a word character precedes it.
  {
    const re = /[A-Za-z0-9][\w.-]*\/[\w.-]+#\d+/g;
    let m;
    while ((m = re.exec(text)) !== null) {
      if (inSpan(m.index, spans)) continue;
      out.push({ line: lineNo, col: m.index + 1, token: m[0], suggestion: SUGGEST.issue });
    }
  }

  // family 2c: project / projects number
  {
    PROJECT_RE.lastIndex = 0;
    let m;
    while ((m = PROJECT_RE.exec(text)) !== null) {
      if (inSpan(m.index, spans)) continue;
      out.push({ line: lineNo, col: m.index + 1, token: m[0], suggestion: SUGGEST.project });
    }
  }
}

// ───────────────────────────── tree walk ─────────────────────────────

function* walk(dir) {
  let entries;
  try { entries = readdirSync(dir, { withFileTypes: true }); } catch { return; }
  for (const e of entries) {
    const abs = join(dir, e.name);
    if (e.isDirectory()) {
      if (EXCLUDE_DIR_NAMES.has(e.name)) continue;
      yield* walk(abs);
    } else if (e.isFile()) {
      if (EXCLUDE_BASENAMES.has(e.name)) continue;
      const dot = e.name.lastIndexOf(".");
      const ext = dot === -1 ? "" : e.name.slice(dot);
      if (!TEXT_EXT.has(ext)) continue;
      yield abs;
    }
  }
}

/**
 * scanTree walks scanDirs under root, applying denylist + structural detectors, honouring the
 * allowlist. Returns an array of {file (repo-relative), line, col, token, suggestion}.
 */
function scanTree(root, denylist, allow, scanDirs) {
  const violations = [];
  for (const d of scanDirs) {
    const base = join(root, d);
    if (!existsSync(base)) continue;
    for (const abs of walk(base)) {
      const rel = relative(root, abs).split("\\").join("/");
      if (allow.fileGlobs.some((re) => re.test(rel))) continue; // whole-file exempt
      const tokenRules = allow.tokenRules.filter((r) => r.glob.test(rel));
      const raw = readFileSync(abs, "utf8");
      const lines = raw.split("\n");
      const fileHits = [];
      for (let i = 0; i < lines.length; i++) scanLine(lines[i], i + 1, denylist, fileHits);
      for (const h of fileHits) {
        if (tokenRules.some((r) => r.token === h.token)) continue; // token-scoped exempt
        violations.push({ file: rel, ...h });
      }
    }
  }
  return violations;
}

function resolveProfile(root) {
  const dir = join(root, "_knowledge");
  for (const name of ["profile.yaml", "profile.yml", "profile.json", "profile.md"]) {
    const p = join(dir, name);
    if (existsSync(p)) return { path: p, isExample: false };
  }
  const ex = join(dir, "profile.example.yaml");
  if (existsSync(ex)) return { path: ex, isExample: true };
  return null;
}

function report(violations) {
  for (const v of violations) {
    console.error(`  ${v.file}:${v.line}:${v.col}: ${v.token}\n      → ${v.suggestion}`);
  }
}

// ───────────────────────────── modes ─────────────────────────────

function runScan(root) {
  const prof = resolveProfile(root);
  const denylist = prof ? buildDenylist(prof.path) : new Set();
  const allow = loadAllowlist(root);
  const src = prof ? (prof.isExample ? "profile.example.yaml (placeholders → denylist likely empty)" : "profile.yaml") : "(no profile found)";
  const violations = scanTree(root, denylist, allow, SCAN_DIRS);
  if (violations.length > 0) {
    console.error(`neutrality: FAIL — ${violations.length} hardcoded-identity violation(s):\n`);
    report(violations);
    console.error(`\ndenylist source: ${src} (${denylist.size} literal(s)); remedy: move the literal into _knowledge/profile.yaml and reference {{profile.<path>}}.`);
    process.exit(1);
  }
  console.log(`neutrality: OK — ${SCAN_DIRS.join(" + ")} clean (denylist source: ${src}, ${denylist.size} literal(s)).`);
}

/**
 * runSelfTest seeds a throwaway tree with a *real* profile + a shipped file that hardcodes its
 * identity, and asserts the lint (a) flags every seeded identifier and (b) does NOT flag a
 * value inside a {{profile.…}} placeholder. Proves AC5's "fails on a seeded identifier"
 * direction without leaving a violating file in the repo. Exit 0 = detection works.
 */
function runSelfTest() {
  const root = mkdtempSync(join(tmpdir(), "ds-neutrality-"));
  try {
    mkdirSync(join(root, "_knowledge"), { recursive: true });
    writeFileSync(
      join(root, "_knowledge", "profile.yaml"),
      [
        "schema_version: 1",
        "identity:",
        '  name: "Ada Lovelace"',
        "  github:",
        '    personal: "adalovelace"',
        "repos:",
        '  default: "acme-corp/secret-project"',
        "board:",
        "  number: 4242",
        "",
      ].join("\n"),
    );
    mkdirSync(join(root, "plugin"), { recursive: true });
    // a shipped-style file that leaks the deployment identity every which way
    writeFileSync(
      join(root, "plugin", "leaky.md"),
      [
        "# guide",
        "Maintained by adalovelace at https://github.com/acme-corp/secret-project.",
        "Filed as #42 on project 4242.",
        "Slug: acme-corp/secret-project.",
        "Qualified ref: acme-corp/secret-project#7 must be caught.",
        "OK line: see {{profile.repos.default}} and issue {{profile.board.number}}.",
        "",
      ].join("\n"),
    );

    const denylist = buildDenylist(join(root, "_knowledge", "profile.yaml"));
    const allow = loadAllowlist(root);
    const v = scanTree(root, denylist, allow, ["plugin"]);
    const tokens = v.map((x) => x.token);

    const expect = {
      "denylist: handle 'adalovelace'": tokens.includes("adalovelace"),
      "denylist: slug 'acme-corp/secret-project'": tokens.includes("acme-corp/secret-project"),
      "structural: bare issue #42": tokens.includes("#42"),
      "structural: github URL": tokens.some((t) => t.startsWith("https://github.com/acme-corp/secret-project")),
      "structural: project 4242": tokens.some((t) => /project\s+4242/.test(t)),
      "structural: qualified ref owner/repo#7": tokens.includes("acme-corp/secret-project#7"),
    };
    // escape must hold: no violation on the {{profile.…}} line (now line 6)
    const escapeHeld = !v.some((x) => x.line === 6);

    let ok = escapeHeld;
    console.log("neutrality --self-test:");
    for (const [k, pass] of Object.entries(expect)) {
      console.log(`  [${pass ? "OK" : "FAIL"}] ${k}`);
      ok = ok && pass;
    }
    console.log(`  [${escapeHeld ? "OK" : "FAIL"}] escape: {{profile.…}} placeholder not flagged`);
    if (!ok) {
      console.error("\nself-test FAILED — the lint did not behave as specified.");
      report(v);
      process.exit(1);
    }
    console.log(`  → detection verified (${v.length} seeded violation(s) caught).`);
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
}

// ───────────────────────────── entry ─────────────────────────────

if (process.argv.includes("--self-test")) runSelfTest();
else runScan(REPO_ROOT);

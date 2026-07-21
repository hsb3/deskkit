#!/usr/bin/env node
// Query-kind drift guard. The librarian's `query` tool (spec §5.6) advertises its fixed set of
// `kind` values in THREE version-controlled places that must never silently diverge:
//
//   1. the jsonschema struct tag on QueryInput.Kind in
//      librarian/internal/modules/librarian/tools/types.go — what the MCP tool schema advertises
//      to a calling agent (the "One of: ..." token list).
//   2. the spec's `docs/pocket-librarian-v1-spec.md` §5.6 quote of that same struct — a
//      hand-copied fenced ```go block, not generated.
//   3. the `switch in.Kind { ... }` block in `func Query`
//      (librarian/internal/modules/librarian/tools/query.go) — the RUNTIME registry: the actual
//      set of kinds the tool will dispatch (anything else falls through to `default:` and errors).
//
// Why an .mjs guard and not a Go test: the kind list exists only as free-text strings — a
// jsonschema struct-tag description and a spec-markdown quote — with no exported Go slice of
// kinds to reflect on, and a Go test cannot cleanly read spec markdown. Every existing doc-vs-
// source string-agreement guard in this repo is a `scripts/check-*.mjs` (check-prompt-drift.mjs,
// check-tool-surface.mjs, check-workflow-pins.mjs); this follows the same house pattern: plain
// Node, no deps, pure text-scan helpers, a `--self-test` mode that seeds an in-memory mismatch.
//
//   node scripts/check-query-kinds.mjs             # scan the tree — FAIL on any drift
//   node scripts/check-query-kinds.mjs --self-test  # prove the guard still detects seeded drift
//
// Exit 1 on any problem (or, under --self-test, if the guard FAILS to flag a seeded violation).

import { readFileSync } from "node:fs";
import { dirname, join, relative } from "node:path";
import { fileURLToPath } from "node:url";

const REPO_ROOT = join(dirname(fileURLToPath(import.meta.url)), "..");
const TYPES_GO = join(REPO_ROOT, "librarian", "internal", "modules", "librarian", "tools", "types.go");
const SPEC = join(REPO_ROOT, "docs", "pocket-librarian-v1-spec.md");
const QUERY_GO = join(REPO_ROOT, "librarian", "internal", "modules", "librarian", "tools", "query.go");

const rel = (p) => relative(REPO_ROOT, p).split("\\").join("/");

// The struct-tag sentinel shared by types.go (canonical) and the spec's hand-copied quote of it.
// Anchored on CONTENT, never a line number — both files are free to grow/shift around this line.
const SENTINEL = 'jsonschema:"description=One of:';

// ── pure helpers (no file I/O — self-test exercises these directly) ──

// Find every line in `text` containing the sentinel substring. Returns an array of matching lines
// (callers assert there is exactly one — more than one means the tag moved/duplicated and this
// guard's single-line assumption needs revisiting, not silent first-match behavior).
export function findSentinelLines(text) {
  return text.split("\n").filter((line) => line.includes(SENTINEL));
}

// Given a line containing `...One of: <tok> <tok> ...;...`, extract the ordered token list.
export function parseOneOfTokens(line) {
  const m = line.match(/One of:\s*([^;]+);/);
  if (!m) return null;
  return m[1].trim().split(/\s+/).filter(Boolean);
}

// Extract `case "<label>":` string labels from a `switch <marker> {` block in `text`, stopping at
// the first line whose trimmed content starts with `default:`. Excludes any other switch in the
// same file (this file has exactly one `default:` line, which is this switch's).
export function extractSwitchCaseLabels(text, marker) {
  const lines = text.split("\n");
  const startIdx = lines.findIndex((l) => l.includes(marker));
  if (startIdx === -1) return null;
  const labels = [];
  for (let i = startIdx + 1; i < lines.length; i++) {
    const trimmed = lines[i].trim();
    if (trimmed.startsWith("default:")) return labels;
    if (trimmed.startsWith("case ")) {
      for (const m of trimmed.matchAll(/"([^"]+)"/g)) labels.push(m[1]);
    }
  }
  return null; // no `default:` found — block never terminated
}

// Ordered, byte-identical comparison of two token lists (joined-string equality). Returns null
// when equal, or a description of the first divergence.
export function diffOrderedTokens(a, b) {
  const joinedA = a.join(" ");
  const joinedB = b.join(" ");
  if (joinedA === joinedB) return null;
  return { joinedA, joinedB };
}

// Order-independent set comparison. Returns null when the sets are equal, or
// { onlyInA: string[], onlyInB: string[] } naming the asymmetric elements.
export function diffTokenSets(a, b) {
  const setA = new Set(a);
  const setB = new Set(b);
  const onlyInA = a.filter((t) => !setB.has(t));
  const onlyInB = b.filter((t) => !setA.has(t));
  if (onlyInA.length === 0 && onlyInB.length === 0) return null;
  return { onlyInA, onlyInB };
}

// ── self-test: prove the pure compare functions still flag seeded drift ──

if (process.argv.includes("--self-test")) {
  const canonical = [
    "live_files",
    "recent",
    "orphans",
    "uncollapsed",
    "findings",
    "summary",
    "adoption",
    "feedback",
    "search",
    "content",
  ];

  // 1. Matching pair — no drift expected.
  const matchOrdered = diffOrderedTokens(canonical, [...canonical]);
  const matchSet = diffTokenSets(canonical, [...canonical]);
  if (matchOrdered !== null || matchSet !== null) {
    console.error(
      `check-query-kinds: SELF-TEST FAIL — identical lists were reported as drifted (ordered=${JSON.stringify(matchOrdered)}, set=${JSON.stringify(matchSet)}).`,
    );
    process.exit(1);
  }

  // 2. Seeded spec list DROPS one kind ("content") — ordered AND set comparisons must both flag it.
  const droppedList = canonical.filter((k) => k !== "content");
  const droppedOrdered = diffOrderedTokens(canonical, droppedList);
  const droppedSet = diffTokenSets(canonical, droppedList);
  if (droppedOrdered === null) {
    console.error("check-query-kinds: SELF-TEST FAIL — a dropped kind was not flagged by diffOrderedTokens.");
    process.exit(1);
  }
  if (droppedSet === null || !droppedSet.onlyInA.includes("content")) {
    console.error(
      `check-query-kinds: SELF-TEST FAIL — a dropped kind ("content") was not reported by diffTokenSets, got ${JSON.stringify(droppedSet)}.`,
    );
    process.exit(1);
  }

  // 3. Seeded runtime registry ADDS a bogus kind — set comparison must flag it as registry-only.
  const addedList = [...canonical, "bogus_kind"];
  const addedSet = diffTokenSets(canonical, addedList);
  if (addedSet === null || !addedSet.onlyInB.includes("bogus_kind")) {
    console.error(
      `check-query-kinds: SELF-TEST FAIL — an added bogus kind was not reported by diffTokenSets, got ${JSON.stringify(addedSet)}.`,
    );
    process.exit(1);
  }

  // 4. Sanity: findSentinelLines / parseOneOfTokens / extractSwitchCaseLabels round-trip on a
  // seeded snippet, so a regression in the extraction regex itself is also caught here.
  const seededStructLine =
    '\tKind string `json:"kind" jsonschema:"description=One of: alpha beta gamma;required"`';
  const seededTokens = parseOneOfTokens(seededStructLine);
  if (JSON.stringify(seededTokens) !== JSON.stringify(["alpha", "beta", "gamma"])) {
    console.error(
      `check-query-kinds: SELF-TEST FAIL — parseOneOfTokens misparsed a seeded struct-tag line, got ${JSON.stringify(seededTokens)}.`,
    );
    process.exit(1);
  }
  const seededSwitch = [
    "switch in.Kind {",
    'case "alpha":',
    "\tdoAlpha()",
    'case "beta", "gamma":',
    "\tdoBetaGamma()",
    "default:",
    "\tpanic(\"unreachable\")",
    "switch row.OtherKind {", // must NOT be picked up — past the first default:
    'case "zzz":',
  ].join("\n");
  const seededLabels = extractSwitchCaseLabels(seededSwitch, "switch in.Kind {");
  if (JSON.stringify(seededLabels) !== JSON.stringify(["alpha", "beta", "gamma"])) {
    console.error(
      `check-query-kinds: SELF-TEST FAIL — extractSwitchCaseLabels misparsed a seeded switch block, got ${JSON.stringify(seededLabels)}.`,
    );
    process.exit(1);
  }

  console.log(
    "check-query-kinds: self-test OK — a dropped kind, an added bogus kind, and a matching pair are all correctly classified; extraction helpers round-trip on seeded input.",
  );
  process.exit(0);
}

// ── real scan ──

function fail(msg) {
  console.error(`check-query-kinds: FAIL — ${msg}`);
  process.exit(1);
}

function loadOneOfTokens(path, label) {
  let text;
  try {
    text = readFileSync(path, "utf8");
  } catch (err) {
    fail(`could not read ${rel(path)} (${err.message}).`);
  }
  const lines = findSentinelLines(text);
  if (lines.length === 0) {
    fail(`no line containing \`${SENTINEL}\` found in ${label} (${rel(path)}) — did the struct-tag move or get reworded?`);
  }
  if (lines.length > 1) {
    fail(
      `expected exactly one \`${SENTINEL}\` line in ${label} (${rel(path)}), found ${lines.length} — this guard's single-line assumption no longer holds.`,
    );
  }
  const tokens = parseOneOfTokens(lines[0]);
  if (!tokens || tokens.length === 0) {
    fail(`found the sentinel line in ${label} (${rel(path)}) but could not parse an "One of: ...;" token list from it:\n    ${lines[0]}`);
  }
  return tokens;
}

const typesTokens = loadOneOfTokens(TYPES_GO, "the canonical MCP schema (types.go)");
const specTokens = loadOneOfTokens(SPEC, "the spec quote");

const orderedDiff = diffOrderedTokens(typesTokens, specTokens);
if (orderedDiff !== null) {
  console.error(`check-query-kinds: FAIL — the spec's "One of:" quote has drifted from the canonical schema.`);
  console.error(`  canonical (${rel(TYPES_GO)}): ${orderedDiff.joinedA}`);
  console.error(`  spec      (${rel(SPEC)}):      ${orderedDiff.joinedB}`);
  console.error(`  Update the spec's §5.6 QueryInput quote to match types.go's "One of: ..." list verbatim.`);
  process.exit(1);
}

let queryText;
try {
  queryText = readFileSync(QUERY_GO, "utf8");
} catch (err) {
  fail(`could not read ${rel(QUERY_GO)} (${err.message}).`);
}
const registryLabels = extractSwitchCaseLabels(queryText, "switch in.Kind {");
if (registryLabels === null) {
  fail(
    `could not locate a terminated \`switch in.Kind {\` ... \`default:\` block in ${rel(QUERY_GO)} — did func Query's dispatch switch move or lose its default case?`,
  );
}

const setDiff = diffTokenSets(typesTokens, registryLabels);
if (setDiff !== null) {
  console.error(`check-query-kinds: FAIL — the runtime registry (query.go's switch) disagrees with the advertised schema.`);
  if (setDiff.onlyInA.length > 0) {
    console.error(`  advertised but NOT handled by the switch: ${setDiff.onlyInA.join(", ")}`);
  }
  if (setDiff.onlyInB.length > 0) {
    console.error(`  handled by the switch but NOT advertised: ${setDiff.onlyInB.join(", ")}`);
  }
  console.error(`  advertised (${rel(TYPES_GO)}): ${typesTokens.join(", ")}`);
  console.error(`  registry   (${rel(QUERY_GO)}):  ${registryLabels.join(", ")}`);
  process.exit(1);
}

console.log(
  `check-query-kinds: OK — ${typesTokens.length} query kind(s) agree across the canonical schema (${rel(TYPES_GO)}), ` +
    `the spec quote (${rel(SPEC)}), and the runtime registry (${rel(QUERY_GO)}): ${typesTokens.join(", ")}.`,
);
process.exit(0);

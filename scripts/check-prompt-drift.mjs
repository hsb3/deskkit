#!/usr/bin/env node
// Prompt-copy drift guard. ADR 0015 (DESK-37) rules
// git-is-truth for the librarian's system prompt: the //go:embed'd source
// (librarian/templates/librarian-system-prompt.txt) is CANONICAL and the v1 spec quotes it
// "verbatim" in a fenced block. This guard asserts those two version-controlled copies are
// byte-identical, so the embed and its "kept verbatim" spec quote cannot silently diverge —
// the exact class of bug ADR 0015 names as the live proof the split was ungoverned (a stale
// six-tool spec quote shipped against a five-tool embed with nothing to catch it).
//
// It mirrors the existing packaged-artifacts drift guard (.github/workflows/ci.yml "Plugin —
// packaged artifacts drift guard": `bun run package` then `git diff --exit-code -- ../plugins/desk-standard/`)
// — fail red naming the file, don't reinvent a diff format. One governed mechanism per
// version-controlled prompt copy. Runs under plain Node (no deps), like the other scripts/
// guards (check-kits.mjs, check-scaffold-frontmatter.mjs).
//
//   node scripts/check-prompt-drift.mjs   scan; exit 1 on drift, exit 0 when the copies agree
//
// Forward note: when the ADR 0014 desk-persona bundle lands (tracked separately), its
// bundle-markdown prompt copies join prompt governance — either by extending this file's copy
// list or via the bundle's own regenerate+diff guard. The requirement is one governed
// mechanism per copy, not one script.

import { readFileSync } from "node:fs";
import { dirname, join, relative } from "node:path";
import { fileURLToPath } from "node:url";

const REPO_ROOT = join(dirname(fileURLToPath(import.meta.url)), "..");
const EMBED = join(REPO_ROOT, "librarian", "templates", "librarian-system-prompt.txt");
const SPEC = join(REPO_ROOT, "docs", "development", "specs", "pocket-librarian-v1-spec.md");

// The spec quotes the embed under this sentinel line; the very next ```text … ``` fence is the
// "kept verbatim" copy. Locate by content, never by line number — the fence moves as the spec
// grows (it sat near :1435 pre-#114, near :1451 after).
const SENTINEL = "The full embedded-default text (the first-run seed, kept verbatim):";

const rel = (p) => relative(REPO_ROOT, p).split("\\").join("/");

function fail(msg) {
  console.error(`check-prompt-drift: FAIL — ${msg}`);
  process.exit(1);
}

// ── extract the fenced text block that follows the sentinel ──
const specText = readFileSync(SPEC, "utf8");
const specLines = specText.split("\n");

const sentinelIdx = specLines.findIndex((l) => l.includes(SENTINEL));
if (sentinelIdx === -1) {
  fail(`sentinel line not found in ${rel(SPEC)} — expected a line containing:\n  "${SENTINEL}"`);
}

let fenceOpen = -1;
for (let i = sentinelIdx + 1; i < specLines.length; i++) {
  if (specLines[i].trim() === "```text") {
    fenceOpen = i;
    break;
  }
}
if (fenceOpen === -1) {
  fail(`no \`\`\`text fence found after the sentinel (line ${sentinelIdx + 1}) in ${rel(SPEC)}`);
}

let fenceClose = -1;
for (let i = fenceOpen + 1; i < specLines.length; i++) {
  if (specLines[i].trim() === "```") {
    fenceClose = i;
    break;
  }
}
if (fenceClose === -1) {
  fail(`unterminated \`\`\`text fence opened at line ${fenceOpen + 1} in ${rel(SPEC)}`);
}

// The fenced block reproduces the file verbatim: its content lines joined by "\n" plus the
// file's single trailing newline.
const blockLines = specLines.slice(fenceOpen + 1, fenceClose);
const specCopy = blockLines.join("\n") + "\n";
const embedCopy = readFileSync(EMBED, "utf8");

if (specCopy === embedCopy) {
  console.log(
    `check-prompt-drift: OK — ${rel(EMBED)} == the "kept verbatim" fenced block in ` +
      `${rel(SPEC)} (fence at lines ${fenceOpen + 2}-${fenceClose}, ${blockLines.length} lines, byte-identical).`,
  );
  process.exit(0);
}

// ── drift: name the file and the first byte where the two copies diverge ──
let d = 0;
while (d < specCopy.length && d < embedCopy.length && specCopy[d] === embedCopy[d]) d++;
const embedLineNo = embedCopy.slice(0, d).split("\n").length; // 1-based line in the embed
const embedShow = embedCopy.split("\n")[embedLineNo - 1] ?? "<end of file>";
const specShow = blockLines[embedLineNo - 1] ?? "<end of fenced block>";

console.error(`check-prompt-drift: FAIL — the two version-controlled prompt copies diverge.`);
console.error(`  embed (truth): ${rel(EMBED)}:${embedLineNo}`);
console.error(`  spec  (quote): ${rel(SPEC)} fenced block (line ${fenceOpen + 1 + embedLineNo})`);
console.error(`  first divergence at embed line ${embedLineNo}:`);
console.error(`    embed: ${JSON.stringify(embedShow)}`);
console.error(`    spec : ${JSON.stringify(specShow)}`);
console.error(
  `  ADR 0015 keeps these byte-identical; regenerate the spec's "kept verbatim" block from ${rel(EMBED)}.`,
);
process.exit(1);

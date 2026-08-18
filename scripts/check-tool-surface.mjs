#!/usr/bin/env node
// Tool-surface drift guard (the JS half). docs/development/specs/tool-surface.md is the repo's authoritative,
// empirically-derived map of every tool-bearing surface. ADR 0016 rules that
// "tool-surface truth lives in docs/development/specs/tool-surface.md, pinned by a drift guard (script or
// generation) so counts can't rot again" — the way VERSION is pinned to the shipped manifests
// (check-version-sync.mjs) and kits.yaml to the kits/ tree (check-kits.mjs). Before this guard,
// the doc's own closing line ("re-run the probe … the numbers in this doc must match") was a
// remember-to-do-it manual step with nothing behind it.
//
// This guard pins the one count that is cheap and exact to derive in-language:
//   - Librarian CLI (16 base): the AddCommand registrations in librarian/cmd/deskkit/main.go
//     plus the framework system commands.
// It cross-checks that against the number the doc's Summary table states. The gate-dependent MCP
// counts (the live 9 / 10 / 21 / 22 totals and the MCP_MODULES=pm → 12 / MCP_MODULES=profile → 4
// mounts) are NOT re-derived here on purpose: reimplementing Go's two-flag × module gate
// arithmetic in JS would be a second copy that can itself drift. Those are pinned by the Go half
// — TestToolSurfaceDoc_MCPCounts in librarian/internal/core/mcp/tool_surface_doc_test.go — which
// reads the same doc counts and asserts them against the real toolcore gate over the registered
// profile + librarian + pm modules, on the `go test ./...` (make test) lane. This guard asserts
// that Go test's PRESENCE, so removing the MCP-count guard fails `make check`.
//
// It pins COUNTS derived-vs-documented, not the doc's bytes — so an unrelated prose/row edit
// (e.g. re-wording the `findings dispose` row) does NOT trip it; only a real count change does.
// Runs under plain Node (no deps), like the other scripts/ guards.
//
//   node scripts/check-tool-surface.mjs             scan; exit 1 on any drift, exit 0 when doc == source
//   node scripts/check-tool-surface.mjs --self-test seed fixtures + assert the guard fails on a
//                                                   tool add/removed without a doc edit (then passes matched)

import { readFileSync, existsSync } from "node:fs";
import { dirname, join, relative } from "node:path";
import { fileURLToPath } from "node:url";

const REPO_ROOT = join(dirname(fileURLToPath(import.meta.url)), "..");
const DOC = join(REPO_ROOT, "docs", "development", "specs", "tool-surface.md");
const CLI_MAIN = join(REPO_ROOT, "librarian", "cmd", "deskkit", "main.go");
// The Go half that pins the gate-dependent MCP counts; its presence is asserted here.
const GO_GUARD = join(REPO_ROOT, "librarian", "internal", "core", "mcp", "tool_surface_doc_test.go");
const GO_GUARD_FUNC = "TestToolSurfaceDoc_MCPCounts";

const rel = (p) => relative(REPO_ROOT, p).split("\\").join("/");

// ───────────────────────────── source derivations ─────────────────────────────

/**
 * Derive the CLI "base" subcommand count from librarian/cmd/deskkit/main.go, mirroring how the
 * doc's §1 table + Summary "16 base" were reached:
 *   - app.RootCmd.AddCommand(…) call sites in registerToolCommands (the tool/tool-core commands)
 *   - the framework system commands the doc attributes to PocketBase / migratecmd: serve +
 *     superuser (the pbLateCommands slice) and migrate (migratecmd.MustRegister)
 * `help`/`completion` (cobra built-ins, auto-registered — not AddCommand'd) and the `pm` group
 * (registered via registerPMCommands, gated by PM_ENABLED) are excluded, exactly as the doc's
 * "16 base (+ pm group under PM_ENABLED)" excludes them.
 */
function deriveCliCount(src) {
  const add = (src.match(/app\.RootCmd\.AddCommand\(/g) || []).length;
  const late = src.match(/pbLateCommands\s*=\s*\[\]string\{([^}]*)\}/);
  const lateCount = late ? late[1].split(",").map((s) => s.trim()).filter(Boolean).length : 0;
  const migrate = /migratecmd\.MustRegister\(/.test(src) ? 1 : 0;
  return { add, lateCount, migrate, total: add + lateCount + migrate };
}

// ───────────────────────────── doc parsing ─────────────────────────────

/** Split a markdown table row into trimmed cells with emphasis/code markers stripped. */
function tableCells(line) {
  const s = line.trim();
  if (!s.startsWith("|")) return null;
  return s
    .replace(/^\|/, "")
    .replace(/\|$/, "")
    .split("|")
    .map((c) => c.replace(/\*\*/g, "").replace(/`/g, "").trim());
}

/** First cell after the label (index >= 1) whose leading token is an integer, or null. */
function firstIntInRow(line) {
  const cells = tableCells(line);
  if (!cells) return null;
  for (let i = 1; i < cells.length; i++) {
    const tok = cells[i].split(/\s+/)[0];
    if (/^\d+$/.test(tok)) return Number(tok);
  }
  return null;
}

/** The CLI base count the doc states in the Summary table ("16 base (+ pm group …)"). */
function parseDocCliCount(md) {
  const row = md.split("\n").find((l) => l.trim().startsWith("|") && /Librarian CLI subcommands/.test(l));
  return row ? firstIntInRow(row) : null;
}

// ───────────────────────────── the check ─────────────────────────────

/**
 * checkAll runs the full pipeline over raw inputs and returns an array of problem strings (empty
 * = clean). Taking raw sources (not file paths) lets the self-test drive the SAME logic against
 * fixture content.
 */
function checkAll({ goSource, docMd, goGuardPresent }) {
  const problems = [];

  // --- Librarian CLI (16 base) ---
  const cli = deriveCliCount(goSource);
  const cliDoc = parseDocCliCount(docMd);
  if (cliDoc === null) {
    problems.push("CLI surface: could not parse the documented base count (Summary table row) in docs/development/specs/tool-surface.md");
  } else if (cli.total !== cliDoc) {
    problems.push(
      `CLI surface: docs/development/specs/tool-surface.md says ${cliDoc} base, but librarian/cmd/deskkit/main.go derives ` +
        `${cli.total} (${cli.add} AddCommand + ${cli.lateCount} pbLate + ${cli.migrate} migratecmd)`,
    );
  }

  // --- Gate-dependent MCP counts: pinned by the Go half; assert its presence ---
  if (!goGuardPresent) {
    problems.push(
      `MCP-count guard missing: the gate-dependent MCP counts (the live 9/10/21/22 totals, MCP_MODULES=pm → 12, ` +
        `MCP_MODULES=profile → 4) are pinned by ${rel(GO_GUARD)} (${GO_GUARD_FUNC}); that file/function is absent ` +
        `— restore it so the MCP counts stay guarded on the go-test lane`,
    );
  }

  return { problems, cli, cliDoc };
}

// ───────────────────────────── modes ─────────────────────────────

function runScan() {
  for (const p of [DOC, CLI_MAIN]) {
    if (!existsSync(p)) {
      console.error(`check-tool-surface: FAIL — expected file not found: ${rel(p)}`);
      process.exit(1);
    }
  }
  const goGuardPresent = existsSync(GO_GUARD) && readFileSync(GO_GUARD, "utf8").includes(GO_GUARD_FUNC);
  const { problems, cli, cliDoc } = checkAll({
    goSource: readFileSync(CLI_MAIN, "utf8"),
    docMd: readFileSync(DOC, "utf8"),
    goGuardPresent,
  });

  if (problems.length > 0) {
    console.error(`check-tool-surface: FAIL — ${problems.length} tool-surface drift(s):`);
    for (const p of problems) console.error(`  ${p}`);
    console.error(
      `\ndocs/development/specs/tool-surface.md is the pinned source of truth (ADR 0016); reconcile the doc's counts with the ` +
        `tool registrations, then re-run.`,
    );
    process.exit(1);
  }

  console.log(
    `check-tool-surface: OK — CLI ${cli.total} base = ${cli.add} AddCommand + ${cli.lateCount} pbLate + ` +
      `${cli.migrate} migratecmd (doc=${cliDoc}); MCP gated counts (live totals + the MCP_MODULES mounts) ` +
      `pinned by ${GO_GUARD_FUNC} on the go-test lane.`,
  );
}

/**
 * runSelfTest drives checkAll against in-memory fixtures — no repo files touched — proving the
 * mechanical acceptance criterion: the guard FAILS when a tool is added/removed without a matching
 * doc edit, and PASSES when the doc matches. It also proves the Go-guard-presence assertion.
 */
function runSelfTest() {
  // A matched baseline: CLI source deriving 16 (13 AddCommand + 2 pbLate + migrate), and a doc
  // snippet whose count agrees.
  const go16 =
    Array.from({ length: 13 }, () => "\tapp.RootCmd.AddCommand(x)\n").join("") +
    'var pbLateCommands = []string{"serve", "superuser"}\n' +
    "\tmigratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{})\n";
  const go17 = "\tapp.RootCmd.AddCommand(x)\n" + go16; // one extra AddCommand → derives 17
  const go15 = go16.replace("\tapp.RootCmd.AddCommand(x)\n", ""); // one fewer → derives 15
  const docMatched = [
    "| Surface | Count | Gate |",
    "|---|---:|---|",
    "| Librarian CLI subcommands | 16 base (+ `pm` group under `PM_ENABLED`) | — |",
    "| Librarian MCP — default | 9 | none |",
    "",
  ].join("\n");
  const docNoCliRow = ["| Surface | Count | Gate |", "|---|---:|---|", "| Librarian MCP — default | 9 | none |", ""].join("\n");

  const cases = [
    {
      name: "CLI command added without a doc edit → RED (names CLI)",
      args: { goSource: go17, docMd: docMatched, goGuardPresent: true },
      wantFail: true,
      wantMatch: /CLI surface/,
    },
    {
      name: "CLI command removed without a doc edit → RED (names CLI)",
      args: { goSource: go15, docMd: docMatched, goGuardPresent: true },
      wantFail: true,
      wantMatch: /CLI surface/,
    },
    {
      name: "doc's CLI row deleted → RED (unparseable documented count)",
      args: { goSource: go16, docMd: docNoCliRow, goGuardPresent: true },
      wantFail: true,
      wantMatch: /could not parse/,
    },
    {
      name: "Go MCP-count guard removed → RED (names the missing Go guard)",
      args: { goSource: go16, docMd: docMatched, goGuardPresent: false },
      wantFail: true,
      wantMatch: /MCP-count guard missing/,
    },
    {
      name: "doc matches source → GREEN",
      args: { goSource: go16, docMd: docMatched, goGuardPresent: true },
      wantFail: false,
      wantMatch: null,
    },
  ];

  console.log("check-tool-surface --self-test:");
  let ok = true;
  for (const c of cases) {
    const { problems } = checkAll(c.args);
    const failed = problems.length > 0;
    const matched = c.wantMatch ? problems.some((p) => c.wantMatch.test(p)) : true;
    const pass = failed === c.wantFail && matched;
    ok = ok && pass;
    console.log(`  [${pass ? "OK" : "FAIL"}] ${c.name}`);
    if (!pass) {
      console.error(`      expected fail=${c.wantFail}${c.wantMatch ? ` matching ${c.wantMatch}` : ""}; got ${problems.length} problem(s): ${JSON.stringify(problems)}`);
    }
  }
  if (!ok) {
    console.error("\nself-test FAILED — the guard did not behave as specified.");
    process.exit(1);
  }
  console.log("  → detection verified (a tool add/remove without a matching doc edit is caught RED).");
}

// ───────────────────────────── entry ─────────────────────────────

if (process.argv.includes("--self-test")) runSelfTest();
else runScan();

// desk-pm plugin bundle guard (D5). Validates the shipped bundle's structure and — the
// load-bearing part — that every PM surface the artifacts name is a REAL frozen surface: the
// twelve D4 tools and no invented ones. A skill or agent that instructs an agent to call a
// tool that does not exist is the failure mode this test exists to catch.
import { test, expect } from "bun:test";
import { readFileSync, statSync, existsSync } from "node:fs";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";
import { parse as parseYaml } from "yaml";

const HERE = dirname(fileURLToPath(import.meta.url));
const REPO_ROOT = join(HERE, "..");
const BUNDLE = join(HERE, "desk-pm");

// The frozen D4 PM tool family (internal/modules/pm/tools/specs.go ToolNames()).
const PM_TOOLS = [
  "get_context", "list_items", "get_item", "create_item", "update_item",
  "transition_item", "block_item", "unblock_item", "add_note", "link_items",
  "claim_item", "release_item",
];
// Read-only tools always exposed; the rest are gated by PM_AUTONOMOUS_WRITES.
const PM_READ_TOOLS = ["get_context", "list_items", "get_item"];
// The four rigid phases (statemachine.Phases()).
const PHASES = ["queue", "work", "review", "terminal"];
const SKILL_NAMES = ["pm-session-open", "pm-advance-item", "pm-triage"];

function read(p: string) { return readFileSync(join(BUNDLE, p), "utf8"); }
function frontmatter(md: string): Record<string, any> {
  const m = md.match(/^---\n([\s\S]*?)\n---\n/);
  if (!m) throw new Error("no frontmatter block");
  return parseYaml(m[1]!) as Record<string, any>;
}
const canonicalVersion = readFileSync(join(REPO_ROOT, "VERSION"), "utf8").trim();

test("plugin.json is valid and version-synced to VERSION", () => {
  const j = JSON.parse(read(".claude-plugin/plugin.json"));
  expect(j.name).toBe("desk-pm");
  expect(j.version).toBe(canonicalVersion);
  expect(typeof j.description).toBe("string");
  expect(j.description.length).toBeGreaterThan(20);
});

test(".mcp.json launches the deskkit binary's PM MCP surface", () => {
  const j = JSON.parse(read(".mcp.json"));
  const srv = j["desk-pm"];
  expect(srv).toBeDefined();
  expect(srv.command).toBe("deskkit");
  expect(srv.args).toEqual(["mcp-serve"]);
  // Feature gate: the plugin's MCP-serve view must request the PM tool family.
  expect(srv.env?.PM_ENABLED).toBe("true");
});

test("all three skills exist with a name matching their directory + a description", () => {
  for (const name of SKILL_NAMES) {
    const md = read(join("skills", name, "SKILL.md"));
    const fm = frontmatter(md);
    expect(fm.name).toBe(name);
    expect(typeof fm.description).toBe("string");
    expect(fm.description.length).toBeGreaterThan(40);
  }
});

test("the pm-operator agent is scoped to exactly Read + the twelve PM tools, correctly named", () => {
  const md = read(join("agents", "pm-operator.md"));
  const fm = frontmatter(md);
  expect(fm.name).toBe("pm-operator");
  expect(typeof fm.description).toBe("string");
  const expected = ["Read", ...PM_TOOLS.map((t) => `mcp__desk-pm__${t}`)];
  // Set equality: no invented tool names, no missing PM tools, no un-namespaced stragglers.
  expect([...fm.tools].sort()).toEqual([...expected].sort());
});

test("every PM tool is referenced by at least one skill or the agent (no undocumented gaps)", () => {
  const corpus = [
    ...SKILL_NAMES.map((n) => read(join("skills", n, "SKILL.md"))),
    read(join("agents", "pm-operator.md")),
  ].join("\n");
  for (const tool of PM_TOOLS) {
    expect(corpus.includes(tool)).toBe(true);
  }
});

// Store/config data-field names that are legitimately two-word snake_case but are NOT tools
// (get_context response fields + the desk_config collection). Kept tiny and explicit.
// MAINTENANCE: if a skill/agent begins referencing another two-word snake_case data field in
// prose, add it here — otherwise the ToolNames-anchored guard below will (correctly) flag it
// as an unrecognized identifier.
const NON_TOOL_FIELDS = new Set([
  "by_court", "by_phase", "desk_config", "recent_transitions", "status_label",
]);

// The guard, as a pure function so it can be proven red on an injected fake. Returns the list
// of tool-shaped identifiers that are neither a real PM tool nor an allowlisted data field.
function unknownToolTokens(text: string): string[] {
  const bad: string[] = [];
  for (const m of text.matchAll(/\b[a-z]+_[a-z]+\b/g)) {
    const tok = m[0];
    if (PM_TOOLS.includes(tok) || NON_TOOL_FIELDS.has(tok)) continue;
    bad.push(tok);
  }
  return bad;
}

test("every tool-shaped identifier in the skills/agent is one of the twelve real PM tools", () => {
  // Assert against the real ToolNames() set — not a hardcoded blocklist — so a BARE invented
  // tool name (e.g. `archive_item`, no mcp__ prefix) fails without needing to be enumerated.
  const corpus = [
    ...SKILL_NAMES.map((n) => read(join("skills", n, "SKILL.md"))),
    read(join("agents", "pm-operator.md")),
    read("README.md"),
  ].join("\n");
  expect(unknownToolTokens(corpus)).toEqual([]);
});

test("the tool-name guard is red-able (catches a bare invented tool name)", () => {
  // Proof the guard can fail: an injected fake must be reported.
  expect(unknownToolTokens("first `archive_item`, then `get_item`.")).toEqual(["archive_item"]);
  // A real tool and an allowlisted field are both accepted.
  expect(unknownToolTokens("`transition_item` updates `status_label`.")).toEqual([]);
});

test("any mcp__desk-pm__<x> reference names one of the twelve tools", () => {
  const corpus = [
    ...SKILL_NAMES.map((n) => read(join("skills", n, "SKILL.md"))),
    read(join("agents", "pm-operator.md")),
  ].join("\n");
  for (const m of corpus.matchAll(/mcp__desk-pm__([a-z_]+)/g)) {
    expect(PM_TOOLS).toContain(m[1]!);
  }
});

test("the advance-item skill enumerates exactly the four real phases", () => {
  const md = read(join("skills", "pm-advance-item", "SKILL.md"));
  for (const p of PHASES) expect(new RegExp(`\\b${p}\\b`).test(md)).toBe(true);
  // `backlog`/`next`/`active`/`in-review` are status LABELS, not phases — never a fifth phase.
  expect(md).not.toMatch(/phase[s]?[^.\n]*\bbacklog\b/i);
});

test("every write-capable artifact documents the read-only (PM_AUTONOMOUS_WRITES=false) fallback", () => {
  // All three skills and the agent describe what changes when writes are gated off — a
  // refactor that drops the boundary in any one of them fails here.
  const files = [
    join("skills", "pm-session-open", "SKILL.md"),
    join("skills", "pm-advance-item", "SKILL.md"),
    join("skills", "pm-triage", "SKILL.md"),
    join("agents", "pm-operator.md"),
  ];
  for (const f of files) expect(read(f)).toContain("PM_AUTONOMOUS_WRITES");
  // The three read tools stay available in that mode — named across the corpus.
  const corpus = files.map(read).join("\n");
  for (const t of PM_READ_TOOLS) expect(corpus.includes(t)).toBe(true);
});

test("hooks.json wires a SessionStart command hook to the shipped script", () => {
  const j = JSON.parse(read(join("hooks", "hooks.json")));
  const ss = j.hooks?.SessionStart;
  expect(Array.isArray(ss)).toBe(true);
  const entry = ss?.[0];
  expect(entry).toBeDefined();
  const cmd = entry?.hooks?.[0];
  expect(cmd).toBeDefined();
  expect(cmd.type).toBe("command");
  expect(cmd.command).toContain("session-briefing.sh");
  expect(cmd.command).toContain("${CLAUDE_PLUGIN_ROOT}");
});

test("the SessionStart hook script exists and is executable", () => {
  const p = join(BUNDLE, "hooks", "session-briefing.sh");
  expect(existsSync(p)).toBe(true);
  const mode = statSync(p).mode;
  expect(mode & 0o111).toBeGreaterThan(0); // some execute bit set
});

test("the hook self-gates on binary-absent AND on non-JSON stdout (PM-off contract)", () => {
  const sh = read(join("hooks", "session-briefing.sh"));
  expect(sh).toContain("command -v deskkit");
  expect(sh).toContain("deskkit pm context");
  expect(sh).toContain("|| exit 0");
  // The load-bearing guard: emit ONLY when stdout is a JSON object. A PM-off desk exits 0 with
  // a cobra error on stdout at exit 0, so an exit-code / non-empty check is insufficient.
  expect(sh).toMatch(/case\s+"\$context"/);
  expect(sh).toContain("'{'*)");
});

test("the marketplace registers desk-pm alongside desk-standard, version-synced", () => {
  const mk = JSON.parse(readFileSync(join(REPO_ROOT, ".claude-plugin", "marketplace.json"), "utf8"));
  const entry = mk.plugins.find((p: any) => p.name === "desk-pm");
  expect(entry).toBeDefined();
  expect(entry.source).toBe("./plugin/desk-pm");
  expect(entry.version).toBe(canonicalVersion);
  // desk-standard must still be present (D5 adds, never replaces).
  expect(mk.plugins.some((p: any) => p.name === "desk-standard")).toBe(true);
});

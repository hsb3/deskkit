// desk-persona bundle guard (D5). Validates the COMPOSED bundle's structure and — the
// load-bearing part — that every surface the artifacts name is a REAL frozen surface: the 5
// librarian tools + 12 PM tools this mount exposes (17 total), and no invented ones. A persona
// or skill that instructs an agent to call a tool that does not exist on this mount is the
// failure mode this test exists to catch. (The former plugin/desk-pm.test.ts — for the now-retired
// standalone desk-pm bundle — was folded into this file when desk-pm was folded into desk-persona.)
import { test, expect } from "bun:test";
import { readFileSync, statSync, existsSync } from "node:fs";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";
import { parse as parseYaml } from "yaml";

const HERE = dirname(fileURLToPath(import.meta.url));
const REPO_ROOT = join(HERE, "..");
const BUNDLE = join(HERE, "desk-persona");

// The 5 librarian tools this mount exposes by default (LIBRARIAN_AUTONOMOUS_WRITES unset/false;
// apply_fix withheld, restore never exposed over MCP — docs/tool-surface.md).
const LIBRARIAN_TOOLS = ["sweep", "patrol", "propose_fix", "query", "record_feedback"];
// The frozen D4 PM tool family (internal/modules/pm/tools/specs.go ToolNames()).
const PM_TOOLS = [
  "get_context", "list_items", "get_item", "create_item", "update_item",
  "transition_item", "block_item", "unblock_item", "add_note", "link_items",
  "claim_item", "release_item",
];
// The composed mount's full exposed set: 5 librarian + 12 PM == 17 (MCP_MODULES=librarian,pm).
const ALL_TOOLS = [...LIBRARIAN_TOOLS, ...PM_TOOLS];
const AGENT_NAMES = ["librarian-operator", "pm-operator"];
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
  expect(j.name).toBe("desk-persona");
  expect(j.version).toBe(canonicalVersion);
  expect(typeof j.description).toBe("string");
  expect(j.description.length).toBeGreaterThan(20);
});

test(".mcp.json launches the deskkit binary's composed librarian+pm MCP surface", () => {
  const j = JSON.parse(read(".mcp.json"));
  const srv = j["desk-persona"];
  expect(srv).toBeDefined();
  expect(srv.command).toBe("deskkit");
  expect(srv.args).toEqual(["mcp-serve"]);
  // Feature gates: PM tools exist, AND the mount is told to register both modules — the thing
  // that distinguishes this composed bundle from desk-pm's single-module mount.
  expect(srv.env?.PM_ENABLED).toBe("true");
  expect(srv.env?.MCP_MODULES).toBe("librarian,pm");
});

test("both agents exist with a name matching their file + a description", () => {
  for (const name of AGENT_NAMES) {
    const md = read(join("agents", `${name}.md`));
    const fm = frontmatter(md);
    expect(fm.name).toBe(name);
    expect(typeof fm.description).toBe("string");
    expect(fm.description.length).toBeGreaterThan(40);
  }
});

test("all three PM skills exist with a name matching their directory + a description", () => {
  for (const name of SKILL_NAMES) {
    const md = read(join("skills", name, "SKILL.md"));
    const fm = frontmatter(md);
    expect(fm.name).toBe(name);
    expect(typeof fm.description).toBe("string");
    expect(fm.description.length).toBeGreaterThan(40);
  }
});

test("the librarian-operator agent is scoped to exactly Read + the five librarian tools, correctly named", () => {
  const md = read(join("agents", "librarian-operator.md"));
  const fm = frontmatter(md);
  expect(fm.name).toBe("librarian-operator");
  const expected = ["Read", ...LIBRARIAN_TOOLS.map((t) => `mcp__desk-persona__${t}`)];
  // Set equality: no invented tool names, no missing librarian tools, no un-namespaced stragglers.
  expect([...fm.tools].sort()).toEqual([...expected].sort());
});

test("the pm-operator agent is scoped to exactly Read + the twelve PM tools, correctly named", () => {
  const md = read(join("agents", "pm-operator.md"));
  const fm = frontmatter(md);
  expect(fm.name).toBe("pm-operator");
  const expected = ["Read", ...PM_TOOLS.map((t) => `mcp__desk-persona__${t}`)];
  // Set equality: no invented tool names, no missing PM tools, no un-namespaced stragglers.
  expect([...fm.tools].sort()).toEqual([...expected].sort());
});

test("every one of the 17 exposed tools is referenced by name somewhere in the composed corpus", () => {
  // Composed corpus = both agents + the three skills (the presence criterion is scoped to the
  // artifacts that claim the tools, not the README).
  const corpus = [
    ...AGENT_NAMES.map((n) => read(join("agents", `${n}.md`))),
    ...SKILL_NAMES.map((n) => read(join("skills", n, "SKILL.md"))),
  ].join("\n");
  for (const tool of ALL_TOOLS) {
    expect(corpus.includes(tool)).toBe(true);
  }
});

// Store/config data-field names that are legitimately two-word snake_case but are NOT tools
// (get_context response fields + the desk_config collection). Plus one addition specific to this
// composed bundle:
// `apply_fix` is a REAL librarian tool (librarian/internal/core/toolcore/toolcore.go,
// docs/tool-surface.md) that this bundle's README legitimately names as the tool withheld unless
// LIBRARIAN_AUTONOMOUS_WRITES=true — it is not part of the 17-tool exposed set on this mount, but
// it is not an INVENTED name either, so it is allowlisted here rather than flagged as a phantom.
// MAINTENANCE: if a skill/agent/README begins referencing another two-word snake_case data field
// or a real-but-unexposed tool in prose, add it here — otherwise the ToolNames-anchored guard
// below will (correctly) flag it as an unrecognized identifier.
const NON_TOOL_FIELDS = new Set([
  "by_court", "by_phase", "desk_config", "recent_transitions", "status_label", "apply_fix", "delegation_parent",
]);

// The guard, as a pure function so it can be proven red on an injected fake. Returns the list
// of tool-shaped identifiers that are neither a real exposed tool nor an allowlisted field.
function unknownToolTokens(text: string): string[] {
  const bad: string[] = [];
  for (const m of text.matchAll(/\b[a-z]+_[a-z]+\b/g)) {
    const tok = m[0];
    if (ALL_TOOLS.includes(tok) || NON_TOOL_FIELDS.has(tok)) continue;
    bad.push(tok);
  }
  return bad;
}

test("every tool-shaped identifier in the agents/skills/README is one of the 17 real exposed tools", () => {
  // Assert against the real exposed-tool set — not a hardcoded blocklist — so a BARE invented
  // tool name (e.g. `archive_item`) fails without needing to be enumerated.
  const corpus = [
    ...AGENT_NAMES.map((n) => read(join("agents", `${n}.md`))),
    ...SKILL_NAMES.map((n) => read(join("skills", n, "SKILL.md"))),
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

test("any mcp__desk-persona__<x> reference names one of the 17 exposed tools", () => {
  const corpus = [
    ...AGENT_NAMES.map((n) => read(join("agents", `${n}.md`))),
    ...SKILL_NAMES.map((n) => read(join("skills", n, "SKILL.md"))),
  ].join("\n");
  for (const m of corpus.matchAll(/mcp__desk-persona__([a-z_]+)/g)) {
    expect(ALL_TOOLS).toContain(m[1]!);
  }
});

test("the marketplace registers desk-persona alongside desk-standard; desk-pm is retired", () => {
  const mk = JSON.parse(readFileSync(join(REPO_ROOT, ".claude-plugin", "marketplace.json"), "utf8"));
  const entry = mk.plugins.find((p: any) => p.name === "desk-persona");
  expect(entry).toBeDefined();
  expect(entry.source).toBe("./plugin/desk-persona");
  expect(entry.version).toBe(canonicalVersion);
  // desk-standard stays; desk-pm was folded into desk-persona and removed from the marketplace
  // (owner ruling 2026-07-21 "fold"; ADR 0014(a) one composed bundle).
  expect(mk.plugins.some((p: any) => p.name === "desk-standard")).toBe(true);
  expect(mk.plugins.some((p: any) => p.name === "desk-pm")).toBe(false);
});

// The SessionStart briefing hook — folded in from the retired desk-pm bundle. Mirrors the hook
// guards in the former plugin/desk-pm.test.ts so the composed bundle carries the same contract.
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

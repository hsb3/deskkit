#!/usr/bin/env node
// D4 / W2a — the OpenCode install path. Copies this plugin's three skills from their canonical
// home (plugins/desk-standard/skills/, the foreman ruling) into the OpenCode skills directory, and
// prints the config snippet that registers the shared MCP server. It never moves or edits the
// source skills, and never re-implements the MCP tools — the OpenCode plugin module
// (opencode/plugin.ts) registers the same server programmatically; this snippet is the manual
// fallback for a user who does not load the plugin module.
//
// Runs under plain Node: `node scripts/install-opencode.mjs [--dest <dir>] [--dry-run]`.
// The copy helpers are exported so bun test can exercise them against a temp dir.

import { cpSync, existsSync, mkdirSync, readdirSync, rmSync, statSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { homedir } from "node:os";

/** The three skills this plugin ships, in a stable order. Source of truth: plugins/desk-standard/skills/. */
export const SKILL_NAMES = ["desk-setup", "conventions-standard", "harvest-loop"];

const HERE = dirname(fileURLToPath(import.meta.url)); // plugin/scripts

/** The canonical skills source: plugins/desk-standard/skills (skills live here once — foreman ruling). */
export function defaultSkillsSrc() {
  return join(HERE, "..", "..", "plugins", "desk-standard", "skills");
}

/**
 * The default OpenCode global skills destination: $XDG_CONFIG_HOME/opencode/skills, else
 * ~/.config/opencode/skills. OpenCode scans this dir for <name>/SKILL.md. Override with --dest or
 * the OPENCODE_SKILLS_DIR env var (e.g. a project-local .opencode/skills).
 */
export function defaultSkillsDest(env = process.env) {
  const override = env.OPENCODE_SKILLS_DIR;
  if (typeof override === "string" && override.trim() !== "") return override.trim();
  const configHome =
    env.XDG_CONFIG_HOME && env.XDG_CONFIG_HOME.trim() !== ""
      ? env.XDG_CONFIG_HOME.trim()
      : join(homedir(), ".config");
  return join(configHome, "opencode", "skills");
}

/**
 * copySkills copies each named skill directory (with all its assets) from srcRoot into destRoot,
 * replacing any existing copy so a reinstall leaves no stale files. Returns the list of
 * { name, from, to } actually copied. With dryRun, plans without touching the filesystem.
 * Throws if a named skill or its SKILL.md is missing from the source (a packaging error).
 */
export function copySkills(srcRoot, destRoot, names = SKILL_NAMES, { dryRun = false } = {}) {
  const copied = [];
  for (const name of names) {
    const from = join(srcRoot, name);
    const skillMd = join(from, "SKILL.md");
    if (!existsSync(from) || !statSync(from).isDirectory()) {
      throw new Error(`install-opencode: source skill not found: ${from}`);
    }
    if (!existsSync(skillMd)) {
      throw new Error(`install-opencode: ${name} is missing SKILL.md at ${skillMd}`);
    }
    const to = join(destRoot, name);
    if (!dryRun) {
      mkdirSync(destRoot, { recursive: true });
      rmSync(to, { recursive: true, force: true });
      cpSync(from, to, { recursive: true });
    }
    copied.push({ name, from, to });
  }
  return copied;
}

/** The identity-neutral OpenCode MCP registration snippet (manual fallback to the plugin module). */
export function mcpConfigSnippet(pluginRoot) {
  return {
    mcp: {
      "desk-standard": {
        type: "local",
        command: ["bun", join(pluginRoot, "mcp", "server.ts")],
        enabled: true,
      },
    },
  };
}

function parseArgs(argv) {
  const opts = { dryRun: false, dest: null };
  for (let i = 0; i < argv.length; i++) {
    const a = argv[i];
    if (a === "--dry-run") opts.dryRun = true;
    else if (a === "--dest") opts.dest = argv[++i];
    else if (a === "--help" || a === "-h") opts.help = true;
  }
  return opts;
}

function main() {
  const opts = parseArgs(process.argv.slice(2));
  if (opts.help) {
    process.stdout.write(
      "Usage: node scripts/install-opencode.mjs [--dest <dir>] [--dry-run]\n" +
        "  Copies the three desk-standard skills into the OpenCode skills dir\n" +
        `  (default: ${defaultSkillsDest()}).\n`,
    );
    return;
  }
  const srcRoot = defaultSkillsSrc();
  const destRoot = opts.dest ?? defaultSkillsDest();
  const pluginRoot = join(HERE, ".."); // plugin/
  const copied = copySkills(srcRoot, destRoot, SKILL_NAMES, { dryRun: opts.dryRun });

  const verb = opts.dryRun ? "would copy" : "copied";
  process.stdout.write(`install-opencode: ${verb} ${copied.length} skill(s) into ${destRoot}\n`);
  for (const c of copied) process.stdout.write(`  - ${c.name}\n`);
  process.stdout.write(
    "\nRegister the shared MCP server (the plugin module does this automatically when loaded;\n" +
      "this is the manual fallback). Add to your OpenCode config:\n\n" +
      JSON.stringify(mcpConfigSnippet(pluginRoot), null, 2) +
      "\n\nTo load the plugin module itself, add its path to the config `plugin` array:\n" +
      `  "${join(pluginRoot, "opencode", "plugin.ts")}"\n`,
  );
}

// Run only when invoked directly (not when imported by tests).
if (process.argv[1] && fileURLToPath(import.meta.url) === process.argv[1]) {
  main();
}

// Profile discovery + parsing — the plugin-side mirror of the Go reference
// (librarian/internal/config/profile.go). Harness-pure: Node built-ins + `yaml` only,
// no MCP / OpenCode / Bun-specific imports (AC2).

import { existsSync, readFileSync, statSync } from "node:fs";
import { dirname, extname, join } from "node:path";
import { parse as parseYAML } from "yaml";

/** A parsed profile is a plain nested mapping of scalars, maps, and lists. */
export type Profile = Record<string, unknown>;

// Extension precedence on a tie within one directory: yaml > yml > json > md
// (mirrors DiscoverProfile's `names` order exactly).
const PROFILE_NAMES = ["profile.yaml", "profile.yml", "profile.json", "profile.md"] as const;

/**
 * discoverProfile walks UP from startDir looking for a single personalization root
 * `_knowledge/profile.{yaml,yml,json,md}` (M-05 surface i). Returns the first match's
 * absolute path, or null. Stops at the filesystem root.
 */
export function discoverProfile(startDir: string): string | null {
  let dir = startDir;
  for (;;) {
    for (const name of PROFILE_NAMES) {
      const p = join(dir, "_knowledge", name);
      try {
        if (existsSync(p) && !statSync(p).isDirectory()) return p;
      } catch {
        // stat race / permission — treat as absent and keep walking
      }
    }
    const parent = dirname(dir);
    if (parent === dir) return null;
    dir = parent;
  }
}

/**
 * discoverKnowledgeDir walks UP from startDir looking for a `_knowledge` directory,
 * returning its absolute path or null. Used by knowledge_index when no profile is present
 * but a `_knowledge/` folder still exists.
 */
export function discoverKnowledgeDir(startDir: string): string | null {
  let dir = startDir;
  for (;;) {
    const p = join(dir, "_knowledge");
    try {
      if (existsSync(p) && statSync(p).isDirectory()) return p;
    } catch {
      // ignore and keep walking
    }
    const parent = dirname(dir);
    if (parent === dir) return null;
    dir = parent;
  }
}

/**
 * loadProfile parses a profile file into a nested mapping, selecting the codec by extension
 * (mirrors LoadProfile): .yaml/.yml -> YAML, .json -> JSON, .md -> the YAML frontmatter
 * block. This does NOT validate against the schema — see loadAndValidateProfile in schema.ts.
 */
export function loadProfile(path: string): Profile {
  const raw = readFileSync(path, "utf8");
  const ext = extname(path).toLowerCase();
  switch (ext) {
    case ".yaml":
    case ".yml":
      return coerceMapping(parseYAML(raw), path);
    case ".json":
      return coerceMapping(JSON.parse(raw), path);
    case ".md":
      return coerceMapping(parseYAML(extractFrontmatter(raw)), path);
    default:
      throw new Error(`profile ${path}: unsupported extension ${ext}`);
  }
}

function coerceMapping(v: unknown, path: string): Profile {
  if (v === null || v === undefined) return {}; // empty doc -> empty map (mirrors Go `m == nil`)
  if (typeof v !== "object" || Array.isArray(v)) {
    throw new Error(`profile ${path}: top level must be a mapping, got ${describe(v)}`);
  }
  return v as Profile;
}

function describe(v: unknown): string {
  if (Array.isArray(v)) return "a list";
  return typeof v;
}

/**
 * extractFrontmatter returns the text between the first pair of `---` fences in a
 * markdown-with-frontmatter file; empty string if there is no opening fence or the fence is
 * unterminated (mirrors extractFrontmatter).
 */
export function extractFrontmatter(text: string): string {
  const lines = text.split("\n");
  if (lines.length === 0 || (lines[0] ?? "").trim() !== "---") return "";
  const body: string[] = [];
  for (let i = 1; i < lines.length; i++) {
    if ((lines[i] ?? "").trim() === "---") return body.join("\n");
    body.push(lines[i] ?? "");
  }
  return ""; // unterminated fence
}

/**
 * indexTree resolves a dotted path (already split into parts) into the profile tree,
 * returning the value or undefined if any segment is missing or not a mapping.
 */
export function indexTree(m: Profile, parts: string[]): unknown {
  let cur: unknown = m;
  for (const p of parts) {
    if (cur === null || typeof cur !== "object" || Array.isArray(cur)) return undefined;
    cur = (cur as Record<string, unknown>)[p];
  }
  return cur;
}

/**
 * scalarString renders a resolved value as its substitutable string, or "" if the value is
 * absent, empty, or a non-scalar (map/list) — mirrors scalarString. Integer-valued numbers
 * render without a decimal point.
 */
export function scalarString(v: unknown): string {
  if (v === null || v === undefined) return "";
  switch (typeof v) {
    case "string":
      return v;
    case "boolean":
      return v ? "true" : "false";
    case "number":
      if (!Number.isFinite(v)) return "";
      return Number.isInteger(v) ? String(v) : String(v);
    default:
      return ""; // maps and lists are not substitutable scalars
  }
}

/**
 * profileScalar resolves a dotted path into the profile tree and returns its scalar value as
 * a string, or "" if absent/empty/non-scalar (mirrors profileScalar).
 */
export function profileScalar(profile: Profile | null, dotted: string): string {
  if (!profile) return "";
  return scalarString(indexTree(profile, dotted.split(".")));
}

// The four MCP tools — SINGLE SOURCE of their input schemas AND their behavior (AC3). The
// MCP server (plugin/mcp) imports TOOLS from here and only adapts them to the transport; it
// re-declares no schema and re-implements no logic. Handlers are harness-pure: they throw on
// error (the server maps a thrown error to an MCP isError result, mirroring the Go server's
// jsonContent posture) and never touch the MCP SDK.

import { dirname } from "node:path";
import { discoverProfile, discoverKnowledgeDir, indexTree, loadProfile, scalarString, type Profile } from "./profile.js";
import { substitute } from "./substitute.js";
import {
  defaultSchemaPath,
  getValidator,
  loadAndValidateProfile,
  validateProfile,
} from "./schema.js";
import { DEFAULT_KNOWLEDGE_BUDGET, knowledgeIndex, type KnowledgeIndex } from "./knowledge.js";

/** JSON Schema fragment for a tool's input (plain JSON — no SDK types). */
export type JsonSchema = Record<string, unknown>;

/** Options threaded into every handler; both default lazily so tests can pin them. */
export interface ToolContext {
  /** Directory the profile/knowledge/schema walk-ups start from. Defaults to process.cwd(). */
  cwd?: string;
  /** Explicit schema path override. Defaults to defaultSchemaPath(). */
  schemaPath?: string;
}

export interface ToolDef<Args = Record<string, unknown>, Result = unknown> {
  name: string;
  description: string;
  inputSchema: JsonSchema;
  handler: (args: Args, ctx?: ToolContext) => Result;
}

function cwdOf(ctx?: ToolContext): string {
  return ctx?.cwd ?? process.cwd();
}

function schemaPathOf(ctx?: ToolContext): string | null {
  return ctx?.schemaPath ?? defaultSchemaPath();
}

/** Discover + load + (schema-)validate the active profile. Throws if the profile violates
 *  schema v1 (AC8). If no schema can be located, loads without validation and notes it. */
function resolveProfile(ctx?: ToolContext): { path: string; profile: Profile } | null {
  const path = discoverProfile(cwdOf(ctx));
  if (!path) return null;
  const schemaPath = schemaPathOf(ctx);
  const profile = schemaPath ? loadAndValidateProfile(path, schemaPath) : loadProfile(path);
  return { path, profile };
}

// --- profile_get -----------------------------------------------------------------------------

export interface ProfileGetArgs {
  path: string;
}
export interface ProfileGetResult {
  path: string;
  value: string;
}

export const profileGet: ToolDef<ProfileGetArgs, ProfileGetResult> = {
  name: "profile_get",
  description:
    "Resolve a dotted profile key (e.g. \"identity.github.personal\") from the discovered " +
    "_knowledge/profile.* to its scalar value. Fails loudly if the key is absent or empty; " +
    "there is no default concept here.",
  inputSchema: {
    type: "object",
    additionalProperties: false,
    properties: {
      path: { type: "string", description: "Dotted profile key, e.g. \"repos.default\"." },
    },
    required: ["path"],
  },
  handler(args, ctx) {
    const key = typeof args?.path === "string" ? args.path.trim() : "";
    if (key === "") throw new Error("profile_get: `path` is required and must be a non-empty string");
    const resolved = resolveProfile(ctx);
    if (!resolved) {
      throw new Error(`profile_get: no _knowledge/profile.* found from ${cwdOf(ctx)}`);
    }
    const value = scalarString(indexTree(resolved.profile, key.split(".")));
    if (value === "") throw new Error(absentKeyMessage(resolved.profile, key));
    return { path: key, value };
  },
};

/** Build a fail-loud message with a suggested-key hint: list the keys available at the deepest
 *  mapping the requested path could be resolved to. */
function absentKeyMessage(profile: Profile, dotted: string): string {
  const parts = dotted.split(".");
  let cur: unknown = profile;
  const reached: string[] = [];
  for (const p of parts) {
    if (cur === null || typeof cur !== "object" || Array.isArray(cur)) break;
    const next = (cur as Record<string, unknown>)[p];
    if (next === undefined) break;
    reached.push(p);
    cur = next;
  }
  const parentPath = reached.slice(0, Math.max(reached.length, 0)).join(".");
  const container = reached.length === 0 ? profile : cur;
  const scope =
    container && typeof container === "object" && !Array.isArray(container)
      ? Object.keys(container as Record<string, unknown>)
      : [];
  const under = parentPath === "" ? "top level" : `"${parentPath}"`;
  const hint = scope.length > 0 ? ` Available keys under ${under}: ${scope.sort().join(", ")}.` : "";
  return `profile_get: key "${dotted}" not found or empty.${hint}`;
}

// --- profile_validate ------------------------------------------------------------------------

export interface ProfileValidateArgs {
  path?: string;
}
export interface ProfileValidateResult {
  valid: boolean;
  errors: string[];
  profilePath: string | null;
}

export const profileValidate: ToolDef<ProfileValidateArgs, ProfileValidateResult> = {
  name: "profile_validate",
  description:
    "Validate the discovered _knowledge/profile.* (or a given path) against desk-standard " +
    "schema v1. Returns { valid, errors, profilePath } — unknown top-level keys and shape " +
    "violations appear in errors.",
  inputSchema: {
    type: "object",
    additionalProperties: false,
    properties: {
      path: {
        type: "string",
        description: "Optional explicit profile path; defaults to the discovered profile.",
      },
    },
  },
  handler(args, ctx) {
    const given = typeof args?.path === "string" && args.path.trim() !== "" ? args.path.trim() : null;
    const profilePath = given ?? discoverProfile(cwdOf(ctx));
    if (!profilePath) {
      return { valid: false, errors: [`no _knowledge/profile.* found from ${cwdOf(ctx)}`], profilePath: null };
    }
    const schemaPath = schemaPathOf(ctx);
    if (!schemaPath) {
      throw new Error(
        "profile_validate: schema v1 not found (set DESK_SCHEMA_PATH or run within the repo)",
      );
    }
    let profile: Profile;
    try {
      profile = loadProfile(profilePath);
    } catch (e) {
      return { valid: false, errors: [errText(e)], profilePath };
    }
    const result = validateProfile(profile, getValidator(schemaPath));
    return { valid: result.valid, errors: result.errors, profilePath };
  },
};

// --- template_render -------------------------------------------------------------------------

export interface TemplateRenderArgs {
  template: string;
}
export interface TemplateRenderResult {
  rendered: string;
}

export const templateRender: ToolDef<TemplateRenderArgs, TemplateRenderResult> = {
  name: "template_render",
  description:
    "Render a template by substituting {{profile.<key>}} / {{env.<VAR>}} placeholders (with " +
    "optional || \"default\") against the discovered profile and process env. A placeholder " +
    "with no default that resolves absent/empty is a hard error listing every offender.",
  inputSchema: {
    type: "object",
    additionalProperties: false,
    properties: {
      template: { type: "string", description: "Template text containing {{profile.…}}/{{env.…}} placeholders." },
    },
    required: ["template"],
  },
  handler(args, ctx) {
    if (typeof args?.template !== "string") {
      throw new Error("template_render: `template` is required and must be a string");
    }
    const resolved = resolveProfile(ctx);
    const profile = resolved?.profile ?? {};
    return { rendered: substitute(args.template, profile) };
  },
};

// --- knowledge_index -------------------------------------------------------------------------

export interface KnowledgeIndexArgs {
  budget?: number;
}

export const knowledgeIndexTool: ToolDef<KnowledgeIndexArgs, KnowledgeIndex> = {
  name: "knowledge_index",
  description:
    "Index the _knowledge/ background folder (M-05 surface ii): recursively list its *.md " +
    "files with metadata (path, bytes, words) and file content up to a byte budget; over-budget " +
    "files are returned as index entries only.",
  inputSchema: {
    type: "object",
    additionalProperties: false,
    properties: {
      budget: {
        type: "integer",
        minimum: 0,
        description: `Content byte budget (default ${DEFAULT_KNOWLEDGE_BUDGET}).`,
      },
    },
  },
  handler(args, ctx) {
    const budget =
      typeof args?.budget === "number" && Number.isFinite(args.budget) && args.budget >= 0
        ? Math.floor(args.budget)
        : DEFAULT_KNOWLEDGE_BUDGET;
    const start = cwdOf(ctx);
    const profilePath = discoverProfile(start);
    // The knowledge root is the directory holding the profile; else the nearest _knowledge dir.
    const root = profilePath ? dirname(profilePath) : discoverKnowledgeDir(start);
    if (!root) {
      return { root: "", budget, fileCount: 0, bytesIncluded: 0, entries: [] };
    }
    return knowledgeIndex(root, budget);
  },
};

// --- registry --------------------------------------------------------------------------------

/** The exact four tools this plugin exposes, in a stable order. */
export const TOOLS: ToolDef<any, any>[] = [profileGet, profileValidate, templateRender, knowledgeIndexTool];

function errText(e: unknown): string {
  return e instanceof Error ? e.message : String(e);
}

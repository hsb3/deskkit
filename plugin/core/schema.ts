// Schema v1 validation — loads schema/profile.schema.yaml (JSON Schema authored in YAML)
// and validates a profile against it. A profile that violates the schema is REJECTED with a
// clear error naming the violations (AC8). Harness-pure (Node built-ins + yaml + ajv).

import { existsSync, readFileSync, statSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
// nodenext resolution needs the explicit .js extension; ajv/ajv-formats ship CJS whose
// default export lands differently under Bun (ESM interop) vs Node require — unwrap a
// possible `.default` at runtime. TS's CJS-interop typing makes the default import the
// module namespace, so pin the shapes we actually use with minimal structural types.
import Ajv2020Import from "ajv/dist/2020.js";
import type { Options, ValidateFunction } from "ajv";
import addFormatsImport from "ajv-formats";

interface AjvLike {
  compile(schema: object): ValidateFunction;
}
const Ajv2020 = ((Ajv2020Import as { default?: unknown }).default ??
  Ajv2020Import) as unknown as new (opts?: Options) => AjvLike;
const addFormats = ((addFormatsImport as { default?: unknown }).default ??
  addFormatsImport) as unknown as (ajv: AjvLike) => unknown;
import { parse as parseYAML } from "yaml";
import { loadProfile, type Profile } from "./profile.js";

const SCHEMA_REL = join("schema", "profile.schema.yaml");

/**
 * discoverSchema walks UP from startDir looking for `schema/profile.schema.yaml`, returning
 * its absolute path or null. Mirrors the profile walk-up so the plugin locates the schema when
 * run from source. Packaging must ship the schema alongside — see the handoff follow-up.
 */
export function discoverSchema(startDir: string): string | null {
  let dir = startDir;
  for (;;) {
    const p = join(dir, SCHEMA_REL);
    try {
      if (existsSync(p) && !statSync(p).isDirectory()) return p;
    } catch {
      // ignore and keep walking
    }
    const parent = dirname(dir);
    if (parent === dir) return null;
    dir = parent;
  }
}

/**
 * defaultSchemaPath resolves the schema location, in order: DESK_SCHEMA_PATH env var, a
 * walk-up from this module's directory (finds it from source), then a walk-up from cwd.
 * Returns null if none is found.
 */
export function defaultSchemaPath(): string | null {
  const fromEnv = process.env.DESK_SCHEMA_PATH;
  if (fromEnv && fromEnv !== "") return fromEnv;
  const moduleDir = dirname(fileURLToPath(import.meta.url));
  return discoverSchema(moduleDir) ?? discoverSchema(process.cwd());
}

/** loadSchemaObject reads and parses the YAML-authored JSON Schema into a plain object. */
export function loadSchemaObject(path: string): Record<string, unknown> {
  const obj = parseYAML(readFileSync(path, "utf8"));
  if (obj === null || typeof obj !== "object" || Array.isArray(obj)) {
    throw new Error(`schema ${path}: expected a JSON-Schema object`);
  }
  return obj as Record<string, unknown>;
}

/**
 * KNOWN_CONTRACT_VERSIONS is the set of profile.schema.yaml `x-contract-version` values this
 * build understands (ADR 0009's shared-contract versioning). A schema file declaring anything
 * else is refused loud in compileValidator rather than silently misread. This is the CONTRACT
 * file's own version — distinct from a profile INSTANCE's `schema_version` (which const-1 pins
 * a _knowledge/profile.yaml to schema v1) and from the store-side module_schema_versions
 * migration mechanism (pm-system spec §8.3 / R7.1).
 */
export const KNOWN_CONTRACT_VERSIONS: readonly number[] = [1];

/**
 * compileValidator builds an ajv draft-2020-12 validator (allErrors so AC8 can name every
 * violation; strict off because the schema carries $schema/$id/format metadata). It first
 * reads the schema's `x-contract-version` marker and THROWS on an unrecognized value, then
 * strips that schema-meta key so ajv never sees it. Memoized by schema path.
 */
const validatorCache = new Map<string, ValidateFunction>();

export function compileValidator(schemaObj: Record<string, unknown>): ValidateFunction {
  const version = schemaObj["x-contract-version"];
  if (typeof version !== "number" || !KNOWN_CONTRACT_VERSIONS.includes(version)) {
    throw new Error(
      `schema contract version ${JSON.stringify(version)} is not recognized ` +
        `(known versions: [${KNOWN_CONTRACT_VERSIONS.join(", ")}])`,
    );
  }
  const { "x-contract-version": _cv, ...schemaForAjv } = schemaObj;
  const ajv = new Ajv2020({ allErrors: true, strict: false });
  addFormats(ajv);
  return ajv.compile(schemaForAjv);
}

export function getValidator(schemaPath: string): ValidateFunction {
  const cached = validatorCache.get(schemaPath);
  if (cached) return cached;
  const v = compileValidator(loadSchemaObject(schemaPath));
  validatorCache.set(schemaPath, v);
  return v;
}

export interface ValidationResult {
  valid: boolean;
  errors: string[];
}

/** validateProfile runs a profile through a compiled validator and returns a flat result with
 *  human-readable violation strings (AC8). */
export function validateProfile(profile: Profile, validate: ValidateFunction): ValidationResult {
  const ok = validate(profile) as boolean;
  if (ok) return { valid: true, errors: [] };
  const errors = (validate.errors ?? []).map(formatError);
  return { valid: false, errors };
}

function formatError(e: {
  instancePath?: string;
  message?: string;
  params?: Record<string, unknown>;
}): string {
  const where = e.instancePath && e.instancePath !== "" ? e.instancePath : "(root)";
  let msg = `${where} ${e.message ?? "is invalid"}`;
  const extra = e.params?.["additionalProperty"];
  if (typeof extra === "string") msg += ` (unknown key "${extra}")`;
  return msg;
}

/**
 * loadAndValidateProfile parses a profile and validates it against schema v1, THROWING with a
 * clear multi-violation message if it is invalid (AC8 — invalid profiles are rejected on load).
 */
export function loadAndValidateProfile(path: string, schemaPath: string): Profile {
  const profile = loadProfile(path);
  const validate = getValidator(schemaPath);
  const result = validateProfile(profile, validate);
  if (!result.valid) {
    throw new Error(`profile ${path} violates schema v1: ${result.errors.join("; ")}`);
  }
  return profile;
}

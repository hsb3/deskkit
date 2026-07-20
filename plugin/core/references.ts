// Schema v1 typed cross-reference guard (ADR 0011) — loads schema/references.yaml and
// validates a { kind, target } reference against it. The reference primitive is one closed
// `kind` enum (seeded `issue`, `url`) plus a raw `target` string; the desk-relative repo
// qualifier is NOT part of the persisted shape — per ADR 0011 it resolves at read time from
// the profile (repos.shorthand.issue_default), so validateReference takes no qualifier.
// Harness-pure (Node built-ins + yaml only), mirroring schema.ts's discovery pattern.

import { existsSync, readFileSync, statSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { parse as parseYAML } from "yaml";
import type { ValidationResult } from "./schema.js";

const REFERENCES_REL = join("schema", "references.yaml");

/** The parsed schema-v1 reference contract: the closed set of legal reference kinds. */
export interface ReferenceVocabulary {
  kinds: string[];
}

/**
 * discoverReferences walks UP from startDir looking for `schema/references.yaml`, returning its
 * absolute path or null. Mirrors discoverSchema so the plugin locates the contract when run
 * from source; packaging ships the copy alongside (plugin/package.json `package` step).
 */
export function discoverReferences(startDir: string): string | null {
  let dir = startDir;
  for (;;) {
    const p = join(dir, REFERENCES_REL);
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
 * defaultReferencesPath resolves the contract location, in order: DESK_REFERENCES_PATH env var,
 * a walk-up from this module's directory (finds it from source), then a walk-up from cwd.
 * Returns null if none is found.
 */
export function defaultReferencesPath(): string | null {
  const fromEnv = process.env.DESK_REFERENCES_PATH;
  if (fromEnv) return fromEnv;
  const moduleDir = dirname(fileURLToPath(import.meta.url));
  return discoverReferences(moduleDir) ?? discoverReferences(process.cwd());
}

/** loadReferenceVocabulary reads and parses schema/references.yaml into the kind vocabulary. */
export function loadReferenceVocabulary(path: string): ReferenceVocabulary {
  const obj = parseYAML(readFileSync(path, "utf8"));
  if (obj === null || typeof obj !== "object" || Array.isArray(obj)) {
    throw new Error(`references ${path}: expected a mapping`);
  }
  const kind = (obj as Record<string, unknown>)["kind"];
  if (!Array.isArray(kind) || kind.length === 0 || !kind.every((k) => typeof k === "string")) {
    throw new Error(`references ${path}: 'kind' must be a non-empty list of strings`);
  }
  const kinds = (kind as string[]).map((k) => k.trim()).filter((k) => k !== "");
  if (kinds.length === 0) {
    throw new Error(`references ${path}: 'kind' enum is empty after trimming`);
  }
  return { kinds };
}

/**
 * validateReference checks a persisted { kind, target } reference against the schema-v1
 * contract: the kind must be known and the target must be non-empty (whitespace-only is
 * empty). No qualifier — the persisted shape never carries one (ADR 0011). Returns a flat
 * result with human-readable violation strings.
 */
export function validateReference(
  kind: string,
  target: string,
  vocab: ReferenceVocabulary,
): ValidationResult {
  const errors: string[] = [];
  if (!vocab.kinds.includes(kind)) {
    errors.push(`unknown reference kind "${kind}" (known: ${vocab.kinds.join(", ")})`);
  }
  if (target.trim() === "") {
    errors.push("reference target must be non-empty");
  }
  return { valid: errors.length === 0, errors };
}

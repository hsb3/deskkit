// Placeholder substitution + env>profile>default precedence — mirrors the Go reference
// (Substitute + pick in librarian/internal/config). Harness-pure.

import type { Profile } from "./profile.js";
import { profileScalar } from "./profile.js";

// Matches {{profile.<dotted.path>}} / {{env.<VAR>}} with an optional `|| "default"`.
// Group 1 = kind, 2 = path/var, 3 = the whole `|| "…"` clause (presence = has default),
// 4 = the default text. Identical to the Go placeholderRe.
export const PLACEHOLDER_RE =
  /\{\{\s*(profile|env)\.([A-Za-z0-9_.]+)\s*(\|\|\s*"((?:[^"\\]|\\.)*)")?\s*\}\}/g;

/** Error thrown by substitute when a required placeholder resolves absent/empty. Carries the
 *  full offender list and the partially-resolved text (fail LOUD, never a silent empty sub). */
export class MissingPlaceholderError extends Error {
  readonly missing: string[];
  readonly partial: string;
  constructor(missing: string[], partial: string) {
    super(
      `profile substitution: missing required key(s) with no default: ${missing.join(", ")}`,
    );
    this.name = "MissingPlaceholderError";
    this.missing = missing;
    this.partial = partial;
  }
}

/**
 * substitute resolves every {{profile.…}} / {{env.…}} placeholder in text against the profile
 * (and process env). A placeholder with NO `|| default` whose value is absent/empty is a hard
 * error: it throws MissingPlaceholderError naming EVERY offending placeholder in one error
 * (mirrors Substitute's collect-all behavior), never a silent empty substitution.
 */
export function substitute(text: string, profile: Profile | null): string {
  const missing: string[] = [];
  const out = text.replace(
    PLACEHOLDER_RE,
    (match, kind: string, path: string, defaultClause: string | undefined, defaultText: string | undefined) => {
      const hasDefault = defaultClause !== undefined;
      const def = unescape(defaultText ?? "");
      const val = kind === "env" ? process.env[path] ?? "" : profileScalar(profile, path);
      if (val === "") {
        if (hasDefault) return def;
        missing.push(match);
        return match; // left in place; the thrown error is authoritative
      }
      return val;
    },
  );
  if (missing.length > 0) throw new MissingPlaceholderError(missing, out);
  return out;
}

// unescape mirrors the Go `unescape`: `\"` -> `"`, then `\\` -> `\`.
function unescape(s: string): string {
  return s.replace(/\\"/g, '"').replace(/\\\\/g, "\\");
}

/**
 * pick applies env > profile > default precedence (mirrors the Go `pick`). An env var set but
 * empty is treated as unset, so it falls through to the profile value, then the default.
 */
export function pick(envKey: string, profileVal: string, def: string): string {
  const v = process.env[envKey];
  if (v !== undefined && v !== "") return v;
  if (profileVal !== "") return profileVal;
  return def;
}

/**
 * resolveValue is the profile-aware convenience over pick: env var > profile dotted value >
 * built-in default. Exposed for future consumers (e.g. adapters resolving a config field).
 */
export function resolveValue(
  profile: Profile | null,
  envKey: string,
  dottedPath: string,
  def: string,
): string {
  return pick(envKey, profileScalar(profile, dottedPath), def);
}

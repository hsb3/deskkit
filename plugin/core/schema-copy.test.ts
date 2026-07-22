// Drift guard for the plugin's schema copies. `plugin/package.json`'s `package` script (`cp
// ../schema/profile.schema.yaml ../plugins/desk-standard/schema/profile.schema.yaml` and the
// analogous references.yaml copy) is the ONLY sanctioned way to update
// plugins/desk-standard/schema/* — it must
// never be hand-edited (see CLAUDE.md "Generated artifacts — never hand-edit"). Mirrors the Go
// guard librarian/internal/core/schema/doctypes_test.go
// (TestDoctypesEmbeddedCopy_MatchesRepoRoot): read both the repo-root source and the shipped
// copy, assert byte-equality, and fail with an instruction to re-run the copy step.

import { test, expect } from "bun:test";
import { readFileSync } from "node:fs";
import { join } from "node:path";

// This file lives at plugin/core/schema-copy.test.ts -> repo root is two levels up.
const repoRoot = join(import.meta.dir, "..", "..");

function assertCopyMatchesSource(relPath: string): void {
  const source = readFileSync(join(repoRoot, "schema", relPath), "utf8");
  const copy = readFileSync(join(repoRoot, "plugins", "desk-standard", "schema", relPath), "utf8");
  if (source !== copy) {
    throw new Error(
      `plugins/desk-standard/schema/${relPath} has drifted from schema/${relPath}; ` +
        "re-run `bun run package` (from plugin/) to re-copy the repo-root source, which is " +
        "the source of truth.",
    );
  }
}

test("plugins/desk-standard/schema/profile.schema.yaml matches the repo-root source", () => {
  assertCopyMatchesSource("profile.schema.yaml");
});

test("plugins/desk-standard/schema/references.yaml matches the repo-root source", () => {
  assertCopyMatchesSource("references.yaml");
});

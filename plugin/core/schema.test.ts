import { test, expect } from "bun:test";
import { mkdtempSync, mkdirSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import {
  compileValidator,
  discoverSchema,
  getValidator,
  loadAndValidateProfile,
  loadSchemaObject,
  validateProfile,
} from "./schema.js";

// The repo schema is found by walking up from this test file's directory.
const schemaPath = discoverSchema(dirname(fileURLToPath(import.meta.url)));
if (!schemaPath) throw new Error("schema/profile.schema.yaml not found from test dir");
const validator = compileValidator(loadSchemaObject(schemaPath));

function writeProfile(body: string): string {
  const root = mkdtempSync(join(tmpdir(), "ds-schema-"));
  mkdirSync(join(root, "_knowledge"), { recursive: true });
  const p = join(root, "_knowledge", "profile.yaml");
  writeFileSync(p, body);
  return p;
}

test("valid profile is accepted", () => {
  const r = validateProfile(
    { schema_version: 1, repos: { default: "octocat/example-repo" }, custom: { anything: "goes" } },
    validator,
  );
  expect(r.valid).toBe(true);
  expect(r.errors).toHaveLength(0);
});

test("unknown top-level key is rejected and named", () => {
  const r = validateProfile({ schema_version: 1, bogus_key: true }, validator);
  expect(r.valid).toBe(false);
  expect(r.errors.join(" ")).toContain("bogus_key");
});

test("missing schema_version is rejected", () => {
  const r = validateProfile({ repos: { default: "octocat/example-repo" } }, validator);
  expect(r.valid).toBe(false);
  expect(r.errors.join(" ")).toContain("schema_version");
});

test("custom block accepts arbitrary nested keys", () => {
  const r = validateProfile(
    { schema_version: 1, custom: { deep: { nested: [1, 2, 3] }, flag: true } },
    validator,
  );
  expect(r.valid).toBe(true);
});

test("bad repo slug pattern is rejected", () => {
  const r = validateProfile({ schema_version: 1, repos: { default: "not-a-slug" } }, validator);
  expect(r.valid).toBe(false);
});

test("loadAndValidateProfile throws on an invalid profile (AC8)", () => {
  const bad = writeProfile("schema_version: 1\nbogus_key: 1\n");
  expect(() => loadAndValidateProfile(bad, schemaPath!)).toThrow(/violates schema v1/);
});

test("loadAndValidateProfile returns a valid profile", () => {
  const good = writeProfile("schema_version: 1\ndesk:\n  name: example-desk\n");
  const p = loadAndValidateProfile(good, schemaPath!);
  expect((p.desk as { name: string }).name).toBe("example-desk");
});

test("getValidator memoizes by path", () => {
  expect(getValidator(schemaPath!)).toBe(getValidator(schemaPath!));
});

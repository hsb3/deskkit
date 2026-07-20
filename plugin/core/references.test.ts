import { test, expect } from "bun:test";
import { dirname } from "node:path";
import { fileURLToPath } from "node:url";
import {
  discoverReferences,
  loadReferenceVocabulary,
  validateReference,
} from "./references.js";

// The repo contract is found by walking up from this test file's directory.
const refPath = discoverReferences(dirname(fileURLToPath(import.meta.url)));
if (!refPath) throw new Error("schema/references.yaml not found from test dir");
const vocab = loadReferenceVocabulary(refPath);

test("kind enum contains at least issue and url", () => {
  expect(vocab.kinds).toContain("issue");
  expect(vocab.kinds).toContain("url");
});

test('validateReference("issue", "wb#42") is valid', () => {
  const r = validateReference("issue", "wb#42", vocab);
  expect(r.valid).toBe(true);
  expect(r.errors).toHaveLength(0);
});

test('validateReference("not-a-kind", "wb#42") is invalid and names the kind', () => {
  const r = validateReference("not-a-kind", "wb#42", vocab);
  expect(r.valid).toBe(false);
  expect(r.errors.join(" ")).toContain("not-a-kind");
});

test('validateReference("issue", "") is invalid (empty target)', () => {
  const r = validateReference("issue", "", vocab);
  expect(r.valid).toBe(false);
  expect(r.errors.join(" ")).toContain("target");
});

test("a whitespace-only target is treated as empty", () => {
  const r = validateReference("issue", "   ", vocab);
  expect(r.valid).toBe(false);
});

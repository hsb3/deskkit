import { test, expect } from "bun:test";
import { mkdtempSync, mkdirSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { knowledgeIndex } from "./knowledge.js";

function mkKnowledge(files: Record<string, string>): string {
  const root = mkdtempSync(join(tmpdir(), "ds-know-"));
  for (const [rel, body] of Object.entries(files)) {
    const abs = join(root, rel);
    mkdirSync(join(abs, ".."), { recursive: true });
    writeFileSync(abs, body);
  }
  return root;
}

test("indexes *.md recursively, excludes the profile files", () => {
  const root = mkKnowledge({
    "profile.yaml": "schema_version: 1\n",
    "profile.example.yaml": "schema_version: 1\n",
    "profile.md": "---\nschema_version: 1\n---\n",
    "glossary.md": "one two three",
    "background/history.md": "alpha beta",
  });
  const idx = knowledgeIndex(root);
  const paths = idx.entries.map((e) => e.path);
  expect(paths).toEqual(["background/history.md", "glossary.md"]); // sorted, profiles excluded
  const glossary = idx.entries.find((e) => e.path === "glossary.md")!;
  expect(glossary.words).toBe(3);
  expect(glossary.contentIncluded).toBe(true);
  expect(glossary.content).toBe("one two three");
});

test("over-budget files are index-only (content omitted), deterministic order", () => {
  const root = mkKnowledge({
    "a.md": "x".repeat(50),
    "b.md": "y".repeat(50),
    "c.md": "z".repeat(50),
  });
  const idx = knowledgeIndex(root, 60); // fits a.md (50) then b.md would exceed
  expect(idx.entries.map((e) => e.path)).toEqual(["a.md", "b.md", "c.md"]);
  expect(idx.entries[0]!.contentIncluded).toBe(true);
  expect(idx.entries[1]!.contentIncluded).toBe(false);
  expect(idx.entries[1]!.content).toBeUndefined();
  expect(idx.entries[2]!.contentIncluded).toBe(false);
  // metadata always present even when content omitted
  expect(idx.entries[1]!.bytes).toBe(50);
  expect(idx.bytesIncluded).toBe(50);
});

test("missing knowledge dir yields an empty index, not an error", () => {
  const idx = knowledgeIndex(join(tmpdir(), "ds-does-not-exist-xyz"));
  expect(idx.fileCount).toBe(0);
  expect(idx.entries).toEqual([]);
});

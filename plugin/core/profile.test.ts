import { test, expect } from "bun:test";
import { mkdtempSync, mkdirSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import {
  discoverProfile,
  extractFrontmatter,
  indexTree,
  loadProfile,
  profileScalar,
  scalarString,
} from "./profile.js";

function mkDesk(files: Record<string, string>): string {
  const root = mkdtempSync(join(tmpdir(), "ds-profile-"));
  for (const [rel, body] of Object.entries(files)) {
    const abs = join(root, rel);
    mkdirSync(join(abs, ".."), { recursive: true });
    writeFileSync(abs, body);
  }
  return root;
}

test("discoverProfile walks up from a subdir", () => {
  const root = mkDesk({ "_knowledge/profile.yaml": "schema_version: 1\n" });
  const sub = join(root, "a", "b", "c");
  mkdirSync(sub, { recursive: true });
  expect(discoverProfile(sub)).toBe(join(root, "_knowledge", "profile.yaml"));
});

test("discoverProfile extension precedence yaml > yml > json > md", () => {
  const root = mkDesk({
    "_knowledge/profile.yaml": "schema_version: 1\n",
    "_knowledge/profile.yml": "schema_version: 1\n",
    "_knowledge/profile.json": "{}",
    "_knowledge/profile.md": "---\n---\n",
  });
  expect(discoverProfile(root)).toBe(join(root, "_knowledge", "profile.yaml"));
});

test("discoverProfile returns null when none up to root", () => {
  const root = mkDesk({ "README.md": "no profile here\n" });
  expect(discoverProfile(root)).toBeNull();
});

test("loadProfile parses .md YAML frontmatter", () => {
  const root = mkDesk({
    "_knowledge/profile.md": "---\nschema_version: 1\ndesk:\n  name: example-desk\n---\n\n# prose\n",
  });
  const p = loadProfile(join(root, "_knowledge", "profile.md"));
  expect(profileScalar(p, "desk.name")).toBe("example-desk");
});

test("loadProfile parses .json", () => {
  const root = mkDesk({ "_knowledge/profile.json": '{"schema_version":1,"repos":{"default":"octocat/example-repo"}}' });
  const p = loadProfile(join(root, "_knowledge", "profile.json"));
  expect(profileScalar(p, "repos.default")).toBe("octocat/example-repo");
});

test("extractFrontmatter returns empty on no fence / unterminated fence", () => {
  expect(extractFrontmatter("# just markdown\n")).toBe("");
  expect(extractFrontmatter("---\nkey: v\n")).toBe(""); // unterminated
  expect(extractFrontmatter("---\nkey: v\n---\nbody")).toBe("key: v");
});

test("scalarString renders scalars, empty for maps/lists/absent", () => {
  expect(scalarString("x")).toBe("x");
  expect(scalarString(true)).toBe("true");
  expect(scalarString(42)).toBe("42");
  expect(scalarString(1.5)).toBe("1.5");
  expect(scalarString(null)).toBe("");
  expect(scalarString(undefined)).toBe("");
  expect(scalarString({ a: 1 })).toBe("");
  expect(scalarString([1, 2])).toBe("");
});

test("indexTree resolves dotted path, undefined through non-maps", () => {
  const p = { a: { b: { c: "deep" } } };
  expect(indexTree(p, ["a", "b", "c"])).toBe("deep");
  expect(indexTree(p, ["a", "x"])).toBeUndefined();
  expect(indexTree(p, ["a", "b", "c", "d"])).toBeUndefined();
});

import { test, expect } from "bun:test";
import { mkdtempSync, mkdirSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { discoverSchema } from "./schema.js";
import {
  TOOLS,
  knowledgeIndexTool,
  profileGet,
  profileValidate,
  templateRender,
  type ToolContext,
} from "./tools.js";

const schemaPath = discoverSchema(dirname(fileURLToPath(import.meta.url)));
if (!schemaPath) throw new Error("schema not found from test dir");

function mkDesk(profileBody: string, extra: Record<string, string> = {}): string {
  const root = mkdtempSync(join(tmpdir(), "ds-tools-"));
  mkdirSync(join(root, "_knowledge"), { recursive: true });
  writeFileSync(join(root, "_knowledge", "profile.yaml"), profileBody);
  for (const [rel, body] of Object.entries(extra)) {
    const abs = join(root, rel);
    mkdirSync(dirname(abs), { recursive: true });
    writeFileSync(abs, body);
  }
  return root;
}

const VALID_PROFILE =
  "schema_version: 1\n" +
  "desk:\n  name: example-desk\n" +
  "repos:\n  default: octocat/example-repo\n" +
  "identity:\n  github:\n    personal: octocat\n";

function ctxFor(cwd: string): ToolContext {
  return { cwd, schemaPath: schemaPath! };
}

// --- AC3: schema shape / single source ---

test("exactly the four tools are exported, in order, with input schemas", () => {
  expect(TOOLS.map((t) => t.name)).toEqual([
    "profile_get",
    "profile_validate",
    "template_render",
    "knowledge_index",
  ]);
  for (const t of TOOLS) {
    expect(typeof t.description).toBe("string");
    expect((t.inputSchema as { type: string }).type).toBe("object");
    expect(t.inputSchema).toHaveProperty("properties");
  }
});

test("required-arg tools declare required[], optional-arg tools do not", () => {
  expect((profileGet.inputSchema as { required?: string[] }).required).toEqual(["path"]);
  expect((templateRender.inputSchema as { required?: string[] }).required).toEqual(["template"]);
  expect((profileValidate.inputSchema as { required?: string[] }).required).toBeUndefined();
  expect((knowledgeIndexTool.inputSchema as { required?: string[] }).required).toBeUndefined();
});

// --- profile_get ---

test("profile_get resolves a dotted value", () => {
  const desk = mkDesk(VALID_PROFILE);
  expect(profileGet.handler({ path: "repos.default" }, ctxFor(desk))).toEqual({
    path: "repos.default",
    value: "octocat/example-repo",
  });
});

test("profile_get fails loud with a suggested-key hint on absent key", () => {
  const desk = mkDesk(VALID_PROFILE);
  expect(() => profileGet.handler({ path: "identity.github.bogus" }, ctxFor(desk))).toThrow(
    /not found or empty.*Available keys under "identity.github": personal/s,
  );
});

// --- profile_validate ---

test("profile_validate reports valid for a good profile", () => {
  const desk = mkDesk(VALID_PROFILE);
  const r = profileValidate.handler({}, ctxFor(desk));
  expect(r.valid).toBe(true);
  expect(r.errors).toHaveLength(0);
  expect(r.profilePath).toBe(join(desk, "_knowledge", "profile.yaml"));
});

test("profile_validate reports errors for a bad profile", () => {
  const desk = mkDesk("schema_version: 1\nbogus_key: 1\n");
  const r = profileValidate.handler({}, ctxFor(desk));
  expect(r.valid).toBe(false);
  expect(r.errors.join(" ")).toContain("bogus_key");
});

// --- template_render ---

test("template_render substitutes over the discovered profile", () => {
  const desk = mkDesk(VALID_PROFILE);
  expect(
    templateRender.handler({ template: "repo={{profile.repos.default}}" }, ctxFor(desk)),
  ).toEqual({ rendered: "repo=octocat/example-repo" });
});

test("template_render fails loud on a missing required placeholder", () => {
  const desk = mkDesk(VALID_PROFILE);
  expect(() => templateRender.handler({ template: "{{profile.no.such}}" }, ctxFor(desk))).toThrow(
    /missing required key/,
  );
});

// --- knowledge_index ---

test("knowledge_index indexes background md under the profile's _knowledge dir", () => {
  const desk = mkDesk(VALID_PROFILE, {
    "_knowledge/background/context.md": "some background prose here",
  });
  const idx = knowledgeIndexTool.handler({}, ctxFor(desk));
  expect(idx.entries.map((e) => e.path)).toContain("background/context.md");
});

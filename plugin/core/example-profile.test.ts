import { test, expect } from "bun:test";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { loadProfile } from "./profile.js";
import { compileValidator, discoverSchema, loadSchemaObject, validateProfile } from "./schema.js";

// AC8 proof gap: schema.test.ts already proves rejection (unknown key, bad slug,
// loadAndValidateProfile throwing) and custom-block acceptance against SYNTHETIC objects.
// The missing proof is that the *actually shipped* artifact — the desk-setup scaffold's
// _knowledge/profile.example.yaml, the file a stranger copies to start — validates against
// schema v1, and that the reject-at-top-level / accept-under-`custom:` pairing holds for a
// realistic agent-written key.

const here = dirname(fileURLToPath(import.meta.url));
const schemaPath = discoverSchema(here);
if (!schemaPath) throw new Error("schema/profile.schema.yaml not found from test dir");
const validator = compileValidator(loadSchemaObject(schemaPath));

// The shipped example lives with the desk-setup scaffold template (the single home for it, K12):
// plugins/desk-standard/skills/desk-setup/assets/template/_knowledge/profile.example.yaml.
const examplePath = join(
  here,
  "..",
  "..",
  "plugins",
  "desk-standard",
  "skills",
  "desk-setup",
  "assets",
  "template",
  "_knowledge",
  "profile.example.yaml",
);

test("shipped profile.example.yaml validates against schema v1 (AC8: the profile validates)", () => {
  const profile = loadProfile(examplePath);
  const r = validateProfile(profile, validator);
  // name any violation explicitly so a schema/example drift fails loud
  expect(r.errors).toEqual([]);
  expect(r.valid).toBe(true);
});

test("an agent-written unplanned key is rejected at top level, accepted under custom: (AC8)", () => {
  // an agent invents a field schema v1 does not know — rejected loudly, no silent pass
  const rejected = validateProfile({ schema_version: 1, deploy_target: "prod-cluster" }, validator);
  expect(rejected.valid).toBe(false);
  expect(rejected.errors.join(" ")).toContain("deploy_target");

  // the sanctioned home for the same data — passes without a schema bump
  const accepted = validateProfile(
    { schema_version: 1, custom: { deploy_target: "prod-cluster" } },
    validator,
  );
  expect(accepted.valid).toBe(true);
  expect(accepted.errors).toEqual([]);
});

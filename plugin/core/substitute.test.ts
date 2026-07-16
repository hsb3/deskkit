import { test, expect } from "bun:test";
import { MissingPlaceholderError, pick, resolveValue, substitute } from "./substitute.js";

const profile = {
  desk: { name: "example-desk" },
  repos: { default: "octocat/example-repo" },
  identity: { github: { personal: "octocat" } },
};

test("substitutes profile placeholders", () => {
  expect(substitute("repo={{profile.repos.default}}", profile)).toBe("repo=octocat/example-repo");
  expect(substitute("{{ profile.identity.github.personal }}", profile)).toBe("octocat");
});

test("substitutes env placeholders", () => {
  process.env.DS_TEST_VAR = "envval";
  expect(substitute("v={{env.DS_TEST_VAR}}", profile)).toBe("v=envval");
  delete process.env.DS_TEST_VAR;
});

test("uses default when value absent/empty", () => {
  expect(substitute('{{profile.missing.key || "fallback"}}', profile)).toBe("fallback");
  expect(substitute('{{env.DS_ABSENT || "def"}}', profile)).toBe("def");
});

test("handles escaped quotes in default", () => {
  expect(substitute('{{profile.absent || "say \\"hi\\""}}', profile)).toBe('say "hi"');
});

test("fails loud, collecting ALL missing offenders in one error", () => {
  let err: unknown;
  try {
    substitute("{{profile.a.b}} and {{env.NOPE_X}} and {{profile.repos.default}}", profile);
  } catch (e) {
    err = e;
  }
  expect(err).toBeInstanceOf(MissingPlaceholderError);
  const m = err as MissingPlaceholderError;
  expect(m.missing).toHaveLength(2);
  expect(m.message).toContain("{{profile.a.b}}");
  expect(m.message).toContain("{{env.NOPE_X}}");
  // the resolvable one is applied in the partial text
  expect(m.partial).toContain("octocat/example-repo");
});

test("pick precedence env > profile > default", () => {
  process.env.DS_PICK = "fromenv";
  expect(pick("DS_PICK", "fromprofile", "def")).toBe("fromenv");
  process.env.DS_PICK = "";
  expect(pick("DS_PICK", "fromprofile", "def")).toBe("fromprofile"); // empty env = unset
  delete process.env.DS_PICK;
  expect(pick("DS_PICK", "fromprofile", "def")).toBe("fromprofile");
  expect(pick("DS_PICK", "", "def")).toBe("def");
});

test("resolveValue threads profile through pick", () => {
  expect(resolveValue(profile, "DS_UNSET_ENV", "desk.name", "d")).toBe("example-desk");
  expect(resolveValue(profile, "DS_UNSET_ENV", "desk.absent", "d")).toBe("d");
});

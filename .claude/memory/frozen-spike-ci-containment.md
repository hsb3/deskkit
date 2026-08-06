---
name: frozen-spike-ci-containment
description: descoped OpenCode spike at plugin/opencode/ still runs under bun test / CI despite README claiming otherwise
metadata:
  type: project
---

`plugin/opencode/` is a descoped "frozen reference" spike. Its own README and commit 850284f
claim it is "wired into nothing / not wired into the package scripts, the manifest, or CI."

**This is false for the test runner.** `plugin/package.json` `test` = bare `bun test`, which
discovers ALL `*.test.ts` including `opencode/plugin.test.ts` (8 tests). CI's `bun test` step
(`.github/workflows/ci.yml`, working-directory plugin) runs it too. So the descoped spike is a
hard CI gate dependency.

Correctly contained: `tsc -p tsconfig.json` build EXCLUDES it (include = core+mcp only),
`check-core-purity` scans only core, nothing imports it, `install-opencode.mjs` is unreferenced.
The `@opencode-ai/plugin` import is type-only (local `.d.ts` shim), so runtime resolves without
the package — currently harmless (8/8 pass), but rot in unmaintained code would block CI.

**How to apply:** never accept a "wired into nothing" claim on face — run
`fd '\.test\.ts$' plugin/` and check whether the test script/CI scope it. To match stated intent,
the fix is to scope `bun test core/` (or a bunfig testMatch), not to trust the README.

**Update (2026-07-17, branch feat/repo-plumbing):** RESOLVED there — `plugin/package.json`
`test` was `bun test core mcp` (scoped), so the opencode spike no longer ran under CI.

**Update (2026-07-23, main):** REVERTED — the glob is now
`"test": "bun test core mcp opencode desk-persona"` (`plugin/package.json:13`). opencode is
BACK in the test scope, so its tests DO run under `make test` (Makefile:38 `bun run test`) and
CI (`.github/workflows/ci.yml:151` `bun run test`). This directly REFUTES any body claiming the
spike is "wired into nothing: not the package scripts, not CI" — a claim that keeps propagating
from the STALE `plugin/opencode/README.md` (which still says "the package test script and CI
scope to core and mcp"). The README is the wrong-mechanism citation; the package.json glob is
authoritative. Re-run `grep -n '"test"' plugin/package.json` every pass — the glob churns
(desk-pm -> desk-persona rename, opencode in/out); never trust the README or a prior memory.

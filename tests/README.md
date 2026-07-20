_Declared home for this repo's test suites — a signpost, not a suite. The tests live with
their products; this page says where._
Status: active (2026-07-19)

# tests/

This is a **two-product monorepo**, and each product's tests are co-located with its source,
not hoisted into a root `tests/` tree. This directory exists to satisfy the repo standard's
`tests/` requirement (ROOT-10) and to point you at the real suites and their canonical
entry points.

## Where the suites live

| Suite | Location | Runner | What it covers |
|---|---|---|---|
| Plugin (TS) | `plugin/{core,mcp,opencode,desk-pm}` | `bun test` (59 tests) | Harness-pure core, MCP tool surface, schema validation, PM tool parity. |
| Librarian (Go) | `librarian/...` (`*_test.go`) | `go test ./...` (314 tests) | Store, write-boundary, PM engine, CLI/MCP surfaces, regression guards. |
| Integration gate | `librarian/verify.sh` | `bash librarian/verify.sh` | End-to-end sweep → patrol → propose-fix → apply-fix → restore, store self-init, XDG store home, desk open-guard (currently 48 checks; the script prints its own `N passed` summary). |

## Canonical entry points

Run these from the repo root — the Makefile is the task interface (`make help` lists targets):

```bash
make test     # fast unit tests, both lanes: plugin (bun) + librarian (go test ./...)
make verify   # the librarian integration gate (librarian/verify.sh, throwaway scratch desk)
```

`make check` (neutrality lint + kit-drift + core purity + actionlint) and
`node scripts/check-version-sync.mjs` round out the gate set; CI (`.github/workflows/ci.yml`)
runs the same checks. See [`../CLAUDE.md`](../CLAUDE.md) for the full command surface and the
one rule that matters (identity-neutrality).

> **Why no test files here:** hoisting suites to a root `tests/` would split each product's
> tests from its source and its language toolchain (Go modules under `librarian/`, bun under
> `plugin/`). The standard requires the directory to exist, not that every suite move into it.

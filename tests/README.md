_Declared home for this repo's test suites — a signpost, not a suite. The tests live with
their products; this page says where._
Status: active (2026-08-18)

# tests/

This repo ships **one binary, one plugin bundle, one schema**, and the Go tests are co-located
with the source they cover rather than hoisted into a root `tests/` tree. This directory exists
to satisfy the repo standard's `tests/` requirement (ROOT-10) and to point you at the real
suites and their canonical entry points.

## Where the suites live

| Suite | Location | Runner | What it covers |
|---|---|---|---|
| deskkit (Go) | `cmd/`, `internal/` (`*_test.go`, repo root module) | `go test ./...` | Store, write-boundary, PM engine, profile/librarian/pm tool specs, CLI/MCP/web surfaces, schema-embed drift guards, regression guards. |
| Integration gate | `verify.sh` (repo root) | `bash verify.sh` | End-to-end sweep → patrol → propose-fix → apply-fix → restore, store self-init, XDG store home, desk open-guard. The script prints its own `N passed` summary. |
| System behaviour (e2e) | `e2e/` (repo root) | `bash e2e/e2e.sh` | Cold-start → profile → librarian → PM → surfaces → release-shaped, on a throwaway desk; offline, no LLM key. |

Suite sizes move with every change, so this page names the runners rather than a count — read
the number off the run, not off this table.

## Canonical entry points

Run these from the repo root — the Makefile is the task interface (`make help` lists targets).
Run them **bare, never piped**: a pipe masks the exit code.

```bash
make test     # fast unit tests: go test ./...
make verify   # the librarian integration gate (verify.sh, throwaway scratch desk)
make e2e      # the end-to-end system-behaviour suite
```

`make check` (neutrality lint + the drift guards + doc-link integrity + shellcheck + actionlint)
and `node scripts/check-version-sync.mjs` round out the gate set; CI
(`.github/workflows/ci.yml`) runs the same checks. See [`../CLAUDE.md`](../CLAUDE.md) for the full
command surface and the one rule that matters (identity-neutrality).

> **Why no test files here:** hoisting the suites to a root `tests/` would split them from the
> source they cover and from the Go module that owns them. The standard requires the directory to
> exist, not that every suite move into it.

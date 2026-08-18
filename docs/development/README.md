_Contributor overview for desk-standard: how to build, test, regenerate media, and cut a release.
The docs index is [`../README.md`](../README.md)._
Status: active

# Development

One product lane sits under this repo: **`deskkit`**, a single Go binary (`librarian/`) carrying the
`profile`, `librarian`, and `pm` tool modules. Alongside it, `plugins/desk-persona/` is the Claude
Code bundle that mounts that binary — authored in place, nothing generated. The **root `Makefile`
is the canonical task interface** — `make help` lists every target; CI
(`.github/workflows/ci.yml`) runs the same checks.

## Build

```bash
make build      # build the deskkit binary (go — version-stamped)
make install    # build + install it to ~/.local/bin (override PREFIX=/usr/local)
make setup      # install the lefthook pre-commit hooks (first-time setup)
```

The version is stamped from the root `VERSION` file via ldflags; a bare `go build` reports `dev`.
Go 1.25 is the floor (PocketBase's `go.mod`); Node is needed only to run the `scripts/*.mjs` gates.
See [install-and-build.md](install-and-build.md).

## Test & gates

Run these before claiming any change done — they mirror CI. Run them **bare, never piped** — a pipe
masks the exit code.

| Command | What it checks |
|---|---|
| `make check` | the repo gates: neutrality lint (+ self-test), the drift guards, doc-link integrity, shellcheck, actionlint |
| `make test` | `go test ./...`, including the schema-embed drift guards |
| `make verify` | the librarian integration gate (throwaway scratch desk; self-init, write-boundary, byte-exact restore, open-guard) |
| `make e2e` | the end-to-end system-behaviour suite on a throwaway desk (offline, no LLM key) |
| `node scripts/check-version-sync.mjs` | `VERSION` vs the shipped manifests |

`make package` generates nothing — it survives as an informational echo saying so, because the
bundle under `plugins/` is authored in place.

## Demo media (VHS)

The demo GIFs in [`../assets/`](../assets/) are **generated artifacts** — never hand-edited. Their
source `.tape` files live in [`../../scripts/vhs-tapes/`](../../scripts/vhs-tapes/), kept separate from the GIF output.

```bash
make media      # rebuild the librarian, then drive vhs over every tape into ../assets/*.gif
```

The recorder (`scripts/record-media.sh`) is hermetic: it seeds throwaway scratch desks under a
scratch `HOME`/`XDG_DATA_HOME` so it never touches a real store. `chat.tape` needs a live
`ANTHROPIC_API_KEY` (its demo makes a real tool call) and is skipped when the key is absent, so
the default `make media` run stays offline. Requires `vhs` + `ttyd` (`brew install vhs ttyd`).

## Release

The binary and the bundle ship under one version: bump `VERSION` + the shipped manifests → move
`[Unreleased]` CHANGELOG entries into a dated section → `make release-prep` →
`git tag v<version> && git push --tags`. `make release-prep` aborts on the first failure and never
auto-tags. `make version-status` flags unreleased drift; `check-changelog.mjs` gates a tag against
a documented CHANGELOG section (ADR 0005, DESK-27).

## Canonical references

- [`specs/pocket-librarian-v1-spec.md`](specs/pocket-librarian-v1-spec.md) — the librarian's build spec.
- [`specs/pm-system-v1-spec.md`](specs/pm-system-v1-spec.md) — the PM system's build spec.
- Architecture Decision Records — on the project board as `DECISION` tasks (cited from code
  and docs as a bare `ADR NNNN`; no path, no board id).

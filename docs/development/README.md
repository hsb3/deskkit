_Contributor overview for desk-standard: how to build, test, regenerate media, and cut a release.
The docs index is [`../README.md`](../README.md)._
Status: active

# Development

Two product lanes sit under one repo: the **plugin** (harness-pure TypeScript core + stdio MCP
server, built/tested with `bun`) and the **librarian** (`pocket-librarian`, a single Go binary).
The **root `Makefile` is the canonical task interface** — `make help` lists every target; CI
(`.github/workflows/ci.yml`) runs the same checks.

## Build

```bash
make build      # build both lanes (plugin via bun/tsc, librarian via go — version-stamped)
make install    # build + install the librarian binary to ~/.local/bin (override PREFIX=/usr/local)
make setup      # install plugin deps + lefthook pre-commit hooks (first-time setup)
```

The librarian's version is stamped from the root `VERSION` file via ldflags; a bare `go build`
reports `dev`. See [releasing.md](releasing.md).

## Test & gates

Run these before claiming any change done — they mirror CI:

| Command | What it checks |
|---|---|
| `make check` | neutrality lint (+ self-test), plugin core purity, actionlint |
| `make test` | plugin `bun test` + librarian `go test ./...` |
| `make verify` | the librarian integration gate (throwaway scratch desk; self-init, write-boundary, byte-exact restore, open-guard) |
| `make package` | regenerate the marketplace plugin bundle + drift-guard it |
| `node scripts/check-version-sync.mjs` | `VERSION` vs the three shipped manifests |

## Demo media (VHS)

The demo GIFs in [`../media/`](../media/) are **generated artifacts** — never hand-edited. Their
source `.tape` files live in [`tapes/`](tapes/), kept separate from the GIF output.

```bash
make media      # rebuild the librarian, then drive vhs over every tape into ../media/*.gif
```

The recorder (`scripts/record-media.sh`) is hermetic: it seeds throwaway scratch desks under a
scratch `HOME`/`XDG_DATA_HOME` so it never touches a real store. `chat.tape` needs a live
`ANTHROPIC_API_KEY` (its demo makes a real tool call) and is skipped when the key is absent, so
the default `make media` run stays offline. Requires `vhs` + `ttyd` (`brew install vhs ttyd`).

## Release

Both products ship under one version. The full runbook is [releasing.md](releasing.md); in short:
bump `VERSION` + the three manifests → move `[Unreleased]` CHANGELOG entries into a dated section
→ `make release-prep` → `git tag v<version> && git push --tags`. `make version-status` flags
unreleased drift; `check-changelog.mjs` gates a tag against a documented CHANGELOG section
([ADR 0005](../decisions/0005-versioning-and-changelog.md)).

## Canonical references

- [`../pocket-librarian-v1-spec.md`](../pocket-librarian-v1-spec.md) — the librarian's build spec.
- [`../decisions/`](../decisions/) — Architecture Decision Records (append-only; cited from code
  and docs — their paths are kept stable).
- `../../_meta/build-brief.md` — the original build brief (repo shape, acceptance criteria).

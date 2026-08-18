_The developer / advanced-operator install paths: build the binary from source, pull a specific
prebuilt release asset, and the env-var + store-path lore. The user install path
is [`../getting-started.md`](../usage/getting-started.md); this page holds the toolchain details it
deliberately leaves out._
Status: active

# Installing and building — developer track

The [user getting-started guide](../usage/getting-started.md) installs a **prebuilt `deskkit` binary**
and drives everything from the TUI, with no toolchain. This page is the other side of that split:
building from source, pulling a release asset directly, and the environment variables that
override desk/store resolution. If you are a desk owner and not a contributor, you do not need
anything here.

## Build from source

`deskkit` is a single Go binary and the only thing built here (Go 1.25.0 is pinned — PocketBase's
`go.mod` floors it). Node is needed both for the repo's `scripts/*.mjs` gates and for the `web/`
SPA that `make build` compiles and embeds via `go:embed`; a bare `go build` skips the SPA step and
serves a placeholder page at `/`. From a clone:

```console
$ make build          # SPA + binary; the binary lands at ./deskkit
$ make install        # build + install deskkit to ~/.local/bin (override with PREFIX=/usr/local)
$ deskkit --version
deskkit version 0.8.0
```

A `make`-built or release binary reports its release version via `--version`; a binary that prints
`dev` was built with a bare `go build` (no version stamp) — pin such a build from its source commit
instead. Full contributor setup is in [`README.md`](README.md).

## Prebuilt binary

The release workflow publishes a `deskkit` binary for macOS and Linux (amd64 + arm64) on every
`v*` tag — no Go toolchain needed to run it elsewhere.

The one-liner in the root README (`curl -fsSL
https://raw.githubusercontent.com/hsb3/deskkit/main/install.sh | bash`) covers the normal case.
To pin a specific release asset instead — or to script the download without the installer — pull
it with `gh`:

```bash
mkdir -p ~/.local/bin
os=$(uname -s | tr '[:upper:]' '[:lower:]')                 # darwin | linux
arch=$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')   # amd64 | arm64
gh release download --repo hsb3/deskkit \
  --pattern "deskkit_*_${os}_${arch}" \
  --output ~/.local/bin/deskkit --clobber
chmod +x ~/.local/bin/deskkit
deskkit --version
```

## Env-var overrides and where the store lives

Run `deskkit` from **inside a desk** (a folder whose `_knowledge/profile.yaml` sets `desk.name`
and `root: "."`) and no environment variables are needed — it walks up from your working directory,
reads `desk.name` as the store name, and uses the folder that owns `_knowledge/` as the desk root.
The two env vars are the **override**, not the requirement, and they win over the profile:

| Variable    | What it sets             | When you need it                                                                                                     |
| ----------- | ------------------------ | -------------------------------------------------------------------------------------------------------------------- |
| `DESK_ROOT` | the desk tree to steward | running the binary from outside the desk tree (e.g. a dev build from the repo root), or a bare folder with no profile |
| `DESK_NAME` | the unique store name    | same — and it must be unique across your desks                                                                       |

```bash
export DESK_ROOT=/path/to/desk DESK_NAME=my-desk    # only when outside a profiled desk
```

Note: `deskkit init` always names the desk from the folder's basename, not from any `DESK_NAME`
you already exported — the two agree only if the folder happens to be named accordingly.

The store lives **outside** the desk tree, at `$XDG_DATA_HOME/deskkit/<DESK_NAME>/` (falling back
to `~/.local/share/deskkit/<DESK_NAME>/`), so the librarian never indexes its own database and a
cloud-synced desk folder never corrupts a live SQLite file. See
ADR 0002 (DESK-24).

On an interactive terminal you don't even have to set these: any store-touching command run where
config can't resolve offers to scaffold the folder as a desk (`Set up this folder as a desk?
[Y/n]`) and continues on accept. `--no-input` (or a non-TTY, e.g. CI) skips the prompt and keeps
the fail-closed error. Operator detail — provider/key resolution, the admin console, the MCP
surface — is in [`../usage/deskkit-reference.md`](../usage/deskkit-reference.md).

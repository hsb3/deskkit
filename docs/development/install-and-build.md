_The developer / advanced-operator install paths: build the librarian from source, pull a prebuilt
release asset while the repo is private, and the env-var + store-path lore. The user install path
is [`../getting-started.md`](../getting-started.md); this page holds the toolchain details it
deliberately leaves out._
Status: active

# Installing and building — developer track

The [user getting-started guide](../getting-started.md) installs a **prebuilt `deskkit` binary**
and drives everything from the TUI, with no toolchain. This page is the other side of that split:
building from source, the release-asset download while the repo is private, and the environment
variables that override desk/store resolution. If you are a desk owner and not a contributor, you
do not need anything here.

## Build from source

The librarian is a single Go binary (Go 1.25.0 is pinned — PocketBase's `go.mod` floors it). From
a clone:

```console
$ make build          # builds both lanes; the librarian binary lands at librarian/deskkit
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

**repo is private** download the release asset with an authenticated `gh`:

```bash
mkdir -p ~/.local/bin
os=$(uname -s | tr '[:upper:]' '[:lower:]')                 # darwin | linux
arch=$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')   # amd64 | arm64
gh release download --repo hsb3/desk-standard \
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
| `DESK_ROOT` | the desk tree to steward | running the binary from outside the desk tree (e.g. a dev build from `librarian/`), or a bare folder with no profile |
| `DESK_NAME` | the unique store name    | same — and it must be unique across your desks                                                                       |

```bash
export DESK_ROOT=/path/to/desk DESK_NAME=my-desk    # only when outside a profiled desk
```

Note: `deskkit init` always names the desk from the folder's basename, not from any `DESK_NAME`
you already exported — the two agree only if the folder happens to be named accordingly.

The store lives **outside** the desk tree, at `$XDG_DATA_HOME/deskkit/<DESK_NAME>/` (falling back
to `~/.local/share/deskkit/<DESK_NAME>/`), so the librarian never indexes its own database and a
cloud-synced desk folder never corrupts a live SQLite file. See
[`decisions/0002-multi-desk-topology-store-per-desk.md`](../decisions/0002-multi-desk-topology-store-per-desk.md).

On an interactive terminal you don't even have to set these: any store-touching command run where
config can't resolve offers to scaffold the folder as a desk (`Set up this folder as a desk?
[Y/n]`) and continues on accept. `--no-input` (or a non-TTY, e.g. CI) skips the prompt and keeps
the fail-closed error. Operator detail — provider/key resolution, the admin console, the MCP
surface — is in [`../../librarian/README.md`](../../librarian/README.md).

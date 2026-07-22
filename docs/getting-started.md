_From zero to a personalized, self-patrolled desk in one sitting: install the plugin, fill your profile, build the librarian, run your first sweep and patrol._
Status: active

# Getting started

desk-standard gives you two things that share one schema and one personalization model: a
**Claude Code plugin** that stands up and maintains an executive desk, and a **deskkit**
binary that indexes that desk and repairs convention violations under a byte-exact undo.
Nothing you install carries a name, org, or repo — you personalize once, in
`_knowledge/profile.yaml`, and never by editing a shipped file.

This page gets you productive in one sitting. Deeper guides: `plugin-guide.md` (the four
skills), `librarian-guide.md` (the daily loop), and `../librarian/README.md` (operator
reference).

![First sweep: index the desk, then see findings and orphans](media/sweep-and-findings.gif)

## 1. Install the plugin

This repo is its own Claude Code plugin marketplace. From any project, add the marketplace
and install the plugin:

```bash
# documented-not-run here: these require an interactive Claude Code session
claude plugin marketplace add hsb3/desk-standard
claude plugin install desk-standard@desk-standard
```

The install copies only `plugin/claude-plugin/` into the plugin cache, so the plugin is
self-contained (its MCP server and schema ship inside it). For local development, point
Claude Code straight at the source tree instead:

```bash
claude --plugin-dir ./plugin/claude-plugin
```

Inside that session, the `desk-setup` skill scaffolds a new desk; `conventions-standard`
and `harvest-loop` run the standing checks and the periodic improvement pass. See
`plugin-guide.md` for when to reach for each.

## 2. Fill your profile

The `desk-setup` skill scaffolds your desk with a shipped placeholder profile at
`_knowledge/profile.example.yaml`. Copy it and fill in your identifiers — handles, repos, board,
machines, preferences. **Never edit files under `plugin/` or `librarian/` to personalize.**

```bash
# run inside your desk — the scaffold ships _knowledge/profile.example.yaml (this repo does not)
cp _knowledge/profile.example.yaml _knowledge/profile.yaml
$EDITOR _knowledge/profile.yaml
```

Working from a plain folder instead (no plugin scaffold to copy from)? Once the
librarian is built (step 3), `deskkit init [dir]` is the fastest path to a
working profile: it writes the minimal `_knowledge/profile.yaml` a folder needs — desk
name from the folder's basename, `root: "."` — with `--force` to overwrite and
`--with-env` to also scaffold a `.env` naming your LLM API-key env var. It never touches
anything outside `_knowledge/` and `.env`, and never creates the store.

The profile validates against schema v1. You can check it with the plugin's
`profile_validate` MCP tool (see `plugin-guide.md`); a well-formed profile returns
`{"valid":true,"errors":[]}`, and every scaffold placeholder resolves from it via
`template_render` — so a missing required key fails loud instead of writing an empty string.

Keep credentials out of the profile: it holds identifiers only. `secrets_ref.llm_api_key`
names the *env var* that holds your key, never the key itself.

## 3. Build the librarian

The librarian is a single Go binary. Build it (Go 1.25.0 is pinned — PocketBase's `go.mod`
floors it):

```console
$ cd librarian
$ make build          # -> ./deskkit
$ ./deskkit --version
deskkit version 0.5.0
```

A `make`-built or release binary reports its release version via `--version`; a binary that
prints `dev` was built with a bare `go build` (no version stamp) — pin such a build from its
source commit instead.

Prefer a prebuilt binary? The release workflow publishes a `deskkit` binary for macOS and
Linux (amd64 + arm64) on every `v*` tag — no Go toolchain needed to run it elsewhere.

**While this repo is private** (it stays private until v1.0.0), download the release asset with
an authenticated `gh`; the public `install.sh` one-liner below only works once the repo is public:

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

**Once the repo is public,** `install.sh` at the repo root does download + sha256-verify +
install in one step:

```bash
curl -fsSL https://raw.githubusercontent.com/hsb3/desk-standard/main/install.sh | bash
# pin a version / preview without writing anything:
#   ./install.sh --version vX.Y.Z --dry-run
```

## 4. First sweep and patrol

The librarian needs to know two things — which desk to steward (`DESK_ROOT`) and a unique
store name (`DESK_NAME`). **You already gave it both in step 2.** It walks up from your
working directory, finds `_knowledge/profile.yaml`, and reads `desk.name` as `DESK_NAME`
with the folder that owns `_knowledge/` as `DESK_ROOT`. So from your desk, with the binary on
your `PATH` (installed by step 3's `install.sh`, or via `make install`), no exports are
needed — just run:

```console
$ cd /path/to/your/desk      # the folder whose _knowledge/profile.yaml you filled
$ deskkit sweep
```

> **Env vars are the override, not the requirement.** Two cases still need them:
> - **A bare folder with no profile** (a scratch or UAT dir) — either run
>   `deskkit init` in it (writes that one-line `_knowledge/profile.yaml` for you),
>   drop the file in by hand (`desk:` with `name:` and `root: "."`), or set the vars for the
>   session: `export DESK_ROOT=/path/to/desk DESK_NAME=my-desk`.
> - **Running the dev build from `librarian/`** (`./deskkit …`) — you're outside the
>   desk tree, so the profile isn't on the walk-up path; export the two vars.
>
> Env always wins over the profile. Either way the store lives outside the desk tree, at
> `$XDG_DATA_HOME/deskkit/<DESK_NAME>/` (see
> `decisions/0002-multi-desk-topology-store-per-desk.md`).
>
> On an interactive terminal, you don't even have to remember `init`: any store-touching
> command run where config can't resolve offers to scaffold it for you ("Set up this
> folder as a desk? [Y/n]") and continues on accept. `--no-input` (or a non-TTY, e.g. CI)
> skips the prompt and keeps the prior fail-closed error.

Index the tree, then flag violations — no setup step needed, `sweep` creates the store on
first run. `sweep` and `patrol` are LLM-free and **never write desk files** — `patrol` is a
pure dry run:

```console
$ deskkit sweep
{
  "total": 4,
  "created": 4,
  "updated": 0,
  "unchanged": 0,
  "soft_deleted": 0
}

$ deskkit patrol
{
  "run_id": "patrol-20260717T171143Z",
  "files_swept": 4,
  "findings_new": 3,
  "by_rule": { "R1": 1, "R2": 1, "R3": 1 }
}
```

(Transcript from a scratch `example-desk` seeded with one violation each of R1/R2/R3.) See
what was flagged with a read-only query:

```console
$ deskkit query findings --pretty
findings: 3

R1 (1)
  tasks/wire-up-ingest.md  missing universal frontmatter: created, updated, tags, synopsis
R2 (1)
  journal/kickoff-notes.md  journal filename not yyyy-mm-dd-*.md: kickoff-notes.md
R3 (1)
  analyses/backlog-triage.md  type 'task' but file lives under 'analyses' (expected tasks/)
```

## What's next

- **Chat with it** — `deskkit chat` opens an interactive session over the same tools
  (a full-screen view on a terminal). Unlike `sweep`/`patrol`, `chat` needs an LLM: set
  `ANTHROPIC_API_KEY` for the default provider, or point `models` + `secrets_ref.llm_api_key`
  at your key in the profile. Details in `../librarian/README.md`.
- **Repair the fixable findings** — the supervised `propose-fix → apply-fix` write path and
  the byte-exact `restore` undo are the daily loop: `librarian-guide.md`.
- **Learn the skills** — greenfield setup, the rule set, the harvest loop, and brownfield
  adoption: `plugin-guide.md`.
- **Verify your build** — `make verify` runs the whole chain against a throwaway desk it
  creates and destroys; safe to run any time.

You now have a personalized desk under a plugin that maintains it and a librarian that
patrols it. That is the whole loop from a clean checkout to a self-patrolled desk.

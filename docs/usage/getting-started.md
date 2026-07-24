_From zero to a personalized, self-patrolled desk in one sitting: install the plugin, stand up your
desk, run your first sweep and patrol, and meet the TUI — all from an install command and `deskkit`
launched inside your desk._
Status: active
Audience: **desk owners** — you run a desk; you are not building the products. Assumes no build
toolchain and only a minimal terminal: one install command and `deskkit` launched inside your desk,
with the TUI and the Claude-session skills carrying the rest. Building from source, release-asset
downloads, and environment-variable lore live in the developer track
([`development/install-and-build.md`](../development/install-and-build.md)).

# Getting started

desk-standard gives you two things that share one schema and one personalization model: a
**Claude Code plugin** that stands up and maintains an executive desk, and a **deskkit**
binary that indexes that desk and repairs convention violations under a byte-exact undo.
Nothing you install carries a name, org, or repo — you personalize once, in
`_knowledge/profile.yaml`, and never by editing a shipped file.

This page gets you productive in one sitting, with **no build toolchain**: an install command, a
guided desk setup, and `deskkit` run from inside your desk. Deeper guides: `plugin-guide.md` (the
four skills) and `pm-guide.md` (the PM work graph). Everything terminal-heavy — building from
source, environment overrides, JSON output — is the developer track, linked where it belongs.

![First sweep: index the desk, then see findings and orphans](../assets/sweep-and-findings.gif)

## 1. Install `deskkit`

`deskkit` installs from the below super-complicated command:

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

## 2. Install the plugin

This repo is its own Claude Code plugin marketplace. In a Claude Code session, add the marketplace
and install the plugin:

```
claude plugin marketplace add hsb3/desk-standard
claude plugin install desk-standard@desk-standard
```

The install copies only the plugin bundle into the plugin cache, so the plugin is self-contained
(its MCP server and schema ship inside it). Inside that session you now have the `desk-setup`,
`conventions-standard`, and `harvest-loop` skills. See `plugin-guide.md` for when to reach for each.

## 3. Stand up your desk

The simplest path is guided: in your Claude Code session, ask for the **`desk-setup`** skill. It
scaffolds a conformant desk, creates your `_knowledge/profile.yaml`, and fills in your identifiers
— handles, repos, board, machines, preferences — resolving every placeholder from your profile so
nothing is hand-typed into a shipped file. **You never edit files under the plugin or the binary to
personalize; personalization lives only in `_knowledge/profile.yaml`.**

Prefer to start from a plain folder without the plugin? From inside that folder, one command writes
the minimal profile for you:

```bash
cd /path/to/your/desk
deskkit init            # writes _knowledge/profile.yaml: desk name from the folder, root "."
```

That is the whole setup — the profile it writes needs no environment variables when you run
`deskkit` from inside the desk. Credentials never go in the profile: it holds identifiers only, and
`secrets_ref.llm_api_key` names the _env var_ that holds your key, never the key itself.

## 4. Your first sweep and patrol

Run `deskkit` from **inside your desk** — the folder whose `_knowledge/profile.yaml` you just
created. It finds the profile by walking up from your working directory, so there is nothing to
configure and no environment variables to export:

```console
$ cd /path/to/your/desk
$ deskkit sweep      # index the desk tree (creates the store on first run)
$ deskkit patrol     # flag convention violations — a dry run; never writes your files
```

`sweep` indexes the tree and `patrol` flags rule violations. Neither needs an LLM key, and **neither
ever writes your desk files** — `patrol` is a pure dry run. To see what was flagged, or to repair
the mechanical findings under a byte-exact undo, the daily loop is in `librarian-guide.md`.

## 5. Meet the TUI

`deskkit chat` opens the full-screen terminal UI — an interactive session over the same tools, and
the home for the PM work-graph views. (Unlike `sweep`/`patrol`, `chat` talks to a model, so it
needs an LLM key set once — the default provider reads `ANTHROPIC_API_KEY`, or point your profile's
`models` + `secrets_ref.llm_api_key` at your key. Details in `../../librarian/README.md`.)

You don't have to memorize anything to find your way around:

- **A tab strip across the top** shows every surface you can reach — `chat | pm context | pm board
| pm item` on a PM-enabled desk — with the active one highlighted. The views are visible on
  sight; you never have to guess they exist.
- **A footer** shows the keys for wherever you are. The essentials:

  | Key      | Does                                                                   |
  | -------- | ---------------------------------------------------------------------- |
  | `ctrl+p` | cycle to the next view (chat → pm context → pm board → pm item → chat) |
  | `esc`    | return to chat from any view                                           |
  | `?`      | open the help overlay listing every key for the current view           |
  | `enter`  | on the **pm board**, open the highlighted item's detail (**pm item**)  |
  | `r`      | refresh the active PM view                                             |
  | `ctrl+c` | quit                                                                   |

- **`?` opens a help overlay** listing every binding in the current view — so the full key set is
  always one keystroke away.

To reach the **PM board** on your first run: launch `deskkit chat`, press `ctrl+p` to step from the
chat into `pm context`, again into `pm board`, and `enter` on an item to open its detail. The tab
strip and footer show you exactly where you are the whole way. The PM views and their keys are
documented in full in `pm-guide.md`. On a desk with the PM module turned off, no PM views mount and
the tab strip is absent — `ctrl+p` tells you so rather than doing nothing.

## What's next

- **The PM work graph** — the phase machine, gates, and every PM surface (CLI, MCP, TUI, plugin):
  `pm-guide.md`.
- **Repair the fixable findings** — the supervised `propose-fix → apply-fix` write path and the
  byte-exact `restore` undo are the daily loop: `librarian-guide.md`.
- **Learn the skills** — greenfield setup, the rule set, the harvest loop, and brownfield adoption:
  `plugin-guide.md`.
- **Build from source, or override the desk/store with environment variables** — the developer
  track: [`development/install-and-build.md`](../development/install-and-build.md).

You now have a personalized desk under a plugin that maintains it and a librarian that patrols it —
reached from one install command and `deskkit` run inside your desk.

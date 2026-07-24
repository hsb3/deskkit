_deskkit: a single Go binary that serves a PocketBase database and stewards a
desk's files under a record-original-first safety boundary._
Status: active

## What this is

deskkit indexes a desk's files, flags convention violations (rules R1–R6), and
can mechanically repair the fixable ones (R1/R2/R3) — always recording the file's original
content before any write, so every change can be reversed byte-exact. It is identity-neutral:
nothing about a person, org, repo, or desk is hardcoded. `DESK_ROOT`, `DESK_NAME`, path
conventions, model id, and provider all come from environment/config.

Full spec: `../docs/development/specs/pocket-librarian-v1-spec.md`.

## Quick start

The binary needs `DESK_ROOT` (which desk) and `DESK_NAME` (a unique store name) — with no
personal defaults, it refuses to run until it can resolve both. There are **two ways** to
supply them; it walks up from the working directory and resolves in the order env > profile:

- **Profile (zero-export).** A `_knowledge/profile.yaml` in the desk supplies both —
  `desk.name` → `DESK_NAME`, and the folder that owns `_knowledge/` → `DESK_ROOT`. Run any
  command from inside the desk with nothing exported. This is the everyday path.

  ```yaml
  # <desk>/_knowledge/profile.yaml
  desk:
    name: my-desk
    root: "."
  ```

  `./deskkit init [dir]` writes exactly this file for you (desk name from the
  folder's basename, `root: "."`) — the fastest way to this profile. It's idempotent
  (`--force` to overwrite), takes `--with-env` to also scaffold a `.env` naming the LLM
  API-key env var, and never creates the store. On an interactive terminal, any
  store-touching command that can't resolve config offers to run it for you
  ("Set up this folder as a desk? [Y/n]"); `--no-input` (or a non-TTY) keeps the prior
  fail-closed error instead of prompting.

- **Environment (override).** Explicit vars always win over the profile — use them for a bare
  folder with no profile, for one-off runs, or when driving the dev build from `librarian/`
  (outside the desk tree, so the profile isn't on the walk-up path):

  ```bash
  export DESK_ROOT=/path/to/your/desk
  export DESK_NAME=my-desk
  ```

The `make` targets run from `librarian/`, so they take the environment form:

```bash
make build          # go build -> ./deskkit
make sweep           # index the desk tree
make patrol          # flag rule violations (dry-run; never writes)
make findings        # or: summary / adoption / orphans / uncollapsed
```

The store self-initializes on first run (ADR 0003) — no separate setup step needed.
`./deskkit migrate up` remains available as an explicit, optional step (idempotent).

`go install github.com/hsb3/desk-standard/librarian/cmd/deskkit@<version>` becomes available once
the repo is public (the module root has no `main` package; the buildable command lives at
`cmd/deskkit`); the local `make install` path (root `Makefile` → build + install `deskkit` to
`~/.local/bin`) is unaffected either way.

`make help` lists every target.

## Where the store lives

By default (no `--dir`), the store resolves to
`$XDG_DATA_HOME/deskkit/<DESK_NAME>/`, falling back to
`~/.local/share/deskkit/<DESK_NAME>/` when `XDG_DATA_HOME` is unset or empty — not a
cwd-relative `pb_data/`. `--dir` is the explicit override. A command that can't resolve
`DESK_NAME` and got no `--dir` errors out rather than silently defaulting to the working
directory. Stores live outside the desk tree on purpose: the librarian must not index its own
database, and SQLite inside an iCloud-synced desk folder is a corruption risk. `DESK_NAME`
must be unique across the estate — it names the store's directory (design in
`../docs/decisions/0002-multi-desk-topology-store-per-desk.md`).

Opening a store also runs a **desk open-guard**: if the store already has rows stamped with a
`desk` different from the configured `DESK_NAME`, the command refuses to run, naming both
values. An empty or brand-new store opens fine.

**Upgrading from `pocket-librarian` (v0.6.0 or earlier)?** Nothing to do: on startup, if
`$XDG_DATA_HOME/deskkit/<DESK_NAME>/` is absent and the old
`$XDG_DATA_HOME/pocket-librarian/<DESK_NAME>/` store exists, the binary moves it to the new
home automatically and logs one line. No desk loses its store across the rename.

If you have an existing store from before this convention (scattered in a scratch or job-tmp
dir), move it to the canonical location before first run at the new default:

```bash
mv <old-store-dir> "${XDG_DATA_HOME:-$HOME/.local/share}/deskkit/$DESK_NAME"
```

Otherwise, a store is a rebuildable cache — a fresh `sweep` at the canonical location
reproduces the same file index, minus that store's revision history.

## Choosing the LLM and setting the API key

Only the `agent` command (and an MCP client driving the tools) needs an LLM. `sweep`,
`patrol`, `propose-fix`, `apply-fix`, `restore`, and `query` are LLM-free — they run with no
provider configured and no API key set.

Provider and model resolve with precedence **env var → profile → default**:

| Setting | Env var | Profile key (`_knowledge/profile.yaml`) | Default |
|---|---|---|---|
| Provider | `LLM_PROVIDER` | `models.provider` | `anthropic` |
| Model | `LLM_MODEL` | `models.model` | `claude-opus-4-8` |

Each provider reads its key from a fixed env var by default — `anthropic` → `ANTHROPIC_API_KEY`,
`openai` → `OPENAI_API_KEY`, `gemini` → `GEMINI_API_KEY`. A profile's `secrets_ref.llm_api_key`
(or the `LLM_API_KEY_ENV` env var) can redirect this: set it to the NAME of the env var that
actually holds the key, and that var is read instead of the provider default. A missing key
fails loud with an actionable message naming the exact var it looked for; nothing silently
falls back.

```bash
export LLM_PROVIDER=anthropic     # or set models.provider in your profile
export ANTHROPIC_API_KEY=sk-...
./deskkit agent "patrol the desk and summarize what you find"
```

`agent` runs the librarian's reasoning loop once over the tool set and exits (one-shot,
manual trigger; step-bounded by `AGENT_MAX_STEP`, default 12).

## Interactive session (`chat`)

`chat` opens a multi-turn conversation over the same agent loop, in the same single
binary, against the desk's store (see "Where the store lives" above). It needs no prior
`migrate up` — like `agent` and `mcp-serve`, it self-initializes the store on first run
(ADR 0003):

```bash
./deskkit chat         # start the multi-turn session
```

That's the whole path from a built binary plus an API key to a live session — one
command. (Design origin: `../docs/decisions/0001-interactive-surface-tui-first.md`,
`../docs/decisions/0004-chat-full-screen-tui.md`.)

Needs an LLM provider and API key exactly like `agent` — see "Choosing the LLM and
setting the API key" above.

On a terminal — stdin and stdout both TTYs — `chat` opens a full-screen TUI: the answer
streams token by token, a finished answer renders as markdown, each tool call collapses
to one faint line (`ctrl+t` expands it), and each turn is prefixed with a colored left
gutter (thick for you, faint for the librarian) and closes with a `model · latency`
footer. `ctrl+g` toggles the full keybind help below the short one. The keys:

| Key | Does |
|---|---|
| `enter` | send |
| `alt+enter` | insert a newline (multi-line input) |
| `esc` | cancel an in-flight turn (badges it `(interrupted)`), or close the resume picker |
| `ctrl+o` | resume a prior conversation |
| `ctrl+n` | start a new conversation |
| `a` (in the picker) | archive / unarchive the selected conversation |
| `A` (in the picker) | show / hide archived conversations |
| `ctrl+t` | toggle tool-step detail |
| `ctrl+y` | copy the last answer's raw markdown (toasts a confirmation) |
| `ctrl+g` | toggle the full keybind help |
| `up` / `down` | at the input's edge rows, walk prompt history (stashes/restores your draft) |
| `pgup` / `pgdn` | scroll the transcript |
| `ctrl+c` | quit |

`ctrl+o` lists recent prior chat conversations — only ones that reached at least one
turn — and resuming one restores the transcript and continues the model with full
context. In the picker you can rename (`r`), delete (`d`, behind a confirm), or archive
(`a`) a conversation. Archiving is a soft, reversible hide — the conversation and its
messages are kept, just dropped from the default list; `A` reveals archived
conversations so you can `a` again to unarchive one. Delete stays a distinct hard
removal. Conversations are not a separate store: they live in the same store as
everything else. When prior conversations exist, `chat` is resume-first: it opens on
that same sessions list at launch, so you land on your history instead of a blank
session — `esc` (or `ctrl+n`) drops into a fresh conversation. A first-run desk with no
prior conversations skips the list and opens straight into a new session.

The color theme resolves once, before the TUI starts: `chat --theme light|dark|auto`
(default `auto`, a single terminal-background probe at startup) or the `LIBRARIAN_THEME`
env var override — precedence is flag > env > auto-detect. `NO_COLOR` strips color from
the whole surface and swaps the spinner for a static "working…" indicator.

Markdown follows the resolved `--theme`/`LIBRARIAN_THEME` (light theme → glamour's light
style, otherwise dark); `GLAMOUR_STYLE` still overrides to another named style. `auto` is
rejected there specifically — an auto glamour style queries the terminal for its
background at render time, which would collide with the TUI's own input handling; the
chat theme itself resolves "auto" safely, once, before the program starts (see above).

When either stdin or stdout is not a terminal (piped input or output), or when
`--plain` is passed, `chat` falls back to the original line REPL instead: the prompt is
`librarian> `, each input line is one turn, and the session replays the recent
conversation so the model sees prior turns. History is bounded to a sliding window of
the most recent turns (a fixed cap, oldest turns dropped, no summarization), so a long
session stays cheap rather than growing unbounded. Exit with `exit`, `quit`, or Ctrl-D.

The session inherits the same gated tool set and boundaries as everything else in this
README: `restore` is never exposed, `apply-fix` only runs when
`LIBRARIAN_AUTONOMOUS_WRITES=true`, and the system prompt is the same data-backed one
`agent` uses. It is desk stewardship, not a general chat assistant.

### Browser session

`serve` also mounts a self-contained session page at `http://127.0.0.1:8090/desk/chat` — a
custom Go route serving a page embedded in the binary via `go:embed`, so there is no
separate frontend build or toolchain needed at runtime (design origin:
`../docs/decisions/0001-interactive-surface-tui-first.md`, option b). One visit is enough:

```bash
./deskkit serve       # then open http://127.0.0.1:8090/desk/chat
```

It drives the same multi-turn session and agent loop as `chat` — the same gated tool set
and the same write boundary: `restore` is never reachable from this surface either, and
`apply-fix` only runs when `LIBRARIAN_AUTONOMOUS_WRITES` is set (checked at execution
time). History is bounded the same way as the REPL — the recent conversation is capped at
40 messages — so a long session stays cheap rather than growing unbounded. Answers stream
to the browser live, token by token, with tool steps shown, over Server-Sent Events, and a
"New conversation" control resets the session.

The route is unauthenticated, exactly like the TUI/REPL — its safety comes from `serve`'s
loopback binding (`127.0.0.1`), not a login. It is a local, on-demand, single-operator
surface, not a hosted service, and it deliberately sits outside the PocketBase superuser
admin login below (that gating would disqualify it as a general session surface). Don't
put `serve` on a public interface.

As a second layer (defense against a page on another site quietly driving your local
session from your browser), the state-changing endpoints — the turn/stream and the reset —
reject any request whose browser `Origin` is not a loopback origin (`127.0.0.1`,
`localhost`, or `[::1]`, any port) with a `403`. Requests with no `Origin` header (curl and
other non-browser tools) are unaffected. This is not authentication — it only closes the
cross-origin browser vector.

## The admin console

The embedded PocketBase serves its full React admin console — browse the `files` index,
patrol `findings`, and `revisions` (recorded originals) directly:

```bash
make gui             # builds, starts serve, opens http://127.0.0.1:8090/_/
```

or by hand: `./deskkit serve` then open `http://127.0.0.1:8090/_/`. If
`PB_SUPERUSER_EMAIL` and `PB_SUPERUSER_PASSWORD` are both set, `serve` auto-creates that
superuser account on first run (idempotent — safe to leave set across restarts). Otherwise,
use the console's first-run screen, or create one non-interactively:

```bash
./deskkit superuser create you@example.com <password>
```

The console is read/write over the database (records, collections) — it does not write
desk files; the `apply-fix` boundary below still holds.

### Reset the system prompt to shipped

The librarian's system prompt is a **re-seeded cache**, not the source of truth — the
canonical copy is the version-controlled embed, and a runtime edit is ephemeral by rule
(ADR 0015, git is truth; `../docs/decisions/0015-prompt-governance.md`). To discard any
GUI/REST edits and restore the shipped prompt: open the admin console, delete the
`librarian.system` row in the `prompts` collection, then run any `deskkit` command (or
restart `serve`). The binary re-seeds the row byte-for-byte from the embed on the next run —
no rebuild and no extra command, just the store's existing seed-if-absent behavior. Durable
customization belongs in `_knowledge/` (the profile), never a DB prompt edit.

## Using it from an agent session (MCP)

`mcp-serve` exposes the tool core over stdio MCP: `sweep`, `patrol`, `propose_fix`, and
`query` always; `apply_fix` only when `LIBRARIAN_AUTONOMOUS_WRITES=true` (default false);
`restore` is deliberately CLI-only. Wire it into a Claude Code project via `.mcp.json`:

```json
{
  "mcpServers": {
    "deskkit": {
      "command": "/path/to/deskkit",
      "args": ["mcp-serve"],
      "env": { "DESK_ROOT": "/path/to/your/desk", "DESK_NAME": "my-desk" }
    }
  }
}
```

## The PM work graph (on by default)

`deskkit` carries a second capability alongside the librarian: the **PM module**, a
document-gated work graph. Work items move through a rigid `queue → work → review → terminal`
phase machine, and a phase advance is **refused until the document that phase requires exists and
validates** against schema v1 — the same schema-v1 engine the librarian uses, reached through a
narrow in-process seam (`docs/decisions/0008-pm-core-modules-architecture.md`).

It is **on by default** (since 1.0; ADR 0008 amendment 2026-07-21). A fresh desk boots with the
`pm` command group, the twelve PM MCP tools, and the five PM collections all present. To run a
desk **librarian-only**, opt out:

```bash
export PM_ENABLED=false        # or set modules.pm.enabled: false in _knowledge/profile.yaml
```

With it off there is no `pm` command, no PM tools, and no PM collections in the store — physical
omission, not inert tables. On (the default), the `pm` command group and the twelve PM MCP tools
appear:

```bash
./deskkit pm create --title "Ship the widget" --type task --court crew
./deskkit pm transition <id> --to work    # refused, naming the missing document, until it validates
./deskkit pm context                      # one-call briefing: active, blocked, stalled, recent
```

`PM_AUTONOMOUS_WRITES` (default `true`) gates whether agents get the write tools over MCP; the
document gate is the real safety regardless. Full surface reference — every `pm` subcommand, the
MCP tools, the TUI views, the `desk-pm` plugin, and the adoption path — is in
[`../docs/usage/pm-guide.md`](../docs/usage/pm-guide.md).

## Triggers — the wake layer under `serve`

`serve` (only `serve` — one-shot CLI commands never enqueue tasks) runs three triggers
that keep the desk patrolled without a human driving each step:

| Trigger | What it does |
|---|---|
| Cron | Hourly job `desk-patrol` (`0 * * * *`) enqueues a `sweep` task, then a `patrol` task. |
| On-record-create hook | A newly indexed row in `files` enqueues a scoped `patrol` task for that file. |
| Claimer | One background goroutine polls the `tasks` queue every `CLAIMER_POLL_INTERVAL` (default `5s`), claims the highest-priority queued task transactionally, and runs it. |

The claimer runs each claimed task by kind: `sweep`, `patrol`, `propose_fix`,
`apply_fix`, and `restore` call the matching tool function directly (no LLM); `query`
and `custom` tasks drive the agent loop. A claimed `apply_fix` still honors the write
gate — with `LIBRARIAN_AUTONOMOUS_WRITES` off it is left `deferred` rather than applied,
for a supervised CLI run later.

These triggers start automatically the moment `serve` is running — no extra command.
Set the poll interval with:

```bash
export CLAIMER_POLL_INTERVAL=10s   # default: 5s
```

## The write path — supervised only

Fixing a finding is split into two steps (decision recorded in the spec §11.2):

1. `propose-fix` — plans the fix and records the file's original content to the `revisions`
   table. **No filesystem write.**
2. `apply-fix` — re-verifies the plan, writes the fix byte-exact, and marks the revision.
   **This is the only tool that mutates desk files.**

`apply-fix` against a **real** desk is intentionally **not** a Makefile target — it is
supervised-only, run by hand:

```bash
./deskkit apply-fix --run <run_id>
```

Any applied fix can be reversed exactly:

```bash
./deskkit restore --by-path <path>
```

Initial and supervised writes are expected to run inside the OS-level sandbox in
`sandbox/` (belt-and-suspenders isolation around the write boundary) — see
`sandbox/README.md`.

## Verifying the build

`make verify` (or `bash verify.sh`) runs the Phase-1 verify gate end to end against a
throwaway scratch desk it creates and destroys itself — it never touches a real desk.
See the header of `verify.sh` for what it checks and the two places it deliberately
substitutes for a spec capability the current build doesn't yet implement.

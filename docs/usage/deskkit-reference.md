_The operator/runtime reference for the `deskkit` binary: quick start, store location, machine-wide
config, the LLM/API key, chat, the browser session, the SPA dev workflow, the admin console, MCP,
the profile module, the PM work graph, triggers, and the supervised write path. For the daily
sweep → patrol → fix → restore loop as a walkthrough, see
[`librarian-guide.md`](librarian-guide.md). Index: [`../README.md`](../README.md)._
Status: active
Audience: **desk owners and developers** — the full command/flag/env-var surface, assuming no build
toolchain beyond a built `deskkit` binary.

# deskkit — reference

## What this is

deskkit indexes a desk's files, flags convention violations (rules R1–R6), and
can mechanically repair the fixable ones (R1/R2/R3) — always recording the file's original
content before any write, so every change can be reversed byte-exact. It is identity-neutral:
nothing about a person, org, repo, or desk is hardcoded. `DESK_ROOT`, `DESK_NAME`, path
conventions, model id, and provider all come from environment/config.

Full spec: [`../development/specs/pocket-librarian-v1-spec.md`](../development/specs/pocket-librarian-v1-spec.md).

## Quick start

`./deskkit --help` groups the whole command menu into six sections (Setup & config, Inspect,
Fix, Work graph, Agent, Admin) instead of one alphabetic list — a fast way to see what the
binary can do before reading further.

The binary needs `DESK_ROOT` (which desk) and `DESK_NAME` (a unique store name) — with no
personal defaults, it refuses to run until it can resolve both. There are **two ways** to
supply them; it walks up from the working directory and resolves in the order env > profile
(with `DESK_NAME` also falling back to the central config's `default_desk`, below):

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
  folder with no profile, for one-off runs, or when driving the dev build from the repo root
  (outside the desk tree, so the profile isn't on the walk-up path):

  ```bash
  export DESK_ROOT=/path/to/your/desk
  export DESK_NAME=my-desk
  ```

The `make` targets run from the repo root, so they take the environment form:

```bash
make build          # go build -> ./deskkit
make sweep           # index the desk tree
make patrol          # flag rule violations (dry-run; never writes)
make findings        # or: summary / adoption / orphans / uncollapsed
```

The store self-initializes on first run (ADR 0003) — no separate setup step needed.
`./deskkit migrate up` remains available as an explicit, optional step (idempotent).

`go install github.com/hsb3/deskkit/cmd/deskkit@latest` installs directly from source (the module
root has no `main` package; the buildable command lives at `cmd/deskkit`); the local `make install`
path (root `Makefile` → build + install `deskkit` to `~/.local/bin`) is unaffected either way.

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
ADR 0002).

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

See which desks this machine already has a store for with:

```bash
./deskkit desks
```

It lists every store dir under the XDG data home, marks the one the current directory resolves
to, and opens no store itself; an empty machine says how to make the first one.

## Machine-wide config

`deskkit config` inspects and edits the central, machine-wide config file — the leg between the
per-desk profile and the built-in default for a handful of fields (`llm.provider`, `llm.model`,
`llm.api_key`, `default_desk`); see "Choosing the LLM and setting the API key" below for what
reads it and in what order. It opens no store.

```bash
./deskkit config show    # every resolved setting, value, and the leg (env/profile/central/default) that won
./deskkit config path    # the file's path, and whether it exists yet
./deskkit config edit    # open it in $VISUAL or $EDITOR, creating it (0600 in a 0700 dir) first
./deskkit config set llm.api_key sk-...   # write one key; secrets are masked back in the confirmation
```

`default_desk` lets `DESK_NAME` resolve on a machine with no profile and no env var set — the
same central leg `LLM_PROVIDER`/`LLM_MODEL` use. It does **not** on its own make an arbitrary
directory a desk: `DESK_ROOT` has no central leg, so with only `default_desk` set the binary
still reports "No desk resolves here". Pair it with a profile (or an exported `DESK_ROOT`) to
name the desk once, machine-wide, and keep the model settings in the same file.

## Choosing the LLM and setting the API key

Only the `agent` command (and an MCP client driving the tools) needs an LLM. `sweep`,
`patrol`, `propose-fix`, `apply-fix`, `restore`, and `query` are LLM-free — they run with no
provider configured and no API key set.

Provider and model resolve with precedence **env var → profile → central config → default**:

| Setting | Env var | Profile key (`_knowledge/profile.yaml`) | Central config key | Default |
|---|---|---|---|---|
| Provider | `LLM_PROVIDER` | `models.provider` | `llm.provider` | `anthropic` |
| Model | `LLM_MODEL` | `models.model` | `llm.model` | `claude-opus-4-8` |

The central config is a machine-wide file at `$XDG_CONFIG_HOME/deskkit/config.yaml` (falling
back to `~/.config/deskkit/config.yaml`), written 0600 in a 0700 dir and shared by every desk
on the machine — below the per-desk profile in precedence, above the built-in default. Manage
it with `deskkit config`, covered in "Machine-wide config" above.

Each provider reads its key from a fixed env var by default — `anthropic` → `ANTHROPIC_API_KEY`,
`openai` → `OPENAI_API_KEY`, `gemini` → `GEMINI_API_KEY`. A profile's `secrets_ref.llm_api_key`
(or the `LLM_API_KEY_ENV` env var) can redirect this: set it to the NAME of the env var that
actually holds the key, and that var is read instead of the provider default. If that env var
is unset, the key falls back to the central config's `llm.api_key` — the ONE place the key may
be stored at rest. A missing key fails loud with an actionable message naming the exact var it
looked for; nothing silently falls back beyond that one central leg.

```bash
export LLM_PROVIDER=anthropic     # or set models.provider in your profile
export ANTHROPIC_API_KEY=sk-...
./deskkit agent "patrol the desk and summarize what you find"
```

Or skip the exports entirely and store the key once, machine-wide:

```bash
./deskkit config set llm.provider anthropic
./deskkit config set llm.api_key sk-...
./deskkit agent "patrol the desk and summarize what you find"   # zero env vars set
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
command. (Design origin: ADR 0001,
ADR 0004.)

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

`serve` mounts an SPA (Svelte + TypeScript, PocketBase JS SDK) at `http://127.0.0.1:8090/` — its
production build is embedded in the binary via `go:embed`, so nothing extra is needed at runtime
once the binary itself was built with `make build`'s Node step (a binary built with a bare
`go build`, skipping that step, still compiles and runs — `/` then serves a small "not built"
placeholder page instead). One visit is enough:

```bash
./deskkit serve       # then open http://127.0.0.1:8090/
```

v1 screens: **chat** (the same multi-turn agent loop as `chat`/the REPL above) and a **read-only
browse** of documents (files), findings, agent runs with their messages, and PM items. Writes
stay on the CLI/MCP tool core — nothing in the SPA calls a write tool. The former standalone
`/desk/chat` page (a single Go route serving embedded HTML) is removed; the URL still works,
because the SPA's index fallback serves the same shell there, like any other client-side route.
`POST /desk/chat/stream` and `POST /desk/chat/reset` are unchanged: same gated tool set and
write boundary as `chat` (`restore` never reachable, `apply-fix` only when
`LIBRARIAN_AUTONOMOUS_WRITES` is set, checked at execution time), same 40-message history cap,
answers still stream to the browser live over Server-Sent Events.

PocketBase's own admin console keeps living at `/_/` (see "The admin console" below); the raw
API is under `/api/`.

**Auth: the SPA always holds a token — how it gets one depends on the RESOLVED bind address,
computed at serve time, never a separate flag** (same derivation as everywhere else in this
README):

- **Loopback bind** (`127.0.0.1`, `localhost`, `::1`, or empty meaning PocketBase's own
  default): `GET /desk/bootstrap` mints a superuser token for the SPA to hold on load, so the
  local operator never sees a login form — a loopback-only route, origin-guarded against a
  cross-site browser page quietly calling it, and not registered at all once the process is in
  public mode. This is the same unauthenticated-by-binding posture the TUI/REPL always had.
- **Public bind**: `/desk/bootstrap` does not exist on this process. The SPA's static shell
  still loads without auth — the same posture as the admin console's own shell below: shell
  loads, data doesn't — but instead of a chat screen it shows a login form, and authenticates
  against the `_superusers` collection through the PocketBase JS SDK. Every domain collection
  keeps its nil (superuser-only) API rules regardless of how the token was minted, so every data
  call behind either shell needs one either way.

Binding `serve` to anything else than loopback — a wildcard address, a bare `:PORT` with no
host, a routable IP, or a hostname that can't be proven loopback — switches the process into
**public mode**: the `/desk/chat/stream` and `/desk/chat/reset` endpoints require a valid auth
token from the `users` or superusers collection (`401` with no token at all; `403` for a valid
token from some other auth collection — PocketBase's own `RequireAuth` distinguishes the two),
and `serve` refuses to even start unless a superuser is guaranteed to exist (see the superuser
env vars below). Both `--http` and `--https` are classified, not just whichever one the
dependency happens to report on the serve event — one exposed listener makes the whole process
public. A hostname that can't be proven loopback classifies as public, fail closed — this is
derived from the bind addresses rather than a `--public` opt-in specifically because an opt-in
flag can be forgotten while the process still binds a wildcard address, which fails open.

As a second layer (defense against a page on another site quietly driving your session from
your browser), those same state-changing endpoints reject a cross-origin browser request with a
`403`. On a loopback bind, the check is the loopback-origin allowlist above (`127.0.0.1`,
`localhost`, or `[::1]`, any port); on a public bind it switches to strict same-origin (the
request's `Origin` must match its own `Host`), because the loopback allowlist would otherwise
reject every real request to a hosted surface. Requests with no `Origin` header (curl and other
non-browser tools) are unaffected in either mode. This check is CSRF defense in depth, not
authentication by itself — on a public bind it runs alongside the route-level auth requirement
above, and it is separate from the CORS header below.

CORS is handled separately from that same-origin check, and the two modes differ: on a public
bind, the embedded store's own default CORS middleware — which otherwise answers every route
with a wildcard `Access-Control-Allow-Origin` — is left unbound, so no
`Access-Control-Allow-Origin` header is emitted at all (this surface is served same-origin and
needs no cross-origin allowlist), unless an explicit `--origins` allowlist is set, in which case
that allowlist is preserved. On a loopback bind, that stock wildcard is left as it always was,
matching the rule that local `serve` behavior stays unchanged. See [`../pattern.md`](../pattern.md)
for the full reusable model behind this section.

### SPA dev workflow

Iterating on the SPA doesn't need a Go rebuild each round-trip:

```bash
./deskkit serve            # backend + store on :8090, one terminal
cd web && npm run dev      # Vite dev server with hot reload, another terminal
```

Vite proxies everything it doesn't own (`/api/`, `/desk/*`, `/_/`) to the running `serve`
instance on `:8090`, so the dev server and the backend share one live store. `make build` (via
the root `Makefile`'s `spa` target) does the real embedded build instead — `npm run build` into
`internal/core/spa/dist/`, picked up by `go:embed` on the next `go build`.

## The admin console

The embedded PocketBase serves its full React admin console — browse the `files` index,
patrol `findings`, and `revisions` (recorded originals) directly:

```bash
make gui             # builds, starts serve, opens http://127.0.0.1:8090/_/
```

or by hand: `./deskkit serve` then open `http://127.0.0.1:8090/_/`. If
`PB_SUPERUSER_EMAIL` and `PB_SUPERUSER_PASSWORD` are both set, `serve` auto-creates that
superuser account on first run (idempotent — an account that already exists under that email
is left alone, its password never re-set, so the bootstrap is safe to leave set across
restarts). Setting only one of the two is a loud fatal error in every mode — a half-configured
pair looks like an operator mistake, not an intentional no-op, so it refuses to start rather
than silently skipping the bootstrap. On a **public** bind (see "Browser session" above),
`serve` additionally refuses to start at all unless one of the two is true: both env vars are
set, or the store already holds a superuser record — a store exposed off-box with no superuser
would otherwise be administered by whoever reaches it first. On a loopback bind, leaving both
unset stays the existing silent no-op. Otherwise, use the console's first-run screen, or create
one non-interactively:

```bash
./deskkit superuser create you@example.com <password>
```

The console is read/write over the database (records, collections) — it does not write
desk files; the `apply-fix` boundary below still holds.

### Reset the system prompt to shipped

The librarian's system prompt is a **re-seeded cache**, not the source of truth — the
canonical copy is the version-controlled embed, and a runtime edit is ephemeral by rule
(ADR 0015, git is truth). To discard any
GUI/REST edits and restore the shipped prompt: open the admin console, delete the
`librarian.system` row in the `prompts` collection, then run any `deskkit` command (or
restart `serve`). The binary re-seeds the row byte-for-byte from the embed on the next run —
no rebuild and no extra command, just the store's existing seed-if-absent behavior. Durable
customization belongs in `_knowledge/` (the profile), never a DB prompt edit.

## Using it from an agent session (MCP)

`mcp-serve` exposes the merged tool core over stdio MCP — every enabled module's tools on one
process. From the librarian module: `sweep`, `patrol`, `propose_fix`, `query`, and
`record_feedback` always; `apply_fix` only when `LIBRARIAN_AUTONOMOUS_WRITES=true` (default
false); `restore` is deliberately CLI-only. Wire it into a project via `.mcp.json`:

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

`MCP_MODULES` narrows one mount to the modules you name (comma-separated: `profile`, `librarian`,
`pm`). Unset, every enabled module's tools are mounted; a name it cannot resolve is a fail-loud
exit, never a silent fallback. The shipped plugin bundle sets `MCP_MODULES=profile,librarian,pm`.

## The personalization surface (`profile` module)

The binary always carries a `profile` module: four read-only tools that answer from the desk's
FILES rather than the store — `profile_get` (resolve a dotted profile key), `profile_validate`
(check the profile against schema v1, including the `x-contract-version` gate), `template_render`
(substitute `{{profile.…}}` / `{{env.…}}` placeholders, fail-loud on an unresolvable one with no
default), and `knowledge_index` (list the `_knowledge/` background files with size metadata, and
inline content up to a byte budget).

It is always enabled and owns no collections, no migrations, and no TUI view — an unpersonalized
desk gets tools that say "this desk declares nothing" rather than tools that vanish. Validation
runs against a `go:embed`ed copy of the schema, kept byte-identical to the repo-root
`schema/profile.schema.yaml` by a test-enforced drift guard.

## The PM work graph (on by default)

`deskkit` carries a second capability alongside the librarian: the **PM module**, a
document-gated work graph. Work items move through a rigid `queue → work → review → terminal`
phase machine, and a phase advance is **refused until the document that phase requires exists and
validates** against schema v1 — the same schema-v1 engine the librarian uses, reached through a
narrow in-process seam (ADR 0008).

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
MCP tools, the TUI views, the `deskkit` plugin, and the adoption path — is in
[`pm-guide.md`](pm-guide.md).

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
`../../sandbox/README.md`.

## Verifying the build

`make verify` (or `bash verify.sh`) runs the Phase-1 verify gate end to end against a
throwaway scratch desk it creates and destroys itself — it never touches a real desk.
See the header of `verify.sh` for what it checks and the two places it deliberately
substitutes for a spec capability the current build doesn't yet implement.

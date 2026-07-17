_pocket-librarian: a single Go binary that serves a PocketBase database and stewards a
desk's files under a record-original-first safety boundary._
Status: active

## What this is

pocket-librarian indexes a desk's files, flags convention violations (rules R1–R6), and
can mechanically repair the fixable ones (R1/R2/R3) — always recording the file's original
content before any write, so every change can be reversed byte-exact. It is identity-neutral:
nothing about a person, org, repo, or desk is hardcoded. `DESK_ROOT`, `DESK_NAME`, path
conventions, model id, and provider all come from environment/config.

Full spec: `../docs/pocket-librarian-v1-spec.md`.

## Quick start

Required environment (no personal defaults — the binary refuses to run without these):

```bash
export DESK_ROOT=/path/to/your/desk
export DESK_NAME=my-desk
```

```bash
make build          # go build -> ./pocket-librarian
./pocket-librarian migrate up   # apply migrations (idempotent)
make sweep           # index the desk tree
make patrol          # flag rule violations (dry-run; never writes)
make findings        # or: summary / adoption / orphans / uncollapsed
```

`make help` lists every target.

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
./pocket-librarian agent "patrol the desk and summarize what you find"
```

`agent` runs the librarian's reasoning loop once over the tool set and exits (one-shot,
manual trigger; step-bounded by `AGENT_MAX_STEP`, default 12).

## Interactive session (`chat`)

`chat` opens a multi-turn REPL over the same agent loop, in the same single binary,
against the local `pb_data`. It requires a prior `migrate up` (or a prior `serve`) — it
opens the DB directly, like `agent` and `mcp-serve`:

```bash
./pocket-librarian migrate up   # once, to create/upgrade the DB
./pocket-librarian chat         # start the multi-turn session
```

That's the whole path from a built binary plus an API key to a live session — two
commands. (Design origin: `../docs/decisions/0001-interactive-surface-tui-first.md`.)

Needs an LLM provider and API key exactly like `agent` — see "Choosing the LLM and
setting the API key" above.

The prompt is `librarian> `; each input line is one turn, and the session replays the
recent conversation so the model sees prior turns. History is bounded to a sliding window
of the most recent turns (a fixed cap, oldest turns dropped, no summarization), so a long
session stays cheap rather than growing unbounded. Exit with `exit`, `quit`, or Ctrl-D.

The session inherits the same gated tool set and boundaries as everything else in this
README: `restore` is never exposed, `apply-fix` only runs when
`LIBRARIAN_AUTONOMOUS_WRITES=true`, and the system prompt is the same data-backed one
`agent` uses. It is desk stewardship, not a general chat assistant — there is no webapp
or browser chat surface built yet; a PocketBase-served browser UI is a recorded, deferred
follow-on (see the ADR above).

## The admin console

The embedded PocketBase serves its full React admin console — browse the `files` index,
patrol `findings`, and `revisions` (recorded originals) directly:

```bash
make gui             # builds, starts serve, opens http://127.0.0.1:8090/_/
```

or by hand: `./pocket-librarian serve` then open `http://127.0.0.1:8090/_/`. If
`PB_SUPERUSER_EMAIL` and `PB_SUPERUSER_PASSWORD` are both set, `serve` auto-creates that
superuser account on first run (idempotent — safe to leave set across restarts). Otherwise,
use the console's first-run screen, or create one non-interactively:

```bash
./pocket-librarian superuser create you@example.com <password>
```

The console is read/write over the database (records, collections) — it does not write
desk files; the `apply-fix` boundary below still holds.

## Using it from an agent session (MCP)

`mcp-serve` exposes the tool core over stdio MCP: `sweep`, `patrol`, `propose_fix`, and
`query` always; `apply_fix` only when `LIBRARIAN_AUTONOMOUS_WRITES=true` (default false);
`restore` is deliberately CLI-only. Wire it into a Claude Code project via `.mcp.json`:

```json
{
  "mcpServers": {
    "pocket-librarian": {
      "command": "/path/to/pocket-librarian",
      "args": ["mcp-serve"],
      "env": { "DESK_ROOT": "/path/to/your/desk", "DESK_NAME": "my-desk" }
    }
  }
}
```

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
./pocket-librarian apply-fix --run <run_id>
```

Any applied fix can be reversed exactly:

```bash
./pocket-librarian restore --by-path <path>
```

Initial and supervised writes are expected to run inside the OS-level sandbox in
`sandbox/` (belt-and-suspenders isolation around the write boundary) — see
`sandbox/README.md`.

## Verifying the build

`make verify` (or `bash verify.sh`) runs the Phase-1 verify gate end to end against a
throwaway scratch desk it creates and destroys itself — it never touches a real desk.
See the header of `verify.sh` for what it checks and the two places it deliberately
substitutes for a spec capability the current build doesn't yet implement.

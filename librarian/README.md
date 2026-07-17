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
manual trigger; step-bounded by `AGENT_MAX_STEP`, default 12). There is no interactive
chat/watch mode yet — richer interaction surfaces are tracked in the repo issues.

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

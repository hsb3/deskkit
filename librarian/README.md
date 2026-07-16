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

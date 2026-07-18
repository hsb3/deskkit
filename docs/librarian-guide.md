_The pocket-librarian daily loop, told as a story: index a desk, flag violations, and repair the mechanical ones under a byte-exact undo._
Status: active

# pocket-librarian — the daily loop

pocket-librarian keeps a desk conformant without you reading every file. It indexes the
tree, flags rule violations, and mechanically repairs the fixable ones — and **every write
it makes is byte-exact reversible.** You get a self-repairing desk with an undo button you
can trust: nothing is changed on disk until you say so, and anything changed can be put
back exactly.

This guide walks the whole loop on a throwaway `example-desk`. Every transcript below is a
real run captured against a scratch desk with a scratch store — never a real desk. For
provider setup, the admin console, the MCP surface, and the sandbox, see the operator
reference in `../librarian/README.md`; this page is the daily-use story.

![The safety loop: propose a fix, apply it, restore it byte-exact](media/propose-apply-restore.gif)

## What you get, at a glance

| Command | What it gives you | Writes desk files? |
|---|---|---|
| `sweep` | A fresh index of the desk tree | No |
| `patrol` | Rule violations (R1–R6) filed as findings | No |
| `query findings` / `summary` | A read-only view of what is wrong | No |
| `propose-fix` | A planned fix + the original recorded for undo | No |
| `apply-fix` | The fix written byte-exact (supervised) | **Yes** |
| `restore` | The exact original put back | **Yes (reverts)** |

The rules `sweep`/`patrol` enforce come from the shared conventions standard — the
`conventions-standard` skill defines them; the librarian mechanizes them (see
`plugin-guide.md`). Mechanical, auto-fixable rules are **R1** (missing frontmatter keys),
**R2** (journal filename not `yyyy-mm-dd-*.md`), and **R3** (doc filed under the wrong
directory for its `type`). Judgment rules (R4–R6) are flagged for a human and never
auto-fixed.

## Where the store lives (and the open-guard)

The librarian is a one-shot CLI; its "instance" is an on-disk SQLite store. With no
`--dir`, the store resolves to `$XDG_DATA_HOME/pocket-librarian/<DESK_NAME>/` (falling back
to `~/.local/share/pocket-librarian/<DESK_NAME>/`) — **outside** the desk tree on purpose,
so the librarian never indexes its own database and a cloud-synced desk folder never
corrupts a live SQLite file. `DESK_NAME` names the store directory and must be unique
across your desks.

![The open-guard refusing a store that belongs to another desk](media/open-guard.gif)

Opening a store runs a **desk open-guard**: if the store already holds rows stamped with a
different `DESK_NAME`, the command refuses and names both values, so a copy-pasted env can
never interleave two desks into one store. The full rationale is
`decisions/0002-multi-desk-topology-store-per-desk.md`.

Every command below uses two required env vars (the binary refuses to run without them) and
a scratch store home so nothing touches a real desk:

```bash
export DESK_ROOT=/path/to/example-desk    # the desk tree to steward
export DESK_NAME=example-desk             # unique store name
export XDG_DATA_HOME=/path/to/scratch/xdg # scratch store home (demo only)
```

## The seeded desk

`example-desk` starts with exactly three fixable violations:

- `tasks/wire-up-ingest.md` — an entity doc missing universal frontmatter keys (**R1**).
- `journal/kickoff-notes.md` — a journal file whose name is not `yyyy-mm-dd-*.md` (**R2**).
- `analyses/backlog-triage.md` — `type: task` but filed under `analyses/` (**R3**).

Their original checksums (used later to prove the undo is exact):

```
f417da0c8fe543166f9d97d0d4df9b89a2288dacaa48d4311d3ab37f742f3f9e  tasks/wire-up-ingest.md
8f4e32335f2e15a985bf3e4dda95057cbc7bb2a1723863f7063c6565f50a446f  journal/kickoff-notes.md
4689c9c6cad1689fc7ce2cb661d328b0ee1e1132cc5058454b88688da7150e25  analyses/backlog-triage.md
```

## 1. Index the tree — `sweep`

`sweep` walks the desk and updates the `files` index. It is idempotent and never touches
desk files.

```console
$ ./pocket-librarian sweep
{
  "total": 4,
  "created": 4,
  "updated": 0,
  "unchanged": 0,
  "soft_deleted": 0
}
```

(No first step needed — `sweep` creates the store on first run. `./pocket-librarian migrate
up` is available as an explicit, optional alternative.)

## 2. Flag violations — `patrol`

`patrol` runs the rules over the swept files and files new findings plus one `patrol_log`
row. It is a **dry run — it never writes desk files.** Note the `run_id`; the write path
keys off it.

![patrol firing real R1 and R2 findings on a seeded desk](media/patrol.gif)

```console
$ ./pocket-librarian patrol
{
  "run_id": "patrol-20260717T171143Z",
  "files_swept": 4,
  "findings_new": 3,
  "by_rule": {
    "R1": 1,
    "R2": 1,
    "R3": 1
  }
}
```

## 3. See what is wrong — `query`

`query findings` groups open findings by rule; `--pretty` renders an aligned table instead
of JSON. Both are read-only.

```console
$ ./pocket-librarian query findings --pretty
findings: 3

R1 (1)
  tasks/wire-up-ingest.md  missing universal frontmatter: created, updated, tags, synopsis

R2 (1)
  journal/kickoff-notes.md  journal filename not yyyy-mm-dd-*.md: kickoff-notes.md

R3 (1)
  analyses/backlog-triage.md  type 'task' but file lives under 'analyses' (expected tasks/)
```

`query summary` gives the counts view (`live_files`, `recent`, `orphans`, `uncollapsed`,
`findings`, `summary`, `adoption` are the available kinds):

```console
$ ./pocket-librarian query summary
{"kind":"summary","files_total":4,"files_by_dir_kind":{"analyses":1,"journal":1,"root":1,"tasks":1},"open_findings_total":3,"open_findings_by_rule":{"R1":1,"R2":1,"R3":1},"open_findings_by_severity":{"mechanical":3}}
```

## 4. Plan the fix and record the originals — `propose-fix`

This is the first half of the write path — and it still **writes no desk files.** It plans
each mechanical fix and records the file's original content to the `revisions` table, so the
undo exists *before* any change does.

```console
$ ./pocket-librarian propose-fix --run patrol-20260717T171143Z
{
  "run_id": "patrol-20260717T171143Z",
  "proposed": [
    { "path": "tasks/wire-up-ingest.md", "rule": "R1", "action": "edit", "new_path": "", "outcome": "recorded" },
    { "path": "journal/kickoff-notes.md", "rule": "R2", "action": "move", "new_path": "journal/2026-07-17-kickoff-notes.md", "outcome": "recorded" },
    { "path": "analyses/backlog-triage.md", "rule": "R3", "action": "move", "new_path": "tasks/backlog-triage.md", "outcome": "recorded" }
  ]
}
```

(`finding_id`/`revision_id` fields elided for width.) The three source files are still
byte-for-byte their originals at this point — the checksums above are unchanged.

## 5. Commit the fix — `apply-fix` (supervised)

`apply-fix` is the **only** command that mutates desk files. Against a real desk it is
deliberately supervised — **not** a Makefile target — and you run it by hand with the
`run_id`. (The MCP `apply_fix` tool and the background claimer add a further gate:
autonomous writes only happen when `LIBRARIAN_AUTONOMOUS_WRITES=true`; the supervised CLI
below is the human-driven path.)

```console
$ ./pocket-librarian apply-fix --run patrol-20260717T171143Z
{
  "run_id": "patrol-20260717T171143Z",
  "outcomes": [
    { "path": "tasks/wire-up-ingest.md", "rule": "R1", "outcome": "applied" },
    { "path": "journal/kickoff-notes.md", "rule": "R2", "outcome": "applied" },
    { "path": "analyses/backlog-triage.md", "rule": "R3", "outcome": "applied" }
  ]
}
```

What each fix did:

- **R1** — the missing universal keys were added from the template (`created`, `updated`,
  `tags: []`, `synopsis: "TODO"`), leaving the body intact.
- **R2** — the journal file was renamed to `journal/2026-07-17-kickoff-notes.md`, content
  **byte-identical** to the original (`8f4e3233…`).
- **R3** — the doc moved to `tasks/backlog-triage.md` (byte-identical, `4689c9c6…`) and a
  `type: pointer` stub was left at the old path pointing to the new one.

The R1 result:

```console
$ cat tasks/wire-up-ingest.md
---
type: task
created: 2026-07-17
updated: 2026-07-17
tags: []
synopsis: "TODO"
---
Wire up the ingest path for the new data surface.
```

Open findings are now zero, and one `adoption_log` row records the run:

```console
$ ./pocket-librarian query adoption
{"kind":"adoption","count":1,"rows":[{"date":"2026-07-17","event":"fix","detail":"run patrol-20260717T171143Z: applied=3"}]}
```

## 6. Undo, byte-exact — `restore`

Every applied fix reverses to the exact recorded original. `restore` is **CLI-only** (never
exposed over MCP) — reversal is a human decision. Restore by the *original* path:

```console
$ ./pocket-librarian restore --by-path tasks/wire-up-ingest.md
{ "revision_id": "…", "path": "tasks/wire-up-ingest.md", "restored": true, "reopened": true }

$ ./pocket-librarian restore --by-path journal/kickoff-notes.md
{ "revision_id": "…", "path": "journal/kickoff-notes.md", "restored": true, "reopened": true }

$ ./pocket-librarian restore --by-path analyses/backlog-triage.md
{ "revision_id": "…", "path": "analyses/backlog-triage.md", "restored": true, "reopened": true }
```

**Proof the reversal is byte-exact** — the checksums equal the originals, and the moved/
renamed copies are gone:

```console
$ shasum -a 256 tasks/wire-up-ingest.md journal/kickoff-notes.md analyses/backlog-triage.md
f417da0c8fe543166f9d97d0d4df9b89a2288dacaa48d4311d3ab37f742f3f9e  tasks/wire-up-ingest.md
8f4e32335f2e15a985bf3e4dda95057cbc7bb2a1723863f7063c6565f50a446f  journal/kickoff-notes.md
4689c9c6cad1689fc7ce2cb661d328b0ee1e1132cc5058454b88688da7150e25  analyses/backlog-triage.md

$ test -e journal/2026-07-17-kickoff-notes.md || echo "renamed copy removed"
renamed copy removed
$ test -e tasks/backlog-triage.md || echo "moved copy removed"
moved copy removed
```

Every hash matches its original above. Restore also **reopens** the findings it undid, so
the next `patrol` sees the same violations again:

```console
$ ./pocket-librarian query summary
{"kind":"summary","files_total":4,"files_by_dir_kind":{"analyses":1,"journal":1,"root":1,"tasks":1},"open_findings_total":3,"open_findings_by_rule":{"R1":1,"R2":1,"R3":1},"open_findings_by_severity":{"mechanical":3}}
```

`restore` fails loud when there is nothing to reverse — it never guesses:

```console
$ ./pocket-librarian restore --by-path does/not/exist.md; echo "exit=$?"
Error: restore: no applied, unrestored revision for path does/not/exist.md
exit=1
```

(The command also prints the usual usage/flags block after the error — trimmed here.)

## Commands that need an LLM (documented, not run here)

`sweep`, `patrol`, `propose-fix`, `apply-fix`, `restore`, and `query` are all LLM-free — no
provider, no API key. Only `agent` (a one-shot reasoning loop over the tools) and `chat` (a
multi-turn conversation over the same loop) call a model. They resolve provider/model with
precedence **env → profile → default** and read the provider's key from a fixed env var
(`anthropic` → `ANTHROPIC_API_KEY`). The two commands below are **documented-not-run in this
guide** because they require a live API key:

```bash
export ANTHROPIC_API_KEY=sk-...
./pocket-librarian agent "patrol the desk and summarize what you find"
./pocket-librarian chat    # full-screen TUI on a terminal, line REPL when piped or --plain
```

Provider selection, key redirection via `secrets_ref.llm_api_key`, history bounds, and the
gated tool set are documented in `../librarian/README.md`.

## Health check — `verify.sh`

`make verify` (or `bash verify.sh`, in `librarian/`) is the operator's end-to-end gate. It
builds the binary, seeds the spec's four fixtures into a throwaway scratch desk under its
own scratch `XDG_DATA_HOME`, and drives the whole chain — sweep → patrol → propose-fix →
apply-fix → restore — asserting the record-original-first boundary, byte-exact restore, and
the store-resolution/open-guard rules, with numbered PASS/FAIL lines. It creates and
destroys its own desk and store, so it is safe to run any time and **never touches a real
desk.** Read the header of `verify.sh` for the two places it deliberately substitutes for a
spec capability the current build does not yet implement.

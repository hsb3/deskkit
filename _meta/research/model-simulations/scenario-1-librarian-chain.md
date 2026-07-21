---
type: analysis
status: active
created: 2026-07-21
updated: 2026-07-21
tags: [model-simulations, librarian]
synopsis: Scenario 1 walkthrough — the librarian write-boundary chain (sweep→patrol→propose_fix→apply_fix→restore) plus the findings-disposition lifecycle, traced against the v1 model with scripted probes.
---

*Scenario 1 of the #126 model simulations (v1 half). Traces the shipped librarian chain end to
end against a throwaway scratch desk, plus the 0.8.0 findings-disposition lifecycle. Derived from
`../2026-07-design-session/workflows.md` §1.1–1.3 and §1.5.*

Status: active (2026-07-21)

## Operator story

A supervisor points the librarian at a desk with two mechanical debts — a task doc missing its
universal frontmatter keys (fires R1) and a task doc mis-filed under `analyses/` (fires R3) —
indexes it, flags the violations, triages one finding, then drafts, applies, and byte-exactly
reverses the fixes.

## Scripted probe

`probes/probe-s1-librarian.sh` (output: `probes/output/out-s1.txt`). Scratch desk under
`mktemp`, hermetic `XDG_DATA_HOME`. Every step below marked **[scripted]** was executed; its
result is the actual probe output.

## Step-by-step trace

| # | Operator action | Surface behavior (observed) | Store entities / fields | v1 verdict | v2 (deferred) |
|---|---|---|---|---|---|
| 1 | `migrate up` (or any first tool call) | Store self-initializes; `migrate up` idempotent | app migrations apply at the `requireConfig` choke point | **OK** [scripted] | — (blocked by #125) |
| 2 | `sweep` | `{"total":3,"created":3}` — walks the tree, upserts one row per file | `files` (create/update/soft-delete); `checksum`, `dir_kind`, `entity_type`, frontmatter fields | **OK** [scripted] | — |
| 3 | `patrol` | `{"by_rule":{"R1":1,"R3":1}}` — R1 (missing FM) + R3 (type/dir mismatch) flagged, one `patrol_log` row | `patrol_findings` (flagged, severity=mechanical), `patrol_log` | **OK** [scripted] | — |
| 4 | `query findings` | `{"by_rule":{"R1":[{"path":..,"detail":..}]}}` — grouped by rule, **{path, detail} only, no finding id** | reads `patrol_findings` where `disposition='open'` | **DEFICIENCY (D1)** [scripted] | — |
| 5 | `findings dispose <id> --as acknowledged` | Requires a record id; the engine resolves via `FindRecordById`. **No `query` kind emits that id** (step 4) → the only source is the admin GUI / raw REST | would write `patrol_findings.disposition` (+ actor/reason/disposed_at) | **DEFICIENCY (D1)** — surface gap; engine itself works (7 tests in `dispose_test.go`) | — |
| 6 | `propose-fix --run <id>` | `[{path:needs-fm.md,outcome:recorded},{path:misfiled.md,outcome:recorded}]`; on-disk R1 file unchanged (`created:` key still absent) | `revisions` (record-original-first: `original_content`/`original_checksum`, `applied=false`) | **OK** [scripted] | — |
| 7 | `apply-fix --run <id>` | `[{needs-fm.md,applied},{misfiled.md,applied}]`; R1 file now carries `created:`; R3 file moved to `tasks/misfiled.md`, old path left a pointer stub | desk filesystem (edit/move/stub); `revisions.applied=true`, `patrol_findings.state='fixed'`, `adoption_log` (`event=fix`) | **OK** [scripted] | — |
| 8 | `restore --by-path tasks/needs-fm.md` | `{"path":..,"restored":true}`; the R1 file's `created:` key is gone again — byte-exact reversal (`created` count back to 0) | verifies `sha256(original_content)==original_checksum` before writing; `revisions.restored=true`, finding reopened to `flagged` | **OK** [scripted] | — |

## Findings from this scenario

### DEFICIENCY D1 — the finding-disposition surface has no id-bearing read path

The disposition lifecycle (`findings dispose <finding-id> --as ...`, added in the 0.8.0 bug
floor, workflows.md §1.5) is deliberately CLI-only and supervised. It takes a **finding record
id**. But the only read surfaces for findings — `query findings` and `query uncollapsed` — reduce
each finding to `findingBrief{path, detail}` (`internal/modules/librarian/tools/query.go:99-102`)
and emit **no id**. By contrast the `feedback` query DOES emit an id
(`feedbackBrief`, `query.go:167-168`). The dispose engine resolves strictly by id
(`FindRecordById`, `dispose.go`). So to dispose a finding from the CLI a supervisor must first
open the PocketBase admin GUI (`make gui`) or hit REST to discover the id — the supervised CLI
workflow cannot be completed from the CLI alone.

Evidence (`out-s1.txt`, step 4): the `query findings` payload is
`{"kind":"findings","count":2,"by_rule":{"R1":[{"path":"tasks/needs-fm.md","detail":"missing
universal frontmatter: ..."}],"R3":[...]}}` — no id field anywhere. The engine itself is sound
(7 passing tests in `dispose_test.go`); this is purely a missing read surface.

Severity **medium** — a shipped 0.8.0 feature is not operable from its own intended surface.
Disposition: **file-as-issue** (draft in `deficiency-report.md`).

## Notes (OK-with-caveat, not gate-bearing)

- **R3 move leaves a pointer stub** at the old path (`apply-fix` writes a stub after the rename).
  Observed and expected (workflows.md §1.3) — recorded so it is not mistaken for a stray copy.
- **A file with no frontmatter at all is not flagged by patrol** — patrol's mechanical rules
  engage typed docs; a wholly un-frontmattered `.md` is caught by `query orphans` instead
  (verified in Scenario 4 / `out-frictions.txt`). Working-as-intended; noted for the baseline
  procedure (run `query orphans` alongside `patrol`).

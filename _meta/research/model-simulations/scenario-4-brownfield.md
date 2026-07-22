---
type: analysis
status: active
created: 2026-07-21
updated: 2026-07-21
tags: [model-simulations, brownfield, adoption]
synopsis: Scenario 4 walkthrough — a messy existing folder through the 9-phase brownfield adoption runbook, traced against the v1 model; judgment phases tabletop, the Phase 8 librarian baseline scripted.
---

*Scenario 4 of the #126 model simulations (v1 half). Follows the shipped 9-phase brownfield
runbook (`plugin/claude-plugin/skills/brownfield-adoption/SKILL.md`). The human-judgment phases
are tabletop; the Phase 8 librarian baseline is scripted against a throwaway messy desk.*

Status: active (2026-07-21)

## Operator story

An operator adopts a real, non-conformant planning folder: locks it (zip incl. `.git/`),
inventories it read-only from a literal disk enumeration, writes a disposition table, gets owner
approval, stages the old content, inits a fresh skeleton, migrates the approved dispositions,
authors + validates the profile, and runs the librarian baseline as the final gate.

## Coverage

- **Tabletop** — phases 1–7 and 9 (lock, inventory, disposition table, the approval GATE, init,
  migrate, author instruments, take stock). These are human-judgment + TS/MCP steps with a
  mandatory user-approval gate; nothing to script and nothing that should be automated.
- **Scripted** — Phase 8, the librarian baseline (`probes/probe-s4-brownfield.sh`, output
  `probes/output/out-s4.txt`): `migrate up` → `sweep` → `patrol` into a **fresh store** over a
  messy desk, recording the patrol run id + store path, then the three-way triage view.

## Step-by-step trace

| # | Phase (K24 track) | Operator action | Surface behavior | v1 verdict | v2 (deferred) |
|---|---|---|---|---|---|
| 1 | Lock | Zip the intact desk incl. `.git/`, verify entry count | procedural; re-runnable archive | **OK** (tabletop) | — (blocked by #125) |
| 2 | Inventory (`untouched→inventoried`) | Disk enumeration (`ls -A`) + `git status --ignored`; status from repo evidence | read-only; never author from memory | **OK** (tabletop) | — |
| 3 | Disposition table as a file | one row per top-level item; mandatory `.gitignore`-flip + ratified-decisions rows | proposal only | **OK** (tabletop) | — |
| 4 | GATE (`inventoried→approved`) | user approves the table + 2–3 residual questions | **no write until approval** | **OK** (tabletop) | — |
| 5 | Init (`approved→reconciled`) | stage old content; init fresh skeleton via K23 | greenfield scaffold + `template_render` | **OK** (tabletop) | — |
| 6 | Migrate | apply approved dispositions row by row; preserve `custom.original_<field>` on retrofit | source stays intact | **OK** (tabletop) | — |
| 7 | Author instruments | write + validate `_knowledge/profile.yaml` against the schema | TS lane | **OK** (tabletop) | — |
| 8a | Baseline: `migrate up` (fresh store) | `rc=0`; store dir created; path recorded | store self-inits | **OK** [scripted] | — |
| 8b | Baseline: `sweep` | `{"total":6,"created":6}` | `files` (6 rows) | **OK** [scripted] | — |
| 8c | Baseline: `patrol` (record run id) | `{"findings_new":4,"by_rule":{"R1":1,"R3":2,"R4":1}}` | `patrol_findings`, `patrol_log`; run id recorded | **OK** [scripted] | — |
| 8d | Triage: mechanical vs judgment | `summary` → `by_severity:{"judgment":1,"mechanical":3}` — R1+R3 auto-fixable, R4 stays flagged for a human | reads `patrol_findings` (`disposition='open'`) | **OK** [scripted] | — |
| 9 | Take stock | "git clean + gitignore covers litter"; leave old folder + zip for the user | procedural | **OK** (tabletop) | — |

## Findings from this scenario

**No new deficiencies.** The Phase 8 baseline behaved exactly as the runbook promises: a messy
desk yields a clean, comparable baseline (run id + store path recorded), and the mechanical /
judgment severity split (`summary.open_findings_by_severity`) directly supports the runbook's
three-way triage (template noise / genuine debt / judgment calls). Observations:

- **`query summary` is the right triage surface** — its `open_findings_by_severity` and
  `open_findings_by_rule` breakdowns map cleanly onto the runbook's mechanical-vs-judgment split.
  Confirmed disposition-aware (counts respect `disposition='open'`), correcting a stale dossier
  claim — see O2 in the deficiency report.
- **Un-frontmattered litter is caught by `query orphans`, not `patrol`** (verified in
  `out-frictions.txt`): a file with no frontmatter is invisible to patrol's typed rules but shows
  up in `query orphans` with `dir_kind:"other"`. The Phase 8 baseline procedure should run
  `query orphans` alongside `patrol` so loose files are not silently missed. Working-as-intended;
  recorded as a procedure note, not a deficiency.
- **`patrol` does not resolve an open finding that merely stops firing** (rule change, hand-fix,
  deletion) — the runbook already warns to re-baseline into a fresh store after upgrading. Not a
  new finding; the fresh-store baseline in Phase 8 is exactly the mitigation.

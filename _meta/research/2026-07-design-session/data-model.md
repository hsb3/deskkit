---
type: analysis
status: draft
created: 2026-07-20
updated: 2026-07-20
tags: [design-session, data-model]
synopsis: Full inventory of the persisted data model across the librarian and PM PocketBase
  lanes plus the shared schema/ contract, so the design session can rule on model changes
  without re-reading source.
---

_Every PocketBase collection, field, enum, and cross-reference-shaped value both product lanes
persist, plus the shared `schema/` contract and where it collides with itself. Grounding dossier
for the 2026-07 design session (see `../README.md` §6)._

> **Correction (2026-07-21):** the #126 v1 model simulations empirically falsified three claims
> below (evidence: `_meta/research/model-simulations/deficiency-report.md`, observations O1–O3):
> **§4** — `items.type` IS validated at create (`create_item` refuses an unknown type); the
> asymmetric `update_item` gap this section originally described was closed by issue #185
> (`update_item` now applies the identical vocabulary check, plus a new refusal for an untyped
> item crossing a document-gated edge) — §4's `items.type` row and §2.1's `type` field note are
> updated in place to reflect this;
> **§5.3** — the disposition-blind-aggregate defect was fixed in the 0.8.0 bug floor;
> **Gaps** — `transitions.event=gate_refused` is live, not dead. Read those sections against
> this callout.

Status: draft (2026-07-20)

All citations are repo-relative `path:line` against commit `51235f6` (post-PR-112 main).

# 1. Collections — librarian lane

Migrations `0001`–`0014`, `librarian/internal/modules/librarian/collections/`. Registered via
`init()` + blank import (unconditional at compile time) — every librarian desk gets all of these.

## 1.1 `files` (0001, altered by 0012)

One row per file under `DESK_ROOT` (`librarian/internal/modules/librarian/collections/0001_files.go:8-9`).

| Field | Type | Notes |
|---|---|---|
| `path` | TextField, Required | Unique key — `idx_files_path` unique index (`0001_files.go:30`) |
| `desk` | TextField | implicit 5000-char cap (no explicit `Max`) |
| `entity_type` | TextField | holds the doc's frontmatter **`type`** value (e.g. `decision`), NOT the schema's `entity_type` enum — see §5 collision |
| `dir_kind` | SelectField | `[decisions, tasks, analyses, journal, meta, memory, root, other, infra]` — `infra` added by migration 0012 (`0001_files.go:17-19`; `librarian/internal/modules/librarian/collections/0012_dir_kind_add_infra.go:15-31`) |
| `status` | TextField | doc frontmatter `status` — implicit 5000 cap |
| `synopsis` | TextField | implicit 5000 cap |
| `origin` | TextField | git origin metadata; implicit 5000 cap |
| `graduated_to` | TextField | see §4 — untyped pointer text; implicit 5000 cap |
| `checksum` | TextField | sha256 hex (see §3); implicit 5000 cap (harmless, hash is fixed-length) |
| `git_last_commit` | TextField | `"<hash>|<date>"` composite string, split at query time (`librarian/internal/modules/librarian/tools/query.go:193-198`) |
| `fm_created`, `fm_updated` | TextField | frontmatter dates as raw strings, not `DateField` |
| `last_seen` | DateField | excluded from sweep's diff comparison (`librarian/internal/modules/librarian/tools/sweep.go:172-174`) |
| `deleted` | BoolField | soft-delete flag (see §3) |

No `RelationField` on `files` itself — it is a relation **target** only (from `patrol_findings.file`,
`0002_patrol_findings.go:18`). None of `files`' TextFields carry an explicit `Max`; every one rides
PocketBase's implicit 5000-char cap (`0001_files.go:14-27`; the repo's own convention comment at
`librarian/internal/modules/librarian/collections/0013_feedback.go:14-16` states the intended
practice — explicit `Max` — which this collection predates and does not follow, and 0011 did not
widen it either).

## 1.2 `patrol_findings` (0002, altered by 0010, 0014)

Dedupe key `(path, rule, checksum)` enforced in **application code**, not a DB index
(`librarian/internal/modules/librarian/collections/0002_patrol_findings.go:8`).

| Field | Type | Notes |
|---|---|---|
| `file` | RelationField → `files`, MaxSelect 1, CascadeDelete false | `0002_patrol_findings.go:18` |
| `rule` | TextField | `R1`–`R6`; implicit 5000 cap |
| `severity` | SelectField | `[mechanical, judgment]` — both wired (mechanical: R1-R3, judgment: R4-R6; `librarian/internal/modules/librarian/tools/patrol.go:113-153`) |
| `detail` | TextField | **content-bearing** (finding description), implicit 5000 cap, NOT widened by 0011 |
| `proposed_fix` | TextField | **content-bearing** (may embed a full replacement), implicit 5000 cap, NOT widened by 0011 |
| `state` | SelectField | `[flagged, dismissed, fixed, resolved]` (`resolved` added by 0010) — **`dismissed` has no setter anywhere in the repo** (see §5) |
| `patrol_run` | TextField | run id string, e.g. `patrol-<UTC-timestamp>`; doubles as a sort key since the collection has no `created` field (`librarian/internal/modules/librarian/tools/patrol.go:258-260`) |
| `checksum` | TextField | the file's checksum **at flag time**, not live — part of the dedupe key |
| `resolved_run` (0010) | TextField | added by `librarian/internal/modules/librarian/collections/0010_patrol_findings_resolved.go:33-35` |
| `disposition` (0014) | SelectField | `[open, acknowledged, triaged, wont_fix]`, added + backfilled to `open` by `librarian/internal/modules/librarian/collections/0014_patrol_findings_disposition.go:32-66` |

`state` and `disposition` are declared **orthogonal** axes (`0014_patrol_findings_disposition.go:9-15`):
`state` tracks lifecycle, `disposition` tracks a supervisor's triage decision, and a disposed finding
stays `flagged` so it survives re-patrol.

## 1.3 `patrol_log` (0003)

| Field | Type | Notes |
|---|---|---|
| `run_id`, `desk` | TextField | implicit 5000 cap |
| `started`, `finished` | DateField | |
| `files_swept`, `findings_new` | NumberField | non-integer-constrained (`OnlyInt` false, the default) — `librarian/internal/modules/librarian/collections/0003_patrol_log.go:8-9` |
| `summary` | TextField | implicit 5000 cap |

No relations, no unique index.

## 1.4 `revisions` (0004, widened by 0011)

The record-original-first ledger (`librarian/internal/modules/librarian/collections/0004_revisions.go:8`).

| Field | Type | Notes |
|---|---|---|
| `path` | TextField, Required | |
| `action` | SelectField | `[edit, move, delete]` |
| `original_content` | TextField | **widened to 50,000,000 chars by migration 0011** (`librarian/internal/modules/librarian/collections/0011_widen_content_fields.go:16,24`) — this is the field the record-original-first safety boundary depends on; the widen migration's own comment explains the pre-0011 5000-char cap could silently break `restore` on any desk file over ~5 KB |
| `original_checksum` | TextField | sha256 hex, verified byte-exact before a `restore` write (`librarian/internal/modules/librarian/tools/restore.go:19,60+`) |
| `new_path` | TextField | populated on `action=move` |
| `finding` | RelationField → `patrol_findings`, MaxSelect 1, CascadeDelete false | `0004_revisions.go:23` |
| `applied`, `restored` | BoolField | guard flags `restore.go` reads before allowing a reversal |
| `run_id` | TextField | |
| `created`, `updated` | AutodateField | |

## 1.5 `adoption_log` (0005)

| Field | Type | Notes |
|---|---|---|
| `date` | DateField | |
| `desk` | TextField | |
| `event` | SelectField | `[patrol, fix, revert, false_positive, friction, note]` (`librarian/internal/modules/librarian/collections/0005_adoption_log.go:15-16`) — **only `fix` is ever written** (see §5) |
| `detail` | TextField | implicit 5000 cap |

No relations.

## 1.6 `agent_runs` (0006) — must migrate before `messages`/`tasks`

| Field | Type | Notes |
|---|---|---|
| `trigger` | SelectField | `[hook, cron, manual, task]` |
| `status` | SelectField | `[running, succeeded, failed, blocked]` — all four confirmed set in `librarian/internal/modules/librarian/agent/persist.go:144,164,167,169,182` and `librarian/internal/modules/librarian/agent/resume.go:149` / `resume_test.go:353` |
| `provider`, `model` | TextField | |
| `run_label` | TextField | **display label only** — `messages.run` relation targets the record id, never `run_label` (`librarian/internal/modules/librarian/collections/0006_agent_runs.go:9-11`) |
| `input_summary`, `output_summary` | TextField | implicit 5000 cap (deliberately bounded summaries per 0011's own comment, `librarian/internal/modules/librarian/collections/0011_widen_content_fields.go:22`) |
| `step_count` | NumberField, OnlyInt, Min 0 | |
| `error` | TextField | implicit 5000 cap |
| `started`, `finished` | DateField | |
| `created` | AutodateField | |
| index | `idx_agent_runs_status` (non-unique) | |

## 1.7 `messages` (0007, widened by 0011)

The ReAct transcript (`librarian/internal/modules/librarian/collections/0007_messages.go:9`).

| Field | Type | Notes |
|---|---|---|
| `run` | RelationField → `agent_runs`, MaxSelect 1, MinSelect 1, CascadeDelete **true** | |
| `seq` | NumberField, OnlyInt, Min 0 | |
| `role` | SelectField | `[system, user, assistant, tool]`, lowercased from the eino framework's Role type (`librarian/internal/modules/librarian/agent/persist.go:79`) |
| `content` | TextField | **widened to 50,000,000 chars by 0011** (full transcript messages incl. embedded tool output/file contents) |
| `tool_calls` | JSONField | |
| `tool_call_id`, `tool_name` | TextField | implicit 5000 cap |
| `created` | AutodateField | |
| indexes | `idx_messages_run_seq` (**unique** on `run, seq` — guards a retried persist from duplicating a loop step), `idx_messages_run` (non-unique) | `0007_messages.go:27-28` |

## 1.8 `tasks` (0008) — the wake queue

| Field | Type | Notes |
|---|---|---|
| `kind` | SelectField | `[sweep, patrol, propose_fix, apply_fix, restore, query, custom]` |
| `payload` | JSONField | |
| `state` | SelectField | `[queued, claimed, done, failed, deferred]` — all five confirmed set in `librarian/internal/modules/librarian/trigger/trigger.go:125,148,156,206,243` |
| `priority` | NumberField, OnlyInt | defaults to zero value 0 |
| `source` | TextField | |
| `claimed_at`, `finished_at` | DateField | |
| `result` | RelationField → `agent_runs`, MaxSelect 1, CascadeDelete false | |
| `created` | AutodateField | |
| index | `idx_tasks_state_priority` (non-unique, on `state, priority`) | |

## 1.9 `prompts` (0009, widened by 0011)

The editable, versioned system-prompt surface (`librarian/internal/modules/librarian/collections/0009_prompts.go:9`). Stable id `pbc_1968329054` for rebuild reproducibility.

| Field | Type | Notes |
|---|---|---|
| `key`, Required | TextField | |
| `name` | TextField | |
| `content` | TextField | **widened to 50,000,000 chars by 0011** — GUI/REST-editable, so it can grow at runtime |
| `version` | NumberField, OnlyInt, Min 0 | |
| `active` | BoolField | |
| `created`, `updated` | AutodateField | |
| index | `idx_prompts_key_active` (non-unique, on `key, active`) | |

The default row is seeded at first run by `prompt.Seed`, not by the migration (`0009_prompts.go:11-12`) —
not independently verified in this pass (see Gaps).

## 1.10 `feedback` (0013)

The librarian's store-native feedback log; DB-only writes, not gated behind
`LIBRARIAN_AUTONOMOUS_WRITES` (`librarian/internal/modules/librarian/collections/0013_feedback.go:9-11`).
**This is the one librarian collection whose content-bearing fields carry an explicit `Max`** —
the migration's own comment states this as the repo convention going forward (`0013_feedback.go:14-16`):

| Field | Type | Notes |
|---|---|---|
| `kind` | SelectField, Required | `[problem, feedback]` |
| `summary` | TextField, Required, **Max 2000** | |
| `detail` | TextField, **Max 50000** | |
| `source` | SelectField, Required | `[agent, user]` |
| `context` | TextField, **Max 2000** | |
| `status` | SelectField, Required | `[open, resolved]` — `open` confirmed set at write (`librarian/internal/modules/librarian/tools/record_feedback.go:59`); `resolved` only found set in a **test** file (`librarian/internal/modules/librarian/tools/record_feedback_test.go:201`), not in production code searched — see Gaps |
| `created` | AutodateField | |
| index | `idx_feedback_status` (non-unique) | |

# 2. Collections — PM lane

Single file, `librarian/internal/modules/pm/collections/collections.go`. **Deliberately NOT**
`init()`-registered — programmatic `Migration` values returned from `Migrations()`, wired only
when the pm module is feature-gated on (`collections.go:1-12`; drift-tested by
`module_test.go TestNoSelfRegisteredMigrations`, not independently re-verified here). Stable
`pbc_*` ids at `collections.go:22-27`. Names: `items, dependencies, transitions, notes, desk_config`
(`collections.go:33`).

## 2.1 `items` (`collections.go:52-88`)

The universal work item. **None of its TextFields carry an explicit `Max`** — every one (below)
rides the implicit 5000-char cap; no PM collection was touched by migration 0011 (that migration
only widens three librarian-lane fields).

| Field | Type | Notes |
|---|---|---|
| `desk` | TextField | `collections.go:54` |
| `title`, Required | TextField | `collections.go:55` |
| `type` | TextField | **validated at both `create_item` and `update_item`** (issue #185 closed the `update_item` gap this row used to describe) — see §4, §5 |
| `phase`, Required | SelectField | `[queue, work, review, terminal]` (`collections.go:57-58`) |
| `blocked` | BoolField | |
| `restore_phase` | SelectField | `[queue, work, review, terminal]` — the §3.2 block/unblock side-state slot, not in the field table the comment references (`collections.go:60-63`) |
| `status_label` | TextField | free-text display label |
| `court` | SelectField | `[owner, desk, crew, vendor, external-session]` |
| `pointer` | TextField | untyped document pointer — see §4 |
| `severity` | SelectField | `[low, medium, high]` |
| `priority` | NumberField, OnlyInt | |
| `claimed_by` | TextField | free-text actor identifier, no relation to any identity/user collection |
| `claim_expires` | DateField | |
| `version` | NumberField, OnlyInt | optimistic-concurrency token, starts at 1 (`engine.go:153`) — see §3 |
| `properties` | JSONField | overflow bag, consulted by `fieldLookup` (`librarian/internal/modules/pm/engine/engine.go:402-406`) |
| `created`, `updated` | AutodateField | |
| `parent`, `root` | RelationField → `items` (self), MaxSelect 1 | added in a **second** `Save` after collection creation, since PocketBase validates relation targets against already-stored collections (`collections.go:49-50,85-87`) |
| index | `idx_items_desk` (non-unique) | |

## 2.2 `dependencies` (`collections.go:94-111`)

Typed edges. `kind` stores **only** the two canonical values — the surface-accepted
`is-blocked-by` is canonicalized to the inverse `blocks` edge by the engine before storage
(`collections.go:90-93`; `librarian/internal/modules/pm/engine/engine.go:626-630`), so it never
appears as a stored value (see §4).

| Field | Type | Notes |
|---|---|---|
| `from`, `to`, Required | RelationField → `items`, MaxSelect 1 | both directions indexed (`idx_dependencies_from`, `idx_dependencies_to`) — the cascade scan queries outgoing edges, the auto-unblock check queries incoming (`collections.go:106-109`) |
| `kind`, Required | SelectField | `[blocks, relates-to]` |
| `unblock_at` | SelectField | `[work, review, terminal]` — `queue` excluded both by omission from Values and by an explicit app-level refusal (`engine.go:633-634`) |
| `cascade` | SelectField | `[auto, manual, auto-reopen, permanent]` |
| `desk` | TextField | |

## 2.3 `transitions` (`collections.go:116-132`)

Append-only audit. Enforced append-only by the engine (never updates/deletes) and hardened by
serve-time hooks refusing `OnRecordUpdate`/`OnRecordDelete`
(`librarian/internal/modules/pm/module.go:109-114`).

| Field | Type | Notes |
|---|---|---|
| `item`, Required | RelationField → `items`, MaxSelect 1 | indexed `idx_transitions_item` |
| `from_phase`, `to_phase` | TextField | **stringly** — not `SelectField`s, unlike `items.phase` |
| `event`, Required | SelectField | `[advance, demote, reopen, block, unblock, claim, release, gate_refused]` — `advance`/`demote`/`reopen` set via `transitionCore` (`engine.go:383`, event var not traced to a literal in this pass), `block`/`unblock`/`claim`/`release` confirmed literal at `engine.go:472,506,562,597,694,784,805`; `gate_refused` not confirmed as a literal in this pass — see Gaps |
| `actor` | TextField | free-text, not a relation |
| `actor_kind` | SelectField | `[human, agent]` — sourced from `Actor.Kind string // "human" \| "agent"` (`engine.go:50`), a plain string field on the `Actor` struct, not itself an enum type |
| `delegation_parent` | TextField | |
| `detail` | TextField | implicit 5000 cap |
| `created` | AutodateField | |

## 2.4 `notes` (`collections.go:135-147`)

Phase-scoped keyed notes.

| Field | Type | Notes |
|---|---|---|
| `item`, Required | RelationField → `items`, MaxSelect 1 | |
| `phase` | TextField | **stringly** — unlike `items.phase` (`SelectField`), a note's `phase` is free text |
| `key` | TextField | the gate-pointer grammar's `note:<key>` addresses this field (see §4) |
| `body` | TextField | **content-bearing, no explicit Max** — rides the implicit 5000-char cap; not touched by 0011 |
| `actor` | TextField | |
| `actor_kind` | SelectField | `[human, agent]` |
| `created` | AutodateField | |

## 2.5 `desk_config` (`collections.go:152-162`)

One row per desk: editable gate rules, status-label vocabulary, claim-TTL override, module-enabled flag.

| Field | Type | Notes |
|---|---|---|
| `desk`, Required | TextField | unique index `idx_desk_config_desk` |
| `rules` | TextField | **the §4.2 gate-rules YAML, human-edited, no explicit Max** — content-bearing and potentially large (many gates × traits); rides the implicit 5000-char cap; validated (not sized) at write time — see §4 |
| `status_labels` | JSONField | |
| `claim_ttl_minutes` | NumberField, OnlyInt | |
| `pm_enabled` | BoolField | |

# 3. Identity & keying

| Concern | Mechanism | Citation |
|---|---|---|
| File keying | Unique index on `path` | `librarian/internal/modules/librarian/collections/0001_files.go:30` |
| Checksum | sha256 hex of raw file bytes | `librarian/internal/modules/librarian/desklib/desklib.go:29-32` |
| Rename | **Soft-delete + fresh insert** — old path's row not seen in the current walk → `deleted=true`; new path never matched `existingByPath` → a brand-new record, no history link between them | `librarian/internal/modules/librarian/tools/sweep.go:49-52` (path-keyed map), `:64-83` (create/update by path), `:86-94` (soft-delete anything unseen) |
| Sweep idempotence | `fileRowDiffers` excludes `path`/`last_seen` from the diff (COMPARE_FIELDS) | `sweep.go:172-188` |
| Findings dedupe | `(path, rule, checksum)` — keyed off the finding's **own stored** checksum at flag time, not the file's current checksum | `librarian/internal/modules/librarian/tools/patrol.go:50-51,60-69,246-252` |
| Findings re-fire vs resolve | `ruleKey{path, rule}` (checksum-independent) tracks "did this rule fire this run"; an open finding whose `ruleKey` didn't fire this run transitions to `resolved` | `patrol.go:75,161-174,236` |
| Disposition inheritance | A fresh finding on the same `(file, rule, checksum)` inherits the most recent prior non-`open` disposition (fail-open to `'open'` on lookup error), ordered `-patrol_run,-id` | `patrol.go:254-273` |
| PM item keying | Auto-generated PocketBase id, or a caller-pinned `ID` for the deterministic import path (rebuild reproducibility) | `librarian/internal/modules/pm/engine/engine.go:111-116,141-143` |
| PM version token | Optimistic-concurrency int, starts at 1 on create, `bump()` increments on every mutation, `checkVersion` refuses a stale write | `engine.go:153` (create), `:194-198` (`checkVersion`), `:218` (`bump`); enforced at every mutating call: `Block` (`:456`), `Unblock` (`:493`), `Claim` (`:545`), `Release` (`:584`), `SetStatusLabel` (`:827`) |
| Record-original-first | `revisions.original_checksum` verified against `sha256(original_content)` **before** any restore write | `librarian/internal/modules/librarian/tools/restore.go:19,60+` (verification step referenced in the function's own doc comment) |

# 4. Typed vs stringly — cross-reference-shaped fields

| Field | Declared type | Actual value grammar | Validated at write? |
|---|---|---|---|
| `files.graduated_to` | TextField | Opaque pointer text (`wb#N`, `#N`, bare number, or URL), populated **only** from an explicit marker: frontmatter `graduated_to:` key, or a canonical inline `graduated to: <ref>` line (regex-anchored at line start) | No — any string accepted; the marker-vs-not distinction is a **read-time heuristic** in `sweep.go`, not a schema constraint | `librarian/internal/modules/librarian/tools/sweep.go:349-376` (marker regex `:359`) |
| `items.pointer` | TextField | A desk-relative file path, optionally suffixed with an advisory `§ heading` section anchor (`"notes.md § Decisions"`); a `#heading`-style suffix is NOT stripped and fails resolution with an actionable hint | Not at write time (no PM-side check); resolved/validated **at gate-evaluation time** by the librarian's `Verdict()` — existence + frontmatter-type + status, never the heading | Declared: `librarian/internal/modules/pm/collections/collections.go:67`. Resolution: `librarian/internal/modules/librarian/module.go:104-175` (`Verdict`), `:203-214` (`sectionFilePart`) |
| `items.type` | TextField (no `Values`) | Intended to be a schema-v1/kit doctype string (e.g. `decision`, `feature-spec`) | **Yes, at both write paths** (issue #185 closed the asymmetry this row used to describe) — `CreateItem` refuses a non-empty, unrecognized type via `schema.Vocab().KnownType`, and `UpdateItem` now applies the identical check; an item left untyped (still legal — a deliberate ADR 0012 scope call) is additionally refused on any edge the desk's gate config binds for at least one known type, closing the "typo'd/absent type advances ungated" gap this row used to name | `librarian/internal/modules/pm/engine/engine.go` `CreateItem`'s vocab check + the `edgeGatedForAnyType`-gated refusal in `transitionCore`; `librarian/internal/modules/pm/engine/queries.go` `UpdateItem`'s matching vocab check |
| `desk_config.rules` | TextField (no explicit `Max`) | A YAML document: `schema_version`, `gates: {type -> transition -> documents}`, `traits: [...]` | **Yes** — `gates.ParseRules` validates `schema_version==1`, every gate's item-type against `schema.Vocab().KnownType`, every transition key against `statemachine.ParseEdgeKey`, every doc requirement's type/status/pointer grammar; bound to `OnRecordCreate`/`OnRecordUpdate` hooks so an invalid config is rejected, never saved | `librarian/internal/modules/pm/gates/gates.go:57-123`; hooks at `librarian/internal/modules/pm/module.go:97-108,160-173` |
| Gate `DocRequirement.Pointer` | Go string field (YAML `pointer:`) | `""`/`"item"` (item's own pointer) or `"note:<key>"` (a note's body) — closed 2-value grammar | Yes, at `ParseRules` time (`validateDocRequirement`) | `gates.go:25,109-123` |
| `dependencies.kind` | SelectField `[blocks, relates-to]` | Surface accepts a third value, `is-blocked-by`, but the engine canonicalizes it to the inverse `blocks` edge **before** the DB write — it is never a stored value | Yes — both by the engine's own switch (`refuse` on anything else) and PocketBase's `SelectField` enum | `pm/collections/collections.go:90-93,99-100`; `pm/engine/engine.go:626-644` |
| `notes.phase` | TextField | Intended to mirror `statemachine.Phase` (`queue/work/review/terminal`) | **No** — unlike `items.phase` (a `SelectField`), `notes.phase` is free text with no enum tie | `pm/collections/collections.go:139` vs `:57-58` |
| `transitions.from_phase`/`to_phase` | TextField | Phase names, written from `string(from)`/`string(to)` at transition time | No enum on the column (audit trail is intentionally loose; the engine is the source of truth for legality) | `pm/collections/collections.go:120-121`; `pm/engine/engine.go:383` |
| `items.claimed_by` / `transitions.actor` / `notes.actor` | TextField | Free-text actor identifier | No — no relation to any user/identity collection exists in either lane | `pm/collections/collections.go:71,124,142` |

# 5. Enum hygiene

## 5.1 Declared enum values with no code path that sets them

| Collection.field | Declared values | Unwired value(s) | Evidence |
|---|---|---|---|
| `patrol_findings.state` | `flagged, dismissed, fixed, resolved` | **`dismissed`** | Enum declared `librarian/internal/modules/librarian/collections/0002_patrol_findings.go:23`. Every `.Set("state", ...)` call in the repo sets one of `flagged` (`patrol.go:93`, `restore.go:106`, `0010_...go:53`), `fixed` (`apply_fix.go:223`), or `resolved` (`patrol.go:168`) — a repo-wide grep for `.Set("state"` (Go files) returns no `"dismissed"` literal anywhere, including tests. The disposition sub-machine added in 0014 (`acknowledged/triaged/wont_fix`) appears to have superseded a plain-dismiss path without removing the enum value. |
| `adoption_log.event` | `patrol, fix, revert, false_positive, friction, note` | **`patrol, revert, false_positive, friction, note`** (5 of 6) | Enum declared `librarian/internal/modules/librarian/collections/0005_adoption_log.go:15-16`. The only production writer is `recordAdoptionLog`, which always writes the literal `"fix"` (`librarian/internal/modules/librarian/tools/apply_fix.go:270`). No other call site writes to `adoption_log` at all (confirmed by a repo-wide grep for the collection name — the only writer is `apply_fix.go`). |
| `feedback.status` | `open, resolved` | `resolved` — **no confirmed production setter** | `open` is set at creation (`librarian/internal/modules/librarian/tools/record_feedback.go:59`). The only place `"resolved"` is set is a test file, `librarian/internal/modules/librarian/tools/record_feedback_test.go:201` — no production tool/CLI path setting `status=resolved` was found in this pass; not exhaustively ruled out (see Gaps). |

## 5.2 The `entity_type` naming collision (files vs schema)

`files.entity_type` (`librarian/internal/modules/librarian/collections/0001_files.go:16`) is
populated **from the document's frontmatter `type` key** — `row.EntityType = fmStr(fm, "type")`
(`librarian/internal/modules/librarian/tools/sweep.go:248`) — so its live values are doctype
strings like `decision`, `task`, `analysis`, `journal`, `entity`, etc.

`schema/doctypes.yaml` separately declares an **unrelated** enum also named `entity_type`:
`[person, company, technology, product, service]` (`schema/doctypes.yaml:29`), bound via
`fieldEnums.entity_type: entity_type` (`schema/doctypes.yaml:32-35`) to a **different** frontmatter
key — the `entity_type` field that appears only on documents whose `type` is `entity`
(`schema/doctypes.yaml:69`: `entity: { status: reference, required: [entity_type] }`).

Same name, disjoint value spaces, disjoint source frontmatter keys, no shared code path — the
`files` DB column and the schema enum are both called `entity_type` by coincidence of naming, not
by any shared contract. Neither `librarian/internal/core/schema/doctypes.go`'s
`ValidateFrontmatter` nor any code in `plugin/core/*.ts` currently enforces the `fieldEnums`/`formats`
checks at all — confirmed by the engine's own scope comment (`librarian/internal/core/schema/doctypes.go:7-11`:
"the `formats:`/`enums:` value-format checks are NOT enforced by this engine yet") and by a
zero-match grep for `entity_type` under `plugin/core/*.ts`.

## 5.3 Disposition-blind aggregate queries (C4 residue)

`query findings` defaults to `disposition = 'open'` (`librarian/internal/modules/librarian/tools/query.go:401-412`),
but `query uncollapsed` (`query.go:381-392`) and `query summary` (`query.go:414-424`) both call
`openFindingRows` with **no** disposition filter — they count every `state='flagged'` row
regardless of disposition, so a supervisor-acknowledged/triaged/wont_fix finding still inflates
`summary`'s `open_findings_total` and rule/severity breakdowns, and still appears in `uncollapsed`.
"Open findings" therefore means two different things depending which query kind is asked.

## 5.4 Enums confirmed fully wired (for contrast)

`files.dir_kind` (all 9 values reachable via `dirKindFor`, `sweep.go:285-321`),
`patrol_findings.severity` (`mechanical`/`judgment`, both wired, `patrol.go:113-153`),
`agent_runs.status` (all 4, `agent/persist.go:144,164,167,169`, `agent/resume.go:149`),
`tasks.state` (all 5, `librarian/internal/modules/librarian/trigger/trigger.go:125,148,156,206,243`),
`disposition` (all 4 reachable through `DisposeFinding`'s validated input,
`librarian/internal/modules/librarian/tools/dispose.go:23-28,38-58`).

# 6. Shared schema contract (`schema/`)

| Artifact | Consumed by | Drift guard |
|---|---|---|
| `schema/doctypes.yaml` | `librarian/internal/core/schema/doctypes.go` via `//go:embed doctypes.yaml` reading a **vendored copy** at `librarian/internal/core/schema/doctypes.yaml` (byte-identical required) | `TestDoctypesEmbeddedCopy_MatchesRepoRoot`, `librarian/internal/core/schema/doctypes_test.go:10-29` — confirmed byte-identical by direct diff in this pass |
| `schema/profile.schema.yaml` | `plugin/core/schema.ts` (walks up from cwd/module dir looking for `schema/profile.schema.yaml`, `plugin/core/schema.ts:33-58`); compiled with ajv draft-2020-12 (`schema.ts:76-88`) | Copied into `plugin/claude-plugin/schema/profile.schema.yaml` by `plugin/package.json:17` (`cp ../schema/profile.schema.yaml claude-plugin/schema/profile.schema.yaml`); CI fails on any diff in `claude-plugin/` — `.github/workflows/ci.yml:85`, `.github/workflows/release.yml:80` |

**`schema/doctypes.yaml` contents** (`schema/doctypes.yaml:17-92`):
- `universal: [type, status, created, updated, tags]` — required on every doc; `status` optional
  when the type is `lightweight` (`:17`, enforced by `ValidateFrontmatter`,
  `librarian/internal/core/schema/doctypes.go:151-158`)
- `formats:` — `created`/`updated` as `date`, `tags` as `kebab-array` — **declared but not enforced**
  by the Go engine (doctypes.go:7-11) or by the TS lane (zero-match grep)
- `enums:` — `priority [P0..P3]`, `workstream [product, engineering, pm]`,
  `entity_type [person, company, technology, product, service]` (`:26-29`) — same non-enforcement
  caveat
- `fieldEnums:` maps `priority→priority`, `affects_workstreams→workstream`,
  `entity_type→entity_type` (`:32-35`)
- `status:` — six status families (`spec, reference, decision, cadence, project, meta`) each with
  its allowed status list (`:38-44`)
- `types:` — ~28 doctypes, each naming a status family (or `lightweight: true`) plus
  required/optional type-specific fields (`:48-86`)

**`schema/profile.schema.yaml`** — a JSON-Schema-in-YAML (draft 2020-12) validating
`_knowledge/profile.{yaml,json,md-frontmatter}`. Top level `additionalProperties: false`, only
`schema_version` (const `1`) required; nested blocks: `identity`, `repos` (with `default`,
`by_role`, `shorthand.issue_default` — all owner/repo-pattern strings), `board`, `desk` (name/root/
paths), `machines[]` (role enum `primary`/`secondary`), `models` (provider enum
`anthropic`/`openai`/`gemini`), `secrets_ref`, `preferences`, and an open `custom:` escape hatch
(`additionalProperties: true`) — `schema/profile.schema.yaml:1-201`.

**The narrow validation seam** consumed by the PM module (`librarian/internal/core/schema/schema.go`)
is deliberately thin: `ArtifactRequirement{Type, RequiredStatus}` in, `Verdict{Exists,
FrontmatterValid, Status, Satisfied, Missing}` out (`schema.go:12-24`), plus an optional
`FrontmatterReader` for trait predicates (`schema.go:34-42`). The librarian module is the sole
implementer (`librarian/internal/modules/librarian/module.go:104-175,181-201`); with no validator
registered, the gate engine fails closed (`librarian/internal/modules/pm/gates/gates.go:183-185`).

# Gaps & uncertainties

Research budget was capped before every enum/setter could be traced. Left unverified,
deliberately flagged rather than silently omitted:

- **`transitions.event`** — `advance`/`demote`/`reopen` are routed through `transitionCore`
  (`pm/engine/engine.go:383`) via a variable, not confirmed against a literal call site per value;
  `gate_refused` was not found set anywhere in this pass — may be dead, may be set in a code path
  not searched (e.g. the tool layer rather than the engine). Needs a targeted grep before any
  ruling treats it as either live or dead.
- **`tasks.kind`** (`sweep, patrol, propose_fix, apply_fix, restore, query, custom`) — not traced
  setter-by-setter; likely wired via the CLI/tool dispatch layer (`cmd/deskkit`) but not confirmed.
- **`revisions.action`** (`edit, move, delete`) — referenced by comment in `apply_fix.go`
  (`plan.Action == "move"`) but the full set of setters was not enumerated.
- **`agent_runs.trigger`** (`hook, cron, manual, task`) — `persist.go:143` sets it from a variable;
  the four call sites that supply each literal value were not traced.
- **`feedback.status = "resolved"`** — only a test-file setter was found
  (`record_feedback_test.go:201`); whether a production CLI/tool path can set it was not
  confirmed. Flagged in §5.1 as a candidate but with lower confidence than `dismissed`/`adoption_log`.
- **`prompts` default-row seeding** (`prompt.Seed`, referenced at `0009_prompts.go:11-12`) was not
  read in this pass.
- **PM module registration drift guard** (`module_test.go TestNoSelfRegisteredMigrations`) was
  cited from the `collections.go` header comment, not independently re-read.
- **Full desklib.ParseFrontmatter / GitOrigin / GitLastCommit implementations** were not read in
  detail — cited only through their call sites and doc comments.
- **`items.court`/`items.severity`/`messages.role`** are open, PocketBase-enum-validated,
  user/framework-supplied fields; this dossier treats them as "typed but not exhaustively
  exercise-verified" rather than hygiene defects, since (unlike `dismissed`) no code path is
  categorically absent — any caller can supply any declared value. Not independently confirmed
  that every value has been observed in practice.
- This dossier covers the persisted **data model** only. Workflow sequencing (sweep → patrol →
  propose_fix → apply_fix → restore; the PM phase/gate/cascade machine in motion) is the remit of
  the sibling `workflows.md` dossier — not duplicated here beyond what's needed to explain a field.
- `plugin/` collections/store: the plugin lane has no PocketBase store of its own (profile/schema/
  template/index tools only) — confirmed by the absence of any collection-defining code under
  `plugin/`, but not exhaustively re-verified beyond the CLAUDE.md architecture description in
  this pass.

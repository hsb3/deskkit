---
type: analysis
status: draft
created: 2026-07-20
updated: 2026-07-20
tags: [design-session, decision-book, data-model, backlog, D8, 1.0.0]
synopsis: Decision brief D8 — the explicitly pull-only backlog bundle (document identity /
  rename history, the `files.entity_type` naming collision, and TextField implicit-cap
  hygiene). The session rules each sub-item only if a use case pulls it; this brief exists so
  the owner can see what deferring each one costs.
---

_Phase 1 of the design-session prep (`../../README.md` §2). Informs the session; does not
rule. Rulings land as ADRs in `docs/decisions/`. Evidence below is lifted from the Phase-0
dossiers beside this book; dossier claims are hypotheses until re-derived where a ruling
binds._

# D8 — Backlog: document identity, `entity_type` collision, TextField hygiene

> **Scope change (2026-07-20 owner sign-off):** the owner PROMOTED D8 from pull-only backlog
> to a **full session decision** — all three sub-items get ruled in the session like D1–D7.
> The "decide whether to decide" framing below is superseded; the per-sub-item questions,
> options, and blast radii stand as written.
> (Recorded from `_meta/signoff/2026-07-20-decision-book-scope/answers.json`.)

> **Platform-stream interaction (2026-07-20 reboot — see `D0-platform-frame.md`):** the
> platform's document + `revisions` mirror substrate (`../platform/plan.md` R2) gives
> sub-item (a) a natural ally: a frontmatter id is the identity that survives both a store
> rebuild (files-are-truth regime) and the mirror loop; store-side inference and git-based
> identity each fail one of the platform's own constraints (non-git desks are in scope —
> the engine was ruled git-agnostic). Sub-items (b) and (c) are regime-independent hygiene.

## 1. The question

**Does the session rule any of three low-urgency, disjoint gaps now, or leave all three
backlogged until a use case pulls them?** Unlike D1–D7, D8 is staged as "decide *whether* to
decide this now," not "decide this now." Each sub-item is independent of the others and of
every other brief in this book (`D8`, depends on: nothing).

Verbatim from the exec-desk agenda seed ("Backlog (rule only if pulled by a use case)"):

- **(a) Document identity / rename history** — a file rename today is soft-delete + fresh
  insert; no identity survives it, so history is discarded the moment a document is renamed.
- **(b) `files.entity_type` column naming collision** — the `files` collection's
  `entity_type` column and the `schema/doctypes.yaml` `entity_type` enum share a name but
  are unrelated: disjoint value spaces, disjoint source frontmatter keys, no shared code
  path. A rename is migration-cheap.
- **(c) TextField implicit 5000-char caps** — several content-bearing `TextField`s carry no
  explicit `Max` and silently ride PocketBase's 5000-char default — the CLAUDE.md gotcha
  ("Content-bearing text fields must set an explicit `Max`"), enumerated to a field list here.

## 2. Why now

None of the three is now — that is the point of a backlog brief: show what deferring costs,
once, rather than let scope silently drop. None shipped in PR #112; all are listed "Untouched"
in the design-prep README's tracking table (`../../README.md` §3, row C8).

- (a) has no filed issue (`document-model-gaps.md` calls it "Gap E ... (unfiled)"). The
  dossier's own verdict: rule it only if doc history becomes a requirement — none exists yet.
- (b) is a naming collision with a confirmed-cheap fix, not a correctness bug — nothing reads
  `files.entity_type` expecting the schema's `entity_type` enum. It causes confusion, not a
  wrong answer.
- (c) is latent risk, not an active incident — neither dossier reports an observed truncation.
  Included because CLAUDE.md already treats the gotcha as project-wide and this is the first
  place its full blast radius is enumerated.

## 3. Evidence

**(a) Document identity / rename history**

- `data-model.md § 3. Identity & keying` (row "Rename") — soft-delete + fresh insert: old
  path's row unseen in the walk → `deleted=true`; new path never matches `existingByPath` →
  brand-new record, no history link (`librarian/internal/modules/librarian/tools/sweep.go:49-52`
  path-keyed map, `:64-83` create/update by path, `:86-94` soft-delete anything unseen).
- `data-model.md § 3. Identity & keying` (row "File keying") — unique index on `path`
  (`librarian/internal/modules/librarian/collections/0001_files.go:30`).
- `document-model-gaps.md § Gap E — no identity across renames (unfiled)` — `files.path` is
  `Required` (`collections/0001_files.go:14`) with UNIQUE index `idx_files_path`
  (`0001_files.go:30`); sweep upserts by `path` — unseen path soft-deleted (`sweep.go:86-93`),
  new path is a fresh `NewRecord` insert (`sweep.go:66-73`); no frontmatter-id /
  checksum-rename-inference / content-identity mechanism exists.
- `document-model-gaps.md § Gap E — no identity across renames (unfiled)` — "**Verdict E —
  UNADDRESSED (as expected).** Files remain path-keyed; rename discards history. A ruling is
  needed only if doc history becomes a requirement, and any identity primitive must survive
  store-rebuild-from-disk. Backlog (agenda item 6)."
- `data-model.md § 3. Identity & keying` (row "Checksum") — sha256 hex of raw file bytes
  (`librarian/internal/modules/librarian/desklib/desklib.go:29-32`) — the nearest existing
  content-identity primitive, used only for findings dedupe today, not rename inference.

**(b) `files.entity_type` collision**

- data-model.md § 5.2 The `entity_type` naming collision (files vs schema) — `files.entity_type`
  (`collections/0001_files.go:16`) is populated from the frontmatter `type` key —
  `row.EntityType = fmStr(fm, "type")` (`tools/sweep.go:248`); live values are doctype strings
  (`decision`, `task`, `analysis`, `journal`, `entity`, etc.).
- data-model.md § 5.2 The `entity_type` naming collision (files vs schema) — `schema/doctypes.yaml`
  separately declares an unrelated `entity_type` enum, `[person, company, technology, product,
  service]` (`schema/doctypes.yaml:29`), bound via `fieldEnums.entity_type: entity_type`
  (`:32-35`) to the distinct `entity_type` frontmatter field on `entity`-typed docs (`:69`).
- data-model.md § 5.2 The `entity_type` naming collision (files vs schema) — same name,
  disjoint value spaces, disjoint source keys, no shared code path; neither
  `librarian/internal/core/schema/doctypes.go`'s `ValidateFrontmatter` nor `plugin/core/*.ts`
  enforces the `fieldEnums`/`formats` checks at all.
- `document-model-gaps.md § Smaller hygiene findings` — the stored column holds a doctype
  while the schema's `entity_type` is a person/company classification; "Spec-consistent but a
  live confusion source (unchanged by 112)."
- `document-model-gaps.md § Analysis §3 agenda → current status` — row "6. Backlog" —
  "`entity_type` column rename to clear the enum collision" listed under "Left to rule."

**(c) TextField implicit caps**

- `document-model-gaps.md § Smaller hygiene findings` — bare `TextField` (Max==0) silently
  caps at 5000 chars in PocketBase. After 0011/0014, explicit `Max` is set on exactly two sets
  of fields: 0011 widens three content bodies to 50,000,000 —
  `revisions.original_content`/`messages.content`/`prompts.content`
  (`collections/0011_widen_content_fields.go:16,23-27`); the `feedback` collection (0013) sets
  explicit Max on `summary`/`detail`/`context` — 2000/50000/2000 (`0013_feedback.go:21,22,24`).
- `document-model-gaps.md § Smaller hygiene findings` — still riding the implicit 5000-char
  cap, unchanged by 0011/0014: `patrol_findings.detail`+`proposed_fix`
  (`0002_patrol_findings.go:21-22`), `patrol_log.summary` (`0003_patrol_log.go:20`),
  `adoption_log.detail` (`0005_adoption_log.go:17`), `agent_runs`
  `input_summary`/`output_summary`/`error` (`0006_agent_runs.go:20,21,23`).
- `data-model.md § 4. Typed vs stringly` (row `desk_config.rules`) — contrast case: a
  TextField with no explicit `Max` that is nonetheless fully validated at write time by
  `gates.ParseRules` — shows Max-caps and value-validation are orthogonal hygiene concerns.
- CLAUDE.md, "Cross-cutting gotchas" — "Content-bearing text fields must set an explicit
  `Max` — a bare `TextField` silently caps at 5000 chars," the standing gotcha this sub-item
  operationalizes into a concrete field list.

## 4. Options

### (a) Document identity / rename history

- **A0 — Leave unaddressed (status quo).** Rename stays soft-delete + fresh insert. Zero cost
  now; any future "what was this file before rename" consumer has no mechanism.
- **A1 — Frontmatter id.** Human-copyable `id:` key, stable across renames, matched at sweep
  time alongside/instead of `path`. *Hypothesis:* cheapest to reason about, portable across
  git and non-git desks; requires retrofitting existing docs and depends on the key not being
  duplicated/dropped on copy-paste.
- **A2 — Store-side rename inference (checksum heuristic).** Use the existing sha256 checksum
  (`data-model.md § 3`) to infer a rename when checksums match within a sweep. *Hypothesis:*
  no frontmatter change needed, but a heuristic — edit+rename in the same sweep breaks the
  match, and it can't disambiguate a rename from an independent delete+recreate of identical
  content.
- **A3 — Git as the identity source.** Delegate to git's own rename detection (`git log
  --follow`) rather than modeling it in the store. *Hypothesis:* zero schema cost, but only
  works for git-backed desks (the desk model doesn't require git) and doesn't help store-only
  queries (`query`, TUI) that don't shell out to git.

### (b) `files.entity_type` collision

- **B0 — Leave unaddressed.** Zero migration cost; relies on readers noticing the
  dossier/comment rather than the schema itself.
- **B1 — Rename the column** (e.g. `files.doctype`) via a forward migration. *Hypothesis:*
  migration-cheap — nothing depends on the *name* matching the schema enum, only the *value*
  being a doctype string; touches the migration, `sweep.go:248`'s `.Set` call, and query/CLI
  output field names. Must be a new migration under "never edit an applied migration" —
  existing stores pick it up on next migrate, `0001_files.go` stays untouched.
- **B2 — Rename the schema enum instead.** Leave `files.entity_type` as-is; rename
  `schema/doctypes.yaml`'s `entity_type` enum (governs only the `entity` doctype's own
  `entity_type:` field). *Hypothesis:* avoids a DB migration (schema/ is a YAML contract, not
  a migrated column) but changes a required frontmatter key name — blast radius includes any
  existing `entity`-typed document on live desks needing its frontmatter key renamed.

### (c) TextField implicit caps

- **C0 — Leave unaddressed.** The five flagged fields keep the implicit 5000-char cap; matches
  the "no incident yet" evidence state.
- **C1 — One-shot hygiene sweep migration.** Forward migration setting explicit `Max` on the
  flagged fields, sized by role (short summaries vs long detail/proposed-fix bodies), following
  the 0011 (content → 50,000,000) / 0013 (`feedback` → 2000/50000/2000) precedent.
  *Hypothesis:* mechanical, low-risk, no code-path change beyond the migration.
- **C2 — Sweep now, add a recurrence guard.** C1 plus a CI gate (in the spirit of
  `scripts/check-*.mjs`) failing if a new librarian-collection `TextField` lacks an explicit
  `Max`. *Hypothesis:* closes the class, not just current instances, but is new scope — needs
  a decision on exemptions (short actor/label fields like `claimed_by`?) and mirrors, without
  currently being, the repo's generated-artifact drift-guard pattern.

## 5. Decision criteria

From the constraint walls (`../../README.md` §7):

- **Store rebuildable from disk.** (a): whatever identity primitive is chosen must be
  re-derivable by a fresh sweep from the desk tree alone — a store-only mechanism (not
  reconstructible after a wipe) violates this wall. A1 (on-disk) and A3
  (external-but-derivable) differ structurally from a pure store-side counter/auto-id here.
- **Forward migrations only; never edit an applied migration.** Binds (b) and (c) — any
  column rename or Max widening is a new migration (`0001_files.go` and the
  `0002`/`0003`/`0005`/`0006` migrations stay untouched); existing stores pick up the change
  on next migrate, not retroactively.
- **Record-original-first / byte-exact reversibility.** Not directly implicated by any
  sub-item (none touch the fix/restore write path) — confirm absence of interaction if (a)'s
  identity primitive is ever used to drive a restore-adjacent feature.
- **Identity-neutrality.** (a) and (b) touch shipped collection schemas — any new field name
  or migration must stay generic, no desk-specific vocabulary baked into shipped migrations.
- **No timelines; ruling only if pulled.** D8's own framing — legitimate only tied to a
  concrete pulling use case, not scheduled preemptively.

## 6. Blast radius

**(a)**, by option: A1 (frontmatter id) — `sweep.go` (match by id), a new migration on
`0001_files.go` (adds `id`/`doc_id`), `schema/doctypes.yaml` (new frontmatter key),
`docs/pocket-librarian-v1-spec.md`, scaffold templates that mint new docs. A2 (checksum
inference) — confined to `sweep.go`'s create/update/soft-delete loop (`:49-94`), no
schema/spec change. A3 (git-based) — no schema change; a new CLI/MCP tool shelling out to
git; `docs/pocket-librarian-v1-spec.md` if documented; no help for non-git desks.

**(b)**, by option: B1 (rename column) — new migration under
`librarian/internal/modules/librarian/collections/`, `sweep.go:248`'s `.Set` site, CLI/query
output field naming (`tools/query.go` and friends). B2 (rename schema enum) —
`schema/doctypes.yaml` (`enums:`/`fieldEnums:`), the vendored byte-identical copy
`librarian/internal/core/schema/doctypes.yaml` (`data-model.md § 6`), any existing
`entity`-typed document's frontmatter on live desks (personalization-layer, outside CI's reach).

**(c)**, by option: C1 (sweep migration) — one new migration per affected collection under
`librarian/internal/modules/librarian/collections/`; no application-code change (Max only
affects write-time cap). C2 (sweep + guard) — C1's radius plus a new `scripts/check-*.mjs`-
style script (or Go equivalent) and a CI job entry, plus a scope decision (all TextFields vs
content-bearing only) that would need to be encoded as a rule.

## 7. Out of scope / interactions

- This brief does not decide *whether* any sub-item is pulled — only equips whoever proposes
  pulling one with options and blast radius. D8 depends on nothing and blocks nothing else.
- (a) has a latent connection to **D2 (typed cross-references)**: if document identity becomes
  a first-class primitive, a typed reference could point at an id rather than a path. D2 does
  not currently propose this, nor does this brief — noted as a future interaction only.
- (b) and (c) have no interaction with any other brief in this book.
- The "Lane B" queue (decision-book index, "Adjacent but already queued elsewhere") is not
  duplicated here; if the session pulls a sub-item, it becomes a new issue there or its own
  planning-desk epic, not a rewrite of this brief.

## 8. Uncertainties

- **(a) No filed issue.** `document-model-gaps.md` calls Gap E "unfiled" — no issue number to
  cite; this brief's options are synthesized from the dossier's candidate list (prep README §5
  "Tough spots"), not an issue thread.
- **(a) Checksum-rename-inference limits are this brief's own extrapolation**, not a dossier
  claim — the dossier confirms the checksum mechanism exists and is used for findings dedupe
  (`data-model.md § 3`, row "Checksum") but does not itself evaluate it as rename inference;
  A2's stated limits are reasoning, flagged hypothesis, not re-derived dossier fact.
- **(b) No confirmation of every consumer of the field *name*.** The dossier confirms value
  semantics and the absence of any `schema/`-side enforcement of the name collision, but did
  not grep every CLI/query/TUI output printing the literal field name — a rename's full
  surface (help text, TUI column headers) was not enumerated in Phase 0.
- **(c) Field-by-field Max sizing is unruled.** The dossier lists which fields lack explicit
  Max but proposes no sizes; Option C1's 0011/0013 precedent is this brief's suggestion, not
  evidence.
- **(c) "No incident yet" is absence-of-evidence.** Neither dossier reports having searched
  production logs or live-store data for actual truncation on the five flagged fields — the
  "latent, not active" framing in §2 rests on no incident being *known*, not ruled out.
- Per `data-model.md`'s own Gaps section: **live stores are unverified from either dossier** —
  every citation above is against migration/source definitions on `main` (`51235f6`), not a
  running store's actual column state. If the owner's own desks have already drifted, this
  brief would not know.

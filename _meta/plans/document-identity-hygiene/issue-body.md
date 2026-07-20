> **Tracking:** #TBD, ADR 0017 (2026-07-20 design session). Rename stops discarding history for
> any document that carries a frontmatter `id`; the `files.entity_type` / schema `entity_type`
> naming collision is cleared; and the repo's own "explicit `Max`" convention gets a migration
> that closes the current gap plus a CI guard that stops it recurring.

## Problem

**(a) A file rename is soft-delete + fresh insert; no identity survives it.** `files.path` is
`Required` with a UNIQUE index (`librarian/internal/modules/librarian/collections/0001_files.go:14,30`).
`Sweep` builds an `existingByPath` map keyed only by path
(`librarian/internal/modules/librarian/tools/sweep.go:49-52`); a path it doesn't see this walk is
soft-deleted (`deleted=true`, `sweep.go:86-94`), and a path it has never seen is a brand-new
`core.NewRecord` insert (`sweep.go:64-73`) with no link back to the row that used to hold the
content. There is no frontmatter-id, checksum-rename-inference, or content-identity mechanism
anywhere on this path today — confirmed by reading the whole function (`sweep.go:26-102`).
Any identity primitive introduced to fix this must be re-derivable by a fresh sweep from the desk
tree alone (files-are-truth, ADR 0009) — a store-only counter does not survive a store rebuild.

**(b) `files.entity_type` and the schema's `entity_type` enum are unrelated values sharing one
name.** The DB column `files.entity_type` (declared `collections/0001_files.go:16`) is populated
from the frontmatter **`type`** key — `row.EntityType = fmStr(fm, "type")`
(`librarian/internal/modules/librarian/tools/sweep.go:248`) — so its live values are doctype
strings (`decision`, `task`, `analysis`, `journal`, ...). `schema/doctypes.yaml:29` separately
declares an **unrelated** `entity_type` enum, `[person, company, technology, product, service]`,
bound (`schema/doctypes.yaml:32-35`) to a **different** frontmatter field that appears only on
`type: entity` documents (`schema/doctypes.yaml:69`). Same name, disjoint value spaces, disjoint
source frontmatter keys, no shared code path — confirmed by a zero-match grep for `entity_type`
under `plugin/core/*.ts` and by neither `doctypes.go`'s `ValidateFrontmatter` nor the TS lane
enforcing the schema's `fieldEnums`/`formats` block at all. Every consumer of the DB column name,
enumerated by grep against `main` @ current HEAD (not previously enumerated in full — the D8
decision-book brief itself flagged this as unconfirmed, "did not grep every CLI/query/TUI output
printing the literal field name"):
  - `librarian/internal/modules/librarian/collections/0001_files.go:16` — declaration.
  - `librarian/internal/modules/librarian/tools/sweep.go:112,131,159,172,177,248` — Go struct
    field (`fileRow.EntityType`), `GetString`/`Set`/compare/assign.
  - `librarian/internal/modules/librarian/tools/query.go:83,184,221-226` — the `fileBrief.EntityType`
    JSON output field (`json:"entity_type"`, the literal key `deskkit query live_files`/`recent`/
    `orphans` emit) and the `isOrphan` predicate reading it.
  - `librarian/internal/modules/librarian/tools/propose_fix.go:358` — `planR3` reads
    `fileRec.GetString("entity_type")` to compute the auto-fix destination directory; this is a
    **functional consumer**, not just a comment, and was not named in the ADR's own Affects list.
  - `librarian/internal/modules/librarian/tools/patrol.go:339,354` — `r3Check`'s `entityType`
    parameter and doc comment.
  - `librarian/cmd/deskkit/pretty.go:45` — `colOrder`, the CLI's stable output column header.
  - Comments only (update for consistency, not gate-blocking): `librarian/Makefile:93`,
    `librarian/verify.sh:19`.
  - Tests: `sweep_test.go`, `query_test.go`, `propose_fix_test.go:83,411`, `patrol_test.go:115`,
    `pretty_test.go:73`.
  - Spec prose: `docs/pocket-librarian-v1-spec.md:472,673,859,901,997,1329,1344,1351` (8 lines).

**(c) Several content-bearing `TextField`s ride PocketBase's implicit 5000-char default with no
guard against recurrence.** A bare `core.TextField{}` (no `Max`) validates at PocketBase's
built-in 5000-char cap. After migrations 0011 (three bodies widened to 50,000,000:
`revisions.original_content`, `messages.content`, `prompts.content` —
`collections/0011_widen_content_fields.go:16,23-27`) and 0013 (`feedback.summary/detail/context`
— `collections/0013_feedback.go:21,22,24` — the one collection whose own header comment states
the "explicit `Max`" convention, `0013_feedback.go:14-16`), the following **content-bearing**
fields still carry no `Max` (grepped against every `core.TextField{` in
`librarian/internal/modules/librarian/collections/*.go` on current `main`, confirmed against the
D8 decision-book's own field list):

| Collection.field | Declared at | Role |
| ---------------- | ----------- | ---- |
| `patrol_findings.detail` | `0002_patrol_findings.go:21` | finding description |
| `patrol_findings.proposed_fix` | `0002_patrol_findings.go:22` | may embed a full replacement body |
| `patrol_log.summary` | `0003_patrol_log.go:20` | one-line run summary |
| `adoption_log.detail` | `0005_adoption_log.go:17` | fix/revert log detail |
| `agent_runs.input_summary` | `0006_agent_runs.go:20` | bounded transcript summary |
| `agent_runs.output_summary` | `0006_agent_runs.go:21` | bounded transcript summary |
| `agent_runs.error` | `0006_agent_runs.go:23` | error text |

No incident is reported against any of the seven — this is latent risk (CLAUDE.md's own
cross-cutting gotcha: "Content-bearing text fields must set an explicit `Max`"), not an observed
truncation, closed here before it becomes one.

## Deliverables

- **A — frontmatter `id` as the document-identity primitive.**
  - `librarian/internal/modules/librarian/tools/sweep.go`: read an optional frontmatter `id` key
    (`fmStr(fm, "id")`, same helper `sweep.go:266-271` already uses for every other key) and match
    an existing row by `id` first, falling back to `path`, before deciding create vs. update vs.
    soft-delete (`sweep.go:44-94`). A document carrying `id` that gets renamed on disk must update
    the SAME row (new `path`, same identity) instead of soft-deleting the old path and inserting a
    new record.
  - Two desk documents sharing one frontmatter `id` within a single sweep: the sweep falls back
    to path-matching for the duplicates and files a patrol-visible finding, never silently merging
    their rows.
  - A new forward migration on the `files` collection (proposed basename
    `0015_files_doc_id.go`, following the alter-in-place pattern of
    `collections/0012_dir_kind_add_infra.go`) adding a new field. **The new field cannot be named
    `id`** — PocketBase reserves that name for the record's own system primary key
    (`FieldNameId = "id"`, vendored `github.com/pocketbase/pocketbase@v0.39.6/core/field.go:20`);
    `checkMinFields` requires the collection's real PK field at that exact name
    (`core/collection_validate.go:371-374`), and a second field also named `id` would fail
    `validation_duplicated_field_name` (`core/collection_validate.go:274`). Use `doc_id` for the
    DB column; the frontmatter key itself stays `id` (no such collision exists in YAML).
  - `schema/doctypes.yaml` **and** its vendored byte-identical copy
    `librarian/internal/core/schema/doctypes.yaml` (guarded by
    `TestDoctypesEmbeddedCopy_MatchesRepoRoot`, `librarian/internal/core/schema/doctypes_test.go:10-29`)
    document `id` as a recognized, optional, universal frontmatter key. `ValidateFrontmatter`
    (`librarian/internal/core/schema/doctypes.go:145-171`) only checks presence of `Universal`
    keys and per-type `Required` keys — it never rejects an unrecognized key — so `id` is already
    non-breaking as plain frontmatter with zero code change; this deliverable is about
    DOCUMENTING it as a consumed key, not gating it.
  - `docs/pocket-librarian-v1-spec.md`: add the `id` frontmatter key + sweep-matching behavior to
    the identity/rename prose (the spec's existing identity discussion is around
    `pocket-librarian-v1-spec.md:859-901` `COMPARE_FIELDS`/frontmatter-parsing prose — the new
    prose is additive, not a rewrite of that section).
  - **Note on the guard's own drift-safety net.** Adding a migration bumps the applied-sequence
    ceiling; `(*Mod).SchemaVersion()` (`librarian/internal/modules/librarian/module.go:44`,
    currently `return 14`) and the `basenames` manifest
    (`librarian/internal/modules/librarian/module.go:72-78`) both need the new entry, or
    `GuardDowngrade` (`librarian/internal/core/migrate/migrate.go:122-135`) will refuse to start on
    the NEXT run — it compares the store's stamped sequence against `SchemaVersion()` and errors
    `"refusing to downgrade"` (`migrate.go:128-131`) whenever the store is ahead of what the binary
    claims, even when the ahead-of-claim binary is the very one that just applied the migration.
    `TestMigrations_MatchesCollectionsDir` (`librarian/internal/modules/librarian/module_test.go:19-54`)
    catches a manifest/disk basename mismatch but does **not** check `SchemaVersion()` itself —
    unlike its PM-module sibling `TestMigrations_MatchOwnedCollections`
    (`librarian/internal/modules/pm/module_test.go:69-101`, whose own comment claims to mirror
    "the librarian module's" `SchemaVersion` check, `:100-101`) — so this is a real, currently-open
    gap in the librarian module's own test suite, not a new one introduced by this issue.

- **B — rename the `files.entity_type` column** (recommend `doctype`) via a forward migration.
  - New migration (proposed basename `0016_files_doctype_rename.go`) renaming the field on the
    existing `files` collection — same alter-in-place shape as `0012_dir_kind_add_infra.go`
    (`FindCollectionByNameOrId("files")`, mutate the field, `app.Save`); `0001_files.go` stays
    untouched.
  - Every reader/writer listed in Problem (b) above updates to the new name: the DB column
    literal in `sweep.go` (3 sites) and `propose_fix.go:358`, the JSON tag + Go field name in
    `query.go` (rename `fileBrief.EntityType` -> `fileBrief.Doctype`, `json:"entity_type"` ->
    `json:"doctype"`), `pretty.go:45`'s `colOrder` entry, the `patrol.go` parameter/comment, the
    two comment-only sites (`Makefile:93`, `verify.sh:19`), and every test enumerated above.
  - `docs/pocket-librarian-v1-spec.md`'s 8 `entity_type` references (lines listed in Problem (b))
    update to the new name, since spec prose is cited from code/tests (CLAUDE.md, "docs/ spec +
    ADR paths are load-bearing").
  - DOWN renames `doctype` back to `entity_type`; row data is preserved in both directions
    because the migration mutates the existing field object in place (stable field id - a SQLite
    column rename, not drop+add).

- **C — the cap sweep + a recurrence guard.**
  - One forward migration (proposed basename `0017_content_field_caps.go`) setting an explicit
    `Max` on the seven fields in the Problem (c) table, following `setContentMax`'s pattern
    (`collections/0011_widen_content_fields.go:43-63`: `FindCollectionByNameOrId`, type-assert to
    `*core.TextField`, soft-continue if the field was since renamed/removed, `app.Save`).
    Recommended sizing (unruled by any dossier — a build-time call, sized by role per the 0011/0013
    precedent): `patrol_findings.detail` and `.proposed_fix` at 50000 (may embed a full body, per
    `data-model.md`'s own characterization of `proposed_fix`); `patrol_log.summary`,
    `adoption_log.detail`, and the three `agent_runs` summary/error fields at 2000 (all
    "deliberately bounded summaries" per 0011's own comment, `0011_widen_content_fields.go:22`).
  - A new CI guard (proposed `scripts/check-textfield-max.mjs`, matching `scripts/check-kits.mjs`'s
    dependency-free Node regex-scan house pattern — no real Go parser, no new dependency) scanning
    every `core.TextField{...}` literal under
    `librarian/internal/modules/librarian/collections/*.go` and failing on any that lacks a `Max:`
    key, **except** an explicit exemption list for short/structural fields that are never expected
    to carry a body (`path`, `checksum`, `git_last_commit`, `fm_created`, `fm_updated`, `run_id`,
    `rule`, `patrol_run`, `desk`, `provider`, `model`, `run_label`, `tool_call_id`, `tool_name`,
    `source`, `key`, `name`, `original_checksum`, `new_path` — see plan.md for the full per-field
    table and the two-category exemption reasoning). Wire it into `make check` and
    `.github/workflows/ci.yml` alongside the other `check-*.mjs` invocations.

## Acceptance criteria

- [ ] **A:** a regression test renames a desk file that carries a frontmatter `id` between two
      sweeps; the `files` row for the new path has the SAME record id as the row that used to sit
      at the old path (proven by asserting `record.Id` equality across the two `Sweep` calls), and
      the old path's row is gone (not soft-deleted-and-orphaned) rather than a fresh insert. This
      test must fail against pre-change `sweep.go` (today's soft-delete-plus-insert behavior) and
      pass after the change — the red-able-regression-test bar.
- [ ] **A:** a rename of a document with NO frontmatter `id` still soft-deletes the old path and
      inserts a new row — today's behavior is unchanged for the common (no-id) case; proven by a
      companion test case.
- [ ] **A:** a fresh sweep of a scratch desk from disk alone (no prior store state) reproduces the
      same `doc_id` values for every `id`-carrying document — proven by a rebuild-from-disk test,
      confirming the identity primitive survives a store wipe (ADR 0009 files-are-truth).
- [ ] **A:** two docs sharing an id: sweep does not merge their rows; the duplicate is surfaced as
      a finding (red-able test).
- [ ] **A:** `librarian/internal/core/schema/doctypes.yaml` and `schema/doctypes.yaml` remain
      byte-identical — `TestDoctypesEmbeddedCopy_MatchesRepoRoot` passes.
- [ ] **B:** `deskkit query live_files` (or `recent`) emits `doctype` (not `entity_type`) as the
      JSON key, verified by running the CLI against a scratch desk and reading the output.
- [ ] **B:** a red-able regression test confirms `propose_fix`'s R3 auto-fix planner
      (`planR3`) still resolves the correct destination directory after the rename — this path
      would silently break (always resolve to the zero-value default) if the rename missed this
      site, so the test must fail if `propose_fix.go:358`'s literal is not updated.
- [ ] **B:** `rg -n "entity_type" librarian/ schema/` (excluding the schema's own unrelated
      `entity_type` enum block, `schema/doctypes.yaml:29,32-35,69`, and its vendored copy) returns
      zero hits — the rename is complete, not partial.
- [ ] **C:** the new migration sets the stated `Max` on all seven listed fields; a regression test
      writes a body longer than 5000 chars (but under the new `Max`) to each and confirms it saves
      without truncation — must fail pre-migration (today's implicit 5000 cap truncates/rejects)
      and pass after.
- [ ] **C:** `node scripts/check-textfield-max.mjs` fails when a seeded fixture adds a new
      `core.TextField{Name: "x"}` (no `Max`, not on the exemption list) under
      `librarian/internal/modules/librarian/collections/`, and passes on the post-migration tree
      — the guard's own self-test, mirroring `check-neutrality.mjs --self-test`'s seed-and-detect
      pattern.
- [ ] **`SchemaVersion()`** (`librarian/internal/modules/librarian/module.go:44`) is bumped to
      match the highest basename added across A/B/C, and the `basenames` manifest
      (`module.go:72-78`) lists all new migrations in order — `TestMigrations_MatchesCollectionsDir`
      passes.
- [ ] `make verify` (`librarian/verify.sh`, scratch desk) passes with no changes to the script
      itself required by this issue (the two comment-only `entity_type` mentions in `verify.sh:19`
      are prose, not assertions).
- [ ] `make test` (`go test ./...` under `librarian/`) is green, including every existing test
      enumerated in Problem (b) that references `entity_type` literally.
- [ ] `node scripts/check-neutrality.mjs` (+ `--self-test`) passes — no bare issue references in
      any new Go comment/test string under `librarian/`.

## Dependencies & gates

- **Librarian integration (`make verify`)** — fires: every slice touches `librarian/`.
- **Unit tests (`make test`)** — fires: new/changed Go tests across `sweep_test.go`,
  `query_test.go`, `propose_fix_test.go`, `patrol_test.go`, `pretty_test.go`, plus the two new
  migration files and their own tests.
- **Repo checks (`make check`)** — fires: neutrality lint (always) + the NEW
  `check-textfield-max.mjs` guard this issue adds to the same target. Kit-manifest drift and
  scaffold-frontmatter checks do NOT fire (no `kits/` or scaffold-asset change).
- **Bundle drift guard (`make package` + `git diff --exit-code` on `claude-plugin/`)** — does
  **NOT** fire. `schema/doctypes.yaml` changes for slice A, but `plugin/package.json:17`'s
  `package` script copies only `schema/profile.schema.yaml` into `claude-plugin/schema/` —
  `doctypes.yaml` has no presence anywhere under `plugin/` (confirmed: zero matches for
  `doctypes` under `plugin/`). The gate that DOES fire for the `doctypes.yaml` edit is
  `TestDoctypesEmbeddedCopy_MatchesRepoRoot` (part of `make test`, not `make package`) — both
  copies must be edited together or that Go test fails.
- **DB migration discipline** — fires, three times over: forward migrations only
  (`0001_files.go` stays untouched for both A and B; the alter-in-place pattern of
  `0012_dir_kind_add_infra.go` is the precedent for all three new migrations), and every new
  content `TextField` this issue itself might introduce carries an explicit `Max` (none does —
  slice A's new field is a short id string; deliberately exempted in the same guard this issue
  adds).
- **Version sync / Kits drift** — do NOT fire: no `VERSION` or `kits/` change.
- **Identity neutrality** — fires (always runs); no bare `#N` refs in any new Go comment/test
  under `librarian/` — ADR/issue references stay in `docs/` or this issue body only.
- **Regression-test bar (#82 practice) + adversarial review (PR #112 practice)** — fires on all
  three slices; A and B are data-safety-adjacent (a rename bug or a mis-renamed column reader can
  silently corrupt or misroute desk content) and should get an independent reviewer re-deriving
  the sweep-matching and column-rename-completeness claims before merge, per the PR #112 practice
  this repo already follows for data-safety changes.
- **CHANGELOG** — an entry under `[Unreleased]` is required before a tagged release
  (`check-changelog.mjs` gates the tag, not the PR).
- **Schema-version chain serialization** — the three new migrations (A, B, C) share ONE ordered
  `basenames` slice and ONE `SchemaVersion()` int on the `librarian` module
  (`module.go:44,72-78`). They cannot be authored as fully independent, simultaneously-merging
  PRs without a basename/version-bump collision; see plan.md "Parallelism + landing order" for the
  recommended sequencing.
- **Cross-issue migration-number coordination (fires):** migration numbers here are PROVISIONAL.
  This issue and `findings-lifecycle-completion` both add three librarian migrations on one shared
  chain; the real sequence (basenames + the `module.go` `SchemaVersion()`/manifest twin) is
  assigned at landing time from true current HEAD — whichever lands first takes the next free
  numbers, the second continues from the true next free number; the two issues' migration commits
  must serialize (rule recorded in the `epic-design-session-v1` body).

## Out of scope

- **Store-side checksum-inference or git-based rename identity** (A2/A3 in the D8 decision-book
  brief). Rejected as primary mechanisms by ADR 0017's Decision text — a heuristic (checksum) or a
  git-only mechanism (doesn't help non-git desks or store-only queries) — frontmatter `id` is the
  sole primitive this issue implements.
- **Retrofitting `id` onto existing documents.** Optional and incremental per ADR 0017's
  Consequences — this issue adds the mechanism; it does not seed ids onto any current desk's
  files.
- **The schema v2 element model / two-track schema (ADR 0009, ADR 0018).** The `id` key is a
  schema-v1 doctypes.yaml addition; it is not coupled to the v2 element-list track.
- **The PM lane's own Max-less content fields** (`pm/collections/collections.go`'s
  `notes.body` and `desk_config.rules` — both already flagged by this repo's own data-model
  dossier as content-bearing with no explicit `Max`). ADR 0017's own evidence (the D8 decision
  book, the "smaller hygiene findings" enumeration) names only the seven librarian-lane fields in
  the Problem (c) table; the PM-lane fields are a known, adjacent, NOT-yet-ruled gap, recorded
  here rather than silently swept in. A follow-up issue should decide whether the CI guard's scope
  (and a future cap migration) extends to `librarian/internal/modules/pm/collections/`.
- **`files`'s own remaining Max-less TextFields** (`desk`, `doctype` (post-rename), `status`,
  `synopsis`, `origin`, `graduated_to` — `collections/0001_files.go:14-27`; none were widened by
  0011 either). Not in the D8 evidence's flagged list; left as known debt, not fixed here.
- **Enum-shrink / down-remap concerns.** Neither the rename (B) nor the cap migration (C) shrinks
  an enum or changes a value space — the 0010/0012 down-remap precedent does not apply here.

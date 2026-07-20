# findings-lifecycle-completion - build plan

_Complete the findings lifecycle ADR 0013 opened: retire the dead `dismissed` state, make every
count surface agree on "open findings," record disposition provenance, and shrink `adoption_log` to
the one event a writer produces._

Status: draft
Date: 2026-07-20

## Tracking

- Issue #118 (`issue-body.md` in this folder)
- ADR: `docs/decisions/0013-disposition-completion-adoption-log.md` (Accepted, 2026-07-20)
- Decision brief: `_meta/research/2026-07-design-session/decision-book/D4-disposition-and-adoption-log.md`
- Predecessor: #93 (shipped the disposition axis in PR #112, the 0.8.0 bug floor)
- Parent epic: `epic-design-session-v1` (ADR 0010-0017 v1 track)

## The problem, grounded in source

All `path:line` against current `main` (post-PR-113 main `e5aee59`, carried on branch HEAD
`ee4739d`).

#93 shipped `disposition` (`open`/`acknowledged`/`triaged`/`wont_fix`) on `patrol_findings`,
orthogonal to `state`, backfilled to `open` by migration `0014`
(`collections/0014_patrol_findings_disposition.go:32-66`), with re-patrol inheritance on
`(file, rule, checksum)` (`tools/patrol.go:254-273`). Four residuals remain, each independently
re-derived here:

1. **`state.dismissed` is dead.** `state` is declared `[flagged, dismissed, fixed]` at
   `collections/0002_patrol_findings.go:23`, with `resolved` appended by `0010`
   (`collections/0010_patrol_findings_resolved.go:29`). Every `.Set("state", ...)` on
   `patrol_findings` in the tree writes `flagged`/`resolved`/`fixed` -
   `tools/patrol.go:93`, `tools/patrol.go:168`, `tools/apply_fix.go:223`, `tools/restore.go:106`,
   `collections/0010_...:53` (the down remap). No `"dismissed"` state literal exists anywhere,
   tests included (verified by repo-wide grep). The 0014 disposition machine superseded a
   plain-dismiss path without removing the enum value.

2. **Counts disagree on "open."** `openFindingRows(app, extraFilter)` (`tools/query.go:313-336`)
   filters `state = 'flagged'` plus the caller's extra clause. `queryFindings`
   (`tools/query.go:401-412`) passes `disposition = 'open'` by default; but `queryUncollapsed`
   (`tools/query.go:381-392`) passes only `rule = 'R5'` and `querySummary`
   (`tools/query.go:414-424`) passes `""` - neither filters disposition, so a disposed finding still
   inflates `open_findings_total` and still shows uncollapsed. The comment at `tools/query.go:398-399`
   states this old behavior explicitly. MCP: the `query` tool handler calls the same `tools.Query`
   (the MCP server is a generic registration loop over `toolcore.ExposedTools`,
   `internal/core/mcp/server.go:69-81`), so MCP inherits the divergence and the fix. TUI: the chat
   transcript IS the TUI (`module.go:62-63`); the TUI has no independent count source (no count
   query exists under `internal/modules/librarian/tui/` - grep-verified); counts come from the
   shared `query` tool. **Confirmed: the TUI has no independent count source** (closes D4
   criterion 1's TUI uncertainty).

3. **No disposition provenance.** `DisposeFinding` (`tools/dispose.go:38-59`) sets `disposition` and
   nothing else (`:49`). No actor, reason, or timestamp.

4. **`adoption_log.event` is 5/6 writerless.** Enum `[patrol, fix, revert, false_positive, friction,
   note]` (`collections/0005_adoption_log.go:15-16`); the only writer is `recordAdoptionLog`, always
   `"fix"` (`tools/apply_fix.go:270`). No other write site touches the collection (verified by
   repo-wide grep for `adoption_log`). Reader: `queryAdoption` (`tools/query.go:449-463`), surface
   `query adoption` (`cmd/deskkit/main.go:778`). Desk-collision role: third in
   `deskCarryingCollections` (`internal/core/store/deskguard.go:13`). Both keep it alive; the ruling
   shrinks the enum, it does NOT kill the collection.

The migration twin: the librarian module declares its schema version in TWO places that must move
together - `SchemaVersion()` (`module.go:44`, currently `return 14`) and the `Migrations()` basename
manifest (`module.go:71-84`, currently ending at `0014_patrol_findings_disposition`). The manifest is
guarded by `TestMigrations_MatchesCollectionsDir` (`module_test.go:19-54`, fails if a
`collections/NNNN_*.go` exists on disk but is absent from the manifest, or vice versa). `SchemaVersion()`
is consumed by `GuardDowngrade` (`internal/core/migrate/migrate.go:119-131`): after migrations run,
`StampModules` stamps the store's `module_schema_versions` row with the highest APPLIED sequence read
from PocketBase's `_migrations` ledger (`migrate.go:82-110`); on the next boot `GuardDowngrade` refuses
if the stamped value exceeds `SchemaVersion()`. So a migration added WITHOUT bumping `SchemaVersion()`
stamps the store past the declared version and the binary refuses to start on the next run - a loud but
runtime-only failure. Bump both, every migration.

## Deliverables

### A - Retire `dismissed` from `patrol_findings.state`

**File scope:**
- `librarian/internal/modules/librarian/collections/0015_patrol_findings_drop_dismissed.go` (new)
- `librarian/internal/modules/librarian/module.go` (`SchemaVersion()` 14->15; manifest += basename)
- a red-able migration test (e.g. `collections/0015_..._test.go` or extend `module_test.go`)
- `docs/pocket-librarian-v1-spec.md` (§4.3 `state` value list at `:499`)

**Shape:** mirror `0010`/`0012` but INVERTED - here the SHRINK is the FORWARD direction (see Open
Questions Q1 for why the ADR's "DOWN remap" wording maps to a forward remap here). FORWARD: (1)
data-first - `FindRecordsByFilter("patrol_findings", "state = 'dismissed'", ...)`, remap each to
`state = 'flagged'` (zero rows expected; defensive); (2) narrow the `state` SelectField `Values` to
`[flagged, fixed, resolved]` (guard-before-shrink so a re-run is a no-op). DOWN: re-add `dismissed`
to `Values` (a grow - no data remap possible or needed; the value's position in the slice is
cosmetic). Idempotent both ways.

**Acceptance:** fresh store's `state` enumerates exactly `flagged, fixed, resolved`; up-then-down
leaves no row outside the current enum; test asserts the shrunk enum + that a pre-seeded
(defensively) `dismissed` row is remapped to `flagged` by the forward migration.

### B - One meaning of "open findings" (disposition-aware counts)

**File scope:**
- `librarian/internal/modules/librarian/tools/query.go` (`queryUncollapsed`, `querySummary`, the
  `:398-399` comment)
- `librarian/internal/modules/librarian/tools/query_test.go` (red-able test)
- (verify only, no edit expected) `cmd/deskkit/pretty.go` renders summary JSON as-is; TUI/MCP route
  through `tools.Query`

**Shape:** `queryUncollapsed` -> `openFindingRows(app, "rule = 'R5' && disposition = 'open'")`;
`querySummary` -> `openFindingRows(app, "disposition = 'open'")`. No migration, no schema, no
`module.go` touch. Update the comment so it stops asserting the old "all flagged rows, no disposition
filter" behavior.

**Acceptance:** with a desk holding one `flagged`+`open` R5 finding and one `flagged`+`wont_fix` R5
finding, `query summary`.`open_findings_total` == 1, `query uncollapsed`.`count` == 1, and both equal
`query findings`.`count` (live default); `--include-disposed` and the summary widen consistently. A
test hits `buildSummary`/`queryUncollapsed`/`queryFindings` against the same fixture and asserts the
three counts agree.

### C - Provenance on `patrol_findings`

**File scope:**
- `librarian/internal/modules/librarian/collections/0016_patrol_findings_provenance.go` (new)
- `librarian/internal/modules/librarian/module.go` (`SchemaVersion()` 15->16; manifest += basename)
- `librarian/internal/modules/librarian/tools/dispose.go` (`DisposeFinding` signature + writes)
- `librarian/internal/modules/librarian/tools/patrol.go` (`inheritedDisposition` -> also carry
  provenance; `fileFinding` sets the inherited fields at `:104`)
- `librarian/cmd/deskkit/main.go` (`findings dispose` flags at `:817-837`)
- `librarian/internal/modules/librarian/tools/dispose_test.go`, `patrol_test.go` (red-able tests)
- `docs/tool-surface.md:44` (dispose signature), `docs/pocket-librarian-v1-spec.md` §5.2

**Shape:** migration `0016` adds three fields (see Open Questions Q2 for names/types/`Max`),
backfilled empty (idempotent, like `0014`); DOWN drops all three (guard-before-remove, no data remap -
whole columns go, mirroring `0014`'s DOWN). `DisposeFinding` gains `actor, reason string` params, sets
`disposed_at = time.Now()` when the target disposition is non-`open` (see Q3 for the open/clear rule).
`inheritedDisposition` (`tools/patrol.go:262-273`) already fetches the most-recent prior disposed
record - refactor it to return `(disposition, actor, reason, disposedAt)` (or a small struct), and have
`fileFinding` (`tools/patrol.go:104`) set all four so a resolve->re-fire cycle preserves who/why/when.

**Acceptance:** `findings dispose <id> --as wont-fix --reason X --by Y` persists actor/reason/
disposed_at; a full-desk patrol that resolves then re-fires the same key files a fresh finding with the
SAME disposition AND provenance; neutrality passes (no default actor). Tests: dispose-writes-provenance;
inherit-on-re-patrol; `--reason`/`--by` optional (existing `dispose <id> --as wont-fix` still works).

### D - Shrink `adoption_log.event`

**File scope:**
- `librarian/internal/modules/librarian/collections/0017_adoption_log_shrink_event.go` (new)
- `librarian/internal/modules/librarian/module.go` (`SchemaVersion()` 16->17; manifest += basename)
- a red-able migration test
- `docs/pocket-librarian-v1-spec.md` §4.6

**Shape:** FORWARD: (1) data-first - find any `adoption_log` row whose `event` is in the dropped set
`{patrol, revert, false_positive, friction, note}` and handle it (zero rows expected; see Q4 for the
handle-vs-delete call); (2) narrow the `event` SelectField `Values` to `[fix]`. DOWN: re-add the five
values. `queryAdoption`, the `query adoption` surface, `recordAdoptionLog`, and deskguard membership
are all UNCHANGED (the ADR keeps `fix` and the reader).

**Acceptance:** fresh store's `event` enumerates exactly `[fix]`; `apply_fix` still writes a `fix`
row and `query adoption` returns it; up-then-down re-widens with no orphan rows; test asserts the
shrunk enum + the `apply_fix` -> `query adoption` round-trip.

## Gate & contract hygiene

ASCII-only cells; run every gate command bare (no pipe), per `_config.md`.

| Gate | Command / trigger | Fires? | Why |
| ---- | ----------------- | ------ | --- |
| DB migration discipline | forward-only; `Max` on `reason`; data-first shrink (0010/0012) | YES (A, C, D) | three shipped-collection migrations 0015-0017 |
| SchemaVersion/manifest twin | `module.go:44` + `:71-84`; `TestMigrations_MatchesCollectionsDir` | YES (A, C, D) | each migration bumps both; manifest unit-guarded, version runtime-guarded (`GuardDowngrade`) |
| Librarian integration | `make verify` (`librarian/verify.sh`) | YES | any `librarian/` change; verify.sh already runs summary/uncollapsed/findings/adoption (`:123-289`) |
| Unit tests | `make test` (`go test ./...`) | YES | behavior change in query/dispose/patrol/migrations |
| Repo checks | `make check` | YES | neutrality + self-test etc. always run |
| Identity neutrality | `scripts/check-neutrality.mjs` | YES | issue-free Go comments; actor/reason free-text, no baked identity |
| Regression-test bar | red-able test per slice + adversarial review on A/C/D | YES | #82 bar + PR #112 data-safety practice |
| CHANGELOG | `[Unreleased]` entry; `check-changelog.mjs` at release | YES (at tag) | product change |
| Bundle drift guard | `make package` + `git diff` on `plugin/claude-plugin/` | NO | no `plugin/core`, `plugin/mcp`, or `schema/` change |
| Plugin bun test/build | `bun test`, `bun run build` | NO | librarian-lane only |
| Version sync | `check-version-sync.mjs` | NO | no `VERSION`/manifest change |
| Kits drift | `check-kits.mjs` | NO | no `kits/` change |
| schema/ contract | `schema/doctypes.yaml`, `profile.schema.yaml` | NO | `state`/`disposition`/`adoption_log` are store collections, not the shared contract; PM lane untouched |
| tool-surface drift guard | (ADR 0016 child, not yet built) | N/A | update `docs/tool-surface.md:44` by hand in C; a guard lands separately |

## Parallelism + landing order

**What parallelizes:** B (disposition-aware counts) is fully independent - it is a pure `query.go`
code change, touches no migration and no `module.go`, and only depends on `disposition` which already
ships (0014). It can land first or concurrently with any of A/C/D.

**What serializes:** A, C, and D each add a migration and therefore each edit the SAME two lines of
`module.go` (`SchemaVersion()` and the manifest) AND claim the next sequential migration number. That
is a shared-file serialization point (foreman rule: shared file gets a serialized chain, not parallel
owners). They can be authored in parallel branches, but they MUST merge one at a time, each rebasing
its migration number + `SchemaVersion()` + manifest entry onto the prior. There is no DATA dependency
among them (A touches the `state` enum, C adds new columns, D touches `adoption_log`) - only the
number/version/manifest chain forces order.

**Recommended safe landing order** (simplest migration first minimizes rework on each rebase):

1. **B** - counts. No schema; closes the most visible incoherence; unblocks nothing but frees the
   others from waiting.
2. **A** - migration `0015` (retire `dismissed`). Pure enum shrink, verified zero affected rows.
3. **C** - migration `0016` (provenance). Adds columns + dispose/patrol code.
4. **D** - migration `0017` (adoption shrink). Enum shrink; independent of the finding changes.

If A/C/D are dispatched to parallel builders, brief each that its migration number and the two
`module.go` lines are provisional and get finalized at merge in the order above; the merging agent
(or foreman) fixes the number/version/manifest on rebase.

**Cross-issue coordination:** migration numbers 0015/0016/0017 here are PROVISIONAL. This issue
and `document-identity-hygiene` both add three librarian migrations on one shared chain; the real
sequence (basenames + the `module.go` `SchemaVersion()`/manifest twin) is assigned at landing time
from true current HEAD - whichever lands first takes the next free numbers, the second continues
from the true next free number; the two issues' migration commits must serialize (rule recorded in
the `epic-design-session-v1` body). Beyond the migrations, this issue also edits `tools/query.go`
and `tools/patrol.go` (and `docs/pocket-librarian-v1-spec.md`), all of which
`document-identity-hygiene` (#123) touches too; the migration rule already forces the two to land
sequentially, so whoever lands second rebases their `query.go` / `patrol.go` hunks onto the first
(disjoint regions today — a rebase, not a redesign; epic coordination rule 3).

## Open questions (with recommended defaults)

**Q1 - "DOWN remap" wording vs. the forward direction (slice A).** ADR 0013 and the brief say
"data-first DOWN remap per the 0012 precedent." In `0010`/`0012` the ADD is forward and the SHRINK
(with the data-first remap) is the DOWN. Retiring `dismissed` INVERTS that: the SHRINK is the FORWARD
migration, so the data-first remap belongs in FORWARD (remap `dismissed`->`flagged` before narrowing),
and the DOWN merely re-adds `dismissed` (a grow needs no remap). **Recommended: forward data-first
remap; down re-adds the value.** This is the direction-correct application of the 0010/0012
"remap-before-shrink" invariant, not a deviation from it. Flagged here because the literal "DOWN
remap" phrasing would otherwise put the remap in the wrong migration half.

**Q2 - Provenance field names / types / Max (slice C).** Recommended: `actor` (TextField, `Max 200` -
short free-text identifier, following `transitions.actor`/`claimed_by` free-text, no relation),
`reason` (TextField, `Max 2000` - content-bearing, matching `feedback.summary`'s ceiling; a short
explanation, not a document), `disposed_at` (DateField, NOT AutodateField - it is set at dispose time,
not record-create time; an AutodateField `OnCreate` would stamp the finding's file time, which is
wrong). All three carry an explicit `Max`/type per the `0013` gotcha. Open to a maintainer rename
(`disposed_by` vs `actor`, `note` vs `reason`) - the migration and dispose code must agree.

**Q3 - Are dispose provenance args required, and what clears them (slice C).** Recommended: `--reason`
and `--by` (actor) are OPTIONAL free-text flags. Rationale: the argv-classification test at
`main_test.go:218` is unaffected either way - that test never executes the command; the real
grounds are neutrality (no default actor can ship) and back-compat for existing scripted callers.
`disposed_at` is set automatically whenever the disposition moves to a non-`open` value. Setting
disposition back to `open` (un-dispose) CLEARS actor/reason/disposed_at (an open finding carries no
disposition provenance). A `wont_fix` with no `--reason` remains anonymous - acceptable, but the CLI
help should nudge toward supplying one. Flag for the owner: require `--reason` on `wont_fix`
specifically? Default recommendation: no (keep optional), record the nudge in help text.

**Q4 - Data-first handling for the adoption_log shrink (slice D).** The five dropped values are
writerless (verified), so no row holds them and there is no meaningful KEPT value to remap them to
(only `fix` survives, and remapping e.g. a `note` row to `fix` would fabricate a fix record).
Recommended: the forward migration DELETES any row holding a dropped `event` value (with a count logged
for observability). A writerless value in a live row is anomalous, not a real adoption record, so
dropping it loses nothing a writer created; this keeps the "no row outside its enum" invariant without
inventing a false `fix`. Alternative to weigh in adversarial review: fail-closed (error) if any
dropped-value row exists, forcing a human decision rather than a silent delete. Given zero rows are
expected, the delete path is defensive-only; the fail-closed alternative is stricter but noisier.

**Q5 - Combine A and C into one migration on `patrol_findings`?** Both touch `patrol_findings`.
Recommended: KEEP SEPARATE (0015 enum shrink, 0016 add columns) - distinct, independently revertible
DOWN paths (0015 re-adds an enum value; 0016 drops three columns), matching the repo's one-concern-per-
migration pattern (0010 resolved, 0014 disposition are separate). They serialize on `module.go`
regardless, so separating them costs nothing and clarifies each revert.

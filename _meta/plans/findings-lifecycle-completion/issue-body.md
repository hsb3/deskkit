> **Tracking:** #TBD, ADR 0013 (2026-07-20 design session). Finish the findings-disposition sub-machine #93 started so "open findings" means one thing everywhere, a disposition records who/why/when, and the enums stop declaring values no code path can reach.

## Problem

#93 (shipped in the 0.8.0 bug floor, PR #112) added a `disposition` axis
(`open`/`acknowledged`/`triaged`/`wont_fix`) to `patrol_findings`, orthogonal to `state`, with
re-patrol inheritance keyed on `(file, rule, checksum)`. It left a half-built machine. The
residual, grounded against current `main`:

- **A dead `state` value.** `patrol_findings.state` still declares `dismissed`
  (`librarian/internal/modules/librarian/collections/0002_patrol_findings.go:23`, plus `resolved`
  added by `0010`), but no code path sets it: every `.Set("state", ...)` in the tree writes
  `flagged` (`tools/patrol.go:93`, `tools/restore.go:106`), `resolved` (`tools/patrol.go:168`), or
  `fixed` (`tools/apply_fix.go:223`). A repo-wide search for a `"dismissed"` state literal returns
  nothing. A declared-but-unreachable value invites a future author to "set a finding dismissed"
  and write a value nothing else understands.
- **"Open findings" means two things.** `query findings` defaults to `disposition = 'open'`
  (`tools/query.go:402`), but `query uncollapsed` (`tools/query.go:382`) and `query summary`
  (`tools/query.go:419`) call `openFindingRows` with no disposition filter, so an
  acknowledged/triaged/`wont_fix` finding still inflates `open_findings_total` and still appears
  uncollapsed. The same `query.go` backs the MCP `query` tool, so MCP inherits the divergence; the
  TUI is a chat-transcript surface (`module.go:62-63`) with no independent count source, so its
  numbers come from the same tool.
- **A disposition can't answer who/why/when.** `DisposeFinding` writes only the new value
  (`tools/dispose.go:49`) - no actor, no reason, no timestamp - so a `wont_fix` is an anonymous,
  unexplained verdict.
- **Five of six `adoption_log.event` values are writerless.** The enum declares
  `[patrol, fix, revert, false_positive, friction, note]`
  (`collections/0005_adoption_log.go:15-16`), but the only writer is `recordAdoptionLog`, which
  always writes the literal `"fix"` (`tools/apply_fix.go:270`). The log is NOT dead weight: it has a
  live reader (`queryAdoption`, `tools/query.go:449-463`, surfaced as `query adoption`) and is third
  in deskguard's desk-collision set (`internal/core/store/deskguard.go:13`) - so the ruling keeps
  `fix` and shrinks the enum rather than killing the collection.

ADR 0013 (`docs/decisions/0013-disposition-completion-adoption-log.md`) rules D4 Option 2:
provenance on the finding, shrink the log.

## Deliverables

- A - **Retire `dismissed` from `patrol_findings.state`.** New forward migration `0015` narrowing
  the `state` SelectField to `[flagged, fixed, resolved]`, data-first (remap any `dismissed` row to
  `flagged` before the shrink; verified zero rows hold it, so this is a defensive no-op). DOWN
  re-adds `dismissed`. Bump `SchemaVersion()` and the `Migrations()` manifest in
  `librarian/internal/modules/librarian/module.go` (both). Correct the spec `state` value list
  (`docs/pocket-librarian-v1-spec.md:499`).
- B - **One meaning of "open findings" everywhere.** In `tools/query.go`, make `queryUncollapsed`
  (`:381-392`) and `querySummary` (`:414-424`) disposition-aware by threading
  `disposition = 'open'` through their `openFindingRows` calls, matching `queryFindings`' default.
  No migration. Reaches CLI + MCP (same code) automatically; confirm the TUI reads via the tool.
  Update the stale comment at `tools/query.go:398-399` that pins the old behavior.
- C - **Provenance on `patrol_findings`.** New forward migration `0016` adding `actor`, `reason`
  (content-bearing, explicit `Max` per the `0013` gotcha), and `disposed_at` (DateField), backfilled
  empty like `0014`. `DisposeFinding` (`tools/dispose.go`) sets them; new CLI flags on
  `findings dispose` (`cmd/deskkit/main.go:817-837`). `inheritedDisposition` (`tools/patrol.go:262`)
  extended so a re-fired finding inherits the provenance alongside the disposition. Bump the
  `SchemaVersion()`/manifest twin. Update `docs/tool-surface.md:44` and spec §5.2.
- D - **Shrink `adoption_log.event` to writer-backed reality.** New forward migration `0017`
  narrowing the `event` SelectField to `[fix]`, data-first (handle any row holding a dropped value;
  verified none exist). DOWN re-adds the five values. Keep `queryAdoption`, its `query adoption`
  surface, and deskguard membership unchanged. Bump the `SchemaVersion()`/manifest twin. Update spec
  §4.6.

## Acceptance criteria

- [ ] A: `grep -R '"dismissed"' librarian/internal/modules/librarian` returns no `state` enum
      declaration; a fresh store's `patrol_findings.state` enumerates exactly `flagged, fixed,
      resolved`; `migrate up` then `migrate down` on a store leaves no row holding a value outside
      its current enum; a red-able regression test asserts the shrunk enum and the data-first remap.
- [ ] B: after disposing a finding `wont_fix`, `query summary`'s `open_findings_total` and
      `open_findings_by_rule`/`_by_severity` exclude it, and `query uncollapsed` no longer lists it -
      the counts equal what `query findings` (live default) reports for the same desk; a red-able
      test pins summary/uncollapsed to disposition-aware counts.
- [ ] C: `findings dispose <id> --as wont-fix --reason "<text>" --by "<actor>"` stores actor/reason/
      disposed_at on the finding; a subsequent full-desk patrol that resolves then re-fires that
      `(file, rule, checksum)` produces a fresh finding carrying the SAME inherited disposition AND
      provenance; a red-able test covers dispose-writes-provenance and inherit-on-re-patrol; no baked
      default actor (neutrality).
- [ ] D: a fresh store's `adoption_log.event` enumerates exactly `[fix]`; `apply_fix` still writes a
      `fix` row and `query adoption` still returns it; `migrate down` re-widens the enum with no
      orphaned rows; a red-able test asserts the shrunk enum and that the `apply_fix` -> `query
      adoption` path is intact.
- [ ] `make verify` (which already exercises `query summary`/`uncollapsed`/`findings`/`adoption` at
      `librarian/verify.sh:123-289`) and `make test` are green; `make check` neutrality passes
      (issue-free Go comments in `librarian/`).

## Dependencies & gates

- **Blocks on:** nothing external; sits at the model + workflow layer (D4 depends on nothing). One
  of the ADR 0010-0017 v1-track slices under the design-session epic.
- **DB migration discipline (fires - librarian):** forward migrations only, never edit an applied
  one; content `TextField`s (`reason`) need an explicit `Max`; enum shrink (A, D) uses the data-first
  down-remap pattern of `0010`/`0012` (remap rows off the value before shrinking). Each migration
  bumps BOTH `SchemaVersion()` and the `Migrations()` manifest in `module.go` - the twin the PR #112
  foreman had to fix; the manifest is guarded by `TestMigrations_MatchesCollectionsDir`
  (`module_test.go:19-54`), and a stale `SchemaVersion()` trips `GuardDowngrade`
  (`internal/core/migrate/migrate.go:128`) at runtime.
- **Migration-number coordination (fires):** migration numbers 0015/0016/0017 here are
  PROVISIONAL. This issue and `document-identity-hygiene` both add three librarian migrations on
  one shared chain; the real sequence (basenames + the `module.go` `SchemaVersion()`/manifest
  twin) is assigned at landing time from true current HEAD - whichever lands first takes the next
  free numbers, the second continues from the true next free number; the two issues' migration
  commits must serialize (rule recorded in the `epic-design-session-v1` body).
- **Librarian integration (fires):** `make verify` - any change under `librarian/`.
- **Unit tests + repo checks (fire):** `make test`, `make check`.
- **Identity-neutrality (fires):** shipped-tree scope; keep Go comments/tests issue-free; `actor`/
  `reason` stay free-text with no baked identity or default actor.
- **CHANGELOG (fires at release):** add an `[Unreleased]` entry.
- **Regression-test bar (fires):** each slice carries a red-able regression test (#82 bar); the three
  data-safety slices (A, C, D - shipped-collection migrations) get independent adversarial review,
  the PR #112 practice.
- **Does NOT fire:** bundle drift guard, `plugin` `bun` build/test, version-sync, kits drift - this
  is librarian-lane Go + docs only; `schema/` is untouched (these are store collections, not the
  shared contract), so the PM lane is unaffected.

## Out of scope

- Wiring any NEW `adoption_log` event (`patrol`/`revert`/`false_positive`/`friction`/`note`) - only
  when a concrete consumer pulls it (ADR 0013). `friction` in particular overlaps the live `feedback`
  collection and must not duplicate it.
- PM-side actor/identity provenance - that is issue #95 (actor param on the PM write tools), a
  different surface (`transitions.actor`), which stays open.
- Whether `dispose` becomes an agent/MCP tool, and per-surface count rendering - D5's boundary.
- Provenance durability across a store rebuild: ADR 0013 accepts that disposition provenance is
  store-only supervisor state and does not survive a rebuild-from-disk under the ADR 0009
  files-are-truth regime; re-opens only if that gate flips.

**Known overlaps.** This issue **supersedes #99** (`Populate or retire query adoption`) - ADR 0013
rules that question (keep `fix` + the reader, shrink the enum), so #99 will be closed citing this.
It is **adjacent to #95** (actor param on PM write tools) - same idea, different lane/surface; #95
stays open.

> **Tracking:** #117, ADR 0012 (2026-07-20 design session). `CreateItem` starts hard-rejecting
> an `items.type` that is not in the shared schema-v1 vocabulary, closing the silent
> ungated-advance hole a typo'd or unknown type opens today.

## Problem

`CreateItem` performs zero vocabulary check on `items.type` today
(`librarian/internal/modules/pm/engine/engine.go`, the function spans lines 129-178). Only
`Title` is validated (`strings.TrimSpace(in.Title) == ""`, line 130); `type` is written
straight from caller input with no check at all (`rec.Set("type", in.Type)`, line 146).

This is a real asymmetry, not a hypothetical gap: the identical checker already gates a
different write path in the same collection family. `gates.ParseRules` validates every
`desk_config.rules` gate-rule item-type string against `schema.Vocab().KnownType` at
config-save time and refuses an unknown one loud
(`librarian/internal/modules/pm/gates/gates.go:57` the function, `:70`
`if !vocab.KnownType(itemType) { return nil, fmt.Errorf(...) }`, and the per-document-requirement
check at `:109-110`). `CreateItem` is the one PM write path the vocabulary check misses.

The consequence is silent, not cosmetic. The gate engine keys its `(item.type, edge)` lookup
directly off the unvalidated string:
`librarian/internal/modules/pm/engine/engine.go:352` —
`reqs := dc.rules.Effective(item.GetString("type"), edgeKey, e.fieldLookup(ctx, item))`. An
item whose `type` matches no gate-config entry gets zero effective requirements back from
`Effective`, so the transition's gate check (`gates.Evaluate`, line 353) passes trivially. A
typo'd or unknown type therefore doesn't fail loudly — it advances through every phase
(queue -> work -> review -> terminal) as if no gate had ever been configured for it, and
nothing in the write path or the audit trail marks that the type was ever suspect.

The checker itself already exists and needs no new code to read: `schema.Vocab()`
(`librarian/internal/core/schema/doctypes.go:57-60`) parses the embedded
`schema/doctypes.yaml` once per process; `(*Vocabulary).KnownType(typ string) bool`
(`doctypes.go:107-111`) is the exact boolean `gates.ParseRules` already calls; and
`(*Vocabulary).TypeNames() []string` (`doctypes.go:132-140`) returns the sorted canonical type
list already built for error messages. `engine.go` already imports the `schema` package
(for `schema.DocumentValidator`), so calling `schema.Vocab()` from `CreateItem` adds no new
import.

Every surface funnels through the one `CreateItem`: the MCP tool (`tools/tools.go:69`
`CreateItem`, calling `newEngine(...).CreateItem`), the CLI/TUI (`scenario/surface.go:46`,
`engineSurface.Create`), and the importer
(`librarian/internal/modules/pm/importer/importer.go`, `e_createItem` at lines 178-199, the
`engine.CreateItem` call at lines 191-196) — "the importer is a THIN driver: every mutation
goes through `engine.CreateItem`/`engine.Link`" (`importer.go:18`). One check at the engine
closes the hole for all of them at once; the importer inherits it for free, no separate wiring.

## Deliverables

- [ ] **The check.** `CreateItem` (`engine.go`, inside the `129-178` span, before
      `rec.Set("type", in.Type)` at line 146) calls `schema.Vocab()` and refuses the write
      (no record created, same transaction aborted) when the vocabulary lookup fails or
      `in.Type` is non-empty and `!vocab.KnownType(in.Type)`. Use the existing `refuse(...)`
      helper (`engine.go:37`, already the pattern for every other actionable `CreateItem`-family
      error, e.g. the dependency-kind check at `engine.go:644`:
      `refuse("unknown dependency kind %q (blocks, is-blocked-by, relates-to)", in.Kind)`).
      The error must name both the offending type and the known types, e.g.
      `refuse("unknown item type %q (known types: %s; see schema/doctypes.yaml)", in.Type,
      strings.Join(vocab.TypeNames(), ", "))`.
- [ ] **Empty/absent `Type` stays legal — this is a scope call made while drafting, flagged
      for the build session to confirm or overrule.** `items.type` is optional in every
      caller-facing shape (`omitempty` in `tools/types.go:44`; a plain unset string field in
      `scenario/surface.go:34` `CreateArgs`), and at least 10 existing call sites across
      `engine`, `tools`, `scenario`, and the realtime tests construct
      `CreateItemInput{Title: ...}` with `Type` left at its zero value (e.g.
      `engine/queries_test.go:49`, `realtime_test.go:79,83`, `tools/parity_test.go:57,67,71,176`).
      ADR 0012's own text discusses only a "typo'd or unknown" type string, never the absence
      of one, and ADR 0012 lists no vocabulary or product-surface change to make `Type`
      mandatory. Given that, the check below is scoped as `in.Type != "" && !vocab.KnownType(in.Type)`
      — an omitted type keeps passing unchanged. Treating a bare empty string as "reject" would
      silently turn an optional field mandatory across MCP/CLI/TUI/importer, an undiscussed
      scope expansion ADR 0012 does not license. If the build session disagrees, that's a
      real design fork worth a line in the PR description, not a silent pick.
- [ ] **Regression test (the #82 bar).** Add `TestCreateItemRejectsUnknownType` to
      `librarian/internal/modules/pm/engine/engine_test.go`, alongside the other
      `CreateItem`-adjacent behavior tests in that file (e.g. `TestUngatedTypeAdvancesFreely`,
      line 153). Assert: (a) `CreateItem(..., CreateItemInput{Title: "x", Type: "no-such-type"})`
      returns a non-nil error and no record is created; (b) the error message names the bad
      type and lists known types (e.g. contains `"analysis"` or another real vocabulary entry);
      (c) a companion case confirms `CreateItemInput{Title: "x"}` (no `Type`) still succeeds,
      locking the deliberate empty-type-passes scope call above. This test must fail on
      pre-change code (today `CreateItem` accepts any string) and pass after the check lands.
- [ ] **Spec prose.** `docs/pm-system-v1-spec.md`:
      - `3.1 items` (the `type` field row, line 391) currently reads "schema-v1 / kit type
        (R2.3, R3.4): decision, task, analysis, ..." with no mention of enforcement — add that
        `create_item` validates it against the schema-v1 vocabulary and hard-refuses an
        unknown type (cite ADR 0012).
      - `4.3 Kit/type ids as the reference vocabulary` (lines 614-624) currently states only
        that "the gate engine validates, at `desk_config` write time, that every `type`/`status`
        referenced by a gate rule is a known kit/schema type" — add a sentence that, since
        ADR 0012, `create_item` enforces the identical check at item birth, closing the
        create-vs-gate-config asymmetry this section already documents on the config side.
- [ ] **Post-creation type-mutation path — verified, found, explicitly out of scope here.**
      ADR 0012's Consequences flag this as unverified ("Post-creation type mutation (if any
      path allows it) must gain the same check when touched"); the D3 decision-book brief's own
      Uncertainties section says the same. It is now verified: `UpdateItem`
      (`librarian/internal/modules/pm/engine/queries.go:519-521`) DOES let a caller change
      `type` post-creation — `if in.Type != nil { item.Set("type", *in.Type) }` — with **zero
      vocabulary check**, exposed through the `update_item` tool
      (`librarian/internal/modules/pm/tools/tools.go:82,89`, `UpdateItem`'s
      `if in.Type != "" { up.Type = &in.Type }` branch). This is recorded, not fixed, in this issue:
      ADR 0012's Decision text and title scope the ruling to creation only, and the D3 brief's
      own "Out of scope / interactions" section explicitly excludes post-creation mutation from
      this decision ("This brief rules only the birth-time (`CreateItem`) enforcement
      question... is out of scope here"). Closing it would extend today's ungated-advance
      protection past what ADR 0012 actually ruled, so it is named here as a known, currently
      unguarded gap and left for a follow-up issue once triaged, rather than folded in silently.

## Acceptance criteria

- [ ] `CreateItem(ctx, CreateItemInput{Title: "x", Type: "<not in schema/doctypes.yaml>"})`
      returns a non-nil error, and no `items` row is created for the call (verifiable via a
      count query in the new test).
- [ ] The returned error's text names the offending type value AND the vocabulary source
      (`schema/doctypes.yaml`, or an equivalent explicit pointer), and lists at least one
      real known type name — independently checkable by reading the error string in the test
      assertion, not by inspecting the implementation.
- [ ] `TestCreateItemRejectsUnknownType` (new, in `engine_test.go`) fails against the
      pre-change `CreateItem` (i.e., reverting only the new check reproduces a red test) and
      passes after the change — this is the red-able-regression-test bar (#82), checked by
      running the test against a stash/revert of the check.
- [ ] `CreateItem(ctx, CreateItemInput{Title: "x"})` (no `Type` set) still succeeds after the
      change — locks the deliberate empty-type-passes scope call.
- [ ] Every existing PM engine/tools/scenario/importer test that already passes a real
      schema-v1 type (`task`, `analysis`, `decision`, etc.) continues to pass unmodified;
      `go test ./...` is green with no fixture edits required beyond the new test.
- [ ] The importer path inherits the check with no importer-side code change: an
      out-of-vocabulary `type` in a manifest's `ManifestItem` now fails `e_createItem`
      (`importer.go:191-196`) with the same actionable error, verified by a table-driven case
      (existing importer tests, or a new one, per whichever this repo's importer test file
      already covers negative-path `CreateItem` failures).
- [ ] `docs/pm-system-v1-spec.md` states the new invariant in both `3.1` (the `type` field
      row) and `4.3` (the kit/type vocabulary section) — checked by reading the diff, not by
      a script.
- [ ] The post-creation type-mutation finding (`UpdateItem`, `queries.go:519-521`) is recorded
      in the PR description or a linked follow-up issue, citing the exact unguarded line, with
      an explicit statement that closing it is out of scope for this ticket (per ADR 0012's own
      creation-only scope and the D3 brief's out-of-scope section).
- [ ] `node scripts/check-neutrality.mjs` passes with no bare issue references introduced in
      any new or edited Go file under `librarian/` (the new test's comments and the new error
      string are issue-free prose).

## Dependencies & gates

- Librarian integration gate (`make verify`, `librarian/verify.sh`, 48 checks, scratch desk):
  fires because this touches `librarian/`. No `pm create-item`/`create_item` invocation is
  currently exercised by `verify.sh` (checked: no match), so this gate is not expected to catch
  a regression here on its own — `go test ./...` (below) is the real bar.
- Unit tests (`make test`): fires. `go test ./...` under `librarian/` must stay green, including
  the new `TestCreateItemRejectsUnknownType` and every existing `CreateItem`-using test
  (`engine_test.go`, `queries_test.go`, `realtime_test.go`, `tools/parity_test.go`,
  `scenario/surface.go`-driven tests).
- Repo checks (`make check`): fires (identity neutrality + self-test always runs). No bare
  `#N` issue refs in any new/edited Go comment or test string under `librarian/` — write
  issue-free prose there; ADR/issue references belong in `docs/` or this issue body only.
- Bundle drift guard (`make package` + `git diff --exit-code`): does NOT fire — this change is
  confined to `librarian/`, not `plugin/core`, `plugin/mcp`, or `schema/`.
- Version sync / Kits drift: do NOT fire — no `VERSION`/manifest change, no `kits/` change.
- DB migration discipline: does NOT fire — `items.type` stays a bare `TextField`; this is an
  application-level check only (the same pattern `gates.ParseRules` already uses for
  `desk_config.rules`), no PocketBase collection/migration change.
- Regression-test bar (#82): fires — see the `TestCreateItemRejectsUnknownType` deliverable
  above; this is a behavior change to a write path with no pre-existing test coverage of the
  negative case.
- CHANGELOG: an entry under `[Unreleased]` is required before this can ship in a tagged release
  (`check-changelog.mjs` gates the tag, not this PR).

## Out of scope

- Changing `schema/doctypes.yaml`'s vocabulary content (adding/removing/renaming types). This
  issue validates against the vocabulary as it stands; it does not touch what's in it.
- The schema v2 element list / two-track model (ADR 0009, ADR 0018). Per the D3 decision-book
  brief's platform-stream note, the vocabulary this check reads is schema-v1 doctypes now and
  becomes the v2 element list when that track lands, unmodified by this change — no coupling
  to the v2 track is introduced here.
- Closing the `UpdateItem` post-creation type-mutation gap (`queries.go:519-521`). Verified to
  exist (see Deliverables); recording it, not fixing it, is this issue's scope per ADR 0012's
  creation-only ruling and the D3 brief's explicit out-of-scope statement on post-creation
  mutation.
- A workflow-layer, patrol-style post-hoc detector for already-created out-of-vocabulary PM
  items (the D3 brief's Option C companion note). Patrol today reaches only the librarian's
  `files` collection, not PM `items`; extending it is new, undecided workflow-layer scope.
- A DB-level `SelectField` enum for `items.type` (a possible future migration). This issue is
  an application-level check only; a later move to an enum column is a forward migration with
  its own data-fix plan for any live-store row already outside the new enum, not part of this
  change.
- Auditing existing desk manifests/seed data for out-of-vocabulary `type` strings that would
  now fail import. The D3 brief flags this as unverified (no dossier confirms whether any
  current fixture is affected); this issue does not include that audit.

# Document identity + schema hygiene (ADR 0017)

Status: draft
Date: 2026-07-20

## Tracking

#123. ADR: `docs/decisions/0017-document-identity-and-hygiene.md` (Accepted,
2026-07-20). Raised by decision book `_meta/research/2026-07-design-session/decision-book/D8-backlog-identity-and-hygiene.md`,
promoted from pull-only backlog to a full session ruling at the owner's 2026-07-20 scope
sign-off. Companion issue body: `issue-body.md` in this folder.

## Problem, grounded

Three independent, low-urgency gaps the 2026-07-20 design session promoted from backlog to a
ruled decision (D8, all three sub-items ruled together as one package):

- **(a) Rename discards history.** `files.path` is the unique key
  (`librarian/internal/modules/librarian/collections/0001_files.go:14,30`); `Sweep` matches only
  by path (`librarian/internal/modules/librarian/tools/sweep.go:49-52`), so an unseen path is
  soft-deleted (`sweep.go:86-94`) and a new path is always a fresh insert (`sweep.go:64-73`) with
  no link to what it used to be. No identity primitive exists today. Any fix must be
  re-derivable by a sweep from disk alone (files-are-truth, ADR 0009) — ruling out a pure
  store-side counter.
- **(b) Naming collision.** `files.entity_type` (`collections/0001_files.go:16`) holds a
  frontmatter `type` value (a doctype string); `schema/doctypes.yaml:29`'s `entity_type` enum is
  an unrelated person/company classification bound to a different frontmatter field
  (`doctypes.yaml:32-35,69`). Same name, disjoint everything else — a confusion trap, not a
  correctness bug (nothing reads the column expecting the schema's enum).
- **(c) Unguarded implicit caps.** Seven content-bearing `TextField`s across four collections
  still ride PocketBase's 5000-char implicit default after migrations 0011/0013 widened three
  other fields and set the "explicit `Max`" convention — see issue-body.md's Problem (c) table for
  the full list with citations.

## Deliverables

| Slice | What ships | File scope (primary) |
| ----- | ---------- | --------------------- |
| A | Frontmatter `id` matched at sweep time; new `files.doc_id` column via forward migration; `schema/doctypes.yaml` (+vendored copy) documents `id`; two docs sharing an `id` fall back to path-matching and file a patrol-visible finding (no silent merge) | `librarian/internal/modules/librarian/tools/sweep.go`; new `librarian/internal/modules/librarian/collections/0015_files_doc_id.go`; `schema/doctypes.yaml`; `librarian/internal/core/schema/doctypes.yaml`; `docs/pocket-librarian-v1-spec.md` |
| B | Rename `files.entity_type` -> `files.doctype` via forward migration; every reader/writer of the literal column/JSON-key name updated; DOWN renames `doctype` back to `entity_type` in place (stable field id, no data loss) | new `librarian/internal/modules/librarian/collections/0016_files_doctype_rename.go`; `tools/sweep.go`, `tools/query.go`, `tools/propose_fix.go`, `tools/patrol.go`; `cmd/deskkit/pretty.go`; `docs/pocket-librarian-v1-spec.md` |
| C | One migration setting explicit `Max` on the seven flagged fields; a new CI recurrence guard | new `librarian/internal/modules/librarian/collections/0017_content_field_caps.go`; new `scripts/check-textfield-max.mjs`; `.github/workflows/ci.yml`; `Makefile` `check` target |

Per-slice acceptance is stated in full in `issue-body.md`'s Acceptance criteria section (not
duplicated here) — this plan cites only the file scope and the gate/sequencing concerns that
issue body doesn't carry.

## Gate & contract hygiene

| Gate | Fires? | Note |
| ---- | ------ | ---- |
| Repo checks (`make check`) | YES | plus the NEW `check-textfield-max.mjs` this issue adds to the same target |
| Unit tests (`make test`) | YES | go test under librarian/; new tests per slice |
| Librarian integration (`make verify`) | YES | touches librarian/ |
| Bundle drift guard (`make package` + diff on `claude-plugin/`) | NO | doctypes.yaml is not copied into claude-plugin/ (only profile.schema.yaml is, per plugin/package.json:17); zero references to "doctypes" anywhere under plugin/ (grep-confirmed) |
| Doctypes vendored-copy test (`TestDoctypesEmbeddedCopy_MatchesRepoRoot`) | YES | part of make test, not make package; fires because slice A edits schema/doctypes.yaml |
| Identity neutrality (`check-neutrality.mjs` + self-test) | YES, always | issue-free comments/tests in every new/edited Go file |
| Version sync / kits drift | NO | no VERSION or kits/ change |
| DB migration discipline | YES, x3 | forward migrations only; 0001_files.go stays untouched for both A and B; alter-in-place pattern per 0012_dir_kind_add_infra.go; any NEW content TextField this issue itself adds needs explicit Max (none does) |
| Regression-test bar (issue #82 practice) | YES, all 3 slices | red-able test per slice, stated in issue-body.md |
| Adversarial review (PR #112 practice) | YES, for A and B | data-safety-adjacent: a rename bug or an incomplete column-rename can silently corrupt or misroute desk content |
| CHANGELOG | YES, at release | check-changelog.mjs gates the tag, not the PR |

## Parallelism and landing order

The real constraint is not file overlap (A, B, and C touch mostly disjoint files) — it is that
**all three slices share ONE ordered manifest and ONE version integer** on the librarian module:
`(*Mod).Migrations()`'s `basenames` slice and `(*Mod).SchemaVersion()`
(`librarian/internal/modules/librarian/module.go:44,72-78`). Two builders each claiming the next
free sequence number (both writing "0015_...") in parallel branches collide on merge; whichever
lands second must renumber, and `SchemaVersion()` must equal the true highest sequence after
either lands or `GuardDowngrade` refuses the store on next start (`migrate.go:122-135`).

**Cross-issue coordination:** migration numbers 0015/0016/0017 here are PROVISIONAL. This issue
and `findings-lifecycle-completion` both add three librarian migrations on one shared chain; the
real sequence (basenames + the `module.go` `SchemaVersion()`/manifest twin) is assigned at landing
time from true current HEAD — whichever lands first takes the next free numbers, the second
continues from the true next free number; the two issues' migration commits must serialize (rule
recorded in the `epic-design-session-v1` body).

Two landing shapes both work; pick one before starting:

- **Single-PR landing (recommended default).** One builder authors all three migrations
  (0015/0016/0017) and bumps `SchemaVersion()` 14 -> 17 in one change. No coordination cost, no
  renumbering risk. Downside: one larger PR, one reviewer covering three independent behaviors —
  mitigate by keeping the three migrations, their call-site edits, and their tests in clearly
  separated commits within the PR.
- **Serialized multi-PR landing.** Three PRs, MERGED IN STRICT SEQUENCE (never in parallel):
  each PR claims the next basename only after the prior PR has actually merged to main (so the
  sequence number and `SchemaVersion()` bump are always based on the true current state, not a
  branch snapshot). Recommended order, by risk and independence:
  1. **C first** (the cap migration + CI guard). Lowest risk (no rename, no new identity
     semantics), and the CI guard is otherwise-independent of A/B's Go changes — landing it first
     means A and B's own new migrations are covered by the guard's exemption-list discipline from
     the start rather than needing a follow-up pass.
  2. **B second** (the doctype rename). Mechanical, fully enumerated blast radius (issue-body.md's
     Problem (b) list), no new schema concept — lower risk than A.
  3. **A last** (frontmatter id + sweep-matching change). The only slice that changes MATCHING
     BEHAVIOR in `Sweep` itself (not just a column name or a Max value) — the highest-risk slice,
     landed last so its own migration is `0017` only after B has already claimed `0016`, and so
     its reviewer is looking at the smallest, cleanest diff against a tree B has already
     stabilized.
  If multi-PR is chosen, state the CHOSEN basename explicitly in each PR description before
  opening it (not "the next available slot") so a concurrent second PR cannot silently claim the
  same number — and name the sibling issue too: `findings-lifecycle-completion` claims the same
  0015/0016/0017 numbers on this same chain, so state there which of the two issues landed first
  and which basenames it actually consumed, not just which of this issue's own three PRs merged
  in what order.

The CI guard itself (part of C) has no Go-code dependency on A or B and could theoretically be
authored standalone first even under single-PR landing — but its exemption list should be
written against the POST-migration field state (i.e., after C's own cap migration removes the
seven fields from needing an exemption), so building the guard and the migration in the same
commit avoids a throwaway intermediate exemption-list edit.

## Open questions (with recommended defaults)

1. **New DB column name for slice A.** The D8 decision-book brief itself left this open
   ("adds `id`/`doc_id`"). Resolved here with a concrete technical reason, not just a naming
   preference: PocketBase reserves the literal field name `id` for the collection's own system
   primary key (`FieldNameId = "id"`, vendored `pocketbase@v0.39.6/core/field.go:20`); a second
   field also named `id` fails collection validation
   (`validation_duplicated_field_name`, `core/collection_validate.go:274`; the PK-field lookup by
   that exact name is `core/collection_validate.go:371-374`). **Default: DB column `doc_id`,
   frontmatter key stays `id`** (no collision at the YAML level).
2. **Where `id` lives in `schema/doctypes.yaml`.** The file has no existing "optional universal
   key" list construct — only `universal:` (required) and per-type `required`/`optional`.
   `ValidateFrontmatter` (`librarian/internal/core/schema/doctypes.go:145-171`) never rejects an
   unrecognized key, so introducing a new unused top-level YAML list (e.g. `optional: [id]`) that
   nothing reads would itself be a minor hygiene smell. **Default: document `id` in prose
   (a comment block near `universal:`, matching the file's existing REFINEMENT-comment style) —
   no new structural YAML key, since nothing in the validation engine would consume one.** If the
   build session later wants `id` enforced (format-checked, deduped across the desk), that is new
   scope beyond ADR 0017's ruling.
3. **CI guard scope: librarian lane only, or librarian + pm?** The ADR's own evidence (the D8
   decision book, `document-model-gaps.md`'s "smaller hygiene findings") enumerates only the
   seven librarian-lane fields; the PM lane's `notes.body` and `desk_config.rules`
   (`librarian/internal/modules/pm/collections/collections.go:141,156`) are ALSO content-bearing
   with no explicit `Max`, per this repo's own data-model dossier, but are not named in D8's
   evidence. Scoping the guard to include the PM lane would make it fail immediately on
   pre-existing, un-ruled fields (scope creep the ADR did not license); scoping it to librarian
   only leaves a known adjacent gap unrecorded if not stated. **Default: scope the guard (and the
   migration) to `librarian/internal/modules/librarian/collections/` only; record the PM-lane gap
   explicitly in the issue's Out of scope (done above) as a follow-up candidate**, rather than
   silently expanding or silently dropping it.
4. **Exemption-list contents for the CI guard.** Two distinct categories, both needed or the
   guard is red on day one:
   - **Permanently-short structural fields** (never expected to carry a body, by role): `path`,
     `checksum`, `git_last_commit`, `fm_created`, `fm_updated`, `run_id`, `rule`, `patrol_run`,
     `desk`, `provider`, `model`, `run_label`, `tool_call_id`, `tool_name`, `source`, `key`,
     `name`, `original_checksum`, `new_path`.
   - **Known-but-deferred content debt** (content-adjacent, Max-less, NOT swept by this ADR's
     migration, per Open Question 3's scope call and issue-body.md's Out of scope): `files.desk`,
     `files.doctype` (post-rename), `files.status`, `files.synopsis`, `files.origin`,
     `files.graduated_to`.
   **Default: both categories go in the exemption list, tagged distinctly in the script's own
   source comment** (mirroring `scripts/check-scaffold-frontmatter.mjs`'s commented
   `INSTRUMENT_ASSETS` array style) so a future reader can tell "this is fine forever" from "this
   is debt, pick it up later" at a glance.
5. **Field-by-field `Max` sizing for slice C.** Unruled by any dossier (flagged explicitly as
   such in the D8 brief's own Uncertainties). **Default: 50000 for `patrol_findings.detail` and
   `.proposed_fix` (may embed a full replacement body); 2000 for `patrol_log.summary`,
   `adoption_log.detail`, and the three `agent_runs` summary/error fields (all "deliberately
   bounded summaries" per 0011's own comment)** — sized by role per the 0011/0013 precedent, not
   evidence of an actual observed body length.
6. **Migration basenames.** Proposed here as `0015_files_doc_id.go`, `0016_files_doctype_rename.go`,
   `0017_content_field_caps.go` — advisory only; the actual sequence depends on which slice lands
   first under whichever landing shape (single-PR vs. serialized) is chosen (see Parallelism
   above). Whoever builds should state the final basenames and confirm `SchemaVersion()` in the
   PR description, not leave it implicit.
7. **Missing `SchemaVersion` drift test on the librarian module.** Not part of ADR 0017's ruling,
   but discovered verifying this plan: `librarian/internal/modules/librarian/module_test.go` has
   `TestMigrations_MatchesCollectionsDir` (basenames vs. disk) but no equivalent of the PM
   module's `TestMigrations_MatchOwnedCollections`'s `SchemaVersion()`-equals-highest-sequence
   assertion (`librarian/internal/modules/pm/module_test.go:69-101`, whose own comment claims to
   mirror "the librarian module's" check — a claim not currently true). **Recommendation (not a
   hard requirement of this ADR): add the equivalent assertion to the librarian module_test.go in
   the same PR that adds these migrations**, since forgetting the `SchemaVersion()` bump is
   exactly the kind of self-inflicted `GuardDowngrade` lockout this three-migration change makes
   newly likely. If deferred, flag it as a named follow-up rather than letting it silently stay
   missing.

## Unverified / flagged for the build session

- **Live-store field lengths.** No dossier or this pass checked an actual running store's column
  data for values already near/over 5000 chars on the seven flagged fields, or for any
  `entity_type` value already relied upon by name in an external script/dashboard outside this
  repo. The cap migration and the rename are both schema-only changes; a live desk with unusually
  long content in one of the seven fields is unaffected either way (widening only helps), but a
  live desk with an external consumer keyed on the literal string `"entity_type"` (not covered by
  this repo's own tests) would need its own migration note — out of this plan's visibility.
- **Whether any current desk file already uses a frontmatter `id:` key for an unrelated purpose.**
  Not grepped in this pass; a collision with an existing, differently-purposed `id:` key on some
  desk's document would be a silent behavior change for that one file once slice A ships. Worth a
  quick `rg "^id:" **/*.md` sweep on the build session's own desk(s) before merging, not
  something this plan can check for every possible desk.

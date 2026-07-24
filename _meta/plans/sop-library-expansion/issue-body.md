> **Tracking:** #36, SOP template library expansion for the librarian's embedded fixer template
> set. **PARKED post-1.0** (owner sign-off, 2026-07-21 — `_meta/HANDOFF.md:74`). This body corrects
> the "Current state" section against the repo tree as of 2026-07-23; it does not reopen or change
> the parked disposition.

## Summary

Grow the librarian's embedded template set from the current **2 minimal fixer templates** into a
**full SOP template library** covering the ~20-23 executive-desk document types. This is the
feature the "librarian knows ~20 template SOPs" mental model was pointing at.

**Correction (2026-07-23):** the original Summary read *"today that catalog does not exist in the
repo; the librarian ships two placeholder-safe templates by design."* The second clause is still
true; the first is now **false and struck**. ~~today that catalog does not exist in the repo~~ —
the catalog now exists in the repo as the top-level `kits/` tree (see "Current state" below). What
remains missing is the librarian-binary/runtime half: embedding or otherwise consuming that catalog
and selecting the right entry per file.

**Target: next major version bump.** This reshapes the "templates-only" content boundary from a
2-template safety net into a real authoring library, and likely touches the `dir_kind` /
document-type classification model and the planner's template-selection logic — too much surface
for a minor bump.

## Current state (reconciled 2026-07-23 against today's tree)

**A. Librarian-embedded templates — unchanged, still exactly 2.**
`librarian/templates/{frontmatter-universal,pointer-stub}.md`, wired via `//go:embed` in
`librarian/templates/templates.go:14-18` (`FrontmatterUniversal`, `PointerStub`; a third embed,
`SystemPrompt`, is the agent system prompt, not a content template). Boundary
(`docs/development/specs/pocket-librarian-v1-spec.md:1541`, §5.4 write path at `:1187-1246`): all written content
comes from approved templates only; planners never synthesize prose. These two remain
remediation-only (R1 frontmatter fence, pointer-stub collapse), placeholder-safe, no document-type
scaffolding.

**B. The ~20-23 type SOP catalog this issue proposed vendoring — already ported, but NOT into the
librarian binary.** `hsb3/desk-standard#49` (epic #55; both issues CLOSED) shipped the same
headcase SOP set this issue's "Source of truth" section names, at a top-level `kits/` tree:

- 23 kit dirs on disk, 23 entries in `kits.yaml` (`find kits -mindepth 1 -maxdepth 1 -type d | wc -l`
  = 23; `grep -c '^  - dir:' kits.yaml` = 23) — the guide/template/example (G/T/E) triad per type,
  matching this issue's 22-named-types-plus-`user-defined` count.
- Indexed by root `kits.yaml`, drift-guarded by `node scripts/check-kits.mjs` (wired into
  `make check` + CI; `scripts/check-kits.mjs:1-20`) — fails if the manifest and `kits/` tree
  disagree in either direction, and separately blocklist-scans `kits/` + `schema/` for origin-vault
  identifiers (`scripts/check-kits.mjs:21-30`).
- Neutralized and documented in `docs/decisions/0006-kit-port-schema-reconciliation.md` (Accepted,
  corrected 2026-07-18 and 2026-07-21): 9 private-vault path leaks found and scrubbed (2026-07-18
  correction), and — load-bearing for this issue — a 2026-07-21 correction stating explicitly:
  **"no lane consumes `kits/` at runtime today."** No shipped plugin skill references `kits/`
  (`docs/decisions/0006-kit-port-schema-reconciliation.md:20-22`), and the librarian binary
  structurally cannot reach it (`go:embed` cannot cross `..`; `kits/` sits at repo root by design —
  same doc, lines 23-24). Verified independently this pass: `grep -rl kits plugin/ plugins/` returns
  zero hits.
- Companion `schema/doctypes.yaml` (added by the same port) reconciles kit types against schema v1's
  doc-type model — 7 types added, `user-defined`'s nonstandard types deliberately excluded (full
  disposition table: `docs/decisions/0006-kit-port-schema-reconciliation.md`, "Schema-gap
  dispositions").
- The only present runtime-adjacent link is a **compatibility test, not integration**:
  `librarian/templates/kit_render_test.go` reads `kits/analysis/template.md` off disk (by relative
  path escape, since embed can't cross `..`) and proves it still renders through the fixer's
  `Render()` — it does not wire kit selection into `propose_fix`/`apply_fix`.

**Net correction:** this issue's residual is no longer "port the vault content into the repo" (done,
via #49) — it is the half #49 explicitly deferred: **how the librarian binary/runtime consumes the
already-vendored `kits/` catalog** (embed-at-build vs. runtime read), plus the selection/
classification logic. The design questions below already anticipated most of this; question 1
("vendor vs. sync") is now answered in one dimension (vendored, at `kits/`) and open only in the
remaining one (how the librarian reaches it).

## Proposed

A first-class SOP template library so the librarian can scaffold/propose the *right document type*
from an approved template, not just patch frontmatter. Each SOP type becomes an approved, embedded
(or embedded-equivalent) content source the librarian knows and can select correctly.

## Source of truth

The `kits/` port (#49) already carried this forward from the origin headcase shared SOP set
(`~/Library/Mobile Documents/iCloud~md~obsidian/Documents/hsb-2026/_headcase/shared/sops/`) — see
"Current state" (B) above for the shipped result. This section is now historical provenance, not a
to-do: the 22 SOP types plus `user-defined` slot are on disk at `kits/`, neutralized, and manifest-
guarded. The triad shape (`template.md` fill-in skeleton with `{{date}}`/`{placeholder}` tokens
compatible with `templates.Render`; `guide.md` when-to-write/how-to/anti-patterns; `example.md`
worked instance) is unchanged from the original description. `postmortem`, `release-notes`,
`user-defined` still lack a `template.md` by design (composite/nonstandard kits — confirmed against
`kits.yaml`).

## Design questions to resolve before build (need rulings — preserved from the original filing)

1. **Vendor vs. sync.** Embed the templates in the binary (current `//go:embed` pattern, stays
   self-contained) or read a desk-local `sops/` dir at runtime (lets a desk override/extend)? Or
   both — embedded defaults + desk-local overrides? **Partially answered 2026-07-23:** the source
   content is now vendored in-repo at `kits/` (not desk-local, not yet binary-embedded); the open
   half is how the librarian binary/runtime reaches that already-vendored set — copy into
   `librarian/templates/` at build time (since `go:embed` can't cross `..`), or read `kits/`-shaped
   content from a synced desk-local copy, or both.
2. **Triad scope.** Ship just `template.md` (the authoring skeleton), or also surface `guide.md`
   (the librarian could cite when-to-write / anti-patterns when proposing a type) and `example.md`?
3. **Type <-> classification.** How does an SOP type map to the existing `dir_kind` model (9 values:
   `decisions, tasks, analyses, journal, meta, memory, root, other, infra` —
   `docs/development/specs/pocket-librarian-v1-spec.md:476`) and the `type:` frontmatter field (23 values in
   `schema/doctypes.yaml`)? The two enums are structurally different sizes today — this gap is
   confirmed still open, not resolved by the #49 port. Does adding the library change `sweep`
   classification or introduce a new "expected type for this dir" check the patrol rules can lean
   on?
4. **Selection.** How does the planner pick the right template for a file/dir — by `type:`
   frontmatter, directory name, an agent judgment call? This is the new logic a major bump implies.
5. **Neutrality.** The `template.md` skeletons look identity-neutral, but `example.md` / `guide.md`
   reference real projects. **Status update:** the #49 port already ran this exercise for the
   vendored `kits/` copy (`docs/decisions/0006-kit-port-schema-reconciliation.md`, item 2 +
   corrections) and `kits/` is inside `scripts/check-neutrality.mjs`'s `SCAN_DIRS`
   (`scripts/check-neutrality.mjs:54`). Anything additionally vendored or copied into
   `librarian/`/`plugin/` under this issue's build must independently pass that lint — copying a
   file across the boundary does not inherit a prior clean scan.
6. **Boundary interaction.** Confirm this stays inside "templates-only, never synthesize": the
   librarian fills tokens/structure from the template but still does not invent prose.

## Acceptance criteria

- [ ] Design ruling recorded as an ADR for the questions above, especially vendor/embed mechanics
      (Q1, now scoped to "how the binary reaches the already-vendored `kits/` set") and
      type<->classification (Q3, confirmed still open against `dir_kind`).
- [ ] The chosen SOP template set is reachable by the librarian at runtime (embedded copy, build-time
      sync, or desk-local read per the Q1 ruling) and `node scripts/check-neutrality.mjs` passes
      clean over `plugin/` + `librarian/` with the new surface included.
- [ ] The librarian can scaffold/propose at least the core types from an approved template, tokens
      substituted via `templates.Render` (or equivalent), zero synthesized prose — verifiable by a
      test asserting the written file's content matches the source template plus only the documented
      substitutions.
- [ ] `librarian/verify.sh` gains a check proving a template-scaffolded file is written from the
      template (extends the existing pattern set by `librarian/templates/kit_render_test.go`, which
      today proves render-compatibility only, not integration).
- [ ] `docs/development/specs/pocket-librarian-v1-spec.md` §5.4 and the `dir_kind`/classification sections are updated
      to describe the library and the resolved type<->classification mapping.
- [ ] If the build touches `kits/` or `kits.yaml`, `node scripts/check-kits.mjs` stays green (manifest
      and tree agree).

## Dependencies & gates

- **Governing dependency: parked post-1.0.** This issue does not schedule or build until an owner
  ruling reopens it post-1.0 (owner sign-off, 2026-07-21 — `_meta/HANDOFF.md:74`). No other gate in
  this section authorizes starting work ahead of that ruling.
- **Identity neutrality** — `node scripts/check-neutrality.mjs` (scope: `plugin/` + `librarian/` +
  `kits/` recursively; `docs/` and repo-root files exempt). Fires on anything this issue vendors or
  copies into `librarian/` or `plugin/`, independent of the #49 port's prior clean scan of `kits/`
  itself.
- **Kit-manifest drift** — `node scripts/check-kits.mjs`. Fires only if the build reads from, copies
  from, or otherwise touches `kits/` or `kits.yaml`.
- **Librarian integration** — `make verify` (`librarian/verify.sh`, scratch desk). Fires because this
  issue's own acceptance criteria require a new `verify.sh` check for template-scaffolded output.
- **Repo checks + unit tests** — `make check` and `make test` fire always (any change under
  `librarian/` or `plugin/`).
- **ADR required** — the design questions above are explicitly unresolved; an ADR recording the
  ruling (vendor/embed mechanics, triad scope, type<->classification, selection logic) is a hard
  precondition per this issue's own acceptance criteria, not an optional gate.
- **Spec drift** — `docs/development/specs/pocket-librarian-v1-spec.md` §5.4 and the `dir_kind` sections must be
  amended in the same change that lands the library, per this issue's acceptance criteria; no
  automated drift guard covers prose spec sections, so this is a manual review gate.
- **Does NOT fire:** version-sync, bundle-drift (`make package`), CHANGELOG (only relevant once work
  actually lands, not at parked/plan stage), DB migration discipline (no PocketBase collection
  change implied by the current design questions).

## Notes

- Prior "template" issues (#10 issue/PR templates, #11 desk-setup scaffold) are unrelated — those are
  repo scaffolding, not SOP content templates.
- The `sops-local/` directory type (build-brief; off-repo decision 0013 item 6) is the desk's home for
  locally-authored SOPs and is the natural landing zone for the desk-local-override half of Q1.
  **Verified 2026-07-23:** no `sops-local` reference exists anywhere in the repo yet (schema, docs,
  or code) — this remains a future concept, not a shipped one.
- This body was reconciled against the shipped `kits/` tree on 2026-07-23; see
  `_meta/plans/sop-library-expansion/plan.md` for the lean parked-plan record.

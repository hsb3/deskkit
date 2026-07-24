---
title: "SOP template library expansion — parked plan"
type: spec
status: parked
created: 2026-07-23
purpose: "Lean scheduling record for #36 (grow the librarian's embedded fixer templates into a full SOP template library), reconciled against the now-shipped kits/ tree so an unparking owner has a source-grounded starting point, not a cold re-read of a stale issue."
notes: "Issue #36 (open, PARKED post-1.0 per owner sign-off 2026-07-21, _meta/HANDOFF.md:74). Related CLOSED work: #49 (kit port) / epic #55 (PM system v1), ADR docs/decisions/0006-kit-port-schema-reconciliation.md."
---

# Plan - SOP library expansion

_Lean, PARKED plan for #36: growing the librarian's embedded fixer template set (currently 2
templates) into a full SOP template library. RESIDUAL ONLY — the source content this issue
originally proposed vendoring from the headcase vault has already shipped, via #49/ADR 0006, as
the top-level `kits/` tree (23 kits, manifest-guarded, neutralized). What is left is exactly the
half #49 explicitly deferred: how the librarian binary/runtime reaches and selects from that
already-vendored catalog. This plan does not reopen or change the parked disposition._

Status: parked (2026-07-23)

## Tracking

- Issue: #36 (open; PARKED post-1.0 — owner sign-off 2026-07-21, `_meta/HANDOFF.md:74`)
- Related, CLOSED (do not re-plan): #49 "Port the 23 headcase SOP kits + schema reconciliation",
  epic #55 "PM system v1 — full build"
- Prior ADR this plan builds on: `docs/decisions/0006-kit-port-schema-reconciliation.md` (Accepted,
  corrected 2026-07-18, 2026-07-21) — the port that shipped `kits/`
- No ADR yet for THIS plan's own design questions — recording one is the first build slice below
- Companion: `_meta/plans/sop-library-expansion/issue-body.md` (the corrected #36 body this plan
  tracks against)

## Unpark condition

**Governing dependency: an owner ruling that reopens #36 post-1.0** (per the 2026-07-21 sign-off
recorded at `_meta/HANDOFF.md:74`). Nothing in this plan authorizes starting build work ahead of
that ruling. The plan exists only so an unparking owner has a scoped, source-grounded starting
point instead of a cold read of a now-partly-stale issue.

## Current state (verified 2026-07-23 against today's tree; full citations in issue-body.md)

| Surface | State |
| --- | --- |
| Librarian-embedded templates (`librarian/templates/`) | Still exactly 2 content templates: `frontmatter-universal.md`, `pointer-stub.md` (`templates.go:14-18`), plus the unrelated system-prompt embed. Unchanged since #36 was filed. |
| SOP catalog source content | Already vendored: 23 kit dirs on disk = 23 `kits.yaml` entries, guarded by `scripts/check-kits.mjs` (`make check` + CI). Shipped via #49/epic #55 (both CLOSED), documented in ADR 0006. NOT part of this issue's remaining scope. |
| Runtime consumption of `kits/` | None. ADR 0006's 2026-07-21 correction states explicitly: no lane consumes `kits/` at runtime. Only a render-compatibility test exists (`librarian/templates/kit_render_test.go`), not a selection/integration path. |
| Type-to-dir_kind classification | Still open: `schema/doctypes.yaml` has 23 types; `dir_kind` (`docs/development/specs/pocket-librarian-v1-spec.md:476`) has 9 values. No mapping between them is defined anywhere. |

**Net effect on scope:** #36's residual has shrunk to a runtime/binary-integration problem. The
"port + neutralize the vault content" work this issue's original body scoped is done and should not
be re-planned.

## First build slice (when unparked): design ruling -> ADR

Before any code lands: resolve the 6 design questions carried in `issue-body.md` (vendor/embed
mechanics now scoped to "how the binary reaches the already-vendored `kits/` set", triad scope,
type<->`dir_kind` classification, selection logic, neutrality re-verification for anything newly
copied across the `librarian/`/`plugin/` boundary, and the templates-only boundary confirmation) as
a single ADR. This is the first slice specifically because every other slice below depends on its
rulings; sequencing build work ahead of it would risk re-doing it under a wrong assumption.

## Sliced build outline (post-ADR; indicative only — re-scope against the ADR's actual rulings)

- **A - Runtime reach.** Wire the librarian to the ADR's chosen mechanism for reaching the vendored
  `kits/` catalog (build-time copy into `librarian/templates/` since `go:embed` cannot cross `..`,
  a desk-local synced read, or both). Likely file scope: `librarian/templates/`, a new copy/embed
  step, possibly `librarian/internal/core/tools/propose_fix.go` at the call site.
- **B - Selection logic.** Implement the type-selection rule the ADR rules on (frontmatter `type:`,
  directory name, or agent judgment) inside the fix-proposal path.
- **C - Classification mapping.** Resolve the type<->`dir_kind` gap per the ADR's ruling; may touch
  `schema/doctypes.yaml` and the `dir_kind` sections of `docs/development/specs/pocket-librarian-v1-spec.md`.
- **D - verify.sh coverage.** New `librarian/verify.sh` check proving a template-scaffolded file's
  content traces to its source template plus only documented substitutions (no synthesized prose).
- **E - Spec + doc updates.** `docs/development/specs/pocket-librarian-v1-spec.md` §5.4 and the `dir_kind` sections
  amended in the same change that lands the library, not deferred.

## Acceptance (verifiable; restated from issue-body.md so an unparking owner needs no cross-reference)

- [ ] ADR recorded ruling the design questions (Q1-Q6 in `issue-body.md`).
- [ ] `node scripts/check-neutrality.mjs` clean over `plugin/` + `librarian/`, including any newly
      embedded or copied kit content (a prior clean scan of `kits/` itself does not carry over).
- [ ] Scaffolded output for at least the core types is provably template-sourced: a test asserts the
      written file matches the source template plus only documented substitutions, zero synthesized
      prose.
- [ ] `librarian/verify.sh` gains the template-scaffold-provenance check.
- [ ] `docs/development/specs/pocket-librarian-v1-spec.md` §5.4 and the `dir_kind`/classification sections updated to
      describe the library and the resolved mapping.
- [ ] `node scripts/check-kits.mjs` stays green if `kits/` or `kits.yaml` is touched.

## Gate & contract hygiene

Reuses the gate reconciliation done in `issue-body.md`'s `## Dependencies & gates`:

| Gate | Fires? | Why |
| --- | --- | --- |
| Governing dependency (parked) | Always, until unparked | Owner sign-off 2026-07-21; no other gate here authorizes starting early |
| Identity neutrality | Yes | Any content this build vendors/copies into `librarian/` or `plugin/` |
| Kit-manifest drift (`check-kits.mjs`) | Only if `kits/`/`kits.yaml` touched | Reading from or copying out of the vendored catalog |
| Librarian integration (`make verify`) | Yes | This plan's own AC adds a verify.sh check |
| Repo checks + unit tests | Always | Any change under `librarian/` or `plugin/` |
| ADR | Yes, hard precondition | Design questions explicitly unresolved |
| Spec drift (manual review) | Yes | No automated guard covers prose spec sections; spec section 5.4 + dir_kind must be amended in-change |
| Version sync, bundle drift, CHANGELOG, DB migration | No | Not implied by the current design shape; re-check once the ADR is written |

## Note on leanness

Deliberately thin, per the parked disposition: this plan carries the reconciliation, the unpark
condition, the mandatory first slice, and a source-grounded outline with verifiable acceptance and
gates — not full per-deliverable file-scope tables or a parallelism/landing-order matrix. That level
of build-readiness is a future pass once the ADR resolves the design questions and an owner actually
schedules the work.

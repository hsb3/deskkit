---
type: sop
status: final
created: 2026-05-12
updated: 2026-05-12
tags: [meta]
---

# SOP — engineering-spec

## When to write one

You're about to start an atomic unit of engineering work and want to be sure the team (including future-you) knows exactly what done looks like. **If you can't define done concretely, the spec isn't ready and you shouldn't start coding.**

**Granularity test**: a single engineering-spec should be implementable in one focused session — usually a day, occasionally a week, never a month.

- ❌ "Build the auth system" — too big
- ❌ "Add login flow" — still too big
- ✓ "Add password-reset email Function and wire it to the reset page" — right

If you're tempted to write a spec covering more than one merge-able unit, you're really writing a product-spec or a one-pager. Use that template instead and let the engineering-specs fall out of it.

## How to write one

1. **Start from the parent product-spec.** Add its wikilink to `parent_product_spec`. If no product-spec exists yet, write that first.
2. **Frame the title as an action**, not a feature: *"Add submit button to feedback page"*, not *"Submit button feature."*
3. **Fill *What it does NOT do* before *What it does*.** This single move prevents most blunt-spec failures. If the section feels short, you haven't thought hard enough.
4. **Make acceptance criteria observable.** Each should fit in <5 minutes of manual verification.
5. **Don't skip the definition-of-done checklist.** Each item must be ticked before status moves to `shipped`.
6. **Test plan is mandatory.** *"We'll write tests"* is not a test plan. Name the file or describe the coverage.
7. **Open questions block shipping.** Either resolve them inline or split into a decision. Don't ship with open questions.

## Status transitions

| From | To | Trigger |
|---|---|---|
| `draft` | `in-review` | Author requests review |
| `in-review` | `approved` | Reviewer signs off; no in-flight edits |
| `approved` | `building` | PR is opened |
| `building` | `shipped` | DoD fully checked; `definition_of_done: true`; deployed |
| any | `shelved` | Won't ship this cycle; reason noted in body |

## Anti-patterns

- Spec covers multiple merge-able units → split into per-merge specs
- Missing or empty *What it does NOT do* → not ready
- Vague acceptance criteria ("the button works") → unverifiable
- DoD checked off without actually doing the work → undermines the entire system
- No `parent_product_spec` link → orphaned; flag in notes
- Spec written after the work is done as documentation → defeats the purpose

## The PM Advisor stance

**Cut scope before adding it.** If reviewing a spec, look for what to remove first. Every line earns its place.

## Example

See `example.md` in this folder for a complete, real engineering-spec drawn from the Quillpad project.

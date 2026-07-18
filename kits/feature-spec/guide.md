---
type: sop
status: final
created: 2026-06-30
updated: 2026-06-30
tags: [meta]
---

# SOP — feature-spec

A **feature-spec** specs a single, named feature: one user story, the problem it solves, the proposed solution, how you'll know it worked, what it deliberately won't do, and the roads not taken.

## Altitude vs product-spec

This sits **one rung below `product-spec`**. A product-spec describes a whole product (or a major surface) and *decomposes into* several features; a feature-spec describes **one** of those features and is concrete enough to build directly. The test: if you can't name the feature in a few words, or if it has more than one user story, you're writing a product-spec — go up a level and link this one to it via `parent_product_spec`.

## When to write one

Write a feature-spec when a single capability is well-enough understood to scope but big enough that "just build it" would lose the intent. Triggers:

- A product-spec named this feature in its decomposition and it's coming up for build.
- A standalone request ("add inbox triage") that's clearly one feature, not a product.

**Don't** write one for a trivial change (an issue is enough) or for a whole product (that's a product-spec).

## How to write one

1. **Pin the user story first.** One user type, one capability, one benefit, in the `As a / I want / So that` form. If you need two stories, you have two features.
2. **State the problem, not the solution.** The gap today and why it's worth closing — tie it to the parent product-spec's goal.
3. **Describe the solution from the user's seat.** What they see and do; the happy path plus the one error path that matters. No implementation detail — that's for the engineering-spec this may decompose into.
4. **Make success criteria observable.** 3–5 statements someone else could check without you. "Correction rate under 40% on the first 50 captures" — not "extraction feels good."
5. **Cut scope explicitly.** Name the adjacent things a reader will assume are in, and put them out. Non-goals are what keep the feature one feature.
6. **Record alternatives.** The approaches you rejected and why — including "do nothing" when it was real. This is the memory that stops the debate reopening.

Set `author` (required). Link `parent_product_spec` when one exists, and set `priority` (P0–P3) when it's been triaged.

## Status transitions

| From | To | Trigger |
|---|---|---|
| `draft` | `in-review` | First complete pass shared for review |
| `in-review` | `approved` | Accepted; cleared to build |
| `approved` | `building` | Implementation underway |
| `building` | `shipped` | Live and meeting its success criteria |
| any | `shelved` | Deprioritized or abandoned; kept as record |

## Anti-patterns

- **Wrong altitude** — multiple user stories or a whole-product scope. Split it, or promote it to a product-spec and link back.
- **Solution as problem** — the Problem section describes the fix instead of the pain. State the gap first; the fix belongs under Proposed solution.
- **Unfalsifiable success criteria** — "works well," "users are happy." If no one but you can check it, it isn't a criterion.
- **Missing non-goals** — an empty Non-goals section on a real feature means the scope hasn't been cut yet; the feature will sprawl.
- **Empty alternatives** — "considered nothing" usually means the first idea was shipped unexamined. At minimum, weigh it against doing nothing.

## Example

See `example.md` in this folder for a worked feature-spec ("Auto-enrich capture metadata") under the Quillpad project.

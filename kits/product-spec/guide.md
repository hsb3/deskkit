---
type: sop
status: final
created: 2026-05-12
updated: 2026-05-12
tags: [meta]
---

# SOP — product-spec

## When to write one

You've approved a one-pager and now need to detail the features. A product-spec is feature-level: it describes one coherent user-facing capability from the perspective of the person using it.

**Granularity test:**
- ✓ "Inbox view with triage actions" — feature-level
- ✓ "Auto-enrich metadata in editor" — feature-level
- ✗ "Build the editor" — too big; this is actually 5 features (syntax highlighting, chat panel, diff view, autosave, version history)
- ✗ "Add a button" — too small; this is an implementation detail for an engineering-spec

**Timing:** Write product-specs after the one-pager is approved but *before* engineering starts. If engineering is already building, the spec is just documentation after-the-fact (defeats the purpose).

## How to write one

### Step 1: Start from the one-pager

What problem does your one-pager solve? What success metrics define "it worked"? Your product-spec must directly advance one of those metrics.

### Step 2: Frame the feature from the user's perspective

Write "what it does" as a narrative, not a spec. Put yourself in the user's shoes. What do they see? What do they click? What happens? Specific enough that an engineer can build it; narrative enough that a stakeholder understands the value.

### Step 3: Scope via "What it does NOT do"

This is the hardest section. Brainstorm all the related things you *could* do. Then explicitly exclude them. Why? Because v1 scope is limited and you need to make those trade-offs visible.

If this section feels empty, you haven't thought hard enough about boundaries.

### Step 4: Define success criteria

Success for a product-spec is: "did the user get the value this feature promised?" Not activity ("we shipped it"), but outcome.

Use the one-pager's metrics as a north star. Which of those does this feature move?

### Step 5: Document the interaction

Walk through the happy path. If there are error cases (network error, permission denied, etc.), name them. If they're complex, explain the recovery flow.

### Step 6: Call out design considerations

What layout choices or information architecture decisions does this feature require? Not mockups, but enough to guide the engineering-specs. If you're unsure, say "engineering to refine" — that's fine.

### Step 7: Decompose into engineering-specs

**The product ↔ engineering handoff contract:**

A product-spec says *what* the user should experience. An engineering-spec says *how* to build it. Multiple engineering-specs decompose a single product-spec.

When you decompose:
- Each engineering-spec is atomic (one merge, one test plan, one definition of done)
- Each engineering-spec references `parent_product_spec: [[this-feature]]` in frontmatter
- This spec lists all children in `decomposes_to` array

Example:
- **Product-spec:** "Inbox view with triage actions"
- **Engineering-specs:**
  - "Load inbox from Firestore and render stateful list"
  - "Add promote/defer/drop buttons; update document status"
  - "Display inbox grouped by status (triage/develop/polish/etc.)"

Don't write engineering-specs yourself unless you're also the engineer building it. Propose the decomposition, then hand off to engineering to refine.

## Status transitions

| From | To | Trigger |
|---|---|---|
| `draft` | `in-review` | Author requests review |
| `in-review` | `approved` | Reviewer + engineering lead sign off; decomposition locked |
| `approved` | `building` | Engineering-specs written and engineering starts work |
| `building` | `shipped` | All child engineering-specs are shipped |
| any | `shelved` | Feature deferred / won't ship this round |

## Anti-patterns

- Spec is actually 3 features bundled together → decompose and write separate product-specs
- "What it does NOT do" is empty → you haven't scoped
- No connection to parent one-pager → why are we building this?
- Success criteria are activity ("we shipped it") not outcome → how do you know it worked?
- Engineering-specs written before product-spec approved → putting the cart before the horse
- No decomposition to engineering-specs → product spec is orphaned; signal that engineering needs clarity

## The PM Advisor stance

**Cut scope before adding it.** If reviewing a product-spec, look for what to remove first. Every feature must earn its place against the one-pager's success metrics.

## Example

See `example.md` in this folder for a complete product-spec for Quillpad's "Inbox with attention groups" feature.

---
type: product-spec
status: draft
created: {{date}}
updated: {{date}}
tags: []
author:
decomposes_to: []
parent_one_pager:
---

# {{FEATURE_NAME}}

## Why this exists

One paragraph. What problem does this feature solve? How does it advance the parent one-pager's success metrics? Reference the parent one-pager explicitly.

## What it does

Narrative description of the feature from the user's perspective. "The user sees X, clicks Y, and Z happens." Concrete enough that a reader unfamiliar with the project can visualize it.

## What it does NOT do

Explicit scope cuts. Bulleted. Address the obvious adjacent concerns and explain why they're deferred.

## Success criteria

How does a product manager know this shipped successfully? Usually 3-5 statements.

- This happens
- That is visible
- This user flow completes

## Interactions

Step-by-step user journey (happy path + error path if complex).

## Design considerations

Visual, organizational, or information-architecture decisions. Not detailed mockups, but enough to guide the engineering-specs that decompose this.

## Related decisions

Are there any architectural or product decisions that shaped this feature? Cross-link them.

## Decomposition

**This feature decomposes into the following engineering-specs:**

- [[spec-1-name]] — atomic unit 1
- [[spec-2-name]] — atomic unit 2

(To be filled in after engineering review. The `decomposes_to` array in frontmatter should match these wikilinks.)

## Open questions

Anything undecided that blocks writing the decomposed engineering-specs.

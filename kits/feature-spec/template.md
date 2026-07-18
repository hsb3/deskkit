---
type: feature-spec
status: draft
created: {{date}}
updated: {{date}}
tags: []
author:
parent_product_spec:
priority:
---

# {Feature name}

_A spec for **one focused feature** — the altitude below a product-spec. It scopes a single capability the user can name, not a whole product. If you're describing more than one feature, you're at the wrong altitude: write (or link) a product-spec and decompose._

## User story

One sentence in the canonical form. Keep it to a single capability and a single benefit.

> **As a** {user type}
> **I want** {capability}
> **So that** {benefit}

## Problem

What's wrong or missing today, concretely. The gap this feature closes and why it's worth closing now — not implementation, just the pain. Reference the parent product-spec's goal this serves.

## Proposed solution

What the user sees and does. "The user does X, the system responds with Y." Concrete enough to picture without a mockup, narrow enough that the whole thing is clearly one feature. Lead with the happy path; note the one error path that matters.

## Success criteria

How you'll know it shipped and works. 3–5 observable, checkable statements — not aspirations.

- [ ] {Observable outcome 1}
- [ ] {Observable outcome 2}
- [ ] {Observable outcome 3}

## Non-goals

Explicit scope cuts. The adjacent things a reader will assume are included — name them and say they're out, so the feature stays one feature.

- {Deferred concern} — {why it's out of scope here}

## Alternatives considered

The other ways this could have been built, and why this one won. One short block per alternative; "do nothing" counts when it's a real option.

- **{Alternative}** — {approach in a line}. Rejected because {reason}.

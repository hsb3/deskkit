---
type: technical-design
status: draft
created: {{date}}
updated: {{date}}
tags: []
author:
priority: P1
affects_workstreams: []
superseded_by:
---

# {TDD title — system-scale, e.g. "Data model for multi-tenant content storage"}

## Problem context

One paragraph. What existing limitation or new requirement drives this design? Link to the parent product-spec(s) or one-pager that prompted this work. If this TDD exists to resolve architectural uncertainty across multiple engineering-specs, state that explicitly.

## Goals & non-goals

### Goals

High-level technical objectives. What does this system-scale design accomplish?

- Goal 1
- Goal 2

### Non-goals

What is explicitly out of scope? What is deferred? Why?

- Non-goal 1
- Non-goal 2

## Proposed design

### High-level overview

Prose description of the overall system. Major components, their interactions, data flows. Include a diagram placeholder if helpful (ASCII art, or note "See diagram at [[link]] for system topology").

### Data model

Entities, schema, relationships. If modifying existing collections, document migration strategy.

### API & contracts

New endpoints or service contracts that downstream code depends on. Specify request/response shapes.

### Alternatives considered

For each major decision, briefly document:
- Option A: [description] — Trade-off: [one con]
- Option B: [description] — Trade-off: [one con]
- Chosen: [which + why in one sentence]

## Infrastructure & deployment

Changes to services, databases, configuration. Rollout constraints or dependencies.

## Risks & mitigation

- **Risk**: [description] → **Mitigation**: [approach]

## Open questions

Anything not yet decided that blocks downstream work. Proposed resolution.


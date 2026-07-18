---
type: engineering-spec
status: draft
created: {{date}}
updated: {{date}}
tags: []
author:
parent_product_spec:
priority: P1
definition_of_done: false
test_plan_link:
---

# {Spec title — atomic, action-shaped, e.g. "Add submit button to feedback page"}

## Why this exists

One paragraph. What problem does this unit of work solve? Reference the parent product-spec.

## What it does

Declarative description of behavior. Atomic. Concrete. No "should/might."

## What it does NOT do

Explicit scope cuts. Bulleted. Address the obvious adjacent concerns. **This section is mandatory — if it feels short, you haven't thought hard enough.**

## Acceptance criteria

A friend should be able to verify each criterion in <5 minutes:

- [ ] Criterion 1
- [ ] Criterion 2

## Definition of done

- [ ] All acceptance criteria met
- [ ] Unit tests added/updated and passing
- [ ] Integration test covers the happy path
- [ ] Documentation updated where touched
- [ ] No new lint or type errors introduced
- [ ] PR reviewed and approved
- [ ] Deployed to <env>

When every box is checked, flip `definition_of_done: true` in frontmatter and move status to `shipped`.

## Test plan

- **Manual**: <list manual test steps>
- **Automated**: <reference test file(s) or describe coverage>
- **Edge**: <known edge cases covered>

## Interaction

Step-by-step user / system flow. Include error paths.

## States

Every state and what triggers transitions.

## Edge cases

Bulleted. Each names a specific scenario and intended behavior.

## Tech notes

Components, libraries, APIs, data shapes — enough that an engineer can start. Not full code.

## Dependencies

- **Upstream**: <other specs / decisions / external blockers>
- **Downstream**: <what this enables>

## Open questions

Anything undecided + proposed resolution path or deadline. **Open questions block `shipped` status.**

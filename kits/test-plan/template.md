---
type: test-plan
status: draft
created: {{date}}
updated: {{date}}
tags: []
author:
priority:
parent_engineering_spec:
---

# Test Plan: {Feature / spec under test}

_How we prove the work is correct, not what we're building. Pairs with an engineering-spec (the WHAT) — link it in `parent_engineering_spec`. One test plan per spec; update it in place as coverage fills in._

## Scope

**In scope** — what this plan verifies:

- {Behavior / surface 1}
- {Behavior / surface 2}

**Out of scope** — explicitly not tested here, and why:

- {Adjacent concern} — {covered elsewhere / deferred / not a risk}

## Approach

How the coverage is layered. Name the level, the tool, and what it owns.

| Level | Tool | What it covers |
|-------|------|----------------|
| Unit | {framework} | {pure logic, edge branches} |
| Integration | {framework} | {component + dependency, happy path + key failures} |
| E2E | {framework} | {full user flow through the real surface} |
| Manual | — | {what only a human can judge: feel, visual, exploratory} |

## Test cases

| ID | Scenario | Steps | Expected |
|----|----------|-------|----------|
| TC-001 | {What this case proves} | {1. … 2. … 3. …} | {Observable result} |
| TC-002 | {…} | {…} | {…} |

## Coverage matrix

Every acceptance criterion in the parent spec maps to at least one test case. A blank cell is a coverage gap.

| Requirement (from spec) | Test case(s) | Level |
|-------------------------|--------------|-------|
| {Acceptance criterion 1} | TC-001 | unit |
| {Acceptance criterion 2} | TC-002 | e2e |

## Environments

Where each level runs and what it needs.

| Environment | Used for | Data / setup |
|-------------|----------|--------------|
| {local / CI / staging} | {levels} | {fixtures, seed, keys} |

## Exit / acceptance criteria

This plan passes — and the spec is provable — when:

- [ ] Every requirement in the coverage matrix has a passing test case
- [ ] All automated suites green in CI
- [ ] Manual checklist completed and signed off
- [ ] No open P0/P1 defects against the feature
- [ ] {Any feature-specific gate, e.g. measured error rate below threshold}

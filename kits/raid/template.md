---
type: raid
status: draft
created: {{date}}
updated: {{date}}
tags: []
author:
owner:
---

# RAID Log: {Project name}

_The living register of project uncertainty: **R**isks · **A**ssumptions · **I**ssues · **D**ependencies. One row per item; keep IDs stable once assigned; update `status` in place rather than deleting rows (a closed risk is still useful history)._

## Risks

_Something that **might** happen and would hurt if it did. Probability × Impact drives how much attention it gets — and every risk needs a mitigation, or it's just worry._

| ID | Risk | Probability | Impact | Mitigation | Owner | Status |
|----|------|-------------|--------|------------|-------|--------|
| R-001 | {What could go wrong, and why it bites} | high / medium / low | critical / significant / minor | {How we cut the likelihood or the blast radius} | {who watches it} | open |

## Assumptions

_Something we're **treating as true** but haven't proven. Each needs a validation plan; a broken assumption usually becomes a Risk or an Issue._

| ID | Assumption | Confidence | Validation plan | Status |
|----|-----------|------------|-----------------|--------|
| A-001 | {What we're betting is true} | high / medium / low | {How and when we'll confirm it} | unvalidated |

## Issues

_Something that has **already gone wrong** and needs resolving now. (An Issue has happened; a Risk hasn't yet.)_

| ID | Issue | Type | Priority | Resolution | Status |
|----|-------|------|----------|------------|--------|
| I-001 | {The problem, concretely} | design gap / tech debt / blocker | P0 / P1 / P2 / P3 | {The plan to resolve, or the decision needed} | open |

## Dependencies

_Something **outside our control** that we need before we can proceed. Track who provides it and what breaks if it's late._

| ID | Dependency | Type | Required by | Provider | Status | Risk if late |
|----|-----------|------|-------------|----------|--------|-------------|
| D-001 | {What we're waiting on} | infra / data / package / decision | {what it unblocks} | {who delivers it} | not started | {what breaks if it slips} |

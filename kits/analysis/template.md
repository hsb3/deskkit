---
type: analysis
status: draft
created: {{date}}
updated: {{date}}
tags: []
author:
conclusion:
---

# Analysis: {The question, framed as a choice}

_An options-evaluation: investigate a question, lay out the live options with their trade-offs, compare them on shared criteria, and land on a recommendation. This is the **analytical front-half that precedes a decision** — analysis weighs the options; the [[_headcase/agents/pm/resources/decision]] (ADR) records the one chosen and locks the rationale. Keep this doc about the weighing; move the verdict into a `decision` note once the call is made._

## Question / Context

What are we trying to decide, and why does it matter now? State the question as a genuine fork — if one option has no merit, you haven't framed it honestly yet. Note who cares about the outcome and what's blocked until it's resolved.

## Current state

How things work today, concretely. What forces this question — a constraint, a stalled merge, a cost, a new requirement? Include the relevant facts an option has to satisfy.

### Metrics _(if applicable)_

- {Metric}: {current value}
- {Metric}: {current value}

## Options

### Option A: {name}

**Description:** How this approach would work.

**Pros:**
- {Advantage}
- {Advantage}

**Cons:**
- {Drawback}
- {Drawback}

**Effort:** {rough size — hours / days / weeks}
**Risk:** low / medium / high — {one line on what the risk is}

### Option B: {name}

**Description:** How this approach would work.

**Pros:**
- {Advantage}
- {Advantage}

**Cons:**
- {Drawback}
- {Drawback}

**Effort:** {rough size}
**Risk:** low / medium / high — {what the risk is}

### Option C: {name}

**Description:** How this approach would work.

**Pros:**
- {Advantage}
- {Advantage}

**Cons:**
- {Drawback}
- {Drawback}

**Effort:** {rough size}
**Risk:** low / medium / high — {what the risk is}

## Comparison matrix

_Score every option on the same criteria. Pick criteria that actually discriminate — drop any row where all options land the same._

| Criterion | Option A | Option B | Option C |
|-----------|----------|----------|----------|
| {Decision driver, e.g. fit to current state} | | | |
| Effort | | | |
| Risk | | | |
| {Maintenance / longevity} | | | |
| {Reversibility} | | | |

## Recommendation

**Preferred: Option {A/B/C}.**

One clear paragraph: which option, and the reasoning that the matrix above supports.

**Trade-offs accepted:**
- {What we give up by choosing this}
- {Why that's acceptable}

**Next step:** record the call in a [[_headcase/agents/pm/resources/decision]] note (this analysis becomes its `Options Considered` backing) and link the two.

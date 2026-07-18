---
type: sop
status: final
created: 2026-05-12
updated: 2026-05-12
tags: [meta]
---

# SOP — one-pager

## When to write one

Before you spend more than 2 days on a new idea, write a one-pager. It forces you to articulate the problem, bound the scope, and lock down success criteria. Don't estimate implementation; just verify the idea is worth building at all.

**Red flag:** If you can't fill a one-pager in 2 hours, the idea isn't ready. Send it back to research.

## How to write one

**The PM Advisor Stance:** Default action is CUT SCOPE.

### Step 1: Lock down the problem

- **What problem:** Not "users want X." Why do they want it? What breaks without it?
- **Who has it:** Name the user type. "Developers" is not specific. "Engineers writing metadata extraction pipelines" is.
- **Evidence:** Did a customer ask? Did you see data? Did something break? Don't invent problems.

### Step 2: Frame why now

This is harder than it sounds. "It's a good idea" is not a why-now. What changes if you ship this in 3 months instead of now? Is a competitor building it? Does a key customer leave without it? Does a technical debt bomb get worse?

If you can't articulate a time pressure, the one-pager goes in the backlog. That's fine. Just don't ship it now.

### Step 3: Scope ruthlessly

**Default stance: Cut everything that's not essential to solving the core problem.**

- **Must Have:** What's the minimum set of features that unblocks the problem?
- **Won't Do:** What's the obvious adjacent thing you're explicitly *not* doing? This section must be non-empty. If it feels short, you haven't scoped hard enough.

Effort estimates are rough (person-weeks, swag). The point is to sanity-check scope. If "must have" is >6 weeks, you're trying to solve two problems at once. Go back and cut.

### Step 4: Define success

Success metrics are how you'll know this shipped value. They're not activity metrics ("we shipped it") or vanity metrics ("100 users signed up"). They're outcomes.

- **Baseline:** Where are we today? (optional — sometimes there's no baseline)
- **Target:** What does "it worked" look like?
- **Measured when:** When can you verify this? Don't pick a date 6 months out.

Table format forces specificity. If a metric is vague, you haven't thought about how to measure it.

### Step 5: Call out risks

Not everything goes to plan. Name the things that could derail the ship. For each, assign probability (high/medium/low) and impact (critical/significant/minor). Then — and this is key — *propose a mitigation*. Don't just list risks; show you've thought about how to dodge them.

### Step 6: Nail down ownership and timeline

- **Target ship date:** When does this need to be done? Make it real. "End of quarter" is vague. "2026-06-15" is clear.
- **Total effort:** Roll up the "effort weeks" column from Must Have. This is your commitment.
- **Milestones:** 2-3 anchors that mark progress. These aren't ceremony; they're check-in gates. If you're not on track to hit one, you have time to re-scope.
- **Decisions:** What unknowns block the build? Who decides, and by when? Surface these so decisions don't become bottlenecks.

## Status transitions

| From | To | Trigger |
|---|---|---|
| `draft` | `in-review` | Author requests review |
| `in-review` | `approved` | Reviewer signs off; scope locked |
| `approved` | `building` | Engineering starts work; becomes product-spec(s) |
| `building` | `shipped` | All features shipped; success metrics hit |
| any | `shelved` | Won't ship; reason noted in body |

## Anti-patterns

- Everything is marked "must have" → you haven't prioritized
- No "won't do" section → you haven't scoped
- Effort totals >6 weeks → you're trying to solve two problems
- No success metrics, or they're all activity ("we shipped it") → how do you know it worked?
- Timeline is 3+ months → you need to break this into phases
- One person decided in a meeting; doc written 2 weeks later → details are stale, ownership is unclear

## Background principles (from PM Advisor)

The one-pager enforces five key constraints:

1. **Working Product > Activity:** Success metrics must measure outcomes, not activity.
2. **Simplicity:** Default: cut scope. Everything in the one-pager earns its place by being essential to the core problem.
3. **Make Work Visible:** No verbal decisions. Everything is written, so a new reader six months later can understand why.
4. **Fixed Time, Variable Scope:** Time box (target ship date) is fixed. If you can't fit it, cut features, not time.
5. **Evolve, Don't Transform:** Ship the smallest version that solves the core problem. Improve on evidence.

## Example

See `example.md` in this folder for a complete one-pager for the Quillpad project.

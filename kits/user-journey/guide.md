---
type: sop
status: final
created: 2026-05-12
updated: 2026-05-12
tags: [meta]
---

# SOP — user-journey

## When to write one

A feature or product area involves 3+ distinct steps or screens, and the overall flow feels clunky or you want to understand where the user gets stuck. **Journeys are flow-diagnostic tools; use them before specifying features when the happy path feels murky.**

**Not a user-journey if:**
- It's a single screen or decision (spec that instead)
- The flow is already crystal clear (skip the analysis)
- You're mapping the entire product end-to-end (too big; break into journeys)

**Is a user-journey if:**
- You're designing "author writes → editor collaborates → publishes" and want to see where collaboration breaks
- You suspect a feature causes customer friction and want to map it to pinpoint the problem
- Multiple product specs converge on the same user flow and you want a single source of truth

## How to write one

1. **Pick one persona and one goal.** Not "all users using everything." Pick Robin publishing a note, or an assistant reviewing inbox items. Narrowness is power.
2. **List the steps they go through.** 5-10 steps is normal. If you exceed 12, break the journey into multiple flows.
3. **For each step, capture four things**: (a) what the user does, (b) what they see, (c) how they feel, (d) where they get stuck.
4. **Keep emotional state honest.** Not "delighted"; more like "anxious about losing draft" or "satisfied with publication." Real emotions guide design.
5. **Name pain points and opportunities separately.** Pain is what breaks flow; opportunity is what would delight or accelerate.
6. **Link to specs that own each touchpoint.** Not a exhaustive map, but if the journey references feature F042, link [[F042-related-spec]].
7. **Don't speculate on fixes.** If a step feels bad, describe it as bad. Let spec work define the solution.

## Status transitions

| From | To | Trigger |
|---|---|---|
| `draft` | `in-review` | Author requests feedback |
| `in-review` | `approved` | Reviewer confirms journey matches reality |
| `approved` | `archived` | Specs derived from journey are shipped, or journey changes significantly |

## Anti-patterns

- Journey is 15 steps long with three sub-flows — break it up
- Emotional state is aspirational ("user feels delighted") instead of realistic
- Pain points have no specifics ("the UI is confusing") — name the exact moment
- No links to specs — journey is orphaned from the design work it should inform
- Journey documented after specs are written as post-hoc explanation — defeats the purpose

## The PM Advisor stance

**Cut the journey to the essentials.** If step 7 doesn't reveal something important about the flow, delete it. A 5-step journey that clearly maps to pain is better than a 10-step chronicle of everything the user might do.

## Example

See `example.md` in this folder for a complete user-journey on Quillpad author publishing.

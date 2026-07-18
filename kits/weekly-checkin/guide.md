---
type: sop
status: final
created: 2026-05-12
updated: 2026-05-12
tags: [meta]
---

# SOP — weekly-checkin

## When to write one

Every Friday afternoon (or whenever your work week ends). Takes 15 minutes. The purpose is to make visible what shipped, what's stuck, and what's next — so you don't have to recreate that context on Monday morning.

Also: forces you to articulate blockers early, so you can resolve them before Monday instead of discovering them mid-week.

**Cadence:** Weekly. Same time. Non-negotiable. It's the only forcing function preventing scope drift.

## How to write one

### Step 1: Fill "Shipped This Week"

What shipped? What's production? What did the team see? Be specific. Not "worked on feature X" but "Shipped [[focus-mode-engineering-spec]]; deployed to production Friday EOD."

If nothing shipped, say so. It's not a failure; it's just information. But if week after week shows nothing shipped, something is blocking and it needs air.

### Step 2: Fill "In Progress"

Anything you started but didn't finish. For each:
- **Status:** `on_track` (finishing this week or next), `at_risk` (slipping, might not finish), `blocked` (can't proceed without external input)
- **Blocker:** If status is `at_risk` or `blocked`, what's the thing stopping you? Be specific.
- **Weeks in progress:** Count how many weeks this item has been in progress. **Flag rule:** if any item has been in progress >2 weeks, add a note calling it out. Something is wrong (too big, or actual blocker).

### Step 3: Fill "Next Week"

What are you committing to next week? Should line up with `in_progress` items +something new. Be realistic. Better to ship something than plan overambitious.

### Step 4: Surface "Decisions Needed"

What decision blocks work next week? Who decides? When? Don't let decisions silently slip.

### Step 5: 30-second retro

**What went well:** One thing. Could be "shipped on time" or "figured out a hard problem" or "had a great collab." Celebrate it.

**What didn't:** One thing. Did a task run long? Did a decision block you? Did something go wrong? Name it so you don't repeat it.

**Action item:** Specific, with owner. Not "improve testing" but "Robin writes 2 unit tests for the auth module by Wednesday." If you don't have a specific action, the retro doesn't land.

## Governance rules

### The "flag items in progress > 2 weeks" rule

If any item in the "In Progress" section has been in progress >2 weeks, add a note. Example:

```
| Rewrite auth module | ... | at_risk | Waiting on design spec from UX | 3 weeks |
```

*Note: Auth module has been in-flight 3 weeks. Blocker is design spec. Robin to follow up with UX by Monday EOD.*

This rule surfaces bottlenecks early so they don't quietly drag on for a month.

### Shipping is the unit of work

"In progress" means really in progress (not planned, not prepped, but actively being built). If something is waiting for you to start, it belongs in "Next Week," not "In Progress."

### One-pager links

If work is tracked against a one-pager, link it. This keeps the one-pager and weekly reality connected. If you're working on something and there's no one-pager, that's a signal the work might be unscoped.

## Status values

- `on_track`: finishing this week or next week without known blockers
- `at_risk`: timeline slipping; will need more time or scope cuts
- `blocked`: can't proceed without external decision / input / dependency

## Anti-patterns

- Nothing ever ships → something is fundamentally broken
- Everything is marked "on track" → you're not being honest about risk
- "Decisions needed" list is empty → you're not surfacing blockers
- Retro action item is vague ("improve testing") → won't stick
- Weekly check-in is 2 weeks old before you write it → defeats the purpose (do Friday EOD)
- Item shows "10 weeks in progress" → this isn't a blocking issue, it's a scope issue; should have been cut or split into smaller chunks

## The PM Advisor stance

**Default action: resolve blockers, don't explain them away.** If something is blocked >1 week, it's either a false blocker (you can work around it) or a real decision needed (make it).

## Variants

**status-report (`audience: client`)** — the same shape, pointed outward. A *status-report* is the client- or stakeholder-facing version of the weekly check-in: what shipped, what's next, risks and asks. Use the same sections, but write for an external reader — no internal shorthand, and frame each blocker as a decision you need *from them*. Set `audience: client` in the frontmatter. It's a qualifier of this SOP, not a separate one.

## Example

See `example.md` in this folder for a complete weekly check-in from a real week on Quillpad.

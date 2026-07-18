---
type: sop
status: final
created: 2026-05-12
updated: 2026-05-12
tags: [meta]
---

# SOP — retro

## When to run one

After a milestone or phase of at least 2 weeks of sustained work. Never less than 2 weeks (single retro on a 3-day sprint is noise; multiple retros per week is ceremony overload).

**Triggers:**
- End of a planned phase (one-pager delivered)
- After a crisis or incident worth post-mortiming
- Natural project milestone (alpha launch, public ship)
- Quarterly (if no phases that natural)

**Not a trigger:** Monday morning re-grouping (that's a standup, not a retro)

## How to facilitate (solo author)

If you're solo (like Robin), the retro is reflective writing, not a meeting.

### Step 1: Gather data

Spend 30 minutes looking back. Read the last 4-8 weekly check-ins. Look at your git log / commit count. Any PRs rejected? Dependencies that surprised you? Decisions you'd reverse?

### Step 2: Fill "What Went Well"

Write down 2-3 patterns that worked. Not "we shipped on time" (activity); but *why* it happened. Examples:

- "Friday shipping gate forced end-of-week clarity. Blockers surfaced early."
- "Pair programming on the tricky auth module surfaced edge cases 2 days earlier than solo review would have."
- "Cutting scope hard on day 3 kept us honest and prevented feature creep."

### Step 3: Fill "What Didn't Work"

2-3 things that were friction. Be honest. Not "testing was hard" but specific: "Manual testing of the email flow was tedious; we caught bugs late that unit tests should have caught first."

### Step 4: Root-cause the big ones

For the 1-2 largest problems, ask why 3 times. Don't stop at the surface.

**Example:**
- Problem: "Deploy process broke twice; both times a secret was misconfigured in GitHub Actions."
- Why 1: "Secrets were documented in an ad-hoc Notion page."
- Why 2: "No standard for where secrets live or how they're validated."
- Why 3: "No pre-deploy checklist; Robin just ran the script and hoped."
- Root cause: "No single source of truth for deployment prerequisites."

### Step 5: Write action items

For each root cause, propose one specific action. Assign it (usually yourself). Add a deadline (usually next sprint start).

Action items are not vague. Not "improve secrets management" but "Robin creates `/infra/SECRETS.md` documenting all required GitHub Actions secrets, their values, and a pre-deploy checklist by 2026-05-31."

### Step 6: Metrics (if relevant)

Did you track anything? Commit count, features shipped, velocity, blockers per week? Include it.

### Step 7: Summarize learnings

One paragraph. "Next sprint, here's what we'll do differently." Make it concrete.

## Status

- `draft`: You're still writing
- `published`: Complete; locked in

## Anti-patterns

- Retro is > 1 week after the period ended → memory is stale
- "What went well" lists are full of vague wins ("great teamwork") → not actionable; think deeper
- Root causes are one level deep ("we didn't test enough") → keep asking why
- Action items are unassigned or vague → won't happen
- Retro findings are never referenced again → just ceremony, not learning

## Solo retro discipline

Because you're not in a meeting with teammates, it's easy to skip the hard parts (root cause, action items). Fight that. Especially:

1. **Be honest about what didn't work.** You're not in a group where admitting a mistake feels risky. Use that freedom.
2. **Dig into root cause.** Three "whys" minimum. Symptoms are easy; causes are where learning lives.
3. **Assign yourself an action.** Write it down with a deadline. Tell someone (or put it in Obsidian as a task link).

## Examples

See `example.md` in this folder for a complete retro after Phase 1 of Quillpad.

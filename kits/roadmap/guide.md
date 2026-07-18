---
type: sop
status: final
created: 2026-06-30
updated: 2026-06-30
tags: [meta]
---

# SOP — roadmap

## When to write one

Write a roadmap once a project has enough committed work that **order matters** — when there are several milestones and the question "what has to be true before we start X?" has a real answer. One roadmap per project (or product line); it's a **living document** that re-sequences as reality lands, not a one-shot plan.

**Red flag:** if your roadmap reads like a calendar — week-by-week rows, target dates per item — you've written a schedule, not a roadmap. A roadmap answers *what unblocks what*; a schedule answers *when*, and dates are the first thing reality breaks.

## How to write one

A roadmap has three layers, from intent down to readiness. Keep them distinct.

| Layer | Answers | The test |
|---|---|---|
| **Vision / outcome** | Where are we headed? | "When this is done, the user can finally ___." |
| **Themes** | What lines of value group the work? | Several milestones share one theme's "why." |
| **Sequenced phases** | What order, and why that order? | Each phase names what it **unblocks**. |

### Vision / outcome
One or two sentences, outcome-framed — what becomes true for the user, not what you'll build. Every theme and phase below should ladder up to it; if one doesn't, it belongs on another roadmap or out of scope.

### Themes
The 2–5 buckets the work clusters into. A theme is a coherent line of value (e.g. "trustworthy capture"), not a feature. Themes give the reader the *why* that several milestones share, and they usually run in parallel — so order them by priority but don't sequence them like phases.

### Sequenced phases
The backbone. Each phase carries a **goal** (the one outcome that defines it), the **theme(s)** it serves, what it **depends on** (the earlier goal that must be true first), and what it **unblocks** (what its completion releases). Express the edges of the dependency graph — "Phase 2 can't start until Phase 1's goal is met" — never "Phase 2 starts on the 14th." A phase is done when its goal is demonstrably met; that proof, not a date, is what releases the next phase.

### Now / Next / Later
The flat view, bucketed by **readiness to start**: *Now* is unblocked and in flight, *Next* is unblocked the moment Now lands, *Later* is real work still waiting on earlier work or an open decision. This is the table people actually check between reviews — keep it honest and keep it dateless.

### Out of scope
Name the adjacent work you're deliberately not covering, and why. A roadmap with nothing out of scope hasn't drawn its boundary; the section must be non-empty.

## Status transitions

| From | To | Trigger |
|---|---|---|
| `draft` | `in-review` | First real sequence shared for review |
| `in-review` | `approved` | Sequence accepted as the plan of record |
| any | `archived` | Project closed or roadmap superseded; kept as record |

## Anti-patterns

- **Dates as commitments** — putting target dates, week numbers, or a Gantt on the roadmap. This is the cardinal sin: it converts a sequence into a schedule, and the schedule is wrong the day after you write it. Commit to *order* and *unblocking*, not *when*. (Owner-imposed deadlines, when they exist, are recorded as constraints elsewhere — not baked into the roadmap as if the sequence implies them.)
- **Phases with no "unblocks"** — if a phase doesn't release anything, it isn't load-bearing in the sequence; either it's parallel theme-work or it's filler.
- **Themes that are really features** — a theme is a line of value, not a single deliverable. "Search" is a feature; "the reader can find anything they captured" is a theme.
- **Everything in Now** — if Now holds the whole backlog, you haven't sequenced by readiness; most of it is blocked and belongs in Next/Later.
- **Empty Out of scope** — no boundary means the roadmap will absorb every adjacent idea and stall.
- **Write-once** — a roadmap that never re-sequences after the first milestone slips is dead. It lives or it's worthless.

## Example

See `example.md` in this folder for a worked roadmap for the Quillpad project.

---
type: sop
status: final
created: 2026-05-12
updated: 2026-05-12
tags: [meta]
---

# SOP — research-synthesis

## When to write one

You need to understand an unknown or validate an assumption. You've done the reading / analysis / experiment, and you're writing down what you learned and why it matters.

**Not a research-synthesis if:**
- You're just documenting how to do something (that's a guide or runbook)
- You're proposing a change (that's a decision or one-pager)
- The answer was obvious and took <30 minutes to validate (document inline in the decision that needed it)

## How to write one

1. **Frame the question clearly.** One or two sentences. What gap were you filling?
2. **List your sources.** Be specific enough that someone could follow the same trail. Include search terms if you did research.
3. **Describe your methodology.** What lens did you apply? If comparing options, state the criteria. If reading papers, say what you were looking for.
4. **State findings factually.** No hedging. You either found something or you didn't. If findings are weak or inconclusive, say so plainly.
5. **Force yourself to write implications.** This is the hard part. "So what?" For each finding, what changes? Is this a blocker? A nice-to-have? Does it inform a decision, or just add color?
6. **Cut ruthlessly.** If a finding doesn't lead to an implication, it probably doesn't belong. Synthesis is not accumulation.

## Status transitions

| From | To | Trigger |
|---|---|---|
| `draft` | `in-review` | Author requests review |
| `in-review` | `approved` | Reviewer confirms findings and implications are sound |
| `approved` | `archived` | Decision it informed is shipped, or finding becomes stale |

## Anti-patterns

- Researcher's bias toward "both sides are valid" — if conclusions are weak, state that; don't pad
- Implications section is missing — synthesis without implications is just a summary
- Sources are vague ("I read some papers") — cite specifically
- Findings exceed the question scope ("I was asked about X, but here's my 2,000-word take on Y")
- Synthesis written and never used — if no decision is informed by it, consider whether it's worth keeping

## The PM Advisor stance

**Kill research before starting it.** Before commissioning a synthesis: what decision does it inform? If you can't name one, the research is premature. During review: if implications are "we should think about this more," the synthesis isn't done. Push back on open-ended exploration; synthesis is targeted.

## Example

See `example.md` in this folder for a complete research-synthesis on AIPMO template precedents.

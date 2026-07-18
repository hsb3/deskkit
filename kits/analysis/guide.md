---
type: sop
status: final
created: 2026-06-30
updated: 2026-06-30
tags: [meta]
---

# SOP — analysis

## When to write one

You have a real question with more than one defensible answer, and you want to do the *weighing* before you commit to the *call*. An analysis is the structured investigation — options, trade-offs, a comparison on shared criteria, a recommendation — that earns a clean decision.

**Analysis precedes a decision.** The two are a pair:

- An **analysis** lays out options A/B/C and weighs them. Its `status` lives in the reference family (it's a study, not a verdict). It can sit in `in-review` while people argue the matrix.
- A **[[decision]]** (ADR) records the *one* option chosen and locks the rationale so nobody re-litigates it in six months. Its `status` is `proposed → accepted`.

Write the analysis first; when the call is made, open a `decision` note, set its `Options Considered` from this doc, and link the two. The analysis is the backing; the decision is the binding.

**Criteria:**
- Multiple options are genuinely defensible (if one is obviously right, just decide).
- The choice is worth the cost of writing it down — it shapes architecture, data model, vendor, or product direction.
- You want the reasoning legible to others (or future-you) *before* committing.

**Not an analysis:**
- A choice with one real answer — that's just a [[decision]], skip the options pass.
- A daily implementation pick resolved at standup.
- A pure preference with no project-wide ripple.

## How to write one

1. **Frame the question as a genuine fork.** Same discipline as a decision: if Option C has zero merit, you haven't found the real options. Every option should be defensible enough that a reasonable person could pick it.
2. **Ground it in the current state.** Options are only as good as the facts they have to satisfy — the constraint, the stalled merge, the cost, the new requirement. State those first so the trade-offs are anchored, not abstract.
3. **Give every option the same four fields** — description, pros, cons, **effort**, **risk**. Effort and risk are what stop an analysis from being a popularity contest.
4. **Make the matrix discriminate.** Pick criteria where the options actually differ. A row where all three score "high" is noise — cut it. Common drivers: fit to current state, effort, risk, maintenance, reversibility.
5. **Recommend, don't hedge.** End with a single preferred option and the trade-offs you're accepting by choosing it. An analysis that won't commit to a lean is half-done — name your pick and let the decision note ratify it.
6. **Set `conclusion:` in frontmatter** to a one-line version of the recommendation, so the call is readable without opening the body.

## Status transitions

The frontmatter `status` tracks the analysis as a study (reference family), *not* the decision it feeds:

| From | To | Trigger |
|---|---|---|
| `draft` | `in-review` | Options + matrix are complete enough to share for argument |
| `in-review` | `approved` | The recommendation is accepted as the basis for the decision |
| any | `archived` | Superseded, or the question went away |

Note the split: this doc reaching `approved` means "the *analysis* is sound." The chosen option becoming binding is a separate `decision` note moving `proposed → accepted`. Don't overload this status with the decision's lifecycle.

## Anti-patterns

- **The sales pitch** — one strong option and two strawmen. If the alternatives aren't defensible, the analysis is theater; do the work to find real ones.
- **No effort/risk** — pros-and-cons with no sizing. Without effort and risk, every option looks equally good and the matrix decides nothing.
- **A matrix that doesn't discriminate** — rows where all options score the same. They pad the table and hide the real drivers.
- **The non-recommendation** — laying out options and refusing to lean. The point of the front-half is to *tee up* the decision; an analysis with no preferred option hands the work back.
- **Analysis that swallows the decision** — recording the final verdict and consequences here instead of in a [[decision]] note. Weigh here; bind there. Keep the two distinct so the ADR record stays clean.

## Example

See `example.md` in this folder for a worked analysis — choosing the canonical content model for the Quillpad project (the study behind RAID item I-001, which it then routes into a `decision` note).

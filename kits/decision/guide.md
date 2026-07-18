---
type: sop
status: final
created: 2026-05-12
updated: 2026-05-12
tags: [meta]
---

# SOP — decision

## When to write one

You're at a fork in the road and the path chosen will ripple across the project. A decision doc locks down the reasoning so that (a) the team understands why, and (b) future-you doesn't re-litigate the same trade-off in six months.

**Criteria:**
- The choice affects architecture, tech stack, vendor selection, or a product direction
- Multiple defensible options exist
- The decision is "closed" — you've talked to stakeholders and made the call
- It's not urgent (if you need to ship by tomorrow, just decide and document it later)

**Not a decision:**
- Daily implementation choices ("should the button be on the left or right?")
- Resolved at standup ("we'll use postgres")
- Pure engineering preference with no project-wide impact

## How to write one

1. **Frame it as a genuine fork, not a sales pitch.** If one option has zero merit, you haven't thought hard enough. All options should be defensible.
2. **Enumerate trade-offs explicitly.** Every pro for Option A is a con for Option B. Say it.
3. **Name the deciders.** Who signed off? Usually 1-3 people, not the whole team. Add `decided_by: [name]` to frontmatter.
4. **Explain the "why now" upfront.** What changed to make this decision ripe today?
5. **Document consequences.** What becomes a hard constraint downstream? What's now ruled out?
6. **Link to affected artifacts.** If this decision shapes a product-spec or engineering-spec, add wikilinks in the notes.

## Status transitions

| From | To | Trigger |
|---|---|---|
| `proposed` | `accepted` | Deciders sign off; rationale locked |
| `accepted` | `superseded` | New decision replaces this one (link with `superseded_by`) |
| any | `rejected` | Option ultimately not chosen (rare — usually you just don't write the doc) |

**Rule:** Once `accepted`, edit only consequences / follow-up sections if new info emerges. Don't re-litigate.

## Anti-patterns

- All options framed as clearly inferior except the chosen one → not a real decision
- Missing the "why now" context → reader can't judge if the decision is durable
- No trade-offs section → you haven't thought through the cost
- Decided in a meeting, doc written 3 months later from memory → detail is stale
- Decision affects the entire project but `decided_by` is one person → should have been wider

## Cross-project visibility

Decisions that affect multiple workstreams should populate `affects_workstreams` array in frontmatter (e.g. `[product, engineering, pm]`). Weekly check-ins can surface affected decisions to keep teams aligned.

## Example

See `example.md` in this folder for a complete decision drawn from the Quillpad project.

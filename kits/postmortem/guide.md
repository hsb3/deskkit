---
type: sop
status: final
created: 2026-06-30
updated: 2026-06-30
tags: [meta]
---

# SOP — postmortem

A **postmortem** is a structured review run *after* an incident or a completed project. It reconstructs what happened, finds the real cause, and turns the lessons into owned follow-up actions.

This is a **Composite Work Doc SOP** — it owns **no `template.md`**. A postmortem introduces no new doc type; it's a *procedure* that assembles and reviews doc types you already keep (the project record) and emits its writeup **as a `retro` note** (with `variant: incident`). The retro SOP owns that template; this SOP composes it. See **References** below.

## When to run one

Run a postmortem when:

- An **incident** disrupted the work — an outage, data loss, a missed promise, a shipped defect that bit users.
- A **project or phase closed** and you want the durable lessons captured before the context evaporates.

Run it promptly, while the record is fresh and the people involved still remember the small decisions. A postmortem that waits a month is archaeology.

**Red flag:** "everyone already knows what happened" is exactly when a postmortem pays off — shared memory is unreliable and uncaptured, and the next person inherits none of it.

## How to run one

The procedure walks the existing project record into a `retro` note. Each step reads source notes the agency already maintains (see References) rather than inventing facts.

1. **Gather the record.** Pull the project's existing trail: the **RAID log** (which risks/issues actually fired), **meeting-notes** (what was decided in the room and when), **weekly-checkin / status-reports** (the week-by-week state), and **decision** notes (what was chosen and why). This is the evidence base — the postmortem reviews it, it doesn't re-litigate it.
2. **Reconstruct the timeline.** From those sources, lay out what happened in order, with timestamps. Anchor each entry to a signal (a check-in line, a meeting note, a log) so the timeline is sourced, not remembered.
3. **Root cause — 5-whys.** For the worst items, ask "why" until you reach a cause you can actually change (a process gap, a missing check), not a person. Stop at the systemic root, not the surface symptom.
4. **Impact.** State concretely what it cost — time lost, work redone, users/readers affected, scope dropped. Quantify where you can.
5. **What went well / what went poorly.** Capture *why* the good things worked (so you can repeat them) and be specific about the bad ("Friday decision wasn't documented, became Monday's blocker"), not vague ("communication was hard").
6. **Remediation + follow-up actions.** Every root cause earns at least one action. Each action has an **owner** and is specific enough to verify. No owner = it won't happen.
7. **Write it up as a retro.** Realize the output as a `retro` note (`variant: incident`) — see the retro SOP for the frontmatter and section shape. Record the corrective actions; where they imply tracked work, mirror them into the **RAID** issues/dependencies or a **decision** note.

## References

This SOP composes these doc types — it reviews their notes as input and reuses one as output. It does **not** define a template of its own.

- **[[raid]]** — the project's RAID log; which risks materialized and which issues/dependencies fired. Input, and a sink for new follow-up items.
- **[[meeting-notes]]** — decisions and discussion captured in the room, with dates. Timeline input.
- **[[weekly-checkin]]** — the week-by-week status trail. Timeline + impact input.
- **[[decision]]** — the recorded choices (and their rationale) under review. Input, and a sink for any new decision the postmortem forces.

**Output:** a **`retro`** note with **`variant: incident`** — `period_covered` set to the incident/project under review. The retro SOP owns that template; this SOP produces an instance of it.

## Anti-patterns

- **Blame.** Naming a person as the root cause ends the inquiry one "why" too early and guarantees no one tells the truth next time. Root causes are systemic.
- **No follow-through.** A postmortem whose actions have no owner, or that no one revisits, is theater. The value is the owned remediation, not the writeup.
- **Unsourced timeline.** A reconstructed-from-memory timeline drifts. Anchor every entry to a note or log.
- **New doc type.** Don't invent a "postmortem" frontmatter type — the output is a `retro`. Reusing the type is the point of a Composite Work Doc SOP.

## Example

See `example.md` in this folder — a worked postmortem for the Quillpad project, realized as a `retro` note (`variant: incident`).

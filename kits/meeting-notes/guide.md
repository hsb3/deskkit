---
type: sop
status: final
created: 2026-06-30
updated: 2026-06-30
tags: [meta]
---

# SOP — meeting-notes

## When to write one

Open a meeting-notes whenever a **working session** produces decisions or commitments worth keeping — a planning meeting, a design review, a kickoff, a client working session. The test: *did something get decided, or did someone leave owing a next step?* If yes, capture it. A pure status round-robin with no decisions usually belongs in a [[weekly-checkin]], not here.

Write the **Agenda** section *before* the meeting (see Variants), then fill the rest during or immediately after — the same note carried forward, never a fresh one.

**Don't use it for:** a structured Q&A to extract knowledge from a subject (that's the `interview` variant — see Variants), or a recurring status cadence (that's [[weekly-checkin]]).

## How to write one

A good meeting note is **decisions and owned action items**, padded by just enough discussion to remember *why*. The five sections, in order of value:

1. **Action items first in your mind.** Everything else is context for these. Each is `owner · item · due` — no owner means it won't happen; no due means it'll slip.
2. **Decisions** — the settled calls. State them plainly. A decision is not "we talked about the data model"; it's "we're going with the item model + a `kind` field." If a topic produced no decision, record that and route it to a follow-up rather than leaving a false impression of resolution.
3. **Discussion / notes** — substance per agenda topic: options weighed, open questions, points raised. Keep it to what you'd need to reconstruct the reasoning. Never transcribe.
4. **Attendees** — mirror the `attendees` frontmatter; note invited-but-absent so you know who to circle back to.
5. **Agenda** — already written before the meeting; leave it intact as the record of intent vs. outcome.

### Three distinctions to keep straight

| This note (working session) | Not this — use instead |
|---|---|
| **meeting-notes** — people meet, decide, leave with actions. | — |
| Elicitation Q&A to mine a subject's knowledge | the **`interview` variant** (below) — feeds a [[research-synthesis]] |
| Recurring "what shipped / what's stuck / what's next" | [[weekly-checkin]] — a status cadence, not a meeting |

The line that matters: **meeting-notes produces decisions; an interview produces material; a weekly-checkin produces status.** When unsure, ask what the note is *for* downstream.

## Status transitions

`meeting-notes` is in the **cadence** family — only two states:

| From | To | Trigger |
|---|---|---|
| `draft` | `draft` | Agenda written pre-meeting; note still being filled |
| `draft` | `published` | Meeting done, decisions + action items captured and shared |

Once `published`, the note is a record — don't reopen it. Follow-on work lives in the action items (and their own notes/issues), not in edits to a closed meeting note.

## Anti-patterns

- **Transcript, not notes** — capturing everything said buries the two things that matter. Distill to decisions + actions.
- **Ownerless action items** — a row with no owner or no due date is a wish, not a commitment. Fill both or cut the row.
- **No decisions recorded** — if the meeting decided things and the Decisions section is empty, the note failed. If it genuinely decided nothing, say so and book a follow-up.
- **A fresh note per session of the same thread** — carry the agenda → notes forward in one note; don't fragment a single meeting across drafts.
- **Using it as a status log** — recurring status belongs in [[weekly-checkin]]. Meeting-notes is for sessions that *decide*.
- **Skipping the agenda** — walking in without written desired outcomes turns the meeting into discussion with no decisions to capture.

## Variants

`meeting-notes` is the anchor type; two qualifiers reuse the same note rather than spawning new SOPs.

### `variant: interview`

Set `variant: interview` in frontmatter when the session is **elicitation** — a structured Q&A to extract knowledge, requirements, or context from a subject — rather than a working session where the group decides. It's the same skeleton, read differently:

- **Attendees** → the subject(s) and the interviewer.
- **Agenda** → your **question set / topic guide** prepared beforehand.
- **Discussion / notes** → the **answers**, organized by question — this is the payload.
- **Decisions** → usually empty (an interview rarely decides anything); use it only for genuine in-session agreements.
- **Action items** → follow-ups (clarifications to chase, docs the subject promised).

An interview's output is **input to something else** — it typically feeds a [[research-synthesis]] that aggregates several interviews into findings. The interview note is the raw source; the synthesis is the conclusion. Don't try to make an interview note carry conclusions itself.

### Pre-meeting `agenda` usage

There is no separate "agenda" type. A pre-meeting agenda is **just this note, written forward**: create the meeting-notes, fill only the **Agenda** section with topics + desired outcomes, leave `status: draft`, and circulate it before the meeting. When the meeting happens you fill Discussion / Decisions / Action items into the *same* note and flip it to `published`. One note spans the whole lifecycle: intent (agenda) → record (outcome). Writing the agenda first is what makes decisions capturable — you can't note an outcome for a topic nobody framed.

## Example

See `example.md` in this folder for a worked meeting note from a real Quillpad working session.

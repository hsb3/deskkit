---
type: meeting-notes
status: published
created: 2026-05-16
updated: 2026-05-16
tags: [quillpad]
meeting_date: 2026-05-16
attendees: [robin, claude]
---

# Meeting Notes: Quillpad — content-model & build-order working session

_Working session to unblock the Quillpad build (see the one-pager + RAID log). Goal: settle the content-model fork blocking the merge and lock the first build slice._

## Agenda

- **Content model** — pick the canonical model so the two source repos can merge. Desired outcome: a decision, captured as a `decision` note.
- **First build slice** — decide what gets built first so work can start this week. Desired outcome: a named slice + owner.
- **Auth** — confirm magic-link is enough for the invite-only audience, or open it as a risk. Desired outcome: confirm or escalate.

## Attendees

- robin — owner / sole author
- claude — build partner

## Discussion / notes

### Content model
- The blocker (RAID I-001): `agent_quillpad` is post-centric; `capture_kit` is item/ingestion-centric. The two can't merge until one model wins.
- Weighed keeping posts as the spine vs. adopting `capture_kit`'s item model with a `kind` discriminator. The item model generalizes — a post becomes `kind: post`, a captured link becomes `kind: bookmark` — without forcing captures to pretend to be posts.
- Cost: the post-centric editor assumptions need a light rework, but that work was coming regardless.

### First build slice
- Robin's stated philosophy: wire the thinnest complete vertical slice end-to-end before adding breadth.
- Candidate slice: capture → item store → editor open → publish, for a single `kind`. Editor is the riskiest unknown (CodeMirror + extensions, RAID A-002), so the slice should exercise it.

### Auth
- Invite-only, single-digit readers near term. Magic-link covers it; accounts/roles are unrequested scope.
- Robin will sanity-check with the first 3 invited readers (already tracked as RAID A-003) — no new action.

## Decisions

- **Adopt `capture_kit`'s item model + a `kind` field as the canonical content model.** Posts become `kind: post`. Resolves RAID I-001 and unblocks the merge. — robin; to be written up as a standalone `decision` note.
- **First build slice = capture → item store → editor → publish for one `kind`.** Editor goes first because it's the riskiest unknown. — robin.
- **No new auth work** — magic-link stands; revisit only if the reader audience grows past invite-only. — robin.

## Action items

| Owner | Item | Due |
|-------|------|-----|
| robin | Write the `decision` note for the item-model + `kind` choice; link it from RAID I-001 and D-002 | 2026-05-19 |
| robin | Build the editor-first vertical slice (CodeMirror + ⌘K assist + diff view) against one `kind` | 2026-05-23 |
| robin | Close RAID I-001 and flip D-002 to met once the decision note lands | 2026-05-19 |

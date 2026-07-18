---
type: sop
status: final
created: 2026-06-30
updated: 2026-06-30
tags: [meta]
---

# SOP — daily-note

## When to write one

Once a day, whenever the work day actually starts (or ends — pick one and keep it). The daily note is a **journal**, not a deliverable: its job is to capture what's in your head before it evaporates, so tomorrow-you doesn't have to reconstruct it.

This is **capture, not ceremony**. A daily note that takes more than a couple of minutes has turned into a status report — which is what the [[weekly-checkin]] is for. If you skip a day, skip it; there's no backfill. A gap is just a quiet day, not a debt.

**Cadence:** Daily, lightweight. One note per day, dated. It's a scratchpad with a date on it.

## How to write one

Open the template, fill what's true, close it. The sections are prompts, not a form to complete — leave any of them empty.

- **Focus for today** — the one or two things that matter most. If you can't name them, that's the signal to think for a minute before diving in.
- **Log / notes** — the running stream: decisions made, dead ends hit, links worth keeping, half-thoughts. Write it as the day happens, not from memory at the end.
- **Done** — what actually shipped or moved. Concrete beats vague: "merged the capture-URL fix" not "worked on capture."
- **Tomorrow** — the first thing to pick up next session, so you start with momentum instead of a cold read.
- **Open loops** — unfinished threads, things waiting on someone else, anything you don't want to drop. This is the bridge between days.

Link freely to the work — one-pagers, RAID logs, decisions — so the daily note stays connected to the things it's about. The journal is the thread; the linked notes are the substance.

## Note on schema

`journal` is a **lightweight type**: only the universal fields are required (`type`, `created`, `updated`, `tags`), the body is freeform, and there's no `status`. Because there's no status, there are **no status transitions** to track — a daily note is done the moment you stop writing it. Don't add ceremony the type deliberately omits.

## Anti-patterns

- **Turning it into a status report** — the daily note is for you, in the moment. If you're writing for an audience, you want the weekly check-in.
- **Backfilling missed days** — reconstructing last Tuesday from memory produces fiction. A gap is fine; leave it.
- **Filling every section** — empty sections are information. A day with no open loops is a clean day, not an incomplete note.
- **Hoarding instead of linking** — don't re-explain a decision the daily note can just link to. The journal points; the linked note holds the detail.
- **Adding a status field** — the type omits it on purpose. A lightweight note that grows a workflow has stopped being lightweight.

## Example

See `example.md` in this folder for a worked daily note from a real day on the Quillpad project.

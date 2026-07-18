---
type: product-spec
status: approved
created: 2026-05-12
updated: 2026-05-12
tags: [quillpad, inbox]
author: robin
decomposes_to: ["[[load-inbox-from-firestore]]", "[[add-triage-actions]]", "[[group-inbox-by-status]]"]
parent_one_pager: "[[quillpad-one-pager]]"
---

# Inbox view with attention groups (F010)

## Why this exists

The one-pager's core problem is friction: metadata extraction, link management, polish passes block Robin's writing cadence. The inbox is where captured content lives before it's ready to publish. An attention-grouped inbox surface (triage / develop / polish / schedule / revisions / cadence) lets Robin see what needs work, prioritize it, and know he's making progress. This directly advances the one-pager's success metric: "backlog of stalled drafts shrinks over time."

## What it does

Robin opens `/admin/inbox`. He sees all his captured content grouped into six attention categories:

1. **Triage** (auto-classified but unread): new captures with type + topic tags but not yet reviewed. Shows entry title, type, topic, and days since capture.
2. **Develop** (assigned to drafting): entries Robin has reviewed and decided are worth developing. Shows status, days in this group, and next suggested action from The Assistant (metadata extraction, chat prompt, etc.).
3. **Polish** (content-complete, needs tightening): entries with substantial text, ready for Claude to suggest polish passes (rewording, cross-links, TL;DR).
4. **Schedule** (polished, ready to publish): entries complete and reviewed, awaiting a publish date.
5. **Revisions** (published but stale): entries Robin published >N days ago that are candidates for updates or spin-offs.
6. **Cadence** (progress visualization): not a triage bucket, but a dashboard mini-section showing "Robin's on pace for 1 post / 7 days" or "overdue for a post."

Each entry card shows:
- Title
- Type icon (tool / paper / person / thought / etc.)
- Topic tags
- Days in current group
- Quick action buttons (promote to next group, drop, open editor)

## What it does NOT do

- Does NOT show published entries here (feed is reader-facing; admin/inbox is triage-only)
- Does NOT require manual tagging on capture (The Assistant auto-classifies; Robin reviews, not creates)
- Does NOT display entry full text in the list (too much noise; click to open editor)
- Does NOT merge inbox with content library (library is archived + published; inbox is active work)
- Does NOT implement calendar-based scheduling (schedule just means "ready"; publish date is set in the editor)
- Does NOT show analytics or engagement metrics (editorial, not product dashboard)
- Does NOT include commenting or collaboration features (Robin is solo author)

## Success criteria

1. Robin opens `/admin/inbox` and sees all in-flight content grouped by status
2. Triage group shows newly captured entries with auto-inferred type + topic tags; Robin can review and accept/reject tags
3. Robin can promote entries from triage → develop; system records promotion and move happens instantly (no page reload)
4. Develop entries show The Assistant's next suggested action (full-metadata extraction needed? polish pass ready? co-writing chat available?)
5. Polish entries show a "view suggestions" button that opens diff view of Claude's proposed edits
6. Robin can drop entries to a trash bucket; entries can be recovered within 7 days
7. Cadence section shows rolling 30-day publish count + target cadence + visual progress bar

## Interactions

### Happy path: Robin trages and promotes an entry

1. Robin opens `/admin/inbox`
2. Page loads; Triage group shows 3 new captures from today
3. First entry: "Why Midjourney's consistency breaker changes how designers think"
   - Type: `tool` (auto-classified)
   - Topics: `[ai, design, workflow]`
   - Robin reviews tags; they're correct
   - Clicks promote button
4. Entry moves to Develop group
5. Robin sees The Assistant's suggestion: "Metadata extracted; ready for co-writing? Open chat to refine focus."
6. Entry now shows:
   - Title
   - Topics
   - Estimated time in Develop (2-5 days typical)
   - Action buttons: "Open chat", "View metadata", "Move to Polish", "Drop"

### Error path: network issue during triage

1. Robin clicks promote
2. Network fails; page shows toast: "Failed to promote entry. Retry?"
3. Robin clicks Retry or navigates away and back
4. Firestore eventually syncs; entry moves to Develop

### Cadence view

Cadence section always visible (a small card above the inbox groups):
- "Last 30 days: 3 published entries"
- "Target cadence: 1 entry / 7 days"
- "Current streak: 8 days since publish"
- Visual: green bar if on pace, yellow if 2+ days behind, red if 7+ days behind
- Text hint: "You're overdue. Ready entries in Polish waiting to ship?"

## Design considerations

**Layout:** Inbox is a columnar list; each entry is a card. Expand on click to show more detail (full text, metadata, suggested edits). Don't force large diffs or chat panels inline.

**Visual hierarchy:** Group headers (Triage, Develop, etc.) are sticky at top so Robin always knows which section he's looking at. Count badge on each header (e.g., "Triage (3)") shows at-a-glance workload.

**Status color coding:** Optional. If implemented, use soft icons instead of loud colors. (Triage = gray, Develop = blue, Polish = green, Schedule = gold, Revisions = muted, Cadence = striped).

**Mobile consideration:** Inbox is admin UI; Robin primarily uses desktop. But responsive stacking (cards full width on mobile, sidebar hides) should still work.

## Related decisions

- [[choose-firebase-architecture]]: Firestore for content + inbox state
- [[genkit-agent-contract]]: The Assistant's metadata extraction and Polish proposals come from Genkit flows (see `F021`, `F025`)
- [[inbox-as-workflow-anchor]]: Inbox is the central hub of the authoring experience (not the editor; not the feed)

## Decomposition

This feature decomposes into three engineering-specs:

1. **[[load-inbox-from-firestore]]** — Query Firestore for all documents with status in [triage, develop, polish, schedule, revisions]; render as grouped list. Includes pagination/lazy-loading if inbox grows large.

2. **[[add-triage-actions]]** — Promote/drop buttons on each entry. Wire to Firestore document updates. Show toast feedback. Handle optimistic UI (button greyed out until Firestore write confirms).

3. **[[group-inbox-by-status]]** — Layout: sticky group headers with count badges. Visual distinction between groups (CSS or icons). Cadence mini-dashboard rendering publish-count + streak + target-cadence bar.

Engineering to refine decomposition boundaries — these are the semantically distinct chunks, but the split might shift based on implementation and shared state management.

## Open questions

1. **Should entries auto-promote (triage → develop) after Robin approves tags, or does he manually click promote?** UX trade-off: auto feels frictionless but might surprise. Manual is clearer but slower. → Recommend: manual promote + snappy feedback (no page reload).

2. **How long before revisions bucket shows entries?** "Stale" is >N days published without edit. N = 30? 60? → Recommend: 30 days; tunable later if Robin finds it's noisy.

3. **Should The Assistant auto-start metadata extraction on capture, or wait for Robin to move to develop?** Auto extraction wastes agent quota on entries Robin might drop. Manual extraction means latency when Robin reaches develop. → Recommend: auto extraction on capture (fire-and-forget), results cached in Firestore. Robin gets results instantly when he enters develop.

4. **Does the cadence nudge stay on-screen at all times, or only show if overdue?** Always-on is motivating but takes up vertical space. → Recommend: always on, but collapse to 1-line summary when not overdue.

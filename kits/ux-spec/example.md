---
type: ux-spec
status: approved
created: 2026-05-16
updated: 2026-05-22
tags: [quillpad]
author: robin
parent_product_spec: "[[capture-product]]"
---

# UX spec — Capture Inbox (triage view)

_Screen-level spec for the inbox where freshly-captured items wait for the author to confirm their auto-extracted metadata before they enter the library. Sits downstream of the bookmarklet/share capture and upstream of the editor. See the one-pager + RAID (R-001 extraction reliability, I-002 bookmarklet URL drop)._

## Surface / screen

`/inbox` — the triage list for items captured but not yet filed. Reached from the left rail (shows a count badge when items are pending). One job, one screen: confirm or correct the metadata Claude extracted, then accept the item into the library or discard it. Editing the item's *body* happens later in the editor — not here.

## Goal

Let the author clear a batch of captured items in seconds each: glance at what Claude proposed for title/topics/source, accept when it's right, fix the one field that's wrong, and move on. The screen must make "accept the suggestion" a single click and "the extraction was wrong" cheap to correct — never silent auto-apply (R-001).

## Layout

- **Header** — screen title, pending count, and a `Capture URL…` manual-add affordance.
- **Filter row** — source-type filter (`All / Article / Video / Note`) and sort (`Newest`).
- **Item list** — one card per captured item, collapsed by default; expands in place to the triage detail.
- **Selection bar** — floating, appears only when ≥1 card is checked; offers `Accept selected`.

```
┌──────────────────────────────────────────────────────────┐
│ Capture Inbox                                [4 pending]  │
│                                          [ Capture URL… ] │
├──────────────────────────────────────────────────────────┤
│ [All] [Article] [Video] [Note]              Sort: [Newest]│
├──────────────────────────────────────────────────────────┤
│ ☐ ┌──────────────────────────────────────────────────┐   │
│   │ Why agent workflows need state machines   2m ago │   │
│   │ Article · example.com · 3 topics suggested       │   │
│   │ [ Accept ]  [ Review ]  [ Discard ]              │   │
│   └──────────────────────────────────────────────────┘   │
│ ☐ ┌──────────────────────────────────────────────────┐   │
│   │ ⚠ Untitled capture                        9m ago │   │
│   │ Article · source URL missing · needs review      │   │
│   │ [ Accept ]  [ Review ]  [ Discard ]              │   │
│   └──────────────────────────────────────────────────┘   │
│                                                          │
│ ┌──────────────────────────────────────────────────┐    │
│ │ 2 selected     [ Accept selected ]  [ Clear ]     │    │
│ └──────────────────────────────────────────────────┘    │
└──────────────────────────────────────────────────────────┘
```

Expanding a card (`Review`) reveals the editable extraction in place — the list does not navigate away:

```
│   ┌──────────────────────────────────────────────────┐   │
│   │ Title    [ Why agent workflows need state … ] ✎  │   │
│   │ Source   [ https://example.com/agents-state ] ✎  │   │
│   │ Kind     ( Article ▾ )                            │   │
│   │ Topics   [ ai ×] [ process ×] [ state-machines ×] │   │
│   │          + add topic                              │   │
│   │ ── Claude's suggestions (tap to apply) ──────────│   │
│   │   topic: agents   ·   topic: workflows           │   │
│   │ [ Accept into library ]   [ Discard ]            │   │
│   └──────────────────────────────────────────────────┘   │
```

## Components

| Component | Shows | Notes |
|----|-------|-------|
| `InboxHeader` | Title, pending count, `Capture URL…` button | Count derives from the live captures query; button opens a small URL-entry dialog. |
| `InboxFilters` | Kind filter tabs + sort control | Client-side filter on `kind`; sort by `capturedAt` desc default. |
| `CaptureCard` (collapsed) | Title, kind, source host, topic count, relative time, checkbox | Title and source come straight from the capture record (no fetch). A `⚠` badge replaces the source host when a required field is missing or low-confidence. |
| `CaptureCard` (expanded / `TriageDetail`) | Editable title, source URL, kind select, topic chips, suggestion strip | The suggestion strip lists Claude's proposals as one-tap chips — applying one is an explicit author action, never automatic. |
| `TopicChips` | Assigned topics as removable chips + `add topic` | Autocompletes against canonical topics; free text creates a new topic on confirm. |
| `CardActions` | `Accept` / `Review` / `Discard` (collapsed) and `Accept into library` / `Discard` (expanded) | `Accept` is enabled only when all required fields resolve; otherwise it's disabled and the card shows `needs review`. |
| `SelectionBar` | "N selected", `Accept selected`, `Clear` | Sticky bottom; appears at ≥1 selection. Batch path accepts only items with no missing required fields. |
| `InboxEmptyState` | Illustration placeholder + message | Shown when zero pending items. |

## States

- **Default** — One or more capture cards, collapsed, newest first. The pending count matches the list length.
- **Empty** — Centered message: *"Inbox clear. Captured items land here for a quick metadata check before they're filed."* The `Capture URL…` action stays available so the author can add one manually.
- **Loading** — On first paint, three skeleton cards while the captures query resolves. Expanding a card never shows a loading state — the record is already in hand; there's no secondary fetch for triage.
- **Error** — If the captures query fails, an error card with the message and a `Retry` button replaces the list (other UI stays interactive). If an individual `Accept` write fails, the card stays in place, the button re-enables, and a toast reads *"Couldn't file that item — try again."*

## Interactions

| Trigger | Result |
|---------|--------|
| Author clicks `Accept` on a card | Wait-for-response (not optimistic — the write can fail). Button shows a spinner, the others disable; on success the card fades out, the count drops, toast *"Filed."*; on failure the card stays and the button re-enables. |
| Author clicks `Review` | Card expands in place to `TriageDetail`; list position is preserved. |
| Author taps a Claude suggestion chip | The proposal is applied to the matching field (e.g. topic added). Explicit, reversible — never applied without the tap (R-001). |
| Author edits the source URL and the field was empty (I-002) | The `⚠ source missing` badge clears once a valid URL is present; `Accept` enables. |
| Author clicks `Accept into library` (expanded) | Persists the corrected metadata, files the item, collapses and removes the card. Same wait-for-response + toast as collapsed accept. |
| Author checks ≥1 card | `SelectionBar` slides up showing the count. Cards with unresolved required fields are excluded from the selection and show a `needs review` hint. |
| Author clicks `Accept selected` | Files each selected item; the bar shows progress (*"Filing 2 of 3…"*); cards remove one by one; any failure stays selected with an error marker for individual retry. |
| Author clicks `Discard` | Confirm dialog (`confirmDialog`, never native `confirm()`); on confirm the capture is deleted and the card removed. |
| A new capture arrives while the screen is open | The live query pushes it in at the top; the count badge updates without a manual refresh. |

## Edge cases

| Case | Behavior |
|------|----------|
| Source URL missing from capture (bookmarklet JS-redirect drop, I-002) | Card shows `⚠ source missing`, `Accept` is disabled, and the expanded view focuses the source field. Author supplies the URL or discards. |
| Low-confidence extraction (title/topics uncertain) | Card surfaces `needs review` instead of an enabled `Accept`; suggestions still appear but nothing is pre-applied. |
| Very long title or many topics | Title truncates with ellipsis in the collapsed card (full text in the expanded field); topic chips wrap to multiple rows, the card grows. |
| Item already filed/discarded in another tab | The live query removes it; if the author is mid-review in that card, show toast *"This item was already handled"* and collapse it. |
| Duplicate capture of an already-filed URL | Card flags `possible duplicate · view existing`; accepting merges into the existing item rather than creating a second. |
| 40+ pending items | List scrolls; the count badge caps display at `40+`. No virtualization needed at expected volumes. |
| Batch accept, partial failure | Succeeded items leave; failed ones stay selected with an error marker; toast *"2 filed, 1 failed."* |

## Accessibility notes

- **Keyboard** — Cards are reachable in list order; `Enter` on a focused card toggles `Review` expand/collapse; `Esc` collapses an expanded card and returns focus to it. The selection checkbox is independently focusable. The `Capture URL…` and `Discard` dialogs trap focus and restore it to the trigger on close.
- **Screen reader** — Each card exposes an accessible name of "{title}, {kind}, captured {relative time}"; the `⚠` state is announced as "needs review", not as a bare icon. Accept/discard results post to a polite live region (*"Filed."* / *"Couldn't file that item."*) so async outcomes aren't silent.
- **Contrast / target size** — The `⚠` needs-review signal is carried by text + icon, never color alone. Action buttons and the checkbox meet the minimum target size; suggestion chips are real buttons, not text spans.

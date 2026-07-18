---
type: roadmap
status: approved
created: 2026-05-16
updated: 2026-05-22
tags: [quillpad]
author: robin
owner: robin
affects_workstreams: [product, engineering]
---

# Roadmap: Quillpad — AI-Augmented Personal Publishing

_The sequence for getting Quillpad from two stalled source repos to a working personal publishing loop (see the one-pager + RAID log). Order and unblocking only — no dates; "done" is proven by the goal, not the calendar._

## Vision / outcome

A single author can capture something interesting, let Claude strip the boring parts into clean metadata, write on top of it in a real editor, and publish to an invite-only audience — without the tool ever growing into a product build that stalls like the last two attempts.

## Themes

- **One content model** — the merge of `agent_quillpad` (post-centric) and `capture_kit` (item/ingestion-centric) onto a single item model with a `kind` field. Everything else sits on this; until it's settled, both halves stay frozen.
- **Trustworthy capture** — bring something in and let extraction suggest metadata, never silently apply it. The "removes the boring parts" promise lives or dies here.
- **Writing surface** — the CodeMirror editor with ⌘K assist and a diff view: where capture becomes published work.
- **Quiet publishing** — magic-link, invite-only delivery to a small reader set. Deliberately the thinnest thing that ships.

## Sequenced phases

### Phase 1 — Settle the content model
- **Goal:** One canonical item model (leaning to `capture_kit`'s items + a `kind` field) is decided and written up as a `decision` note; both source repos can map onto it.
- **Theme:** One content model
- **Unblocks:** Everything. The capture pipeline and the editor both read/write items, so neither can be built against two incompatible schemas. This is the keystone (RAID I-001 / D-002).

### Phase 2 — Thin end-to-end slice
- **Goal:** One item can be captured, get suggested metadata, be opened in the editor, and be published to a single test reader — a complete loop on one item, narrow but real.
- **Theme:** Trustworthy capture + Writing surface
- **Depends on:** Phase 1 (the model the slice reads/writes).
- **Unblocks:** Proof the architecture holds before widening. Surfaces the real cold-start and extraction-quality questions on something you can actually click through.

### Phase 3 — Harden capture
- **Goal:** Extraction ships as an accept/edit suggestion with a measured correction rate; the bookmarklet resolves canonical URLs server-side; capture is reliable enough to use unattended-ish.
- **Theme:** Trustworthy capture
- **Depends on:** Phase 2 (a working capture path to harden) and the first ~50 real captures (the correction-rate signal from RAID R-001).
- **Unblocks:** Trusting capture at volume — the precondition for writing being the main activity rather than babysitting extraction.

### Phase 4 — Editor depth
- **Goal:** ⌘K assist and the diff view feel good on real writing sessions; the in-editor chat has a warm path so it doesn't lag the author out of flow.
- **Theme:** Writing surface
- **Depends on:** Phase 2 (the editor slice exists) and Phase 3 (clean items to write on).
- **Unblocks:** The editor becoming the place the author lives, which is the point of the whole tool.

### Phase 5 — Publishing polish
- **Goal:** Magic-link delivery to the invited reader set is smooth end-to-end; the reading experience is presentable.
- **Theme:** Quiet publishing
- **Depends on:** Phase 4 (work worth publishing) and confirmation that magic-link is acceptable to the first readers (RAID A-003).
- **Unblocks:** Closing the loop — capture-to-reader — which is the success condition in the one-pager.

## Now / Next / Later

| Now | Next | Later |
|-----|------|-------|
| Decide the canonical content model and write the `decision` note (Phase 1) | Build the thin end-to-end slice on one item (Phase 2) | Harden capture + measure correction rate (Phase 3); editor depth (Phase 4); publishing polish (Phase 5) |

## Out of scope

- **Multi-author / accounts / roles** — invite-only single author is the whole audience model; no team features.
- **A heavier editor framework** — CodeMirror + a few extensions until it demonstrably can't carry ⌘K assist and diff (RAID A-002); no migration on spec.
- **Native/mobile apps** — web only.
- **Public, open publishing** — invite-only by design; an open feed is a different product.

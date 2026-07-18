---
type: feature-spec
status: approved
created: 2026-05-16
updated: 2026-05-22
tags: [quillpad]
author: robin
parent_product_spec: quillpad-product-spec
priority: P1
---

# Auto-enrich capture metadata

_One feature of Quillpad — AI-Augmented Personal Publishing. Decomposed from the Quillpad product-spec; tracked against R-001 in the project RAID log._

## User story

> **As a** writer capturing a link or note into Quillpad
> **I want** Claude to suggest the title, summary, tags, and source URL for me
> **So that** I can file a clean, searchable item without stopping to type metadata by hand.

## Problem

Capture today is a friction wall: every bookmark or clipping lands as a bare URL or a blob of text, and the boring metadata — a real title, a one-line summary, a few tags, the canonical URL — has to be typed by hand before the item is findable. So items get dumped raw and the library rots into an unsearchable pile, which is exactly the "removes the boring parts" promise the product-spec is built on. The two prior attempts stalled here. This feature is the first beat of that promise.

## Proposed solution

On capture, Quillpad sends the page (or pasted text) to Claude and shows a **pre-filled enrichment card** above the new item: suggested title, a one-to-two sentence summary, 3–5 suggested tags, and the resolved canonical URL. Each field is editable inline; the user accepts the card with one click or edits any field first. Nothing is written silently — the suggestion is always a draft the user confirms.

- **Happy path:** capture → card appears pre-filled in ~2s → user glances, clicks **Accept** → item is filed enriched.
- **Error path:** if extraction fails or times out, the card falls back to "title = page title, everything else empty" and surfaces a quiet "couldn't enrich — fill in manually" note. Capture never blocks on enrichment.

## Success criteria

- [ ] Every capture shows an enrichment card with title, summary, tags, and resolved URL pre-filled before the item is saved.
- [ ] Suggestions are never auto-applied — the item is written only after an explicit Accept (per R-001's mitigation).
- [ ] Median enrichment round-trip is under 3 seconds; a failure falls back gracefully without blocking capture.
- [ ] Across the first 50 real captures, the field-correction rate is under 40% (the R-001 tripwire to widen vs. tighten the prompt).
- [ ] The canonical URL is resolved server-side, so JS-redirect sources store the final URL (closes RAID I-002).

## Non-goals

- **Auto-applying suggestions without review** — deferred indefinitely; the accept-step is the whole trust model (R-001).
- **Bulk re-enrichment of the existing backlog** — this feature covers new captures only; a backfill pass is its own spec.
- **Taxonomy management** — suggesting tags from the existing set is in; creating, merging, or governing the tag vocabulary is out.
- **Full-text extraction of PDFs / paywalled pages** — first cut handles HTML pages and pasted text only.

## Alternatives considered

- **Silent auto-fill, no card** — enrich and write in one step, no confirmation. Rejected: R-001 says extraction isn't reliable enough to trust unattended; a wrong silent title is worse than a blank one, and it kills user trust early.
- **Manual metadata only (do nothing)** — keep typing it by hand. Rejected: this *is* the boring part the product exists to remove, and it's where both prior attempts died.
- **A separate "enrich" button the user clicks later** — capture stays bare, enrichment is opt-in afterward. Rejected: deferred enrichment never happens; the value is in making the clean path the default path, at the moment of capture.

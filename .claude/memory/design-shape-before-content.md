---
name: design-shape-before-content
description: "Henry wants UI design work sequenced shapes → structure → content → visual; a first pass carrying real data and populated screens is second-round material delivered too early"
metadata:
  node_type: memory
  type: feedback
  originSessionId: dacdc277-972f-498d-a544-1a641f9202fd
  modified: 2026-08-19T00:00:00.000Z
---

Henry, reviewing the first SPA wireframes (2026-08-19): "it's a lot. i would really like just
some basic outlines with annotations first... this stuff just jumped in the deepend. there's so
much shit just thrown on every page i get no sense of what visual structure would look like. all
the stuff in html files seem to me to be what i'd expect to see as a second round annotation on
top of basic wireframes."

The wireframes were not bad work. They were the right work at the wrong stage — the brief asked
for "real fixture data, never lorem" and populated screens, which produced a round-two artifact
before round one existed.

**Why:** Populated screens hide the layout. When every region is full of realistic content, the
question "is this the right arrangement of regions?" cannot be seen, let alone answered. He needs
to react to shape before he can react to anything else, and a detailed first pass forces him to
either accept a structure he never got to evaluate or throw away work.

**How to apply:** Sequence UI design as **shapes → structure → content → visual**, and deliver
one stage at a time. The first pass is labelled region boxes only — no fixture data, no styling,
no component choices — in a sketch medium (`.excalidraw`, drawio) rather than working HTML, so it
reads as a draft. Give each screen a one-line intent and ONE open structural question to react
to; that question is the deliverable's real payload. Framework and component choices stay
explicitly deferred until structure is agreed.

When briefing a subagent for this, "real fixture data, never lorem" is the wrong instruction for
a first pass — it is correct only from the content stage onward. Verify a visual deliverable by
rendering it and looking at it before handing it over; a structure pass that validates as JSON
can still have clipped labels and overflowing regions. Related: [[decision-briefings-outside-terminal]].

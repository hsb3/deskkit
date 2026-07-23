---
name: decision-briefings-outside-terminal
description: "Henry needs decision-support briefings as non-markdown artifacts outside the terminal (deck/PDF/audio), quickly digestible"
metadata: 
  node_type: memory
  type: feedback
  originSessionId: ced690ed-69b8-4f42-be23-6760e4c2f8da
---

When work reaches a decision point for Henry (rulings, design sessions, roadmap forks), a
markdown report in the terminal is not a usable decision input for him. He asked explicitly
(2026-07-20, ahead of the desk-standard post-bug-floor design session): "for me to digest
well, we're going to need non markdown briefings / outside the terminal that i can go
through quickly to help with the decision making."

**Why:** He digests and decides from flippable visual artifacts, not scrollback prose.

**How to apply:** Deliver decision packages as a deck (exec-desk:comms standard, composed
with exec-desk:pptx-themes / deck-builder; PDF export as the portable must-have; audio
narration welcome), staged in the relevant desk's `_meta/briefings/yyyy-mm-dd-subject/`.
Keep the register per [[feedback-decks-explanatory-register]] and simplify per
[[feedback-plain-language-on-request]]. Markdown analyses remain the source of truth on the
desk; the briefing is the decision surface.

**Collecting his answers (added 2026-07-20):** the response side of the same preference —
whenever a batch of decisions/questions needs Henry's input, ALWAYS run the
`owner-signoff:owner-signoff` skill (HTML form, recommendations preselected, one-click
all-defaults, answers to `answers.json`). Never present a decision batch as chat questions
he has to answer inline. He re-asked for this pattern by name (2026-07-20, design-session
rulings) even while a served form already existed — when a form is pending, re-open the URL
and say so; the chat message alone isn't enough to put it in front of him.

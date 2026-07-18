---
type: project-brief
status: approved
created: 2026-05-09
updated: 2026-05-16
tags: [quillpad]
author: robin
owner: robin
---

# Project Brief: Quillpad — AI-Augmented Personal Publishing

_North-star for the Quillpad project. The one-pager scopes the first shippable version; the product-spec plans the build. This brief is the vision both of them answer to — revisited whenever scope or audience drifts._

## Vision / North-star

Robin has interesting things to say and a small circle who'd genuinely value reading them — but every attempt to publish dies in the friction between thinking and sharing. Quillpad is a personal publishing workspace where an AI co-writer removes the boring parts (capture, metadata, link management, polish) so that writing happens **on cadence** instead of in abandoned bursts. A year out: Robin opens the editor without dread, publishes every week or two, and a handful of friends look forward to the next piece.

**North-star:** _I don't dread opening the editor — so I actually publish._

## Problem & target user

- **Who:** Robin — a solo author who wants to publish curated tech and thought pieces to a small invited audience, **without becoming a product builder** to do it.
- **Problem:** The act of writing isn't the bottleneck; the surrounding machinery is. Setup, metadata extraction, link hygiene, styling, and deploy plumbing sit between a finished thought and a shared post — and that friction kills cadence. Ideas become unfinished drafts become abandoned projects.
- **Today's workaround:** Two stalled repos chasing the same vision — `agent_quillpad` (a clean Astro + Firebase publishing site with a toy UI and no AI workflow) and `capture_kit` (a working Python ingestion pipeline, local-only, with no publishing surface). Each solves half the problem; neither ships.
- **Evidence it's real:** Robin said it plainly — "I want to share some thoughts with friends." Two separate build attempts confirm both the desire and the friction are genuine, not invented.

## Value proposition

The bet is that **an AI-augmented editor changes the author's relationship to publishing** — from a chore to be endured to a thing pulled toward. Off-the-shelf tools (Substack, Ghost, a static-site generator) all make Robin the metadata clerk, link checker, and deploy operator. Quillpad makes the agent do that work, so the only thing left for Robin is the part he actually wants to do: think and write. The value isn't "another blog platform" — it's **removed dread**, measured by whether the writing actually happens.

## Scope boundaries

**In scope**
- A unified capture → triage → write → publish loop for a single author.
- AI co-writing inside the editor: metadata extraction, polish-pass proposals, in-context chat.
- A clean, distraction-free reader surface for a small invited audience.
- The merge of the two stalled repos into one coherent system.

**Out of scope**
- **Multi-author / collaboration** — this is Robin's personal site; shared authorship adds auth and permission complexity the vision doesn't need.
- **Audience growth machinery** — no public signup, subscriber analytics, A/B testing, or engagement optimization. The bar is editorial quality, not reach.
- **Community features** — no comments or discussion threads; this is editorial, not a product.
- **Newsletter/email as a delivery channel** — web publish + RSS carry v1; email is a later question, not part of the north-star.

## Success criteria

- Robin publishes on a real cadence — roughly one post every 7–14 days — sustained past the first month, versus zero posts in the prior ~12.
- The draft backlog stops growing: pieces complete and clear instead of accumulating as stalled drafts.
- Time from idea to published post drops to under ~30 minutes of friction (capture → publish entirely through the UI, no terminal).
- At least two readers spontaneously say some version of "I look forward to your posts."
- The honest subjective test passes: Robin stops dreading the editor and actually uses the thing.

## Constraints

- **Solo build, summer 2026 window** — Robin is the only builder and wants the workflow locked before fall semester; the vision has to be achievable by one person in a season, which forces ruthless scope.
- **Firebase / GCP stack** — building on the `agent_quillpad` foundation (Astro + React islands, Firestore, Firebase Functions + Genkit) rather than introducing a new platform.
- **Claude as the co-writer** — AI features depend on the Anthropic API; the editor must stay usable (write + publish) even when the agent is unavailable.
- **Invite-only audience** — no public-registration surface to design or moderate; magic-link auth for a known, small reader set.

## Open questions

- **Canonical content model** — `agent_quillpad` is post-centric, `capture_kit` is item/ingestion-centric. Which model wins shapes the whole merge and gates build start. (Leaning toward `capture_kit`'s item model + a `kind` field; tracked as a decision.)
- **Publishing cadence target** — weekly vs. bi-weekly changes how aggressively the UI should nudge, and what "on cadence" means for the success criteria.
- **Soft-launch audience size** — 3 close friends vs. ~20 affects how much onboarding polish the reader surface needs before first launch.
- **Where the AI stops** — how much the co-writer proposes vs. silently applies; the trust boundary that keeps "removes the boring parts" from becoming "writes things I didn't say."

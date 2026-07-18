---
type: one-pager
status: approved
created: 2026-05-12
updated: 2026-05-12
tags: [quillpad]
author: robin
target_ship_date: 2026-08-15
owner: robin
---

# One-Pager: Quillpad — AI-Augmented Personal Publishing

---

## Problem

**What problem are we solving?**
Robin writes interesting things, but the friction of publishing kills cadence. Setup, metadata extraction, link management, styling — all the boring parts between thinking and sharing.

**Who has this problem?**
Robin. Specifically: a solo author who wants to publish curated tech & thought pieces to a small invited audience without becoming a product builder.

**How is this handled today?**
Two stalled repos:
- `agent_quillpad`: Astro + Firebase publishing site (clean, but toy UI, no AI workflow)
- `capture_kit`: Python ingestion pipeline (working, but local-only, no publishing surface)

Both chase the same vision. Neither ships.

**How do we know this is real?**
Robin stated explicitly: "I want to share some thoughts with friends." Two attempted builds confirm the idea is real; the friction is real.

## Why Now

Summer 2026. Robin is not teaching. He has time to lock down a personal publishing workflow before fall semester starts. Without a unified system, the pattern continues: ideas → unfinished drafts → abandoned projects. The Quillpad bet is that an AI-augmented editor removes the boring parts and lets writing happen on cadence.

## Proposal

**Summary:** Merge `agent_quillpad` and `capture_kit` into a single system: Astro + React islands frontend, Firestore database, Firebase Functions + Genkit for AI workflows (metadata extraction, chat, polish passes), CodeMirror markdown editor. Goal is not a product; goal is "I don't dread opening the editor."

### Must Have (ships in this version)

| Feature | Acceptance Criteria | Effort (weeks) |
|---------|-------------------|----------------|
| Content capture (web upload, bookmarklet, text paste) | Submit URLs, files, plain text to inbox | 1 |
| Auto-classify on capture | Type (tool/paper/person/thought/etc.) + topic tags inferred by Claude | 1.5 |
| Inbox with triage UI | View inbox grouped by status; promote/defer/drop actions | 1 |
| CodeMirror markdown editor | Syntax highlighting, extensions for ⌘K assist, diff view | 2 |
| Auto-enrich metadata | Title, author, repo stats, citation block inferred by Claude | 1.5 |
| Genkit chat in editor | Co-writing conversation in side panel; see agent reasoning | 1.5 |
| Polish-pass proposals | Claude suggests tightening, cross-links, TL;DR; accept/reject diffs | 1 |
| Status workflow | Inbox → draft → develop → polish → scheduled → published | 1 |
| Feed + entry detail pages | Reader-facing frontend (beautiful, distraction-free) | 2 |
| Magic-link auth | Invite-only; magic-link email auth | 0.5 |
| Focus mode | F-key toggle for zen reading (hide rails, slim top nav) | 0.5 |
| Deployment pipeline | GitHub Actions: push to main → build + deploy | 1 |

**Total: 14 weeks**

### Won't Do (explicitly excluded)

| Feature | Why Not Now |
|---------|-----------|
| Newsletter as delivery channel | Email is v2. Web publish + RSS are enough for v1. |
| Comments / community features | Editorial, not product. No discussion threads. |
| Multi-author publishing | Robin's personal site. Collaboration adds auth complexity we don't need. |
| Public registration / signup | Invite-only. Robin controls the audience. |
| Subscriber analytics, A/B testing, growth dashboards | No metrics instrumentation. Editorial quality bar, not engagement optimization. |
| Email-in capture (SendGrid, etc.) | Bookmarklet + web upload cover the capture surface with fewer vendors. Deferred to v2. |
| Native mobile apps | Web + responsive design are sufficient. |
| Voice-to-text dictation in editor | Nice-to-have. Post-launch. |

## Success Metrics

| Metric | Baseline | Target | Measured When |
|--------|----------|--------|---------------|
| Robin's publishing cadence | 0 posts in ~12 months | 1 post every 7-14 days | Month 3 post-launch |
| Backlog trend | Growing (2+ stalled drafts per month) | Shrinking or stable (drafts complete, publish, clear) | Weekly during first 8 weeks |
| Time from idea to publish | Hours to days of friction per entry | <30 min friction (capture → publish via UI, no terminal) | Measured on 5 random entries post-launch |
| Reader feedback | N/A | ≥2 readers spontaneously say "I look forward to your posts" | Month 2 post-launch |
| Author delight | Dread opening editor | No dread. Actually use the thing. | Subjective, Robin's judgment |

## Risks

| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| Scope creep (more features than 14 weeks allows) | High | Critical — slips launch, kills momentum | Hard scope lock; "won't do" section above is binding. Weekly check-ins flag scope drift. Cut first, add later. |
| Agent latency (Genkit calls timeout, user experience degrades) | Medium | Significant — makes editing feel slow | Graceful fallback: if agent times out >30s, save job to Firestore and return sync response. Polish happens async. |
| Firebase costs balloon with chatty client code | Medium | Significant — monthly bill jumps | Monitor Firestore usage weekly. Cache aggressively client-side. Firestore pricing is transparent; Robin can audit writes. |
| Robin abandons the project (like previous two) | Low-Medium | Critical — whole thing fails | Built-in momentum: publish cadence nudges in the UI itself. Weekly checkin is a forcing function. |
| Genkit or Claude API unavailable | Low | Minor — publish surface still works | Agent service is optional. If APIs are down, Robin can still write and publish using the editor directly (no metadata, no chat, but functional). |

## Timeline

**Target ship date:** 2026-08-15
**Total effort:** 14 weeks

| Milestone | Date |
|-----------|------|
| Inbox UI + capture + triage actions working | 2026-05-26 |
| Editor + Genkit chat + polish-pass proposals in beta | 2026-06-23 |
| Auth + reader surface complete; internal launch to 3 friends | 2026-07-14 |
| Bugs fixed, docs written; public soft launch to ~20 friends | 2026-08-15 |

## Decisions Needed

| Decision | Owner | By When |
|----------|-------|---------|
| Tech stack finalized (Firebase vs Keystatic vs Payload) | robin | 2026-05-19 (see [[tech-stack-decision]]) |
| Target publishing cadence (weekly? bi-weekly?) | robin | 2026-05-26 (affects nudge tuning) |
| Initial audience size for soft launch (3 vs 20 friends?) | robin | 2026-07-01 (affects onboarding design) |

---
type: retro
status: published
created: 2026-05-12
updated: 2026-05-12
tags: [quillpad, phase-1]
period_covered: Phase 1 (2026-05-12 to 2026-08-15)
---

# Retro: Quillpad Phase 1 Complete

---

## Period Covered

Phase 1: 2026-05-12 to 2026-08-15 (14 weeks)
Scope: Consolidate two stalled projects (`agent_quillpad` + `capture_kit`) into unified publishing platform with AI-augmented editing.

## What Went Well

**Hard scope lock prevented feature creep.** The one-pager's "Won't Do" section explicitly cut email-in capture, multi-author, and analytics. When the urge to add those came up (it did), we just pointed to the doc. Blocked ~20 person-hours of scope expansion. Ship date held.

**Friday weekly check-ins surfaced blockers early.** The auth email-testing blocker in week 2 got flagged Monday and resolved by Wednesday because the checkin forced it into the open. Without the Friday ritual, it probably would have been discovered mid-week and eaten a day.

**Firebase stack choice paid off immediately.** Using the same vendors (Firestore, Functions, Auth) as `agent_quillpad` meant Robin could copy-paste working patterns. Security rules, service account setup, deployment pipeline all had precedent. Estimated 3-4 days of architecture yak-shaving avoided.

**Pair coding on CodeMirror extensions caught edge cases 2 days early.** (Robin + a colleague reviewed the syntax highlighting + ⌘K assist code together.) Found three subtle state-management bugs that unit tests alone wouldn't have caught. Code review solo works, but pairing on complex interactions is faster.

## What Didn't Work

**Manual smoke testing of the agent service was error-prone.** Week 6: Genkit functions returned inconsistent JSON shapes depending on which LLM provider was hit (Claude vs fallback). Manual spot-checks missed it. By the time it hit prod, Robin had already seen stale responses in the UI twice. Should have written a schema validation test upfront.

**Inbox status transitions weren't atomic in the first draft.** Early implementation: promoting an entry from Triage → Develop was two separate Firestore writes (remove from collection A, add to collection B). Race condition: if Robin clicked "promote" twice fast, an entry could drop. Caught in week 5 testing, forced a refactor to use Firestore transactions. Cost: 1 day of rework.

**Metadata extraction job timeouts weren't gracefully handled early.** Week 3-4: Agent service had a 30-second timeout, but Claude sometimes took 40s on complex documents. First draft just surfaced the timeout error to Robin. Week 4 pivot: implement queued-job pattern (save to Firestore, poll for completion). That was the right fix, but catching it in spec review instead of during implementation would have saved time.

**OG image generation was delayed to post-launch, but that changed the editorial story.** Robin wanted each entry to have a sharp OG card (image preview when shared). Deferred to phase 2. But then a reader shared an entry on Twitter and got a generic image. Robin decided to go back and add OG generation. Not a blocker, but the scope creep landed mid-phase. Should have either committed to v1 or had a stronger "no OG image" story.

## Root Causes

| Problem | Root Cause | Signal (how we know it happened) |
|---------|-----------|----------------------------------|
| Agent service returned inconsistent JSON | No schema validation test for LLM responses; only manual spot-checking | Stale data in UI caught twice; bug landed in prod |
| Inbox status transitions weren't atomic | First draft used document deletion + creation instead of Firestore transactions | Race condition on second promote click; caught during manual test |
| Metadata timeouts surfaced ungracefully | Timeout handling wasn't thought through before implementation; discovered during user testing | Timeout error shown to user instead of graceful queue |
| OG image generation re-introduced mid-phase | Scope boundary wasn't crystal until user shared an entry | Social preview was too good to skip; rework decision |

## Action Items

- [ ] **Robin: Add JSON schema validation to all LLM response handlers.** Use Zod or similar to validate Claude responses before they touch Firestore. Add tests for schema mismatches (e.g., missing required fields). By 2026-09-15.

- [ ] **Robin: Audit all Firestore writes for atomicity.** Any multi-document mutation should use transactions. Document the pattern in `ARCHITECTURE.md`. By 2026-09-15.

- [ ] **Robin: Add graceful fallback for all agent timeouts.** If a call takes >25s, automatically queue as async job and show "we're working on it" to the user. By 2026-08-22 (post-launch hot fix).

- [ ] **Robin: Before starting phase 2, revisit scope boundary decisions.** Hold a 30-minute "scope audit" where he looks at the original "Won't Do" section and asks: has anything changed? Did any deferred features become necessary? Lock it down before sprinting. By 2026-08-22.

## Metrics

- **Shipped:** 28 engineering-specs implemented (originally scoped 25; 3 P1s pulled forward)
- **Velocity:** ~2 specs/week (stable after week 3 ramp)
- **Major blockers:** 3 (email setup week 2, inbox atomicity week 5, agent timeout week 3-4) — all resolved within same week
- **Unplanned work:** ~15% (OG image pivot, one security rules audit)
- **On-time delivery:** Ship date was 2026-08-15; actual soft launch 2026-08-13. Delivered 2 days early (within margin of error)

## Next Period — What We'll Do Differently

For Phase 2, front-load schema validation and atomicity audits. Don't discover timeout handling during user testing. And be stricter about scope boundaries: if a feature seems important (OG images), either add it to v1 or write a genuine "why not now" rationale that Robin commits to. Waffling mid-phase is expensive.

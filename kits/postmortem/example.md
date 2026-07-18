---
type: retro
status: published
created: 2026-05-19
updated: 2026-05-21
tags: [quillpad]
period_covered: Quillpad incident 2026-05-18
variant: incident
author: robin
owner: robin
---

# Postmortem: Quillpad capture outage — 2026-05-18

_Worked postmortem for the Quillpad — AI-Augmented Personal Publishing build, realized as a `retro` note (`variant: incident`). Reconstructed from the RAID log, the 2026-05-18 weekly check-in, and the content-model decision note._

## Incident Summary

On 2026-05-18, the bookmarklet capture path silently stored items with the wrong canonical URL whenever the source used a JS redirect. Eleven captures over the prior week pointed at redirect-shim URLs instead of the real articles, so the reading list rendered dead links. The capture endpoint returned `200` the whole time — nothing surfaced the failure until a reader reported a broken link.

## Timeline

| When | Event | Signal |
|------|-------|--------|
| 2026-05-11 | Bookmarklet capture path shipped; URL stored as the browser's `location.href` at click time. | Editor + capture build commit |
| 2026-05-12 → 05-17 | 11 captures land from redirect-using sources, each storing the shim URL. No error raised. | Firestore capture rows |
| 2026-05-18 09:40 | Reader reports two dead links in the published list. | Reader email |
| 2026-05-18 10:15 | Reproduced: capture stores pre-redirect URL; issue traced to client-side URL read. | Matches RAID **I-002** (open) |
| 2026-05-18 11:30 | Hotfix: resolve final URL server-side on capture before storing. | Capture route patch |
| 2026-05-18 14:00 | Backfilled the 11 bad rows by re-resolving stored URLs. | One-off backfill script |

## Impact

- **11 captures** stored with wrong URLs; the published reading list showed dead links for ~6 days.
- **~2 hours** of unplanned work (diagnosis + hotfix + backfill) pulled off the editor slice.
- **Trust cost:** the first external reader's first signal was a broken link — the worst possible first impression for an invite-only audience.
- No data lost; original URLs were recoverable by re-resolving the shims.

## Root Cause (5-whys)

| Step | Why? |
|------|------|
| Dead links in the list | The stored canonical URL was the pre-redirect shim, not the article. |
| Why the shim URL? | Capture read `location.href` **client-side**, before the source's JS redirect resolved. |
| Why was that not caught? | The capture endpoint returned `200` and never validated that the stored URL resolved. |
| Why no validation? | URL-resolution was assumed reliable; capture was treated as a thin write with no post-store check. |
| **Root** | **No server-side canonicalization or health check on capture** — the path trusted client-supplied data and had no signal when it was wrong. (Systemic, not a one-off slip; this was already logged as RAID **I-002** at P2 and under-prioritized.) |

## What Went Well

- **RAID log already named it.** The exact failure was logged as **I-002** before it bit — diagnosis took minutes because the cause was pre-identified. The register earned its keep.
- **Fast, contained hotfix.** Server-side URL resolution was a small, well-scoped change; backfill of the 11 rows was a one-off script, not a migration.
- **No silent data loss.** Because the shim URLs were resolvable, the bad rows were fully recoverable.

## What Went Poorly

- **A known P2 issue (I-002) sat open until it became a P0 incident.** It was real, logged, and de-prioritized — the postmortem-worthy miss is the prioritization, not the discovery.
- **Capture had no health signal.** A `200` with bad data is worse than a `500`; the path could fail meaningfully while looking healthy.
- **An external reader was the monitoring.** The first detection of a data-quality failure came from outside, not from any check we owned.

## Remediation & Follow-up Actions

| # | Action | Owner | Status |
|---|--------|-------|--------|
| 1 | Resolve final/canonical URL server-side on every capture before storing (shipped as the hotfix; keep as the standing contract). | robin | done |
| 2 | Add a capture-time health check: after store, verify the URL resolves to a `200`; flag the row for review if not. | robin | open |
| 3 | Re-rank RAID severity rules so a "stores wrong user-facing data" issue can't sit at P2 — data-correctness issues start at P1. | robin | open |
| 4 | Record the "no client-supplied canonical data trusted unvalidated" rule as a `decision` note so it binds future capture sources. | robin | open |

## Next Period — What We'll Do Differently

Two changes. First, **own the monitoring** — no user-facing data path ships without a check we run, so the next failure is caught by us, not a reader. Second, **respect the RAID log's own escalation** — a logged data-correctness issue is a P1, not a someday-P2; the register only protects us if its priorities reflect real blast radius.

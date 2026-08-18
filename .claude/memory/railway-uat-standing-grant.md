---
name: railway-uat-standing-grant
description: Standing owner grant to redeploy the Railway deskkit service without asking; it is his UAT environment
metadata: 
  node_type: memory
  type: feedback
  originSessionId: 82cb607f-beb0-46e8-94a3-e98331c6704d
  modified: 2026-08-18T16:10:43.603Z
---

**Standing grant (owner, 2026-08-18): redeploy the Railway `deskkit` service whenever a change
warrants it — no per-deploy confirmation needed.** The hosted service is the owner's UAT
environment; his review and testing of the deployed build is how he generates future dev tasks.

**Why:** treating each redeploy as an outward-facing action needing sign-off starved him of the
thing he actually reviews. A stale hosted build is worse than a redeploy he did not pre-approve,
because the published docs already claim the hosted surface works.

**How to apply:** after a merge that changes runtime behavior or the SPA, redeploy and report the
result. Still verify the deploy actually took (`GET /` returns the SPA shell, not a PocketBase
404 — the pre-SPA image failed exactly this way). The grant covers redeploying the existing
service; it does not extend to changing Railway project/volume/env configuration, adding services,
or anything that could destroy stored data.

Related: [[app-family-domain-split]] for what deskkit is; the deploy pattern card is
`docs/pattern.md`.

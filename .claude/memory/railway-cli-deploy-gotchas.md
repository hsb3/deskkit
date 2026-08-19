---
name: railway-cli-deploy-gotchas
description: Railway operational traps measured on the deskkit service — env-var delete does not redeploy, and a GitHub-triggered deploy can hang in SNAPSHOT_CODE
metadata:
  node_type: memory
  type: reference
---

Two Railway behaviours measured on the `deskkit` service (2026-08-18) that each cost a wasted
round trip, neither obvious from the CLI output.

**`railway variable delete <KEY>` does NOT trigger a redeploy.** It exits 0, the variable is gone
from `railway variables`, and the *running process keeps the old value in its environment* — so
anything resolving that variable at use time still sees it. Confirm with `railway deployment list`
that no new deployment appeared, then `railway redeploy --yes`. Verify against a live endpoint that
reports resolved config, not against `railway variables`.

**A git-push deploy can fail with zero build logs.** Symptom: status FAILED, `railway logs --build`
empty, and `deployment(id:){meta}` carrying `queuedReason: "Deployment queued due to upstream
GitHub issues"`. `deploymentEvents` shows `SNAPSHOT_CODE` running ~40 minutes before aborting —
Railway never fetched the source. Nothing is wrong with the image. Recovery: `railway up` from a
clean local checkout, which uploads directly and bypasses GitHub entirely.

Also: a deployment's recorded `builder` is resolved from the snapshot, so a failed snapshot records
the default (`RAILPACK`) even when the repo has a Dockerfile. That value is a symptom, not the
cause — do not "fix" the builder setting in response to it.

Diagnosis beyond the CLI's own subcommands goes through `railway api '<graphql>'`; useful fields
are `deployment(id:){status meta}` and `deploymentEvents(id:){edges{node{step createdAt
completedAt}}}` (note: `DeploymentEvent` has no `status` or `reason` field).

Related: [[railway-uat-standing-grant]] for when redeploying is pre-authorized.

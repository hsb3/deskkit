---
type: runbook
status: approved
created: 2026-05-18
updated: 2026-05-22
tags: [quillpad]
author: robin
owner: robin
---

# Runbook: Deploy Quillpad to Firebase

_The repeatable release for Quillpad — AI-Augmented Personal Publishing (see the one-pager + RAID log). Ships the Hosting bundle and the Functions backend (capture endpoint + in-editor Genkit chat) to the production Firebase project in one pass._

## Purpose / when to run this

Run this to cut a production release of Quillpad: after a feature merges to `main` and you want it live. It deploys both the static site (Hosting) and the server (Functions) atomically-ish, then verifies the live capture endpoint. Robin runs it; it assumes one author and the single `quillpad-prod` Firebase project.

## Prerequisites

- [ ] `firebase-tools` installed and authed: `firebase login` already done, `firebase --version` >= 13.
- [ ] On `main`, clean working tree, CI green for the commit you're shipping (`git status` clean, `gh run list -L1` success).
- [ ] Node 24 active (`node -v` → `v24.*`) — Functions runtime is pinned to nodejs24.
- [ ] `ANTHROPIC_API_KEY` set as a Functions secret in `quillpad-prod` (see Troubleshooting if unsure).
- [ ] Default project selected: `firebase use quillpad-prod`.

## Procedure

1. **Confirm you're on the right project and commit.**

   ```bash
   firebase use quillpad-prod && git log -1 --oneline
   ```

   _Expected: `Now using project quillpad-prod` and the SHA you intend to ship._

2. **Install and build the web bundle + functions.**

   ```bash
   npm ci
   npm run build          # vite build → dist/, tsc functions → functions/lib/
   ```

   _Expected: `dist/` and `functions/lib/` written, no type errors._

3. **Deploy Functions first** (so the new backend is live before the site that calls it).

   ```bash
   firebase deploy --only functions
   ```

   _Expected: `✔ functions: Finished running predeploy script` then a `Function URL` line per function (`capture`, `editorChat`)._

4. **Deploy Hosting.**

   ```bash
   firebase deploy --only hosting
   ```

   _Expected: `✔ Deploy complete!` and a `Hosting URL: https://quillpad-prod.web.app`._

5. **Tag the release.**

   ```bash
   git tag -a "deploy-$(date +%Y%m%d-%H%M)" -m "prod deploy" && git push --tags
   ```

   _Expected: tag pushed; this is the marker you roll back to._

## Verification

Hit the live capture endpoint and the site, independent of the deploy output:

```bash
curl -s -o /dev/null -w "%{http_code}\n" https://quillpad-prod.web.app/
curl -s https://us-central1-quillpad-prod.cloudfunctions.net/capture/health
```

_Success looks like: the site returns `200`, and `/capture/health` returns `{"ok":true,"model":"claude-..."}`. Then load `https://quillpad-prod.web.app`, sign in with a magic link, and confirm the editor's ⌘K assist streams a response._

## Rollback

Roll the **Hosting** site back instantly; Functions roll back by redeploying the prior tag.

1. **Roll back Hosting to the previous release** (no rebuild — Firebase keeps prior versions):

   ```bash
   firebase hosting:rollback
   ```

   _Expected: previous version goes live within seconds._

2. **Roll back Functions** to the last good tag, if the backend is the problem:

   ```bash
   git checkout <previous-deploy-tag>
   npm ci && npm run build
   firebase deploy --only functions
   ```

   _Point of no return: a Firestore schema/data migration run as part of a release is **not** covered here — those don't auto-roll-back. Quillpad has none today; if one is ever added, gate it behind its own runbook._

## Troubleshooting

| Symptom | Likely cause | Fix |
|---------|--------------|-----|
| `/capture/health` returns 500, logs show `missing ANTHROPIC_API_KEY` | Secret not set on the prod project | `firebase functions:secrets:set ANTHROPIC_API_KEY` then re-run step 3 |
| `firebase deploy` hangs on `predeploy` | Stale `functions/lib/` or wrong Node | `rm -rf functions/lib && nvm use 24 && npm run build`, redeploy |
| Editor ⌘K assist is slow on first use after deploy | Functions cold start (RAID R-003) | Expected for the first request post-deploy; hit `/editorChat/health` once to warm it |
| `HTTP 412 Precondition Failed` on hosting deploy | Wrong project selected | `firebase use quillpad-prod`, re-run step 4 |
| `permission denied` deploying functions | `firebase login` expired | `firebase login --reauth`, retry |

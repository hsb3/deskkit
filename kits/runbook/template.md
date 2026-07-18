---
type: runbook
status: draft
created: {{date}}
updated: {{date}}
tags: []
author:
owner:
---

# Runbook: {Procedure name}

_A repeatable operational procedure. The reader should be able to execute this start-to-finish by copy-pasting the commands in order — no judgement calls, no "figure out the right value here." If a step needs a decision, the runbook is missing a step._

## Purpose / when to run this

_One or two sentences: what this procedure does and the trigger that calls for it (a deploy, a key rotation, a recovery from a specific failure). State who is expected to run it._

## Prerequisites

_Everything that must be true before step 1. Access, installed tools with versions, env vars, branch state. A reader who fails here should stop, not improvise._

- [ ] {Access / role you need — e.g. `gcloud` authed as the deploy SA}
- [ ] {Tool + version — e.g. `node >=24`, `supabase` CLI linked to the right project}
- [ ] {Env / secret present — e.g. `ANTHROPIC_API_KEY` exported, `.env.local` populated}
- [ ] {Starting state — e.g. on `main`, clean working tree, CI green}

## Procedure

_Numbered, ordered, idempotent where possible. Each step = one action with the exact command in a fenced block, plus what you expect to see. Never "deploy the app" — show the command._

1. **{What this step does}**

   ```bash
   {exact command}
   ```

   _Expected: {the output / state that confirms the step worked}._

2. **{Next step}**

   ```bash
   {exact command}
   ```

   _Expected: {…}._

3. **{Next step}**

   ```bash
   {exact command}
   ```

   _Expected: {…}._

## Verification

_How to prove the procedure succeeded — independent of the steps above. A command whose output confirms the end state, not just "it ran without error."_

```bash
{verification command}
```

_Success looks like: {concrete observable — an HTTP 200, a row count, a version string}._

## Rollback

_How to undo, if verification fails or something breaks. Numbered, with commands. If rollback is impossible past a certain step, say so explicitly and name the point of no return._

1. **{Undo step}**

   ```bash
   {exact command}
   ```

## Troubleshooting

_Known failure modes. Symptom → fix, so the runner doesn't have to debug from scratch._

| Symptom | Likely cause | Fix |
|---------|--------------|-----|
| {What you see when it goes wrong} | {Why} | {The command or action that resolves it} |

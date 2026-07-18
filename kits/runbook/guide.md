---
type: sop
status: final
created: 2026-06-30
updated: 2026-06-30
tags: [meta]
---

# SOP — runbook

## When to write one

Write a runbook when an **operational procedure** will be run more than once and you want it to run the same way every time — by you next quarter, or by someone who has never done it. Deploys, key/secret rotations, backup and restore, recovery from a known failure, provisioning a new environment.

The test: **could a competent person who's never done this execute it from the document alone, by copy-pasting?** If yes, it's a runbook. If the procedure needs judgement at every step, you don't have a runbook yet — you have notes.

**Runbook vs. technical-design.** A `technical-design` explains *why* the system is shaped the way it is — architecture, trade-offs, the decisions behind it. A runbook is the repeatable *how* — the exact commands, in order, to perform one operation. Don't put architecture rationale in a runbook, and don't bury an operational procedure inside a design doc. If you find yourself explaining *why* in a runbook step, that sentence belongs in the linked TDD.

## How to write one

1. **Name the trigger.** The Purpose section says what calls for this procedure — a release, an expired key, a downed service. A runbook nobody knows when to reach for is dead weight.
2. **Front-load prerequisites.** Everything that must be true before step 1 — access, tool versions, env vars, branch state — as a checklist. A reader who can't satisfy the prerequisites should stop, not improvise.
3. **Number every step, one action each, with the exact command.** Fenced command blocks the reader can copy verbatim. No "deploy the app" — show `gcloud run deploy …`. Substitute real-looking values, not `<FOO>` placeholders the reader has to decode.
4. **State the expected result per step.** What confirms the step worked — the output, the status, the state. Without it, the runner can't tell a silent failure from success.
5. **Make it idempotent where you can.** A step that's safe to re-run beats one that corrupts state on a retry. Note any step that is *not* safe to re-run.
6. **Always include Verification, Rollback, and Troubleshooting.** Verification proves success independently of the steps. Rollback undoes the change (and names the point of no return if there is one). Troubleshooting is a symptom→fix table for the failures you already know about. A runbook without these three is half-written.

## Status transitions

| From | To | Trigger |
|---|---|---|
| `draft` | `in-review` | Procedure drafted; sent for review or a dry-run |
| `in-review` | `approved` | Run successfully end-to-end at least once; accepted as the canonical procedure |
| any | `archived` | Procedure obsolete (system retired, replaced by a new runbook); kept as record |

## Anti-patterns

- **Narrative prose instead of copy-pasteable steps** — "first you'll want to push the image, then update the service" is a story, not a runbook. Number it and show the commands.
- **Placeholder soup** — `<PROJECT_ID>`, `<REGION>`, `<TOKEN>` scattered everywhere with no legend forces the runner to reverse-engineer every value. Use real-looking defaults and call out the one or two things they must change.
- **No verification** — "run the deploy" with no way to confirm it worked. The runner can't distinguish success from a silent failure.
- **No rollback** — every mutating procedure needs an undo, or an explicit statement that there isn't one and where the point of no return is.
- **Missing prerequisites** — the procedure assumes you're already authed / on the right branch / have the tool installed, and fails cryptically at step 3 instead of stopping at step 0.
- **Why instead of how** — architecture rationale belongs in the technical-design; a runbook step that explains *why* the system works this way has drifted out of scope.
- **Write-once and stale** — a runbook whose commands no longer match reality is worse than none, because it's trusted. Re-run it (or dry-run it) when the system changes.

## Example

See `example.md` in this folder for a worked runbook — deploying the Quillpad project to Firebase.

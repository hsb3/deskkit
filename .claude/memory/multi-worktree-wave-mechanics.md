---
name: multi-worktree-wave-mechanics
description: Hard-won mechanics for running parallel PR lanes via .claude/worktrees in this repo
metadata: 
  node_type: memory
  type: project
  originSessionId: d5b00ade-8ece-4e30-8ebc-c5eaac3be228
---

Lessons from the 2026-07-20 v1 build wave (10 lanes, PRs #136–#146), beyond what the repo docs record:

**Why:** parallel worktree lanes + a merge queue create failure modes single-branch work never hits.

**How to apply:**
- `git fetch` updates `origin/*` only — **pull local `main`** before any scout/agent reads the main tree, or brief-vs-tree contradictions follow (a scout's stop-condition caught a 3-merge-stale tree; incident in `_meta/HANDOFF.md` §5).
- Every lane's CHANGELOG bullet top-inserts under `[Unreleased]` → the second-to-merge PR ALWAYS conflicts there. Rebase with incoming-bullet-on-top; spec/doc edits in different sections auto-merge fine.
- A conflicting PR reports **"no checks reported"** (GitHub can't build the merge commit, so `pull_request` workflows never fire) — check `mergeStateStatus` before blaming CI.
- `gh pr checks --watch` exits 1 if run before checks register; sleep ~30-45s after push before watching.
- Chain `gh pr merge` → worktree/branch cleanup with `&&`, never `;` (a failed merge otherwise still deletes the local branch; recovery = re-checkout from the remote branch).
- Shared append points across lanes (`Makefile check:` recipe, CLAUDE.md checks table, ci.yml steps): let each lane make a minimal append, have it quote the exact hunk in its handoff, and resolve at merge time — cheaper than pre-serializing every lane.

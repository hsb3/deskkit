---
name: scope-vs-worktree-gate-attribution
description: A repo-wide gate (neutrality, make check) can be RED from uncommitted files OUTSIDE the reviewed diff; scan git status before attributing a gate failure to the change under review
metadata:
  type: project
---

When reviewing a scoped change (e.g. #91 = engine.go + engine_test.go), the working tree
often carries OTHER uncommitted edits that a repo-wide gate scans too. Real example
(2026-07-20): `node scripts/check-neutrality.mjs` failed with 6 `#67`/`#93` violations, but
ALL were in `librarian/cmd/deskkit/main.go` + `main_test.go` — uncommitted changes NOT in the
reviewed diff, and appearing AFTER the session-start `git status` snapshot (that snapshot is
frozen; re-run `git status --short` live). The two engine files under review were clean.

**How to apply:** before blaming a RED aggregate gate on the change under review, run
`git status --short` live and `grep -n` the ACTUAL target files. Attribute the failure to the
files that carry it. A RED gate is still worth reporting (it blocks CI/`make check`), but
classify it: defect-in-reviewed-change vs pre-existing/adjacent-uncommitted.

Also reinforced here: piping the gate masks its exit code. `check-neutrality.mjs | tail -5;
echo $?` printed `0` (tail's code) while the script actually exited 1. Run gates BARE. See
[[claudemd-count-drift]].

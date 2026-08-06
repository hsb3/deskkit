---
name: pm-pointer-overload-and-gate-path
description: desk-standard PM pointer field is overloaded (GitHub-URL mirror vs document-gate file path); the :// reject lives in librarian/module.go, NOT pm/gates
metadata:
  type: project
---

The PM `pointer` field is semantically overloaded, and builders keep misciting where the gate enforcement lives.

**Fact 1 — the `://` reject is in the LIBRARIAN module, not pm/gates.**
`librarian/internal/modules/librarian/module.go` (`DocumentValidator.Verdict` ~:126 and
`Frontmatter` ~:192, re-verified 2026-08-05) does `strings.Contains(file, "://")` → fail
"pointer is not a desk file." There is NO `librarian/internal/modules/pm/gates/module.go` —
that file does not exist. The pm gates package is `defaults.go` / `gates.go` only. The
document-gate validator is supplied by the librarian module via the engine seam.

**Fact 2 — the pointer field is overloaded.**
- Spec `docs/development/specs/pm-system-v1-spec.md` (R6.1): an item MAY carry a `pointer` to a
  GitHub issue URL (read-only mirror).
- But shipped default gates (`pm/gates/defaults.go:28,36`) use `pointer: item` for both
  `decision` (review->terminal) and `task` (work->review). So a decision/task item whose pointer
  is a GitHub URL fails its default gated transition fail-closed.
- `pm/collections/collections.go` — `pointer` is a plain TextField, stores anything. Nothing
  distinguishes a URL-mirror pointer from a file-gate pointer. The only escape is ruling gates
  to use `note:<key>` instead of `item`.

**Why:** a builder framed this as a contradiction and cited the wrong path. The contradiction is
real (arguably broader — it hits shipped defaults), but any citation to `pm/gates/module.go` is wrong.

**How to apply:** When verifying any PM claim about pointer/URL/gate enforcement, read
`librarian/internal/modules/librarian/module.go` (the validator) + `pm/gates/defaults.go` (the
shipped rules) + `pm/engine/engine.go` pointerResolver. Related: [[scope-vs-worktree-gate-attribution]].

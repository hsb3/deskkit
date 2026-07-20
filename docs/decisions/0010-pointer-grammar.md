# 0010 · Pointer grammar — ratify shipped behavior; URL refs are not gate pointers

_Settles what a PM `pointer` may be and what gates do with each form._

- **Status:** Accepted
- **Date:** 2026-07-20
- **Raised by:** decision book `_meta/research/2026-07-design-session/decision-book/D1-pointer-grammar.md` (exec-desk agenda §3.2; #102's proper fix)

## Context

The shipped grammar was implemented but specified nowhere: a pointer resolves by its
desk-relative file path; a `§ heading` suffix is advisory (gates deliberately ignore it so
heading renames cannot break transitions); `#` anchors and `://` scheme-bearing strings fail
closed with actionable hints. Meanwhile the PM spec (line ~744) promised GitHub-issue-URL
pointers — a contradiction that bites the shipped default decision/task gates (independently
re-derived; decision-book Verification record, claim 2).

## Decision

**Ratify the shipped behavior and specify it** (D1 Option A, owner-ruled 2026-07-20):

- A gate pointer is a desk-relative file path, optionally suffixed `§ <heading>`; the suffix
  is advisory and never checked by gates. `#` anchors and URLs fail closed with their hints.
- **Issue and URL references are not gate pointers.** They belong to the typed reference
  contract (ADR 0011). The PM spec's URL-pointer sentence is corrected accordingly.
- Real sub-file addressing (stable section identity) is deferred until a use case pulls it;
  rename tolerance is the standing priority.

## Consequences

- The grammar gets a normative spec section; the validator's behavior is now by-ruling, not
  by-accident. No code change to `Verdict`/`sectionFilePart` is required.
- Items that want to reference an issue/URL use a typed reference field, not `pointer` —
  the field stops being overloaded.
- Revisit only if a pulled use case needs sub-file loci; that reopens the anchor-id design.

## Affects

`docs/pm-system-v1-spec.md` (pointer prose incl. the corrected URL sentence) ·
`librarian/internal/modules/librarian/module.go` (`Verdict`, `sectionFilePart` — behavior
pinned by tests, now cited) · `librarian/internal/modules/pm/gates/defaults.go` · ADR 0011.

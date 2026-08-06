---
name: ruled-not-shipped-overstatement
description: design-session issue bodies overstate ADR status — an ADR that RULED a fix is written as if the fix already "shipped/fixed"; cross-check ADR status + build-plan gate label
metadata:
  type: project
---

Recurring defect in the 2026-07-20 design-session `_meta/plans/*/issue-body.md` set: an issue
body cites an ADR as having "**already found and fixed**" something when the ADR only FOUND it
and RULED a fix that is still a pending build slice.

Real example (2026-07-20): `_meta/plans/schema-versioning/issue-body.md:22-26` — "the naming
collision **ADR 0017 already found and fixed once** (`files.entity_type` vs the schema's
`entity_type` enum)". Reality: ADR 0017 Context(b)+Decision(b) FOUND the collision and RULED a
column rename via forward migration, but the rename is an UNSHIPPED v1-epic child
(`document-identity-hygiene`, build-plan.md:29, gate:1.0.0). The cited source
(`data-model.md:324-343` / §5.2) describes it present-tense as LIVE and unresolved and never
mentions ADR 0017. The sibling `typed-reference-contract/issue-body.md:35` correctly calls the
SAME collision "the repo's **live** naming collision" — a direct cross-body contradiction.

**How to apply:** when an issue body says an ADR "fixed/closed/shipped" something, verify against
(1) the ADR's own Decision vs Consequences wording (a Decision is a RULING, not a landed change),
(2) whether a build-plan slice / epic child still tracks it (gate label = not yet shipped), and
(3) the cited evidence doc's tense. "Ruled" ≠ "shipped". Also diff sibling bodies describing the
same artifact — one calling it "fixed" and another "live" is the tell. See
[[claudemd-count-drift]] (re-run/re-check, never trust the self-report).

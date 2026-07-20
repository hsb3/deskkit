---
type: journal
status: active
created: <YYYY-MM-DD>
updated: <YYYY-MM-DD>
tags: [self-maintenance, friction-ledger]
synopsis: "Desk self-maintenance ledger — the maintenance backlog, pass log, and instruction-friction ledger the harvest loop reads (K21)."
---

# Improvement log — <desk name>

<!--
  Per-desk friction-ledger template (K21). Copy into a desk as _meta/improvement-log.md, then
  fill in the placeholders — <desk name> above and <YYYY-MM-DD> in the frontmatter (created =
  the copy date; updated stays static unless the ledger's structure itself changes — appending
  M-nn/INS-nn entries never requires touching frontmatter, so the instantiated file is
  conformant with the frontmatter contract (K3) from the moment it's copied in).
  One file, three ledgers. The harvest loop (harvest-loop skill) folds the INS-nn entries into
  versioned revisions of the conventions standard; absorbed entries are marked resolved with a
  pointer to the revision that absorbed them.
-->

## Maintenance backlog (M-nn)

_Desk-local upkeep. One line per open item; mark resolved with a pointer, never delete.
Mostly not harvested — these are this desk's own to-dos._

- **M-01 — <short title>.** <what needs doing and why> _Status: open._

## Pass log

_What each maintenance/de-noising pass did, dated. De-noising is trigger-based, never
scheduled — a pass fires on an event (a workstream closes, a rule change touches 2+ files, a
duplicate is noticed)._

- **<YYYY-MM-DD> — <pass name>.** <what the pass changed>.

## Instruction-friction ledger (INS-nn)

_The harvest target. Each entry: where a convention rubbed against real work, and the rule
change that would fix it. When the harvest loop absorbs an entry, mark it resolved here with a
pointer to the revision that absorbed it._

- **INS-01 — <what rubbed>.** <the friction, concretely> → <the rule change that would fix it>.
  _Status: open._ <!-- when harvested: Status: absorbed by <plugin vX.Y.Z / revision ref>. -->

---
type: journal
status: active
created: <YYYY-MM-DD>
updated: <YYYY-MM-DD>
tags: [self-maintenance, friction-ledger]
synopsis: "Desk self-maintenance ledger — the maintenance backlog, pass log, and instruction-friction ledger the harvest loop reads (K21)."
---

# Improvement log — {{profile.desk.name}}

<!--
  The desk's single self-maintenance file (K21). Three ledgers in one file. The harvest loop
  folds the friction-ledger entries into versioned revisions of the conventions standard.
  Seed stub — replace the placeholders as real entries arrive, including the frontmatter's
  <YYYY-MM-DD> (created = the day this scaffold was materialized; updated stays static unless
  the ledger's structure itself changes — appending M-nn/INS-nn entries never touches
  frontmatter, so this file is conformant with the frontmatter contract (K3) as instantiated).
-->

## Maintenance backlog (M-nn)

_One line per open maintenance item; mark resolved with a pointer, don't delete._

## Pass log

_What each maintenance/de-noising pass did, dated. De-noising is trigger-based, never scheduled._

## Instruction-friction ledger (INS-nn)

_Convention-friction findings from live work — the entries the harvest loop harvests. Each:
what rubbed, and what rule change would fix it. Mark absorbed entries resolved with a pointer to
the revision that absorbed them._

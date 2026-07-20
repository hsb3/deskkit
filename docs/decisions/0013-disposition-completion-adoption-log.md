# 0013 · Findings lifecycle completed — provenance on the finding, adoption log shrunk

_Finishes the disposition sub-machine #93 started and rules what the adoption log is for._

- **Status:** Accepted (corrected 2026-07-20)
- **Date:** 2026-07-20
- **Raised by:** decision book `_meta/research/2026-07-design-session/decision-book/D4-disposition-and-adoption-log.md` (exec-desk agenda §3.4)

> **Correction (2026-07-20):** the migration DIRECTION was mislabeled below as a "down-remap".
> Retiring an enum value makes the *shrink* the FORWARD step, so the data-first remap (move rows
> off `dismissed`, then drop it from the enum) belongs in the **forward (up)** migration; the
> **down** migration merely re-adds `dismissed`. This is the *mirror* of the 0012 extension
> precedent (where the shrink — and thus the remap — lives in the down migration), not a literal
> copy of it. The decision is unchanged; the `findings-lifecycle-completion` plan already builds
> it correctly (remap in FORWARD, plan.md Q1). The struck phrases below are corrected here.

## Context

#93 shipped `disposition` (open/acknowledged/triaged/wont_fix) orthogonal to `state`, with
re-patrol inheritance. Residue: `state` still declares `dismissed` with no setter anywhere;
`query summary`/`uncollapsed` counts ignore disposition (so "open findings" disagrees across
surfaces); dispositions carry no who/when/why; and 5 of 6 `adoption_log.event` values have
no writer. The log is NOT dead weight: it has live readers (`query adoption` on CLI/MCP/TUI)
and is third in deskguard's desk-collision set (independently re-derived; Verification
record, claim 1).

## Decision

**Provenance on the finding; shrink the log** (D4 Option 2, owner-ruled 2026-07-20):

- Retire `dismissed` from the `state` enum (forward migration; ~~data-first down-remap~~
  data-first remap-then-shrink IN THE FORWARD migration, mirroring the 0010/0012 precedent — see
  the Correction above). `disposition` is the one human-judgment axis.
- **"Open findings" means one thing everywhere**: `query summary` / `uncollapsed` (and the
  MCP/TUI surfaces that read them) become disposition-aware, matching `query findings`'
  live default.
- Add provenance to `patrol_findings`: actor, reason, disposed-at — set by `findings
  dispose` (CLI) and inherited consistently on re-patrol.
- Shrink `adoption_log` to writer-backed reality: keep `fix` (and its readers + deskguard
  role); drop the five writerless event values via forward migration with a data-first down
  path. Wiring new events happens only when a concrete consumer pulls them.
- Under the standing files-are-truth regime (ADR 0009), disposition provenance is store-only
  supervisor state and does not survive a store rebuild — **accepted**; re-opens if the 0009
  gate flips.

## Consequences

- The counts divergence closes; the dead enum value stops implying an unbuilt feature.
- Dropping enum values is a shipped-collection change — forward migration only, ~~with the
  enum-extension down-remap pattern~~ applying remap-then-shrink IN THE FORWARD migration (the
  0012 precedent runs remap-then-shrink in its *down* migration because there the shrink is the
  down step; retiring an enum makes the shrink the forward step — see the Correction above).
- Each slice carries a red-able regression test (#82 bar); data-safety slices get
  independent adversarial review (the PR #112 practice).

## Affects

`librarian/internal/modules/librarian/collections/` (new forward migration) ·
`tools/query.go` (summary/uncollapsed) · `tools/dispose.go` (provenance) · `patrol.go`
(inheritance) · TUI/MCP count surfaces · `docs/pocket-librarian-v1-spec.md` §5 · the
adoption-log spec prose.

# 0021 · desk-standard 1.0.0 — release direction (graduated from the executive desk)

_Graduates the off-repo "decision 0021" that several 1.0.0 lanes cite by fork label (F1–F7) into
an in-repo record, so a fresh reader or agent can find the ruling the repo's own provenance rule
requires. Records each fork's direction, what it amends, and the 2026-07-21 owner sign-off that
supplied the still-missing pieces for F2, F5, and F7._

- **Status:** Accepted (2026-07-21)
- **Date:** 2026-07-21
- **Raised by:** the paired executive desk's decision `0021-desk-standard-1-0-0-direction` (its
  decision spine, off-repo — see Provenance); graduation requested by issue #170 (the dangling-F5
  citation found at triage) and ruled by the 2026-07-21 owner sign-off (item 10, "graduate it
  wholesale into a repo decision record").

## Context

The path to **1.0.0** was ruled on the paired executive desk on 2026-07-19 as a single decision,
numbered **0021** there, splitting into seven forks (F1–F7). Several in-repo artifacts cite those
forks as their authority:

- issue #83 (PM default-on) cites "decision 0021 fork F2";
- issue #84 and issue #170 (the `_knowledge/` move) cite "decision 0021 fork F5";
- issue #88 (cohesion + value evaluation) cites "decision 0021 fork F7";
- the design-session research dossier (`_meta/research/2026-07-design-session/platform/*`) cites
  `0021`, `0021`-F3, and `0021`-F7;
- `_meta/HANDOFF.md` cites "decision 0021 F5".

But the repo's own decision records stop at 0020, so every one of those citations dangles: the
repo's rule is that **decisions bind from `docs/decisions/`, not from off-repo process
documents** (issue #170 flagged this at triage — "no decision 0021 exists in-repo"). The forks
also carry contract amendments (to ADR 0001 and ADR 0008) and a deviation (from ADR 0005) that a
reader must be able to trace. This ADR is the graduation the provenance rule requires; it records
the direction, not the code-repo ADR edits (those are filed by their own lanes when each builds).

## Decision

Graduate the executive desk's decision 0021 **wholesale** into this record (owner sign-off item
10). The seven forks, each quoted from the source (see Provenance) with its in-repo citations and,
where the 2026-07-21 sign-off ruled a still-open piece, that update:

**F1 — 1.0.0 is a MATURITY milestone.** "It means feature-complete-enough to trust and require,
not necessarily a breaking contract change." ADR 0005 ties MAJOR to a breaking product-contract
change, so "badging 1.0.0 for maturity is a deliberate **deviation** — it must be recorded
(CHANGELOG `[1.0.0]` rationale + a dated correction/note on ADR-0005 in the code repo)." _Not
cited in-repo by fork label; enacted by the release-cut lane._

**F2 — the PM module goes DEFAULT-ON for 1.0.** "For 1.0, every desk gets PM. This **amends
ADR-0008's default** and is contingent on the first-adoption bugs being fixed first." Its
consequence becomes 1.0 scope: "the seeded gate/label defaults … ship to every desk, so they get
re-ruled/blessed before 1.0." _Cited by issue #83._
- **Owner sign-off (2026-07-21):** approved — flip PM on by default for 1.0 **and bless the
  shipped seed as-is** (the two gate rules — a decision needs review before terminal, a task needs
  review before done — and the status vocabulary `backlog/next → active → in-review →
  done/dropped/superseded`, plus `blocked`/`waiting` flags). The seed survived a real adoption
  dry-run and ships to every desk unchanged. ADR 0008's ship-dark record gets its dated amendment
  when that lane builds.

**F3 — the webapp is a pre-1.0 build lane.** "The PocketBase-served chat surface moves out of
post-1.0 deferral into 1.0 scope. This **amends desk-standard ADR-0001**." Clarified same session:
"the webapp is a **stub, sequenced dead-last** … blocked by every other 1.0 lane and precedes only
the release cut." The React-vs-Go-route sub-decision "is made when the lane starts, not here."
_Cited by the design-session dossier; the lane (issue #19) is in build._

**F4 — the 1.0 blocker floor is the four first-adoption bugs:** #78, #79, #80, #67. "All cheap;
three are credibility/safety-grade. Ratified as recommended." _Not cited in-repo by fork label._

**F5 — `_knowledge/` gets done right (pre-1.0 refactor).** "Introduce a single shared profile-root
constant/config FIRST (kills the cross-language drift risk), THEN move `_knowledge/` in one
coordinated rename. Not a `git mv`." _Cited by issues #84 and #170 and by `_meta/HANDOFF.md`._
- **Owner sign-off (2026-07-21):** the move is **approved in principle, but the target is still
  unnamed** — the sign-off's "move it" choice was submitted with an empty target field, and triage
  had already found the destination was never ruled anywhere. Record this fork as
  **approved-pending-target**: the shared profile-root constant groundwork has landed (one
  authoritative constant across schema/TS/Go, guarded by a drift check), so the rename is a small
  low-risk sweep, but it stays **gated by issue #170** until the owner names the target folder.
  Until then `_knowledge/` stands.

**F6 — conformance for 1.0.** "Run the `mise-en-place-scaffold` …, author the three design-bearing
docs (**CLAUDE.md, AGENTS.md, docs/CHARTER.md**), and rule the root **`tests/`** question (author a
real `tests/` … or add a standard-level exemption)." _Not cited in-repo by fork label._

**F7 — system cohesion + value evaluation is a pre-1.0 requirement.** Before 1.0, testing +
evaluation must confirm the whole system "forms a **cohesive system that behaves as expected** AND
delivers **significant value over (a) a naive approach and (b) an off-the-shelf alternative**,"
via "a value evaluation with a rubric + those two baselines and an honest win/tie/lose verdict."
_Cited by issue #88 and the design-session dossier._
- **Owner sign-off (2026-07-21):** approved the lane's scoping — **baseline** = a hand-maintained
  markdown folder with manual hygiene (the naive approach); **comparator** = Notion (docs +
  databases + templates, the closest mainstream analogue); **rubric** = three dimensions (session
  continuity, gate-enforced work state, safe automated hygiene), each scored **win / tie / lose**
  honestly. A losing or tied dimension is a finding that feeds back into scope before 1.0.

**What 0021 does NOT change** (recorded for completeness): the Claude-only v1 scope (Claude +
OpenCode dual-format stays post-1.0), the SOP-library and the LATER PM capabilities stay post-1.0,
and the librarian v1 scope boundary (ADR 0014) is untouched.

## Consequences

- **The dangling citations now resolve here.** Any artifact citing "decision 0021 / fork FN" points
  to this ADR; the epics whose provenance lines named the off-repo decision (#83/F2, #84 &
  #170/F5, #88/F7) should have those lines re-pointed at `docs/decisions/0021-…` — an out-of-repo
  edit to issue bodies, tracked as follow-up, not performed by this doc.
- **The contract amendments are still enacted by their own lanes.** This ADR records direction; it
  does **not** itself edit ADR 0001 (webapp deferral), ADR 0008 (PM ship-dark default), or ADR
  0005 (MAJOR = breaking contract). Those dated amendments/corrections land in the same change as
  each lane's build, citing this record.
- **F5 stays gated.** The `_knowledge/` move cannot start until the owner names a target
  (issue #170); the shared-constant half is already done and is independent of the move.
- **F2 and F7 are unblocked to build** on their blessed seed / approved scoping; F3 is in build as
  the dead-last stub.
- If any fork's premise is later found wrong, repair per the ADR discipline (supersession if the
  *decision* changed; a dated in-place correction if a *factual premise* was wrong) — never leave a
  falsified claim readable as current.

## Affects

The 1.0.0 lane epics (#83 PM default-on · #84 `_knowledge/` · #88 cohesion+evaluation · #19 webapp
stub · #170 the `_knowledge/` move-target gate) · [ADR 0001](0001-interactive-surface-tui-first.md)
(webapp deferral, amended by F3) · [ADR 0005](0005-versioning-and-changelog.md) (MAJOR semantics,
deviated by F1) · [ADR 0008](0008-pm-core-modules-architecture.md) (PM ship-dark default, amended
by F2) · [ADR 0014](0014-agent-integration-contract.md) (v1 scope boundary, untouched).

## Provenance

- **Executive-desk source (source-by-reference, off-repo):** the paired executive desk's decision
  spine, `…/desk-standard-desk/_structure/decisions/README.md` records decision 0021 as an
  **inherited record** governed from the sibling `dotfiles-agents-desk` (formerly `dev-tooling-desk`);
  the document itself is
  `…/dotfiles-agents-desk/_structure/decisions/0021-desk-standard-1-0-0-direction.md` (status
  `accepted`, decided 2026-07-19), with the release-sequenced roadmap at
  `…/desk-standard-desk/analyses/desk-standard-1-0-0-roadmap-2026-07-19.md`. All F-fork quotes above
  are lifted from that document; it remains the authoritative source for anything not restated here.
- **Owner sign-off batch:** `_meta/signoff/2026-07-21-decision-queue/` (`answers.json` +
  `index.html`) — item `g0021` = `graduate-wholesale` (this graduation), and the pieces it says
  "supply the still-missing pieces": item `pm` = `flip-on-bless-defaults` (F2), item `move` =
  `move-to-target` with an empty target (F5, approved-pending-target), item `cohesion` =
  `approve-scoping` (F7).

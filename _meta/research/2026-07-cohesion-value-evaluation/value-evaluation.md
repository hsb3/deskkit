# Value evaluation — desk-standard vs a naive markdown desk and vs Notion

_Purpose: apply the owner-approved 1.0.0 value rubric and deliver an honest win/tie/lose verdict.
A tied or losing dimension is a finding that feeds back into 1.0 scope, not a failure to hide._
Status: active — 2026-07-21

**Scope ruling (owner sign-off 2026-07-21, ADR `docs/decisions/0021-desk-standard-1-0-0-direction.md`
fork F7; recorded in `_meta/signoff/2026-07-21-decision-queue/answers.json` item `cohesion`):**

- **Baseline (naive):** a hand-maintained markdown folder with **manual** hygiene.
- **Comparator (off-the-shelf):** **Notion** — docs + databases + templates, the closest mainstream analogue.
- **Rubric:** three dimensions — **session continuity**, **gate-enforced work state**, **safe
  automated hygiene** — each scored **win / tie / lose** against **each** baseline, honestly.

Every claim below is grounded in a behaviour the E2E suite (`librarian/e2e/e2e.sh`) actually
exercises; the deep gates that pin the same behaviours are `librarian/verify.sh` (fix chain) and
`librarian/dogfood-pm.sh` (PM graph).

---

## Two-minute verdict

| Dimension | vs naive markdown desk | vs Notion |
|---|---|---|
| 1 · Session continuity | **WIN (narrow)** | **WIN (on the agent/terminal axis)** |
| 2 · Gate-enforced work state | **WIN (decisive)** | **WIN** |
| 3 · Safe automated hygiene | **WIN (decisive)** | **WIN (by default — Notion is out of this space)** |

**Bottom line.** desk-standard beats both baselines on all three claimed dimensions, but the wins
are not equal and honesty matters more than a clean sweep:

- The **sharp, earned, head-to-head wins are dimensions 2 and 3.** Gate-enforced work state
  (a real state machine that *refuses* illegal transitions and blocked-item advances) and safe
  automated hygiene (rule detection + mechanical fixes that are byte-exact reversible) are
  categorical capabilities neither baseline has. These are the product's reason to exist.
- **Dimension 1 (session continuity) is the narrowest win and the finding that feeds scope.**
  Against a *disciplined* hand-maintained `HANDOFF.md` — which is the product's *own* recommended
  practice — the marginal gain is incremental: it comes almost entirely from the PM cold-start
  briefing being auto-injected and generated from live state (so it cannot go stale), not from a
  categorically new capability. Against Notion the win holds only on the *agent-in-a-terminal*
  axis; Notion is the stronger surface for a *human* browsing work state.
- **Two of the three "wins vs Notion" are partly uncontested rather than earned in a fair fight.**
  Notion does not operate on a git-tracked markdown desk at all, so dimension 3 is a win by
  forfeit, and half of dimension 1's win is "Notion was never designed to inject state into a
  terminal AI session." This is itself a finding: **Notion is a fair comparator for continuity and
  work-state, and a poor one for repo hygiene** — the honest off-the-shelf contest is dimensions
  1-2, and desk-standard wins those on its target workflow.

---

## The three dimensions in detail

### Dimension 1 — Session continuity

**What the product does (E2E-proven):** `deskkit pm context` returns a structured cold-start
briefing (active / blocked / stalled / recent transitions / counts) computed from live work-graph
state; the plugin's `SessionStart` hook (`plugin/desk-persona/hooks/session-briefing.sh`) injects
that briefing into a new agent session automatically, and self-gates to a silent no-op when PM is
off or the store is unreachable. The desk template also ships a `_meta/HANDOFF.md`.

- **vs naive markdown → WIN (narrow).** A well-kept `HANDOFF.md` is genuinely good continuity —
  it is exactly what the product's own protocol recommends, so the product does not *replace* it,
  it *supplements* it. The earned delta is three things a hand-maintained note cannot give:
  (a) the briefing is **auto-injected** at session start (no "remember to read the handoff"),
  (b) it is **generated from live state**, so it cannot drift or lie about what is blocked/stalled,
  and (c) it is **machine-queryable** by an agent. Honest caveat: for a disciplined solo human, a
  fresh HANDOFF closes most of this gap, which is why this is a *narrow* win.
- **vs Notion → WIN, on the agent/terminal axis.** Out of the box, starting a terminal AI coding
  session gets **nothing** from Notion; bridging Notion state into that session requires the API
  plus custom glue. desk-standard is built for exactly that workflow and injects state with zero
  glue. Honest caveat: for a *human* browsing continuity (linked databases, search, mobile),
  Notion's surface is richer — the win is scoped to the claimed workflow (an AI agent resuming
  work in a terminal), not to human PM ergonomics.

### Dimension 2 — Gate-enforced work state

**What the product does (E2E-proven):** items move through a fixed phase machine
`queue → work → review → terminal`; an illegal skip is **refused** (E2E asserts
`queue → terminal` returns "no legal transition", non-zero exit); a **blocked** item's advance is
**refused** until it is unblocked (E2E asserts a `blocks` edge makes the blocked item's transition
fail); the document-gate requires review before a terminal state. The refusal is enforced in the
shared tool core, so it holds identically across every surface (CLI, MCP, agent), not just in a UI.

- **vs naive markdown → WIN (decisive).** A markdown folder enforces nothing. You can type
  `status: done` on a blocked, unreviewed item and no one and nothing objects. This is
  enforcement vs convention — a categorical gap the naive approach cannot close without becoming
  the product.
- **vs Notion → WIN.** Notion databases have status fields, select options, relations, and even
  automations — but they **advise, they do not gate**. You can drag any card to Done regardless of
  its blocking relations; Notion has no "illegal transition" that hard-refuses a human, and its
  relations do not block. desk-standard's gate is a hard refusal with a non-zero exit at the tool
  core. Honest caveat: Notion's automations can *nudge* toward a process, and for a team the
  visible board + permissions may matter more than machine-enforced state — but on the literal
  dimension ("gate-**enforced**"), Notion displays state and desk-standard enforces it.

### Dimension 3 — Safe automated hygiene

**What the product does (E2E-proven):** `patrol` detects rule violations (R1 missing frontmatter,
R3 type/location mismatch, and R2/R4-R6) automatically; `propose-fix` records the original bytes
*before* touching anything (no filesystem write at propose time); `apply-fix` performs the
mechanical repair (fills frontmatter, moves a misfiled doc and leaves a pointer stub); and
`restore` reverses any fix **byte-exact** and reopens the finding. The E2E asserts the restored
files are byte-identical (sha256) to their originals — the record-original-first safety boundary
(ADR `docs/decisions/0014-agent-integration-contract.md`) end to end.

- **vs naive markdown → WIN (decisive).** "Manual hygiene" is precisely the labour the product
  automates — and the product adds a safety net the by-hand editor lacks: a **per-fix, byte-exact,
  provenance-tracked reversal**. Honest caveat: disciplined `git` use gives the naive approach a
  *coarse* form of reversibility, but git cannot reverse a single logical fix while reopening its
  finding, nor prove byte-identity per fix — so the win stands.
- **vs Notion → WIN, by default.** Notion is a database, not a filesystem of git-tracked markdown
  with a rule engine; "misfiled markdown", "missing frontmatter", and "graduated-but-not-collapsed
  doc" are not concepts it has. desk-standard wins because Notion does not compete here at all.
  This is the honest finding restated: on repo hygiene, Notion is not a real alternative — which is
  a point *for* the product's existence, but it is a win by forfeit, not a head-to-head result.

---

## Findings that feed 1.0 scope

1. **Lead on gates + hygiene, not continuity, as the headline value.** Dimensions 2 and 3 are the
   decisive, defensible, head-to-head differentiators; dimension 1 is a narrow win over a
   disciplined HANDOFF. Product/marketing framing for 1.0 should centre the enforced work-state and
   reversible hygiene story. _(Non-blocking; scope/positioning finding.)_
2. **Make the PM cold-start briefing earn its incremental value.** Because the continuity edge over
   a good HANDOFF rests almost entirely on the briefing being auto-injected and always-current, the
   briefing should make *blocked* and *stalled* work unmissable at session start (it currently
   returns them as JSON arrays alongside everything else). A more opinionated "here is the one thing
   to look at" render would convert the narrow win into a clear one. _(Candidate 1.0 polish; file
   as an enhancement.)_
3. **Name Notion's scope honestly in any comparison we publish.** Notion is a fair comparator for
   continuity and work-state and a non-comparator for repo hygiene. A published comparison that
   claimed a clean 3-0 sweep without this caveat would be dishonest; the honest claim is "wins the
   contest on its target workflow (agent + terminal + git-tracked markdown desk)." _(Truth-discipline
   finding; no code impact.)_

None of these three findings blocks 1.0 on the "significantly better than naive/off-the-shelf"
requirement: the product clears that bar decisively on two of three dimensions and narrowly-but-
genuinely on the third. There is **no tied or losing dimension** on the claimed workflow.

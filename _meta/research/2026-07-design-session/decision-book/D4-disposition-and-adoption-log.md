---
type: analysis
status: draft
created: 2026-07-20
updated: 2026-07-20
tags: [design-session, decision-book, data-model, workflows, findings-disposition, adoption-log, 1.0.0]
synopsis: D4 decision brief — complete the findings-disposition sub-machine (dead `dismissed`,
  disposition-blind counts, actor/reason provenance, auto-resolution vs human disposition) and
  rule what the `adoption_log` collection is FOR (5 of 6 event values writerless). Model + workflow
  layer; depends on nothing. Options span wire / shrink / kill; no ruling.
---

_Phase-1 decision brief (index: `README.md`; prep: `../README.md`). Informs the session; does not
rule. Evidence resolves to Phase-0 dossier sections beside the prep doc; dossier claims are
hypotheses until re-derived where a ruling binds._

Status: draft (2026-07-20)

> **Platform-stream interaction (2026-07-20 reboot — see `D0-platform-frame.md`):** the
> provenance-durability criterion (must who-disposed-and-why survive a store rebuild?) is
> regime-dependent. Under files-are-truth (the standing regime per D0.b's hypothesis),
> findings/disposition provenance is store-only supervisor state and rebuild loses it — rule
> whether that's acceptable now, and note that a future PB-as-truth gate flip would make
> store-side provenance durable by definition. The cohesion review's two-data-natures split
> (`../platform/system-cohesion-and-datamodel.md` Thread 2) classifies findings as authored
> store-native state, consistent with this framing.

# D4 — Findings disposition sub-machine + what the adoption log is FOR

## 1. The question

**What is the complete findings-disposition sub-machine — its remaining enum hygiene, its
count-coherence contract, and its actor/reason provenance — and what, if anything, is the
`adoption_log` collection actually FOR?**

#93 (in the 0.8.0 bug floor, PR #112) shipped a `disposition` axis
(`open/acknowledged/triaged/wont_fix`) orthogonal to `state`, with re-patrol inheritance keyed on
`(file, rule, checksum)` and checksum re-open. It left a half-built machine: a dead `state.dismissed`
value that no code path sets; count surfaces that disagree on what "open findings" means; no record
of *who* disposed a finding or *why*; and — surfaced only by the Phase-0 inventory, broader than the
#93 analysis knew — an `adoption_log` collection whose `event` enum declares six kinds of which five
have no writer (restore, the literal act of reverting a fix, logs nothing at all). The session must
rule the finished sub-machine, including how patrol's automatic resolution composes with a
supervisor's manual disposition, and settle the adoption log's purpose.

## 2. Why now

The disposition axis is live but incoherent, and the incoherence is the kind that ships silent:

- **A supervisor's triage is invisible to the numbers.** `query findings` respects disposition;
  `query summary` and `query uncollapsed` do not — an acknowledged/triaged/`wont_fix` finding still
  inflates the summary totals and still shows in `uncollapsed`. "Open findings" means two different
  things depending which query you ask. The same `query.go` code backs the MCP surface, so MCP
  inherits the divergence.
- **A dead enum value invites a future bug.** `state.dismissed` reads like a legal state but can
  never be reached; the next author who "sets a finding dismissed" writes a value nothing else
  understands.
- **The disposition record can't answer "who/why."** `DisposeFinding` writes only the new value —
  no actor, no reason, no timestamp — so a `wont_fix` is an anonymous, unexplained verdict.
- **`adoption_log` is a mostly-empty promise.** Its six-value `event` enum implies an adoption/
  friction ledger; only `fix` is ever written. `restore` — a revert — logs nothing, so the `revert`
  value is dead; `false_positive`/`friction`/`note`/`patrol` have no writer either.

Shipped issues that were symptoms: **#93** (disposition lifecycle — shipped the axis, left this
residue) inside the **#112** bug floor. The adoption-log breadth is a Phase-0 finding, not a filed
issue. This is agenda item 4 and prep-doc row C4; it sits at the model + workflow layer and depends
on nothing, so it can be ruled first.

## 3. Evidence

Bullets resolve to Phase-0 dossier sections beside the prep doc; `path:line` cites are lifted from
those dossiers (against post-PR-112 `main` @ `51235f6`), not re-derived here.

- `data-model.md § 1.2 \`patrol_findings\` (0002, altered by 0010, 0014)` — `state` is a
  SelectField `[flagged, dismissed, fixed, resolved]` with **`dismissed` having no setter anywhere**;
  `disposition` (`[open, acknowledged, triaged, wont_fix]`) was added and backfilled to `open` by
  migration 0014, declared **orthogonal** to `state` — a disposed finding stays `flagged` so it
  survives re-patrol (`0002_patrol_findings.go:64`-table; `0014_patrol_findings_disposition.go:9-15,32-66`).
- `data-model.md § 5.1 Declared enum values with no code path that sets them` — every `.Set("state",…)`
  in the repo writes only `flagged`/`fixed`/`resolved` (`patrol.go:93,168`; `apply_fix.go:223`;
  `restore.go:106`; `0010_…go:53`) — `dismissed` is dead. `adoption_log.event`: the only production
  writer is `recordAdoptionLog`, which always writes the literal `"fix"` (`apply_fix.go:270`); no
  other call site writes `adoption_log` at all — 5 of 6 values (`patrol/revert/false_positive/
  friction/note`) are unwired.
- `data-model.md § 5.3 Disposition-blind aggregate queries (C4 residue)` — `query findings` defaults
  to `disposition='open'` (`query.go:401-412`), but `query uncollapsed` (`query.go:381-392`) and
  `query summary` (`query.go:414-424`) call `openFindingRows` with **no** disposition filter, so
  disposed findings still inflate `open_findings_total` and still appear uncollapsed.
- `data-model.md § 3 Identity & keying` — disposition inheritance: a fresh finding on the same
  `(file, rule, checksum)` inherits the most recent prior non-`open` disposition (fail-open to
  `open`), ordered `-patrol_run,-id` (`patrol.go:254-273`); findings dedupe on the finding's **own
  stored** checksum at flag time (`patrol.go:50-51,60-69,246-252`); re-fire-vs-resolve tracks
  `ruleKey{path, rule}`, checksum-independent (`patrol.go:75,161-174,236`).
- `data-model.md § 1.5 \`adoption_log\` (0005)` — `event` SelectField
  `[patrol, fix, revert, false_positive, friction, note]` (`0005_adoption_log.go:15-16`); **no
  relations** — an adoption row cannot point at the finding it concerns.
- `data-model.md § 5.4 Enums confirmed fully wired (for contrast)` — `disposition` all four values
  reachable through `DisposeFinding`'s validated input (`dispose.go:23-28,38-58`): the disposition
  axis is fully wired while `state.dismissed` and the five `adoption_log` events are not.
- `workflows.md § 1.5 Findings dispose — supervised disposition lifecycle (new in the 0.8.0 bug floor)`
  — `deskkit findings dispose <id> --as …` is **CLI-only**, not an MCP/agent tool — the same
  supervised posture as `restore` (`main.go:809-837`); `DisposeFinding` sets
  `patrol_findings.disposition` and **nothing else**, `state` deliberately untouched (`dispose.go:38-59`);
  `query findings` defaults to `disposition='open'`, needs `--include-disposed` to widen (`main.go:773-807`).
- `workflows.md § 1.2 Patrol — flag rule violations (dry-run)` — **resolution**: any open finding
  whose `(path, rule)` did not re-fire this run transitions to `state=resolved` with `resolved_run`;
  a scoped (`--path`) patrol only resolves within its scope (`patrol.go:155-174`); **disposition
  inheritance** carries a supervisor's call across a resolve→re-fire cycle (`patrol.go:254-273`,
  called at `:104`). This is where automatic resolution composes with human disposition.
- `workflows.md § 1.3 propose_fix → apply_fix → restore (the write boundary)` — `apply_fix` appends
  one `adoption_log` row per batch via `recordAdoptionLog` (`apply_fix.go:242-273`); `restore` writes
  `revisions.restored=true` and reopens the finding to `state='flagged'` but appends **no**
  `adoption_log` row (`restore.go:90-116`) — so `revert` has no writer even though restore *is* a revert.
- `workflows.md § 1.4 record_feedback` — `record_feedback` writes the `feedback` collection only
  (DB-only, not gated by autonomous writes) when "a tool fails or a desk convention doesn't fit"
  (`record_feedback.go:22-70`) — the "friction" concept already has its own live collection,
  parallel to `adoption_log.friction`.
- `document-model-gaps.md § Gap C — no disposition lifecycle (#93)` — the residue, enumerated against
  source: (1) `dismissed` still declared, still no setter (`0002_patrol_findings.go:23`); (2)
  `query summary`/`uncollapsed` ignore disposition, with the behavior stated in a code comment
  (`query.go:382,419`, comment `:398-399`); (3) `adoption_log.false_positive` is still a detached log,
  not a finding transition (`0005_adoption_log.go:16`); (4) **no `actor`/`reason`/`disposed_at`
  columns** — `DisposeFinding` records only the new disposition value (`dispose.go:49`).
- `document-model-gaps.md § Smaller hygiene findings` — `adoption_log.detail` rides the implicit
  5000-char cap as a bare `TextField` with no `Max` (`0005_adoption_log.go:17`); the 0013 `feedback`
  collection is the repo's precedent for setting explicit `Max` on content-bearing fields.
- `document-model-gaps.md § Analysis §3 agenda → current status` — agenda item 4 "left to rule":
  retire dead `state.dismissed`; make `query summary`/`uncollapsed` disposition-aware; reconcile
  `adoption_log.false_positive`; add `actor`/`reason`/`disposed_at`.

## 4. Options

Two intertwined sub-decisions: **(A)** the disposition sub-machine's shape and **(B)** the adoption
log's fate. The coherence fixes in (A) — retire the dead `dismissed`, make every count agree on
"open" — are near-forced (any ruling must close the lies; see §5); the genuine spread is over
**provenance depth** and how it composes with **wire / shrink / kill** for `adoption_log`. The four
packages below span that space. All are hypotheses, not rulings.

**Option 1 — Close the lies, build nothing new.**
Retire `state.dismissed`; make `query summary`/`uncollapsed`/MCP count on `disposition='open'` so
"open findings" means one thing everywhere; shrink `adoption_log.event` to the one written value
(`fix`) or narrow it to a deliberately-small set. No new columns, no new writers.
- *Consequences:* cheapest migration (drop dead enum values, one code change in the `openFindingRows`
  callers). Leaves "who/why disposed" unrecorded and the activity-ledger ambition abandoned. A
  `wont_fix` stays an anonymous verdict; `adoption_log` becomes an honest but thin fix-log.

**Option 2 — Provenance on the finding.**
Option 1's coherence fixes, **plus** `actor`/`reason`/`disposed_at` columns on `patrol_findings`
(mirroring the PM `transitions.actor`/`actor_kind` pattern), so a disposition is self-describing.
Then `adoption_log.false_positive` is *provably redundant* — a `wont_fix` disposition with a reason
already records what a false-positive event would (hypothesis) — so `adoption_log` shrinks to what
survives (`fix`, possibly `revert` wired from `restore`).
- *Consequences:* medium migration (add + backfill columns like 0014 did; change the "frozen"
  `DisposeFinding` signature → CLI flags + tool spec). The disposition record becomes the single
  home of finding-level provenance. Still builds no general activity ledger.

**Option 3 — `adoption_log` as the activity ledger.**
Wire all six event values from their natural call sites: patrol run → `patrol`, `restore` →
`revert`, a `wont_fix`/false-positive dispose → `false_positive`, a `record_feedback` "convention
didn't fit" → `friction`, a note → `note`. Provenance lives in adoption events rather than finding
columns; the disposition axis gets only the coherence fixes.
- *Consequences:* the largest surface — new writers threaded through `patrol.go`, `restore.go`,
  `dispose.go`, and a `record_feedback` bridge; `adoption_log.detail` needs an explicit `Max` (0013
  convention). Two provenance homes (adoption events *and* the disposition axis) risk re-creating the
  exact "detached log vs finding transition" split the #93 analysis flagged — the ledger and the
  finding can disagree. Overlaps the live `feedback` collection (which already records friction),
  so `friction` must reconcile with `feedback` or duplicate it.

**Option 4 — Kill `adoption_log`.**
Drop the collection entirely; fold its one live use (batch fix-logging) into an existing ledger
(`patrol_log`, or the `revisions` trail), and give the disposition axis full provenance columns (as
Option 2) so `false_positive`/`wont_fix` live on the finding, not a log.
- *Consequences:* removes a whole dead surface and the confusion of a six-value enum with one writer.
  It is the one destructive-shaped migration here: dropping a shipped, **desk-carrying** collection
  and the read path that surfaces it (a `query adoption` view exists — source-observed, see §9).
  Loses any future single "adoption feed." Heaviest migration semantics for the smallest ongoing code.

## 5. Decision criteria

Any ruling must satisfy (constraint walls cited from `../README.md` §7):

1. **One meaning of "open findings"** across `query findings`, `query summary`, `query uncollapsed`,
   the MCP query tool, and the TUI. This is the C4 tough spot ("Disposition coherence across
   surfaces", `../README.md` §5) — the count divergence lives in `query.go`, which backs both CLI and
   MCP, so a code fix there covers both; the TUI's own count source must be confirmed (§9).
2. **No dead enum values left declared.** Either wire or remove `state.dismissed` and each unwired
   `adoption_log.event` value — a declared-but-unreachable value is the defect this whole decision
   exists to close.
3. **Forward migrations only** on shipped collections; never edit an applied one (§7). A new column
   backfills on live stores exactly as 0014 backfilled `disposition` to `open`; removing an enum
   value is clean only because no row holds `dismissed`/the five events (§9 flags the value-removal
   mechanics as unverified).
4. **Identity-neutrality** (§7) — any `actor`/`reason` field stays free-text with no baked identity,
   following the existing free-text `claimed_by`/`transitions.actor`/`notes.actor` pattern; it must
   not introduce a user/identity relation or a default actor.
5. **Provenance durability decision.** The store rebuilds from disk and never persists document
   bodies (§7); findings and dispositions are store-only supervisor state, **not** derivable from the
   tree — so any provenance added is lost on a store rebuild. The ruling must state whether who/why-
   disposed needs to survive rebuild (and if so, whether it belongs on disk, not only in the store).
6. **Auto-resolution ∘ human-disposition truth table.** Patrol auto-resolves a non-re-firing finding
   to `resolved` independent of disposition, while inheritance carries a supervisor's call across
   resolve→re-fire. The ruling must state the full composition: e.g. whether a `wont_fix` suppresses
   re-fire entirely, whether auto-resolution clears or preserves the disposition, and what a disposed
   `resolved` finding contributes to counts — so a supervisor's `wont_fix` never silently resurfaces
   as an open finding.

## 6. Blast radius

Per option class (live stores exist — the owner's own desks — so every model change is a forward
migration):

- **Collections / migrations (librarian lane only):**
  - *Retire `dismissed`* (all options): forward migration narrowing `patrol_findings.state`'s
    SelectField `Values`. Safe iff no row holds it (dossiers confirm no setter; §9 flags the removal
    mechanics).
  - *Disposition-aware counts* (all options): **no migration** — a code change to the
    `openFindingRows` callers in `query.go` (`:382,:419`). Reaches CLI + MCP (same code) automatically;
    the TUI's count rendering must be updated separately if it doesn't route through the same path.
  - *`actor`/`reason`/`disposed_at`* (Options 2, 4): forward migration adding 2–3 fields to
    `patrol_findings`, backfilled empty (0014 precedent); content-bearing `reason` needs an explicit
    `Max` (0013 convention). Changes the "frozen" `DisposeFinding` signature → new CLI flags + MCP
    input type + tool spec.
  - *Wire `adoption_log`* (Option 3): new writers in `patrol.go`/`restore.go`/`dispose.go` + a
    `record_feedback` bridge; explicit `Max` on `adoption_log.detail`; consider adding a
    RelationField so an event can point at its finding (today it has none).
  - *Shrink `adoption_log`* (Options 1, 2): forward migration narrowing the `event` SelectField.
  - *Kill `adoption_log`* (Option 4): forward migration dropping the collection; remove
    `recordAdoptionLog`, the `query adoption` read path, and the collection from
    `deskguard`'s desk-carrying set (all source-observed, §9). The one destructive-shaped migration.
- **Spec sections:** `docs/development/specs/pocket-librarian-v1-spec.md` — §5.2 (patrol), §5.3–§5.5 (write boundary /
  restore), and the §5.4 `adoption_log` row-format the spec pins (`apply_fix.go:242` cites it). The
  0.8.0 disposition-lifecycle section gains the coherence + provenance + composition deltas.
- **`schema/` contract:** **untouched.** `state`/`disposition`/`adoption_log` are librarian-lane
  store collections, not part of `schema/doctypes.yaml` or `profile.schema.yaml` — this decision does
  not move the shared contract or the PM lane, which keeps its blast radius entirely librarian-side.
- **Skills / personas:** `dispose` is CLI-only, not an agent tool, so the agent persona is largely
  unaffected; if provenance columns land, any agent guidance that references "open findings"
  semantics (the librarian system prompt) should be checked. Whether `dispose` should *become* an
  agent/MCP tool is a D5 surface question, not decided here.
- **Docs:** an ADR recording the ruling; `docs/development/specs/tool-surface.md` if the `dispose` signature changes or
  a query kind is added/removed; `CHANGELOG.md`.

## 7. Out of scope / interactions

- **PM-lane audit model.** `transitions` is the PM lane's append-only, actor-bearing audit
  (`data-model.md § 2.3`); D4 does not touch it, but it is the natural analog for finding-level
  provenance (`actor`/`actor_kind`). Borrow the pattern; don't re-decide PM's audit here.
- **The `feedback` collection's own lifecycle.** `feedback.status='resolved'` is itself an unwired-
  setter candidate (`data-model.md § 5.1`, lower confidence); D4 does not rule it, but Option 3's
  `friction` event overlaps `feedback`'s purpose, so wiring `friction` must reconcile with (or defer
  to) the `feedback` collection rather than duplicate it.
- **Surface exposure of `dispose` and of disposition-aware counts** — whether `dispose` becomes an
  agent/MCP tool, and how each surface renders counts — is **D5**'s (agent contract & surface parity)
  boundary. D4 sets the *model*: one meaning of "open findings," and which count surfaces must agree.
- **Typed cross-references.** `adoption_log` having no RelationField to its finding is a
  typed-reference-shaped gap, but the reference primitive is **D2**'s remit; D4 may adopt a relation
  as a local fix without ruling the primitive.
- **General TextField `Max` sweep** and the `entity_type` naming collision are **D8** backlog;
  `adoption_log.detail`'s `Max` is in-scope here only if Option 3 wires the collection.
- **Dependencies.** D4 depends on nothing (rulable first); it feeds D5 (which surfaces expose
  the dispose verb and disposition-aware counts) and lightly touches D2 (a relation on
  `adoption_log` would be an instance of the reference primitive). No dependency on a later brief.

## 8. Uncertainties

- **MCP / TUI count behavior.** The evidence base (data-model, workflows, document-model-gaps)
  grounds the count divergence in `query.go`'s CLI query kinds. The seed asserts MCP and the TUI also
  disagree. MCP runs the same `query.go` tool, so it inherits the divergence — but the **TUI's own
  count source** is covered by `surface-matrix.md` (not in this brief's evidence base) and must be
  re-derived before the coherence ruling (criterion 1) is closed.
- **`adoption_log` has a reader.** Sanity-checked in source: `queryAdoption` reads `adoption_log`
  (`query.go:450`), surfaced as a `query adoption` CLI kind (`main.go:778`). The evidence dossiers
  inventory the *writer* (only `fix`), not this reader — the kill option's blast radius (§6) depends
  on it. Session to re-derive from source.
- **Desk-carrying collection.** `adoption_log` appears in `deskguard`'s desk-carrying-collections set
  (`deskguard.go:13`) — source-observed here, not dossier-grounded; it affects the kill/rebuild
  semantics of Option 4. Session to re-derive.
- **Enum-value removal mechanics.** The dossiers confirm no rows hold `dismissed` or the five dead
  `adoption_log` events, but the PocketBase mechanics of *removing* a value from a live `SelectField`
  (whether it's a clean forward migration or needs a data-integrity check) were not traced. Build to
  verify.
- **Provenance durability.** Whether the owner wants who/why-disposed to survive a store rebuild-
  from-disk is a genuine open call (criterion 5), not resolvable from source — it is a requirement
  the session must state, since findings/dispositions are store-only and rebuild loses them.
- **`feedback.status='resolved'`** is flagged in `data-model.md § 5.1` with lower confidence than
  `dismissed`/`adoption_log` (only a test-file setter found); if Option 3 routes `friction` through
  `feedback`, the `feedback` lifecycle must be confirmed first.

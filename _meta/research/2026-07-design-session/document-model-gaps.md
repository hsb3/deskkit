---
type: analysis
status: draft
created: 2026-07-20
updated: 2026-07-20
tags: [design-session, data-model, document-model]
synopsis: Delta-verification of the exec-desk analysis §2 (document-model gaps A–F + hygiene) against post-PR-112 main — separating the symptoms PR 112 closed from the modeling questions still open for the design session.
---

_Grounding dossier — delta-verifies the exec-desk analysis's Concern 2 ("document data model",
gaps A–F + hygiene) against post-PR-112 `main` (`51235f6`). For each gap: what the analysis
claimed | what PR 112 changed | what remains open as a modeling question, with fresh `path:line`
citations. Informs the design session; does not itself rule. **Dossier claims are hypotheses** —
anything a ruling binds on gets re-derived in the session._

Status: draft (2026-07-20)

# Document data model — gaps A–F delta-verified against PR 112

The exec-desk analysis (`EXECUTIVE_DESK/…/desk-standard-agent-symmetry-and-document-model-2026-07-20.md`,
§2) predates PR 112 (the 0.8.0 bug floor, merged as `51235f6`). PR 112 closed the **symptoms**
of #92/#93/#102 (and #91); it left the **modeling questions** under them open by design — the
analysis's own "fix-shape implication" (§2 close) said the deeper model calls belong to the
requirements session, not the bug floor. This dossier confirms exactly that split.

All citations are against `main` @ `51235f6`. Base path for the Go lane is
`librarian/internal/`; specs are under `docs/`. Live proof: `go test ./internal/modules/librarian/...
./internal/modules/pm/engine/...` is green (run below), and the named per-gap tests are cited inline.

---

## Gap A — no sub-file addressing (#102)

**Analysis claimed:** gate pointers must be whole desk-file paths; `://` fails closed and section
anchors "resolve as nonexistent filenames"; spec §3.1 defines `pointer` broadly ("doc path / issue
URL / other locus") but the whole-file-only constraint "is implemented but specified nowhere";
latent until the D8 seeder emits section refs.

**What PR 112 changed:** the verdict path now *tolerates* a `§ heading` section anchor by resolving
only the FILE part, and gives `#`-anchored pointers an actionable failure instead of a bare
not-found.
- `sectionFilePart(pointer)` splits on `§` and trims, returning the file portion —
  `"notes.md § Decisions"` → `"notes.md"`; `#` is deliberately left untouched
  (`modules/librarian/module.go:209-214`). `Verdict` calls it at `module.go:122`; the twin trait
  reader `Frontmatter` calls it at `module.go:188`.
- The heading is advisory and never checked — the file must exist + validate, the heading need
  not (`module.go:116-121`). Proven by `TestVerdict_ToleratesSectionAnchorSuffix`
  (`module_test.go:198-259`; an absent heading still passes, `:236-238`).
- A `file.md#heading` pointer fails closed with a named cause:
  `note: "#heading" anchors are not stripped from pointers — write "file.md § Heading" instead`
  (`module.go:136-138`). Proven by `TestVerdict_HashAnchorNotStripped` (`module_test.go:261-290`:
  must NOT resolve to its file part, and the failure must mention `§`).
- `://` still fails closed as before (`module.go:123-124`).

**What remains open (modeling question):** the pointer **grammar is implemented, not specified.**
The spec's only definition of a pointer is still the broad one-liner — `pm-system-v1-spec.md:396`
(`| pointer | text | doc path / issue URL / other locus (R2.3) |`) and the "MAY carry a pointer to
a GitHub issue URL" prose at `:744`. Nothing in `docs/` states: that `§` is the section delimiter,
that the heading is advisory (gates ignore it so heading renames can't break a transition), that
`#` is rejected, or what a sub-file locus would ever *satisfy* gate-wise. Real sub-file addressing
(the D8 seeder's need) requires stable section identity without reintroducing rename-fragility —
the analysis's latent-failure prediction is deferred, not resolved, because the anchor is
tolerated-and-ignored rather than actually resolved to a section.

> **Verdict A — SYMPTOM CLOSED, MODEL OPEN.** `#102` no longer fails a seeded section-ref item on
> its first gated transition; the pointer grammar remains unspecified and sub-file addressing
> remains unbuilt (heading is advisory-only). Maps to agenda item 2.

---

## Gap B — untyped, repo-unqualified cross-refs (#92)

**Analysis claimed:** `graduated_to` = first `#N`/URL substring in any ≤40-line doc
(`sweep.go:253-259` pre-112), conflating "is a graduation stub" with "mentions an issue"; the bare
`#N` never consults `repos.shorthand.issue_default`, so the same value means different issues on
different desks. "The extraction is the bug; the single untyped text field is the gap under it."

**What PR 112 changed:** extraction is now **explicit-marker-only**. `scanFile` populates
`graduated_to` from `graduationMarker(text)` (`sweep.go:261`), which takes (1) a frontmatter
`graduated_to:` key, else (2) a canonical inline line matched by `inlineGraduationRe`
(`sweep.go:359`, `graduationMarker` at `:368-376`). A bare `#N` merely quoted in prose no longer
populates the column, and this same gate now backs R5 (§5.2), fixing the false-fire at the root.
The old `lines <= 40 AND leftmost ISSUE_REF match` heuristic is gone from the write path (the
`issueRefFind`/`hashRefRe`/`findBareHashRef` helpers survive at `sweep.go:378-427` but are no
longer called by `scanFile` — see Gaps & uncertainties).

**What remains open (modeling question):** the field is **still one untyped `text` column with no
repo qualification** — the analysis's "gap under it" is untouched.
- Storage: `files.graduated_to` is a bare `TextField` (`collections/0001_files.go:23`); the plain-Go
  row carries it as a single `GraduatedTo string` (`sweep.go:117`, set at `:164`). Spec pins it as
  `text` (`pocket-librarian-v1-spec.md:477`, `:680`).
- **The spec-pinned regex accepts a bare number.** `inlineGraduationRe` =
  `(?im)^\s*graduated to:?\s+(wb#\d+|#?\d+|https?://\S+)` — the `#?\d+` alternative makes
  `#` optional, so `graduated to: 42` captures `"42"` as opaque pointer text
  (`sweep.go:359`; the `#?\d+` intent is spelled out in the comment `:357-358`). This is pinned
  **verbatim** by the spec at `pocket-librarian-v1-spec.md:911` (graduated_to precedence) and again
  at `:999` (R5). Proven by `TestGraduationMarker` case
  `"inline canonical line, bare number is opaque pointer text (spec-verbatim)", "graduated to: 42\n", "42"`
  (`sweep_test.go:276`).
- **No repo qualification.** Nothing on the write path consults `repos.shorthand.issue_default`
  (or any repo qualifier); the captured string is stored as-is, so `42` / `#42` / `wb#42` mean
  different issues on different desks — reference identity is still desk-ambiguous. (Grep: no
  `issue_default` / `repos.shorthand` reference exists in `sweep.go` or the config loader's
  graduated-to path.)

> **Verdict B — EXTRACTION FIXED, TYPE/QUALIFIER GAP OPEN.** `#92`'s false-positive is closed at
> the root (marker-only, R5 shares the gate); `graduated_to` remains a single untyped column whose
> spec-pinned grammar accepts a bare, repo-unqualified number. Maps to agenda item 3.

---

## Gap C — no disposition lifecycle (#93)

**Analysis claimed:** `patrol_findings.state` declares `dismissed` (migration 0002) but no code path
ever sets it; no actor/reason/disposed_at columns; `adoption_log`'s `false_positive` is a detached
log, not a finding transition. "Half-specified vocabulary, unwired."

**What PR 112 changed:** a disposition lifecycle **shipped**, orthogonal to `state`.
- Migration **0014** adds a `disposition` `SelectField` = `{open, acknowledged, triaged, wont_fix}`,
  default-backfilled to `open`, explicitly orthogonal to `state`
  (`collections/0014_patrol_findings_disposition.go:32-66`; rationale `:8-15`).
- `DisposeFinding` sets **only** `disposition` and leaves `state` untouched, so a disposed finding
  stays `flagged` and survives re-patrol (`tools/dispose.go:38-59`; enum guard `:23-28`).
- **(file, rule, checksum) inheritance:** on a resolve→re-fire cycle patrol files a fresh row and
  inherits the prior non-open disposition via `inheritedDisposition(txApp, row.ID, rule, row.Checksum)`
  (`tools/patrol.go:104`; helper at `patrol.go:262-273`). Proven by
  `TestPatrol_DisposedFindingPersistsAcrossRepatrol`,
  `TestPatrol_ResolvedDisposedFindingRefiredInheritsDisposition`,
  `TestPatrol_ChangedChecksumReopensDisposedFinding` (all PASS).
- CLI surface: `deskkit findings n <finding-id> --as <open|acknowledged|triaged|wont-fix>` bound to
  `tools.DisposeFinding`, plus `query findings --include-nd` to show disposed rows
  (`cmd/deskkit/main.go`). Tool tests: `TestDisposeFinding_*` (`tools/dispose_test.go:39-140`, all
  PASS).

**What remains open (residue — verified against current source):**
1. **`state.dismissed` still declared, still no setter.** Migration 0002 keeps
   `state = {flagged, dismissed, fixed}` (`collections/0002_patrol_findings.go:23`); 0010 adds
   `resolved` (`0010_patrol_findings_resolved.go:20-31`). Grep of every `Set("state", …)` in the
   librarian module shows only `flagged` / `fixed` / `resolved` on `patrol_findings`
   (`patrol.go:93,168`; `apply_fix.go:223`; `restore.go:106`; `0010`'s Down at `:53`) — the
   trigger-lane setters (`trigger.go:125-243`: claimed/failed/done/deferred/queued) are on `tasks`,
   not findings. **`dismissed` remains a dead enum value** — its role was taken by the orthogonal
   `disposition` axis, but the value was never removed from `state`.
2. **`query summary` and `query uncollapsed` ignore disposition.** `queryFindings` (default) filters
   `disposition = 'open'` (`query.go:401-406`), but `queryUncollapsed` calls
   `openFindingRows(app, "rule = 'R5'")` (`query.go:382`) and `querySummary` calls
   `openFindingRows(app, "")` (`query.go:419`) — **no disposition filter**, counting every flagged
   row. The code comment states this explicitly: "queryUncollapsed and querySummary keep their prior
   behavior (all flagged rows, no disposition filter)" (`query.go:398-399`). So "open findings" means
   two different things across surfaces — the exact coherence gap the analysis flagged.
3. **`adoption_log.false_positive` is still a detached log.** It remains an `event` enum value on
   `adoption_log` (`collections/0005_adoption_log.go:16`), not a finding transition — disposing a
   finding writes `patrol_findings.disposition`, never an adoption_log row.
4. No `actor` / `reason` / `disposed_at` columns were added — `DisposeFinding` records only the new
   `disposition` value (`dispose.go:49`), so *who* disposed a finding and *why* is unrecorded.

> **Verdict C — LIFECYCLE SHIPPED, RESIDUE OPEN.** `#93` has a real disposition sub-machine with
> re-patrol inheritance; the `dismissed` dead value, the disposition-blind summary/uncollapsed
> counts, the detached `false_positive`, and the missing actor/reason fields are all unresolved.
> Maps to agenda item 4.

---

## Gap D — non-atomic PM mutations (#91)

**Analysis claimed:** `Transition` = three separate saves (item → audit → cascade); "zero
`RunInTransaction` in the PM module (verified)."

**What PR 112 changed:** the "zero RunInTransaction" claim is now **STALE**. The PM engine wraps its
mutations in `RunInTransaction`. Nine mutating methods each open a transaction:
`CreateItem` (`engine.go:134`), `Transition` (`:291`), `Block` (`:450`), `Unblock` (`:487`),
`Claim` (`:539`), `Release` (`:578`), `Link` (`:654`), `SetStatusLabel` (`:818`), `AddNote`
(`:894`). Every inner read AND write runs through a tx-scoped engine copy (`withApp`, `engine.go:238-244`),
so the version-guard read and the write it authorizes share one transaction — closing the §3.6
check-then-act TOCTOU. For `Transition`, `transitionCore` runs the whole §4.1 sequence
(load → version-check → mutate → save → audit → cascade) inside the tx (`engine.go:316-393`; cascade
at the success tail), so phase write + audit row + cascade side-writes commit or roll back as one
unit. Proven by `TestTransitionRollsBackOnLaterFailure` (PASS) via the `txFailpoint` seam
(`engine.go:229-236`, wired only in `transitionCore` per `:225-228`).

**pendingAudit design (deliberate, non-atomic on the refusal path only):** a gate refusal mutates
nothing, so its transaction rolls back — but the `gate_refused` transitions row must still persist
(observable, not silent). So the refusal audit is *captured inside* the tx as a `pendingAudit`
struct (`engine.go:246-254`) and *written after* the tx settles (`Transition` at `:300-304`;
`SetStatusLabel` mirrors it at `:817,852`). This is intentional: it reproduces the pre-transaction
single non-atomic audit write — the refusal stands even if the audit write later fails
(`engine.go:247-250`).

**What remains open (modeling question):** essentially nothing modeling-wise — the analysis itself
scoped this as "transactional, not modeling — but same neighborhood." The only residue is
documentary: the other eight tx methods share the failpoint mechanics but have no forced-failure
test (`engine.go:226-228` invites adding `runFailpoint()` there "for parity if one is ever wanted").

> **Verdict D — STALE (CLOSED).** `#91` is fixed: PM mutations are atomic via `RunInTransaction`
> across nine methods, with a deliberately non-atomic refusal-audit escape hatch. Not a live design
> item; carry only as "the analysis's zero-RunInTransaction claim no longer holds."

---

## Gap E — no identity across renames (unfiled)

**Analysis claimed:** `files` keyed by unique `path`; rename = soft-delete + fresh insert, history
discarded; unspecified by design (store is rebuildable); needs a ruling only if doc history becomes
a requirement.

**Delta:** unchanged by PR 112. `files.path` is `Required` (`collections/0001_files.go:14`) with a
UNIQUE index `idx_files_path` (`0001_files.go:30`). The sweep upserts by `path`: an unseen path is
soft-deleted (`sweep.go:86-93`, `deleted=true`), a new path is a fresh `NewRecord` insert
(`sweep.go:66-73`) — so a rename produces one soft-deleted old row and one new row, with no identity
carried across. No frontmatter-id / checksum-rename-inference / content-identity mechanism exists.

> **Verdict E — UNADDRESSED (as expected).** Files remain path-keyed; rename discards history.
> A ruling is needed only if doc history becomes a requirement, and any identity primitive must
> survive store-rebuild-from-disk. Backlog (agenda item 6).

---

## Gap F — `items.type` unconstrained (unfiled)

**Analysis claimed:** plain TextField (`pm/collections/collections.go:56`), unvalidated at
`CreateItem`; a typo'd type matches no gate rules and advances ungated.

**Delta:** unchanged by PR 112. `items.type` is a bare `TextField` (`pm/collections/collections.go:56`).
`CreateItem` validates only that `Title` is non-empty (`engine.go:130-132`) then does
`rec.Set("type", in.Type)` with **no vocabulary check** (`engine.go:146`) — the string is stored
as-is. Type is enforced on the *rule* side (a gate config binds `(type, edge)` requirements,
`engine.go:352`), so a type that matches no rule has no `Effective` requirements and the gate
evaluates to satisfied — i.e. a typo'd type advances ungated. The `schema/doctypes.yaml` vocabulary
(the natural validation source) is consulted for document frontmatter but never for `items.type`.

> **Verdict F — UNADDRESSED (as expected).** `items.type` is still unvalidated at birth; unknown
> types advance ungated. Maps to agenda item 5 (validate at birth vs rule ungated-advance
> deliberate).

---

## Smaller hygiene findings

**Implicit 5000-char TextField caps.** A bare `TextField` (Max==0) silently caps at 5000 chars in
PocketBase. After 0011/0014, **explicit `Max` is set on exactly two sets of fields:**
- 0011 widens three content-bearing bodies to 50,000,000: `revisions.original_content`,
  `messages.content`, `prompts.content` (`collections/0011_widen_content_fields.go:16,23-27`).
- The `feedback` collection (0013) sets explicit Max on all its text fields: `summary` (2000),
  `detail` (50000), `context` (2000) (`collections/0013_feedback.go:21,22,24`).
- Migration 0014 adds only a `SelectField` (`disposition`) — no Max relevance.

Every field the analysis flagged **still rides the implicit 5000-char cap** (verified as bare
`TextField`s with no Max, unchanged by 0011/0014): `patrol_findings.detail` + `proposed_fix`
(`0002_patrol_findings.go:21-22`), `patrol_log.summary` (`0003_patrol_log.go:20`),
`adoption_log.detail` (`0005_adoption_log.go:17`), and the `agent_runs` summaries
`input_summary`/`output_summary`/`error` (`0006_agent_runs.go:20,21,23`).

**`files.entity_type` column vs `schema` `entity_type` enum collision.** The `files.entity_type`
column stores the frontmatter `type` key — `row.EntityType = fmStr(fm, "type")` (`sweep.go:248`),
column declared at `0001_files.go:16`. But `schema/doctypes.yaml` defines a **distinct**
`entity_type` enum = `[person, company, technology, product, service]` (`doctypes.yaml:29`, a
required field of the `entity` doctype at `:69`). So the same name means two unrelated things — the
stored column holds a doctype (`decision`/`analysis`/…), while the schema's `entity_type` is a
person/company classification. Spec-consistent but a live confusion source (unchanged by 112). The
`dir_kind` bucket exists only store-side and bridges only **4 doctypes** to a directory —
`config.EntityDirMap` maps `decision`/`task`/`analysis`/`journal` (`core/config/config.go:52-59`) —
while `doctypes.yaml` defines ~28 doctypes; everything unmapped falls to `other`/`root`/`infra`
(`sweep.go:287-321`).

---

## Analysis §3 agenda → current status

Mapping the exec-desk analysis's six requirements-session agenda items to post-PR-112 reality
(this restates the same six against source; the design-prep README §3 C1–C8 is the companion view):

| §3 item | Gap(s) | PR 112 did | Left to rule |
|---|---|---|---|
| **1. Agent-surface parity** | (Concern 1, not §2) | Untouched | Out of this dossier's scope — see `agent-symmetry.md`. Not addressed by 112. |
| **2. Pointer grammar** | A / #102 | Symptom closed: `§` resolves by file part, `#` fails with a hint (`module.go:122,136-138,209-214`) | What a pointer MAY be (whole file / sub-file locus / issue URL / external), what each satisfies gate-wise, fail-closed rules, stable section identity. **Grammar is implemented but specified nowhere** (spec only `pm-system-v1-spec.md:396`). |
| **3. Cross-reference typing** | B / #92 | Symptom closed: marker-only extraction shared with R5 (`sweep.go:261,359,368`) | A typed reference primitive (kind + target + repo/desk qualifier). `graduated_to` is still one untyped column; the spec-pinned regex accepts a bare, unqualified `42`. |
| **4. Findings disposition model** | C / #93 | Lifecycle shipped: orthogonal `disposition` axis + re-patrol inheritance + CLI (`0014`, `dispose.go`, `patrol.go:104,262`) | Complete the sub-machine: retire dead `state.dismissed`; make `query summary`/`uncollapsed` disposition-aware (`query.go:382,419`); reconcile `adoption_log.false_positive`; add actor/reason/disposed_at. |
| **5. `items.type` validation** | F | Untouched (`collections.go:56`, `engine.go:146`) | Validate against the doctype vocabulary at `CreateItem`, or rule ungated-advance deliberate. |
| **6. Backlog** | E + hygiene | Untouched | Rename identity / doc history (Gap E, must survive rebuild); TextField Max sweep (5 fields still implicit-capped); `entity_type` column rename to clear the enum collision. |

**Net:** of the four Tier-1 issues the analysis tied to §2, **#91 is fully closed** (Gap D stale),
**#92/#93/#102 had their symptoms closed but their modeling questions deferred** (Gaps A/B/C — the
exact "fix the symptom now, rule the model later" split the analysis's §2 close recommended), and
the two unfiled gaps (E, F) plus the hygiene items are entirely untouched.

## Gaps & uncertainties

- **Most consequential open modeling question:** cross-reference typing (Gap B / agenda 3). It is
  the one place a *shipped, spec-pinned* value (`graduated to: 42` → `"42"`) is knowingly wrong at
  the model layer — repo-unqualified and untyped — so it means different issues on different desks
  while looking resolved. It also generalizes: gate pointers (Gap A) and `graduated_to` are the same
  latent "typed, qualified reference" primitive, so ruling B well subsumes part of A.
- **Dead-code note (not traced to a conclusion):** the pre-112 heuristic helpers
  `issueRefFind`/`ghURLRe`/`hashRefRe`/`findBareHashRef` remain in `sweep.go:378-427`. Production
  `scanFile` now calls `graduationMarker` (`sweep.go:261`), so these appear unused by the write path
  but are still exercised by `sweep_test.go:107`. I did not confirm whether any other production
  call site remains — flagged as possible dead code, not verified as such.
- **Doctype count** is stated as ~28 (`rg` count over `doctypes.yaml`), refining the analysis's
  "~40"; I did not hand-count every entry, so treat "~28" as approximate. The load-bearing fact —
  `EntityDirMap` bridges only 4 — is exact (`config.go:52-59`).
- **Migration 0012/0013** were read only for the fields relevant here (dir_kind `infra`; feedback
  Max). I did not audit their full bodies for other hygiene deltas.
- **Live stores** (the owner's own desks) are unverified from here; every claim is against the
  migration/source definitions on `main`, not against a running store's actual column state.
- Scope discipline: this dossier verifies §2 only. Concern 1 (agent symmetry) is delegated to
  `agent-symmetry.md`; the §3-item-1 row above is a pointer, not a verification.

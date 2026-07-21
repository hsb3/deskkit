---
type: analysis
status: active
created: 2026-07-21
updated: 2026-07-21
tags: [model-simulations, deficiency-report, schema-v2, gate]
synopsis: The #126 model-simulations deficiency roll-up (v1 half) — every DEFICIENCY with severity, source scenario/step, and a disposition per the pass bar; FRICTIONs listed separately; v2 section reserved for #125.
---

*The output artifact of #126 (deliverable C). Rolls up every deficiency the v1-half walkthroughs
surfaced, each with a disposition satisfying the pass bar. FRICTIONs (harvest-loop input, not
gate) are listed separately. The v2 section is reserved — blocked by `element-model-revision`
(#125).*

Status: active (2026-07-21) — v1 half complete.

## Pass bar (deliverable D, mirroring the schema-v2 epic close-when)

> Every deficiency listed here is either **filed as its own issue** (linked) or **explicitly
> recorded as accepted-risk with a stated rationale**, before `element-model-revision` (#125) is
> marked finalized. Each v2 deficiency (added later) must additionally be closed by an amendment
> to #125 or recorded accepted-risk before #125 finalizes.

**v1-half assessment: PASS.** Both v1 deficiencies below have a disposition (both file-as-issue,
with drafted titles/bodies ready to file). No deficiency is left undispositioned.

## Deficiencies (gate-bearing)

| ID | Title | Severity | Surfaced in | Disposition |
|---|---|---|---|---|
| D1 | Finding-disposition surface has no id-bearing read path | Medium | Scenario 1, steps 4–5 | **File-as-issue** (draft below) |
| D2 | `update_item` skips the `type` vocab check `create_item` enforces; untyped items bypass gates | Medium | Scenario 2, Part A / D2 probes | **File-as-issue** (draft below) |

### D1 — Finding-disposition surface has no id-bearing read path

**What.** `deskkit findings dispose <finding-id> --as <...>` (0.8.0, CLI-only supervised,
workflows.md §1.5) resolves its target by record id (`FindRecordById`, `dispose.go`). But the
finding read surfaces `query findings` and `query uncollapsed` emit only
`findingBrief{path, detail}` (`internal/modules/librarian/tools/query.go:99-102`) — **no id**.
The `feedback` query, by contrast, emits an id (`feedbackBrief`, `query.go:167-168`). So the id a
supervisor needs is obtainable only from the PocketBase admin GUI or raw REST — the CLI
disposition workflow cannot be completed from the CLI.

**Evidence.** `probes/output/out-s1.txt` step 4: `query findings` returns
`{"kind":"findings","count":2,"by_rule":{"R1":[{"path":"tasks/needs-fm.md","detail":"..."}],...}}`
with no id. Engine is sound (7 tests, `dispose_test.go`); this is a missing read surface only.

**Severity.** Medium — a shipped feature is not operable from its own intended (CLI) surface.

**Drafted issue (ready to file):**

- **Title:** `fix(librarian): expose finding record id in query findings/uncollapsed so 'findings dispose' is usable from the CLI`
- **Body:**
  > `deskkit findings dispose <finding-id>` (the supervised disposition lifecycle, §1.5) takes a
  > patrol-finding **record id**, but no read surface emits one: `query findings` and
  > `query uncollapsed` reduce each finding to `{path, detail}` (`query.go:99-102`), unlike the
  > `feedback` query which includes `id` (`query.go:167-168`). Today the only way to get the id is
  > the admin GUI / REST, so the CLI-only disposition workflow can't be completed from the CLI.
  >
  > **Fix:** add the finding `id` to `findingBrief` (and thus to `query findings` /
  > `query uncollapsed` / the `--include-disposed` view), matching `feedbackBrief`. Keep `path` +
  > `detail`. No migration; JSON-additive. Update the tool-surface doc + `verify.sh` to assert the
  > id is present and round-trips through `findings dispose`.
  >
  > Surfaced by the #126 model simulations, Scenario 1 (`_meta/research/model-simulations/
  > scenario-1-librarian-chain.md`). Identity-neutral.

### D2 — `type` validation is asymmetric; untyped items bypass document gates

**What.** Document gates key on `items.type` (`engine.go:367`). Two verified supervised
operations leave an item with a type the gate cannot bind, silently disabling it:

1. **`update_item` does not validate `type`** although `create_item` does. Create refuses an
   unknown type (`engine.go:142` `if !vocab.KnownType(in.Type)`); update sets it blindly
   (`queries.go:530-531`). Verified: `pm create --type tsak` → `Error: unknown item type "tsak"`;
   `pm update <id> --type tsak` → succeeds, type becomes `tsak` (`probes/output/out-d3d.txt`).
2. **An item created with no `type`** advances through gates ungated (empty type binds nothing).
   Verified: a no-type item advanced `work→review` `rc=0 {"phase":"review"}` with no gate doc on
   disk (`probes/output/out-d3c.txt`).

**Severity.** Medium — the gate system's core safety rests on `type`; an ordinary update or an
omitted type removes the gate without any signal.

**Drafted issue (ready to file):**

- **Title:** `fix(pm): validate items.type in update_item (parity with create_item) and decide the empty-type gate policy`
- **Body:**
  > Document gates bind on `items.type` (`engine.go:367`). `create_item` validates `type` against
  > the schema vocabulary and refuses an unknown one (`engine.go:134-144`), but `update_item` sets
  > it with no check (`queries.go:530-531`), and an item created with no type at all advances
  > through every document gate ungated. Both let a supervised operation silently disable a gate.
  >
  > Repro (verified in #126 sims): `pm create --type tsak` is rejected; `pm update <id> --type
  > tsak` is accepted (type becomes `tsak`); a no-type item advances `work→review` with no gate
  > document present.
  >
  > **Fix:** (a) apply the same `vocab.KnownType` check in `update_item` that `create_item` uses;
  > (b) decide + enforce the empty-`type` policy — either forbid transitioning an untyped item
  > through a gated edge, or require `type` at create. Add engine tests for both. Reconcile the
  > stale claim in `_meta/research/2026-07-design-session/data-model.md` §4 ("type is
  > unvalidated") in the same change. Surfaced by #126 Scenario 2. Identity-neutral.

## Frictions (harvest-loop input — NOT gate-bearing)

Listed separately per the brief: frictions feed the improvement/harvest loop, they do not gate
v2 finalization.

| ID | Friction | Severity | Surfaced in | Disposition |
|---|---|---|---|---|
| F1 | Cascade-initiated block/unblock is attributed to the triggering human actor | Low | Scenario 2, step B6 | Harvest-loop item; optional accepted-risk |

**F1 detail.** An engine-cascade auto-unblock records `actor:"operator", actor_kind:"human"` — the
identity of whoever advanced the blocker — with only the prose `detail` distinguishing it from a
human action (`probes/output/out-frictions.txt`). Filtering the audit by `actor_kind` misclassifies
cascade rows. Low-severity audit-fidelity nuance; the cascade works correctly. Suggested harvest
action: attribute cascade-initiated `transitions` rows to a system/agent actor kind (or add a
`cascade: true` flag) so automated writes are distinguishable on a queryable axis. If not taken,
**accepted-risk**: the `detail` string already records the cause, so provenance is not lost, only
harder to query.

## Accuracy observations (recorded with evidence; not gate-bearing)

Three are **stale claims in `../2026-07-design-session/data-model.md`** (commit 51235f6) that the
live tree contradicts. Correcting that dossier is an **out-of-scope follow-up** (this work owns
only `_meta/research/model-simulations/`); recorded here so the corrections are not lost.

| ID | Observation | Evidence |
|---|---|---|
| O1 | data-model.md §4 says `items.type` is unvalidated ("a typo'd type advances ungated"). **Stale** — create validates (`engine.go:142`). Real residual is D2 (update + empty type). | `out-d3d.txt`, `out-d3c.txt` |
| O2 | data-model.md §5.3 (C4 residue: `query summary`/`uncollapsed` count disposed findings). **Stale/resolved** — the 0.8.0 bug floor made all three surfaces filter `disposition='open'` (`query.go:483,505,522`). | source read (`query.go:495-527`) |
| O3 | data-model.md Gaps flags `transitions.event = gate_refused` as maybe-dead. **Live** — every gate refusal writes a `gate_refused` row. | `out-s2.txt` step A7 |
| O4 | Un-frontmattered files escape patrol's typed rules but are caught by `query orphans` (working-as-intended). Baseline procedure should run `query orphans` alongside `patrol`. | `out-frictions.txt` F2 section |

## v2 model deficiencies (RESERVED — blocked by #125)

_No v2 walkthroughs were run: the v2 side of #126 is blocked by `element-model-revision` (#125),
which must first absorb the ADR 0018 Q1–Q4 rulings and both adversarial reviews' fixes. When #125
lands a testable model, run the same four scenarios against it, add a "v2 (deferred)" result to
each walkthrough table, and record each v2 deficiency below with the same disposition discipline —
each closed by an amendment to #125 or recorded accepted-risk before #125 finalizes._

| ID | Title | Severity | Surfaced in | Disposition |
|---|---|---|---|---|
| _(none yet — pending #125)_ | | | | |

## Board follow-ups (out of scope for this deliverable, listed for the owner)

- File D1 and D2 as issues (drafts above), then link them here.
- Record the D2 blocked-by/depends-on relationship between #125 and #126 on the board (issue
  #126 acceptance-criteria bullet 5: "#125 carries a recorded blocked-by/depends-on to this
  issue").
- Correct data-model.md §4 (O1), §5.3 (O2), and Gaps/`gate_refused` (O3) — same change that
  files D2 is the natural home for the O1 correction.

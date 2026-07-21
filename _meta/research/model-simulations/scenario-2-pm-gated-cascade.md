---
type: analysis
status: active
created: 2026-07-21
updated: 2026-07-21
tags: [model-simulations, pm]
synopsis: Scenario 2 walkthrough — a PM item through queue→work→review→terminal with two real gate refusals and a cascaded block/unblock, traced against the v1 model with scripted probes.
---

*Scenario 2 of the #126 model simulations (v1 half). Drives the PM gated-transition and
block/unblock cascade against a throwaway scratch desk with PM enabled (PM is default-on).
Derived from `../2026-07-design-session/workflows.md` §2.2–2.3.*

Status: active (2026-07-21)

## Operator story

A supervisor creates a `task` item, advances it queue→work, is refused at work→review twice (once
because the gate document is missing, once because it is at the wrong status), then succeeds once
the document is `active`, and completes it to terminal. Separately, a blocker item and a gated
item demonstrate the auto-cascade: linking `blocks` initial-blocks the target, and the blocker
reaching its unblock phase auto-clears it.

## Scripted probe

`probes/probe-s2-pm.sh` (output: `probes/output/out-s2.txt`), plus `probes/verify-frictions.sh`
(F1 detail), `probes/verify-d3*.sh` (D2 detail). The shipped default gate ruleset
(`internal/modules/pm/gates/defaults.go`) binds exactly two gates: `task work->review` needs a
`type: task, status: active` doc at the item's pointer; `decision review->terminal` needs a
`type: decision, status: accepted` doc. The gate verdict reads the desk **filesystem**, not the
store (`module.go` `Verdict`).

## Step-by-step trace — Part A: gated transition + refusals

| # | Operator action | Surface behavior (observed) | Store entities / fields | v1 verdict | v2 (deferred) |
|---|---|---|---|---|---|
| A0 | `pm list` (confirm PM default-on) | `[]` (empty JSON array) — PM tools present with no env flag | reads `items` | **OK** [scripted] | — (blocked by #125) |
| A1 | `pm create --type task --pointer tasks/t1.md` | `{"item":{"phase":"queue","version":1,...}}` | `items` (phase=queue, version=1); `type` validated against schema vocab | **OK** [scripted] | — |
| A2 | `pm transition T --to work` | `{"phase":"work","version":2}` — ungated forward edge | `items.phase/version`, `transitions` (event=advance) | **OK** [scripted] | — |
| A3 | `pm transition T --to review` (no doc on disk) | `rc=1 Error: required document (type=task) at "tasks/t1.md" does not exist` — **gate refuses; item unchanged** | `transitions` (event=gate_refused) written post-tx; item rolls back | **OK** [scripted] | — |
| A4 | create `tasks/t1.md` at `status: draft`; retry | `rc=1 Error: required document (type=task, status=active) ... is at status "draft", needs "active"` | second `gate_refused` audit row | **OK** [scripted] | — |
| A5 | rewrite doc to `status: active`; retry | `{"phase":"review","version":3}` — gate satisfied, advances | `items.phase/version`, `transitions` (event=advance) | **OK** [scripted] | — |
| A6 | `pm transition T --to terminal` | `{"phase":"terminal","status_label":"done"}` — ungated for a task | `items.phase/status_label`, `transitions` | **OK** [scripted] | — |
| A7 | `pm get T` audit trail | newest-first: `advance(review→terminal), advance(work→review), gate_refused, gate_refused, advance(queue→work)` | `transitions` append-only; `event=gate_refused` **is a live literal** | **OK** [scripted] | — |

## Step-by-step trace — Part B: block/unblock cascade

| # | Operator action | Surface behavior (observed) | Store entities / fields | v1 verdict | v2 (deferred) |
|---|---|---|---|---|---|
| B1 | `pm create` blocker B, target G | two `queue` items | `items` | **OK** [scripted] | — |
| B2 | `pm link B G --kind blocks --unblock-at work --cascade auto` | edge stored; **G initial-blocks** (B in queue < work) | `dependencies` (kind=blocks); `items(G).blocked=true`, `restore_phase`; `transitions` (event=block) | **OK** [scripted] | — |
| B3 | `pm get G` right after link | `{"blocked":true}` | — | **OK** [scripted] | — |
| B4 | `pm transition B --to work` (reaches unblock_at) | `{"B_phase":"work"}`; cascade fires | `items(B)`; then `tryAutoUnblock(G)` | **OK** [scripted] | — |
| B5 | `pm get G` after B reaches unblock_at | `{"blocked":false}` — **G auto-unblocked** | `items(G).blocked=false`; `transitions` (event=unblock) | **OK** [scripted] | — |
| B6 | `pm get G` audit trail | newest-first: `unblock, block` — both `actor_kind:"human"`, `actor:"operator"` | `transitions` | **FRICTION (F1)** [scripted] | — |

## Findings from this scenario

### FRICTION F1 — cascade-initiated block/unblock is attributed to the triggering human

The auto-unblock in B5 was performed by the engine's cascade, not by a human decision, yet its
`transitions` audit row records `actor:"operator", actor_kind:"human"` — the identity of whoever
advanced the blocker. Only the prose `detail` field distinguishes it
(`"auto-unblocked: \"<id>\" reached its release phase"`). Because `actor_kind` is the
machine-queryable axis and `detail` is free text, filtering the audit by `actor_kind` to separate
human from automated actions will misclassify every cascade row.

Evidence (`out-frictions.txt`, F1 section): the unblock row is
`{"event":"unblock","actor":"operator","actor_kind":"human","detail":"auto-unblocked: ..."}`.

Severity **low** — an audit-fidelity nuance; the cascade itself works correctly. This is not
gate-bearing. Disposition: **feeds the harvest loop** (not a gate item). Listed separately in
`deficiency-report.md`.

### DEFICIENCY D2 — `type` validation is asymmetric, and untyped items bypass gates

Document gates key on `items.type` (`engine.go:367` `dc.rules.Effective(item.GetString("type"),
...)`). Two verified paths let a caller hold a type the gate cannot bind, silently disabling the
gate:

- **`update_item` does not validate `type`** while `create_item` does. `CreateItem` refuses an
  unknown type (`engine.go:142` `if !vocab.KnownType(in.Type)`); but `UpdateItem` sets it blindly
  (`queries.go:530-531` `if in.Type != nil { item.Set("type", *in.Type) }`). Verified: `pm create
  --type tsak` → `Error: unknown item type "tsak"`; `pm update <id> --type tsak` → succeeds, type
  becomes `tsak` (`out-d3d.txt`).
- **An item created with no `type` at all** advances through document gates ungated (empty type
  binds no gate). Verified: a no-type item advanced work→review with `rc=0 {"phase":"review"}`
  though no gate document existed (`out-d3c.txt`).

Severity **medium** — the gate system's safety rests on `type`, and two ordinary supervised
operations (an update, or an omitted type) silently remove the gate. Disposition:
**file-as-issue**.

> **Doc-staleness correction (O1).** `../2026-07-design-session/data-model.md` §4 asserts
> "`items.type` — **No** [validation] — `CreateItem` sets it directly from caller input with zero
> vocabulary check; a typo'd type advances ungated." The live tree contradicts this: **create
> validates** (`engine.go:134-144`). The real residual is the create/update asymmetry + empty
> type above, not create. The dossier predates the type-check landing; correcting it is an
> out-of-scope follow-up (recorded in the deficiency report).

## Notes (OK, verified expectations)

- **`transitions.event = gate_refused` is live** (O3). data-model.md's Gaps section flags it as
  "not confirmed as a literal ... may be dead"; every gate refusal in Part A wrote a
  `gate_refused` row (`out-s2.txt`, step A7). Confirmed wired.
- **A gate refusal is observable but rolls everything else back** — the `gate_refused` row is a
  `pendingAudit` written after the transaction settles; the item's phase/version are unchanged
  (workflows.md §2.2 step 8). Confirmed: version stayed 2 across both refusals, bumped to 3 only
  on the successful advance.
- **The shipped default gate set is minimal** (only `decision review→terminal` and `task
  work→review`) and `defaults.go` self-documents this as a "KNOWN UNAUTHORED DESIGN GAP" pending
  an owner ruling. Not a new finding — already flagged in source; noted for the v2 pass.

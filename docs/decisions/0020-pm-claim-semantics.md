# 0020 · PM claim semantics — a live claim is authoritative over every direct mutation

_Rules whether a live foreign claim gates only phase transitions or every direct mutation of the
claimed item, and records why the authoritative reading is implemented. **This ADR shipped
`Proposed` with the PR review as the ruling mechanism; the coordinating session accepted option
(a) at PR merge (2026-07-21). If the owner prefers the advisory reading, supersede per the
"if the reviewer prefers (b)" consequence below.**_

- **Status:** Accepted — owner-confirmed 2026-07-21
- **Date:** 2026-07-21
- **Raised by:** issue #96 (R-7, deep-dive review 2026-07-19, product-exec-desk); finding I-3 (CONFIRMED)

> **Owner confirmation (2026-07-21):** the authoritative reading (option (a) — a live foreign claim
> refuses **every** direct mutation) is confirmed by the owner via the sign-off batch
> `_meta/signoff/2026-07-21-decision-queue/` (item `adr20` = `confirm-authoritative`). The
> "PR review is the ruling; owner may supersede" window is now **closed**: the option-(b) advisory
> supersession path in the Consequences below is no longer available without a new, superseding ADR.

## Context

The PM engine's `liveForeignClaim` helper (`librarian/internal/modules/pm/engine/engine.go`) reports
whether an item carries a live claim held by someone other than the acting actor (an expired claim is
treated as free). Before this ADR it gated exactly three call sites:

- `Transition` (advance / demote / reopen) — engine.go
- `Claim` (a foreign live claim blocks a re-claim) — engine.go
- `Release` (only the holder may release a live claim) — engine.go

Three other *direct, version-checked* mutating paths did **not** consult it:

- `Block` — engine.go (set the blocked side-state)
- `Unblock` — engine.go (clear the blocked side-state)
- `UpdateItem` — queries.go (edit first-class fields: title, type, court, pointer, severity,
  priority, properties, status_label)

So a non-holder who happens to hold the current version token could `Block` / `Unblock` / `UpdateItem`
a claimed item. The claim was **advisory-over-transitions** in practice, but nothing stated that
contract. The footgun (I-3): an adopter who reads "claim" as a lock assumes `update_item` on a claimed
item is refused; it was not. The mismatch surfaces the day multiple writers overlap (GUI + MCP + a cron
patrol) — the exact concurrency scenario the optimistic-version token (R2.6) is meant to make safe.

The version token and the claim solve **different** problems and do not substitute for each other. The
version token prevents a *lost update* (you overwrite a change you never saw); it does nothing to stop
two actors from serially trampling each other's in-flight work on an item one of them has explicitly
claimed. The claim is the coordination primitive; leaving it unenforced on the mutating paths that most
need coordination (field edits, blocking) is the gap.

### Scope guard (why this stays inside the PM module)

This decision touches **only** `modules/pm/engine`. It does **not** touch the librarian's
record-original-first / `restore` reversibility boundary (a different subsystem) — so it carries none
of that boundary's escalation weight. The change is additive refusal logic reusing an existing helper;
it is byte-for-byte reversible by reverting the three added checks.

## Options

**(a) Authoritative — a claim is a lock over every direct mutation (RECOMMENDED, implemented here).**
Extend `liveForeignClaim` coverage to `Block`, `Unblock`, and `UpdateItem`, so a non-holder is refused
on every direct mutation of a claimed item, exactly as it already is on `Transition`. The claim means
what an adopter intuits: "I hold this; others keep off until my TTL lapses or I release."

- **For:** eliminates the footgun at the source rather than papering over it with docs; one uniform
  rule ("a live foreign claim refuses every direct mutation") is easier to teach than a per-verb
  carve-out; reuses the existing helper and audit path; fully contained and reversible.
- **Against:** a claim now carries more weight, so a stale claim (holder crashed, TTL not yet elapsed)
  blocks more operations — mitigated by the 30-min default TTL (ADR 0019) and `release`.
- **Deliberately unchanged:** the **cascade / auto-unblock** paths (`setBlocked`, `tryAutoUnblock`,
  `reblock`) call the side-state helpers directly, NOT the public `Block`/`Unblock`, so automated
  dependency-driven (un)blocking is unaffected by claims — only *direct human/agent* calls are gated.
  This is correct: a claim coordinates people/agents, not the graph's own derived state.

**(b) Advisory-over-transitions-only — leave coverage as-is, document the scope.**
Keep the three original call sites; state in the spec, the guide, and the `claim` tool description that
a claim gates transitions/claim/release only and is advisory over field edits and blocking.

- **For:** zero behavior change; a claim stays "cheap advice" that never surprises a writer with a
  refusal on a plain field edit.
- **Against:** preserves the very mismatch the finding flags; "claim, but it doesn't stop edits" is a
  contract adopters must *read* to know, and the failure mode (silent trample) is exactly what bites.

## Decision

Adopt **(a) Authoritative**. `Block`, `Unblock`, and `UpdateItem` each check `liveForeignClaim` after
the version check and refuse a non-holder with a message naming the holder and the claim expiry, in the
same shape the transition path already uses. A live foreign claim now refuses **every** direct mutation
of the item; an expired or absent claim, and the holder's own actions, proceed as before. The cascade /
auto-unblock paths are untouched.

## Consequences

- **Contract, stated once:** "A live foreign claim refuses every direct mutation of the item
  (transition, block, unblock, update); it is honored by the holder and lapses at its TTL." This lands
  in `docs/development/specs/pm-system-v1-spec.md` (§3.6) and `docs/usage/pm-guide.md`, and the `claim_item` tool description.
- **Tests:** a non-holder `Block` / `Unblock` / `UpdateItem` on a claimed item is refused, one test
  pinning each path; the holder's own call and the expired-claim case still succeed.
- **Behavior-change fallout:** any existing test or fixture that mutated a claimed item as a non-holder
  and expected success is a legitimate behavior change to update (not a test to preserve).
- **If the reviewer prefers (b):** revert the three engine/queries checks + their tests and keep only
  the documentation, re-labeling the code deliverable as dropped. The docs wording flips from
  "refuses every direct mutation" to "gates transitions/claim/release only; advisory elsewhere."

## Affects

`librarian/internal/modules/pm/engine/engine.go` (`Block`, `Unblock` claim checks) ·
`librarian/internal/modules/pm/engine/queries.go` (`UpdateItem` claim check) ·
engine/queries tests · `docs/development/specs/pm-system-v1-spec.md` §3.6 · `docs/usage/pm-guide.md` ·
the `claim_item` tool description · [ADR 0019](0019-durable-pm-defaults.md) (claim TTL default,
the mitigation for a stale claim under reading (a)).

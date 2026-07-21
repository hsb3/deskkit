---
name: pm-advance-item
description: >-
  Move a PM work item forward one phase and handle the document gate. Use when advancing an
  item (queue → work → review → terminal), when a transition was refused and you need to
  understand and satisfy what the gate demands, or when demoting/reopening an item. Reads the
  item with get_item, requests the change with transition_item, and — on a gate refusal —
  routes the missing document to the normal authoring path before retrying. Use whenever
  someone says "advance this", "move it to review", "mark it done", "why won't it transition",
  or "it says the gate failed".
---

# pm-advance-item — walk an item through the machine and its gates

Advancing an item is two things at once: a **rigid state machine** (only certain phase edges
are legal) and a **document gate** (some edges require a validated document to exist first). The
machine refuses illegal edges before the gate ever runs; the gate refuses a legal edge until its
required document is present. This skill drives both correctly.

## Step 1 — read the item

Call **`get_item <id>`**. You need, from the result:

- the current **`phase`** (`queue` | `work` | `review` | `terminal`) and `status_label`,
- the **`version`** token (optimistic-concurrency guard — you pass it back on the write),
- whether it is **`blocked`** (a blocked item cannot advance until unblocked),
- its **`pointer`** (the document/issue the item tracks), notes, dependency edges.

## Step 2 — request the transition

Call **`transition_item`** with the item id and the **target phase**. The tool runs the machine,
then the gate. CLI equivalent: `deskkit pm transition <id> --to <phase>`.

Legal edges only (everything else is refused by the machine, before gates):

| Kind | Edges |
|---|---|
| advance | `queue→work`, `work→review`, `review→terminal` |
| demote | `review→work`, `work→queue` |
| reopen | `terminal→work` |

Pass the `version` you read in step 1. If another writer moved the item first, the write is
refused as a version conflict — re-read with `get_item` and retry against the new version.

A **live foreign claim** refuses the write too, but it is a distinct case from a version
conflict: it means someone else is actively holding the item (via `claim_item`), and the refusal
holds regardless of version until the claim's TTL lapses or the holder releases it. If you intend
to mutate an item across more than one call, `claim_item` it first so a foreign write cannot land
underneath you; a non-holder is refused on every direct mutation (transition, block, unblock,
update) of a claimed item.

A **blocked** item is refused: clear the block first (see pm-triage / `unblock_item`), then
transition.

## Step 3 — handle a gate refusal

When a legal edge is refused by the **gate**, the refusal names exactly what is missing — the
required document's **type**, the **status** it must hold, and where it is expected (the item's
`pointer`). For example, a `decision` item advancing `review→terminal` requires its decision
document to be **accepted**; a `task` item advancing `work→review` requires its task document to
exist, validate, and be **active**. (These rules are the desk's `desk_config` gate ruleset — a
desk edits them; the shipped default carries only these two examples. Not every edge gates —
`queue→work` is ungated by default, for instance; only the `(type, edge)` pairs the desk's
config names carry a document requirement.)

"Validate" means the document's frontmatter carries every schema-v1 **universal** key —
`type`, `status`, `created`, `updated`, `tags` (`status` optional on lightweight types) — plus
every field its doctype requires. `updated` is a universal key like the rest: a document that is
otherwise correct but missing `updated` still fails the gate, and the refusal names it.

The gate reads a document verdict; **it never writes the document, and neither do you through the
PM system.** To satisfy a refusal:

1. **Author the required document through the normal path** — the human, or the librarian under
   its supervised binding-doc discipline (flag-only, record-original-first). The PM system writes
   only its own store; it never writes desk files.
2. If the document lives somewhere the item does not yet point at, set the item's `pointer` with
   **`update_item`** (version-checked) so the gate can find and validate it. Write a **resolvable
   document pointer** — a desk-relative file path. A trailing `§ heading` section anchor is
   tolerated (the gate resolves the file part and ignores the heading), but prefer clean
   file-only pointers; the heading is advisory and never checked.
3. **Re-run `transition_item`.** With the document present, validating, and at the required
   status, the same edge now succeeds.

Do not try to "force" past a gate — there is no bypass. The document is the gate.

## Step 4 — record the rationale

Attach the why with **`add_note`** (a phase-scoped keyed note, e.g. `key: rationale` or
`key: handoff`). Notes are the lighter artifact; they do not satisfy a document gate but they
carry the reasoning forward for the next session.

## Demote and reopen

`transition_item` is the one tool for all three directions. Demotes (`review→work`, `work→queue`)
and reopen (`terminal→work`) pass ungated by default; a desk MAY gate them via `desk_config`. Use
a demote to walk an item back when review finds it not ready; use reopen to resurrect a terminal
item back into work.

## When you are read-only

If the write tools (`transition_item`, `update_item`, `add_note`, `unblock_item`) are absent
while the read tools are present, the desk has `PM_AUTONOMOUS_WRITES=false`. Diagnose the gate
and state exactly what document is missing and what edge it blocks, but leave the mutation to the
owner via `deskkit pm transition ...`.

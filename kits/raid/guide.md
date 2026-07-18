---
type: sop
status: final
created: 2026-06-30
updated: 2026-06-30
tags: [meta]
---

# SOP — raid

## When to write one

Open a RAID log once a project has real uncertainty — external dependencies, unproven assumptions, or risks worth tracking. One RAID per project (or engagement); it's a **living register**, not a one-shot document. Revisit it at every status check-in.

**Red flag:** an empty Assumptions or Dependencies section on a non-trivial project means you haven't looked hard enough, not that there's nothing there.

## How to write one

A RAID log holds four registers. The discipline is keeping the four **distinct** — most teams blur Risk and Issue, which hides what needs action *now* versus what needs *watching*.

| Register | Tense | The test |
|---|---|---|
| **Risk** | *might* happen | "If X happens, it'll hurt." Not happened yet. |
| **Assumption** | *believed* true | "We're proceeding as if Y is true." Unproven. |
| **Issue** | *already* happened | "X is broken right now." Needs resolution. |
| **Dependency** | *external* need | "We can't proceed until Z, and Z isn't ours." |

The flow between them: an **Assumption** that proves false becomes a **Risk** (if not yet realized) or an **Issue** (if already biting). A **Dependency** that slips becomes an **Issue**.

### Risks
Score every risk **Probability** (high/medium/low) × **Impact** (critical/significant/minor) — that product is what tells you which risks earn active management. Then write a **mitigation**: a concrete action that lowers the probability or the blast radius. A risk with no mitigation is just an anxiety; either find one or accept it explicitly.

### Assumptions
State the belief plainly, rate your **confidence**, and give a **validation plan** — how and when you'll confirm it. Assumptions you can't validate cheaply are themselves risks; promote them.

### Issues
The thing that's broken *now*. Give it a **priority** (P0–P3) and a **resolution** — the plan, or the decision that's needed. Issues drive this week's work.

### Dependencies
What you need that you don't control. Name the **provider**, what it **unblocks**, and the **risk if late** — that last column is what makes a dependency actionable rather than a footnote.

### ID conventions
Stable prefixes, zero-padded, never reused: `R-001`, `A-001`, `I-001`, `D-001`. When an item closes, set its `status` (closed / mitigated / validated / resolved / met) — don't delete the row. The history is the point.

## Status transitions

The note's frontmatter `status` tracks the *register as a whole*:

| From | To | Trigger |
|---|---|---|
| `draft` | `in-review` | First real pass shared for review |
| `in-review` | `approved` | Register accepted as the project's tracking baseline |
| any | `archived` | Project closed; RAID kept as record |

Individual rows carry their own per-item status in their table (open / mitigated / resolved / …).

## Anti-patterns

- **Risk/Issue blur** — listing things that already broke as "risks." If it happened, it's an Issue.
- **Mitigation-free risks** — a risk column with no mitigation is worry, not management.
- **Assumptions with no validation plan** — an untested assumption is a hidden risk; give it a check or promote it.
- **Delete-on-close** — removing closed rows erases the history that makes the log valuable. Update `status` instead.
- **Write-once** — a RAID opened at kickoff and never touched is dead weight. It lives or it's worthless.

## Example

See `example.md` in this folder for a worked RAID log for the Quillpad project.

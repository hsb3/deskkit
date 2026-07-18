---
type: sop
status: final
created: 2026-06-30
updated: 2026-06-30
tags: [meta]
---

# SOP — project-brief

## When to write one

Write a project brief at the **start of a whole project** — the moment an idea is big enough to span more than one deliverable and you need everyone (including future-you) anchored to the same north-star. It's the highest-altitude doc in the stack: it frames *why this project exists and what winning looks like*, and everything downstream cites it.

One brief per project. It's a **living vision doc**, not a one-shot — revisit it whenever scope, audience, or success criteria drift, and let it settle once the direction is locked.

**Know what it is NOT:**

| Doc | Altitude | Answers |
|---|---|---|
| **project-brief** (this) | Vision | *Why* this project, for *whom*, and what "done well" looks like |
| **one-pager** | Pitch | Is this *one idea* worth building? Scoped must-haves + ship date |
| **product-spec** | Requirements | *What* exactly gets built — the PRD, features, acceptance criteria |

If you're listing features or acceptance criteria, you've dropped into product-spec territory — pull back up. The brief sets the destination; the spec plans the route.

**Red flag:** if the whole brief reads like a feature list, you're writing a spec wearing a brief's name. The test: could a stranger read it and explain *why* the project matters, not just what it does?

## How to write one

Stay at vision altitude throughout. Seven sections, in order:

### 1. Vision / North-star
One paragraph painting the future state, plus a single repeatable north-star sentence. The discipline: write the **destination, not the route**. If a reader can't repeat the north-star back after one read, it's too complicated — cut it down to the one thing that orients every later decision.

### 2. Problem & target user
Name a **specific** person or role — "users" fails the test, "a solo author publishing to an invited audience" passes. State what actually breaks for them today, how they cope now, and the evidence the need is real. Don't invent problems; if you can't point to evidence, the brief goes back to research.

### 3. Value proposition
The bet. Why *this* project wins over the alternatives — including doing nothing. State the value as the user would *feel* it, not as a feature you'd ship.

### 4. Scope boundaries
Vision-level **in / out** — the shape of the project, not a feature inventory. The **Out** list is the contract that protects the brief from creeping into a product build; an empty Out list means you haven't bounded the vision. Detailed scope lives in the product-spec.

### 5. Success criteria
**Outcomes, not activity.** "We shipped it" and "100 signups" are not success; "the target user does X on cadence" is. A handful, each one verifiable — if a criterion is vague, you haven't decided how you'd know it worked.

### 6. Constraints
The fixed realities the vision must respect — platform/stack commitments, hard dates, budget, team, legal/policy. List the ones that *shape* the project; push fine detail down to the spec.

### 7. Open questions
The unresolved forks that could change the project's shape. Each names what's unknown and what resolving it unblocks. Prune as answers land — a brief whose open-questions list never shrinks is a brief no one's revisiting.

## Status transitions

The note's frontmatter `status` tracks the brief as a whole (`reference` family):

| From | To | Trigger |
|---|---|---|
| `draft` | `in-review` | First full pass shared for review |
| `in-review` | `approved` | Vision accepted as the project's north-star; spec work can start |
| any | `archived` | Project closed or vision superseded; brief kept as record |

Approval doesn't freeze it — a living vision doc is edited in place as direction evolves; archive only when the project itself ends.

## Anti-patterns

- **Spec-in-disguise** — feature lists and acceptance criteria belong in the product-spec. If the brief reads like a PRD, raise the altitude.
- **Vague target user** — "users" / "developers" / "teams" tells you nothing. Name the specific person or role, or you can't reason about value.
- **Activity as success** — "we shipped it" / "X signups" measure motion, not value. Success criteria are outcomes the target user produces.
- **Empty Out-of-scope** — no boundaries means no project shape; it'll sprawl into a product build. The Out list is the contract.
- **Route, not destination** — describing *how* you'll build instead of *why* and *what winning is*. Save the how for the spec.
- **Write-once north-star** — a brief authored at kickoff and never revisited drifts out of sync with reality. It lives or it's dead weight.

## Example

See `example.md` in this folder for a complete project brief for the Quillpad project.

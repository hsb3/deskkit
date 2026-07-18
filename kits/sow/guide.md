---
type: sop
status: final
created: 2026-06-30
updated: 2026-06-30
tags: [meta]
---

# SOP — sow

A **Statement of Work** is the client-facing contract for an engagement: the document you send a client that defines what you'll do, what they'll receive, how "done" is judged, and on what terms. It's the surface where a conversation becomes a commitment.

Keep it distinct from its two internal cousins:

| Doc | Audience | Answers |
|---|---|---|
| **project-brief** | internal | "What's the vision and why does this project exist?" |
| **one-pager** | internal | "Should we build this, and what's the smallest version?" |
| **sow** *(this)* | **the client** | "Here's exactly what we'll deliver, and the terms." |

The brief and one-pager are where you argue with yourself. The SOW is where you make a promise to someone else — so every line has to be one you'd stand behind on signature.

## When to write one

Write a SOW when an engagement crosses from "we're talking" to "we're committing" — a paid client project, a fixed-scope deliverable, or any work where you and the client need a shared, signable definition of done. One SOW per engagement. Re-scoping a live engagement is a **change request** against the existing SOW (or a new SOW), not a quiet edit to the old one.

**Red flag:** if you can't fill in the **Acceptance criteria** as objective, checkable conditions, you don't yet understand what you've sold. Stop and pin down "done" with the client before the SOW goes out.

## How to write one

The SOW's whole job is to make scope and acceptance unambiguous. Two readers — you and the client — should reach the same answer to "is this done?" and "is this included?" without negotiation.

### Context / background
The client's situation in the client's terms. Enough that a reader who's never met them understands why the work matters. Resist the urge to lead with your solution — lead with their problem.

### Objectives
Outcomes, not activities. "A staging site the marketing team can publish to without a developer," not "build a CMS." Each objective is something the client would agree is worth paying for.

### Scope — in and out
The **Out of scope** list is the most valuable part of the document. It's where you head off the assumption that sinks fixed-fee work ("I thought migrations were included"). It must be non-empty; if it feels short, you haven't thought about what a reasonable client would *assume* comes with the work. Name those things and exclude them explicitly.

### Deliverables
The concrete artifacts the client receives — each in the table with a format and an acceptance hook. A deliverable the client can't point at and check isn't a deliverable; it's a hope.

### Approach / phases
How the work runs, as a sequence of gates with outputs and client checkpoints — **not a calendar**. Phases show dependency and review points; dates, if the client needs them, are a separate constraint (use `target_ship_date` in frontmatter), never baked into the phase table as a schedule.

### Acceptance criteria
The objective tests for "done," written so neither party needs to make a judgment call. Each deliverable traces to one or more criteria. This is the section that prevents the end-of-engagement standoff — invest in it.

### Assumptions & dependencies
What you're treating as true, and what you need *from the client* to proceed. The classic engagement-killer is a dependency the client owns (content, access, sign-off) that arrives late. Name each, say who provides it, and say what stalls without it — that last part is what makes it actionable rather than a footnote.

### Commercial terms (optional)
Pricing model, total, payment schedule, and — critically — the **change-request mechanism**: how out-of-scope requests get priced and approved. Omit the whole section if commercials live in a separate agreement. When present, the change-request line is what keeps "could you also just…" from eroding a fixed fee.

### Sign-off
Both parties accept the document. An unsigned SOW is a draft, not a contract.

## Status transitions

| From | To | Trigger |
|---|---|---|
| `draft` | `in-review` | Sent to the client for review |
| `in-review` | `approved` | Both parties sign off; scope and terms locked |
| `approved` | `building` | Engagement underway; SOW is now the delivery baseline |
| `building` | `shipped` | All deliverables accepted against the criteria |
| any | `shelved` | Engagement won't proceed; reason noted in body |

A material change to an `approved` or `building` SOW is a change request — re-open to `in-review` and re-sign, or issue a new SOW. Don't edit a signed contract in place.

## Anti-patterns

- **Empty or thin "Out of scope"** — the boundary that protects a fixed fee. If it's short, you haven't scoped against client assumptions.
- **Activities dressed as objectives** — "build X" instead of the outcome X delivers. The client bought the outcome.
- **Unmeasurable acceptance** — "client is satisfied" is not a criterion. If it needs a judgment call, it's not done-able.
- **Dates in the phase table** — phases are gates, not a schedule. A hard date is a constraint (`target_ship_date`), not a row.
- **Silent re-scoping** — editing a signed SOW to absorb new work. Use a change request; keep the contract honest.
- **No change-request mechanism** (when commercials are included) — every "can you also…" then erodes the fee with no path to re-price.
- **Promising more than the internal plan supports** — the SOW must not commit to scope the project-brief / one-pager can't actually deliver.

## Example

See `example.md` in this folder for a worked SOW for a small client engagement.

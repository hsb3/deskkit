---
type: sop
status: final
created: 2026-06-30
updated: 2026-06-30
tags: [meta]
---

# SOP — ux-spec

## When to write one

Write a ux-spec when you're about to build (or rebuild) **one screen** and need the layout, components, states, and interactions pinned down before code. It's the screen-level companion to the feature-spec: the feature-spec says *what the feature does and why*; the ux-spec says *exactly how this one surface looks and behaves*.

**The flow-vs-screen line — the distinction that matters most:**

| | user-journey | ux-spec |
|---|---|---|
| **Altitude** | a flow **across** screens | **one** surface in detail |
| **Answers** | "where does the user get stuck moving through this?" | "what does *this screen* contain and how does each state and interaction resolve?" |
| **Unit** | steps (5–10), each a different moment | regions, components, states, interactions of a single view |

If you find yourself writing "then they navigate to the next screen," stop — that second screen is its own ux-spec, and the hop between them belongs in a user-journey. Spec one surface per file.

**Write one if:** you're building a non-trivial screen (a list with filters and a detail panel, an editor, a multi-state form) and the empty/loading/error behavior or the edge cases aren't obvious.

**Don't if:** the surface is a single static element with no states (just describe it in the feature-spec), or you're mapping a multi-screen flow (write a user-journey).

## How to write one

1. **Name the surface and its one goal.** One screen, one job. "The capture inbox triages newly captured items" — not "the inbox plus the editor plus settings."
2. **Lay out the regions, then draw it in words.** List regions top-to-bottom, then an ASCII wireframe so the structure is unambiguous. The wireframe is the spec's spine — most disagreements surface the moment you try to draw it.
3. **Break it into named components.** Each gets a row: what it shows, what it owns, collapsed vs expanded. A reader should be able to derive the component tree.
4. **Specify all four states — this is the part people skip.** Default, empty, loading, error. A spec that only covers the populated happy path is half-written; the empty and error states are where real screens live or die.
5. **Write interactions as trigger → result.** Every meaningful action: what fires, what the user sees, whether it's optimistic or waits for a response, what feedback lands.
6. **Hunt the edge cases.** Overflow, concurrent edits from another tab, partial/failed data, the 0-item and 100-item extremes. Name the behavior for each.
7. **Note accessibility deliberately.** Keyboard order and focus management, screen-reader labels and live-region announcements, anything non-default. Don't leave it to the implementer to guess.

Keep it concrete. A ux-spec earns its keep by removing ambiguity an engineer would otherwise resolve by guessing — so favor the specific ("Submit disabled until the textarea has ≥10 chars") over the vague ("validate input").

## Status transitions

| From | To | Trigger |
|---|---|---|
| `draft` | `in-review` | First full pass shared for design/eng feedback |
| `in-review` | `approved` | Layout, states, and interactions accepted as the build target |
| `approved` | `building` | Implementation underway against this spec |
| `building` | `shipped` | Surface is live and matches the spec (or the spec was updated to match) |
| any | `shelved` | Surface cut or deferred indefinitely |

## Anti-patterns

- **Flow creep** — speccing the next screen too. One surface per file; link sibling ux-specs and the parent journey instead.
- **Happy-path-only** — no empty/loading/error states. The missing states are exactly where implementation stalls.
- **Vague interactions** — "clicking does the thing" with no result, no feedback, no optimistic-vs-wait decision.
- **No wireframe** — prose-only layout that every reader pictures differently. Draw it.
- **Accessibility as an afterthought** — omitting keyboard/focus/SR behavior guarantees it ships broken.
- **Component soup** — listing every `<div>` instead of the meaningful, nameable pieces. Spec components, not markup.

## Example

See `example.md` in this folder for a worked ux-spec on the Quillpad capture-inbox triage screen.

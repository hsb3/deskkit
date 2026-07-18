---
type: ux-spec
status: draft
created: {{date}}
updated: {{date}}
tags: []
author:
parent_product_spec:
---

# UX spec — {surface name, e.g. "Capture Inbox"}

_A screen-level spec for **one** surface: its layout, components, every state it can be in, and how each interaction resolves. If you're describing a flow that crosses screens, that's a user-journey, not a ux-spec — link to it and spec each screen here._

## Surface / screen

{The one screen this spec owns — its route/location and where it sits in the product, e.g. "`/inbox` — the triage list for newly captured items, reached from the left rail."}

## Goal

{What the user is here to accomplish on this screen, in one or two plain sentences. The single job the surface must do well.}

## Layout

_Regions top-to-bottom / left-to-right, then a wireframe-in-words so the structure is unambiguous._

- **{Region 1, e.g. header}** — {what it holds}
- **{Region 2, e.g. list / canvas}** — {what it holds}
- **{Region 3, e.g. detail panel}** — {what it holds}

```
┌──────────────────────────────────────────────┐
│ {Header region — title, count, primary action}│
├──────────────────────────────────────────────┤
│ {Filter / control row}                         │
├──────────────────────────────────────────────┤
│ {Main region — rows / cards / canvas}          │
│   {one item, collapsed}                         │
│   {one item, collapsed}                         │
└──────────────────────────────────────────────┘
```

## Components

_The discrete UI pieces this surface is built from — name each, say what it shows and what it owns. A reader should be able to turn this into a component tree._

| Component | Shows | Notes |
|----|-------|-------|
| {ItemCard} | {summary fields} | {collapsed vs expanded; what's derived vs fetched} |
| {ActionBar} | {buttons} | {variants, disabled conditions} |

## States

_Every state the surface can be in. A spec that only describes the happy path is half-written._

- **Default** — {the populated, working state}
- **Empty** — {nothing to show yet; the message and any call-to-action}
- **Loading** — {first paint / fetch in flight; skeleton vs spinner}
- **Error** — {fetch or action failed; what the user sees and how they recover}

## Interactions

_Each meaningful action as **trigger → result**. Include what's optimistic vs wait-for-response, and what feedback the user gets._

| Trigger | Result |
|---------|--------|
| {User clicks X} | {What happens, what they see, what the system does} |
| {User types / submits Y} | {…} |

## Edge cases

_The awkward inputs and concurrent conditions that break naive implementations._

| Case | Behavior |
|------|----------|
| {Long / overflowing content} | {how it's handled} |
| {Concurrent change from another session/tab} | {…} |
| {Partial / failed data} | {…} |

## Accessibility notes

- **Keyboard** — {tab order, shortcuts, focus management on open/close}
- **Screen reader** — {labels, live-region announcements for async results}
- **Contrast / target size** — {anything non-default the implementer must honor}

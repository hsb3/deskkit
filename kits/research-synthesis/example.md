---
type: research-synthesis
status: approved
created: 2026-05-12
updated: 2026-05-12
tags: [meta, process, template-design]
author: robin
affects_workstreams: [product, pm]
commissioned_by: self
---

# Survey of prior PMO Obsidian template frameworks

## Question

What patterns and artifact types from prior AIPMO attempts are worth keeping in the new v1? Which proved durable and which created friction?

## Approach

### Sources consulted

- An earlier PM-template vault — AIPMO v0 (2025): 6-artifact model (PRD, Decision, Architecture, Feature-spec, Status, Retro)
- A Notion PMO workspace export — experiments (2024): decision-tracking, sprint boards, velocity tracking
- Prior Obsidian note experiments (2023-2025): various schema attempts, tag hierarchies, status enums

### Methodology

Evaluated each framework for: (1) artifact durability (did the type serve its purpose consistently?), (2) workflow friction (did it get in the way of decision-making?), (3) audience clarity (did outputs serve their intended readers?), (4) minimal-ness (did it introduce unnecessary structure?). Focused on team and solo-founder contexts.

## Findings

**Finding 1: Six core artifact types proved durable across all frameworks.**
All three frameworks converged on roughly the same 6 types: problem statement (PRS/PRD), decision record, architecture/design, feature spec, status checkin, and retrospective. These solved the core workflows (what, why, how, did-it-work) without churn.

**Finding 2: Attempt to differentiate "executive" vs "team" artifacts created friction.**
Both Notion and early AIPMO v0 tried to maintain dual audiences (exec summaries + detailed specs). This led to duplication and kept two "source of truth" docs in sync — a losing game. Simpler: write the detail once; executives read what they need.

**Finding 3: Weekly checklists worked; velocity dashboards did not.**
Frameworks that asked "what shipped / what's blocked / what's next" generated real value. Frameworks that tried to measure velocity (points, burndown) were either ignored or manipulated. Dropped velocity tracking entirely from v1.

**Finding 4: Sop (Standard Operating Procedure) per artifact type was under-documented.**
Early frameworks had templates but no "when to write one / how to avoid pitfalls" guide. Caused scope creep and mixed granularity. The new AIPMO includes SOP as a required artifact paired with each template.

**Finding 5: Wikilinks + minimal frontmatter outperformed tag hierarchies.**
Obsidian note experiments showed that deep tag hierarchies (e.g., `team/engineering/backend/database`) became stale and were never queried. Wikilinks (`[[parent-note]]`) were natural and self-enforcing. Minimal frontmatter (5-7 fields) was more reliable than complex JSON schemas.

## Implications

**Implication 1: Lock in the 6-artifact model as v1 standard.**
Do not attempt multi-audience documents or dual artifact types. Each has one purpose, one audience, one format. Reduces decision fatigue.

**Implication 2: Add SOP as a mandatory paired artifact.**
Every template (`technical-design`, `feature-spec`, etc.) gets a SOP. This is the "guidance + anti-patterns + PM stance" living document. Prevents on-boarding friction and scope creep.

**Implication 3: Drop velocity / burndown tracking; keep flow metrics only.**
Ask "what's blocked" not "how many points"; measure cycle time or whether items shipped on target. Simpler to track, more honest.

**Implication 4: Frontmatter is the interface; keep it minimal.**
Do not introduce custom fields beyond the 5-7 core ones. If a workflow needs a new field, use it manually in a few docs first; only promote to frontmatter if 3+ documents need it. This keeps the schema readable and slow to change.

**Implication 5: Invest in wikilink UX and cross-linking discipline.**
The decomposition contract (`product-spec` → `engineering-specs`) and parent links (`engineering-spec` → `product-spec`) are the skeleton of the system. Obsidian graph view and backlinks enforcement matter.

## Open questions

- Should we version the artifact definitions if they change? (Defer: assume v1 is stable until pain point emerges)
- How do we handle artifacts that span multiple workstreams in frontmatter tagging? (Partial answer: `affects_workstreams` array; unclear if this scales)
- Is there a "zero-artifact" path for one-off decisions that don't warrant a full decision record? (Defer: try documenting inline in related artifacts first)

## Sources

- The earlier PM-template vault's README
- Its conventions reference (the methodology's system layer)
- The Notion PMO decision-tracker export
- A tag-usage analysis from the prior note-experiment backups

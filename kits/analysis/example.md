---
type: analysis
status: approved
created: 2026-05-15
updated: 2026-05-19
tags: [quillpad]
author: robin
affects_workstreams: [engineering, product]
conclusion: "Adopt capture_kit's item model with a `kind` field as the canonical content model; map agent_quillpad posts onto it as `kind: thought`."
---

# Analysis: Which content model wins the Quillpad merge?

_The study behind RAID item I-001. Quillpad merges two stalled repos with incompatible content models; the merge can't proceed until one wins. This weighs the options — the call gets ratified in a follow-up [[decision]] note._

## Question / Context

Quillpad unifies `agent_quillpad` (Astro + Firebase publishing site) and `capture_kit` (Python ingestion pipeline). The two were built around **incompatible content models**, and every downstream feature — capture, the inbox, the editor, the reader feed — needs a single schema to write against. Until one model wins, both halves stay stalled (D-002 blocks the build).

This is a genuine fork: the publishing surface wants a clean post object; the capture surface wants a flexible ingestion item. Whichever we pick, the other half has to bend to it.

## Current state

Two schemas, neither wrong, neither compatible:

- **`agent_quillpad` — post-centric.** A first-class `Post` with `title`, `body`, `slug`, `publishedAt`, `tags`. Clean for the reader feed; assumes everything is a finished essay. No notion of an un-promoted capture, no source provenance.
- **`capture_kit` — item/ingestion-centric.** A generic `Item` with `source_url`, `captured_at`, `status`, `extracted_metadata`, and a `type` discriminator (tool / paper / person / thought). Built for the inbox → triage → enrich flow. Has no rendering/publishing concept at all.

What the model has to satisfy (from the one-pager Must-Haves): capture of heterogeneous things (URLs, files, paste), auto-classify on capture, an inbox with a status workflow (inbox → draft → … → published), *and* a reader-facing feed of finished pieces.

### Metrics

- Capture types to support on day one: 4+ (tool, paper, person, thought).
- Status states in the workflow: 6 (inbox → draft → develop → polish → scheduled → published).
- Repos to reconcile onto one schema: 2.

## Options

### Option A: Post-centric (extend `agent_quillpad`'s model)

**Description:** Keep `Post` as the core object; bolt capture/triage fields onto it. A capture is just a `Post` in `status: inbox` with empty body.

**Pros:**
- Reader feed + entry-detail pages work with zero remodeling.
- Simplest mental model for the publishing half.

**Cons:**
- Forces non-essay captures (a person, a tool) into a "post" shape they don't fit — provenance and extraction metadata become awkward bolt-ons.
- The ingestion pipeline (`capture_kit`'s working code) has to be rewritten to emit Posts.
- Loses the `type` discriminator that auto-classify and triage are built around.

**Effort:** ~1.5 weeks (rewrite ingestion to emit Posts).
**Risk:** medium — throws away the one half that already works; capture flexibility is the project's whole point and this constrains it.

### Option B: Item model + `kind` field (extend `capture_kit`'s model)

**Description:** Adopt the generic `Item` as canonical, with a `kind` discriminator (tool / paper / person / thought). A publishable piece is an `Item` of `kind: thought` (or any kind promoted to `status: published`); the reader feed is a query over published items. Map `agent_quillpad`'s `Post` fields (`slug`, `publishedAt`, `body`) onto the item as render fields.

**Pros:**
- Capture, classify, triage, and the status workflow all work as-is — the working `capture_kit` code keeps running.
- `kind` cleanly models the heterogeneous capture types the one-pager requires.
- Publishing becomes a *view* over items, not a separate object — one source of truth.

**Cons:**
- Reader feed + detail pages need a thin mapping layer (item → renderable post).
- Slightly more abstract object for the publishing half to reason about.

**Effort:** ~1 week (add `kind`, add render fields, write the item→feed mapping).
**Risk:** low — keeps the working pipeline; the only new code is a mapping layer that's well-understood.

### Option C: Greenfield hybrid schema

**Description:** Design a fresh schema from scratch with a `Capture` object and a separate `Publication` object, linked. Migrate both repos onto it.

**Pros:**
- Cleanest conceptual separation of "thing I captured" vs "thing I published."
- No legacy assumptions from either repo.

**Cons:**
- Two objects + a join means more surface to build, test, and keep in sync.
- Migrates *both* repos — maximum throwaway work, directly against the one-pager's "don't become a product builder" guardrail.
- Re-litigates a model `capture_kit` already proved in practice.

**Effort:** ~3 weeks (new schema + migrate both halves + sync logic).
**Risk:** high — most code to write, biggest scope-creep vector (R-002), and the dual-object split is a maintenance tax forever.

## Comparison matrix

| Criterion | A: Post-centric | B: Item + `kind` | C: Greenfield hybrid |
|-----------|-----------------|------------------|----------------------|
| Fit to capture requirements | Poor | Strong | Strong |
| Reuses working code | No (rewrites ingestion) | Yes | No (rewrites both) |
| Effort | ~1.5 wk | ~1 wk | ~3 wk |
| Risk | Medium | Low | High |
| Scope-creep exposure (R-002) | Medium | Low | High |
| Long-term maintenance | Medium | Low | High (two objects) |

## Recommendation

**Preferred: Option B — adopt `capture_kit`'s item model with a `kind` field.** It's the only option that keeps the half that already works, models the heterogeneous captures the product depends on, and turns publishing into a view rather than a second object. It's also the lowest effort and lowest risk, and it stays well clear of the scope-creep that killed the two prior attempts (R-002). The post-centric option fights the product's core (flexible capture); the greenfield option buys conceptual purity at triple the cost and the most throwaway work.

**Trade-offs accepted:**
- The reader feed gets a thin item→renderable mapping layer instead of querying `Post` directly — a small, contained cost.
- The publishing half reasons about a slightly more abstract object (`Item` of `kind: thought`) rather than a literal `Post`.

**Next step:** record the call in a [[decision]] note — this analysis becomes its `Options Considered` — and close RAID I-001 / unblock D-002 once the decision is `accepted`.

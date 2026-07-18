---
type: sop
status: final
created: 2026-05-12
updated: 2026-05-12
tags: [meta]
---

# SOP — technical-design

## When to write one

You're designing something that spans multiple engineering-specs or affects multiple teams, and the scope is too large for a single atomic spec. **If two or more engineering-specs need the same architectural context, a TDD should exist first.**

**Granularity test**: a TDD addresses system-scale decisions (data model, infrastructure topology, API contracts, rollout strategy). Single-feature specs should stand alone.

- ❌ "Add a submit button" — use engineering-spec
- ❌ "Build a feature with three UI screens" — decompose to engineering-specs
- ✓ "Redesign the data model to support multi-tenant entries" — TDD, then engineering-specs

If you're writing a TDD because one engineering-spec feels too big, pause. Likely the spec itself isn't broken down finely enough. Write smaller specs.

## How to write one

1. **Start from the parent one-pager or product-spec.** What business problem prompted this design work? Link it.
2. **State goals and non-goals before proposing anything.** This focuses the design and prevents scope creep.
3. **Describe the architecture at system level first.** Components, data flows, major decisions. Skip implementation detail.
4. **Document trade-offs, not just the chosen path.** A reader should understand why you didn't pick the alternatives.
5. **Be specific about downstream impacts.** What constraints does this design place on engineering-specs? What APIs must they use?
6. **Address risks explicitly.** If this design has deployment complexity, data migration hazards, or new failure modes, document mitigation.
7. **Leave open questions that can't be resolved here.** Don't speculate; call them out for downstream decision.

## Status transitions

| From | To | Trigger |
|---|---|---|
| `draft` | `in-review` | Author requests review |
| `in-review` | `approved` | Reviewers sign off; all open questions resolved or deferred |
| `approved` | `building` | First related engineering-spec is opened |
| `building` | `shipped` | All derived engineering-specs are shipped |
| any | `superseded` | New TDD replaces this design; old one archived |

## Anti-patterns

- TDD covering a single engineering-spec's worth of work — split into a spec instead
- Missing alternatives section — reader can't judge the design
- Vague risk section ("there might be performance issues") — unmitigatable
- Open questions that should have been resolved before approval — blocks downstream work
- TDD written after implementation as documentation — defeats the purpose

## The PM Advisor stance

**Simplify before designing.** Most system-scale problems shrink when you cut scope. Push back on "we need a generic multi-tenant model" when you're shipping single-user first. If the TDD is more than 1,500 words, ask: what's the core decision, and what can be deferred or kept simple?

## Variants

**api-spec (`variant: api-spec`)** — a technical-design whose subject *is* an interface contract. When the design is mostly the API surface (routes/RPCs, request/response shapes, error codes, auth, versioning), write a technical-design with `variant: api-spec` and lead with the contract: an endpoint table, payload schemas, an error catalog, and the auth/versioning model. It's a qualifier of this SOP — no separate template.

## Example

See `example.md` in this folder for a complete technical-design on data modeling for a content platform.

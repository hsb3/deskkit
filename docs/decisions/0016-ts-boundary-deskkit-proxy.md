# 0016 · TS plugin boundary — extend via a designed proxy to deskkit; drift-guarded tool truth

_Settles the spec-vs-reality gap on what the TS `plugin/mcp` boundary promises._

> **Superseded by [0022](0022-collapse-onto-deskkit-single-surface.md) (2026-07-23):** the TS
> boundary and its designed deskkit proxy are retired — the four profile tools move into the Go
> binary and the whole surface ships as one plugin / one MCP server. Text kept for provenance.

- **Status:** Superseded-by-0022 (2026-07-23)
- **Date:** 2026-07-20
- **Raised by:** decision book `_meta/research/2026-07-design-session/decision-book/D7-spec-reality-reconciliation.md`; owner sign-off note 2026-07-20

## Context

The librarian spec read as if the TS `plugin/mcp` boundary could call librarian tools; the
shipped TS server carries exactly four profile tools and no librarian tools (independently
re-derived; the spec sentence judged ambiguous, leaning a client-caller reading —
Verification record, claim 3). The owner's sign-off note: "we need to/should take advantage
of the typescript plugin seam to enable server-backed capabilities. it's a significant
convenience." The platform stack (ADR 0009) routes agent integration through the Go server.

## Decision

**Extend the TS boundary via a designed proxy to deskkit** (D7 Option B as reconciled,
owner-ruled 2026-07-20):

- The TS `plugin/mcp` boundary gains server-backed capabilities by **proxying deskkit's Go
  `mcp-serve`** — never by reimplementing librarian logic in TS. The proxy design (process
  lifecycle, availability behavior, which tools surface, gating per ADR 0014) is a **scoped
  work item** that precedes any implementation.
- Meanwhile the spec is corrected to describe shipped reality (four profile tools) plus the
  planned extension, so no reader mistakes the promise for the present.
- **Tool-surface truth lives in `docs/tool-surface.md`**, pinned by a drift guard (script or
  generation) so counts can't rot again (#94's doc becomes the guarded source).

## Consequences

- The proxy honors harness-purity: `plugin/core` stays pure; the proxy lives at the server
  boundary. Fail-loud + mount-signal rules (ADR 0014) apply to the proxied surface.
- Until the proxy ships, the TS boundary's promise is explicitly "planned", not implied.
- If the proxy design surfaces a blocker (e.g. lifecycle coupling), the fallback is ADR
  0016 re-opened with Option A (amend-to-reality only) — record it, don't improvise.

## Affects

`docs/pocket-librarian-v1-spec.md` (§7.2 clause + §3.3/§5 counts) · `docs/tool-surface.md`
(+ its new drift guard in `scripts/`) · `plugin/mcp/server.ts` (eventual proxy) · the ADR
0014 contract audit · `librarian/README.md` stale counts.

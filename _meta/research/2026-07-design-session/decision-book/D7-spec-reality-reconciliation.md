---
type: analysis
status: draft
created: 2026-07-20
updated: 2026-07-20
tags: [design-session, decision-book, surface, spec-reality, tool-surface]
synopsis: D7 — the spec still promises the TS `plugin/mcp` boundary can call the librarian's
  tools directly; the shipped surface carries 4 fixed profile tools and no librarian tools at
  all. Options for reconciling spec to reality and for where tool-surface truth lives going
  forward, dependency-parented on D5's contract ruling.
---

_Decision book brief D7 (`../README.md`). Informs the session; does not itself rule. Evidence
resolves to Phase-0 dossier sections beside the prep doc; dossier claims are hypotheses until
re-derived where a ruling binds._

Status: draft (2026-07-20)

> **Owner input (2026-07-20 scope sign-off — input to weigh, not a ruling):** "we need
> to/should take advantage of the typescript plugin seam to enable server-backed
> capabilities. it's a significant convenience." This leans toward the extend-the-boundary
> option class (the TS `plugin/mcp` seam carrying server-backed capabilities) over the
> evidence-favored amend-spec-to-reality hypothesis — the session weighs it alongside the
> review finding that the spec sentence is ambiguous rather than flatly broken.
> (Recorded from `_meta/signoff/2026-07-20-decision-book-scope/answers.json`.)

> **Platform-stream interaction (2026-07-20 reboot — see `D0-platform-frame.md`):** the
> reconciliation is now three-sided. The owner's note leans extend-the-TS-seam (Option B);
> the platform stack routes agent integration through the **Go** MCP server (persona bundle
> wired to deskkit — `../platform/plan.md` R1/R2). These compose if the TS boundary's
> server-backed capabilities are delivered by **proxying deskkit's Go server** rather than
> reimplementing — exactly the undesigned proxy mechanism Option B flags. Rule them
> together, not separately.

# D7 — Spec ↔ reality reconciliation

## 1. The question

Should the spec's claim that the TS `plugin/mcp` boundary can call the librarian's tools
directly be amended to match shipped reality, or should the TS boundary be extended to deliver
what the spec promises — and, independent of that call, where should tool-surface truth live
going forward (the spec's own prose, the standing `docs/tool-surface.md`, or a generated/
drift-guarded source) and should a mechanical guard pin it?

The spec's §7.2 "Outbound MCP server" paragraph says a Claude Code or OpenCode session "**or
the dual-format plugin's `plugin/mcp` boundary**" can call the librarian's tools directly
(`docs/pocket-librarian-v1-spec.md:1786-1787`). Shipped reality is narrower on the second
clause: `plugin/mcp/server.ts` is a wholly separate TS-core server exposing exactly four fixed
profile tools (`profile_get`, `profile_validate`, `template_render`, `knowledge_index`) and no
librarian tools at all, no gate (`docs/tool-surface.md` §3). The first clause — a bare Claude
Code/OpenCode session mounting `deskkit mcp-serve` directly — is not in dispute; only the "the
dual-format plugin's `plugin/mcp` boundary" clause names a route that does not exist in the
shipped TS server. D7 rules how that gap closes and where the surface-truth documentation the
repo now has (`docs/tool-surface.md`, shipped by #94) sits relative to the spec text going
forward.

## 2. Why now

#94 shipped `docs/tool-surface.md` as an empirically-derived, counted map of all three
tool-bearing surfaces — but it is a **new, separate** document; #94 did not edit the spec text
that makes the now-disproven promise. The spec still says the TS boundary reaches librarian
tools directly, and still carries stale "six-tool core" language in its TOC, §3.3 subcommand
table, and §5 heading (the librarian's tool count is 7 in Go source, 5 default over MCP — see
`agent-symmetry.md`'s Tools row). Nothing mechanical stops this drift from recurring: the spec
lives under `docs/`, which is explicitly **exempt** from the shipped-tree neutrality scan
(CLAUDE.md "scope = `plugin/` + `librarian/` recursively; `docs/` ... are EXEMPT"), and no
drift guard ties spec prose to `plugin/core/tools.ts` or the Go tool specs the way `VERSION`
is pinned to the shipped manifests (`scripts/check-version-sync.mjs`) or `kits.yaml` is pinned
to the `kits/` tree (`scripts/check-kits.mjs`). Left unruled, the spec keeps advertising a
capability an agent following it will fail to find, and the next surface change (a fifth TS
tool, a librarian tool rename) has nothing forcing the spec or `docs/tool-surface.md` to move
with it. D5's contract ruling is a hard input here: whether the TS plugin/mcp boundary is meant
to be a third agent-integration instantiation (alongside the librarian's in-binary loop and
PM's desk-pm bundle) or deliberately profile/template-scoped tooling with librarian access
routed only through the Go surfaces is exactly what D5's "one contract, N instantiations"
ruling should settle — D7's reconciliation direction is downstream of that answer.

## 3. Evidence

- `surface-matrix.md § 0. Anchor and how this dossier extends it` — "Plugin TS MCP: reading
  `plugin/core/tools.ts:240`'s `TOOLS` array is exactly the 4 the doc names" — independent
  source-level re-check of `docs/tool-surface.md`'s TS count, no discrepancy found.
- `surface-matrix.md § 1. Surfaces legend` — S7 row: "**TS MCP server** | `plugin/mcp/server.ts`,
  stdio, over the harness-pure TS core | `TOOLS` array, `plugin/core/tools.ts:240` — 4 tools,
  fixed, no gate."
- `surface-matrix.md § 4. TS plugin lane — operations x surfaces` — the four-row table naming
  `profile_get`, `profile_validate`, `template_render`, `knowledge_index` as the TS server's
  complete operation set, each with its `plugin/core/tools.ts` line, plus: "Server registration
  for all four: `plugin/mcp/server.ts:18` ... `:29-34` ... `:37-60`. No env-var gate exists for
  this surface — fixed set of 4, always all four."
- `surface-matrix.md § 6. Unclaimed-surface findings (the ride-along problem)` — finding 4:
  "`profile_get` and `knowledge_index` are reachable but claimed by nobody. All 4 Claude Code
  plugin skills were grepped for all 4 TS tool names; only `template_render` appears
  (`desk-setup`, `brownfield-adoption`)." — even the TS boundary's existing 4 tools are
  under-claimed by the instruction layer that would need to grow if the boundary were extended.
- `agent-symmetry.md § Verdict table — the 6 "accidental / unruled" asymmetries` — row 5,
  "Spec-promised route that doesn't exist": "Still true. Spec still promises the plugin/mcp
  boundary can 'call the librarian's tools directly'; the TS server exposes only `TOOLS` from
  `plugin/core` = 4 profile tools. Spec counts still stale ('six-tool core' TOC/§3.3 table;
  'default set … four' in §7.2). #94 shipped a NEW authoritative doc (`docs/tool-surface.md`)
  rather than editing the spec, so the spec residue persists." Citation given there: spec
  `docs/pocket-librarian-v1-spec.md:1784-1790` (route), `:399` + TOC `:14` ("six-tool"), `:1789`
  ("four"); `plugin/mcp/server.ts:29-35` + `plugin/core/tools.ts:65,127,175,205` (4 tools).
- `agent-symmetry.md § New since the analysis (PR 112 — bears on the symmetry decision)` —
  "`docs/tool-surface.md` is now the authoritative surface inventory (#94) ... It documents the
  ride-along (asymmetry #2) explicitly but does not resolve it, and it does **not** correct the
  spec's stale 'six-tool core' / 'four default' language (asymmetry #5 residue)."
  (`docs/tool-surface.md:1-152`)
- `agent-symmetry.md § Gaps & uncertainties` — "the actual sentence is at `~:1784-1790` in
  current main (the outbound-MCP paragraph)" (re-located from the analysis's stale
  `spec:1775-1776`) — also names a second stale-count site: "`librarian/README.md:214-215`
  lists the mcp-serve default as ... (4) — it omits `record_feedback`, so the real default is 5
  ... noted for the spec-reconcile lane, not fixed here."

**Spec / tool-surface primary sources (read directly for this brief, not dossier-mediated):**

- `docs/pocket-librarian-v1-spec.md` TOC line 14: "5. [The six tools](#5-the-six-tools)."
- `docs/pocket-librarian-v1-spec.md:399` — subcommand table: "`mcp-serve` | Expose the
  **six-tool core** as an MCP stdio server (model-facing; §7.2) | via gated `apply_fix` only."
- `docs/pocket-librarian-v1-spec.md:1783-1799` (§7.2, "Outbound MCP server") — the full paragraph:
  "...a Claude Code or OpenCode session — or the dual-format plugin's `plugin/mcp` boundary —
  can call the librarian's tools directly... the CLI, the eino agent loop, and this MCP server
  are three surfaces over the **same** `tools.*` functions, with **zero logic duplication**..."
  followed by the model-facing gate description ("default set is `sweep`, `patrol`,
  `propose_fix`, `query`" — 4, not the 5 that `docs/tool-surface.md` and `specs.go` now show,
  missing `record_feedback`).
- `docs/tool-surface.md` §3 "Plugin TypeScript MCP server" (lines 94-107) and the Summary table
  (lines 111-123) — the authoritative, empirically-derived counts, explicitly billed as "the
  reference that replaces the informal (and wrong) 'seven-tool core' shorthand" (line 5).
- `CLAUDE.md` "Generated artifacts — never hand-edit" and "Architectural rules and their
  enforcing checks" tables — the repo's existing precedent for pinning a fact to source with a
  CI drift guard (`scripts/check-version-sync.mjs`, `scripts/check-kits.mjs`).

## 4. Options

**Option A — Amend the spec to match reality; drop the TS-boundary librarian-tools claim.**
Rewrite the §7.2 clause to say the librarian's tools are reachable via the Go `mcp-serve`
surface and the CLI, not via the TS plugin's own MCP server; retitle away from "six-tool core"
everywhere it appears (TOC, §3.3 table, §5 heading); point at `docs/tool-surface.md` for the
live counts rather than restating a number in prose. *Consequence:* cheapest option — prose-only,
no code or schema change; immediately closes the misleading-promise gap; leaves the TS boundary
exactly as shipped (profile/template/knowledge-scoped). *Hypothesis:* this is the option the
evidence most directly supports, since route 1 (a bare Claude Code/OpenCode session mounting
`deskkit mcp-serve`) already satisfies "reach the librarian's tools from Claude Code" without
touching the TS lane at all — the spec's error looks like a naming conflation ("the dual-format
plugin's `plugin/mcp` boundary") rather than a missing capability.

**Option B — Extend the TS boundary to actually deliver what the spec promises.** Add
librarian-tool proxying to `plugin/core`/`plugin/mcp/server.ts` (mechanism undetermined —
subprocess exec of `deskkit`, an HTTP call to a running `serve`, or a TS reimplementation).
*Consequence:* the most expensive option — a materially larger, ongoing-maintenance surface;
directly duplicates or re-implements what §7.2 itself calls "zero logic duplication" over the
Go `tools.*` functions, since a TS proxy is necessarily a fourth surface, not the same function
call; repeats the surface-matrix's existing unclaimed-tool problem at greater scale (the
existing 4 TS tools are already under-claimed by skills — `surface-matrix.md § 6` finding 4);
risks tension with `plugin/core`'s harness-purity gate depending on the chosen mechanism (a
subprocess spawn or network call is not obviously a "harness API," but the purity script's
actual boundary was not re-derived for this brief); overlaps D5 — if D5 rules the TS plugin is
not meant to be a third agent-integration instantiation, Option B contradicts that ruling before
it's made.

**Option C — Re-scope the amend-vs-extend call onto D5; rule only truth-location now.** Treat
the "does the TS boundary carry librarian tools" question as a D5 output (a contract parameter,
per the dependency the index already states), and use D7 now only to rule where tool-surface
truth lives and whether it's drift-guarded — leaving the spec's specific routing clause as
carried-forward residue until a D5-informed follow-up executes Option A or B. *Consequence:*
avoids ruling ahead of a dependency that could invalidate the choice; but leaves the misleading
spec promise live for at least one more design-session cycle, and defers deciding whether
`docs/tool-surface.md` should gain a drift guard against a still-undetermined final tool set.

**Option D — Defer entirely; no reconciliation this cycle.** Leave the spec's promise as known,
tracked residue (an Uncertainty / backlog pointer), revisit after D5 and after any librarian
Claude Code bundle work (README.md §4's "ship a librarian Claude Code bundle" hypothesis) lands,
since that bundle — mounting the Go MCP server directly, not the TS one — could itself be read
as satisfying route 1 of the spec's claim without any spec edit. *Consequence:* zero blast
radius now; but `docs/tool-surface.md` and the spec continue to disagree with no pointer between
them, and a reader who hits the spec first (it's the older, more narratively complete document)
gets the wrong story with no forcing function to notice.

## 5. Decision criteria

- **Harness-purity of `plugin/core`** (CLAUDE.md; enforced by `scripts/check-core-purity.mjs`)
  binds directly on Option B — any librarian-tool proxy added to `plugin/core` must not import a
  harness/runtime API; the exact boundary (subprocess spawn, network call) was not tested by this
  brief and needs its own check before Option B could proceed.
- **Identity-neutrality scope** (CLAUDE.md: `docs/` is EXEMPT from `scripts/check-neutrality.mjs`)
  means no existing gate will ever catch spec-vs-reality drift by itself — any "truth lives in
  the spec" ruling (Option A) needs its own discipline (the pointer-not-restate pattern) since it
  cannot lean on a CI scan the way shipped-tree identity claims can.
- **Makefile-as-task-interface** (CLAUDE.md §"Commands"; "gates run bare, never piped") binds on
  any drift-guard option: a new `scripts/check-tool-surface.mjs`-style script must be wired into
  `make check`, not left an unwired standalone script, to match the repo's existing gate pattern
  (`check-version-sync.mjs`, `check-kits.mjs`, both listed in CLAUDE.md's enforcing-checks table).
- **Generated-artifacts precedent** (CLAUDE.md "Generated artifacts — never hand-edit" +
  "Architectural rules and their enforcing checks" tables) is the repo's own stated norm for this
  exact shape of problem — a fact that can drift between a hand-authored doc and source gets a
  regeneration script + a CI drift check, not a periodically-refreshed prose doc. Any ruling that
  keeps `docs/tool-surface.md` or spec counts hand-maintained should say explicitly why this
  precedent doesn't apply here, if it doesn't.
- **D5 dependency** (`README.md` "Known interactions": "D5→D7 — what the TS boundary should
  promise is a contract parameter") — a D7 ruling that commits to Option B before D5 rules the
  integration-contract shape risks being overturned; any ruling should record whether it treats
  D5 as already-effectively-answered or genuinely blocking.

## 6. Blast radius

- **Option A:** `docs/pocket-librarian-v1-spec.md` — TOC line 14, §3.3 table line 399, §5
  heading, §7.2 paragraph lines 1783-1799 (all "six-tool"/stale-count sites); no `schema/`
  change; no code change; no new CI gate under a narrow reading of this option alone. The
  dossier-flagged second stale site, `librarian/README.md:214-215`, is adjacent residue a
  spec-truth ruling would plausibly also want fixed, though it is not itself a spec file.
- **Option B:** `plugin/core/tools.ts`, `plugin/core/*` (new tool definitions + proxy
  mechanism), `plugin/mcp/server.ts`; regenerated `plugin/claude-plugin/mcp/server.js` +
  `plugin/claude-plugin/schema/` via `make package` (existing CI `git diff --exit-code` drift
  guard already covers this regeneration); `scripts/check-core-purity.mjs` boundary review; at
  least one new/updated Claude Code skill to claim the new tools (else this option imports the
  unclaimed-tool problem at larger scale); `docs/tool-surface.md` §3 + Summary table counts
  change from 4 to 4+N.
- **Option C:** no spec text change yet; `docs/tool-surface.md` gains a `status`/authority
  annotation per whatever the truth-location ruling says; a new `scripts/check-tool-surface.mjs`
  (or equivalent) comparing `plugin/core/tools.ts`'s `TOOLS.length` and the Go tool specs'
  counts against the doc's stated numbers, wired into `make check`; CLAUDE.md's
  "Architectural rules and their enforcing checks" table gains a row.
- **Option D:** none now; the residue is carried as an explicit backlog/Uncertainty item.

## 7. Out of scope / interactions

- D7 does not decide whether a librarian Claude Code bundle ships (README.md §4's hypothesis,
  grounded in `agent-symmetry.md`'s asymmetry #1) — that bundle would mount the Go MCP server
  (S1) directly, a different mechanism from any TS-boundary extension, and could independently
  satisfy the spec's "Claude Code session ... can call the librarian's tools directly" clause
  without Option B ever being built. That ruling belongs to D5/C1, not here.
- D7 does not decide the per-module-vs-shared-mount question (README.md §4, §5 "tough spots")
  — that governs the Go MCP surface's shape, not the TS boundary this brief is about.
- D7 does not re-open the unclaimed-tool findings themselves (`surface-matrix.md § 6`, all six)
  as a general problem — it only notes that Option B would add to that specific debt if chosen
  without a matching skill-instruction update. The general unclaimed-surface ruling belongs to
  D5/C1.
- D5 owns "what the TS boundary should promise" as a contract parameter (README.md "Known
  interactions"); D7 assumes whatever D5 rules there as an input to the amend-vs-extend call
  and does not attempt to pre-empt it.

## 8. Uncertainties

- **Reading of "the dual-format plugin's `plugin/mcp` boundary."** This brief's own read of the
  raw spec sentence (§3, primary-source cite) separates it into two distinct sub-claims — a
  bare Claude Code/OpenCode session reaching `mcp-serve` directly (real today) vs. the TS
  plugin's own server proxying librarian tools (not real) — and leans on that separation to
  favor Option A. Neither dossier disambiguates the sentence this way; `agent-symmetry.md`'s row
  5 treats it as one claim ("the TS server exposes only 4 profile tools"). This split is this
  brief's own interpretation, not a dossier finding, and should be checked against the spec
  author's intent (or re-derived live) before it drives a ruling.
- **Option B's proxy mechanism is undetermined.** No dossier or spec section specifies how a TS
  proxy would reach the Go tool core (subprocess, HTTP, reimplementation); if chosen, this needs
  its own design pass, including a fresh read of `scripts/check-core-purity.mjs`'s actual
  boundary (not re-derived here) to know which mechanisms trip it.
- **Third stale-count site.** `agent-symmetry.md`'s Gaps section names `librarian/README.md:214-
  215` as stale the same way (misses `record_feedback`) — adjacent to D7's scope but a different
  file (product README, not the spec); flagged, not folded into blast radius as an owned edit.
- **Drift-guard feasibility for prose.** Option A's "point at `docs/tool-surface.md` instead of
  restating a number" reduces drift but isn't itself mechanically checkable (a CI script can
  diff `TOOLS.length` against a number, not verify a paragraph "points" correctly) — whether
  that residual risk is acceptable, or Option C/a hybrid is preferred, is for the session.
- **No re-run of `docs/tool-surface.md`'s empirical probe.** Per `surface-matrix.md § 8`, the
  counts here were source-verified, not re-run against a live build; re-run the JSON-RPC probe
  in `docs/tool-surface.md`'s "How the counts were derived" section for a fresh empirical count.

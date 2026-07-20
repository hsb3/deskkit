> **Tracking:** #TBD, ADR 0016 (2026-07-20 design session). Pin `docs/tool-surface.md`'s tool
> counts to source with a mechanical drift guard, so the next tool add/remove cannot silently
> make the doc wrong the way the old "seven-tool core" shorthand was.

## Problem

`docs/tool-surface.md` (shipped by #94) is the repo's authoritative, empirically-derived map of
every tool-bearing surface: Librarian CLI subcommands (16 base, + the `pm` group under
`PM_ENABLED`, `docs/tool-surface.md:28-56`), the Librarian MCP server's gate-dependent counts (5 / 6 / 17 / 18
depending on `LIBRARIAN_AUTONOMOUS_WRITES` and `PM_ENABLED`, `docs/tool-surface.md:59-91`), and
the Plugin TS MCP server's fixed count of 4 (`docs/tool-surface.md:94-107`), summarized at
`docs/tool-surface.md:111-124`. ADR 0016 rules: "Tool-surface truth lives in
`docs/tool-surface.md`, pinned by a drift guard (script or generation) so counts can't rot again
(#94's doc becomes the guarded source)" (`docs/decisions/0016-ts-boundary-deskkit-proxy.md:29-30`).

Today nothing enforces that pin. The doc's own closing line says re-verification is a manual,
remember-to-do-it step: "To re-verify after any tool add/remove, re-run the probe above; the
numbers in this doc must match" (`docs/tool-surface.md:152`) — there is no CI check behind that
sentence. The design-session evidence names the exact gap: "no drift guard ties spec prose to
`plugin/core/tools.ts` or the Go tool specs the way `VERSION` is pinned to the shipped manifests
(`scripts/check-version-sync.mjs`) or `kits.yaml` is pinned to the `kits/` tree
(`scripts/check-kits.mjs`)"
(`_meta/research/2026-07-design-session/decision-book/D7-spec-reality-reconciliation.md:66-69`).
The doc is fresh and currently correct — a source-level re-check found no discrepancy
(`_meta/research/2026-07-design-session/surface-matrix.md § 0`, "`plugin/core/tools.ts:240`'s
`TOOLS` array is exactly the 4 the doc names") — so the residual problem is entirely about the
NEXT change, not the current state: this is exactly the failure mode that produced the original
dispute the doc narrates in its own "Why counts were disputed" section (`docs/tool-surface.md:22-24`).

Where the real registrations live, so a guard has something to check against:

- **Plugin TS MCP server** — the `TOOLS` array in `plugin/core/tools.ts:240`
  (`[profileGet, profileValidate, templateRender, knowledgeIndexTool]`), consumed unmodified by
  `plugin/mcp/server.ts:18,29-35` (`ListTools` maps `TOOLS` 1:1). A static count of this array's
  length is exact and cheap — no gating, no build step needed.
- **Librarian CLI** — subcommands registered in `librarian/cmd/deskkit/main.go`
  (`registerToolCommands`, plus `registerPMCommands` gated by `PM_ENABLED`), per
  `docs/tool-surface.md:34-53`.
- **Librarian MCP server** — the gated subset via `toolcore.ExposedTools(cfg)`
  (`librarian/internal/core/mcp/server.go`), with per-tool `AgentDefault`/`AgentGated` flags in
  `librarian/internal/modules/librarian/tools/specs.go` and
  `librarian/internal/modules/pm/tools/specs.go`. This count is **not** statically obvious from
  source the way the TS array is — it depends on two independent env-var gates
  (`docs/tool-surface.md:63-76`), and the doc's own derivation method was a live JSON-RPC probe
  against a built binary (`docs/tool-surface.md:127-152`), not a source read.

## Deliverables

- A — `scripts/check-tool-surface.mjs` (recommended name, following the house pattern of
  `scripts/check-kits.mjs` / `scripts/check-version-sync.mjs`: plain Node, no deps). Its job:
  parse the counts `docs/tool-surface.md` asserts and cross-check each against source.
  - The TS surface count is cheap and exact in-language: count `plugin/core/tools.ts`'s `TOOLS`
    array entries (or import/parse the file) and diff against the doc's stated "4" (§3 + Summary
    table).
  - The CLI surface count (16 base) can likely be derived the same text-parsing way
    `check-kits.mjs` parses `kits.yaml` — count `registerToolCommands`/`registerPMCommands`
    call sites or the `AddCommand` list in `librarian/cmd/deskkit/main.go`.
  - The MCP gated counts (5/6/17/18) are the contested part — **OPEN DESIGN QUESTION, default
    given below, keep it open for the builder:**
    - *Default recommendation:* don't reimplement Go's two-flag gate arithmetic inside the JS
      guard (a second copy that can itself drift). Instead add a Go-side assertion — a test
      under `librarian/` (near `toolcore.ExposedTools` or the specs files) that computes
      `len(ExposedTools(cfg))` for the four gate combinations and asserts them against the same
      numbers `docs/tool-surface.md` states, riding the existing `go test ./...` lane
      (already part of `make test`). This keeps each language checking what it can see
      cheaply, at the cost of the guard being split across two commands instead of one.
    - *Alternative, if a single script is preferred:* have `scripts/check-tool-surface.mjs`
      build the `deskkit` binary and re-run the doc's own JSON-RPC probe
      (`docs/tool-surface.md:135-146`) for each gate combination — a single-language guard, but
      it adds a Go build step to `make check`, which today runs build-free
      (`make check` is neutrality + self-test, kit drift, scaffold frontmatter, plugin core
      purity, actionlint — no build). Flag this tradeoff explicitly in the PR that implements it.
  - A self-test mode (`--self-test`), mirroring `scripts/check-neutrality.mjs`'s pattern: seed a
    scratch fixture tree with an added/removed tool and an unchanged doc snippet, assert the
    guard exits 1 naming the discrepancy, then assert a matched fixture exits 0. This is the
    mechanical proof behind the acceptance criterion below, not a hand-run check.
- B — `make check` wiring: add the new script (and, if the split design is used, the new Go
  test's presence) to the `check:` recipe in `Makefile`, matching the existing five-step order
  (neutrality + self-test, kit drift, scaffold frontmatter, plugin core purity, actionlint).
- C — a row in `CLAUDE.md`'s "Architectural rules and their enforcing checks" table naming the
  new guard, per CLAUDE.md's own discipline ("Update CLAUDE.md in the same change as the code it
  describes").
- D — whatever structural change `docs/tool-surface.md` needs to stay machine-checkable.
  *Default recommendation:* none — keep the doc's current prose + tables, and have the guard's
  parser target the exact lines already there (the "Count: 4, fixed" sentence in §3, the numeric
  column of the Summary table at `docs/tool-surface.md:111-124`). This avoids forcing a format
  change on a doc that was only just written (#94). *Open alternative* if prose-parsing proves
  too fragile in practice: a small machine-readable block (e.g. an HTML-comment JSON block) the
  guard reads directly instead of parsing prose — keep this contested, try the no-format-change
  default first.

## Acceptance criteria

- [ ] `node scripts/check-tool-surface.mjs` exits 0 against the tree at merge time (every count
      `docs/tool-surface.md` states verified against its source).
- [ ] The guard's self-test mode (`--self-test` or equivalent) seeds a fixture where a tool is
      added or removed without a matching doc edit, and asserts the guard exits 1, naming which
      surface/count disagrees — the mechanical proof that "the guard FAILS when a tool is
      added/removed without the doc changing."
- [ ] `make check` runs the new guard (visible in the `Makefile` `check:` recipe and therefore in
      CI's aggregate check job).
- [ ] If the MCP-gated counts are verified by a separate Go test (per the default design above),
      that test is named in the implementing PR and its pass is shown (`go test ./...` output
      including the new assertion).
- [ ] `CLAUDE.md`'s "Architectural rules and their enforcing checks" table carries a row for the
      new guard.

## Dependencies & gates

- Not blocked by anything; independent of `ts-proxy-design` (this issue only guards the doc that
  the proxy will later need to extend — see the Out of scope note below).
- `make check` — this issue's own deliverable IS a new step in this gate; it will fire on every
  future change, which is the point.
- `make verify` (librarian integration gate, `librarian/verify.sh`) does **NOT** fire for this
  issue's own change unless the chosen design adds a Go test under `librarian/` (per the default
  recommendation above) — a JS-only guard under `scripts/` plus a `docs/` edit touches neither
  `librarian/` nor triggers `make verify` on its own. Be explicit in the implementing PR about
  which of the two applies.
- Identity-neutrality (`node scripts/check-neutrality.mjs`) does **NOT** scan `scripts/` or
  `docs/` — its scan scope is `plugin/`, `librarian/`, `kits/` only
  (`scripts/check-neutrality.mjs:52`, confirmed in `_meta/plans/_config.md`'s gate menu). Nothing
  in this issue needs identity-neutral phrasing to pass CI, though the script/doc should stay
  in that spirit for consistency with the rest of the repo.
- Kit-drift (`check-kits.mjs`) and version-sync (`check-version-sync.mjs`) do not fire — no
  `kits/` or manifest touch.
- CHANGELOG: this is a product-facing tooling change, so it needs an `[Unreleased]` entry per
  `_meta/plans/_config.md`'s CHANGELOG gate row ("any product change").

## Out of scope

- Changing any tool surface (adding, removing, or renaming a tool) — this issue only guards
  `docs/tool-surface.md` against future drift; it does not correct or alter today's counts.
- The TS-to-`deskkit` proxy itself (`ts-proxy-design`, tracked separately) — not designed or
  implemented here.
- Note for whoever builds the proxy later: when ADR 0016's proxy ships, its new tools become a
  new counted source location (either a new row under the Plugin TS MCP surface, or a fourth
  surface entirely, depending on how the proxy's design doc resolves "which tools surface"). The
  guard built by this issue must be extended to read that new source at that time — this issue
  does not attempt to pre-build for a shape that isn't designed yet.

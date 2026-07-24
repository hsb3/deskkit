# 0022 · Collapse the plugin surface onto deskkit — one binary, one MCP server, one bundle

_Supersedes 0016: retire the TS MCP server and its proxy design; the Go `deskkit` binary carries the whole tool surface, shipped as a single plugin._

- **Status:** Accepted (2026-07-23)
- **Date:** 2026-07-23
- **Raised by:** early-user adoption feedback (2026-07-23, "the plugin is super-confusing — why two
  plugins and two MCP servers? just make it one") + owner rulings the same session. Roadmap:
  `_meta/plans/adoption-feedback-roadmap.md`.

## Context

A user installing desk-standard today installs **two** marketplace plugins and gets **two** MCP
servers:

- `plugins/desk-standard/` — a harness-pure TypeScript bundle whose `.mcp.json` runs
  `bun mcp/server.js`, exposing the four profile tools (`profile_get`, `profile_validate`,
  `template_render`, `knowledge_index`) plus the setup/convention skills.
- `plugins/desk-persona/` — whose `.mcp.json` runs the Go binary `deskkit mcp-serve` with
  `MCP_MODULES=librarian,pm`, exposing the librarian + PM tool families plus two agents, three PM
  skills, and a SessionStart hook.

Both are listed side by side in `.claude-plugin/marketplace.json`. The two servers are
un-composed — "neither calls the other; they coexist only by both appearing in a session's MCP
config" (`docs/development/ts-proxy-design.md`).

**ADR 0016** settled the spec-vs-reality gap on the TS boundary by choosing to *keep the TS server
and extend it via a designed proxy to deskkit* — the TS server would spawn `deskkit mcp-serve` as a
child and re-expose librarian tools through one mount, "never by reimplementing librarian logic in
TS." That proxy was designed (`docs/development/ts-proxy-design.md`) but **never built**; its own
slice 0 is a spawn-feasibility go/no-go probe, and it explicitly kept PM on a separate mount, so
even the designed unification did not collapse to a single server. **ADR 0014** packaged the agent
surface as one composed desk-persona bundle and ruled per-module mounts out, but left the TS
profile bundle as a separate shipped artifact on a separate runtime.

The owner reframed the goal after the feedback: **the Go binary is required for all the real value**
— librarian, PM, the record-original write boundary, the embedded store, and the visual data
browser all live in `deskkit`. A user who wants any of that must install the binary regardless. A
second TypeScript MCP server and a second plugin therefore add adoption friction and a confusing
mental model **without buying the user anything** the binary does not already require. The
harness-purity of `plugin/core` — 0016's reason to keep the TS boundary separate and bridge it via a
proxy — is an internal engineering property, not a user benefit, and it is not worth two servers.

## Decision

**Owner-ruled 2026-07-23: collapse the entire shipped surface onto the Go `deskkit` binary.** The
four calls:

- **(a) One MCP server.** Reimplement the four profile tools (`profile_get`, `profile_validate`,
  `template_render`, `knowledge_index`) in Go and expose them from `deskkit mcp-serve` behind a new
  `MCP_MODULES` value (`profile`), reusing what the Go lane already has — `config.DiscoverProfile` /
  `LoadProfile`, `schema/profile.schema.yaml`, the fixer `templates.Render`, and the librarian
  `files` index. The TypeScript MCP server (`plugin/mcp/server.ts` → the generated
  `plugins/desk-standard/mcp/server.js`) is **retired**; its emit leaves `make package`.

- **(b) One plugin bundle.** The two marketplace bundles collapse into a single installable plugin:
  one `.mcp.json` running `deskkit mcp-serve` with `MCP_MODULES=profile,librarian,pm`, carrying the
  merged skills (`desk-setup`, `conventions-standard`, `harvest-loop`, `brownfield-adoption`,
  `pm-session-open`, `pm-advance-item`, `pm-triage`), both agents (`librarian-operator`,
  `pm-operator`), and the SessionStart hook. `.claude-plugin/marketplace.json` lists **one** plugin.

- **(c) Harness-purity stops being load-bearing for the shipped surface.** `plugin/core` may remain
  as an internal/reference library, but it ships no MCP server; the `check-core-purity` guard's
  scope narrows to that reduced role or retires with the TS server. The two-lane split
  (`plugin/` TS ↔ `librarian/` Go) is no longer a *shipping* boundary — Go is the one runtime the
  product ships behind.

- **(d) The ts-proxy design is withdrawn.** The collapse reaches the same "one mount" outcome the
  0016 proxy aimed at, without a spawned-child process — removing exactly the lifecycle-coupling
  risk the proxy design flagged as its central unknown (slice 0). `docs/development/ts-proxy-design.md`
  is marked withdrawn.

**Fidelity rule (mirrors 0016's "record the fallback, don't improvise").** If a specific profile
tool cannot port 1:1 to Go — e.g. `template_render` or `knowledge_index` carries TS-specific
behavior — the loss is **documented** and that one capability may keep a thin, explicit shell-out;
the fallback is never to reopen the two-server split.

This **supersedes ADR 0016** (on acceptance, 0016's status → `Superseded-by-0022` and this row in
the index flips from Proposed to Accepted). It **amends ADR 0014**: the desk-persona bundle absorbs
the profile tools, so there is genuinely one bundle (not the two 0014 left standing), and the TS
mount 0014 assumed is gone. ADR 0014's core — the integration contract, tool-level gating, and the
librarian-only in-binary loop — is **unchanged**; that amendment lands as a dated note on 0014 in
the same change that builds the collapse.

## Consequences

- **The user-facing model is what the feedback asked for:** one plugin, one MCP server, one binary.
  "deskkit is the engine; the plugin is its Claude Code face" becomes literally true.
- **Tool surface** on `deskkit mcp-serve` now spans profile + librarian + PM. `docs/tool-surface.md`
  and its drift guard (`scripts/check-tool-surface.mjs`) update; the count lives there, not in this
  ADR.
- **Neutrality** applies to the new Go profile tools (they land under `librarian/`, inside
  `check-neutrality`'s scan scope) — they must stay identity-neutral independent of any prior TS
  scan.
- **`plugin/` becomes non-shipping.** `plugin/core` / `plugin/mcp` either stay as a reference
  library or are removed in the build; `make package` no longer generates a TS server; the bundle
  drift guard now guards only the single collapsed bundle.
- **Backlog reconciliation:** this moots **#199** (the ts-proxy doc correction) and the unfilled
  ts-proxy build — note both as superseded when the collapse issue is filed. It **simplifies #12**
  (OpenCode dual-format): there is one Go MCP server to expose to a second harness, not a TS server
  to fan out.
- **Sequencing:** the build (roadmap Wave 3) is **gated on this ADR being accepted.**

## Affects

- Supersedes [ADR 0016](0016-ts-boundary-deskkit-proxy.md) (status flip on acceptance) · amends
  [ADR 0014](0014-agent-integration-contract.md) (dated note at build; one bundle, TS mount retired).
- Withdraws `docs/development/ts-proxy-design.md`.
- `librarian/internal/core/mcp/server.go` (register profile tools; new `MCP_MODULES=profile`) + the
  new Go profile-tool implementations (reusing `config`/`schema`/`templates`/`files`-index).
- `plugin/mcp/server.ts`, `plugin/package.json` (retire TS-server emit), `scripts/check-core-purity.mjs`
  (scope) · `plugins/desk-standard/` + `plugins/desk-persona/` → one bundle ·
  `.claude-plugin/marketplace.json` (one entry).
- `docs/tool-surface.md` (+ `scripts/check-tool-surface.mjs`) · `CLAUDE.md` (architecture section) ·
  `librarian/README.md`.

## Provenance

Early-user feedback + owner rulings, 2026-07-23 (design session, this repo). Roadmap and full design
detail: `_meta/plans/adoption-feedback-roadmap.md`. Decision recorded in project memory
`consolidation-single-binary-decision`. Note: the `trigger-design` (#127) and
`prompt-tuning-centralized` (#128) plan drafts both provisionally eyed ADR **0022** for their own
future records — with this ADR taking 0022, those take the next free number at landing time (their
plans already hedge this).

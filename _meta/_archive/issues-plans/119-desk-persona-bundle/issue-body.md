> **Tracking:** #119, ADR 0014 (2026-07-20 design session). Ship ONE desk-persona Claude Code
> bundle that composes the librarian + PM personas under the integration contract — the
> platform's v1 proof surface (ADR 0009) — as a new shipped, drift-guarded, identity-neutral
> artifact.

## Problem

ADR 0014(a) rules that the platform's "desk agent" is the composition of the librarian + PM
personas under the contract — ONE bundle, not a separate librarian bundle and not a third
instantiation. Today only two fronts exist: the `desk-standard` plugin (4 profile tools,
`.claude-plugin/marketplace.json` -> `./plugin/claude-plugin`) and the `desk-pm` bundle
(`plugin/desk-pm/`, mounting `deskkit mcp-serve` with `PM_ENABLED=true`). The librarian's only
Claude Code path is a hand-authored README snippet (`_meta/research/2026-07-design-session/agent-symmetry.md`
comparison-table row "Claude Code mount"; `.claude-plugin/marketplace.json` lists only
`desk-standard` + `desk-pm`). There is no composed desk persona, so:

- the librarian tools a session actually needs (`sweep patrol propose_fix query record_feedback`)
  are claimed by no persona or skill (`_meta/research/2026-07-design-session/surface-matrix.md`
  findings 1 and 4); and
- the "fit to built-in, bring your own harness" story is asymmetric where it is most visible — one
  product, two very different front doors.

The bundle is a NEW SHIPPED ARTIFACT: it enters the neutrality-lint surface
(`scripts/check-neutrality.mjs:52`, scan dirs `plugin/`, `librarian/`, `kits/`), needs its own
packaging + drift guard (ADR 0014 Consequences), and — per ADR 0015 — its persona markdown becomes
one of the version-controlled prompt copies that must not silently diverge from the canonical
source.

## Deliverables

- **A - Bundle layout.** A new `plugin/desk-persona/` tree mirroring `plugin/desk-pm/`'s shape:
  `.claude-plugin/plugin.json`, `.mcp.json` (mounts `deskkit mcp-serve`, declaring the module set
  the contract's gating expects), `agents/` (the composed desk persona), `skills/`, `README.md`.
  (Target under `plugin/` recommended; see Open questions.)
- **B - Composed persona / skill / agent contents.** The desk persona composed from the contract:
  the librarian persona (claiming `sweep patrol propose_fix query record_feedback` and, when
  gated on, `apply_fix`) plus the PM persona/skills (the `pm-operator` claim of the 12 PM tools).
  The PM contribution is composed from the existing desk-pm content, not re-authored, so there is
  one source per surface (ADR 0014(d), ADR 0015).
- **C - Packaging integration.** A marketplace entry in `.claude-plugin/marketplace.json`
  (source `./plugin/desk-persona`, versioned) and its `.claude-plugin/plugin.json`, both added to
  the `check-version-sync.mjs` SOURCES so `VERSION` stays canonical. Like desk-pm, the bundle
  mounts the deskkit binary via `.mcp.json` and carries no generated TS `server.js`, so it does not
  need the `plugin` `package` bun-build step.
- **D - Drift guard.** A CI guard so the bundle's librarian-persona content cannot be hand-edited
  out of sync with the canonical librarian instruction source (`librarian/templates/librarian-system-prompt.txt`
  per ADR 0015). Recommended: generate the bundle's librarian-persona body from the canonical
  source and guard with `regenerate + git diff --exit-code` (the repo's generated-artifact pattern).
- **E - Neutrality compliance + docs.** Identity-neutral persona/skill text (no person/org/repo/issue),
  and docs: a bundle README and the marketplace description, plus a pointer from the docs index.

## Acceptance criteria

- [ ] `node scripts/check-neutrality.mjs` passes with `plugin/desk-persona/` present, and
      `--self-test` still detects a seeded violation.
- [ ] The bundle's composed persona claims, by name, the previously unclaimed librarian tools it
      should own per the contract (`sweep patrol propose_fix query record_feedback`); a grep/test
      asserts their presence.
- [ ] The drift guard fails (non-zero) when the bundle's librarian-persona body is hand-edited away
      from the canonical source, and passes after regeneration.
- [ ] `node scripts/check-version-sync.mjs` passes with the new `plugin.json` + marketplace entry
      wired into its SOURCES, all at `VERSION`.
- [ ] The marketplace lists exactly the intended plugin set (the composed desk-persona bundle
      present; no duplicate PM persona source — see `_meta/plans/desk-persona-bundle/plan.md`
      Open questions on desk-pm's disposition).

## Dependencies & gates

- **Blocked by:** agent-integration-contract (#114) — the bundle instantiates the contract (its
  persona claims, its mount's declared module set, and its drift-guarded prompt copy all reference
  the contract this issue cannot pre-empt).
- **Fires.** `make check` neutrality (`scripts/check-neutrality.mjs`) — a new persona under
  `plugin/`; `check-version-sync.mjs` (new manifest + marketplace entry); the new bundle drift
  guard; `make test` (plugin `bun test`) if a bundle test is added — `plugin/package.json:13`'s
  test glob is an explicit dir list (`bun test core mcp opencode desk-pm`) that would not
  discover a `plugin/desk-persona/` test on its own. Default: the bundle's file scope also gains
  `plugin/package.json` (append `desk-persona` to the test glob) when a bun-side test is added;
  otherwise the bundle's checks ride `scripts/`/Go instead. Regression-test bar (#82): the
  drift guard is the red-able check.
- **Does NOT fire.** No DB migration. No librarian `go test`/`make verify` change (the bundle adds
  no Go code; it mounts the existing binary). No `plugin/claude-plugin/` server.js drift guard (the
  bundle carries no generated TS server). No kits drift.
- **Depends on:** ADR 0015 (`docs/decisions/0015-prompt-governance.md`) — git-is-truth makes the
  bundle markdown a guarded prompt copy (deliverable D).

## Out of scope

- The contract definition, mount gating, prompt fix, eino slice, and surface audit — the
  agent-integration-contract issue (this issue's blocker).
- A separate `desk-librarian` bundle — explicitly ruled out (ADR 0014(a): the desk agent is the
  composition, not a third instantiation).
- Chat-to-schema capture — rides the contract as a claimed tool later.
- Promoting `import` to a bundle-claimed tool — reserved for D8.

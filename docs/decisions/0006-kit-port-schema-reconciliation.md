_ADR for porting the 23 headcase SOP kits into this repo and reconciling their doc types with
schema v1 — what moved, what was neutralized, and the disposition of every kit-type schema gap._
Status: Accepted (corrected 2026-07-18, 2026-07-21)

# 0006 — SOP kit port + schema-v1 doc-type reconciliation

> **Correction (2026-07-18):** Decision item 2 originally asserted "No GitHub/issue/org/path/email
> refs were present in the kits." That was FALSE. Independent verification found **9 private-vault
> path references** carried verbatim from the origin vault: eight `[[_headcase/agents/pm/resources/decision]]`
> wikilinks (in `kits/analysis/{template,guide,example}.md` and `kits/postmortem/guide.md`) and one
> `[[1_Engineering/headcase-obsidian]]` link (`kits/weekly-checkin/example.md`). All nine were
> rewritten to neutral targets (`[[decision]]` / `[[quillpad-settings-ia]]`) and the vault name was
> scrubbed from every shipped surface. A static blocklist gate (`scripts/check-kits.mjs`, in
> `make check` + CI) now enforces this over `kits/` + `schema/` so the class of leak cannot recur.
> The false sentence is struck below.

> **Correction (2026-07-21):** Two places overstated kit *consumption*. Decision item 1 called
> `kits/` *"consumed by both lanes (plugin skills today; the PM module's document gates next)"* and
> the first Alternative said *"Kits are consumed by the librarian/PM module too, not just the
> plugin."* Both are FALSE as of v0.7.0: **no lane consumes `kits/` at runtime.** No shipped plugin
> skill references kits — `plugin/claude-plugin/skills/` (the four skills `desk-setup`,
> `conventions-standard`, `harvest-loop`, `brownfield-adoption`) has zero `kit` references, and no
> file under `plugin/` references the `kits/` path at all. The librarian binary can't reach `kits/`
> either (`go:embed` can't cross `..`; kits sit at repo root by design). Today's only consumers are
> the **contract/guard layer**: the drift + origin-vault blocklist guard `scripts/check-kits.mjs`,
> the neutrality scan surface (`scripts/check-neutrality.mjs` `SCAN_DIRS` includes `kits`), and the
> render-**compatibility** guard `librarian/templates/kit_render_test.go` (reads
> `kits/analysis/template.md`, renders it through the fixer's `Render()` — a compat check, not an
> integration). Runtime consumption is deferred: librarian template-selection (needs a design
> ruling) and the PM module's document gates (#55 D2/D3). Both false phrases are struck below.
> Evidence: `plugin/claude-plugin/skills/` listing (four skills, no `kit` hit) ·
> `scripts/check-neutrality.mjs:52` (`SCAN_DIRS = ["plugin", "librarian", "kits"]`) ·
> `scripts/check-kits.mjs:18` (`KITS_DIR`) · `librarian/templates/kit_render_test.go`.

## Context

The desk's authoring templates — 23 SOP "kits" (a guide + template + example per doc type) —
lived only in the headcase Obsidian vault (`hsb-2026/_headcase/shared/sops/`), outside the plugin
they were meant to standardize. Desk decision **0013 S4(a)** ruled the end-state: kits + their
schema live in this repo behind a `kits.yaml` manifest, and the vault copies freeze into a
read-only journal. Routing was ruled **Option B — port now** (dev-tooling desk
`0018-sop-kit-routing.md`, via the PM-system ruling form), ahead of the PM-system build, because
that build gates work-item transitions on schema-conforming documents and the kits are the
definition of "filled" per doc type. Work order: `hsb3/desk-standard#49` (epic #55).

Owner intent (verbatim, #49): adopt the artifacts/SOPs and frontmatter schema *with tracking to /
inspiration from* headcase — **not a byte-frozen copy**; refinements to the schemas are **expected**
to make them compatible with database-backed workflows; flag refinements, don't silently fork.

## Decision

**1. Kit home = top-level `kits/`.** One subdir per kit, the guide/template/example (G/T/E)
structure preserved. `kits/` is a product-neutral shared surface, peer to `schema/`,
~~consumed by both lanes (plugin skills today; the PM module's document gates next)~~ (struck —
see the **2026-07-21 Correction** above: no lane consumes `kits/` at runtime today; only the
contract/guard layer references it, and lane consumption is deferred). Indexed by the root
**`kits.yaml`** manifest (the 0013 S4(a) mechanism).

**2. Neutralized per the identity-neutrality constraint (0013 item 9).** The vault examples were a
coherent worked narrative about a real personal project. Deterministic, coherence-preserving
neutralization: `Henry → Robin` (persona), `Toolbox → Quillpad` (sample product), `The Intern →
the Assistant` (sample agent), plus `FunctionForm → Bramble Studio`, `Robin Burden → Robin Vale`,
and `ai_kb → capture_kit`. Third-party tech (Firebase, Genkit) is neutral and kept.
~~No GitHub/issue/org/path/email refs were present in the kits.~~ (Struck — see the Correction
callout above: nine private-vault path wikilinks were present and have been scrubbed.)
**`kits/` is now inside the neutrality
lint's scan surface** (`scripts/check-neutrality.mjs` `SCAN_DIRS`), so future kit edits are held to
the same bar. (Note: the lint's identity denylist derives from the *profile*, which ships as
placeholders only — so the lint alone would not have caught "Henry"; the prose neutralization was
done by hand and adversarially verified, not left to the lint.)

**3. Drift guard.** `scripts/check-kits.mjs` (wired into `make check` + CI) fails if `kits.yaml`
and the `kits/` tree disagree in either direction — a missing/added file or an untracked/absent
kit dir, named explicitly. Red-able by removing a kit/file/manifest row or dropping in a stray.

**4. Doc-type schema reconciliation.** The repo's schema v1 was **profile-only** before this port
(`schema/profile.schema.yaml`); the vault's doc-type model was never carried over. This port adds
**`schema/doctypes.yaml`** — the product-neutral successor of the vault `types:`/`enums`/`status`/
`formats`/`universal` model — as schema v1's doc-type dimension. The seven kit types the desk's
index flagged as absent from the vault schema are **added** so every ported kit maps to an entry;
the `user-defined` kit's nonstandard types are **deliberately excluded**. The runtime validation
*engine* that consumes this contract is the PM build's job (#55 D2/D3), not this port.

**5. Composite kits keep their missing templates by design.** `postmortem` and `release-notes`
ship guide+example only — they are assembled from other kits / the project record, not authored
from a blank template. `kits.yaml` marks them `composite: true`.

## Schema-gap dispositions

Every kit type reconciled against schema v1's doc-type model. **None dropped silently.**

| Kit type | Vault-schema status | Disposition | Reason |
|---|---|---|---|
| analysis | absent from `types:` | **ADDED** (`status: reference`, req `author`, opt `conclusion`) | Kit existed; family/fields derived from its template frontmatter |
| meeting-notes | absent | **ADDED** (`reference`, req `meeting_date, attendees`) | Same — new type + assigned family (refinement) |
| raid | absent | **ADDED** (`reference`, req `author, owner`) | Living register; reference family |
| release-notes | absent | **ADDED** (`cadence`, req `version`) | Composite kit emits `type: release-notes`; published/draft = cadence |
| roadmap | absent | **ADDED** (`reference`, req `author, owner`) | New type + assigned family |
| runbook | absent | **ADDED** (`reference`, req `author, owner`) | New type + assigned family |
| ux-spec | absent | **ADDED** (`spec`, req `author, parent_product_spec`) | Screen-level spec; spec family |
| daily-note (→ `journal`) | emits `journal` (in schema) | **carried** (aliased) | Kit name ≠ emitted type; `journal` already a lightweight type |
| postmortem (→ `retro`) | emits `retro` (in schema) | **carried** (aliased, composite) | Emits a `retro` with `variant: incident`; `retro` already present |
| user-defined `meeting` | absent | **EXCLUDED** (kit-only) | Uses status `scheduled` (outside all families) — nonstandard by design |
| user-defined `note` | absent | **EXCLUDED** (kit-only) | Uses `category` field, freeform; nonstandard escape hatch |
| user-defined `task` variant | `task` is lightweight | **EXCLUDED from canonicalizing its extras** | Adds `priority: p2` (lowercase — violates P0–P3 enum), `status: backlog` (outside families); `task` stays lightweight in canon |
| user-defined `decision` variant | `decision` is canonical | **carried by the canonical `decision` type** | The stub is a project-flavored variant; canon `decision` governs |
| project | present in vault `types:` (no kit) | **carried verbatim** | Kitless type, but carried so its `project:` status family is not left orphaned; the PM build uses a project concept |
| all other carried types | present in vault `types:` | **carried verbatim** | Product-neutral already; no change |

## Consequences

- `schema/doctypes.yaml` is the source of truth for the kit frontmatter contract; the vault
  `schema.yaml` `types:` block is frozen. The PM build consumes `doctypes.yaml`; refinements there
  (families for the 7 added types) are the flagged, non-silent evolution the owner asked for.
- `kits.yaml` + `check-kits.mjs` make the kit tree self-guarding. `make check` now runs the guard.
- The neutrality lint scans `kits/`, so the identity-neutral property is enforced going forward.
- Deferred to the PM build (#55): the validation engine, per-desk gate rules, and any
  DB-backed field-shape refinements beyond the family assignments recorded here.

## Alternatives considered

- **Kits under `plugin/`** — rejected: `schema/` sets the precedent that shared, cross-lane,
  product-neutral surfaces sit at the repo root, not inside one lane. ~~Kits are consumed by the
  librarian/PM module too, not just the plugin.~~ (struck — see the **2026-07-21 Correction**
  above: kits are *designed* as a cross-lane surface but no lane consumes them at runtime yet;
  the placement rationale — a shared surface belongs at the root, peer to `schema/` — stands
  regardless.)
- **Manifest as JSON (`kits.json`)** — rejected: 0013 S4(a) names `kits.yaml`; a flat YAML subset
  is parseable dependency-free by the guard (the same posture as the other `scripts/` guards).
- **Add the `user-defined` types to canon** — rejected: they intentionally diverge from the schema
  (out-of-family statuses, lowercase priority); canonicalizing them would fork the enum semantics.

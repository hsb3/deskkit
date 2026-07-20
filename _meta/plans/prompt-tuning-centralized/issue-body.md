> **Tracking:** #128, ADR 0015 (2026-07-20 design session). Record where a centralized prompt set
> is tuned once, and how that tuning propagates to the embed, the plugin bundle, and every desk's
> already-seeded store row — a v2-track design item, not a build.

## Problem

ADR 0015 (`docs/decisions/0015-prompt-governance.md:28-30`) records an owner requirement,
verbatim intent: prompt **tuning** must become possible in a **centralized** fashion — one
canonical prompt set tuned in one place, never divergent per-project/per-store prompt versions.
This is distinct from (and depends on) the git-is-truth *governance* ruling the sibling
`prompt-governance` issue implements: that issue settles which store wins and what "reset to
shipped" means per surface; this issue is about **tuning propagation** — how an owner-authored
edit to the canonical wording reaches every place it is instantiated.

**Today's reality has no "prompt set" abstraction at all**, grounded directly in source:

- The librarian's instructions live in exactly one file,
  `librarian/templates/librarian-system-prompt.txt`, `//go:embed`'d into the binary
  (`librarian/templates/templates.go:20-21`). Tuning it means editing that file and cutting a new
  `deskkit` release — propagation is "ship a new binary."
- The PM instructions live in `plugin/desk-pm/agents/pm-operator.md` plus its skills
  (`plugin/desk-pm/skills/*/SKILL.md`) — version-controlled markdown with **no** DB-backed row
  (no `pm.system` prompt row or seed exists in shipped code — a grep across `plugin/` +
  `librarian/` returns zero matches; the only mentions are design docs discussing its absence).
  Tuning it means editing that markdown and republishing the plugin marketplace bundle — a
  **second, independent** propagation path.
- These two paths do not share a mechanism, a format, or a release cadence. There is no single
  place an owner edits once and both surfaces (let alone a future third — the ADR 0014
  desk-persona bundle) pick up the change together.

**A sharper, independently re-derived gap: propagation to an *existing* desk is not "manual today,
automatic tomorrow" — it is currently silent.** `prompt.Seed`
(`librarian/internal/modules/librarian/prompt/prompt.go:24-40`) only checks whether a
`librarian.system` row **exists** (`prompt.go:26-28`); it never compares the existing row's
content or a version marker against the currently-embedded default. Concretely: a desk that has
been running for a while already has a seeded `prompts` row from whatever binary version first
created it. Upgrading `deskkit` to a binary with a *tuned* embedded prompt does **not** update
that row — `Seed` sees a row already present and no-ops, every time
(`main.go:251-253`, `main.go:648-655`, both call `Seed` unconditionally on every run). The only way
a tuning edit reaches an already-provisioned desk today is the manual reset-to-shipped affordance
the sibling `prompt-governance` issue documents (delete the row, let it re-seed) — there is no
automatic "the shipped default changed, adopt it" path. Whether that should change, and how, is
squarely this design's question, not an assumption to make silently.

This design is a **v2-track item** (`_meta/plans/epic-schema-v2-track/issue-body.md:11`,
`:22`): it is not part of the 1.0.0 promise, carries no milestone, and its deliverable is a
**recorded design**, not code.

## Deliverables

- A design document (an ADR under `docs/decisions/` — the repo's append-only, reviewed-ruling
  convention already used for direction-setting items like ADR 0018's element-model direction — or
  a design doc elsewhere under `docs/` if the maintainer prefers pre-ADR staging; the choice of
  file is the builder's call, not fixed here) that:
  - Restates the ADR 0015 owner requirement (one canonical prompt set, tuned in one place, never
    per-project/per-store divergence) as the design's stated goal.
  - Names the current propagation reality above (embed-and-release for the librarian lane,
    markdown-and-republish for the PM lane, no update path for an already-seeded desk) as the
    baseline the design must improve on.
  - Answers **where the canonical tuned set lives.** Candidate mechanisms worth naming (not
    choosing): a shared prompt-set directory that both the Go embed and the plugin markdown are
    generated/copied from, mirroring the existing `schema/` -> `plugin/claude-plugin/schema/`
    copy-and-drift-guard pattern (`plugin/package.json:17`, `.github/workflows/ci.yml:81-85`); or
    keeping today's two per-lane sources but adding a shared manifest/registry that ties tuning
    edits together. Left **open** with a stated default.
  - Answers **how a tuning edit propagates** to: the Go embed (today: rebuild + release), the
    plugin markdown bundle (today: `make package` + republish), and a per-desk store's already-
    seeded `prompts` row (today: never, automatically — see Problem). The design must state
    whether upgrading the binary should re-seed on a version/content mismatch, or whether reaching
    an existing desk stays an explicit, operator-driven reset (the `prompt-governance` issue's
    reset-to-shipped affordance). Left **open** with a stated default.
  - States its relationship to ADR 0015's git-is-truth ruling and the re-seed path: this design
    must not reopen DB-as-truth (that only re-opens if the ADR 0009 trust gate flips, per ADR
    0015) while still explaining how centrally-tuned wording reaches a desk that predates the
    tuning.
  - Names what "tuned" means across desks: whether a tuning release is an atomic version all
    desks eventually converge on after upgrading, or whether a desk can pin/opt out of a given
    prompt version. Left **open** with a stated default.
  - Notes `schema/profile.schema.yaml`'s existing `preferences` (`schema/profile.schema.yaml:185`)
    and `custom` (`:197`, and the "anything not yet a first-class field belongs under `custom`"
    comment at `:7`) fields as a candidate home for a future tuning-related schema block, without
    committing to using them.
  - Names its own build slices (a bulleted breakdown of the eventual implementation work), so a
    later builder can pick a slice cold, matching this repo's plan-folder convention
    (`_meta/plans/README.md`).

## Acceptance criteria

- [ ] A design document exists under `docs/` that restates the ADR 0015 owner requirement,
      enumerates today's two independent propagation paths plus the silent-on-upgrade gap for an
      already-seeded desk, and states where the canonical tuned set will live.
- [ ] The document states how a tuning edit reaches the Go embed, the plugin markdown bundle, and
      an existing desk's seeded store row, and how that interacts with ADR 0015's git-is-truth
      ruling and reset-to-shipped path.
- [ ] Every genuinely open question above (canonical-set location; per-desk propagation on
      upgrade; opt-out/pinning) is recorded as **open** with a stated default — none is silently
      decided.
- [ ] The document names its own build slices.
- [ ] The document is reviewed before being marked accepted (if an ADR: `Accepted` status per this
      repo's ADR lifecycle; if a design doc: an explicit review pass) — not merged as a first
      draft with no second look, matching this repo's adversarial-review practice for design
      items.
- [ ] Closing this issue requires only the recorded, reviewed design — no code, schema, or script
      change ships from this issue.

## Dependencies & gates

- Docs-only change: `make check`, `make test`, and `make verify` do **not** fire — no code,
  `schema/`, `kits/`, or plugin-bundle surface is touched.
- Identity neutrality, version sync, kits drift, bundle drift guard — do **not** fire, for the
  same reason.
- CHANGELOG — does **not** fire. A recorded design is not a shipped product change, consistent
  with how other design-only ADRs in this batch (e.g. ADR 0018's element-model direction) were not
  individually CHANGELOG'd.
- If the design lands as a new ADR: `docs/decisions/README.md`'s index table gains a row, per the
  append-only ADR convention (`docs/decisions/README.md:1-9`) — an in-scope edit for whichever PR
  lands the ADR.
- Depends on nothing new to *start*; the sibling `prompt-governance` issue should land first in
  practice, since this design assumes and extends its git-is-truth and reset-to-shipped
  vocabulary — cite it once merged, but this is a sequencing preference, not a hard blocker.
- The eventual **build** that implements whichever mechanism this design names inherits its own
  gates (a new schema field, a new drift guard, embed/bundle regeneration, etc.) — not enumerable
  yet, since the mechanism is exactly what is undecided here.

## Out of scope

- Implementing the mechanism (schema changes, propagation code, new drift guards). That is the
  eventual build, tracked separately once the design is accepted, via the slices the design
  document itself names.
- Re-litigating or reopening ADR 0015's git-is-truth ruling, or DB-as-truth — this design works
  within that ruling; it only re-opens if the ADR 0009 trust gate flips.
- The ADR 0014 desk-persona bundle's own build (tracked separately as `desk-persona-bundle`) — the
  design should name it as a future prompt surface the centralized mechanism must eventually
  cover, without building for it here.
- The sibling v2-track item `trigger-design` (ADR 0018 Q4, triggers for exec outputs) — an
  unrelated design item in the same `epic-schema-v2-track` epic, not addressed here.
- The `prompt-governance` issue's own deliverables (Seed-semantics documentation, the prompt-copy
  drift guard, reset-to-shipped documentation) — this issue depends on that vocabulary but does
  not redo it.

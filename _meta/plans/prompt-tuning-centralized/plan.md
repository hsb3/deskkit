---
title: "Centralized prompt tuning -- design plan"
type: spec
status: planned
created: 2026-07-23
purpose: "Ground and schedule the v2-track design ADR 0015 requires: where a centralized, once-tuned prompt set would live, and how a tuning edit propagates to the librarian go:embed, the desk-persona plugin markdown, and an already-seeded desk's prompts row, without breaking the existing prompt-copy drift guarantees."
notes: "Issue #128 (open, unscheduled backlog, v2-track, epic #130). ADR 0015 owner requirement (2026-07-20 design session). This issue's own deliverable is a recorded design (ADR or design doc), not code -- this plan schedules and grounds that authoring work, it does not author the design itself."
---

# Plan - centralized prompt tuning design

_Plan for producing the v2-track design ADR 0015 requires: where a centralized, once-tuned prompt
set would live, and how a tuning edit propagates to the librarian's `go:embed`, the desk-persona
plugin markdown, and an already-provisioned desk's seeded `prompts` row, without reopening
DB-as-truth or breaking the prompt-copy drift guards already shipped. RESIDUAL: none -- nothing
toward this design has shipped; this issue's own deliverable is the design record itself, not the
mechanism it will eventually name (implementing the mechanism is explicitly out of scope of #128)._

Status: draft (2026-07-23)

## Tracking

- Issue: **#128** "feat: centralized prompt tuning design" -- open, label `enhancement`,
  unscheduled backlog, no milestone (`https://github.com/hsb3/desk-standard/issues/128`).
- Epic: **#130** `epic-schema-v2-track` -- #128 is a listed child
  (`_meta/plans/epic-schema-v2-track/issue-body.md:22`: "#128 `prompt-tuning-centralized` -- ADR
  0015 owner requirement: one canonical prompt set tuned in one place"). The epic's Close-when
  clause requires "the centralized-tuning design are recorded (ADR or reviewed design ...)"
  (`_meta/plans/epic-schema-v2-track/issue-body.md:30`) before #130 can close.
- ADR: `docs/decisions/0015-prompt-governance.md:28-30` -- the owner requirement this issue exists
  to satisfy ("prompt *tuning* must become possible in a **centralized** fashion -- one canonical
  prompt set tuned in one place, never divergent per-project/per-store prompt versions. The tuning
  mechanism is a v2-track design item.") and `:20-27` -- the git-is-truth ruling this design must
  work within without reopening.
- Also binds: `docs/decisions/0014-agent-integration-contract.md:37-38` -- contract parameter "(d)
  Instructions: the contract names exactly one instruction source per surface; the sourcing/sync
  mechanism is ADR 0015's." This design is that mechanism.
- Sequencing precedent: sibling issue **#120** "feat: prompt governance -- git-is-truth semantics +
  a prompt-copy drift guard" is **CLOSED** (archived at `_meta/_archive/120-prompt-governance/`); it
  shipped the git-is-truth vocabulary, the reset-to-shipped affordance, and
  `scripts/check-prompt-drift.mjs`, all of which this design depends on and extends per the staged
  issue body's own Dependencies section (`_meta/plans/prompt-tuning-centralized/issue-body.md:118-120`).
  #120 being closed removes the one sequencing precondition the issue body names as "should land
  first in practice" -- it already has.
- Contract impact: **none directly** -- the deliverable is docs-only (a design record). The design's
  *output*, once accepted, will define the contract impact of a later, separately-tracked build (a
  candidate schema field, a new drift guard, embed/bundle regeneration semantics) -- not enumerable
  yet, per the issue body's own Dependencies & gates section
  (`_meta/plans/prompt-tuning-centralized/issue-body.md:121-123`).

## The problem (grounded in source) -- the verified prompt-copy inventory today

There is no "prompt set" abstraction today. Three independent surfaces carry agent instructions,
propagated by three different mechanisms, with one silent gap. Verified directly against source
(not against the issue body's citations, several of which have drifted -- see Contradictions below):

| # | Surface | Canonical source | Version-controlled copies | Guard | Propagation today |
| - | ------- | ----------------- | -------------------------- | ----- | ------------------ |
| 1 | Librarian system prompt | `librarian/templates/librarian-system-prompt.txt` (30 lines), `//go:embed`'d at `librarian/templates/templates.go:20-21` (`var SystemPrompt string`) | (a) spec quote: `docs/pocket-librarian-v1-spec.md`, fenced `` ```text `` block under the sentinel "The full embedded-default text (the first-run seed, kept verbatim):" at line 1517, fence lines 1520-1549; (b) generated persona body: `plugins/desk-persona/agents/librarian-operator.md`, whose body is the same prompt text embedded verbatim under a `<!-- GENERATED -->` header | (a) `scripts/check-prompt-drift.mjs` -- byte-identical assert between the embed and the spec's "kept verbatim" fence; wired into `make check` (`Makefile:45`) AND its own CI step (`.github/workflows/ci.yml:64-65`, "Prompt-copy drift guard -- embed vs spec quote"). (b) `scripts/check-persona-drift.mjs` -- regenerate-and-compare of the derived agent file (`scripts/check-persona-drift.mjs:90-95` DERIVED manifest); wired into `make check` (`Makefile:49`) but **not** its own CI step (no `check-persona-drift` reference anywhere in `.github/workflows/ci.yml` -- verified by grep; it rides `make check`/lefthook locally, not the CI job list at `.github/workflows/ci.yml:46-176`) | Edit the embed (git-truth per ADR 0015), then `node scripts/check-persona-drift.mjs --write` to regenerate the derived agent file, then a new `deskkit` release for the embed itself. The spec quote must be hand-kept byte-identical (no `--write` mode for it). |
| 2 | PM instructions | `plugins/desk-persona/agents/pm-operator.md` (81 lines) + 3 skills (`plugins/desk-persona/skills/{pm-triage,pm-session-open,pm-advance-item}/SKILL.md`, 94/87/105 lines) | **none** -- these files are the *one authored source* per surface, not a copy of anything (`scripts/check-persona-drift.mjs:14-21`: when desk-pm folded into desk-persona, ADR 0014(a), these files "stopped being copies of an upstream desk-pm source and became the ONE authored PM source per surface themselves"). Confirmed no DB row backs them: `grep -rn "pm.system\|pm\\.system" plugin librarian` returns zero matches (only unrelated `pm-system spec` doc-title hits) | none exists, and per the comment above none is needed today ("with no second copy to diverge from, a copy/compare guard for them would guard nothing") | Edit the markdown in git, then a plugin-marketplace republish (a version bump of the `desk-persona` bundle referenced from `.claude-plugin/marketplace.json`) -- no build step, no `make package` involvement (`make package` only regenerates `plugins/desk-standard/` per `plugin/package.json:17`, a different bundle). |
| 3 | Per-desk seeded store row | PocketBase `prompts` collection, row `key = 'librarian.system'`, created by `prompt.Seed` (`librarian/internal/modules/librarian/prompt/prompt.go:38-54`) | n/a (runtime data, not version-controlled; ADR 0015 rules it a **re-seeded cache**, not canonical) | none needed for content (by design -- it is explicitly not the source of truth), but **`Seed` never re-checks an existing row against the current embed**: the only branch is an existence check (`prompt.go:39-42`, `FindFirstRecordByFilter(...); err == nil { return nil }`) -- no content or version comparison against `templates.SystemPrompt` | `Seed` runs unconditionally on every `serve` start (`librarian/cmd/deskkit/main.go:264`) and on every one-shot CLI command (`librarian/cmd/deskkit/main.go:734`, added specifically so a CLI/MCP-only desk still gets a seeded row). Because it only checks existence, an already-seeded desk's row **never** picks up a tuning edit to the embed after an upgrade -- the only way today is the manual reset-to-shipped affordance (delete the row; the next run re-seeds byte-for-byte). |

**The silent-on-upgrade gap is real and independently reproduced from source**, not just asserted by
the issue body: `main.go:264` and `main.go:734` both call `prompt.Seed` unconditionally on every
run, and `Seed`'s only gate is the existence check at `prompt.go:39-42` -- there is no code path
anywhere in `librarian/internal/modules/librarian/prompt/prompt.go` that compares row content or a
version marker to `templates.SystemPrompt` once a row exists.

A candidate future home for tuning-related config, noted without commitment (matching the issue
body's own framing): `schema/profile.schema.yaml`'s `preferences` block (line 193, "Deployment-wide
preferences prompts may reference") and `custom` escape hatch (line 205), governed by the "anything
not yet a first-class field belongs under `custom`" convention stated at line 15.

**Contradictions found in the staged issue body (not fixed -- `issue-body.md` is out of scope for
this plan; flagged here per the verification brief):**

- `_meta/plans/prompt-tuning-centralized/issue-body.md:21-22` cites `plugin/desk-pm/agents/pm-operator.md`
  and `plugin/desk-pm/skills/*/SKILL.md`. That path no longer exists -- `plugin/desk-pm/` was folded
  into `desk-persona` in #180, then moved under `plugins/` in #191 (`_meta/HANDOFF.md:61-63`,
  documenting the identical staleness already flagged for a *different* doc under #199). The current,
  correct paths are `plugins/desk-persona/agents/pm-operator.md` and
  `plugins/desk-persona/skills/*/SKILL.md`, used throughout this plan.
- `_meta/plans/prompt-tuning-centralized/issue-body.md:63-64` cites `plugin/package.json:17` (this
  one checks out exactly) but several other line-anchored citations in the issue body (e.g.
  `prompt.go:24-40` for `Seed`, `main.go:251-253`/`:648-655` for the call sites, `.github/workflows/ci.yml:81-85`
  for the packaged-artifacts drift guard, `schema/profile.schema.yaml:185`/`:197`/`:7` for
  preferences/custom/the comment) have drifted from current line numbers as the files grew; this
  plan re-verified and cites the current lines (`prompt.go:38-54`, `main.go:264`/`:734`,
  `.github/workflows/ci.yml:143-147`, `schema/profile.schema.yaml:193`/`:205`/`:15`). None of these
  drifts changes the underlying claims -- they are stale line anchors, not wrong facts.

## Design questions the eventual design document must answer

These map directly to the issue's own Deliverables section
(`_meta/plans/prompt-tuning-centralized/issue-body.md:60-85`) and its Acceptance criteria
(`:87-104`, third bullet: "Every genuinely open question above ... is recorded as **open** with a
stated default -- none is silently decided"). The defaults below are this plan's grounded
**recommendation for the design author to weigh**, not a decision -- the issue's acceptance
criteria requires the design document itself record each as OPEN with its own stated default, even
if that default matches the one recommended here.

1. **Where does the canonical tuned set live?** Candidates the issue body names
   (`issue-body.md:63-65`): (i) a shared prompt-set directory both the Go embed and the plugin
   markdown are generated/copied from, mirroring the existing `schema/` -> `plugins/desk-standard/schema/`
   copy-and-drift-guard pattern (`plugin/package.json:17`, `.github/workflows/ci.yml:143-147`); or
   (ii) keep today's two per-lane sources, add a shared manifest/registry that ties tuning edits
   together without moving the files. **Recommended default: (i), a shared directory** (e.g.
   `prompts/` at repo root or under `schema/`) that both `librarian/templates/librarian-system-prompt.txt`
   and `plugins/desk-persona/agents/pm-operator.md` + its skills are copied/generated from. Rationale:
   the repo already has exactly one governed pattern for "one authored source, N shipped copies"
   (copy-and-drift-guard), used for `schema/` and now for the librarian-operator persona body
   (`scripts/check-persona-drift.mjs`); a second, differently-shaped mechanism (a registry that
   points AT files without moving them) adds a second pattern to maintain for no proven benefit, and
   the PM lane's markdown is not currently a generated file at all (see inventory row 2) -- making it
   one is the natural way to give it the same guardable-drift property the librarian lane already has.
2. **How does a tuning edit reach the Go embed?** Today: hand-edit the embed, cut a `deskkit`
   release. **Recommended default: unchanged** -- the embed must stay a compiled-in Go string
   (`//go:embed`, `templates.go:20-21`); no mechanism can make a compiled binary pick up a file
   change without a rebuild. The centralization only changes *where the source-of-truth text is
   authored* (a shared prompt-set directory, question 1), not that the librarian binary still needs
   a rebuild+release to carry a new value.
3. **How does a tuning edit reach the plugin markdown bundle?** Today: hand-edit
   `plugins/desk-persona/agents/pm-operator.md` + skills directly (they are the one authored source,
   inventory row 2), then a marketplace republish. **Recommended default: if question 1 resolves to
   a shared directory, these become generated files** (like `librarian-operator.md` already is via
   `scripts/check-persona-drift.mjs`) with `make check`/`--write` extended to cover them, so a tuning
   edit to the shared source and a regenerate step are the propagation mechanism, and republishing
   stays a separate, unavoidable step (the marketplace has no push-update mechanism today; a
   plugin-marketplace release IS how any bundle content change reaches an installed desk, prompt or
   not -- this is a platform constraint, not something the design can change).
4. **How does a tuning edit reach an already-seeded desk's `prompts` row?** This is the sharpest
   open question -- the issue body is explicit it must not be silently decided
   (`issue-body.md:66-71`, `:96-97`). Two shapes: (i) `Seed` gains a content/version comparison
   against the embed and re-seeds (or seeds a new version + flips `active`) on mismatch, making an
   upgrade self-propagating; or (ii) reaching an existing desk stays an explicit, operator-driven
   reset (the `prompt-governance` issue's already-shipped reset-to-shipped affordance -- delete the
   row, next run re-seeds). **Recommended default: (ii), stay explicit-reset for now** -- ADR 0015's
   git-is-truth ruling already treats a DB row as a re-seeded cache whose divergence from git is
   expected and resolved by an explicit reset; auto-re-seeding on version mismatch would silently
   discard *any* GUI/REST edit a desk operator made (even a since-abandoned one), which is exactly
   the kind of surprise ADR 0015's Consequences section already flags as "documented, not an
   accident" for the *reset* path but never sanctions for an *automatic* path. An auto-reseed also
   needs new machinery `Seed` does not have (a version marker on the embed itself to compare
   against) that does not exist yet. This default is the more conservative of the two and should be
   revisited if operator-driven reset proves too manual in practice once the mechanism ships.
5. **What does "tuned" mean across desks -- atomic version or pin/opt-out?** Whether a tuning
   release is a single version every desk eventually converges on after upgrading its binary/bundle,
   or a desk can pin to / opt out of a given prompt version. **Recommended default: atomic version,
   no pinning at v1 of this mechanism** -- ADR 0015's stated goal is explicitly "never divergent
   per-project/per-store prompt versions" (`0015-prompt-governance.md:29`); per-desk pinning
   reintroduces exactly the divergence the owner requirement rules out. If a real need for pinning
   surfaces later, it is a strict superset of the atomic-version mechanism (a version field plus an
   opt-out flag), so choosing "no pinning" now does not foreclose adding it.
6. **Relationship to ADR 0015's git-is-truth ruling and the reset-to-shipped path.** The design must
   state explicitly that it does not reopen DB-as-truth (only the ADR 0009 trust gate flip does
   that, per `0015-prompt-governance.md:31`) while still explaining how centrally-tuned wording
   reaches a desk that predates the tuning (this is fully answered by question 4's default: the
   reset-to-shipped affordance is unchanged and remains the propagation path to already-provisioned
   desks). **Recommended default: state it as a corollary of question 4's default**, not a
   separately-decided point -- there is no daylight between "how does tuning reach an existing desk"
   and "does this reopen DB-as-truth" once question 4 is answered conservatively.

## Deliverable shape

**Recommended default: a new numbered ADR under `docs/decisions/`** (next slot `0022`, per
`docs/decisions/README.md:32` -- `0021` is the current highest), not a standalone design doc
elsewhere under `docs/`. Rationale, grounded in this repo's own precedent: ADR 0018 ("Element-model
direction") is cited by the issue body itself (`issue-body.md:52-53`) as the direction-setting
precedent this design should follow, and ADR 0015 -- the ruling this design directly extends --
is itself an ADR; keeping the extension in the same append-only, `Accepted`-gated series as the
ruling it extends keeps one citable lineage (`0015` -> `0022`) instead of splitting the prompt-
governance story across an ADR and a separately-tracked design doc. The issue body leaves the file
choice explicitly to "the builder's call, not fixed here" (`issue-body.md:53-54`); a standalone
design doc remains open if the maintainer prefers pre-ADR staging (e.g. because the mechanism is
expected to change materially before a build starts), consistent with the issue's own phrasing.

If it lands as ADR 0022: `docs/decisions/README.md`'s index table gains a row (append-only
convention, `docs/decisions/README.md:6-8`) -- an in-scope edit for whichever PR lands the ADR, per
the issue body's own Dependencies & gates section (`issue-body.md:115-117`).

## Deliverables (file scope + acceptance)

Single deliverable -- the issue's own Acceptance criteria (`issue-body.md:103-104`) states
"Closing this issue requires only the recorded, reviewed design -- no code, schema, or script
change ships from this issue," so this plan has exactly one ownable unit.

### A - the design record (ADR 0022 or a design doc; see Deliverable shape)

- **Scope:** one new file under `docs/decisions/` (or elsewhere under `docs/` if a design doc is
  chosen) plus, if an ADR, a one-row addition to `docs/decisions/README.md`'s index table. No other
  file changes.
- **Content, each independently checkable against the issue's acceptance criteria
  (`issue-body.md:87-104`):**
  1. Restates the ADR 0015 owner requirement (`0015-prompt-governance.md:28-30`) as the design's
     stated goal.
  2. Names the current propagation reality (this plan's Problem section inventory, rows 1-3) as the
     baseline the design improves on, including the silent-on-upgrade gap.
  3. Answers design questions 1-5 above, each recorded **open** with a stated default (this plan's
     recommendations are a starting point, not a substitute for the design author's own reasoning).
  4. States its relationship to ADR 0015's git-is-truth ruling (design question 6).
  5. Names its own build slices -- a bulleted breakdown of the eventual implementation work (e.g.:
     stand up the shared prompt-set directory; convert the PM markdown to generated files + extend
     `check-persona-drift.mjs`; add the embed-version marker and/or the `Seed` comparison logic if
     question 4 resolves toward auto-detection later; update `docs/pocket-librarian-v1-spec.md`
     and/or a new agent-integration-contract-adjacent doc), matching `_meta/plans/README.md`'s
     plan-folder convention (a later plan.md can be authored per slice, cold, once this design is
     accepted).
- **Acceptance (independently verifiable):**
  - `grep -n "ADR 0015" <design-file>` (or equivalent restatement) returns a hit that names the
    centralized-tuning requirement.
  - The design-questions section above (1-5) is fully covered: `grep -c` for however many numbered
    open questions the design records is >= 5, each with a locatable "default:" or equivalent
    phrase, and none reads as a silently-closed decision (a human/adversarial review check, not a
    grep -- see the review-pass criterion below).
  - `docs/decisions/README.md`'s index table contains a row for the new ADR number, if the ADR path
    is chosen (`grep -n "^| \[0022\]" docs/decisions/README.md` returns one line).
  - The document carries a build-slices bullet list (non-empty).
  - The document is reviewed before being marked `Accepted` (if an ADR) or given an explicit review
    pass (if a design doc) -- matching this repo's adversarial-review practice for design items
    (issue body's own acceptance bullet, `issue-body.md:99-102`). Not a grep-checkable criterion;
    requires a second reader, same as any ADR going `Proposed` -> `Accepted`.
  - `make check`, `make test`, and `make verify` all still pass unmodified after the design file
    lands (docs-only change; see Gate and contract hygiene below) -- run bare, never piped, per this
    repo's gate-running convention (`CLAUDE.md` "Commands" section).

## Gate and contract hygiene

### This plan's own landing (docs-only, per `_meta/plans/_config.md`'s gate menu)

| Gate | Fires? | Why |
| ---- | ------ | --- |
| `make check` (neutrality + self-test, kit drift, prompt drift, tool-surface drift, scaffold frontmatter, persona drift, textfield-max, query-kind drift, core purity, shellcheck, actionlint, workflow-pin drift, profile-root drift) | no | a new `docs/decisions/*.md` file (or design doc) touches no file any of these guards scan: `docs/` is exempt from neutrality (`CLAUDE.md` "identity-neutrality" section: "docs/ and repo-root files are EXEMPT"); prompt-drift and persona-drift only compare the two files named in this plan's inventory rows 1(a)/1(b), neither of which this deliverable edits; the rest scan `kits/`, `schema/`, `librarian/` collections, `plugin/`, or workflow files, none touched. |
| `make test` (plugin `bun test` + librarian `go test ./...`) | no | no code changed |
| `make verify` (librarian integration, scratch desk) | no | nothing under `librarian/` touched |
| Bundle drift guard (`make package` + `git diff --exit-code`) | no | `plugin/core`, `plugin/mcp`, `schema/` untouched |
| Version sync / kits drift / CHANGELOG | no | no manifest, no `kits/`, and CHANGELOG explicitly does not fire for a design-only record, consistent with how ADR 0018's element-model direction was not individually CHANGELOG'd (issue body's own framing, `issue-body.md:112-114`) |
| `docs/decisions/README.md` index-table convention | **manual, in-scope** | append-only ADR index; the PR that lands the ADR adds the row by hand (no automated guard exists for this table today) |

### Gates a later, separately-tracked BUILD of the design's chosen mechanism would rewire

Not this issue's scope to implement (`issue-body.md:127-129` rules the mechanism itself out of
scope) -- named here so a future builder inherits this plan's gate analysis instead of
rediscovering it, per the issue's own "not enumerable yet" framing (`issue-body.md:121-123`):

- **`scripts/check-prompt-drift.mjs`** currently hard-codes exactly one embed path and one spec-quote
  sentinel (`check-prompt-drift.mjs:28-29,34`). If question 1 resolves to a shared prompt-set
  directory, this guard's `EMBED` constant becomes "shared source -> generated embed" instead of
  "embed -> spec quote," or a new upstream step (shared source -> embed AND -> spec quote) is
  inserted before it. Either way the guard's invariant (embed and spec quote byte-identical) must
  survive the transition unbroken -- the guard's own comment already anticipates this ("Forward
  note: when the ADR 0014 desk-persona bundle lands ... its bundle-markdown prompt copies join
  prompt governance -- either by extending this file's copy list or via the bundle's own
  regenerate+diff guard," `check-prompt-drift.mjs:18-21` -- written for a bundle that has since
  landed and could be reused verbatim for a shared prompt-set source).
- **`scripts/check-persona-drift.mjs`** currently derives `librarian-operator.md` FROM the embed
  (`buildLibrarianAgent` reads `LIBRARIAN_PROMPT`, `check-persona-drift.mjs:58-64`). If the embed
  itself becomes a generated copy of a shared source (question 1), this script's `DERIVED` chain
  gains a link (shared source -> embed -> persona body) rather than changing shape; the existing
  regenerate-and-compare pattern (`--write` / bare-compare) already tolerates chained derivation.
  If question 3's default lands (PM markdown becomes generated too), this script's `DERIVED` array
  (`check-persona-drift.mjs:90-95`) gains entries for `pm-operator.md` and the 3 skill files -- a
  pure extension, no restructuring, since the manifest is already a list of independent
  `{target, build}` pairs.
- **A new guard is needed, not an extension of an existing one**, if question 4 resolves toward
  auto-detection (the non-default option): comparing a live `prompts` row's content/version against
  the current embed at runtime is a Go-side concern inside
  `librarian/internal/modules/librarian/prompt/prompt.go`, not a `scripts/*.mjs` static-file guard
  -- the existing drift-guard pattern (regenerate + diff two on-disk files) does not apply to
  comparing a compiled-in string against runtime DB state at binary startup.
- **Keeping gates green through the transition**: because the file-move (question 1) and the
  generated-ness change (question 3) are the parts of the eventual mechanism that touch existing
  guards, the safe order for that later build is: (i) stand up the shared source with the OLD guards
  still pointed at the OLD locations (no drift, both sets of files still exist and agree); (ii) flip
  each guard's source pointer one at a time, confirming `make check` stays green after each flip;
  (iii) only then delete the old per-lane authored copies. Never flip a guard's target and delete the
  old file in the same change -- that is exactly the two-step-in-one-commit pattern that let the
  original stale-prompt bug (ADR 0015's own motivating incident) ship ungoverned.

## Parallelism and landing order

A single-file docs deliverable has no internal parallelism to schedule. The relevant sequencing is
external:

- **No blocking dependency to start.** #120 (the prerequisite the issue body names) is already
  closed; nothing else is a hard blocker (issue body's own framing, `issue-body.md:118-120`:
  "Depends on nothing new to *start*").
- **Landing order relative to sibling v2-track items:** independent of `#125` `element-model-revision`,
  `#126` `model-simulations`, and `#127` `trigger-design` -- all are children of the same epic
  `#130` but touch disjoint files (schema element model vs. prompt governance) and carry no shared
  file scope with this deliverable. Any order among them is safe.
- **This plan's own landing:** one PR, one file (plus the README index-table row if ADR), reviewed
  once, no serialization needed.

## Dependencies and what would make this schedulable

Unscheduled today (no milestone, v2-track, `issue-body.md:45-47`). What moves it from backlog to
schedulable, without inventing a timeline:

- **A maintainer decision to prioritize the v2-track epic (#130) over other backlog**, since #128 is
  a child deliverable of that epic and carries no milestone on its own -- scheduling is an epic-level
  call, not something this plan can force.
- **An owner/maintainer confirming the Deliverable-shape default (ADR vs. design doc)** before
  drafting starts, so the drafting agent/human does not have to make that call mid-draft (this
  plan's open question 1 below).
- **No code prerequisite** -- the design can be drafted at any time; it reads current source (this
  plan's Problem section already re-verified it) and does not depend on any in-flight PR.
- Once scheduled, the actual authoring effort is small and single-owner (one document, reviewed
  once) -- it does not need a parallel wave or a worktree; a single builder session against this
  plan is sufficient.

## Open questions / owner decisions (plan-level)

Distinct from the design-questions section above (those are questions the design DOCUMENT must
answer and record as open); these are questions about how to RUN this plan:

1. **ADR vs. standalone design doc (Deliverable shape).** **Default: ADR 0022** (see Deliverable
   shape section above for the full rationale). Alternative kept open: a design doc under
   `docs/development/` if the maintainer wants pre-ADR staging room before the mechanism commits to
   the append-only ADR series.
2. **Who authors the design -- human maintainer or a dispatched agent working from this plan?**
   **Default: either** -- this plan is written to be picked up cold by a builder (human or agent);
   nothing in it requires the plan's own author to also write the design. If dispatched as an agent
   task, the review-before-`Accepted` acceptance criterion still requires a second reader/session,
   not the same session that drafted it.
3. **Should this plan's row be added to `_meta/plans/README.md`'s ACTIVE table now?** The
   planning-desk standard's step 7 calls for it (`references/plan.md:57`), but `_meta/plans/README.md`
   is outside this plan's owned file scope (owned scope is `_meta/plans/prompt-tuning-centralized/`
   only). **Flagged as a required follow-up for whoever owns `_meta/plans/README.md`** -- add a row:
   `prompt-tuning-centralized | #128 | planned; v2-track; epic #130; no milestone`.

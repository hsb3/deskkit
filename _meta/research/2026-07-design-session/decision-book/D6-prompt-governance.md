---
type: analysis
status: draft
created: 2026-07-20
updated: 2026-07-20
tags: [design-session, decision-book, prompt-governance, agent-symmetry, surfaces, 1.0.0]
synopsis: D6 — one source of truth (or a documented split) for agent instructions across the
  Go embed, the runtime-editable DB `prompts` collection, and the version-controlled plugin
  markdown. Options span git-is-truth, DB-is-truth, a documented split by prompt class, and
  documented status quo — each with its sync/regeneration and "reset to shipped" story.
  Depends on D5; rules the MECHANISM, not the per-surface contract. No ruling.
---

_Phase-1 decision brief (`../README.md` §2, book index `./README.md`). Informs the session; does
not rule. Every Evidence bullet resolves to a Phase-0 dossier section beside the prep doc; dossier
claims are hypotheses until re-derived where a ruling binds._

Status: draft (2026-07-20) — list signed off 2026-07-20 (D6 included); awaiting the ruling.

> **Platform-stream interaction (2026-07-20 reboot — see `D0-platform-frame.md`):** the
> option space is regime-dependent. Under files-are-truth (the standing regime per D0.b's
> hypothesis), Option A's "DB edits are ephemeral" is coherent and Option B needs the
> disk-backed home it lacks; under a future PB-as-truth flip, Option B becomes the aligned
> choice. Rule for the standing regime and state the gate-flip consequence. The persona
> bundle (D5.a / platform R1 decision 2) adds a fourth prompt surface (bundle markdown) to
> the three-store problem — whatever mechanism wins must cover it.

# D6 — Prompt governance / single-sourcing

**Layer:** surface · **Depends on:** D5 (the contract rules THAT each surface names one
instruction source; D6 rules the MECHANISM of single-sourcing).

## 2. The question

**One source of truth for agent instructions — or a documented split — across the three stores an
instruction can live in: the Go embed (`//go:embed` seed), the DB `prompts` collection (GUI/REST-
editable at runtime), and the plugin markdown (version-controlled).**

The librarian's system prompt is embedded in the binary, seeded into a store collection on first
run, and thereafter served from whichever the runtime resolver prefers; the PM lane's instructions
live only in plugin markdown. That is already a split — but an *accidental*, unmanaged one. The
decision is which of these stores is canonical, what "reset to shipped" means, and how (or whether)
runtime-editable DB prompts reconcile against a git source that fundamentally cannot drift-guard
them.

## 3. Why now

- The repo already owns the pattern for *generated artifacts with a git drift guard* (schema copied
  into the plugin bundle, CI fails on any diff). But a runtime-editable DB prompt cannot be
  drift-guarded against git the same way — a GUI edit is a legitimate divergence, not drift. So the
  question is genuinely a governance choice, not a mechanical "add a guard."
- Phase-0 surfaced the symptom of leaving it ungoverned: the librarian's own in-binary system prompt
  is **stale independent of PM** — it advertises tools the agent does not hold, and never mentions
  the PM tools it *does* receive once PM is enabled. Staleness recurs precisely because no store is
  the ruled source and nothing keeps the embed, the DB row, and the actual tool set in sync.
- Identity-neutrality makes this load-bearing, not cosmetic: a user's runtime prompt edit is
  personalization (like `_knowledge/profile.yaml`) and must **never** flow back into a shipped
  artifact. Without a ruling, the DB-editable path is one careless "export the prompt" away from
  baking a deployment identity into a distributed binary.
- The store rebuilds from disk. A runtime prompt edit that lives only in the store — not in the desk
  tree — is silently discarded on rebuild. Whether that is correct (ephemeral overlay) or a data-loss
  bug (durable personalization) is exactly what D6 must rule.

## 4. Evidence

- **agent-symmetry.md § Verdict table — the 7-row comparison table** — Instructions row, CONFIRMED:
  `prompt.Seed` writes `templates.SystemPrompt` to `prompts` key `librarian.system` on first run;
  `systemPrompt` reloads the active row per run (GUI/REST-editable). PM instructions live in markdown;
  **no `pm.system` row anywhere** (grep negative). (`librarian/…/librarian/prompt/prompt.go:16-40`;
  `agent.go:42-53`; `plugin/desk-pm/agents/pm-operator.md` + `skills/*`)
- **agent-symmetry.md § Verdict table — the 6 "accidental / unruled" asymmetries** — #4
  "Prompt-governance split," CONFIRMED and unchanged: two governance models, one per surface —
  librarian DB-backed/GUI-editable vs PM version-controlled markdown; **neither model chosen for
  both.** (`prompt.go:16-40` + `agent.go:42-53` (DB) vs `pm-operator.md` + `skills/*` (markdown))
- **agent-symmetry.md § Verdict table — the 6 "accidental / unruled" asymmetries** — #3 "In-binary
  loop gets PM tools without PM discipline": `systemPrompt` resolves only `librarian.system`; **no
  `pm.system` is ever seeded**, so the in-binary prompt is librarian-only even when the merged
  registry hands the loop 12 PM tools. (`pm/module.go:78-81` → `agent.go:107-117` →
  `toolcore.go:133-142`; prompt `agent.go:42-53`)
- **surface-matrix.md § 6. Unclaimed-surface findings (the ride-along problem)** — finding 2: the
  in-binary eino agent's prompt (`librarian/templates/librarian-system-prompt.txt:1-33`) is the
  librarian-only prompt (7 tools) and never mentions the PM tools it receives when PM is on — "the
  *model's own instructions* silently going stale the moment a second module is enabled … directly
  grounds C6."
- **surface-matrix.md § 6. Unclaimed-surface findings (the ride-along problem)** — finding 3: **even
  with PM off**, the same prompt (`librarian-system-prompt.txt:5-14`) unconditionally lists
  `apply_fix` and `restore`; `restore` is never given to the agent (`toolcore.go:144-150`) and
  `apply_fix` is absent unless `LIBRARIAN_AUTONOMOUS_WRITES=true` (default off) — so a default desk's
  prompt claims 2 tools it does not hold.
- **surface-matrix.md § 7. Persona inventory** — the in-binary prompt
  (`librarian/templates/librarian-system-prompt.txt`) drives BOTH the chat TUI (S3) and the `agent`
  one-shot (S4); the PM persona (`plugin/desk-pm/agents/pm-operator.md`, frontmatter `tools:` lines
  13-26) is a separate, markdown-only instruction body. One canonical prompt already fans out to two
  surfaces on the librarian side.
- **surface-matrix.md § 5. Gating column — what flips a cell** — `LIBRARIAN_AUTONOMOUS_WRITES`
  default `false` (`config.go:112`), consumed by `toolcore.AgentTools` (`toolcore.go:133-142`) — the
  reason the embedded prompt's `apply_fix`/`restore` lines are wrong on a default desk (finding 3).
- **data-model.md § 1.9 `prompts` (0009, widened by 0011)** — the DB store of record: `content`
  widened to 50,000,000 chars by 0011, "GUI/REST-editable, so it can grow at runtime"; `version`
  (int), `active` (bool), index `idx_prompts_key_active`; **stable id `pbc_1968329054` for rebuild
  reproducibility**; the default row is seeded at first run by `prompt.Seed`, not by the migration
  (`0009_prompts.go:11-12`).
- **data-model.md § 6. Shared schema contract (`schema/`)** — the existing generated-artifact drift
  guard this decision is measured against: `schema/profile.schema.yaml` is copied into
  `plugin/claude-plugin/schema/profile.schema.yaml` (`plugin/package.json:17`) and CI fails on any
  diff (`.github/workflows/ci.yml:85`, `release.yml:80`). The template for a git-is-truth prompt
  guard already exists; the DB prompt is the artifact it cannot cover.
- **data-model.md § 3. Identity & keying** — the store's rebuild-reproducibility posture (PM items
  can carry caller-pinned ids for the deterministic import path); the `prompts` row's stable id
  (§1.9) is the same reproducibility discipline — but nothing re-imports an *edited* `prompts.content`
  from disk on rebuild, unlike the desk-tree-sourced collections.

## 5. Options

Options span the packet's required range (git-is-truth, DB-is-truth, documented split by class) plus
the null option. Consequences are stated; none is a ruling. Hypotheses are labeled.

### Option A — git-is-truth (embed/markdown canonical; DB is a re-seeded cache; runtime edits forbidden or ephemeral)

The Go embed (`templates.SystemPrompt`) and the plugin markdown are the single sources of truth,
drift-guarded exactly like `schema/profile.schema.yaml` today (data-model § 6). The DB `prompts` row
is a first-run cache re-seeded from the embed; GUI/REST editing is disabled (or treated as an
overlay a rebuild wipes). The resolver's fall-back-to-embed (`agent.go:42-53`) becomes the *only*
path, not a fallback.

- **Sync/regeneration:** a new drift guard asserts the DB seed == the embedded default == the
  committed prompt file; regeneration is "re-run `prompt.Seed`" / rebuild the store.
- **"Reset to shipped":** trivial and automatic — a rebuild or re-seed *is* the reset.
- **Consequence:** neutralizes the identity-neutrality hazard by construction (there is no write path
  from runtime into a shipped artifact) and the rebuild-from-disk hazard (nothing durable to lose).
  Symmetric with PM (markdown == git). **But** it retires the runtime-editability affordance the
  50M-char GUI-editable `content` field (data-model § 1.9) was built for — that mechanism becomes
  vestigial, and the resolver's "prefer active row" branch is dead code to remove.
- *Hypothesis (labeled):* lowest-surprise given the repo's existing drift-guard machinery and the
  "personalize only via `profile.yaml`, never edit shipped artifacts" wall — but it is the option
  that most reduces a shipped capability, so it must be a deliberate walk-back, not a default.

### Option B — DB-is-truth (embed/markdown ship a seed only; runtime edits are durable personalization)

The embed is a one-time seed; once a desk exists, the DB row is authoritative — which is already how
the resolver behaves (`agent.go:42-53`). To honor rebuild-from-disk, an edited prompt must be
persisted to a desk-tree file (a `_knowledge/`-class personalization artifact) and re-imported on
rebuild — otherwise the edit is lost (data-model § 3: nothing re-imports edited `content`).

- **Sync/regeneration:** requires a new on-disk personalization source + an import/export path
  (analogous to the not-yet-wired PM importer reserved for a later adoption path); the DB row and the
  disk file must round-trip.
- **"Reset to shipped":** delete the personalization file/row → the resolver already falls back to
  `prompt.Embedded()` when no active row exists (`agent.go:44`). Clean, provided "no active row" is
  reachable.
- **Consequence:** makes prompt edits first-class personalization, symmetric with
  `_knowledge/profile.yaml`. **But** it introduces new disk-artifact + import machinery and, critically,
  a fresh identity-neutrality trap: the personalization file MUST be gitignored like the real profile
  and MUST never be mistaken for a shipped artifact. It also does nothing for PM (no DB prompt exists),
  so either PM gets the same disk-overlay treatment or the split is admitted (Option C).
- *Hypothesis (labeled):* only coherent if paired with a hard rule and a CI check that the
  personalization prompt file is untracked — the neutrality scanner exempts `docs/` and repo-root but
  scans the shipped tree; a mis-placed prompt overlay under `plugin/` or `librarian/` would fail it,
  which is the desired failure, but the file's home must be chosen so that failure can't be silenced.

### Option C — documented split by prompt class (shipped persona git-canonical; user customization an explicit overlay)

Separate two objects that today share one field: (1) the **shipped persona/base** — git-canonical,
drift-guarded, same mechanism per lane (embed for librarian, markdown for PM); (2) an **explicit
user-customization overlay** — bounded and additive, composed with the base at resolve time. Because
the base and the overlay are *different objects*, the drift guard covers the base and is by
construction silent about the overlay — which resolves the "DB can't be drift-guarded against git"
tough spot by not trying to guard the runtime layer at all.

- **Sync/regeneration:** the base regenerates like any generated artifact; the overlay is never
  drift-checked. The resolver composes `base + overlay` (append/prepend, or a small set of named
  knobs), with the base's safety-critical instructions non-overridable.
- **"Reset to shipped":** drop the overlay; the base is untouched and always intact.
- **Consequence:** keeps runtime editability without letting an edit masquerade as shipped truth; the
  base stays drift-guardable and the overlay stays personalization. **But** it is the most build:
  needs a compose semantics, an overlay schema (plausibly a `schema/profile.schema.yaml` block —
  data-model § 6 — since it already has `preferences` + an open `custom:` escape hatch), and a rule
  that the overlay is additive-only so a user cannot silently delete a shipped safety instruction. It
  must still answer B's rebuild-durability question for the overlay (persist to disk, or accept
  ephemerality).
- *Hypothesis (labeled):* the only option that both keeps the GUI-editable affordance and stays
  neutrality-safe; the cost is defining a compose contract that both lanes and the in-binary resolver
  honor identically — which is where it touches D5's "one contract, two instantiations."

### Option D — status quo, documented as a deliberate per-lane split (no mechanism change)

Keep librarian = embed → DB → resolver (DB wins at runtime, ephemeral across rebuild) and PM =
markdown; simply *document* it as the intended split. Treat the stale-prompt symptom (surface § 6
findings 2/3) as a separate content bug to fix in the embed, not a governance change.

- **Sync/regeneration:** none added; the DB prompt stays un-drift-guarded by design.
- **"Reset to shipped":** delete the active row (resolver falls back to embed) — already available,
  just undocumented.
- **Consequence:** cheapest, and it names reality. **But** it leaves the recurring staleness
  ungoverned: nothing keeps the embed, the DB row, and the live tool set consistent, so findings 2/3
  will re-appear the next time the tool set changes. It also leaves the two lanes asymmetric with no
  stated contract, which D5 may reject.
- *Hypothesis (labeled):* acceptable only if the session also rules that the in-binary loop never
  composes a PM prompt (C1/D5) — i.e., the split is defensible *because* the two brains are
  deliberately different, not merely unreconciled.

## 6. Decision criteria

Any ruling must satisfy the constraint walls that bind here (`../README.md` §7):

- **Identity-neutrality (CI-enforced, `scripts/check-neutrality.mjs`).** No runtime prompt edit may
  flow into a shipped artifact. The mechanism must make "export my edited prompt into the embed/plugin"
  either impossible or a neutrality-scanner failure — never a silent success. (Grounded by
  agent-symmetry.md § Verdict table — the 6 … asymmetries #4.)
- **Store rebuilds from disk; document bodies are never persisted (sole exception: the revisions
  pre-image ledger).** The ruling must state explicitly what happens to a runtime prompt edit on
  rebuild — durable (persisted to a desk-tree file and re-imported) or ephemeral (re-seeded from the
  embed). Silence here is the current bug. (Grounded by data-model.md § 1.9 and § 3.)
- **Personalization only via `_knowledge/profile.yaml`, never by editing shipped artifacts.** If
  runtime prompt edits are personalization (Options B/C), they must live where personalization lives
  and be gitignored accordingly; if they are not durable personalization (Options A/D), the ruling
  must say so and remove the affordance's implied promise.
- **Forward migrations only on shipped collections.** The `prompts` collection (0009) is shipped and
  carries a stable id for rebuild reproducibility (data-model § 1.9); any governance change that
  alters its writability or adds a source field is a *new* forward migration, never an edit to 0009.
- **One mechanism, honestly scoped to both lanes.** The generated-artifact drift guard (data-model
  § 6) is the benchmark: whatever is git-canonical must be guardable by that pattern, and whatever is
  runtime-editable must be explicitly *outside* the guard's scope — the ruling states which store is
  which, for the librarian embed AND the PM markdown.

## 7. Blast radius

Concrete artifacts a ruling touches, by option class. Live stores exist (the owner's own desks), so
each existing desk already has a seeded — possibly edited — `prompts` row; migration reality is part
of the radius.

| Artifact class | Git-is-truth (A) | DB-is-truth (B) | Documented split (C) | Status quo documented (D) |
|---|---|---|---|---|
| **Collections / migrations** | new migration making `prompts` read-only / seed-only; decide fate of existing edited rows (reset) | new migration + a disk-import path for edited prompts; existing rows preserved or exported | new migration adding an overlay field/row distinct from the base | none |
| **Go embed + resolver** | `librarian/templates/librarian-system-prompt.txt`, `prompt.Seed`, and the resolver's prefer-active-row branch (`agent.go:42-53`) — the branch becomes dead | resolver unchanged; add export-on-edit / import-on-rebuild | resolver gains a `base + overlay` compose step | resolver unchanged; embed content fixed (findings 2/3) |
| **Drift guard** | extend the schema-copy pattern (data-model § 6) to assert embed == seed == committed prompt | none on the base; overlay explicitly un-guarded | guard the base; overlay explicitly out of scope | none |
| **schema/ contract** | none | possible prompt-overlay block in `schema/profile.schema.yaml` | likely overlay schema in `profile.schema.yaml` (`preferences`/`custom:`) | none |
| **Skills / personas** | PM markdown becomes symmetric (git-canonical, already is) | PM needs a disk-overlay twin or the split is admitted | both lanes adopt base+overlay; `pm-operator.md` composes | document the per-lane split as intended |
| **Docs / ADRs** | new ADR; spec §4.10/§6.1 (prompt seed/load, cited in `prompt.go` header) reconciled | new ADR + personalization-file doc | new ADR + compose-contract spec | new ADR recording the deliberate split |
| **Stale-prompt content fix (findings 2/3)** | fixed as a side effect once the embed is the single guarded source | still a separate embed edit | still a separate embed edit | explicitly a separate bug |

## 8. Out of scope / interactions

- **D5 owns the contract; D6 owns the mechanism.** Whether each surface *should* name exactly one
  instruction source, and whether the in-binary loop gets a composed PM prompt or loses PM tools, is
  D5 (agent contract & surface parity). D6 rules only *how* single-sourcing works once D5 says a
  surface has one source. The "one mechanism honestly scoped to both lanes" criterion is where D6
  feeds a parameter back to D5, not where it decides parity.
- **The specific content fix to the stale librarian prompt** (surface § 6 findings 2/3 — advertising
  `apply_fix`/`restore` it may not hold, never naming PM tools) is a *symptom*, not the governance
  ruling. D6 decides which store is canonical; the edit itself is downstream. It is the canary that
  proves the governance gap is real, not a task D6 completes.
- **A general disk-import path for store-native state** (the not-yet-wired PM importer, reserved for a
  later adoption path per surface § 6 finding 5) is D8-adjacent. Options B/C would *reuse* such a path
  for prompt overlays, but building the generic importer is not D6's remit.
- **`desk_config` gate-rule schema and the `status_label` vocabulary** are Lane B, queued elsewhere
  (book index note); D6 does not touch them even though `desk_config` is also a per-desk editable
  store row.

## 9. Uncertainties

- **Rebuild path for the `prompts` collection is not directly cited.** The dossiers establish that
  `prompts.content` is store-native and GUI-editable (data-model § 1.9) and that the store rebuilds
  from the desk tree (constraint wall), which *implies* an edited prompt is re-seeded from the embed on
  rebuild — but no dossier traces the actual rebuild path for this collection specifically. The
  "ephemeral vs durable" premise the options rest on needs that path re-derived in the session before
  a ruling binds.
- **`prompt.Seed` default-row seeding was flagged unverified in the data-model pass** (data-model
  § Gaps: "seeded at first run by `prompt.Seed` … not independently verified in this pass"). I
  sanity-checked `prompt.go` directly and confirmed the seed writes `templates.SystemPrompt` to key
  `librarian.system`, is idempotent, and never clobbers a GUI/REST edit — but this is a spot-check, not
  a re-derivation; treat it as a hypothesis where a ruling binds.
- **PM symmetry direction is D5's, but D6 must assume one.** The evidence confirms `pm.system` is
  absent (agent-symmetry § Verdict … 7-row, grep negative). Whether PM *should* gain a DB-editable
  prompt, adopt the base+overlay split, or stay markdown-only is a D5 parity call that changes which
  D6 option is coherent — carried forward as a cross-brief dependency, not resolved here.
- **"Reset to shipped" via the empty-row fallback is only partially verified.** The resolver falls back
  to `prompt.Embedded()` when no active row is found (`agent.go:44`, read in source), but whether an
  *edited-non-empty* row can be returned to "no active row" through any shipped affordance (vs a raw
  admin-console delete) was not established in the dossiers. Options B and D lean on this fallback and
  need the reset affordance confirmed.

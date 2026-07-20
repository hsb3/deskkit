> **Tracking:** #TBD, ADR 0015 (2026-07-20 design session). Document the `Seed`/reset semantics
> ADR 0015 rules and add a drift guard across the version-controlled prompt copies, so the embed
> and the spec's own "kept verbatim" quote of it cannot silently diverge again.

## Problem

ADR 0015 (`docs/decisions/0015-prompt-governance.md`) rules git-is-truth for agent instructions:
the version-controlled sources (the Go embed, plugin markdown, and later bundle markdown) are
canonical; the DB `prompts` row is a re-seeded cache; runtime GUI/REST edits are ephemeral **by
documented rule**; "reset to shipped" = clear the row, the embed re-seeds; durable customization
is `_knowledge/` only. None of that governance is currently *stated* anywhere near the code, and
the one on-disk copy-relationship that exists today has already drifted with nothing to catch it.

**How seeding actually works today** (re-derived directly from source, not just the ADR):

- `prompt.Embedded()` returns the `//go:embed`'d default (`librarian/templates/templates.go:20-21`
  embeds `librarian/templates/librarian-system-prompt.txt`;
  `librarian/internal/modules/librarian/prompt/prompt.go:16-18` exposes it).
- `prompt.Seed(app)` (`prompt.go:24-40`) inserts that embedded text as the `prompts` row for
  `key = "librarian.system"` **only if no row exists yet** (`prompt.go:26-28`: any existing row,
  regardless of its content, short-circuits to a no-op). The migration itself
  (`librarian/internal/modules/librarian/collections/0009_prompts.go:11-12`) creates the
  collection but seeds no row — seeding is `prompt.Seed`'s job, mirroring the
  `.librarian-ignore` auto-create.
- `Seed` runs at **every** entry point, not only `serve`: under `OnServe`
  (`librarian/cmd/deskkit/main.go:251-253`) and inside `requireConfig`
  (`main.go:648-655`), the choke point every one-shot CLI/MCP command passes through. This means
  today's "reset to shipped" already works mechanically — delete the row, the very next command
  or a `serve` restart re-seeds it — but the affordance is not documented anywhere an operator
  would find it.
- At run start, `systemPrompt` (`librarian/internal/modules/librarian/agent/agent.go:42-53`)
  loads the active, highest-`version` row and falls back to `prompt.Embedded()` if none exists
  (`agent.go:43-51`) — the DB row wins whenever one is present, exactly as ADR 0015 names it (a
  cache the resolver prefers, not the frozen embed).
- The DB-seed-equals-embed leg already has a Go-level guard:
  `librarian/internal/modules/librarian/prompt/prompt_test.go:16-58`
  (`TestSeed_CreatesPromptRowThenIsIdempotent`) asserts `rec.GetString("content") == Embedded()`
  on first seed and that a second `Seed` call never replaces the row (`prompt_test.go:36-38`,
  `:55-57`). What is **not** guarded is the *other* on-disk copy below.

**The drift that already exists (independently re-derived, not just cited from the decision
book).** `docs/pocket-librarian-v1-spec.md:1433` introduces a fenced block as "The full
embedded-default text (the first-run seed, kept verbatim)" (lines 1435-1464), and a second
"Decisions" bullet at `:2315-2324` describes the same content. Both are stale relative to the
real source:

- The real file (`librarian/templates/librarian-system-prompt.txt:5-14`) lists **seven** tools,
  including `record_feedback` (`:14`, plus its usage paragraph at `:28-29`).
- The spec's quoted block (`docs/pocket-librarian-v1-spec.md:1440-1448`) lists **six** tools and
  never mentions `record_feedback` at all — confirmed by a negative grep: `record_feedback`
  appears zero times anywhere in `docs/pocket-librarian-v1-spec.md`. Both the quoted block's
  intro line (`:1440`, "You have six tools") and the separate Decisions bullet
  (`:2316`, "the six tools") repeat the stale count.

This is the exact class of bug ADR 0015 names as the live proof the split was ungoverned — it is
not hypothetical, it is checked in today. Nothing currently asserts these two copies agree, the
way `schema/profile.schema.yaml` vs. its plugin-bundle copy already is (`.github/workflows/ci.yml`
step "Plugin — packaged artifacts drift guard": `bun run package` then
`git diff --exit-code -- claude-plugin/`, `.github/workflows/ci.yml:81-85`).

Separately: nothing in code or docs currently states the ephemeral-by-rule / reset-to-shipped
framing in ADR 0015's own words — `prompt.go`'s comments describe mechanism ("GUI/REST edits are
never clobbered," `prompt.go:23`) without stating the governance intent (a runtime edit is
*designed* to be lost at reset, not an accident to work around).

## Deliverables

- **(a) `Seed` semantics documented as governance, not just mechanism.** Extend the doc comments
  in `librarian/internal/modules/librarian/prompt/prompt.go` (package doc `:1-8`, `Seed` doc
  `:20-23`) and `docs/pocket-librarian-v1-spec.md` §4.10 (`:603-632`) to state, in ADR 0015's own
  terms: the `prompts` row is a re-seeded cache, not canonical; GUI/REST edits are ephemeral **by
  design** (a rebuild/re-seed is the intended reset path, not data loss); the durable
  customization path is `_knowledge/` only, never a DB prompt edit and never an edit to a shipped
  artifact. Cite ADR 0015 from both.
- **(b) A drift guard across the version-controlled prompt copies.** A new script,
  `scripts/check-prompt-drift.mjs`, dependency-free like `scripts/check-kits.mjs`, that extracts
  the fenced ```text``` block in `docs/pocket-librarian-v1-spec.md` (currently `:1435-1464`,
  identifiable by the preceding sentinel line `:1433`) and asserts it is byte-identical to
  `librarian/templates/librarian-system-prompt.txt`. Fails naming the file and where they diverge,
  mirroring the existing `git diff --exit-code -- claude-plugin/` packaged-artifact pattern
  (`.github/workflows/ci.yml:81-85`) rather than reinventing a diff format. Wire it into
  `make check` (Makefile `check:` target) and the CI `ci` job (`.github/workflows/ci.yml`),
  alongside the existing `check-kits.mjs` step. Today's actual drift (deliverable (a)'s finding)
  means this guard **fails at the moment it is added** until the spec text is corrected — land the
  correction (folded into deliverable (d)) in the same PR so the guard merges green, and note in
  the PR description that it was red against `main` first (the repo's own red-then-green
  regression-test bar).
  Prose note, not a hard dependency: when the ADR 0014 desk-persona bundle lands (tracked
  separately as `desk-persona-bundle`), it adds a further prompt surface (bundle markdown). When
  it does, its prompt copies join prompt governance — either by extending this guard's file list
  or via the bundle's own regenerate+diff guard (see `_meta/plans/desk-persona-bundle/plan.md`);
  the requirement is one governed mechanism per copy, not one script.
- **(c) "Reset to shipped" documented for an operator.** Add a short subsection to
  `librarian/README.md`, in or immediately after "The admin console" (`README.md:191-210`, the
  section that already documents `make gui` / `http://127.0.0.1:8090/_/`): open the console,
  delete the `librarian.system` row in `prompts`, and the next `deskkit` command or a `serve`
  restart re-seeds it from the embed — no new code needed, this is `prompt.Seed`'s existing
  seed-if-absent behavior (`prompt.go:26-28`, called from both `main.go:251-253` and
  `main.go:648-655`).
- **(d) Spec prose reconciled.** In `docs/pocket-librarian-v1-spec.md`:
  - §4.10 (`:603-632`): add the ADR 0015 governance framing (re-seeded cache; ephemeral edits;
    `_knowledge/` as the only durable path) so "editable, versioned" no longer reads as an implicit
    durability promise.
  - Fix the quoted "verbatim" block (`:1435-1464`) and the Decisions bullet (`:2315-2324`) to
    match the real seven-tool `librarian-system-prompt.txt`, including `record_feedback` and its
    usage line. This is what makes deliverable (b)'s guard pass. Note: the embed's intro line
    carries no count word at all (`You have these tools:`,
    `librarian/templates/librarian-system-prompt.txt:5`), so — since deliverable (b)'s guard
    asserts byte-identity to that intro line — the byte-identical spec correction REMOVES the
    stale "six tools" count from the quoted block's intro (`:1440`) rather than replacing it with
    "seven"; the separate Decisions bullet at `:2316` also drops its count rather than updating it.
  - Add a citation to ADR 0015 near §4.10 and §6.1, per the repo's cite-the-ADR-where-it-binds
    convention.

## Acceptance criteria

- [ ] `node scripts/check-prompt-drift.mjs` exits non-zero, naming the drifted file/section, when
      `librarian/templates/librarian-system-prompt.txt` and the spec's quoted block disagree
      (demonstrated once against `main`, before deliverable (d) lands); exits 0 once the spec text
      is corrected.
- [ ] `make check` runs the new guard (wired into the Makefile `check:` target and
      `.github/workflows/ci.yml`) and is green on the merged PR.
- [ ] `librarian/internal/modules/librarian/prompt/prompt.go` and
      `docs/pocket-librarian-v1-spec.md` §4.10 both state, reviewably by reading them: the `prompts`
      row is a re-seeded cache, not canonical; GUI/REST edits are ephemeral by rule; `_knowledge/`
      is the only durable customization path.
- [ ] `librarian/README.md` documents the reset-to-shipped affordance (delete the row via the
      admin console; the next command or restart re-seeds it).
- [ ] `TestSeed_CreatesPromptRowThenIsIdempotent` (`prompt_test.go:16-58`) continues to pass
      unchanged — `go test ./internal/modules/librarian/prompt/...` (run from `librarian/`) is
      green, showing the DB-seed-equals-embed leg still holds at the Go-test layer.
- [ ] `make test` and `make verify` are green (no runtime-behavior change: `Seed`'s seed-if-absent
      logic and the resolver's prefer-active-row branch in `agent.go:42-53` are documented here,
      not altered).

## Dependencies & gates

- Repo checks (`make check`) — fires, and is extended: add the new `check-prompt-drift.mjs` step
  next to `check-kits.mjs` (Makefile + `.github/workflows/ci.yml`).
- Unit tests (`make test`) — fires because it always fires; no new/changed Go behavior is
  expected to require new test cases beyond the existing `prompt_test.go` coverage.
- Librarian integration (`make verify`) — fires because `librarian/` doc comments and
  `librarian/README.md` change; no `prompts`-specific check exists in `librarian/verify.sh` today,
  and this issue does not add one (flagged as a possible future gap, not required here).
- Identity neutrality (`scripts/check-neutrality.mjs`) — mixed: the new `check-prompt-drift.mjs`
  script itself lives under `scripts/`, which is outside the scan scope
  (`scripts/check-neutrality.mjs:52`'s `SCAN_DIRS` is `plugin`, `librarian`, `kits` only) and the
  `docs/pocket-librarian-v1-spec.md` prose edits (deliverable (d)) land under `docs/`, also
  exempt. But deliverable (c)'s `librarian/README.md` edit DOES fire: `librarian` is scanned
  recursively (`scripts/check-neutrality.mjs:52`), and that recursion reaches
  `librarian/README.md` — there is no root-level carve-out for it. The gate passes only if the
  new "reset to shipped" subsection stays free of any person/org/repo/#issue token, exactly like
  every other edit under `librarian/`.
- Bundle drift guard (`make package` + `git diff --exit-code` on `plugin/claude-plugin/`) — does
  **not** fire: nothing under `plugin/core`, `plugin/mcp`, or `schema/` changes.
- Version sync / kits drift — do **not** fire: `VERSION`, the shipped manifests, and `kits/` are
  untouched.
- DB migration discipline — does **not** fire: `0009_prompts.go` is not touched; no collection is
  added, altered, or made read-only. The DB `prompts` collection and its GUI-editable `content`
  field stay exactly as shipped; this issue documents and guards, it does not retire them.
- CHANGELOG — fires: a new gate plus a doc reconciliation is a product change; add an
  `[Unreleased]` entry on the closing PR, and update the "Architectural rules and their enforcing
  checks" table in root `CLAUDE.md` (it names each rule + its enforcing script) with a row for the
  new prompt-copy drift guard, per `CLAUDE.md`'s own "update it in the same change" rule.
- Regression-test bar — the new script is itself the regression-test surface for this class of
  bug; show it red against `main`'s current spec/embed mismatch before the fix, matching the
  repo's red-then-green practice.

## Out of scope

- The centralized prompt-tuning mechanism (`prompt-tuning-centralized`, its own v2-track design
  item — see that issue).
- Flipping DB-as-truth. ADR 0015 only re-opens toward it if the ADR 0009 trust gate flips; that is
  not this issue.
- The ADR 0014 desk-persona bundle's own prompt sources (tracked separately as
  `desk-persona-bundle`, a sibling child of the ADR 0010-0017 v1 epic). This issue's guard is
  designed to extend to it later; wiring that bundle's actual files in is that child's work.
- The stale-prompt **content** bug beyond the "six tools" line needed to make the drift guard
  pass — i.e., the librarian prompt unconditionally advertising `apply_fix`/`restore` regardless
  of `LIBRARIAN_AUTONOMOUS_WRITES`, and never mentioning PM tools when PM is enabled, is a
  content-accuracy problem the design-session decision book explicitly calls a symptom, not the
  governance ruling to fix here (`_meta/research/2026-07-design-session/decision-book/D6-prompt-governance.md`,
  section 8, "Out of scope / interactions").
- Removing or disabling the DB `prompts` collection or its GUI-editable `content` field — ADR
  0015 keeps it as a re-seeded cache; it is not retired by this issue.

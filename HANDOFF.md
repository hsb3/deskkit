_Session-to-session bridge for desk-standard. Read this before working; update it at the end
of any substantial session. Secret-free — keep live URLs/credentials out of the repo entirely._
Status: active (2026-08-05, post working-desk removal: `_meta/` deleted, this file now lives at
repo root, doc-link repoints green-but-uncommitted — see §1; prior eras: 2026-07-24 reorg-reconciliation,
2026-07-23 ADR 0022 acceptance; older versions in git history)

# HANDOFF

## 0. Orientation

Two products over one shared schema, all identity-neutral (nothing shipped carries a
person/org/repo/issue): **`plugin/`** (harness-pure TS core + stdio MCP server) and
**`librarian/`** (deskkit: single Go binary, embedded PocketBase, eino agent loop,
record-original-first write boundary), with **`schema/`** as the contract both read. The two
marketplace bundles live in a top-level **`plugins/`** tree (`plugins/desk-standard/` — the TS
plugin adapter, GENERATED server.js; `plugins/desk-persona/` — the composed librarian+PM bundle);
extracted from under `plugin/` on 2026-07-22, and a marketplace install copies ONLY
`plugins/desk-standard/`. Watch the one-letter `plugin/` vs `plugins/` distinction. **Superseded
direction — ADR 0022 (Accepted 2026-07-23):** this two-bundle / two-MCP-server split is being
collapsed onto the Go binary (one plugin, one MCP server; TS server retired) — decided, NOT yet
built (roadmap Wave 3); the description above is current CODE reality. Personalization happens only in a desk's `_knowledge/profile.yaml` (the repo
itself is NOT a desk — ruled 2026-07-21, PR #188). Root `README.md` is the front door;
`CLAUDE.md` is the agent digest (hot-loaded — don't duplicate it here); CHARTER at
`docs/development/CHARTER.md`, live specs at `docs/development/specs/` (§3 has the full docs
layout); operator docs stay at `librarian/README.md`.

Siblings: `hsb3/dotfiles-agents` (pattern source). The **paired executive desk** is
`~/Documents/EXECUTIVE_DESK/Projects/desk-standard-desk`; the off-repo decision spine it
inherits from lives in `…/Projects/dotfiles-agents-desk/_structure/decisions/` (formerly
`dev-tooling-desk`, whose stale copy sits in `…/Projects/ARCHIVE/dev-tooling-desk-old/` — don't
read the archive copy as current).

## 1. Current standing (as of 2026-08-05) + what's next

**⚠ Working desk REMOVED (2026-08-05, commits `3be0ef6` + `c029607`).** The entire `_meta/`
tree — plans (incl. the adoption-feedback roadmap and the #197 build plan), research, archives,
signoff records, plan utils — was deleted; only this handoff survived, moved to repo-root
`HANDOFF.md` (tracked). **Every `_meta/...` path named below resolves only in git history**
(checkout `206d969`, the commit before the removal). Same day:

- `c029607` also **deleted the frozen OpenCode spike `plugin/opencode/`** (plugin `bun test` is
  now 65 tests across core/mcp/desk-persona) and repointed the doc-link breakage the removal
  caused (CLAUDE.md → `_meta/HANDOFF.md`, element-model-v2-draft's research citations) plus a
  ~263-line README rewrite. `make check` and `node scripts/check-doc-links.mjs` exit 0.
- ADR "Raised by" provenance citations into `_meta/research/` (ADRs 0009–0016, 0018, 0020) now
  point at git history; they're prose backticks, gate-invisible — annotate opportunistically.
- `a6f5eb7`: `.claude/agent-memory/` folded into the tracked `.claude/memory/` (9 durable
  reviewer memories migrated with dated fact-updates; 7 pruned as mooted by these same
  removals or superseded by code).

**Wave-v4 EXECUTED same-session** (6 crews + 1 reconciliation PR, all merged): schema-v2 arc
closed for this cycle (#124, #125, #126 v2 half — the model stays `status: draft`, gated on the
owner's #87 call), all findings-driven small buildables closed (#184, #185, #182, #181, #187),
and both owner-raised UX/docs issues closed (#189 TUI discoverability, #190 user/dev docs
split). Final union of `main` re-verified bare post-merge: `make check`, `make test`,
`make verify` (61/61, up from 55), `make e2e` (43/45, 2 LLM-key skips). **The owner decision
queue is EMPTY** going into and coming out of this run — no item needed a ruling; #185's
empty-type gate policy and #190's docs-location call were engineering judgment calls the crews
made and documented themselves. **The 1.0.0 milestone still holds only #87** (the release-cut
epic).

**Decision-record state:** ADRs 0009–0018 bind (2026-07-20 design session); **ADR 0020**
(authoritative PM claims) owner-confirmed 2026-07-21, supersession window closed; **ADR 0021**
graduates the off-repo "decision 0021" (F-fork citations resolve there; §F5 executed as a
REMOVAL, not a rename — repo-root `_knowledge/` gone, desk convention + `profile_root` constant
untouched, example profile single-homed in the desk-setup scaffold template); **ADR 0008**
amended: PM ships default-on. Decision packages for Henry are always the owner-signoff HTML
form, never chat questions (project memory; the 2026-07-21 batch proved the pattern at 10-items
scale).

**Adoption-feedback design session EXECUTED 2026-07-23** (docs committed 2026-07-23,
`1ac1ba1`/`ba42fd7`; the roadmap file was later deleted with `_meta/` — git history only): early
user feedback ("super-confusing; two plugins + two MCP servers; alphabetical CLI; no 'list my
desks'; unclear config / no central LLM-key home; visual data browser?"). Two owner rulings →
**(1)** collapse everything onto the Go binary (one plugin/one MCP server; the 4 profile tools move
to Go; TS server retired) = **ADR 0022 (Accepted)**, supersedes 0016; **(2)** a machine-local
`~/.config/deskkit/config.yaml` (0600) stores provider + model + **the LLM API key value**
(precedence env > per-desk profile > central > default). Full design + ordered 4-wave roadmap:
`_meta/plans/adoption-feedback-roadmap.md`. New memories: `consolidation-single-binary-decision`,
`feedback-no-public-1.0.0-nagging`. Nothing built yet; the 2026-07-23 planning-desk reconciliation
run is now a §2 era pointer.

**✓ Reorg reconciled (2026-07-24, PR #201):** the 2026-07-23 `docs/` reorg had broken 3 `make check`
gates + the tool-surface Go test; ~90 citations repointed, all green. The durable outcomes — the
docs-layout contract and the `check-doc-links` gate — live in §3; blow-by-blow in PR #201.
Leftover (pre-existing, #77): `docs/development/README.md:62` + `schema/README.md` still name the
long-gone `_meta/build-brief.md` in gate-invisible backticks — remove or redirect when convenient.

**NEXT, in order of consequence:**
1. **#197** (gate:v2-final, filed 2026-07-22) — reconcile the software-spec phase-machine with the
   PM item phase-machine and name the building→shipped gate rule. The one open item blocking full
   schema-v2 epic (#130) closure. Build plan was `_meta/plans/phase-machine-reconciliation/` —
   deleted 2026-08-05, recover from `206d969`. (Target: `docs/development/specs/element-model-v2-draft.md`.)
2. **Adoption-feedback roadmap, Waves 1–3** (roadmap doc deleted with `_meta/` — recover from
   `206d969`; ADR 0022 gate cleared) — issues NOT yet filed. Wave 1 quick wins (CLI command groups, `deskkit desks`,
   `deskkit config`, surface the existing PocketBase data browser), Wave 2 central config, Wave 3
   the collapse. All Go-lane; Wave 1 items are independent/parallelizable.
3. ~~**#199** + **ts-proxy implementation**~~ **MOOTED by ADR 0022** — the ts-proxy path is
   withdrawn (the collapse reaches one mount without a spawned-child proxy). Mark #199 + the
   unfilled ts-proxy build superseded when Wave 3 files.

**Carried-over question, MOOT via ADR 0022:** the 4 TS MCP tools not surfacing agent-callable in a
live session — superseded by the collapse; re-check at Wave 3's live test.

**Standing, settled decisions:** **Do NOT prompt/nag Henry about going public or cutting 1.0.0
(#87)** — his call, not a standing agenda item (2026-07-23; memory `feedback-no-public-1.0.0-nagging`).
Public launch stays deferred / repo PRIVATE (Henry, 2026-07-19; public `curl|bash` 404 is by-design
— §3 has the authed-`gh` workaround). #87 (1.0.0 maturity) is owner's-court, well-informed (value
eval `_meta/research/2026-07-cohesion-value-evaluation/`; model-sim deficiency report
`_meta/research/model-simulations/`) — record it, don't surface it. OpenCode #12 parked until ≥
v1.0.0 (simpler post-ADR-0022: one Go server to expose). #36 SOP library deferred post-1.0.

**Open backlog: the live triaged view is pinned issue #155** — refresh it (edit the body via
`gh issue edit 155 --body-file`, bump its date) at session boundaries alongside this file; it
lives as an issue precisely so refreshes never touch commit history.

## 2. Delivery eras (newest first — blow-by-blow lives in #155's body, the PRs, and this file's git history)

- **2026-07-23 · planning-desk reconciliation + adoption-feedback design** (no code shipped) —
  every open non-epic issue given a reviewed, source-grounded plan (`_meta/plans/<slug>/`, all
  governance scripts exit 0); 16 shipped plan folders archived to `_meta/_archive/`; #155 re-titled
  📌. Then a feedback-driven design session: **ADR 0022 (Accepted)** collapse-onto-deskkit
  (supersedes 0016) + the 4-wave adoption roadmap (§1). Docs UNCOMMITTED.
- **2026-07-22 · wave-v4** — 7 PRs (#192–#196, #198, #200); closed #124 #125 #126 #181 #182 #184
  #185 #187 #189 #190; schema-v2 arc closed this cycle (contract versioning, reviewed v2 model,
  v1+v2 sims walked), TUI discoverability + user/dev docs split, 3 librarian bugfixes, tool-surface
  cleanup, reconciliation. Filed #197, #199.
- **2026-07-21 · waves v1–v3 + v0.8.0** — PRs #156–#188; ~40 issues closed (Go module rename, CI
  hardening, PM completion ADR 0019/0020 + default-on, browser session surface, TUI archive
  lifecycle, profile-root constant+guard, repo-root `_knowledge/` removed, `make e2e` 45-check gate,
  gofmt/query-kind gates, desk-pm folded into desk-persona); **v0.8.0 released**. Detail in PRs + #155.
- **2026-07-16→21 · genesis → v1 build (pre-wave)** — ADRs 0001–0018; chat TUI, XDG store, store
  self-init, `pocket-librarian`→`deskkit` rename, PM shipped dark (v0.7.0); epic #129 v1 build
  (PRs #105/#112/#113/#136–#147); integration-testing PR #152 (report at
  `_meta/research/2026-07-20-integration-agent-led-testing/`). Detail in git history + #155.

## 3. Conventions & gotchas

_Core coding conventions — gates, generated artifacts, PocketBase-bootstraps-before-cobra,
migration patterns, config resolution, store self-init, the neutrality lint's bare-issue-ref
trap — are in **CLAUDE.md** (hot-loaded). Only what CLAUDE.md doesn't cover lives below._

- **Installing from the private repo** (until public launch): `make install` from a clone, or
  authed `gh release download v0.8.0 --repo hsb3/desk-standard --pattern
  "deskkit_*_$(uname -s|tr '[:upper:]' '[:lower:]')_$(uname -m|sed 's/x86_64/amd64/;s/aarch64/arm64/')"
  --output ~/.local/bin/deskkit --clobber && chmod +x ~/.local/bin/deskkit` (verify against
  `checksums.txt`). **A release never updates PATH binaries or the installed plugin** — separate
  artifacts, update each deliberately.
- **Gate discipline beyond CLAUDE.md's "bare, never piped"**: in zsh, `${PIPESTATUS[0]}` is
  bash-only (zsh is lowercase `$pipestatus`), so `${PIPESTATUS[0]:-$?}` silently reads the
  pipe-tail's exit and can mask a failing gate — redirect gate output to a file and check `$?`
  bare. Local shellcheck can disagree with CI's version (SC2317 vs SC2319/SC2329 skew): "clean
  locally" doesn't prove the CI gate; main-branch CI is the authoritative signal.
- **Multi-agent wave playbook** (distilled from three foreman runs; auto-memory mirrors the
  harness bits):
  - Background crews routinely stop mid-task ("now let me…"): SendMessage the SAME agent
    "continue to completion" + restate the checklist. Never re-brief. Works even for agents
    orphaned by a dead parent process (they resume from transcript) — don't launch duplicates.
    A lead's worker notifications route to the ROOT session — relay them; tell leads to verify
    worker state themselves. Under API instability, brief writer agents to emit long files
    INCREMENTALLY (Write first section, Edit-append the rest) so a dropped stream loses one
    section, not the file.
  - After ANY crew reports, check `git branch --show-current` in the main tree before
    ref-sensitive commands (an isolation worktree once came up ORPHANED and the agent worked in
    the main checkout, leaving it on its branch — nearly mis-tagged a release).
  - Two crews touching the same file (root Makefile/ci.yml/README) WILL collide: plan
    disjoint scopes, expect a union rebase (`git rebase origin/main`, keep both intents,
    re-gate, `--force-with-lease`). Collision predictions in a plan are hypotheses —
    diff-check before merging.
  - `git pull` local main before any agent reads the main tree; refresh `plugin/node_modules`
    (`bun install`) on main after a multi-worktree session before trusting a local gate
    failure; chain merge→cleanup with `&&` never `;`.
  - Briefs that produce Go comments/tests under `librarian/` must restate the bare-issue-ref
    neutrality trap; verify a resumed/interrupted agent's build yourself; verify an AI
    reviewer's "cleaner code" suggestion empirically (one would have reintroduced a bug via
    cobra's Execute()-time flag merge).
  - Shared machine-global test store (`~/.local/share/deskkit/pm-actor-subprocess-test`)
    collides across worktrees at different schema versions ("refusing to downgrade") — delete
    the throwaway store and re-run; proper fix tracked as #182.
- **eino/TUI streaming gotchas** (bind any new streaming caller): output stream is bursty under
  the anthropic tool-call checker (use model-callback stream copies for live tokens);
  ctx-cancel doesn't abort a stuck provider stream; failed tools fire `OnError` only; zero-arg
  tool calls need the `argNormalizingTool` adapter. No terminal queries after bubbletea starts;
  theme resolves ONCE pre-program (`tui.ResolveTheme`); new TUI colors go in `newStyles`'s
  per-theme switch, never AdaptiveColor (guarded in `tui/defects_test.go`).
- **Docs layout** (reconciled 2026-07-24 — see §1): specs at `docs/development/specs/`, CHARTER at
  `docs/development/`, guides at `docs/usage/`, ADRs at `docs/decisions/`, the docs index at
  `docs/README.md`. The layout contract is `docs/development/docs-layout.md`; `scripts/check-doc-links.mjs`
  (in `make check`) gates every published+shipped doc/media citation. VHS tapes at `scripts/vhs-tapes/`,
  GIFs at `docs/assets/` (moved from `docs/media/`); re-recording re-encodes ALL GIFs (`git restore`
  unchanged ones); tapes need `ANTHROPIC_API_KEY="$(secret get ANTHROPIC_API_KEY)" bash scripts/record-media.sh`.
- **Commits auto-close**: `Resolves #N` closes when the commit lands on `main` — post the proof
  comment first (close-with-comment fails on an already-closed issue).
- **Pre-staged files**: a parallel agent may `git add` before you commit; `git commit` with a
  pathspec still sweeps the whole index — check `git status` before every commit.
- **Worktree provisioning**: untracked-and-load-bearing = `plugin/node_modules` (run
  `bun install --frozen-lockfile`). Everything else is committed (project memory is tracked at
  `.claude/memory/` since 2026-08-05; `.claude/agent-memory/` no longer exists).
- **A persistent 404 on release assets the authed API can see** = check repo visibility
  (`gh repo view`) before blaming CDN propagation.

## 4. Incident log

Durable lessons are promoted into §3 (latest promotion 2026-08-05: the two 2026-07-23
API-instability/agent-recovery incidents, now folded into the wave playbook). Dated blow-by-blow
entries live in this file's git history and the PRs they cite. Add new incidents here dated;
promote and prune at the next distillation pass. Currently empty.

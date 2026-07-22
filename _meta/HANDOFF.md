_Session-to-session bridge for desk-standard. Read this before working; update it at the end
of any substantial session. Secret-free — live URLs/credentials belong in `_meta/operations/`
(untracked) if that dir is ever created._
Status: active (2026-07-22, post wave-v4; distilled from the wave-log form — prior
versions live in this file's git history)

# HANDOFF

## 0. Orientation

Two products over one shared schema, all identity-neutral (nothing shipped carries a
person/org/repo/issue): **`plugin/`** (harness-pure TS core + stdio MCP server) and
**`librarian/`** (deskkit: single Go binary, embedded PocketBase, eino agent loop,
record-original-first write boundary), with **`schema/`** as the contract both read. The two
marketplace bundles live in a top-level **`plugins/`** tree (`plugins/desk-standard/` — the TS
plugin adapter, GENERATED server.js; `plugins/desk-persona/` — the composed librarian+PM bundle);
extracted from under `plugin/` on 2026-07-22, and a marketplace install copies ONLY
`plugins/desk-standard/`. Watch the one-letter `plugin/` vs `plugins/` distinction. Personalization happens only in a desk's `_knowledge/profile.yaml` (the repo
itself is NOT a desk — ruled 2026-07-21, PR #188). Root `README.md` is the front door;
`CLAUDE.md` is the agent digest (hot-loaded — don't duplicate it here); `docs/CHARTER.md` is the
canonical page (precedence rule); `docs/README.md` indexes the docs;
`docs/pocket-librarian-v1-spec.md` is the librarian's build spec (operator docs
`librarian/README.md`).

Siblings: `hsb3/dotfiles-agents` (pattern source). The **paired executive desk** is
`~/Documents/EXECUTIVE_DESK/Projects/desk-standard-desk`; the off-repo decision spine it
inherits from lives in `…/Projects/dotfiles-agents-desk/_structure/decisions/` (formerly
`dev-tooling-desk`, whose stale copy sits in `…/Projects/ARCHIVE/dev-tooling-desk-old/` — don't
read the archive copy as current).

## 1. Current standing (as of 2026-07-22) + what's next

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

**NEXT, in order of consequence:**
1. **#197** (gate:v2-final, filed 2026-07-22 by the wave-v4 model-simulations crew) — reconcile
   the software-spec phase-machine with the PM item phase-machine and name the
   building→shipped gate rule. The one open item blocking full schema-v2 epic (#130) closure.
2. **1.0.0 maturity call (#87)** — owner's call, now better-informed: the value-evaluation
   verdict (wins all 3 rubric dimensions vs both baselines; honest caveats) is on file at
   `_meta/research/2026-07-cohesion-value-evaluation/`, and the model-simulations deficiency
   report (v1 + v2, both models walked) is complete at `_meta/research/model-simulations/`.
3. **#199** (filed 2026-07-22) — `docs/development/ts-proxy-design.md` still cites the retired
   `plugin/desk-pm/` path (folded into desk-persona in #180, then moved under `plugins/` in
   #191); needs the ts-proxy design owner's judgment, not a mechanical rename.
4. **ts-proxy implementation** — `docs/development/ts-proxy-design.md` §5; slice 0 (host
   spawn-capability probe) is the go/no-go; blocked behind #199's doc correction.

**Carried-over open question** (2026-07-20, still unresolved): despite
`enabledPlugins["desk-standard@desk-standard"]: true`, the plugin's 4 MCP tools would not
surface as agent-callable in a live session in this repo — needs a from-scratch session test.

**Standing, settled decisions:** public launch deferred until ≥ v1.0.0 (Henry, 2026-07-19; repo
stays PRIVATE; the public `curl|bash` 404 is by-design — §3 has the authed-`gh` workaround;
don't re-surface as a blocker). OpenCode #12 parked (Henry, 2026-07-17) until ≥ v1.0.0.
#36 SOP library deferred post-1.0 (owner sign-off, 2026-07-21).

**Open backlog: the live triaged view is pinned issue #155** — refresh it (edit the body via
`gh issue edit 155 --body-file`, bump its date) at session boundaries alongside this file; it
lives as an issue precisely so refreshes never touch commit history.

## 2. Delivery eras (newest first — blow-by-blow lives in #155's body, the PRs, and this file's git history)

- **2026-07-22 · wave-v4** — 7 PRs (#192–#196, #198, #200); closed #124 #125 #126 #181 #182 #184
  #185 #187 #189 #190; schema-v2 arc closed for this cycle (contract versioning, reviewed v2
  element model, v1+v2 model simulations both walked), TUI discoverability (tab strip, footer,
  help overlay) + user/dev docs split, 3 findings-driven librarian bugfixes, tool-surface label
  cleanup, reconciliation pass. Filed #197, #199. One collision (wave 6 vs. wave 5 on
  `docs/getting-started.md`, predicted in-brief) resolved directly by the coordinating session.
- **2026-07-21 · wave-v3 + sign-off + v0.8.0** — 8 PRs (#176–#180, #183, #186, #188); closed
  #19 #81 #83 #84 #88 #170 #171; browser session surface, TUI archive lifecycle, PM default-on,
  desk-pm folded into desk-persona, K24 retrofit defaults ruled, `make e2e` gate (45 checks),
  v1 model sims + deficiency report, repo-root `_knowledge/` removed; v0.8.0 released.
- **2026-07-21 · wave-v2 + triage** — PRs #172–#175; closed #82 #85 #163–#169; gofmt/query-kind
  gates, verify.sh 48→55, PM polish (unset-vs-empty = presence-not-value), profile-root
  constant + drift guard; Dependabot action majors merged.
- **2026-07-21 · wave-v1** — PRs #156–#162; ~20 issues closed; Go module rename, CI hardening
  lanes, transcript-flush fix, TUI sessions surface, PM completion (ADR 0019/0020), content
  indexing + search/content query kinds.
- **2026-07-20/21 · integration-testing wave** — PR #152 (4 bug fixes); report at
  `_meta/research/2026-07-20-integration-agent-led-testing/`; manual dogfood scripts on main
  (deliberately NOT in CI).
- **2026-07-20 · v1 build wave + design session** — epic #129 closed (PRs #136–#147); ADRs
  0009–0018 ruled (PR #113); 0.8.0 bug floor (PR #112); Lane 6 conformance (PR #105).
- **2026-07-16→19 · v0.4.0→v0.7.0 era** — ADRs 0001–0007; chat TUI; XDG store; store self-init;
  rename `pocket-librarian` → `deskkit`; PM system shipped dark at v0.7.0.

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
    "continue to completion" + restate the checklist. Never re-brief. A lead's worker
    notifications route to the ROOT session — relay them; tell leads to verify worker state
    themselves.
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
- **Docs layout**: Using vs Development tracks per `docs/README.md`; spec + ADRs deliberately
  stay at `docs/` top level (cited from code/skills/neutrality allowlist — don't move). VHS
  tapes in `docs/development/tapes/`, GIFs land in `docs/media/`; re-recording re-encodes ALL
  GIFs (`git restore` the unchanged ones); tapes need
  `ANTHROPIC_API_KEY="$(secret get ANTHROPIC_API_KEY)" bash scripts/record-media.sh`.
- **Commits auto-close**: `Resolves #N` closes when the commit lands on `main` — post the proof
  comment first (close-with-comment fails on an already-closed issue).
- **Pre-staged files**: a parallel agent may `git add` before you commit; `git commit` with a
  pathspec still sweeps the whole index — check `git status` before every commit.
- **Worktree provisioning**: untracked-and-load-bearing = `plugin/node_modules` (run
  `bun install --frozen-lockfile`) and `.claude/agent-memory/`. Everything else is committed.
- **A persistent 404 on release assets the authed API can see** = check repo visibility
  (`gh repo view`) before blaming CDN propagation.

## 4. Incident log

Durable lessons are promoted into §3; the dated blow-by-blow entries (2026-07-17 → 2026-07-21,
thirteen incidents across four foreman runs) live in this file's git history (versions at and
before commit `c6d1655`) and in the PRs they cite. Add new incidents here dated; promote and
prune them at the next distillation pass.

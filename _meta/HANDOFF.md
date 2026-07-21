_Session-to-session bridge for desk-standard. Read this before working; update it at the end
of any substantial session. Secret-free — live URLs/credentials belong in `_meta/operations/`
(untracked) if that dir is ever created._
Status: active (2026-07-21)

# HANDOFF

## 0. Orientation

Two products over one shared schema, all identity-neutral (nothing shipped carries a
person/org/repo/issue): **`plugin/`** (harness-pure TS core + stdio MCP server, wrapped as a
Claude Code plugin with four skills) and **`librarian/`** (deskkit: single Go binary, embedded
PocketBase, eino agent loop, record-original-first write boundary), with **`schema/`** as the
contract both read. Personalization is only via `_knowledge/profile.yaml`. Root `README.md` is
the front door; `CLAUDE.md` is the agent digest (hot-loaded every session — don't duplicate its
content here) and `docs/CHARTER.md` the canonical page (precedence rule); `docs/README.md`
indexes the docs (Using ┃ Development); `docs/pocket-librarian-v1-spec.md` is the librarian's
build spec (operator docs are `librarian/README.md`).

Siblings: `hsb3/dotfiles-agents` (pattern source for workflows/templates). The **paired
executive desk** is `~/Documents/EXECUTIVE_DESK/Projects/desk-standard-desk` (dedicated to this
product since 2026-07-19): the 1.0.0 roadmap/design-review analyses, plans, briefings, staged
issue bodies, and the friction ledger live there.

## 1. Current standing + top priority

**SESSION 2026-07-20 (integration testing + bug-fix wave) — 4 real bugs found by real agent-led
testing, all fixed; PR #152 MERGED 2026-07-21 (squash `bbfc1a8`, closed #148–#151, main CI
green post-merge).** Henry asked for evidence the v1 build
wave actually works when driven by an agent, not just unit tests + `librarian/verify.sh`.
Foreman-led: 3 parallel testers (plugin MCP + skills; librarian's real `deskkit agent` LLM loop
+ a live MCP protocol session; the PM module's real work-graph workflow) → 2 adversarial
reviewers (9 claims re-derived from scratch: 8 confirmed, 1 plausible, 0 refuted) → filed
**#148** (high — `deskkit` built/run from a `$TMPDIR` path, e.g. `mktemp -d`, silently triggers
PocketBase dev-mode and dumps raw SQL to stdout, corrupting an MCP client's JSON-RPC stream),
**#149** (medium — the real agent loop could drop a state-mutating tool call from its own
`messages` transcript when it hit `AGENT_MAX_STEP` right after that call — an audit-trail
integrity gap), **#150** (low — `pm --actor`'s help placement implied pre-leaf usage that
actually failed; real root cause was `main()`'s pre-cobra unknown-command guard, not cobra
config), **#151** (low — `conventions-standard`'s frontmatter-exemption prose omitted
`_knowledge/`, which the shipped scaffold ships unfrontmattered even though the machine gate
already exempts it by basename). Full report:
`_meta/research/2026-07-20-integration-agent-led-testing/report.md`. **Open question, not filed
as a bug** (unclear if it's a desk-standard defect or Claude-Code-harness/session-timing
behavior): despite `enabledPlugins["desk-standard@desk-standard"]: true`, no tester or reviewer
could get the plugin's 4 MCP tools to surface as agent-callable in a live session working in
this repo — needs a from-scratch session test to pin down. Three reusable scripts landed on
`main` (committed `41357e2`, all pass neutrality + shellcheck): `plugin/scripts/verify-mcp-
protocol.sh` (deterministic), `librarian/dogfood-agent.sh` (manual, real-LLM, NOT in CI),
`librarian/dogfood-pm.sh` (deterministic) — none wired into `make check`/`make verify` yet
(deliberate). All four bugs then fixed on **PR #152** (branch `fix/agent-led-test-findings`,
commits `26b4495`+`79daa00`, squash-merged as `bbfc1a8` closing #148–#151): flat fan-out (2 sonnet builders on the
bounded fixes, 1 opus on the #149 persistence fix given the risk of touching agent-transcript
semantics), each fix with a red-before/green-after regression test, foreman reconciliation
caught a builder-left compile break + 11 neutrality-lint violations (bare issue numbers in new
Go test comments — a brief-writing miss, see incident log), full gates green (`make check` /
`make test` / `make verify`), `claude-review`'s 6 comments addressed (one suggested
"simplification" was empirically proven wrong before declining it — see incident log). Follow-up
**#153** filed for the two gaps review surfaced but PR #152 deliberately left out of scope
(`Session.StreamTurn()` has the same MaxStep transcript gap; a non-MaxStep abort right after a
tool call is still unflushed). **MERGE COMPLETED 2026-07-21** (Henry's go-ahead): the bot's
second review pass (ran against head `79daa00`) came back all-confirmations with zero blockers;
its one non-blocking cleanup idea was filed as **#154** (`printJSON` hardwires `os.Stdout`,
forcing the pipe-swap capture pattern in `pm` command tests — give it an `io.Writer` /
`cmd.OutOrStdout()`); all 12 review threads were replied-to and resolved; closing keywords were
added to the PR body pre-merge (the body originally had NO `closes #N` keywords — the handoff's
"closes on merge" claim would silently not have happened); squash-merged as `bbfc1a8`, branch
deleted, #148–#151 auto-closed, main CI green post-merge.

**v1 build wave landed 2026-07-20 — epic #129 closed, `gate:1.0.0` label query empty.** All ten
children (#114–#123) shipped as PRs #136–#147 (main HEAD `cd22b9f` pre-this-session); `make
verify` 48/48. Highlights live in CLAUDE.md (guard families, tool-surface counts, desk-persona
bundle) — not repeated here.

**NEXT, in order of consequence:**
1. **Owner ruling: fold `desk-pm` into `desk-persona`, or ship both?** Deliberately additive for
   now (duplicate `pm-operator` agent name if both installed; PR #143's body has the facets). A
   fold is a small follow-up PR.
2. **Release cut (owner-gated)** — `[Unreleased]` holds the whole v1 wave + the 0.8.0 bug floor
   + the #152 fixes; `make version-status` will shout. Runbook `docs/development/releasing.md`.
3. **Epic #130 (schema-v2, #124–#128)** — its own arc, deliberately no milestone; v1+v2 model
   simulations before v2 finalizes (owner's signoff note).
4. **ts-proxy implementation** — `docs/development/ts-proxy-design.md` §5 slices; slice 0 (host
   spawn-capability probe) is the go/no-go before anything else.

**Design session (2026-07-20) is RULED — ADRs 0009–0018 bind** (`docs/decisions/README.md` has
the index). The build plan derived from those ADRs is live as epic #129 (v1, closed above) +
epic #130 (schema-v2, no milestone). The full process record (decision book, dossiers,
platform-stream reconciliation, sign-off trails) is historical reference at `_meta/research/
2026-07-design-session/` and `_meta/signoff/` — the ADRs are what binds, don't re-derive from
process docs. Session record: PR #113 (rulings) → PR #131 (build plan, scaffolded
`_meta/plans/`). **Decision packages for Henry are always non-markdown decks/PDFs via the
owner-signoff HTML form, never chat questions or terminal markdown** (project memory).

**Standing, settled decisions:** public launch deferred until ≥ v1.0.0 (Henry, 2026-07-19; repo
stays PRIVATE, the `curl|bash` install-path 404 is by-design, not a bug — §4 has the authed-`gh`
workaround; don't re-surface as a blocker); going public AND OpenCode #12 are parked (Henry,
2026-07-17) until ≥ v1.0.0 — focus is dogfooding shipped Claude Code, buildable follow-ups
(#34/#35) can proceed.

**Open backlog, ranked** (no ruling gates the buildable ones): **#12** dual-format
Claude+OpenCode fan-out (architectural, needs an OpenCode-target ruling, ON HOLD above) ·
**#34** CI hardening — shellcheck/actionlint/SHA-pin-drift exist locally, not yet in the CI
pipeline · **#35** coverage — `requireConfig` self-init for non-`query` commands + a behavioral
test for the TS MCP server · **#19** PB-served webapp chat (deferred; ADR 0001 option b —
`StreamTurn`'s JSON-taggable events already give it a substrate) · **#36** template SOP library
v-next major, needs a design ruling first (vendor-vs-sync, type↔dir_kind) — the "librarian
knows ~20 SOPs" premise never existed, the 2-template boundary was intentional · **#153** the
two transcript-gap follow-ups from the bug-fix wave (above) · **#154** test-hygiene cleanup
(`printJSON` → `io.Writer`, drops the stdout pipe-swap in `pm` tests; filed from #152 review).

## 2. Recent deliveries (newest first — full detail in the cited PRs / ADRs / issues)

- **2026-07-21 — PR #152 merged** (`bbfc1a8`, closed #148–#151); review threads dispositioned,
  #154 filed. See §1 top entry.
- **2026-07-20 — agent-led integration testing + bug-fix wave.** See §1 top entry. Report +
  PR #152 + issues #153/#154.
- **2026-07-20 — v1 build wave, epic #129 closed.** PRs #136–#147; see §1 + CLAUDE.md.
- **2026-07-20 — design session ruled + Phase 4 filed.** ADRs 0009–0018; PRs #113, #131; see §1.
- **2026-07-20 — 0.8.0 bug floor merged.** PR #112 (`51235f6`): closed #67/#78/#79/#80/#91/#92/
  #93/#94/#102, each with a red-able regression test; three data-safety slices independently
  adversarially reviewed, 0 defects. `VERSION` stayed 0.7.0 (release cut is separate/owner-gated).
- **2026-07-20 — Lane 6 conformance + exec-desk split.** PR #105 (`217b5e6`, closes #86):
  mise-en-place scaffold, repo-compliance-audit 70/0 (was 55/14); exec desk `desk-standard-desk`
  bootstrapped; issues #91–#104 filed, 0.8.0 milestone created.
- **2026-07-19 — v0.7.0 released** (PM system ship-dark; CLI renamed `pocket-librarian` →
  `deskkit`) and **v0.6.0 era collapsed** (PRs #37–#61: chat full-screen TUI/ADR 0004, Charm v2
  migration/ADR 0007, `make install`, `record_feedback`, docs dev/use split). Durable rules in
  §4; incidents in §5.
- **2026-07-16/17 — v0.4.0 build-out.** ADR 0001 (chat REPL + trigger layer), ADR 0002 (XDG
  store home + desk open-guard), ADR 0003 (store self-init), `install.sh` + SHA-pinned Actions,
  first release v0.4.0.

## 3. Where to start next

- **The design session is RULED and Phase 4 is FILED** (§1): ADRs 0009–0018 bind; pick up #114
  first (blocks #119/#122) if opening a new v1 build lane, then the independent slices.
- **Cut the next release** when `[Unreleased]` warrants — `docs/development/releasing.md`
  (bump VERSION + 3 manifests → dated CHANGELOG section → `make release-prep` → tag).
- **Repo-visibility decision** (§1) gates the public `curl|bash` path — Henry's call, on hold.
- **#19 webapp**: substrate ready (`StreamTurn`'s JSON-taggable events marshal onto a
  PB-served SSE route with no new frontend toolchain).

## 4. Conventions & gotchas

_Core coding conventions — gates, generated artifacts, PocketBase-bootstraps-before-cobra,
migration patterns, config resolution, store self-init — are in **CLAUDE.md** (hot-loaded every
session). Only what CLAUDE.md doesn't cover lives below._

- **Installing from the private repo** (until public launch ≥ v1.0.0): the public `install.sh` /
  `curl|bash` path 404s by design — use authed `gh` instead.
  - From a local clone: `make install` (root or `librarian/`) → `~/.local/bin/deskkit`.
  - Straight from a release: `gh release download v0.7.0 --repo hsb3/desk-standard --pattern
    "deskkit_*_$(uname -s|tr '[:upper:]' '[:lower:]')_$(uname -m|sed 's/x86_64/amd64/;s/aarch64/arm64/')"
    --output ~/.local/bin/deskkit --clobber && chmod +x ~/.local/bin/deskkit` (verify against
    the release's `checksums.txt`).
- **Docs layout**: `docs/README.md` indexes two tracks — **Using**
  (`getting-started`/`plugin-guide`/`librarian-guide` + `media/*.gif`) and **Development**
  (`docs/development/` = overview + `releasing.md` + `tapes/*.tape`). The spec and ADRs
  deliberately stay at `docs/` top level (cited from code/skills/neutrality allowlist — don't
  move them). VHS tape sources live in `docs/development/tapes/`; `.gif` output lands in
  `docs/media/` (repo-root-relative `Output` path, survives tape moves).
- **Commits auto-close**: `Resolves #N` in a commit message closes the issue when that commit
  lands on `main` (push or merge) — post the proof comment first, or `gh issue comment` after
  (close-with-comment fails on an already-closed issue).
- **Beware pre-staged files in multi-agent waves**: a parallel agent may `git add` its own scope
  before you commit; `git commit` with a pathspec still sweeps the whole index. Check `git
  status` before every commit — reconfirmed this session (3 builders shared one tree by
  disjoint file scope, not worktree isolation).
- **eino streaming gotchas** (bind any new streaming caller to these): agent output stream is
  bursty under the anthropic tool-call checker (use model-callback stream copies for live
  tokens); ctx-cancel doesn't abort a stuck provider stream; failed tools fire `OnError` only;
  zero-arg tool calls need the `argNormalizingTool` adapter. **No terminal queries after
  bubbletea starts** (leaks into the textarea; regression-guarded in
  `internal/modules/librarian/tui/defects_test.go`). Chat theme resolves ONCE pre-program
  (`tui.ResolveTheme`); new TUI colors go in `newStyles`'s per-theme switch, never AdaptiveColor.
- **VHS chat tapes** (`chat.tape` dark + `chat-light.tape` light) need `ANTHROPIC_API_KEY`:
  `ANTHROPIC_API_KEY="$(secret get ANTHROPIC_API_KEY)" bash scripts/record-media.sh`.
  Re-recording re-encodes ALL GIFs; `git restore` the ones whose tapes didn't change.
- **Worktree provisioning**: untracked-and-load-bearing = `plugin/node_modules` (run `bun
  install --frozen-lockfile`) and `.claude/agent-memory/` (machine-local). Everything else a
  worktree agent needs is committed.

## 5. Incident log

- 2026-07-20 (bug-fix wave): background builder agents were interrupted mid-response/mid-edit
  by connection errors 3 times; a resumed builder's self-reported "done" was wrong once — it
  left `pm_test.go` missing 3 imports (a non-compiling state) that only surfaced when the
  foreman ran `go build`/`go vet` directly instead of trusting the report. Lesson: after
  resuming an interrupted agent, verify the build/tests yourself before trusting its summary —
  an interruption mid-edit can leave incoherent file state the agent's own confidence won't flag.
- 2026-07-20 (bug-fix wave): builder briefs for 3 parallel Go fixes didn't warn about the
  neutrality lint's bare-issue-number trap (CLAUDE.md's own "classic trip") — new test comments
  referencing "issue #150"/"#67" tripped `make check` post-hoc, caught only at foreman
  reconciliation. Lesson: any brief that will produce Go comments/tests under `librarian/` must
  explicitly restate this gotcha; don't assume a fresh builder read CLAUDE.md's fine print.
- 2026-07-20 (bug-fix wave): a `claude-review` suggestion ("simplify `groupCommandValueFlags` to
  `c.Flags()` alone") was empirically wrong — applying it would have silently reintroduced #150,
  since cobra only merges a command's own persistent flags into `Flags()` during `Execute()`,
  which runs AFTER this code's call site in `main()`. Caught by writing a throwaway probe test
  before applying the suggestion, not by inspection alone. Lesson: verify an AI review's
  "cleaner code" suggestion empirically, especially anything touching ordering/timing — a
  plausible-sounding simplification can be a regression.
- 2026-07-20 (build wave): a read-only scout REFUSED its brief because the main tree
  contradicted it — root cause: `git fetch` after merges but no `git pull` of local `main`, so
  the working tree was three merges stale. In multi-worktree waves, `git pull` local main before
  any agent reads the main tree (worktrees fork from commits, so they're unaffected).
- 2026-07-20 (build wave): `gh pr merge` failed on a conflicting PR but the `;`-chained cleanup
  (worktree remove + branch delete) ran anyway — recovery needed a re-checkout from the remote
  branch. Chain merge→cleanup with `&&`, never `;`.
- 2026-07-17: first real release (v0.4.0) surfaced that the repo is private — every
  unauthenticated install-flow URL 404s. Root-caused via `gh repo view` visibility, not a CDN
  lag. Lesson: a persistent (not transient) 404 on release assets the authed API can see = check
  repo visibility before blaming propagation.
- 2026-07-18: the `pocket-librarian` on PATH was a stale build — `/plugin` updates the plugin,
  NOT the standalone binary (separate artifacts). Lesson: after a release, update the binary
  separately (`make install` or `gh release download`).
- 2026-07-17: a gate command piped to `tail` masked neutrality's exit 1 (pipeline status is the
  LAST command's) — a commit landed locally with lint violations, caught on the next standalone
  run and amended before push. Lesson: when a command GATES a commit, run it bare and check its
  own exit code — never pipe it.

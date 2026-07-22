_Session-to-session bridge for desk-standard. Read this before working; update it at the end
of any substantial session. Secret-free — live URLs/credentials belong in `_meta/operations/`
(untracked) if that dir is ever created._
Status: active (2026-07-21, post wave-v3 + owner sign-off run)

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

**SESSION 2026-07-21 latest (wave-v3 + owner sign-off) — the nine-item owner decision queue was
batched into ONE signoff form, ALL recommendations approved, and everything executable was
executed same-session: 7 PRs merged (#176–#180, #183, #186), 5 issues closed (#19 #81 #83 #88
#171), 5 filed (#181 #182 #184 #185 #187), release 0.8.0 STAGED (release-prep + CI
green — awaiting only the owner's `git tag v0.8.0 && git push --tags`), main at `340a480`.**
Fable-led foreman session; 7 worktree-isolated crews, every branch gate-verified by the session
before landing; final union green incl. the NEW `make e2e` gate (45 checks). Delivered:
- **#177** browser session surface `/desk/chat` (closed #19) — same agent session + gated
  registry as the REPL, SSE over StreamTurn events, ≤40 bound, loopback-origin guard.
- **#176** TUI polish (closed #171) — themed picker, resume-first, archive lifecycle
  (migration 0022).
- **#178** records (closed #81) — **ADR 0020 owner-confirmed** (supersession window closed),
  **ADR 0021 graduates the off-repo "decision 0021"** (the dangling F-fork citations across
  #83/#84/#88/#170 + this file are resolved by that ADR), K24 retrofit defaults ruled into
  conventions-standard + brownfield-adoption.
- **#179** PM default-on (closed #83) — env > profile > default-ON three-state; ADR 0008
  amended in place; all docs at the on-by-default posture.
- **#180** desk-pm folded into desk-persona (hook + 3 skills ported; bundle retired;
  version-sync set 7→5; persona-drift now generates librarian-operator only).
- **#183** #126 v1-half sims — 4 walkthroughs + scripted probes, deficiency report (pass bar
  met): D1→#184, D2→#185 filed; stale data-model dossier corrected in place (O1–O3).
- **#186** Lane 8 (closed #88) — `make e2e` suite + cohesion assessment + value evaluation
  under the owner-approved scoping (markdown baseline, Notion comparator, 3-dim rubric):
  **wins all three dimensions**, honest caveats recorded in
  `_meta/research/2026-07-cohesion-value-evaluation/value-evaluation.md`.
The signoff trail is `_meta/signoff/2026-07-21-decision-queue/` (form + answers.json). The ONE
unresolved sign-off field: the `_knowledge/` move TARGET NAME (#170 — approved in principle,
name never supplied).

**PREVIOUS SESSION 2026-07-21 late (triage + wave-v2 run) — pinned #155 refreshed with delivery plan v2
and the plan executed the same session: 4 PRs (#172–#175) merged, 9 issues closed (#82 #85
#163–#169), #84's first half shipped, main at `eca19d6`, `make check`/`make test`/`make verify`
(55/55) green on the final union, main CI green.** Fable-led foreman session; 4 worktree-isolated
opus-lead crews (each spawning its own builders), every branch gate-verified by the session
before landing:
- **#172** docs-accuracy (closed epic #85) — spec retitled `deskkit` + `Status: active`
  (file path unchanged — load-bearing), getting-started leads with the authed `gh release
  download` path, stale-ref sweep with a justified-survivals table.
- **#174** hygiene gates (closed #163–#166) — gofmt gate (`make -C librarian fmt`, CI librarian
  lane), dogfood-*.sh shellcheck-clean + in the lint set, verify.sh **48 → 55 checks**
  (search/content/orphans coverage), query-kind drift guard `check-query-kinds.mjs` (+self-test).
- **#175** PM polish (closed #167–#169) — manifest `body` round-trip; **unset-vs-empty ruled as
  presence-not-value** (omission/`null` = unchanged; present `""`/`--body ""` = clear; MCP
  optional fields → `*string`, CLI keys off `Flags().Changed()`; documented in pm-guide); spec
  §12 rows for ADR 0020 + body.
- **#173** profile-root constant (**first half of #84**) — canonical `schema/paths.yaml`
  `profile_root:`, lane constants `PROFILE_ROOT_DIR` (TS) / `ProfileRootDir` (Go), drift guard
  `check-profile-root.mjs` (+self-test) in `make check` + CI, root `_knowledge/README.md`.
  **The move half is gated by #170** (triage found the epic's "decision 0021 F5" provenance
  exists nowhere in-repo — the move TARGET was never ruled; move-day site checklist is a comment
  on #84). Also merged: the 4 Dependabot action-major PRs (#106 #108 #109 #110 — checkout v7,
  setup-node v7, upload-artifact v7, claude-code-action 1.0.171; their `claude-review` failures
  are dependabot-can't-read-secrets, and it's not a required check). 9 new issues filed at
  triage (#163–#171); #163–#169 closed same-day, #170 (owner) + #171 (TUI backlog) remain.

**PREVIOUS SESSION 2026-07-21 (delivery-wave run) — the ENTIRE #155 delivery plan v1, waves 0–3,
executed and merged: 7 PRs (#156–#162), 20 issues closed, all gates green.**
Fable-led foreman session; each branch built by an isolated worktree crew (builders or
lead-driven teams), every branch gate-verified independently by the session before landing,
every claude-review thread dispositioned (applied or declined-with-reasoning):
- **#156** wave 0 — Go module renamed to `github.com/hsb3/desk-standard/librarian` (#98);
  token-scoped neutrality-allowlist entries per the owner ruling on the issue.
- **#157** docs — ADR 0006 kit-consumption correction (erratum protocol) + **ADR 0019** durable
  PM defaults (#63, #103).
- **#158** hardening — shellcheck/actionlint/SHA-pin-drift enforced in CI (+ release gate
  mirror), schema-copy drift test, secrets-shaped-field guard, requireConfig self-init + TS MCP
  behavioral tests (#34, #97, #74, #35); action majors bumped at landing with API-resolved SHAs.
- **#159** — transcript flush on ANY agent-loop abort, both Run() and StreamTurn() (#153);
  adversarial reviewer re-derived all 8 claims (0 refuted; red-before proven via `go test
  -overlay`).
- **#160** — TUI sessions surface / token accounting / $EDITOR hatch (#51, #52, #64); the
  cross-crew `agent/stream.go` collision with #159 was merged by the session as a union and
  race-verified.
- **#161** — PM completion (#90 body field via migration 0006, #96 claim semantics, #68 bounded
  ordered realtime, #154 printJSON io.Writer, #95 confirmed already-shipped-by-#66 + docs,
  #101 gating contract). **ADR 0020 (authoritative claims) ACCEPTED at PR merge** per the
  delivery plan's "PR review is the ruling" mechanism — Henry can supersede; the ADR lists the
  exact revert set for the advisory reading.
- **#162** wave 3 — content indexing + `query search`/`content` kinds on CLI+MCP (no tool-count
  changes), orphans de-noise (`--show-index`), R6 self-clear via handoff-excluding git pathspec
  (#89, #100); sweep result now carries a `truncated` counter (no silent caps).

**Wave 4 (schema-v2, epic #130) deliberately NOT started**: its own gate #126 (v1+v2 model
simulations) is an owner-signoff, desk-side condition — starting #124/#125 before that ruling
risks rework. The run record + accumulated small-backlog list live in **issue #155's body**
("Delivery plan — EXECUTED 2026-07-21").

**Carried-over open question** (from the 2026-07-20 session, still unresolved): despite
`enabledPlugins["desk-standard@desk-standard"]: true`, the plugin's 4 MCP tools would not
surface as agent-callable in a live session in this repo — needs a from-scratch session test.

**v1 build wave landed 2026-07-20 — epic #129 closed, `gate:1.0.0` label query empty.** All ten
children (#114–#123) shipped as PRs #136–#147 (main HEAD `cd22b9f` pre-this-session); `make
verify` 48/48. Highlights live in CLAUDE.md (guard families, tool-surface counts, desk-persona
bundle) — not repeated here.

**NEXT, in order of consequence:**
1. **Owner: tag 0.8.0** — everything is staged and green; the go/no-go is
   `git tag v0.8.0 && git push --tags` (then update the PATH binary separately — `make install`
   or `gh release download`).
2. **Owner: name the `_knowledge/` move target** (#170) — the one blank sign-off field; a
   one-line answer unblocks #84's second half (constant + drift guard + move-day checklist all
   ready).
3. **Schema-v2 arc (#130) — now UNBLOCKED, next session's work**: #124 (version the schema
   contract) → #125 (element model revision under ADR 0018; the v1 sims walkthroughs carry a
   ready `v2 (deferred)` column) → #126 v2 half. Design-heavy; deliberately not started at this
   session's depth.
4. **Small buildables from this run's findings**: #184 (finding id missing from query surfaces),
   #185 (update_item type-validation gate bypass), #182 (subprocess-test store collision),
   #181 + #187 (coupled label/doc passes — one PR could take both).
5. **1.0.0 gate after the above**: only #84's move half (#170) and the #87 release-cut epic
   itself remain on the milestone; the value-evaluation verdict (wins all 3 dimensions) is on
   file for the 1.0 claim.
6. **ts-proxy implementation** — `docs/development/ts-proxy-design.md` §5 slices; slice 0 (host
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

**Open backlog: the live triaged view is pinned issue #155** ("meta: triaged open-issue
backlog") — ranked Now / 1.0.0 lanes / schema-v2 / backlog / on-hold, with a maintenance
protocol in its body. Refresh it (edit the issue body via `gh issue edit 155 --body-file`,
bump its date) at session boundaries alongside this handoff — it lives as an issue, not a
tracked file, precisely so refreshes never touch commit history. Headline ranking as of
2026-07-21: Now = #153 transcript-gap follow-ups · #34 CI hardening · #35 coverage · #154
test-hygiene; #12 stays ON HOLD (owner ruling above); #36 needs a design ruling first.

## 2. Recent deliveries (newest first — full detail in the cited PRs / ADRs / issues)

- **2026-07-21 latest — wave-v3 + owner sign-off run: PRs #176–#180, #183, #186 merged; #19 #81
  #83 #88 #171 closed; ADRs 0020 confirmed / 0021 graduated / 0008 amended; PM default-on;
  desk-pm folded; `make e2e` gate added; 0.8.0 staged awaiting the owner tag.** See §1 top entry
  and #155's "Delivery plan v3 — EXECUTED" section.
- **2026-07-21 late — triage + wave-v2 run: PRs #172–#175 merged, 9 issues closed, 9 filed,
  Dependabot majors merged, #155 refreshed twice.** See §1 top entry and #155's "Delivery plan
  v2 — EXECUTED" section.
- **2026-07-21 — delivery-wave run: PRs #156–#162 merged, 20 issues closed.** See §1 second entry
  and issue #155's "Delivery plan v1 — EXECUTED" section.
- **2026-07-21 — PR #152 merged** (`bbfc1a8`, closed #148–#151): the 2026-07-20 agent-led
  integration-testing wave's 4 bug fixes (#148 $TMPDIR dev-mode stdout corruption, #149 MaxStep
  transcript drop, #150 pre-cobra flag guard, #151 skill prose); 12 review threads
  dispositioned, #153/#154 filed as follow-ups (both closed by this session's run). Test report:
  `_meta/research/2026-07-20-integration-agent-led-testing/report.md`. Reusable scripts on main
  (`41357e2`): `plugin/scripts/verify-mcp-protocol.sh`, `librarian/dogfood-agent.sh` (manual,
  real-LLM, NOT in CI), `librarian/dogfood-pm.sh` — deliberately not wired into `make
  check`/`make verify`.
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

- **All delivery plans (v1, v2, v3) are EXECUTED** — the buildable backlog is the small-items
  list in #155's "Now" section (#181/#182/#184/#185/#187); the big arc is schema-v2 (§1 NEXT-3).
- **0.8.0 is staged** — only the owner tag remains (§1 NEXT-1). The next release after that cuts
  per `docs/development/releasing.md` (bump VERSION + 5 manifests — the guard is authoritative).
- **Repo-visibility decision** (§1) gates the public `curl|bash` path — Henry's call, on hold.

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

- 2026-07-21 (wave-v3 run): **an isolation-worktree agent got an ORPHAN directory** (not a
  registered git worktree) — its cwd silently resolved to the MAIN checkout, where it created
  and left the tree sitting on its research branch. Hazard realized-but-caught: the owner's
  staged `git tag v0.8.0` command would have tagged the wrong commit. Lessons: (a) after ANY
  crew reports, check `git branch --show-current` in the main tree before handing the owner
  ref-sensitive commands; (b) an agent reporting "worktree anomaly" means audit the main tree
  immediately.
- 2026-07-21 (wave-v3 run): **background crews stopped mid-task ~6 times** (ending a turn on
  "now let me…" instead of the deliverable). The nudge playbook worked every time: SendMessage
  the same agent "continue to completion without pausing between steps" + restate the remaining
  checklist. Treat mid-flight stop notifications as a normal beat, not failure; never re-brief.
- 2026-07-21 (wave-v3 run): the fixed-DESK_NAME subprocess-test store collision bit two crews
  (schema v21 binary vs v22-stamped shared store at `~/.local/share/deskkit/
  pm-actor-subprocess-test`) — playbook: delete the throwaway store, re-run; filed as #182.
- 2026-07-21 (wave-v2 run): the worker→root notification-routing quirk (next entry) recurred
  three times; the documented relay pattern (root forwards the worker report to the lead via
  SendMessage with "verify status yourself, don't wait") worked every time — treat it as the
  standing playbook, not an anomaly. Also reconfirmed: two crews wiring gates into the same
  root `Makefile`/`ci.yml` conflicted exactly as the collision-map lesson predicts; a lead-driven
  union rebase (keep both crews' wiring, re-run full gates, `--force-with-lease`) resolved it
  cleanly.
- 2026-07-21 (wave-v2 run): zsh nuance on the "never pipe a gate" rule — `${PIPESTATUS[0]}` is
  BASH; in zsh it's the lowercase `$pipestatus` array, so `${PIPESTATUS[0]:-$?}` silently falls
  back to the pipe-tail's exit code and can mask a failing gate while LOOKING careful. Redirect
  gate output to a file and check `$?` bare instead.
- 2026-07-21 (delivery-wave run): a worker/lead task-notification routing quirk — a lead's
  spawned worker's completion notification routes to the ROOT session, not the waiting lead, so
  a lead that stops "awaiting the notification" can orphan forever. Pattern that worked: the
  root session forwards the worker's report to the lead via SendMessage with "verify its status
  yourself; don't wait on notifications from me." Also: a process restart mid-run stops all
  live agents — resumed agents must re-verify tree state before trusting their transcript.
- 2026-07-21 (delivery-wave run): CI shellcheck flagged SC2317 on verify.sh's trap-invoked
  cleanup while the LOCAL shellcheck (older) flagged the same code as SC2319/SC2329 — version
  skew between Homebrew and the runner image means "clean locally" does not prove the CI gate;
  the disable comment now carries both codes.
- 2026-07-21 (delivery-wave run): the #155 collision map said the TUI branch was "tui/ pkg
  only", but #52's token accounting legitimately extended `agent/stream.go` — colliding with
  #159's rewrite of the same struct. Lesson: surface predictions in a delivery plan are
  hypotheses; the session must diff-check overlap before merging parallel branches (the union
  merge + race re-verification caught it cleanly here).
- 2026-07-21 (delivery-wave run): `make test` failed on the MAIN tree right after the last
  merge — stale `plugin/node_modules` (each worktree had installed its own; main's predated the
  new SDK import path). Re-`bun install` fixed it. Lesson: after a multi-worktree session,
  refresh the main tree's deps before reading a local gate failure as a broken main; the
  authoritative signal is main-branch CI.
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

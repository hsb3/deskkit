_Session-to-session bridge for desk-standard. Read this before working; update it at the end
of any substantial session. Secret-free — live URLs/credentials belong in `_meta/operations/`
(untracked) if that dir is ever created._
Status: active (2026-07-20)

# HANDOFF

## 0. Orientation

Two products over one shared schema, all identity-neutral (nothing shipped carries a
person/org/repo/issue): **`plugin/`** (harness-pure TS core + stdio MCP server, wrapped as a
Claude Code plugin with four skills) and **`librarian/`** (pocket-librarian: single Go
binary, embedded PocketBase, eino agent loop, record-original-first write boundary), with
**`schema/`** as the contract both read. Personalization is only via `_knowledge/profile.yaml`.
Root `README.md` is the front door; `CLAUDE.md` is the agent digest and `docs/CHARTER.md` the
canonical page (precedence rule — both since PR #105); `docs/README.md` indexes the docs (Using ┃
Development); `docs/pocket-librarian-v1-spec.md` is the librarian's spec (build spec, not operator
docs — operator docs are `librarian/README.md`). The build brief moved to the exec desk in PR #77.

Siblings: `hsb3/dotfiles-agents` (the pattern source for workflows/templates). The **paired
executive desk** is `~/Documents/EXECUTIVE_DESK/Projects/desk-standard-desk` (dedicated to this
product since the 2026-07-19 split): the 1.0.0 roadmap/design-review analyses, plans, briefings,
staged issue bodies, and the greenfield friction ledger live there. The off-repo decision
records the build brief cites (0013/0014/0015 — not 0016, which is dotfiles-agents — plus 0021,
the 1.0.0 direction) remain in `dev-tooling-desk`'s append-only spine, indexed from the new
desk's genesis record 0001.

## 1. Current standing + top priority

**SESSION 2026-07-20 (integration testing) — first agent-led integration pass across both
products; 4 real bugs filed, 1 open question flagged.** Henry noted the v1 build wave had no
integration tests or agent-led usage proof beyond unit tests + `librarian/verify.sh`'s scripted
CLI gate. Foreman-led: 3 parallel testers (plugin/ MCP + skills; librarian core's real
`deskkit agent` LLM loop + a live MCP protocol session; the PM module's real work-graph
workflow) → 2 independent adversarial reviewers (9 claims re-derived from scratch: 8
CONFIRMED, 1 PLAUSIBLE, 0 REFUTED). Full report: `_meta/research/2026-07-20-integration-
agent-led-testing/report.md`. **Filed as issues** (all independently confirmed):
**#148** (high — building/running `deskkit` from a `$TMPDIR` path, e.g. `mktemp -d`, silently
triggers PocketBase dev-mode and dumps raw SQL to stdout, corrupting a real MCP client's
JSON-RPC stream), **#149** (medium — the real agent loop can drop a state-mutating tool call
from its own transcript when it hits `AGENT_MAX_STEP` right after that call — an audit-trail
integrity gap, store `revisions` show the mutation, `messages` doesn't), **#150** (low —
`pm --actor` flag's help placement implies pre-leaf usage that fails), **#151** (low —
`conventions-standard`'s frontmatter-exemption prose omits `_knowledge/README.md`, which the
plugin's own shipped scaffold ships unfrontmattered — the machine gate is already correct,
only the prose checklist is stale). **Open question, NOT filed as a bug** (unclear if it's a
desk-standard defect or a Claude Code harness/session-timing behavior): despite
`.claude/settings.json`'s `enabledPlugins["desk-standard@desk-standard"]: true`, neither the
original tester nor either reviewer could get the plugin's 4 MCP tools to surface as
agent-callable in a live session working in this repo — needs a from-scratch session test to
pin down. Backlog candidates not yet filed (§7 of the report): a `propose_fix --run <id>`
scoping trap, `AGENT_MAX_STEP`'s tight default (12), LLM mechanical-vs-judgment
non-determinism across runs, two divergently-drifted `improvement-log.md` template copies.
Three reusable scripts left behind (all pass neutrality + shellcheck):
`plugin/scripts/verify-mcp-protocol.sh` (deterministic MCP-protocol harness),
`librarian/dogfood-agent.sh` (manual, real-LLM, NOT wired into CI — 19/19 passing),
`librarian/dogfood-pm.sh` (deterministic, 13/13 passing) — none yet wired into `make
check`/`make verify`, a deliberate follow-on decision. **This session's new files (the report
+ 3 scripts + this handoff edit) are UNCOMMITTED** — commit when Henry's ready.

**SESSION 2026-07-20 (build) — THE V1 BUILD WAVE IS LANDED; EPIC #129 IS CLOSED; THE
`gate:1.0.0` LABEL QUERY IS EMPTY.** All ten children (#114–#123) shipped same-day as PRs
#136–#145 plus the serial reconciliation pass #146 (main HEAD `cd22b9f`; `make verify` 48/48
run on it; the epic-close comment on #129 carries the full issue→PR→commit table and the
close-when walk). Highlights: the librarian store schema went 14→**20** (migrations 0015–0020,
serialized #118→#123 with landing-time renumbering per the epic rule); `make check` now runs
EIGHT guard families — the wave added prompt-drift (byte-exact embed↔spec), tool-surface
(+self-test, three-axis counts), persona-drift (generated bundle files), and textfield-max
(explicit-Max recurrence); `plugin/desk-persona/` is the composed librarian+PM bundle
(17-tool mount `MCP_MODULES=librarian,pm`, one authored source per surface, desk-pm
byte-untouched); `MCP_MODULES` module gating is live (unset=all; explicit-empty/unresolvable
= exit 1); the TS→deskkit proxy has a reviewed design doc (`docs/development/
ts-proxy-design.md`, verdict closable-as-is 26/26) with build slices 0–6 as the follow-on.
Foreman process: every lane got a distilled scout packet or lead-driven build, adversarial
review dispositions recorded on each PR, and layered verification (lead proof packages
spot-checked + foreman-run bare gates before every merge).

**NEXT, in order of consequence:**
1. **Owner ruling: fold `desk-pm` into `desk-persona`, or ship both?** Deliberately additive
   for now; facets in PR #143's body (duplicate `pm-operator` agent name if both installed;
   the SessionStart wake hook is NOT composed into the bundle). A fold is a small follow-up PR.
2. **Release cut (owner-gated)** — `[Unreleased]` now holds the entire wave + the 0.8.0 bug
   floor; `make version-status` will shout. Runbook `docs/development/releasing.md`.
3. **Epic #130 (schema-v2 track, #124–#128)** — its own arc, deliberately NO milestone; v1+v2
   model simulations before v2 finalizes (owner's signoff note).
4. **ts-proxy implementation** — design doc §5 slices; slice 0 (host spawn-capability probe)
   is the go/no-go before anything else.
5. Small ledger: PM-module Max-less fields (`notes.body`, `desk_config.rules`); guard-output
   polish notes recorded on PRs #143/#144.

The pre-build session below (planning wave, adversarial review, PR #131–#135) is now
history — its detail lives in those PRs and the epic-close comment:
- **#131 → `16e9259`** — the Phase-4 planning wave (epics #129/#130, children #114–#128).
- **#132 → `103021a`** — briefings relocated to the exec desk. **Briefings now live ONLY at
  `desk-standard-desk/_meta/briefings/`, never in the product repo** (the design-session
  decision deck + this session's EOD/status deck moved there; the repo copy was removed and the
  HANDOFF pointers repointed). The authoritative design records stay in-repo (ADRs,
  `_meta/research/`, `_meta/signoff/`).
- **#133 → `9c4b514`** — the design/planning corpus was **packaged**
  (`_meta/research/2026-07-design-session/design-planning-package.md`) and put through a
  **17-agent adversarial review** (6 dimension reviewers → per-finding source re-derivation →
  opus-xhigh synthesis): **GO-WITH-FIXES, 0 blockers** — 10 findings raised, all 10 confirmed
  against source, 0 refuted, 4 de-escalated. Report:
  `_meta/research/2026-07-design-session/adversarial-review-2026-07-20.md`.
- **#134 → `4f2a018`** — the two MAJORs + the ADR 0013 erratum applied: (1) #114 deliverable E
  now claims the three unclaimed TS plugin tools (`profile_get`/`profile_validate`/`knowledge_index`)
  so epic #129's close-when is satisfiable; (2) the **epic #129 coordination rule now names THREE
  shared surfaces** — the migration chain, the #114-C/#120 prompt-embed drift guard (**#114 lands
  before #120**; slice C also moves the mirrored spec block `pocket-librarian-v1-spec.md:1435-1464`
  + `:2316`), and `query.go`/`patrol.go` between #118/#123 — mirrored in the three plans; (3) ADR
  0013's "down-remap" corrected (a retirement remaps in the FORWARD migration). Live #129/#114
  bodies were updated to match. The **eight MINOR/NIT** review findings stay scoped to their owning
  slices (see the report).
- **Sibling tooling:** the planning-desk `_utils/` toolkit was hardened against its automated
  review (GraphQL variables, arg guards, saturation warnings, import robustness, `run()` rename)
  and the SAME fixes ported upstream to the skill source (**dotfiles-agents PR #150 → `dc00999`**
  on `dev`); **dotfiles-agents issue #151** filed (add CONTRIBUTING + a primitive-authoring skill).

**NEXT — open the build lanes.** #114 (agent-integration-contract) first: it blocks #119/#122.
Then the independent slices (#115/#116/#117/#118/#120/#121/#123). Honor the epic #129
coordination rules: #118+#123 migration commits serialize; **#114 lands before #120**;
#118/#123 rebase their shared `query.go`/`patrol.go` hunks. Epic #130 (schema-v2: #124–#128)
is its own arc, no milestone. Desk governance: staged bodies live in `_meta/plans/*/`, pushed
to live issues (epics have no `plan.md`, so `sync-bodies.py` skips them — push epics with
`gh issue edit`); gates are `conformance.py` / `reconcile.py` / `sync-bodies.py` under
`_meta/plans/_utils/`.

---

**v0.7.0 is released** (the PM system, ship-dark; adoption dry-run; PM spec/README reconcile)
and the CLI binary is **renamed `pocket-librarian` → `deskkit`** — release assets are
`deskkit_0.7.0_*`; `make install` drops `deskkit` into `~/.local/bin`. The stale
`pocket-librarian` 0.6.0 on PATH was removed 2026-07-19 (the 07-18 incident class — don't
resurrect it). **Lane 6 shipped** (PR #105 → `217b5e6`, epic #86 closed): repo-compliance-audit
**70 pass / 0 gap** (was 55/14); `CLAUDE.md` + `AGENTS.md` (symlink) + `docs/CHARTER.md`
authored; root `tests/` ruled a declared-home index (recorded in `_meta/mise-en-place.yml`).
**The deep-dive backlog is filed and triaged**: #91–#104 with milestones set (0.8.0 = bug floor
#67/#78/#79/#80 + #91/#92/#93/#102, #94 stretch — plan at the exec desk's
`_meta/plans/release-0.8.0.md`; the rest 1.0.0; onboarding epic #104 post-1.0); the #79
widening comment is posted; #81 rides the 1.0.0 milestone (R20 ruling). Cutting the next
release follows `docs/development/releasing.md`.

**Session 2026-07-20 — owner ruled the sequencing: bug floor → design session → features.**
After 0.8.0's bugs, a design session rules direction before any feature lane starts. Two
owner concerns were assessed against source and independently verified: (1) the librarian
and PM agent integrations are NOT mirror images — different kinds of objects on different
layers (in-binary Go eino loop vs plugin-markdown agent), with four unruled asymmetries
(no librarian Claude Code bundle; ride-along librarian tools on the PM MCP mount; the
in-binary loop gets PM tools with no PM prompt; split prompt governance); (2) the store
models documents as path-keyed frontmatter indexes only — no sub-file addressing, untyped
repo-unqualified cross-refs, an unwired `dismissed` disposition, no rename identity —
**#92/#93/#102 are symptoms of these gaps: fix them as modeling decisions, not spot
patches.** Evidence + the six-decision design-session agenda: exec desk
`analyses/desk-standard-agent-symmetry-and-document-model-2026-07-20.md` (§3). A
plain-language status briefing (15-slide PDF + MP3, comms standard) is at exec desk
`_meta/briefings/2026-07-20-status-and-direction/`. Owner preference (saved to project
memory): decision packages for Henry are non-markdown decks/PDFs, never terminal markdown.

**BUG FLOOR MERGED (2026-07-20) — PR #112 squash-merged to main as `51235f6`.** All nine
0.8.0 milestone issues shipped: #67/#78/#79/#80/#91/#92/#93/#102 + #94 (stretch, done); the
PR body's "Closes" auto-closed all nine. Each fix carries a **red-able regression test**
(#82's bar); the three data-safety slices (transactions #91, disposition #93, graduation
#78/#92) were each independently adversarially reviewed — 0 defects. The automated PR review
ran three rounds (all findings addressed, fixes folded into the squash): round 1 → dispose
lookup sorts `-patrol_run,-id`, `findings dispose --as` cobra-required, migration 0014
backfill paged (500/id-sorted), Verdict's not-found error hints `§` when a pointer carries a
`#` anchor (+ pinning test), `graduated to: <bare number>` confirmed spec-verbatim (§5.1/R5,
declined tightening, + pinning test); round 2 → engine.go doc comments (failpoint wired only
in transitionCore; SetStatusLabel's outer checkVersion guards the same-phase fast path;
AddNote keeps its tx for phase-snapshot consistency — refactor declined). Round-3 verdict
"clean and ready", two non-blocking notes NOT yet filed as issues: `universalFMKeys` omits
`status` deliberately (lightweight types) but says nothing — wants a one-liner comment; and
`cascade`/`tryAutoUnblock` use unbounded dependency queries (benign at desk scale). Full
gates green pre-merge (`make check` · `make test` bun 59 + go · `make verify` 48/48).
**VERSION stays 0.7.0** — the 0.8.0 release cut (bump + dated CHANGELOG + tag) is the separate,
owner-gated step (runbook `docs/development/releasing.md`); the changes sit under `[Unreleased]`.
**Next: #79's merge unblocks the PM default-on lane #83, and the design session rules feature
direction** (agenda on the exec desk — see the §1 sequencing note above/below). Non-blocking
follow-ups filed in the PR #112 body: the #79 not-on-PATH host-drop residual (unfixable in deskkit
— mitigated by the skill note + mount signal), a #93 `query summary`/`uncollapsed` count that
ignores disposition, `pm/engine/queries.go`'s still-non-atomic item-update (outside #91's scope),
and the neutrality scanner walking `.claude/` scratch (gitignore hardened for nested agent-memory
this session).

**DESIGN-SESSION PREP IN FLIGHT (2026-07-20, post-merge).** Owner ruled: no 0.8.0 release cut
(sole user); proceed straight to design-session prep. The prep doc — design lens (data model +
workflows are the core; MCP/CLI/skills/personas/TUI/human are projections), 5-phase process
outline (Ground → Frame → Decide → Record → Plan), known-changes table C1–C8, best-guess
choices, tough spots, constraint walls — is at
`_meta/research/2026-07-design-session/README.md`. **Phase 0 (Ground) is COMPLETE**: all
five evidence dossiers landed beside it (`data-model.md`, `workflows.md`, `surface-matrix.md`,
`agent-symmetry.md`, `document-model-gaps.md`), each cited path:line against merged main
`51235f6` with an explicit gaps section; the prep-doc §6 table carries per-dossier headlines.
Foreman-verified highlights: the exec-desk analysis
(`~/Documents/EXECUTIVE_DESK/Projects/desk-standard-desk/analyses/desk-standard-agent-symmetry-and-document-model-2026-07-20.md`)
survives PR #112 nearly intact (14/15 symmetry claims CONFIRMED; only #79's mount claim changed
— resolved fail-loud, not self-gating); gap D (#91 transactions) fully closed; A/B/C
symptom-closed but model-open; E/F untouched; NEW: unwired enums are broader than the analysis
knew (5 of 6 `adoption_log.event` values have no writer — even restore logs nothing;
independently re-derived), patrol has SIX rules (R1–R6), and unclaimed tools exist on BOTH
agent surfaces (stale librarian system prompt independent of PM; 3 of 4 TS tools skill-less;
PM `import` + admin console persona-less). **Phase 1 (Frame) is COMPLETE (2026-07-20, later session): the decision book is written and
reviewed.** Eight briefs + index at `_meta/research/2026-07-design-session/decision-book/`
(D1 pointer grammar · D2 typed cross-refs · D3 `items.type` · D4 disposition+adoption-log ·
D5 agent contract & parity · D6 prompt governance · D7 spec↔reality · D8 pull-only backlog),
dependency-ordered model → workflow → surface. Foreman-led: 8 parallel builders (one file
each), then two report-only reviews — (1) conformance/coverage: every evidence bullet
resolves to a real dossier heading, sampled cites lifted not invented, seed matrix complete
(agenda §3 + C1–C8 + Phase-0 findings), cross-brief boundaries coherent; sole defect was
D4's leaked tool-envelope tags + an extra section (both fixed in place); (2) adversarial
re-derivation of the five beyond-dossier claims — ALL survived, two sharpened (full verdicts
in the book README's "Verification record"; headline sharpenings: `adoption_log` has live
CLI+MCP+TUI readers plus deskguard membership, so "kill it" has real blast radius; the
pm-spec:744 GitHub-URL-pointer promise is rejected by the SHIPPED DEFAULT decision/task
gates, not just an edge; runtime prompt edits are ephemeral across a store rebuild).
**REBOOT (2026-07-20, owner directive): the desk-platform stream merged in; combined decide
phase is LIVE.** The parallel stream on dev-tooling-desk (rounds R1–R3: "grow deskkit", the
persona bundle as v1 proof surface, files+held-diffs KB model with a PB-becomes-truth
direction, the three-plane element model + two adversarial reviews) was reviewed against the
decision book — five interaction points found (truth regime undercuts four briefs' criteria;
R3 redesigns D3's vocabulary; D5.a pre-answered; D7 three-sided; two pending owner gates).
Reconciliation applied: platform docs migrated to
`_meta/research/2026-07-design-session/platform/` (originals frozen with pointers — work no
longer happens on that desk); **`D0-platform-frame.md`** added to the book (ratify R1 · truth
regime staged-with-gate · element model as the schema-v2 two-track · imports platform
Q1–Q4); D2–D8 annotated; deck regenerated (17 slides + 4:04 audio, same folder). **THE SESSION IS RULED
(2026-07-20 13:54Z):** the owner accepted every recommendation on the combined form
(`_meta/signoff/2026-07-20-design-session-rulings/answers.json`) with three notes — v1+v2
**model simulations** before v2 finalizes; **centralized prompt tuning**; exec outputs
**trigger-gated** (candidates: meeting, milestone/marker). **Phase 3 RECORDED: ADRs
0009–0018** landed in `docs/decisions/` (platform frame w/ staged truth regime · pointer
grammar · typed-reference contract · item-type validation · disposition completion ·
agent contract · prompt governance · TS-boundary-via-deskkit-proxy · identity & hygiene ·
element-model direction), indexed in the README; the two contradicted spec passages
corrected in place (pm-spec R6.1; librarian-spec §7.2). **The session record is on
PR #113** (branch `docs/design-session-rulings`, commit `5d3046c`: ADRs + decision book +
five dossiers + migrated platform docs + both signoff trails with `answers.json` + the
briefing package + the two spec corrections + `.claude/settings.json` plugin enablement;
`make check` green pre-commit). **CI is GREEN** (verified pre-clear: `ci: pass`,
`claude-review: pass`, `claude: skipping`) — the PR is merge-ready; merging is the owner's
call. Note (2026-07-20, post-record): the old dev-tooling exec desk was **renamed
`dotfiles-agents-desk`** — the frozen platform originals + migration pointers traveled with
it; stale `dev-tooling-desk` paths in earlier session docs resolve there.
Decision batches for Henry ALWAYS go through the owner-signoff HTML form;
decision packages as deck/PDF (project memory).

**PHASE 4 DELIVERED (2026-07-20, post-reboot session). PR #113 merged to main (`e5aee59`);
the wave is FILED.** The planning desk was scaffolded at `_meta/plans/` (planning-desk
standard: 7-script `_utils/` toolkit, README index, `_config.md` with the real gate menu);
the ADR→issue derivation is recorded at
`_meta/research/2026-07-design-session/build-plan.md`. Live on GitHub: **#114–#128** (15
children) + **epic #129** (v1 build-out, milestone 1.0.0, native sub-issues #114–#123,
`gate:1.0.0` on the nine build slices — #122 is a design item, no gate) + **epic #130**
(schema-v2 track, deliberately NO milestone, sub-issues #124–#128, `gate:v2-final` on
#124/#125/#126). Native blocked-by edges: #119 and #122 ← #114; #126 ← #125. **#99 closed
superseded-by-ADR-0013** (implemented by #118 slice D). Process: 10 parallel draft agents →
5 adversarial report-only reviewers (every load-bearing citation re-derived from source;
zero refuted; the one MAJOR was the **migration-number collision between #118 and #123** —
both claim 0015–0017 on the shared librarian chain; both plans + epic #129 now carry the
coordination rule: numbers PROVISIONAL, real sequence assigned at landing from true HEAD,
the two issues' migration commits must SERIALIZE) → 4 scoped fix agents → foreman
verification. Deep `plan.md`s exist for #114/#118/#119/#123; the other bodies are staged +
review-conformed (`coverage.py` lists them as the plan backlog). Desk gates green:
`reconcile.py` clean, all 17 new bodies conformant (the 28 flagged are pre-existing
backlog), `sync-bodies.py` in-sync. **Defaults applied, cheap to re-rule** (also in PR
#131's body): Epic A rides 1.0.0; Epic B no milestone; new labels `gate:1.0.0` /
`gate:v2-final`. Verified findings a builder should know: `UpdateItem` allows unguarded
post-creation `type` mutation (recorded in #117, deliberately out of its scope); the
librarian spec's "kept verbatim" prompt block is ALREADY stale on main, so #120's drift
guard fails red on day one (correct); PocketBase reserves the field name `id` → the #123
column is `doc_id` (frontmatter key stays `id`); `MCP_MODULES` unset must mean all-enabled
or the documented tool-count probe breaks (#114). **The session record is PR #131** (branch
`docs/phase-4-build-plan`: commits `ee4739d` scaffold, `7dd93a2` reviewed wave, `0ea9385`
filed + back-filled). **Next: merge PR #131, then the build lanes open — #114 first (it
blocks #119/#122); #115/#116/#117/#118/#120/#121/#123 are independent; #118+#123 migration
commits serialize per the epic-#129 rule.** Everything below this paragraph predates the
reboot.

**Owner SIGNED OFF on the decision list (2026-07-20 12:11Z**, recorded at
`_meta/signoff/2026-07-20-decision-book-scope/answers.json` — the batch dir with the form is
the sign-off trail): D1–D7 included as-is; **D8 PROMOTED from pull-only backlog to a full
session decision** (scope-change note added to the D8 brief + index); no missing decisions.
One owner-input note rides D7 (verbatim in the brief): leverage the TS-plugin seam for
server-backed capabilities — leans extend-the-boundary over amend-spec-to-reality; input to
weigh, not a ruling. **Phase 1 is CLOSED. Phase 2 package is DELIVERED**: 15-slide deck (PDF, boardroom, zero
overflow) + 4:36 audio + `sources.md` at the exec desk `_meta/briefings/2026-07-20-design-session-decisions/` (relocated from the repo 2026-07-20 — briefings live on the exec desk)
— one slide per decision (two for D5), option labels quoted from the briefs, the five
review-proven facts on their own weights slide, D7's owner note and D8's promotion carried.
**Next: the decide phase** — Henry rules the eight (second signoff form or live); then
Phase 3 rulings → ADRs in `docs/decisions/` + spec deltas, then Phase 4 the build plan.
The decision book + its Verification record is the session's deep material; the deck is
the map. NOTE: the whole design-session tree (`_meta/research/2026-07-design-session/`,
`_meta/signoff/`, and the decision deck now at the exec desk
`_meta/briefings/2026-07-20-design-session-decisions/`) plus this handoff
edit is UNCOMMITTED — commit when Henry's ready (`_meta/` is track-by-default).

**Public launch is deferred until ≥ v1.0.0 (Henry, 2026-07-19) — this is settled, not an open
blocker.** The repo stays PRIVATE until then, so the public `curl|bash` path is expected to 404
by design (every unauthenticated URL `install.sh` needs — `raw…/install.sh`, `/releases/latest`,
`/releases/download/…`). install.sh's URL + checksum logic are already proven correct (authed
assets verify); the only missing link is the unauthenticated fetch, which will resolve for free
when the repo goes public at v1.0.0. **Until then, install from the private repo with `gh` auth**
(see §4 "Installing from the private repo"). Do NOT keep surfacing the private-repo 404 as a
blocker — it's a planned gate, not a bug.

**ON HOLD (Henry, 2026-07-17): going public AND OpenCode (#12) are parked** — public until
≥ v1.0.0 (above); the focus is dogfooding/stabilizing shipped Claude-Code, not opening or
extending. Don't action either without his go-ahead. Buildable follow-ups (#34, #35) can proceed.

**Small open follow-ons:** two accepted PR #40 review nits — the locked-store hint's substring
match could false-positive on a path containing "locked" (harmless; the hint is phrased as a
question), and the chat tapes' 30s answer sleep is tight on slow API days (only matters on
manual re-record; committed GIFs verified good). One accepted PR #48 nit: `ctrl+y` copies an
interrupted turn's partial text (judged correct — it's real answer text). The `[Unreleased]`
CHANGELOG now holds `make install` + the docs split + the whole chat-TUI pass; `make
version-status` advises a bump — cut the next release when Henry's ready.

**v0.6.0 era (2026-07-18/19), collapsed:** chat-TUI UX pass + `record_feedback` merged
(PRs #48/#54); all five TUI roadmap rulings accepted (**ADR 0007** — Charm v2 GO with zero
visual change; #51/#52 ride post-1.0 per the roadmap); v2 migration then core+modules
refactor merged in order (#59→#60) — TUI code lives at
`librarian/internal/modules/librarian/tui/`; #58 closed won't-fix (VHS-toolchain cosmetic;
committed GIFs depict v2 accurately — reopen only if a real redesign needs fresh captures);
**v0.6.0** tagged, assets verified. Detail: the cited PRs/ADRs,
`_meta/briefings/2026-07-18-tui-roadmap-rulings/`, and this file's git history.

**Open backlog, ranked** (no ruling gates the buildable ones):
- **#12** — dual-format Claude+OpenCode fan-out (consumes `bun run package` as the seed).
  Architectural; needs an OpenCode-target ruling before fan-out. **ON HOLD (above).**
- **#34** — CI hardening: enforce shellcheck (incl. `install.sh`) + actionlint + a SHA-pin drift
  guard in CI (all exist locally/pre-commit, not yet in the pipeline).
- **#35** — coverage: unit-test `requireConfig` self-init for non-`query` commands + a behavioral
  test for the TS MCP server.
- **#19** — PB-served webapp chat surface (deferred; ADR 0001 option b — substrate ready, see §3).
- **#36** — Template SOP library (v-next **major**): grow the 2 fixer templates to a full ~20-type
  SOP library from the headcase shared set (`_headcase/shared/sops/`). Needs a design ruling first
  (vendor-vs-sync, type↔dir_kind, planner template-selection, neutrality of vendored examples).
  Corrects the "librarian knows ~20 SOPs" premise — that catalog never existed; the 2-template
  boundary was intentional. A status-against-roadmap briefing deck lives at
  `_meta/briefings/2026-07-17-status-against-roadmap/`.

## 2. Recent deliveries (newest first — full detail in the cited PRs / ADRs / git)

- **2026-07-20 (bug-floor session) — 0.8.0 bug floor shipped as PR #112 (CI green, awaiting
  merge).** Foreman-led crew: 3 scouts → 6 builders + 1 lead (disjoint file scopes on the
  shared tree) → 3 adversarial reviewers, then foreman reconciliation + gates. Closed #67
  (unknown-cmd non-zero exit via a known-command guard before `app.Start()`), #78 (R5 gated on
  an explicit graduation marker), #79 (`mcp-serve` fail-loud + stderr mount signal + brownfield
  skill prereq; `.mcp.json` left as-is — shape frozen by its test, not-on-PATH is host-level),
  #80 (both scaffold `improvement-log.md` copies carry frontmatter + `check-scaffold-frontmatter.mjs`
  guard wired into `make check`), #91 (every mutating PM `Engine` method in `RunInTransaction`,
  version guard inside), #92 (`graduated_to` extraction gated on the explicit marker), #93
  (findings disposition lifecycle — migration `0014` w/ backfill-to-`open`, `findings dispose`
  CLI, live-default `query findings` + `--include-disposed`, `(file,rule,checksum)` inheritance +
  checksum re-open), #94 (killed the false "seven-tool core" string; `docs/tool-surface.md` with
  empirically-verified counts — MCP 5/6/17/18, plugin TS 4), #102 (transition gate resolves
  `file § heading` by its file part; already-seeded desks repaired with no migration). Commit
  `1a3ef75` (+1979/−274); spec §5.1/§5.2 + CHANGELOG `[Unreleased]` updated. Foreman caught +
  fixed cross-slice drift the owners correctly refused to cross into: the `module.go` `0014`
  migration-manifest declaration + `SchemaVersion 13→14` twin (`gatedon_test.go` + `migrate.go`
  comment), S6's initial patch of the wrong scaffold file (redirected to the `desk-setup`
  template), neutrality `#N` literals across two lanes, and untracked agent-memory cruft under
  `librarian/.claude/` (removed; gitignore hardened to `**/.claude/agent-memory/`). Detail: PR #112.
- **2026-07-20 (later session) — owner concerns assessed + briefing package (no repo
  commits; handoff/memory edits only).** Two verified assessments (agent-integration
  asymmetry; document data-model gaps) recorded in the exec desk analysis cited in §1;
  sequencing directive + non-markdown-briefing preference recorded in both handoffs +
  project memory; 15-slide status-and-direction deck (PDF + 4.5-min MP3, comms standard)
  delivered at exec desk `_meta/briefings/2026-07-20-status-and-direction/` (two accepted
  cosmetic nits: footer link spacing, stat sub-label order — only matter on regeneration).
- **2026-07-20 — Lane 6 conformance + exec-desk split + deep-dive triage.** PR #105 (squash
  `217b5e6`, closes #86): mise-en-place scaffold (`.claude/` skeleton, `.mcp.json`, dependabot
  npm@`/plugin` + gomod@`/librarian`, ADR 0000-template) + the three authored entry docs +
  the tests/ ruling; audit 70/0; docs adversarially reviewed 10/10; the dependabot directory
  bug was caught in PR review and fixed pre-merge. Off-repo, same session: the dedicated exec
  desk `desk-standard-desk` bootstrapped greenfield (friction ledger INS-01..10 — including the
  0.5.0-plugin vs 0.7.0-binary version skew and the `template_render` cwd walk-up gap), the
  content lift out of dev-tooling-desk completed, issues #91–#104 filed, 0.8.0 milestone created.
- **2026-07-19 — v0.6.0 release** (PR #61, tag `v0.6.0`). Minor bump 0.5.0→0.6.0: VERSION +
  3 manifests + `[Unreleased]` CHANGELOG rolled into a dated `[0.6.0]` section (SOP kit library,
  `make install`, `record_feedback`, chat-TUI UX pass, `init` onramp, Charm v2 migration,
  core+modules refactor). PR CI + claude-review green, squash-merged; `make release-prep` green
  from clean main; release.yml published all four platform binaries + plugin bundle + checksums;
  darwin/arm64 asset verified (runs, sha256 matches). On-PATH binary refreshed via `make install`.
- **2026-07-18 — v0.5.0→v0.6.0 build-up, collapsed:** v0.5.0 + versioning/changelog guard
  (PR #41, ADR 0005) · chat full-screen TUI, 5 phases (PRs #37–#40, ADR 0004 — `StreamTurn`
  substrate reusable for the #19 webapp SSE route) · `make install` + docs dev/use split
  (PR #42) · profile-first docs onramp (PR #46) · chat-TUI UX pass (PR #48) ·
  `record_feedback` (PR #54, migration 0013). Durable rules live in §4 (versioning/release,
  docs layout, eino/TUI gotchas); detail in the PRs.
- **2026-07-16/17 — v0.4.0 build-out, collapsed:** ADR 0001 (chat REPL + trigger layer,
  `secrets_ref`) · ADR 0002 (XDG store home + desk open-guard) · ADR 0003 (store self-init)
  · distribution + hardening (`install.sh`, SHA-pinned Actions) · root Makefile/VERSION/
  guides + first release **v0.4.0** · field-UX + content-widen batches (PRs #21–#33).
  Durable rules in §4; incidents in §5.

## 3. Where to start next

- **The 0.8.0 bug floor is DONE (merged, unreleased)** — changes sit under `[Unreleased]`;
  the release cut stays owner-gated. Current gate: the decision-list sign-off (§1), then
  Phase 2 of the design session. #79's merge unblocks the PM default-on lane (#83) but that
  is a feature lane — it waits on the design session per the sequencing directive. Deferred
  TUI UX items remain in `docs/development/chat-tui-ux-survey.md`.
- **The design session is RULED and Phase 4 is FILED** (§1): ADRs 0009–0018 bind; the build
  plan is live as epics #129/#130 with children #114–#128. Once PR #131 merges, pick up
  #114 first (blocks #119/#122), then the independent slices. The decision book, dossiers,
  and migrated platform docs under `_meta/research/2026-07-design-session/` are the
  historical record — the ADRs are what binds. Decision packages for Henry stay
  non-markdown (deck/PDF); his answers are collected via the owner-signoff HTML form,
  never chat questions.
- **Cut the next release** when `[Unreleased]` warrants — follow `docs/development/releasing.md`
  (bump VERSION + 3 manifests → roll `[Unreleased]` into a dated CHANGELOG section →
  `make release-prep` → tag). `check-changelog` gates the tag; `make version-status` flags drift.
- **Repo-visibility decision** (§1) gates the public `curl|bash` path — Henry's call, on hold.
- **#19 webapp** (deferred; ADR 0001 option b): substrate is ready — `StreamTurn`'s JSON-taggable
  events (ADR 0004) marshal straight onto a PB-served SSE route (no runtime frontend toolchain).
- Any new streaming/TUI work: gates per §4; live proof via VHS (tapes in `docs/development/tapes/`,
  method in the §4 eino/bubbletea note).

## 4. Conventions & gotchas

- **Gates** (run all before claiming done — via the root Makefile since PR #26):
  `make check` (neutrality + self-test + purity + actionlint) · `make test` (bun 59 + go 314) ·
  `make verify` (`librarian/verify.sh`, 48 checks — **now also runs in CI + the release gate**, PR #33) ·
  `make package` (drift guard) · `node scripts/check-version-sync.mjs`. CI (`ci.yml`) is the
  aggregate required check. Note: shellcheck + actionlint are NOT yet CI-enforced (→ #34).
- **Versioning/release (ADR 0005, since PR #41 — runbook `docs/development/releasing.md`)**: SemVer on the
  single root `VERSION`; a bump = edit `VERSION` + the three manifests (sync-guarded) **and** move
  `[Unreleased]` CHANGELOG entries into a dated `## [<version>]` section. `check-changelog.mjs` is
  a **hard gate** (release-prep + release.yml) — a tag with no CHANGELOG section fails.
  `check-version-status.mjs` is a **non-blocking advisory** (`make version-status` + a CI step,
  always exit 0) that warns when `plugin/`/`librarian/` drifted past the last tag without a bump.
  `ci.yml` checks out `fetch-depth: 0` for it. Both scripts + CHANGELOG live outside the neutrality
  surface (repo root / `scripts/`). `make install` builds + drops the **`deskkit`** binary in `~/.local/bin`
  (override `PREFIX=`; renamed from `pocket-librarian` in 0.7.0); `/plugin` updates the plugin
  only — the binary is a separate artifact.
- **Installing from the private repo** (until the public launch at ≥ v1.0.0): the public
  `install.sh` / `curl|bash` path 404s by design while private — use authed `gh` instead. Two ways:
  - **From a local clone** (version-stamped, what a dev should use): `make install` (root or
    `librarian/`) → `~/.local/bin/deskkit`.
  - **Straight from the release** (no clone/build): download the platform asset with `gh`:
    ```bash
    gh release download v0.7.0 --repo hsb3/desk-standard \
      --pattern "deskkit_*_$(uname -s|tr '[:upper:]' '[:lower:]')_$(uname -m|sed 's/x86_64/amd64/;s/aarch64/arm64/')" \
      --output ~/.local/bin/deskkit --clobber && chmod +x ~/.local/bin/deskkit
    ```
    (Assets are `deskkit_<ver>_{darwin,linux}_{amd64,arm64}` since 0.7.0; verify against the
    release's `checksums.txt`.)
- **Docs layout (since the dev/use split)**: `docs/` is indexed by `docs/README.md` in two
  tracks — **Using** (`getting-started`/`plugin-guide`/`librarian-guide` + `media/*.gif`) and
  **Development** (`docs/development/` = overview README + `releasing.md` + `tapes/*.tape`). The
  **spec** (`docs/pocket-librarian-v1-spec.md`) and **ADRs** (`docs/decisions/`) deliberately stay
  at the `docs/` top level — they're cited from code, skills, and the neutrality-lint allowlist, so
  their paths are kept stable (don't move them to `development/`). VHS **tape sources** live in
  `docs/development/tapes/`; their **`.gif` output** lands in `docs/media/` (the `Output` path is
  repo-root-relative, so tapes moved without touching it).
- **Generated, never hand-edit**: `plugin/claude-plugin/mcp/server.js` and
  `plugin/claude-plugin/schema/profile.schema.yaml` — regen with `cd plugin && bun run package`;
  CI drift-guards them. Bundle output is byte-identical across macOS/linux (proven).
- **Neutrality lint scope** = `plugin/` + `librarian/` recursively. Bare issue refs (`#11`)
  in Go comments/tests inside `librarian/` FAIL the lint — write issue-free comments. `docs/` is
  exempt. `.claude-plugin/marketplace.json` (owner identity) is deliberately outside the surface
  (recorded in `_meta/m-05-data-surfaces.md`).
- **Commits auto-close**: `Resolves #N` in a commit message closes the issue on push to
  main — post the proof comment first, or use `gh issue comment` after (close-with-comment
  fails on an already-closed issue).
- **Beware pre-staged files**: parallel agents may `git add` their scope; `git commit` with a
  pathspec still sweeps the whole index. Check `git status` before every commit.
- Librarian env: `DESK_ROOT`/`DESK_NAME` required by all tool commands; LLM only needed by
  `agent`/`chat`/MCP-driven calls (`LLM_PROVIDER` env → `profile.models` → anthropic; key
  from the env var named by `LLM_API_KEY_ENV` / `secrets_ref.llm_api_key`, else per-provider
  `ANTHROPIC_API_KEY`/`OPENAI_API_KEY`/`GEMINI_API_KEY`); `LIBRARIAN_AUTONOMOUS_WRITES=true`
  gates `apply_fix` (MCP and enqueued tasks — checked at execution time); `restore` is
  CLI-only. `serve` extras: `PB_SUPERUSER_EMAIL`/`PB_SUPERUSER_PASSWORD` (idempotent
  first-run superuser), `CLAIMER_POLL_INTERVAL` (wake-layer claimer).
- **PocketBase bootstrap runs before cobra**: `Execute()` calls `Bootstrap()` (which CREATES
  the data dir) before `RootCmd.Execute()` dispatches any RunE/PreRunE — anything that must
  prevent store creation has to run in `main()` before `app.Start()` (the argv-scan location
  guard does). And PocketBase registers `serve`/`superuser` INSIDE `Start()` then discards
  their RunE errors (goroutine) — fail-closed behavior in serve paths needs a direct
  `os.Exit(1)` (see the OnServe desk-guard), not a returned error. Worked example since
  PR #48: `init` executes standalone in `main()` BEFORE the app exists (and stays out of
  `storeTouchingCommands`) precisely so Bootstrap can't create a stray store dir.
- **Store location** (since PR #24): no `--dir` → `$XDG_DATA_HOME/deskkit/
  <DESK_NAME>/` (dir renamed with the binary in 0.7.0); unresolvable DESK_NAME + no `--dir` →
  exit 1 (serve/migrate included).
  verify.sh exports a scratch XDG_DATA_HOME — keep it hermetic when adding checks.
- Store self-init (ADR 0003) and the bare-`TextField` 5000-char cap: both covered in
  CLAUDE.md (hot-loaded) — not repeated here.
- **Altering a shipped collection: add a forward migration, don't edit the applied one**
  (precedent: `0010`, `0011`, `0012`). Editing `000N`'s field decl only reaches fresh stores; a
  new migration that mutates the field via `FindCollectionByNameOrId` + `Save(c)` fixes existing
  stores on next migrate. Source decls carry a pointer comment (`// … widened in 0011`,
  `// "infra" added in 0012`). Two variants proven: **TextField Max** (`0011`) and **SelectField
  Values** enum-extension (`0012` adds `infra` to `dir_kind`). An enum-extension's DOWN migration
  must remap rows off the new value FIRST (`0012`: `infra`→`other`) before dropping it from the
  enum, or a rollback leaves a row outside its reverted enum (same data-first pattern as `0010`).
- **eino streaming gotchas** (bind any new streaming caller to these — origin: PRs #37–#38 / ADR 0004):
  agent output stream is bursty under the anthropic tool-call checker (use model-callback
  stream copies for live tokens); ctx-cancel doesn't abort a stuck provider stream; failed
  tools fire OnError only; zero-arg tool calls need the `argNormalizingTool` adapter (in
  place at buildTools — keep new tools behind it). **No terminal queries after bubbletea
  starts** (no glamour WithAutoStyle / lazy lipgloss adaptive colors) — responses leak into
  the textarea; regression-guarded in `internal/modules/librarian/tui/defects_test.go`
  (path since the #60 modules refactor). Since PR #48 the chat
  theme resolves ONCE pre-program (`tui.ResolveTheme`) and `tui.Run` pins
  `lipgloss.SetHasDarkBackground` to it — load-bearing for embedded bubbles components whose
  DEFAULT styles use AdaptiveColor (the textarea): without the pin they're only safe via a
  bubbletea v1 init workaround that v2 removes. New TUI colors go in `newStyles`'s per-theme
  switch, never AdaptiveColor; new renderers take `(width, theme)`.
- **VHS chat tapes** (`chat.tape` dark + `chat-light.tape` light) both need
  `ANTHROPIC_API_KEY` — record with `ANTHROPIC_API_KEY="$(secret get ANTHROPIC_API_KEY)"
  bash scripts/record-media.sh`. Re-recording re-encodes ALL GIFs; `git restore` the
  non-chat ones when their tapes didn't change (byte churn, no content).
- **Worktree provisioning**: untracked-and-load-bearing = `plugin/node_modules` (run
  `bun install --frozen-lockfile`) and `.claude/agent-memory/` (machine-local). Everything
  else a worktree agent needs is committed.
- `_meta/` holds `build-brief.md`, `m-05-data-surfaces.md`, `briefings/`, and this handoff — the
  full taxonomy (operations/ ignore stanza etc.) comes via the mise-en-place scaffold when needed.

## 5. Incident log

- 2026-07-20 (build wave): a read-only scout REFUSED its brief because the main tree
  contradicted it — root cause: the session had been `git fetch`ing after merges but never
  pulled local `main`, so the working tree was three merges stale while `origin/main` was
  current. In multi-worktree waves, `git pull` local main before any agent reads the main
  tree (worktrees fork from commits, so they were unaffected). The scout's stop-condition
  discipline caught a foreman error; keep briefing scouts with hard stop conditions.
- 2026-07-20 (build wave): `gh pr merge` failed on a conflicting PR but the `;`-chained
  cleanup (worktree remove + branch delete) ran anyway — recovery needed a re-checkout from
  the remote branch. Chain merge→cleanup with `&&`, never `;`.
- 2026-07-17: first real release (v0.4.0) cut successfully, but the live `install.sh` e2e
  surfaced that **the repo is private** — every unauthenticated URL the public install flow needs
  (`raw…/install.sh`, `/releases/latest`, `/releases/download/…`) returns 404. Initially looked
  like CDN propagation lag (persisted >6 min); root-caused by checking `gh repo view` visibility.
  Authed `gh release download` proved the assets are valid (binary runs, sha256 matches
  checksums.txt), so install.sh is correct — the blocker is purely repo visibility. Resolution:
  make the repo public when launching (§1), or add token auth to install.sh if it stays private.
  Lesson: a persistent (not transient) 404 on release assets that the authed API can see = check
  repo visibility before blaming propagation.
- 2026-07-18: the `pocket-librarian` on PATH was a stale build (no `chat`, `--version` printed
  `(untracked)`) — `/plugin` updates the plugin, NOT the standalone binary (separate artifacts).
  Fixed by installing the checksum-verified v0.5.0 release binary + adding `make install`. Lesson:
  after a release, update the binary separately (`make install` or `gh release download`).
- 2026-07-16: first #11 commit accidentally swept 27 pre-staged pocketbase files into the
  fix commit (parallel agent had staged them); caught pre-push, reset --soft and recommitted
  per-stream. See the pre-staged-files gotcha above.
- 2026-07-17: a gate command piped to `tail` (`check-neutrality.mjs 2>&1 | tail -1 && git add`)
  masked neutrality's exit 1 — a pipeline's status is the LAST command's (`tail`=0), so the
  `&&` proceeded and a #18 commit landed locally with 4 bare-`#18` lint violations in
  `librarian/`. Caught on the next standalone run, `--amend`ed before push (nothing bad reached
  the remote). Lesson: when a command GATES a commit, run it bare and check its own exit code —
  never pipe it. (Neutrality scope reminder: `docs/` is exempt, so the spec's `#18` refs are
  fine; `librarian/` Go comments/tests are not.)

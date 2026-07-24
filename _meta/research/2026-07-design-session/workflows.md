---
type: analysis
status: draft
created: 2026-07-20
updated: 2026-07-20
tags: [design-session, workflows]
synopsis: End-to-end trigger/steps/writes/invariants for every librarian and PM workflow, cited to code, for the 2026-07 design session.
---

*Purpose: give the design session every product workflow end-to-end (trigger, steps, what it
writes, invariants) with code citations, so workflow changes can be reasoned about without
re-reading source.*

Status: draft (2026-07-20)

All citations are `path:line` against commit `51235f6` (branch `fix/0.8.0-bug-floor`). Spec
references (`docs/development/specs/pocket-librarian-v1-spec.md` §N, `docs/development/specs/pm-system-v1-spec.md` §N) are given for
orientation; **code is ground truth** per the assignment brief, and any place code and spec text
appear to diverge is called out inline.

---

## 1. Librarian lane (`librarian/internal/modules/librarian/`)

### 1.1 Sweep — reindex the tree

- **Trigger**: CLI `deskkit sweep` (one-shot); hourly cron (`trigger.RegisterCron`,
  `librarian/internal/modules/librarian/trigger/trigger.go:52-61`) enqueues a `sweep` task the
  claimer dispatches (`trigger.go:187-190`).
- **Steps** (`librarian/internal/modules/librarian/tools/sweep.go:26-102`):
  1. Validate `DESK_ROOT` exists (`sweep.go:28-30`).
  2. Walk the tree pruning `.git`, `logs`, and any `pb_*` dir/file (`walkDeskFiles`,
     `sweep.go:192-226`).
  3. Inside one `app.RunInTransaction`, for each file: checksum + parse frontmatter
     (`desklib.ParseFrontmatter`), derive `dir_kind` via `EntityDirMap` prefix match
     (`dirKindFor`, `sweep.go:287-321`), and extract `graduated_to` **only from an explicit
     marker** — a frontmatter `graduated_to:` key or a canonical `^graduated to: <ref>` line
     (`graduationMarker`, `sweep.go:368-376`; regex `sweep.go:359`) — never from a bare `#N`
     merely quoted in prose (`sweep.go:253-261`).
  4. Upsert against the existing `files` row by path; idempotent via `fileRowDiffers`
     (COMPARE_FIELDS excludes `path`/`last_seen`, `sweep.go:172-188`).
  5. Any pre-existing `files` row not seen this walk is **soft-deleted** (`deleted=true`, never
     removed) — `sweep.go:86-94`.
- **Writes**: `files` collection only (create/update/soft-delete rows); no filesystem writes.
- **Invariants**: idempotent (unchanged rows produce zero writes); a per-file read error is
  recorded and skipped, never aborts the whole sweep (`sweep.go:58-62`); `dir_kind` is derived
  for every file, not just `.md` (`sweep.go:229-230`).
- Spec: `docs/development/specs/pocket-librarian-v1-spec.md` §5.1 (lines 825-946).

### 1.2 Patrol — flag rule violations (dry-run)

- **Trigger**: CLI `deskkit patrol [--path]`; cron enqueues a full-desk `patrol` task each hour
  alongside `sweep` (`trigger.go:52-61`); a new `files` row (from sweep) enqueues a
  **path-scoped** `patrol` task via the `OnRecordAfterCreateSuccess("files")` hook
  (`trigger.go:36-46`).
- **Steps** (`librarian/internal/modules/librarian/tools/patrol.go:24-195`):
  1. Load non-deleted `files` rows, optionally filtered to `--path`'s subtree
     (`pathOrSubtree`, `patrol.go:38-48`).
  2. Load currently-`flagged` findings to build the dedupe/resolution sets (`patrol.go:52-70`).
  3. Run **mechanical** rules R1 (missing universal frontmatter keys), R2 (journal filename
     shape), R3 (file under wrong type-dir) in that fixed order (`mechanicalRules`,
     `patrol.go:201`, `checkR1/R2/R3` at `patrol.go:306-355`).
  4. Run **judgment** rules separately: R4 (invalid/empty decision status — detection is
     mechanical but the fix requires a supervisor's semantic pick of
     `{proposed,accepted,rejected,superseded}`, so it is filed as judgment and has no fixer,
     `patrol.go:123-134,357-381`); R5 (a >40-line entity doc carrying an explicit graduation
     marker should have collapsed to a pointer stub, `patrol.go:383-413`); R6 (HANDOFF
     staleness vs. the newest desk commit date, runs **only** when `cfg.HandoffPath` is in the
     filtered set, `patrol.go:143-153,415-450`).
  5. **Dedupe**: a finding is filed only if no open finding shares its `(path, rule, checksum)`
     key (`findingDedupeKey`/`isDuplicateFinding`, `patrol.go:230-252`); a still-firing
     `(path, rule)` is tracked in `fired` regardless of dedupe, so a deduped finding is never
     mistakenly resolved (`patrol.go:72-75,83-84`).
  6. **Disposition inheritance**: a freshly-filed (re-fired, non-deduped) finding inherits the
     most recent PRIOR finding's non-`'open'` disposition on the same `(file, rule, checksum)`
     (ordered `-patrol_run,-id`), so a supervisor's `acknowledged`/`triaged`/`wont_fix` call
     survives a resolve→re-fire cycle; absent a prior disposed row it defaults to `'open'`
     (`inheritedDisposition`, `patrol.go:254-273`, called at `patrol.go:104`).
  7. **Resolution**: any open finding whose `(path, rule)` did not re-fire this run transitions
     to `state=resolved` with `resolved_run` set; a scoped (`--path`) patrol only resolves
     findings inside its scope (`patrol.go:155-174`).
  8. Append one `patrol_log` row summarizing files/findings/resolved counts
     (`patrol.go:176-189`).
- **Writes**: `patrol_findings` (create/resolve), `patrol_log` (one row per run). **Never
  touches the filesystem.**
- **Invariants**: mechanical-vs-judgment split is a filing property (`severity` field), not a
  detection-difficulty property — R4's detection is mechanical but its *fix* is judgment
  (`patrol.go:123-127`); dedupe key is the finding's own **stored** checksum (at flag time), not
  the file's current checksum (`patrol.go:50-51,95`).
- Spec: `docs/development/specs/pocket-librarian-v1-spec.md` §5.2 (lines 946-1030).

### 1.3 propose_fix → apply_fix → restore (the write boundary)

**propose_fix** (`librarian/internal/modules/librarian/tools/propose_fix.go:27-104`):

- **Trigger**: CLI `deskkit propose-fix [--run <id>] [--rules R1,R2,...]`; wake-layer `propose_fix`
  task (`trigger.go:196-200`).
- **Steps**, guard order is EXACT (`propose_fix.go:21-26,128-210`):
  1. Load only `state='flagged' && severity='mechanical'` findings (R1/R2/R3; R4/R5/R6 are
     judgment and never auto-proposed) intersected with the caller's `--rules` filter
     (`fixableRules = {R1,R2,R3}`, `propose_fix.go:106-126`).
  2. **Load the ignore list FIRST** (`desklib.LoadIgnoreList`); if absent/unreadable, **fail
     closed for the whole batch** — every candidate gets outcome `ignored`, nothing is read or
     planned (`propose_fix.go:74-88`).
  3. Per candidate, in order: ignore check → file-exists check → read+checksum → staleness
     guard (`finding.checksum` vs current on-disk checksum) → compute the fix plan
     (`computePlan`: `planR1`/`planR2`/`planR3`, `propose_fix.go:242-372`) → **RECORD ORIGINAL
     FIRST** (a `revisions` row with `original_content`/`original_checksum`, `applied=false`,
     `restored=false`) before any outcome is reported "recorded" (`propose_fix.go:184-209`,
     "Boundary 1, decision 0014").
  4. A failure to record the original (e.g. store error) yields a per-file `error` outcome and
     the batch continues — no filesystem write can ever follow a finding without a `revisions`
     row (`propose_fix.go:96-101,184-191`).
- **Writes**: `revisions` (new rows only). No filesystem write ever.
- **Invariants**: planners are pure functions of `(rule, file record, original bytes)`
  (`propose_fix.go:238-241`) so `apply_fix` can re-derive the identical plan later; R2/R3 never
  clobber an existing destination (`propose_fix.go:346-348,365-367`).

**apply_fix** (`librarian/internal/modules/librarian/tools/apply_fix.go:23-274`):

- **Trigger**: CLI `deskkit apply-fix`; wake-layer `apply_fix` task — but only actually applies
  when `LIBRARIAN_AUTONOMOUS_WRITES=true`, otherwise the task is left `state=deferred` for a
  supervised CLI run (`trigger.go:201-213`).
- **Steps**, per revision (`applyOne`, `apply_fix.go:113-240`):
  1. **Reload the ignore list**; fail closed for the whole batch if unreadable
     (`apply_fix.go:31-45`).
  2. Re-check ignore → confirm the file still exists → re-run the staleness guard against
     `finding.checksum` (not `original_checksum` — the authority is the checksum recorded at
     flag time, `apply_fix.go:136-149`) → **re-derive the plan deterministically** from the
     backing file record + current bytes (`apply_fix.go:151-179`).
  3. Write byte-exact: `edit` → `desklib.WriteExact`; `move` → `os.Rename` (+ optional pointer
     stub write) — **outside** the DB transaction, since the filesystem is not transactional
     (`apply_fix.go:181-212`).
  4. Mark `revisions.applied=true` and `patrol_findings.state='fixed'` atomically
     (`apply_fix.go:214-236`).
  5. If the stub write fails after a successful rename (a **half-applied move**), log a
     warning naming the revision + new path; `restore --by-path` can recover
     (`apply_fix.go:201-211`). If the DB patch fails after a successful write, log a warning;
     the revision stays `applied=false` (`apply_fix.go:226-236`).
  6. Append one `adoption_log` row per batch: `"run <id>: outcome=count, ..."` or "nothing to
     fix" (`recordAdoptionLog`, `apply_fix.go:242-273`).
- **Writes**: the desk filesystem (edit/move/stub) — **the only tool that mutates the desk
  tree**; `revisions.applied`, `patrol_findings.state`, `adoption_log`.
- **Invariants**: registration for the autonomous agent is gated by
  `LIBRARIAN_AUTONOMOUS_WRITES` (§5.4) purely by **exclusion from the tool slice**
  (`toolcore.AgentTools`, `librarian/internal/core/toolcore/toolcore.go:133-142`) — the function
  itself has no runtime gate check; the gate lives at registration time, and separately at
  dispatch time on the wake path (`trigger.go:205-209`).

**restore** (`librarian/internal/modules/librarian/tools/restore.go:21-179`):

- **Trigger**: CLI `deskkit restore --revision <id>` or `--by-path <path>` — **supervised only,
  never in the autonomous agent tool set and never exposed over MCP**
  (`librarian/internal/core/mcp/server.go:16-21`).
- **Steps**:
  1. `--by-path` resolves to the latest `applied=true && restored=false` revision whose
     `path`/`new_path` matches; if none, falls back to a filesystem-confirmed half-applied-move
     scan (`resolveRevisionByPath`, `restore.go:121-152`).
  2. Guards: already-restored is a hard error; never-applied is a hard error **unless** the
     filesystem confirms the exact half-applied-move crash window (file absent at `path`,
     present at `new_path`) — `confirmHalfApplied`, `restore.go:154-178` — in which case the
     applied flag is caught up inside the restore transaction (`restore.go:52-58,92-96`).
  3. **Verify `sha256(original_content) == original_checksum` before writing anything**
     (`restore.go:60-65`).
  4. If the action was a move and the moved file still exists, remove it; write the exact
     original bytes back at the original path (byte-exact, `desklib.WriteExact`)
     (`restore.go:72-85`).
  5. Atomically: `revisions.restored=true` (+ `applied` catch-up if half-applied) and reopen the
     originating finding to `state='flagged'` (`restore.go:90-116`).
- **Writes**: the desk filesystem (byte-exact restore); `revisions.restored`,
  `patrol_findings.state`.
- **Invariants**: byte-exact reversal is checksum-verified before any write — this is the
  "record-original-first, byte-exact reversible" boundary named in the assignment (decision
  0014); a half-applied *edit* (vs. move) is indistinguishable from a concurrent user edit and
  is deliberately **never** auto-confirmed (`restore.go:158-161`).

Spec: `docs/development/specs/pocket-librarian-v1-spec.md` §5.3-§5.5 (lines 1030-1308).

### 1.4 record_feedback

- **Trigger**: the agent, mid-loop, when a tool fails or a desk convention doesn't fit
  (`kind="problem"`), or when the user explicitly asks it to record something
  (`kind="feedback"`) — `record_feedback.go:13-21`.
- **Steps** (`RecordFeedback`, `librarian/internal/modules/librarian/tools/record_feedback.go:22-70`):
  validate `kind ∈ {problem, feedback}` and non-empty `summary`; default `source="agent"`
  (model sets `"user"` when asked); insert one `feedback` row with `status="open"`.
- **Writes**: `feedback` collection only — **no desk file is touched**.
- **Invariants**: this is a DB-only write, so — unlike `apply_fix` — it is **not** gated behind
  `LIBRARIAN_AUTONOMOUS_WRITES` (`record_feedback.go:18-21`); it is `AgentDefault` in every
  configuration.

### 1.5 Findings dispose — supervised disposition lifecycle (new in the 0.8.0 bug floor)

- **Trigger**: CLI `deskkit findings dispose <finding-id> --as <open|acknowledged|triaged|
  wont-fix>` — **deliberately a CLI-only subcommand, not an MCP/agent tool**, the same
  supervised-only posture as `restore` (`librarian/cmd/deskkit/main.go:809-837`).
- **Steps** (`DisposeFinding`, `librarian/internal/modules/librarian/tools/dispose.go:38-59`):
  normalize `wont-fix`→`wont_fix` (`normalizeDisposition`, `dispose.go:61-70`), validate against
  `{open, acknowledged, triaged, wont_fix}` (`dispositionValues`, `dispose.go:23-28`), set
  `patrol_findings.disposition` and **nothing else** — `state` is deliberately untouched.
- **Writes**: `patrol_findings.disposition` only.
- **Invariants**: `disposition` is orthogonal to `state` — a disposed finding stays `flagged`
  and survives re-patrol via the dedupe-plus-inheritance mechanism in §1.2 step 6
  (`dispose.go:30-33`); `query findings` defaults to `disposition='open'` and needs
  `--include-disposed` to widen (`main.go:773-807`). This tool (plus the disposition column and
  its inheritance-on-re-fire semantics) is one of the fixes landed in the 0.8.0 "bug floor" pass
  — see the repo's most recent commits (`git log`: "fix(0.8.0): bug floor — ... findings
  disposition ...").

### 1.6 The eino ReAct agent loop

- **Trigger**: manual CLI invocation (Phase-1 entry point, `Run` is called by the CLI's chat
  path) or an agentic (`query`/`custom`) wake-layer task (`trigger.go:219-223`).
- **Steps** (`librarian/internal/modules/librarian/agent/agent.go`):
  1. Open an `agent_runs` row (`status="running"`) — `createAgentRun`, referenced at
     `agent.go:149`, implemented in `persist.go` (not read in depth; see Gaps).
  2. Build the provider chat model (`chatModelFactory` = `provider.NewChatModel`,
     `agent.go:36,155`).
  3. Build the gated tool slice: `toolcore.AgentTools(cfg)` — every `AgentDefault` tool always,
     plus each `AgentGated` tool (i.e. `apply_fix`) only when `AutonomousWrites` is set
     (`toolcore.go:133-142`); each tool is wrapped in `argNormalizingTool` to coerce an
     empty/whitespace `ArgumentsInJSON` to `"{}"` — a zero-argument tool call (e.g. `sweep`)
     streams no argument deltas, and the concatenated `""` fails eino's `InferTool`
     unmarshal on the **streaming** path only (`agent.go:119-139`).
  4. Build the `react.Agent`: `MessageModifier` resolves the **system prompt** at RUN START
     (not registration) from the `prompts` collection — `key="librarian.system" &&
     active=true`, newest `version` — falling back to the embedded seed
     (`prompt.Embedded()`) if no active row exists, then prepends a desk-facts preamble
     interpolated only from config (identity-neutral) — `systemPrompt`/
     `interpolateDeskFacts`, `agent.go:42-62,91-93`. `MaxStep` bounds the loop
     (`cfg.AgentMaxStep`, default 12). The Anthropic provider gets a custom
     `StreamToolCallChecker` (`claudeToolCallChecker`) that buffers the whole stream looking
     for a tool call, because Claude does not emit it in the first chunk and eino's default
     checker would prematurely conclude "no tool call" (`agent.go:64-82,96-99`).
  5. Drive the loop via `ag.Generate`, with the **single** transcript-persistence callback
     (`rc.persistHandler()`, implemented in `persist.go` — not traced in depth, see Gaps)
     registered as a compose callback (`agent.go:168-170`).
  6. Persist the final assistant message explicitly on success (it never appears in a
     subsequent model *input*, so the input-side callback never captures it) and finalize the
     `agent_runs` row (`succeeded`/`failed`/`blocked`, `step_count`) — `agent.go:172-185`,
     `finishRun` (in `persist.go`, not read).
- **Writes**: `agent_runs` (one row per run), `messages` (transcript, via the persistence
  callback — mechanism not traced in depth, see Gaps).
- **Invariants**: `apply_fix`'s write gate and `restore`'s total exclusion are enforced purely
  by **which tools are in the slice**, never by a runtime check inside the loop
  (`agent.go:102-106`); the system prompt is a **live** DB read every run, so a GUI/REST edit to
  the `prompts` collection applies to the very next run with no redeploy (`agent.go:38-41`).
- Spec: `docs/development/specs/pocket-librarian-v1-spec.md` §6 (lines 1389-1728).

### 1.7 Wake layer / cron hooks

- **Trigger sources** (`librarian/internal/modules/librarian/trigger/trigger.go`):
  - A new `files` row (from sweep) → `OnRecordAfterCreateSuccess("files")` enqueues a
    path-scoped `patrol` task (`trigger.go:36-46`).
  - Hourly cron (`"0 * * * *"`, job id `"desk-patrol"`) → enqueues a full-desk `sweep` task then
    a full-desk `patrol` task; enqueue failures inside the cron job are swallowed — the next
    tick retries (`trigger.go:52-61`).
  - `StartClaimer` runs one background goroutine (started under `serve` only, wired in
    `librarian/cmd/deskkit/main.go:286`), polling at `cfg.ClaimerPollInterval` (default 5s,
    `trigger.go:67-70`).
- **Claim/dispatch/finish cycle** (`ClaimOnce`, `trigger.go:104-162`):
  1. Find the highest-priority `state='queued'` task (`-priority,created`).
  2. Claim it **transactionally**, re-checking `state=='queued'` inside the transaction so a
     concurrent poll can never double-claim (`trigger.go:114-135`).
  3. Dispatch outside the claim transaction, by `kind`: `sweep`/`patrol`/`propose_fix`/
     `apply_fix`/`restore` call the matching tool function **directly, no model, no loop**
     (`dispatch`, `trigger.go:182-227`); `apply_fix` additionally checks
     `cfg.AutonomousWrites` and, if false, marks the task terminal `state='deferred'` rather
     than applying on the wake path (`trigger.go:201-213`); `query`/`custom` require the
     injected `AgentAction` (the eino loop, wired from `main.go` to avoid an import cycle) —
     absent injection is a hard error (`trigger.go:219-223`).
  4. Mark the task `done`/`failed` (with the error folded into `payload.error`,
     `withError`, `trigger.go:298-307`) — `ClaimOnce`, `trigger.go:146-162`.
  5. Both the poll loop (`StartClaimer`) and the per-task dispatch (`safeDispatch`) have their
     own `recover()` — a panicking tool/action becomes a normal `failed` task rather than
     killing the claimer goroutine or `serve` itself (`trigger.go:80-92,164-177`).
- **Writes**: `tasks` (enqueue/claim/finish), plus whatever the dispatched tool writes (§§1.1-1.5
  above).
- **Invariants**: hooks/cron enqueue rather than run inline, keeping record hooks fast and
  non-blocking, with every wake auditable via the `tasks` row (`trigger.go:1-9`); registered
  only under `serve` (`librarian/cmd/deskkit/main.go:264-269`) so one-shot CLI commands that also
  create records never enqueue tasks.
- Spec: `docs/development/specs/pocket-librarian-v1-spec.md` §2.4 (lines 158-227).

---

## 2. PM lane (`librarian/internal/modules/pm/`)

### 2.1 The phase state machine

- Four rigid phases in rank order: `queue`(0) → `work`(1) → `review`(2) → `terminal`(3)
  (`librarian/internal/modules/pm/statemachine/statemachine.go:13-21,34-46`).
- Legal edges are a fixed, code-owned table — `queue→work`/`work→review`/`review→terminal`
  are `Advance`; `review→work`/`work→queue` are `Demote`; `terminal→work` is `Reopen`
  (`legalEdges`, `statemachine.go:64-71`). The spec's diagram lists `review→work` under both
  demote and reopen; the code resolves this to **one** canonical label (`Demote`), reserving
  `Reopen` for `terminal→work` (`statemachine.go:58-63`).
- `blocked` is a **bool side-state on the item, not a phase** — the machine only answers "is
  this edge legal"; the engine owns blocked/claim handling (`statemachine.go:1-6`).
- Status labels are a friendly vocabulary mapped onto phases (`DefaultStatusLabels`,
  `statemachine.go:118-128`): `backlog`/`next`→queue, `active`→work, `in-review`→review,
  `done`/`dropped`/`superseded`→terminal. Freely editable per desk via
  `desk_config.status_labels`. `blocked`/`waiting` are explicitly **not** labels — they surface
  the blocked flag regardless of phase (`statemachine.go:115-117`).
- Spec: `docs/development/specs/pm-system-v1-spec.md` §3.2-§3.3 (lines 410-459).

### 2.2 Gated transitions — the §4.1 sequence (machine → blocked → claim → gates → write)

- **Trigger**: `transition_item` tool (any surface — MCP/CLI/TUI, all thin adapters over the
  same engine call, `librarian/internal/modules/pm/tools/tools.go:117-127`); also invoked
  internally by `SetStatusLabel` for a cross-phase label change (§2.5 below).
- **Steps** (`Engine.transitionCore`,
  `librarian/internal/modules/pm/engine/engine.go:311-390` — always called on a tx-scoped
  engine, so it composes atomically with a caller's further writes):
  1. Load the item (desk-scoped: `loadItem` refuses an item on a different desk,
     `engine.go:180-190`) and check the **optimistic-concurrency version token**
     (`checkVersion`, `engine.go:192-201`) — a mismatch refuses naming the current vs. expected
     version, never silently overwrites.
  2. The **machine** admits the edge or refuses before gates are even consulted
     (`statemachine.Edge`, `engine.go:330-334`).
  3. **Blocked** refuses forward (`Advance`) edges — a blocked item cannot advance until
     unblocked (`engine.go:335-338`).
  4. A **live foreign claim** (someone else's non-expired claim) refuses advance/demote/reopen
     alike (`liveForeignClaim`, `engine.go:203-215,339-344`).
  5. The **gate engine** evaluates whatever the desk's config binds to `(item.type, edge key)`
     — forward edges gate by default; demote/reopen gate only when the config explicitly names
     them (`dc.rules.Effective`, `gates.Effective`, `librarian/internal/modules/pm/gates/gates.go:133-150`,
     called at `engine.go:345-352`).
  6. **`Evaluate`** (`gates.go:174-214`) resolves each requirement's document pointer
     (`"item"` → the item's own `pointer` field; `"note:<key>"` → the body of a keyed note,
     `pointerResolver`, `engine.go:419-443`) and asks the injected `DocumentValidator.Verdict`
     for each. **With no validator registered, every documented gate fails closed** naming "no
     document validator available" (`gates.go:183-185`).
  7. **The verdict reads DISK, not the store**: the librarian module's `Verdict`
     (`librarian/internal/modules/librarian/module.go:104-175`) resolves the pointer against
     `DESK_ROOT` (containment-checked, `resolveDeskPath`, `module.go:220-231`), does a direct
     `os.ReadFile` (`module.go:130`), parses frontmatter with `desklib.ParseFrontmatter`
     (`module.go:143`), and validates against the schema-v1 vocabulary — **it is never a
     `files`-collection lookup**, so a gate verdict never depends on sweep freshness and needs
     no store handle of its own (`module.go:96-103`). Trait predicates read the same way via
     the companion `Frontmatter` method (`module.go:181-201`,
     `schema.FrontmatterReader`, `librarian/internal/core/schema/schema.go:34-42`).
  8. On refusal: a `gate_refused` transitions row is prepared as a **`pendingAudit`**
     (`engine.go:246-254,361-364`) but **written only after the transaction settles**
     (`Transition`, `engine.go:288-309`) — a refusal mutates nothing, so the transaction rolls
     back, but the refusal must still be observable, which requires a non-atomic write outside
     the rolled-back tx. The `*Refusal` returned names exactly what is missing
     (`gates.go:152-167`).
  9. On success: write the new `phase`, re-derive `status_label` to the phase's default if the
     current label no longer maps to it (`engine.go:372-375`), bump the version token
     (`bump`, `engine.go:218`), save, append an audit `transitions` row (`audit`,
     `engine.go:256-273`), then run the **cascade** scan (§2.3 below) — all inside the same
     transaction (`engine.go:369-389`).
- **Writes**: `items` (phase/status_label/version), `transitions` (one audit row, or one
  `gate_refused` row on refusal), plus whatever the cascade touches.
- **Invariants**: `version` check + mutating write share one transaction via `withApp`
  (closing the check-then-act TOCTOU, `engine.go:238-244`); a gate refusal is always observable
  (never silent) even though it rolls back every other write; the same `transitionCore` is the
  **one** path every surface routes through (no second write path).
- Spec: `docs/development/specs/pm-system-v1-spec.md` §4.1 (lines 548-573), §4.2 (573-614), §3.6 (492-517).

### 2.3 Cascade — Block/Unblock + auto/auto-reopen/manual/permanent

- **Trigger**: automatically after every successful `Transition` (`engine.go:386,718`); also the
  initial-block check when `Link` creates a new gating edge (`engine.go:682-698`).
- **Block/Unblock** (direct, supervisor/agent-invoked side-state, `engine.go:448-531`):
  - `Block` refuses on a `terminal` item, is idempotent if already blocked, else sets
    `blocked=true` and records `restore_phase` (the phase to return to), bumps version, audits
    (`engine.go:448-482`).
  - `Unblock` is idempotent if not blocked, else clears the flag and **restores the item to its
    recorded `restore_phase`** (`setBlocked`, `engine.go:518-531`).
- **Cascade scan** (`Engine.cascade`, `engine.go:708-745`): every outgoing `blocks` edge from the
  item that just transitioned applies its `cascade` rule to its target:
  - `auto`: target auto-clears once the blocker **reaches** `unblock_at` (rank comparison,
    one-shot; a later regression does not re-block) — `tryAutoUnblock`, `engine.go:747-786`.
  - `auto-reopen`: same auto-clear, **plus** the blocker regressing back below `unblock_at`
    re-blocks the target — `reblock`, `engine.go:788-807`.
  - `manual`: never auto-clears (surfaced as "unblockable"; a human/agent must clear it).
  - `permanent`: never auto-clears; the edge itself must be deleted.
  - A target auto-clears only when **every** gating edge into it is satisfied AND
    auto-clearable — one still-manual/permanent/unsatisfied edge keeps it blocked
    (`tryAutoUnblock`, `engine.go:764-777`).
- **Writes**: `items.blocked`/`restore_phase`/`version` on the target(s); `transitions` audit
  rows for every block/unblock the cascade performs.
- **Invariants**: cascade writes are desk-scoped defense-in-depth — `tryAutoUnblock`/`reblock`
  refuse to touch an item on a different desk even if an edge somehow points there
  (`engine.go:756,795-797`).
- Spec: `docs/development/specs/pm-system-v1-spec.md` §3.5 (lines 476-492).

### 2.4 Claim/Release

- **Trigger**: `claim_item`/`release_item` tools.
- **Claim** (`Engine.Claim`, `engine.go:537-572`): version check; refuse if a **live** foreign
  claim exists (`liveForeignClaim`, `engine.go:203-215`); an **expired** claim is treated as
  free; re-claiming your own claim renews the TTL. Sets `claimed_by` + `claim_expires` (TTL from
  `desk_config.claim_ttl_minutes` / `PM_CLAIM_TTL` env / 30m default,
  `loadDeskConfig`, `engine.go:64-100`).
- **Release** (`Engine.Release`, `engine.go:574-607`): only the live holder may release a live
  claim; anyone may clear an expired one.
- **Writes**: `items.claimed_by`/`claim_expires`/`version`; one `transitions` audit row
  (`event="claim"`/`"release"`).
- **Invariants**: a live foreign claim also refuses `Transition` (advance/demote/reopen) and
  refuses `Claim` — §2.2 step 4 and this section share the same `liveForeignClaim` predicate,
  so a claim is a hold over every mutating path, not just claim/release itself.
- Spec: `docs/development/specs/pm-system-v1-spec.md` §3.6 (lines 492-517).

### 2.5 SetStatusLabel — same-phase vs. cross-phase

- **Trigger**: `update_item` tool, when `StatusLabel` is set (`UpdateItem`,
  `librarian/internal/modules/pm/engine/queries.go:544-548`), or the `transition_item`... no —
  status-label writes route only through `update_item`; direct callers can also invoke
  `SetStatusLabel` (`engine.go:815-875`).
- **Steps**:
  1. Version check up front (fast-fail before the desk-config load, `engine.go:827-829`).
  2. Resolve the label against `desk_config.status_labels`; an unknown label is refused
     (`engine.go:834-836`).
  3. **Same-phase** (the label maps to the item's current phase): a plain field write — set
     `status_label`, bump version, save. `transitionCore` never runs (`engine.go:838-846`).
  4. **Cross-phase** (the label maps to a different phase): routed through
     `transitionCore` **inside the same transaction** as the label pin, so the phase change and
     the label write commit or roll back as one unit (`engine.go:847-864`) — `transitionCore`
     already bumped the version and wrote the audit row, so the label-pin save does **not**
     bump again (keeping the caller's `version+1` expectation aligned with the audit trail).
  5. A gate refusal from the cross-phase path records its `gate_refused` row after the
     transaction settles, identically to the direct `Transition` path (`engine.go:866-870`).
- **Writes**: same as §2.2 (cross-phase) or a plain `items.status_label`/`version` write
  (same-phase).
- **Invariants**: "the label and the machine cannot drift" — a label mapping to a different
  phase is *always* a gated transition request, never a bare label overwrite.
- Spec: `docs/development/specs/pm-system-v1-spec.md` §3.3 (lines 442-459).

### 2.6 AddNote

- **Trigger**: `add_note` tool.
- **Steps** (`Engine.AddNote`, `engine.go:892-921`): inside a transaction (to keep the note's
  `phase` snapshot consistent with the item read — a non-tx read could record a stale phase if
  a transition lands in between), insert one `notes` row: `item`, `phase` (the item's *current*
  phase at note time), `key`, `body`, `actor`/`actor_kind`.
- **Writes**: `notes` collection only (no version bump on the item, no audit `transitions` row).
- **Invariants**: notes are phase-scoped snapshots, not versioned or gated.
- Spec: `docs/development/specs/pm-system-v1-spec.md` §3.7 (lines 517-533).

### 2.7 The importer

- **Trigger**: programmatic only (test lane §10.8 rebuild-reproducibility test; the D8
  adoption seed path) — not a CLI/MCP tool in this slice.
- **Steps** (`librarian/internal/modules/pm/importer/importer.go:87-174`):
  1. Validate the manifest up front (no empty/duplicate keys, no unknown parent/dep references)
     before any write (`importer.go:93-118`).
  2. Derive each item's record id **deterministically** from `sha256(desk + "\x00" + key)`,
     truncated to 15 lowercase-hex chars — satisfies PocketBase's id constraint and makes two
     imports of the same manifest into fresh stores produce identical ids (`ItemID`,
     `importer.go:74-81`).
  3. Create items **parents-first** in waves, with a **sorted-keys-within-wave** pass order so
     the create sequence (and any incidental store-side ordering) is stable across rebuilds,
     independent of manifest order (`importer.go:125-160`); an unresolvable parent cycle is a
     hard error (`importer.go:157-159`).
  4. Create dependency edges via the **same** `engine.Link` every surface uses (never a second
     write path), idempotently skipping an edge that already exists at the same canonicalized
     `(from, to, kind)` (`e_createDep`, `importer.go:201-226`).
  5. Every mutation goes through `engine.CreateItem`/`engine.Link` — the importer is a thin
     driver, owning no transition/gate/cascade logic itself (`importer.go:18-19`).
- **Writes**: `items`, `dependencies` (via the engine — so also `transitions` for any initial
  block the engine's `Link` applies, §2.2/§2.3).
- **Invariants**: idempotent (a re-run against a store that already holds an item by its
  deterministic id skips it, `importer.go:15-16,178-186`); the `GraphSnapshot`/`Canonical`
  oracle (`importer.go:269-328`) projects a deterministic, timestamp-excluded view of the graph
  so two rebuilds from the same manifest can be asserted byte-identical.
- Spec: `docs/development/specs/pm-system-v1-spec.md` §8.1-§8.2 (lines 759-794).

### 2.8 Version-token optimistic concurrency (cross-cutting)

Every mutating engine call (`Transition`, `Block`, `Unblock`, `Claim`, `Release`, `UpdateItem`,
`SetStatusLabel`) takes the version its caller last read and calls `checkVersion`
(`engine.go:192-201`) as its first act after loading the item; a mismatch is a `*Refusal` naming
`(current, yours)`, never a silent overwrite. Every successful mutation calls `bump`
(`engine.go:218`) exactly once — the `transitionCore`-then-label-pin composition in §2.5 is the
one place with special-cased bump discipline (only one bump across the two writes) to keep the
caller's `version+1` expectation aligned with the audit trail (`engine.go:856-858`).

### 2.9 pendingAudit for gate-refused rows (cross-cutting)

`pendingAudit` (`engine.go:246-254`) is the general mechanism behind the "always observable, even
when it rolls back" property in §2.2 step 8 and §2.5 step 5: a gate refusal happens **inside** a
transaction (so it can inspect item state consistently) but must be **recorded outside** it (so
the record survives the rollback). `Transition` and `SetStatusLabel` are the two call sites that
thread a `*pendingAudit` out of the transaction closure and write it via `e.audit(...)`
afterward (`engine.go:301-304,866-870`); a failure in that post-tx audit write is swallowed (`_
= e.audit(...)`) — the refusal still stands to the caller even if the audit row fails to land.

---

## 3. Cross-lane touch points

- **`schema.DocumentValidator` / `schema.FrontmatterReader`** — the narrow seam PM consumes
  instead of ever reading librarian collections (`librarian/internal/core/schema/schema.go:12-42`).
  The librarian module is the sole implementer: `Verdict`
  (`librarian/internal/modules/librarian/module.go:104-175`) and `Frontmatter`
  (`module.go:181-201`), both reading the desk filesystem directly (never the `files`
  collection) — this is the exact fact the "gate verdict reads DISK not store" framing in the
  brief refers to.
- **`module.Register`** (`librarian/internal/core/module/module.go:74-116`) is where the seam is
  wired: it asserts no owned-collection collision across enabled modules
  (`module.go:81-86`), merges every enabled module's `Tools()` into the shared `toolcore`
  registry (`module.go:95`), captures the first `DocumentValidator` implementer (the librarian
  module) into `Registry.Validator` (`module.go:96-103`), and — in a **second pass**, after
  every module's tools are collected — injects that validator into any `ValidatorConsumer`
  (the pm module's `SetValidator`, `librarian/internal/modules/pm/module.go:52-54`)
  (`module.go:105-112`). Consumers must therefore read the injected value **lazily** at
  invoke time (`pm/module.go:78-81`'s closure captures `func() schema.DocumentValidator {
  return m.validator }`, passed into `pmtools.Specs`), since `Tools()` runs before injection.
- **Shared config** — `internal/core/config.Config` is one struct both lanes read
  (`librarian/internal/core/config/config.go:24-48`); the pm module receives it via
  `Configure(cfg)` at registration (`module.Configurable`, `module.go:45-52`,
  `pm/module.go:48-50`) exactly like the librarian module does (`librarian/module.go:36-39`).
- **Shared MCP server** — `librarian/internal/core/mcp/server.go` loops over the **merged**
  `toolcore` registry (`toolcore.ExposedTools(cfg)`, `server.go:57-81`) rather than a
  hand-maintained per-lane switch, so PM's twelve tools appear on the same stdio MCP surface as
  the librarian's seven with zero server.go changes per module
  (`server.go:1-32` package doc; `NewServer`, `server.go:69-81`). The same
  §5.4-style write gate (`AgentDefault`/`AgentGated`) and the restore-never-exposed rule are
  `ToolSpec` flags shared across both lanes (`toolcore.go:35-51,133-156`).
- **`toolcore.ToolSpec`** itself is the mechanical union point: one struct shape
  (`Module`/`Name`/`Description`/`InputType`/`WritesFiles`/`AgentDefault`/`AgentGated` +
  two closures) that both lanes populate via `toolcore.New[I]`
  (`librarian/internal/core/toolcore/toolcore.go:35-88`), consumed identically by the eino
  agent loop (`AgentTools`, `toolcore.go:133-142`, called from
  `librarian/internal/modules/librarian/agent/agent.go:107-117`) and the MCP server
  (`ExposedTools`, `toolcore.go:144-156`).

---

## 4. Feature gating

| Env var | Gates | Checked where |
|---|---|---|
| `PM_ENABLED` | Whether the pm module is registered at all: its 5 collections
  (`items`/`dependencies`/`transitions`/`notes`/`desk_config`), its migrations, its 12 tools, its
  3 TUI views, its hooks, its realtime emitter. Disabled ⇒ a librarian-only desk gets **no PM
  collections physically** (migrations are only registered into PocketBase's runner when
  enabled). | `(*pm.Mod).Enabled`, `librarian/internal/modules/pm/module.go:63` (reads
  `cfg.PMEnabled`); resolved in `config.Load` as `env PM_ENABLED > profile
  modules.pm.enabled > false`, `librarian/internal/core/config/config.go:113-116`; consumed by
  `module.Register`'s enable/collision-check loop, `librarian/internal/core/module/module.go:74-89`. |
| `LIBRARIAN_AUTONOMOUS_WRITES` | Whether `apply_fix` is included in the **autonomous agent's**
  tool slice (both the eino loop and the MCP server) — `restore` is never included regardless.
  Also gates whether an enqueued `apply_fix` wake-task actually runs or is left `deferred`. |
  Registration-time: `toolcore.AgentTools`, `librarian/internal/core/toolcore/toolcore.go:129-142`
  (`AgentGated && cfg.AutonomousWrites`); resolved in `config.Load`,
  `librarian/internal/core/config/config.go:112` (`envBool`, default `false`); dispatch-time
  re-check on the wake path: `librarian/internal/modules/librarian/trigger/trigger.go:201-213`.
  It does **not** gate `record_feedback` (a DB-only write, `record_feedback.go:18-21`) or
  `dispose`/`restore` (both CLI-only, never in the agent slice at all). |
| `PM_AUTONOMOUS_WRITES` | Whether the PM module's **write** tools (`create_item`,
  `update_item`, `transition_item`, `block_item`, `unblock_item`, `add_note`, `link_items`,
  `claim_item`, `release_item`) are `AgentDefault` for the autonomous agent — the read tools
  (`get_context`, `list_items`, `get_item`) are always `AgentDefault`. Default is **ON**
  (unlike `LIBRARIAN_AUTONOMOUS_WRITES`'s default-off), because PM tools write only the store
  (never desk files) and the real safety is `transition_item`'s document gates, not a registration
  gate. | `(*pm.Mod).Tools`, `librarian/internal/modules/pm/module.go:78-81` (`writes :=
  m.cfg == nil || m.cfg.PMAutonomousWrites`, passed as `writesEnabled` into
  `pmtools.Specs`, `librarian/internal/modules/pm/tools/specs.go:27-104` where it becomes the
  `w` flag on every write tool's `AgentDefault` position); resolved in `config.Load`,
  `librarian/internal/core/config/config.go:118-121` (`envBool("PM_AUTONOMOUS_WRITES", true)`). |

Both `LIBRARIAN_AUTONOMOUS_WRITES` and `PM_AUTONOMOUS_WRITES` are **registration-time**
gates — they decide which `ToolSpec`s land in `AgentTools(cfg)`'s slice, not a runtime
conditional inside any tool body. `PM_ENABLED` is stronger still: it is a **module**-level gate
checked once in `module.Register`, upstream of tool registration, migrations, hooks, and
realtime all at once.

---

## Gaps & uncertainties

- **Transcript persistence mechanism** (`librarian/internal/modules/librarian/agent/persist.go`)
  was not read in depth — `createAgentRun`, `rc.persistHandler()`, and `finishRun` are cited by
  call site only (`agent.go:149,168,185`), not by their own implementation. The exact shape of
  the `messages` rows and the callback wiring into eino's `compose.WithCallbacks` are unverified.
- **Session resume** (`agent/resume.go`, `agent/session.go`, `agent/stream.go`) were not read at
  all; if the design session needs multi-turn/resume semantics, that is an open gap.
- **`gates/defaults.go`** (`DefaultConfig()`, the shipped default gate-rules bundle consulted by
  `loadDeskConfig` when no `desk_config` row exists, `engine.go:72-76`) was not read — the exact
  default gate bindings (which item types/transitions require which documents out of the box)
  are unverified.
- **`pm/collections/`** (the 5 PM collection schema definitions + their migrations) was not
  read — field types/constraints for `items`/`dependencies`/`transitions`/`notes`/`desk_config`
  are inferred only from their usage in `engine.go`/`queries.go`, not verified against the
  migration source.
- **`pm/tui/`** views and the CLI's `pm` subcommand tree (`librarian/cmd/deskkit/main.go`'s PM
  section) were not traced — the brief's "one core, three surfaces" claim
  (`pm/tools/tools.go:1-4`) is taken as ground truth by comment, not independently verified for
  the TUI/CLI adapters specifically (only the tool-function layer was verified).
- **`librarian/internal/modules/librarian/tools/specs.go`** (the exact per-tool
  `AgentDefault`/`AgentGated`/`WritesFiles` flags for the librarian's seven tools) was not read
  directly — the gating claims for `apply_fix`/`restore`/`record_feedback` in §4 are sourced
  from comments in `apply_fix.go`, `restore.go`, and `record_feedback.go` rather than the specs
  table itself.
- **`core/schema/doctypes.go`** (the schema-v1 vocabulary: `Vocab()`, `KnownType`,
  `StatusAllowed`, `ValidateFrontmatter`) was not read — gate-config validation
  (`gates.ParseRules`) and the `Verdict` frontmatter check both depend on it, but its own
  correctness (e.g. exactly which types/statuses are legal) is unverified here.
- **`provider.NewChatModel`** (`librarian/internal/modules/librarian/provider/`) — provider
  selection/construction (Anthropic/OpenAI/Gemini) was not traced; only its call site in the
  agent loop is cited.
- **R1-R6 vs. the brief's "R1-R5"**: the brief's task description says "rules R1–R5"; the code
  implements six rules, R1 through **R6** (HANDOFF staleness), all cited in §1.2. This dossier
  follows the code (ground truth per the brief's own instruction); the design session should be
  aware the rule count is six, not five, if that number is load-bearing elsewhere.
- **`RegisterHooks`/`RegisterRealtime`/`StartClaimer` exact serve-time wiring** in
  `librarian/cmd/deskkit/main.go` was read only in the two windows cited (`main.go:170-239,
  255-294`); the full `OnServe` callback (superuser bootstrap ordering, error-handling paths
  before line 255) was not read end-to-end. The wiring **shown** (module loop calling
  `RegisterHooks` then `RegisterRealtime` per enabled module, then one `StartClaimer` call) is
  verified; anything earlier in the same callback is not.
- **Divergence check**: no spec-vs-code divergence was found beyond the R4-filed-as-judgment
  behavior and the review→work demote/reopen labeling choice, both of which are explicitly
  reasoned about in code comments as deliberate interpretations, not bugs. The
  `pm-system-v1-spec.md` "Interpretations / deviations to report" section (lines 996-1012)
  describes the D2-era toolcore generalization as then-incomplete; the code now shows it
  completed (`toolcore.go` is fully generalized, both lanes register through it), so that
  concern reads as resolved rather than open.

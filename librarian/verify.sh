#!/bin/bash
# verify.sh — the deskkit Phase-1 verify gate (spec §9.4), adapted to the
# single-binary Go CLI: build, migrate, seed the spec's four concrete fixtures (§9.4 table)
# into a THROWAWAY scratch desk under mktemp (never a real desk), then drive the tool chain
# sweep -> patrol -> propose-fix -> apply-fix -> restore, asserting the record-original-first
# safety boundary (decision 0014) end to end. Numbered PASS/FAIL lines; non-zero exit on any
# failure. Run by the operator/CI, never asserted by the agent (spec §9.4 note).
#
# Known gaps vs the spec text (see HANDOFF for detail; do not silently "fix" by inventing
# flags):
#   - The spec's dead-store-refusal trigger `LIBRARIAN_FAULT_INJECT=revision-store-down`
#     (§9.2/§9.4 check 10) is NOT implemented in the current Go tree (verified: no match
#     anywhere in librarian/). This script substitutes a filesystem-permission fault
#     (chmod the store tree read-only) to force the same revisions-insert failure and
#     proves the same invariant (rc != 0, no fs write) without inventing the env flag.
#   - `serve` is never started here: none of the checks in this gate's brief require a live
#     HTTP server, and starting/stopping one only adds process-management risk. `make serve`
#     / `make gui` remain the way to exercise `serve` manually.
#   - `query live_files` does not expose a per-file checksum (only path/dir_kind/doctype/
#     status/graduated_to/git_last_commit — see internal/tools/query.go fileBrief). The
#     "per-path checksum set" reproducibility check (§9.4 check 7) is therefore proven by an
#     independent shasum snapshot of the scratch desk tree (mirroring sweep's own file-walk
#     pruning), not by reading checksums back out of the CLI.
# The idiom `<condition>; check "<desc>" $?` recurs throughout this harness: a test condition is
# evaluated purely for its exit status, which is then fed to check() as pass/fail. SC2319 ("$? after
# a condition usually means you wanted the command before it") structurally misfires on that
# intentional pattern at every call site, so it is disabled file-wide below; every OTHER lint rule
# stays enforced.
# shellcheck disable=SC2319
set -uo pipefail
cd "$(dirname "$0")" || exit 1

BIN="deskkit"
PASS=0
FAIL=0
N=0

check() { # check <description> <rc: 0=pass, nonzero=fail>
  N=$((N + 1))
  if [ "$2" -eq 0 ]; then
    printf '[%02d] PASS  %s\n' "$N" "$1"
    PASS=$((PASS + 1))
  else
    printf '[%02d] FAIL  %s\n' "$N" "$1"
    FAIL=$((FAIL + 1))
  fi
}

sha() { shasum -a 256 "$1" | awk '{print $1}'; }

# snapshot mirrors internal/tools/sweep.go's walkDeskFiles pruning (skip .git/, logs/, and
# any pb_*-prefixed dir or file) so it is an independent oracle for "the same files with the
# same bytes", not a re-read of the DB the tools themselves populate.
# .librarian-ignore is also pruned: it is expected infrastructure auto-created by the FIRST
# tool invocation (after the as-seeded snapshot), not desk drift — including it would fail
# the as-seeded and rebuild comparisons for a file the tools are supposed to create.
snapshot() {
  # Portable across GNU/BSD find+xargs: no `xargs -r` (GNU-only "no run if empty" — BSD
  # xargs lacks it and would otherwise invoke shasum with zero args, which reads stdin and
  # blocks), no `find -quit` (GNU-only).
  local prune=( -name '.git' -o -name 'logs' -o -name 'pb_*' -o -name '.librarian-ignore' )
  local first
  first=$(find "$WORK" -mindepth 1 \( "${prune[@]}" \) -prune -o -type f -print 2> /dev/null | head -n 1)
  if [ -z "$first" ]; then
    return 0
  fi
  find "$WORK" -mindepth 1 \( "${prune[@]}" \) -prune -o -type f -print0 \
    | sort -z \
    | xargs -0 shasum -a 256
}

WORK=$(mktemp -d "${TMPDIR:-/tmp}/deskkit-verify.XXXXXX")

# Hermetic store home (ADR 0002 §2): point XDG_DATA_HOME at a throwaway scratch for the WHOLE
# run. Every store-touching command now resolves its store to
# $XDG_DATA_HOME/deskkit/<DESK_NAME> when no --dir is passed, so exporting a scratch
# XDG_DATA_HOME guarantees NO check — including the new resolution/guard checks below — can ever
# touch the operator's real ~/.local/share store. Exported so every child process inherits it.
XDG_DATA_HOME=$(mktemp -d "${TMPDIR:-/tmp}/deskkit-xdg.XXXXXX")
export XDG_DATA_HOME
# The main run drives the "verify-desk" desk; with no --dir its store resolves to this dir.
STORE="$XDG_DATA_HOME/deskkit/verify-desk"

# Section 13's own scratch dirs are declared (empty) here, before the EXIT trap is installed,
# so ONE trap covers the whole script — including a SIGINT mid-section-13, after these are
# mktemp'd in section 13 but before that section finishes. Each is guarded with an empty-string
# check in cleanup() since they are unset for most of the run.
XDG2=""
DESK2=""
DIR3=""
XDG4=""
XDG5=""
DESK5=""

# shellcheck disable=SC2329,SC2317  # invoked indirectly via the `trap cleanup EXIT` below; SC2317 is the same finding under newer shellcheck (runner) versions
cleanup() {
  chmod -R u+w "$WORK" 2>/dev/null || true
  rm -rf "$WORK"
  chmod -R u+w "$XDG_DATA_HOME" 2>/dev/null || true
  rm -rf "$XDG_DATA_HOME"
  [ -n "$XDG2" ] && { chmod -R u+w "$XDG2" 2>/dev/null || true; rm -rf "$XDG2"; }
  [ -n "$DESK2" ] && { chmod -R u+w "$DESK2" 2>/dev/null || true; rm -rf "$DESK2"; }
  [ -n "$DIR3" ] && { chmod -R u+w "$DIR3" 2>/dev/null || true; rm -rf "$DIR3"; }
  [ -n "$XDG4" ] && { chmod -R u+w "$XDG4" 2>/dev/null || true; rm -rf "$XDG4"; }
  [ -n "$XDG5" ] && { chmod -R u+w "$XDG5" 2>/dev/null || true; rm -rf "$XDG5"; }
  [ -n "$DESK5" ] && { chmod -R u+w "$DESK5" 2>/dev/null || true; rm -rf "$DESK5"; }
}
trap cleanup EXIT

# run_lib inherits the exported scratch XDG_DATA_HOME above; it passes NO --dir on purpose, so
# the run exercises the ADR 0002 §2 XDG default-resolution path end to end.
run_lib() { DESK_ROOT="$WORK" DESK_NAME="verify-desk" ./"$BIN" "$@"; }

# Single-writer guard (spec §2.7): refuse to blow away the store out from under a live `serve`.
if [ -f .deskkit.pid ] && kill -0 "$(cat .deskkit.pid)" 2>/dev/null; then
  echo "FATAL: a serve process is running (pid $(cat .deskkit.pid)) — run 'make stop' first." >&2
  exit 1
fi

echo "scratch desk: $WORK"
echo

# --- 0. build + clean the scratch store ---------------------------------------------------
rm -rf "$STORE"
go build -o "$BIN" ./cmd/deskkit
check "build ./$BIN" $?

# --- 0b. self-initialization (ADR 0003) ---------------------------------------------------
# A tool command against a store that was never `migrate up`'d must succeed by auto-applying
# the app migrations at the requireConfig choke point, NOT leak the bare sql.ErrNoRows. Uses a
# DISTINCT desk name so its store is a fresh, never-migrated location under the scratch XDG home
# (cleaned by the EXIT trap); DESK_ROOT reuses $WORK (still empty here). Regression guard for the
# uninitialized-store leak.
DESK_ROOT="$WORK" DESK_NAME="verify-selfinit-desk" ./"$BIN" query summary > /dev/null 2>&1
check "self-init: query on a never-migrated store succeeds (ADR 0003)" $?

# --- 1. migrate: applies + is idempotent (spec §9.4 check 2) -----------------------------
run_lib migrate up > /dev/null 2>&1
check "migrate up" $?
run_lib migrate up > /dev/null 2>&1
check "migrate up is idempotent (second run is a no-op)" $?

# --- 2. seed the spec §9.4 fixture table verbatim ------------------------------------------
mkdir -p "$WORK/tasks" "$WORK/analyses" "$WORK/_structure/decisions"

# F-R1: missing frontmatter, correctly placed under tasks/ -> R1 only.
cat > "$WORK/tasks/r1-fixture-missing-fm.md" <<'EOF'
---
type: task
---
one body line
EOF

# F-R3: full frontmatter, type: task but filed under analyses/ -> R3 only.
cat > "$WORK/analyses/r3-fixture-type-task.md" <<'EOF'
---
type: task
created: 2026-07-15
updated: 2026-07-15
tags: []
synopsis: "type mismatch fixture"
---
short body line
EOF

# F-R5: full frontmatter, correctly placed, >40 raw lines + a bare issue ref -> R5 (judgment).
{
  cat <<'EOF'
---
type: analysis
created: 2026-07-15
updated: 2026-07-15
tags: []
synopsis: "graduated fixture"
---
graduated to #111
EOF
  for i in $(seq 1 45); do printf 'padding line %s\n' "$i"; done
} > "$WORK/analyses/r5-fixture-graduated.md"

# F-IGN: missing frontmatter (genuine R1) but under _structure/decisions/, an embedded-default
# ignore-list prefix entry -> R1 finding exists but propose_fix/apply_fix must refuse it.
cat > "$WORK/_structure/decisions/9999-ignore-fixture.md" <<'EOF'
---
type: decision
status: accepted
---
one body line
EOF

check "seeded the four §9.4 fixtures (F-R1, F-R3, F-R5, F-IGN)" $?

SHA_R1_ORIG=$(sha "$WORK/tasks/r1-fixture-missing-fm.md")
SHA_R3_ORIG=$(sha "$WORK/analyses/r3-fixture-type-task.md")
SHA_IGN_ORIG=$(sha "$WORK/_structure/decisions/9999-ignore-fixture.md")
SNAP_SEEDED=$(snapshot)

R3_SRC="analyses/r3-fixture-type-task.md"
R3_DST="tasks/r3-fixture-type-task.md"
R1_PATH="tasks/r1-fixture-missing-fm.md"
IGN_PATH="_structure/decisions/9999-ignore-fixture.md"

# --- 3. sweep indexes the fixtures + is idempotent (spec §9.4 checks 3, 6) ----------------
SWEEP1=$(run_lib sweep)
RC=$?
check "sweep runs" $RC
N_FILES=$(echo "$SWEEP1" | jq -r '.total')
[ "$RC" -eq 0 ] && [ "$N_FILES" -ge 4 ]
check "sweep indexed the desk tree (total=$N_FILES)" $?

SWEEP2=$(run_lib sweep)
CREATED2=$(echo "$SWEEP2" | jq -r '.created')
UPDATED2=$(echo "$SWEEP2" | jq -r '.updated')
DELETED2=$(echo "$SWEEP2" | jq -r '.soft_deleted')
[ "$CREATED2" -eq 0 ] && [ "$UPDATED2" -eq 0 ] && [ "$DELETED2" -eq 0 ]
check "re-running sweep changes nothing (created=$CREATED2 updated=$UPDATED2 soft_deleted=$DELETED2)" $?

# --- 4. patrol produces the expected findings (spec §9.4 check 9) ------------------------
PATROL1=$(run_lib patrol)
RC=$?
check "patrol runs" $RC
RUN_ID=$(echo "$PATROL1" | jq -r '.run_id')
N_R1=$(echo "$PATROL1" | jq -r '.by_rule.R1 // 0')
N_R3=$(echo "$PATROL1" | jq -r '.by_rule.R3 // 0')
N_R5=$(echo "$PATROL1" | jq -r '.by_rule.R5 // 0')
[ "$N_R1" -ge 2 ] && [ "$N_R3" -ge 1 ] && [ "$N_R5" -ge 1 ]
check "patrol found >=2 R1 (F-R1+F-IGN), >=1 R3 (F-R3), >=1 R5 judgment (F-R5): R1=$N_R1 R3=$N_R3 R5=$N_R5" $?

# --- 5. every query kind exits 0 with non-empty output (spec §9.4 check 5) ---------------
for kind in live_files recent orphans uncollapsed findings summary adoption; do
  OUT=$(run_lib query "$kind" 2>&1)
  RC=$?
  [ "$RC" -eq 0 ] && [ -n "$OUT" ]
  check "query $kind" $?
done

N_UNCOLLAPSED_BEFORE=$(run_lib query uncollapsed | jq -r '.count')

# --- 5b. finding id round-trips through query -> findings dispose (issue: finding briefs must
# carry the record id, so `findings dispose <id>` is reachable from the CLI alone) -------------
FINDINGS_JSON=$(run_lib query findings)
FIRST_FINDING_ID=$(echo "$FINDINGS_JSON" | jq -r '[.by_rule[][] | .id] | .[0] // empty')
[ -n "$FIRST_FINDING_ID" ]
check "query findings: each finding brief carries a non-empty id" $?

UNCOLLAPSED_JSON=$(run_lib query uncollapsed)
UNCOLLAPSED_COUNT=$(echo "$UNCOLLAPSED_JSON" | jq -r '.count')
UNCOLLAPSED_WITH_ID=$(echo "$UNCOLLAPSED_JSON" | jq -r '[.findings[] | select((.id // "") != "")] | length')
[ "$UNCOLLAPSED_WITH_ID" -eq "$UNCOLLAPSED_COUNT" ]
check "query uncollapsed: every finding brief carries an id ($UNCOLLAPSED_WITH_ID/$UNCOLLAPSED_COUNT)" $?

DISPOSE_OUT=$(run_lib findings dispose "$FIRST_FINDING_ID" --as acknowledged --reason "verify.sh id round-trip")
RC=$?
echo "$DISPOSE_OUT" | jq -e --arg id "$FIRST_FINDING_ID" '.id == $id and .disposition == "acknowledged"' > /dev/null
DISPOSE_SHAPE_OK=$?
[ "$RC" -eq 0 ] && [ "$DISPOSE_SHAPE_OK" -eq 0 ]
check "findings dispose <id-from-query> acknowledges the finding, echoing the same id" $?

FINDINGS_AFTER=$(run_lib query findings)
echo "$FINDINGS_AFTER" | jq -e --arg id "$FIRST_FINDING_ID" '([.by_rule[][] | .id] | index($id)) == null' > /dev/null
check "query findings (default) stops listing the disposed id" $?

FINDINGS_INCLUDE_DISPOSED=$(run_lib query findings --include-disposed)
echo "$FINDINGS_INCLUDE_DISPOSED" | jq -e --arg id "$FIRST_FINDING_ID" '([.by_rule[][] | .id] | index($id)) != null' > /dev/null
check "query findings --include-disposed still lists the disposed id" $?

# Restore to 'open' so the mechanical-vs-judgment uncollapsed-count comparison later in this
# gate (§8, "F-R5's judgment finding is untouched by apply-fix") is unaffected by this detour.
run_lib findings dispose "$FIRST_FINDING_ID" --as open > /dev/null
check "findings dispose <id> --as open restores it (keeps later uncollapsed counts stable)" $?

# --- 6. dead-store refusal (spec §9.2/§9.4 check 10 — see the header note: the documented
# LIBRARIAN_FAULT_INJECT env flag is not implemented, so a filesystem-permission fault stands
# in for it) --------------------------------------------------------------------------------
chmod -R -w "$STORE"
run_lib propose-fix --run "$RUN_ID" > /dev/null 2>&1
RC=$?
chmod -R u+w "$STORE"
[ "$RC" -ne 0 ] \
  && [ "$(sha "$WORK/$R3_SRC")" = "$SHA_R3_ORIG" ] \
  && [ ! -f "$WORK/$R3_DST" ]
check "dead-store fault: propose-fix refuses (rc=$RC), F-R3 untouched, no destination write" $?

# --- 7. propose-fix: real run -------------------------------------------------------------
PROPOSE_OUT=$(run_lib propose-fix --run "$RUN_ID")
RC=$?
check "propose-fix runs" $RC

OUT_R1=$(echo "$PROPOSE_OUT" | jq -r --arg p "$R1_PATH" '.proposed[] | select(.path==$p) | .outcome')
OUT_R3=$(echo "$PROPOSE_OUT" | jq -r --arg p "$R3_SRC" '.proposed[] | select(.path==$p) | .outcome')
OUT_IGN=$(echo "$PROPOSE_OUT" | jq -r --arg p "$IGN_PATH" '.proposed[] | select(.path==$p) | .outcome')
[ "$OUT_R1" = "recorded" ] && [ "$OUT_R3" = "recorded" ]
check "propose-fix recorded originals for F-R1 and F-R3 (R1=$OUT_R1 R3=$OUT_R3)" $?
[ "$OUT_IGN" = "ignored" ]
check "propose-fix refuses the ignored F-IGN path (outcome=$OUT_IGN)" $?

# no filesystem write happens at propose time regardless of outcome
[ "$(sha "$WORK/$R1_PATH")" = "$SHA_R1_ORIG" ] && [ "$(sha "$WORK/$R3_SRC")" = "$SHA_R3_ORIG" ]
check "propose-fix performed no filesystem write" $?

# --- 8. apply-fix: commit the recorded revisions (spec §9.4 checks 11-17) ----------------
APPLY_OUT=$(run_lib apply-fix --run "$RUN_ID")
RC=$?
check "apply-fix runs" $RC

APPLY_R1=$(echo "$APPLY_OUT" | jq -r --arg p "$R1_PATH" '.outcomes[] | select(.path==$p) | .outcome')
APPLY_R3=$(echo "$APPLY_OUT" | jq -r --arg p "$R3_SRC" '.outcomes[] | select(.path==$p) | .outcome')
[ "$APPLY_R1" = "applied" ] && [ "$APPLY_R3" = "applied" ]
check "apply-fix applied F-R1 (edit) and F-R3 (move): R1=$APPLY_R1 R3=$APPLY_R3" $?

grep -q '^created:' "$WORK/$R1_PATH" \
  && grep -q '^updated:' "$WORK/$R1_PATH" \
  && grep -q '^tags: \[\]$' "$WORK/$R1_PATH" \
  && grep -q '^synopsis: "TODO"$' "$WORK/$R1_PATH" \
  && grep -q '^type: task$' "$WORK/$R1_PATH"
check "F-R1 gained its missing universal frontmatter from the template" $?

[ -f "$WORK/$R3_DST" ] && [ "$(sha "$WORK/$R3_DST")" = "$SHA_R3_ORIG" ]
check "F-R3 moved to $R3_DST, byte-identical to the original" $?

grep -q '^type: pointer$' "$WORK/$R3_SRC" \
  && grep -qF "Moved to $R3_DST." "$WORK/$R3_SRC"
check "pointer stub left at F-R3's old path ($R3_SRC)" $?

[ "$(sha "$WORK/$IGN_PATH")" = "$SHA_IGN_ORIG" ]
check "F-IGN untouched by apply-fix (still ignored)" $?

ADOPTION_OUT=$(run_lib query adoption)
ADOPTION_COUNT=$(echo "$ADOPTION_OUT" | jq -r '.count')
[ "$ADOPTION_COUNT" -ge 1 ]
check "an adoption_log row was written (count=$ADOPTION_COUNT)" $?

N_UNCOLLAPSED_AFTER=$(run_lib query uncollapsed | jq -r '.count')
[ "$N_UNCOLLAPSED_AFTER" -eq "$N_UNCOLLAPSED_BEFORE" ]
check "F-R5's judgment finding is untouched by apply-fix (mechanical-only): $N_UNCOLLAPSED_AFTER == $N_UNCOLLAPSED_BEFORE" $?

# --- 9. restore reverses both fixes byte-identically (spec §9.4 checks 18-19) ------------
RESTORE_R3_OUT=$(run_lib restore --by-path "$R3_SRC")
RC=$?
check "restore --by-path $R3_SRC runs" $RC
[ "$(sha "$WORK/$R3_SRC")" = "$SHA_R3_ORIG" ] && [ ! -f "$WORK/$R3_DST" ]
check "F-R3 restore is byte-identical to the original; moved copy removed" $?
echo "$RESTORE_R3_OUT" | jq -e '.reopened == true' > /dev/null
check "F-R3's finding was reopened to flagged" $?

RESTORE_R1_OUT=$(run_lib restore --by-path "$R1_PATH")
RC=$?
check "restore --by-path $R1_PATH runs" $RC
[ "$(sha "$WORK/$R1_PATH")" = "$SHA_R1_ORIG" ]
check "F-R1 restore is byte-identical to the original" $?
echo "$RESTORE_R1_OUT" | jq -e '.reopened == true' > /dev/null
check "F-R1's finding was reopened to flagged" $?

SNAP_RESTORED=$(snapshot)
diff <(echo "$SNAP_SEEDED") <(echo "$SNAP_RESTORED") > /dev/null
check "whole scratch-desk tree matches its as-seeded state after both restores" $?

# --- 10. rebuild-from-scratch reproduces the file count + checksum set (§9.4 check 7) ----
chmod -R u+w "$STORE" 2>/dev/null || true
rm -rf "$STORE"
run_lib migrate up > /dev/null 2>&1
check "rebuild: migrate up on a fresh store" $?
SWEEP3=$(run_lib sweep)
N_REBUILT=$(echo "$SWEEP3" | jq -r '.total')
[ "$N_REBUILT" -eq "$N_FILES" ]
check "rebuild reproduces the same file count ($N_REBUILT == $N_FILES)" $?
SNAP_REBUILT=$(snapshot)
diff <(echo "$SNAP_SEEDED") <(echo "$SNAP_REBUILT") > /dev/null
check "rebuild reproduces the same per-path checksum set (independent shasum oracle)" $?

# --- 11. error-exit check: restore --by-path on a path with no revision (spec's fail-loud
# discipline; not itself in the numbered §9.4 list but required by this brief) ------------
run_lib restore --by-path "does/not/exist.md" > /dev/null 2>&1
RC=$?
[ "$RC" -ne 0 ]
check "restore --by-path nonexistent exits non-zero (rc=$RC)" $?

# --- 12. interactive surface is registered (key-free: --help never opens a Session, so this
# needs no LLM provider, no API key, and no running serve) --------------------------------
./"$BIN" chat --help > /dev/null 2>&1
check "chat --help exits 0 (interactive session subcommand is registered)" $?

./"$BIN" --help 2>&1 | grep -q '^  chat '
check "root --help lists the chat command" $?

# --- 13. store-location resolution + desk open-guard (ADR 0002 §2/§3) ---------------------
# Self-contained: its own throwaway XDG home + desk roots, isolated from the main run's store.
# XDG2/DESK2/DIR3 are declared (empty) up top, next to the main WORK/XDG_DATA_HOME scratch
# dirs, so the single EXIT trap removes them too — no separate cleanup13 trap/call needed.
XDG2=$(mktemp -d "${TMPDIR:-/tmp}/deskkit-xdg2.XXXXXX")
DESK2=$(mktemp -d "${TMPDIR:-/tmp}/deskkit-desk2.XXXXXX")
DIR3=$(mktemp -d "${TMPDIR:-/tmp}/deskkit-dir3.XXXXXX")

# (i) no --dir + XDG_DATA_HOME set -> store created at $XDG/deskkit/<DESK_NAME>.
XDG_DATA_HOME="$XDG2" DESK_ROOT="$DESK2" DESK_NAME="desk-one" ./"$BIN" migrate up > /dev/null 2>&1
RC=$?
[ "$RC" -eq 0 ] && [ -d "$XDG2/deskkit/desk-one" ]
check "no --dir: store resolves to \$XDG_DATA_HOME/deskkit/<DESK_NAME>" $?

# populate a desk row so the open-guard has a stored desk to compare against.
XDG_DATA_HOME="$XDG2" DESK_ROOT="$DESK2" DESK_NAME="desk-one" ./"$BIN" sweep > /dev/null 2>&1
check "no --dir: sweep populates the XDG-resolved store" $?

# (ii) reopen the SAME physical store dir with a different DESK_NAME -> refused, error names
# both desks. DESK_NAME is itself the store's dir name, so the mismatch the guard defends
# against (ADR 0002 §3) is a copy-pasted --dir / env pointing a second desk at the first's
# store; --dir at desk-one's store while configuring "desk-two" reproduces exactly that.
STORE_ONE="$XDG2/deskkit/desk-one"
GUARD_OUT=$(DESK_ROOT="$DESK2" DESK_NAME="desk-two" ./"$BIN" query summary --dir "$STORE_ONE" 2>&1)
RC=$?
[ "$RC" -ne 0 ] \
  && echo "$GUARD_OUT" | grep -q 'desk-one' \
  && echo "$GUARD_OUT" | grep -q 'desk-two'
check "desk open-guard: mismatched DESK_NAME on the same store refused (rc=$RC), error names both desks" $?

# (iii) explicit --dir wins: store lands at the chosen dir, NOT under XDG.
XDG_DATA_HOME="$XDG2" DESK_ROOT="$DESK2" DESK_NAME="desk-three" ./"$BIN" migrate up --dir "$DIR3" > /dev/null 2>&1
RC=$?
[ "$RC" -eq 0 ] && [ -f "$DIR3/data.db" ] && [ ! -d "$XDG2/deskkit/desk-three" ]
check "--dir overrides the XDG default (store lands at the explicit dir, not XDG)" $?

# --- 14. legacy store-home auto-migration (D2b, spec §2.10) -------------------------------
# A pre-rename store at $XDG/pocket-librarian/<DESK_NAME>/ must move to $XDG/deskkit/<DESK_NAME>/
# on startup — exactly one logged line, contents intact. Fresh-XDG runs (section 13 (i)) already
# prove the no-old-home case is a silent no-op.
XDG4=$(mktemp -d "${TMPDIR:-/tmp}/deskkit-xdg4.XXXXXX")
mkdir -p "$XDG4/pocket-librarian/legacy-desk"
echo "legacy-marker" > "$XDG4/pocket-librarian/legacy-desk/marker.txt"
MIG_OUT=$(XDG_DATA_HOME="$XDG4" DESK_ROOT="$DESK2" DESK_NAME="legacy-desk" ./"$BIN" migrate up 2>&1)
RC=$?
[ "$RC" -eq 0 ] \
  && [ -d "$XDG4/deskkit/legacy-desk" ] \
  && [ ! -d "$XDG4/pocket-librarian/legacy-desk" ] \
  && [ "$(cat "$XDG4/deskkit/legacy-desk/marker.txt" 2>/dev/null)" = "legacy-marker" ] \
  && [ "$(echo "$MIG_OUT" | grep -c 'deskkit: migrated store')" -eq 1 ]
check "legacy store auto-migrated to the deskkit home (one log line; contents intact)" $?

# --- 15. content search + retrieval + orphans index/entry visibility (isolated scratch) ----
# The query surfaces added in the recent content-index work — `query search` (keyword retrieval
# over the swept file body), `query content` (fetch one file's stored body by path), and the
# orphans `--show-index` flag (reveal the by-design-unreferenced index/entry files hidden from the
# default view) — are exercised here against their OWN throwaway XDG home + desk, fully isolated
# from the main run above: no propose-fix/apply-fix ever touches these fixtures, so the
# record-original/restore reproducibility invariant (sections 9-10) is untouched. XDG5/DESK5 are
# declared (empty) up top so the single EXIT trap removes them. The store self-initializes on the
# first sweep (ADR 0003), so no explicit `migrate up` is needed here.
XDG5=$(mktemp -d "${TMPDIR:-/tmp}/deskkit-xdg5.XXXXXX")
DESK5=$(mktemp -d "${TMPDIR:-/tmp}/deskkit-desk5.XXXXXX")
mkdir -p "$DESK5/tasks"

# A fully-conformant task (no rule violations) carrying a distinctive token in its body, for the
# search-hit and content-retrieval assertions.
cat > "$DESK5/tasks/searchable.md" <<'EOF'
---
type: task
created: 2026-07-20
updated: 2026-07-20
tags: []
synopsis: "content-search fixture"
---
The librarian indexes this body so keyword search can locate the marker zephyrmarker7788 here.
EOF

# An index/entry file (basename README.md) with NO frontmatter -> empty doctype -> a structural
# orphan that the DEFAULT orphans view hides (an index doc is what other docs point at, never a
# misfiled orphan) and that --show-index opts back in.
cat > "$DESK5/README.md" <<'EOF'
Desk index page — deliberately frontmatter-free.
EOF

run5() { XDG_DATA_HOME="$XDG5" DESK_ROOT="$DESK5" DESK_NAME="verify-queries" ./"$BIN" "$@"; }

run5 sweep > /dev/null 2>&1
check "queries: sweep self-inits + indexes the isolated query desk" $?

# search: a hit for the seeded token, no hit for an absent one.
SEARCH_HIT=$(run5 query search --term zephyrmarker7788)
HIT_COUNT=$(echo "$SEARCH_HIT" | jq -r '.count')
echo "$SEARCH_HIT" | jq -e 'any(.matches[].path; . == "tasks/searchable.md")' > /dev/null
HIT_HAS_FILE=$?
[ "$HIT_COUNT" -ge 1 ] && [ "$HIT_HAS_FILE" -eq 0 ]
check "query search: seeded token hits its file (count=$HIT_COUNT, matches tasks/searchable.md)" $?

SEARCH_MISS=$(run5 query search --term absent_token_should_not_match_9999)
MISS_COUNT=$(echo "$SEARCH_MISS" | jq -r '.count')
[ "$MISS_COUNT" -eq 0 ]
check "query search: an absent term returns no matches (count=$MISS_COUNT)" $?

# content: retrieve the full stored body by path (found=true), and a missing path (found=false).
CONTENT_HIT=$(run5 query content --path tasks/searchable.md)
echo "$CONTENT_HIT" | jq -e '.found == true and (.content | contains("zephyrmarker7788"))' > /dev/null
check "query content: retrieves the stored body for a live path (found=true, body present)" $?

CONTENT_MISS=$(run5 query content --path tasks/does-not-exist.md)
echo "$CONTENT_MISS" | jq -e '.found == false' > /dev/null
check "query content: a path with no live row returns found=false" $?

# orphans: the index/entry README is hidden by default, revealed by --show-index.
ORPH_DEFAULT=$(run5 query orphans)
ORPH_DEF_COUNT=$(echo "$ORPH_DEFAULT" | jq -r '.count')
[ "$ORPH_DEF_COUNT" -eq 0 ]
check "query orphans (default) hides the index/entry README (count=$ORPH_DEF_COUNT)" $?

ORPH_SHOWN=$(run5 query orphans --show-index)
ORPH_SHOWN_COUNT=$(echo "$ORPH_SHOWN" | jq -r '.count')
echo "$ORPH_SHOWN" | jq -e 'any(.files[].path; . == "README.md")' > /dev/null
SHOWN_HAS_README=$?
[ "$ORPH_SHOWN_COUNT" -ge 1 ] && [ "$SHOWN_HAS_README" -eq 0 ]
check "query orphans --show-index reveals the index/entry README (count=$ORPH_SHOWN_COUNT)" $?

# --- done ----------------------------------------------------------------------------------
echo
echo "verify: $PASS passed, $FAIL failed ($N total)"
[ "$FAIL" -eq 0 ]
exit $?

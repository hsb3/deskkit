#!/bin/bash
# verify.sh — the pocket-librarian Phase-1 verify gate (spec §9.4), adapted to the
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
#   - `query live_files` does not expose a per-file checksum (only path/dir_kind/entity_type/
#     status/graduated_to/git_last_commit — see internal/tools/query.go fileBrief). The
#     "per-path checksum set" reproducibility check (§9.4 check 7) is therefore proven by an
#     independent shasum snapshot of the scratch desk tree (mirroring sweep's own file-walk
#     pruning), not by reading checksums back out of the CLI.
set -uo pipefail
cd "$(dirname "$0")" || exit 1

BIN="pocket-librarian"
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

WORK=$(mktemp -d "${TMPDIR:-/tmp}/pocket-librarian-verify.XXXXXX")

# Hermetic store home (ADR 0002 §2): point XDG_DATA_HOME at a throwaway scratch for the WHOLE
# run. Every store-touching command now resolves its store to
# $XDG_DATA_HOME/pocket-librarian/<DESK_NAME> when no --dir is passed, so exporting a scratch
# XDG_DATA_HOME guarantees NO check — including the new resolution/guard checks below — can ever
# touch the operator's real ~/.local/share store. Exported so every child process inherits it.
XDG_DATA_HOME=$(mktemp -d "${TMPDIR:-/tmp}/pocket-librarian-xdg.XXXXXX")
export XDG_DATA_HOME
# The main run drives the "verify-desk" desk; with no --dir its store resolves to this dir.
STORE="$XDG_DATA_HOME/pocket-librarian/verify-desk"

cleanup() {
  chmod -R u+w "$WORK" 2>/dev/null || true
  rm -rf "$WORK"
  chmod -R u+w "$XDG_DATA_HOME" 2>/dev/null || true
  rm -rf "$XDG_DATA_HOME"
}
trap cleanup EXIT

# run_lib inherits the exported scratch XDG_DATA_HOME above; it passes NO --dir on purpose, so
# the run exercises the ADR 0002 §2 XDG default-resolution path end to end.
run_lib() { DESK_ROOT="$WORK" DESK_NAME="verify-desk" ./"$BIN" "$@"; }

# Single-writer guard (spec §2.7): refuse to blow away the store out from under a live `serve`.
if [ -f .pocket-librarian.pid ] && kill -0 "$(cat .pocket-librarian.pid)" 2>/dev/null; then
  echo "FATAL: a serve process is running (pid $(cat .pocket-librarian.pid)) — run 'make stop' first." >&2
  exit 1
fi

echo "scratch desk: $WORK"
echo

# --- 0. build + clean the scratch store ---------------------------------------------------
rm -rf "$STORE"
go build -o "$BIN" ./cmd/pocket-librarian
check "build ./$BIN" $?

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
XDG2=$(mktemp -d "${TMPDIR:-/tmp}/pocket-librarian-xdg2.XXXXXX")
DESK2=$(mktemp -d "${TMPDIR:-/tmp}/pocket-librarian-desk2.XXXXXX")
DIR3=$(mktemp -d "${TMPDIR:-/tmp}/pocket-librarian-dir3.XXXXXX")
cleanup13() {
  chmod -R u+w "$XDG2" "$DESK2" "$DIR3" 2>/dev/null || true
  rm -rf "$XDG2" "$DESK2" "$DIR3"
}

# (i) no --dir + XDG_DATA_HOME set -> store created at $XDG/pocket-librarian/<DESK_NAME>.
XDG_DATA_HOME="$XDG2" DESK_ROOT="$DESK2" DESK_NAME="desk-one" ./"$BIN" migrate up > /dev/null 2>&1
RC=$?
[ "$RC" -eq 0 ] && [ -d "$XDG2/pocket-librarian/desk-one" ]
check "no --dir: store resolves to \$XDG_DATA_HOME/pocket-librarian/<DESK_NAME>" $?

# populate a desk row so the open-guard has a stored desk to compare against.
XDG_DATA_HOME="$XDG2" DESK_ROOT="$DESK2" DESK_NAME="desk-one" ./"$BIN" sweep > /dev/null 2>&1
check "no --dir: sweep populates the XDG-resolved store" $?

# (ii) reopen the SAME physical store dir with a different DESK_NAME -> refused, error names
# both desks. DESK_NAME is itself the store's dir name, so the mismatch the guard defends
# against (ADR 0002 §3) is a copy-pasted --dir / env pointing a second desk at the first's
# store; --dir at desk-one's store while configuring "desk-two" reproduces exactly that.
STORE_ONE="$XDG2/pocket-librarian/desk-one"
GUARD_OUT=$(DESK_ROOT="$DESK2" DESK_NAME="desk-two" ./"$BIN" query summary --dir "$STORE_ONE" 2>&1)
RC=$?
[ "$RC" -ne 0 ] \
  && echo "$GUARD_OUT" | grep -q 'desk-one' \
  && echo "$GUARD_OUT" | grep -q 'desk-two'
check "desk open-guard: mismatched DESK_NAME on the same store refused (rc=$RC), error names both desks" $?

# (iii) explicit --dir wins: store lands at the chosen dir, NOT under XDG.
XDG_DATA_HOME="$XDG2" DESK_ROOT="$DESK2" DESK_NAME="desk-three" ./"$BIN" migrate up --dir "$DIR3" > /dev/null 2>&1
RC=$?
[ "$RC" -eq 0 ] && [ -f "$DIR3/data.db" ] && [ ! -d "$XDG2/pocket-librarian/desk-three" ]
check "--dir overrides the XDG default (store lands at the explicit dir, not XDG)" $?

cleanup13

# --- done ----------------------------------------------------------------------------------
echo
echo "verify: $PASS passed, $FAIL failed ($N total)"
[ "$FAIL" -eq 0 ]
exit $?

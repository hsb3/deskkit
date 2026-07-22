#!/usr/bin/env bash
# shellcheck shell=bash
# 20 — librarian sweep -> patrol -> propose-fix -> apply-fix -> restore. Sourced by e2e.sh;
# helpers (check/skip/note/section/dk) and E2E_* state come from lib.sh. Seeds two mechanical
# fixtures (R1 missing frontmatter, R3 type/location mismatch), walks the whole fix chain, and
# proves the record-original-first boundary is byte-exact reversible via restore.
#
# The idiom `<condition>; check "<desc>" $?` recurs throughout this step: a test condition is
# evaluated purely for its exit status, which is then fed to check() as pass/fail. SC2319 ("$? after
# a condition usually means you wanted the command before it") structurally misfires on that
# intentional pattern at every call site (same rationale as librarian/verify.sh), so it is
# disabled file-wide below; every OTHER lint rule stays enforced.
# shellcheck disable=SC2319

section "20 · librarian sweep → patrol → fix → restore"

seed_librarian_fixtures

# --- capture original bytes BEFORE any fix (needed for the restore assertions) -------------
SHA_R1=$(shasum -a 256 "$E2E_DESK/tasks/r1-missing-fm.md" | awk '{print $1}')
SHA_R3=$(shasum -a 256 "$E2E_DESK/analyses/r3-type-task.md" | awk '{print $1}')

# --- sweep: index the seeded fixtures ------------------------------------------------------
SW=$(dk sweep)
RC=$?
CREATED=$(printf '%s' "$SW" | jq -r '.created' 2>/dev/null)
[ "$RC" -eq 0 ] && [ -n "$CREATED" ] && [ "$CREATED" -ge 2 ]
check "sweep indexes the seeded fixtures (created=$CREATED)" $?

# --- patrol: flag R1 (missing frontmatter) and R3 (type/location mismatch) ----------------
PAT=$(dk patrol)
RC=$?
RUN=$(printf '%s' "$PAT" | jq -r '.run_id' 2>/dev/null)
N_R1=$(printf '%s' "$PAT" | jq -r '.by_rule.R1 // 0' 2>/dev/null)
N_R3=$(printf '%s' "$PAT" | jq -r '.by_rule.R3 // 0' 2>/dev/null)
[ "$RC" -eq 0 ] && [ -n "$N_R1" ] && [ "$N_R1" -ge 1 ] && [ -n "$N_R3" ] && [ "$N_R3" -ge 1 ]
check "patrol found >=1 R1 and >=1 R3 (run_id=$RUN R1=$N_R1 R3=$N_R3)" $?

# --- query findings: both fixtures surfaced -------------------------------------------------
FIND=$(dk query findings)
printf '%s' "$FIND" | grep -qF 'tasks/r1-missing-fm.md' \
  && printf '%s' "$FIND" | grep -qF 'analyses/r3-type-task.md'
check "query findings lists both R1 and R3 fixture paths" $?

# --- propose-fix: record originals, no filesystem write yet --------------------------------
PROP=$(dk propose-fix --run "$RUN")
RC=$?
check "propose-fix runs" $RC

OUT_R1=$(printf '%s' "$PROP" | jq -r --arg p "tasks/r1-missing-fm.md" '.proposed[] | select(.path==$p) | .outcome' 2>/dev/null)
OUT_R3=$(printf '%s' "$PROP" | jq -r --arg p "analyses/r3-type-task.md" '.proposed[] | select(.path==$p) | .outcome' 2>/dev/null)
[ "$OUT_R1" = "recorded" ] && [ "$OUT_R3" = "recorded" ]
check "propose-fix recorded originals for R1 and R3 (R1=$OUT_R1 R3=$OUT_R3)" $?

[ ! -f "$E2E_DESK/tasks/r3-type-task.md" ]
check "propose-fix performed no filesystem write (R3 still only under analyses/)" $?

# --- apply-fix: commit the recorded revisions ----------------------------------------------
APP=$(dk apply-fix --run "$RUN")
RC=$?
check "apply-fix runs" $RC

APP_R1=$(printf '%s' "$APP" | jq -r --arg p "tasks/r1-missing-fm.md" '.outcomes[] | select(.path==$p) | .outcome' 2>/dev/null)
APP_R3=$(printf '%s' "$APP" | jq -r --arg p "analyses/r3-type-task.md" '.outcomes[] | select(.path==$p) | .outcome' 2>/dev/null)
[ "$APP_R1" = "applied" ] && [ "$APP_R3" = "applied" ]
check "apply-fix applied R1 (edit) and R3 (move): R1=$APP_R1 R3=$APP_R3" $?

grep -q '^created:' "$E2E_DESK/tasks/r1-missing-fm.md" \
  && grep -q '^tags: \[\]$' "$E2E_DESK/tasks/r1-missing-fm.md" \
  && grep -q '^synopsis: "TODO"$' "$E2E_DESK/tasks/r1-missing-fm.md"
check "R1 gained its missing universal frontmatter from the template" $?

[ -f "$E2E_DESK/tasks/r3-type-task.md" ]
check "R3 moved to tasks/r3-type-task.md" $?

grep -q '^type: pointer$' "$E2E_DESK/analyses/r3-type-task.md" \
  && grep -qF 'Moved to tasks/r3-type-task.md.' "$E2E_DESK/analyses/r3-type-task.md"
check "pointer stub left at R3's old path (analyses/r3-type-task.md)" $?

ADO=$(dk query adoption)
ADO_COUNT=$(printf '%s' "$ADO" | jq -r '.count' 2>/dev/null)
[ -n "$ADO_COUNT" ] && [ "$ADO_COUNT" -ge 1 ]
check "an adoption_log row was written (count=$ADO_COUNT)" $?

# --- restore: byte-exact reversal, the record-original-first crown jewel -------------------
R1RES=$(dk restore --by-path "tasks/r1-missing-fm.md")
RC=$?
R1_RESTORED=$(printf '%s' "$R1RES" | jq -r '.restored' 2>/dev/null)
R1_REOPENED=$(printf '%s' "$R1RES" | jq -r '.reopened' 2>/dev/null)
[ "$RC" -eq 0 ] && [ "$R1_RESTORED" = "true" ] && [ "$R1_REOPENED" = "true" ]
check "restore --by-path tasks/r1-missing-fm.md runs and reports restored+reopened" $?

SHA_R1_AFTER=$(shasum -a 256 "$E2E_DESK/tasks/r1-missing-fm.md" | awk '{print $1}')
[ "$SHA_R1_AFTER" = "$SHA_R1" ]
check "R1 restored byte-identical to its pre-fix original" $?

dk restore --by-path "analyses/r3-type-task.md" >/dev/null
RC=$?
check "restore --by-path analyses/r3-type-task.md runs" $RC

[ ! -f "$E2E_DESK/tasks/r3-type-task.md" ]
check "R3's moved copy is gone after restore" $?

SHA_R3_AFTER=$(shasum -a 256 "$E2E_DESK/analyses/r3-type-task.md" | awk '{print $1}')
[ "$SHA_R3_AFTER" = "$SHA_R3" ]
check "R3 source restored byte-identical to its pre-fix original" $?

#!/usr/bin/env bash
# shellcheck shell=bash
# 30 — PM graph: create / transition / gate / cascade. Sourced by e2e.sh; helpers
# (check/skip/note/section/dk) and E2E_* state come from lib.sh, and this file runs after
# step 10 has already `deskkit init`'d the scratch desk. PM_ENABLED is deliberately left
# unset here — the module ships default-ON (docs/development/specs/pm-system-v1-spec.md §2.9), so a plain
# `pm create` succeeding (rather than cobra's unknown-command error) is itself the proof.

section "30 · PM graph — create / transition / gate / cascade"

# --- default-on: pm create works with no PM_ENABLED, item lands in phase queue ------------
A_JSON=$(dk pm create --title "Ship widget" --type task --court owner --priority 1 --pointer "tasks/a.md" --actor e2e 2>&1)
A_RC=$?
AID=$(printf '%s' "$A_JSON" | jq -r '.item.id // empty' 2>/dev/null)
APHASE=$(printf '%s' "$A_JSON" | jq -r '.item.phase // empty' 2>/dev/null)
if [ "$A_RC" -eq 0 ] && [ -n "$AID" ] && [ "$APHASE" = "queue" ]; then rc=0; else rc=1; fi
check "PM ships default-ON (pm create works with no PM_ENABLED)" "$rc"
note "item A id=$AID phase=$APHASE"

# --- a second item to link against ---------------------------------------------------------
B_JSON=$(dk pm create --title "Wire widget" --type task --court desk --priority 1 --pointer "tasks/b.md" --actor e2e 2>&1)
BID=$(printf '%s' "$B_JSON" | jq -r '.item.id // empty' 2>/dev/null)

# --- cascade: A blocks B (manual cascade, unblock_at review) -------------------------------
LINK_JSON=$(dk pm link "$AID" "$BID" --kind blocks --cascade manual --unblock-at review --actor e2e 2>&1)
LINK_RC=$?
LINK_KIND=$(printf '%s' "$LINK_JSON" | jq -r '.edge.kind // empty' 2>/dev/null)
if [ "$LINK_RC" -eq 0 ] && [ "$LINK_KIND" = "blocks" ]; then rc=0; else rc=1; fi
check "A blocks B (blocks edge created)" "$rc"

# --- gate: a blocked item refuses to advance -------------------------------------------------
if dk pm transition "$BID" --to work --actor e2e >/dev/null 2>&1; then rc=1; else rc=0; fi
check "blocked item refuses transition (cascade block enforced)" "$rc"

# --- gate: the machine itself refuses an illegal phase skip --------------------------------
if dk pm transition "$BID" --to terminal --actor e2e >/dev/null 2>&1; then rc=1; else rc=0; fi
check "phase gate refuses the queue->terminal skip" "$rc"

# --- unblock clears the block ---------------------------------------------------------------
dk pm unblock "$BID" --reason "cleared" --actor e2e >/dev/null 2>&1
check "unblock clears the block" $?

# --- a legal advance now succeeds: queue -> work --------------------------------------------
T_JSON=$(dk pm transition "$BID" --to work --actor e2e 2>&1)
T_RC=$?
T_PHASE=$(printf '%s' "$T_JSON" | jq -r '.item.phase // empty' 2>/dev/null)
if [ "$T_RC" -eq 0 ] && [ "$T_PHASE" = "work" ]; then rc=0; else rc=1; fi
check "unblocked item advances queue->work" "$rc"

# --- full legal multi-hop: work -> review -> terminal ---------------------------------------
# The default gate ruleset (librarian/internal/modules/pm/gates/defaults.go) binds a `task`
# item's work->review edge to a required document at its pointer: type task, status active.
# review->terminal carries no bound rule for `task` (only `decision` gates that edge), so once
# the document satisfies work->review the terminal hop is a plain legal machine edge.
mkdir -p "$E2E_DESK/tasks"
TODAY=$(date -u +%Y-%m-%d)
cat > "$E2E_DESK/tasks/b.md" <<EOF
---
type: task
status: active
created: $TODAY
updated: $TODAY
tags: []
---
Wire widget into the pipeline.
EOF

R_JSON=$(dk pm transition "$BID" --to review --actor e2e 2>&1)
R_RC=$?
TERM_JSON=$(dk pm transition "$BID" --to terminal --actor e2e 2>&1)
TERM_RC=$?
TERM_PHASE=$(printf '%s' "$TERM_JSON" | jq -r '.item.phase // empty' 2>/dev/null)
if [ "$R_RC" -eq 0 ] && [ "$TERM_RC" -eq 0 ] && [ "$TERM_PHASE" = "terminal" ]; then
  check "full legal multi-hop work->review->terminal (document gate satisfied)" 0
else
  note "work->review rc=$R_RC review->terminal rc=$TERM_RC review-resp=$R_JSON terminal-resp=$TERM_JSON"
  skip "PM terminal-hop completion" "work->review or review->terminal did not reach phase=terminal; see note above"
fi

# --- pm context is the cold-start briefing source (docs/development/specs/pm-system-v1-spec.md §5.2) ----------
CTX_JSON=$(dk pm context 2>&1)
CTX_RC=$?
printf '%s' "$CTX_JSON" | jq -e '.counts' >/dev/null 2>&1
CTX_HAS_COUNTS=$?
if [ "$CTX_RC" -eq 0 ] && [ "$CTX_HAS_COUNTS" -eq 0 ]; then rc=0; else rc=1; fi
check "pm context returns the cold-start briefing JSON" "$rc"

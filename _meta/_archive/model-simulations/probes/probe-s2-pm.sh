#!/bin/bash
# Scenario 2 probe: PM gated transition (queue->work->review) with a real gate refusal, plus a
# cascaded block/unblock. Throwaway scratch desk + hermetic XDG store. PM is default-on.
# Refusals print "Error: <refusal>" to stderr and exit 1 (an expected domain outcome here).
set -uo pipefail
BIN="${1:?path to deskkit binary}"
WORK=$(mktemp -d "${TMPDIR:-/tmp}/s2-desk.XXXXXX")
XDG_DATA_HOME=$(mktemp -d "${TMPDIR:-/tmp}/s2-xdg.XXXXXX")
export XDG_DATA_HOME
cleanup() { chmod -R u+w "$WORK" "$XDG_DATA_HOME" 2>/dev/null; rm -rf "$WORK" "$XDG_DATA_HOME"; }
trap cleanup EXIT
run() { DESK_ROOT="$WORK" DESK_NAME="s2-desk" PM_ENABLED=true "$BIN" "$@"; }
pm() { run pm "$@" --actor operator; }
echo "== scratch desk: $WORK =="
run migrate up >/dev/null 2>&1; echo "migrate rc=$?"
echo "--- confirm PM default-on: pm list returns a JSON array ---"
run pm list | jq -c '{count: length}'

echo; echo "=== PART A: gated transition with a real gate refusal ==="
T=$(pm create --title "Ship widget" --type task --pointer "tasks/t1.md" --court desk | jq -r '.item.id')
echo "created task item T=$T (phase queue)"
echo "--- advance T queue->work (ungated by default) ---"
pm transition "$T" --to work | jq -c '{phase: .item.phase, version: .item.version}'

echo "--- attempt T work->review with NO doc on disk -> GATE REFUSAL ---"
OUT=$(pm transition "$T" --to review 2>&1); RC=$?; echo "rc=$RC :: $OUT"

echo "--- create tasks/t1.md at status: draft (not active) -> GATE REFUSAL (status mismatch) ---"
mkdir -p "$WORK/tasks"
cat > "$WORK/tasks/t1.md" <<'EOF'
---
type: task
status: draft
created: 2026-07-15
updated: 2026-07-15
tags: []
synopsis: "the task doc"
---
work to do
EOF
OUT=$(pm transition "$T" --to review 2>&1); RC=$?; echo "rc=$RC :: $OUT"

echo "--- flip doc to status: active -> work->review SUCCEEDS ---"
cat > "$WORK/tasks/t1.md" <<'EOF'
---
type: task
status: active
created: 2026-07-15
updated: 2026-07-15
tags: []
synopsis: "the task doc"
---
work to do
EOF
pm transition "$T" --to review | jq -c '{phase: .item.phase, version: .item.version}'
echo "--- review->terminal (ungated for a task; decision items WOULD gate here) ---"
pm transition "$T" --to terminal | jq -c '{phase: .item.phase, status_label: .item.status_label}'
echo "--- audit trail on T (newest first): advance(terminal), advance(review), gate_refused x2, advance(work) ---"
pm get "$T" | jq -c '[.recent_transitions[] | {event, from, to}]'

echo; echo "=== PART B: cascade block/unblock ==="
B=$(pm create --title "Blocker" --type task --court desk | jq -r '.item.id')
G=$(pm create --title "Gated by B" --type task --court desk | jq -r '.item.id')
echo "created B=$B  G=$G"
echo "--- link B blocks G (unblock_at=work, cascade=auto) -> G initial-blocks (B in queue < work) ---"
pm link "$B" "$G" --kind blocks --unblock-at work --cascade auto | jq -c '.edge'
echo "--- G blocked right after link (expect true) ---"
pm get "$G" | jq -c '{blocked: (.blocked // false)}'
echo "--- advance B queue->work (B reaches unblock_at=work) -> cascade auto-unblocks G ---"
pm transition "$B" --to work | jq -c '{B_phase: .item.phase}'
echo "--- G blocked after B reaches unblock_at (expect false) ---"
pm get "$G" | jq -c '{blocked: (.blocked // false)}'
echo "--- G audit trail (newest first): unblock then block ---"
pm get "$G" | jq -c '[.recent_transitions[] | {event, actor_kind}]'
echo "== done =="

#!/bin/bash
# Scenario 1 probe: sweep -> patrol -> propose_fix -> apply_fix -> restore + findings disposition.
# Throwaway scratch desk under mktemp, hermetic XDG store (verify.sh pattern). Never a real desk.
set -uo pipefail
BIN="${1:?path to deskkit binary}"
WORK=$(mktemp -d "${TMPDIR:-/tmp}/s1-desk.XXXXXX")
XDG_DATA_HOME=$(mktemp -d "${TMPDIR:-/tmp}/s1-xdg.XXXXXX")
export XDG_DATA_HOME
cleanup() { chmod -R u+w "$WORK" "$XDG_DATA_HOME" 2>/dev/null; rm -rf "$WORK" "$XDG_DATA_HOME"; }
trap cleanup EXIT
run() { DESK_ROOT="$WORK" DESK_NAME="s1-desk" "$BIN" "$@"; }
echo "== scratch desk: $WORK =="

mkdir -p "$WORK/tasks" "$WORK/analyses"
cat > "$WORK/tasks/needs-fm.md" <<'EOF'
---
type: task
---
body line
EOF
cat > "$WORK/analyses/misfiled.md" <<'EOF'
---
type: task
created: 2026-07-15
updated: 2026-07-15
tags: []
synopsis: "misfiled fixture"
---
body
EOF

echo "--- 1. migrate up (self-init also works; explicit here) ---"
run migrate up >/dev/null 2>&1; echo "rc=$?"
echo "--- 2. sweep ---"
run sweep | jq -c '{total,created}'
echo "--- 3. patrol ---"
PATROL=$(run patrol); echo "$PATROL" | jq -c '{run_id,by_rule}'
RUN_ID=$(echo "$PATROL" | jq -r '.run_id')

echo "--- 4. query findings (default disposition=open) — NOTE: output carries NO finding id ---"
run query findings | jq -c '.'
echo "    (findings dispose <id> requires a record id; no query kind on this surface emits one)"

echo "--- 5. propose-fix --run $RUN_ID (record-original-first; no fs write) ---"
run propose-fix --run "$RUN_ID" | jq -c '[.proposed[] | {path, outcome}]'
echo "R1 file unchanged on disk after propose (expect created key absent -> 0):"
grep -c '^created:' "$WORK/tasks/needs-fm.md"

echo "--- 6. apply-fix --run $RUN_ID (the write path) ---"
run apply-fix --run "$RUN_ID" | jq -c '[.outcomes[] | {path, outcome}]'
echo "R1 needs-fm.md now has created key (expect 1):"; grep -c '^created:' "$WORK/tasks/needs-fm.md"
echo "R3 moved to tasks/misfiled.md:"; [ -f "$WORK/tasks/misfiled.md" ] && echo yes || echo no
echo "R3 old path left a pointer stub (expected move behavior):"; [ -f "$WORK/analyses/misfiled.md" ] && head -1 "$WORK/analyses/misfiled.md"

echo "--- 7. restore --by-path tasks/needs-fm.md (byte-exact reversal) ---"
run restore --by-path "tasks/needs-fm.md" | jq -c '{path, restored}' 2>/dev/null || echo "(restore ran)"
echo "R1 created key after restore (expect 0 = original bytes back):"; grep -c '^created:' "$WORK/tasks/needs-fm.md"
echo "== done =="

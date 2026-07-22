#!/bin/bash
# D3 probe: an unvalidated items.type silently disables the document gate the correct type enforces.
set -uo pipefail
BIN="${1:?}"
WORK=$(mktemp -d "${TMPDIR:-/tmp}/d3.XXXXXX"); XDG_DATA_HOME=$(mktemp -d "${TMPDIR:-/tmp}/d3-xdg.XXXXXX"); export XDG_DATA_HOME
cleanup(){ chmod -R u+w "$WORK" "$XDG_DATA_HOME" 2>/dev/null; rm -rf "$WORK" "$XDG_DATA_HOME"; }
trap cleanup EXIT
run(){ DESK_ROOT="$WORK" DESK_NAME="d3" PM_ENABLED=true "$BIN" "$@"; }
pm(){ run pm "$@" --actor operator; }
run migrate up >/dev/null 2>&1

echo "### Control: a correctly-typed 'task' item is gated at work->review (no doc -> refused)"
T=$(pm create --title good --type task --pointer tasks/g.md --court desk | jq -r '.item.id')
pm transition "$T" --to work >/dev/null
OUT=$(pm transition "$T" --to review 2>&1); echo "task rc=$?  :: $OUT"

echo
echo "### D3: a typo'd type 'tsak' (should be 'task') accepted verbatim, gate does NOT bind -> advances UNGATED"
X=$(pm create --title typo --type tsak --pointer tasks/g.md --court desk | jq -r '.item.id')
echo "created item type field: $(pm get "$X" | jq -r '.type')"
pm transition "$X" --to work >/dev/null
OUT=$(pm transition "$X" --to review 2>&1); RC=$?
echo "tsak rc=$RC  :: $(echo "$OUT" | jq -c '{phase: .item.phase}' 2>/dev/null || echo "$OUT")"
echo "(no doc on disk exists at tasks/g.md, yet the typo'd item reached review)"
[ -f "$WORK/tasks/g.md" ] && echo "doc exists" || echo "confirmed: tasks/g.md does NOT exist"

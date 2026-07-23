#!/bin/bash
# Does an item created with NO type bypass the document gate a typed item would enforce?
set -uo pipefail
BIN="${1:?}"
WORK=$(mktemp -d "${TMPDIR:-/tmp}/d3c.XXXXXX"); XDG_DATA_HOME=$(mktemp -d "${TMPDIR:-/tmp}/d3c-xdg.XXXXXX"); export XDG_DATA_HOME
cleanup(){ chmod -R u+w "$WORK" "$XDG_DATA_HOME" 2>/dev/null; rm -rf "$WORK" "$XDG_DATA_HOME"; }
trap cleanup EXIT
run(){ DESK_ROOT="$WORK" DESK_NAME="d3c" PM_ENABLED=true "$BIN" "$@"; }
pm(){ run pm "$@" --actor operator; }
run migrate up >/dev/null 2>&1
echo "### create with NO --type, then try to update type to a typo"
X=$(pm create --title notype --court desk --pointer tasks/g.md | jq -r '.item.id')
echo "no-type item id: $X ; type field: '$(pm get "$X" | jq -r '.type // ""')'"
pm transition "$X" --to work >/dev/null
echo "--- no-type item work->review (task gate needs a doc; no type binds no gate) ---"
OUT=$(pm transition "$X" --to review 2>&1); RC=$?
echo "rc=$RC :: $(echo "$OUT" | jq -c '{phase: .item.phase}' 2>/dev/null || echo "$OUT")"
[ -f "$WORK/tasks/g.md" ] && echo "doc exists" || echo "tasks/g.md does NOT exist"
echo
echo "### can pm update set type to a typo after creation?"
pm update "$X" --type tsak 2>&1 | head -3; echo "update rc=${PIPESTATUS[0]}"

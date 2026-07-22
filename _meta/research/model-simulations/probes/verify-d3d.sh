#!/bin/bash
# Confirm the create-vs-update asymmetry: create rejects an unknown type; update accepts it.
set -uo pipefail
BIN="${1:?}"
WORK=$(mktemp -d "${TMPDIR:-/tmp}/d3d.XXXXXX"); XDG_DATA_HOME=$(mktemp -d "${TMPDIR:-/tmp}/d3d-xdg.XXXXXX"); export XDG_DATA_HOME
cleanup(){ chmod -R u+w "$WORK" "$XDG_DATA_HOME" 2>/dev/null; rm -rf "$WORK" "$XDG_DATA_HOME"; }
trap cleanup EXIT
run(){ DESK_ROOT="$WORK" DESK_NAME="d3d" PM_ENABLED=true "$BIN" "$@"; }
pm(){ run pm "$@" --actor operator; }
run migrate up >/dev/null 2>&1
X=$(pm create --title x --type task --court desk | jq -r '.item.id')
echo "create --type task -> type is now: $(pm get "$X" | jq -r '.type')"
echo "create --type tsak (rejected):"; pm create --title y --type tsak --court desk 2>&1 | head -1
echo "update the valid item --type tsak -> type is now: $(pm update "$X" --type tsak | jq -r '.item.type')"

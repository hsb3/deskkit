#!/bin/bash
set -uo pipefail
BIN="${1:?}"
WORK=$(mktemp -d "${TMPDIR:-/tmp}/vf.XXXXXX"); XDG_DATA_HOME=$(mktemp -d "${TMPDIR:-/tmp}/vf-xdg.XXXXXX"); export XDG_DATA_HOME
cleanup(){ chmod -R u+w "$WORK" "$XDG_DATA_HOME" 2>/dev/null; rm -rf "$WORK" "$XDG_DATA_HOME"; }
trap cleanup EXIT
run(){ DESK_ROOT="$WORK" DESK_NAME="vf" PM_ENABLED=true "$BIN" "$@"; }
pm(){ run pm "$@" --actor operator; }
run migrate up >/dev/null 2>&1

echo "### F2: a file with NO frontmatter at all"
mkdir -p "$WORK/notes"
printf 'loose note, no frontmatter\n' > "$WORK/notes/loose.md"
printf -- '---\ntype: task\n---\nx\n' > "$WORK/notes/typed-nofm.md"   # has type, missing universal keys
run sweep >/dev/null 2>&1
echo "patrol by_rule:"; run patrol | jq -c '.by_rule'
echo "any finding whose path is notes/loose.md? ->"; run query findings | jq -c '[.by_rule[][] | select(.path=="notes/loose.md")]'
echo "query orphans (does it surface loose.md?):"; run query orphans | jq -c '{count, paths: [.files[]?.path]}'
echo "how sweep indexed loose.md (dir_kind / entity_type):"; run query live_files | jq -c '[.files[] | select(.path=="notes/loose.md")]'

echo
echo "### F1: exact actor fields on a cascade-driven unblock row"
B=$(pm create --title B --type task --court desk | jq -r '.item.id')
G=$(pm create --title G --type task --court desk | jq -r '.item.id')
pm link "$B" "$G" --kind blocks --unblock-at work --cascade auto >/dev/null
pm transition "$B" --to work >/dev/null
echo "G recent_transitions (full rows):"; pm get "$G" | jq -c '.recent_transitions[] | {event, actor, actor_kind, detail}'

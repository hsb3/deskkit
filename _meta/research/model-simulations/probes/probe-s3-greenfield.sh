#!/bin/bash
# Scenario 3 (greenfield) SCRIPTED TAIL only: a freshly-scaffolded, conformant desk through its
# first sweep -> patrol -> PM item creation. The scaffold copy + template_render placeholder
# materialization (TS/MCP, K23 steps 3-7) is tabletop-only (see the walkthrough doc). Throwaway
# desk; hermetic XDG store.
set -uo pipefail
BIN="${1:?path to deskkit binary}"
WORK=$(mktemp -d "${TMPDIR:-/tmp}/s3-desk.XXXXXX")
XDG_DATA_HOME=$(mktemp -d "${TMPDIR:-/tmp}/s3-xdg.XXXXXX")
export XDG_DATA_HOME
cleanup() { chmod -R u+w "$WORK" "$XDG_DATA_HOME" 2>/dev/null; rm -rf "$WORK" "$XDG_DATA_HOME"; }
trap cleanup EXIT
run() { DESK_ROOT="$WORK" DESK_NAME="s3-desk" PM_ENABLED=true "$BIN" "$@"; }
echo "== scratch desk: $WORK =="

# Minimal freshly-scaffolded desk shape (a stand-in for what desk-setup K23 materializes):
# CLAUDE.md (orient), _meta/README + HANDOFF, _structure/decisions/README. Conformant frontmatter.
cat > "$WORK/CLAUDE.md" <<'EOF'
---
type: home
status: final
created: 2026-07-21
updated: 2026-07-21
tags: [desk]
synopsis: "Desk entry brief — orients only."
---
# Desk
Orient only.
EOF
mkdir -p "$WORK/_meta" "$WORK/_structure/decisions"
cat > "$WORK/_meta/README.md" <<'EOF'
---
type: readme
status: final
created: 2026-07-21
updated: 2026-07-21
tags: [meta]
synopsis: "The _meta working-desk taxonomy."
---
# _meta
EOF
cat > "$WORK/_meta/HANDOFF.md" <<'EOF'
---
type: home
status: final
created: 2026-07-21
updated: 2026-07-21
tags: [handoff]
synopsis: "Cold-start bridge."
---
# HANDOFF
EOF
cat > "$WORK/_structure/decisions/README.md" <<'EOF'
---
type: readme
status: final
created: 2026-07-21
updated: 2026-07-21
tags: [decisions]
synopsis: "Decision spine index, newest first."
---
# Decisions
EOF

echo "--- self-init: first store-touching command on a never-migrated store ---"
run query summary | jq -c '{files_total, open_findings_total}'
echo "--- first sweep on the fresh desk ---"
run sweep | jq -c '{total, created}'
echo "--- first patrol (a conformant scaffold should be clean or near-clean) ---"
run patrol | jq -c '{run_id, findings_new, by_rule}'
echo "--- first PM item creation (a real greenfield first action) ---"
FIRST=$(run pm create --title "Stand up the desk" --type task --court desk --actor operator | jq -r '.item.id')
echo "created first item: $FIRST"
run pm context | jq -c '{active: (.active | length), blocked: (.blocked | length)}'
echo "== done =="

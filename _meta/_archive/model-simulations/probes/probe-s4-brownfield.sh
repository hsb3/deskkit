#!/bin/bash
# Scenario 4 (brownfield) SCRIPTED PHASE 8 only: the librarian baseline gate — migrate up ->
# sweep -> patrol into a FRESH store over a messy, non-conformant desk, recording the patrol run
# id and store path (brownfield-adoption Phase 8). Phases 1-7 and 9 (lock, inventory, disposition
# table, approval gate, init, migrate, author instruments, take stock) are tabletop-only (human
# judgment + template_render); see the walkthrough doc. Throwaway desk; hermetic XDG store.
set -uo pipefail
BIN="${1:?path to deskkit binary}"
WORK=$(mktemp -d "${TMPDIR:-/tmp}/s4-desk.XXXXXX")
XDG_DATA_HOME=$(mktemp -d "${TMPDIR:-/tmp}/s4-xdg.XXXXXX")
export XDG_DATA_HOME
cleanup() { chmod -R u+w "$WORK" "$XDG_DATA_HOME" 2>/dev/null; rm -rf "$WORK" "$XDG_DATA_HOME"; }
trap cleanup EXIT
run() { DESK_ROOT="$WORK" DESK_NAME="s4-desk" "$BIN" "$@"; }
echo "== scratch desk (messy pre-existing folder): $WORK =="

# A messy brownfield desk: mixed debt the baseline patrol should surface.
mkdir -p "$WORK/tasks" "$WORK/analyses" "$WORK/notes"
# (a) a task doc missing universal frontmatter -> R1
printf -- '---\ntype: task\n---\ndo the thing\n' > "$WORK/tasks/legacy-task.md"
# (b) a task doc mis-filed under analyses/ -> R3
printf -- '---\ntype: task\ncreated: 2025-01-01\nupdated: 2025-01-01\ntags: []\nsynopsis: "x"\n---\nbody\n' > "$WORK/analyses/should-be-a-task.md"
# (c) a decision doc with an invalid/empty status -> R4 (judgment, no auto-fix)
printf -- '---\ntype: decision\nstatus: \ncreated: 2025-01-01\nupdated: 2025-01-01\ntags: []\nsynopsis: "d"\n---\nbody\n' > "$WORK/_decision.md" 2>/dev/null || true
mkdir -p "$WORK/_structure/decisions"
printf -- '---\ntype: decision\nstatus: maybe\ncreated: 2025-01-01\nupdated: 2025-01-01\ntags: []\nsynopsis: "d"\n---\nbody\n' > "$WORK/_structure/decisions/0001-thing.md"
# (d) a free-form note with no frontmatter at all -> R1
printf -- 'just a loose note, no frontmatter\n' > "$WORK/notes/loose.md"

echo "--- Phase 8 baseline: migrate up (fresh store) ---"
run migrate up >/dev/null 2>&1; echo "migrate rc=$?"
STORE="$XDG_DATA_HOME/deskkit/s4-desk"
echo "recorded store path: $STORE"; [ -d "$STORE" ] && echo "store dir exists: yes"
echo "--- Phase 8 baseline: sweep ---"
run sweep | jq -c '{total, created}'
echo "--- Phase 8 baseline: patrol (record run id) ---"
PATROL=$(run patrol); echo "$PATROL" | jq -c '{run_id, findings_new, by_rule, by_severity}'
echo "recorded patrol run id: $(echo "$PATROL" | jq -r '.run_id')"
echo "--- triage view: mechanical (auto-fixable) vs judgment (stays flagged for a human) ---"
run query summary | jq -c '{open_findings_total, by_severity: .open_findings_by_severity, by_rule: .open_findings_by_rule}'
echo "== done =="

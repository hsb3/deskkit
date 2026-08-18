#!/bin/bash
# examples/agent-loop.sh — Manual, REAL-LLM walkthrough of the deskkit agent loop (`deskkit agent`,
# an actual LLM autonomously choosing tools via the eino ReAct loop) and the librarian MCP
# server (`deskkit mcp-serve`) driven as a real multi-message JSON-RPC session. NEEDS A REAL
# ANTHROPIC_API_KEY, MAKES REAL LLM CALLS (billed), and is intentionally NOT wired into
# `make check` / `make verify` / CI — an operator runs this by hand. Mirrors the numbered
# PASS/FAIL, throwaway-mktemp-desk style of verify.sh and examples/pm-walkthrough.sh.
#
# What it exercises, end to end against a throwaway scratch desk (never a real one):
#   1. builds deskkit fresh (or reuses $DESKKIT_BIN to skip the build on repeat runs)
#   2. seeds five small fixtures designed to trip R1/R2/R3/R4/R5 (own design, distinct from
#      verify.sh's F-R1/F-R3/F-R5/F-IGN — see the comment above each fixture below)
#   3. sweep + patrol against those fixtures, asserting the expected finding counts
#   4. a REAL `deskkit agent` run (no LIBRARIAN_AUTONOMOUS_WRITES) asked to sweep/patrol/
#      propose fixes — asserts no fixture file is mutated (apply_fix is structurally absent
#      from the agent's tool slice when the env var is unset, per toolcore.AgentTools/§5.4)
#   5. a REAL `deskkit agent` run WITH LIBRARIAN_AUTONOMOUS_WRITES=true asked to apply
#      mechanical fixes — asserts a real fix lands on disk, then independently proves the
#      record-original-first boundary by round-tripping `deskkit restore --by-path`
#      byte-exact back to the pre-fix original (sha256 comparison)
#   6. a real multi-message MCP protocol session (`deskkit mcp-serve`) for both the 5-tool
#      default surface and the 6-tool LIBRARIAN_AUTONOMOUS_WRITES=true surface: initialize ->
#      notifications/initialized -> tools/list (asserts the exact count) -> sequential
#      tools/call invocations with real arguments against a shared persistent store
#
# AGENT_MAX_STEP is raised to 30 here (default is 12): dogfooding found the default tight —
# a simple sweep+patrol+propose_fix instruction can burn through all 12 ReAct graph-node
# steps on reasoning/query calls before ever reaching a terminal answer, erroring with
# "[GraphRunError] exceeds max steps" even though the tool sequence chosen was sane.
#
# --dev=false is passed on every invocation below: building the binary into a mktemp WORK
# dir (as this script does) trips PocketBase's own go-run heuristic and silently turns on
# SQL debug logging to stdout, which corrupts JSON output (and would corrupt a real MCP
# client's stdio JSON-RPC stream the same way) — see the run_desk() comment for detail.
#
# Usage: bash examples/agent-loop.sh   (from anywhere; it cd's to the repo root)
#        DESKKIT_BIN=/path/to/deskkit bash examples/agent-loop.sh   (skip the build)
#
# Every check is a real assertion against real command output — nothing here is a
# self-report. Non-zero exit if any check fails.

set -uo pipefail
cd "$(dirname "$0")/.." || exit 1

if [ -z "${ANTHROPIC_API_KEY:-}" ]; then
  echo "FATAL: ANTHROPIC_API_KEY is not set. This script makes real LLM calls and needs a" >&2
  echo "real key, e.g.: export ANTHROPIC_API_KEY=\$(secret get ANTHROPIC_API_KEY)" >&2
  exit 1
fi

PASS=0
FAIL=0
N=0
check() { # check <description> <rc: 0=pass, nonzero=fail>
  N=$((N + 1))
  if [ "$2" -eq 0 ]; then
    printf '[%02d] PASS  %s\n' "$N" "$1"
    PASS=$((PASS + 1))
  else
    printf '[%02d] FAIL  %s\n' "$N" "$1"
    FAIL=$((FAIL + 1))
  fi
}

sha() { shasum -a 256 "$1" | awk '{print $1}'; }

WORK=$(mktemp -d)
trap 'rm -rf "$WORK"' EXIT

DESK="$WORK/desk"
STORE="$WORK/store"
mkdir -p "$DESK/tasks" "$DESK/journal" "$DESK/analyses" "$DESK/_structure/decisions" "$DESK/_knowledge" "$STORE"

if [ -n "${DESKKIT_BIN:-}" ]; then
  BIN="$DESKKIT_BIN"
  if [ -x "$BIN" ]; then rc=0; else rc=1; fi
  check "reuse existing binary (DESKKIT_BIN=$BIN)" "$rc"
else
  BIN="$WORK/deskkit"
  go build -o "$BIN" ./cmd/deskkit
  check "build $BIN from ./cmd/deskkit" $?
fi

# --- profile (secrets_ref points the agent at the real key already exported above) --------
cat > "$DESK/_knowledge/profile.yaml" <<'EOF'
schema_version: 1
identity:
  name: "Dogfood Tester"
  github:
    personal: "dogfood-personal"
    work: "dogfood-work"
  email: "dogfood@example.com"
repos:
  default: "dogfood/scratch-repo"
  by_role:
    product: "dogfood/scratch-product"
    incubation: "dogfood/scratch-incubation"
  shorthand:
    issue_default: "dogfood/scratch-repo"
board:
  provider: "github-projects"
  url: "https://example.com/board"
  number: 0
desk:
  name: "dogfood-agent"
  root: "."
  paths:
    decisions: "_structure/decisions"
    tasks: "tasks"
    analyses: "analyses"
    journal: "journal"
    secrets: "_meta/secrets"
    handoff: "_meta/HANDOFF.md"
    knowledge: "_knowledge"
machines:
  - name: "dogfood-host"
    role: "primary"
    projects_root: "."
models:
  provider: "anthropic"
  model: "claude-haiku-4-5-20251001"
  alternates: ["claude-sonnet-5", "claude-opus-4-8"]
secrets_ref:
  llm_api_key: "ANTHROPIC_API_KEY"
preferences:
  commit_style: "conventional"
  register: "terse"
custom: {}
EOF

# --- seed fixtures --------------------------------------------------------------------------

# R1 (partial): has type+created, missing updated/tags/synopsis.
cat > "$DESK/tasks/dogfood-r1-partial-fm.md" <<'EOF'
---
type: task
created: 2026-07-18
---
R1 fixture: only type/created present, so R1 should report exactly three
missing keys (updated, tags, synopsis).
EOF

# R2: journal dir, filename doesn't match yyyy-mm-dd-*.md.
cat > "$DESK/journal/dogfood-meeting-notes.md" <<'EOF'
---
type: journal
created: 2026-07-18
updated: 2026-07-18
tags: []
synopsis: "R2 fixture: wrong filename shape"
---
R2 fixture body.
EOF

# R3 (reverse direction from verify.sh's fixture): type: analysis filed under tasks/.
cat > "$DESK/tasks/dogfood-analysis-misfiled.md" <<'EOF'
---
type: analysis
created: 2026-07-18
updated: 2026-07-18
tags: []
synopsis: "R3 fixture: analysis doc filed under tasks/"
---
R3 fixture body.
EOF

# R5 via the frontmatter graduated_to: key (not verify.sh's inline "graduated to:" text form).
{
  cat <<'EOF'
---
type: analysis
created: 2026-07-18
updated: 2026-07-18
tags: []
synopsis: "R5 fixture: graduated via frontmatter key"
graduated_to: "77"
---
EOF
  for i in $(seq 1 45); do printf 'padding line %s\n' "$i"; done
} > "$DESK/analyses/dogfood-graduated-via-frontmatter.md"

# R4 (invalid decision status) + R1 (missing tags/synopsis), filed under the default-ignored
# _structure/decisions/ prefix -> the R1 finding is mechanical but must be REFUSED by
# propose_fix (ignore boundary); R4 is judgment-only and files regardless of the ignore list.
cat > "$DESK/_structure/decisions/9997-dogfood-ignored-decision.md" <<'EOF'
---
type: decision
created: 2026-07-18
updated: 2026-07-18
status: bogus
---
R4+R1 fixture, deliberately on the ignore-listed decisions/ prefix.
EOF

check "seeded 5 fixtures (R1 partial, R2, R3 reverse, R5 frontmatter marker, R4+ignored-R1)" $?

export XDG_DATA_HOME="$WORK/xdg-home"
export DESK_ROOT="$DESK"
export DESK_NAME="dogfood-agent"
export AGENT_MAX_STEP=30

# --dev=false: without this, a binary built into a mktemp WORK dir (as here) trips
# PocketBase's own `IsProbablyGoRun()` heuristic (DefaultDev in cmd/deskkit/main.go), which
# auto-enables dev-mode SQL query logging to STDOUT. That logging then corrupts any
# stdout-JSON consumer of this script (and, worse, would corrupt deskkit mcp-serve's
# newline-delimited JSON-RPC stdio transport for a real MCP client under the same
# condition) — found by dogfooding this exact script. Force it off explicitly.
run_desk() { "$BIN" --dir "$STORE" --dev=false "$@"; }

# --- sweep + patrol --------------------------------------------------------------------

run_desk sweep > /dev/null; RC=$?
check "sweep runs" $RC
PATROL_OUT=$(run_desk patrol); RC=$?
check "patrol runs" $RC
N_R1=$(echo "$PATROL_OUT" | jq -r '.by_rule.R1 // 0')
N_R2=$(echo "$PATROL_OUT" | jq -r '.by_rule.R2 // 0')
N_R3=$(echo "$PATROL_OUT" | jq -r '.by_rule.R3 // 0')
N_R4=$(echo "$PATROL_OUT" | jq -r '.by_rule.R4 // 0')
N_R5=$(echo "$PATROL_OUT" | jq -r '.by_rule.R5 // 0')
if [ "$N_R1" -eq 2 ] && [ "$N_R2" -eq 1 ] && [ "$N_R3" -eq 1 ] && [ "$N_R4" -eq 1 ] && [ "$N_R5" -eq 1 ]; then rc=0; else rc=1; fi
check "patrol found exactly the designed findings (R1=$N_R1 R2=$N_R2 R3=$N_R3 R4=$N_R4 R5=$N_R5)" "$rc"

echo
echo "--- patrol findings detail ---"
run_desk query findings

# --- real agent run, NO autonomous writes ------------------------------------------------

SHA_BEFORE_R1=$(sha "$DESK/tasks/dogfood-r1-partial-fm.md")
SHA_BEFORE_R2=$(sha "$DESK/journal/dogfood-meeting-notes.md")
SHA_BEFORE_R3=$(sha "$DESK/tasks/dogfood-analysis-misfiled.md")

unset LIBRARIAN_AUTONOMOUS_WRITES
echo
echo "--- deskkit agent (no autonomous writes) ---"
AGENT_OUT_1=$(run_desk agent "sweep the desk, patrol it for rule violations, and propose fixes for anything mechanical you find" 2>&1)
AGENT_RC_1=$?
echo "$AGENT_OUT_1"
check "agent run (no writes) exits 0" $AGENT_RC_1

if [ "$(sha "$DESK/tasks/dogfood-r1-partial-fm.md")" = "$SHA_BEFORE_R1" ] \
  && [ "$(sha "$DESK/journal/dogfood-meeting-notes.md")" = "$SHA_BEFORE_R2" ] \
  && [ "$(sha "$DESK/tasks/dogfood-analysis-misfiled.md")" = "$SHA_BEFORE_R3" ]; then rc=0; else rc=1; fi
check "no fixture file was mutated (apply_fix is structurally absent from the agent's tool slice)" "$rc"

# --- real agent run WITH autonomous writes, then restore round-trip ---------------------

export LIBRARIAN_AUTONOMOUS_WRITES=true
echo
echo "--- deskkit agent (LIBRARIAN_AUTONOMOUS_WRITES=true) ---"
AGENT_OUT_2=$(run_desk agent "apply any mechanical fixes you can safely make" 2>&1)
AGENT_RC_2=$?
echo "$AGENT_OUT_2"
check "agent run (autonomous writes) exits 0" $AGENT_RC_2

SHA_AFTER_R1=$(sha "$DESK/tasks/dogfood-r1-partial-fm.md" 2>/dev/null || echo "MISSING")
if [ "$SHA_AFTER_R1" != "$SHA_BEFORE_R1" ]; then rc=0; else rc=1; fi
check "a real mechanical fix landed on disk (R1 file content changed)" "$rc"

RESTORE_OUT=$(run_desk restore --by-path "tasks/dogfood-r1-partial-fm.md"); RESTORE_RC=$?
echo "restore: $RESTORE_OUT"
check "restore --by-path runs" $RESTORE_RC
SHA_RESTORED=$(sha "$DESK/tasks/dogfood-r1-partial-fm.md")
if [ "$SHA_RESTORED" = "$SHA_BEFORE_R1" ]; then rc=0; else rc=1; fi
check "restore is byte-exact back to the pre-fix original (sha256 match)" "$rc"
unset LIBRARIAN_AUTONOMOUS_WRITES

# --- real multi-message MCP protocol session, both tool surfaces ------------------------
#
# Framing per docs/development/specs/tool-surface.md "How the counts were derived": one-shot init+initialized+
# call per invocation against the SAME persistent --dir store, so state (files/findings/
# revisions) carries across calls even though each call is its own process — this is the
# same pattern examples/pm-walkthrough.sh uses for its MCP session. A trailing `sleep 1` keeps stdin open
# long enough for the response to flush before EOF races the server shutdown.

INIT_REQ='{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"dogfood-agent","version":"0"}}}'
INITD_REQ='{"jsonrpc":"2.0","method":"notifications/initialized"}'

mcp_call() { # mcp_call <store> <tool-name> <json-args> [env KEY=VAL ...]
  local store="$1" tool="$2" args="$3"; shift 3
  local call_req
  call_req=$(jq -n --arg name "$tool" --argjson args "$args" \
    '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":$name,"arguments":$args}}')
  { printf '%s\n%s\n%s\n' "$INIT_REQ" "$INITD_REQ" "$call_req"; sleep 1; } \
    | env "$@" "$BIN" --dir "$store" --dev=false mcp-serve 2>>"$WORK/mcp-stderr.log" \
    | jq -c 'select(.id == 2)'
}

mcp_list() { # mcp_list <store> [env KEY=VAL ...]
  local store="$1"; shift
  local list_req='{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}'
  { printf '%s\n%s\n%s\n' "$INIT_REQ" "$INITD_REQ" "$list_req"; sleep 1; } \
    | env "$@" "$BIN" --dir "$store" --dev=false mcp-serve 2>>"$WORK/mcp-stderr.log" \
    | jq -c 'select(.id == 2)'
}

MCPSTORE5="$WORK/mcp-store-5tool"
MCPSTORE6="$WORK/mcp-store-6tool"
MCPDESK="$WORK/mcp-desk"
mkdir -p "$MCPSTORE5" "$MCPSTORE6" "$MCPDESK/tasks" "$MCPDESK/_knowledge"
cp "$DESK/_knowledge/profile.yaml" "$MCPDESK/_knowledge/profile.yaml"
cat > "$MCPDESK/tasks/mcp-fixture.md" <<'EOF'
---
type: task
---
MCP session fixture, missing most universal frontmatter keys.
EOF

echo
echo "--- MCP session: 5-tool default surface ---"
LIST5=$(mcp_list "$MCPSTORE5" DESK_ROOT="$MCPDESK" DESK_NAME=mcp-probe-5 MCP_MODULES=librarian); RC=$?
echo "tools/list: $LIST5"
COUNT5=$(printf '%s' "$LIST5" | jq -r '.result.tools | length')
if [ "$RC" -eq 0 ] && [ "$COUNT5" -eq 5 ]; then rc=0; else rc=1; fi
check "5-tool surface: tools/list returns exactly 5 tools (got $COUNT5)" "$rc"

SWEEP5=$(mcp_call "$MCPSTORE5" sweep '{}' DESK_ROOT="$MCPDESK" DESK_NAME=mcp-probe-5 MCP_MODULES=librarian); RC=$?
echo "sweep: $SWEEP5"
check "5-tool surface: sweep call succeeds" $RC

PATROL5=$(mcp_call "$MCPSTORE5" patrol '{}' DESK_ROOT="$MCPDESK" DESK_NAME=mcp-probe-5 MCP_MODULES=librarian); RC=$?
echo "patrol: $PATROL5"
check "5-tool surface: patrol call succeeds" $RC

QUERY5=$(mcp_call "$MCPSTORE5" query '{"kind":"findings"}' DESK_ROOT="$MCPDESK" DESK_NAME=mcp-probe-5 MCP_MODULES=librarian); RC=$?
echo "query findings: $QUERY5"
check "5-tool surface: query call succeeds" $RC

FEEDBACK5=$(mcp_call "$MCPSTORE5" record_feedback '{"kind":"feedback","summary":"dogfood MCP probe"}' DESK_ROOT="$MCPDESK" DESK_NAME=mcp-probe-5 MCP_MODULES=librarian); RC=$?
echo "record_feedback: $FEEDBACK5"
check "5-tool surface: record_feedback call succeeds" $RC

echo
echo "--- MCP session: 6-tool LIBRARIAN_AUTONOMOUS_WRITES=true surface ---"
LIST6=$(mcp_list "$MCPSTORE6" DESK_ROOT="$MCPDESK" DESK_NAME=mcp-probe-6 MCP_MODULES=librarian LIBRARIAN_AUTONOMOUS_WRITES=true); RC=$?
echo "tools/list: $LIST6"
COUNT6=$(printf '%s' "$LIST6" | jq -r '.result.tools | length')
if [ "$RC" -eq 0 ] && [ "$COUNT6" -eq 6 ]; then rc=0; else rc=1; fi
check "6-tool surface: tools/list returns exactly 6 tools, incl. apply_fix (got $COUNT6)" "$rc"

mcp_call "$MCPSTORE6" sweep '{}' DESK_ROOT="$MCPDESK" DESK_NAME=mcp-probe-6 MCP_MODULES=librarian LIBRARIAN_AUTONOMOUS_WRITES=true > /dev/null
mcp_call "$MCPSTORE6" patrol '{}' DESK_ROOT="$MCPDESK" DESK_NAME=mcp-probe-6 MCP_MODULES=librarian LIBRARIAN_AUTONOMOUS_WRITES=true > /dev/null

PROPOSE6=$(mcp_call "$MCPSTORE6" propose_fix '{}' DESK_ROOT="$MCPDESK" DESK_NAME=mcp-probe-6 MCP_MODULES=librarian LIBRARIAN_AUTONOMOUS_WRITES=true); RC=$?
echo "propose_fix: $PROPOSE6"
check "6-tool surface: propose_fix call succeeds" $RC

APPLY6=$(mcp_call "$MCPSTORE6" apply_fix '{}' DESK_ROOT="$MCPDESK" DESK_NAME=mcp-probe-6 MCP_MODULES=librarian LIBRARIAN_AUTONOMOUS_WRITES=true); RC=$?
echo "apply_fix: $APPLY6"
APPLIED_OUTCOME=$(printf '%s' "$APPLY6" | jq -r '.result.content[0].text' | jq -r '.outcomes[0].outcome // empty')
[ "$RC" -eq 0 ] && [ "$APPLIED_OUTCOME" = "applied" ]
check "6-tool surface: apply_fix actually writes (outcome=applied)" $?

echo
echo "===================================================================="
echo "$PASS passed, $FAIL failed (of $N checks)"
echo "===================================================================="

[ "$FAIL" -eq 0 ]
exit $?

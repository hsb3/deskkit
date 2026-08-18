#!/bin/bash
# examples/pm-walkthrough.sh — manual, offline walkthrough of the PM module (`deskkit pm`, feature-gated behind
# PM_ENABLED). NOT wired into `make check`/`make verify`/CI — a human runs this by hand to
# prove the document-gated work graph end to end without needing an LLM call. Mirrors the
# style of verify.sh (numbered PASS/FAIL lines, throwaway mktemp desk + store, never a real
# desk). What it exercises:
#
#   1. builds deskkit fresh into a throwaway temp dir (or reuses $DESKKIT_BIN if set, to skip
#      the build on repeat runs)
#   2. seeds two items via the `pm` CLI: item A (owner court) and item B (desk court), then
#      links A --blocks--> B (cascade auto, unblock_at work) — B comes up blocked immediately
#   3. drives the PM MCP surface over raw newline-delimited JSON-RPC stdio
#      (`deskkit mcp-serve`, framing per docs/development/specs/tool-surface.md "How the counts were derived"):
#        transition_item (refused: B is blocked)
#          -> unblock_item (clears the block)
#          -> transition_item (now succeeds: queue -> work)
#          -> add_note (rationale)
#          -> claim_item -> release_item
#   4. confirms the MCP_MODULES=pm mount exposes EXACTLY the 12 PM tools and none of the 5
#      librarian ride-alongs (sweep, patrol, propose_fix, query, record_feedback)
#
# Usage: bash examples/pm-walkthrough.sh   (from anywhere; it cd's to the repo root)
#        DESKKIT_BIN=/path/to/deskkit bash examples/pm-walkthrough.sh   (skip the build)
#
# Exit code is non-zero if any check fails. Every check is a real assertion against real
# command output — nothing here is a self-report.
#
# Gotcha (found while writing this script): PocketBase's own osutils.IsProbablyGoRun()
# heuristic (vendored via go.mod; wired at cmd/deskkit/main.go:193 as
# `DefaultDev: osutils.IsProbablyGoRun()`) flags a binary as "probably `go run`" whenever
# os.Args[0] is prefixed by os.TempDir() or the Go cache dir. `mktemp -d`'s default root IS
# os.TempDir() on macOS (/var/folders/.../T/, distinct from the stable /tmp), so a binary
# built straight into a mktemp'd scratch dir (an obvious, common pattern for a throwaway
# test build) silently defaults `--dev` to true, which makes PocketBase print verbose
# per-query SQL debug lines to STDOUT — corrupting the JSON any script/tool is trying to
# parse from that same stdout. Every deskkit invocation below passes `--dev=false`
# explicitly to neutralize this regardless of where $BIN happens to live.

set -uo pipefail
cd "$(dirname "$0")/.." || exit 1

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

WORK=$(mktemp -d)
trap 'rm -rf "$WORK"' EXIT

DESK="$WORK/desk"
STORE="$WORK/store"
mkdir -p "$DESK" "$STORE"

if [ -n "${DESKKIT_BIN:-}" ]; then
  BIN="$DESKKIT_BIN"
  if [ -x "$BIN" ]; then rc=0; else rc=1; fi
  check "reuse existing binary (DESKKIT_BIN=$BIN)" "$rc"
else
  BIN="$WORK/deskkit"
  go build -o "$BIN" ./cmd/deskkit
  check "build $BIN from ./cmd/deskkit" $?
fi

export PM_ENABLED=true
export DESK_ROOT="$DESK"
export DESK_NAME=dogfood-pm

# NOTE: --actor is a flag local to each `pm <subcommand>` leaf (not a root/pm-group
# persistent flag despite appearing under `pm --help`'s own Flags list) — it must be placed
# after the leaf subcommand, never before it. --dev=false guards against the go-run false
# positive documented above.
run_cli() { "$BIN" --dir "$STORE" --dev=false "$@" --actor dogfood-script; }

# --- seed: item A, item B, A --blocks--> B (cascade auto, unblock_at work) ---

A_JSON=$(run_cli pm create --title "Ship the widget" --type task --court owner --priority 1 --pointer "tasks/widget.md" 2>&1)
check "create item A" $?
A_ID=$(printf '%s' "$A_JSON" | jq -r '.item.id')

B_JSON=$(run_cli pm create --title "Wire widget into pipeline" --type task --court desk --priority 1 --pointer "tasks/pipeline.md" 2>&1)
check "create item B" $?
B_ID=$(printf '%s' "$B_JSON" | jq -r '.item.id')

run_cli pm link "$A_ID" "$B_ID" --kind blocks --unblock-at work --cascade auto > /dev/null 2>&1
check "link: A blocks B (cascade=auto, unblock_at=work)" $?

B_GET=$(run_cli pm get "$B_ID" 2>&1)
B_BLOCKED=$(printf '%s' "$B_GET" | jq -r '.blocked // false')
if [ "$B_BLOCKED" = "true" ]; then rc=0; else rc=1; fi
check "item B comes up blocked=true after the link" "$rc"
B_VERSION=$(printf '%s' "$B_GET" | jq -r '.version')

# --- MCP stdio JSON-RPC helper (one-shot tools/call; framing per docs/development/specs/tool-surface.md) ---
#
# A trailing `sleep 1` inside the printf group keeps stdin open long enough for the response
# to flush before the pipe's EOF races the server shutdown (same reason docs/development/specs/tool-surface.md's
# own MCP_MODULES probe uses it).

INIT_REQ='{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"dogfood-pm","version":"0"}}}'
INITD_REQ='{"jsonrpc":"2.0","method":"notifications/initialized"}'

mcp_call() { # mcp_call <tool-name> <json-args> [MCP_MODULES value; OMIT the arg entirely for unset]
  local tool="$1" args="$2"
  local call_req
  call_req=$(jq -n --arg name "$tool" --argjson args "$args" \
    '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":$name,"arguments":$args}}')
  # NOTE: MCP_MODULES="" (set-but-empty) is NOT the same as unset — per docs/development/specs/tool-surface.md
  # §2.1, an empty/unresolvable MCP_MODULES fails the process closed (exit 1), it does not
  # fall back to "no filter". So when no module filter is wanted, MCP_MODULES must be left
  # entirely unset, never set to "" — hence the branch instead of always exporting it.
  if [ $# -ge 3 ] && [ -n "$3" ]; then
    {
      printf '%s\n%s\n%s\n' "$INIT_REQ" "$INITD_REQ" "$call_req"
      sleep 1
    } | MCP_MODULES="$3" "$BIN" --dir "$STORE" --dev=false mcp-serve 2>"$WORK/mcp-stderr.log" \
      | jq -c 'select(.id == 2)'
  else
    {
      printf '%s\n%s\n%s\n' "$INIT_REQ" "$INITD_REQ" "$call_req"
      sleep 1
    } | "$BIN" --dir "$STORE" --dev=false mcp-serve 2>"$WORK/mcp-stderr.log" \
      | jq -c 'select(.id == 2)'
  fi
}

# result text is itself a JSON string on success ({"item":{...}}); on refusal it's a plain
# message string and .result.isError is true.
mcp_is_error() { printf '%s' "$1" | jq -r '.result.isError // false'; }
mcp_text() { printf '%s' "$1" | jq -r '.result.content[0].text'; }

# 1) transition while blocked -> must be refused
RESP1=$(mcp_call transition_item "$(jq -n --arg id "$B_ID" --argjson v "$B_VERSION" '{item_id:$id,target_phase:"work",version:$v}')")
ERR1=$(mcp_is_error "$RESP1")
if [ "$ERR1" = "true" ]; then rc=0; else rc=1; fi
check "transition_item refused while B is blocked" "$rc"
echo "      refusal message: $(mcp_text "$RESP1")"

# 2) unblock
RESP2=$(mcp_call unblock_item "$(jq -n --arg id "$B_ID" --argjson v "$B_VERSION" '{item_id:$id,version:$v,reason:"dependency check complete"}')")
ERR2=$(mcp_is_error "$RESP2")
if [ "$ERR2" != "true" ]; then rc=0; else rc=1; fi
check "unblock_item succeeds" "$rc"
B_VERSION=$(mcp_text "$RESP2" | jq -r '.item.version')

# 3) transition again -> must now succeed
RESP3=$(mcp_call transition_item "$(jq -n --arg id "$B_ID" --argjson v "$B_VERSION" '{item_id:$id,target_phase:"work",version:$v}')")
ERR3=$(mcp_is_error "$RESP3")
NEWPHASE=$(mcp_text "$RESP3" | jq -r '.item.phase // empty')
if [ "$ERR3" != "true" ] && [ "$NEWPHASE" = "work" ]; then rc=0; else rc=1; fi
check "transition_item succeeds once unblocked (queue -> work)" "$rc"
B_VERSION=$(mcp_text "$RESP3" | jq -r '.item.version')

# 4) add a note
RESP4=$(mcp_call add_note "$(jq -n --arg id "$B_ID" '{item_id:$id,key:"rationale",body:"Unblocked after confirming the widget dependency was satisfied; advancing to work."}')")
ERR4=$(mcp_is_error "$RESP4")
if [ "$ERR4" != "true" ]; then rc=0; else rc=1; fi
check "add_note succeeds" "$rc"

# 5) claim
RESP5=$(mcp_call claim_item "$(jq -n --argjson v "$B_VERSION" --arg id "$B_ID" '{item_id:$id,version:$v}')")
ERR5=$(mcp_is_error "$RESP5")
CLAIMED_BY=$(mcp_text "$RESP5" | jq -r '.item.claimed_by // empty')
if [ "$ERR5" != "true" ] && [ -n "$CLAIMED_BY" ]; then rc=0; else rc=1; fi
check "claim_item succeeds (claimed_by=$CLAIMED_BY)" "$rc"
B_VERSION=$(mcp_text "$RESP5" | jq -r '.item.version')

# 6) release
RESP6=$(mcp_call release_item "$(jq -n --argjson v "$B_VERSION" --arg id "$B_ID" '{item_id:$id,version:$v}')")
ERR6=$(mcp_is_error "$RESP6")
if [ "$ERR6" != "true" ]; then rc=0; else rc=1; fi
check "release_item succeeds" "$rc"

# --- module-gating check: MCP_MODULES=pm exposes exactly the 12 PM tools ---

LIST_REQ='{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}'
LIST_OUT=$( { printf '%s\n%s\n%s\n' "$INIT_REQ" "$INITD_REQ" "$LIST_REQ"; sleep 1; } \
  | MCP_MODULES=pm "$BIN" --dir "$STORE" --dev=false mcp-serve 2>"$WORK/mcp-list-stderr.log" \
  | jq -c 'select(.id == 2)' )

TOOL_NAMES=$(printf '%s' "$LIST_OUT" | jq -r '.result.tools[].name' | sort)
TOOL_COUNT=$(printf '%s' "$TOOL_NAMES" | grep -c .)

EXPECTED=$(printf '%s\n' \
  add_note block_item claim_item create_item get_context get_item link_items \
  list_items release_item transition_item unblock_item update_item | sort)

[ "$TOOL_COUNT" -eq 12 ]
check "MCP_MODULES=pm exposes exactly 12 tools (got $TOOL_COUNT)" $?

[ "$TOOL_NAMES" = "$EXPECTED" ]
check "MCP_MODULES=pm tool set matches the 12 PM tools exactly (no ride-alongs)" $?

echo
echo "Exposed tool set:"
printf '%s\n' "$TOOL_NAMES" | sed 's/^/  - /'

echo
echo "===================================================================="
echo "$PASS passed, $FAIL failed (of $N checks)"
echo "===================================================================="

[ "$FAIL" -eq 0 ]
exit $?

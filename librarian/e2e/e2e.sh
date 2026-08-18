#!/usr/bin/env bash
# e2e.sh — desk-standard end-to-end SYSTEM-BEHAVIOUR suite (issue-scope: 1.0.0 cohesion lane).
#
# Walks the whole product as one system against a THROWAWAY scratch desk (never a real store),
# in the order the cohesion lane prescribes:
#
#   cold-start (init) -> profile (validate/get) -> librarian sweep/patrol/propose-fix/apply-fix/
#   restore -> PM graph create/transition/gate/block/cascade -> shipped surfaces (the Go MCP
#   server's default + gated mounts, skills/agents, SessionStart hook) -> release-shaped checks
#   (version sync, changelog, version stamp, marketplace bundle shape).
#
# Every step is a real assertion against real command output — nothing here is a self-report.
# Steps that genuinely need an LLM key (the eino agent loop, the chat TUI) are SKIPPED with a
# visible notice, never silently passed. Numbered PASS/FAIL/SKIP lines; non-zero exit if any
# check FAILs (SKIPs do not fail the run).
#
# Usage:  bash librarian/e2e/e2e.sh          (from anywhere; it locates the repo from its path)
#         DESKKIT_BIN=/path/to/deskkit bash librarian/e2e/e2e.sh   (reuse a prebuilt binary)
#
# It builds deskkit fresh (unless DESKKIT_BIN is set), drives that binary's MCP server over stdio,
# and shells the SessionStart hook — all offline. No network, no real desk, no secrets.

set -uo pipefail

E2E_DIR="$(cd "$(dirname "$0")" && pwd)"
E2E_REPO="$(cd "$E2E_DIR/../.." && pwd)"

# shellcheck source=/dev/null  # lib.sh is resolved at runtime relative to this script
. "$E2E_DIR/lib.sh"

# --- throwaway scratch world --------------------------------------------------------------
E2E_WORK="$(mktemp -d)"
# shellcheck disable=SC2329,SC2317  # invoked indirectly by the EXIT trap
cleanup() { chmod -R u+w "$E2E_WORK" 2>/dev/null || true; rm -rf "$E2E_WORK"; }
trap cleanup EXIT

# The desk directory basename becomes the scaffolded profile's desk.name (deskkit init derives
# it from the folder), so name it to match DESK_NAME below — env and profile then agree.
E2E_DESK="$E2E_WORK/e2e-desk"
E2E_STORE="$E2E_WORK/store"
E2E_BINDIR="$E2E_WORK/bin"
mkdir -p "$E2E_DESK" "$E2E_STORE" "$E2E_BINDIR"

# Config resolution for every deskkit call: DESK_ROOT + DESK_NAME (CLAUDE.md resolution rule 2).
export DESK_ROOT="$E2E_DESK"
export DESK_NAME="e2e-desk"

export E2E_DIR E2E_REPO E2E_WORK E2E_DESK E2E_STORE E2E_BINDIR

echo "desk-standard E2E system-behaviour suite"
echo "repo:        $E2E_REPO"
echo "scratch desk: $E2E_DESK"
echo "scratch store: $E2E_STORE"

# --- build the binary as literally `deskkit`, on PATH (the hook needs `command -v deskkit`) --
section "build"
if [ -n "${DESKKIT_BIN:-}" ] && [ -x "${DESKKIT_BIN:-}" ]; then
  cp "$DESKKIT_BIN" "$E2E_BINDIR/deskkit"
  E2E_BIN="$E2E_BINDIR/deskkit"
  check "reuse prebuilt binary (DESKKIT_BIN=$DESKKIT_BIN)" 0
else
  E2E_BIN="$E2E_BINDIR/deskkit"
  ( cd "$E2E_REPO/librarian" && go build -o "$E2E_BIN" ./cmd/deskkit )
  check "build deskkit from ./librarian/cmd/deskkit" $?
fi
export E2E_BIN
export PATH="$E2E_BINDIR:$PATH"

# --- source and run each chain link in numeric order --------------------------------------
for step in "$E2E_DIR"/steps/[0-9]*.sh; do
  [ -e "$step" ] || continue
  # shellcheck source=/dev/null
  . "$step"
done

# --- summary ------------------------------------------------------------------------------
section "summary"
printf 'checks: %d   PASS: %d   FAIL: %d   SKIP: %d\n' "$N" "$PASS" "$FAIL" "$SKIP"
if [ "$FAIL" -ne 0 ]; then
  echo "E2E: FAIL"
  exit 1
fi
echo "E2E: OK"
exit 0

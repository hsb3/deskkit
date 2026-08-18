#!/usr/bin/env bash
# lib.sh — shared harness for the deskkit E2E system-behaviour suite (e2e.sh).
#
# Sourced by e2e.sh BEFORE any steps/*.sh. Defines the whole contract a step file may rely
# on: assertion helpers (check / skip / note / section), a deskkit runner (dk), the MCP
# JSON-RPC driver (mcp_go), a librarian fixture seeder, and the exported state
# (E2E_* vars + DESK_ROOT/DESK_NAME) that pins every step to ONE throwaway scratch desk.
#
# Design constraints baked in here so step authors don't have to rediscover them:
#   * --dev=false on every deskkit call. A binary whose path is under os.TempDir() (mktemp's
#     macOS root /var/folders/.../T) trips PocketBase's IsProbablyGoRun heuristic and defaults
#     --dev true, which dumps SQL debug lines to STDOUT and corrupts any JSON a step parses.
#   * Go MCP framing is initialize -> notifications/initialized -> request, then a `sleep 1`
#     holds stdin open long enough for the server to answer before the pipe closes.
#   * The binary is built as literally `deskkit` on PATH so the SessionStart hook's
#     `command -v deskkit` resolves it.
#   * Identity-neutral throughout (scanned by scripts/check-neutrality.mjs): no person/org/repo
#     names, no bare issue refs. Cite specs/ADRs by relative path only.

# Assertion counters (read by e2e.sh for the summary + exit code).
PASS=0
FAIL=0
SKIP=0
N=0

# check <description> <rc>  — rc 0 = PASS, nonzero = FAIL.
check() {
  N=$((N + 1))
  if [ "$2" -eq 0 ]; then
    printf '[%02d] PASS  %s\n' "$N" "$1"
    PASS=$((PASS + 1))
  else
    printf '[%02d] FAIL  %s\n' "$N" "$1"
    FAIL=$((FAIL + 1))
  fi
}

# skip <description> <reason>  — a step that genuinely cannot run here (e.g. needs an LLM key).
# NEVER a silent pass: it prints a visible SKIP line and is tallied separately.
skip() {
  N=$((N + 1))
  SKIP=$((SKIP + 1))
  printf '[%02d] SKIP  %s  (%s)\n' "$N" "$1" "$2"
}

# note <text> — non-assertion commentary (observed value, seam note). No counter effect.
note() { printf '        · %s\n' "$1"; }

# section <title> — a visible band between chain links.
section() { printf '\n=== %s ===\n' "$1"; }

# dk <args...> — run the built deskkit against the scratch store, dev-mode neutralised.
dk() { "$E2E_BIN" --dir "$E2E_STORE" --dev=false "$@"; }

# JSON-RPC request literals shared by both MCP drivers.
E2E_MCP_INIT='{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"e2e","version":"0"}}}'
E2E_MCP_INITD='{"jsonrpc":"2.0","method":"notifications/initialized"}'

# mcp_go <modules|-> <request-json-id-2>  — drive the Go MCP stdio server (the one server: the
# profile, librarian and PM modules) for a single id:2 request; prints the bare `.result` object
# (compact JSON) on stdout. Pass "-" for the default mount (no MCP_MODULES); pass e.g. "pm" or
# "profile" to gate the mount. DESK_ROOT (exported by e2e.sh) points every mount at the scratch
# desk, so the profile module resolves that desk's scaffolded profile.
mcp_go() {
  local modules="$1" req="$2"
  if [ "$modules" = "-" ]; then
    { printf '%s\n%s\n%s\n' "$E2E_MCP_INIT" "$E2E_MCP_INITD" "$req"; sleep 1; } \
      | "$E2E_BIN" --dir "$E2E_STORE" --dev=false mcp-serve 2>/dev/null \
      | jq -c 'select(.id==2) | .result'
  else
    { printf '%s\n%s\n%s\n' "$E2E_MCP_INIT" "$E2E_MCP_INITD" "$req"; sleep 1; } \
      | MCP_MODULES="$modules" "$E2E_BIN" --dir "$E2E_STORE" --dev=false mcp-serve 2>/dev/null \
      | jq -c 'select(.id==2) | .result'
  fi
}

# seed_librarian_fixtures — plant a minimal, deterministic set of rule-violating docs on the
# scratch desk so the sweep/patrol/fix chain has something to find. Mirrors the shapes
# verify.sh seeds from the spec fixture table, trimmed to the two mechanical rules this walk
# asserts (R1 missing frontmatter; R3 type/location mismatch).
seed_librarian_fixtures() {
  mkdir -p "$E2E_DESK/tasks" "$E2E_DESK/analyses"
  # R1: correctly placed under tasks/, but missing the universal frontmatter fields.
  printf -- '---\ntype: task\n---\none body line\n' > "$E2E_DESK/tasks/r1-missing-fm.md"
  # R3: full frontmatter, type: task, but filed under analyses/ (expected tasks/).
  cat > "$E2E_DESK/analyses/r3-type-task.md" <<'FIX'
---
type: task
created: 2026-07-15
updated: 2026-07-15
tags: []
synopsis: "type/location mismatch fixture"
---
short body line
FIX
}

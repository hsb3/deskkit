#!/usr/bin/env bash
# shellcheck shell=bash
# 10 — cold-start + profile.  Sourced by e2e.sh; helpers (check/skip/note/section/dk/mcp_go)
# and E2E_* state come from lib.sh. A brand-new empty folder becomes a working desk via
# `deskkit init`, config then resolves it by walk-up, and the profile module validates the
# scaffolded profile through the Go MCP surface — all with zero personalization.

section "10 · cold-start + profile"

# --- cold start: init scaffolds a minimal profile on an empty folder ----------------------
dk init "$E2E_DESK" >/dev/null 2>&1
check "cold-start: deskkit init scaffolds a fresh desk" $?

if [ -f "$E2E_DESK/_knowledge/profile.yaml" ]; then RC=0; else RC=1; fi
check "cold-start: _knowledge/profile.yaml written" "$RC"

# The scaffold is the least a folder needs: schema_version + desk.name + root.
grep -q '^schema_version: 1' "$E2E_DESK/_knowledge/profile.yaml"
check "cold-start: scaffolded profile declares schema_version 1" $?

# --- self-initialising store: a tool command on a never-migrated store must succeed --------
# (ADR 0003 — the requireConfig choke point auto-applies migrations; no manual `migrate up`.)
dk query summary >/dev/null 2>&1
check "self-init: query on a never-migrated store succeeds (ADR 0003)" $?

# --- profile resolves and validates against schema v1 (via the Go MCP profile module) ------
VAL=$(mcp_go - '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"profile_validate","arguments":{"path":"'"$E2E_DESK"'/_knowledge/profile.yaml"}}}')
# The tool result nests the validator payload as JSON text in content[0].text.
VALID=$(printf '%s' "$VAL" | jq -r '.content[0].text | fromjson | .valid' 2>/dev/null)
[ "$VALID" = "true" ]
check "profile: scaffolded profile validates against schema v1 (MCP profile_validate)" $?
note "profile_validate returned valid=$VALID"

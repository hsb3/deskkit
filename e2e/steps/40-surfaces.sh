#!/usr/bin/env bash
# shellcheck shell=bash
# 40 — shipped surfaces: the Go MCP server (default mount + the profile/pm gated mounts), skills,
# agents, SessionStart hook. Sourced by e2e.sh; helpers (check/skip/note/section/mcp_go) and
# E2E_* state come from lib.sh. LLM-gated surfaces (agent loop, chat TUI) are explicitly
# skipped, never silently passed.
#
# The idiom `<condition>; check "<desc>" $?` recurs throughout this step: a test condition is
# evaluated purely for its exit status, which is then fed to check() as pass/fail. SC2319 ("$? after
# a condition usually means you wanted the command before it") structurally misfires on that
# intentional pattern at every call site (same rationale as verify.sh), so it is
# disabled file-wide below; every OTHER lint rule stays enforced.
# shellcheck disable=SC2319

section "40 · shipped surfaces — MCP, skills, agents, hooks"

# --- Go MCP: default mount (profile always-on + PM default-on -> 21 tools) ------------------
GO=$(mcp_go - '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}')
CNT=$(printf '%s' "$GO" | jq -r '.tools | length' 2>/dev/null)
GO_NAMES=$(printf '%s' "$GO" | jq -r '.tools[].name' 2>/dev/null)
[ "$CNT" = "21" ] \
  && printf '%s\n' "$GO_NAMES" | grep -qx 'sweep' \
  && printf '%s\n' "$GO_NAMES" | grep -qx 'patrol' \
  && printf '%s\n' "$GO_NAMES" | grep -qx 'query' \
  && printf '%s\n' "$GO_NAMES" | grep -qx 'create_item' \
  && printf '%s\n' "$GO_NAMES" | grep -qx 'transition_item' \
  && printf '%s\n' "$GO_NAMES" | grep -qx 'profile_get' \
  && printf '%s\n' "$GO_NAMES" | grep -qx 'knowledge_index'
check "MCP default mount exposes 21 tools (4 profile + 5 librarian + 12 PM; PM default-on)" $?
note "MCP default mount tool count: $CNT"

# --- Go MCP: PM opted out (PM_ENABLED=false -> the 4 profile + 5 librarian tools) -----------
PMOFF=$(PM_ENABLED=false mcp_go - '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}')
PMOFF_CNT=$(printf '%s' "$PMOFF" | jq -r '.tools | length' 2>/dev/null)
PMOFF_NAMES=$(printf '%s' "$PMOFF" | jq -r '.tools[].name' 2>/dev/null)
[ "$PMOFF_CNT" = "9" ] \
  && printf '%s\n' "$PMOFF_NAMES" | grep -qx 'profile_get' \
  && printf '%s\n' "$PMOFF_NAMES" | grep -qx 'sweep' \
  && ! printf '%s\n' "$PMOFF_NAMES" | grep -qx 'create_item'
check "PM_ENABLED=false mount exposes 9 tools (4 profile + 5 librarian; no PM tools)" $?
note "PM-off mount tool count: $PMOFF_CNT"

# --- Go MCP: gated profile-only mount (MCP_MODULES=profile -> exactly the 4 profile tools) --
# The profile module is ungated, so it rides every UNFILTERED mount — but a mount that declares
# a module set must still be narrowed to it. Assert the names, not just the count.
PROFOUT=$(mcp_go profile '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}')
PROF_CNT=$(printf '%s' "$PROFOUT" | jq -r '.tools | length' 2>/dev/null)
PROF_NAMES=$(printf '%s' "$PROFOUT" | jq -r '.tools[].name' 2>/dev/null | sort | paste -sd, -)
[ "$PROF_CNT" = "4" ] \
  && [ "$PROF_NAMES" = "knowledge_index,profile_get,profile_validate,template_render" ]
check "MCP_MODULES=profile mount exposes exactly the 4 profile tools (profile_get/validate, template_render, knowledge_index)" $?
note "profile-only mount tools/list: $PROF_NAMES"

# profile_get resolves a dotted key from the scratch desk's OWN scaffolded profile — a real
# tools/call against the Go server, not just a listing. The profile module discovers the desk
# from the resolved desk root (DESK_ROOT, exported by e2e.sh). `deskkit init` derives desk.name
# from the desk directory's basename, so that is the expected value.
PG_DESKNAME=$(basename "$E2E_DESK")
PG=$(mcp_go profile '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"profile_get","arguments":{"path":"desk.name"}}}')
PG_VALUE=$(printf '%s' "$PG" | jq -r '.content[0].text | fromjson | .value' 2>/dev/null)
[ "$PG_VALUE" = "$PG_DESKNAME" ]
check "MCP profile_get resolves a profile key (desk.name from the scratch desk's scaffolded profile)" $?
note "profile_get desk.name: $PG_VALUE (expected basename of scratch desk: $PG_DESKNAME)"

# --- Go librarian MCP: gated PM-only mount (MCP_MODULES=pm -> 12 tools, no ride-alongs) -----
PMOUT=$(mcp_go pm '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}')
PMCNT=$(printf '%s' "$PMOUT" | jq -r '.tools | length' 2>/dev/null)
PM_NAMES=$(printf '%s' "$PMOUT" | jq -r '.tools[].name' 2>/dev/null)
[ "$PMCNT" = "12" ] \
  && ! printf '%s\n' "$PM_NAMES" | grep -qx 'sweep' \
  && ! printf '%s\n' "$PM_NAMES" | grep -qx 'patrol' \
  && ! printf '%s\n' "$PM_NAMES" | grep -qx 'query' \
  && ! printf '%s\n' "$PM_NAMES" | grep -qx 'propose_fix' \
  && ! printf '%s\n' "$PM_NAMES" | grep -qx 'record_feedback'
check "MCP_MODULES=pm mount exposes exactly the 12 PM tools (no librarian ride-alongs)" $?
note "pm-only mount tool count: $PMCNT"

# --- skills present: all 7 in the single shipped bundle --------------------------------------
SKILLS_MISSING=0
for skill in \
  "plugins/deskkit/skills/desk-setup/SKILL.md" \
  "plugins/deskkit/skills/conventions-standard/SKILL.md" \
  "plugins/deskkit/skills/harvest-loop/SKILL.md" \
  "plugins/deskkit/skills/brownfield-adoption/SKILL.md" \
  "plugins/deskkit/skills/pm-session-open/SKILL.md" \
  "plugins/deskkit/skills/pm-triage/SKILL.md" \
  "plugins/deskkit/skills/pm-advance-item/SKILL.md"; do
  [ -f "$E2E_REPO/$skill" ] || SKILLS_MISSING=$((SKILLS_MISSING + 1))
done
[ "$SKILLS_MISSING" -eq 0 ]
check "all 7 shipped skills present (4 desk + 3 PM, one bundle)" $?

# --- agents present --------------------------------------------------------------------------
[ -f "$E2E_REPO/plugins/deskkit/agents/librarian-operator.md" ] \
  && [ -f "$E2E_REPO/plugins/deskkit/agents/pm-operator.md" ]
check "librarian-operator and pm-operator agents present" $?

# --- SessionStart hook: LLM-free cold-start briefing on a PM-default-on desk ----------------
BR=$(CLAUDE_PLUGIN_ROOT="$E2E_REPO/plugins/deskkit" bash "$E2E_REPO/plugins/deskkit/hooks/session-briefing.sh" 2>/dev/null)
printf '%s' "$BR" | grep -qF 'cold-start briefing' && printf '%s' "$BR" | grep -qF '{'
check "SessionStart hook emits a PM cold-start briefing on a default-on desk" $?

# --- LLM-gated surfaces: explicit, visible skips (never silent) -----------------------------
skip "agent loop (deskkit agent)" "needs an LLM API key (no key in this environment)"
skip "chat TUI (deskkit chat)" "needs an LLM API key / interactive terminal"

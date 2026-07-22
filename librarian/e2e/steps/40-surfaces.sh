#!/usr/bin/env bash
# shellcheck shell=bash
# 40 — plugin surfaces: TS plugin MCP, Go librarian/PM MCP (default + gated mounts), skills,
# agents, SessionStart hook. Sourced by e2e.sh; helpers (check/skip/note/section/have_bun/
# mcp_ts/mcp_go) and E2E_* state come from lib.sh. LLM-gated surfaces (agent loop, chat TUI)
# are explicitly skipped, never silently passed.
#
# The idiom `<condition>; check "<desc>" $?` recurs throughout this step: a test condition is
# evaluated purely for its exit status, which is then fed to check() as pass/fail. SC2319 ("$? after
# a condition usually means you wanted the command before it") structurally misfires on that
# intentional pattern at every call site (same rationale as librarian/verify.sh), so it is
# disabled file-wide below; every OTHER lint rule stays enforced.
# shellcheck disable=SC2319

section "40 · plugin surfaces — MCP, skills, agents, hooks"

# --- TS plugin MCP: tools/list + one tools/call --------------------------------------------
if have_bun; then
  TS=$(mcp_ts '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}')
  NAMES=$(printf '%s' "$TS" | jq -r '.tools[].name' | sort | paste -sd, -)
  [ "$NAMES" = "knowledge_index,profile_get,profile_validate,template_render" ]
  check "plugin MCP exposes the 4 core tools (profile_get/validate, template_render, knowledge_index)" $?
  note "plugin MCP tools/list: $NAMES"

  # profile_get resolves a dotted key from the scratch desk's OWN scaffolded profile. mcp_ts
  # (lib.sh) runs the server from the desk cwd, so the plugin core's walk-up discovery finds
  # _knowledge/profile.yaml — exactly how the plugin behaves inside a desk. `deskkit init`
  # derives desk.name from the desk directory's basename, so that is the expected value.
  PG_DESKNAME=$(basename "$E2E_DESK")
  PG=$(mcp_ts '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"profile_get","arguments":{"path":"desk.name"}}}')
  PG_VALUE=$(printf '%s' "$PG" | jq -r '.content[0].text | fromjson | .value' 2>/dev/null)
  [ "$PG_VALUE" = "$PG_DESKNAME" ]
  check "plugin MCP profile_get resolves a profile key (desk.name from the scratch desk's scaffolded profile)" $?
  note "profile_get desk.name: $PG_VALUE (expected basename of scratch desk: $PG_DESKNAME)"
else
  skip "plugin MCP surface" "bun not on PATH"
fi

# --- Go librarian MCP: default mount (PM default-on -> 17 tools) ---------------------------
GO=$(mcp_go - '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}')
CNT=$(printf '%s' "$GO" | jq -r '.tools | length' 2>/dev/null)
GO_NAMES=$(printf '%s' "$GO" | jq -r '.tools[].name' 2>/dev/null)
[ "$CNT" = "17" ] \
  && printf '%s\n' "$GO_NAMES" | grep -qx 'sweep' \
  && printf '%s\n' "$GO_NAMES" | grep -qx 'patrol' \
  && printf '%s\n' "$GO_NAMES" | grep -qx 'query' \
  && printf '%s\n' "$GO_NAMES" | grep -qx 'create_item' \
  && printf '%s\n' "$GO_NAMES" | grep -qx 'transition_item'
check "librarian MCP default mount exposes 17 tools (5 librarian + 12 PM; PM default-on)" $?
note "librarian MCP default mount tool count: $CNT"

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

# --- skills present: 4 desk-standard + 3 desk-persona PM skills -----------------------------
SKILLS_MISSING=0
for skill in \
  "plugins/desk-standard/skills/desk-setup/SKILL.md" \
  "plugins/desk-standard/skills/conventions-standard/SKILL.md" \
  "plugins/desk-standard/skills/harvest-loop/SKILL.md" \
  "plugins/desk-standard/skills/brownfield-adoption/SKILL.md" \
  "plugins/desk-persona/skills/pm-session-open/SKILL.md" \
  "plugins/desk-persona/skills/pm-triage/SKILL.md" \
  "plugins/desk-persona/skills/pm-advance-item/SKILL.md"; do
  [ -f "$E2E_REPO/$skill" ] || SKILLS_MISSING=$((SKILLS_MISSING + 1))
done
[ "$SKILLS_MISSING" -eq 0 ]
check "all 7 shipped skills present (4 desk-standard + 3 desk-persona PM)" $?

# --- agents present --------------------------------------------------------------------------
[ -f "$E2E_REPO/plugins/desk-persona/agents/librarian-operator.md" ] \
  && [ -f "$E2E_REPO/plugins/desk-persona/agents/pm-operator.md" ]
check "librarian-operator and pm-operator agents present" $?

# --- SessionStart hook: LLM-free cold-start briefing on a PM-default-on desk ----------------
BR=$(CLAUDE_PLUGIN_ROOT="$E2E_REPO/plugins/desk-persona" bash "$E2E_REPO/plugins/desk-persona/hooks/session-briefing.sh" 2>/dev/null)
printf '%s' "$BR" | grep -qF 'cold-start briefing' && printf '%s' "$BR" | grep -qF '{'
check "SessionStart hook emits a PM cold-start briefing on a default-on desk" $?

# --- LLM-gated surfaces: explicit, visible skips (never silent) -----------------------------
skip "agent loop (deskkit agent)" "needs an LLM API key (no key in this environment)"
skip "chat TUI (deskkit chat)" "needs an LLM API key / interactive terminal"

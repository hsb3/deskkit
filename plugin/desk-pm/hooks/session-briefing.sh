#!/usr/bin/env bash
# SessionStart hook — inject the PM cold-start briefing into a new session's context.
#
# Optional and self-gating: it degrades to a SILENT no-op (exit 0, no output) on any of —
#   * the deskkit binary is not installed / not on PATH;
#   * the PM module is not enabled on this desk (`pm` subcommand unregistered -> error);
#   * config cannot resolve (DESK_ROOT / DESK_NAME), or the store is unreachable;
#   * the briefing comes back empty.
# So it is safe to ship enabled: on a non-PM desk it does nothing.
#
# `deskkit pm context` prints the get_context briefing (active / blocked / stalled / recent
# transitions) as JSON. It opens the store directly, so it must not run while `deskkit serve`
# holds the DB — in that case it simply errors here and the hook no-ops.
set -euo pipefail

command -v deskkit >/dev/null 2>&1 || exit 0

# Bound the call so a stuck store (e.g. the DB held by a running `deskkit serve`) can never
# hang session start. `timeout` is not on stock macOS, so use it only when present; without
# it the `|| exit 0` still makes a nonzero exit a silent no-op.
if command -v timeout >/dev/null 2>&1; then
  context="$(timeout 5s deskkit pm context 2>/dev/null)" || exit 0
else
  context="$(deskkit pm context 2>/dev/null)" || exit 0
fi
[ -z "$context" ] && exit 0

header="PM work graph — cold-start briefing (from \`deskkit pm context\`). The desk's active, blocked, and stalled work follows as JSON. Use the pm-session-open skill to render it and pick the next item."

if command -v jq >/dev/null 2>&1; then
  # Proper JSON envelope with the briefing escaped into additionalContext.
  jq -n --arg h "$header" --arg c "$context" \
    '{hookSpecificOutput: {hookEventName: "SessionStart", additionalContext: ($h + "\n\n" + $c)}}'
else
  # No jq: SessionStart adds a hook's stdout to the session context directly.
  printf '%s\n\n%s\n' "$header" "$context"
fi

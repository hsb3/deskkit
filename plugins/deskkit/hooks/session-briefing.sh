#!/usr/bin/env bash
# SessionStart hook — inject the PM cold-start briefing into a new session's context.
#
# Optional and self-gating: it degrades to a SILENT no-op (exit 0, no output) on any of —
#   * the deskkit binary is not installed / not on PATH;
#   * the PM module is not enabled on this desk (`pm` subcommand unregistered);
#   * config cannot resolve (DESK_ROOT / DESK_NAME), or the store is unreachable;
#   * the briefing comes back empty.
# So it is safe to ship enabled: on a non-PM desk it does nothing.
#
# CRITICAL contract detail: `deskkit pm context` is NOT reliable via its exit code. On a PM-off
# desk the `pm` command is unregistered, but PocketBase's Execute() swallows the RootCmd error
# and EXITS 0 while cobra prints `Error: unknown command "pm" ...` to STDOUT (an upstream
# PocketBase quirk). So `2>/dev/null`, `|| exit 0`, and a non-empty check all pass on that
# error text. The only robust gate is the SHAPE of stdout: a real briefing is a JSON object
# (`{`...); anything else — a cobra error, a usage dump, empty — is not a briefing. We emit
# ONLY when the captured output begins with `{`.
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

# Emit only a real JSON-object briefing; PM-off (cobra error on stdout, exit 0), usage text,
# and empty output all fall through to a silent no-op — the contract holds regardless.
# Strip any leading whitespace first (current output has none; defensive if the format shifts).
context="${context#"${context%%[![:space:]]*}"}"
case "$context" in
  '{'*) ;;
  *) exit 0 ;;
esac

header="PM work graph — cold-start briefing (from \`deskkit pm context\`). The desk's active, blocked, and stalled work follows as JSON. Use the pm-session-open skill to render it and pick the next item."

# A SessionStart hook's stdout is added to the session context directly. Emit the header +
# the (already human-readable) briefing JSON as plain text — no dependency, identical
# behaviour on every machine, and no risk of an unrecognised JSON envelope landing verbatim.
printf '%s\n\n%s\n' "$header" "$context"

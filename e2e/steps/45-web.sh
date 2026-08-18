#!/usr/bin/env bash
# shellcheck shell=bash
# 45 — embedded web: the SPA shell served at `/`, the SPA index fallback at the old `/desk/chat`
# URL, the loopback dev-bootstrap token endpoint (`/desk/bootstrap`, same-origin only), and the
# PocketBase REST API staying fail-closed unless it carries that token. Sourced by e2e.sh;
# helpers (check/skip/note/section) and E2E_* state come from lib.sh. The build step now embeds
# the real SPA (see e2e.sh's build section), so the shell assertion below checks the real markup
# rather than splitting on a built/not-built flavor.
#
# Starts its OWN `serve` process against the scratch store (steps 10-40 never needed one), on a
# $$-derived high port. ponytail: $$-derived, not a reserved/bound port — good enough for one
# suite process at a time; move to `nc -z`-style probing first if this suite ever runs
# concurrently on one host.
#
# The idiom `<condition>; check "<desc>" $?` recurs below: a test condition is evaluated purely
# for its exit status, fed to check() as pass/fail. SC2319 structurally misfires on that
# intentional pattern at every call site (same rationale as steps/40-surfaces.sh), so it is
# disabled file-wide; every other lint rule stays enforced. SC2064 also misfires on the trap
# lines below: E2E_WEB_PRIOR_TRAP is a fixed string captured once above, so expanding it at
# `trap`-registration time (not at signal time) is the intended, correct behaviour.
# shellcheck disable=SC2319,SC2064

section "45 · embedded web — SPA shell, chat fallback, bootstrap token, REST fail-closed"

E2E_WEB_PORT=$((20000 + ($$ % 20000)))
E2E_WEB_BASE="http://127.0.0.1:${E2E_WEB_PORT}"
E2E_WEB_LOG="$E2E_WORK/serve-45.log"
E2E_WEB_BODY="$E2E_WORK/serve-45-body"

"$E2E_BIN" --dir "$E2E_STORE" --dev=false serve --http "127.0.0.1:${E2E_WEB_PORT}" \
  >"$E2E_WEB_LOG" 2>&1 &
E2E_WEB_PID=$!

# kill_web_45 — stop this step's serve process. Idempotent: safe to call more than once.
kill_web_45() { kill "$E2E_WEB_PID" 2>/dev/null; wait "$E2E_WEB_PID" 2>/dev/null; }

# This step is sourced into the SAME shell as every other step, so it shares e2e.sh's own
# `trap cleanup EXIT` (scratch-dir removal). Replacing that trap outright would silently drop
# the outer cleanup, so chain onto whatever is already registered instead.
E2E_WEB_PRIOR_TRAP="$(trap -p EXIT | sed -E "s/^trap -- '(.*)' EXIT\$/\1/")"
trap "kill_web_45; ${E2E_WEB_PRIOR_TRAP}" EXIT

# finish_45 — tear down the serve process and restore the outer trap. Called on every exit path
# out of this step: normal completion, or the early bail below if serve never becomes ready.
finish_45() {
  kill_web_45
  trap "${E2E_WEB_PRIOR_TRAP}" EXIT
}

# --- poll /api/health until serve actually answers (timeout ~15s) --------------------------
E2E_WEB_READY=1
for _ in $(seq 1 30); do
  if curl -fsS -o /dev/null "${E2E_WEB_BASE}/api/health" 2>/dev/null; then
    E2E_WEB_READY=0
    break
  fi
  sleep 0.5
done
check "serve became ready on 127.0.0.1:${E2E_WEB_PORT} within 15s" "$E2E_WEB_READY"
if [ "$E2E_WEB_READY" -ne 0 ]; then
  note "serve never answered /api/health — log tail: $(tail -n 20 "$E2E_WEB_LOG" 2>/dev/null)"
  finish_45
  return 1
fi

# --- GET / -> the embedded SPA shell --------------------------------------------------------
CODE=$(curl -s -o "$E2E_WEB_BODY" -w '%{http_code}' "${E2E_WEB_BASE}/")
[ "$CODE" = "200" ] && grep -qF '<div id="app">' "$E2E_WEB_BODY"
check "GET / serves the embedded SPA shell (200, <div id=\"app\">)" $?

# --- GET /desk/chat -> SPA index fallback (the old standalone chat page is gone; URL still works) --
CODE=$(curl -s -o /dev/null -w '%{http_code}' "${E2E_WEB_BASE}/desk/chat")
[ "$CODE" = "200" ]
check "GET /desk/chat falls back to the SPA index" $?

# --- GET /desk/bootstrap on loopback -> a superuser token -----------------------------------
CODE=$(curl -s -o "$E2E_WEB_BODY" -w '%{http_code}' "${E2E_WEB_BASE}/desk/bootstrap")
BOOT_TOKEN=$(jq -r '.token // empty' <"$E2E_WEB_BODY" 2>/dev/null)
[ "$CODE" = "200" ] && [ -n "$BOOT_TOKEN" ]
check "GET /desk/bootstrap on loopback returns a token" $?

# --- same route, a cross-site Origin -> refused ----------------------------------------------
CODE=$(curl -s -o /dev/null -w '%{http_code}' -H 'Origin: https://example.com' "${E2E_WEB_BASE}/desk/bootstrap")
[ "$CODE" = "403" ]
check "GET /desk/bootstrap with a cross-site Origin is refused (403)" $?

# --- PocketBase REST stays fail-closed --------------------------------------------------------
CODE=$(curl -s -o /dev/null -w '%{http_code}' "${E2E_WEB_BASE}/api/collections/files/records")
[ "$CODE" = "401" ] || [ "$CODE" = "403" ]
check "unauthenticated REST read is refused (401/403)" $?

CODE=$(curl -s -o /dev/null -w '%{http_code}' -H "Authorization: ${BOOT_TOKEN}" "${E2E_WEB_BASE}/api/collections/files/records")
[ "$CODE" = "200" ]
check "REST read with the bootstrap token succeeds (200)" $?

skip "chat streaming (/desk/chat/stream, /desk/chat/reset)" "needs an LLM API key; contract unchanged here, not exercised"

finish_45

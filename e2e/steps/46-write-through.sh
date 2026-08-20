#!/usr/bin/env bash
# shellcheck shell=bash
# 46 — write-through: the browser's one door from screen to disk. POST /desk/doc/write lands a
# frontmatter edit on the real file through the SAME tool the CLI calls, a stale base checksum is
# refused rather than merged (the outside-edit case: the owner writes in another app), the desk-tree
# watcher re-indexes that outside edit with no manual sweep, and the whole thing stays reversible —
# `restore --by-path` puts the recorded original back byte-for-byte. Sourced by e2e.sh; helpers
# (check/skip/note/section/dk) and E2E_* state come from lib.sh.
#
# Starts its OWN `serve` process against the scratch store, on a $$-derived high port with a
# different base than step 45's, so the two could coexist within one suite process.
# ponytail: $$-derived, not a reserved/bound port — good enough for one suite process at a time;
# move to `nc -z`-style probing first if this suite ever runs concurrently on one host.
#
# The idiom `<condition>; check "<desc>" $?` recurs below: a test condition is evaluated purely
# for its exit status, fed to check() as pass/fail. SC2319 structurally misfires on that
# intentional pattern at every call site (same rationale as steps/45-web.sh), so it is disabled
# file-wide; every other lint rule stays enforced. SC2064 also misfires on the trap lines below:
# E2E_WT46_PRIOR_TRAP is a fixed string captured once above, so expanding it at
# `trap`-registration time (not at signal time) is the intended, correct behaviour.
# shellcheck disable=SC2319,SC2064

section "46 · write-through — POST /desk/doc/write, conflict refusal, watcher, restore"

E2E_WT46_PORT=$((21000 + ($$ % 20000)))
E2E_WT46_BASE="http://127.0.0.1:${E2E_WT46_PORT}"
E2E_WT46_LOG="$E2E_WORK/serve-46.log"
E2E_WT46_BODY="$E2E_WORK/serve-46-body"

# --- seed the document BEFORE serve starts, so the watcher's convergence sweep indexes it ----
WT46_REL="notes/write-through.md"
WT46_DOC="$E2E_DESK/$WT46_REL"
WT46_SEED_COPY="$E2E_WORK/write-through-seed.md" # byte-exact keepsake for the restore assertion
mkdir -p "$E2E_DESK/notes"
cat >"$WT46_DOC" <<'SEED'
---
type: guide
status: draft
---

# Write-through fixture

body line
SEED
cp "$WT46_DOC" "$WT46_SEED_COPY"
WT46_SEED_SHA=$(shasum -a 256 "$WT46_DOC" | awk '{print $1}')

"$E2E_BIN" --dir "$E2E_STORE" --dev=false serve --http "127.0.0.1:${E2E_WT46_PORT}" \
  >"$E2E_WT46_LOG" 2>&1 &
E2E_WT46_PID=$!

# kill_web_46 — stop this step's serve process. Idempotent: safe to call more than once.
kill_web_46() { kill "$E2E_WT46_PID" 2>/dev/null; wait "$E2E_WT46_PID" 2>/dev/null; }

# This step is sourced into the SAME shell as every other step, so it shares e2e.sh's own
# `trap cleanup EXIT` (scratch-dir removal). Replacing that trap outright would silently drop
# the outer cleanup, so chain onto whatever is already registered instead.
E2E_WT46_PRIOR_TRAP="$(trap -p EXIT | sed -E "s/^trap -- '(.*)' EXIT\$/\1/")"
trap "kill_web_46; ${E2E_WT46_PRIOR_TRAP}" EXIT

# finish_46 — tear down the serve process and restore the outer trap. Called on every exit path
# out of this step: the early bail below, and once the HTTP assertions are done (the CLI `restore`
# assertion runs AFTER it, so only one process ever holds the SQLite store at a time).
finish_46() {
  kill_web_46
  trap "${E2E_WT46_PRIOR_TRAP}" EXIT
}

# --- poll /api/health until serve actually answers (timeout ~15s) --------------------------
E2E_WT46_READY=1
for _ in $(seq 1 30); do
  if curl -fsS -o /dev/null "${E2E_WT46_BASE}/api/health" 2>/dev/null; then
    E2E_WT46_READY=0
    break
  fi
  sleep 0.5
done
check "serve became ready on 127.0.0.1:${E2E_WT46_PORT} within 15s" "$E2E_WT46_READY"
if [ "$E2E_WT46_READY" -ne 0 ]; then
  note "serve never answered /api/health — log tail: $(tail -n 20 "$E2E_WT46_LOG" 2>/dev/null)"
  finish_46
  return 1
fi

# --- the loopback bootstrap token (needed for the REST read that observes the watcher) -------
CODE=$(curl -s -o "$E2E_WT46_BODY" -w '%{http_code}' "${E2E_WT46_BASE}/desk/bootstrap")
WT46_TOKEN=$(jq -r '.token // empty' <"$E2E_WT46_BODY" 2>/dev/null)
[ "$CODE" = "200" ] && [ -n "$WT46_TOKEN" ]
check "GET /desk/bootstrap on loopback returns a token" $?

# --- POST /desk/doc/write with the loaded checksum -> the edit lands on disk ------------------
CODE=$(curl -s -o "$E2E_WT46_BODY" -w '%{http_code}' -X POST \
  -H 'Content-Type: application/json' \
  -d "{\"path\":\"${WT46_REL}\",\"base_checksum\":\"${WT46_SEED_SHA}\",\"set\":{\"status\":\"active\"}}" \
  "${E2E_WT46_BASE}/desk/doc/write")
WT46_OUTCOME=$(jq -r '.outcome // empty' <"$E2E_WT46_BODY" 2>/dev/null)
[ "$CODE" = "200" ] && [ "$WT46_OUTCOME" = "written" ]
check "POST /desk/doc/write returns 200 outcome=written (got $CODE/$WT46_OUTCOME)" $?

grep -qx 'status: active' "$WT46_DOC"
check "the edit is on DISK — the file itself now reads status: active" $?

# --- an outside edit (another app saves underneath) then a save carrying the STALE checksum ---
cat >"$WT46_DOC" <<'OUTSIDE'
---
type: guide
status: active
---

# Write-through fixture

edited outside this surface
OUTSIDE
WT46_OUTSIDE_SHA=$(shasum -a 256 "$WT46_DOC" | awk '{print $1}')

CODE=$(curl -s -o "$E2E_WT46_BODY" -w '%{http_code}' -X POST \
  -H 'Content-Type: application/json' \
  -d "{\"path\":\"${WT46_REL}\",\"base_checksum\":\"${WT46_SEED_SHA}\",\"set\":{\"status\":\"archived\"}}" \
  "${E2E_WT46_BASE}/desk/doc/write")
WT46_OUTCOME=$(jq -r '.outcome // empty' <"$E2E_WT46_BODY" 2>/dev/null)
[ "$CODE" = "409" ] && [ "$WT46_OUTCOME" = "conflict" ]
check "a stale base_checksum is refused 409 outcome=conflict (got $CODE/$WT46_OUTCOME)" $?

grep -qx 'edited outside this surface' "$WT46_DOC" && ! grep -qx 'status: archived' "$WT46_DOC"
check "the refused save never clobbered the outside edit on disk" $?

WT46_CONFLICT_SHA=$(jq -r '.current_checksum // empty' <"$E2E_WT46_BODY" 2>/dev/null)
[ "$WT46_CONFLICT_SHA" = "$WT46_OUTSIDE_SHA" ]
check "the 409 payload carries the disk's current checksum, so the screen can show the difference" $?

# --- watcher liveness: the index follows the disk with NO manual sweep (timeout ~15s) ---------
WT46_INDEXED=1
for _ in $(seq 1 30); do
  WT46_ROW_SHA=$(curl -s -G -H "Authorization: ${WT46_TOKEN}" \
    --data-urlencode "filter=path='${WT46_REL}'" \
    "${E2E_WT46_BASE}/api/collections/files/records" 2>/dev/null | jq -r '.items[0].checksum // empty' 2>/dev/null)
  if [ "$WT46_ROW_SHA" = "$WT46_OUTSIDE_SHA" ]; then
    WT46_INDEXED=0
    break
  fi
  sleep 0.5
done
check "the watcher re-indexed the outside edit with no sweep (files.checksum == disk sha256)" "$WT46_INDEXED"
if [ "$WT46_INDEXED" -ne 0 ]; then
  note "row checksum stalled at '${WT46_ROW_SHA}', disk is '${WT46_OUTSIDE_SHA}'"
fi

# --- stop serve BEFORE the CLI touches the same store, then prove reversibility ---------------
finish_46

# restore --by-path reverses the LATEST applied revision for this path. The 409 recorded none, so
# that is the write-through above, whose recorded original is the seeded `status: draft` bytes.
WT46_RESTORE=$(dk restore --by-path "$WT46_REL")
RC=$?
WT46_RESTORED=$(printf '%s' "$WT46_RESTORE" | jq -r '.restored' 2>/dev/null)
[ "$RC" -eq 0 ] && [ "$WT46_RESTORED" = "true" ]
check "restore --by-path ${WT46_REL} reverses the write-through revision" $?

cmp -s "$WT46_DOC" "$WT46_SEED_COPY"
check "the file is byte-identical to its pre-write original after restore" $?

#!/usr/bin/env bash
# docker-smoke.sh — scripted proof of the Dockerfile/entrypoint contract: build the image, run it
# against a real named volume, and assert every deployment requirement by HTTP
# status code. Prints one numbered PASS/FAIL line per assertion; exits nonzero if any fail.
# Cleans up its own containers/volumes on exit (trap), regardless of outcome.
#
# Identity-neutral: every credential/email below is a placeholder (you@example.com,
# example.com) — never a real deployment identity.
set -uo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.." || exit 1

IMAGE="deskkit-smoke-test:local"
VOL_MAIN="deskkit-smoke-vol-$$"
VOL_FRESH="deskkit-smoke-vol-fresh-$$"
VOL_SEEDED="deskkit-smoke-vol-seeded-$$"
C1="deskkit-smoke-c1-$$"
C2="deskkit-smoke-c2-$$"
C3="deskkit-smoke-c3-$$"
C4="deskkit-smoke-c4-$$"
C5="deskkit-smoke-c5-$$"
PORT=18173
SEEDED_PORT=$((PORT + 2))
SEED_BUILD_PORT=$((PORT + 3))
# The unprivileged account baked into the image (Dockerfile addgroup/adduser, docker-entrypoint.sh
# APP_UID). The entrypoint drops to it and hands it the two data trees; everything below asserts
# that drop actually happened rather than trusting it.
APP_UID=10001
SEEDED_FILE="/data/store/legacy-root-owned.txt"
SEEDED_FILE_CONTENT="legacy"
# The store's SQLite database, inside $STORE_DIR (docker-entrypoint.sh defaults STORE_DIR=/data/store).
SEEDED_DB="/data/store/data.db"
SU_EMAIL="you@example.com"
SU_PASSWORD="a-strong-superuser-password-123"
USER_EMAIL="newuser@example.com"
USER_PASSWORD="a-strong-user-password-123"
# Written into the seeded volume's database BEFORE that volume is handed back to root, so it is
# the one row whose survival proves a pre-existing database was migrated rather than replaced.
PRE_CHOWN_EMAIL="pre-chown-user@example.com"
# Ceiling for the public SPA shell body. The real shell measured 391 bytes; 4096 leaves room for
# ordinary shell growth (another asset tag, a meta line) while still being an order of magnitude
# below any response that had chat history, findings or a records payload appended to it.
CHAT_SHELL_MAX_BYTES=4096

PASS=0
FAIL=0
N=0

note() { printf '%s\n' "$*"; }

# check NAME EXPECTED ACTUAL — prints "N. PASS/FAIL: NAME (expected=E got=A)".
check() {
	N=$((N + 1))
	local name="$1" expected="$2" actual="$3"
	if [ "$actual" = "$expected" ]; then
		printf '%d. PASS: %s (status=%s)\n' "$N" "$name" "$actual"
		PASS=$((PASS + 1))
	else
		printf '%d. FAIL: %s (expected=%s got=%s)\n' "$N" "$name" "$expected" "$actual"
		FAIL=$((FAIL + 1))
	fi
}

# check_in NAME "E1 E2" ACTUAL — pass if actual is one of a small set of acceptable codes
# (the DoD accepts either 401 or 403 for "unauthenticated -> refused").
check_in() {
	N=$((N + 1))
	local name="$1" expected_set="$2" actual="$3"
	for e in $expected_set; do
		if [ "$actual" = "$e" ]; then
			printf '%d. PASS: %s (status=%s)\n' "$N" "$name" "$actual"
			PASS=$((PASS + 1))
			return
		fi
	done
	printf '%d. FAIL: %s (expected one of [%s] got=%s)\n' "$N" "$name" "$expected_set" "$actual"
	FAIL=$((FAIL + 1))
}

REQ_BODY_FILE="$(mktemp)"
BUILD_LOG="$(mktemp)"

# shellcheck disable=SC2329,SC2317  # invoked indirectly via the `trap cleanup EXIT` below
cleanup() {
	note "--- cleanup ---"
	docker rm -f "$C1" "$C2" "$C3" "$C4" "$C5" >/dev/null 2>&1
	# The two derived fail-closed volumes are listed here as well as at their inline removals: an
	# early exit (or a failure between creation and removal) would otherwise leak one $$-named
	# volume per run, and nothing else ever collects them.
	docker volume rm "$VOL_MAIN" "$VOL_FRESH" "$VOL_SEEDED" \
		"${VOL_FRESH}-badpw" "${VOL_FRESH}-selftest" >/dev/null 2>&1
	rm -f "$REQ_BODY_FILE" "$BUILD_LOG"
}
trap cleanup EXIT

# req METHOD URL [DATA] [AUTH_TOKEN] — makes exactly one curl call, writes the response body to
# $REQ_BODY_FILE, and prints the HTTP status code. Kept as one call per request (never re-issued
# for status vs body) so a non-idempotent request (signup, approve) is never sent twice.
req() {
	local method="$1" url="$2" data="${3:-}" token="${4:-}"
	local args=(-s -o "$REQ_BODY_FILE" -w '%{http_code}' -X "$method" "$url")
	if [ -n "$data" ]; then
		args+=(-H 'Content-Type: application/json' -d "$data")
	fi
	if [ -n "$token" ]; then
		args+=(-H "Authorization: $token")
	fi
	curl "${args[@]}"
}

# wait_for_health PORT TRIES — polls /api/health until 200 or the tries are exhausted.
wait_for_health() {
	local port="$1" tries="${2:-60}"
	local i=0
	while [ "$i" -lt "$tries" ]; do
		code=$(curl -s -o /dev/null -w '%{http_code}' --max-time 1 "http://127.0.0.1:${port}/api/health" 2>/dev/null || true)
		if [ "$code" = "200" ]; then
			return 0
		fi
		i=$((i + 1))
		sleep 0.5
	done
	return 1
}

# owner_uid CONTAINER PATH — prints the numeric owner uid of PATH inside a RUNNING container, or
# "unknown" when it cannot be read (dead container, failed exec, missing path). The "unknown"
# fallback matters: a bare command substitution prints an empty string on failure, and an empty
# string silently satisfies a "this is not uid 0" assertion — the exact vacuous pass the
# privilege-drop assertions below exist to rule out.
owner_uid() {
	local out
	out=$(docker exec "$1" stat -c %u "$2" 2>/dev/null || true)
	case "$out" in
	'' | *[!0-9]*) printf 'unknown' ;;
	*) printf '%s' "$out" ;;
	esac
}

# check_pid1_unprivileged LABEL CONTAINER — 2 assertions about the SERVING process, not an exec'd
# one. `docker exec <c> id -u` reports the uid of the newly exec'd process (root, the image
# default, since the Dockerfile has no USER directive) and says nothing at all about the server;
# the only honest subject is PID 1 itself, whose /proc/1 entry is owned by the uid it runs as.
# Both halves are asserted on purpose: "not root" is the security property, "exactly $APP_UID"
# catches a drop to some other (wrong) account that still happens not to be root.
check_pid1_unprivileged() {
	local label="$1" uid rootness
	uid=$(owner_uid "$2" /proc/1)
	if [ "$uid" = "0" ] || [ "$uid" = "unknown" ]; then
		rootness="root-or-unreadable"
	else
		rootness="not-root"
	fi
	check "${label}: PID 1 (the serving process) is not root" "not-root" "$rootness"
	check "${label}: PID 1 runs as the image's unprivileged uid" "$APP_UID" "$uid"
}

# start_and_check LABEL CONTAINER DOCKER_RUN_ARGS... — starts a detached container and makes 2
# assertions about the start ITSELF, before anything downstream trusts it. Without these, a failed
# `docker run` is invisible: its output is discarded, and every later assertion silently retargets
# whatever else happens to hold that host port — a foreign process can turn the whole section green
# (observed: a leftover container on the same port made 3 assertions pass against the wrong
# server). $IMAGE must be the last argument, as with a bare `docker run`.
start_and_check() {
	local label="$1" cname="$2"
	shift 2
	local started
	if docker run -d --name "$cname" "$@" >/dev/null 2>&1; then
		started="started"
	else
		started="failed"
	fi
	check "${label}: container starts (docker run exits 0 — no port collision, no bad flag)" "started" "$started"
	check "${label}: container is actually running (later assertions target IT, not another process on that port)" "true" \
		"$(docker inspect -f '{{.State.Running}}' "$cname" 2>/dev/null || printf 'no-such-container')"
}

note "=== docker build ==="
if docker build -t "$IMAGE" . >"$BUILD_LOG" 2>&1; then
	N=$((N + 1))
	printf '%d. PASS: docker build succeeds\n' "$N"
	PASS=$((PASS + 1))
else
	N=$((N + 1))
	printf '%d. FAIL: docker build succeeds\n' "$N"
	FAIL=$((FAIL + 1))
	tail -n 60 "$BUILD_LOG"
	note "docker build failed; aborting the rest of the smoke run"
	exit 1
fi
IMAGE_SIZE=$(docker images "$IMAGE" --format '{{.Size}}')
note "image size: $IMAGE_SIZE"

note "=== container 1: boot + unauth/superuser checks ==="
docker volume create "$VOL_MAIN" >/dev/null
start_and_check "container 1" "$C1" -p "${PORT}:8090" \
	-v "${VOL_MAIN}:/data" \
	-e PB_SUPERUSER_EMAIL="$SU_EMAIL" -e PB_SUPERUSER_PASSWORD="$SU_PASSWORD" \
	"$IMAGE"

if ! wait_for_health "$PORT"; then
	note "container 1 never became healthy; log follows"
	docker logs "$C1" 2>&1 | tail -n 60
	exit 1
fi

BASE="http://127.0.0.1:${PORT}"

check_pid1_unprivileged "fresh volume" "$C1"

check "GET /api/health" 200 "$(req GET "${BASE}/api/health")"
check "unauthenticated GET /api/collections/files/records" 403 "$(req GET "${BASE}/api/collections/files/records")"
# /desk/chat is the SPA's own route, resolved by the index fallback: in public mode the static
# shell is served to anyone by design ("shell loads, data doesn't" — the admin-console stance),
# and only the two chat SESSION routes below are behind RequireAuth. So 200 here is correct, and
# the body assertion is what keeps it honest: an inert 391-byte shell, never chat data.
CHAT_SHELL_CODE=$(req GET "${BASE}/desk/chat")
check "unauthenticated GET /desk/chat serves the public SPA shell" 200 "$CHAT_SHELL_CODE"
if grep -qF '<div id="app">' "$REQ_BODY_FILE"; then
	CHAT_SHELL_BODY="spa-shell"
else
	CHAT_SHELL_BODY="not-spa-shell"
fi
check "unauthenticated GET /desk/chat returns the SPA shell markup" "spa-shell" "$CHAT_SHELL_BODY"
# "Shell, not data" needs an upper bound as well as a marker: the marker alone cannot fail if a
# payload were appended after it. An inert shell is small and stays small ($CHAT_SHELL_MAX_BYTES);
# anything carrying chat history or records is far larger.
CHAT_SHELL_BYTES=$(wc -c <"$REQ_BODY_FILE" | tr -dc '0-9')
if [ -n "$CHAT_SHELL_BYTES" ] && [ "$CHAT_SHELL_BYTES" -gt 0 ] && [ "$CHAT_SHELL_BYTES" -le "$CHAT_SHELL_MAX_BYTES" ]; then
	CHAT_SHELL_SIZE="inert-shell"
else
	CHAT_SHELL_SIZE="unexpected-body-size(${CHAT_SHELL_BYTES:-none})"
fi
check "unauthenticated GET /desk/chat carries no data (body within the inert-shell ceiling)" "inert-shell" "$CHAT_SHELL_SIZE"
check "unauthenticated POST /desk/chat/stream" 401 "$(req POST "${BASE}/desk/chat/stream")"
check "unauthenticated POST /desk/chat/reset" 401 "$(req POST "${BASE}/desk/chat/reset")"

SU_LOGIN_CODE=$(req POST "${BASE}/api/collections/_superusers/auth-with-password" \
	"{\"identity\":\"${SU_EMAIL}\",\"password\":\"${SU_PASSWORD}\"}")
check "superuser login" 200 "$SU_LOGIN_CODE"
SU_TOKEN=$(jq -r '.token // empty' "$REQ_BODY_FILE")

check "superuser-authenticated GET /api/collections/files/records" 200 \
	"$(req GET "${BASE}/api/collections/files/records" "" "$SU_TOKEN")"

note "=== container 1: approval gate ==="
SIGNUP_CODE=$(req POST "${BASE}/api/collections/users/records" \
	"{\"email\":\"${USER_EMAIL}\",\"password\":\"${USER_PASSWORD}\",\"passwordConfirm\":\"${USER_PASSWORD}\"}")
check "self-signup succeeds" 200 "$SIGNUP_CODE"
USER_ID=$(jq -r '.id // empty' "$REQ_BODY_FILE")

check "self-signup attempting approved:true is rejected" 400 \
	"$(req POST "${BASE}/api/collections/users/records" \
		"{\"email\":\"sneaky-${USER_EMAIL}\",\"password\":\"${USER_PASSWORD}\",\"passwordConfirm\":\"${USER_PASSWORD}\",\"approved\":true}")"

check_in "login as unapproved self-signup user is refused" "401 403" \
	"$(req POST "${BASE}/api/collections/users/auth-with-password" \
		"{\"identity\":\"${USER_EMAIL}\",\"password\":\"${USER_PASSWORD}\"}")"

check "superuser approves + verifies the user" 200 \
	"$(curl -s -o "$REQ_BODY_FILE" -w '%{http_code}' -X PATCH "${BASE}/api/collections/users/records/${USER_ID}" \
		-H "Authorization: ${SU_TOKEN}" -H 'Content-Type: application/json' \
		-d '{"approved":true,"verified":true}')"

USER_LOGIN_CODE=$(req POST "${BASE}/api/collections/users/auth-with-password" \
	"{\"identity\":\"${USER_EMAIL}\",\"password\":\"${USER_PASSWORD}\"}")
check "login after approval succeeds" 200 "$USER_LOGIN_CODE"
USER_TOKEN=$(jq -r '.token // empty' "$REQ_BODY_FILE")

# The auth gate has to be exercised on a route that is actually gated. GET /desk/chat is the
# public SPA shell (asserted above), so a token proves nothing there; POST /desk/chat/reset is
# 401 without a token (asserted above) and 204 with an approved user's token — observed against a
# running container, not assumed. The pair is the real before/after of the gate.
check "approved user's token is accepted by the gated chat session route (POST /desk/chat/reset)" 204 \
	"$(req POST "${BASE}/desk/chat/reset" "" "$USER_TOKEN")"

note "=== restart persistence (same named volume) ==="
docker rm -f "$C1" >/dev/null

start_and_check "container 2 (restart)" "$C2" -p "${PORT}:8090" \
	-v "${VOL_MAIN}:/data" \
	-e PB_SUPERUSER_EMAIL="$SU_EMAIL" -e PB_SUPERUSER_PASSWORD="$SU_PASSWORD" \
	"$IMAGE"

# A failed restart must still produce numbered, named FAIL lines: bumping $FAIL without $N used to
# shrink the suite's own total ("of 38") and named nothing, which is exactly the shape of a gate
# that quietly stops covering something. So health is captured as a value and every assertion below
# runs unconditionally — an unhealthy restart fails three named assertions instead of one anonymous
# counter, and the total stays the same whatever happens.
if wait_for_health "$PORT"; then
	RESTART_HEALTH="healthy"
else
	RESTART_HEALTH="unhealthy"
	note "container 2 (restart) never became healthy; log follows"
	docker logs "$C2" 2>&1 | tail -n 60
fi
check "restarted container: becomes healthy again on the same volume" "healthy" "$RESTART_HEALTH"
check "restarted container: GET /api/health" 200 "$(req GET "${BASE}/api/health")"
check "restarted container: approved user's record survived (login still succeeds)" 200 \
	"$(req POST "${BASE}/api/collections/users/auth-with-password" \
		"{\"identity\":\"${USER_EMAIL}\",\"password\":\"${USER_PASSWORD}\"}")"
docker rm -f "$C2" >/dev/null

note "=== pre-existing ROOT-OWNED volume: chown-then-drop migration ==="
# The upgrade case, and the one that would break a live deployment: a volume already carrying a
# REAL database, written before the privilege drop existed.
#
# Seeding it with `deskkit init` plus an empty store directory would prove nothing: no database
# would exist, so the boot under test would create one itself, and "a write succeeds" would only
# exercise a database that same boot had just made. The seed therefore boots the image normally,
# inserts a row through the API (which creates data.db and its -wal/-shm alongside it), and only
# then hands the whole tree back to root.
docker volume create "$VOL_SEEDED" >/dev/null

start_and_check "seed boot" "$C5" -p "${SEED_BUILD_PORT}:8090" \
	-v "${VOL_SEEDED}:/data" \
	-e PB_SUPERUSER_EMAIL="$SU_EMAIL" -e PB_SUPERUSER_PASSWORD="$SU_PASSWORD" \
	"$IMAGE"

if wait_for_health "$SEED_BUILD_PORT"; then
	SEED_BOOT_HEALTH="healthy"
else
	SEED_BOOT_HEALTH="unhealthy"
	note "seed boot never became healthy; log follows"
	docker logs "$C5" 2>&1 | tail -n 60
fi
check "seed boot: serves, so a real database now exists on the volume" "healthy" "$SEED_BOOT_HEALTH"

SEED_BASE="http://127.0.0.1:${SEED_BUILD_PORT}"
check "seed boot: a row is written into that database BEFORE the volume is handed back to root" 200 \
	"$(req POST "${SEED_BASE}/api/collections/users/records" \
		"{\"email\":\"${PRE_CHOWN_EMAIL}\",\"password\":\"${USER_PASSWORD}\",\"passwordConfirm\":\"${USER_PASSWORD}\"}")"
docker rm -f "$C5" >/dev/null

# Hand the whole tree — database included — back to root, exactly as an older root-running image
# would have left it. The entrypoint is bypassed (--user 0 + --entrypoint sh) so no privilege drop
# can happen here; the text file is written in the same pass as a plain non-database companion.
if docker run --rm --user 0 --entrypoint sh -v "${VOL_SEEDED}:/data" "$IMAGE" \
	-c "echo ${SEEDED_FILE_CONTENT} > ${SEEDED_FILE} && chown -R 0:0 /data" >/dev/null 2>&1; then
	SEED_RESULT="ok"
else
	SEED_RESULT="failed"
fi
check "seeded volume: the whole tree, database included, is handed back to root" "ok" "$SEED_RESULT"
# Guard against a seed that silently did nothing: if the database were not root-owned going in,
# every migration assertion below would be measuring an already-correct volume and proving nothing.
check "seeded volume: the database really IS root-owned before the boot under test" "0" \
	"$(docker run --rm --user 0 --entrypoint sh -v "${VOL_SEEDED}:/data" "$IMAGE" \
		-c "stat -c %u ${SEEDED_DB} 2>/dev/null || printf unknown")"

start_and_check "seeded root-owned volume" "$C4" -p "${SEEDED_PORT}:8090" \
	-v "${VOL_SEEDED}:/data" \
	-e PB_SUPERUSER_EMAIL="$SU_EMAIL" -e PB_SUPERUSER_PASSWORD="$SU_PASSWORD" \
	"$IMAGE"

if wait_for_health "$SEEDED_PORT"; then
	SEEDED_HEALTH="healthy"
else
	SEEDED_HEALTH="unhealthy"
	note "seeded-volume container never became healthy; log follows"
	docker logs "$C4" 2>&1 | tail -n 60
fi
# The remaining assertions run unconditionally even when the boot failed: each one reports its own
# red line (uid "unknown", body mismatch, curl status 000) instead of being silently skipped.
check "seeded root-owned volume: container still boots and serves" "healthy" "$SEEDED_HEALTH"

check_pid1_unprivileged "seeded root-owned volume" "$C4"

check "seeded root-owned volume: /data/desk ownership migrated to the app uid" "$APP_UID" \
	"$(owner_uid "$C4" /data/desk)"
check "seeded root-owned volume: /data/store ownership migrated to the app uid" "$APP_UID" \
	"$(owner_uid "$C4" /data/store)"
# The top of each tree could be chowned by a non-recursive fix while every file under it stayed
# root-owned (which is what actually breaks writes), so a pre-existing FILE is checked too.
check "seeded root-owned volume: the pre-existing root-owned file survived the migration" \
	"$SEEDED_FILE_CONTENT" "$(docker exec "$C4" cat "$SEEDED_FILE" 2>/dev/null || printf 'unreadable')"
check "seeded root-owned volume: chown -R reached the tree's contents, not just its top dir" "$APP_UID" \
	"$(owner_uid "$C4" "$SEEDED_FILE")"
# The file that actually matters: a root-owned database the dropped process must be able to write.
check "seeded root-owned volume: the pre-existing database file itself migrated to the app uid" "$APP_UID" \
	"$(owner_uid "$C4" "$SEEDED_DB")"
# Least-privilege boundary: only the two data trees are handed over; their parent stays root-owned
# because the app never creates entries directly in it (docker-entrypoint.sh says so explicitly).
check "seeded root-owned volume: /data itself stays root-owned (only the two data trees are handed over)" "0" \
	"$(owner_uid "$C4" /data)"

# Ownership that merely looks right still has to carry real reads and writes on the migrated
# database: the row seeded BEFORE the chown must still be there (no data loss), and a new insert
# must succeed (the store is writable, not just readable).
SEEDED_BASE="http://127.0.0.1:${SEEDED_PORT}"
SEEDED_SU_CODE=$(req POST "${SEEDED_BASE}/api/collections/_superusers/auth-with-password" \
	"{\"identity\":\"${SU_EMAIL}\",\"password\":\"${SU_PASSWORD}\"}")
check "seeded root-owned volume: superuser login succeeds (the migrated store is readable)" 200 "$SEEDED_SU_CODE"
SEEDED_SU_TOKEN=$(jq -r '.token // empty' "$REQ_BODY_FILE" 2>/dev/null)

check "seeded root-owned volume: the users list is readable with that token" 200 \
	"$(req GET "${SEEDED_BASE}/api/collections/users/records?perPage=200" "" "$SEEDED_SU_TOKEN")"
# The no-data-loss proof: this row was inserted into the database while it was still owned by the
# seed boot's user, then the whole tree was chowned to root, then this boot chowned it back and
# opened it. Counting the matching rows (not just "the request worked") keeps it non-vacuous — a
# missing row, an unreadable body or a wiped database all yield something other than exactly 1.
check "seeded root-owned volume: the row written BEFORE the chown is still readable after it" "1" \
	"$(jq --arg e "$PRE_CHOWN_EMAIL" '[.items[]? | select(.email == $e)] | length' "$REQ_BODY_FILE" 2>/dev/null || printf 'unreadable')"

check "seeded root-owned volume: a real DB write (self-signup insert) succeeds" 200 \
	"$(req POST "${SEEDED_BASE}/api/collections/users/records" \
		"{\"email\":\"${USER_EMAIL}\",\"password\":\"${USER_PASSWORD}\",\"passwordConfirm\":\"${USER_PASSWORD}\"}")"

docker rm -f "$C4" >/dev/null
docker volume rm "$VOL_SEEDED" >/dev/null 2>&1

# The refusal's log message (deskkit: internal/core/store/superuser.go CheckServeAuthPrereqs).
# Matched as a substring so a reworded surrounding sentence doesn't break this — but re-verify
# this literal against the actual source if the Go refusal path changes.
FAILCLOSED_MSG="refusing to serve on a non-loopback address"

# refused_cleanly VOL EXTRA_DOCKER_RUN_ARGS... — runs $IMAGE with the given extra `docker run` args
# on the named volume and prints exactly one of four values:
#
#   yes           the container exited 1 AND its log names the specific refusal ($FAILCLOSED_MSG)
#   no            the container exited some other way, without that message
#   still-running it was still up when the wait ran out — it booted, it did not refuse
#   not-started   `docker run` failed, or the container could not be inspected afterwards
#
# The last two exist because "not yes" is not a fact. The earlier two-value version collapsed
# every non-refusal into "no", so a name collision on $C3, a missing image or a docker hiccup made
# the "no" self-test below PASS while proving nothing — the same false-pass class as an unchecked
# `docker run`. None of the three assertions accepts "not-started" or "still-running", so an
# infrastructure failure now reddens every one of them instead of silently satisfying one.
# No PASS/FAIL/N side effects here; only check() counts and prints.
refused_cleanly() {
	local vol="$1"
	shift
	local cname="$C3"
	if ! docker run -d --name "$cname" "$@" -v "${vol}:/data" "$IMAGE" >/dev/null 2>&1; then
		docker rm -f "$cname" >/dev/null 2>&1
		note "  (docker run failed: the container never started, so nothing was proved)" >&2
		printf 'not-started'
		return
	fi
	local exitcode log state="" i=0
	while [ "$i" -lt 40 ]; do
		state=$(docker inspect -f '{{.State.Running}}' "$cname" 2>/dev/null || printf 'gone')
		if [ "$state" != "true" ]; then
			break
		fi
		i=$((i + 1))
		sleep 0.5
	done
	if [ "$state" = "true" ]; then
		docker rm -f "$cname" >/dev/null 2>&1
		note "  (still running after the wait: it booted rather than refusing)" >&2
		printf 'still-running'
		return
	fi
	exitcode=$(docker inspect -f '{{.State.ExitCode}}' "$cname" 2>/dev/null || printf '')
	log=$(docker logs "$cname" 2>&1)
	docker rm -f "$cname" >/dev/null 2>&1
	if [ -z "$exitcode" ]; then
		note "  (container could not be inspected: gone before its exit code could be read)" >&2
		printf 'not-started'
		return
	fi
	if [ "$exitcode" = "1" ] && printf '%s' "$log" | grep -qF "$FAILCLOSED_MSG"; then
		printf 'yes'
	else
		printf 'no'
	fi
	note "  (exit=$exitcode; refusal message present=$(printf '%s' "$log" | grep -qF "$FAILCLOSED_MSG" && echo yes || echo no))" >&2
}

note "=== fail-closed: public bind, no superuser prerequisites, fresh volume ==="
docker volume create "$VOL_FRESH" >/dev/null
check "fail-closed — no superuser env on a public bind refuses to start (exit=1 + refusal message)" "yes" \
	"$(refused_cleanly "$VOL_FRESH" -p "$((PORT + 1)):8090")"

note "=== fail-closed: public bind, a password the framework rejects (too short) ==="
docker volume create "${VOL_FRESH}-badpw" >/dev/null
# This is the branch that used to silently boot before the Go-side fix (the gate predicted
# success instead of verifying it): both env vars are set, but PB_SUPERUSER_PASSWORD is below
# the framework's minimum, so EnsureSuperuser's Save() fails and the gate must refuse fatally.
check "fail-closed — a rejected (too-short) superuser password refuses on a public bind" "yes" \
	"$(refused_cleanly "${VOL_FRESH}-badpw" -e PB_SUPERUSER_EMAIL="$SU_EMAIL" -e PB_SUPERUSER_PASSWORD=short)"
docker volume rm "${VOL_FRESH}-badpw" >/dev/null 2>&1

note "=== proving that check is not vacuous: a DIFFERENT startup refusal must NOT satisfy it ==="
docker volume create "${VOL_FRESH}-selftest" >/dev/null
# Same public bind, but a HALF-configured superuser pair (email set, password unset) — a real,
# deterministic, code-reachable refusal (ValidateSuperuserEnv) that also exits 1 but logs a
# completely different message ("half-configured superuser environment: ..."), never the
# no-superuser-on-a-public-bind text. Expect "no": proves refused_cleanly can go red, i.e. the
# assertion above is not just checking "container isn't running".
check "self-test: a different refusal (half-configured superuser) is correctly NOT counted as the fail-closed refusal" "no" \
	"$(refused_cleanly "${VOL_FRESH}-selftest" -e PB_SUPERUSER_EMAIL="$SU_EMAIL")"
docker volume rm "${VOL_FRESH}-selftest" >/dev/null 2>&1

note ""
note "=== summary: ${PASS} passed, ${FAIL} failed (of $((PASS + FAIL))) ==="
if [ "$FAIL" -gt 0 ]; then
	exit 1
fi
exit 0

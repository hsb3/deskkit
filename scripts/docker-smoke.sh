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
C1="deskkit-smoke-c1-$$"
C2="deskkit-smoke-c2-$$"
C3="deskkit-smoke-c3-$$"
PORT=18173
SU_EMAIL="you@example.com"
SU_PASSWORD="a-strong-superuser-password-123"
USER_EMAIL="newuser@example.com"
USER_PASSWORD="a-strong-user-password-123"

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
	docker rm -f "$C1" "$C2" "$C3" >/dev/null 2>&1
	docker volume rm "$VOL_MAIN" "$VOL_FRESH" >/dev/null 2>&1
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
	local port="$1" tries="${2:-30}"
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
docker run -d --name "$C1" -p "${PORT}:8090" \
	-v "${VOL_MAIN}:/data" \
	-e PB_SUPERUSER_EMAIL="$SU_EMAIL" -e PB_SUPERUSER_PASSWORD="$SU_PASSWORD" \
	"$IMAGE" >/dev/null

if ! wait_for_health "$PORT"; then
	note "container 1 never became healthy; log follows"
	docker logs "$C1" 2>&1 | tail -n 60
	exit 1
fi

BASE="http://127.0.0.1:${PORT}"

check "GET /api/health" 200 "$(req GET "${BASE}/api/health")"
check "unauthenticated GET /api/collections/files/records" 403 "$(req GET "${BASE}/api/collections/files/records")"
check "unauthenticated GET /desk/chat" 401 "$(req GET "${BASE}/desk/chat")"
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

check "approved user's token authenticates GET /desk/chat" 200 \
	"$(req GET "${BASE}/desk/chat" "" "$USER_TOKEN")"

note "=== restart persistence (same named volume) ==="
docker rm -f "$C1" >/dev/null

docker run -d --name "$C2" -p "${PORT}:8090" \
	-v "${VOL_MAIN}:/data" \
	-e PB_SUPERUSER_EMAIL="$SU_EMAIL" -e PB_SUPERUSER_PASSWORD="$SU_PASSWORD" \
	"$IMAGE" >/dev/null

if ! wait_for_health "$PORT"; then
	note "container 2 (restart) never became healthy; log follows"
	docker logs "$C2" 2>&1 | tail -n 60
	FAIL=$((FAIL + 1))
else
	check "restarted container: GET /api/health" 200 "$(req GET "${BASE}/api/health")"
	check "restarted container: approved user's record survived (login still succeeds)" 200 \
		"$(req POST "${BASE}/api/collections/users/auth-with-password" \
			"{\"identity\":\"${USER_EMAIL}\",\"password\":\"${USER_PASSWORD}\"}")"
fi
docker rm -f "$C2" >/dev/null

# The refusal's log message (deskkit: internal/core/store/superuser.go CheckServeAuthPrereqs).
# Matched as a substring so a reworded surrounding sentence doesn't break this — but re-verify
# this literal against the actual source if the Go refusal path changes.
FAILCLOSED_MSG="refusing to serve on a non-loopback address"

# refused_cleanly VOL EXTRA_DOCKER_RUN_ARGS... — runs $IMAGE with the given extra `docker run`
# args on the named volume and prints "yes"/"no": "yes" only when BOTH the container's own exit
# code is 1 AND its log names the specific refusal ($FAILCLOSED_MSG) — never on "not running" or
# "some log line" alone, since an unrelated startup crash (e.g. a bad --dir) produces exactly that
# same pair and would otherwise make the caller's assertion pass vacuously. No PASS/FAIL/N side
# effects here; only check() below does the counting/printing, so this can be called once for the
# real refusal and once more (with different args) to prove the check can go red.
refused_cleanly() {
	local vol="$1"
	shift
	local cname="$C3"
	docker run -d --name "$cname" "$@" -v "${vol}:/data" "$IMAGE" >/dev/null
	sleep 3
	local exitcode log
	exitcode=$(docker inspect -f '{{.State.ExitCode}}' "$cname" 2>/dev/null || echo "-1")
	log=$(docker logs "$cname" 2>&1)
	docker rm -f "$cname" >/dev/null 2>&1
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

#!/usr/bin/env bash
# run.sh — the SPA browser gate: seed a throwaway desk, serve it, drive a real browser at it.
#
# Never touches a real desk or a real store: the desk and the store are both under mktemp and
# both are removed on exit, including on SIGINT.
#
# Usage:  bash e2e/spa/run.sh
#         DESKKIT_BIN=/path/to/deskkit bash e2e/spa/run.sh   (reuse a prebuilt binary)

set -uo pipefail

SPA_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO="$(cd "$SPA_DIR/../.." && pwd)"

WORK="$(mktemp -d "${TMPDIR:-/tmp}/deskkit-spa.XXXXXX")"
DESK="$WORK/spa-desk"
STORE="$WORK/store"
SERVE_PID=""

# shellcheck disable=SC2329,SC2317  # invoked indirectly by the EXIT trap
cleanup() {
  [ -n "$SERVE_PID" ] && kill "$SERVE_PID" 2> /dev/null
  chmod -R u+w "$WORK" 2> /dev/null || true
  rm -rf "$WORK"
}
trap cleanup EXIT

echo "deskkit SPA browser gate"
echo "  desk:  $DESK"

# The binary must embed the CURRENT SPA build, so `make build` (which builds web/ first) is the
# only correct way to produce it — a bare `go build` serves a placeholder page and every check
# below would fail for the wrong reason.
BIN="${DESKKIT_BIN:-}"
if [ -z "$BIN" ]; then
  echo "  building deskkit (with the SPA embedded)…"
  make -C "$REPO" build > /dev/null || {
    echo "FATAL: make build failed"
    exit 1
  }
  BIN="$REPO/deskkit"
fi
echo "  binary: $BIN"

node "$SPA_DIR/fixture.mjs" "$DESK" || {
  echo "FATAL: could not seed the fixture desk"
  exit 1
}

# --dev=false: a binary under mktemp trips PocketBase's go-run heuristic, which turns on SQL
# debug output and corrupts anything parsing stdout.
(cd "$DESK" && "$BIN" sweep --dir "$STORE" --dev=false > /dev/null) || {
  echo "FATAL: sweep failed"
  exit 1
}

# Port 0 is not an option (serve needs a concrete bind), so take an ephemeral one from the OS and
# use it. A collision in the gap between asking and binding is possible but not worth a retry
# loop for a local gate.
PORT="$(node -e 'const s=require("net").createServer();s.listen(0,"127.0.0.1",()=>{console.log(s.address().port);s.close()})')"
URL="http://127.0.0.1:$PORT"
echo "  serving on $URL"

(cd "$DESK" && "$BIN" serve --dir "$STORE" --http "127.0.0.1:$PORT" --dev=false > "$WORK/serve.log" 2>&1) &
SERVE_PID=$!

READY=""
for _ in $(seq 1 40); do
  if curl -fsS -o /dev/null "$URL/api/health" 2> /dev/null; then
    READY=1
    break
  fi
  sleep 0.25
done
if [ -z "$READY" ]; then
  echo "FATAL: serve never became healthy. Log:"
  cat "$WORK/serve.log"
  exit 1
fi

echo
node "$SPA_DIR/checks.mjs" "$URL" "$DESK"
RC=$?

if [ "$RC" -ne 0 ]; then
  echo
  echo "serve log:"
  tail -n 20 "$WORK/serve.log"
fi

exit "$RC"

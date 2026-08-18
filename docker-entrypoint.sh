#!/bin/sh
# Minimal entrypoint: default PORT, idempotently scaffold the desk onto the volume, exec serve.
set -eu

PORT="${PORT:-8090}"
# ":-" defaults on unset OR empty, so an explicit `-e STORE_DIR=` (blank) is already caught here.
STORE_DIR="${STORE_DIR:-/data/store}"
# DESK_ROOT has no such default (the Dockerfile ENV supplies the normal value): ":?" refuses a
# genuinely empty value instead of letting `deskkit init ""` scaffold into this shell's cwd ("/",
# since the runtime stage sets no WORKDIR) — an explicit `-e DESK_ROOT=` used to slip past `set -u`
# because the var is still SET, just empty.
: "${DESK_ROOT:?DESK_ROOT must be a non-empty path}"

# The desk lives on the volume, never baked into the image, so a redeploy (new container,
# same volume, no rebuild) finds the existing profile and does nothing here.
if [ ! -f "${DESK_ROOT}/_knowledge/profile.yaml" ]; then
  deskkit init "$DESK_ROOT"
fi

# exec so signals reach the binary directly and its exit code is not swallowed by this shell.
exec deskkit serve --http "0.0.0.0:${PORT}" --dir "$STORE_DIR"

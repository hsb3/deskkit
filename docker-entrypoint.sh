#!/bin/sh
# Entrypoint: default PORT, hand the volume to an unprivileged user and permanently drop to it,
# idempotently scaffold the desk onto the volume, exec serve.
set -eu

# The unprivileged account baked into the image (Dockerfile: addgroup/adduser). Not configurable
# on purpose — a wrong value here means either a failed boot or a silent root run.
APP_USER=deskkit
APP_UID=10001

PORT="${PORT:-8090}"
# ":-" defaults on unset OR empty, so an explicit `-e STORE_DIR=` (blank) is already caught here.
STORE_DIR="${STORE_DIR:-/data/store}"
# DESK_ROOT has no such default (the Dockerfile ENV supplies the normal value): ":?" refuses a
# genuinely empty value instead of letting `deskkit init ""` scaffold into this shell's cwd ("/",
# since the runtime stage sets no WORKDIR) — an explicit `-e DESK_ROOT=` used to slip past `set -u`
# because the var is still SET, just empty.
: "${DESK_ROOT:?DESK_ROOT must be a non-empty path}"

# --- privilege drop -------------------------------------------------------------------------
# The image has no USER directive: the container starts as root solely so this block can hand
# the mounted volume to $APP_USER, then replace itself with the unprivileged process. A platform
# volume arrives owned by root, and so does everything an older root-running image wrote to it,
# so the chown has to happen at boot, on the volume, by someone who already holds root.
#
# Everything the server writes lives under $DESK_ROOT and $STORE_DIR, so those two trees are the
# whole chown surface. Their parent (/data by default) is deliberately left root-owned: the app
# never creates entries directly in it. The server also needs a writable /tmp — PocketBase spills
# oversized request bodies to os.TempDir() — which a container's default sticky-bit /tmp already
# gives an unprivileged user; a read-only or restrictively-mounted /tmp will break uploads.
if [ "$(id -u)" = "0" ]; then
  # Refuse an obviously wrong target before mutating anything: `chown -R` on "/" would rewrite the
  # whole container filesystem. Cheap guard on the one value an operator can get catastrophically
  # wrong via -e DESK_ROOT=/ or -e STORE_DIR=/.
  for dir in "$DESK_ROOT" "$STORE_DIR"; do
    if [ "$dir" = "/" ]; then
      echo "entrypoint: FATAL refusing to chown \"/\"; set DESK_ROOT/STORE_DIR to a real subdirectory" >&2
      exit 1
    fi
  done

  mkdir -p "$DESK_ROOT" "$STORE_DIR"

  # Unconditional and recursive, every boot. Sampling the top directory to skip an already-owned
  # tree looks like a free optimisation and is a trap: `chown -R` chowns the top FIRST and keeps
  # going past errors, so a run interrupted or partially failed part-way leaves the top owned and
  # the contents root-owned. The next boot would then skip, drop privileges, and serve a database
  # it cannot write — with a passing healthcheck, which is the worst possible failure shape.
  # Re-running it every time makes that state self-healing. Measured cost: ~4000 files in <20ms,
  # so the skip was buying nothing.
  # ponytail: full walk each boot; revisit only if a desk ever gets big enough to measure.
  for dir in "$DESK_ROOT" "$STORE_DIR"; do
    echo "entrypoint: chowning ${dir} to ${APP_USER} (uid ${APP_UID})" >&2
    if ! chown -R "${APP_USER}:${APP_USER}" "$dir"; then
      echo "entrypoint: FATAL cannot chown ${dir} to ${APP_USER}; refusing to serve as root" >&2
      exit 1
    fi
  done

  # exec, so the dropped process REPLACES this root shell: no root process survives the drop.
  # Re-entering this same script (rather than exec'ing serve directly) keeps every later step —
  # `deskkit init` included — unprivileged, so a first boot cannot write root-owned files onto
  # the volume. The re-entry cannot loop: $APP_UID is not 0, so this block is skipped next pass.
  # If su-exec is missing or the account does not exist, exec fails and the container dies loudly
  # rather than continuing with root privileges.
  echo "entrypoint: dropping privileges to ${APP_USER} (uid ${APP_UID})" >&2
  exec su-exec "$APP_USER" "$0" "$@"
fi

# Past this point the process is unprivileged — either because the block above dropped it, or
# because the container was started with an explicit non-root --user. In the latter case there is
# no root to chown with, so refuse a boot that would only fail later, deeper, and less clearly.
for dir in "$DESK_ROOT" "$STORE_DIR"; do
  if [ ! -d "$dir" ] || [ ! -w "$dir" ]; then
    echo "entrypoint: FATAL ${dir} is not a writable directory for uid $(id -u)." >&2
    echo "entrypoint: run this image without an explicit --user so it can chown the volume itself," >&2
    echo "entrypoint: or pre-chown the mounted volume to that uid." >&2
    exit 1
  fi
done

# The desk lives on the volume, never baked into the image, so a redeploy (new container,
# same volume, no rebuild) finds the existing profile and does nothing here.
if [ ! -f "${DESK_ROOT}/_knowledge/profile.yaml" ]; then
  deskkit init "$DESK_ROOT"
fi

# exec so signals reach the binary directly and its exit code is not swallowed by this shell.
exec deskkit serve --http "0.0.0.0:${PORT}" --dir "$STORE_DIR"

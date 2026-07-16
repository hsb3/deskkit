#!/bin/bash
# run-sandboxed.sh — the macOS sandbox-exec invocation for an initial/supervised
# pocket-librarian run (spec §10.5, default profile). Belt-and-suspenders isolation around
# the record-original-first write boundary (decision 0014): it constrains what the process
# can touch regardless of a tool bug. It does NOT change the agent harness, tool logic, or
# the §5.4 write gate — the same binary runs, just fenced in. Owner-overridable: run the
# binary directly (no sandbox-exec) if you need to step outside these three subtrees.
#
# Required: DESK_ROOT (the desk to steward) and the command to run (e.g. `apply-fix --run
# <run_id>` or `serve --http=127.0.0.1:8090`).
#
# Adaptation note: the spec's illustrative example nests PB_DATA under DESK_ROOT
# ("$DESK_ROOT/pb_data"). This build's actual default is PocketBase's own convention — a
# `pb_data/` directory relative to the binary's working directory (see ../.gitignore),
# i.e. normally this sandbox/ directory's parent (the librarian module root), NOT under
# DESK_ROOT. PB_DATA below defaults to that real location; override it if you run the
# binary from elsewhere or pass --dir explicitly.
set -euo pipefail
cd "$(dirname "$0")/.."

: "${DESK_ROOT:?set DESK_ROOT to the desk this run stewards}"
: "${DESK_NAME:?set DESK_NAME}"
BIN="${BIN:-./pocket-librarian}"
PB_DATA="${PB_DATA:-$(pwd)/pb_data}"
BIN_DIR="$(cd "$(dirname "$BIN")" && pwd)"

# PROVIDER_HOSTPORT is derived from the resolved provider base URL (spec §10.5): the default
# below is Anthropic's; swap to api.openai.com:443 or generativelanguage.googleapis.com:443
# when LLM_PROVIDER changes, so the sandbox's network allowance always matches the active
# provider.
PROVIDER_HOSTPORT="${PROVIDER_HOSTPORT:-api.anthropic.com:443}"

if [ "$#" -eq 0 ]; then
  echo "usage: DESK_ROOT=... DESK_NAME=... $0 <pocket-librarian subcommand + args>" >&2
  echo "  e.g.: $0 apply-fix --run <run_id>" >&2
  echo "  e.g.: $0 serve --http=127.0.0.1:8090" >&2
  exit 2
fi

exec sandbox-exec \
  -D DESK_ROOT="$DESK_ROOT" \
  -D PB_DATA="$PB_DATA" \
  -D BIN_DIR="$BIN_DIR" \
  -D PROVIDER_HOSTPORT="$PROVIDER_HOSTPORT" \
  -f sandbox/pocket-librarian.sb \
  "$BIN" "$@"

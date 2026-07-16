#!/bin/bash
# docker-run.sh — the portable / CI alternative to the macOS sandbox-exec profile (spec
# §10.5). DESK_ROOT is bind-mounted read-write at the SAME path inside the container (so
# config/paths stay identity-neutral), the provider endpoint is reachable, and there is no
# other egress. Docker does not filter egress per-host natively, so provider-only egress is
# enforced by a user-defined network whose firewall/egress policy allows only the provider
# host (the commented HTTPS_PROXY line is the allow-list-proxy alternative). The provider
# host is the same base-URL-derived host as the macOS profile (PROVIDER_HOSTPORT there,
# PROVIDER_HOST here) — a provider swap updates one allow-list entry in both.
#
# Prerequisite (one-time, outside this script): create the egress-restricted network, e.g.
#   docker network create pocket-librarian-egress
# and configure its firewall/egress policy to permit only $PROVIDER_HOST:443 + DNS.
set -euo pipefail

: "${DESK_ROOT:?set DESK_ROOT to the desk this run stewards}"
: "${DESK_NAME:?set DESK_NAME}"
IMAGE="${IMAGE:-pocket-librarian:latest}"
NETWORK="${NETWORK:-pocket-librarian-egress}"
PROVIDER_HOST="${PROVIDER_HOST:-api.anthropic.com}" # swap per LLM_PROVIDER (api.openai.com / generativelanguage.googleapis.com)

if [ "$#" -eq 0 ]; then
  echo "usage: DESK_ROOT=... DESK_NAME=... $0 <pocket-librarian subcommand + args>" >&2
  echo "  e.g.: $0 apply-fix --run <run_id>" >&2
  exit 2
fi

# Alternative to --network "$NETWORK" above: drop it and instead pass
# `-e HTTPS_PROXY=http://allowlist-proxy:3128` with only $PROVIDER_HOST allow-listed at
# that proxy.
exec docker run --rm \
  -v "$DESK_ROOT":"$DESK_ROOT":rw \
  -e ANTHROPIC_API_KEY -e DESK_ROOT -e DESK_NAME \
  --network "$NETWORK" \
  "$IMAGE" \
  "$@"

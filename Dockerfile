# Multi-stage build for the deskkit binary (librarian lane). See docs/pattern.md.
# Architecture-neutral: no hardcoded GOARCH/base-arch tag, so this builds on both
# arm64 (local Apple Silicon Docker) and amd64 (typical hosting target).

FROM golang:1.25-alpine AS build
WORKDIR /src
COPY VERSION VERSION
COPY librarian/go.mod librarian/go.sum librarian/
RUN cd librarian && go mod download
COPY librarian/ librarian/
RUN cd librarian && CGO_ENABLED=0 go build \
      -ldflags "-X main.version=$(cat ../VERSION)" \
      -o /out/deskkit ./cmd/deskkit

# alpine: smallest base that still ships a shell (docker-entrypoint.sh) and a way to fetch
# CA certs (outbound LLM calls) and a healthcheck client (busybox wget, already in the base
# image) — a distroless/scratch base has none of the three.
FROM alpine:3.21
RUN apk add --no-cache ca-certificates
COPY --from=build /out/deskkit /usr/local/bin/deskkit
COPY docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
RUN chmod +x /usr/local/bin/docker-entrypoint.sh

# The desk (files) and the PocketBase store both live under /data so one volume mount (docker
# run -v, or a platform volume) keeps both across a redeploy. No VOLUME directive: hosting
# platforms that manage volumes themselves reject it, and every runner here mounts explicitly.
# Baked-in identity-neutral defaults; override with docker run -e DESK_ROOT=... -e DESK_NAME=...
ENV DESK_ROOT=/data/desk DESK_NAME=desk
EXPOSE 8090

# Runs as root: the platform mounts a fresh named volume owned by root with no init system
# to chown-then-drop-privileges, and adding one (gosu/su-exec) is more than this slice needs.
# Stated plainly per the brief rather than shipping a non-root user that half-works.

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
  CMD wget -q -O /dev/null "http://127.0.0.1:${PORT:-8090}/api/health" || exit 1

ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]

# Multi-stage build for the deskkit binary. See docs/pattern.md.
# Architecture-neutral: no hardcoded GOARCH/base-arch tag, so this builds on both
# arm64 (local Apple Silicon Docker) and amd64 (typical hosting target).

# SPA (web/): built first and copied into the Go build stage below so go:embed picks it up.
FROM node:22-alpine AS webbuild
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN mkdir -p /src/internal/core/spa/dist && npm run build

FROM golang:1.25-alpine AS build
WORKDIR /src
COPY VERSION VERSION
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ cmd/
COPY internal/ internal/
COPY templates/ templates/
COPY --from=webbuild /src/internal/core/spa/dist internal/core/spa/dist
RUN CGO_ENABLED=0 go build \
      -ldflags "-X main.version=$(cat VERSION)" \
      -o /out/deskkit ./cmd/deskkit

# alpine: smallest base that still ships a shell (docker-entrypoint.sh) and a way to fetch
# CA certs (outbound LLM calls) and a healthcheck client (busybox wget, already in the base
# image) — a distroless/scratch base has none of the three.
FROM alpine:3.21
# su-exec: a ~10KB setuid-free exec-and-drop helper. It is what lets the entrypoint start as
# root (to hand the volume over) and then become the unprivileged user for good, in-process,
# with no init system and no extra PID.
RUN apk add --no-cache ca-certificates su-exec \
 && addgroup -g 10001 -S deskkit \
 && adduser -u 10001 -S -G deskkit -h /home/deskkit deskkit
COPY --from=build /out/deskkit /usr/local/bin/deskkit
COPY docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
RUN chmod +x /usr/local/bin/docker-entrypoint.sh

# The desk (files) and the PocketBase store both live under /data so one volume mount (docker
# run -v, or a platform volume) keeps both across a redeploy. No VOLUME directive: hosting
# platforms that manage volumes themselves reject it, and every runner here mounts explicitly.
# Baked-in identity-neutral defaults; override with docker run -e DESK_ROOT=... -e DESK_NAME=...
# HOME is set explicitly because the process is not started by a login shell: without it the
# dropped-privilege process inherits root's HOME and the config resolver looks for its central
# config under a directory it cannot read.
ENV DESK_ROOT=/data/desk DESK_NAME=desk HOME=/home/deskkit
EXPOSE 8090

# No USER directive, deliberately. The container starts as root for exactly one reason: the
# mounted volume arrives owned by root (a fresh platform volume, or a tree left behind by an
# older root-running image), and only root can hand it to an unprivileged user. The entrypoint
# does that chown and then `exec su-exec deskkit` — replacing itself, so nothing after that
# point holds root and there is no root process left alive to re-enter. A bare `USER deskkit`
# here would skip the chown and fail to boot on any volume it did not create.
# The drop is asserted, not assumed: scripts/docker-smoke.sh checks the uid of PID 1.
# Port note: the dropped user CAN still bind below 1024 under Docker's defaults, which set
# net.ipv4.ip_unprivileged_port_start=0 (measured in this image: PORT=80 serves as uid 10001).
# That is the runtime's default, not a property of this image, so do not rely on it on a runtime
# that leaves the kernel's 1024 floor in place.

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
  CMD wget -q -O /dev/null "http://127.0.0.1:${PORT:-8090}/api/health" || exit 1

ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]

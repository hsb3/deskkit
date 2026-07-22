#!/usr/bin/env bash
# shellcheck shell=bash
# 50 — release-shaped checks. Sourced by e2e.sh; helpers (check/note/section) and E2E_* state
# come from lib.sh. Proves the repo's release invariants hold: VERSION stays the single
# source of truth across shipped manifests and the changelog, the release ldflags stamp
# actually reaches the binary, and the marketplace bundle stays self-contained (the
# marketplace install copies only plugin/claude-plugin/, so its committed artifacts must
# already carry everything a fresh install needs).

section "50 · release-shaped checks"

# --- VERSION single-source-of-truth across shipped manifests -------------------------------
( cd "$E2E_REPO" && node scripts/check-version-sync.mjs ) >/dev/null 2>&1
check "VERSION matches all shipped manifests (check-version-sync)" $?

# --- CHANGELOG documents the current VERSION ------------------------------------------------
( cd "$E2E_REPO" && node scripts/check-changelog.mjs ) >/dev/null 2>&1
check "CHANGELOG documents the current VERSION (check-changelog)" $?

# --- version-stamping end-to-end: ldflags carry VERSION into the binary ---------------------
V=$(cat "$E2E_REPO/VERSION")

( cd "$E2E_REPO/librarian" && go build -ldflags "-X main.version=$V" -o "$E2E_WORK/deskkit-stamped" ./cmd/deskkit ) >/dev/null 2>&1
check "version-stamped build succeeds" $?

OUT=$("$E2E_WORK/deskkit-stamped" --version 2>&1)
printf '%s' "$OUT" | grep -qF "$V"
check "stamped binary --version reports the release VERSION ($V)" $?
note "unstamped suite binary reports dev (dk --version -> deskkit version dev); the ldflags stamp is what carries the release version"

# --- marketplace bundle self-containment: only plugin/claude-plugin/ is copied on install ---
SERVER_JS="$E2E_REPO/plugin/claude-plugin/mcp/server.js"
PROFILE_SCHEMA="$E2E_REPO/plugin/claude-plugin/schema/profile.schema.yaml"
REFERENCES_YAML="$E2E_REPO/plugin/claude-plugin/schema/references.yaml"

[ -f "$SERVER_JS" ] \
  && [ "$(wc -c < "$SERVER_JS")" -gt 1000 ] \
  && [ -f "$PROFILE_SCHEMA" ] \
  && [ -f "$REFERENCES_YAML" ]
check "marketplace bundle artifacts present + self-contained (server.js, profile.schema.yaml, references.yaml)" $?

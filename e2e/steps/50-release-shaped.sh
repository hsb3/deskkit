#!/usr/bin/env bash
# shellcheck shell=bash
# 50 — release-shaped checks. Sourced by e2e.sh; helpers (check/note/section) and E2E_* state
# come from lib.sh. Proves the repo's release invariants hold: VERSION stays the single
# source of truth across shipped manifests and the changelog, the release ldflags stamp
# actually reaches the binary, and the marketplace bundle stays self-contained (a marketplace
# install copies only the bundle directory, so whatever it needs must already be inside it —
# which now means the binary, launched by its .mcp.json, and no bundled JS/schema copies).

section "50 · release-shaped checks"

# --- VERSION single-source-of-truth across shipped manifests -------------------------------
( cd "$E2E_REPO" && node scripts/check-version-sync.mjs ) >/dev/null 2>&1
check "VERSION matches all shipped manifests (check-version-sync)" $?

# --- CHANGELOG documents the current VERSION ------------------------------------------------
( cd "$E2E_REPO" && node scripts/check-changelog.mjs ) >/dev/null 2>&1
check "CHANGELOG documents the current VERSION (check-changelog)" $?

# --- version-stamping end-to-end: ldflags carry VERSION into the binary ---------------------
V=$(cat "$E2E_REPO/VERSION")

( cd "$E2E_REPO" && go build -ldflags "-X main.version=$V" -o "$E2E_WORK/deskkit-stamped" ./cmd/deskkit ) >/dev/null 2>&1
check "version-stamped build succeeds" $?

OUT=$("$E2E_WORK/deskkit-stamped" --version 2>&1)
printf '%s' "$OUT" | grep -qF "$V"
check "stamped binary --version reports the release VERSION ($V)" $?
note "unstamped suite binary reports dev (dk --version -> deskkit version dev); the ldflags stamp is what carries the release version"

# --- marketplace bundle: one bundle, self-contained, and free of generated JS/schema copies --
# The tool surface it needs is the deskkit binary, so the bundle carries a .mcp.json launching
# `deskkit mcp-serve` and NOT a bundled server.js or a copied schema — those existed only to
# feed the retired TypeScript stdio server. Assert both halves: what must be there, and what
# must not have come back.
BUNDLE="$E2E_REPO/plugins/deskkit"

BUNDLE_DIRS=$(find "$E2E_REPO/plugins" -mindepth 1 -maxdepth 1 -type d | wc -l | tr -d ' ')
if [ "$BUNDLE_DIRS" = "1" ] && [ -d "$BUNDLE" ]; then RC=0; else RC=1; fi
check "plugins/ holds exactly one shipped bundle (deskkit)" "$RC"

[ -f "$BUNDLE/.mcp.json" ] \
  && grep -q '"deskkit"' "$BUNDLE/.mcp.json" \
  && grep -q '"mcp-serve"' "$BUNDLE/.mcp.json"
check "bundle .mcp.json launches the deskkit binary (deskkit mcp-serve)" $?

# JS in any module flavour (.js/.mjs/.cjs), and a schema copy ANYWHERE in the bundle — matched by
# the canonical file NAMES rather than a schema/ path, so a copy dropped in another directory is
# still caught while the desk-setup template's own profile.example.yaml stays legitimate.
JS_COUNT=$(find "$E2E_REPO/plugins" \( -name '*.js' -o -name '*.mjs' -o -name '*.cjs' \) | wc -l | tr -d ' ')
SCHEMA_COUNT=$(find "$E2E_REPO/plugins" \( -name 'profile.schema.yaml' -o -name 'references.yaml' \) | wc -l | tr -d ' ')
[ "$JS_COUNT" = "0" ] && [ "$SCHEMA_COUNT" = "0" ]
check "bundle is TS-free: no generated server.js and no bundled schema copies under plugins/" $?
note "plugins/ .js files: $JS_COUNT; bundled schema yaml files: $SCHEMA_COUNT"

#!/usr/bin/env bash
# record-media.sh — regenerate docs/assets/*.gif from scripts/vhs-tapes/*.tape (VHS).
#
# The tapes are the source of truth; the GIFs are generated artifacts (never hand-edited —
# re-run this script to regenerate them after any tape edit). This script is fully
# self-contained: it builds the deskkit binary, seeds a handful of throwaway scratch desks (one per
# tape, isolated from each other so no tape's writes can bleed into another's demo), points
# EVERY store-touching invocation at a scratch XDG_DATA_HOME + HOME so nothing can ever touch
# the operator's real ~/.local/share/deskkit, pre-runs whatever setup each tape's demo
# assumes already happened (migrate/sweep/patrol), then drives `vhs` over each tape (sources in
# scripts/vhs-tapes/), writing the GIFs to docs/assets/ (each tape's `Output docs/assets/*.gif`
# path resolves from the repo root). Idempotent: every run starts from a fresh scratch tree and leaves
# no litter behind (trap-cleaned on exit, including on error/interrupt).
set -euo pipefail
#TODO: update to output gifs to docs/assets/
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
BIN="$REPO_ROOT/deskkit"
MEDIA_DIR="$REPO_ROOT/docs/assets"

log() { printf 'record-media: %s\n' "$1"; }

command -v vhs >/dev/null 2>&1 || { echo "record-media: vhs not found on PATH (brew install vhs)" >&2; exit 1; }
command -v ttyd >/dev/null 2>&1 || { echo "record-media: ttyd not found on PATH (brew install ttyd)" >&2; exit 1; }

log "building deskkit..."
make -C "$REPO_ROOT" build

# --- scratch setup ---------------------------------------------------------------------------
# A fixed /tmp base (NOT ${TMPDIR:-/tmp}): the GIFs are committed, baked artifacts, and this
# sandbox's $TMPDIR can embed session-specific path components (repo/session identifiers) that
# would otherwise get typed into a permanent recording. /tmp is stable and identity-neutral on
# both macOS and Linux.
SCRATCH=$(mktemp -d "/tmp/deskkit-media.XXXXXX")
cleanup() {
  chmod -R u+w "$SCRATCH" 2>/dev/null || true
  rm -rf "$SCRATCH"
}
trap cleanup EXIT INT TERM

# Scratch HOME + a default scratch XDG_DATA_HOME, exported for the WHOLE script (every child
# process, including every `vhs`/deskkit invocation below, inherits these) so no
# recording can ever resolve its store to the operator's real ~/.local/share/deskkit.
# Each desk below still gets its OWN XDG_DATA_HOME override (one store per tape).
export HOME="$SCRATCH/home"
mkdir -p "$HOME"
export XDG_DATA_HOME="$SCRATCH/xdg-default"
mkdir -p "$XDG_DATA_HOME"

# deskkit on $PATH as a bare command — both for `Require deskkit` (checked by
# vhs itself before it opens a shell) and so no tape needs to know the repo's on-disk layout.
export PATH="$REPO_ROOT:$PATH"

seed_decision() { # seed_decision <path>
  mkdir -p "$(dirname "$1")"
  cat > "$1" <<'EOF'
---
type: decision
status: accepted
created: 2026-07-01
updated: 2026-07-01
tags: []
synopsis: "example decision for the demo desk"
---
We standardize on the example workflow for this desk.
EOF
}

seed_r1_task() { # seed_r1_task <path> — missing universal frontmatter keys (R1)
  mkdir -p "$(dirname "$1")"
  cat > "$1" <<'EOF'
---
type: task
---
Ping the vendor about the renewal quote.
EOF
}

seed_r1r2_journal() { # seed_r1r2_journal <path> — no frontmatter (R1) + bad filename (R2)
  mkdir -p "$(dirname "$1")"
  cat > "$1" <<'EOF'
Quick log entry, not following the naming convention yet.
EOF
}

seed_stray_note() { # seed_stray_note <path> — no frontmatter, not under any entity dir (orphan)
  mkdir -p "$(dirname "$1")"
  cat > "$1" <<'EOF'
Scratch note, not filed anywhere formal yet.
EOF
}

run_lib() { # run_lib <desk_root> <xdg_home> <desk_name> <args...>
  local desk="$1" xdg="$2" name="$3"
  shift 3
  DESK_ROOT="$desk" XDG_DATA_HOME="$xdg" DESK_NAME="$name" "$BIN" "$@"
}

# --- desk A: sweep-and-findings.tape --------------------------------------------------------
# Only `migrate up` is pre-run (hidden setup noise); sweep itself runs LIVE in the recording so
# it reports real "created" counts. A hidden `patrol` runs between the live sweep and the live
# `query findings` so findings has real content to show (findings/orphans are populated by
# patrol, not sweep — see internal/tools/patrol.go) without pulling `patrol` into the command
# sequence this tape is actually meant to teach.
DESK_SWEEP="$SCRATCH/desks/sweep-and-findings"
XDG_SWEEP="$SCRATCH/xdg/sweep-and-findings"
mkdir -p "$DESK_SWEEP" "$XDG_SWEEP"
seed_stray_note "$DESK_SWEEP/NOTES.md"
seed_decision "$DESK_SWEEP/_structure/decisions/0001-example-decision.md"
seed_r1_task "$DESK_SWEEP/tasks/status-update.md"
run_lib "$DESK_SWEEP" "$XDG_SWEEP" example-desk migrate up > /dev/null

# --- desk B: patrol.tape --------------------------------------------------------------------
# Pre-swept (hidden): patrol reads already-indexed `files` rows, so the desk must have been
# swept at least once before the live `patrol` command has anything to check.
DESK_PATROL="$SCRATCH/desks/patrol"
XDG_PATROL="$SCRATCH/xdg/patrol"
mkdir -p "$DESK_PATROL" "$XDG_PATROL"
seed_decision "$DESK_PATROL/_structure/decisions/0001-example-decision.md"
seed_r1_task "$DESK_PATROL/tasks/status-update.md"
seed_r1r2_journal "$DESK_PATROL/journal/notes.md"
run_lib "$DESK_PATROL" "$XDG_PATROL" example-desk migrate up > /dev/null
run_lib "$DESK_PATROL" "$XDG_PATROL" example-desk sweep > /dev/null

# --- desk C: propose-apply-restore.tape ---------------------------------------------------
# Pre-swept + pre-patrolled (hidden) so the recording can start straight at propose-fix, per
# the brief's command sequence. Deliberately a SINGLE open finding (no other seeded violations
# in this desk) so the before/after cat stays a focused one-file demo.
DESK_APPLY="$SCRATCH/desks/propose-apply-restore"
XDG_APPLY="$SCRATCH/xdg/propose-apply-restore"
mkdir -p "$DESK_APPLY" "$XDG_APPLY"
seed_r1_task "$DESK_APPLY/tasks/status-update.md"
run_lib "$DESK_APPLY" "$XDG_APPLY" example-desk migrate up > /dev/null
run_lib "$DESK_APPLY" "$XDG_APPLY" example-desk sweep > /dev/null
run_lib "$DESK_APPLY" "$XDG_APPLY" example-desk patrol > /dev/null

# --- desk D: open-guard.tape ----------------------------------------------------------------
# Pre-swept (hidden) under desk name "example-desk" so the store has a `desk`-carrying row for
# the guard to compare against (ADR 0002 §3 — an empty store has nothing to compare and always
# passes). The live recording reopens the SAME store dir with a different DESK_NAME.
DESK_GUARD="$SCRATCH/desks/open-guard"
XDG_GUARD="$SCRATCH/xdg/open-guard"
mkdir -p "$DESK_GUARD" "$XDG_GUARD"
seed_decision "$DESK_GUARD/_structure/decisions/0001-example-decision.md"
run_lib "$DESK_GUARD" "$XDG_GUARD" example-desk migrate up > /dev/null
run_lib "$DESK_GUARD" "$XDG_GUARD" example-desk sweep > /dev/null
STORE_ONE="$XDG_GUARD/deskkit/example-desk"

# --- desk E: chat.tape -----------------------------------------------------------------------
# The only demo whose live step (the model's own `sweep` tool call, mid-conversation) needs a
# working LLM provider key — unlike desks A-D, this one is GATED (below, in the record section)
# on ANTHROPIC_API_KEY so the default hermetic `make media` run never depends on network access
# or a secret. Desk/store setup here is unconditional and cheap (no network), so the gate only
# has to wrap the actual `vhs` invocation. Seeded with a small, mixed desk (a decision, an R1
# task, a stray note) so the live sweep + the model's own summary have real content to describe.
DESK_CHAT="$SCRATCH/desks/chat"
XDG_CHAT="$SCRATCH/xdg/chat"
mkdir -p "$DESK_CHAT" "$XDG_CHAT"
seed_decision "$DESK_CHAT/_structure/decisions/0001-example-decision.md"
seed_r1_task "$DESK_CHAT/tasks/status-update.md"
seed_stray_note "$DESK_CHAT/NOTES.md"
run_lib "$DESK_CHAT" "$XDG_CHAT" example-desk migrate up > /dev/null

# --- desk F: chat-light.tape -----------------------------------------------------------------
# Same recipe as desk E, but its own desk/store so the light-theme recording's live sweep can't
# collide with chat.tape's. Gated on ANTHROPIC_API_KEY alongside chat.tape (below): this is the
# light-terminal readability proof for the chat TUI's per-theme palette.
DESK_CHAT_LIGHT="$SCRATCH/desks/chat-light"
XDG_CHAT_LIGHT="$SCRATCH/xdg/chat-light"
mkdir -p "$DESK_CHAT_LIGHT" "$XDG_CHAT_LIGHT"
seed_decision "$DESK_CHAT_LIGHT/_structure/decisions/0001-example-decision.md"
seed_r1_task "$DESK_CHAT_LIGHT/tasks/status-update.md"
seed_stray_note "$DESK_CHAT_LIGHT/NOTES.md"
run_lib "$DESK_CHAT_LIGHT" "$XDG_CHAT_LIGHT" example-desk migrate up > /dev/null

# --- desk G: init-onramp.tape ---------------------------------------------------------------
# Deliberately BARE: no profile, no env, no migrate — the recording's whole point is that the
# first-run onramp scaffolds the profile itself and the store self-initializes afterward. The
# desk name in the GIF is the folder basename, so the dir name below is part of the recording.
DESK_INIT="$SCRATCH/desks/demo-desk"
XDG_INIT="$SCRATCH/xdg/init-onramp"
mkdir -p "$DESK_INIT" "$XDG_INIT"

# --- record ------------------------------------------------------------------------------
# Each tape's Hidden preamble reads its scratch desk root / XDG home from the env vars set here
# (MEDIA_DESK_ROOT / MEDIA_XDG_HOME / MEDIA_STORE_ONE); only the DESK_ROOT/DESK_NAME exports the
# tape types on screen are meant to be read, so the tape sources contain no baked-in scratch
# path — this script is what supplies the actual (session-local, /tmp-based) values.
cd "$REPO_ROOT"

record_tape() { # record_tape <tape-file> <desk_root> <xdg_home> [store_one]
  local tape="$1" desk="$2" xdg="$3" store_one="${4:-}"
  log "recording $tape"
  MEDIA_DESK_ROOT="$desk" MEDIA_XDG_HOME="$xdg" MEDIA_STORE_ONE="$store_one" \
    vhs "scripts/vhs-tapes/$tape"
}

record_tape sweep-and-findings.tape "$DESK_SWEEP" "$XDG_SWEEP"
record_tape patrol.tape "$DESK_PATROL" "$XDG_PATROL"
record_tape propose-apply-restore.tape "$DESK_APPLY" "$XDG_APPLY"
record_tape open-guard.tape "$DESK_GUARD" "$XDG_GUARD" "$STORE_ONE"
record_tape init-onramp.tape "$DESK_INIT" "$XDG_INIT"

# The chat tapes need a real LLM provider key (their live `sweep` tool call talks to
# Anthropic) — skip them, with a clear log line, when the operator hasn't exported one. Guarded
# with `${ANTHROPIC_API_KEY:-}` so this check itself is safe under `set -u` when the var is
# unset entirely; skipping (rather than failing) keeps the default `make media` run hermetic.
if [ -n "${ANTHROPIC_API_KEY:-}" ]; then
  record_tape chat.tape "$DESK_CHAT" "$XDG_CHAT"
  record_tape chat-light.tape "$DESK_CHAT_LIGHT" "$XDG_CHAT_LIGHT"
else
  log "skipping chat.tape + chat-light.tape — ANTHROPIC_API_KEY not set in the environment (export it and re-run to record these)"
fi

log "done — GIFs written under $MEDIA_DIR"

#!/usr/bin/env bash
#
# install.sh — one-shot installer for desk-standard's two deliverables.
#
#   curl -fsSL https://raw.githubusercontent.com/hsb3/desk-standard/main/install.sh | bash
#
# It consumes the artifacts published by the release workflow
# (.github/workflows/release.yml) on every `v*` tag:
#
#   - pocket-librarian_<version>_<os>_<arch>   (the librarian Go binary; release.yml line 125)
#   - desk-standard-plugin_<version>.tar.gz    (the plugin bundle;        release.yml line 154)
#   - checksums.txt                            (sha256sum of every asset; release.yml line 182)
#
# This script (1) downloads the librarian binary matching the host OS/arch, (2) verifies its
# sha256 against checksums.txt and FAILS LOUDLY on mismatch, (3) installs it to a no-root path,
# and (4) guides the Claude Code plugin install (marketplace add + install).
#
# NOTE (live e2e): no v* release is published yet, so the download/verify/install path cannot be
# exercised end-to-end here. It is shellcheck-clean and the --dry-run plan is verified against the
# exact artifact strings in release.yml. Real end-to-end verification is a post-first-release
# foreman step (see the handoff note).
#
set -euo pipefail

# ---- constants (kept in lock-step with .github/workflows/release.yml) -----------------------
REPO="hsb3/desk-standard"          # gh release lives here; also the plugin marketplace slug
BINARY_NAME="pocket-librarian"     # installed command name (release.yml line 126: -o pocket-librarian)
CHECKSUMS_FILE="checksums.txt"     # release.yml line 182
PLUGIN_ID="desk-standard@desk-standard"  # `claude plugin install desk-standard@desk-standard`

# ---- defaults (overridable by flag or env) --------------------------------------------------
VERSION="${LIBRARIAN_VERSION:-latest}"   # a tag like v0.4.0, a bare 0.4.0, or "latest"
PREFIX="${PREFIX:-${HOME}/.local/bin}"
DRY_RUN=false
WITH_PLUGIN=false                        # --with-plugin runs the claude commands (opt-in)

# Test/override hooks: force a target that differs from the host (used by --dry-run transcripts
# and CI). Leave unset in normal use — the host is auto-detected.
OS_OVERRIDE="${INSTALL_OS:-}"
ARCH_OVERRIDE="${INSTALL_ARCH:-}"

# ---- output helpers -------------------------------------------------------------------------
info()  { printf '%s\n' "$*"; }
step()  { printf '\n==> %s\n' "$*"; }
warn()  { printf 'warning: %s\n' "$*" >&2; }
die()   { printf 'error: %s\n' "$*" >&2; exit 1; }

usage() {
  cat <<EOF
install.sh — install the desk-standard librarian binary and guide the plugin install.

Usage:
  install.sh [options]

Options:
  --version <v>     Release to install (e.g. v0.4.0 or 0.4.0). Default: latest.
                    Env: LIBRARIAN_VERSION
  --prefix <dir>    Directory to install the binary into (no root needed).
                    Default: \$HOME/.local/bin.  Env: PREFIX
  --with-plugin     Run the Claude Code plugin install commands (requires \`claude\` on PATH).
                    Default: print the commands for you to run.
  --dry-run         Print every planned action, URL, and artifact name; download/write nothing.
  -h, --help        Show this help.

Environment overrides (mainly for testing):
  INSTALL_OS    Force os target: darwin | linux   (default: auto-detect from uname -s)
  INSTALL_ARCH  Force arch target: amd64 | arm64  (default: auto-detect from uname -m)

Examples:
  curl -fsSL https://raw.githubusercontent.com/${REPO}/main/install.sh | bash
  ./install.sh --version v0.4.0 --prefix ~/bin
  INSTALL_OS=linux INSTALL_ARCH=amd64 ./install.sh --version v0.4.0 --dry-run
EOF
}

# ---- arg parsing ----------------------------------------------------------------------------
while [ "$#" -gt 0 ]; do
  case "$1" in
    --version)     [ "$#" -ge 2 ] || die "--version needs an argument"; VERSION="$2"; shift 2 ;;
    --version=*)   VERSION="${1#*=}"; shift ;;
    --prefix)      [ "$#" -ge 2 ] || die "--prefix needs an argument"; PREFIX="$2"; shift 2 ;;
    --prefix=*)    PREFIX="${1#*=}"; shift ;;
    --with-plugin) WITH_PLUGIN=true; shift ;;
    --dry-run)     DRY_RUN=true; shift ;;
    -h|--help)     usage; exit 0 ;;
    *)             die "unknown option: $1 (try --help)" ;;
  esac
done

# ---- prerequisite tooling -------------------------------------------------------------------
have() { command -v "$1" >/dev/null 2>&1; }

have curl || die "curl is required but not found on PATH."

# sha256 tool differs by platform: coreutils sha256sum (linux) vs BSD shasum (macOS).
SHA_TOOL=""
if have sha256sum; then
  SHA_TOOL="sha256sum"
elif have shasum; then
  SHA_TOOL="shasum -a 256"
else
  die "need sha256sum or shasum to verify the download; neither found."
fi

# ---- detect / resolve target OS + arch ------------------------------------------------------
detect_os() {
  if [ -n "$OS_OVERRIDE" ]; then printf '%s' "$OS_OVERRIDE"; return; fi
  case "$(uname -s)" in
    Darwin) printf 'darwin' ;;
    Linux)  printf 'linux'  ;;
    *)      die "unsupported OS '$(uname -s)': release binaries exist only for darwin and linux." ;;
  esac
}

detect_arch() {
  if [ -n "$ARCH_OVERRIDE" ]; then printf '%s' "$ARCH_OVERRIDE"; return; fi
  case "$(uname -m)" in
    x86_64|amd64)  printf 'amd64' ;;
    arm64|aarch64) printf 'arm64' ;;
    *)             die "unsupported arch '$(uname -m)': release binaries exist only for amd64 and arm64." ;;
  esac
}

OS="$(detect_os)"
ARCH="$(detect_arch)"
case "$OS" in darwin|linux) ;; *) die "invalid INSTALL_OS='$OS' (expected darwin or linux)." ;; esac
case "$ARCH" in amd64|arm64) ;; *) die "invalid INSTALL_ARCH='$ARCH' (expected amd64 or arm64)." ;; esac

# ---- resolve the release version ------------------------------------------------------------
# The workflow tags releases `v<version>` (release.yml lines 32/195) but names the binary
# artifact with the BARE version (release.yml line 125: pocket-librarian_${version}_...).
# So we track both: TAG (with the leading v, used in download URLs) and VERSION_BARE.
resolve_version() {
  if [ "$VERSION" = "latest" ]; then
    info "Resolving latest release of ${REPO} via the GitHub API..."
    local api tag
    api="https://api.github.com/repos/${REPO}/releases/latest"
    # Parse tag_name without requiring jq.
    tag="$(curl -fsSL "$api" | grep -m1 '"tag_name"' | sed -E 's/.*"tag_name"[[:space:]]*:[[:space:]]*"([^"]+)".*/\1/' || true)"
    [ -n "$tag" ] || die "could not resolve the latest release tag from ${api} (no release published yet?). Pin one with --version."
    printf '%s' "$tag"
  else
    # Accept either "v0.4.0" or "0.4.0" from the user; normalise to a v-prefixed tag.
    case "$VERSION" in
      v*) printf '%s' "$VERSION" ;;
      *)  printf 'v%s' "$VERSION" ;;
    esac
  fi
}

TAG="$(resolve_version)"
VERSION_BARE="${TAG#v}"

# ---- construct artifact names + URLs (must match release.yml exactly) ------------------------
# release.yml line 125: out="pocket-librarian_${version}_${goos}_${goarch}"
ARTIFACT="${BINARY_NAME}_${VERSION_BARE}_${OS}_${ARCH}"
BASE_URL="https://github.com/${REPO}/releases/download/${TAG}"
BINARY_URL="${BASE_URL}/${ARTIFACT}"
CHECKSUMS_URL="${BASE_URL}/${CHECKSUMS_FILE}"
DEST="${PREFIX}/${BINARY_NAME}"

# ---- plan summary ---------------------------------------------------------------------------
step "Plan"
info "  repo:        ${REPO}"
info "  release tag: ${TAG}  (bare version: ${VERSION_BARE})"
info "  target:      ${OS}/${ARCH}"
info "  artifact:    ${ARTIFACT}"
info "  binary URL:  ${BINARY_URL}"
info "  checksums:   ${CHECKSUMS_URL}"
info "  install to:  ${DEST}"
info "  sha tool:    ${SHA_TOOL}"
$DRY_RUN && info "  mode:        DRY RUN (no downloads, no writes)"

# ---- download + verify + install ------------------------------------------------------------
install_binary() {
  step "Download librarian binary"
  if $DRY_RUN; then
    info "  [dry-run] curl -fL -o <tmp>/${ARTIFACT} ${BINARY_URL}"
    info "  [dry-run] curl -fL -o <tmp>/${CHECKSUMS_FILE} ${CHECKSUMS_URL}"
    info "  [dry-run] verify sha256 of ${ARTIFACT} against ${CHECKSUMS_FILE} (fail loudly on mismatch)"
    info "  [dry-run] mkdir -p ${PREFIX}"
    info "  [dry-run] install ${ARTIFACT} -> ${DEST} (chmod +x)"
    return
  fi

  local tmp
  tmp="$(mktemp -d)"
  # shellcheck disable=SC2064  # expand tmp now so the trap removes this specific dir.
  trap "rm -rf '${tmp}'" EXIT

  info "  downloading ${ARTIFACT}"
  curl -fL --retry 3 -o "${tmp}/${ARTIFACT}" "${BINARY_URL}" \
    || die "download failed: ${BINARY_URL}"
  info "  downloading ${CHECKSUMS_FILE}"
  curl -fL --retry 3 -o "${tmp}/${CHECKSUMS_FILE}" "${CHECKSUMS_URL}" \
    || die "download failed: ${CHECKSUMS_URL}"

  step "Verify sha256"
  # checksums.txt is produced by `sha256sum ./*` run inside dist/ (release.yml line 182),
  # so each line is "<hash>  ./<artifact>". Match on the basename, tolerating the ./ prefix.
  local expected actual
  expected="$(awk -v f="${ARTIFACT}" '{ n=$2; sub(/^\.\//,"",n); if (n==f) { print $1; exit } }' "${tmp}/${CHECKSUMS_FILE}")"
  [ -n "$expected" ] || die "no checksum entry for ${ARTIFACT} in ${CHECKSUMS_FILE}."
  actual="$(${SHA_TOOL} "${tmp}/${ARTIFACT}" | awk '{print $1}')"
  if [ "$expected" != "$actual" ]; then
    die "sha256 MISMATCH for ${ARTIFACT}
       expected: ${expected}
       actual:   ${actual}
     Refusing to install a binary that does not match the published checksum."
  fi
  info "  ok: ${actual}"

  step "Install"
  mkdir -p "${PREFIX}"
  # install(1) sets mode + copies atomically on both macOS and linux; idempotent on re-run.
  install -m 0755 "${tmp}/${ARTIFACT}" "${DEST}"
  info "  installed ${DEST}"

  case ":${PATH}:" in
    *":${PREFIX}:"*) : ;;
    *) warn "${PREFIX} is not on your PATH. Add it, e.g.:
       echo 'export PATH=\"${PREFIX}:\$PATH\"' >> ~/.zshrc && exec \$SHELL" ;;
  esac
}

verify_install() {
  $DRY_RUN && { info "  [dry-run] ${DEST} --version   (expect: ${BINARY_NAME} version ${VERSION_BARE})"; return; }
  step "Check"
  if "${DEST}" --version 2>/dev/null; then :; else
    warn "'${DEST} --version' did not run cleanly; the file is installed but may need a PATH/permissions fix."
  fi
}

# ---- plugin install guidance / execution ----------------------------------------------------
plugin_step() {
  step "Claude Code plugin"
  info "  The plugin installs from this repo's own marketplace inside Claude Code:"
  info "      claude plugin marketplace add ${REPO}"
  info "      claude plugin install ${PLUGIN_ID}"
  info "  Then personalize your desk (never edit shipped files):"
  info "      cp _knowledge/profile.example.yaml _knowledge/profile.yaml"

  if ! $WITH_PLUGIN; then
    info "  (Re-run with --with-plugin to have this script run the two claude commands for you.)"
    return
  fi

  if $DRY_RUN; then
    info "  [dry-run] claude plugin marketplace add ${REPO}"
    info "  [dry-run] claude plugin install ${PLUGIN_ID}"
    return
  fi

  if ! have claude; then
    warn "--with-plugin was requested but 'claude' is not on PATH; run the two commands above manually."
    return
  fi
  info "  running: claude plugin marketplace add ${REPO}"
  claude plugin marketplace add "${REPO}" || warn "marketplace add reported a non-zero exit (already added?)."
  info "  running: claude plugin install ${PLUGIN_ID}"
  claude plugin install "${PLUGIN_ID}" || warn "plugin install reported a non-zero exit (already installed?)."
}

# ---- run ------------------------------------------------------------------------------------
install_binary
verify_install
plugin_step

step "Done"
if $DRY_RUN; then
  info "Dry run complete — nothing was downloaded or written."
else
  info "librarian installed at ${DEST}. Next: fill _knowledge/profile.yaml and see docs/getting-started.md."
fi

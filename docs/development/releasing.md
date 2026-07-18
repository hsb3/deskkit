_How to version and cut a release of desk-standard (the plugin + the librarian, one shared
version). The why is [ADR 0005](../decisions/0005-versioning-and-changelog.md)._
Status: active

# Releasing

Both products ship under **one** version — the root `VERSION` file. A release is a single git tag
`v<VERSION>`; pushing it triggers `.github/workflows/release.yml`, which re-runs the gates,
cross-compiles the librarian for all four platforms, rebuilds the plugin bundle, and publishes a
GitHub release with `checksums.txt`.

## Versioning rule

[Semantic Versioning](https://semver.org). A change to **either** product drives the shared bump:

- **MAJOR** — a breaking change to either product's contract (schema, tool surface, CLI, plugin API).
- **MINOR** — a backward-compatible feature (e.g. the chat TUI in 0.5.0).
- **PATCH** — a backward-compatible fix.

## Cutting a release

1. **Bump `VERSION`** and the three shipped manifests so they agree:
   - `VERSION`
   - `plugin/claude-plugin/.claude-plugin/plugin.json`
   - `plugin/package.json`
   - `.claude-plugin/marketplace.json`

   `node scripts/check-version-sync.mjs` fails until all four match.

2. **Update `CHANGELOG.md`.** Move the accumulated `[Unreleased]` entries into a new dated
   `## [<version>] — YYYY-MM-DD` section (Keep a Changelog: `Added` / `Changed` / `Fixed` / …).
   `node scripts/check-changelog.mjs` fails until the current `VERSION` has a non-empty section —
   this is the hard gate that keeps a release from shipping undocumented.

3. **Run the pre-tag gate** from a clean `main`:

   ```bash
   make release-prep
   ```

   It asserts a clean tree on `main`, runs version-sync + changelog + the advisory, then
   `make check` and `make test`, and prints the exact tag/push command. (It does **not** tag —
   that's a deliberate human go/no-go.)

4. **Tag and push:**

   ```bash
   git tag v<version> && git push --tags
   ```

   The release workflow asserts the tag equals `VERSION`, re-runs the CI gates (including
   `check-changelog`), and publishes the binaries + plugin bundle + `checksums.txt`.

5. **Verify the published assets** (as done for v0.4.0): download the binary for your platform,
   run `--version`, and confirm its sha256 matches the published `checksums.txt`.

## The missing-bump guard

You don't have to remember to bump. Two mechanisms flag it (see [ADR 0005](../decisions/0005-versioning-and-changelog.md)):

- **`make version-status`** (and a non-blocking CI step) warns when `plugin/` or `librarian/`
  changed since the last `v*` tag but `VERSION` hasn't moved — unreleased work piling up. Advisory
  only; it never fails a build.
- **`check-changelog.mjs`** at release time is the hard stop: no documented section for the tagged
  version ⇒ the release gate fails.

## Notes

- A bare `go build` librarian binary reports `dev`; only a `make`-built or released binary is
  version-stamped (ldflags `-X main.version`).
- Pre-`0.4.0` history (including `v0.0.1-alpha`) lives in the git log; the changelog starts at the
  first tagged release, `0.4.0`.

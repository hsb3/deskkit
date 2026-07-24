_ADR for how this repo versions its two products, records changes, and keeps a version bump from
silently going missing between a feature merge and a release._
Status: Accepted — 2026-07-18

# 0005 — Versioning policy, CHANGELOG, and a missing-bump guard

## Context

The repo ships two products under **one** version — the root `VERSION` file drives the librarian
binary (ldflags stamp) and the three shipped manifests (`plugin.json`, `plugin/package.json`,
`marketplace.json`), with `check-version-sync.mjs` failing CI/pre-commit on drift among the four.
A release is one `git tag v<VERSION>`; the release workflow asserts the tag equals `VERSION`.

That machinery guarantees the four version strings *agree*, but nothing guaranteed the version
was *bumped at all*. The full chat-TUI build (PRs #37–#40 — streaming, full-screen UI, resume;
[ADR 0004](0004-chat-full-screen-tui.md)) merged to `main` while `VERSION` stayed at `0.4.0`. The
handoff caught it by hand ("bump 0.4.0 → 0.5.0 at next release"), but there was no changelog to
record what shipped and no automated flag that user-facing work had accumulated past the last tag.

## Decision

**1. Semantic Versioning of the single repo version.** MAJOR for breaking changes to either
product's contract, MINOR for backward-compatible features, PATCH for fixes. Both products move
together on the one `VERSION`; a change to either can drive the bump.

**2. A `CHANGELOG.md` in [Keep a Changelog](https://keepachangelog.com) form** at the repo root,
covering both products. User-facing changes accrue under `[Unreleased]` as they land; at release
time they roll into a dated `## [<version>]` section. It complements the GitHub release notes
(`--generate-notes`, per-PR) with a curated, human-readable summary.

**3. A two-part missing-bump guard, chosen deliberately over a per-PR hard gate:**

- **Release gate (hard).** `scripts/check-changelog.mjs` asserts `CHANGELOG.md` has a non-empty
  section for the current `VERSION`. Wired into `make release-prep` and the `release.yml` gate —
  so a tag cannot be cut for a version that isn't documented. Bumping `VERSION` therefore *forces*
  a changelog section, and writing that section forces you to have accrued the entries.
- **Drift advisory (non-blocking).** `scripts/check-version-status.mjs` compares `VERSION` to the
  latest `v*` tag; when they're equal but `plugin/` or `librarian/` changed since that tag, it
  warns that unreleased work is piling up. It runs as a non-required CI step and `make
  version-status`, and **always exits 0** — a continuous nudge, never a blocked merge.

Per-PR changelog enforcement (fail any PR touching product code that doesn't add an `[Unreleased]`
entry) was rejected: it taxes pure refactor/test/docs PRs and pushes noise into the changelog. The
release gate catches the only moment that actually matters — cutting a release — and the advisory
covers the gap in between.

## Consequences

- A release is now: bump `VERSION` + the three manifests → move `[Unreleased]` entries into a
  dated `## [<version>]` section → `make release-prep` (now also runs `check-changelog` and prints
  the advisory) → `git tag v<version> && git push --tags`. Written up in
  [`docs/development/README.md`](../development/README.md).
- `ci.yml` checks out with `fetch-depth: 0` so the advisory can see tags and diff since the last
  release. The advisory degrades to a soft note (still exit 0) if history/tags are unavailable.
- `CHANGELOG.md` and both guard scripts live at the repo root / `scripts/`, outside the neutrality
  lint surface (`plugin/` + `librarian/`), so they carry no shipped-identity constraints.

## Alternatives considered

- **Per-PR hard gate** — rejected (friction on non-feature PRs; changelog noise), as above.
- **Advisory only, no release gate** — rejected: it never *stops* an undocumented release, which
  is the failure this ADR exists to prevent.
- **Auto-derive the changelog from commit messages** — deferred. GitHub's `--generate-notes`
  already gives the per-PR list; the curated `CHANGELOG.md` is intentionally human-written.

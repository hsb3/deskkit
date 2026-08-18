_The docs layout contract: what lives where, which paths are load-bearing, and how the working desk
differs from published documentation. Index: [`../README.md`](../README.md)._
Status: active

# Docs layout contract

This repo keeps two kinds of prose deliberately separate, and one automated gate keeps the boundary
honest. The distinction is the thing to internalize:

- **Published + shipped surface** — documentation and code meant to be read and navigated: `docs/`,
  the root digests (`README.md`, `CLAUDE.md`, and its `AGENTS.md` symlink), and the code tree
  (`cmd/`, `internal/`, `templates/`, `schema/`, `plugins/`, `scripts/`, `kits/`, `web/`). Every
  doc/media path cited here MUST resolve to a file that exists. This is enforced.
- **Working desk** — point-in-time, hand-curated working state: `.claude/` (agent config + memory).
  References here are provenance, not maintained navigation, so they are NOT gated (see "Why the
  working desk is not gated" below). Handoffs, plans, and research live on the project board, not
  in the tree — the `_meta/` folder that used to hold them was removed in 2026-08 and is in git
  history.

## The `docs/` tree

| Path | Holds | Load-bearing? |
|---|---|---|
| `docs/README.md` | The docs index, split Using vs Developing | no |
| `docs/usage/` | User guides: `getting-started`, `librarian-guide`, `deskkit-reference` (the operator/runtime reference for the binary, as opposed to `librarian-guide`'s daily-loop walkthrough), `pm-guide`, `plugin-guide` | no |
| `docs/development/` | Contributor docs: `CHARTER.md` (canonical direction), `README.md`, `install-and-build.md`, this file | no |
| `docs/development/specs/` | The build specs — see the load-bearing table below | **YES** |
| — | Architecture Decision Records moved to the project board as `DECISION` tasks (2026-08-18); nothing under `docs/` holds them now | cited widely as a bare `ADR NNNN` — never as a path or a board id |
| `docs/assets/` | Rendered demo GIFs (generated from `scripts/vhs-tapes/*.tape`) | no |

## Load-bearing spec paths (read by CI gates or a test — do not move without repointing the reader)

| Spec | Read by |
|---|---|
| `docs/development/specs/pocket-librarian-v1-spec.md` | `scripts/check-query-kinds.mjs` (§5.6 kind list) |
| `docs/development/specs/tool-surface.md` | `internal/core/mcp/tool_surface_doc_test.go` (CLI + gated MCP counts, on `make test`) — ADR 0016 |
| `docs/development/specs/agent-integration-contract-v1-spec.md` | cited by `plugins/deskkit` and the neutrality allowlist (`schema/neutrality-lint.allow`) |
| `docs/development/specs/pm-system-v1-spec.md`, `docs/development/specs/element-model-v2-draft.md` | cited across code, plans, and ADRs; the element-model draft is the live target of the phase-machine reconciliation |

If you move one of these, repoint its reader **in the same change** — the reader hard-codes the path
(a `join(...)` in the `.mjs`, a `filepath.Join(...)` in the Go test), so a stale path fails the gate
loudly (ENOENT), which is the good outcome.

## Specs cite code — they never byte-quote it

A spec names the **path plus the symbol** (`templates/librarian-system-prompt.txt`,
`func systemPrompt`) and lets the reader open the file. It does not reproduce the source in a fenced
block. A second copy in prose rots the moment the first one changes, and the only way to hold it
honest is a byte-identical drift guard per copy — a gate whose whole subject is a duplication the
spec never needed. Illustrative excerpts that are clearly labelled as sketches are fine; a block
claiming to be "kept verbatim" is not.

## The rule

**Moving or deleting a doc means fixing every citation to it in the same change.** On the
published+shipped surface this is not a convention you have to remember — `scripts/check-doc-links.mjs`
(run in `make check` and CI) fails on any dangling `docs/...` reference or broken relative
markdown link, naming the file and the missing target. It exists because of the 2026-07-24 incident:
a bulk `git mv` out of `docs/` left dozens of dangling citations that no gate saw, and the only
symptom was an unrelated gate crashing on ENOENT when it tried to read a moved spec.

A path that legitimately does not exist on disk (an illustrative example, a doc a plan promises to
create later) is allow-listed one per line in `scripts/check-doc-links.allow` as
`<citing-file> -> <cited-path>`. Keep that file short — a growing allow-list means the layout is
drifting.

## Why the working desk is not gated

`.claude/` is excluded from the doc-link gate on purpose. Memory notes are dated snapshots and
agent config cites files at authoring-time paths; those references are provenance, not navigation,
so gating them would either force rewriting history or block CI on stale scratch. Working-desk
hygiene is a **curation discipline**, not a gate:

- Superseded research and plans get archived on the project board (or simply left in git history)
  rather than rotting in the tree next to live material. Since `_meta/` was removed in 2026-08,
  the repo carries no in-tree freezer at all.
- Genuinely-archival design docs (superseded proposals, completed research, old plan folders) do
  not belong in `docs/`. `docs/` is for material a reader is meant to navigate now.
- The split that caused the 2026-07-24 incident — moving still-live build specs into an archive
  folder — is the anti-pattern this contract exists to prevent: if a gate or the shipped binary
  reads a file, it is load-bearing and belongs on the published surface, not in a freezer.

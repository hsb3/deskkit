# `_meta/` — the local working desk

This repo follows the **code-desk** meta-structure convention: a slim set of folders that orient
coding agents and humans on the next release. `_meta/` is tracked by default (ADR-0006 in
`dotfiles-agents`, the estate-wide track-by-default decision — not this repo's local ADR-0006);
the only ignored path is `operations/` content (secrets / live-ops), which stays machine-local
while the empty dir survives a clone.

| Entry | Purpose |
|---|---|
| `_archive/` | Superseded working material — moved, never deleted |
| `briefings/` | Dated readouts (`yyyy-mm-dd-subject/`) |
| `plans/` | The code planning desk — issue bodies and build plans authored by coding agents. `plans/inbox/` holds communication packages received from the strategy desk |
| `operations/` | Live URLs, credentials, runbooks with secrets — never tracked, never in `docs/` |
| `research/` | Live investigations; findings graduate to `docs/` or issues |
| `HANDOFF.md` | Cold-start bridge — tracked, secret-free |
| `mise-en-place.yml` | Per-repo variance manifest (owner/repo/default-branch) |

The executive-desk / strategy layer for this project lives **outside this repo**
(`dev-tooling-desk`); this repo carries only the code-desk convention. Durable, audience-facing
material belongs in `docs/`; fast-moving working notes belong here.

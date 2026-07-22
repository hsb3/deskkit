# _knowledge/ — personalization + background

The single personalization root. Two kinds of content:

- **`profile.yaml` (or `.json`/`.md`) — the structured, substitutable config.** Copy the
  shipped `profile.example.yaml` in this directory to `profile.yaml`, fill in this deployment's
  identifiers and preferences (handles, repos, board, machines), and the system substitutes them
  into prompts and templates in place of hardcoded literals. The real profile is gitignored —
  only `profile.example.yaml` ships — so nothing committed carries a deployment identity. Holds
  identifiers and preferences only — **never credentials** (those stay in `_meta/secrets/` or env,
  referenced by name via `secrets_ref`).
- **Freeform background** — unstructured markdown the agent reads for context and judgment:
  project history, a glossary, who the people and orgs are, "how things are done here" prose
  that does not reduce to a profile key. Not work-state (the board owns that), not secrets.

A fact the system must drop into a prompt verbatim is a profile key; a fact that only informs
judgment is background prose. When a background fact starts being referenced by prompts,
graduate it to a profile key.

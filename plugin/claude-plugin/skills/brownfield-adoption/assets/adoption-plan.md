# Adoption plan — the subject desk

<!--
  Brownfield-adoption disposition table (K24, Phase 3). The adopting agent fills this in as a
  FILE the user reads at the approval gate (Phase 4) — never a wall of prose. One row per
  top-level item enumerated from a literal `ls -A` at the desk root (Phase 2), then annotated.
  Never authored from memory, a handoff, or a scout report.

  Status track: untouched → inventoried → approved → reconciled. No file is written, moved, or
  deleted until the user approves this plan.
-->

## Disposition table

Disposition is one of: **keep-as-is · move · rename · archive · drop**.

| Top-level item | Disposition | Target location under the standard | Notes |
|---|---|---|---|
| `<item from ls -A>` | keep-as-is / move / rename / archive / drop | `<path>` or `—` | `<why>` |

### Mandatory explicit rows

These two rows are always present and never buried in another row:

| Concern | Disposition | Notes |
|---|---|---|
| `.gitignore` semantics flip | move to template stanza | Storage-semantics change: ignore-by-default-with-negations → track-by-default-except-secrets. Its own approved row (K24). |
| Ratified decision record(s) | keep-as-is | Adopted as-is — never retroactively exploded into per-decision files (K24 invariant). |

### Gitignored inventory (from `git status --ignored`)

Every ignored item gets an explicit keep/drop call.

| Ignored item | Class | Keep or drop | Notes |
|---|---|---|---|
| `<path>` | load-bearing / litter | keep / drop | `<why — operational skill dir, local settings, .DS_Store, cache …>` |

## Residual gate questions

Two or three explicit judgment questions for the user at the approval gate. **Always include the
history question.**

1. **Fresh git history vs preserve history** — start the adopted desk from a clean initial commit,
   or migrate the existing `.git/` history intact?
2. `<residual question — e.g. an item whose target location is genuinely ambiguous>`
3. `<residual question — e.g. what to archive vs drop>`

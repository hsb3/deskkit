---
name: desk-setup
description: >-
  Runbook for standing up a NEW executive desk (greenfield) or adopting an EXISTING
  folder as one (brownfield). Use when creating a desk from scratch, or when bringing a
  pre-existing planning folder under the standard. Ships the standard-free scaffold
  skeleton in assets/template/ and drives materializing it from the deployment's profile
  via the template_render MCP tool. For the rules the setup produces, see the
  conventions-standard skill.
---

# desk-setup

Two modes: **greenfield** (build a new desk) and **brownfield** (adopt an existing folder).
Both produce a desk that conforms to the conventions standard (consult the
`conventions-standard` skill for every rule referenced here). This skill owns the *procedure*
and the *scaffold*, not the rules.

The full **brownfield-remediation** capability — mechanically bringing a large, non-conformant
desk all the way up to standard — is **out of scope for v1**; it is a queued design pass. This
skill covers greenfield setup and the read-only, approval-gated brownfield *adoption* states
below.

## The scaffold asset

`assets/template/` is the copyable desk skeleton. It is deliberately **standard-free** (K25):
folders plus placeholder instrument files, with **no** conventions prose baked in. If it is ever
regenerated from an upstream source, re-strip any embedded standard text. Its seed files carry
`{{profile.<dotted.path>}}` placeholders wherever a deployment-specific identifier would appear;
those are resolved from the deployment's profile at setup time (never hand-typed identities into
shipped scaffolding).

Layout the scaffold provides:

```
CLAUDE.md                 # entry: orients only (K8) — placeholders for desk name / role / repos
.gitignore                # the standard gitignore stanza, incl. _meta/secrets/ keep-only
_meta/
  README.md               # the _meta/ taxonomy (working desk)
  HANDOFF.md              # cold-start bridge skeleton
  improvement-log.md      # the friction ledger the harvest-loop skill reads (K21)
  secrets/.gitkeep        # the one secrets home (K29) — keep-only; contents never tracked
_structure/
  decisions/README.md     # the decision spine index (K5, K15), newest-first
_knowledge/
  README.md               # what this folder is + the profile pointer (surfaces i/ii)
```

The scaffold does **not** ship a `profile.example.*` of its own — the canonical placeholder
profile is shipped once with the plugin (under the repo's `_knowledge/`); the scaffold's
`_knowledge/README.md` points at it, so there is one owner for that file (K12).

## Greenfield runbook (K23)

Ordered; each step verifies before the next:

1. **Name and place the desk *outside* the code repos it will oversee.** A desk lives on a cloud
   drive (K28); it is never nested inside a project it watches.
2. **Fix goal and role.** State what the desk oversees and the critical rule (desk drafts; owner
   applies every external write, K1). Milestones, if any, enter later as owner *constraints*
   (K11) — they are not scaffold content.
3. **Copy the scaffold** (`assets/template/`) into the new desk root.
4. **Create the profile.** Copy the plugin's shipped `profile.example.*` to
   `_knowledge/profile.yaml` (or `.json`/`.md`) and fill it in — the deployment's identifiers
   (handles, repos, board, machines, preferences) live here, and nowhere else.
5. **Materialize the seed files.** Run the plugin's **`template_render` MCP tool** to resolve
   every `{{profile.…}}` placeholder in the scaffold against `_knowledge/profile.yaml`. This is
   **fail-loud**: a required placeholder with no default whose key is absent aborts with an
   error — it never silently substitutes an empty string. Optional placeholders carry a
   `{{profile.x || "default"}}` fallback. Do **not** hand-edit identifiers into the seed files;
   let the tool render them.
6. **Fill the entry brief and `_meta/` instruments.** Complete `CLAUDE.md` (orient only, K8),
   `_meta/HANDOFF.md`, `_meta/README.md`.
7. **Write thin app/project-instructions** (K26): the host app's instruction field is a
   **pointer** to the version-visible desk files — substance and fast-changing state never live
   in that field.
8. **Connect the overseen repo read-only** and match the plugins/skills the desk needs.

Result: a flat desk (K9) — heavier structure (`analyses/`, `deliverables/`, `communications/`,
workstream dirs, scope docs) is added only when a live thread needs it.

## Brownfield adoption (K24)

Adopting an existing folder is a **distinct mode** with its own status track and a hard
approval gate. Never silently migrate.

**Status track:** `untouched → inventoried → approved → reconciled`.

1. **`untouched` → `inventoried`: read-only inventory.** Survey the existing folder without
   writing anything. Build the desk's real state from **repo evidence** — git log, the issue
   tracker/board, an existing handoff — **never from stale docs** (a doc may lie; the evidence
   does not).
2. **Produce a proposed disposition table.** For each existing item: keep-as-is / move / rename /
   archive / drop, with the target location under the standard. This is a *proposal*, not an
   action.
3. **`inventoried` → `approved`: the user-approval gate.** No file is written, moved, or deleted
   until the user approves the disposition table. This gate is mandatory.
4. **`approved` → `reconciled`: apply the approved dispositions**, then scaffold any missing
   instruments and materialize placeholders as in greenfield steps 3–7.
5. **Leave the old folder for the user to delete.** Adoption never deletes the source; the user
   removes it once satisfied.

**Two adoption invariants:**

- **Write STATUS from evidence, not stale docs** (step 1) — if an existing status doc disagrees
  with the git/board reality, the reality wins and the doc is corrected.
- **Do not explode a ratified decisions page.** If the existing folder already keeps a
  consolidated, ratified decisions record, adopt it as-is rather than splitting it into per-
  decision files retroactively — respect what was already ruled.

## Verifying the setup

Hand the result to the `conventions-standard` skill's adherence checklist (read-only) to confirm
the new or adopted desk conforms before declaring setup done. Fix findings by re-running the
relevant step, not by hand-patching identities into shipped files.

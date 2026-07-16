---
type: design
status: draft
created: 2026-07-15
updated: 2026-07-15
tags: [desk-standard-plugin, pocket-librarian, identity-neutral, data-surfaces, schema, m-05]
synopsis: "M-05 design: the two personalization data surfaces a generic, identity-neutral system reads so a shipped plugin/binary carries zero hardcoded identifiers — a structured groomable profile (keys the prompts substitute) plus a _knowledge/ freeform background folder, closed by a graduation lint that fails the build on hardcoded name/org/repo/issue tokens whose sanctioned escape is a profile reference."
---

# M-05 — personalization data surfaces

_Work product — does not govern this desk._

Status: draft — design pass, pending sign-off
Date: 2026-07-15
Governs (when adopted): the desk-tooling graduation package (plugin + pocket-librarian)

## Why this exists

`_structure/decisions/0013` **item 9** (accepted 2026-07-15) makes it a **binding build
constraint** that everything the plugin ships — agent definitions, skills, commands,
templates, code, code comments, docs — be **identity-neutral and project-agnostic**: no
person's name, no specific GitHub org/repo/project/issue. The test is that a stranger
installs the plugin and it runs on *their* projects with no find-and-replace through the
definitions. Personalization is supplied through **designated data surfaces the generic
system reads at runtime**, never baked into the definitions — a system-vs-data separation,
like code vs. config.

`_meta/improvement-log.md` **M-05** is the named home for that design and gates item 9. It
asks for two surfaces: **(i)** a configurable structured config/profile (GitHub handles,
preferences, machines, …), populated and groomed by the user and/or an agent at any time,
that prompts **reference** in place of hardcoded literals; and **(ii)** a `_knowledge/`-style
freeform background folder. Both feed **(iii)** a graduation lint that fails the build on
hardcoded name/org/repo/issue tokens.

This document defines all three. It is written product-neutral (both the plugin and the
pocket-librarian binary read these surfaces); librarian-specific consumption is called out
inline. The profile schema is part of **schema v1** — the single shared schema both products
consume (`0013` item 8: the librarian's rule source becomes the plugin's schema) and the seed
of Henry's estate-wide schema (`0013` item 4 framing).

The governing principle, stated once so the three surfaces below stay coherent:

> **Structured = keys the system SUBSTITUTES (mechanical, deterministic). Freeform = background
> the agent READS (interpretive). Secrets = neither — they stay in `_meta/secrets/` / env.**
> A fact the system must drop into a prompt verbatim is a profile key. A fact that only
> informs an agent's judgment is `_knowledge/` prose. A credential is neither.

---

## Surface (i) — the structured config/profile

### Format & location

- **Format (ruled 2026-07-15):** the profile may be authored as **YAML, JSON, or
  Markdown-with-frontmatter**; the loader selects by file extension (`.yaml`/`.yml`, `.json`,
  `.md`). Plain scalars and maps only (no anchors/merge keys, no rich text) so it round-trips
  byte-exact under hand- and agent-edit, and a lint can enumerate every literal value it
  contains. YAML / MD-frontmatter values containing `: ` (colon-space) are quoted — the INS-01
  rule from `_meta/improvement-log.md` applies to this file too.
- **Location (ruled 2026-07-15 — single personalization root):** `_knowledge/profile.{yaml,json,md}`,
  at the root of the consuming repo/desk, co-located with the freeform surface (ii) so there is
  **one** personalization folder, not two scattered homes. Discovered by **walking up from the
  current working directory** until a `_knowledge/profile.*` is found (the same discovery a
  `.env` or `.git` gets) — this keeps the path repo-relative and identity-neutral; nothing in the
  shipped system names an absolute path.
- **Tracking:** the profile is **non-secret deployment data** and may be tracked in the
  consuming repo. It holds identifiers and preferences, never credentials — API keys and
  passwords stay in env / `_meta/secrets/` and are referenced indirectly (below), never copied
  into the profile.
- **Shipped state:** the plugin ships **no** populated profile — only a
  `_knowledge/profile.example.*` (all placeholder values; default `.yaml`) and the schema. The
  consumer's real profile is created on setup (user- or agent-scaffolded), in whichever accepted
  format they prefer.

### Field set (schema v1 profile block)

Every key is optional at the schema level; a **prompt that references a key it needs** is what
makes that key required *for that deployment* (see the missing-key rule under Substitution).
The `custom:` namespace is the open-ended escape hatch — mirrors the `0013` item 6 "always keep
an `other` type with ad-hoc metadata" ruling so agents never hit a wall for lack of a key.

```yaml
schema_version: 1                 # ties the profile to schema v1 (0013 item 8)

identity:
  name: "…"                       # display name; used only where a human name is unavoidable
  github:
    personal: "…"                 # personal handle
    work: "…"                     # work/org handle
  email: "…"                      # optional

repos:
  default: "owner/repo"           # the primary code repo
  by_role:                        # named roles the prompts can address by role, not literal
    product: "owner/repo"
    incubation: "owner/repo"
  shorthand:                      # optional: bare "#N" resolves against this repo
    issue_default: "owner/repo"

board:
  provider: "github-projects"     # generic provider tag
  url: "…"
  number: 0                       # project number

desk:
  name: "…"                       # stamped where a desk/deployment name is needed
  root: "."                       # repo-relative root; "." = the discovered profile's dir
  paths:                          # entity-dir map — the librarian's path constants live here
    decisions: "_structure/decisions"
    tasks: "tasks"
    analyses: "analyses"
    journal: "journal"
    secrets: "_meta/secrets"
    handoff: "_meta/HANDOFF.md"
    knowledge: "_knowledge"

machines:                         # zero or more; matched by hostname at runtime
  - name: "…"                     # hostname
    role: "primary"               # primary | secondary
    projects_root: "…"            # where this machine keeps working repos

models:
  provider: "anthropic"           # anthropic | openai | gemini
  model: "claude-opus-4-8"
  alternates: ["claude-sonnet-5", "gpt-5.4", "gemini-3-pro-preview"]

secrets_ref:                      # NAMES of env vars, never the values
  llm_api_key: "ANTHROPIC_API_KEY"

preferences:
  commit_style: "conventional"    # deployment-wide prefs prompts may reference
  register: "explanatory"

custom: {}                        # open-ended: agents add ad-hoc keys here without a schema bump
```

### The substitution / templating convention

Generic prompt and template text **never** contains a personal literal. Where it would, it
carries a placeholder the loader resolves against the profile at load time:

```
{{profile.<dotted.path>}}                       # required — missing key fails loudly (below)
{{profile.<dotted.path> || "fallback text"}}    # optional — fallback used if key absent
{{env.<VAR>}}                                    # indirection for a secret-adjacent value
```

- **Dotted path** indexes into the YAML tree: `{{profile.identity.github.personal}}`,
  `{{profile.repos.default}}`, `{{profile.desk.paths.decisions}}`.
- **Missing-key rule (fail loud, not silent).** A `{{profile.x}}` with **no** `|| default`
  whose key is absent/empty is a **hard load error** — the system refuses to run rather than
  silently substitute an empty string (a silent empty is how identity leaks or breaks quietly).
  A placeholder that is legitimately optional must carry a `|| "…"` default.
- **`{{env.VAR}}`** exists so a prompt can reference the *name* of a runtime value without the
  value ever touching the profile or the shipped artifact (used for the secrets seam).
- **Why `{{profile.…}}` double-brace with a `profile.`/`env.` prefix:** it is trivially
  distinguishable from an accidental literal, which is exactly what the graduation lint keys on
  — a would-be-flagged token is sanctioned **iff** it is inside a `{{profile.…}}`/`{{env.…}}`
  placeholder (see surface iii).

Illustrative generic template line (ships in the plugin) vs. its resolved form on this desk:

```
# shipped (identity-neutral):
Route new code-repo work to {{profile.repos.default}} and file it on the board at
{{profile.board.url}} (project {{profile.board.number}}).

# resolved at runtime against _knowledge/profile.yaml on Henry's desk:
Route new code-repo work to hsb3/dotfiles-agents and file it on the board at
https://github.com/users/hsb3/projects/11 (project 11).
```

### Grooming (user-edit + agent-edit)

- **User-edit:** hand-edit the YAML at any time. No migration ceremony; the file is read fresh
  on the next load.
- **Agent-edit:** an agent may add or update keys at any time (e.g. "noticed a new work handle,
  recorded it"). Two invariants keep agent edits safe: **(1)** edits validate against schema v1
  before write (an agent-written profile that fails schema is rejected, not shipped forward);
  **(2)** keys the schema does not define go under `custom:` — agents never invent new
  top-level keys, so the schema stays the contract and `custom:` absorbs the unplanned. This is
  the profile analogue of the `sops-local/` "other"-type ruling (M-03 / `0013` item 6); the two
  should share one classification model when M-03 lands.
- Grooming is **any-time and idempotent** — the profile is data, not a decision; changing it
  binds nothing and needs no ruling.

### How the pocket-librarian consumes surface (i)

The librarian already treats its identity as config, not code (`pocket-librarian-v1-spec.md`
§10.1, §11.2 "identity-neutral binary"): `DESK_ROOT`, `DESK_NAME`, the path constants
(`DECISIONS_DIR`, `TASKS_DIR`, …), model/provider, and GitHub handles all come from env +
config, and the embedded system prompt "names no person, org, repo, or issue" (§6.1). Today
those arrive as **env vars**; the profile is the **shared, higher-level surface** that supplies
them:

- **Reconciliation:** the librarian's config loader reads `_knowledge/profile.yaml` first, then
  applies env-var overrides. `DESK_NAME` ← `profile.desk.name`; `DESK_ROOT` ← the discovered
  profile's directory (or `profile.desk.root`); the path constants ← `profile.desk.paths.*`;
  `LLM_PROVIDER`/`LLM_MODEL` ← `profile.models.*`. Env still wins when set (CI, one-off runs),
  preserving §10.1's "never override an already-set env var" posture. This removes the last
  hardcoded-default risk: there is one identity source, not two.
- **The `prompts` collection (spec §4.10)** is the librarian's *own* prompt data surface — an
  editable, versioned system prompt seeded from the embedded default. It is **complementary**,
  not a duplicate: `prompts` holds the prompt *body* (groomed via the admin GUI, versioned in
  the DB); the profile holds the *desk facts* the prompt interpolates. `systemPrompt(ctx, app,
  cfg)` already "interpolates `cfg.DeskName` and the configured paths" (§6.1) — those `cfg`
  values now originate in the profile. No spec change to §4.10 is required; this design names
  where its `cfg` comes from.

---

## Surface (ii) — the `_knowledge/` freeform background folder

### Contract

- **What goes in it:** unstructured markdown background the agent reads for *context and
  judgment* — project history and rationale, a domain glossary, who the people/orgs are, "how I
  like things done" prose that does not reduce to a key, prior-art notes. It is the deployment's
  memory-of-context.
- **What does NOT go in it:** **secrets** (→ `_meta/secrets/` / env), **work-state** (the board
  is the single truth; `_knowledge/` never restates counts/status), and anything the system
  must substitute **mechanically** — that is a profile key, not prose.
- **How the agent reads it:** at session/run start the agent loads `_knowledge/**/*.md` (minus
  `profile.yaml`/`profile.example.yaml`) as **background context** — the plugin surface treats
  it the way a `CLAUDE.md` import or a knowledge directory is loaded (read-only orientation).
  Bounded: a size/word budget caps what is auto-loaded so a large folder does not blow the
  context window; over-budget files are indexed/summarized rather than loaded whole. The agent
  reads `_knowledge/` for background; it does not treat it as an instruction/rule source
  (rules = schema v1, per `0013` item 8).
- **Relationship to surface (i):** structured = the keys the system substitutes; freeform = the
  prose the agent reasons over. **Promotion path:** a `_knowledge/` fact that starts getting
  referenced by prompts (the system needs it verbatim) should **graduate to a profile key**;
  a profile value that grows explanatory nuance keeps the key and adds the nuance as
  `_knowledge/` prose. The two surfaces are a spectrum with a clear seam, not rivals.

### Folder layout

```
_knowledge/
  profile.yaml            # surface (i) — the structured, substitutable config
  profile.example.yaml    # shipped placeholder-only template (the only profile the plugin ships)
  README.md               # one-liner: what this folder is and the (i)/(ii) split
  background/             # freeform prose the agent reads for context
    project-context.md    # what this deployment is, why it exists
    glossary.md           # domain terms
    people-and-orgs.md    # who's who (names live here as data, never in shipped code)
    conventions.md        # "how I like things done" that doesn't reduce to a profile key
```

The `background/` subfolder is a convention, not a requirement — flat `_knowledge/*.md` is
also valid; the loader reads recursively.

### How the pocket-librarian relates to surface (ii)

`_knowledge/` lives under `DESK_ROOT`, so the librarian will **index** it on sweep (it is
visible to `query`), but it must **never write** to it — it is freeform user data, not a
templated desk entity. **Required follow-up (out of my scope — librarian spec owner):** add
`_knowledge/` to the default `.librarian-ignore` embedded defaults (`pocket-librarian-v1-spec.md`
§10.1), so it is write-excluded / flag-only exactly as `_meta/` is. Until then the librarian
could propose a "fix" against a `_knowledge/` file; the ignore-list entry closes that.

---

## Surface (iii) — the graduation lint (mechanical neutrality enforcement)

Item 9 must be **mechanically enforced, not aspirational** (`0013` "Affects": "the graduation
gate should fail on any hardcoded name/org/repo/issue reference … a lint candidate alongside
the S3 plugin-lint set"). This is that gate.

### What it scans

- **In scope (the shipped surface):** every file that gets **packaged/distributed** — the
  plugin's `plugin/` tree (agent/skill/command definitions, templates, docs) and the
  librarian's `librarian/` shipped tree (Go source, code comments, embedded templates such as
  `templates/librarian-system-prompt.txt`, docs). This is the build output, not the whole repo.
- **Out of scope (explicitly allowlisted):** `_knowledge/` (both surfaces above — that is where
  identifiers are *supposed* to live), `profile.example.yaml` (placeholders only), the desk's
  own instance files (`CLAUDE.md`, `_meta/HANDOFF.md`, `_structure/decisions/**` — a
  deployment's data, per item 9 scope), the `analyses/`/plan deliberation provenance, and test
  fixtures explicitly marked as neutrality-exempt.

### Token patterns it flags

Two families. The first is **self-closing** — it uses surface (i) as its own denylist:

1. **Profile-value occurrences.** The lint loads `_knowledge/profile.yaml` (or, in CI, a
   reference profile), collects **every literal scalar value** in it — handles, repo slugs,
   board number, desk name, machine names, email, display name — and flags any occurrence of
   those literals in an in-scope shipped file. This is the elegant closure: the profile is both
   the de-identification *source* and the lint's *denylist*. Anything a real deployment would
   personalize, hardcoded into shipped code, fails.

2. **Structural identifier patterns** (independent of any profile, catches identifiers the
   author never put in a profile):
   - **Bare issue refs:** `(?<![\w&])#\d+` — reuse the librarian's `ISSUE_REF_RE`
     (`pocket-librarian-v1-spec.md` §11.2; RE2 has no lookbehind, so match `#\d+` with a
     preceding-byte rejection check).
   - **GitHub owner/repo slugs & URLs — host-form only:** `github\.com/[\w.-]+/[\w.-]+`,
     `git@github\.com:[\w.-]+/[\w.-]+`, and full `https?://github\.com/…` URLs. **Bare,
     hostless `owner/repo` slugs are deliberately NOT structurally matched** — a raw
     `[\w-]+/[\w-]+` is indistinguishable from a file path or a language import (`plugin/core`,
     `os/exec`, `encoding/json`) and would match pervasively across the shipped tree, making the
     "returns zero on a clean tree" criterion (D8) unsatisfiable. The real identifiers that
     matter — the deployment's actual owner/repo — are already caught by the **profile-value
     denylist** (family 1 above), which flags the literal wherever it appears, host or not.
     Residual gap (accepted): a hostless slug that is neither in the profile nor host-qualified
     cannot be caught by regex without also flagging every path; the profile denylist +
     host-form patterns + review cover it, and any such leak found is remediated by adding it
     to the profile (which then makes it a hard-fail via the denylist).
   - **Project-number references:** `project\s+\d+` / `projects/\d+`.
   - **Known-handle seeds (optional, tiny):** an allowlist-managed short list of obvious
     personal handles/orgs to catch leaks even before a profile exists.

### The sanctioned escape

A would-be-flagged token **passes** in exactly one of these forms — this is the mechanical
statement of "values live in config, not code":

- it appears inside a **`{{profile.<path>}}` or `{{env.<VAR>}}` placeholder** (surface i
  substitution), or
- it lives in an **allowlisted path** (surface ii `_knowledge/`, `profile.example.yaml`, marked
  fixtures), or
- it is an explicit entry in a **tiny `neutrality-lint.allow` list** for legitimate literals
  (the placeholder module path `github.com/example/pocket-librarian`; clearly-marked example
  values in docs). Keep this list minimal — every entry is a hole in the guard.

Everything else is a **build failure**: report `file:line`, the matched token, and the
**suggested profile key** to move it to (e.g. `hsb3` → `{{profile.identity.github.personal}}`).
The remediation is always "move the literal into `profile.yaml` and reference it," never "delete
the name" — de-identification means *reference the config surface*, per
`.claude/memory/plugin-artifact-neutrality.md`.

### Where it runs

- **CI gate** on the plugin repo: the neutrality lint is a required check on the build/graduation
  path; a hardcoded identifier cannot merge.
- **Graduation pass:** when the spec + templates graduate *into* the plugin repo (`0013`
  "Affects": "a de-identification pass runs when the spec + templates graduate"), the lint is the
  acceptance gate for that pass.
- Runs alongside the S3 plugin-lint set and the INS-01 unquoted-`synopsis` YAML check
  (`_meta/improvement-log.md`).

---

## Deliverables

1. **schema v1 `profile` block** — the field set above, expressed in the shared `schema/`
   (validates a `_knowledge/profile.yaml`), plus `profile.example.yaml` (placeholders only) as
   the shipped template.
2. **The substitution loader** — a small, product-shared resolver: discover the profile by
   walking up from cwd, resolve `{{profile.…}}` / `{{env.…}}` placeholders, fail loud on a
   missing no-default key. One implementation the plugin loader and the librarian's config
   loader both use (or two thin adapters over one contract).
3. **`_knowledge/` loader contract** — recursive markdown read at session/run start, minus the
   profile files, under a bounded context budget.
4. **The neutrality lint** — profile-value denylist + structural patterns + the `{{profile.…}}`/
   allowlist escape, wired as a required CI check and the graduation-pass gate.
5. **Librarian reconciliation** — the librarian's config loader reads the profile then applies
   env overrides; the required-follow-up ignore-list entry for `_knowledge/` (below) is filed
   against the librarian spec.

## Acceptance criteria

- A shipped plugin/binary, grepped, contains **zero** personal identifiers: no name, no
  `hsb3`/org, no `owner/repo`, no `project N`, no bare `#N` outside a `{{profile.…}}` placeholder
  or an allowlisted path. The neutrality lint passes on a clean tree and **fails** on a seeded
  hardcoded identifier (test both directions).
- A stranger installs the plugin, fills `_knowledge/profile.yaml` from `profile.example.yaml`,
  and the system runs on *their* repos with **no edit to any shipped definition** (the item 9
  "no find-and-replace" test).
- A required `{{profile.x}}` with no default and an absent key produces a **loud load error**,
  not a silent empty substitution.
- The pocket-librarian resolves `DESK_NAME` / paths / model from the profile, env still
  overrides, and its embedded prompt + `prompts` collection remain identity-neutral
  (`pocket-librarian-v1-spec.md` §6.1/§4.10 unchanged).
- The profile validates against schema v1; an agent-written profile that violates the schema is
  rejected; unplanned keys land under `custom:` without a schema bump.

## Required out-of-scope follow-ups (other owners)

- **Librarian spec owner** — add `_knowledge/` to the `.librarian-ignore` embedded defaults
  (`pocket-librarian-v1-spec.md` §10.1) so the freeform folder is write-excluded / flag-only,
  and name the profile as the origin of `cfg.DeskName` + path constants in §10.1/§11.2. I do not
  edit that file.
- **Schema/sim-pod owner (schema v1, `0012`)** — fold the `profile` block into schema v1 and
  reconcile its `custom:`/"other" open-ended pattern with the M-03 `sops-local/` "other"-type +
  template-scoping design (`0013` item 6) so there is one classification model, not two.
- **Build-brief owner (`build-brief.md`)** — wire the neutrality lint into the plugin repo's
  required-checks list and cite this design for the data-surface contract.

## Decisions I made that this session should ratify

1. **One folder, two kinds of file — RATIFIED (Henry, 2026-07-15): single root.** The structured
   profile lives at `_knowledge/profile.{yaml,json,md}` *inside* the freeform folder, not as a
   separate top-level home — one personalization root to discover, secure, and allowlist. Henry
   also ruled the profile may be authored as YAML, JSON, or Markdown-with-frontmatter (loader
   picks by extension — see "Format & location").
2. **`{{profile.…}}` double-brace + `profile.`/`env.` prefix** as the substitution syntax
   (over `${…}` or bare `{{…}}`), chosen specifically so the lint's escape rule is unambiguous.
3. **Fail-loud on a missing required key** (vs. silent empty substitution) — the safer default
   for a de-identification mechanism, at the cost of requiring `|| "default"` on genuinely
   optional placeholders.
4. **The profile doubles as the lint's denylist** — the single most load-bearing design choice
   here; it makes neutrality self-closing (anything a deployment personalizes, hardcoded, fails)
   but means the lint needs a reference profile available at CI time.

## Deliberately deferred

- The **exact context budget** for `_knowledge/` auto-load (word/byte cap, summarize-vs-load
  threshold) — a tuning parameter for the build, not a design fork.
- **Per-artifact-format placeholder rendering** (how a Claude Code skill file vs. an OpenCode
  `.ts` plugin vs. Go embedded text each perform the `{{profile.…}}` substitution) — an
  implementation detail of the shared loader; the contract is fixed here, the bindings are the
  build's.
- The **`custom:` ↔ `sops-local/` "other"-type** unification — flagged to the schema owner
  above; not resolved here because it is M-03's design pass.
- A **secrets-in-profile-by-reference** convention beyond `secrets_ref` naming env vars — kept
  minimal on purpose; credentials stay in env / `_meta/secrets/`.

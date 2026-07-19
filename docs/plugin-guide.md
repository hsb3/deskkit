_The four desk-standard skills as user journeys: when to reach for each, what it does to your desk, and what you get — grounded in the shipped skill contracts and the MCP tools behind them._
Status: active

# The desk-standard plugin — four skills

The plugin gives Claude Code four skills that stand up, check, evolve, and adopt an
**executive desk** — a planning surface that oversees your projects and keeps decisions,
analyses, status, and drafts out of the repos it watches. The payoff: a desk that conforms
to one shared standard, personalized entirely from your `_knowledge/profile.yaml`, with
**no identity ever hand-typed into a shipped file.**

Under the skills sits a harness-pure core exposed as four MCP tools — `profile_get`,
`profile_validate`, `template_render`, `knowledge_index`. The skills call these to resolve
your profile and materialize scaffolding. This guide shows real tool output so you can see
what a skill actually does before you run it. For install, see `getting-started.md`; for the
rules the skills enforce, the `conventions-standard` skill is the reference.

<!-- A recording of desk-setup running inside a Claude Code session would fit here; it
requires an interactive harness, so it is left for a manual capture. -->

## Which skill for which moment

| You want to… | Reach for | What you get |
|---|---|---|
| Stand up a brand-new desk, or lightly adopt a folder | `desk-setup` | A conformant scaffold, placeholders resolved from your profile |
| Know what a rule requires, or check a desk against the standard | `conventions-standard` | The rule set (K1–K29) + a read-only adherence checklist |
| Fold real friction back into the standard | `harvest-loop` | A versioned, changelogged revision of the standard |
| Bring a large, messy existing folder fully up to standard | `brownfield-adoption` | A locked, staged, approval-gated migration |

## The engine underneath: the four MCP tools

Every skill personalizes from your profile through these tools. The transcripts below are
real invocations against an `example-desk` whose `_knowledge/profile.yaml` is filled with
placeholder identifiers.

**`profile_get`** resolves a dotted key to its scalar, and **fails loud** if the key is
absent — there is no silent empty default:

```console
$ profile_get { "path": "repos.default" }
{"path":"repos.default","value":"example-org/example-desk"}

$ profile_get { "path": "identity.github.personal" }
{"path":"identity.github.personal","value":"example-personal"}
```

**`profile_validate`** checks the discovered profile against schema v1 and reports every
violation — including unknown top-level keys:

```console
# a well-formed profile
$ profile_validate {}
{"valid":true,"errors":[],"profilePath":".../example-desk/_knowledge/profile.yaml"}

# a profile with an out-of-schema key
$ profile_validate {}
{
  "valid": false,
  "errors": ["(root) must NOT have additional properties (unknown key \"bogus_top_level_key\")"],
  "profilePath": ".../bad-desk/_knowledge/profile.yaml"
}
```

**`template_render`** substitutes `{{profile.…}}` / `{{env.…}}` placeholders against your
profile — this is exactly how `desk-setup` materializes scaffold seed files:

```console
$ template_render { "template": "# {{profile.desk.name}}\nOwner: {{profile.identity.name}} (@{{profile.identity.github.personal}})\nDefault repo: {{profile.repos.default}}\nModel: {{profile.models.model}}" }
# example-desk
Owner: Example Owner (@example-personal)
Default repo: example-org/example-desk
Model: claude-opus-4-8
```

It is **fail-loud**: a placeholder with no `|| "default"` fallback that resolves absent
aborts and names every offender — it never silently writes an empty string into your
scaffolding:

```console
$ template_render { "template": "repo: {{profile.repos.nonexistent}}" }
ERROR: profile substitution: missing required key(s) with no default: {{profile.repos.nonexistent}}
```

**`knowledge_index`** lists your `_knowledge/` background files with size metadata (and
inlines content up to a byte budget), so an agent can pull in freeform context:

```console
$ knowledge_index {}
{"fileCount":1,"bytesIncluded":96,"entries":[{"path":"context.md","bytes":96,"words":15}]}
```

(The real response also carries `root`, `budget`, and the inlined `content` per entry —
elided here for width.)

## `desk-setup` — stand up (or lightly adopt) a desk

**Reach for it when** you are creating a new desk from scratch, or bringing a small
pre-existing folder under the standard.

**What it does to your desk.** It owns the *procedure* and the *scaffold*, not the rules.
It ships a deliberately **standard-free** skeleton in `assets/template/` — folders plus
placeholder instrument files with no conventions prose baked in:

```
CLAUDE.md                 # entry: orients only — placeholders for desk name / role / repos
.gitignore                # the standard gitignore stanza (secrets kept keep-only)
_meta/{README,HANDOFF,improvement-log}.md
_meta/secrets/.gitkeep    # the one never-committed secrets home
_structure/decisions/README.md
_knowledge/README.md
```

The **greenfield runbook** is ordered and verifies each step before the next: name and
place the desk *outside* the repos it watches → fix goal and role (the critical rule: the
desk drafts, the owner applies every external write) → copy the scaffold → create the
profile → **run `template_render` to resolve every `{{profile.…}}` placeholder** →
fill the entry brief and `_meta/` instruments → write thin app-instructions that only
*point* at the version-visible files → connect the overseen repo read-only.

**What you get.** A flat, conformant desk with your identifiers rendered in — never
hand-typed. `desk-setup` also defines the read-only, approval-gated **brownfield adoption**
mode (`untouched → inventoried → approved → reconciled`) with a mandatory user-approval gate
before any write; for a *large* real migration it hands off to `brownfield-adoption` below.
It closes by handing the result to the `conventions-standard` adherence checklist.

## `conventions-standard` — the rule set and the checklist

**Reach for it when** you are authoring, reviewing, structuring, or auditing a desk, or you
just need to know what a rule requires.

**What it does.** This is reference content, not an executable checker. It states the
executive-desk conventions standard v1 as rules **K1–K29** — the critical rule (desk drafts;
owner applies every external write), the frontmatter contract (enforced by schema v1), the
two status vocabularies, decision-record rules (append-only; supersession vs. dated
correction; renumber-the-later-doc on a number collision), the closed ALL-CAPS instrument
set, the one secrets home, the decision spine, and the five-layer skeleton.

**What you get.** A single consultable source of truth for the standard, plus a **read-only
adherence checklist**: nine testable rule statements and a non-mutating procedure that
reports `path — rule — what is wrong` deltas and changes nothing. Mechanical enforcement —
scan, flag, auto-fix, restore — is deliberately **not** this skill's job; that belongs to
the deskkit (see `librarian-guide.md`), and this skill defines the very rules those
tools enforce so the two never own the same fact twice.

## `harvest-loop` — evolve the standard from real use

**Reach for it when** enough friction has accumulated, or a theme recurs across desks. The
loop is trigger-based, never scheduled.

**What it does.** Every desk keeps a **friction ledger** (`_meta/improvement-log.md`) with
three parts: a maintenance backlog (`M-nn`), a dated pass log, and the harvest target — the
**instruction-friction ledger** (`INS-nn`), where a convention rubbed against real work. A
harvest pass collects the `INS-nn` entries across live desks, triages transferable lessons
from desk-local quirks, folds the transferable ones into **one versioned revision** that
cites the absorbed entry IDs (`absorbs INS-04, INS-07`), marks those entries resolved with a
pointer back to the revision, and bumps the plugin's single authoritative version plus a
newest-first changelog entry.

**What you get.** A standard that improves without ambient drift, with a traceable audit
trail from friction to rule change. A rule change that does not bump the version is a bug.
The loop evolves the *standard and plugin* — it does not touch desk work-state (the board
owns that) and does not mutate desk files itself.

## `brownfield-adoption` — the hardened migration runbook

**Reach for it when** you are adopting a *real*, large, non-conformant planning folder — not
a blank greenfield — and need the full field-tested procedure behind K24.

**What it does.** It owns the procedure (not the rules), as a phased runbook mapped onto the
K24 status track, with the librarian baseline deliberately the **final** gate:

1. **Lock** — zip the intact desk *including `.git/`* and verify the zip before touching
   anything.
2. **Inventory** (read-only) — build the disposition spine from a **literal disk
   enumeration** (`ls -A`), never from memory or a handoff; split gitignored content into
   load-bearing vs. litter with an explicit keep/drop call each.
3. **Disposition table as a file** — one row per top-level item; two rows are mandatory and
   explicit: the `.gitignore` semantics flip, and "ratified decision records adopted as-is."
4. **GATE** — mandatory user approval before *any* write, packaged as the table plus 2–3
   residual judgment questions (always including fresh-history vs. preserve-history).
5. **Init → migrate** — stage the old content, init a fresh skeleton via `desk-setup`, then
   apply the approved dispositions row by row; the source stays intact.
6. **Author instruments → librarian baseline** — write and validate the profile, then
   `migrate up → sweep → patrol` into a **fresh** store, recording the run id and store path.

**What you get.** A migration that is safe and repeatable, not hands-free: judgment
migrations stay the adopter's call, non-mechanical findings stay flagged, and the runbook
**never deletes** the source or the lock zip — you remove them when satisfied. If the plugin
is not installable in the environment, the skill degrades to a docs-driven mode against the
`conventions-standard` docs.

## Notes on the transcripts

The MCP-tool transcripts above were produced by invoking the shipped `plugin/core` tool
handlers directly (via `bun`) against scratch desks with placeholder profiles; store paths
are elided for width. The `claude plugin install` step and running the skills inside an
interactive Claude Code session are covered in `getting-started.md` and are
documented-not-rerun here (they require an interactive harness). Skill behavior is quoted
from the shipped `SKILL.md` contracts.

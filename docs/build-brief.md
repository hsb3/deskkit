---
type: plan
status: draft
created: 2026-07-15
updated: 2026-07-15
tags: [desk-tooling, desk-standard-plugin, pocket-librarian, dual-format, opencode, claude-code, mcp, schema-v1, identity-neutral, w4, build-brief]
synopsis: "W4 build brief for the joint desk-tooling repo: one repo, two products (the dual-format desk-standard plugin + pocket-librarian Go binary) over a shared MCP tool boundary and shared schema v1, with identity-neutrality enforced as a required CI check wired to the M-05 data surfaces."
---

# desk-tooling — W4 build brief

_Work product — does not govern this desk._

Status: draft — Phase 2 of the graduation package (`README.md`), pending Henry's sign-off
Date: 2026-07-15
Governs (when the crew adopts it): the coding-agent build of the new shared plugin + librarian repo.

> Register note: this brief names things in plain English first, id second (e.g. "the
> desk-standard plugin", "the librarian's six tools"). It states **deliverables, acceptance
> criteria, and parallelism — never timelines** (desk conv 5).

> **Decisions ruled by Henry (2026-07-15) — these resolve §8's open items:**
> 1. **Repo name = `desk-standard`.**
> 2. **Profile location = a single personalization root, `_knowledge/`** (not two scattered
>    homes). The profile may be authored as **YAML, JSON, or Markdown-with-frontmatter** in
>    `_knowledge/`; the loader selects by extension. (Feeds back to `m-05-data-surfaces.md`,
>    updated to match.)
> 3. **The MCP seam = YES, outbound.** The librarian exposes its six tools **as an MCP server —
>    the librarian's "hands"** — consumed by both plugin formats, AND as CLI subcommands over the
>    same tool core, modeled on the **`hsb3/outlook-mcp`** architecture (one standalone binary =
>    MCP server + CLI, read-only default with write-mode gating + dry-run). This extends the
>    librarian spec (today inbound-MCP only, §7.2) — carried as punch-list item 4 (§7).

---

## 1. Objective + W4 framing

**Build the joint "desk-tooling" repository** ruled by Henry on 2026-07-15
(`_structure/decisions/0013` item 3, sub-question resolved): a single new standalone GitHub
repo that houses **two products under one roof over a shared schema** — the **desk-standard
plugin** (Claude Code + OpenCode) and **pocket-librarian** (the single Go binary,
`_structure/decisions/0015`) — the librarian sharing the repo because it consumes the plugin's
schema v1 as its rule source (`0013` item 8, item 3 sub-question).

This is a **W4 graduation** (desk workflow W4, CLAUDE.md). The division of labour is fixed and
non-negotiable:

- **The desk drafts this brief and the design** (`README.md`, `m-05-data-surfaces.md`, the
  librarian spec). **The desk never writes production code and never merges to main.**
- **Coding agents build the repo** from this brief. The desk reviews against the acceptance
  criteria in §6; Henry merges.

Inputs the crew builds FROM (all in `_meta/plans/desk-tooling/`):

| Input | What it supplies |
|---|---|
| `README.md` | Package framing; the ruled one-repo / two-products / shared-schema shape. |
| `m-05-data-surfaces.md` | The identity-neutral data surfaces (profile + `_knowledge/` + neutrality lint) the build consumes. §5 of this brief wires to it. |
| `pocket-librarian-v1-spec.md` | The librarian build spec (runtime ruled `0015`; six tools §5; extension points §7; `.librarian-ignore` §10.1). |
| `_structure/decisions/0013` item 9 | The binding identity-neutral constraint (§5 here). |
| `_structure/decisions/0013` item 8 | Schema v1 is the librarian's single rule source (§3 here). |

---

## 2. Repo shape

Henry ruled **one repo, two top-level products + shared `schema/`** (`README.md`,
2026-07-15). This brief refines that with the package split the OpenCode porting guide
recommends for anything beyond a simple skill or agent
(`OPENCODE_TO_CLAUDE_CODE_GUIDE.md:53-72`, "Recommended Architecture"):

```
desk-standard/              # ruled 2026-07-15 (identity-neutral, 0013 item 9)
  plugin/                   # the desk-standard plugin — DUAL-FORMAT
    core/                   # shared domain logic — imports NEITHER harness
    mcp/                    # the plugin's shared tool boundary (MCP) — both formats consume it (librarian outbound-MCP exposure is §8 open decision #2, not assumed here)
    opencode/               # OpenCode adapter — TS module on Bun, @opencode-ai/plugin, thin
    claude-plugin/          # Claude Code adapter — .claude-plugin/plugin.json, declarative, thin
  librarian/                # pocket-librarian — single Go binary: MCP server + CLI over one tool core (outlook-mcp pattern)
  schema/                   # schema v1 + the M-05 profile block; shared by plugin + librarian
  _knowledge/               # M-05 freeform background folder (+ profile.example.yaml — see §8 open decision)
  README.md
```

**Package-split rationale (cited to the guide).** The guide's core rule is a strict dependency
direction (`OPENCODE_TO_CLAUDE_CODE_GUIDE.md:74-81`):

- `core/` "accepts plain inputs and returns plain outputs. It must not import
  `@opencode-ai/plugin`, invoke Bun shell helpers, or parse Claude hook payloads"
  (`:76`). It is pure domain logic (`:58`, `:230`).
- `mcp/` "owns operations the agent should call directly … It is the shared tool boundary"
  (`:77`).
- The OpenCode adapter "translates plugin hooks and custom tools into calls to `core` or
  `mcp`" (`:78`).
- The Claude adapter "is packaging plus thin hook executables" (`:79`).

This split is what makes the dual-format requirement (§4) tractable: capability lives in
`core/`+`mcp/`, and each harness gets a thin adapter, never a mechanical translation of the
other (`:9`, `:216`).

---

## 3. The two products + the shared seam

### 3.1 Product A — the desk-standard plugin

The desk-side sibling to `project-workflow` (`0013` item 2): it encodes the executive-desk
conventions/standard as agent definitions, skills, commands, and templates, shipped in **both**
Claude Code and OpenCode formats (§4). Its build plan is the S2/S3 item set adopted in `0013`
item 1 (including the **brownfield-remediation** capability Henry added — bringing an existing
non-conformant desk up to standard; its shape is a queued design pass, `0013` "Affects", not
resolved in this brief).

### 3.2 Product B — pocket-librarian

The single Go binary specced in `pocket-librarian-v1-spec.md` (runtime ruled `0015`: PocketBase
imported as a library, `eino` for the provider-agnostic agent loop). It exposes **six tools** —
sweep / patrol / propose_fix / apply_fix / restore / query (`pocket-librarian-v1-spec.md:770`,
§5) — under the binding safety boundary (`_structure/decisions/0014`: record-original-first,
templates-only, ignore-list). Built as-specced; this brief does not re-open the spec, it wires
it into the repo.

### 3.3 The shared seam — MCP boundary + schema v1

Two things unify the two products so they are one repo, not two colocated ones:

**(a) Schema v1 (the rule seam).** One shared `schema/` is the single source of rules and
structure both products consume. `0013` item 8 makes the librarian a **consumer** of the
plugin's schema v1 as its rule source (one owner per rule) — it stops carrying its own embedded
rules at the mid roadmap stage. Schema v1 also carries the **M-05 `profile` block**
(`m-05-data-surfaces.md` "Field set (schema v1 profile block)"), so personalization is
schema-validated for both products. `0013` item 4 frames schema v1 as the seed of Henry's single
estate-wide schema, so `schema/` is authored product-neutral.

**(b) The MCP tool boundary (the capability seam).** The guide names MCP as "the portable
custom-tool mechanism" / "the shared tool boundary"
(`OPENCODE_TO_CLAUDE_CODE_GUIDE.md:25,77`) and says to build the MCP surface first (`:130-131`).
So model-callable desk operations live behind the `plugin/mcp/` server and are consumed by both
plugin formats.

> **RULED 2026-07-15 — outbound MCP exposure = YES (the librarian's "hands").** The librarian's
> six tools are **exposed outbound as an MCP server** so both plugin formats can call them — the
> seam that makes the librarian's capabilities reachable from a Claude Code or OpenCode session —
> AND as CLI subcommands over the same tool core, per the **`hsb3/outlook-mcp`** architecture (one
> standalone binary = MCP server + CLI, read-only default with write-mode gating + dry-run). The
> librarian spec today frames MCP as an **inbound** vector only ("eino exposes MCP servers' tools
> as `InvokableTool`s", `pocket-librarian-v1-spec.md:1724-1729`, §7.2) — i.e. the librarian
> *consuming* external MCP tools. Exposing its own six tools *as* an MCP server is the **same eino
> tool contract**, an additive extension not a contradiction; the librarian-spec owner adds the
> outbound surface + its safety-mode gating (punch-list item 4, §7).

---

> **Scope change (2026-07-16, Henry):** v1 ships Claude Code only. D4 (OpenCode adapter) is
> descoped — superseded by a common-core fan-out build that produces both harness instances,
> tracked at `hsb3/dotfiles-agents-workbench#50`; a frozen partial-adapter spike is preserved
> unwired at `plugin/opencode/`. AC1's OpenCode half and AC3's cross-format clause apply to
> that endeavor, not v1.

## 4. Dual-format requirement

Henry ruled the plugin ships **Claude Code AND OpenCode formats** (`README.md`, 2026-07-15).
The binding principle, cited: **"Port the capability, not the implementation"** — do NOT
mechanically convert one format into the other
(`OPENCODE_TO_CLAUDE_CODE_GUIDE.md:9`; restated `:216` "share content, libraries, services, and
MCP contracts; generate or hand-write harness adapters … the runtimes are not isomorphic").

### 4.1 The two formats (verified facts, cited)

**OpenCode plugin** = a TypeScript module running in-process on Bun
(`OPENCODE_TO_CLAUDE_CODE_GUIDE.md:23`), exporting a `Plugin`-typed const from
`@opencode-ai/plugin` — verified in the working example: `import type { Plugin } from
"@opencode-ai/plugin"` (`foreman-kit.ts:24`) and `export const ForemanKit: Plugin = async ({
client, directory }) => {` (`foreman-kit.ts:117`). Loaded from local `.opencode/plugins/`,
global `~/.config/opencode/plugins/`, or an npm package (`GUIDE:24`). **No separate manifest** —
the module *is* the plugin (`GUIDE:7,23-24`; contrast the Claude Code manifest below).

**Claude Code plugin** = a declarative packaged bundle (`GUIDE:7,23-24`), with the layout
(`GUIDE:176-186`):

```
claude-plugin/
  .claude-plugin/plugin.json
  hooks/hooks.json
  skills/<skill-name>/SKILL.md
  agents/<agent-name>.md
  bin/<hook-handler>.mjs
  .mcp.json
```

Marketplace distribution is supported (`GUIDE:24`). Smoke-test with `claude --plugin-dir
./plugin/claude-plugin`; validate with `claude plugin validate` (`GUIDE:190-194,235`).

### 4.2 Shared-once vs re-authored-per-format

Author **once**; re-author **per format**. Both columns cited to the guide:

| Author ONCE (shared) | Cite |
|---|---|
| Skill / prompt prose (`SKILL.md`) — "High" fidelity; preserve instructions and references | `GUIDE:35` |
| MCP server implementation + tool schemas — "Keep server implementation and tool schemas unchanged; translate registration format only" | `GUIDE:38` |
| The `core/` domain library — plain in/out, no harness imports | `GUIDE:58,76,230` |

| Re-author PER FORMAT | Cite |
|---|---|
| Agent frontmatter — "rewrite model, tool, permission, and mode frontmatter to Claude Code syntax" ("Medium" fidelity) | `GUIDE:36` |
| All hook / lifecycle behavior — each hook classified **equivalent / approximate / unavailable** | `GUIDE:232` |
| Provider / auth surfaces — "No comparable general plugin extension surface"; keep OpenCode-only | `GUIDE:28,47` |
| UI / notification surfaces — treat as harness-native; omit or re-design with Claude notifications | `GUIDE:49,226` |

### 4.3 The hook mapping is real but medium-fidelity

The guide's portability matrix pairs the lifecycle events (written OpenCode → Claude Code
direction; the pairing is symmetric):

- CC `SessionStart` ↔ OpenCode `session.created` — "Medium … but no OpenCode SDK client"
  (`GUIDE:42`).
- CC `Stop` ↔ OpenCode `session.idle` — "Medium … Claude `Stop` is turn completion, not
  necessarily session idle" (`GUIDE:43`; also `:224`).
- CC `PreCompact` / `PostCompact` ↔ OpenCode `session.compacted` (`GUIDE:44`). In the OpenCode
  format this is **guarded** via `command.execute.before` (block a manual `/compact`) plus
  `experimental.session.compacting` (warn when compaction proceeds) — the pattern proven in the
  working example (`foreman-kit.ts:14,279,309`).

The `chat.*` / system-prompt / provider transforms are **Low / None** fidelity — OpenCode-only,
no Claude equivalent (`GUIDE:45-48`). The build's README must name every intentional loss of
fidelity (`GUIDE:237`).

---

## 5. Identity-neutrality as a REQUIRED CI check

`0013` item 9 is a **binding build constraint**: everything the plugin ships — agent
definitions, skills, commands, templates, code, code comments, docs — must be **identity-neutral
and project-agnostic** (no person's name, no specific GitHub org/repo/project/issue). The test:
a stranger installs the plugin and it runs on *their* projects with **no find-and-replace
through the definitions**.

This brief makes it a **required CI check**, wired to the M-05 design
(`m-05-data-surfaces.md`, surface iii — "the graduation lint"):

- **The neutrality lint runs in the repo's CI as a required check on the build/graduation path**;
  a hardcoded identifier **cannot merge** (`m-05-data-surfaces.md` "Where it runs"; `0013`
  "Affects": "the graduation gate should fail on any hardcoded name/org/repo/issue reference").
- **What it scans:** the shipped surface only — `plugin/` (agent/skill/command definitions,
  templates, docs) and `librarian/` shipped tree (Go source, comments, embedded templates such
  as the system-prompt text, docs) (`m-05-data-surfaces.md` "What it scans").
- **What it flags:** (1) every literal scalar value in `_knowledge/profile.yaml` (the profile
  doubles as the lint's denylist — the self-closing design), and (2) structural identifier
  patterns — bare issue refs (`#\d+`, reusing the librarian's `ISSUE_REF_RE`,
  `pocket-librarian-v1-spec.md:989`, §11.2), **host-qualified** `github.com/owner/repo` slugs &
  URLs only (bare hostless `owner/repo` slugs are deliberately NOT matched — they collide with
  file paths / import paths; the profile denylist covers real ones), `project N`
  references (`m-05-data-surfaces.md` "Token patterns it flags").
- **The sanctioned escape:** a would-be-flagged token passes **iff** it is inside a
  `{{profile.<path>}}` or `{{env.<VAR>}}` placeholder resolved from the M-05 profile, OR lives
  in an allowlisted path (`_knowledge/`, `profile.example.yaml`, marked fixtures), OR is a tiny
  explicit `neutrality-lint.allow` entry (`m-05-data-surfaces.md` "The sanctioned escape").
  Everything else is a **build failure** reporting `file:line`, the token, and the suggested
  profile key.

The remediation is always "move the literal into `profile.yaml` and reference it," never "delete
the name" (`m-05-data-surfaces.md`; `.claude/memory/plugin-artifact-neutrality.md`). The
librarian is already designed to this posture — `DESK_ROOT` / `DESK_NAME` / paths / model /
handles all come from config + env, module path `github.com/example/pocket-librarian`
(`pocket-librarian-v1-spec.md:2161-2166,2199`, §11.2) — so the lint's job is to *keep* it that
way across both products.

---

## 6. Deliverables + acceptance criteria

Each criterion is mechanically verifiable — a command passes, a lint returns zero, or an
artifact exists. **No timelines.**

### 6.1 Deliverables

| # | Deliverable | Where |
|---|---|---|
| D1 | `schema/` — schema v1 + the M-05 `profile` block; validates a `_knowledge/profile.yaml`; ships `profile.example.yaml` (placeholders only) | `schema/` |
| D2 | `plugin/core/` — pure domain library (no harness imports) | `plugin/core/` |
| D3 | `plugin/mcp/` — the MCP server (shared tool boundary); model-callable desk operations | `plugin/mcp/` |
| D4 | `plugin/opencode/` — thin OpenCode adapter (`Plugin`-typed TS module) | `plugin/opencode/` |
| D5 | `plugin/claude-plugin/` — thin Claude Code adapter (`.claude-plugin/plugin.json`, `hooks/hooks.json`, `skills/`, `agents/`, `bin/`, `.mcp.json`) | `plugin/claude-plugin/` |
| D6 | `librarian/` — pocket-librarian built to `pocket-librarian-v1-spec.md`; six tools; safety boundary | `librarian/` |
| D7 | The M-05 substitution loader + `_knowledge/` loader contract (one shared resolver, or thin adapters over one contract) | shared (`plugin/core/` + `librarian/internal/config`) |
| D8 | The neutrality lint, wired as a required CI check | CI + `schema/` allowlist |
| D9 | `README.md` naming both harness versions tested and every intentional loss of fidelity (`GUIDE:237`) | repo root |

### 6.2 Acceptance criteria

1. **Both formats load.** OpenCode: the plugin loads from `.opencode/plugins/` /
   `~/.config/opencode/plugins/` and exports a `Plugin`-typed const. Claude Code: `claude plugin
   validate` passes and `claude --plugin-dir ./plugin/claude-plugin` smoke-tests clean
   (`GUIDE:190-194,235`).
2. **`core/` is harness-pure.** A build/lint asserts `plugin/core/` imports neither
   `@opencode-ai/plugin` nor Claude hook payload types nor Bun shell helpers (`GUIDE:76,230`).
3. **The MCP surface is shared.** The same MCP tool schemas back the operation in both formats;
   no capability is double-registered or duplicated across the OpenCode and Claude installs
   (`GUIDE:236`).
4. **The librarian builds and passes its own gate.** `go build` yields the single binary; the
   spec's verify gate passes (`pocket-librarian-v1-spec.md` §9), including
   rebuild-from-scratch reproducing the same file count + per-path checksum set.
5. **Neutrality lint — both directions.** The lint returns **zero** on a clean tree AND
   **fails** on a seeded hardcoded identifier (test both). A grep of the shipped
   plugin/binary contains no name, no `hsb3`/org, no `owner/repo`, no `project N`, no bare `#N`
   outside a `{{profile.…}}` placeholder or an allowlisted path
   (`m-05-data-surfaces.md` "Acceptance criteria").
6. **The stranger test.** Install the plugin, fill `_knowledge/profile.yaml` from
   `profile.example.yaml`, run on a different repo with **no edit to any shipped definition**
   (`0013` item 9; `m-05-data-surfaces.md`).
7. **Fail-loud substitution.** A required `{{profile.x}}` with no default and an absent key
   produces a loud load error, not a silent empty substitution
   (`m-05-data-surfaces.md` "Missing-key rule").
8. **Schema validation.** The profile validates against schema v1; an agent-written profile
   that violates the schema is rejected; unplanned keys land under `custom:` without a schema
   bump (`m-05-data-surfaces.md` "Acceptance criteria").

### 6.3 Parallelism + the one dependency

**Naturally parallel** (disjoint file scopes — one owner per top-level dir per wave):

- The **librarian Go build** (`librarian/`) — self-contained per its spec.
- The **plugin adapters** (`plugin/opencode/`, `plugin/claude-plugin/`) once `core/`+`mcp/`
  exist.
- **Schema v1** (`schema/`) — authored product-neutral.

**The one hard dependency:** the **M-05 profile + neutrality-lint spine precedes the lint
check.** The neutrality lint (D8) uses the profile as its denylist and the `{{profile.…}}`
placeholder as its escape, so D1 (schema v1 profile block + `profile.example.yaml`) and D7 (the
substitution loader) must land before D8 can gate. Order: **D1 + D7 → D8**; everything else
fans out. (This mirrors `m-05-data-surfaces.md` — the profile "doubles as the lint's denylist …
the lint needs a reference profile available at CI time".)

---

## 7. Reconciliation punch-list

Cross-file reconciliations this build owns, surfaced by the M-05 design pass. Each is a
required build task, not optional:

1. **Librarian spec ⇄ `_knowledge/`.** Add `_knowledge/` to the `.librarian-ignore` **embedded
   defaults** (`pocket-librarian-v1-spec.md:2015-2024`, §10.1) so the freeform folder is
   write-excluded / flag-only exactly as `_meta/` is — otherwise the librarian could propose a
   "fix" against a `_knowledge/` file. AND name the **profile** as the origin of `cfg.DeskName`
   + the path constants in §10.1/§11.2 (today they arrive as env vars; the profile is the shared
   higher-level surface, env still overriding). Source: `m-05-data-surfaces.md` "How the
   pocket-librarian consumes surface (i)/(ii)" + "Required out-of-scope follow-ups".
   > Ownership note: editing `pocket-librarian-v1-spec.md` is the **librarian spec owner's**
   > job, not this brief's — the M-05 doc files it as a required out-of-scope follow-up. This
   > brief carries it so the crew building `librarian/` implements the reconciled behavior and
   > the spec edit is tracked, not dropped.
2. **Schema v1 ⇄ profile block + the "other"-type model.** Fold the `profile` block into
   schema v1 (`m-05-data-surfaces.md` "Field set"), and reconcile the profile's open `custom:`
   field with the M-03 `sops-local/` **"other"-type** design (`0013` item 6) so there is **one**
   classification model, not two (`m-05-data-surfaces.md` "Required out-of-scope follow-ups",
   schema owner). M-03 is a separate queued design pass; this build must not invent a second
   classification model that contradicts it.
3. **Wire the neutrality lint into required checks.** The neutrality lint (§5, D8) is a
   **required** CI check on the build/graduation path, running alongside the S3 plugin-lint set
   and the INS-01 unquoted-`synopsis` YAML check (`m-05-data-surfaces.md` "Where it runs";
   `0013` "Affects").
4. **Librarian spec ⇄ outbound MCP-server exposure (RULED 2026-07-15).** The librarian exposes
   its six tools as an **MCP server** (consumed by `plugin/mcp/`, thus both formats) AND as CLI
   subcommands over one tool core — the `hsb3/outlook-mcp` dual-surface pattern (one binary: MCP
   server + CLI, read-only default, write-mode gating, dry-run). The spec today frames MCP
   **inbound** only (`pocket-librarian-v1-spec.md:1724-1729`, §7.2); the librarian-spec owner
   adds the outbound-server surface + its safety-mode gating. Same eino tool contract — additive,
   not a contradiction. Ownership: librarian-spec owner, tracked here so it is not dropped.

---

## 8. Name candidates + open decisions for Henry

> **All three RESOLVED 2026-07-15 — see the "Decisions ruled by Henry" callout at the top of this
> brief.** Name = `desk-standard`; profile = single `_knowledge/` root (yaml/json/md+frontmatter);
> outbound MCP = yes (outlook-mcp pattern). The candidates/options below are retained for
> provenance. Item 3 (brownfield-remediation shape) remains a queued design pass (M-04).

### 8.1 Repo name candidates (identity-neutral, `0013` item 9)

Henry picks; all three are project-agnostic (no person/org/repo token):

1. **`desk-standard`** — names the product's job (encodes + enforces the desk standard); pairs
   naturally with "the desk-standard plugin" already used in the package docs.
2. **`desk-tooling`** — the working name in `README.md`; broad, accommodates the librarian as a
   second product without implying "just a plugin".
3. **`deskkit`** — shortest; reads as a toolkit for any desk.

(Henry may also prefer a fourth; these are defensible starting options, not an exhaustive set.)

### 8.2 Open decisions to record (do NOT resolve here)

1. **Profile location — one root vs two surfaces.** The M-05 design co-located the structured
   profile at **`_knowledge/profile.yaml`** (a single personalization root to discover, secure,
   and allowlist — `m-05-data-surfaces.md` "Decisions I made that this session should ratify" #1).
   But `0013` item 9 and the neutrality memory frame the **profile and `_knowledge/` as TWO
   surfaces**, and Henry may want the profile at **repo-root `profile.yaml`** instead. It is a
   **one-line move with no other change** (M-05 states this explicitly). **His call.**
2. **Outbound MCP exposure of the librarian's six tools.** Per §3.3: this brief proposes
   exposing the librarian's six tools as an MCP server so both plugin formats can call them (the
   unifying seam). The spec currently frames MCP as an inbound extension vector only
   (`pocket-librarian-v1-spec.md:1724-1729`). Confirming outbound exposure — vs. keeping the
   librarian CLI-/hook-only and the plugin's MCP surface separate — is a design decision for
   Henry + the crew, not settled here.
3. **Brownfield-remediation capability shape.** `0013` item 1 (extended) requires the plugin to
   bring a non-conformant desk up to standard; its scope/shape is a queued roadmap design pass
   (`0013` "Affects"), not resolved in this brief. Flagged so the crew does not improvise it.

---

## Sources

- `_meta/plans/desk-tooling/README.md` — package framing + ruled repo shape.
- `_meta/plans/desk-tooling/m-05-data-surfaces.md` — the three data surfaces + neutrality lint.
- `_meta/plans/desk-tooling/pocket-librarian-v1-spec.md` — the librarian build spec.
- `_structure/decisions/0013-executive-desk-plugin-and-managed-resources.md` — items 1–9 (esp. 8, 9).
- `_structure/decisions/0015-pocket-librarian-runtime.md` — the librarian runtime ruling (Go in-process + eino). (Off-repo executive-desk record; not vendored here.)
- `~/dotfiles/_docs/reference/opencode/OPENCODE_TO_CLAUDE_CODE_GUIDE.md` — OpenCode↔Claude Code porting facts.
- `~/.config/opencode/plugins/foreman-kit.ts` — the working OpenCode plugin example.

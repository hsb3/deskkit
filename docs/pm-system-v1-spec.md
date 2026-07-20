# PM system v1 — Product & Technical Specification

_The build spec for the document-gated work-graph system: a project-management module that
lives beside the librarian on the same single-binary PocketBase chassis, refuses phase
transitions until the required documents exist and validate, and ships as a complementary
plugin for the agent-facing surfaces._

Status: Delivered in v0.7.0 (epic #55, slices D1–D8) — frozen build contract, retained for provenance. The shipped code + tests are authoritative; see §12 (test traceability), CHANGELOG `[0.7.0]`, and ADRs 0002 / 0008.
Date: 2026-07-18 (delivered 2026-07-19)

## Table of contents

1. [Overview, goals, non-goals](#1-overview-goals-non-goals)
2. [Architecture — core + modules (R5.5)](#2-architecture--core--modules-r55)
3. [PM data model (R2)](#3-pm-data-model-r2)
4. [Gates — the spine (R3)](#4-gates--the-spine-r3)
5. [Surfaces (R4)](#5-surfaces-r4)
6. [The complementary plugin (R5.1, R5.3)](#6-the-complementary-plugin-r51-r53)
7. [Boundaries (R6)](#7-boundaries-r6)
8. [Versioning, migration, ops (R7)](#8-versioning-migration-ops-r7)
9. [LATER — designed-for, not built (R1.3, R1.4, R4.4)](#9-later--designed-for-not-built-r13-r14-r44)
10. [Test plan (D6)](#10-test-plan-d6)
11. [Build slices (D2–D8)](#11-build-slices-d2d8)
12. [Requirements-traceability table](#12-requirements-traceability-table)
13. [Design decisions taken by this spec](#13-design-decisions-taken-by-this-spec)

The requirements this spec builds against are the frozen contract (requirements v1). Each
section cites the requirement ids it satisfies. `MUST` binds; `SHOULD` is the default unless
this spec argues otherwise (arguments are flagged inline); `LATER` is designed-for, not built.

---

## 1. Overview, goals, non-goals

### 1.1 What the PM system is

The PM system is a **document-gated work graph** (R1.1). A desk juggles many concurrent
threads, most of which are not GitHub-issue-shaped: design decisions, standards work, research,
vendored builds. Each such piece of work is one **work item** in a self-referencing graph. An
item advances through a small, rigid phase machine — but a transition is **refused by the
server** unless the documents that phase requires already exist, validate against the schema,
and carry the required status (R3.1, R3.2). The librarian already validates documents against
schema v1; the PM module owns the graph; a narrow in-process seam joins them so that
*documents are the accountability artifacts*, and the ported SOP kits (`#49`) define what a
"filled" document looks like per type (R3.4).

The system is self-hosted with no external PM SaaS (R1.2): the schemas, workflows, gate rules,
and agents are all self-defined and run on the same embedded-PocketBase Go binary the librarian
already uses (R5.4).

### 1.2 Goals

- The **minimal provable version** of the gate mechanism (R1.3): one universal work-item graph,
  server-enforced document gates, the exec-desk vocabulary as first-class fields, and a
  single-call cold-start briefing. Everything beyond that is an add-on after it proves out.
- **One binary, one store per desk** (R5.5, R5.2): the PM module and the librarian module share
  the same PocketBase app, store, config, and surface stack — no second daemon, no Docker.
- **Extractable domains** (R5.5): the PM module never reaches into the librarian's collections;
  it obtains document verdicts through a narrow Go interface. The two domains stay separable
  even though they run in one process.
- **Off by default** (R5.5c, R1.3): a desk runs librarian-only until it opts the PM module on.

### 1.3 Non-goals (v1)

- No agent-assisted fast-track mode (R1.4 — LATER; §9).
- No autonomous queue-drain / Ralph loop (R4.4 — LATER; §9).
- No parallel copy of the GitHub board; GitHub linkage is a read-only pointer at most (R6.1).
- No crypto identity layer for actor attribution (R2.5 — plain actor strings only).
- No general-purpose chat; the TUI stays scoped to desk stewardship + PM operations (inherited
  from the librarian's chat scope, ADR 0004).

---

## 2. Architecture — core + modules (R5.5)

R5.5 rules **one PocketBase app / binary / store per desk**, structured internally as
**core + compile-time Go modules** (no runtime plugin loading). This section maps that ruling
onto the *actual current package layout* under `librarian/` and names what moves where.

### 2.1 The ruling, restated as the target shape

- **Core** owns the shared substrate: PocketBase bootstrap, the XDG store resolver + desk
  open-guard, config layering, the schema-v1 validation engine, the tool-core registry that
  feeds MCP/CLI/TUI, and the migration framework (with a per-module schema-version meta
  collection).
- **Each module** (`librarian`, `pm`) registers its own collections, migrations, tools, hooks,
  and TUI views through core's registration interfaces.
- **Three binding disciplines** (R5.5 a/b/c):
  - **(a) narrow validation seam** — the PM module obtains document verdicts through an internal
    Go interface, never by querying the librarian's collections directly.
  - **(b) module-scoped migrations + collection ownership** — each module owns a set of
    collections and a schema version recorded in a meta collection.
  - **(c) per-desk feature gating** — the PM module is inert unless enabled; a desk can run
    librarian-only.

### 2.2 Current layout (what exists today)

The single Go module `github.com/example/pocket-librarian` is today a flat `internal/` tree
(verified against the repo):

```text
librarian/
├── cmd/pocket-librarian/main.go     # the spine: config.Load → StoreDir → pocketbase.NewWithConfig
│                                     #   → migratecmd.MustRegister(Automigrate) → OnServe hook
│                                     #   (deskguard, ignore-file, prompt seed, superuser,
│                                     #   trigger hooks/cron/claimer) → registerToolCommands
├── migrations/                       # 0001..0013 Go up/down pairs, blank-imported by main
├── internal/
│   ├── config/     # Load (env > profile > default), StoreDir (XDG), profile
│   ├── bootstrap/  # CheckDeskGuard (deskCarryingCollections), EnsureSuperuser
│   ├── desklib/    # checksum, frontmatter parse, git meta, write_exact, ignore
│   ├── tools/      # the SEVEN tools + registry.go (ToolSpec, Registry, AgentTools §5.4 gate)
│   ├── agent/      # eino ReAct loop, Session, buildTools (map name→InvokableTool builder)
│   ├── mcp/        # MCP stdio server (switch name→handler; toolInputTypes map)
│   ├── tui/        # full-screen Bubble Tea TUI (ADR 0004)
│   ├── trigger/    # record hooks + cron + the claimer queue (serve-only)
│   ├── prompt/, provider/, setup/
│   └── ...
└── templates/, schema/ (repo-level), plugin/ (the TS plugin lane)
```

Two facts about this layout drive the refactor:

1. **The tool core is librarian-specific.** `internal/tools/registry.go` hardcodes the
   **seven** librarian tools as `Registry`: the original six (`sweep`, `patrol`, `propose_fix`,
   `apply_fix`, `restore`, `query`) **plus `record_feedback`** (added since the original spec —
   a D2 builder migrating "the six tools" would under-migrate; several codebase comments still
   say "six"). Both model-facing surfaces bind tools by a name-keyed switch:
   `internal/agent/agent.go`'s `buildTools` uses a `map[string]func(...)` of eino-tool builders,
   and `internal/mcp/server.go`'s `register` uses a `switch name` plus a `toolInputTypes` map.
   Neither can admit a second module's tools without generalization.
2. **Migrations are global.** `migratecmd.MustRegister(app, ..., {Automigrate: true})` plus the
   blank import `_ ".../migrations"` apply *all* registered migrations in filename order through
   PocketBase's single ordered runner. There is no per-module scoping today.

### 2.3 Target layout (what the D2 refactor produces)

```text
librarian/
├── cmd/pocket-librarian/main.go      # the spine; wires core + the enabled module set
│                                     #   (renamed to cmd/deskkit/ in D2b — AFTER the D2 gate; §2.10)
├── internal/
│   ├── core/
│   │   ├── app/         # PB bootstrap, NewWithConfig, OnServe orchestration
│   │   ├── store/       # StoreDir (moved from config), desk open-guard (moved from bootstrap)
│   │   ├── config/      # config layering (env > profile > default), unchanged semantics
│   │   ├── schema/      # schema-v1 validation engine (the single validator; §2.5)
│   │   ├── toolcore/    # ToolSpec + Registry abstraction; MCP/CLI/TUI binding adapters
│   │   ├── migrate/     # migration framework + module_schema_versions meta collection
│   │   └── module/      # the Module registration interface (§2.4) + the module registry
│   └── modules/
│       ├── librarian/   # all existing librarian code, moved verbatim, registered as a module
│       │   ├── collections/ (the 0001..0013 migrations, re-homed, ids preserved)
│       │   ├── tools/   # the seven tools (unchanged bodies; incl. record_feedback)
│       │   ├── hooks/, tui/, prompt/, agent/, desklib/, provider/, setup/
│       │   └── module.go   # implements core.Module; also implements core.DocumentValidator
│       └── pm/          # NEW — the PM module
│           ├── collections/   # items, notes, dependencies, transitions, desk_config
│           ├── statemachine/  # the rigid phase machine (§3.2)
│           ├── gates/          # gate engine + YAML config loader (§4)
│           ├── tools/          # the PM tool family (§5.1)
│           ├── hooks/, tui/    # realtime wiring + PM TUI views
│           └── module.go       # implements core.Module; consumes core.DocumentValidator
```

The binary rename (`pocket-librarian` → `deskkit`; owner-approved, name included) is
**deliberately NOT part of D2**: `verify.sh` builds `./cmd/pocket-librarian` (line 112), so a
rename inside D2 would contradict the §2.7 "verify.sh passes unchanged" gate. D2 keeps
`cmd/pocket-librarian/` and `verify.sh` untouched; the rename ships as its own slice **D2b**
(§2.10), sequenced immediately after the D2 gate has been demonstrated green. The librarian's
tool subcommands **stay top-level** (`sweep`, `patrol`, `chat`, …) — there is no `librarian`
command group, in D2 or after; moving them would break `verify.sh` and the §2.7 gate. The new
`pm` group (§5.3) is the only command group.

### 2.4 The module registration interface

Core defines one interface every module implements. Modules are registered at compile time in
`main` (a slice literal), never loaded at runtime:

```go
// package core/module
type Module interface {
    Name() string                       // "librarian" | "pm"; the meta-collection key
    SchemaVersion() int                 // this build's expected schema version for the module
    Enabled(cfg *config.Config) bool    // feature gate (R5.5c); pm defaults false

    Migrations() []migrate.Migration    // module-owned up/down pairs, tagged with Name()
    OwnedCollections() []string         // collection names this module owns (ownership guard)

    Tools() []toolcore.ToolSpec         // contributed to the shared registry (§2.6)
    RegisterHooks(app core.App, cfg *config.Config) error   // serve-only record/cron hooks
    TUIViews() []tui.View               // views mounted into the shared TUI

    // Optional capability interfaces a module may also satisfy:
    //   DocumentValidator (librarian provides; §2.5)
    //   RealtimeSource    (pm provides; §5.4)
}
```

`main` builds the module set, filters by `Enabled(cfg)`, and hands the enabled set to core.
Core then: applies each enabled module's migrations through the framework (§2.8), asserts no two
enabled modules claim the same owned collection, merges their `Tools()` into the shared
registry, calls `RegisterHooks` under `OnServe`, and mounts their `TUIViews`.

### 2.5 The narrow validation seam (R5.5a)

The single most important discipline: **the PM module never reads the librarian's collections.**
It asks core for a verdict on a document pointer:

```go
// package core/schema (or core/validate)
type ArtifactRequirement struct {
    Type          string   // schema-v1 / kit type, e.g. "decision" (R3.4)
    RequiredStatus string  // e.g. "accepted"; empty = existence + frontmatter validity only
}

type Verdict struct {
    Exists          bool
    FrontmatterValid bool
    Status          string   // the doc's actual status, for the refusal message
    Satisfied       bool     // Exists && FrontmatterValid && (RequiredStatus=="" || Status==RequiredStatus)
    Missing         []string // human-readable reasons, verbatim into the gate refusal (R3.1)
}

type DocumentValidator interface {
    Verdict(ctx context.Context, pointer string, req ArtifactRequirement) (Verdict, error)
}
```

The **librarian module implements `DocumentValidator`** by reusing its existing frontmatter
parser (`desklib`) and the schema-v1 validation engine that core now owns. Core injects the
implementation into the PM module at registration. If no `DocumentValidator` is registered
(a librarian-less desk — not a supported config in v1, but structurally possible), the gate
engine fails closed: every documented gate refuses, naming "no document validator available."

This keeps the gate *in-process* (one binary, no IPC) while keeping the domains *extractable*:
the PM module depends only on the `DocumentValidator` interface in core, not on `modules/librarian`.

### 2.6 The shared tool core (R5.5, R4.1)

Core generalizes `internal/tools/registry.go` into `core/toolcore`. A `ToolSpec` gains the two
pieces the current name-keyed switches hold implicitly, so a module can self-register a tool on
every surface without editing agent/mcp/cli by hand:

```go
// package core/toolcore
type ToolSpec struct {
    Module       string          // owning module (namespacing + provenance)
    Name         string          // e.g. "get_context", "transition_item"
    Description  string
    InputType    reflect.Type    // replaces mcp.toolInputTypes; drives schema reflection
    WritesFiles  bool
    AgentDefault bool            // §5.4 gate semantics, carried over
    AgentGated   bool
    Invoke       func(ctx, app, cfg, rawJSON) (any, error)  // the single core function
}
```

- **CLI** binding: core generates a cobra subcommand per `ToolSpec` (or a module supplies a
  richer command); the librarian keeps its existing hand-written commands during D2 to preserve
  byte-behavior, then may migrate opportunistically (not required for zero-change).
- **MCP** binding: `core/toolcore` provides the generalized equivalent of today's
  `buildInputSchema` + `register`, iterating the merged registry instead of a hardcoded switch.
  The §5.4 write gate and the restore-never-exposed rule carry over as `ToolSpec` flags.
- **TUI/eino** binding: `buildTools` becomes a loop over the merged registry rather than a
  name-keyed builder map.

The librarian's **seven** `ToolSpec`s (six original + `record_feedback`) move under
`modules/librarian/tools` unchanged in behavior; the PM tool family (§5.1) is added under
`modules/pm/tools`.

### 2.7 Zero-behavior-change guarantee (D2 gate)

The D2 refactor is a **pure move + generalize**, gated by: *the librarian's existing `verify.sh`
Phase-1 gate and the full Go + plugin test suites pass unchanged, and every stable collection id
is preserved.* Concretely:

- Migrations `0001..0013` move under `modules/librarian/collections` **with their
  `NewBaseCollection(...).Id = "pbc_…"` stable ids intact** (rebuild reproducibility, §8.2).
- The librarian's tool bodies, hooks, TUI, prompt seeding, deskguard, and config semantics move
  verbatim; only import paths and package homes change.
- The librarian module's `Enabled()` returns `true` unconditionally (it is the base module).
- `pm` is registered but `Enabled()` defaults `false`, so a D2-only build is behaviorally
  identical to today's binary. The PM collections are not created and no PM surface appears until
  a desk opts in (§8.1).
- **`cmd/pocket-librarian/` and `verify.sh` stay untouched in D2.** The binary/store rename is
  D2b (§2.10), a separate slice that runs only after this gate has been demonstrated green.
- The codebase's own stale "six tools" comments are corrected to seven during the move (§2.2
  fact 1) — a comment-only fix, inside the zero-*behavior*-change envelope.

Any test that changes is a regression to fix, not a test to update — the gate is that the
*existing* suite is green as-is.

### 2.10 D2b — the chassis rename (owner-approved)

The owner has approved renaming the binary to **`deskkit`**. It ships as its own slice,
**D2b**, sequenced immediately after the D2 zero-change gate is green — a deliberately
librarian-visible change with its own acceptance, never mixed into D2:

- `cmd/pocket-librarian/` → `cmd/deskkit/`; `verify.sh` (line 112 build path) and both
  Makefiles are updated **in the same commit**; acceptance = full suite + verify gate green
  *after* the rename.
- **The store home renames with the binary** (`config.AppDirName`:
  `$XDG_DATA_HOME/pocket-librarian/<DESK>/` → `$XDG_DATA_HOME/deskkit/<DESK>/`), with an
  explicit **automatic path migration**: on startup, if the new home is absent and the old
  `pocket-librarian/<DESK>/` directory exists, move it to the new home and log one line. No
  desk loses its store across the rename.
- ADR 0002's store-path literal gets a **dated correction callout** in the same PR that ships
  D2b (per the repo's decision-record discipline: a correction, not a supersession — the
  store-per-desk decision stands; only the literal path changed).

### 2.8 The migration framework + meta collection (R5.5b, R7.1)

Core owns `core/migrate`. Each `Migration` is tagged with its owning module. The framework:

- applies migrations for **enabled** modules only, in `(module load order, then intra-module
  sequence)`;
- after applying a module's migrations, upserts a row in the **`module_schema_versions`** meta
  collection: `{ module: string, version: int, applied_at: date }`;
- on startup, compares each enabled module's `SchemaVersion()` (the build's expectation) against
  the stored row and **refuses to serve if the store is ahead of the binary** (a downgrade — data
  it cannot safely read), while a store *behind* the binary triggers the pending migrations. This
  is what lets a desk lag one module's schema without the other module caring: the librarian's
  version row is independent of the PM's (R7.1).

**Interpretation flagged (§13):** PocketBase's own migration runner and `migratecmd` are global
and compile-time. To keep the D2 zero-change gate, the librarian's migrations continue to run
through the existing PocketBase runner; `core/migrate` wraps that runner and maintains
`module_schema_versions` as the *logical* per-module version of record. **Feature gating (§2.9)
controls whether a module's collections are physically created**: when `pm.Enabled()` is false,
the PM migrations are not registered into the runner at all, so a librarian-only desk has no PM
collections (true physical omission, not inert tables). Flipping the gate on runs the PM
migrations and stamps `module_schema_versions`.

**The mechanism, pinned (binding on D2/D3 builders):**

- **(a) PM migrations MUST NOT use the librarian's `init()` + `m.Register` self-registration
  pattern.** The librarian's migration files register themselves into PocketBase's global list
  via package `init()` + a blank import — unconditional at compile time. A PM migration written
  that way would register regardless of the feature gate and **break it**. Instead, the PM
  module's `Migrations()` manifest (§2.4) returns its ordered up/down pairs as values, and
  `core/migrate` registers them into the PocketBase runner **programmatically, only when
  `pm.Enabled(cfg)` is true**, before the runner executes. A D3 builder copying the librarian
  migration-file pattern is a bug, and test §10.6 catches it (PM collections must not exist on
  a gated-off desk).
- **(b) Librarian migrations `0001..0013` keep their existing `init()`/blank-import path through
  the PB runner unchanged** (the D2 zero-change gate). Per-module stamping works by
  **observation, not interception**: after the runner completes (post-`RunAppMigrations` /
  post-automigrate, at the same points the store is opened today), `core/migrate` reads
  PocketBase's `_migrations` applied-list and matches applied migration file basenames against
  each enabled module's declared manifest — the librarian module's `Migrations()` lists its
  `NNNN_*.go` basenames (a manifest-vs-disk drift test keeps this list honest); the PM module's
  manifest is its programmatic list from (a). It then upserts each module's
  `module_schema_versions` row with the highest applied sequence number for that module. One
  stamping mechanism, both registration styles.

### 2.9 Feature gating (R5.5c, R1.3)

The PM module is enabled per desk via config layering (env > profile > default), default off:

- env `PM_ENABLED=true`, or
- `_knowledge/profile.*` key `modules.pm.enabled: true`, or
- a `desk_config` row once the module is on (the profile/env decides *initial* activation; the
  store carries it thereafter).

When disabled: PM migrations are not registered (no collections), PM tools are absent from every
surface, PM hooks/cron are not bound, and PM TUI views are not mounted. The librarian is wholly
unaffected. This directly serves R1.3 (ship minimal; prove; add on) — the PM module can ride in
the same binary release while staying dark until a desk turns it on.

---

## 3. PM data model (R2)

Five collections, all owned by the PM module. Names follow the requirement (R2.1–R2.6); each
gets a preserved stable `pbc_…` id at authoring time (rebuild reproducibility, §8.2). All API
rules default nil (superuser-only), matching the librarian's collections.

### 3.1 `items` — the universal work item (R2.1, R2.3)

One row per unit of work at any level (thread, epic, task) — a self-referencing graph, not a
table-per-level design.

| Field | Type | Notes |
|---|---|---|
| `id` | text (pbc id) | |
| `desk` | text | desk-scoping + open-guard participation (§7, ADR 0002) |
| `title` | text, required | |
| `parent` | relation → items (single) | the graph edge; null = a root |
| `root` | relation → items (single) | denormalized root for fast subtree/ancestor queries |
| `type` | text | schema-v1 / kit type (R2.3, R3.4): decision, task, analysis, … — `create_item` validates it against the schema-v1 vocabulary and hard-refuses an unknown type (ADR 0012); an absent type stays legal |
| `phase` | select, required | the rigid machine: `queue`,`work`,`review`,`terminal` (§3.2) |
| `blocked` | bool | the side-state flag; independent of `phase` (§3.2) |
| `status_label` | text | friendly vocabulary over the phase (R2.2; §3.3) |
| `court` | select | `owner`,`desk`,`crew`,`vendor`,`external-session` (R2.3) |
| `pointer` | text | desk-relative file path; grammar defined in §3.1a (ADR 0010) |
| `severity` | select | `low`,`medium`,`high` (R2.3) |
| `priority` | number | ordinal within a court/queue (R2.3) |
| `claimed_by` | text | actor string holding the claim (R2.6) |
| `claim_expires` | date | claim TTL horizon (R2.6; §3.6) |
| `version` | number | optimistic-concurrency token (R2.6; §3.6) |
| `properties` | json | typed-field overflow (kit-specific fields) |
| `created`,`updated` | autodate | |

`phase` + `blocked` are the machine (rigid, small, code-owned). `status_label`, `court`,
`type`, `severity`, `priority` are the friendly/first-class vocabulary (data-owned). Keeping
them separate (R2.2) is what lets the label set evolve per desk without touching the transition
logic.

### 3.1a `pointer` grammar (ADR 0010)

The `items.pointer` field is a **desk-relative file path** — never an issue URL and never an
arbitrary locus. The grammar below was shipped before it was specified; this subsection is the
normative definition **ADR 0010** (`docs/decisions/0010-pointer-grammar.md`) ratifies, with no
change to the shipped behavior it names.

- **Form.** A pointer is a path relative to the desk root, optionally suffixed with an advisory
  section anchor: `<path>` or `<path> § <heading>` (e.g. `notes.md § Decisions`).
- **The `§ <heading>` suffix is advisory only and is never checked by a gate.** `Verdict`
  resolves and validates only the FILE part of the pointer, dropping the suffix via
  `sectionFilePart`; the heading names a location inside the document for a human reader, not
  part of the file's identity, so renaming a heading never breaks an already-gated pointer
  (`librarian/internal/modules/librarian/module.go:96-175` — `Verdict`, whose doc comment at
  `:116-121` states the advisory rule explicitly; `:203-214` — `sectionFilePart`).
- **Two forms fail closed, each with an actionable hint:**
  - A `://`-scheme-bearing pointer (a URL) is refused outright: `"pointer %q is not a desk
    file; a document gate needs a file path"` (`module.go:123-124`).
  - A `#`-anchored pointer (the markdown convention, e.g. `file.md#heading`) is **not**
    stripped — only `§` delimits a section anchor — and fails closed, naming the supported `§`
    form instead of leaving a bare not-found (`module.go:136-138`).
- **Pinned by tests:** `TestVerdict_ToleratesSectionAnchorSuffix`
  (`librarian/internal/modules/librarian/module_test.go:198-258` — an absent heading still
  passes, a genuinely missing file still fails, a URL with a section anchor still fails) and
  `TestVerdict_HashAnchorNotStripped` (`module_test.go:260-292` — a `#`-anchored pointer fails
  closed and the failure names `§`).

**Not the same question.** The gate config's `DocRequirement.Pointer` **selector**
(`librarian/internal/modules/pm/gates/gates.go:25`; values `""`/`"item"` — the default, resolve
via the item's own `pointer` field — or `"note:<key>"`) answers *which document* a gate reads.
The `items.pointer` field grammar defined above answers *where that document lives on disk*. A
reader must not conflate "which document" with "where the document lives" — they are distinct
questions answered by distinct fields in distinct collections.

No code change: `Verdict`/`sectionFilePart` (`module.go`) and `DocRequirement`/
`validateDocRequirement` (`gates.go`) are cited above, not modified — ADR 0010 ratifies their
existing, test-pinned behavior as-is.

Issue and URL references are **not** gate pointers under this grammar — they are a
cross-reference, typed per ADR 0011 (§7 R6.1; `docs/decisions/0011-typed-reference-contract.md`).
This section states only that boundary; the typed-reference contract itself lives in
`schema/references.yaml` — a `{kind, target}` primitive with a closed `kind` enum (`issue`,
`url`) and a validation guard in both lanes — where the qualifier is documented as read-time
resolved from `profile.repos.shorthand.issue_default` and never persisted. No `items.pointer`
field migrates onto that shape in this cycle; today's pointer resolution is unchanged.

### 3.2 The rigid state machine (R2.2)

A small, code-enforced machine, deliberately separate from `status_label`:

```
        ┌─────────┐   advance    ┌──────┐   advance    ┌────────┐   advance   ┌──────────┐
        │  queue  │ ───────────► │ work │ ───────────► │ review │ ──────────► │ terminal │
        └─────────┘              └──────┘              └────────┘             └──────────┘
             ▲                      │  ▲                   │  ▲                     │
             └──────────────────────┴──┴───────────────────┴──┘  reopen (review/terminal → work)
                                    demote (review → work, work → queue)

   blocked  is an orthogonal BOOL, not a phase. block(item) sets blocked=true and records the
   restore phase; unblock(item) clears it and restores the item to the phase it held. An item
   can be blocked in any non-terminal phase.
```

- **Legal phase transitions**: `queue→work`, `work→review`, `review→terminal` (advance);
  `review→work`, `work→queue` (demote); `review→work`, `terminal→work` (reopen). All others are
  refused by the machine before gates are even consulted.
- **`blocked`** is a side-state (R2.2): setting it preserves the current phase and stores a
  `restore_phase`; clearing it returns the item to `restore_phase`. Advance is refused while
  blocked.
- **Gates bind edges per the gate config** (§4): a transition is admitted by the machine, then
  the gate engine evaluates whatever requirements the desk's gate config binds to that edge;
  either can refuse. **By default only forward (advance) edges carry gates**; `reopen`/`demote`
  are legal edges with **no gate unless the config binds one** (walking work backward is
  ungated by default).

The machine table lives in `modules/pm/statemachine` as data-in-code; extending it (e.g. a
fast-track edge, R1.4) is a code change, not a config change — deliberate (§9).

### 3.3 `status_label` — the friendly vocabulary (R2.2)

Concrete default label set, seeded into `desk_config` and freely editable per desk (this spec
takes the choice; §13). Each label maps to exactly one phase; identity-neutral:

| Phase | Default labels |
|---|---|
| queue | `backlog`, `next` |
| work | `active` |
| review | `in-review` |
| terminal | `done`, `dropped`, `superseded` |
| (blocked flag) | surfaced as `blocked` / `waiting` regardless of phase |

Setting a `status_label` that maps to a different phase than the item's current `phase` is a
transition request routed through the machine + gates, not a free field write — the label and
the machine cannot drift.

### 3.4 `dependencies` — typed edges (R2.4)

One row per directed dependency edge.

| Field | Type | Notes |
|---|---|---|
| `from` | relation → items | |
| `to` | relation → items | |
| `kind` | select | `blocks`, `is-blocked-by`, `relates-to` (stored canonically as `blocks`; see below) |
| `unblock_at` | select | the phase of the blocker at which the block releases: `work`,`review`,`terminal` (R2.4) |
| `cascade` | select | `auto`, `manual`, `auto-reopen`, `permanent` (R2.4; §3.5) |
| `desk` | text | scoping |

`is-blocked-by` is stored as the inverse `blocks` edge (canonical direction: the blocker is
`from`) so the graph has one representation; the surfaces present both directions.
`relates-to` is a non-gating informational link.

### 3.5 Cascade semantics (R2.4)

For a gating edge where **A blocks B** with `unblock_at = P`:

- **auto** — when A reaches phase P, B's `blocked` flag auto-clears (restores B to its stored
  phase). One-shot tasks. Once cleared, later A movement does not re-block B.
- **auto-reopen** — like auto, but if A later regresses below P (reopen/demote), B is
  automatically re-blocked. This is the **standing-workstream** semantics (R2.4): B stays gated on
  A's ongoing state.
- **manual** — reaching P surfaces B as *unblockable* but a human/agent must clear it explicitly.
- **permanent** — the edge never auto-clears; B is gated on A for the item's life (a hard
  structural dependency); clearing requires deleting the edge.

Cascade evaluation is driven by the transition hook (§5.4): every phase change on A scans A's
outgoing gating edges and applies the rule to each B.

### 3.6 `transitions` — append-only audit (R2.5) + concurrency (R2.6)

`transitions` is append-only; nothing mutates or deletes a row.

| Field | Type | Notes |
|---|---|---|
| `item` | relation → items | |
| `from_phase`,`to_phase` | text | (or `block`/`unblock`/`claim`/`gate_refused` event kinds) |
| `event` | select | `advance`,`demote`,`reopen`,`block`,`unblock`,`claim`,`release`,`gate_refused` |
| `actor` | text | who: a human handle or an agent id (R2.5) |
| `actor_kind` | select | `human`,`agent` |
| `delegation_parent` | text | the parent agent/session id when an agent acted under delegation (R2.5) |
| `detail` | text | e.g. the gate refusal message, for the audit trail |
| `created` | autodate | |

No crypto identity layer (R2.5) — `actor` is a plain string sourced from the calling surface
(the CLI passes `$USER` or a `--actor`; the MCP/agent surface passes its agent id + delegation
parent).

**Optimistic concurrency + claim (R2.6, SHOULD — adopted):** every mutating PM tool takes the
item's `version` and refuses on mismatch (`409`-style, "item changed since you read it"). A
**claim** sets `claimed_by` + `claim_expires` (default TTL 30 min, `PM_CLAIM_TTL` configurable);
`advance`/`demote` on a live, foreign claim is refused. An expired claim is treated as free. This
is the "enough to stop two desk agents double-working an item" bar (R2.6), not a distributed lock.

### 3.7 `notes` — lighter artifacts (R3.2)

Phase-scoped keyed notes on an item, for the cases where a full validated document is overkill
(R3.2 "notes remain available as lighter artifacts"). A note is *not* a gate artifact by default;
a gate rule may require a note key exists (a cheaper gate than a document) but the spine gate is
the document (§4).

| Field | Type | Notes |
|---|---|---|
| `item` | relation → items | |
| `phase` | text | the phase this note belongs to |
| `key` | text | e.g. `rationale`, `handoff` |
| `body` | text | |
| `actor`,`actor_kind` | text/select | |
| `created` | autodate | |

### 3.8 `desk_config` — per-desk workflow config (R3.3, R5.2)

One row per desk (the desk-scoped config layer, R5.2). Holds the editable gate rules (§4),
the `status_label` vocabulary (§3.3), the claim TTL override, and the module-enabled flag. The
YAML the human edits is stored as a `rules` text/json field; the loader validates it against a
schema (§4.3) at write time and fails loud on a malformed config.

---

## 4. Gates — the spine (R3)

The gate engine is the distinctive move: a phase advance is refused unless the target phase's
required **documents** exist and validate (R3.1, R3.2). Gates are **server-enforced**, never
prompt-dependent (R3.1).

### 4.1 Enforcement point (R3.1)

The gate check lives in the PM module's transition path — the single code path every surface
(MCP/CLI/TUI) routes through — *not* in a prompt or an agent instruction. **One generic
transition tool serves every legal edge**: `transition_item(item, target_phase)` (advance,
demote, and reopen are all requests through this one tool; the machine derives the edge kind
from current → target phase). Sequence:

1. The state machine admits the edge, else refuse ("no legal transition `X→Y`").
2. If `item.blocked` and the edge is a forward (advance) edge, refuse ("item is blocked:
   `<reason>`").
3. If a live foreign claim exists, refuse (R2.6).
4. The gate engine looks up the requirements the desk's gate config binds to
   `(item.type, edge)` (§4.2) — **forward edges by default; demote/reopen carry no gate unless
   the config binds one** — calls `DocumentValidator.Verdict(...)` per required artifact (§2.5),
   and refuses with the **exact list of what is missing** if any verdict is unsatisfied (R3.1).
   Example refusal: *"cannot advance decision item to terminal: required document
   (type=decision, status=accepted) at `_structure/decisions/0021-x.md` is at status
   `proposed`, needs `accepted`."* An edge with no bound gate passes this step trivially.
5. On success: write the new phase, append a `transitions` row, run the cascade scan (§3.5), emit
   the realtime event (§5.4).

A refusal is recorded as a `gate_refused` transition row (audit; §3.6) so refusals are
observable, not silent.

### 4.2 Gate rules as editable YAML (R3.3)

Gate rules live in `desk_config.rules` (per-desk, editable), not in code (R3.3). Concrete schema
this spec adopts (§13):

```yaml
schema_version: 1
gates:
  # keyed by item type (schema-v1 / kit type) → transition → requirements
  decision:
    "review->terminal":
      documents:
        - type: decision          # a schema-v1 / kit type (R3.4)
          status: accepted         # required frontmatter status; omit = existence + validity only
          pointer: item            # resolve the doc via item.pointer (default) | note:<key>
  task:
    "work->review":
      documents:
        - type: task
          status: active
traits:
  # cross-cutting rules composed onto any item matching a predicate (R3.3, borrow #9)
  - name: governs-desk-operations
    match: { field: governs, equals: desk-operations }
    on: "review->terminal"
    documents:
      - type: decision
        status: accepted
```

- **Per-type rules** map `(type, transition)` → required documents.
- **Traits** compose cross-cutting requirements by matching an item field (or its pointed doc's
  frontmatter), so a rule like "anything that governs desk operations needs an accepted decision"
  is written once (R3.3). A trait's `match` predicate may reference either a **first-class item
  field** or a **frontmatter field of the item's pointed document** — as `governs` above, which
  is not an `items` field but doc frontmatter, resolved through the validation seam (§2.5), never
  by reading librarian collections.
- Effective requirements for a transition = the per-type rule ∪ every matching trait.
- The loader validates this YAML against the gate-config schema on write; an invalid config is
  rejected (fail-loud, R7 discipline) rather than silently disabling gates.

### 4.3 Kit/type ids as the reference vocabulary (R3.4)

The `type` and `status` values in gate rules reference the **schema-v1 / SOP-kit ids** produced
by the kit port (`#49`, D1). The kit port ships `kits.yaml` (the manifest of ported kit types)
and reconciles the schema-v1 `types:` list (adding the 7 known-missing types or recording their
deliberate exclusion — see `#49` D3). This spec **depends on** that vocabulary; the gate engine
validates, at `desk_config` write time, that every `type`/`status` referenced by a gate rule is a
known kit/schema type, and refuses unknown ids (so a gate can never reference a non-existent
document type). D1 therefore precedes D3 in the build order (§11); if D1 slips, the gate engine
still functions but the desk's `desk_config` can only reference the base schema-v1 types.

Since ADR 0012, `create_item` enforces the identical `KnownType` check at item birth, closing
the create-vs-gate-config asymmetry this section documents on the config side: a gate rule could
never reference an unknown type, but an item could still be born with one until that change.

### 4.4 What a document verdict means (R3.2)

Via the seam (§2.5), a document is "filled" for a gate when: it **exists** at the pointer, its
**frontmatter validates** against schema v1 for its `type`, and it carries the **required
status**. The librarian's existing validation (the same engine behind its patrol R-rules and the
plugin's `profile_validate`) is the single source of the verdict — the PM system adds no second,
divergent notion of document validity.

---

## 5. Surfaces (R4)

### 5.1 The PM tool family on the shared tool core (R4.1)

The same three surfaces as the librarian — MCP (agents), CLI (scripts/owner), chat TUI (console)
— over one tool core (R4.1, §2.6). The v1 PM tool set:

| Tool | Writes | Purpose |
|---|---|---|
| `get_context` | no | single-call cold-start briefing (§5.2, R4.2) |
| `list_items` | no | filtered graph query (by phase, court, type, blocked, parent) |
| `get_item` | no | one item + its notes, deps, recent transitions, ancestor chain |
| `create_item` | yes | add a work item to the graph |
| `update_item` | yes | edit first-class fields (version-checked; R2.6) |
| `transition_item` | yes | request any legal phase transition — advance/demote/reopen — via `(item, target_phase)` (runs the machine + gates; §4.1) |
| `block_item` / `unblock_item` | yes | set/clear the blocked side-state |
| `add_note` | yes | attach a phase-scoped note (§3.7) |
| `link_items` | yes | create a typed dependency edge (§3.4) |
| `claim_item` / `release_item` | yes | claim TTL for multi-agent safety (R2.6) |

Write tools carry the `WritesFiles=false` flag semantics adapted to "mutates the store" — the
§5.4 librarian write gate governs *desk-file* writes; PM tools write only the store, so they are
not behind `LIBRARIAN_AUTONOMOUS_WRITES`. The PM module MAY define its own gate for autonomous
transitions if a desk wants agents read-only over the graph (a `PM_AUTONOMOUS_WRITES` flag,
mirroring the librarian pattern) — this spec ships it defaulting **on** for PM writes (agents are
expected to drive the graph) while keeping `transition_item`'s document gates as the real
safety (§13).

See [`docs/agent-integration-contract-v1-spec.md`](agent-integration-contract-v1-spec.md) for
the full cross-product agent-integration contract: this write-gate asymmetry is one of its five
named parameters, ratified as policy rather than drift to reconcile against the librarian.

### 5.2 `get_context` — the cold-start briefing (R4.2)

One call returns the desk's working state, replacing the hand-maintained handoff §1 threads index
(R4.2, R7.3). Response shape:

```json
{
  "desk": "<desk>",
  "generated_at": "<ts>",
  "active":  [ { item summary, court, phase, status_label, pointer } ],
  "blocked": [ { item summary, blocked_reason, blocking_items[] } ],
  "stalled": [ { item summary, days_since_last_transition } ],
  "recent_transitions": [ { item, event, from, to, actor, at } ],
  "ancestors": { "<itemId>": [ root..parent chain ] },
  "counts": { "by_phase": {...}, "by_court": {...} }
}
```

- **active** = non-terminal, non-blocked items, ordered by `(court, priority)`.
- **blocked** = `blocked=true`, each with its blocking items resolved via the dependency graph.
- **stalled** = non-terminal items whose last `transitions` row is older than a threshold
  (default 14 days; configurable).
- All scoped to the desk (R4.2). This is the single query the session-open briefing and the TUI
  landing view both call.

### 5.3 CLI + TUI

- **CLI**: a `pm` command group (`deskkit pm context`, `pm transition <id> --to review`,
  `pm list --court owner`, …). The owner/script surface. `--actor` sets the audit actor;
  `--json` is the default machine contract (mirrors the librarian tools' JSON-first output).
- **TUI**: PM views mounted into the shared Bubble Tea TUI (ADR 0004) via `Module.TUIViews()` — a
  graph/queue view, an item detail view, and the `get_context` landing view. The console surface
  for a human working the desk interactively.

### 5.4 Realtime events (R4.3, SHOULD — adopted)

PocketBase's native realtime is used to emit an event on every `transitions` write (advance,
block/unblock, claim, gate_refused) and on dependency-driven cascades. Consumers: a watching
agent that wants to wake on a state change, or a dashboard. The PM module satisfies an optional
`RealtimeSource` capability (§2.4) that core wires to PB's realtime subsystem under `serve`.
Because this is PB-native, the cost is a thin hook, not a new transport — hence adopting the
SHOULD (R4.3). Realtime is serve-only (like the librarian's trigger layer); one-shot CLI calls
emit no events.

---

## 6. The complementary plugin (R5.1, R5.3)

Per A2/R5.1, the agent-facing pieces ship as a **separate, complementary plugin in the same
repo** — distinct from the existing `desk-standard` plugin, sharing the marketplace. The
distribution layer is separate (A2); only the data/runtime layer is unified (R5.5).

### 6.1 Plugin contents

- **PM workflow skills** — e.g. a session-open skill that calls `get_context` and renders the
  briefing; an "advance an item" skill that surfaces the gate refusal and points the agent at the
  document it must produce; a triage skill.
- **Agent defs** — a PM-operator agent scoped to the PM tool family.
- **Hooks** — optional session-start hook that runs the cold-start briefing.
- **`.mcp.json`** pointing at the binary's `pm` MCP surface (the same one-binary MCP server,
  exposing the PM tool family when `PM_ENABLED`).

The plugin is registered in `.claude-plugin/marketplace.json` as a second entry (shipped as
`desk-pm`), following the existing `desk-standard` plugin's structure
(`plugin/` TS lane, `claude-plugin/` bundle).

### 6.2 Identity neutrality (R5.3)

All shipped plugin artifacts and all module-shipped strings (MCP server name, seeded prompts,
default `desk_config`, skill text) are **identity-neutral** — no personal names, orgs, repos,
issue numbers, or private paths (R5.3, decision 0013 item 9). Personalization is via the
profile / `_knowledge/` data surfaces the librarian already reads (config layering, §2.9). The
repo's existing **neutrality lint** (`scripts/check-neutrality.mjs`, run first in CI) extends to
cover the new plugin tree and the PM module's shipped strings; a hardcoded identity fails the
build (the D8 gate that already exists).

---

## 7. Boundaries (R6)

- **R6.1 — the GitHub board stays the single work-state truth for code-repo work.** The PM system
  never forks a copy of board items. An item MAY reference a GitHub issue — but per ADR 0010
  an issue URL is **not** a gate `pointer` (the pointer grammar is a desk-relative file path;
  gates fail `://` closed): it is a cross-reference, typed per ADR 0011. Any GitHub
  integration is a **read-only mirror/linkage** (display the issue's state beside the item),
  never a write-back or a parallel backlog. v1 ships no GitHub connector at all — the
  reference is a plain string until ADR 0011's contract migration lands; a read-only
  enrichment is LATER.
- **R6.2 — the librarian write boundary carries over.** The PM system writes only its own store
  collections; it never writes desk files. When a gate needs a document produced, that document is
  authored through the normal (human or librarian-supervised) path, under the librarian's binding-doc
  flag-only + record-original-first discipline (decision 0014). The gate reads verdicts; it never
  writes documents.
- **R6.3 — secrets never in the store (SHOULD — adopted).** The PM store holds pointers, never
  secret values; `_meta/secrets/` stays the secrets home. The `Config` struct already carries
  nothing secret (verified), and PM adds no secret-bearing field.

---

## 8. Versioning, migration, ops (R7)

### 8.1 Adoption path for a desk (R7.3)

The generic adoption sequence a desk follows to turn the PM module on and seed it (described
identity-neutrally — the doc ships neutral):

1. **Enable** the module (`PM_ENABLED=true` or the profile key; §2.9). On next `serve`/`migrate`,
   the PM migrations run and stamp `module_schema_versions`.
2. **Seed `desk_config`** — write the gate rules (§4.2) and the `status_label` vocabulary for the
   desk's types.
3. **Import the desk's existing work surfaces** as the first dataset: the handoff threads index
   rows, any standing plans/task rows, each becoming an `items` row with its `court`, `type`,
   `pointer`, and phase. A one-time import script (or the `create_item` tool driven from a
   manifest) does this; the import is idempotent and desk-scoped.
4. **Retire the old surfaces to pointers** — the flat threads index becomes a pointer to
   `get_context`; the plans/tasks docs collapse to pointers, matching the desk's graduation
   discipline.
5. The **live desk is never written during a dry-run** (D8, §10); import into a scratch store
   first, observe `get_context`, then adopt.

### 8.2 Rebuild-from-scratch reproducibility (R7.2)

The librarian's verify-gate discipline extends to the PM collections: **the store is derivable
from the file/doc layer + config.** Stable `pbc_…` collection ids are preserved across rebuilds
(as the librarian already does), migrations are deterministic, and the import (§8.1) is a pure
function of the desk's documents + `desk_config`. A rebuild = fresh store → run migrations → run
the import → identical graph. The `module_schema_versions` meta collection makes the target
schema state explicit and checkable.

The D2b rename (§2.10) moves the store *home* but never its contents' semantics: the automatic
path migration (old `pocket-librarian/<DESK>/` → new home, one logged line) is a directory
move, and a post-move rebuild-from-scratch at the new home must produce the identical store —
the reproducibility gate runs against the renamed layout too.

### 8.3 Version + migration discipline (R7.1)

- Explicit schema versioning per module via `module_schema_versions` (§2.8). A desk on version X
  is not expected to conform to X+1 without running the migration step — "building the bus while
  driving it" is the operating reality (R7.1, C2 comment).
- Migrations are strictly additive/ordered per module; a binary refuses to serve a store whose
  module version is *ahead* of the binary (§2.8).
- Product version + CHANGELOG follow the repo's existing policy (ADR 0005); the version number
  for this build is confirmed with the owner at bump time (the build order flags 0.6.0 as
  presumptively the owner's next number — do not assume).

---

## 9. LATER — designed-for, not built (R1.3, R1.4, R4.4)

Each LATER item is named so the v1 design does not preclude it, and the constraint it imposes on
v1 is stated.

- **R1.4 — agent-assisted fast-track mode.** A sanctioned way to move an item through phases
  quickly with an agent back-filling the artifacts. **Constraint on v1:** the state machine
  (§3.2) must keep the gate check as a *separate step from the phase write* (it is — §4.1 step 4),
  so a future privileged `fast_track` edge can advance the phase and enqueue the artifact
  back-fill without a machine redesign. The transition audit (§3.6) already distinguishes actor
  kinds, so a fast-track transition is attributable. Not built: no fast-track edge, no artifact
  back-fill automation.
- **R4.4 — autonomous queue-drain (Ralph pattern).** Fresh agent per item, circuit breakers.
  **Constraint on v1:** the claim + TTL (§3.6) and the append-only transitions (§3.6) must be
  sufficient to let an external driver pick the next unclaimed, unblocked item, claim it, and
  record its work — they are. `get_context`'s `active`/`stalled` sets are the queue a drainer
  would read. Not built: no driver loop, no worktree/circuit-breaker orchestration.
- **R1.3 — minimal-first.** The whole PM module is feature-gated off by default (§2.9) so v1 can
  ship in the binary and prove out on one desk before any desk is required to adopt it.
- **R5.2 (partial) — portfolio read-only fan-out.** The cross-desk read-only portfolio view is
  **explicitly LATER** (ADR 0002 itself frames it as "if one ever materializes"). **Constraint on
  v1:** the store layout keeps all desks enumerable under one XDG application root
  (`$XDG_DATA_HOME/<app>/<DESK>/` — one subdirectory per desk), so a read-only fan-out reader
  can later be added by iterating that root, without any schema or layout change. Not built: no
  fan-out surface, no cross-desk query.

Also parked (open question in requirements, not a requirement): the ruling-form pattern (HTML Q/A
+ markdown export) as a first-class workflow surface. v1 leaves room via `notes` + `pointer`
(a ruling form is an artifact an item can point at) but ships no form surface.

---

## 10. Test plan (D6)

The build is done when the repo's own hard gates pass (not when an agent says so). Test lanes:

- **10.1 Gate red-ability (R3.1, R3.2).** For each gate kind: construct an item whose target-phase
  document is absent / invalid-frontmatter / wrong-status, assert `advance` is refused and the
  refusal names exactly what is missing; then satisfy the document and assert the advance
  succeeds. A gate that cannot be made to fail is not proven.
- **10.2 State-machine legality (R2.2).** Every illegal edge is refused by the machine before
  gates; `blocked` blocks advance; demote/reopen pass ungated by default and are gated when
  (and only when) the desk's config binds a gate to them.
- **10.3 Cascade (R2.4).** auto clears once; auto-reopen re-blocks on regression; manual surfaces
  but does not auto-clear; permanent never clears. `unblock_at` releases at the named phase, not
  before.
- **10.4 Concurrency (R2.6).** Version mismatch is refused; a live foreign claim blocks advance;
  an expired claim is treated as free.
- **10.5 The narrow seam (R5.5a).** A build-time check (import-graph test) asserts `modules/pm`
  does not import `modules/librarian`; a unit test drives the gate through a *stub*
  `DocumentValidator` to prove the PM module needs no librarian internals.
- **10.6 Module-scoped migration (R7.1).** A store at the librarian's version N with the PM module
  at version M runs the librarian unaffected; enabling PM applies only PM migrations and stamps
  its own meta row; a store ahead of the binary is refused.
- **10.7 Zero-behavior-change (D2 gate).** The librarian's existing `verify.sh` Phase-1 gate and
  the full Go + plugin suites pass unchanged after the refactor, with all stable collection ids
  preserved.
- **10.8 Rebuild reproducibility (R7.2).** Fresh store → migrate → import → assert the graph
  matches a golden snapshot; a second rebuild is identical.
- **10.9 Neutrality (R5.3).** The neutrality lint (extended to the PM module + plugin) is green,
  and its self-test (seeded-violation detection) still red-ables.
- **10.10 Surface parity (R4.1).** The same `get_context` result is reachable via CLI, MCP, and
  the TUI landing view (one core, three surfaces).

All lanes fold into `make check` / `make test` / the librarian `verify` gate and the single CI
`ci` job (the existing aggregate required check).

---

## 11. Build slices (D2–D8)

Deliverables, criteria, and parallelism — no timelines. D1 (the kit port, `#49`) runs in
parallel and gates only the gate-vocabulary reconciliation (§4.3).

| Slice | Deliverable | Done when | Parallelism |
|---|---|---|---|
| **D1** (`#49`) | Kit port + schema-v1 reconciliation | 23 kits ported, neutrality-lint green, `kits.yaml` drift-guard red-able, every schema-gap disposition recorded | Parallel with D2; gates §4.3 vocabulary |
| **D2** | Core + module refactor (R5.5) — NO rename: `cmd/pocket-librarian` + `verify.sh` untouched; migrates all SEVEN librarian tools (incl. `record_feedback`) and corrects the codebase's stale "six tools" comments during the move | §2.7 zero-behavior-change gate green (existing verify + full suite unchanged, stable ids preserved); `pm` registered but disabled | Serial foundation — blocks D2b, D3–D5 |
| **D2b** | Chassis rename (§2.10): binary → `deskkit`, store home → `$XDG_DATA_HOME/deskkit/<DESK>/` with automatic path migration; verify.sh + Makefiles updated in the same commit; ADR 0002 dated correction callout in the same PR | Full suite + verify gate green AFTER the rename; old-store auto-migration observed (one logged line) | Immediately after the D2 gate is demonstrated green; before D4's surface docs freeze names |
| **D3** | PM module: collections, machine, cascade, concurrency, gate engine (R2, R3) | §10.1–10.6 green; gates refuse naming what is missing | After D2; consumes D1 vocabulary for §4.3 |
| **D4** | Surfaces: PM tool family on the shared core → MCP/CLI/TUI, `get_context`, realtime (R4) | §10.10 parity green; `get_context` returns the four sets; realtime emits on transitions | After D3 (needs the tools' core functions) |
| **D5** | Complementary plugin: skills, agent defs, hooks, `.mcp.json` (R5.1, R5.3) | Plugin loads; neutrality green; a session-open skill renders the briefing | After D4 (needs the MCP surface); parallel with D6 |
| **D6** | Tests: all §10 lanes | Full suite + `make verify` + `ci` green at tip; every gate red-able | Interleaved with D3–D5 (tests land with their slice) |
| **D7** | Docs: this spec (first), ADRs (R5.5 architecture; kit-port dispositions), README + CHANGELOG, per-surface usage | Docs merged; ADRs cite the code they bind | Parallel; ADRs land with D2/D3 |
| **D8** | Adoption dry-run (R7.3) | Scratch store seeded from desk thread data; `get_context` cold-start observed; one gate refused-then-satisfied; one dependency auto-unblock; live desk never written | Last; foreman-observed |

Shared-file discipline: `core/toolcore`, `core/module`, and `main` are shared seams — serialize
edits to them (one owner per wave), disjoint-scope the module trees.

---

## 12. Requirements-traceability table

Every requirement id → the section that covers it (or its LATER disposition), and the named
test that verifies the shipped implementation. This table is the coverage contract for D6's
requirements-coverage acceptance criterion. Unqualified `Test…` names are Go tests under
`librarian/internal/`; the tree home is given where it is not the PM module engine. Rows whose
disposition is LATER, or which are architectural/doctrine (build- or policy-enforced rather
than unit-tested), name the enforcing artifact and are called out in the note below the table.

| Req | MUST/SHOULD/LATER | Covered by | Verified by (test) |
|---|---|---|---|
| R1.1 serve concurrent non-issue threads | MUST | §1.1, §3.1 | `TestFixtures_ThroughBothSurfaces`, `TestRunner_ReplaysImportedItems` (`.../pm/scenario`) |
| R1.2 no external PM SaaS | MUST | §1.1, §2 (self-hosted on the chassis) | Architectural — embedded PocketBase, no network dep (build/`verify.sh`); no unit test |
| R1.3 minimal provable v1 | MUST | §1.2, §2.9 (feature gate), §9 | `TestEnabled_Gate` (`.../pm`), `TestAdoptionDryRun` (`.../pm/scenario`) |
| R1.4 fast-track mode | LATER | §9 (constraint: gate separate from phase write) | LATER — deferred; not built, no test |
| R2.1 universal self-referencing item | MUST | §3.1 | `TestGetItem_Detail`, `TestImport_BuildsTheGraph` (`.../pm/importer`) |
| R2.2 rigid machine ⟂ status labels | MUST | §3.2, §3.3 | `TestEdge_Legality`, `TestDefaultStatusLabels` (`.../pm/statemachine`), `TestSetStatusLabelRoutesThroughMachine` |
| R2.3 court/pointer/type/severity/priority | MUST | §3.1 | `TestListItems_Filters`, `TestGetItem_Detail` |
| R2.4 typed edges, unblock_at, cascade | MUST | §3.4, §3.5 | `TestCascadeAuto`, `TestCascadeAutoReopen`, `TestCascadeManualAndPermanent`, `TestCascadeMultiBlocker`, `TestLinkIsBlockedByCanonicalizes` |
| R2.5 append-only audit + actor attribution | MUST | §3.6 | `TestAuditTrail` |
| R2.6 optimistic concurrency + claim TTL | SHOULD → adopted | §3.6 | `TestVersionMismatchRefused`, `TestClaimSemantics`, `TestClaimTTLFromDeskConfig`, `TestReleaseClearsClaim` |
| R3.1 server-enforced transitions, names what's missing | MUST | §4.1 | `TestGateRefusedThenSatisfied`, `TestIllegalEdgeRefusedBeforeGates`, `TestBlockedRefusesAdvanceOnly` |
| R3.2 artifacts = librarian-validated documents | MUST | §4.4, §2.5; notes §3.7 | `TestGateFailsClosedWithoutValidator`, `TestEvaluate_StubValidator` (`.../pm/gates`) |
| R3.3 gate rules in editable per-desk YAML + traits | MUST | §4.2 | `TestDeskConfigOverridesDefaults`, `TestTraitCompositionThroughFrontmatter`, `TestParseRules_SpecExample`, `TestEffective_TraitComposition` (`.../pm/gates`) |
| R3.4 kit/type ids as reference vocabulary | MUST | §4.3 (depends on D1/#49) | `TestParseRules_Refuses` (rejects unknown type/status/edge), `TestDefaultRulesYAML_Parses` (`.../pm/gates`) |
| R4.1 MCP+CLI+TUI on one tool core | MUST | §5.1, §5.3, §2.6 | `TestGetContext_SurfaceParity`, `TestToolBodies_EndToEnd` (`.../pm/tools`) |
| R4.2 get_context single-call cold-start | MUST | §5.2 | `GetContext` (`.../pm/engine/queries.go`) via `TestGetContext_FourSets`, `TestGetContext_ActiveOrdering`; cold-start observed by `TestAdoptionDryRun` |
| R4.3 realtime events | SHOULD → adopted | §5.4 | `TestRealtime_EmitsOnTransitions` (`.../pm`) |
| R4.4 autonomous queue-drain | LATER | §9 (constraint: claim + append-only sufficient) | LATER — deferred; not built, no test |
| R5.1 in-repo complementary plugin + PM module | MUST | §6, §2.3 | `plugin/desk-pm.test.ts` (skills/agent/tool-reference suite), `TestGatedOnDeskHasPMSurfaces` (`.../pm/gatedon`) |
| R5.2 store-per-desk, config layering, open-guard | MUST | §2.1, §2.9, §3.1 desk field, §7 — built; **portfolio read-only fan-out: LATER (§9)**, with the v1 constraint that all desks stay enumerable under one XDG root so fan-out needs no schema/layout change | `TestStoreDir_EmbedsDeskName` (`.../core/store`), `TestCheckDeskGuard_MismatchOnFilesRowErrors` (`.../core/store`), `TestLoadDotEnvNeverOverrides` (`.../core/config`) |
| R5.3 identity-neutral artifacts | MUST | §6.2 | `scripts/check-neutrality.mjs` lint (in `make check` + CI); `plugin/desk-pm.test.ts` neutrality assertions |
| R5.4 single-binary posture | SHOULD → adopted | §2 (one binary, embedded PB) | Architectural — release cross-compile + `verify.sh`; no unit test |
| R5.5 unified app+store, core+modules, 3 disciplines | MUST | §2 (whole section) | `TestNoLibrarianImports`, `TestNoSelfRegisteredMigrations`, `TestMigrations_MatchOwnedCollections` (`.../pm`); `scripts/check-core-purity.mjs` |
| R6.1 GitHub board stays truth; read-only linkage | MUST | §7 | `TestSpecs_NoDeskFileWrites` (`.../pm/tools`); board-linkage otherwise doctrine (§7), no automated test |
| R6.2 librarian write boundary carries over | MUST | §7 | `TestSpecs_NoDeskFileWrites` (`.../pm/tools`) — PM tools write only the store, never desk files |
| R6.3 secrets never in store | SHOULD → adopted | §7 | Doctrine (§7) — no automated test; see note below |
| R7.1 explicit schema/config versioning + migration | MUST | §2.8, §8.3 | `TestGatedOnDeskCreatesPMCollectionsAndStamps`, `TestPMDownMigrationsReverse` (`.../pm/gatedon`) |
| R7.2 rebuild-from-scratch reproducibility | MUST | §8.2 | `TestRebuildReproducibility`, `TestDepSnapshotKindTiebreak` (`.../pm/importer/rebuild_test.go`) |
| R7.3 adoption path for the desk | SHOULD → adopted | §8.1 | `TestAdoptionDryRun` (`.../pm/scenario/dryrun_test.go`) |

**Coverage notes.** Every MUST/SHOULD requirement that is a built behavior has a named
verifying test above. Four rows are intentionally not unit-tested and are enforced elsewhere:
**R1.4** and **R4.4** are LATER (deferred, not built); **R1.2** and **R5.4** are architectural
(one binary with embedded PocketBase, no network dependency — verified by the release
cross-compile and `verify.sh`, not a unit test); and **R6.3** (secrets never in the store) is a
handling doctrine (§7) with no automated guard — a candidate for a future negative test. The
D8 adoption oracle `TestAdoptionDryRun` (§8.1) exercises R7.3 end-to-end and, in one run,
observes the R4.2 cold-start briefing, an R3.1 gate refused-then-satisfied, and an R2.4
dependency auto-unblock — never writing the live desk. R7.2 reproducibility is doubly pinned:
`TestRebuildReproducibility` plus `TestDepSnapshotKindTiebreak`, which fixes the (from,to)-tie
sort determinism (issue #71).

---

## 13. Design decisions taken by this spec

Where the requirements left a genuine design choice, this spec makes a concrete one. Each is
flagged here for the foreman/owner to review; none is claimed as ruled.

> **As shipped (v0.7.0, 2026-07-19):** every choice below shipped as-drafted — as the default
> or seed, not as a ruling. Item 1 (binary rename) shipped as slice **D2b** (#62), and the
> companion plugin (§6.1) shipped under the name **`desk-pm`**, resolving the "final name at
> build time" note. The **seeded default `desk_config` gate rules** (item 3 / §4.2) and the
> **default `status_label` vocabulary** (item 2) remain explicitly re-rulable by the owner from
> the exec desk — see `docs/pm-guide.md` and CHANGELOG `[0.7.0]`. Shipping a default did not
> convert it into a binding decision.

1. **Binary/chassis rename — approved, and sequenced OUT of D2.** The owner has approved
   renaming the unified binary to `deskkit` (it now serves two modules; a librarian-named binary
   misrepresents the chassis). The design decision this spec takes is the **sequencing and store
   handling**: the rename ships as its own slice D2b after the D2 zero-change gate (never inside
   D2, which would contradict the "verify.sh passes unchanged" gate), the store home renames with
   the binary, and an automatic old→new path migration plus an ADR 0002 dated correction callout
   ship with it (§2.10).
2. **Default `status_label` vocabulary** (§3.3): `backlog`/`next` (queue), `active` (work),
   `in-review` (review), `done`/`dropped`/`superseded` (terminal), `blocked`/`waiting` (flag).
   Small and neutral; fully editable per desk. The requirements left the exact set open.
3. **Gate-config YAML schema** (§4.2): the `gates:` (per-type→transition→documents) + `traits:`
   (predicate-matched cross-cutting) shape. Validated at write time against a config schema.
4. **`module_schema_versions` meta collection** (§2.8) as the per-module version of record.
5. **PM collections are unprefixed** (`items`, `notes`, …) per the requirement's literal naming,
   with collision prevented by the ownership guard (§2.4) rather than a `pm_` prefix. Alternative:
   prefix for defense-in-depth; rejected to match the requirement text and keep the surfaces
   readable.
6. **Cascade `auto` vs `auto-reopen`** given the concrete standing-vs-one-shot semantics in §3.5.
7. **Claim TTL default 30 min** (`PM_CLAIM_TTL` override), and version-token optimistic
   concurrency as the "light" concurrency bar (R2.6).
8. **`stalled` threshold default 14 days** in `get_context` (§5.2), configurable.
9. **PM writes default autonomous-on** (§5.1): the real safety is the document gate on `advance`,
   not a write flag, so agents may drive the graph by default; a `PM_AUTONOMOUS_WRITES=false` desk
   can make agents read-only. Alternative: default off (mirror the librarian's `apply_fix` gate);
   rejected because graph mutation is the PM system's whole point and it writes no desk files.
10. **Realtime (R4.3) and secrets/concurrency SHOULDs adopted**, not argued down — each is
    cheap on the existing chassis (PB-native realtime; the seam; the version token). The R5.2
    portfolio read-only fan-out is NOT adopted in v1 — it is LATER (§9), with the enumerable
    one-XDG-root layout as the designed-for constraint.

### Interpretations / deviations to report

- **R5.5b/c + R7.1 "a desk can lag one module's schema" / "run librarian-only"** are delivered as
  **feature-gated physical omission + a per-module logical version**, not as partial migration of
  a shared runner. When PM is disabled its migrations are not registered, so a librarian-only desk
  has no PM collections; the `module_schema_versions` row gives each module an independent version
  of record. This is the honest reading given PocketBase's global compile-time migration runner
  and the D2 zero-behavior-change gate — full partial-migration of one shared runner would fight
  both. See §2.8.
- **The current tool-core surfaces are not module-ready** (verified): `internal/agent`'s
  `buildTools` name-keyed builder map and `internal/mcp`'s `switch name` + `toolInputTypes` map
  are both hardcoded to the seven librarian tools. D2 must generalize them into `core/toolcore`
  (§2.6) before D4 can add PM tools — this is real refactor work, not just a file move, and is the
  main risk to the "pure move" framing of D2.
- **No conflict found** between the requirements and the codebase on the core chassis facts
  (store-per-desk, XDG, desk-guard, config layering, single binary, embedded PB) — all exist and
  the PM module reuses them as-is.

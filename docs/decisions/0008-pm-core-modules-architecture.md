_ADR for the core + modules architecture (ruling R5.5) the PM system introduces: one binary/store
per desk, split into a shared core and compile-time Go modules bound by three disciplines — a
narrow validation seam, module-scoped migrations, and per-desk feature gating._
Status: Accepted — 2026-07-19 (amended 2026-07-21 — PM default-on; see Amendments)

# 0008 — PM system architecture: core + compile-time modules (R5.5)

> **Amendment (2026-07-21) — PM ships default-on for 1.0.** Decision leg §5 ("Per-desk feature
> gating") originally set the config-layering default to **off**: `PM_ENABLED` env > profile
> `modules.pm.enabled` > default off — the PM module rode dark in the binary so v1 could prove out
> on one desk before any desk adopted it (R1.3, spec §2.9). Having proved out, the **owner ruled
> the default leg flipped ON** for the 1.0 maturity milestone (decision-queue sign-off batch,
> item `pm` = `flip-on-bless-defaults`, `_meta/signoff/2026-07-21-decision-queue/answers.json`).
> The same ruling blessed the shipped gate-rule seed (spec §4.2) and `status_label` vocabulary
> (spec §3.3) as the defaults every fresh desk now receives. **What changed:** only the default
> leg — env and profile still override, so `PM_ENABLED=false` or `modules.pm.enabled: false`
> cleanly runs a desk librarian-only. Everything else in this ADR (the seam, module-scoped
> migrations, physical-omission-when-disabled, the versioning discipline) stands unchanged; the
> feature gate's machinery is exactly as designed — only its default answer moved. The original §5
> reasoning is retained below for provenance.

## Context

The librarian shipped as a single Go module with a flat `internal/` tree: the tool registry,
migrations, MCP/CLI/TUI bindings, and store code were all librarian-specific. The PM system
(document-gated work graph, `docs/development/specs/pm-system-v1-spec.md`) had to live **in the same binary and the
same per-desk store** — ruling R5.5 rejected a second service, a shared multi-desk database, and
any runtime plugin loading. But the two domains must stay *extractable*: the PM module must not
reach into the librarian's collections, and a desk must be able to run librarian-only.

Three forces shaped the design:

1. **One process, one store, one desk.** ADR 0002 already fixed the store-per-desk topology; the
   PM collections join the librarian's in that same PocketBase store. No IPC, no second daemon.
2. **Domains coupled only through documents.** A PM gate needs to know whether a required
   document (e.g. an `accepted` decision) exists and validates — nothing more. That is a narrow
   contract, not a dependency on the librarian's internals.
3. **Ship dark, prove on one desk (R1.3).** The PM module rides in the binary release but stays
   inert until a desk opts in, so v1 can prove out before any desk is required to adopt it.

The refactor (D2), the chassis rename (D2b, ADR 0002 corrected in place), and the PM module +
surfaces + plugin (D3–D5) implement this shape; this ADR records the architecture they bind to
and cites the code that enforces each discipline.

## Decision

**1. Core + compile-time modules.** `librarian/internal/` splits into `core/` (the shared
substrate) and `modules/` (`librarian`, `pm`). Modules are registered in `main` as a compile-time
slice — never loaded at runtime. The single registration contract is `module.Module`
(`internal/core/module/module.go`): `Name()`, `SchemaVersion()`, `Enabled(cfg)`,
`Migrations()`, `OwnedCollections()`, `Tools()`, `RegisterHooks`, `TUIViews()`, plus optional
capability interfaces. `module.Register` filters the set by `Enabled(cfg)`, asserts no two
enabled modules claim the same owned collection, merges each module's `Tools()` into the shared
`toolcore` registry, and captures a `schema.DocumentValidator` if a module provides one.

**2. The narrow validation seam (R5.5a).** The PM module never reads the librarian's collections.
It obtains a verdict on a document pointer through one interface in core —
`schema.DocumentValidator` (`internal/core/schema/schema.go`), whose `Verdict(ctx, pointer, req)`
returns `{Exists, FrontmatterValid, Status, Satisfied, Missing}`. The **librarian module
implements it** (`internal/modules/librarian/module.go`, `func (m *Mod) Verdict`, reusing its
`desklib` frontmatter parser + the schema-v1 engine core now owns). Core injects the
implementation into the PM module after registration via `module.ValidatorConsumer.SetValidator`
(`internal/modules/pm/module.go`); the PM tool closures read it lazily at invoke time. This keeps
the gate **in-process** while keeping the domains extractable — the PM module depends only on the
core interface. A build-time import-graph test (`internal/modules/pm/module_test.go`, spec §10.5)
fails if any non-test `modules/pm` package — or any `internal/core` package — imports
`internal/modules/librarian`.

**3. Fail closed with no validator.** If no `DocumentValidator` is registered (a librarian-less
desk — structurally possible, unsupported in v1), the gate engine refuses every documented gate,
naming the absence: `"no document validator available; documented gates fail closed"`
(`internal/modules/pm/gates/gates.go`, `Evaluate`).

**4. Module-scoped migrations + a schema-version meta collection (R5.5b).** Core owns
`internal/core/migrate`. Each `Migration` is tagged with its owning module; two registration
styles coexist. The librarian's `0001..NNNN` keep their `init()` + blank-import path through
PocketBase's global runner (`SelfRegistered: true`) — the D2 zero-behavior-change gate. The PM
module's migrations are **programmatic values** returned by `collections.Migrations()`
(`internal/modules/pm/collections/collections.go`) and registered into the runner by core
**only when `pm.Enabled(cfg)` is true**. This is load-bearing: a PM migration written the
librarian's self-registering way would run regardless of the feature gate and break it — the
collections file carries a `BINDING DISCIPLINE` comment saying so. After the runner completes,
`StampModules` observes PocketBase's applied-migrations list and upserts a
`module_schema_versions` row per enabled module (`{module, version, applied_at}`;
`internal/core/migrate/0000_module_schema_versions.go`). `GuardDowngrade` refuses to serve a store
whose stored version for a module is *ahead* of the binary. The librarian's version row is
independent of the PM's, so a desk can lag one module's schema without the other caring.

**5. Per-desk feature gating (R5.5c).** The PM module is inert unless enabled.
`pm.Mod.Enabled(cfg)` returns `cfg.PMEnabled` (`internal/modules/pm/module.go`), resolved by
config layering `PM_ENABLED` env > profile `modules.pm.enabled` > default ~~**off**~~ **ON since
1.0** (`internal/core/config/config.go`; the default leg was flipped on 2026-07-21 — see the
Amendment at the top). When disabled: PM migrations are not registered (the
collections physically do not exist — not inert tables), PM tools are absent from every surface,
PM hooks/realtime are not bound, and PM TUI views are not mounted. The librarian is wholly
unaffected. Flipping the gate on runs the PM migrations and stamps the meta row.

**6. Shared tool core, one contract, three surfaces.** The librarian's hand-keyed tool switch is
generalized into `internal/core/toolcore`. A `ToolSpec` (`toolcore.New[I]`) carries `Module`,
`Name`, `Description`, an input type for schema reflection, and the `WritesFiles` /
`AgentDefault` / `AgentGated` flags. CLI, MCP, and the TUI/eino loop bind the merged registry
instead of a hardcoded switch, so a module self-registers a tool on every surface. The PM tool
family (`internal/modules/pm/tools/specs.go`) and the PM CLI group (`cmd/deskkit/pm.go`) and the
PM TUI views (`internal/modules/pm/tui/views.go`) are three surfaces over the one engine
(`internal/modules/pm/engine`) — parity is asserted by test §10.10.

**7. The chassis rename rode with this work (D2b).** The binary and store home renamed
`pocket-librarian` → `deskkit`; `store.AppDirName = "deskkit"` with automatic one-time migration
of a legacy `$XDG_DATA_HOME/pocket-librarian/<DESK>/` store to the new home
(`internal/core/store/storedir.go`). ADR 0002's store-path literal carries a dated correction for
this; the store-per-desk *decision* stands unchanged.

## Consequences

- The PM module is a template for a third module: implement `module.Module`, own a disjoint
  collection set with programmatic migrations, consume core seams (never a sibling module), and
  the gate/feature-gate/versioning disciplines come for free.
- A desk that never sets `PM_ENABLED` sees the exact librarian it saw before — same CLI, same MCP
  tools, same store layout, no PM collections. The D2 refactor preserved every stable `pbc_…`
  collection id (rebuild reproducibility, spec §8.2).
- The gate's honesty is structural: it can only refuse or admit based on a real `Verdict`, and it
  fails closed when it cannot get one. Tests §10.1 (gate red-ability) and §10.5 (the seam) pin
  both halves.
- "Building the bus while driving it" is supported: independent per-module schema versions let a
  desk adopt a PM schema bump without touching the librarian's, and a binary refuses a store it
  cannot safely read (downgrade guard).
- The rename is transparent to existing desks (auto-migration); `install.sh` falls back to the
  pre-rename asset name for releases up to v0.6.0.

## Alternatives considered

- **A second PM service / its own store** — rejected by R5.5: cross-store joins, a second daemon
  to run and back up, and IPC for what is a one-process gate check. The narrow interface gives the
  same extractability without the operational cost.
- **Runtime plugin loading (Go plugins / a registry read at startup)** — rejected: compile-time
  module registration is simpler, statically checkable (the import-graph test), and needs no
  dynamic-loading machinery for a fixed, in-repo module set.
- **PM reads the librarian's `files`/`findings` collections directly** — rejected: it welds the
  domains together and makes the librarian's schema the PM module's dependency. The
  `DocumentValidator` seam is the whole point — a stub validator drives the PM gate in tests with
  no librarian at all (spec §10.5).
- **PM migrations self-register like the librarian's** — rejected: `init()` + blank-import runs at
  compile time unconditionally, defeating the feature gate. Programmatic registration gated on
  `Enabled(cfg)` is what makes "no PM collections on a librarian-only desk" physically true.
- **One global schema version for the binary** — rejected: it would force every desk to migrate
  both modules in lockstep. Per-module `module_schema_versions` rows decouple them.

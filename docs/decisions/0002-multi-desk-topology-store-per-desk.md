_ADR for pocket-librarian's multi-desk topology: how an estate of many desks maps to stores, where stores live on disk, and what the `desk` field is for._
Status: Accepted — 2026-07-17

# 0002 — Multi-desk topology: store-per-desk, XDG store home, `desk` as open-guard

## Context

An estate can hold many executive desks (the concrete case: 7). The v1 shape didn't say how
that scales: seven stores or one shared store with a desk dimension? Where do stores live?
Issue #20 captured the facts; a design session on 2026-07-17 ruled on all four questions.

Facts as verified at ruling time:

- **No resident instance.** The binary is a one-shot CLI; an "instance" is a store (a
  `pb_data/`-style SQLite dir). The choice is N stores vs 1.
- **The `desk` field is populated, not latent.** `sweep`, `patrol`, and `apply_fix` all write
  `cfg.DeskName` into rows (`internal/tools/sweep.go`, `patrol.go`, `apply_fix.go`); the field
  exists on `files`, `patrol_log`, and `adoption_log`. What it lacks is participation in
  uniqueness — `idx_files_path` is unique on `path` alone (`migrations/0001_files.go`), so two
  desks in one store collide on any shared relative path (e.g. `_meta/HANDOFF.md`).
- **Config is per-desk by design and file-less** (env > walk-up `.env` > the desk's
  `_knowledge/profile.yaml` > defaults); one `DESK_ROOT`/`DESK_NAME` per process. Unchanged
  by this ADR.
- **Store location was ad hoc**: PocketBase's stock cwd-relative `pb_data/` unless `--dir` is
  passed. Stores had already scattered across scratch and job-tmp dirs in practice.
- **SQLite single-writer is an operating rule** (spec §10.4).

## Decision

Four rulings, taken together:

### 1. Topology: one store per desk

Store-per-desk; a portfolio view, if one ever materializes, is **read-only aggregation
(query fan-out across stores), not a shared write store**.

- The desk is the governance unit — profile, ignore boundary, and the record-original/restore
  safety net are all per-desk. A shared store makes one corruption event an estate-wide
  incident.
- Single-writer SQLite means N desks patrolling one DB serialize; N stores never contend.
- The only thing a shared store buys — cross-desk queries — is cheaper as fan-out (sweep is
  ~1s/desk), and `desk` remains the ready-made join key.

Rejected: shared store with composite `(desk, path)` keys (serializes writers, widens blast
radius, desk columns metastasize across every collection); hybrid write-stores + materialized
aggregate (most machinery, no proven roll-up need — revisit only if fan-out proves
insufficient).

### 2. Canonical store home: XDG data home

When `--dir` is absent, the store resolves to
**`$XDG_DATA_HOME/pocket-librarian/<DESK_NAME>/`** (falling back to
`~/.local/share/pocket-librarian/<DESK_NAME>/` when `$XDG_DATA_HOME` is unset), replacing
the cwd-relative `pb_data/` default. `--dir` remains the explicit override.

Constraints that drove it: stores must live **outside the desk tree** (the librarian must not
index its own DB, and SQLite inside an iCloud-synced folder is a corruption risk — desks are
iCloud-synced) and outside ad-hoc scratch dirs. One identity-neutral convention covers macOS
and Linux; `~/Library/Application Support` was rejected as a second platform code path with
no functional gain for a dev tool.

`DESK_NAME` must be unique across the estate — it is the store's directory name. The guard in
ruling 3 catches violations.

### 3. The `desk` field: keep, single-valued, promoted to an open-guard

The field stays on `files`/`patrol_log`/`adoption_log` and keeps being written from
`cfg.DeskName`. It gains a job: **on opening a store, if existing rows carry a desk name
different from the configured `DESK_NAME`, refuse to run** (clear error naming both values).
This catches the failure mode the location convention creates — two desks resolving to the
same store dir via a name collision or copy-pasted env — before their rows interleave.

No composite `(desk, path)` uniqueness: under store-per-desk a store only ever holds one
desk, so `path`-alone uniqueness is correct. Dropping the field was rejected — it is already
populated everywhere, it is the cross-store join key for any future fan-out roll-up, and
removal would touch three collections plus three tools for negative value.

### 4. MCP scope: one desk per process

The MCP server keeps binding exactly one `DESK_ROOT`/`DESK_NAME` per process. A multi-desk
estate registers one MCP entry per desk it wants live (or points a single entry at the desk
under work). A multi-desk routing server was rejected: real routing/config work that reopens
the shared-state questions this ADR closes, with no current demand.

## Operational notes (multi-desk)

- **Version skew across desks is self-healing.** Migrations run via automigrate on every
  start, per store; each store upgrades to the binary's schema the next time any command
  touches it. Desks may briefly sit on different schema versions between touches — harmless,
  since nothing reads across stores.
- **Patrol scheduling needs no coordination.** Stores never contend with each other; the
  single-writer rule applies per store, exactly as today.
- **Backup/restore is per-store**: one desk's store can be snapshotted, rebuilt (sweep is
  reproducible), or discarded without touching the others.

## Consequences

Implementation work (build cycle, not this ADR): the XDG default-resolution when `--dir` is
absent, and the desk open-guard. Until those land, the cwd-relative default and the silent
collision window persist. Existing scattered stores are simply re-created at the canonical
location by a fresh sweep — no data migration needed (stores are rebuildable caches of the
desk tree plus revision history; a store worth keeping can be moved with `mv` before first
run).

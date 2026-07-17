_ADR for pocket-librarian store initialization: whether a one-shot tool command may run against a store whose schema was never created._
Status: Accepted — 2026-07-17

# 0003 — Store initialization: tool commands self-initialize (auto-run app migrations)

## Context

The app migrations that create the desk's collections (`files`, `patrol_findings`,
`patrol_log`, `adoption_log`, `prompts`, …) are applied only by two entry points: `serve`
(via PocketBase's bootstrap, which runs pending migrations before serving) and the explicit
`migrate up` command. Every other store-touching command — `sweep`, `patrol`, `propose-fix`,
`apply-fix`, `query`, `agent`, `chat`, `mcp-serve`, `gui`, `restore` — is a plain Cobra
subcommand that opens the store and reads/writes directly, and does **not** trigger a
migration run.

The consequence: a tool command against a store that was never served or `migrate up`'d finds
no collections at all. PocketBase's lookup returns the bare `database/sql` sentinel
`sql.ErrNoRows`, indistinguishable at the call site from a genuinely-empty-but-existing
result. Two symptoms followed:

- **#25** — `query` leaked `sql: no rows in result set`. Fixed at the `Query()` dispatch point
  by translating the sentinel into an actionable message (PR #30).
- **#31** — the *same* leak still surfaces from `sweep`/`patrol`/other tool commands on an
  uninitialized store. The #25 message pointed users at `migrate up`, but that only papered
  over one command; the underlying asymmetry (serve/migrate initialize, tool commands don't)
  remained.

Every guide already instructs `migrate up` as a one-time first step, so the failure only bites
a user who skips the docs — but there is no scenario in which we *want* `sweep` on a fresh desk
to fail, given the migration is idempotent and safe.

## Decision

**Tool commands self-initialize the store.** `requireConfig` — the single choke point every
tool/`agent`/`chat`/`mcp-serve`/`gui` RunE passes through — runs `app.RunAppMigrations()` as
its first step, before the desk open-guard and any seeding. `RunAppMigrations()` applies only
migrations not yet recorded in the `_migrations` table, so it is idempotent and cheap
(a few `SELECT`s) once the store is current.

Ordering is deliberate: migrations run **before** `CheckDeskGuard`, so the guard consults
now-existing (empty) collections and passes a first-run store by construction, exactly as it
did before when the collections happened to exist. The command body then runs against real,
empty collections and returns empty results instead of an error.

Unchanged by this ADR:

- `migrate up` remains the explicit, standalone initialization/upgrade path.
- `serve` continues to migrate via its own bootstrap; `migrate` stays exempt from the desk
  open-guard (ADR 0002 §3).
- The fail-closed store-**location** guard (ADR 0002 §2) is untouched: an unresolvable
  `DESK_NAME` with no `--dir` still exits 1 before any store is opened. This ADR governs schema
  creation at a *resolved* location, not location resolution.
- The #25 error translation stays in place as defense-in-depth. With auto-migration it is no
  longer reachable on the normal path (collections always exist by the time `Query` runs), but
  it remains a correct fallback for any residual no-rows.

## Consequences

- A fresh desk works with any command: `sweep`, `query`, `chat`, `mcp-serve`, etc. no longer
  require a manual `migrate up` first. Docs drop it from a prerequisite to an optional explicit
  step.
- Behavior for one-shot tool commands now matches `serve`/`migrate up` — one mental model
  ("the binary manages its store"), not two.
- No new store-creation risk versus the prior state: the store directory was already
  pre-created `0700` for every store-touching command (ADR 0002 §2), and `migrate up`/`serve`
  already auto-create schema. A mistyped `DESK_NAME` could already spin up a fresh store; this
  ADR does not widen that.
- Minor per-invocation cost: an already-current store pays a `_migrations` check on each tool
  command. Negligible for a local single-user tool.
- Concurrency: two tool commands racing on a brand-new store could both attempt the first
  migration; PocketBase runs each migration in a transaction guarded by the `_migrations`
  table, and this is a local, effectively single-writer tool, so the race is benign.

## Alternatives considered

- **(1) Per-command error translation only** — apply the #25-style `sql.ErrNoRows` →
  actionable-message translation at each tool command's entry, keeping `migrate up` mandatory.
  Rejected: it is purely cosmetic over a papercut that serves no one, and preserves the
  serve-vs-tool-command inconsistency. It trades a good error message for the absence of the
  error — the error should not exist.
- **(2) Auto-generate + apply on schema drift via `migratecmd` Automigrate** — already enabled
  (`Automigrate: true`) but it governs *generating* migration files from Admin-UI schema
  changes in dev, not *applying* the blank-imported Go migrations on a plain command. It does
  not close this gap.

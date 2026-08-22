# Changelog

All notable changes to this repository — the **`deskkit` binary** (`pocket-librarian` through
0.6.0) and the **`deskkit` Claude Code plugin bundle** — are recorded here. Both ship under one
repo version (the root `VERSION` file); a release tags that single version.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html). See
[`docs/development/README.md`](docs/development/README.md) for how a version is bumped and cut, and
ADR 0005 (DESK-27)
for why this policy exists.

## [0.11.2] — 2026-08-22

### Security

- **A sweep no longer stores the body of anything on the desk's ignore list — the fix v0.11.0
  claimed and did not deliver.** The ignore list was a WRITE boundary only: `sweep` never
  consulted it, so a desk indexed `.claude/settings.local.json` — credentials included — with
  its full contents in `files.content`, which feeds `query search` and the records API. Now
  every content-indexing path (`sweep` and the watcher's single-file reindex) loads the list
  fail-closed — an unreadable list refuses the operation rather than indexing past a broken
  boundary — and stores no body for a matching path. Matching rows are still *indexed as
  metadata* (path, checksum, git meta, frontmatter): the list also write-protects the binding
  docs (decisions, handoff, `CLAUDE.md`), and patrol must keep flagging what it may not edit,
  a contract `verify.sh` enforces (ruled on the board, 2026-08-22).

  **Retroactive:** a rule that starts matching an already-indexed row clears that row's stored
  content on the next sweep, so existing stores self-heal — run `deskkit sweep` once after
  upgrading. There is still no purge path for anything else (`deskkit` has no "forget this
  file"); the remedy for a store that must forget more than content remains deleting the store.
  Gated by tests derived from the shipped seed (proven to fail with the boundary removed) and
  an end-to-end check that credential-shaped files are unreachable through `query search` on
  the real binary.

### Changed

- **The SPA's "adding an entity is a config entry" claim now states its scope, in both places a
  builder reads it.** The claim is true for genuine CRUD collections, and `documents` is the only
  browse entry at its designed shape. `findings`, `runs` and `pm` are read-only stand-ins for
  surfaces the design gives a *different* template — findings and the landing queue are an inbox,
  agent runs fold inline into the thread, work items get a phase board with a gate panel — so
  those three entries are meant to be replaced, not grown. Unqualified, the claim reads as licence
  to add an `edit` block to `findings` and build the wrong screen convincingly. Comments only, in
  `CLAUDE.md` and the header of `web/src/lib/collections.ts`; no behaviour changes.

## [0.11.0] — 2026-08-20

### Fixed

- **The seeded ignore list's `.claude/` entry was one file wide — the WRITE boundary now covers
  the whole directory.** ⚠️ *Corrected 2026-08-22: this entry originally claimed the change
  stopped a sweep from indexing agent config. It did not — the ignore list was a write boundary
  only, `sweep` never consulted it, and sweeps kept indexing `.claude/settings.local.json`
  (credentials included) into the searchable store. The indexing fix landed later; see the
  Security entry above.* What this release actually changed: the default list named
  `.claude/memory/MEMORY.md` and therefore write-protected nothing else in that directory; the
  entry is now the whole directory, so the librarian's write tools (`write_doc`, `delete_doc`,
  `propose_fix`, `apply_fix`) refuse everything under it. A test asserts the property against
  the SHIPPED seed rather than a copy of it, and fails naming the exact path if the rule is
  weakened. Entries are directory prefixes or exact paths only — `IsIgnored` does no glob
  matching — so broader patterns (`*.pem`, nested `.env`) still cannot be expressed.

  **This reaches new desks only.** `EnsureIgnoreFile` leaves an existing `.librarian-ignore`
  untouched (spec §10.1 — the file is the operator's once created), so an existing desk keeps
  its old boundary and must be updated by hand.

## [0.10.0] — 2026-08-20

### Added

- **A real-browser gate for the SPA** (`make spa-verify`, DESK-90). The browser app had no
  behavioural verification: two phases shipped on type-checks and unit tests alone. The gate seeds
  a throwaway desk, serves it, and drives the embedded SPA in a real Chromium — and a third of its
  checks read the file on disk after a save, which is the half a DOM-only harness structurally
  cannot cover and the half where the damage would have been. It runs in CI, not only on demand,
  because a gate someone has to remember to run is the failure mode it exists to end. Playwright is
  pinned in `e2e/spa/` and deliberately kept out of `web/`; when it is absent the gate exits
  non-zero with the install commands rather than skipping, since a silent skip reads as "verified"
  when nothing ran.
- **The CRUD template, built once and proven on Library** (SPA overhaul phase 2). The finder is no
  longer a screen that happens to browse `files` — every collection is described declaratively and
  the component reads that description, so adding an entity is a config entry and nothing else.
  That claim is the design's own falsifiable test, so it is asserted by a unit test that derives a
  second writable collection's behaviour from config alone with no component change. Three
  allocations of the one screen: looking (the finder IS the screen, rows carrying a preview line),
  examining (the finder minimises into its lit rail button), changing (the same allocation, verbs
  swapped to save / revert / delete). `esc` out of an edit IS revert.
- **A document's structured surface is editable, its prose is not.** `status` and `type` are
  editable through the write-through route; the status picker is FAMILY-AWARE, deriving its legal
  values from the doctype — and from the *drafted* doctype, so changing a document's type
  re-derives its legal statuses in the same interaction. A status left stranded by a type change is
  kept and marked, never silently dropped or auto-corrected.
- **`GET /desk/doctypes`** — the doc-type vocabulary (status families, per-type required/optional
  frontmatter) served from the embedded schema. Unauthenticated on both bind modes, like
  `/desk/models`: it carries no desk data. Lists are always present, never null.
- **`POST /desk/doc/delete`** — delete through the same door and the same auth posture as the write
  route, and reversible by construction: the original lands in the revisions ledger BEFORE the file
  leaves the disk, so `deskkit restore --by-path` reverses it byte-exact. Write-protected paths
  (`.librarian-ignore`) are refused, and a stale checksum is a 409 like any other write.
- **`preferences.editor_url` in the profile schema** — the desk names where its prose gets written,
  as a URL template (`{path}` / `{abs}`), and the browser renders an anchor. Nothing shells out; a
  desk that declares none gets no control. Reported by `GET /desk/settings/resolved` and by
  `deskkit config show`.
- **Icons on the rail**, hand-written SVG with no new dependency — and the digit stays visible
  beneath each glyph, because losing it would cost the rail its second job as the shortcut legend.
  `o` opens the body, `⌘⌫` arms the same two-step delete confirm the button uses.

- **The shell — a shortcut rail that is also the keyboard map** (SPA overhaul phase 1). The flat
  four-link nav is replaced by a 34px rail carrying one button per WORK MODE and nothing else:
  Queue, Library, Patrol, Work, Agent, Config (Kits and Desks are reached from Config; the queue
  itself arrives with the inbox screens). The rail is navigation, window manager and shortcut
  legend at once — every button has a key and every key has a button, so nothing is reachable
  only one way and the whole app is drivable without a mouse: `⌘1`–`⌘6` modes, `⌘B` finder,
  `⌘K` search (live, even mid-edit), `j`/`k` rows, `↵` open, `e` modify, `⌘↵` save, `esc` back
  out exactly one level (editing → reading → finder). `Ctrl` stands in for `⌘` off a Mac. The map
  lives in one table (`web/src/lib/keys.ts`) rather than scattered across components, because
  that promise is only checkable if it does.
- **Space follows engagement.** The finder is the whole screen while you are looking; opening an
  item minimises it INTO its rail button, lit, so the way back is visible rather than remembered;
  editing is that same allocation with different verbs and never navigates. Each mode's list also
  gained a search box (the `⌘K` target), and the row the keyboard is on is always visible.
- **A user setting: "keep the finder minimised between items"**, default on — with it on, `j`/`k`
  walk a collection without reopening the list; off, moving on hands the screen back to the
  finder. Stored per desk on the `settings` singleton (`sticky_finder`, migration 0025) and
  toggled from the Config panel, which also picked up the Sign out button the old nav carried.
- Legacy hashes (`#/documents`, `#/findings`, `#/pm`, `#/chat`, `#/runs`, `#/settings`) still
  resolve, onto their new modes.

- **The write-through path — one door from browser to disk** (DESK-78, SPA overhaul phase 0,
  implementing the DESK-75 files-stay-authoritative ruling). `tools.WriteDoc` writes a desk
  document to disk and updates its index row as one operation: original recorded to `revisions`
  first (reversible via `restore --by-path`), byte-exact write via `desklib.WriteExact`, row
  re-indexed in the same transaction. Compare-and-swap on the file's checksum: a save over a file
  that changed underneath is refused with the current content returned — never overwritten, never
  merged. Surfaces: the `write-doc` CLI subcommand, and `POST /desk/doc/write` for the SPA (public
  mode: `apis.RequireAuth` + strict same-origin, exactly like the chat routes). Deliberately NOT an
  MCP/agent tool — the autonomous write boundary stays `apply_fix`-only, gated.
- **A desk-tree watcher under `serve`.** Outside edits are the normal case (the desk is authored
  in external editors), so the index now follows the disk: fsnotify on the desk tree, debounced
  single-file re-index, soft-delete on remove, one convergence sweep at start. Failure to start
  degrades to manual `sweep`, never fails serve.
- **SPA: a document's `status` is editable** on the Documents browse pane — the first field
  through the write-through path, with a conflict view (refused save shows the disk's current
  content; overwrite is an explicit second action) and live row updates via PocketBase realtime.
- **`desklib.SetFrontmatterField`** — server-side single-field frontmatter edit preserving every
  other byte, so browsers never rewrite YAML.

- **Settings panel in the SPA** (DESK-66). Provider, model, and API key are now settable from the
  browser instead of only `deskkit config set`. Settings live in a new `settings` collection — a
  migration-seeded singleton — rather than in the central YAML file, because on a hosted desk the
  central file resolves outside the mounted data volume and does not survive a redeploy. The key
  is a PocketBase **hidden** field, so it is structurally absent from every API response instead of
  being filtered by hand; a companion `llm_api_key_hint` holds the display suffix and is recomputed
  server-side on save. Collection rules stay nil (superuser-only) — the database enforces the auth
  posture, with no hand-rolled middleware. Provider and model render as dropdowns off a generated
  catalog, each with a custom-value escape hatch. Decision: DESK-73.
- **A store leg in config resolution.** The chain is now `env > per-desk profile > store settings >
  central config > built-in default`. The store outranks the machine-wide central file because it
  is per-desk; a desk's declared profile still outranks it. `deskkit config show` reports it like
  any other leg.
- **`GET /desk/models`** — the generated model catalog the panel's dropdowns read. Unauthenticated
  in both bind modes (it carries no secrets and is needed before login). Regenerate with
  `node scripts/gen-model-catalog.mjs`, which filters the models.dev catalog to tool-capable chat
  models. Deliberately ungated by CI: its source is a remote API, and a network-dependent gate
  would be flaky.
- **`GET /desk/settings/resolved`** — reports which leg currently supplies each field, so the panel
  can show a value as locked by an environment variable rather than accepting an edit that will
  never take effect. Superuser-only on a public bind (401 without a token, 403 for a
  `users`-collection token); never returns the key value.
- **Container smoke in CI** — a path-filtered workflow running `scripts/docker-smoke.sh`, which had
  never been wired to CI. Kept out of the required `ci` job.

### Changed

- **The hosted container no longer runs as root** (DESK-68). It starts as root only to hand the
  data volume to a fixed unprivileged account, then replaces itself with that account via
  `su-exec` before the app starts. The chown is unconditional on every boot: a conditional one
  self-heals nothing, because `chown -R` writes the top directory first and continues past errors,
  so an interrupted run leaves the contents root-owned and the next boot would serve a read-only
  database behind a passing healthcheck. `DESK_ROOT=/` is refused before anything is mutated. No
  `USER` directive, since the drop happens at runtime — the cost, now documented in
  `docs/pattern.md`, is that `docker exec` lands as root and an orchestrator gating on
  `runAsNonRoot` will reject the image.
- `scripts/docker-smoke.sh` grew from 19 to 49 assertions, and now covers the case that actually
  matters: a volume seeded with root-owned contents from a prior root-run container, asserting the
  database migrates ownership and a row written before the handback is still readable after it. It
  also asserts the serving process is not root, which nothing checked before.
- Provider and model are re-resolved from the store when an agent session is created, so a change
  saved in the panel takes effect without restarting the process. Applied to the chat session, the
  claimer's run path, and session resume — all three read the same long-lived config and had the
  same staleness.

### Fixed

- **`/admin` now redirects to the admin console at `/_/`** (DESK-55). It previously fell through to
  the SPA's index fallback and silently rendered the app shell, which reads as "the console is
  gone" rather than "wrong path".
- The `desk-setup` template's `profile.example.yaml` no longer ships concrete `models:` and
  `secrets_ref:` values. Their placeholders (`"<model-id>"`, `"<ENV_VAR_NAME>"`) are not rejected by
  schema validation and are non-empty, so on a desk built from the template they won the profile
  leg and pinned a garbage model id — and made the missing-key error name an environment variable
  that cannot exist. Both blocks stay in the file, commented, as documentation.
- `CLAUDE.md`'s target list omitted `media` and `shellcheck` (DESK-58). The ticket's stated symptom
  — help text still naming the retired binary — no longer existed; the rename had already removed
  it.

## [0.9.0] — 2026-08-18

### Added

- **Embedded SPA v1** (DESK-52). `serve` now serves a Svelte + TypeScript single-page app (the
  PocketBase JS SDK, built with Vite from a new repo-root `web/` tree) at `/`, replacing the old
  standalone `/desk/chat` HTML page — that URL still resolves, via the SPA's own index fallback.
  `make build` builds the SPA before the Go binary and embeds its dist via `go:embed`; a binary
  built with a bare `go build` (no Node step) still compiles and runs, serving a small "not
  built" placeholder at `/` instead. v1 ships a chat screen (unchanged `/desk/chat/stream` and
  `/desk/chat/reset` endpoints underneath) plus a read-only browse of documents, findings, agent
  runs with their messages, and PM items; writes stay on the CLI/MCP tool core. Auth: on a
  loopback bind, a new loopback-only, origin-guarded `GET /desk/bootstrap` route mints the SPA a
  superuser token so the local operator sees no login; a public bind has no bootstrap route and
  the SPA instead shows a login form authenticating against `_superusers` via the SDK — resolving
  the previously accepted limitation that a hosted `/desk/chat` was not browser-navigable. The
  SPA's static shell loads without auth in both cases (same posture as the admin console's own
  shell: shell loads, data doesn't) — every domain collection keeps its nil, superuser-only API
  rules underneath either shell.
- **`schema/` contract versioning** (#124; ADR 0009). `schema/doctypes.yaml` and
  `schema/profile.schema.yaml` carry an explicit version marker (`contract_version` /
  `x-contract-version`, both `1` today); both lanes' loaders reject an unrecognized version loud
  instead of silently misreading it, distinct from the already-shipped store-side
  `module_schema_versions` migration mechanism.
- **Schema-v2 element model, reviewed draft** (#125, #126 v2 half; ADR 0018). The draft element
  model graduated to `docs/element-model-v2-draft.md`, folding in ADR 0018's four owner rulings
  and both adversarial reviews' 18 gaps. Tabletop simulations walked the model against six
  scenarios (the four v1 flows plus two v2-native software/research walkthroughs); the
  deficiency report at `_meta/research/model-simulations/` records every gap's disposition. Model
  stays `status: draft`; finalization is the owner's 1.0.0 maturity call (#87). Filed #197
  (software/PM phase-machine reconciliation) as a follow-up.
- **Discoverable TUI views + guided first-run** (#189). The chat TUI now shows a tab strip
  (`chat | pm context | pm board`), a keybinding footer, a `?` help overlay, and an
  announce-or-explain behavior for PM module views instead of the prior unadvertised `ctrl+p`
  chord and silent no-op on a no-mount desk.
- **User/developer docs split** (#190). `docs/README.md` and the Using-track pages
  (`getting-started.md`, `plugin-guide.md`, `librarian-guide.md`, `pm-guide.md`) now assume no
  build toolchain and no env-var exports for the core sweep/patrol/PM flow; build-from-source and
  env-var/store-path lore moved to a new `docs/development/install-and-build.md`.
- **Central machine-wide config + a self-explanatory CLI** (DESK-50). New `deskkit config
  show|path|edit|set` manages `$XDG_CONFIG_HOME/deskkit/config.yaml` (created 0600 in a 0700
  dir) carrying `llm.provider`, `llm.model`, `llm.api_key`, and `default_desk`. Resolution per
  field is env > per-desk profile > central config > built-in default, and `deskkit config show`
  prints every resolved value with the leg that won — so `deskkit config set llm.api_key <key>`
  once is enough to run `agent`/`chat` with zero env vars set. New `deskkit desks` lists the
  desks this machine has a store for (marking the one the cwd resolves to), and `deskkit --help`
  groups the command menu into six functional sections instead of one alphabetic list.
- **Hardened serve on a non-loopback bind** (DESK-51). `serve`'s auth posture is derived from
  every resolved bind address, never from a flag: a loopback bind keeps the unauthenticated
  local UX; anything else (wildcard, bare `:PORT`, routable IP, or a hostname that can't be
  proven loopback) makes the whole process public mode. Public mode refuses to open a listener
  unless it can verify an administrable superuser exists (provisioned fatally from
  `PB_SUPERUSER_EMAIL`/`PB_SUPERUSER_PASSWORD` and re-counted, excluding the framework's
  installer-placeholder row), puts the chat session routes behind auth, leaves the loopback-only
  token-mint route unregistered, switches CSRF to strict same-origin, and drops the framework's
  default CORS wildcard unless an explicit `--origins` allowlist is set. The stock `users` auth
  collection is now gated behind operator approval instead of shipping enabled.
- **Container packaging with a scripted smoke proof** (DESK-51). A Dockerfile packages the
  binary (SPA embedded) for hosted deploys, with a docker-smoke script proving serve + health
  from the built image; the `VOLUME` directive was dropped because volume-managing platforms
  reject it. The full deploy pattern is documented in `docs/pattern.md`.

### Fixed

- **`findings dispose` unusable from the CLI** (#184). `query findings` / `query uncollapsed`
  now include each finding's record `id`, matching `feedback`'s existing shape — the CLI-only
  disposition workflow no longer needs the admin GUI/REST to find an id.
- **`update_item` skipped type validation** (#185). `update_item` now validates `items.type`
  against the schema vocabulary like `create_item` does; an untyped item is refused on any edge
  the desk config gates for a known type, closing a supervised-gate bypass.
- **Subprocess test store collision across worktrees** (#182). `TestPMActorBeforeLeaf_Subprocess`
  now resolves its store under a per-run temp `XDG_DATA_HOME`, so parallel worktrees at different
  schema versions no longer collide on a shared machine-global store.
- **Stale "desk-pm mount" label** (#181, #187). Renamed to "pm-only mount" across
  `docs/tool-surface.md`, the MCP server + its tests, and the e2e suite; the default-mount tool
  count corrected from the pre-PM-default-on `5` to the current `17`.
- **`install.sh` corrupted artifact URLs when resolving `latest`** (DESK-56). `resolve_version()`
  wrote its progress line to stdout inside `TAG="$(resolve_version)"`, so the line was captured
  into `TAG` and corrupted every artifact name and download URL built from it — masked until now
  because the stale repo slug 404ed first. Progress goes to stderr, the pinned slug is
  `hsb3/deskkit`, and a failed tag resolution now names the real candidate causes (repo missing,
  private/needs auth, or no published release) instead of guessing one.
- **`make help` never listed `e2e`** (DESK-56) — the target-name pattern excluded digits.
- **`.dockerignore` let a host `node_modules` into the build context** (DESK-56), so a darwin
  host's modules could overlay the image's linux ones and break the container build.
- **npm dependabot updater pointed at the deleted `plugin/` lane** (DESK-56). Because
  `open-pull-requests-limit: 0` suppresses routine PRs it looked deliberate, but security
  updates ignore that limit — so the SPA's npm deps, which ship inside the distributed binary
  via `go:embed`, had no security-update coverage. Now pointed at `/web`.
- **`examples/agent-loop.sh` asserted a pre-consolidation tool surface** (DESK-70). The harness
  called `mcp-serve` without `MCP_MODULES`, mounting the full tool surface where it asserted the
  old five-tool one — every model run carried two phantom failures, which nearly argued for
  keeping the expensive default model from a bug. The harness now pins its mount and reports
  19/19 against the current default.

### Changed

- **One binary, one plugin bundle** (ADR 0022, DESK-48). The TypeScript lane (`plugin/`) is
  deleted: its profile tools were ported to a new Go `profile` MCP module (`profile_get`,
  `profile_validate`, `template_render`, `knowledge_index`) on the `deskkit` binary, and the two
  marketplace bundles collapsed into one whose single `.mcp.json` launches `deskkit mcp-serve`
  with `MCP_MODULES=profile,librarian,pm`. One runtime, one agent-facing bundle, one schema —
  there is no second MCP server or Node runtime to install.
- **Go module moved to the repo root; everything is named `deskkit`** (DESK-56, DESK-64). The
  module path is now `github.com/hsb3/deskkit` (install from source with
  `go install github.com/hsb3/deskkit/cmd/deskkit@latest`), the `librarian/` container directory
  is retired (runtime at `internal/`, binary at `cmd/deskkit`), the bundle is `plugins/deskkit/`,
  and the marketplace slug is `deskkit` — plugin install reads
  `claude plugin marketplace add hsb3/deskkit` then `claude plugin install deskkit@deskkit`. The
  GitHub repo itself was renamed `hsb3/desk-standard` → `hsb3/deskkit` and made public (the old
  slug redirects). The two per-lane Makefiles merged into one root Makefile, and the operator
  reference moved to `docs/usage/deskkit-reference.md`. Module *names* are unchanged:
  `MCP_MODULES=profile,librarian,pm` still reads exactly that way.
- **Gate diet** (DESK-49). The drift guards whose duplicated copies no longer exist
  post-collapse were retired, and the tool-surface doc counts folded into a Go test
  (`make test`) instead of a standalone script.
- **Docs and comments made true of the one-binary system** (DESK-57, DESK-59, DESK-65, DESK-69).
  README rebuilt as value + proof with real SPA screenshots; stale "two products" / private-repo
  / TS-lane claims cleared across the published surface; `kits/README.md` documents the
  add/edit/remove procedure; a comment-hygiene sweep trimmed narration across the Go tree and
  shell entry points (prose only — zero executable lines changed); and the `agent` subcommand is
  ratified as approved CLI surface in the tool-surface spec.
- **Built-in default LLM model changed from `claude-opus-4-8` to `claude-haiku-4-5-20251001`**
  (DESK-67). The old pin was stale and the most expensive tier, for a workload (document
  linting, rule findings, mechanical fixes) that does not need it — every user who never sets
  `LLM_MODEL`/`models.model`/`llm.model` gets this model. Measured before pinning rather than
  assumed: three `examples/agent-loop.sh` runs each of `claude-haiku-4-5-20251001` and
  `claude-sonnet-5` plus one `claude-opus-4-8` baseline (18 August 2026). Haiku passed 17/19
  checks on all three runs (the 2 residual failures are a harness bug, DESK-70, unrelated to
  model choice); Sonnet passed 17/19 twice but 16/19 once, with a genuine tool-calling flake
  (`agent run (no writes) exits 0` failed for real, not the harness bug). Haiku was also 2.5-4x
  faster (28-32s vs 80-117s per run) and is the cheaper tier. This does not touch the
  resolution order (env > per-desk profile > central config > built-in default) — only the
  bottom leg's value.
- **Repo-root `_knowledge/` instance removed** (#170, #84 second half; ADR 0021 §F5). Per the
  owner ruling on issue #170 (2026-07-21), the repo itself is not an exec desk, so it no longer
  ships a repo-root `_knowledge/` — the `_knowledge/` convention belongs only to the desks the
  tools stand up. This is **not a rename**: the desk-side convention is unchanged (the
  `schema/paths.yaml` `profile_root`, the TS `PROFILE_ROOT_DIR` / Go `ProfileRootDir` constants,
  the `check-profile-root` drift guard, and every product string naming `_knowledge/profile.yaml`
  all stay exactly as they were). The canonical placeholder example profile now has a single home
  **with the desk-setup scaffold template**
  (`plugins/deskkit/skills/desk-setup/assets/template/_knowledge/profile.example.yaml`), so
  copying the scaffold brings it into a new desk; the repo-facing docs (README, CLAUDE.md,
  getting-started) re-point there, and the repo-root `.gitignore` now ignores any local `_knowledge/`
  wholesale as a dogfooding artifact.

## [0.8.0] — 2026-07-21

### Added

- **Browser session surface** (#19, PR #177). `deskkit serve` now serves a purpose-built chat
  page at `/desk/chat` via a custom route on the embedded PocketBase — the same multi-turn
  stewardship session as the `chat` REPL, over the same `internal/agent` session type and gated
  tool registry (no new write path; `apply_fix` stays env-gated; `restore` unreachable).
  Responses stream as SSE frames riding `StreamTurn`'s JSON-tagged events; history keeps the
  ≤ 40-message bound. The page is a single `go:embed`ded self-contained file — zero frontend
  toolchain. Posture: unauthenticated + loopback binding, with a loopback-origin guard on the
  write/stream routes (cross-origin browser POSTs get 403 before any SSE header).
- **TUI polish** (#171, PR #176). Sessions-list delegate now themed per-palette (no more
  hardcoded bubbles dark colors); resume-first launch (prior conversations open on the sessions
  list; fresh desks drop straight into a new session); session **archive lifecycle** — soft,
  reversible hide distinct from hard delete (migration `0022_agent_runs_archived`, `a`/`A`
  picker keys, archived excluded from the default listing with an opt-in reveal). The picker
  N+1 GROUP-BY collapse stays deliberately unbuilt (conditional on `pickerLimit` growing).
- **ADR 0021 — decision-0021 graduation** (PR #178). The off-repo executive-desk ruling that
  several 1.0.0 lanes cited by fork label (F1–F7) is now an in-repo decision record, with the
  2026-07-21 owner sign-off rulings recorded for F2 (PM default-on), F5 (`_knowledge/` move —
  approved pending target), and F7 (value-evaluation scoping). ADR 0020 carries its owner
  confirmation (supersession window closed). K24 brownfield **retrofit-preservation defaults**
  ruled into the standard (#81): `custom.original_<field>` preservation on overwrites, fixed
  `created`-derivation order, tags merge-not-replace; retrofit-over-exempt is a supported
  disposition.

- **Chat TUI sessions surface** (#51). The `ctrl+o` resume picker grows into a keyboard-driven
  sessions manager: per-thread message counts + last-activity, built-in fuzzy filter, a live
  preview pane of the highlighted thread's recent transcript, inline rename, and delete behind a
  y/n confirm. Delete is a hard delete (messages cascade via the existing `messages.run`
  relation; no new migration) — the record-original-first boundary governs desk files, not the
  user's own chat history. New `agent` package surface: `RenameConversation`,
  `DeleteConversation`, `PreviewConversation`; `ListConversations` now enriches rows with
  `MsgCount`/`LastActivity`.
- **Chat TUI context + token accounting** (#52). Provider token usage flows from the eino stream
  callbacks through the `Event` substrate (json-tagged `prompt_tokens`/`completion_tokens`/
  `total_tokens`, so a future SSE webapp inherits it) to a header `NN% ctx · N tok` segment and a
  per-turn footer counter. The context window resolves `LLM_CONTEXT_WINDOW` env > profile
  `models.context_window` > a per-model table; ctx% reads the latest step's prompt tokens (the
  live replayed context). Usage is live-only, not persisted.
- **Chat TUI external-editor hatch** (#64, the one item that issue marked build-now). `ctrl+e`
  writes the draft to a temp `.md`, hands the terminal to `$VISUAL`/`$EDITOR` via
  `tea.ExecProcess`, reads the composed text back on return, and always removes the temp file;
  no-op while streaming; the draft is never auto-sent.
- **PM items carry a long-form `body`** (#90). New forward migration `0006_pm_items_body`
  (explicit 50,000,000-char cap per ADR 0017; PM store schema version 5 → 6) plus end-to-end
  threading: `create_item`/`update_item` tool params, `pm create`/`pm update --body` flags, and
  `get_item` returns it (detail-only — list/context summaries stay lean). Omitting `body` on
  update leaves it unchanged; an explicit empty `body` clears it (the #168 convention below).
- **CI enforcement of the local-only gates** (#34, #97, #74, #35). Branch CI (and the release
  gate, as a strict superset) now runs shellcheck over the shell entry points, actionlint
  (version-pinned build via the Go toolchain), and a new SHA-pin drift guard
  (`scripts/check-workflow-pins.mjs` + `--self-test`) asserting every workflow `uses:` stays
  pinned to a 40-hex commit; first-party actions bumped to their latest majors at resolved SHAs.
  New enforcing tests: byte-equality drift guard for the plugin's two schema copies (`bun test`
  lane, #97); a secret-shaped-field recurrence guard over every desk-owned collection — librarian
  AND PM (#74); `requireConfig` self-init coverage on a genuinely never-migrated store plus a
  behavioral TS MCP server test over an in-memory transport pair (#35).
- **Sweep-time content indexing + `query search`/`content` kinds** (#89). Sweep now stores each
  file's body in a new `files.content` column (migration `0021`), so swept content is retrievable
  and searchable through a tool surface for the first time — previously the raw bytes were used
  only to compute a checksum and parse frontmatter, and nothing indexed was retrievable. Two new
  `query` **kinds** carry it (no new tool, no new CLI subcommand — the count surfaces are
  unchanged): `search` does substring/keyword retrieval over the indexed body via PocketBase's
  LIKE-contains operator (`content ~ term`, not FTS5; embeddings/vector search are out of scope),
  returning each match with a context snippet (`--term`, `--limit` default 20 / hard-capped 200);
  `content` returns one file's stored body by desk-relative `--path`. Indexing is UTF-8-only,
  never indexes a file under `SECRETS_DIR`, and is capped rune-safe at 1,000,000 chars; the body
  is re-derivable by a fresh sweep, so the store stays disposable. Store schema version 20 → 21.
- **Document identity + schema hygiene** (#123, ADR 0017). Three independent fixes shipped as
  one package. (A) **Frontmatter `id`** is now a recognized, OPTIONAL document-identity
  primitive: sweep reads it into a new `files.doc_id` column (migration `0018`) and matches an
  existing row by `doc_id` FIRST, falling back to `path`, so a renamed document that carries an
  `id` updates its existing record at the new path instead of being soft-deleted and
  re-inserted — rename stops discarding history. A document with no `id` keeps today's
  behavior unchanged, and two documents sharing one `id` within a sweep are never merged: the
  duplicate falls back to path-matching and is surfaced as a patrol-visible finding
  (`duplicate-doc-id`). (B) **`files.entity_type` is renamed to `files.doctype`**
  (migration `0019`, in place, reversible) — the old name collided with the schema's unrelated
  `entity_type` enum (a person/company classification); only the column name changed, the
  value did not, and every code/test/spec literal referring to the old column name moved with
  it. (C) **Explicit `Max` caps on seven content-bearing `TextField`s** that still rode
  PocketBase's implicit 5,000-char default (migration `0020`; `patrol_findings.detail` /
  `proposed_fix` widen to 50,000, five summary/detail/error fields tighten to 2,000), plus a
  new dependency-free `scripts/check-textfield-max.mjs` recurrence guard (wired into
  `make check` + CI) that fails any future uncapped content `TextField` by name. Store schema
  version 17 → 20.
- **Tool-surface drift guard** (#121, ADR 0016). `docs/tool-surface.md` — the authoritative,
  empirically-derived map of every tool-bearing surface (#94) — is now pinned to source by a
  mechanical guard, closing the manual "remember to re-run the probe" gap that let the old
  "seven-tool core" shorthand rot. A dependency-free `scripts/check-tool-surface.mjs` (wired into
  `make check` + CI next to `check-prompt-drift.mjs`) cross-checks the **Plugin TS MCP server** count
  (4, the `TOOLS` array in `plugin/core/tools.ts`) and the **Librarian CLI** base count (16, the
  `AddCommand` registrations in `librarian/cmd/deskkit/main.go` plus the framework system commands)
  against the numbers the doc states, with a `--self-test` proving it fails RED on a tool
  added/removed without a matching doc edit. The gate-dependent **Librarian MCP** counts
  (5 / 6 / 17 / 18 by `LIBRARIAN_AUTONOMOUS_WRITES` × `PM_ENABLED`, and the `MCP_MODULES=pm` desk-pm
  mount → 12) are pinned by a Go test (`TestToolSurfaceDoc_MCPCounts`) that reads the same doc counts
  and re-derives them from the real `toolcore` gate on the `go test ./...` lane — no reimplemented
  gate arithmetic to drift. The guard pins **counts**, not the doc's bytes, so an unrelated prose/row
  edit never trips it. A new gate only; no runtime behavior changes.
- **`desk-persona` — the composed librarian + PM Claude Code bundle** (#119, ADR 0014(a), ADR
  0015; the platform's v1 proof surface per ADR 0009). A new `plugin/desk-persona/` marketplace
  entry mounts one `deskkit mcp-serve` server with `MCP_MODULES=librarian,pm`, exposing all 17
  tools (the 5 librarian tools `sweep`/`patrol`/`propose_fix`/`query`/`record_feedback` plus the
  12 PM tools) behind two agents (`librarian-operator`, `pm-operator`) and the three PM skills
  (`pm-session-open`, `pm-advance-item`, `pm-triage`). The PM-sourced artifacts are copies of the
  existing `desk-pm` content with the MCP namespace rewritten `mcp__desk-pm__` →
  `mcp__desk-persona__` (one authored PM source; `desk-pm` coexists, unchanged); the
  librarian-operator body is generated from the canonical librarian instruction
  (`librarian/templates/librarian-system-prompt.txt`). A new drift guard
  (`scripts/check-persona-drift.mjs`, wired into `make check`) fails non-zero if either generated
  surface is hand-edited out of sync with its source. `plugin/desk-persona.test.ts` (`bun test`,
  `plugin/package.json`'s test glob extended) asserts the composed tool set by name, guards
  against phantom/invented tool identifiers across the bundle, and pins the marketplace/version
  wiring. `docs/README.md` gains a pointer to the bundle's `README.md`.
- **Findings lifecycle completed** (#118, ADR 0013). Finishes the disposition sub-machine #93
  opened. (1) The dead `state.dismissed` enum value is retired — migration `0015` remaps any
  residual row to `flagged` (data-first) then shrinks `patrol_findings.state` to
  `flagged`/`fixed`/`resolved`. (2) "Open findings" now means one thing everywhere: `query
  summary` and `query uncollapsed` became disposition-aware (`disposition = 'open'`), matching
  `query findings`' live default, so a disposed finding no longer inflates counts on some surfaces
  and not others. (3) Dispositions carry provenance — migration `0016` adds `actor` (max 200),
  `reason` (max 2000), and `disposed_at` (a plain date set at dispose time) to `patrol_findings`;
  `deskkit findings dispose <id> --as <disposition> [--by <actor>] [--reason <text>]` persists
  them (both flags optional, no baked default actor), and a re-fired finding inherits disposition
  AND provenance on `(file, rule, checksum)`. (4) `adoption_log.event` shrinks to writer-backed
  reality — migration `0017` retires the five writerless values, keeping only `fix` (its readers
  and deskguard role unchanged). Store schema version 14 → 17.
- **Prompt-copy drift guard + git-is-truth prompt governance** (#120, ADR 0015). A new
  dependency-free `scripts/check-prompt-drift.mjs` (wired into `make check` + CI next to
  `check-kits.mjs`) asserts the librarian system-prompt embed
  (`librarian/templates/librarian-system-prompt.txt`) and its "kept verbatim" quote in
  `docs/pocket-librarian-v1-spec.md` §6.1 stay **byte-identical**, closing the drift class that
  shipped a stale six-tool spec quote against the five-tool embed with nothing to catch it.
  `prompt.Seed`'s doc comments, spec §4.10/§6.1, and `librarian/README.md` now state ADR 0015
  in its own terms: the DB `prompts` row is a **re-seeded cache** (not canonical), GUI/REST
  edits are **ephemeral by rule**, and `_knowledge/` is the only durable customization path —
  with a documented **"reset to shipped"** affordance (delete the row via the admin console; the
  next command or a `serve` restart re-seeds it from the embed). Documentation + a new gate only;
  no runtime behavior changes (`Seed`'s seed-if-absent logic and the resolver are unchanged).
- **Tool-level MCP module gating on a shared mount** (`MCP_MODULES`; the agent integration
  contract — ADR 0014 (DESK-36),
  `docs/agent-integration-contract-v1-spec.md`, #114 under epic #129). `deskkit mcp-serve` now filters its
  exposed tool set to the modules named in `MCP_MODULES`, keyed on each tool's `ToolSpec.Module`
  (`internal/core/mcp/server.go` → `toolcore.SelectByModules` over `toolcore.ExposedSpecs(cfg)`),
  so a shared MCP mount carries only the tools it is meant to expose. The **desk-pm** plugin mount
  (`plugin/desk-pm/.mcp.json`) declares `MCP_MODULES=pm` alongside `PM_ENABLED=true` and therefore
  exposes **exactly the 12 PM tools, dropping the 5 librarian ride-alongs (17 → 12)**. The
  semantics are three-way and deliberately non-collapsing: `MCP_MODULES` **unset** exposes every
  module (the 5 / 6 / 17 / 18 counts are unchanged); **set-but-empty** (`""`, `" , "`) or
  **unresolvable** (a typo, or a module not registered/enabled on this desk) **fails loud** with a
  direct `os.Exit(1)` and an actionable stderr line — never a silent fallback to "all". The
  `deskkit mcp-serve` mount signal now names the gated set (`modules: pm; 12 tool(s) exposed: …`).
  `docs/tool-surface.md` gains the module-gating axis (§2.1) and an extended count-derivation
  method; the eino agent loop stays librarian-only (ADR 0014(c)) and its prompt no longer names PM
  tools it never receives.
- **Typed cross-reference contract in `schema/`** (#116, ADR 0011). `schema/references.yaml`
  adds schema v1's third dimension — a `{kind, target}` reference primitive with a closed
  `kind` enum (seeded `issue`, `url`) and a raw `target` string. The desk-relative repo
  qualifier is documented as read-time-resolved from `profile.repos.shorthand.issue_default`
  and never persisted (no default qualifier ships — identity-neutral). A validation guard lands
  in both lanes — `ReferenceVocab`/`ValidateReference` (Go, `librarian/internal/core/schema/
  references.go`) and `validateReference` (TS, `plugin/core/references.ts`) — each drift-guarded
  byte-for-byte against the canonical file. No field migrates onto the shape and no store
  migration ships: `graduated_to` / `items.pointer` behavior is unchanged, and the field
  migrations ride the schema-v2 track. Forward-pointer notes added to
  `docs/pocket-librarian-v1-spec.md` and `docs/pm-system-v1-spec.md`.
- **Normative `pointer` grammar spec section** (#115). `docs/pm-system-v1-spec.md` §3.1a now
  defines the `items.pointer` grammar ADR 0010 ratified — desk-relative file path with an
  advisory `§ <heading>` suffix; URL and `#`-anchored forms fail closed — and corrects the
  field-table row that still described the pointer as "doc path / issue URL / other locus".
  Docs-only; the shipped, test-pinned behavior is unchanged.
- **Findings disposition lifecycle** (#93). Patrol findings now carry a `disposition`
  (`open`/`acknowledged`/`triaged`/`wont_fix`), orthogonal to `state`. `deskkit findings dispose
  <id> --as <disposition>` marks a finding; `query findings` defaults to live (undisposed) items
  with `--include-disposed` to show history. Dispositions survive re-patrol by inheriting on
  (file, rule, checksum); a finding whose evidence (checksum) changes re-opens automatically.
  Migration `0014_patrol_findings_disposition` backfills existing rows to `open`.
- **`items.type` validated at creation** (#117). `create_item` hard-refuses a non-empty `type`
  outside the schema-v1 vocabulary (ADR 0012), naming the offending value, the known types, and
  `schema/doctypes.yaml`; an absent type stays legal. The importer inherits the check with no
  importer-side code (regression-tested both places). Engine-level only — no DB migration;
  `items.type` stays a bare TextField.
- **Authoritative tool-surface document** (#94) — `docs/tool-surface.md` (linked from
  `docs/README.md`) enumerates every surface with empirically-verified counts: the librarian MCP
  tools by gate (5 default / 6 with `LIBRARIAN_AUTONOMOUS_WRITES` / 17 with `PM_ENABLED` / 18 both),
  the CLI subcommands, and the plugin's separate 4-tool TS MCP server. Replaces the false
  "seven-tool core" help string, which matched no real surface.
- **`deskkit mcp-serve` mount signal** (#79) — a one-line readiness line on stderr (server identity
  + tool count) so an absent PM tool surface is diagnosable instead of silent.
- **Scaffold-frontmatter drift guard** (#80) — `scripts/check-scaffold-frontmatter.mjs` (wired into
  `make check`) asserts scaffold-shipped instruments carry conformant frontmatter.

- **Repo-conformance pass to the code-desk standard** (Lane 6, #86). The repo now passes its own
  `repo-compliance-audit` with zero gaps:
  - **Root entry-doc set** — a `CLAUDE.md` agent-navigation guide (command surface, the one
    identity-neutrality rule, order-sensitive chains, config resolution), `AGENTS.md` (a symlink to
    `CLAUDE.md` — one source, no drift), and `docs/CHARTER.md`, the canonical page with an explicit
    precedence rule and the settled 1.0.0 direction.
  - **Scaffolded meta-structure** (`mise-en-place-scaffold`, additive-only) — `.claude/{agents,hooks,
    rules,memory}` with a tracked `settings.json` + `memory/MEMORY.md` index, root `.mcp.json`,
    `.github/dependabot.yml`, and the ADR template.
  - **`tests/` declared home** — `tests/README.md` documents that suites live with their products
    (`plugin/` bun, `librarian/` go, `librarian/verify.sh`), keeping `make test` / `make verify` as
    the canonical entries rather than hoisting suites to a root tree.

- **Single authoritative profile-root constant** (first half of #84). The `_knowledge` directory
  name now has one canonical definition (`schema/paths.yaml` `profile_root:`) with one constant
  per lane (`plugin/core/profile.ts` `PROFILE_ROOT_DIR`, `librarian/internal/core/config`
  `ProfileRootDir`) driving all path resolution, pinned identically by a new
  `scripts/check-profile-root.mjs` drift guard (+ self-test) in `make check` and CI. Root
  `_knowledge/README.md` added. The directory move itself stays gated on the #170 ruling.
- **Hygiene gates** (#163 #164 #165 #166). A gofmt gate (`make -C librarian fmt`, run in the CI
  librarian lane) after formatting three drifted files; the manual `dogfood-*.sh` harnesses are
  shellcheck-clean and inside the lint gate (lint-only — still never executed in CI);
  `librarian/verify.sh` grows 48 → 55 checks (standing coverage for `query search`/`content` and
  orphans `--show-index`); a query-kind drift guard (`scripts/check-query-kinds.mjs` + self-test)
  pins types.go ↔ spec §5.6 ↔ the query.go switch.
- **PM manifest round-trip carries `body`** (#167). The importer forwards `body` end-to-end; a
  round-trip test asserts byte-equality into a fresh store.
- **Unset-vs-empty convention for optional string params** (#168). Presence, not value, signals
  intent on both surfaces: an omitted param (or JSON `null`) leaves the stored field unchanged; a
  present empty string (`--body ""`, `"body": ""`) deliberately clears it. MCP optional fields
  move to `*string`, the CLI keys off `Flags().Changed()`; documented in `docs/pm-guide.md`.

### Changed

- **The PM module now ships default-on** (#83, PR #179). Layering unchanged — env `PM_ENABLED`
  > profile `modules.pm.enabled` > default — but the default leg flips to ON; opt out with
  `PM_ENABLED=false` or `modules.pm.enabled: false` (three-state resolution: only a literal
  profile boolean decides, absent inherits the default). A fresh desk boots with the blessed
  §4.2 gate-rule seed and §3.3 status-label vocabulary. Owner-ruled 2026-07-21 (decision-queue
  sign-off); ADR 0008 carries the dated amendment; spec/pm-guide/READMEs/CHARTER/tool-surface
  updated to the on-by-default posture.
- **`desk-pm` folded into `desk-persona`; standalone bundle retired** (PR #180; owner ruling
  2026-07-21, ADR 0014(a)). desk-persona now carries the SessionStart PM briefing hook and the
  three PM skills; the duplicate `pm-operator` agent-name collision is gone. Persona-drift now
  generates only `librarian-operator` (the PM agent/skills are authored in place — one source
  per surface, ADR 0014(d)); the version-sync manifest set shrinks 7 → 5.
- **Docs made true to the shipped product** (#85). The spec (`docs/pocket-librarian-v1-spec.md`,
  path unchanged) is retitled to `deskkit` with `Status: active`; `docs/getting-started.md` leads
  with the authenticated `gh release download` install path while the repo is private (the
  `curl | bash` path is labeled post-public); stale `pocket-librarian` command/path references
  swept across docs with naming-provenance mentions kept; stale spec prose corrected (query kinds
  7 → 10, verify.sh count 48 → 55 where cited).
- **PM spec §12 traceability** (#169) gains rows for ADR 0020 (authoritative claim semantics) and
  the `body` field, each citing its verifying test.
- **CI actions bumped a major** (Dependabot #106 #108 #109 #110): checkout v7, setup-node v7,
  upload-artifact v7, claude-code-action 1.0.171 — all SHA-pinned as before.
- **Go module renamed to its real hosting path** (#98). `librarian/go.mod` moves off the
  placeholder `github.com/example/pocket-librarian` to `github.com/hsb3/desk-standard/librarian`
  (78 importing files rewritten), unblocking the remote
  `go install github.com/hsb3/desk-standard/librarian/cmd/deskkit@<version>` flow once the repo
  is public; the local `make install` path is unchanged. The module-path literal is sanctioned by
  token-scoped `schema/neutrality-lint.allow` entries (owner ruling on the issue): a Go module
  path is compile-time public API and cannot be profile-templated.
- **A live PM claim is now authoritative over every direct mutation** (#96,
  ADR 0020 (DESK-41)). `liveForeignClaim` now gates `Block`,
  `Unblock`, and `UpdateItem` (including the `status_label` path) alongside `Transition` — a
  non-holder is refused with a message naming the holder and expiry until the claim lapses (TTL,
  ADR 0019) or is released. Cascade/auto-unblock paths are deliberately untouched (claims
  coordinate actors, not the graph's own derived state). Docs now also state the actor-attribution
  surface (#95 — shipped earlier with the PM surfaces PR, docs were the gap) and which
  transitions gate under the shipped defaults (#101: only `decision review→terminal` and
  `task work→review`; frontmatter validation is key-presence-only).
- **PM realtime broadcast is bounded and per-client ordered** (#68). The unbounded per-message
  goroutine fan-out becomes a per-client dispatcher: one 64-message queue + one drain goroutine
  per client (FIFO by construction), non-blocking enqueue with drop-newest under sustained
  backpressure (realtime is the observer channel — a dropped event is acceptable, a blocked
  transition is not), reap-on-broadcast + idle recheck so no goroutine outlives its client.
- **`printJSON` takes an `io.Writer`** (#154). All 19 cobra callers pass `cmd.OutOrStdout()`,
  ending the `os.Pipe`/`os.Stdout`-swap capture pattern in the command tests.
- **`query orphans` hides by-design-unreferenced index/entry files by default** (#100). Basename
  `CLAUDE.md`, `README.md`, and `INDEX.md` (case-insensitive) are structural orphans — empty-doctype
  `.md` files outside `meta`/`memory`/`infra` — but an entry/index doc is what *other* docs point at,
  so it is never a misfiled orphan. The default view now filters them out (an ADDITIONAL filter on
  top of the unchanged structural `isOrphan` predicate) so `orphans` returns only genuine orphans;
  the new `--show-index` flag (`show_index`) opts them back in. No new tool or CLI subcommand.

### Fixed

- **The agent loop flushes the pending transcript round on ANY abort** (#153). The prior fix
  covered only the MaxStep path of the non-streaming turn; now `Run()` flushes on any abort with
  a non-empty pending buffer (MaxStep, cancellation, provider error), and `Session.StreamTurn()`
  gains the same guarantee — the turn-events layer reconstructs each step's assistant message
  from the streamed chunks, records the tool round as pending, and flushes it on any abort, with
  in-memory history rolled back to stay a valid replay input while the persisted transcript
  keeps the complete round. Three regression tests, each proven red against pre-fix source;
  independently adversarially reviewed (8/8 claims confirmed).
- **R6 handoff-staleness self-clears on a handoff update, without a re-baseline** (#100). Patrol now
  measures the handoff against the newest change it GUARDS — the newest desk commit **excluding the
  handoff file itself** (`git log -1 --format=%cs -- . :(exclude)<HANDOFF_PATH>`, new
  `desklib.GitNewestCommitExcluding`). Previously `newest` was the whole-tree newest commit, which
  included the handoff's own update commit: the moment a handoff refresh was committed, that commit
  became the newest and the handoff could never be "current with" it, so the finding could only be
  cleared by a re-baseline. An updated handoff dated on/after the newest guarded change now clears
  R6 at the next patrol. The pure `r6Check(text, newest)` core is unchanged — only how `newest` is
  computed at the caller.

## [0.7.0] — 2026-07-19

### Added

- **PM module — collections, state machine, gates** (D3, `docs/pm-system-v1-spec.md` §3–§4,
  epic #55). The document-gated work graph lands as a feature-gated module (`PM_ENABLED` /
  profile `modules.pm.enabled`; OFF by default): five collections (`items`, `dependencies`,
  `transitions`, `notes`, `desk_config`) created by PROGRAMMATIC migrations only when the
  module is enabled — a librarian-only desk gets no PM collections, physically; the rigid
  queue→work→review→terminal machine with the `blocked` side-state; per-desk editable YAML
  gate rules (validated against the schema-v1/kit vocabulary, now embedded from
  `schema/doctypes.yaml` with a byte-identity drift guard) + cross-cutting traits; the §4.1
  transition path (machine → blocked → claim → gates → write + audit + cascade) with
  refusals that name exactly what is missing and land as `gate_refused` audit rows; typed
  dependency edges with `auto`/`auto-reopen`/`manual`/`permanent` cascade semantics;
  optimistic-concurrency version tokens + claim TTL (`PM_CLAIM_TTL`, default 30m). The
  librarian module's `DocumentValidator` seam is now fully wired (desk-file read +
  frontmatter parse + schema-v1 validation); gate evaluation consumes it and fails closed
  without it. The shipped default gate rules are a seed the owner may re-rule.
- **PM surfaces — CLI, MCP tools, TUI views** (D4, `docs/pm-system-v1-spec.md` §5, epic #55).
  The PM tool family is exposed on all three surfaces over one engine (parity asserted by a
  test): the **twelve PM MCP tools** (`get_context`, `list_items`, `get_item`, `create_item`,
  `update_item`, `transition_item`, `block_item`, `unblock_item`, `add_note`, `link_items`,
  `claim_item`, `release_item`) on `deskkit mcp-serve`; the matching `deskkit pm <sub>` CLI
  group (`context`, `list`, `get`, `create`, `update`, `transition`, `block`, `unblock`, `note`,
  `link`, `claim`, `release`) — JSON-first, present only when the module is enabled; and three
  TUI views (`pm context` landing / `pm board` / `pm item`) mounted into the full-screen TUI.
  `get_context` is the single-call cold-start briefing (active, blocked, stalled, recent
  transitions). The three read tools are always agent-available; the nine write tools are gated
  behind `PM_AUTONOMOUS_WRITES` (default ON) — a desk can make agents read-only over the graph
  while the document gate stays the real safety. PM tools write only the store, never desk files.
  Realtime events emit on transitions under `serve`. New PM env vars: `PM_AUTONOMOUS_WRITES`,
  `PM_STALLED_DAYS` (default 14).
- **`desk-pm` complementary plugin** (D5, `docs/pm-system-v1-spec.md` §6, epic #55). A separate
  Claude Code plugin (shared marketplace) that turns the PM graph into an agent surface over the
  MCP tools: the `pm-session-open`, `pm-advance-item`, and `pm-triage` skills; a `pm-operator`
  agent that operates the graph over the twelve tools but never authors gate documents or writes
  a repo; a `SessionStart` hook injecting `deskkit pm context` (silent no-op when PM is off or
  `deskkit` is absent); and a `.mcp.json` launching `deskkit mcp-serve` with `PM_ENABLED=true`.
  Identity-neutral by construction — no person, org, repo, issue, or desk name is hardcoded.
- Adoption dry-run (`TestAdoptionDryRun`): seeds a scratch store from a neutral manifest via the
  importer, observes `get_context` cold-start, a gate refused-then-satisfied, and a dependency
  auto-unblock, proving the live desk is never written (spec §8.1).

### Changed

- **Chassis rename: `pocket-librarian` → `deskkit`** (D2b, `docs/pm-system-v1-spec.md` §2.10,
  epic #55). The Go binary, its build/install/release artifact names, and the canonical store
  home (`$XDG_DATA_HOME/pocket-librarian/<DESK_NAME>/` → `$XDG_DATA_HOME/deskkit/<DESK_NAME>/`)
  are renamed. On startup, a store still at the old home is moved to the new one automatically
  (one logged line) — no desk loses its store across the rename. `install.sh` falls back to the
  pre-rename asset name for releases up to v0.6.0; ADR 0002's store-path literal carries a dated
  correction. The Go module path (`github.com/example/pocket-librarian`) is unchanged.

### Fixed

- dep-snapshot sort now tiebreaks on `Kind` so two edges sharing a (from,to) pair but differing
  in kind order deterministically, removing a flaky false-failure in the rebuild-reproducibility
  oracle (issue #71).

## [0.6.0] — 2026-07-18

### Added

- **SOP kit library (`kits/`) + schema-v1 doc-type dimension.** The 23 headcase SOP kits
  (guide/template/example per doc type) ported into the repo, neutralized to identity-neutral
  shipped artifacts (0013 S4(a), item 9). Indexed by the root `kits.yaml` manifest with a
  `scripts/check-kits.mjs` drift guard (in `make check` + CI). The kit frontmatter contract now
  lives in `schema/doctypes.yaml` (schema v1's doc-type dimension, successor of the vault
  `types:` model), with the seven previously-unschematized kit types added and the `user-defined`
  nonstandard types deliberately excluded. `kits/` is inside the neutrality-lint scan surface.
  Port + gap dispositions: ADR 0006 (DESK-28) (epic #55,
  D1). The vault copies are frozen (read-only journal).
- **`make install`** — build the version-stamped librarian binary and install it to `~/.local/bin`
  (override with `make install PREFIX=/usr/local`). The one-command update-from-source path.
- **`record_feedback`** — the librarian can log feedback entries into its own store: a `problem`
  entry when a tool fails or a desk convention doesn't fit mid-task, or a `feedback` entry when
  the user asks it to record something. New `feedback` collection (migration 0013), the tool on
  all surfaces (agent, chat, MCP) plus a `record-feedback` CLI subcommand, and a `feedback` kind
  for `query` (open entries, newest first).
- **Chat TUI UX pass.** Per-role left gutters (a thick accent bar for the user, a faint thin bar
  for the librarian), a per-turn `model · latency` footer, and a fuller bubbles/help footer:
  `ctrl+g` toggles the full keybind help, `ctrl+y` copies the last answer's raw markdown (with a
  toast confirmation), and `up`/`down` at the textarea's edge rows walk prompt history, stashing
  and restoring the in-progress draft. Streaming shows a "▼ new output" hint when the transcript
  is scrolled up while new tokens arrive. `NO_COLOR` now also swaps the spinner for a static
  "working…" indicator, alongside the existing color-free rendering.
- **`pocket-librarian init [dir]`** — scaffolds the minimal `_knowledge/profile.yaml` a folder
  needs to work as a desk (desk name from the folder's basename, `root: "."`); idempotent,
  `--force` to overwrite, `--with-env` to also write a `.env` stub. It never creates the store. A
  store-touching command that can't resolve config on an interactive terminal now offers to run
  it ("Set up this folder as a desk? [Y/n]") and continues seamlessly on accept; the root
  `--no-input` flag (and a non-TTY) keeps the prior fail-closed error.

### Changed

- **Core + modules architecture.** The librarian is refactored into a shared `internal/core/`
  (config, store, migrate, mcp, schema, module registry) with librarian-specific code moved under
  `internal/modules/librarian/`, making the librarian the first module on a reusable substrate
  (epic #55, D2). Internal reorganization only — no change to the CLI, MCP tool surface, schema,
  or store layout.
- **Chat TUI migrated to the Charm v2 stack** (bubbletea v2, lipgloss v2, bubbles v2,
  glamour v2 — the `charm.land` modules), recorded as
  ADR 0007 (DESK-29). No feature or visual changes: the
  TUI keeps rendering on the terminal's own background, the theme is still resolved once
  pre-program (flag > env > one background probe), and no terminal query ever runs after
  startup — glamour v2 removing auto-style detection makes part of that guarantee
  structural. Drops the v1-era `termenv` dependency and the global background-cache pin.

### Fixed

- **Unreadable chat answers on light terminals.** `chat`'s full-screen TUI rendered with a single
  fixed dark palette. Add `chat --theme light|dark|auto` (default `auto`, a one-shot terminal-
  background probe run once before the Bubble Tea program starts — never at render time) and a
  `LIBRARIAN_THEME` env override, with precedence flag > env > auto-detect.

## [0.5.0] — 2026-07-18

The `chat` interactive surface graduates from a line REPL to a full-screen terminal UI, and the
streaming substrate underneath it becomes reusable. Recorded as
ADR 0004 (DESK-26).

### Added

- **Full-screen chat TUI.** On a terminal (stdin **and** stdout are TTYs), `librarian chat` now
  opens a full-screen Bubble Tea UI: the answer streams token by token, a finished answer renders
  as markdown, and each tool call collapses to one faint line.
- **Conversation resume and switching.** `ctrl+o` opens a picker of prior conversations to
  resume; `ctrl+n` starts a fresh one — both no-ops while a turn is streaming. Resumed history is
  rehydrated for the model with orphaned user rows collapsed.
- **Cancel an in-flight turn.** `esc` interrupts a streaming turn (the reply is badged
  `(interrupted)`) or closes the resume picker; `ctrl+t` toggles tool-step detail.
- **Streaming event layer** (`agent.Session.StreamTurn`) emitting `token` / `tool_start` /
  `tool_end` / `final` / `error` events over a JSON-taggable `Event` type — the reusable
  substrate the deferred webapp SSE route (ADR 0001, option b) can marshal directly.

### Changed

- **`chat` auto-detects the terminal.** It launches the TUI when interactive and falls back to
  the original line REPL when either end is piped or `--plain` is passed — the non-TTY path is
  byte-for-byte the previous REPL.
- `Turn()` is now a thin drain over `StreamTurn`, so the REPL and the TUI share one persistence
  path.

### Fixed

- **Multi-turn transcript persistence.** The persistence high-water-mark was per-session, so
  multi-turn sessions duplicated the prior assistant row (no-tool turns) or dropped the new user
  row (tool turns). It is now re-baselined per turn and guarded by an exactly-once transcript
  regression test.
- **Zero-argument tool calls** (e.g. `sweep`, `patrol`) streamed no argument deltas, leaving
  `ArgumentsInJSON == ""` and killing the turn on unmarshal; a normalizing adapter now maps `""`
  → `"{}"` at tool registration.

## [0.4.0] — 2026-07-17

First tagged release — the distribution and hardening baseline. (Pre-`0.4.0` development history,
including the `v0.0.1-alpha` tag, lives in the git log and the merged PRs.)

### Added

- **Curl-able installer** (`install.sh`) and a tag-triggered **release workflow** that
  cross-compiles the librarian for darwin/linux × amd64/arm64 (pure-Go), builds the plugin
  bundle, and publishes a GitHub release with sha256 `checksums.txt`.
- **Unified repo version.** One canonical `VERSION` drives the librarian binary (via ldflags) and
  the three shipped manifests (`plugin.json`, `plugin/package.json`, `marketplace.json`),
  drift-guarded in CI and pre-commit.
- **Makefile task interface** (`make help`) as the canonical entry point, plus lefthook
  pre-commit hooks mirroring CI, and three user guides + demo media.
- **XDG store home + desk open-guard** (ADR 0002, DESK-24):
  stores default to `$XDG_DATA_HOME/pocket-librarian/<DESK_NAME>/`; a store refuses a mismatched
  desk name.
- **Line-REPL `chat` + trigger wake layer** (ADR 0001, DESK-23).

### Changed

- **Tool commands self-initialize the store** (ADR 0003, DESK-25):
  `sweep`/`query`/`patrol`/`chat`/etc. run the app migrations idempotently at first touch, so a
  fresh desk needs no manual `migrate up`.

### Fixed

- `query` on an uninitialized store now returns an actionable message instead of a bare
  `sql: no rows in result set`.
- Record-original-first is no longer capped at PocketBase's 5000-char default; content fields are
  widened so large desk files record and restore byte-exact.

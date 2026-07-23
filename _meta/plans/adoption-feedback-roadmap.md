# Adoption-feedback roadmap — usability + consolidation

Source: early-user feedback (2026-07-23) that desk-standard is "super-confusing" to adopt, plus two
owner rulings made the same day. This is the sequencing doc — what to build in what order. Each wave
gets its own `_meta/plans/<slug>/` folder (issue-body.md + plan.md) and a tracking issue when picked
up. State lives on the board; build detail lives here. See [`README.md`](README.md) for the loop.

Ordering principle: **user-facing value + risk**, never a release cut or public launch
(owner-gated — do not sequence around 1.0.0/#87 or going public).

## The feedback (verbatim, grouped)

| Raw feedback | Theme |
|---|---|
| "plugin is super-confusing, maybe make it one" / "why two plugins and two MCP servers?" | **Consolidate** — one plugin, one MCP server |
| "deskkit CLI too confusing" / "CLI menu is alphabetic instead of by functionality" | **CLI UX** — command groups |
| "no command to see which desks I have and pick one" | **Desk discovery** |
| "what is configurable?" / "central config for all desks? store LLM key + model there?" / "per-project too?" | **Config** — central + per-project |
| "visually explore the DB data?" | **Data browser** (discoverability — it already exists) |

## Owner decisions (2026-07-23)

1. **Consolidation = "everything on deskkit."** Move the 4 TS profile tools into the Go binary;
   retire the TS MCP server; ship ONE plugin + ONE MCP server (`deskkit mcp-serve`). Reverses ADR
   0016 → needs a superseding ADR; withdraws `docs/development/ts-proxy-design.md`; moots #199 and
   the ts-proxy build; simplifies #12 (OpenCode). See [[consolidation-single-binary-decision]].
2. **Central config stores the actual API key** — machine-local `~/.config/deskkit/config.yaml`
   (0600) holds `llm.provider` + `llm.model` + `llm.api_key`.

## Current-state anchors (verified 2026-07-23)

- Two bundles: `plugins/desk-standard/` (bun `mcp/server.js`, 4 profile tools) +
  `plugins/desk-persona/` (`deskkit mcp-serve`, `MCP_MODULES=librarian,pm`, 17 tools + 2 agents + 3
  PM skills + SessionStart hook); both in `.claude-plugin/marketplace.json`.
- CLI flat-alphabetical (cobra default) — wired in `librarian/cmd/deskkit/main.go:758`
  (`registerToolCommands`) + `pm.go:54`; zero cobra groups anywhere.
- No desk-list command; stores at `$XDG_DATA_HOME/deskkit/<DESK_NAME>/`
  (`librarian/internal/core/store/storedir.go:24,41`).
- No central config, no `deskkit config` command; resolution env > `.env` > per-desk
  `_knowledge/profile.*` > default (`config.go:66`, `pick` `:171`); LLM key env-only
  (`provider/adapter.go:82`).
- Data browser EXISTS — PocketBase admin `:8090/_/` via `deskkit gui` (`main.go:1074`); only
  discoverable via README/Makefile; superuser-gated.

## Waves

### Wave 1 — Adoption quick wins  *(parallelizable, Go-lane, no architecture change)*
Highest visible payoff per unit effort; each item independent.
- **CLI command groups** — `RootCmd.AddGroup` + per-command `GroupID` in `registerToolCommands`/
  `registerPMCommands`. Groups: Setup & config · Inspect · Fix · Work graph · Agent · Admin
  (Admin commands candidates for `Hidden`). *Done when:* `deskkit --help` renders grouped sections.
- **`deskkit desks`** — enumerate `$XDG_DATA_HOME/deskkit/*`, mark the current desk. Add
  `store.ListDesks()` reusing `dataHome()`. *Done when:* lists name + store path + current marker.
- **`deskkit config` command** (reads existing sources only): `config show` (resolved values + a
  source column: env/profile/default; key masked), `config path`, `config edit`. *Done when:*
  `config show` prints every resolved value and where it came from.
- **GUI discoverability** — `gui` in the Inspect group with "open the visual data browser" help;
  pointer from `config show`/`desks`/first-run; smooth first-run superuser. *Done when:* a user who
  never read the README can find the browser from the CLI.

### Wave 2 — Central config  *(extends Wave 1's `config` command)*
- New `$XDG_CONFIG_HOME/deskkit/config.yaml` (0600): `llm.provider`, `llm.model`, `llm.api_key`,
  optional `default_desk`, `pb_url`.
- Layer into `config.Load`: precedence **env > per-desk profile > central > default** (thread a
  central leg through `pick`). Keep `Config` secret-free — `resolveAPIKey` reads the central key
  after env, before failing loud.
- `config set`/`config edit` write the central file. *Done when:* a key in the central file (env
  unset) is picked up by `agent`/`chat`; `config show` proves the precedence; file is 0600.

### Wave 3 — The consolidation  *(GATE: superseding ADR accepted first)*
- New ADR supersedes 0016 (collapse onto Go; profile tools in Go; TS MCP server retired); mark
  `docs/development/ts-proxy-design.md` withdrawn; reconcile ADR 0014 (one bundle).
- Reimplement the 4 profile tools in Go behind a `profile` value in `MCP_MODULES`
  (`librarian/internal/core/mcp/server.go`); reuse `config.DiscoverProfile`/`LoadProfile` +
  `schema/profile.schema.yaml` + `templates.Render` + the librarian `files` index.
- Retire the TS server; drop its emit from `make package` (`plugin/package.json`).
- One marketplace bundle: one plugin dir, one `.mcp.json`
  (`deskkit mcp-serve`, `MCP_MODULES=profile,librarian,pm`), merged skills + agents + hook; one
  entry in `.claude-plugin/marketplace.json`.
- Update `docs/tool-surface.md` (+ `scripts/check-tool-surface.mjs`), `CLAUDE.md`, `librarian/README.md`.
- *Done when:* fresh install = one plugin, one MCP server exposing all tools; the 4 profile tools
  work via the Go server against a scratch desk; `make package` + `git diff` clean; neutrality +
  tool-surface + purity gates green; `make e2e` passes; ADR 0016 marked `Superseded-by`.

### Wave 4 — Existing backlog, reconciled
- Pull the cheap confusion-killers from onboarding epic **#104** forward: D2 mental-model map, D4
  reference cards, D5 failure-modes runbook.
- Then owner-sequenced: **#36** (SOP library), **#127** (exec-output triggers), **#12** (OpenCode —
  simpler post-collapse), schema-v2 design items (**#130**).
- Reconcile on filing Wave 3: note **#199** + the ts-proxy build as mooted by the superseding ADR.

## Provenance
Design session + owner rulings 2026-07-23. Full plan snapshot (machine-local, not tracked):
`~/.claude/plans/in-simple-terms-what-silly-kurzweil.md`. Decisions recorded in
[[consolidation-single-binary-decision]] and [[feedback-no-public-1.0.0-nagging]].

_Index of the desk-standard docs, split by **audience**: the pages a desk owner reads to run a desk,
and the pages a contributor reads to build the products._
Status: active

# Documentation

Two products over one shared schema — the **plugin** (Claude Code plugin + MCP server) and the
**librarian** (`deskkit` Go binary). The docs are organized by **who you are**, not by verb.

## For desk owners — running a desk

**Audience: you run a desk; you are not building the products.** These pages assume no build
toolchain and only a minimal terminal — an install command and `deskkit` launched inside your desk,
with the TUI and the Claude-session skills carrying the rest. No `make`, no Go/bun toolchain, and no
environment-variable exports are needed to reach your first sweep and patrol.

| Doc | What it covers |
|---|---|
| [getting-started.md](getting-started.md) | Install the plugin and `deskkit`, stand up your desk (guided), run your first sweep + patrol, and tour the TUI. |
| [plugin-guide.md](plugin-guide.md) | The four skills as user journeys — when to reach for each and what it does to your desk. |
| [librarian-guide.md](librarian-guide.md) | The daily loop: sweep → patrol → fix → byte-exact restore. |
| [pm-guide.md](pm-guide.md) | The PM work graph: the phase machine + gates, the TUI views and their keys, opting out, and the CLI / MCP / TUI / `desk-persona` plugin surfaces. On by default. |
| [../librarian/README.md](../librarian/README.md), [../plugin/README.md](../plugin/README.md), [../plugins/desk-persona/README.md](../plugins/desk-persona/README.md), [../schema/README.md](../schema/README.md) | Per-product operator detail. |

Demo GIFs referenced by these guides live in [`media/`](media/). (Their VHS source tapes are a
development artifact — see below.)

## For contributors — building the products

**Audience: you build, test, release, or change the products.** These pages carry the toolchain:
`make`, `bun`, `go`, the gates, the release flow, the specs, and the ADRs.

| Doc | What it covers |
|---|---|
| [development/](development/) | Contributor overview — build/test gates, media regeneration, release flow. |
| [development/install-and-build.md](development/install-and-build.md) | Build the librarian from source, pull a prebuilt release asset (incl. the private-repo interim), and the env-var + store-path overrides. |
| [development/releasing.md](development/releasing.md) | How to version and cut a release (both products, one version). |
| [development/tapes/](development/tapes/) | VHS `.tape` sources that regenerate the demo GIFs in `media/`. |
| [development/ts-proxy-design.md](development/ts-proxy-design.md) | The TS→deskkit proxy design (ADR 0016 Option B implementation design). |
| [pocket-librarian-v1-spec.md](pocket-librarian-v1-spec.md) | The librarian's product + technical build spec. |
| [pm-system-v1-spec.md](pm-system-v1-spec.md) | The PM-system (document-gated work graph) product + technical build spec — the core+modules refactor, PM module, gates, surfaces, and plugin. |
| [tool-surface.md](tool-surface.md) | The authoritative map of every tool/command across all three surfaces (librarian CLI, librarian MCP by gate, plugin TS server) with empirically-verified counts. |
| [decisions/](decisions/) | Architecture Decision Records (append-only; cited from code and docs). |
| [../CHANGELOG.md](../CHANGELOG.md) | What changed in each release. |

The spec and the ADRs live at the top of `docs/` (not under `development/`) because they're
canonical references cited from code, skills, and the neutrality-lint allowlist — their paths are
kept stable on purpose.

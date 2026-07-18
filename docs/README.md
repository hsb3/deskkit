_Index of the desk-standard docs, split into two tracks: **using** the products and **developing**
them._
Status: active

# Documentation

Two products over one shared schema — the **plugin** (Claude Code plugin + MCP server) and the
**librarian** (`pocket-librarian` Go binary). The docs are organized by what you're here to do.

## Using it

Read these to install and run the products. No build toolchain needed beyond the getting-started steps.

| Doc | What it covers |
|---|---|
| [getting-started.md](getting-started.md) | Install the plugin, fill your profile, build the librarian, run your first sweep + patrol. |
| [plugin-guide.md](plugin-guide.md) | The four skills as user journeys — when to reach for each and what it does to your desk. |
| [librarian-guide.md](librarian-guide.md) | The daily loop: sweep → patrol → fix → byte-exact restore. |
| [../librarian/README.md](../librarian/README.md), [../plugin/README.md](../plugin/README.md), [../schema/README.md](../schema/README.md) | Per-product operator detail. |

Demo GIFs referenced by these guides live in [`media/`](media/). (Their VHS source tapes are a
development artifact — see below.)

## Developing it

Read these to build, test, release, or change the products.

| Doc | What it covers |
|---|---|
| [development/](development/) | Contributor overview — build/test gates, media regeneration, release flow. |
| [development/releasing.md](development/releasing.md) | How to version and cut a release (both products, one version). |
| [development/tapes/](development/tapes/) | VHS `.tape` sources that regenerate the demo GIFs in `media/`. |
| [pocket-librarian-v1-spec.md](pocket-librarian-v1-spec.md) | The librarian's product + technical build spec. |
| [pm-system-v1-spec.md](pm-system-v1-spec.md) | The PM-system (document-gated work graph) product + technical build spec — the core+modules refactor, PM module, gates, surfaces, and plugin. |
| [decisions/](decisions/) | Architecture Decision Records (append-only; cited from code and docs). |
| [../CHANGELOG.md](../CHANGELOG.md) | What changed in each release. |

The spec and the ADRs live at the top of `docs/` (not under `development/`) because they're
canonical references cited from code, skills, and the neutrality-lint allowlist — their paths are
kept stable on purpose.

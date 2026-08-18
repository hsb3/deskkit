_The documentation index for deskkit, split into two tracks: **Using** (run a desk) and
**Developing** (build the tools). The front door is the root [`README.md`](../README.md); the
canonical direction is [`development/CHARTER.md`](development/CHARTER.md) (it wins any conflict)._
Status: active

# Documentation

## Using — run a desk

- [`usage/getting-started.md`](usage/getting-started.md) — install, fill your profile, build the deskkit binary, first sweep + patrol.
- [`usage/plugin-guide.md`](usage/plugin-guide.md) — the desk-shaping skills as user journeys: when to reach for each, what you get.
- [`usage/librarian-guide.md`](usage/librarian-guide.md) — the daily loop: sweep → patrol → fix → byte-exact restore.
- [`usage/deskkit-reference.md`](usage/deskkit-reference.md) — the operator/runtime reference for the binary: quick start, store location, machine-wide config, the LLM/API key, chat, the browser session, MCP, and the supervised write path.
- [`usage/pm-guide.md`](usage/pm-guide.md) — the PM work graph: enable it, the phase machine + gates, and the CLI / MCP / TUI / `deskkit` plugin surfaces.

## Developing — build the tools

- [`development/CHARTER.md`](development/CHARTER.md) — the canonical page: what the project is, the 1.0.0 direction, the precedence rule.
- [`development/README.md`](development/README.md) — contributor overview: build, test, regenerate media, cut a release.
- [`development/install-and-build.md`](development/install-and-build.md) — toolchain floors and the build lanes in detail.
- [`development/specs/`](development/specs/) — the build specs (librarian + PM technical/product specs, the tool-surface map, the agent-integration contract, the v2 element-model draft). Several are read by CI gates.
- [`development/docs-layout.md`](development/docs-layout.md) — the docs layout contract: what lives where, which paths are load-bearing, and how the working desk differs.
- [`pattern.md`](pattern.md) — the single-binary shape (CLI + embedded store + MCP + web + plugin), the config ladder, and the auth model, written for a sibling application to copy.
- Architecture Decision Records live on the project board as `DECISION` tasks (moved off disk
  2026-08-18); cited from docs and code as a bare `ADR NNNN`.

Working-desk material — session handoffs, plans, research — lives on the project board, not in
this repo (the `_meta/` tree that used to hold it was removed in 2026-08; it is in git history).
The only working state still on disk is `.claude/` (agent config + memory), which is not published
documentation; see [`development/docs-layout.md`](development/docs-layout.md) for the boundary.

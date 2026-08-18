# E2E system-behaviour suite

_Purpose: prove deskkit behaves as **one system** — the deskkit binary, the shipped plugin
bundle, the librarian, and the PM module — by walking the whole chain on a throwaway desk._
Status: active

## Entry point

```
bash e2e/e2e.sh
```

Runs offline against a fresh `mktemp` scratch desk (never a real store, no network, no secrets)
— with one exception: the build step also builds the embedded SPA (the Makefile's `spa`
target), and the FIRST run on a machine needs network once, to `npm ci` into `web/node_modules`.
Every run after that reuses the cached `node_modules` and stays fully offline.
Reuse a prebuilt binary to skip the Go build (and the SPA build) entirely:

```
DESKKIT_BIN=/path/to/deskkit bash e2e/e2e.sh
```

Exit status is non-zero if any check FAILs. `SKIP`s (steps that genuinely need an LLM key) never
fail the run but are always printed — never silently passed.

## The chain it walks

| Step | Link | Proves |
|---|---|---|
| 10 | cold-start + profile | `deskkit init` scaffolds a working desk; store self-initialises (ADR 0003); the scaffolded profile validates against schema v1 through the MCP `profile_validate` tool |
| 20 | librarian sweep → patrol → fix → restore | rule detection (R1/R3), record-original-first `propose-fix`, `apply-fix`, and byte-exact reversible `restore` (ADR 0014 boundary) |
| 30 | PM graph | PM ships **default-on**; create → legal transition; the phase gate refuses illegal skips; blocking/cascade refuses a blocked item's advance |
| 40 | shipped surfaces | one Go MCP server carries the whole tool core (21 default / 9 with PM off / 12 pm-gated / 4 profile-gated) and answers a real `tools/call`; skills + agents present; the SessionStart hook emits a cold-start briefing |
| 45 | embedded web | starts its own `serve` against the scratch store; the embedded SPA shell at `/`, the `/desk/chat` SPA-index fallback, the loopback `/desk/bootstrap` token endpoint (refused cross-site), and the PocketBase REST API staying fail-closed until that token is presented |
| 50 | release-shaped | VERSION ↔ manifests sync, CHANGELOG coverage, ldflags version-stamping, and a single self-contained marketplace bundle with no generated JS/schema copies |

## What it deliberately does NOT cover (skip-with-notice)

The LLM-driven surfaces — the eino **agent loop** (`deskkit agent`) and the **chat TUI**
(`deskkit chat`) — need a real API key and are non-deterministic, so they are `SKIP`ped with a
notice. Exercise them by hand via [`../examples/agent-loop.sh`](../examples/agent-loop.sh) (which
makes real, billed LLM calls — see [`../examples/README.md`](../examples/README.md)).

## How it relates to the other gates

- **`verify.sh`** (`make verify`) is the deep librarian-only fix-chain gate (spec §9.4
  fixtures, dead-store fault, ignore-list). This suite walks the fix chain at system granularity
  and adds the cross-module links (cold-start, PM, plugin surfaces, release shape) verify.sh omits.
- **[`../examples/pm-walkthrough.sh`](../examples/pm-walkthrough.sh)** is the deep PM-only
  walkthrough (MCP transition sequence), free and offline. Step 30 here is the system-level PM
  slice.

## Layout

```
e2e.sh                driver: scratch lifecycle, build, source steps, summary/exit
lib.sh                shared helpers (check/skip/note/section/dk/mcp_go) + fixture seeder
steps/[0-9]*.sh       chain links, sourced in numeric order; each is one self-contained link
```

Step files are **sourced** by the driver in one shell; they share the helper functions and the
`E2E_*` scratch-desk state and must stay identity-neutral (the tree is scanned by
`scripts/check-neutrality.mjs`).

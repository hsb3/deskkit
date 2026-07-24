# Cohesion assessment — desk-standard as one system

_Purpose: record what the E2E system-behaviour suite exercised, and where the seams between the
modules and surfaces are coherent vs incoherent — each finding fixed-in-scope, filed, or
accepted with reasoning._
Status: active — 2026-07-21

Companion to the value evaluation in this folder. The evidence base is
`librarian/e2e/e2e.sh` (`make e2e`), which walks the whole system on a throwaway desk and, on
this branch, reports **45 checks — 43 PASS, 0 FAIL, 2 SKIP** (the 2 skips are the LLM-gated
surfaces, below). The deep single-lane gates that back it are `librarian/verify.sh` (`make verify`,
55/55) and `librarian/dogfood-pm.sh`.

---

## What the E2E run exercises (the chain, end to end)

| Link | Surfaces / modules crossed | Behaviour proven |
|---|---|---|
| Cold-start + profile | `deskkit init`, store bootstrap, plugin core (TS MCP) | an empty folder becomes a working desk; the store self-initialises (ADR 0003); the scaffolded profile validates against schema v1 through `profile_validate` |
| Librarian chain | CLI + tool core + store + DocumentValidator | `sweep → patrol → propose-fix → apply-fix → restore`; rule detection (R1/R3), record-original-first, byte-exact reversible restore (ADR 0014 boundary) |
| PM graph | PM module + state machine + gate engine + store | PM **default-on** (no env flag); create → legal transition; the phase gate refuses the `queue→terminal` skip; a `blocks` edge refuses a blocked item's advance; a document-gated `work→review→terminal` path |
| Plugin surfaces | TS plugin MCP, Go librarian/PM MCP, skills, agents, SessionStart hook | the TS MCP exposes the 4 core tools and resolves a profile key; the Go MCP default mount exposes 17 tools and the `MCP_MODULES=pm` mount exactly 12; all 7 shipped skills + both agents present; the hook emits a cold-start briefing |
| Release-shaped | version-sync, changelog, ldflags stamp, marketplace bundle | VERSION ↔ 5 manifests, CHANGELOG coverage, `--version` reports the stamped VERSION, the committed `claude-plugin/` bundle is self-contained |

**The seams the issue named are proven under real workflows:** the **DocumentValidator** (the
PM gate reads the pointer document's frontmatter to permit `work→review` — E2E check 30), the
**tool-core** (the same operations surface identically over CLI and MCP — E2E checks 24-35), and
the **store** (self-init, record-original-first revisions, PM work graph all on one embedded
PocketBase — E2E checks 5, 10-23, 24-31).

**Bounded coverage (stated, not hidden):** the two LLM-driven surfaces — the eino **agent loop**
(`deskkit agent`) and the **chat TUI** (`deskkit chat`) — are `SKIP`ped, since no API key is
available and their output is non-deterministic. They are covered by hand via
`librarian/dogfood-agent.sh`. Everything else in the chain runs offline and is asserted.

---

## Seam findings

Coherence is high — every cross-module link the E2E walks behaves as one system, and all four
repo gates are green. The findings below are seams a careful integrator should know; none blocks
1.0, and the incoherent ones are naming/documentation, not behaviour.

### Accepted with reasoning (coherent, but worth naming)

1. **CLI and MCP name the same operation differently.** The PM operations are `deskkit pm create`
   / `pm transition` / `pm block` on the CLI but `create_item` / `transition_item` / `block_item`
   as MCP tools; likewise the librarian's `propose-fix` (CLI) is `propose_fix` (MCP). This is a
   real naming seam across surfaces, but a *justified* one: MCP tools live in one flat namespace
   (so they need the `_item` qualifier and snake_case), while the CLI groups them under a `pm`
   parent (so the noun is redundant). The mapping is documented in `docs/development/specs/tool-surface.md`. Accept.

2. **"Desk name" has two independent sources.** Store resolution uses the `DESK_NAME` env var
   (config rule 2), while the profile carries its own `desk.name` — and `deskkit init` derives
   that `desk.name` from the **desk directory's basename**, not from `DESK_NAME`. On a normal desk
   they coincide (the folder is named after the desk); they can diverge if you `init` into a
   directory whose basename differs from the desk name you resolve the store by. Behaviour is
   correct and each source is documented (CLAUDE.md "Configuration resolution"), but the coupling
   is implicit. The E2E now names its scratch desk to make the two agree, which is also the
   honest recommendation. _Finding to file: a one-line note in the init/using docs that
   `init` scaffolds `desk.name` from the directory basename._

3. **Profile discovery is cwd-relative (walk-up) on every surface.** Both the TS plugin MCP and
   the Go binary discover `_knowledge/profile.*` by walking up from the current working directory.
   This is by design (config rule 3) and fails loud with an actionable message, but it is a seam:
   running the plugin MCP server from a non-desk cwd yields "no profile found," not a silent
   wrong answer. The E2E exercises this correctly by running the TS server from the desk cwd.
   Accept (fail-loud behaviour is the right posture).

4. **`apply-fix` "move" leaves a pointer stub, not a bare delete.** A moved doc appears at BOTH
   its new path (the real content) and its old path (a `type: pointer` stub reading "Moved to
   …"). This can momentarily read like a duplicate, but it is the intended record-original-first
   behaviour and is fully reversible (`restore` removes the moved copy and restores the source
   byte-identically — E2E checks 21-23). Accept.

### Documentation clarity (non-blocking, file for a doc pass)

5. **The MCP default-mount count is 5 in one sub-table and 17 in the prose.** With PM default-on,
   the live default librarian MCP mount exposes **17** tools (5 librarian + 12 PM), which the E2E
   asserts (check 34) and `docs/development/specs/tool-surface.md` states in prose (and pins with
   `TestToolSurfaceDoc_MCPCounts`). But the doc's §2.1 sub-table still lists a "Librarian MCP
   (default) … 5" row (the PM-*off* matrix case). The two are reconciled by the surrounding prose,
   so this is a **consistency-of-presentation nit, not a behaviour mismatch** — a reader skimming
   only the sub-table could misread the live default. _File a doc-clarity tweak; no code impact._

### Fixed in scope (this lane)

6. **E2E harness: the TS-MCP helper could not reach the scratch desk's profile.** While building
   the suite, `lib.sh`'s `mcp_ts` ran the plugin server from the repo's `plugin/` dir, so its
   walk-up discovery never found the scratch desk — a `profile_get` through it always hit the
   fail-loud path. Fixed in `librarian/e2e/lib.sh` (run the server from the desk cwd) and
   `librarian/e2e/e2e.sh` (name the scratch desk to match `DESK_NAME`). This is a harness fix, not
   a product defect — the product's cwd-relative discovery is correct (finding 3); the harness
   simply has to invoke it from inside the desk, as a real session does.

---

## Verdict

desk-standard behaves as **one coherent system**: every cross-module seam the E2E walks
(cold-start, profile, librarian fix chain, PM work graph, plugin surfaces, release shape) passes,
and all four repo gates are green on this branch. The incoherences found are **naming and
documentation seams** (findings 1, 5) and **implicit couplings worth a doc line** (findings 2, 3),
not behavioural breaks. No finding blocks 1.0; findings 2 and 5 are worth filing as small doc
tweaks before the release.

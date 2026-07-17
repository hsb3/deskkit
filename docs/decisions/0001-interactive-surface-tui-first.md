_ADR for the pocket-librarian on-demand interactive surface: how a human reaches a live stewardship session._
Status: Accepted (corrected 2026-07-17) — 2026-07-16

# 0001 — Interactive surface: terminal session first, PocketBase-served webapp deferred

> **Correction (2026-07-17):** two clarifications to this record, neither of which changes the
> decision.
> 1. **"TUI" here means a line-oriented REPL, not a full-screen/graphical terminal UI.** What
>    shipped is `chat` — a `bufio.Scanner` + prompt read-eval loop (`cmd/pocket-librarian/main.go`,
>    `runChat`), with no bubbletea/tview/tcell dependency. The word "TUI" at the Decision heading
>    and in the rationale below (and the `-tui-` filename slug) is loose usage; read it as
>    "terminal session / REPL," the accurate term this ADR already uses in the Decision body.
> 2. **The "≤2 commands (`migrate up`, then `chat`)" figure below is superseded by ADR 0003.**
>    Tool commands now self-initialize the store, so `chat` needs no prior `migrate up` — the path
>    is one command.

## Context

The v1 spec (§7.4) records a **research spike, not a decision**: "whether a chat surface for
the agent can be served without a separate frontend" via one of three options, explicitly
listed as out of Phase 1/2 build scope with "no design committed in this spec." Meanwhile the
product now needs an **on-demand, human-friendly, local** way to converse with the librarian —
inspect findings, direct a patrol, and review/approve proposed fixes in a multi-turn session —
in addition to the autonomous triggers (§2.4) and the one-shot `agent "<instruction>"` CLI.

This ADR converts the §7.4 non-decision into a committed direction for that interactive surface.

### Requirements the surface must satisfy

1. **Human-friendly** — a real multi-turn conversation, not a single-shot invocation.
2. **On-demand** — a person starts it when they want it; it is not a always-listening daemon.
3. **Local** — runs against the local `pb_data` on the operator's machine; no hosted service.
4. **Preserves the settled identity** — pocket-librarian is a *single Go binary* that is
   simultaneously the PocketBase server and the agent (spec §1.1, §2.1). A surface that forces a
   second build artifact, a Node toolchain, or a separate frontend deploy erodes that identity.
5. **Respects the write boundary** (§5.4, §5.5) — nothing interactive may mutate desk files
   except through the propose→apply path; `restore` stays supervised/CLI-only; `apply_fix` stays
   gated by `LIBRARIAN_AUTONOMOUS_WRITES`. The surface must reuse the existing gated tool set,
   not open a new write path.
6. **Stays in the stewardship lane** (§1.3 non-goal: "Not a general chat assistant"). The surface
   inherits the librarian's data-backed system prompt and gated tools; it does not become a
   free-form assistant.

## The three §7.4 options, evaluated

The spike framed the surface as a choice between three ways to serve a *web* chat page. Evaluated
against the requirements:

| Option | What it is | Assessment against the requirements |
|---|---|---|
| **(a) Extend the built-in admin GUI** | Add a chat panel inside the PocketBase superuser admin UI. | The admin GUI is **superuser-gated** (§10.3), so this inherits that gating — the spike itself flags it "is not a public chat surface." It also means customizing a vendored, upstream-owned SPA (fragile across PocketBase upgrades) and hand-editing generated GUI assets. Meets *local*; strains *human-friendly* (login friction) and *single-binary identity* (couples us to admin-UI internals). |
| **(b) Custom Go route serving a purpose-built page** | Mount an HTML/JS chat page in `OnServe`, like the existing `GET /api/desk/summary` route (§2.4). | Keeps everything in the one binary (page can be `//go:embed`'d). But it requires **writing and maintaining front-end code** (HTML/JS, streaming transport, auth), and only works while `serve` is running — coupling an on-demand human session to the long-running server process. Medium cost, medium fit; real front-end surface area for a first cut. |
| **(c) Separate PocketBase-served React frontend** | A standalone React SPA talking to the PB API. | **Highest cost.** Introduces a Node/React build, a second deploy artifact, and a bundle to ship — directly against the single-binary identity. Ties to the §7.4 "skills-registry" maybe-someday item. Best long-term *human-friendly* ceiling, worst *identity* and *complexity* fit for now. |

**Common finding:** all three options assume a **web** surface, and each adds a front-end
maintenance burden (a, b) or a whole second toolchain (c). None is the cheapest path to a
*human-friendly, on-demand, local* multi-turn session, because the binary already contains
everything needed to run one: the eino ReAct loop, the gated tool set, and the data-backed system
prompt. The missing piece is not a web page — it is a **conversational entry point** over the loop
that already exists.

## Decision

**Ship a terminal session (TUI) first; defer the PocketBase-served webapp as the recorded
follow-on.**

- **Now — terminal session.** Add a `chat` subcommand that runs a multi-turn REPL over the
  existing eino agent loop, in the same single binary, against the local `pb_data`. It reuses the
  gated tool set (`tools.AgentTools` — `restore` never exposed, `apply_fix` only when the write
  gate is on) and the existing data-backed system prompt, so the write boundary and the
  stewardship-lane boundary (§1.3) hold with no new surface area. This satisfies all six
  requirements with the least new code and **zero erosion of the single-binary identity** — no new
  build artifact, no front-end toolchain, no second process required for a human to talk to the
  librarian on demand.

- **Later — PocketBase-served webapp.** A browser surface remains desirable for non-terminal
  users. When built, prefer **option (b)** (a custom Go route serving an embedded page in
  `OnServe`) over (a) and (c): it preserves the single-binary identity, avoids the superuser-gating
  constraint that disqualifies (a) as a general surface, and avoids the second-toolchain cost of
  (c). Option (c) is only justified if the separate-frontend "skills-registry" maybe-someday item
  (§7.4) is also pursued, at which point the React frontend is shared. This follow-on is **deferred,
  not rejected** — it is the natural next increment once the TUI proves the interaction model.

### Rationale summary

- The requirement is a *conversation*, not a *web page*; the binary can already hold a conversation.
- TUI is the lowest-cost path that keeps the single-binary identity intact and the write/stewardship
  boundaries unchanged (it reuses the gated loop verbatim).
- Deferring — not killing — the webapp keeps the door open and records the preferred option (b) so
  the follow-on does not re-run this spike from scratch.

## Consequences

- A `chat` (session) subcommand exists: multi-turn REPL over the agent loop, reachable from a built
  binary in ≤2 commands (`migrate up`, then `chat`). Documented in `librarian/README.md`.
- The interactive surface adds **no new write path**: it is built from `tools.AgentTools(cfg)`, so
  the §5.4/§5.5 boundary is enforced by the same exclusion the autonomous loop and MCP surface use.
- Spec §7.4 is annotated with a pointer to this ADR (the non-decision is superseded in direction for
  the interactive surface; the webapp remains a recorded, deferred follow-on).
- Reversible: the deferred webapp can be built later without unwinding the TUI — both are entry
  points onto the same in-process loop.

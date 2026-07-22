**Separate user docs (non-developers) from developer docs — and hold the user track to a
register non-developers will actually tolerate: minimal terminal, TUI-first, guided
walkthroughs. Owner-raised 2026-07-22.**

## Evidence

The split nominally exists — `docs/README.md` indexes two tracks ("Using it" / "Developing
it") — but the **Using track is written in developer register**, so the separation is a label,
not a contract:

- `docs/README.md` §Using claims "No build toolchain needed beyond the getting-started steps,"
  while `getting-started.md` §3 has the user `cd librarian && make build` a Go binary (toolchain
  floor: Go 1.25), or else run a `uname | tr` / `sed` / authenticated-`gh` download pipeline.
- `getting-started.md` also assumes: `$EDITOR` on raw YAML (§2), env-var override lore
  (`DESK_ROOT`/`DESK_NAME`, "env always wins") and XDG store paths (§4), and JSON transcripts as
  the primary output the reader is shown (§4).
- The product's own target persona makes this acute: an **executive desk** owner is a
  non-developer by definition. The tolerable surface for that persona is roughly: one install
  command, one `deskkit` launch inside their desk, and everything else guided — in the TUI or in
  a Claude session via the skills.
- Related friction filed the same day: #189 (TUI discoverability + guided first-run) — the TUI
  has to carry the guided burden the user docs point at.

## Scope

1. **Rule the audience contract (the real deliverable).** Two named audiences with a register
   rule each:
   - **User track** — assumes no toolchain and minimal terminal: at most an install one-liner
     and `deskkit` launched inside the desk. Flows are shown TUI-first or as Claude-session
     skill journeys; CLI incantations, env vars, and JSON output move to appendices or the
     developer track. Prebuilt binary is the only install path shown.
   - **Developer track** — keeps `make`/`bun`/`go`, gates, release flow, specs, ADRs (unchanged;
     `docs/development/` already exists and is honest).
2. **Restructure the index around audience, not verb.** `docs/README.md` labels the tracks by
   reader ("I use a desk" / "I build the products"), and each Using-track page states its
   assumed reader at the top. Whether user pages move under a `docs/user/` dir or stay top-level
   is an implementation call — but **spec + ADR paths must not move** (they're load-bearing:
   cited from code, skills, and the neutrality allowlist).
3. **Rewrite `getting-started.md` as the user path**: install prebuilt `deskkit` (one command;
   note the pre-public interim), install the plugin from the marketplace, launch, be guided.
   Build-from-source, the release-asset shell pipeline, env-var overrides, and store-path lore
   relocate to the developer track (e.g. `development/` or the librarian README).
4. **Sweep the rest of the Using track** (`plugin-guide.md`, `librarian-guide.md`,
   `pm-guide.md`, product READMEs) against the register rule: TUI/skill flow first, terminal as
   fallback, no unexplained JSON as the primary reading experience.

## Acceptance

- A non-developer reaches first sweep + patrol using only the user track: **no `make`, no Go/bun
  toolchain, no env-var exports, no shell pipelines** — an install command and `deskkit` inside
  their desk, with the TUI/skills carrying the rest.
- `docs/README.md` names the audience for each track; no Using-track page links into
  developer-register content as a required step.
- The "no build toolchain needed" claim in the index becomes true.
- Spec + ADR paths unchanged; `CLAUDE.md`'s docs pointers updated in the same change if any page
  moves.

_Relationships: #189 owns the in-TUI guided/discoverability work this track leans on; #104 (the
post-1.0 onboarding program, D2–D6) will slot its deep artifacts into the user track defined
here — this issue rules the information architecture those land in. Dedupe at planning._

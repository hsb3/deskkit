_ADR for the pocket-librarian `chat` surface: upgrading the interactive session from a line REPL to a full-screen terminal UI, and the streaming substrate that makes it reusable._
Status: Accepted — 2026-07-18

# 0004 — `chat`: full-screen Bubble Tea TUI, streaming event layer, resume

## Context

ADR 0001 committed the interactive surface as a **terminal session first**, with a
PocketBase-served webapp deferred as the recorded follow-on (its option b). What shipped under
that decision was `chat` as a line-oriented REPL — a `bufio.Scanner` read-eval loop with no
full-screen/graphical terminal UI and no bubbletea dependency. ADR 0001's 2026-07-17 Correction
records exactly that: it clarified that "TUI" in the original text meant this line REPL.

The REPL proved the interaction model but left the human experience thin: replies land as one
blocking block (no live tokens or tool steps), an in-flight turn cannot be canceled, and there
is no way back into a prior conversation. This ADR takes the next increment ADR 0001 anticipated
("the natural next increment once the TUI proves the interaction model") and upgrades `chat` to
a genuine full-screen terminal UI, while keeping the REPL as the non-interactive path.

**Supersession scope — narrow and deliberate.** This ADR supersedes **only** the first
clarification in ADR 0001's Correction — the "'TUI' means a line REPL with no bubbletea
dependency" note. ADR 0001's actual Decision is untouched and this ADR builds directly on it:
the surface is still a terminal session first, in the same single binary, on-demand, against the
local store, reusing the gated tool set, with the webapp still deferred as option b. ADR 0001's
status line is therefore unchanged. The word "TUI" in ADR 0001 now reads literally.

## Decision

**`chat` auto-detects the terminal and runs a full-screen Bubble Tea TUI when interactive;
the pre-existing line REPL remains the non-TTY / `--plain` path.**

- **Routing.** `chat`'s RunE checks both stdio ends with `go-isatty`. When stdin **and** stdout
  are terminals and `--plain` is not set, it launches the TUI (`internal/tui`); otherwise
  (either end piped, or `--plain`) it runs the existing `runChat` REPL. Detection happens after
  `requireConfig`, so any self-init/open-guard notice prints before the alternate screen opens.

- **Streaming event layer as reusable substrate.** `internal/agent/stream.go` defines a flat,
  JSON-taggable `Event` type (kinds `token`, `tool_start`, `tool_end`, `final`, `error`) and
  `Session.StreamTurn(ctx, input) <-chan Event`. `Event` is serializable-only — no error values
  or channels — so it can be marshaled directly onto the deferred webapp's SSE route (ADR 0001
  option b) without a translation layer. `Session.Turn` is now a thin drain over `StreamTurn`,
  so the REPL and the TUI share one code path and identical persistence/history semantics.

- **Cancel and resume, over the same run rows.** `esc` cancels an in-flight turn; `ctrl+o`
  opens a picker of prior manual conversations to resume. Resume reuses the existing
  `agent_runs` row as the conversation and its `messages` as the transcript — **no new
  collections and no migration**.

- **No new write path.** The TUI drives the same session over `tools.AgentTools(cfg)`:
  `restore` is never model-exposed and `apply_fix` stays gated by `LIBRARIAN_AUTONOMOUS_WRITES`.
  The §5.4/§5.5 write boundary and the stewardship-lane boundary hold unchanged — the surface
  is new, the tool set is not.

## Consequences

- **Transcript persistence corrected.** The per-session high-water mark that tracks which
  history rows are already persisted was computed once per session, which in multi-turn runs
  duplicated the prior assistant row or dropped the new user row — a latent bug that shipped in
  the REPL. `StreamTurn` re-baselines the hwm per turn (first model input is
  `[system] + history + [userN]`; baseline is the count of already-persisted leading messages,
  and 0 on the first turn so the system row persists exactly once). Multi-turn transcript row
  patterns therefore change strictly as a correction, for both the REPL and the TUI.

- **Cancel semantics are model-safe.** A canceled turn that produced partial text persists a
  **plain** assistant row carrying that text — no synthetic marker in the DB, so the model never
  sees "(interrupted)" text; the TUI renders the badge in the UI only. A cancel before any token
  rolls the in-memory history back to its pre-turn length, but the user row already persisted at
  model-input time may remain as an orphaned row.

- **Resume collapses orphans.** Rehydrating a resumed conversation replays only the clean
  alternating user + final-assistant history to the model; an orphaned user row (from a canceled
  or errored turn that never landed an answer) is dropped so the model input is never two user
  messages back to back. The rule — keep a user row iff the next retained message is an
  assistant — is documented in `internal/agent/resume.go`'s header and covers interior, leading,
  and trailing orphans alike.

- **Dependencies accepted.** Four charmbracelet libraries are added — `bubbletea`, `bubbles`,
  `lipgloss`, `glamour` — all pure Go, so the `CGO_ENABLED=0` cross-compile matrix is unaffected.
  `go-isatty` is promoted to a direct dependency (for the TTY routing), and `sahilm/fuzzy`
  enters as an indirect transitive of `bubbles`. The binary grows from ~70.5 MiB (v0.4.0) to
  ~78.3 MiB.

- **Reversible and additive.** The REPL is not removed — it is the non-interactive and
  `--plain` path — and the streaming layer is the same in-process loop ADR 0001 committed. The
  deferred webapp can still be built later onto `StreamTurn`'s events without unwinding any of
  this.

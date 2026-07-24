_ADR for the chat TUI's UI foundation: migrate from the Charm v1 stack to the v2 generation,
retaining the terminal-background rendering policy and the no-runtime-terminal-query rule._
Status: Accepted — 2026-07-18

# 0007 — Chat TUI moves to the Charm v2 stack (terminal background retained)

## Context

The chat TUI is built on the Charm v1 stack (bubbletea 1.3.x, lipgloss 1.1.x, bubbles 1.0,
glamour 1.0). The product direction after the chat-TUI UX pass (PR #48) is a full chat
application — session management, context-window visibility, richer chrome — with the design
language deliberately lifted from the current generation of Charm-built clients (Crush being
the reference; survey in `_meta/_archive/chat-tui-ux-survey.md`).

That generation is built on the Charm **v2** stack, which is also where Charm's development
investment now goes. Staying on v1 means hand-rolling components v2 ships (session lists,
dialogs, status-bar patterns), and a strictly larger migration later, after more code sits on
v1 APIs. Two constraints bind any stack choice:

- **ADR 0004's no-query rule**: no terminal queries after the Bubble Tea program starts
  (an unmanaged OSC background query leaks its response into the textarea). On v1 this is
  enforced by resolving the theme once pre-program and pinning lipgloss's background cache.
- **Licensing**: Crush itself is FSL-licensed (not MIT). Its *designs* are lifted; its code
  is not. The Charm libraries themselves (bubbletea, lipgloss, bubbles, glamour) are MIT.

## Decision

**1. Migrate `librarian/internal/tui` to the Charm v2 modules** (bubbletea v2, lipgloss v2,
bubbles v2, glamour v2) in a dedicated migration PR with **no feature or visual changes** —
behavior parity is the acceptance bar, proven by the existing test suite (adapted
mechanically, not weakened) and re-recorded VHS media.

**2. The TUI keeps rendering on the terminal's background** (not a painted app background,
which v2's `tea.View.BackgroundColor` would allow, Crush-style). The shipped theming model
carries over unchanged as policy: static per-theme palettes resolved **once, pre-program**
(`--theme` flag > `LIBRARIAN_THEME` env > one background probe, dark on indeterminate).
Painting the background is explicitly revisitable later on v2 without waste.

**3. The no-query rule survives the migration in its strongest form**: the background probe
remains a single pre-program call (lipgloss v2's standalone query helper); the TUI does not
adopt v2's in-program managed background-query flow (`tea.RequestBackgroundColor`), even
though the v2 runtime serializes it safely. Static-after-startup stays the invariant the
defects tests pin — glamour v2 removing auto-style detection makes part of the old guard
structural. If a future need arises (e.g. live theme switching), adopting the managed flow
is a correction to record here, not a silent change.

## Consequences

- #51 (session-management surface) and #52 (context/token display) build on v2 components
  (bubbles list/dialog/help) instead of hand-rolled equivalents.
- The v1-era workaround pinning (`lipgloss.SetHasDarkBackground` before `tea.NewProgram`)
  disappears with v1's global background state; per-theme concrete colors are passed
  explicitly, which is the model the code already follows.
- One-time mechanical cost: API deltas across the four modules (Model/key/message shapes,
  bubbles component APIs, glamour style selection). No architectural rework — the
  model/update/view structure, the streaming event layer (ADR 0004), and the REPL fallback
  are unchanged.
- The FSL discipline is recorded: lift designs from FSL-licensed reference apps, write
  original code on the MIT Charm libraries.

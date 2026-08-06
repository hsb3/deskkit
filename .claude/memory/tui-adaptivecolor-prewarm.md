---
name: tui-adaptivecolor-prewarm
description: librarian chat TUI's no-terminal-query invariant (ADR 0004) is NOT self-enforced — textarea's un-overridden AdaptiveColor defaults are made safe only by bubbletea's tea_init.go pre-warm
metadata:
  type: project
---

The librarian chat TUI (`librarian/internal/tui/`) claims a hard ADR-0004 invariant: no terminal
background query (OSC 11) may fire after `tea.NewProgram` starts, or the response leaks into the
textarea as typed input. The branch's own code upholds its part: zero `lipgloss.AdaptiveColor` in
TUI source, `help.Styles` all 7 fields overridden with concrete colors (styles.go helpStyles),
`glamourStyle` pins static Light/Dark (never AutoStyle), theme resolved pre-program via
`termenv.HasDarkBackground` in `ResolveTheme`.

**BUT the invariant is NOT self-enforced.** The `bubbles/textarea` DefaultStyles use
`lipgloss.AdaptiveColor` (CursorLine bg, Text, EndOfBuffer, LineNumber — textarea.go ~323-339),
and `model.go newModel` does NOT override them (only sets Placeholder/Prompt/ShowLineNumbers/
CharLimit/InsertNewline/Focus). Rendering the textarea in `View()` (after program start) calls
`lipgloss.Renderer.HasDarkBackground()`, whose `getBackgroundColor sync.Once` fires the real
OSC-11 query on FIRST call.

**What saves it:** `bubbletea@v1.3.10/tea_init.go` has a package `init()` that runs
`_ = lipgloss.HasDarkBackground()` at import time (before `main`), pre-warming that sync.Once
cache in the safe window. So the render-time textarea AdaptiveColor resolves from cache — no live
query. The upstream comment says this workaround "will be removed in v2."

**How to apply:** the "no AdaptiveColor transitively via bubbles components" claim is FALSE
(textarea has it, un-overridden) — but the functional no-query invariant still HOLDS in the
current build via the bubbletea init pre-warm. Two latent risks worth flagging, not blockers:
(1) a bubbletea v2 bump drops the pre-warm → the textarea AdaptiveColor reintroduces the exact
ADR-0004 leak; (2) the textarea colors follow the REAL auto-detected background, ignoring an
explicit `--theme light|dark`, so `--theme` doesn't fully control the surface (model palette can
mismatch the textarea palette). Self-enforcing fix would be to override textarea styles with the
resolved theme's concrete colors, or call `lipgloss.SetHasDarkBackground(theme==dark)` pre-program.
Verify the pre-warm still exists on any bubbletea bump: `rg -n HasDarkBackground <bubbletea>/tea_init.go`.

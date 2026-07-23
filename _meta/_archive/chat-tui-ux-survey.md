# Chat-TUI UX survey — conventions → gap list → recommendations

_Findings note for the chat-TUI UX pass (#45): what modern AI chat TUIs converge on,
where `deskkit chat` falls short, and what we adopt vs. defer._
Status: active (2026-07-18)

## Method

Two research passes over current-generation terminal AI chat clients, verified against
official docs and (for the Charm wave) the actual repo sources at HEAD:

- **Wave A** — Claude Code ([interactive-mode](https://code.claude.com/docs/en/interactive-mode),
  [accessibility](https://code.claude.com/docs/en/accessibility)), aider
  ([commands](https://aider.chat/docs/usage/commands.html),
  [options](https://aider.chat/docs/config/options.html)), gptme (repo source).
- **Wave B** — Crush, Mods (charmbracelet repo sources), Elia (dormant since 2024-10;
  design reference only), plus the bubbles/lipgloss/glamour/termenv ecosystem itself.

## What the field converges on

| Convention | Who does it | Note |
|---|---|---|
| Turn separation beyond bold labels | Claude Code (user-message background fill), Crush (colored left-gutter border `▌` per role + 1-line gap), Elia (bordered message boxes) | gptme's label-only approach — our status quo — is the floor of the field |
| Copy the **raw markdown**, not the screen | aider `/copy`, Crush focused-message `y`/`c` (OSC 52 + native clipboard + status toast), Claude Code's copy-as-raw-Markdown option | Both leaders explicitly note terminal selection of wrapped text is broken |
| Input history + draft preservation | Claude Code (per-project history, Esc-clear saves draft), aider (persistent history file), Crush (history from session DB + draft stash) | bubbles' textarea ships **no** history — always app-land |
| Self-documenting keys | Crush/bubbles `help` bubble from `key.Binding` (one-line footer, `?`-style show-all toggle), Elia auto-footer, Claude Code `?` panel + `/` menu | The `key.Binding` primitive makes the footer self-maintaining |
| Progress beyond ready/busy | aider's always-on `Tokens: … Cost: …` line, Crush per-turn `◇ model via provider in 3.2s` footer + sidebar context %, Claude Code `/cost` + status line | Latency is cheap; cost needs a pricing table |
| Cancel keeps partial work | Claude Code Esc, aider Ctrl-C ("partial response remains in the conversation") | Universal among the leaders |
| Theme resolved once at startup | Crush run-mode `lipgloss.HasDarkBackground` pre-program; aider explicit `--dark-mode`/`--light-mode`; glamour v2 **removed** AutoStyle — its upgrade guide's recipe is detect-once → static light/dark style | Validates the ADR 0004 constraint; nobody queries mid-program |
| NO_COLOR degradation | Free via termenv/colorprofile when all output flows through the profile-aware writer | Gotcha: pre-rendered escapes written raw leak color |
| Scroll-anchored streaming | Mods/bubbles viewport: auto-follow only if `ScrollPercent()==1`; Claude Code "Jump to bottom (N new)" | Never yank the reader down mid-scrollback |
| External `$EDITOR` escape hatch | All of Claude Code, aider, gptme, Crush, Mods | The single most universal input affordance |

## Gap list for `deskkit chat` (pre-pass state)

1. **Readability on light terminals** — fixed dark palette (`Color("15")` body, glamour
   DarkStyle). Tracked as the blocking bug; fixed first in this pass.
2. **Turn separation** — bold role labels only; no gutter, border, or spacing rhythm.
3. **Copy affordance** — none.
4. **Progress signal** — spinner + ready/busy only; no per-turn latency or token info.
5. **Discoverability** — hand-rolled one-line footer; no help expansion; keybinds only.
6. **Input** — no history recall, no draft preservation on cancel/clear.
7. **Streaming scroll** — follow behavior vs. reading scrollback unverified.
8. **Accessibility** — no NO_COLOR verification, no reduced-motion path.

## Adopted in this pass (ranked)

1. **Startup theme resolution + per-theme palettes** (the readability bug): TTY-guarded
   `termenv.HasDarkBackground()` once before `tea.NewProgram`, `--theme light|dark|auto` +
   `LIBRARIAN_THEME` overrides, static glamour light/dark style. Concrete per-theme colors,
   no `AdaptiveColor` — keeps the no-query-after-start invariant trivially auditable.
2. **Left-gutter role borders + spacing rhythm** (Crush pattern): colored `▌`/`│` left edge
   per role + blank-line gap — survives NO_COLOR (the glyph separates even uncolored).
3. **Per-turn assistant footer**: muted `model · 3.2s` line per finished turn.
4. **`key.Binding` keymap + `bubbles/help` footer** with `?`-toggled full help.
5. **Copy-last-answer** keybind: raw markdown out via OSC 52 + native clipboard fallback,
   confirmed by a transient footer toast.
6. **Input history + draft stash**: Up/Down at input edges walks prior user prompts from
   the session transcript; the in-progress draft is stashed and restored.
7. **Scroll-anchored streaming**: auto-follow only when already at bottom.
8. **Cancel keeps the partial response** in the transcript (verify/ensure; badge already
   exists for interrupted turns).
9. **NO_COLOR / reduced-motion**: verify NO_COLOR degrades sanely through the lipgloss
   profile; static "working…" text instead of the spinner when set.

## Deferred (recorded, not lost)

- **`/` slash-command menu / palette** — the command surface is small; the help footer
  covers discoverability at this scale. Revisit if commands grow.
- **External `$EDITOR` escape hatch** (`tea.ExecProcess` + temp file) — universal
  convention, small build; deferred only to keep this PR reviewable.
- **Token/cost display** — needs a pricing table + token counts from the provider events;
  per-turn latency ships first, counts can append later without layout change.
- **Elia-style selection mode / code-block jump** — best-in-class, but large; a regex over
  fenced blocks in raw markdown is the portable substitute when wanted.
- **Crush's incremental glamour streaming cache** — only needed once full re-render is
  measurably slow.
- **Owning the background** (`tea.View.BackgroundColor`) — requires the bubbletea v2 stack;
  a stack upgrade is its own decision.

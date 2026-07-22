**Make the TUI self-explanatory and the first-run experience guided — visible view switching, discoverable keybindings, and a near-term onboarding-docs patch. Owner-raised 2026-07-22.**

## Evidence

- **The PM views are effectively invisible.** The owner — the person who commissioned the PM
  module — did not know the three PM TUI views existed. They mount into `deskkit chat` behind a
  single unadvertised chord (`ctrl+p` cycles chat → `pm context` → `pm board` → `pm item`;
  `librarian/internal/modules/librarian/tui/host_views.go`), with no on-screen affordance that
  views are mounted, no view tabs/titles, and no keybinding hint anywhere in the UI.
- **Silent no-op failure mode.** On a desk with no mounted views (PM off), `ctrl+p` is a
  deliberately disabled binding with zero feedback (`TestViews_DisabledWithoutMount`) — an
  operator can't distinguish "PM off", "stale binary", and "no such feature".
- **The bar has moved.** Modern agent TUIs (opencode, deepagents-code) ship persistent footer
  keybinding hints, visible panes/tabs, and command palettes as table stakes. Against that bar,
  the current interaction model is "know the chord."
- **Docs don't bridge the gap.** `docs/pm-guide.md` §"The TUI views" names the three views but
  not how to reach them; `docs/getting-started.md` has no TUI tour at all. The deep fix (the
  human co-operator onboarding program) is #104 — but that epic is post-1.0.0, and the friction
  above is a today problem.

## Scope — three workstreams

### 1. TUI UX design (the core of this issue)

- Persistent footer/status line showing the active view's keybindings (chat and module views).
- A visible view indicator — tabs or a titled header (`chat | pm context | pm board | pm item`) —
  so mounted views are discoverable on sight, replacing blind `ctrl+p` cycling as the only path.
- `?` help overlay listing every binding in the current view.
- Feedback instead of silence: launching on a PM-enabled desk announces the mounted views;
  `ctrl+p` with nothing mounted explains why (module off) instead of no-op.
- Consider a view picker / command palette if it falls out naturally from the above; not required.
- Design context to respect: ADR 0001 (TUI-first), ADR 0004 (full-screen chat), ADR 0007
  (charm v2 stack), spec §5.3 (module-view plug-point).

### 2. Guided first-run

- A fresh-desk `deskkit` launch should orient, not assume: point the operator at the next step
  (init → profile → sweep → patrol → chat) rather than dropping into a bare prompt.
- First TUI launch shows a one-time hint line (views mounted, `?` for help).

### 3. Onboarding-docs patch (near-term slice; the program is #104)

- `docs/getting-started.md`: add a TUI tour — surfaces, views, keybindings, what a healthy first
  session looks like.
- `docs/pm-guide.md` §TUI views: document how to reach the views (`ctrl+p`) and their keys
  (`enter` board→item, `r` refresh).
- Cross-link relationship: #104's D2 (mental-model map) and D4 (reference cards) are the deep
  versions of this material — dedupe at planning; this workstream is the minimal patch that
  should not wait for the epic.

## Acceptance

- A new operator discovers every mounted view without reading source or docs (visible affordance
  in the TUI itself).
- Every keybinding is discoverable in-TUI (footer or `?` overlay).
- `ctrl+p` (or its successor) never fails silently — mounted views are announced; absence is explained.
- getting-started + pm-guide document the TUI surfaces and keys; a cold reader can reach the PM
  board from `deskkit chat` on the first try.
- TUI behavior changes stay inside the existing tuiview plug-point contract (module views keep
  mounting via `Module.TUIViews`).

_Relationship: complements #104 (post-1.0 onboarding program) — this is the pre-epic slice: make
the shipped TUI legible and patch the docs gap now; the training desk + full doc program remain #104._

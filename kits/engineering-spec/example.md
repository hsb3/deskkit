---
type: engineering-spec
status: approved
created: 2026-05-12
updated: 2026-05-12
tags: [reader-surface, focus-mode]
author: robin
parent_product_spec: "[[focus-mode-product-spec]]"
priority: P0
definition_of_done: false
test_plan_link: "[[focus-mode-test-plan]]"
---

# Add focus/zen mode toggle to all reader pages

## Why this exists

Readers (and Robin as author) need a distraction-free reading view. Rails and chrome that help orient on a feed page get in the way when reading a long-form entry. From the parent product-spec: *"every page must terminate cleanly; no Medium-style pile-on."* A toggleable focus mode lets the reader settle in without losing the navigation surface entirely.

## What it does

- A small button in the top bar — between the search field and "log in" — toggles focus mode
- Keyboard shortcut: `F` (only when not typing in an input)
- When active: both rails hide, top nav slims to wordmark + exit-focus button only, footer slims to one line, main column widens to max 860px
- URL gets `?focus=1` written via `history.replaceState` — shareable and persistent in the session
- State persists across in-site navigation within the session

## What it does NOT do

- Does NOT persist across browser sessions (no localStorage)
- Does NOT change sort order or any other state
- Does NOT hide the search field permanently — `/` keystroke still opens search in focus mode (handled by separate spec [[add-search-keyboard-shortcut]])
- Does NOT animate the transition — instantaneous toggle
- Does NOT support per-page focus modes (it's global per-session)
- Does NOT show a first-run tutorial or hint
- Does NOT toggle dark mode along with it (separate concern)

## Acceptance criteria

- [ ] Focus button visible in the top bar of all four reader pages (`/feed`, `/entry/:slug`, `/topics`, `/about`)
- [ ] Clicking the button hides both rails and slims top nav + footer
- [ ] Pressing `F` (when not focused in an input) toggles the same way
- [ ] Pressing `F` again exits focus mode
- [ ] URL reflects `?focus=1` when active and updates without page reload
- [ ] Clicking the exit-focus icon in the slimmed top bar exits
- [ ] No layout shift when toggling (only `display` changes; no DOM reflow that moves content vertically)
- [ ] Mobile (<800px): button still functions; rails were already hidden, so the slim is purely top-nav

## Definition of done

- [ ] All acceptance criteria met
- [ ] Unit test for the toggle function (`focus-toggle.test.ts`)
- [ ] Playwright integration test: click button on each of 4 pages, assert rails hidden + URL updated; press `F`, repeat
- [ ] Manual smoke on Safari + Chrome desktop, Safari mobile
- [ ] No new lint or type errors
- [ ] PR reviewed and approved
- [ ] Deployed to production

## Test plan

- **Manual**: load each of 4 reader pages; click button; verify rails hide; verify URL gains `?focus=1`; click exit; verify everything restores. Repeat with `F` keyboard shortcut. Then test with focus inside the search field — `F` must NOT toggle.
- **Automated**: Playwright spec `tests/focus-mode.spec.ts` covers all 4 pages × 2 toggle methods (button + `F` key).
- **Edge**: type "f" in the search field — focus mode should NOT toggle. Verified in Playwright.

## Interaction

1. User on `/feed`
2. Clicks focus button OR presses `F`
3. Both rails hide (`display: none`)
4. Top nav loses "feed / topics / pipeline / about" items, keeps wordmark + exit-focus icon
5. Footer collapses to a single line
6. Main column re-centers, widens to 860px max
7. URL updates to `/feed?focus=1` via `history.replaceState`
8. User reads; in-site navigation preserves focus state
9. User clicks exit-focus OR presses `F` again
10. All chrome restores; URL drops `?focus=1`

## States

- `idle` (default): rails visible, full top nav, normal column width
- `focused`: rails hidden, slim top nav, 860px column
- Transitions: button click or `F` keyboard
- No intermediate / loading states — toggle is synchronous

## Edge cases

- **User is typing in an input when `F` is pressed**: do nothing (let the character go to the input)
- **User toggles on `/feed`, then navigates to `/entry/notes-on-middleware`**: focus mode persists (URL keeps `?focus=1`)
- **User shares URL with `?focus=1`**: new visitor lands in focus mode
- **JS disabled**: focus mode unavailable; the toggle button doesn't render (server-rendered without it)
- **iOS Safari**: keyboard `F` may not fire reliably; button is the primary affordance there

## Tech notes

- Implementation: a small React island mounted in the top bar, plus a vanilla JS module for keyboard handling and URL state
- Files: `src/components/FocusToggle.tsx` (React) + `src/lib/focus-mode.ts` (state)
- CSS: a `body.focus-active` class triggers all layout changes via the existing design tokens in `src/styles/focus.css`
- No new dependencies

## Dependencies

- **Upstream**: [[design-tokens-finalized]] (CSS variables for rail widths and gutters)
- **Downstream**: [[add-search-keyboard-shortcut]] — search must still work in focus mode

## Open questions

- Should focus mode also pause autoplay on any media embeds? Probably yes, but no media embeds exist in v1. → **Defer**; revisit when first media entry is published.

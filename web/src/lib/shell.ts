// The shell: which work mode you are in, and how much of the screen the thing you are working
// on has taken.
//
// SPACE FOLLOWS ENGAGEMENT — the ruled shape of this app. These are not three panes sharing a
// screen, they are three allocations of the same screen: while you are looking for something,
// the finder IS the screen; once you open an item it minimises INTO its rail button, lit, so
// the way back is visible rather than remembered; editing is that same allocation with
// different verbs. ESC unwinds exactly one step of it and never more.
//
// The state lives here rather than in props because two unrelated components need it — the rail
// draws it, the finder obeys it — and threading it through App would make every future screen
// re-thread it.
import { writable } from 'svelte/store'
import type { Action } from './keys'
import type { IconName } from './icons'

/** A rail button: one distinct work mode and nothing else — that is the whole rule for what
 * earns one. `id` is the hash segment, so a link and a keystroke land in the same place. */
export interface Mode {
  id: string
  label: string
  /** What this mode is for, in one line — the rail is 34px, so the words live in its tooltip. */
  hint: string
  /** The glyph on the button. Decoration ABOVE the digit, never instead of it: the rail is the
   * shortcut legend, so the number stays visible. */
  icon: IconName
}

/** Rail order top to bottom, and therefore the ⌘1..⌘6 order. Kits and Desks are deliberately
 * absent: they are reached from Config, because neither is a distinct work mode. */
export const MODES: Mode[] = [
  { id: 'queue', label: 'Queue', hint: 'What needs you, across findings and work items', icon: 'queue' },
  { id: 'library', label: 'Library', hint: 'The desk’s documents', icon: 'library' },
  { id: 'patrol', label: 'Patrol', hint: 'Findings the librarian has flagged', icon: 'patrol' },
  { id: 'work', label: 'Work', hint: 'The work graph', icon: 'work' },
  { id: 'agent', label: 'Agent', hint: 'The agent conversation and its runs', icon: 'agent' },
  { id: 'config', label: 'Config', hint: 'This desk’s settings', icon: 'config' },
]

/** Hash segments the pre-rail SPA used. Kept so existing links and bookmarks still land
 * somewhere sensible instead of on the default mode. */
const ALIASES: Record<string, string> = {
  documents: 'library',
  findings: 'patrol',
  pm: 'work',
  chat: 'agent',
  runs: 'agent',
  settings: 'config',
}

/** Resolve a hash segment to a mode id, falling back to the landing mode. */
export function resolveMode(page: string): string {
  const id = ALIASES[page] ?? page
  return MODES.some((m) => m.id === id) ? id : MODES[0].id
}

/** The modifier glyph this machine's user actually presses, for the rail's tooltips. */
export const MOD_LABEL =
  typeof navigator !== 'undefined' && /Mac|iPhone|iPad/.test(navigator.userAgent) ? '⌘' : 'Ctrl+'

export type Level = 'finder' | 'reading' | 'editing'

/** ESC: exactly one level out. From the finder there is nowhere further to go — backing out of
 * the app is not something a keystroke gets to do. */
export function outOf(l: Level): Level {
  return l === 'editing' ? 'reading' : 'finder'
}

/** Which levels each level-SCOPED action may fire at. Actions absent from this table fire at
 * every level (mode switches, the finder toggle, search, j/k, esc).
 *
 * A table rather than an `if` inside each case, because a MISSING check is invisible: `delete`
 * had none, so a stray ⌘⌫ while browsing the list armed a delete on whatever record was last
 * opened — with its "Really delete?" button off screen, since that button only renders inside
 * the instance. The two-step confirm only protects anything if the second step is visible.
 */
const FIRES_AT: Partial<Record<Action['kind'], Level[]>> = {
  open: ['finder'],
  modify: ['finder', 'reading'],
  body: ['reading'],
  save: ['editing'],
  delete: ['reading', 'editing'],
}

/** May this action fire at this level? Unlisted actions are unrestricted. */
export function mayFire(kind: Action['kind'], at: Level): boolean {
  return FIRES_AT[kind]?.includes(at) ?? true
}

export const level = writable<Level>('finder')

/** Where ⌘B returns you when you bring the finder back. Module-local because it is a detail of
 * the toggle, not state anything else may read. */
let restore: Level = 'reading'

/** ⌘B / the F rail button: the finder is a toggle against wherever you were, so the button it
 * minimised into is the button that brings it back — including back into an unfinished edit. */
export function toggleFinder(): void {
  level.update((l) => {
    if (l === 'finder') return restore
    restore = l
    return 'finder'
  })
}

export function setLevel(l: Level): void {
  if (l !== 'finder') restore = l
  level.set(l)
}

/** "Keep the finder minimised between items" — ruled a user setting, default on.
 * On, j/k walk the collection without reopening the list at all; off, moving to the next item
 * hands the screen back to the finder. Loaded from the desk's settings row; the default here is
 * what an unreachable or older store falls back to. */
export const stickyFinder = writable(true)

// --- the action bus ---------------------------------------------------------------------
// App owns modes and levels; the row-level intents (next/prev/open/modify/save/search) belong to
// whichever finder is on screen, and that changes with the mode. A one-line bus beats threading
// callbacks through every future screen, and a mode with no finder simply has no listener — the
// keystroke then does nothing, which is the honest outcome.

type Listener = (a: Action) => void
const listeners = new Set<Listener>()

/** Subscribe to the row-level actions. Returns the unsubscribe, for onMount's teardown. */
export function onAction(fn: Listener): () => void {
  listeners.add(fn)
  return () => {
    listeners.delete(fn)
  }
}

/** Deliver an action. Returns whether anything was listening, so the caller can decide whether
 * the keystroke was consumed or should fall through to the browser. */
export function dispatch(a: Action): boolean {
  if (listeners.size === 0) return false
  for (const fn of [...listeners]) fn(a)
  return true
}

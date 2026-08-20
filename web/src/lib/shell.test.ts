import { beforeEach, describe, expect, it, vi } from 'vitest'
import { get } from 'svelte/store'
import {
  MODES,
  dispatch,
  level,
  mayFire,
  onAction,
  resolveMode,
  setLevel,
  toggleFinder,
} from './shell'
import { ICONS } from './icons'

beforeEach(() => setLevel('finder'))

describe('resolveMode', () => {
  it('resolves every rail id to itself', () => {
    for (const m of MODES) expect(resolveMode(m.id)).toBe(m.id)
  })

  it('carries the pre-rail hash segments to their new mode', () => {
    expect(resolveMode('documents')).toBe('library')
    expect(resolveMode('findings')).toBe('patrol')
    expect(resolveMode('pm')).toBe('work')
    expect(resolveMode('chat')).toBe('agent')
    expect(resolveMode('runs')).toBe('agent')
    expect(resolveMode('settings')).toBe('config')
  })

  it('falls an empty or unknown segment back to the landing mode', () => {
    expect(resolveMode('')).toBe(MODES[0].id)
    expect(resolveMode('nonsense')).toBe(MODES[0].id)
  })
})

describe('toggleFinder', () => {
  it('returns you to the level the finder minimised from — including an unfinished edit', () => {
    setLevel('editing')
    toggleFinder()
    expect(get(level)).toBe('finder')
    toggleFinder()
    expect(get(level)).toBe('editing')
  })

  // Where the toggle lands before anything has been opened is a property of a FRESH shell, and
  // the level it remembers is module state — so this one gets its own module instance rather
  // than inheriting whatever the test above left behind.
  it('opens onto reading before anything has been opened', async () => {
    vi.resetModules()
    const fresh = await import('./shell')
    fresh.toggleFinder()
    expect(get(fresh.level)).toBe('reading')
    fresh.toggleFinder()
    expect(get(fresh.level)).toBe('finder')
  })
})

describe('the action bus', () => {
  it('reports whether anything was listening, so an unclaimed keystroke can fall through', () => {
    expect(dispatch({ kind: 'next' })).toBe(false)
    const seen = vi.fn()
    const off = onAction(seen)
    expect(dispatch({ kind: 'next' })).toBe(true)
    expect(seen).toHaveBeenCalledWith({ kind: 'next' })
    off()
    expect(dispatch({ kind: 'next' })).toBe(false)
    expect(seen).toHaveBeenCalledTimes(1)
  })
})

// The rail is the app's navigation AND its shortcut legend. The icon pass added the first job's
// glyph; this is the check that it never gets to cost the second one.
describe('the rail buttons', () => {
  it('gives every mode a glyph that actually exists', () => {
    for (const m of MODES) {
      expect(m.icon, m.id).toBeTruthy()
      expect(ICONS[m.icon], m.id).toBeTruthy()
    }
  })

  it('draws every glyph on the same 24x24 grid, so they line up at rail size', () => {
    for (const [name, d] of Object.entries(ICONS)) {
      expect(d, name).toMatch(/^[MmLlHhVvAaCcQqZz0-9 .,-]+$/)
      for (const n of d.match(/\d+(\.\d+)?/g) ?? []) expect(Number(n), `${name}: ${n}`).toBeLessThanOrEqual(24)
    }
  })
})

// The gate that stops a verb firing at a level where its on-screen control is not rendered.
// `delete` is the one that made this necessary: it armed from the finder, where the
// "Really delete?" button does not exist, so the second press had nothing to confirm against.
describe('mayFire', () => {
  it('refuses delete from the finder, where the confirm button is not on screen', () => {
    expect(mayFire('delete', 'finder')).toBe(false)
    expect(mayFire('delete', 'reading')).toBe(true)
    expect(mayFire('delete', 'editing')).toBe(true)
  })

  it('lets every level-scoped verb fire only where its control is rendered', () => {
    expect(mayFire('open', 'finder')).toBe(true)
    expect(mayFire('open', 'reading')).toBe(false)
    expect(mayFire('save', 'editing')).toBe(true)
    expect(mayFire('save', 'finder')).toBe(false)
    expect(mayFire('save', 'reading')).toBe(false)
    expect(mayFire('body', 'reading')).toBe(true)
    expect(mayFire('body', 'finder')).toBe(false)
    expect(mayFire('body', 'editing')).toBe(false)
    expect(mayFire('modify', 'finder')).toBe(true)
    expect(mayFire('modify', 'reading')).toBe(true)
    expect(mayFire('modify', 'editing')).toBe(false)
  })

  it('leaves the level-independent actions alone at every level', () => {
    for (const at of ['finder', 'reading', 'editing'] as const) {
      for (const kind of ['mode', 'finder', 'search', 'next', 'prev', 'back'] as const) {
        expect(mayFire(kind, at), `${kind} @ ${at}`).toBe(true)
      }
    }
  })
})

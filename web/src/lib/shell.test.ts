import { beforeEach, describe, expect, it, vi } from 'vitest'
import { get } from 'svelte/store'
import { MODES, dispatch, level, onAction, resolveMode, setLevel, toggleFinder } from './shell'

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

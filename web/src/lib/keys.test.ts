import { describe, expect, it } from 'vitest'
import { isTyping, resolveKey, MODE_KEY_COUNT } from './keys'
import { MODES, outOf, type Level } from './shell'

const mod = (key: string) => ({ key, metaKey: true })

describe('resolveKey', () => {
  it('maps every mode digit, and nothing past the rail', () => {
    for (let i = 1; i <= MODE_KEY_COUNT; i++) {
      expect(resolveKey(mod(String(i)), false)).toEqual({ kind: 'mode', index: i - 1 })
    }
    expect(resolveKey(mod('7'), false)).toBeNull()
    expect(resolveKey(mod('0'), false)).toBeNull()
  })

  it('treats Control as the same modifier as Meta', () => {
    expect(resolveKey({ key: '2', ctrlKey: true }, false)).toEqual({ kind: 'mode', index: 1 })
    expect(resolveKey({ key: 'b', ctrlKey: true }, false)).toEqual({ kind: 'finder' })
  })

  it('keeps search, save and back alive inside a text field', () => {
    expect(resolveKey(mod('k'), true)).toEqual({ kind: 'search' })
    expect(resolveKey(mod('Enter'), true)).toEqual({ kind: 'save' })
    expect(resolveKey({ key: 'Escape' }, true)).toEqual({ kind: 'back' })
    expect(resolveKey(mod('1'), true)).toEqual({ kind: 'mode', index: 0 })
  })

  it('never steals a bare letter from someone who is typing', () => {
    for (const key of ['j', 'k', 'e', 'o', 'Enter']) {
      expect(resolveKey({ key }, true)).toBeNull()
    }
  })

  it('maps the bare row keys when nothing is being typed into', () => {
    expect(resolveKey({ key: 'j' }, false)).toEqual({ kind: 'next' })
    expect(resolveKey({ key: 'k' }, false)).toEqual({ kind: 'prev' })
    expect(resolveKey({ key: 'Enter' }, false)).toEqual({ kind: 'open' })
    expect(resolveKey({ key: 'e' }, false)).toEqual({ kind: 'modify' })
    expect(resolveKey({ key: 'o' }, false)).toEqual({ kind: 'body' })
  })

  // Destructive, so it is a chord rather than a bare letter — and like the other chords it is
  // deliberately NOT suppressed mid-edit: it arms a two-step confirm, so a stray press costs a
  // second press, never a file.
  it('maps the delete chord on either modifier, inside a field as well as outside', () => {
    expect(resolveKey(mod('Backspace'), false)).toEqual({ kind: 'delete' })
    expect(resolveKey({ key: 'Backspace', ctrlKey: true }, false)).toEqual({ kind: 'delete' })
    expect(resolveKey(mod('Backspace'), true)).toEqual({ kind: 'delete' })
  })

  it('leaves a bare Backspace to the browser — deleting a character is not deleting a file', () => {
    expect(resolveKey({ key: 'Backspace' }, false)).toBeNull()
    expect(resolveKey({ key: 'Backspace' }, true)).toBeNull()
  })

  it('leaves unmapped and Alt-composed chords to the browser', () => {
    expect(resolveKey(mod('s'), false)).toBeNull()
    expect(resolveKey({ key: 'x' }, false)).toBeNull()
    expect(resolveKey({ key: 'j', altKey: true }, false)).toBeNull()
    expect(resolveKey({ key: 'k', metaKey: true, altKey: true }, false)).toBeNull()
  })

  // The promise the rail makes: every button has a key, and every key has a button.
  it('covers exactly the rail buttons with digits', () => {
    expect(MODES.length).toBe(MODE_KEY_COUNT)
    const reached = MODES.map((_, i) => resolveKey(mod(String(i + 1)), false))
    expect(reached.every((a) => a?.kind === 'mode')).toBe(true)
  })
})

describe('isTyping', () => {
  it('recognizes the fields where a keystroke is text', () => {
    expect(isTyping({ tagName: 'INPUT' } as unknown as EventTarget)).toBe(true)
    expect(isTyping({ tagName: 'TEXTAREA' } as unknown as EventTarget)).toBe(true)
    expect(isTyping({ tagName: 'SELECT' } as unknown as EventTarget)).toBe(true)
    expect(isTyping({ tagName: 'DIV', isContentEditable: true } as unknown as EventTarget)).toBe(true)
  })

  it('is false for everything else, including nothing at all', () => {
    expect(isTyping({ tagName: 'DIV' } as unknown as EventTarget)).toBe(false)
    expect(isTyping({ tagName: 'BUTTON' } as unknown as EventTarget)).toBe(false)
    expect(isTyping(null)).toBe(false)
  })
})

describe('outOf', () => {
  it('unwinds exactly one level and stops at the finder', () => {
    const walked: Level[] = ['editing', outOf('editing'), outOf(outOf('editing'))]
    expect(walked).toEqual(['editing', 'reading', 'finder'])
    expect(outOf('finder')).toBe('finder')
  })
})
